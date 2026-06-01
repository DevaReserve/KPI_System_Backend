package helper

import (
	"fmt"
	"net/smtp"
)

// SendEmailNotification bertugas mengirimkan email berformat HTML ke pegawai
func SendEmailNotification(toEmail string, subject string, htmlBody string) error {
	// KONFIGURASI SMTP GMAIL
	// Nanti Anda bisa ganti ini agar membaca dari file setting.ini Anda
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	senderEmail := "puturevanarendra@gmail.com"      // Ganti dengan email pengirim
	senderPass := "qeja khou ycub ixmi"     // Ganti dengan 16 digit App Password tanpa spasi

	auth := smtp.PlainAuth("", senderEmail, senderPass, smtpHost)

	// Merakit struktur email standar (MIME) agar mendukung format HTML tebal/miring
	mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	message := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n%s\r\n%s", toEmail, subject, mimeHeaders, htmlBody))

	// Proses menembakkan email
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, senderEmail, []string{toEmail}, message)
	if err != nil {
		fmt.Println("Error: Gagal mengirim email ke", toEmail, "| Detail:", err)
		return err
	}

	fmt.Println("Sukses: Email notifikasi berhasil dikirim ke", toEmail)
	return nil
}