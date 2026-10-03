package common

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSMTPBehavior struct {
	blockGreeting  bool
	closeAfterData bool
	blockDataReply bool
	recipientReply int
	dataReply      int
	quitReply      int
	connected      chan struct{}
	disconnected   chan struct{}
}

type fakeSMTPServer struct {
	listener          net.Listener
	host              string
	port              int
	cert              tls.Certificate
	advertiseSTARTTLS bool
	authMechanisms    []string
	messages          chan string
	authCommands      chan string
	startTLSCommands  chan string
	behavior          fakeSMTPBehavior
}

func newFakeSMTPServer(t *testing.T) *fakeSMTPServer {
	return newFakeSMTPServerWithSTARTTLSAdvertisement(t, true)
}

func newFakeSMTPServerWithSTARTTLSAdvertisement(t *testing.T, advertiseSTARTTLS bool, behavior ...fakeSMTPBehavior) *fakeSMTPServer {
	t.Helper()

	cert, err := newTestTLSCertificate()
	require.NoError(t, err)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	host, portText, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)

	server := &fakeSMTPServer{
		listener:          listener,
		host:              host,
		port:              port,
		cert:              cert,
		advertiseSTARTTLS: advertiseSTARTTLS,
		authMechanisms:    []string{"PLAIN", "LOGIN"},
		messages:          make(chan string, 1),
		authCommands:      make(chan string, 1),
		startTLSCommands:  make(chan string, 1),
	}
	if len(behavior) > 0 {
		server.behavior = behavior[0]
	}
	go server.serve()
	return server
}

func newFakeImplicitTLSSMTPServer(t *testing.T) *fakeSMTPServer {
	t.Helper()

	cert, err := newTestTLSCertificate()
	require.NoError(t, err)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	host, portText, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)

	server := &fakeSMTPServer{
		listener:          tls.NewListener(listener, &tls.Config{Certificates: []tls.Certificate{cert}}),
		host:              host,
		port:              port,
		cert:              cert,
		advertiseSTARTTLS: false,
		authMechanisms:    []string{"PLAIN", "LOGIN"},
		messages:          make(chan string, 1),
		authCommands:      make(chan string, 1),
		startTLSCommands:  make(chan string, 1),
	}
	go server.serve()
	return server
}

func (s *fakeSMTPServer) close() {
	_ = s.listener.Close()
}

