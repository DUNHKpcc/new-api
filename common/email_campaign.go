package common

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// CampaignEmailMessage is an individual queue delivery, never a recipient list.
// The queue supplies a stable MessageID for the lifetime of the delivery.
type CampaignEmailMessage struct {
	Subject        string
	Receiver       string
	HTMLBody       string
	TextBody       string
	MessageID      string
	UnsubscribeURL string
}

// CampaignEmailDeliveryError deliberately excludes the server response and the
// original error, which may contain credentials or usable unsubscribe tokens.
// Uncertain deliveries must not be retried automatically: DATA may be accepted.
type CampaignEmailDeliveryError struct {
	Phase     string
	Code      int
	Temporary bool
	Uncertain bool
}

func (e *CampaignEmailDeliveryError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("campaign SMTP %s failed (code %d)", e.Phase, e.Code)
	}
	return "campaign SMTP " + e.Phase + " failed"
}

func campaignSMTPError(phase string, err error, dataStarted bool) error {
	result := &CampaignEmailDeliveryError{Phase: phase}
	var reply *textproto.Error
	if errors.As(err, &reply) && reply.Code >= 400 && reply.Code < 600 {
		result.Code = reply.Code
		result.Temporary = reply.Code < 500
		return result
	}
	result.Uncertain = dataStarted
	var networkError net.Error
	result.Temporary = !dataStarted && (errors.As(err, &networkError) || errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
	return result
}

// SendCampaignEmail reuses the configured SMTP service without changing the
// transactional SendEmail path. The full operation is bounded to 30 seconds,
// shorter than the queue lease; cancellation closes the active socket as well.
func SendCampaignEmail(ctx context.Context, message CampaignEmailMessage) error {
	OptionMapRWMutex.RLock()
	host, port := SMTPServer, SMTPPort
	from, systemName := SMTPFrom, SystemName
	if from == "" {
		from = SMTPAccount
	}
	implicitTLS := SMTPSSLEnabled || (port == 465 && !SMTPStartTLSEnabled)
	startTLS := SMTPStartTLSEnabled && !implicitTLS
	tlsConfig := smtpTLSConfig()
	authenticate := shouldAuthenticateSMTP()
	auth := snapshotSMTPAuth()
	OptionMapRWMutex.RUnlock()

	if host == "" || port < 1 || port > 65535 || strings.ContainsAny(host, "\r\n\x00") {
		return &CampaignEmailDeliveryError{Phase: "configuration"}
	}
	for _, header := range []string{message.Subject, message.Receiver, message.MessageID, message.UnsubscribeURL, from, systemName} {
		if strings.ContainsAny(header, "\r\n\x00") {
			return &CampaignEmailDeliveryError{Phase: "headers"}
		}
	}
	receiver, err := mail.ParseAddress(message.Receiver)
	if err != nil {
		return &CampaignEmailDeliveryError{Phase: "recipient"}
	}
	sender, err := mail.ParseAddress(from)
	if err != nil {
		return &CampaignEmailDeliveryError{Phase: "sender"}
	}
	sender.Name = systemName
	if strings.TrimSpace(message.Subject) == "" || (message.HTMLBody == "" && message.TextBody == "") {
		return &CampaignEmailDeliveryError{Phase: "content"}
	}
	if !strings.HasPrefix(message.MessageID, "<") || !strings.HasSuffix(message.MessageID, ">") || len(message.MessageID) > 254 {
		return &CampaignEmailDeliveryError{Phase: "message-id"}
	}
	id := message.MessageID[1 : len(message.MessageID)-1]
	idAddress, err := mail.ParseAddress(id)
	if err != nil || idAddress.Name != "" || idAddress.Address != id || strings.ContainsAny(id, " <>\t") {
		return &CampaignEmailDeliveryError{Phase: "message-id"}
	}
	for _, ch := range message.MessageID {
		if ch < 33 || ch > 126 {
			return &CampaignEmailDeliveryError{Phase: "message-id"}
		}
	}
	unsubscribe, err := url.Parse(message.UnsubscribeURL)
	if err != nil || unsubscribe.Scheme != "https" || unsubscribe.Hostname() == "" || unsubscribe.User != nil || unsubscribe.Fragment != "" || strings.ContainsAny(message.UnsubscribeURL, " <>\t") || len(message.UnsubscribeURL) > 900 {
		return &CampaignEmailDeliveryError{Phase: "unsubscribe-url"}
	}

	var body bytes.Buffer
	parts := multipart.NewWriter(&body)
	for _, part := range []struct{ kind, body string }{{"plain", message.TextBody}, {"html", message.HTMLBody}} {
		if part.body == "" {
			continue
		}
		writer, err := parts.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {"text/" + part.kind + "; charset=UTF-8"},
			"Content-Transfer-Encoding": {"quoted-printable"},
		})
		if err != nil {
			return &CampaignEmailDeliveryError{Phase: "encoding"}
		}
		encoded := quotedprintable.NewWriter(writer)
		if _, err := io.WriteString(encoded, part.body); err != nil {
			return &CampaignEmailDeliveryError{Phase: "encoding"}
		}
		if err := encoded.Close(); err != nil {
			return &CampaignEmailDeliveryError{Phase: "encoding"}
		}
	}
	if err := parts.Close(); err != nil {
		return &CampaignEmailDeliveryError{Phase: "encoding"}
	}
	var payload bytes.Buffer
	fmt.Fprintf(&payload, "To: %s\r\nFrom: %s\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=%q\r\nList-Unsubscribe: <%s>\r\nList-Unsubscribe-Post: List-Unsubscribe=One-Click\r\n\r\n",
		receiver.String(), sender.String(), mime.QEncoding.Encode("UTF-8", message.Subject), time.Now().Format(time.RFC1123Z), message.MessageID, parts.Boundary(), message.UnsubscribeURL)
	payload.Write(body.Bytes())

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return campaignSMTPError("connect", err, false)
	}
	defer conn.Close()
	stopCancellation := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stopCancellation()
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return campaignSMTPError("deadline", err, false)
	}
	var smtpConn net.Conn = conn
	if implicitTLS {
		encrypted := tls.Client(conn, tlsConfig)
		if err := encrypted.HandshakeContext(ctx); err != nil {
			return campaignSMTPError("tls", err, false)
		}
		smtpConn = encrypted
	}
	client, err := smtp.NewClient(smtpConn, host)
	if err != nil {
		return campaignSMTPError("greeting", err, false)
	}
	defer client.Close()
	if startTLS {
		if err := client.StartTLS(tlsConfig); err != nil {
			return campaignSMTPError("starttls", err, false)
		}
	}
	if authenticate {
		if err := client.Auth(auth); err != nil {
			return campaignSMTPError("auth", err, false)
		}
	}
	if err := client.Mail(sender.Address); err != nil {
		return campaignSMTPError("sender", err, false)
	}
	if err := client.Rcpt(receiver.Address); err != nil {
		return campaignSMTPError("recipient", err, false)
	}
	writer, err := client.Data()
	if err != nil {
		return campaignSMTPError("data", err, false)
	}
	if _, err := writer.Write(payload.Bytes()); err != nil {
		return campaignSMTPError("data-write", err, true)
	}
	if err := writer.Close(); err != nil {
		return campaignSMTPError("data-result", err, true)
	}
	// A successful DATA reply is the delivery boundary. QUIT is only cleanup;
	// retrying because QUIT failed would duplicate an already accepted message.
	_ = client.Quit()
	return nil
}
