package controllers

import (
	"KPI_System_Backend/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NotificationController struct {
	DB *gorm.DB
}

func NewNotificationController(db *gorm.DB) *NotificationController {
	return &NotificationController{DB: db}
}

// GetMyNotifications: Mengambil notifikasi user yang sedang login
func (nc *NotificationController) GetMyNotifications(c *gin.Context) {
	userID, _ := c.Get("userID")

	var notifications []models.Notification
	if err := nc.DB.Where("user_id = ?", userID).Order("created_at desc").Limit(20).Find(&notifications).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil notifikasi", nil)
		return
	}

	Response(c, http.StatusOK, "Notifikasi berhasil diambil", notifications)
}

// MarkAsRead: Menandai notifikasi telah dibaca
func (nc *NotificationController) MarkAsRead(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("userID")

	var notification models.Notification
	if err := nc.DB.Where("id = ? AND user_id = ?", id, userID).First(&notification).Error; err != nil {
		Response(c, http.StatusNotFound, "Notifikasi tidak ditemukan", nil)
		return
	}

	notification.IsRead = true
	if err := nc.DB.Save(&notification).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menandai notifikasi", nil)
		return
	}

	Response(c, http.StatusOK, "Notifikasi ditandai sudah dibaca", nil)
}

// MarkAllAsRead
func (nc *NotificationController) MarkAllAsRead(c *gin.Context) {
	userID, _ := c.Get("userID")

	if err := nc.DB.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Update("is_read", true).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menandai semua notifikasi", nil)
		return
	}

	Response(c, http.StatusOK, "Semua notifikasi ditandai sudah dibaca", nil)
}
