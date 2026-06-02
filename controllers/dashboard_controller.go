package controllers

import (
	"KPI_System_Backend/logger"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DashboardController struct {
	DB *gorm.DB
}

func NewDashboardController(db *gorm.DB) *DashboardController {
	return &DashboardController{DB: db}
}

func (dc *DashboardController) GetCompanyPerformance(c *gin.Context) {
	isExecutiveInterface, exists := c.Get("is_executive")
	if !exists || isExecutiveInterface.(bool) == false {
		logger.Warn("Akses ilegal ke Executive Dashboard ditolak")
		Response(c, http.StatusForbidden, "Akses Ditolak: Khusus Eksekutif", nil)
		return
	}

	periodID := c.Query("period_id")

	// 1. METRIK UMUM
	var totalEmployees, totalDivisions, totalCompletedEvals int64
	dc.DB.Table("employees").Where("is_active = ?", true).Count(&totalEmployees)
	dc.DB.Table("divisions").Count(&totalDivisions)

	evalQuery := dc.DB.Table("evaluations").Where("status = ?", "submitted")
	if periodID != "" {
		evalQuery = evalQuery.Where("period_id = ?", periodID)
	}
	evalQuery.Count(&totalCompletedEvals)

	// 2. RATA-RATA DIVISI
	type DivisionPerformance struct {
		DivisionName string  `json:"division_name"`
		AverageScore float64 `json:"average_score"`
	}
	var performanceData []DivisionPerformance
	baseQuery := `
		SELECT d.name as division_name, COALESCE(AVG(e.total_score), 0) as average_score
		FROM divisions d
		LEFT JOIN employees emp ON emp.division_id = d.id
		LEFT JOIN evaluations e ON e.employee_id = emp.id AND e.status = 'submitted'`
	if periodID != "" {
		baseQuery += " AND e.period_id = ?"
	}
	baseQuery += " GROUP BY d.id"
	
	if periodID != "" {
		dc.DB.Raw(baseQuery, periodID).Scan(&performanceData)
	} else {
		dc.DB.Raw(baseQuery).Scan(&performanceData)
	}

	// 3. DISTRIBUSI STATUS EVALUASI (Untuk Donut Chart)
	type StatusCount struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	var statusCounts []StatusCount
	statusQuery := dc.DB.Table("evaluations").Select("status, count(*) as count")
	if periodID != "" {
		statusQuery = statusQuery.Where("period_id = ?", periodID)
	}
	statusQuery.Group("status").Scan(&statusCounts)

	// 4. TOP 5 PEGAWAI TERBAIK
	type TopEmployee struct {
		Name         string  `json:"name"`
		DivisionName string  `json:"division_name"`
		TotalScore   float64 `json:"total_score"`
	}
	var topEmployees []TopEmployee
	topQuery := dc.DB.Table("evaluations e").
		Select("emp.name, d.name as division_name, e.total_score").
		Joins("JOIN employees emp ON emp.id = e.employee_id").
		Joins("JOIN divisions d ON d.id = emp.division_id").
		Where("e.status = ?", "submitted")
	
	if periodID != "" {
		topQuery = topQuery.Where("e.period_id = ?", periodID)
	}
	topQuery.Order("e.total_score DESC").Limit(5).Scan(&topEmployees)

	// GABUNGKAN RESPON
	responseData := gin.H{
		"metrics": gin.H{
			"total_employees":       totalEmployees,
			"total_divisions":       totalDivisions,
			"total_completed_evals": totalCompletedEvals,
		},
		"division_performance": performanceData,
		"status_distribution":  statusCounts,
		"top_employees":        topEmployees,
	}

	Response(c, http.StatusOK, "Data Dashboard Berhasil Diambil", responseData)
}