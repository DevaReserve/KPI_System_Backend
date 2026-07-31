package helper

import (
	"context"
	"fmt"
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
)

var (
	waClient   *whatsmeow.Client
	waClientMu sync.Mutex
	waReady    bool
)

// InitWhatsApp hanya melakukan inisialisasi store dan client.
// Jika sudah ada sesi tersimpan, ia akan langsung Connect().
// Jika belum, ia tidak akan melakukan apa-apa (menunggu trigger dari frontend).
func InitWhatsApp() {
	waClientMu.Lock()
	defer waClientMu.Unlock()

	if waClient != nil {
		return // Sudah diinisialisasi
	}

	logger.Info("Memulai inisialisasi WhatsApp Service...")

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

	if client.Store.ID != nil {
		// Sudah ada sesi, langsung connect
		err = client.Connect()
		if err != nil {
			logger.Error("Gagal menghubungkan WhatsApp client", zap.Error(err))
		} else {
			waReady = true
			logger.Info("[WhatsApp] Client berhasil terhubung menggunakan sesi tersimpan.")
		}
	} else {
		logger.Info("[WhatsApp] Belum ada sesi tersimpan. Menunggu pairing dari Frontend.")
	}

	waClient = client
}

// RequestQRPairing meminta channel QR untuk keperluan scanning dinamis.
// Fungsi ini mengembalikan channel events.
func RequestQRPairing(ctx context.Context) (<-chan whatsmeow.QRChannelItem, error) {
	waClientMu.Lock()
	client := waClient
	waClientMu.Unlock()

	if client == nil {
		return nil, fmt.Errorf("whatsapp client belum diinisialisasi")
	}

	if client.Store.ID != nil {
		// Jika sudah login, pastikan connect saja
		if !client.IsConnected() {
			client.Connect()
			waClientMu.Lock()
			waReady = true
			waClientMu.Unlock()
		}
		return nil, fmt.Errorf("already_logged_in")
	}

	// Jika belum terhubung, hubungkan untuk mendapatkan QR
	if !client.IsConnected() {
		err := client.Connect()
		if err != nil {
			return nil, err
		}
	}

	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		return nil, err
	}

	return qrChan, nil
}

// SetWAReady dipanggil oleh controller jika menerima event "success" dari qrChan
func SetWAReady() {
	waClientMu.Lock()
	defer waClientMu.Unlock()
	waReady = true
	logger.Info("[WhatsApp] QR Code berhasil di-scan! Sesi tersimpan.")
}

// LogoutWhatsApp menghapus sesi WA saat ini.
func LogoutWhatsApp() error {
	waClientMu.Lock()
	defer waClientMu.Unlock()
	
	waReady = false
	if waClient == nil {
		return nil
	}

	err := waClient.Logout(context.Background())
	if err != nil {
		// Jika gagal logout via network, kita paksa putus koneksi
		waClient.Disconnect()
		return err
	}
	return nil
}

// IsWhatsAppReady mengembalikan status koneksi WhatsApp.
func IsWhatsAppReady() bool {
	waClientMu.Lock()
	defer waClientMu.Unlock()
	return waReady && waClient != nil && waClient.IsConnected() && waClient.IsLoggedIn()
}

func formatPhoneForWA(phone string) string {
	if len(phone) == 0 {
		return phone
	}
	cleaned := ""
	for _, c := range phone {
		if c >= '0' && c <= '9' || c == '+' {
			cleaned += string(c)
		}
	}
	if len(cleaned) > 0 && cleaned[0] == '0' {
		cleaned = "62" + cleaned[1:]
	}
	if len(cleaned) > 0 && cleaned[0] == '+' {
		cleaned = cleaned[1:]
	}
	return cleaned
}

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