func (s *fakeSMTPServer) serve() {
	conn, err := s.listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	if s.behavior.connected != nil {
		close(s.behavior.connected)
	}
	if s.behavior.disconnected != nil {
		defer close(s.behavior.disconnected)
	}
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if s.behavior.blockGreeting {
		_, _ = io.Copy(io.Discard, conn)
		return
	}

	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	if err := writeSMTPLine(rw, "220 fake.smtp.local ESMTP"); err != nil {
		return
	}

	encrypted := false
	for {
		line, err := rw.ReadString('\n')
		if err != nil {
			return
		}
		command := strings.TrimRight(line, "\r\n")
		upperCommand := strings.ToUpper(command)

		switch {
		case strings.HasPrefix(upperCommand, "EHLO"):
			if err := writeSMTPLine(rw, "250-fake.smtp.local"); err != nil {
				return
			}
			if !encrypted && s.advertiseSTARTTLS {
				if err := writeSMTPLine(rw, "250-STARTTLS"); err != nil {
					return
				}
			}
			if len(s.authMechanisms) > 0 {
				if err := writeSMTPLine(rw, "250 AUTH "+strings.Join(s.authMechanisms, " ")); err != nil {
					return
				}
			} else if err := writeSMTPLine(rw, "250 8BITMIME"); err != nil {
				return
			}
		case upperCommand == "STARTTLS":
			if encrypted || !s.advertiseSTARTTLS {
				if err := writeSMTPLine(rw, "502 5.5.1 STARTTLS not supported"); err != nil {
					return
				}
				continue
			}
			select {
			case s.startTLSCommands <- command:
			default:
			}
			if err := writeSMTPLine(rw, "220 2.0.0 Ready to start TLS"); err != nil {
				return
			}
			tlsConn := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{s.cert}})
			if err := tlsConn.Handshake(); err != nil {
				return
			}
			conn = tlsConn
			rw = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
			encrypted = true
		case strings.HasPrefix(upperCommand, "AUTH"):
			select {
			case s.authCommands <- command:
			default:
			}
			if err := writeSMTPLine(rw, "235 2.7.0 Authentication successful"); err != nil {
				return
			}
		case strings.HasPrefix(upperCommand, "MAIL FROM:"):
			if err := writeSMTPLine(rw, "250 2.1.0 Sender OK"); err != nil {
				return
			}
		case strings.HasPrefix(upperCommand, "RCPT TO:"):
			if s.behavior.recipientReply != 0 {
				_ = writeSMTPLine(rw, fmt.Sprintf("%d rejected sensitive-server-detail", s.behavior.recipientReply))
				continue
			}
			if err := writeSMTPLine(rw, "250 2.1.5 Recipient OK"); err != nil {
				return
			}
		case upperCommand == "DATA":
			if err := writeSMTPLine(rw, "354 End data with <CR><LF>.<CR><LF>"); err != nil {
				return
			}
			var data strings.Builder
			for {
				dataLine, err := rw.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(dataLine, "\r\n") == "." {
					break
				}
				data.WriteString(dataLine)
			}
			s.messages <- data.String()
			if s.behavior.closeAfterData {
				return
			}
			if s.behavior.blockDataReply {
				_, _ = io.Copy(io.Discard, conn)
				return
			}
			if s.behavior.dataReply != 0 {
				_ = writeSMTPLine(rw, fmt.Sprintf("%d rejected sensitive-server-detail", s.behavior.dataReply))
				continue
			}
			if err := writeSMTPLine(rw, "250 2.0.0 Queued"); err != nil {
				return
			}
		case upperCommand == "QUIT":
			if s.behavior.quitReply != 0 {
				_ = writeSMTPLine(rw, fmt.Sprintf("%d sensitive-server-detail", s.behavior.quitReply))
				return
			}
			_ = writeSMTPLine(rw, "221 2.0.0 Bye")
			return
		default:
			if err := writeSMTPLine(rw, "502 5.5.1 Command not implemented"); err != nil {
				return
			}
		}
	}
}

func writeSMTPLine(rw *bufio.ReadWriter, line string) error {
	_, err := rw.WriteString(line + "\r\n")
	if err != nil {
		return err
	}
	return rw.Flush()
}

