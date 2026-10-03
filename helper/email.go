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
	"time"

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

// SendPhoneOTPEmail mengirim email kode OTP untuk verifikasi nomor telepon / WhatsApp.
func SendPhoneOTPEmail(toEmail string, otpCode string, phone string) error {
	subject := "Verifikasi Nomor Telepon - KPI System"

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
							<td style="background:linear-gradient(135deg,#2563eb 0%%,#1d4ed8 100%%);padding:32px 40px;text-align:center;">
								<h1 style="margin:0;font-size:22px;font-weight:700;color:#ffffff;letter-spacing:-0.5px;">Verifikasi Kontak</h1>
								<p style="margin:6px 0 0;font-size:13px;color:#bfdbfe;">PT. Cakra Media Data</p>
							</td>
						</tr>
						<!-- Body -->
						<tr>
							<td style="padding:40px;">
								<h2 style="margin:0 0 8px;font-size:20px;font-weight:700;color:#1e293b;">Kode OTP Verifikasi</h2>
								<p style="margin:0 0 24px;font-size:14px;color:#64748b;line-height:1.6;">
									Anda mengajukan verifikasi nomor kontak <b>%s</b> pada aplikasi KPI System. Gunakan kode verifikasi OTP berikut untuk menyelesaikan proses verifikasi:
								</p>
								<!-- OTP Box -->
								<div style="background:#f1f5f9;border:2px dashed #cbd5e1;border-radius:12px;padding:24px;text-align:center;margin:0 0 24px;">
									<span style="font-size:36px;font-weight:800;letter-spacing:12px;color:#2563eb;font-family:'Courier New',monospace;">%s</span>
								</div>
								<p style="margin:0 0 6px;font-size:13px;color:#ef4444;font-weight:600;">
									⏰ Kode verifikasi ini berlaku selama 10 menit.
								</p>
								<p style="margin:0 0 24px;font-size:13px;color:#64748b;">
									Verifikasi ini penting untuk memastikan nomor kontak Anda terdaftar secara valid sehingga Anda dapat menerima notifikasi WhatsApp resmi dari perusahaan.
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
	</html>`, phone, otpCode)

	return SendEmailNotification(toEmail, subject, htmlBody)
}

// SendWarningEmail mengirim email notifikasi resmi Surat Peringatan (SP) kepada pegawai.
func SendWarningEmail(toEmail string, recipientName string, spLevel string, reason string, description string) error {
	subject := fmt.Sprintf("Pemberitahuan Resmi: Penerbitan %s - PT. Cakra Media Data", spLevel)
	
	colorTheme := "#3b82f6" // Blue for SP1
	if spLevel == "SP2" {
		colorTheme = "#f97316" // Orange for SP2
	} else if spLevel == "SP3" {
		colorTheme = "#ef4444" // Red for SP3
	}

	issuedDate := time.Now().Format("02 Jan 2006")

	htmlBody := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
	</head>
	<body style="margin:0;padding:0;font-family:'Segoe UI',Arial,sans-serif;background-color:#f8fafc;">
		<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="min-width:100%%;background-color:#f8fafc;padding:40px 20px;">
			<tr>
				<td align="center">
					<table role="presentation" width="520" cellpadding="0" cellspacing="0" style="max-width:520px;width:100%%;background-color:#ffffff;border-radius:16px;overflow:hidden;box-shadow:0 4px 24px rgba(0,0,0,0.06);border-top: 6px solid %s;">
						<!-- Header -->
						<tr>
							<td style="padding:32px 40px 20px;text-align:center;">
								<h1 style="margin:0;font-size:24px;font-weight:800;color:#1e293b;letter-spacing:-0.5px;">SURAT PERINGATAN</h1>
								<p style="margin:4px 0 0;font-size:13px;color:#64748b;font-weight:600;text-transform:uppercase;letter-spacing:1px;">%s</p>
							</td>
						</tr>
						<!-- Body -->
						<tr>
							<td style="padding:0 40px 40px;">
								<p style="margin:0 0 16px;font-size:14px;color:#334155;line-height:1.6;">
									Kepada Yth. <br>
									<b style="color:#1e293b;font-size:15px;">%s</b>
								</p>
								<p style="margin:0 0 20px;font-size:14px;color:#334155;line-height:1.6;">
									Manajemen PT. Cakra Media Data menyampaikan pemberitahuan resmi bahwa perusahaan telah menerbitkan <b>%s</b> atas nama Anda, terhitung sejak tanggal dokumen ini dikirim.
								</p>
								
								<!-- Detail Box -->
								<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f8fafc;border:1px solid #e2e8f0;border-radius:12px;padding:20px;margin-bottom:24px;">
									<tr>
										<td style="padding-bottom:12px;">
											<span style="font-size:11px;font-weight:700;color:#64748b;text-transform:uppercase;display:block;">Tanggal Diterbitkan:</span>
											<span style="font-size:13.5px;font-weight:700;color:#1e293b;line-height:1.5;display:block;margin-top:2px;">%s</span>
										</td>
									</tr>
									<tr>
										<td style="border-top:1px solid #e2e8f0;padding-top:12px;padding-bottom:12px;">
											<span style="font-size:11px;font-weight:700;color:#64748b;text-transform:uppercase;display:block;">Alasan Pelanggaran / Masalah Kinerja:</span>
											<span style="font-size:13.5px;font-weight:700;color:#1e293b;line-height:1.5;display:block;margin-top:2px;">%s</span>
										</td>
									</tr>
									<tr>
										<td style="border-top:1px solid #e2e8f0;padding-top:12px;">
											<span style="font-size:11px;font-weight:700;color:#64748b;text-transform:uppercase;display:block;">Catatan Tambahan / Keterangan:</span>
											<span style="font-size:13px;font-style:italic;color:#334155;line-height:1.5;display:block;margin-top:2px;">"%s"</span>
										</td>
									</tr>
								</table>

								<p style="margin:0 0 20px;font-size:13px;color:#475569;line-height:1.6;">
									Silakan login ke akun KPI System Anda dan buka menu <b>"Riwayat SP"</b> untuk meninjau secara menyeluruh serta mengunduh salinan resmi dokumen PDF Surat Peringatan tersebut.
								</p>
								
								<p style="margin:0 0 24px;font-size:13px;color:#475569;line-height:1.6;">
									Manajemen berharap Anda dapat segera melakukan perbaikan kinerja dan koordinasi dengan atasan langsung.
								</p>

								<hr style="border:none;border-top:1px solid #e2e8f0;margin:24px 0;">
								<p style="margin:0;font-size:11px;color:#94a3b8;text-align:center;">
									PT. Cakra Media Data • Jl. Raya Mambal Ubud, Badung, Bali <br>
									Email ini dikirim secara otomatis oleh sistem. Mohon tidak membalas email ini.
								</p>
							</td>
						</tr>
					</table>
				</td>
			</tr>
		</table>
	</body>
	</html>`, colorTheme, spLevel, recipientName, spLevel, issuedDate, reason, description)

	return SendEmailNotification(toEmail, subject, htmlBody)
}

