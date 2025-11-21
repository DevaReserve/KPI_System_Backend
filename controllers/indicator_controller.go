package controllers

import (
	"KPI_System_Backend/db_var"
	"KPI_System_Backend/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type IndicatorController struct {
	DB *gorm.DB
}

// NewIndicatorController adalah "constructor"
func NewIndicatorController(db *gorm.DB) *IndicatorController {
	return &IndicatorController{DB: db}
}

// --- Struct untuk Request Binding ---

type IndicatorRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	// Tipe harus "umum" atau "spesifik"
	IndicatorType string  `json:"indicator_type" binding:"required"`
	Weight        float64 `json:"weight" binding:"required,gt=0"`
	// Boleh null jika tipenya "umum"
	DivisionID *uint `json:"division_id"`
}

// --- Helper Function untuk Validasi Bobot ---
// Ini adalah logic bisnis paling penting di controller ini
func validateWeight(tx *gorm.DB, indicatorType string, divisionID *uint, newWeight float64, currentIndicatorID ...uint) (float64, bool) {
	var totalWeight float64
	query := tx.Model(&models.PerformanceIndicator{})

	if indicatorType == db_var.IndicatorTypeUmum {
		query = query.Where("indicator_type = ? AND division_id IS NULL", db_var.IndicatorTypeUmum)
	} else if indicatorType == db_var.IndicatorTypeSpesifik && divisionID != nil {
		query = query.Where("indicator_type = ? AND division_id = ?", db_var.IndicatorTypeSpesifik, *divisionID)
	} else {
		// Tipe tidak valid atau DivisionID null untuk "spesifik"
		return 0, false 
	}

	// Jika ini adalah 'Update', kita harus mengecualikan bobot lama dari indikator yang sedang diedit
	if len(currentIndicatorID) > 0 {
		query = query.Where("id != ?", currentIndicatorID[0])
	}

	// Hitung total bobot yang sudah ada
	if err := query.Select("COALESCE(SUM(weight), 0)").Scan(&totalWeight).Error; err != nil {
		return 0, false // Gagal query
	}

	// Cek apakah bobot baru + bobot lama melebihi 100
	return totalWeight, (totalWeight + newWeight) <= 100
}


// --- CRUD Functions ---

// CreateIndicator: Membuat indikator baru
// @Route: POST /api/admin/indicators
func (ic *IndicatorController) CreateIndicator(c *gin.Context) {
	var req IndicatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid", err.Error())
		return
	}

	// Validasi Logic Bisnis
	if req.IndicatorType == db_var.IndicatorTypeUmum && req.DivisionID != nil {
		Response(c, http.StatusBadRequest, "Indikator 'umum' tidak boleh memiliki DivisionID", nil)
		return
	}
	if req.IndicatorType == db_var.IndicatorTypeSpesifik && req.DivisionID == nil {
		Response(c, http.StatusBadRequest, "Indikator 'spesifik' harus memiliki DivisionID", nil)
		return
	}

	// Mulai Transaksi
	tx := ic.DB.Begin()
	
	// Validasi Bobot
	totalWeight, isValid := validateWeight(tx, req.IndicatorType, req.DivisionID, req.Weight)
	if !isValid {
		tx.Rollback()
		message := "Bobot tidak valid. Total bobot (termasuk yang ini) tidak boleh melebihi 100%."
		if totalWeight > 0 {
			message = "Bobot tidak valid. Sisa bobot yang tersedia adalah " + gorm.ErrUnsupportedDriver.Error() // Perlu konversi float to string
		}
		Response(c, http.StatusBadRequest, message, gin.H{"total_weight_existing": totalWeight, "new_weight": req.Weight})
		return
	}

	// Buat model
	indicator := models.PerformanceIndicator{
		Name:          req.Name,
		Description:   req.Description,
		IndicatorType: req.IndicatorType,
		Weight:        req.Weight,
		DivisionID:    req.DivisionID,
	}

	// Simpan ke DB
	if err := tx.Create(&indicator).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusInternalServerError, "Gagal menyimpan indikator", nil)
		return
	}

	tx.Commit()
	Response(c, http.StatusCreated, db_var.MsgIndicatorCreated, indicator)
}

