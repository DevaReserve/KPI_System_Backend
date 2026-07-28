package controllers

import (
	"KPI_System_Backend/helper"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type WhatsAppController struct {
	DB *gorm.DB
}

func NewWhatsAppController(db *gorm.DB) *WhatsAppController {
	return &WhatsAppController{DB: db}
}

// GetWAStatus mengembalikan status koneksi WhatsApp bot.
func (wc *WhatsAppController) GetWAStatus(c *gin.Context) {
	ready := helper.IsWhatsAppReady()
	if ready {
		Response(c, http.StatusOK, "WhatsApp client terhubung dan siap mengirim pesan.", gin.H{
			"connected": true,
		})
		return
	}

	// Cek apakah ada QR code yang perlu di-scan
	qrCode, hasQR := helper.GetWhatsAppQRCode()
	if hasQR {
		Response(c, http.StatusOK, "WhatsApp belum terhubung. Silakan scan QR Code berikut menggunakan WhatsApp perusahaan.", gin.H{
			"connected": false,
			"qr_code":   qrCode,
		})
		return
	}

	Response(c, http.StatusServiceUnavailable, "WhatsApp client belum siap. Restart backend dan scan QR Code yang muncul di terminal.", gin.H{
		"connected": false,
		"qr_code":   nil,
	})
}