func newTestTLSCertificate() (tls.Certificate, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "aixinexchange01.aixin-chip.com",
		},
		NotBefore:   time.Now().Add(-time.Hour),
		NotAfter:    time.Now().Add(time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{"aixinexchange01", "aixinexchange01.aixin-chip.com"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	return tls.X509KeyPair(certPEM, keyPEM)
}

func withSMTPSettings(t *testing.T) {
	t.Helper()
	originalSMTPServer := SMTPServer
	originalSMTPPort := SMTPPort
	originalSMTPSSLEnabled := SMTPSSLEnabled
	originalSMTPStartTLSEnabled := SMTPStartTLSEnabled
	originalSMTPInsecureSkipVerify := SMTPInsecureSkipVerify
	originalSMTPForceAuthLogin := SMTPForceAuthLogin
	originalSMTPAccount := SMTPAccount
	originalSMTPFrom := SMTPFrom
	originalSMTPToken := SMTPToken
	originalSystemName := SystemName

	t.Cleanup(func() {
		SMTPServer = originalSMTPServer
		SMTPPort = originalSMTPPort
		SMTPSSLEnabled = originalSMTPSSLEnabled
		SMTPStartTLSEnabled = originalSMTPStartTLSEnabled
		SMTPInsecureSkipVerify = originalSMTPInsecureSkipVerify
		SMTPForceAuthLogin = originalSMTPForceAuthLogin
		SMTPAccount = originalSMTPAccount
		SMTPFrom = originalSMTPFrom
		SMTPToken = originalSMTPToken
		SystemName = originalSystemName
	})
}

func TestSendEmailUsesExplicitStartTLSWithInsecureCertificate(t *testing.T) {
	server := newFakeSMTPServer(t)
	defer server.close()
	withSMTPSettings(t)

	SMTPServer = server.host
	SMTPPort = server.port
	SMTPSSLEnabled = false
	SMTPStartTLSEnabled = true
	SMTPInsecureSkipVerify = true
	SMTPForceAuthLogin = false
	SMTPAccount = "sender@example.com"
	SMTPFrom = "sender@example.com"
	SMTPToken = "secret"
	SystemName = "New API"

	err := SendEmail("Verification", "receiver@example.com", "<p>123456</p>")
	require.NoError(t, err)

	select {
	case message := <-server.messages:
		require.Contains(t, message, "Subject: =?UTF-8?B?")
		require.Contains(t, message, "<p>123456</p>")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SMTP DATA")
	}
}

func TestSendEmailExplicitStartTLSRequiresServerSupport(t *testing.T) {
	server := newFakeSMTPServerWithSTARTTLSAdvertisement(t, false)
	defer server.close()
	withSMTPSettings(t)

	SMTPServer = server.host
	SMTPPort = server.port
	SMTPSSLEnabled = false
	SMTPStartTLSEnabled = true
	SMTPInsecureSkipVerify = true
	SMTPForceAuthLogin = false
	SMTPAccount = "sender@example.com"
	SMTPFrom = "sender@example.com"
	SMTPToken = "secret"
	SystemName = "New API"

	err := SendEmail("Verification", "receiver@example.com", "<p>123456</p>")
	require.Error(t, err)
	require.Contains(t, err.Error(), "STARTTLS")
}

func TestSendEmailDoesNotAutoUpgradeWhenStartTLSDisabled(t *testing.T) {
	server := newFakeSMTPServerWithSTARTTLSAdvertisement(t, true)
	defer server.close()
	withSMTPSettings(t)

	SMTPServer = server.host
	SMTPPort = server.port
	SMTPSSLEnabled = false
	SMTPStartTLSEnabled = false
	SMTPInsecureSkipVerify = false
	SMTPForceAuthLogin = false
	SMTPAccount = "sender@example.com"
	SMTPFrom = "sender@example.com"
	SMTPToken = "secret"
	SystemName = "New API"

	err := SendEmail("Verification", "receiver@example.com", "<p>123456</p>")
	require.NoError(t, err)

	select {
	case command := <-server.startTLSCommands:
		t.Fatalf("unexpected SMTP STARTTLS command: %s", command)
	default:
	}

	select {
	case message := <-server.messages:
		require.Contains(t, message, "<p>123456</p>")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SMTP DATA")
	}
}

func TestSMTPPlainAuthRejectsRemotePlaintextConnection(t *testing.T) {
	server := newFakeSMTPServerWithSTARTTLSAdvertisement(t, false)
	defer server.close()
	withSMTPSettings(t)

	SMTPServer = "smtp.example.com"
	SMTPPort = server.port
	SMTPSSLEnabled = false
	SMTPStartTLSEnabled = false
	SMTPInsecureSkipVerify = false
	SMTPForceAuthLogin = false
	SMTPAccount = "sender@example.com"
	SMTPFrom = "sender@example.com"
	SMTPToken = "secret"

	conn, err := net.Dial("tcp", net.JoinHostPort(server.host, strconv.Itoa(server.port)))
	require.NoError(t, err)
	client, err := smtp.NewClient(conn, SMTPServer)
	require.NoError(t, err)

	err = client.Auth(getSMTPAuth())
	require.Error(t, err)
	require.Contains(t, err.Error(), "unencrypted connection")

	select {
	case command := <-server.authCommands:
		t.Fatalf("unexpected SMTP auth command: %s", command)
	default:
	}
}

func TestNewSMTPClientHonorsExplicitStartTLSWhenPortIs465(t *testing.T) {
	server := newFakeSMTPServer(t)
	defer server.close()
	withSMTPSettings(t)

	SMTPServer = server.host
	SMTPPort = 465
	SMTPSSLEnabled = false
	SMTPStartTLSEnabled = true
	SMTPInsecureSkipVerify = true

	client, err := newSMTPClient(fmt.Sprintf("%s:%d", server.host, server.port))
	require.NoError(t, err)
	defer client.Close()

	select {
	case command := <-server.startTLSCommands:
		require.Equal(t, "STARTTLS", command)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SMTP STARTTLS")
	}
}

func TestNewSMTPClientKeepsImplicitTLSForLegacyPort465(t *testing.T) {
	server := newFakeImplicitTLSSMTPServer(t)
	defer server.close()
	withSMTPSettings(t)

	SMTPServer = server.host
	SMTPPort = 465
	SMTPSSLEnabled = false
	SMTPStartTLSEnabled = false
	SMTPInsecureSkipVerify = true

	client, err := newSMTPClient(fmt.Sprintf("%s:%d", server.host, server.port))
	require.NoError(t, err)
	defer client.Close()
}

func TestSendEmailSkipsAuthWhenCredentialsAreEmpty(t *testing.T) {
	server := newFakeSMTPServerWithSTARTTLSAdvertisement(t, false)
	defer server.close()
	withSMTPSettings(t)

	SMTPServer = server.host
	SMTPPort = server.port
	SMTPSSLEnabled = false
	SMTPStartTLSEnabled = false
	SMTPInsecureSkipVerify = false
	SMTPForceAuthLogin = false
	SMTPAccount = ""
	SMTPFrom = "sender@example.com"
	SMTPToken = ""
	SystemName = "New API"

	err := SendEmail("Verification", "receiver@example.com", "<p>123456</p>")
	require.NoError(t, err)

	select {
	case command := <-server.authCommands:
		t.Fatalf("unexpected SMTP auth command: %s", command)
	default:
	}

	select {
	case message := <-server.messages:
		require.Contains(t, message, "<p>123456</p>")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SMTP DATA")
	}
}

func TestSendEmailSkipsAuthWhenCredentialsAreIncomplete(t *testing.T) {
	server := newFakeSMTPServerWithSTARTTLSAdvertisement(t, false)
	defer server.close()
	withSMTPSettings(t)

	SMTPServer = server.host
	SMTPPort = server.port
	SMTPSSLEnabled = false
	SMTPStartTLSEnabled = false
	SMTPInsecureSkipVerify = false
	SMTPForceAuthLogin = false
	SMTPAccount = "sender@example.com"
	SMTPFrom = "sender@example.com"
	SMTPToken = ""
	SystemName = "New API"

	err := SendEmail("Verification", "receiver@example.com", "<p>123456</p>")
	require.NoError(t, err)

	select {
	case command := <-server.authCommands:
		t.Fatalf("unexpected SMTP auth command: %s", command)
	default:
	}

	select {
	case message := <-server.messages:
		require.Contains(t, message, "<p>123456</p>")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SMTP DATA")
	}
}

func TestSendEmailUsesNTLMWhenServerOnlySupportsNTLM(t *testing.T) {
	server := newFakeSMTPServer(t)
	server.authMechanisms = []string{"NTLM"}
	defer server.close()
	withSMTPSettings(t)

	SMTPServer = server.host
	SMTPPort = server.port
	SMTPSSLEnabled = false
	SMTPStartTLSEnabled = true
	SMTPInsecureSkipVerify = true
	SMTPForceAuthLogin = false
	SMTPAccount = "no-reply"
	SMTPFrom = "no-reply@example.com"
	SMTPToken = "secret"
	SystemName = "New API"

	err := SendEmail("Verification", "receiver@example.com", "<p>123456</p>")
	require.NoError(t, err)

	select {
	case command := <-server.authCommands:
		require.True(t, strings.HasPrefix(command, "AUTH NTLM "), "unexpected auth command: %s", command)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SMTP AUTH")
	}
}

func TestSendEmailUsesNTLMForMicrosoftAccountWhenServerOnlySupportsNTLM(t *testing.T) {
	server := newFakeSMTPServer(t)
	server.authMechanisms = []string{"NTLM"}
	defer server.close()
	withSMTPSettings(t)

	SMTPServer = server.host
	SMTPPort = server.port
	SMTPSSLEnabled = false
	SMTPStartTLSEnabled = true
	SMTPInsecureSkipVerify = true
	SMTPForceAuthLogin = false
	SMTPAccount = "no-reply@contoso.onmicrosoft.com"
	SMTPFrom = "no-reply@contoso.onmicrosoft.com"
	SMTPToken = "secret"
	SystemName = "New API"

	err := SendEmail("Verification", "receiver@example.com", "<p>123456</p>")
	require.NoError(t, err)

	select {
	case command := <-server.authCommands:
		require.True(t, strings.HasPrefix(command, "AUTH NTLM "), "unexpected auth command: %s", command)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SMTP AUTH")
	}
}

func TestSendEmailExplicitStartTLSRejectsUntrustedCertificateByDefault(t *testing.T) {
	server := newFakeSMTPServer(t)
	defer server.close()
	withSMTPSettings(t)

	SMTPServer = server.host
	SMTPPort = server.port
	SMTPSSLEnabled = false
	SMTPStartTLSEnabled = true
	SMTPInsecureSkipVerify = false
	SMTPForceAuthLogin = false
	SMTPAccount = "sender@example.com"
	SMTPFrom = "sender@example.com"
	SMTPToken = "secret"
	SystemName = "New API"

	err := SendEmail("Verification", "receiver@example.com", "<p>123456</p>")
	require.Error(t, err)
	require.Contains(t, fmt.Sprint(err), "certificate")
}

func configureCampaignSMTP(t *testing.T, server *fakeSMTPServer) CampaignEmailMessage {
	t.Helper()
	withSMTPSettings(t)
	SMTPServer, SMTPPort = server.host, server.port
	SMTPAccount, SMTPToken = "", ""
	SMTPFrom, SystemName = "sender@example.com", "New API"
	SMTPSSLEnabled, SMTPStartTLSEnabled = false, false
	SMTPInsecureSkipVerify, SMTPForceAuthLogin = false, false
	return CampaignEmailMessage{
		Subject:        "平台通知 · Platform notice",
		Receiver:       "Receiver <receiver@example.com>",
		HTMLBody:       "<p>平台通知 &amp; update</p>",
		TextBody:       "平台通知 & update",
		MessageID:      "<campaign.42.delivery.7@example.com>",
		UnsubscribeURL: "https://example.com/api/email/unsubscribe?token=private-token",
	}
}

func TestSendCampaignEmailAcceptedDespiteQuitFailure(t *testing.T) {
	server := newFakeSMTPServerWithSTARTTLSAdvertisement(t, false, fakeSMTPBehavior{quitReply: 421})
	defer server.close()
	message := configureCampaignSMTP(t, server)
	require.NoError(t, SendCampaignEmail(context.Background(), message))
	var raw string
	select {
	case raw = <-server.messages:
	case <-time.After(2 * time.Second):
		t.Fatal("missing accepted message")
	}
	parsed, err := mail.ReadMessage(strings.NewReader(raw))
	require.NoError(t, err)
	assert.Equal(t, message.MessageID, parsed.Header.Get("Message-ID"))
	assert.Equal(t, "<"+message.UnsubscribeURL+">", parsed.Header.Get("List-Unsubscribe"))
	assert.Equal(t, "List-Unsubscribe=One-Click", parsed.Header.Get("List-Unsubscribe-Post"))
	assert.Empty(t, parsed.Header.Get("Bcc"))
	assert.Empty(t, parsed.Header.Get("DKIM-Signature"))
	subject, err := (&mime.WordDecoder{}).DecodeHeader(parsed.Header.Get("Subject"))
	require.NoError(t, err)
	assert.Equal(t, message.Subject, subject)
	contentType, params, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
	require.NoError(t, err)
	assert.Equal(t, "multipart/alternative", contentType)
	parts := multipart.NewReader(parsed.Body, params["boundary"])
	for _, expected := range []struct{ contentType, text string }{{"text/plain; charset=UTF-8", message.TextBody}, {"text/html; charset=UTF-8", message.HTMLBody}} {
		part, err := parts.NextPart()
		require.NoError(t, err)
		assert.Equal(t, expected.contentType, part.Header.Get("Content-Type"))
		decoded, err := io.ReadAll(part)
		require.NoError(t, err)
		assert.Equal(t, expected.text, string(decoded))
	}
	_, err = parts.NextPart()
	assert.ErrorIs(t, err, io.EOF)
}

func TestSendCampaignEmailRejectsInvalidHeadersBeforeDial(t *testing.T) {
	withSMTPSettings(t)
	SMTPServer, SMTPPort = "127.0.0.1", 1
	SMTPFrom = "sender@example.com"
	for _, tc := range []struct {
		name, subject, receiver, messageID, unsubscribe, phase string
	}{
		{name: "subject injection", subject: "hi\r\nBcc: victim@example.com", phase: "headers"},
		{name: "recipient injection", receiver: "one@example.com\nBcc: victim@example.com", phase: "headers"},
		{name: "multiple recipients", receiver: "one@example.com,two@example.com", phase: "recipient"},
		{name: "semicolon recipients", receiver: "one@example.com;two@example.com", phase: "recipient"},
		{name: "id injection", messageID: "<id@example.com>\r\nBcc: victim@example.com", phase: "headers"},
		{name: "invalid id", messageID: "id@example.com", phase: "message-id"},
		{name: "url injection", unsubscribe: "https://example.com/\r\nBcc: victim@example.com", phase: "headers"},
		{name: "http url", unsubscribe: "http://example.com/unsubscribe", phase: "unsubscribe-url"},
		{name: "credential url", unsubscribe: "https://user:password@example.com/unsubscribe", phase: "unsubscribe-url"},
		{name: "url fragment", unsubscribe: "https://example.com/unsubscribe#token", phase: "unsubscribe-url"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			message := CampaignEmailMessage{Subject: "Notice", Receiver: "receiver@example.com", TextBody: "body", MessageID: "<id@example.com>", UnsubscribeURL: "https://example.com/unsubscribe"}
			if tc.subject != "" {
				message.Subject = tc.subject
			}
			if tc.receiver != "" {
				message.Receiver = tc.receiver
			}
			if tc.messageID != "" {
				message.MessageID = tc.messageID
			}
			if tc.unsubscribe != "" {
				message.UnsubscribeURL = tc.unsubscribe
			}
			var deliveryError *CampaignEmailDeliveryError
			require.ErrorAs(t, SendCampaignEmail(context.Background(), message), &deliveryError)
			assert.Equal(t, tc.phase, deliveryError.Phase)
			assert.False(t, deliveryError.Temporary)
			assert.False(t, deliveryError.Uncertain)
		})
	}
	SMTPServer = ""
	var deliveryError *CampaignEmailDeliveryError
	require.ErrorAs(t, SendCampaignEmail(context.Background(), CampaignEmailMessage{}), &deliveryError)
	assert.Equal(t, "configuration", deliveryError.Phase)
}

