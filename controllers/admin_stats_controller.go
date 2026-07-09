package controllers

import (
	"KPI_System_Backend/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminStatsController struct {
	DB *gorm.DB
}

func NewAdminStatsController(db *gorm.DB) *AdminStatsController {
	return &AdminStatsController{DB: db}
}

// ============================================================
// GET /api/admin/dashboard
// Mengembalikan statistik lengkap untuk dashboard admin
// ============================================================
func (ac *AdminStatsController) GetAdminDashboard(c *gin.Context) {
	periodID := c.Query("period_id")

	// 1. Total Pegawai Aktif
	var totalActiveEmployees int64
	ac.DB.Model(&models.Employee{}).Where("is_active = ?", true).Count(&totalActiveEmployees)

	// 2. Periode yang dipilih / periode aktif
	var activePeriod models.EvaluationPeriod
	if periodID != "" {
		if err := ac.DB.Where("id = ?", periodID).First(&activePeriod).Error; err != nil {
			ac.DB.Where("is_active = ?", true).First(&activePeriod)
		}
	} else {
		ac.DB.Where("is_active = ?", true).First(&activePeriod)
	}

	// 3. Total evaluasi submitted pada periode yang dipilih
	var totalEvaluationsDone int64
	if activePeriod.ID > 0 {
		ac.DB.Table("evaluations").Where("period_id = ? AND status = ?", activePeriod.ID, "submitted").Count(&totalEvaluationsDone)
	}

	// 4. Total SP Aktif (SP1, SP2, SP3)
	var totalWarnings int64
	ac.DB.Model(&models.Warning{}).Count(&totalWarnings)

	// 5. SP per level
	type WarningLevelCount struct {
		Level string `json:"level"`
		Count int    `json:"count"`
	}
	var warningCounts []WarningLevelCount
	ac.DB.Model(&models.Warning{}).Select("level, count(*) as count").Group("level").Scan(&warningCounts)

	// 6. Distribusi Grade (A/B/C/D/E) periode aktif
	type GradeDistribution struct {
		Grade string `json:"grade"`
		Count int    `json:"count"`
	}
	var gradeDistribution []GradeDistribution

	if activePeriod.ID > 0 {
		// Kita ambil semua total_score dan hitung grade secara Go
		var scores []float64
		ac.DB.Table("evaluations").
			Select("total_score").
			Where("period_id = ? AND status = ?", activePeriod.ID, "submitted").
			Pluck("total_score", &scores)

		gradeMap := map[string]int{"A": 0, "B": 0, "C": 0, "D": 0, "E": 0}
		for _, s := range scores {
			switch {
			case s >= 86:
				gradeMap["A"]++
			case s >= 71:
				gradeMap["B"]++
			case s >= 56:
				gradeMap["C"]++
			case s >= 41:
				gradeMap["D"]++
			default:
				gradeMap["E"]++
			}
		}
		for _, g := range []string{"A", "B", "C", "D", "E"} {
			gradeDistribution = append(gradeDistribution, GradeDistribution{Grade: g, Count: gradeMap[g]})
		}
	}

	// 7. Evaluasi per status (untuk periode aktif)
	type EvalStatusCount struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	var evalStatusCounts []EvalStatusCount
	if activePeriod.ID > 0 {
		ac.DB.Table("evaluations").
			Select("status, count(*) as count").
			Where("period_id = ?", activePeriod.ID).
			Group("status").
			Scan(&evalStatusCounts)
	}

	// 8. Pegawai belum dievaluasi di periode aktif
	type UnevaluatedEmployee struct {
		EmployeeName string `json:"employee_name"`
		DivisionName string `json:"division_name"`
		ManagerName  string `json:"manager_name"`
	}
	var unevaluatedEmployees []UnevaluatedEmployee
	var totalEmployeesForEval int64

	if activePeriod.ID > 0 {
		ac.DB.Model(&models.Employee{}).Where("is_active = ?", true).Count(&totalEmployeesForEval)

		// Dapatkan daftar pegawai yang BELUM di-submit pada periode ini
		ac.DB.Raw(`
			SELECT e.name as employee_name, d.name as division_name, m.name as manager_name
			FROM employees e
			LEFT JOIN divisions d ON e.division_id = d.id
			LEFT JOIN employees m ON m.id = d.manager_id
			WHERE e.is_active = true AND e.id NOT IN (
				SELECT employee_id FROM evaluations WHERE period_id = ? AND status IN ('submitted', 'draft')
			)
		`, activePeriod.ID).Scan(&unevaluatedEmployees)
	}

	notEvaluated := len(unevaluatedEmployees)

	// 9. Rata-rata skor perusahaan periode aktif
	var avgScore float64
	if activePeriod.ID > 0 {
		ac.DB.Table("evaluations").
			Select("COALESCE(AVG(total_score), 0)").
			Where("period_id = ? AND status = ?", activePeriod.ID, "submitted").
			Scan(&avgScore)
	}

	responseData := gin.H{
		"active_period":          activePeriod,
		"total_active_employees": totalActiveEmployees,
		"total_warnings":         totalWarnings,
		"warning_by_level":       warningCounts,
		"grade_distribution":     gradeDistribution,
		"eval_status_counts":     evalStatusCounts,
		"not_evaluated_count":    notEvaluated,
		"unevaluated_employees":  unevaluatedEmployees,
		"company_avg_score":      avgScore,
		"total_evaluations_done": totalEvaluationsDone,
	}

	Response(c, http.StatusOK, "Statistik dashboard admin berhasil diambil", responseData)
}

