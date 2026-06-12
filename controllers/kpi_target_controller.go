package controllers

import (
	"KPI_System_Backend/helper"
	"KPI_System_Backend/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type KPITargetController struct {
	DB *gorm.DB
}

func NewKPITargetController(db *gorm.DB) *KPITargetController {
	return &KPITargetController{DB: db}
}

// ============================================================
// GET /api/manager/targets?employee_id=X&period_id=Y
// Manager: lihat target yang sudah diset untuk pegawai di periode tertentu
// ============================================================
func (kc *KPITargetController) GetTargets(c *gin.Context) {
	employeeID := c.Query("employee_id")
	periodID := c.Query("period_id")

	if employeeID == "" || periodID == "" {
		Response(c, http.StatusBadRequest, "Parameter employee_id dan period_id wajib diisi", nil)
		return
	}

	type TargetDetail struct {
		ID            uint    `json:"id"`
		IndicatorID   uint    `json:"indicator_id"`
		IndicatorName string  `json:"indicator_name"`
		Weight        float64 `json:"weight"`
		IndicatorType string  `json:"indicator_type"`
		TargetScore   int     `json:"target_score"`
		Notes         string  `json:"notes"`
	}

	var results []TargetDetail
	kc.DB.Table("kpi_targets kt").
		Select(`kt.id, kt.indicator_id, pi.name as indicator_name, pi.weight, pi.indicator_type, kt.target_score, kt.notes`).
		Joins("JOIN performance_indicators pi ON pi.id = kt.indicator_id").
		Where("kt.employee_id = ? AND kt.period_id = ?", employeeID, periodID).
		Scan(&results)

	if results == nil {
		results = []TargetDetail{}
	}

	Response(c, http.StatusOK, "Target KPI berhasil diambil", results)
}

// ============================================================
// POST /api/manager/targets/bulk
// Manager: set target untuk semua indikator seorang pegawai di 1 periode
// Body: { employee_id, period_id, targets: [{indicator_id, target_score, notes}] }
// ============================================================
func (kc *KPITargetController) SetTargetsBulk(c *gin.Context) {
	type TargetItem struct {
		IndicatorID uint   `json:"indicator_id" binding:"required"`
		TargetScore int    `json:"target_score" binding:"required,min=1,max=5"`
		Notes       string `json:"notes"`
	}

	var req struct {
		EmployeeID uint         `json:"employee_id" binding:"required"`
		PeriodID   uint         `json:"period_id" binding:"required"`
		Targets    []TargetItem `json:"targets" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid", err.Error())
		return
	}

	// Ambil identity manager
	managerUserID, _ := c.Get("userID")
	var managerUser models.User
	kc.DB.First(&managerUser, managerUserID)

	// Hapus semua target lama untuk employee+period ini, lalu insert ulang (upsert manual)
	kc.DB.Where("employee_id = ? AND period_id = ?", req.EmployeeID, req.PeriodID).
		Delete(&models.KPITarget{})

	var newTargets []models.KPITarget
	for _, t := range req.Targets {
		newTargets = append(newTargets, models.KPITarget{
			EmployeeID:  req.EmployeeID,
			PeriodID:    req.PeriodID,
			IndicatorID: t.IndicatorID,
			TargetScore: t.TargetScore,
			Notes:       t.Notes,
			SetByID:     managerUser.EmployeeID,
		})
	}

	if err := kc.DB.Create(&newTargets).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menyimpan target KPI", nil)
		return
	}

	// Audit log
	if idUint, ok := managerUserID.(uint); ok {
		var emp models.Employee
		kc.DB.First(&emp, req.EmployeeID)
		helper.LogActivity(kc.DB, idUint, "SET_KPI_TARGET", "Menetapkan target KPI untuk: "+emp.Name, c.ClientIP())
	}

	Response(c, http.StatusCreated, "Target KPI berhasil ditetapkan", newTargets)
}

// ============================================================
// GET /api/employee/targets?period_id=Y
// Employee: lihat target yang ditetapkan untuk diri sendiri
// Juga menyertakan actual score jika evaluasi sudah submitted
// ============================================================
func (kc *KPITargetController) GetMyTargets(c *gin.Context) {
	periodID := c.Query("period_id")

	userID, _ := c.Get("userID")
	var user models.User
	kc.DB.First(&user, userID)

	type TargetWithActual struct {
		IndicatorID   uint    `json:"indicator_id"`
		IndicatorName string  `json:"indicator_name"`
		Weight        float64 `json:"weight"`
		TargetScore   int     `json:"target_score"`
		ActualScore   int     `json:"actual_score"`   // 0 jika belum dievaluasi
		ConvertedActual int   `json:"converted_actual"`
		Notes         string  `json:"notes"`
	}

	query := kc.DB.Table("kpi_targets kt").
		Select(`kt.indicator_id, pi.name as indicator_name, pi.weight, 
			kt.target_score, kt.notes,
			COALESCE(es.score, 0) as actual_score,
			COALESCE(es.converted_score, 0) as converted_actual`).
		Joins("JOIN performance_indicators pi ON pi.id = kt.indicator_id").
		Joins(`LEFT JOIN evaluations ev ON ev.employee_id = kt.employee_id 
			AND ev.period_id = kt.period_id AND ev.status = 'submitted'`).
		Joins("LEFT JOIN evaluation_scores es ON es.evaluation_id = ev.id AND es.indicator_id = kt.indicator_id").
		Where("kt.employee_id = ?", user.EmployeeID)

	if periodID != "" {
		query = query.Where("kt.period_id = ?", periodID)
	}

	var results []TargetWithActual
	if err := query.Scan(&results).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil target", nil)
		return
	}

	if results == nil {
		results = []TargetWithActual{}
	}

	Response(c, http.StatusOK, "Target KPI Anda berhasil diambil", results)
}

// ============================================================
// GET /api/manager/targets/indicators?employee_id=X&period_id=Y
// Ambil semua indikator yang relevan untuk dijadikan target
// ============================================================
func (kc *KPITargetController) GetIndicatorsForTarget(c *gin.Context) {
	employeeID := c.Query("employee_id")
	if employeeID == "" {
		Response(c, http.StatusBadRequest, "employee_id wajib diisi", nil)
		return
	}

	var employee models.Employee
	if err := kc.DB.First(&employee, employeeID).Error; err != nil {
		Response(c, http.StatusNotFound, "Pegawai tidak ditemukan", nil)
		return
	}

	var indicators []models.PerformanceIndicator
	kc.DB.Where(
		kc.DB.Where("indicator_type = ?", "umum").
			Or("indicator_type = ? AND division_id = ?", "spesifik", employee.DivisionID),
	).Find(&indicators)

	Response(c, http.StatusOK, "Indikator berhasil diambil", indicators)
}
