package controllers

import (
	"KPI_System_Backend/logger"
	"KPI_System_Backend/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type EvaluationPeriodController struct {
	DB *gorm.DB
}

func NewEvaluationPeriodController(db *gorm.DB) *EvaluationPeriodController {
	return &EvaluationPeriodController{DB: db}
}

type PeriodRequest struct {
	Name      string    `json:"name" binding:"required"`
	StartDate time.Time `json:"start_date" binding:"required"`
	EndDate   time.Time `json:"end_date" binding:"required"`
}

func (pc *EvaluationPeriodController) CreatePeriod(c *gin.Context) {
	var req PeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid", err.Error())
		return
	}
	if req.EndDate.Before(req.StartDate) {
		Response(c, http.StatusBadRequest, "Tanggal Selesai harus setelah Tanggal Mulai", nil)
		return
	}
	period := models.EvaluationPeriod{
		Name:      req.Name,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		IsActive:  false,
	}
	if err := pc.DB.Create(&period).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menyimpan periode", nil)
		return
	}
	Response(c, http.StatusCreated, "Periode berhasil dibuat", period)
}

func (pc *EvaluationPeriodController) GetAllPeriods(c *gin.Context) {
	var periods []models.EvaluationPeriod
	if err := pc.DB.Order("is_active desc, start_date desc").Find(&periods).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil data periode", nil)
		return
	}
	Response(c, http.StatusOK, "Data periode berhasil diambil", periods)
}

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
	period.Name = req.Name
	period.StartDate = req.StartDate
	period.EndDate = req.EndDate
	if err := pc.DB.Save(&period).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal memperbarui periode", nil)
		return
	}
	Response(c, http.StatusOK, "Periode berhasil diperbarui", period)
}

func (pc *EvaluationPeriodController) DeletePeriod(c *gin.Context) {
	id := c.Param("id")
	var evaluationCount int64
	if err := pc.DB.Model(&models.Evaluation{}).Where("period_id = ?", id).Count(&evaluationCount).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal memverifikasi penggunaan periode", nil)
		return
	}
	if evaluationCount > 0 {
		Response(c, http.StatusBadRequest, "Periode tidak dapat dihapus karena sudah memiliki data evaluasi", nil)
		return
	}
	if err := pc.DB.Delete(&models.EvaluationPeriod{}, id).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menghapus periode", nil)
		return
	}
	Response(c, http.StatusOK, "Periode berhasil dihapus", nil)
}

func (pc *EvaluationPeriodController) SetActivePeriod(c *gin.Context) {
	id := c.Param("id")
	err := pc.DB.Exec(`UPDATE evaluation_periods SET is_active = CASE WHEN id = ? THEN 1 ELSE 0 END`, id).Error

	if err != nil {
		logger.Error("Gagal update periode", zap.Error(err))
		Response(c, http.StatusInternalServerError, "Gagal mengubah status periode", nil)
		return
	}
	Response(c, http.StatusOK, "Periode berhasil diaktifkan", nil)
}