// ============================================================
// GET /api/admin/warnings
// Mengembalikan semua SP dari semua pegawai (bisa filter)
// ============================================================
func (ac *AdminStatsController) GetAllWarnings(c *gin.Context) {
	levelFilter := c.Query("level")      // "SP1", "SP2", "SP3"
	employeeID := c.Query("employee_id") // ID Pegawai

	type WarningDetail struct {
		ID           uint   `json:"id"`
		EmployeeID   uint   `json:"employee_id"`
		EmployeeName string `json:"employee_name"`
		DivisionName string `json:"division_name"`
		IssuedByName string `json:"issued_by_name"`
		Level        string `json:"level"`
		Reason       string `json:"reason"`
		Description  string `json:"description"`
		IssuedAt     string `json:"issued_at"`
	}

	query := ac.DB.Table("warnings w").
		Select(`w.id, w.employee_id, 
			emp.name as employee_name, 
			d.name as division_name,
			issuer.name as issued_by_name,
			w.level, w.reason, w.description,
			w.issued_at`).
		Joins("LEFT JOIN employees emp ON emp.id = w.employee_id").
		Joins("LEFT JOIN divisions d ON d.id = emp.division_id").
		Joins("LEFT JOIN employees issuer ON issuer.id = w.issued_by_id").
		Order("w.issued_at DESC")

	if levelFilter != "" {
		query = query.Where("w.level = ?", levelFilter)
	}
	if employeeID != "" {
		query = query.Where("w.employee_id = ?", employeeID)
	}

	var results []WarningDetail
	if err := query.Scan(&results).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil data SP", nil)
		return
	}

	if results == nil {
		results = []WarningDetail{}
	}

	Response(c, http.StatusOK, "Data semua SP berhasil diambil", results)
}

// ============================================================
// GET /api/admin/divisions/stats?period_id=X
// Mengembalikan statistik kinerja per divisi
// ============================================================
func (ac *AdminStatsController) GetDivisionStats(c *gin.Context) {
	periodID := c.Query("period_id")

	type DivisionStat struct {
		DivisionID     uint    `json:"division_id"`
		DivisionName   string  `json:"division_name"`
		TotalEmployees int     `json:"total_employees"`
		EvaluatedCount int     `json:"evaluated_count"`
		AverageScore   float64 `json:"average_score"`
		TopScore       float64 `json:"top_score"`
		LowestScore    float64 `json:"lowest_score"`
	}

	baseSelect := `
		d.id as division_id,
		d.name as division_name,
		COUNT(DISTINCT emp.id) as total_employees,
		COUNT(DISTINCT CASE WHEN ev.status = 'submitted' THEN ev.employee_id END) as evaluated_count,
		COALESCE(AVG(CASE WHEN ev.status = 'submitted' THEN ev.total_score END), 0) as average_score,
		COALESCE(MAX(CASE WHEN ev.status = 'submitted' THEN ev.total_score END), 0) as top_score,
		COALESCE(MIN(CASE WHEN ev.status = 'submitted' THEN ev.total_score END), 0) as lowest_score
	`

	query := ac.DB.Table("divisions d").
		Select(baseSelect).
		Joins("LEFT JOIN employees emp ON emp.division_id = d.id AND emp.is_active = true").
		Where("d.name != ?", "Board of Directors").
		Group("d.id, d.name").
		Order("average_score DESC")

	if periodID != "" {
		query = query.Joins("LEFT JOIN evaluations ev ON ev.employee_id = emp.id AND ev.period_id = ?", periodID)
	} else {
		query = query.Joins("LEFT JOIN evaluations ev ON ev.employee_id = emp.id")
	}

	var results []DivisionStat
	if err := query.Scan(&results).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil statistik divisi", nil)
		return
	}

	if results == nil {
		results = []DivisionStat{}
	}

	Response(c, http.StatusOK, "Statistik per divisi berhasil diambil", results)
}
