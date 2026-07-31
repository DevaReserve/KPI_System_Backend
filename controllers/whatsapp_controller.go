package controllers

import (
	"KPI_System_Backend/helper"
	"KPI_System_Backend/logger"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type WhatsAppController struct {
	DB *gorm.DB
}

func NewWhatsAppController(db *gorm.DB) *WhatsAppController {
	return &WhatsAppController{DB: db}
}

// GetStatus mengecek apakah WhatsApp bot sedang terhubung atau tidak
func (wc *WhatsAppController) GetStatus(c *gin.Context) {
	status := "disconnected"
	if helper.IsWhatsAppReady() {
		status = "connected"
	}
	Response(c, http.StatusOK, "WhatsApp status", gin.H{"status": status})
}

// PairStream melakukan streaming (SSE) QR code ke frontend
// dan menunggu hingga QR di-scan atau timeout setelah 55 detik
func (wc *WhatsAppController) PairStream(c *gin.Context) {
	// Set header untuk SSE
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	// Pastikan flush dapat dipanggil
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.String(http.StatusInternalServerError, "Streaming unsupported")
		return
	}

	qrChan, err := helper.RequestQRPairing(c.Request.Context())
	if err != nil {
		if err.Error() == "already_logged_in" {
			c.SSEvent("success", "already_logged_in")
			flusher.Flush()
			return
		}
		c.SSEvent("error", err.Error())
		flusher.Flush()
		return
	}

	// Timeout aman Vercel (Hobby max 60s, kita stop di 55s)
	timeout := time.After(55 * time.Second)

	for {
		select {
		case evt := <-qrChan:
			if evt.Event == "code" {
				c.SSEvent("qr", evt.Code)
				flusher.Flush()
			} else if evt.Event == "success" {
				helper.SetWAReady()
				
				// Audit trail
				if userID, exists := c.Get("userID"); exists {
					if idUint, ok := userID.(uint); ok {
						helper.LogActivity(wc.DB, idUint, "WA_LOGIN", "Admin menghubungkan Bot WhatsApp", c.ClientIP())
					}
				}

				c.SSEvent("success", "ok")
				flusher.Flush()
				return
			}
		case <-timeout:
			// Beritahu frontend bahwa timeout terjadi agar bisa me-refresh
			c.SSEvent("timeout", "timeout")
			flusher.Flush()
			return
		case <-c.Request.Context().Done():
			// Client menutup tab/browser
			logger.Info("Client terputus saat pairing WhatsApp")
			return
		}
	}
}

// Logout memutuskan sesi WhatsApp saat ini
func (wc *WhatsAppController) Logout(c *gin.Context) {
	err := helper.LogoutWhatsApp()
	if err != nil {
		logger.Error("Gagal logout WhatsApp", zap.Error(err))
		Response(c, http.StatusInternalServerError, "Gagal memutus sesi WhatsApp", nil)
		return
	}

	// Audit trail
	if userID, exists := c.Get("userID"); exists {
		if idUint, ok := userID.(uint); ok {
			helper.LogActivity(wc.DB, idUint, "WA_LOGOUT", "Admin memutus Bot WhatsApp", c.ClientIP())
		}
	}

	Response(c, http.StatusOK, "Sesi WhatsApp berhasil diputus", nil)
}
