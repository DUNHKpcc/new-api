package controller

import (
	"fmt"
	"html"
)

func renderEmailVerificationContent(systemName, code, logoURL string, validMinutes int) string {
	systemName = html.EscapeString(systemName)
	code = html.EscapeString(code)
	logoURL = html.EscapeString(logoURL)

	return fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="color-scheme" content="light">
  <title>DPCC API 邮箱验证</title>
</head>
<body style="margin:0;padding:0;background:#faf9f5;color:#141413;font-family:Arial,'PingFang SC','Microsoft YaHei',sans-serif;">
  <div style="display:none;max-height:0;overflow:hidden;opacity:0;">你的 DPCC API 邮箱验证码是 %[2]s，%[4]d 分钟内有效。</div>
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="width:100%%;background:#faf9f5;">
    <tr>
      <td align="center" style="padding:40px 16px;">
        <table role="presentation" width="560" cellpadding="0" cellspacing="0" border="0" style="width:100%%;max-width:560px;border:1px solid #e8e6dc;background:#efe9de;">
          <tr>
            <td style="height:4px;background:#d97757;font-size:0;line-height:0;">&nbsp;</td>
          </tr>
          <tr>
            <td style="padding:32px 36px 24px;">
              <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0">
                <tr>
                  <td width="56" valign="middle">
                    <img src="%[3]s" width="48" height="48" alt="DPCC API" style="display:block;width:48px;height:48px;border:0;">
                  </td>
                  <td valign="middle" style="padding-left:12px;font-family:Georgia,'Times New Roman',serif;font-size:20px;font-weight:bold;color:#141413;">
                    DPCC API
                  </td>
                </tr>
              </table>
            </td>
          </tr>
          <tr>
            <td style="padding:0 36px 36px;">
              <h1 style="margin:0 0 14px;font-family:Georgia,'Times New Roman','Songti SC',serif;font-size:30px;line-height:1.3;font-weight:normal;color:#141413;">验证你的邮箱</h1>
              <p style="margin:0 0 28px;font-size:15px;line-height:1.8;color:#5e5d59;">请使用下方验证码完成 DPCC API 的邮箱验证。</p>

              <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0">
                <tr>
                  <td align="center" style="padding:20px 16px;background:#d97757;color:#fffaf7;font-family:'Courier New',monospace;font-size:32px;line-height:1.2;font-weight:bold;letter-spacing:8px;">
                    %[2]s
                  </td>
                </tr>
              </table>

              <p style="margin:24px 0 0;font-size:14px;line-height:1.8;color:#5e5d59;">验证码将在 <strong style="color:#141413;">%[4]d 分钟</strong>后失效。请勿向任何人透露此验证码。</p>
              <p style="margin:10px 0 0;font-size:14px;line-height:1.8;color:#5e5d59;">如果这不是你的操作，可以安全地忽略此邮件。</p>
            </td>
          </tr>
          <tr>
            <td style="border-top:1px solid #ded8cd;padding:20px 36px;font-size:12px;line-height:1.6;color:#77756f;">
              DPCC API 基于 %[1]s 提供。此邮件由系统自动发送，请勿回复。
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, systemName, code, logoURL, validMinutes)
}