// GetAllIndicators: Mendapatkan semua indikator
// @Route: GET /api/admin/indicators
func (ic *IndicatorController) GetAllIndicators(c *gin.Context) {
	var indicators []models.PerformanceIndicator

	// Gunakan Preload("Division") untuk otomatis JOIN dan mengambil data divisi
	// Ini bisa karena Anda sudah mendefinisikan relasinya di models/indicator.go
	if err := ic.DB.Preload("Division").Find(&indicators).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil data indikator", nil)
		return
	}

	Response(c, http.StatusOK, "Data indikator berhasil diambil", indicators)
}

// GetIndicator: Mendapatkan satu indikator
// @Route: GET /api/admin/indicators/:id
func (ic *IndicatorController) GetIndicator(c *gin.Context) {
	id := c.Param("id")
	var indicator models.PerformanceIndicator

	if err := ic.DB.Preload("Division").First(&indicator, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			Response(c, http.StatusNotFound, "Indikator tidak ditemukan", nil)
			return
		}
		Response(c, http.StatusInternalServerError, "Gagal mengambil data indikator", nil)
		return
	}

	Response(c, http.StatusOK, "Data indikator berhasil diambil", indicator)
}

// UpdateIndicator: Memperbarui indikator
// @Route: PUT /api/admin/indicators/:id
func (ic *IndicatorController) UpdateIndicator(c *gin.Context) {
	id := c.Param("id")

	var req IndicatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid", err.Error())
		return
	}

	// Validasi Logic Bisnis
	if req.IndicatorType == db_var.IndicatorTypeUmum && req.DivisionID != nil {
		Response(c, http.StatusBadRequest, "Indikator 'umum' tidak boleh memiliki DivisionID", nil)
		return
	}
	if req.IndicatorType == db_var.IndicatorTypeSpesifik && req.DivisionID == nil {
		Response(c, http.StatusBadRequest, "Indikator 'spesifik' harus memiliki DivisionID", nil)
		return
	}

	// Mulai Transaksi
	tx := ic.DB.Begin()

	// Cari indikator yang ada
	var indicator models.PerformanceIndicator
	if err := tx.First(&indicator, id).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			Response(c, http.StatusNotFound, "Indikator tidak ditemukan", nil)
			return
		}
		Response(c, http.StatusInternalServerError, "Gagal mencari indikator", nil)
		return
	}

	// Validasi Bobot (dengan pengecualian ID indikator saat ini)
	totalWeight, isValid := validateWeight(tx, req.IndicatorType, req.DivisionID, req.Weight, indicator.ID)
	if !isValid {
		tx.Rollback()
		message := "Bobot tidak valid. Total bobot (termasuk yang ini) tidak boleh melebihi 100%."
		Response(c, http.StatusBadRequest, message, gin.H{"total_weight_existing": totalWeight, "new_weight": req.Weight})
		return
	}

	// Update data
	indicator.Name = req.Name
	indicator.Description = req.Description
	indicator.IndicatorType = req.IndicatorType
	indicator.Weight = req.Weight
	indicator.DivisionID = req.DivisionID

	// Simpan perubahan
	if err := tx.Save(&indicator).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusInternalServerError, "Gagal memperbarui indikator", nil)
		return
	}

	tx.Commit()
	Response(c, http.StatusOK, "Indikator berhasil diperbarui", indicator)
}

// DeleteIndicator: Menghapus indikator
// @Route: DELETE /api/admin/indicators/:id
func (ic *IndicatorController) DeleteIndicator(c *gin.Context) {
	id := c.Param("id")

	// PENTING! Cek Keamanan: Jangan hapus indikator jika sudah pernah dipakai menilai.
	var scoreCount int64
	if err := ic.DB.Model(&models.EvaluationScore{}).Where("indicator_id = ?", id).Count(&scoreCount).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal memverifikasi penggunaan indikator", nil)
		return
	}

	if scoreCount > 0 {
		Response(c, http.StatusBadRequest, "Indikator tidak dapat dihapus karena sudah digunakan dalam penilaian", nil)
		return
	}

	// Jika aman (belum pernah dipakai), lanjutkan proses hapus
	if err := ic.DB.Delete(&models.PerformanceIndicator{}, id).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menghapus indikator", nil)
		return
	}

	Response(c, http.StatusOK, "Indikator berhasil dihapus", nil)
}