func TestSendCampaignEmailDeliveryClassification(t *testing.T) {
	for _, tc := range []struct {
		name      string
		behavior  fakeSMTPBehavior
		code      int
		temporary bool
		uncertain bool
	}{
		{name: "recipient transient", behavior: fakeSMTPBehavior{recipientReply: 450}, code: 450, temporary: true},
		{name: "recipient permanent", behavior: fakeSMTPBehavior{recipientReply: 550}, code: 550},
		{name: "data transient", behavior: fakeSMTPBehavior{dataReply: 451}, code: 451, temporary: true},
		{name: "data permanent", behavior: fakeSMTPBehavior{dataReply: 554}, code: 554},
		{name: "data reply lost", behavior: fakeSMTPBehavior{closeAfterData: true}, uncertain: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := newFakeSMTPServerWithSTARTTLSAdvertisement(t, false, tc.behavior)
			defer server.close()
			message := configureCampaignSMTP(t, server)
			var deliveryError *CampaignEmailDeliveryError
			require.ErrorAs(t, SendCampaignEmail(context.Background(), message), &deliveryError)
			assert.Equal(t, tc.code, deliveryError.Code)
			assert.Equal(t, tc.temporary, deliveryError.Temporary)
			assert.Equal(t, tc.uncertain, deliveryError.Uncertain)
			assert.NotContains(t, deliveryError.Error(), "sensitive-server-detail")
			assert.NotContains(t, deliveryError.Error(), "private-token")
		})
	}
}

