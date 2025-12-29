package controllers

import (
	"KPI_System_Backend/db_var"
    "KPI_System_Backend/helper" // <--- Import Helper
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
	Name          string  `json:"name" binding:"required"`
	Description   string  `json:"description"`
	IndicatorType string  `json:"indicator_type" binding:"required"`
	Weight        float64 `json:"weight" binding:"required,gt=0"`
	DivisionID    *uint   `json:"division_id"`
}

// --- Helper Function untuk Validasi Bobot ---
func validateWeight(tx *gorm.DB, indicatorType string, divisionID *uint, newWeight float64, currentIndicatorID ...uint) (float64, bool) {
	var totalWeight float64
	query := tx.Model(&models.PerformanceIndicator{})

	if indicatorType == db_var.IndicatorTypeUmum {
		query = query.Where("indicator_type = ? AND division_id IS NULL", db_var.IndicatorTypeUmum)
	} else if indicatorType == db_var.IndicatorTypeSpesifik && divisionID != nil {
		query = query.Where("indicator_type = ? AND division_id = ?", db_var.IndicatorTypeSpesifik, *divisionID)
	} else {
		return 0, false 
	}

	if len(currentIndicatorID) > 0 {
		query = query.Where("id != ?", currentIndicatorID[0])
	}

	if err := query.Select("COALESCE(SUM(weight), 0)").Scan(&totalWeight).Error; err != nil {
		return 0, false 
	}

	return totalWeight, (totalWeight + newWeight) <= 100
}


// --- CRUD Functions ---

// CreateIndicator: Membuat indikator baru
func (ic *IndicatorController) CreateIndicator(c *gin.Context) {
	var req IndicatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid", err.Error())
		return
	}

	if req.IndicatorType == db_var.IndicatorTypeUmum && req.DivisionID != nil {
		Response(c, http.StatusBadRequest, "Indikator 'umum' tidak boleh memiliki DivisionID", nil)
		return
	}
	if req.IndicatorType == db_var.IndicatorTypeSpesifik && req.DivisionID == nil {
		Response(c, http.StatusBadRequest, "Indikator 'spesifik' harus memiliki DivisionID", nil)
		return
	}

	tx := ic.DB.Begin()
	
	totalWeight, isValid := validateWeight(tx, req.IndicatorType, req.DivisionID, req.Weight)
	if !isValid {
		tx.Rollback()
		message := "Bobot tidak valid. Total bobot (termasuk yang ini) tidak boleh melebihi 100%."
		Response(c, http.StatusBadRequest, message, gin.H{"total_weight_existing": totalWeight, "new_weight": req.Weight})
		return
	}

	indicator := models.PerformanceIndicator{
		Name:          req.Name,
		Description:   req.Description,
		IndicatorType: req.IndicatorType,
		Weight:        req.Weight,
		DivisionID:    req.DivisionID,
	}

	if err := tx.Create(&indicator).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusInternalServerError, "Gagal menyimpan indikator", nil)
		return
	}

	tx.Commit()

    // --- [AUDIT TRAIL] ---
    actorID, _ := c.Get("userID")
    if idUint, ok := actorID.(uint); ok {
	    helper.LogActivity(ic.DB, idUint, "CREATE_INDICATOR", "Membuat indikator KPI: "+indicator.Name, c.ClientIP())
    }

	Response(c, http.StatusCreated, db_var.MsgIndicatorCreated, indicator)
}

// GetAllIndicators: Mendapatkan semua indikator
func (ic *IndicatorController) GetAllIndicators(c *gin.Context) {
	var indicators []models.PerformanceIndicator

	if err := ic.DB.Preload("Division").Find(&indicators).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil data indikator", nil)
		return
	}

	Response(c, http.StatusOK, "Data indikator berhasil diambil", indicators)
}

// GetIndicator: Mendapatkan satu indikator
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
func (ic *IndicatorController) UpdateIndicator(c *gin.Context) {
	id := c.Param("id")

	var req IndicatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid", err.Error())
		return
	}

	if req.IndicatorType == db_var.IndicatorTypeUmum && req.DivisionID != nil {
		Response(c, http.StatusBadRequest, "Indikator 'umum' tidak boleh memiliki DivisionID", nil)
		return
	}
	if req.IndicatorType == db_var.IndicatorTypeSpesifik && req.DivisionID == nil {
		Response(c, http.StatusBadRequest, "Indikator 'spesifik' harus memiliki DivisionID", nil)
		return
	}

	tx := ic.DB.Begin()

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

	totalWeight, isValid := validateWeight(tx, req.IndicatorType, req.DivisionID, req.Weight, indicator.ID)
	if !isValid {
		tx.Rollback()
		message := "Bobot tidak valid. Total bobot (termasuk yang ini) tidak boleh melebihi 100%."
		Response(c, http.StatusBadRequest, message, gin.H{"total_weight_existing": totalWeight, "new_weight": req.Weight})
		return
	}

	indicator.Name = req.Name
	indicator.Description = req.Description
	indicator.IndicatorType = req.IndicatorType
	indicator.Weight = req.Weight
	indicator.DivisionID = req.DivisionID

	if err := tx.Save(&indicator).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusInternalServerError, "Gagal memperbarui indikator", nil)
		return
	}

	tx.Commit()

    // --- [AUDIT TRAIL] ---
    actorID, _ := c.Get("userID")
    if idUint, ok := actorID.(uint); ok {
	    helper.LogActivity(ic.DB, idUint, "UPDATE_INDICATOR", "Mengupdate indikator KPI: "+indicator.Name, c.ClientIP())
    }

	Response(c, http.StatusOK, "Indikator berhasil diperbarui", indicator)
}

// DeleteIndicator: Menghapus indikator
func (ic *IndicatorController) DeleteIndicator(c *gin.Context) {
	id := c.Param("id")

    // Ambil data sebelum hapus
    var indicator models.PerformanceIndicator
    ic.DB.First(&indicator, id)

	var scoreCount int64
	if err := ic.DB.Model(&models.EvaluationScore{}).Where("indicator_id = ?", id).Count(&scoreCount).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal memverifikasi penggunaan indikator", nil)
		return
	}

	if scoreCount > 0 {
		Response(c, http.StatusBadRequest, "Indikator tidak dapat dihapus karena sudah digunakan dalam penilaian", nil)
		return
	}

	if err := ic.DB.Delete(&models.PerformanceIndicator{}, id).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menghapus indikator", nil)
		return
	}

    // --- [AUDIT TRAIL] ---
    actorID, _ := c.Get("userID")
    if idUint, ok := actorID.(uint); ok {
        desc := "Menghapus indikator ID " + id
        if indicator.Name != "" { desc = "Menghapus indikator KPI: " + indicator.Name }
	    helper.LogActivity(ic.DB, idUint, "DELETE_INDICATOR", desc, c.ClientIP())
    }

	Response(c, http.StatusOK, "Indikator berhasil dihapus", nil)
}