// SendNewAccountEmail mengirimkan notifikasi ke pegawai baru bahwa akun telah dibuat
// dan menganjurkan mereka untuk segera melengkapi data diri di menu Profil Saya.
func SendNewAccountEmail(toEmail string, employeeName string, username string, tempPassword string) error {
	subject := "Selamat Datang di PT Cakra Media Data - Akun KPI System Anda Telah Dibuat"

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
					<table role="presentation" width="520" cellpadding="0" cellspacing="0" style="max-width:520px;width:100%%;background-color:#ffffff;border-radius:16px;overflow:hidden;box-shadow:0 4px 24px rgba(0,0,0,0.08);">
						<!-- Header -->
						<tr>
							<td style="background:linear-gradient(135deg,#1e3a8a 0%%,#3b82f6 100%%);padding:32px 40px;text-align:center;">
								<h1 style="margin:0;font-size:22px;font-weight:700;color:#ffffff;letter-spacing:-0.5px;">Selamat Datang!</h1>
								<p style="margin:6px 0 0;font-size:13px;color:#dbeafe;">PT. Cakra Media Data • KPI & Performance System</p>
							</td>
						</tr>
						<!-- Body -->
						<tr>
							<td style="padding:40px;">
								<h2 style="margin:0 0 12px;font-size:18px;font-weight:700;color:#1e293b;">Halo, %s! 👋</h2>
								<p style="margin:0 0 20px;font-size:14px;color:#475569;line-height:1.6;">
									Selamat bergabung dengan <b>PT Cakra Media Data</b>. Akun Anda untuk mengakses Aplikasi Sistem Manajemen Kinerja dan KPI Pegawai telah berhasil dibuat oleh Admin HRD.
								</p>

								<!-- Kredensial Login -->
								<div style="background:#f8fafc;border:1px solid #e2e8f0;border-left:4px solid #3b82f6;border-radius:10px;padding:20px;margin-bottom:24px;">
									<span style="font-size:11px;font-weight:700;color:#64748b;text-transform:uppercase;display:block;margin-bottom:8px;">Informasi Akun Login Anda:</span>
									<table role="presentation" width="100%%" cellpadding="0" cellspacing="0">
										<tr>
											<td width="130" style="font-size:13px;color:#64748b;padding:4px 0;">Username:</td>
											<td style="font-size:14px;font-weight:700;color:#1e293b;padding:4px 0;">%s</td>
										</tr>
										<tr>
											<td width="130" style="font-size:13px;color:#64748b;padding:4px 0;">Password Sementara:</td>
											<td style="font-size:14px;font-weight:700;color:#2563eb;padding:4px 0;font-family:monospace;">%s</td>
										</tr>
									</table>
								</div>

								<!-- Instruksi Penting & Lengkapi Data Diri -->
								<div style="background:#fffbeb;border:1px solid #fde68a;border-radius:12px;padding:20px;margin-bottom:24px;">
									<h3 style="margin:0 0 10px;font-size:14px;font-weight:700;color:#92400e;display:flex;align-items:center;">
										⚠️ TINDAKAN PENTING YANG HARUS SEGERA DILAKUKAN:
									</h3>
									<ol style="margin:0;padding-left:20px;font-size:13px;color:#78350f;line-height:1.6;">
										<li style="margin-bottom:8px;">
											<b>Ubah Password Anda:</b> Segera login ke aplikasi dan ganti password sementara di menu <i>Profil Saya &rarr; Tab Keamanan</i> demi keamanan akun Anda.
										</li>
										<li>
											<b>Lengkapi Data Diri & Kontak:</b> Sangat diharapkan untuk segera melengkapi informasi biodata di menu <i>Profil Saya &rarr; Lengkapi/Edit Biodata</i>. Harap lengkapi:
											<ul style="margin:6px 0 0;padding-left:16px;">
												<li><b>No. Telepon / WhatsApp & Verifikasi OTP</b> (Untuk koordinasi resmi)</li>
												<li><b>Alamat Domisili & Tempat/Tanggal Lahir</b> (Untuk administrasi kepegawaian)</li>
												<li><b>Kontak Darurat (Emergency Contact)</b> (Untuk keselamatan kerja)</li>
												<li><b>Media Sosial / LinkedIn & Bio Singkat</b></li>
											</ul>
										</li>
									</ol>
								</div>

								<p style="margin:0 0 24px;font-size:13px;color:#475569;line-height:1.6;">
									Manajemen berharap Anda dapat meraih prestasi terbaik bersama PT. Cakra Media Data. Jika ada kesulitan akses, silakan hubungi tim Admin HRD.
								</p>

								<hr style="border:none;border-top:1px solid #e2e8f0;margin:24px 0;">
								<p style="margin:0;font-size:11px;color:#94a3b8;text-align:center;">
									PT. Cakra Media Data • Jl. Raya Mambal Ubud, Badung, Bali <br>
									Email ini dikirim secara otomatis oleh sistem. Mohon tidak membalas email ini.
								</p>
							</td>
						</tr>
					</table>
				</td>
			</tr>
		</table>
	</body>
	</html>`, employeeName, username, tempPassword)

	return SendEmailNotification(toEmail, subject, htmlBody)
}