func TestSendCampaignEmailCancellationClosesSocket(t *testing.T) {
	for _, dataStarted := range []bool{false, true} {
		t.Run(fmt.Sprintf("data_started_%t", dataStarted), func(t *testing.T) {
			behavior := fakeSMTPBehavior{blockGreeting: !dataStarted, blockDataReply: dataStarted, connected: make(chan struct{}), disconnected: make(chan struct{})}
			server := newFakeSMTPServerWithSTARTTLSAdvertisement(t, false, behavior)
			defer server.close()
			message := configureCampaignSMTP(t, server)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() { result <- SendCampaignEmail(ctx, message) }()
			if dataStarted {
				select {
				case <-server.messages:
				case <-time.After(2 * time.Second):
					t.Fatal("message did not reach DATA")
				}
			} else {
				select {
				case <-behavior.connected:
				case <-time.After(2 * time.Second):
					t.Fatal("client did not connect")
				}
			}
			cancel()
			select {
			case err := <-result:
				var deliveryError *CampaignEmailDeliveryError
				require.ErrorAs(t, err, &deliveryError)
				assert.Equal(t, !dataStarted, deliveryError.Temporary)
				assert.Equal(t, dataStarted, deliveryError.Uncertain)
			case <-time.After(2 * time.Second):
				t.Fatal("cancellation did not interrupt SMTP")
			}
			select {
			case <-behavior.disconnected:
			case <-time.After(2 * time.Second):
				t.Fatal("cancelled SMTP socket remained open")
			}
		})
	}
}

