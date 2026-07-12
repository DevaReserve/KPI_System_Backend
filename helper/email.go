package helper

import (
	"KPI_System_Backend/config"
	"KPI_System_Backend/logger"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"math/big"
	"net/smtp"
	"strings"

	"go.uber.org/zap"
)

// SendEmailNotification bertugas mengirimkan email berformat HTML ke pegawai.
// Sekarang membaca konfigurasi SMTP dari config package (Setting.ini).
func SendEmailNotification(toEmail string, subject string, htmlBody string) error {
	smtpHost := config.SMTPHost
	smtpPort := config.SMTPPort
	senderEmail := config.SMTPSenderEmail
	senderPass := config.SMTPSenderPassword
	senderName := config.SMTPSenderName

	if senderEmail == "" || senderPass == "" {
		logger.Error("SMTP credentials not configured. Check Setting.ini [SMTPConfig].")
		return fmt.Errorf("SMTP tidak dikonfigurasi")
	}

	if !strings.Contains(smtpHost, ".") {
		logger.Error("SMTP host tidak valid", zap.String("host", smtpHost))
		return fmt.Errorf("SMTP host tidak valid")
	}

	fromHeader := fmt.Sprintf("From: %s <%s>", senderName, senderEmail)
	toHeader := fmt.Sprintf("To: %s", toEmail)
	subjectHeader := fmt.Sprintf("Subject: %s", subject)
	mimeHeaders := "MIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n"
	message := []byte(strings.Join([]string{fromHeader, toHeader, subjectHeader, mimeHeaders, "", htmlBody}, "\r\n"))

	client, err := smtp.Dial(smtpHost + ":" + smtpPort)
	if err != nil {
		logger.Error("Gagal menghubungi SMTP server", zap.Error(err))
		return err
	}
	defer client.Close()

	if err = client.Hello("localhost"); err != nil {
		return err
	}

	if err = client.StartTLS(&tls.Config{InsecureSkipVerify: true, ServerName: smtpHost}); err != nil {
		return err
	}

	auth := smtp.PlainAuth("", senderEmail, senderPass, smtpHost)
	if err = client.Auth(auth); err != nil {
		logger.Error("Gagal autentikasi SMTP", zap.Error(err))
		return err
	}

	if err = client.Mail(senderEmail); err != nil {
		return err
	}
	if err = client.Rcpt(toEmail); err != nil {
		return err
	}
	wc, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = wc.Write(message); err != nil {
		wc.Close()
		return err
	}
	if err = wc.Close(); err != nil {
		return err
	}

	logger.Info("Email berhasil dikirim", zap.String("to", toEmail), zap.String("subject", subject))
	return nil
}

// GenerateOTP menghasilkan 6 digit OTP acak menggunakan crypto/rand (aman secara kriptografi).
func GenerateOTP() (string, error) {
	// Menggunakan crypto/rand bukan math/rand untuk keamanan
	max := big.NewInt(999999)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	// Format dengan leading zeros agar selalu 6 digit (misal: 007234)
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// SendOTPEmail mengirim email OTP dengan template HTML yang rapi.
func SendOTPEmail(toEmail string, otpCode string) error {
	subject := "Kode OTP Reset Password - KPI System"

	htmlBody := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
	</head>
	<body style="margin:0;padding:0;font-family:'Segoe UI',Arial,sans-serif;background-color:#f0f4f8;">
		<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="min-width:100%%;background-color:#f0f4f8;padding:40px 20px;">
			<tr>
				<td align="center">
					<table role="presentation" width="480" cellpadding="0" cellspacing="0" style="max-width:480px;width:100%%;background-color:#ffffff;border-radius:16px;overflow:hidden;box-shadow:0 4px 24px rgba(0,0,0,0.08);">
						<!-- Header -->
						<tr>
							<td style="background:linear-gradient(135deg,#1e293b 0%%,#334155 100%%);padding:32px 40px;text-align:center;">
								<h1 style="margin:0;font-size:22px;font-weight:700;color:#ffffff;letter-spacing:-0.5px;">KPI System</h1>
								<p style="margin:6px 0 0;font-size:13px;color:#94a3b8;">PT. Cakra Media Data</p>
							</td>
						</tr>
						<!-- Body -->
						<tr>
							<td style="padding:40px;">
								<h2 style="margin:0 0 8px;font-size:20px;font-weight:700;color:#1e293b;">Reset Password</h2>
								<p style="margin:0 0 24px;font-size:14px;color:#64748b;line-height:1.6;">
									Anda menerima email ini karena ada permintaan reset password untuk akun Anda. Gunakan kode OTP berikut:
								</p>
								<!-- OTP Box -->
								<div style="background:#f1f5f9;border:2px dashed #cbd5e1;border-radius:12px;padding:24px;text-align:center;margin:0 0 24px;">
									<span style="font-size:36px;font-weight:800;letter-spacing:12px;color:#1e293b;font-family:'Courier New',monospace;">%s</span>
								</div>
								<p style="margin:0 0 6px;font-size:13px;color:#ef4444;font-weight:600;">
									⏰ Kode ini berlaku selama 10 menit.
								</p>
								<p style="margin:0 0 24px;font-size:13px;color:#64748b;">
									Jika Anda tidak merasa meminta reset password, abaikan email ini. Akun Anda tetap aman.
								</p>
								<hr style="border:none;border-top:1px solid #e2e8f0;margin:24px 0;">
								<p style="margin:0;font-size:12px;color:#94a3b8;text-align:center;">
									Email ini dikirim secara otomatis oleh sistem. Mohon tidak membalas email ini.
								</p>
							</td>
						</tr>
					</table>
				</td>
			</tr>
		</table>
	</body>
	</html>`, otpCode)

	return SendEmailNotification(toEmail, subject, htmlBody)
}
