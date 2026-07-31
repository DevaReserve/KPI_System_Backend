package helper

import (
	"context"
	"fmt"
	"os"
	"sync"

	_ "github.com/lib/pq"

	"KPI_System_Backend/config"
	"KPI_System_Backend/logger"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"github.com/skip2/go-qrcode"
)

var (
	waClient   *whatsmeow.Client
	waClientMu sync.Mutex
	waReady    bool
)

// InitWhatsApp menginisialisasi koneksi WhatsApp menggunakan whatsmeow.
// Session disimpan di file SQLite wa_session.db agar tidak perlu scan QR ulang.
func InitWhatsApp() {
	fmt.Println(">>> [DEBUG] InitWhatsApp() mulai dipanggil...")
	logger.Info("Memulai inisialisasi WhatsApp Service...")

	waClientMu.Lock()
	defer waClientMu.Unlock()

	dbLog := waLog.Noop

	dbConfig := config.GetIniDatabase()
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=require TimeZone=Asia/Jakarta",
		dbConfig.Host,
		dbConfig.User,
		dbConfig.Password,
		dbConfig.DatabaseName,
		dbConfig.Port,
	)

	container, err := sqlstore.New(context.Background(), "postgres", dsn, dbLog)
	if err != nil {
		logger.Error("Gagal membuat WhatsApp Postgres store", zap.Error(err))
		return
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		logger.Error("Gagal mendapatkan device WhatsApp dari store", zap.Error(err))
		return
	}

	clientLog := waLog.Noop
	client := whatsmeow.NewClient(deviceStore, clientLog)

	if client.Store.ID == nil {
		fmt.Println(">>> [DEBUG] Belum ada session, meminta QR channel...")
		// Belum pernah login, perlu scan QR — tulis QR ke file teks
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			logger.Error("Gagal connect WhatsApp client untuk QR", zap.Error(err))
			return
		}
		go func() {
			for evt := range qrChan {
				if evt.Event == "code" {
					// Windows terminal sering bermasalah me-render bentuk persegi panjang/spasi QR code.
					// Kita buatkan saja file gambarnya (.png) agar bisa dibuka oleh pengguna.
					errQR := qrcode.WriteFile(evt.Code, qrcode.Medium, 256, "wa_qr.png")
					
					fmt.Println("\n========================================================")
					if errQR != nil {
						fmt.Println("    [ERROR] Gagal membuat file wa_qr.png:", errQR)
					} else {
						fmt.Println("    [WhatsApp OTP] QR Code Berhasil Dibuat!")
						fmt.Println("    Silakan buka file 'wa_qr.png' yang ada di folder")
						fmt.Println("    KPI_System_Backend untuk men-scan QR-nya.")
						fmt.Println("    (File akan terus diperbarui otomatis tiap 20 detik)")
					}
					fmt.Println("========================================================\n")
					
					// Simpan raw code ke file teks juga
					_ = os.WriteFile("wa_qr.txt", []byte(evt.Code), 0644)
					logger.Info("[WhatsApp] Menunggu QR code di-scan...")
				} else if evt.Event == "success" {
					logger.Info("[WhatsApp] QR Code berhasil di-scan! Sesi tersimpan.")
					waReady = true
					_ = os.Remove("wa_qr.txt") // hapus file QR teks
					_ = os.Remove("wa_qr.png") // hapus file QR gambar
				}
			}
		}()
	} else {
		err = client.Connect()
		if err != nil {
			logger.Error("Gagal menghubungkan WhatsApp client", zap.Error(err))
			return
		}
		waReady = true
		logger.Info("[WhatsApp] Client berhasil terhubung menggunakan sesi tersimpan.")
	}

	waClient = client
}

// IsWhatsAppReady mengembalikan status koneksi WhatsApp.
func IsWhatsAppReady() bool {
	waClientMu.Lock()
	defer waClientMu.Unlock()
	return waReady && waClient != nil && waClient.IsConnected() && waClient.IsLoggedIn()
}

// GetWhatsAppQRCode membaca file wa_qr.txt jika ada (saat belum login).
func GetWhatsAppQRCode() (string, bool) {
	data, err := os.ReadFile("wa_qr.txt")
	if err != nil {
		return "", false
	}
	return string(data), true
}

// formatPhoneForWA mengubah nomor lokal Indonesia menjadi format internasional.
// Contoh: "08123456789" -> "628123456789"
func formatPhoneForWA(phone string) string {
	if len(phone) == 0 {
		return phone
	}
	// Hapus spasi, tanda hubung, tanda kurung
	cleaned := ""
	for _, c := range phone {
		if c >= '0' && c <= '9' || c == '+' {
			cleaned += string(c)
		}
	}
	// Ganti awalan "0" dengan "62" (kode negara Indonesia)
	if len(cleaned) > 0 && cleaned[0] == '0' {
		cleaned = "62" + cleaned[1:]
	}
	// Hapus tanda "+" jika ada
	if len(cleaned) > 0 && cleaned[0] == '+' {
		cleaned = cleaned[1:]
	}
	return cleaned
}

// SendPhoneOTPWhatsApp mengirimkan kode OTP ke nomor WhatsApp karyawan.
func SendPhoneOTPWhatsApp(phone string, otpCode string, recipientName string) error {
	waClientMu.Lock()
	client := waClient
	ready := waReady
	waClientMu.Unlock()

	if !ready || client == nil || !client.IsConnected() || !client.IsLoggedIn() {
		return fmt.Errorf("WhatsApp client belum siap atau belum login. Silakan scan QR Code terlebih dahulu di panel admin")
	}

	formattedPhone := formatPhoneForWA(phone)
	jid := types.NewJID(formattedPhone, types.DefaultUserServer)

	message := fmt.Sprintf(
		"*[KPI System - PT. Cakra Media Data]*\n\nHalo, *%s*! 👋\n\nKode OTP verifikasi nomor WhatsApp Anda adalah:\n\n*%s*\n\n⏰ Kode ini berlaku selama *5 menit*.\n🔒 Jangan berikan kode ini kepada siapapun, termasuk tim IT.\n\n_Jika Anda tidak merasa meminta kode ini, abaikan pesan ini._",
		recipientName,
		otpCode,
	)

	_, err := client.SendMessage(context.Background(), jid, &waE2E.Message{
		Conversation: proto.String(message),
	})
	if err != nil {
		logger.Error("[WhatsApp] Gagal mengirim pesan OTP", zap.String("phone", formattedPhone), zap.Error(err))
		return fmt.Errorf("gagal mengirim OTP ke WhatsApp: %w", err)
	}

	logger.Info("[WhatsApp] OTP berhasil dikirim", zap.String("phone", formattedPhone))
	return nil
}