package controllers

import (
	"KPI_System_Backend/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type EvaluationPeriodController struct {
	DB *gorm.DB
}

// NewEvaluationPeriodController adalah "constructor"
func NewEvaluationPeriodController(db *gorm.DB) *EvaluationPeriodController {
	return &EvaluationPeriodController{DB: db}
}

// --- Struct untuk Request Binding ---

type PeriodRequest struct {
	Name      string    `json:"name" binding:"required"`
	StartDate time.Time `json:"start_date" binding:"required"`
	EndDate   time.Time `json:"end_date" binding:"required"`
}

// --- CRUD Functions ---

// CreatePeriod: Membuat periode baru
// @Route: POST /api/admin/periods
func (pc *EvaluationPeriodController) CreatePeriod(c *gin.Context) {
	var req PeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid", err.Error())
		return
	}

	// Validasi: EndDate harus setelah StartDate
	if req.EndDate.Before(req.StartDate) {
		Response(c, http.StatusBadRequest, "Tanggal Selesai harus setelah Tanggal Mulai", nil)
		return
	}

	period := models.EvaluationPeriod{
		Name:      req.Name,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		IsActive:  false, // Periode baru defaultnya tidak aktif
	}

	if err := pc.DB.Create(&period).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menyimpan periode", nil)
		return
	}

	Response(c, http.StatusCreated, "Periode berhasil dibuat", period)
}

// GetAllPeriods: Mendapatkan semua periode
// @Route: GET /api/admin/periods
func (pc *EvaluationPeriodController) GetAllPeriods(c *gin.Context) {
	var periods []models.EvaluationPeriod

	// Urutkan berdasarkan yang paling baru (IsActive dulu, lalu StartDate)
	if err := pc.DB.Order("is_active desc, start_date desc").Find(&periods).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil data periode", nil)
		return
	}

	Response(c, http.StatusOK, "Data periode berhasil diambil", periods)
}

// GetPeriod: Mendapatkan satu periode
// @Route: GET /api/admin/periods/:id
func (pc *EvaluationPeriodController) GetPeriod(c *gin.Context) {
	id := c.Param("id")
	var period models.EvaluationPeriod

	if err := pc.DB.First(&period, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			Response(c, http.StatusNotFound, "Periode tidak ditemukan", nil)
			return
		}
		Response(c, http.StatusInternalServerError, "Gagal mengambil data periode", nil)
		return
	}

	Response(c, http.StatusOK, "Data periode berhasil diambil", period)
}

// UpdatePeriod: Memperbarui periode
// @Route: PUT /api/admin/periods/:id
func (pc *EvaluationPeriodController) UpdatePeriod(c *gin.Context) {
	id := c.Param("id")

	var period models.EvaluationPeriod
	if err := pc.DB.First(&period, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			Response(c, http.StatusNotFound, "Periode tidak ditemukan", nil)
			return
		}
		Response(c, http.StatusInternalServerError, "Gagal mencari periode", nil)
		return
	}

	var req PeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid", err.Error())
		return
	}

	// Validasi
	if req.EndDate.Before(req.StartDate) {
		Response(c, http.StatusBadRequest, "Tanggal Selesai harus setelah Tanggal Mulai", nil)
		return
	}

	// Update data
	period.Name = req.Name
	period.StartDate = req.StartDate
	period.EndDate = req.EndDate
	// Kita tidak mengizinkan update IsActive dari sini, harus lewat endpoint khusus

	if err := pc.DB.Save(&period).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal memperbarui periode", nil)
		return
	}

	Response(c, http.StatusOK, "Periode berhasil diperbarui", period)
}

// DeletePeriod: Menghapus periode
// @Route: DELETE /api/admin/periods/:id
func (pc *EvaluationPeriodController) DeletePeriod(c *gin.Context) {
	id := c.Param("id")

	// PENTING! Cek Keamanan: Jangan hapus periode jika sudah ada evaluasi di dalamnya.
	var evaluationCount int64
	if err := pc.DB.Model(&models.Evaluation{}).Where("period_id = ?", id).Count(&evaluationCount).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal memverifikasi penggunaan periode", nil)
		return
	}

	if evaluationCount > 0 {
		Response(c, http.StatusBadRequest, "Periode tidak dapat dihapus karena sudah memiliki data evaluasi", nil)
		return
	}

	// Jika aman, lanjutkan proses hapus
	if err := pc.DB.Delete(&models.EvaluationPeriod{}, id).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menghapus periode", nil)
		return
	}

	Response(c, http.StatusOK, "Periode berhasil dihapus", nil)
}

// --- Fungsi Khusus ---

// SetActivePeriod: Mengaktifkan satu periode dan menonaktifkan yang lain
// @Route: PATCH /api/admin/periods/:id/activate
func (pc *EvaluationPeriodController) SetActivePeriod(c *gin.Context) {
	id := c.Param("id")

	// 1. Mulai Transaksi
	tx := pc.DB.Begin()
	if tx.Error != nil {
		Response(c, http.StatusInternalServerError, "Gagal memulai transaksi", nil)
		return
	}

	// 2. Nonaktifkan semua periode lain
	// Kita set IsActive = false DIMANA id BUKAN id yang kita tuju
	if err := tx.Model(&models.EvaluationPeriod{}).Where("id != ?", id).Update("is_active", false).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusInternalServerError, "Gagal menonaktifkan periode lain", nil)
		return
	}

	// 3. Aktifkan periode yang kita tuju
	if err := tx.Model(&models.EvaluationPeriod{}).Where("id = ?", id).Update("is_active", true).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusInternalServerError, "Gagal mengaktifkan periode", nil)
		return
	}

	// 4. Commit Transaksi
	if err := tx.Commit().Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menyimpan perubahan", nil)
		return
	}
	
	Response(c, http.StatusOK, "Periode berhasil diaktifkan", nil)
}