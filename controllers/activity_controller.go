package controllers

import (
	"KPI_System_Backend/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ActivityController struct {
	DB *gorm.DB
}

func NewActivityController(db *gorm.DB) *ActivityController {
	return &ActivityController{DB: db}
}

// GetAllLogs: Melihat log dengan filter tanggal
func (ac *ActivityController) GetAllLogs(c *gin.Context) {
	var logs []models.ActivityLog

	// Mulai query dasar
	query := ac.DB.Preload("User").Preload("User.Employee").Order("created_at desc")

	// 1. Ambil Parameter Tanggal dari URL (contoh: ?start_date=2023-01-01&end_date=2023-01-31)
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// 2. Terapkan Filter jika parameter ada
	if startDate != "" {
		query = query.Where("DATE(created_at) >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("DATE(created_at) <= ?", endDate)
	}

	// Batasi 500 log terakhir agar tidak terlalu berat (bisa disesuaikan)
	if err := query.Limit(500).Find(&logs).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil log", nil)
		return
	}

	Response(c, http.StatusOK, "Data activity log berhasil diambil", logs)
}