func TestSendCampaignEmailDeadlineCoversGreeting(t *testing.T) {
	server := newFakeSMTPServerWithSTARTTLSAdvertisement(t, false, fakeSMTPBehavior{blockGreeting: true})
	defer server.close()
	message := configureCampaignSMTP(t, server)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	var deliveryError *CampaignEmailDeliveryError
	require.ErrorAs(t, SendCampaignEmail(ctx, message), &deliveryError)
	assert.True(t, deliveryError.Temporary)
	assert.False(t, deliveryError.Uncertain)
}

func TestSendCampaignEmailPreservesTLSConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name       string
		implicit   bool
		skipVerify bool
		wantPhase  string
	}{
		{name: "starttls", skipVerify: true},
		{name: "implicit TLS", implicit: true, skipVerify: true},
		{name: "verify untrusted certificate", wantPhase: "starttls"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var server *fakeSMTPServer
			if tc.implicit {
				server = newFakeImplicitTLSSMTPServer(t)
			} else {
				server = newFakeSMTPServer(t)
			}
			defer server.close()
			message := configureCampaignSMTP(t, server)
			SMTPSSLEnabled, SMTPStartTLSEnabled = tc.implicit, !tc.implicit
			SMTPInsecureSkipVerify = tc.skipVerify
			SMTPAccount, SMTPToken = "sender@example.com", "secret"
			err := SendCampaignEmail(context.Background(), message)
			if tc.wantPhase == "" {
				require.NoError(t, err)
			} else {
				var deliveryError *CampaignEmailDeliveryError
				require.ErrorAs(t, err, &deliveryError)
				assert.Equal(t, tc.wantPhase, deliveryError.Phase)
				assert.False(t, deliveryError.Uncertain)
			}
		})
	}
}

