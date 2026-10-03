package controllers

import (
	"KPI_System_Backend/db_var"
	"KPI_System_Backend/models"
	"errors"
	"fmt" // Tambahkan fmt untuk debugging
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MyPerformanceController struct {
	DB *gorm.DB
}

func NewMyPerformanceController(db *gorm.DB) *MyPerformanceController {
	return &MyPerformanceController{DB: db}
}

// --- Helper Internal ---

// getEmployeeIDFromToken: Mengambil EmployeeID yang login dari token
func (pc *MyPerformanceController) getEmployeeIDFromToken(c *gin.Context) (uint, error) {
	// 1. Ambil data dari Context (diset oleh Middleware)
	val, exists := c.Get("userID")
	if !exists {
		return 0, errors.New("user ID tidak ditemukan dalam token context")
	}

	// 2. Pastikan tipe datanya uint (sesuai middleware)
	// Gunakan type assertion yang aman
	userID, ok := val.(uint)
	if !ok {
		// Coba casting dari float64 (kadang JWT numeric jadi float)
		if floatVal, okFloat := val.(float64); okFloat {
			userID = uint(floatVal)
		} else {
			return 0, errors.New("format user ID invalid")
		}
	}

	// 3. Cari User di Database
	var user models.User
	if err := pc.DB.First(&user, userID).Error; err != nil {
		return 0, err // User tidak ada di tabel users
	}

	// 4. Debugging: Cetak di terminal server
	fmt.Printf("[DEBUG] Login sebagai UserID: %d, EmployeeID: %d, Role: %s\n", user.ID, user.EmployeeID, user.Role)

	return user.EmployeeID, nil
}

// --- Employee Functions ---

// GetMyPerformanceHistory: Mendapatkan riwayat semua evaluasi yang sudah selesai
// @Route: GET /api/employee/history
func (pc *MyPerformanceController) GetMyPerformanceHistory(c *gin.Context) {
	// 1. Dapatkan EmployeeID dari token
	employeeID, err := pc.getEmployeeIDFromToken(c)
	if err != nil {
		Response(c, http.StatusNotFound, "Data pegawai tidak ditemukan (Token Invalid)", nil)
		return
	}

	fmt.Printf("[DEBUG] Mencari Evaluasi untuk EmployeeID: %d dengan status 'submitted'\n", employeeID)

	// 2. Query data evaluasi
	var evaluations []models.Evaluation
	if err := pc.DB.
		Where("employee_id = ? AND status = ?", employeeID, "submitted"). // Hardcode string 'submitted' biar aman
		Order("submitted_at desc").
		Find(&evaluations).Error; err != nil {
		
		Response(c, http.StatusInternalServerError, "Gagal mengambil riwayat evaluasi", nil)
		return
	}

	fmt.Printf("[DEBUG] Ditemukan %d data evaluasi\n", len(evaluations))

	// 3. Mapping response
	type HistoryResponse struct {
		models.Evaluation
		PeriodName string `json:"period_name"`
	}

	var response []HistoryResponse
	for _, e := range evaluations {
		var pName string = "Periode Tidak Diketahui"
		
		// Cari nama periode manual
		if e.PeriodID != 0 {
			var p models.EvaluationPeriod
			if err := pc.DB.Unscoped().First(&p, e.PeriodID).Error; err == nil {
				pName = p.Name
			}
		}

		res := HistoryResponse{
			Evaluation: e,
			PeriodName: pName,
		}
		response = append(response, res)
	}

	// Return array kosong [] jika null
	if response == nil {
		response = []HistoryResponse{}
	}

	Response(c, http.StatusOK, "Riwayat evaluasi berhasil diambil", response)
}

// GetMyLatestPerformance: Mendapatkan 1 evaluasi terbaru (untuk dashboard)
// @Route: GET /api/employee/latest
func (pc *MyPerformanceController) GetMyLatestPerformance(c *gin.Context) {
	employeeID, err := pc.getEmployeeIDFromToken(c)
	if err != nil {
		Response(c, http.StatusNotFound, "Data pegawai tidak ditemukan", nil)
		return
	}

	var evaluation models.Evaluation
	if err := pc.DB.Preload("Period").
		Where("employee_id = ? AND status = ?", employeeID, "submitted").
		Order("submitted_at desc").
		First(&evaluation).Error; err != nil {
		
		if err == gorm.ErrRecordNotFound {
			Response(c, http.StatusOK, "Belum ada data evaluasi yang selesai", nil)
			return
		}
		Response(c, http.StatusInternalServerError, "Gagal mengambil evaluasi terbaru", nil)
		return
	}
	
	Response(c, http.StatusOK, "Evaluasi terbaru berhasil diambil", evaluation)
}

// GetMyEvaluationDetail: Mendapatkan detail lengkap 1 evaluasi
// @Route: GET /api/employee/evaluations/:id
func (pc *MyPerformanceController) GetMyEvaluationDetail(c *gin.Context) {
	id := c.Param("id")

	employeeID, err := pc.getEmployeeIDFromToken(c)
	if err != nil {
		Response(c, http.StatusNotFound, "Data pegawai tidak ditemukan", nil)
		return
	}

	var evaluation models.Evaluation
	if err := pc.DB.First(&evaluation, id).Error; err != nil {
		Response(c, http.StatusNotFound, "Evaluasi tidak ditemukan", nil)
		return
	}

	// Pengecekan Keamanan:
	// Izinkan jika ini milik pegawai yg login OR jika yg login adalah Admin/Manager (untuk view detail)
	role, _ := c.Get("role")
	userRole := role.(string)

	isOwner := evaluation.EmployeeID == employeeID
	isEvaluator := (userRole == db_var.RoleAdmin || userRole == db_var.RoleManager)

	if !isOwner && !isEvaluator {
		// Cek apakah pegawai yang login satu divisi dengan pemilik evaluasi (Untuk transparansi Top 5 Divisi)
		var viewer models.Employee
		pc.DB.First(&viewer, employeeID)
		
		var owner models.Employee
		pc.DB.First(&owner, evaluation.EmployeeID)
		
		if viewer.DivisionID != owner.DivisionID {
			Response(c, http.StatusForbidden, "Anda tidak memiliki akses ke evaluasi ini", nil)
			return
		}
	}

	// Ambil data pendukung
	var scores []models.EvaluationScore
	pc.DB.Preload("Indicator").
		Preload("Indicator.Division").
		Where("evaluation_id = ?", id).
		Find(&scores)

	var evaluator models.Employee
	pc.DB.First(&evaluator, evaluation.EvaluatorID)

	var period models.EvaluationPeriod
	pc.DB.First(&period, evaluation.PeriodID)
	
	// Tambahan: Ambil data pegawai yang dinilai (agar nama muncul di header saat admin lihat detail)
	var employeeDetail models.Employee
	pc.DB.First(&employeeDetail, evaluation.EmployeeID)

	response := gin.H{
		"evaluation_header": evaluation,
		"evaluator_name":    evaluator.Name,
		"employee_detail":   employeeDetail, // Penting untuk header
		"period_detail":     period,
		"scores":            scores,
	}

	Response(c, http.StatusOK, "Detail evaluasi berhasil diambil", response)
}

// GetMyTopOneStatus: Cek apakah pegawai ini merupakan top performer (skor tertinggi) di periode aktif
func (pc *MyPerformanceController) GetMyTopOneStatus(c *gin.Context) {
	employeeID, err := pc.getEmployeeIDFromToken(c)
	if err != nil {
		Response(c, http.StatusNotFound, "Data pegawai tidak ditemukan", nil)
		return
	}

	// 1. Dapatkan periode aktif
	var activePeriod models.EvaluationPeriod
	if err := pc.DB.Where("is_active = ?", true).First(&activePeriod).Error; err != nil {
		Response(c, http.StatusOK, "Tidak ada periode aktif", gin.H{"is_top_one": false})
		return
	}

	// 2. Dapatkan pegawai dengan nilai tertinggi di periode tersebut
	type TopOneEval struct {
		EmployeeID uint
	}
	var topOne TopOneEval
	if err := pc.DB.Table("evaluations").
		Select("employee_id").
		Where("period_id = ? AND status = ?", activePeriod.ID, "submitted").
		Order("total_score DESC").
		Limit(1).
		Scan(&topOne).Error; err != nil || topOne.EmployeeID == 0 {
		Response(c, http.StatusOK, "Tidak ada data evaluasi", gin.H{"is_top_one": false})
		return
	}

	isTopOne := topOne.EmployeeID == employeeID
	Response(c, http.StatusOK, "Berhasil memeriksa status top 1", gin.H{
		"is_top_one": isTopOne,
	})
}

// GetTopInDivision: Mendapatkan Top 5 Pegawai di divisi yang sama dengan pegawai yang login
func (pc *MyPerformanceController) GetTopInDivision(c *gin.Context) {
	employeeID, err := pc.getEmployeeIDFromToken(c)
	if err != nil {
		Response(c, http.StatusNotFound, "Data pegawai tidak ditemukan", nil)
		return
	}

	// Ambil Divisi Pegawai
	var employee models.Employee
	if err := pc.DB.First(&employee, employeeID).Error; err != nil {
		Response(c, http.StatusNotFound, "Data pegawai tidak ditemukan", nil)
		return
	}

	var activePeriod models.EvaluationPeriod
	if err := pc.DB.Where("is_active = ?", true).First(&activePeriod).Error; err != nil {
		Response(c, http.StatusOK, "Tidak ada periode aktif", []string{})
		return
	}

	type TopDivisionEmployee struct {
		EvaluationID uint    `json:"evaluation_id"`
		Name         string  `json:"name"`
		TotalScore   float64 `json:"total_score"`
		Grade      string  `json:"grade"`
	}
	var topEmployees []TopDivisionEmployee

	pc.DB.Table("evaluations e").
		Select("e.id as evaluation_id, emp.name, e.total_score").
		Joins("JOIN employees emp ON emp.id = e.employee_id").
		Where("emp.division_id = ? AND e.period_id = ? AND e.status = ?", employee.DivisionID, activePeriod.ID, "submitted").
		Order("e.total_score DESC").
		Limit(5).
		Scan(&topEmployees)

    for i, emp := range topEmployees {
        if emp.TotalScore >= 86 {
            topEmployees[i].Grade = "A"
        } else if emp.TotalScore >= 71 {
            topEmployees[i].Grade = "B"
        } else if emp.TotalScore >= 56 {
            topEmployees[i].Grade = "C"
        } else if emp.TotalScore >= 41 {
            topEmployees[i].Grade = "D"
        } else {
            topEmployees[i].Grade = "E"
        }
    }

	if topEmployees == nil {
		topEmployees = []TopDivisionEmployee{}
	}

	Response(c, http.StatusOK, "Berhasil mengambil top divisi", topEmployees)
}