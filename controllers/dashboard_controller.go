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

	// 1. TOTAL EVALUASI SELESAI
	var totalCompletedEvals int64
	evalQuery := dc.DB.Table("evaluations").Where("status = ?", "submitted")
	if periodID != "" {
		evalQuery = evalQuery.Where("period_id = ?", periodID)
	}
	evalQuery.Count(&totalCompletedEvals)

	// 2. PERFORMA DIVISI (Diurutkan dari tertinggi ke terendah)
	type DivisionPerformance struct {
		DivisionName string  `json:"division_name"`
		AverageScore float64 `json:"average_score"`
	}
	var divisionData []DivisionPerformance
	
	baseQuery := `
		SELECT d.name as division_name, COALESCE(AVG(e.total_score), 0) as average_score
		FROM divisions d
		LEFT JOIN employees emp ON emp.division_id = d.id
		LEFT JOIN evaluations e ON e.employee_id = emp.id AND e.status = 'submitted'`
	
	if periodID != "" {
		baseQuery += " AND e.period_id = ?"
	}
	baseQuery += " GROUP BY d.id ORDER BY average_score DESC"

	if periodID != "" {
		dc.DB.Raw(baseQuery, periodID).Scan(&divisionData)
	} else {
		dc.DB.Raw(baseQuery).Scan(&divisionData)
	}

	// 3. MENGHITUNG RATA-RATA PERUSAHAAN & DIVISI TERBAIK
	var companyAvgScore float64
	var topDivisionName = "Belum Ada Data"
	
	if len(divisionData) > 0 && divisionData[0].AverageScore > 0 {
		topDivisionName = divisionData[0].DivisionName
	}
	
	avgQuery := "SELECT COALESCE(AVG(total_score), 0) FROM evaluations WHERE status = 'submitted'"
	if periodID != "" {
		dc.DB.Raw(avgQuery+" AND period_id = ?", periodID).Scan(&companyAvgScore)
	} else {
		dc.DB.Raw(avgQuery).Scan(&companyAvgScore)
	}

	// 4. TREN KINERJA (6 PERIODE TERAKHIR - Data Baru Untuk Grafik Garis)
	type TrendData struct {
		PeriodName   string  `json:"period_name"`
		AverageScore float64 `json:"average_score"`
	}
	var trends []TrendData
	dc.DB.Raw(`
		SELECT p.name as period_name, COALESCE(AVG(e.total_score), 0) as average_score 
		FROM evaluation_periods p
		LEFT JOIN evaluations e ON e.period_id = p.id AND e.status = 'submitted'
		GROUP BY p.id, p.name, p.start_date
		ORDER BY p.start_date ASC
		LIMIT 6
	`).Scan(&trends)

	// 5. DISTRIBUSI STATUS (Untuk Donut Chart)
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

	// 6. TOP PEGAWAI TERBAIK
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

	// 7. GABUNGKAN SEMUA RESPON
	responseData := gin.H{
		"metrics": gin.H{
			"company_average_score": companyAvgScore,
			"top_division":          topDivisionName,
			"total_completed_evals": totalCompletedEvals,
		},
		"division_performance": divisionData,
		"company_trends":       trends,
		"status_distribution":  statusCounts,
		"top_employees":        topEmployees,
	}

	Response(c, http.StatusOK, "Data Dashboard Eksekutif Berhasil Diambil", responseData)
}