func TestCampaignSMTPAuthUsesConfigurationSnapshot(t *testing.T) {
	for _, forceLogin := range []bool{false, true} {
		t.Run(fmt.Sprintf("force_login_%t", forceLogin), func(t *testing.T) {
			withSMTPSettings(t)
			SMTPServer = "smtp.original.example.com"
			SMTPAccount, SMTPToken = "original@example.com", "original-secret"
			SMTPForceAuthLogin = forceLogin
			auth := snapshotSMTPAuth()
			SMTPServer = "smtp.changed.example.com"
			SMTPAccount, SMTPToken = "changed@example.com", "changed-secret"
			SMTPForceAuthLogin = !forceLogin
			mechanism, response, err := auth.Start(&smtp.ServerInfo{Name: "smtp.original.example.com", TLS: true, Auth: []string{"PLAIN", "LOGIN"}})
			require.NoError(t, err)
			if forceLogin {
				assert.Equal(t, "LOGIN", mechanism)
				response, err = auth.Next([]byte("Username:"), true)
				require.NoError(t, err)
				assert.Equal(t, "original@example.com", string(response))
				response, err = auth.Next([]byte("Password:"), true)
				require.NoError(t, err)
				assert.Equal(t, "original-secret", string(response))
			} else {
				assert.Equal(t, "PLAIN", mechanism)
				assert.Equal(t, "\x00original@example.com\x00original-secret", string(response))
			}
		})
	}
}
