package controllers

import (
	"KPI_System_Backend/logger"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DashboardController struct {
	DB *gorm.DB
}

func NewDashboardController(db *gorm.DB) *DashboardController {
	return &DashboardController{DB: db}
}

// GetCompanyPerformance: API Khusus untuk Executive Dashboard (CEO)
func (dc *DashboardController) GetCompanyPerformance(c *gin.Context) {
	// 1. VALIDASI HAK AKSES EKSEKUTIF (Mengambil data dari Middleware)
	isExecutiveInterface, exists := c.Get("is_executive")

	// Jika tidak ada data eksekutif, atau nilainya false, TOLAK AKSESNYA
	if !exists || isExecutiveInterface.(bool) == false {
		logger.Warn("Akses ilegal ke Executive Dashboard ditolak", zap.Any("userID", c.MustGet("userID")))
		Response(c, http.StatusForbidden, "Akses Ditolak: Fitur ini khusus untuk level Eksekutif (CEO)", nil)
		return
	}

	// 2. QUERY KE DATABASE (Mencari rata-rata nilai KPI per divisi)
	// Kita buat struktur sementara untuk menampung hasil query
	type DivisionPerformance struct {
		DivisionName string  `json:"division_name"`
		AverageScore float64 `json:"average_score"`
	}

	var performanceData []DivisionPerformance

	/* Contoh Query GORM tingkat lanjut (asumsi tabel divisions, employees, dan evaluations sudah direlasikan).
	   Query ini akan menghitung rata-rata Total Skor dari seluruh evaluasi yang ada di setiap divisi.
	*/
	periodID := c.Query("period_id")

	baseQuery := `
		SELECT d.name as division_name, COALESCE(AVG(e.total_score), 0) as average_score
		FROM divisions d
		LEFT JOIN employees emp ON emp.division_id = d.id
		LEFT JOIN evaluations e ON e.employee_id = emp.id AND e.status = 'submitted'`

	if periodID != "" {
		baseQuery += " AND e.period_id = ?"
	}

	baseQuery += "\n\t\tGROUP BY d.id"

	var err error
	if periodID != "" {
		err = dc.DB.Raw(baseQuery, periodID).Scan(&performanceData).Error
	} else {
		err = dc.DB.Raw(baseQuery).Scan(&performanceData).Error
	}

	if err != nil {
		logger.Error("Gagal mengambil data performa perusahaan", zap.Error(err))
		Response(c, http.StatusInternalServerError, "Gagal memuat data dashboard", nil)
		return
	}

	// 3. KEMBALIKAN DATA KE FRONTEND
	Response(c, http.StatusOK, "Data Performa Perusahaan Berhasil Diambil", performanceData)
}
