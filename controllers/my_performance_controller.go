package controllers

import (
	"KPI_System_Backend/db_var"
	"KPI_System_Backend/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MyPerformanceController struct {
	DB *gorm.DB
}

// NewMyPerformanceController adalah "constructor"
func NewMyPerformanceController(db *gorm.DB) *MyPerformanceController {
	return &MyPerformanceController{DB: db}
}

// --- Helper Internal ---

// getEmployeeIDFromToken: Mengambil EmployeeID yang login dari token
func (pc *MyPerformanceController) getEmployeeIDFromToken(c *gin.Context) (uint, error) {
	userID, _ := c.Get("userID")
	var user models.User
	if err := pc.DB.First(&user, userID).Error; err != nil {
		return 0, err
	}
	return user.EmployeeID, nil
}

// --- Employee Functions ---

// GetMyPerformanceHistory: Mendapatkan riwayat semua evaluasi yang sudah selesai
// @Route: GET /api/employee/history
func (pc *MyPerformanceController) GetMyPerformanceHistory(c *gin.Context) {
	// 1. Dapatkan EmployeeID dari token
	employeeID, err := pc.getEmployeeIDFromToken(c)
	if err != nil {
		Response(c, http.StatusNotFound, "Data pegawai tidak ditemukan", nil)
		return
	}

	// 2. Cari semua evaluasi yang:
	//    - Milik pegawai ini (employee_id = ?)
	//    - Sudah di-submit (status = "submitted")
	var evaluations []models.Evaluation
	if err := pc.DB.Preload("Period"). // Ambil juga data periode
		Where("employee_id = ? AND status = ?", employeeID, db_var.EvaluationStatusSubmitted).
		Order("submitted_at desc"). // Urutkan dari yang terbaru
		Find(&evaluations).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil riwayat evaluasi", nil)
		return
	}

	Response(c, http.StatusOK, "Riwayat evaluasi berhasil diambil", evaluations)
}

// GetMyLatestPerformance: Mendapatkan 1 evaluasi terbaru (untuk dashboard)
// @Route: GET /api/employee/latest
func (pc *MyPerformanceController) GetMyLatestPerformance(c *gin.Context) {
	// 1. Dapatkan EmployeeID dari token
	employeeID, err := pc.getEmployeeIDFromToken(c)
	if err != nil {
		Response(c, http.StatusNotFound, "Data pegawai tidak ditemukan", nil)
		return
	}

	// 2. Cari 1 evaluasi terbaru
	var evaluation models.Evaluation
	if err := pc.DB.Preload("Period").
		Where("employee_id = ? AND status = ?", employeeID, db_var.EvaluationStatusSubmitted).
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
	// 1. Ambil ID Evaluasi dari URL
	id := c.Param("id")

	// 2. Dapatkan EmployeeID dari token (untuk keamanan)
	employeeID, err := pc.getEmployeeIDFromToken(c)
	if err != nil {
		Response(c, http.StatusNotFound, "Data pegawai tidak ditemukan", nil)
		return
	}

	// 3. Ambil data evaluasi utama
	var evaluation models.Evaluation
	if err := pc.DB.First(&evaluation, id).Error; err != nil {
		Response(c, http.StatusNotFound, "Evaluasi tidak ditemukan", nil)
		return
	}

	// 4. --- Pengecekan Keamanan Ganda ---
	// 4a. Cek apakah ini MILIKNYA
	if evaluation.EmployeeID != employeeID {
		Response(c, http.StatusForbidden, "Anda tidak memiliki akses ke evaluasi ini", nil)
		return
	}
	// 4b. Cek apakah sudah di-SUBMIT (pegawai tidak boleh lihat draf)
	if evaluation.Status != db_var.EvaluationStatusSubmitted {
		Response(c, http.StatusForbidden, "Evaluasi ini belum selesai dinilai", nil)
		return
	}

	// 5. Ambil semua data pendukung (sama seperti di manager_controller)
	
	// Ambil skor, indikator, dan divisi
	var scores []models.EvaluationScore
	pc.DB.Preload("Indicator").
		Preload("Indicator.Division").
		Where("evaluation_id = ?", id).
		Find(&scores)

	// Ambil data penilai (evaluator/manajer)
	var evaluator models.Employee
	pc.DB.First(&evaluator, evaluation.EvaluatorID)

	// Ambil data periode
	var period models.EvaluationPeriod
	pc.DB.First(&period, evaluation.PeriodID)

	// Gabungkan semua data
	response := gin.H{
		"evaluation_header": evaluation,
		"evaluator_name":    evaluator.Name, // Cukup namanya saja
		"period_detail":     period,
		"scores":            scores,
	}

	Response(c, http.StatusOK, "Detail evaluasi berhasil diambil", response)
}