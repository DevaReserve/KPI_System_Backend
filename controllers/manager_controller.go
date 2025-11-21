package controllers

import (
	"KPI_System_Backend/db_var"
	"KPI_System_Backend/models"
	"net/http"
	"time" // Pastikan 'time' ada di import

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ManagerController struct {
	DB *gorm.DB
}

// NewManagerController adalah "constructor"
func NewManagerController(db *gorm.DB) *ManagerController {
	return &ManagerController{DB: db}
}

// --- Struct untuk Request Binding ---

type EvaluationSubmitRequest struct {
	Feedback string `json:"feedback"`
	Scores   []struct {
		ScoreID uint   `json:"score_id" binding:"required"`
		Score   int    `json:"score" binding:"required,min=1,max=5"`
		Notes   string `json:"notes"`
	} `json:"scores" binding:"required"`
}

// --- Helper Functions ---

// getActivePeriod: Helper internal untuk mendapatkan periode yang sedang aktif
func (mc *ManagerController) getActivePeriod() (*models.EvaluationPeriod, error) {
	var activePeriod models.EvaluationPeriod
	if err := mc.DB.Where("is_active = ?", true).First(&activePeriod).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &activePeriod, nil
}

// --- Manager Functions ---

// GetMyTeam: Mendapatkan daftar pegawai yang harus dinilai oleh manajer
// @Route: GET /api/manager/my-team
func (mc *ManagerController) GetMyTeam(c *gin.Context) {
	// Ambil userID (yang merupakan ID Manajer) dari token
	managerUserID, _ := c.Get("userID")

	// 1. Cari EmployeeID si manajer
	var managerUser models.User
	if err := mc.DB.First(&managerUser, managerUserID).Error; err != nil {
		Response(c, http.StatusNotFound, "Data manajer tidak ditemukan", nil)
		return
	}
	managerEmployeeID := managerUser.EmployeeID

	// 2. Cari semua pegawai yang atasan langsungnya adalah manajer ini
	var team []models.EmployeeDetail
	query := mc.DB.Model(&models.Employee{}).
		Select("employees.*, divisions.name as division_name").
		Joins("left join divisions on divisions.id = employees.division_id").
		Where("employees.direct_supervisor_id = ? AND employees.is_active = ?", managerEmployeeID, true)

	if err := query.Scan(&team).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil data tim", nil)
		return
	}

	Response(c, http.StatusOK, "Data tim berhasil diambil", team)
}

// GetTeamEvaluationStatus: Mendapatkan status evaluasi tim untuk periode aktif
// @Route: GET /api/manager/team-status
func (mc *ManagerController) GetTeamEvaluationStatus(c *gin.Context) {
	// 1. Dapatkan periode aktif
	activePeriod, err := mc.getActivePeriod()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			Response(c, http.StatusBadRequest, "Saat ini tidak ada periode evaluasi yang aktif", nil)
			return
		}
		Response(c, http.StatusInternalServerError, "Gagal mendapatkan periode aktif", nil)
		return
	}

	// 2. Ambil ID manajer
	managerUserID, _ := c.Get("userID")
	var managerUser models.User
	if err := mc.DB.First(&managerUser, managerUserID).Error; err != nil {
		Response(c, http.StatusNotFound, "Data manajer tidak ditemukan", nil)
		return
	}
	managerEmployeeID := managerUser.EmployeeID

	// 3. Ambil data tim (mirip GetMyTeam)
	var team []models.Employee
	if err := mc.DB.Where("direct_supervisor_id = ? AND is_active = ?", managerEmployeeID, true).Find(&team).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil data tim", nil)
		return
	}

	// 4. Buat response
	type TeamStatusResponse struct {
		EmployeeID     uint   `json:"employee_id"`
		EmployeeName   string `json:"employee_name"`
		EvaluationID   *uint  `json:"evaluation_id"` // ID evaluasi jika sudah dibuat
		EvaluationStatus string `json:"evaluation_status"` // "Belum Dibuat", "Draft", "Submitted"
	}

	var response []TeamStatusResponse
	
	// Loop untuk setiap anggota tim
	for _, employee := range team {
		var evaluation models.Evaluation
		status := TeamStatusResponse{
			EmployeeID:     employee.ID,
			EmployeeName:   employee.Name,
			EvaluationStatus: "Belum Dibuat", // Default
		}
		
		// Cek apakah sudah ada evaluasi untuk pegawai ini di periode aktif
		err := mc.DB.Where("employee_id = ? AND period_id = ?", employee.ID, activePeriod.ID).First(&evaluation).Error
		
		if err == nil { // Evaluasi ditemukan
			status.EvaluationID = &evaluation.ID
			status.EvaluationStatus = evaluation.Status
		}
		
		response = append(response, status)
	}
	
	Response(c, http.StatusOK, "Status evaluasi tim berhasil diambil", response)
}


// StartEvaluation: Membuat draf evaluasi untuk seorang pegawai
// @Route: POST /api/manager/evaluations/start
func (mc *ManagerController) StartEvaluation(c *gin.Context) {
	var req struct {
		EmployeeID uint `json:"employee_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid, 'employee_id' diperlukan", nil)
		return
	}

	// 1. Dapatkan periode aktif
	activePeriod, err := mc.getActivePeriod()
	if err != nil {
		Response(c, http.StatusBadRequest, "Tidak ada periode evaluasi yang aktif", nil)
		return
	}
	
	// 2. Ambil data manajer (evaluator)
	managerUserID, _ := c.Get("userID")
	var managerUser models.User
	mc.DB.First(&managerUser, managerUserID)
	evaluatorEmployeeID := managerUser.EmployeeID
	
	// 3. Ambil data pegawai yang akan dinilai
	var employee models.Employee
	if err := mc.DB.First(&employee, req.EmployeeID).Error; err != nil {
		Response(c, http.StatusNotFound, "Pegawai tidak ditemukan", nil)
		return
	}
	
	// 4. Keamanan: Pastikan manajer ini adalah atasan si pegawai
	if employee.DirectSupervisorID == nil || *employee.DirectSupervisorID != evaluatorEmployeeID {
		Response(c, http.StatusForbidden, "Anda bukan atasan langsung dari pegawai ini", nil)
		return
	}
	
	// 5. Cek apakah evaluasi sudah ada
	var existingEval models.Evaluation
	if err := mc.DB.Where("employee_id = ? AND period_id = ?", req.EmployeeID, activePeriod.ID).First(&existingEval).Error; err == nil {
		Response(c, http.StatusConflict, "Evaluasi untuk pegawai ini di periode ini sudah ada", existingEval)
		return
	}

	// 6. Dapatkan semua Indikator yang relevan
	var indicators []models.PerformanceIndicator
	// Ambil indikator "umum" (DivisionID IS NULL) ATAU "spesifik" (DivisionID = divisi pegawai)
	mc.DB.Where("indicator_type = ? AND division_id IS NULL", db_var.IndicatorTypeUmum).
		Or("indicator_type = ? AND division_id = ?", db_var.IndicatorTypeSpesifik, employee.DivisionID).
		Find(&indicators)

	if len(indicators) == 0 {
		Response(c, http.StatusBadRequest, "Tidak ada indikator penilaian yang di-set untuk divisi ini atau umum", nil)
		return
	}
	
	// 7. Mulai Transaksi
	tx := mc.DB.Begin()
	
	// 7a. Buat 'Evaluation' (header)
	evaluation := models.Evaluation{
		EmployeeID:  req.EmployeeID,
		EvaluatorID: evaluatorEmployeeID,
		PeriodID:    activePeriod.ID,
		TotalScore:  0, // Belum dihitung
		Status:      db_var.EvaluationStatusDraft, // Status draf
	}
	if err := tx.Create(&evaluation).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusInternalServerError, "Gagal membuat header evaluasi", nil)
		return
	}

	// 7b. Buat 'EvaluationScore' (detail) untuk setiap indikator
	var scores []models.EvaluationScore
	for _, indicator := range indicators {
		score := models.EvaluationScore{
			EvaluationID:  evaluation.ID,
			IndicatorID:   indicator.ID,
			Score:         0, // Default 0 (belum dinilai)
			ConvertedScore: 0,
		}
		scores = append(scores, score)
	}

	if err := tx.Create(&scores).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusInternalServerError, "Gagal membuat detail skor evaluasi", nil)
		return
	}

	// 8. Commit Transaksi
	tx.Commit()

	// 9. Kembalikan ID evaluasi yang baru
	Response(c, http.StatusCreated, "Draf evaluasi berhasil dibuat", gin.H{"evaluation_id": evaluation.ID})
}

// GetEvaluationDetail: Mengambil form evaluasi yang siap diisi
// @Route: GET /api/manager/evaluations/:id
func (mc *ManagerController) GetEvaluationDetail(c *gin.Context) {
	id := c.Param("id")
	
	// Ambil data evaluasi utama
	var evaluation models.Evaluation
	if err := mc.DB.First(&evaluation, id).Error; err != nil {
		Response(c, http.StatusNotFound, "Evaluasi tidak ditemukan", nil)
		return
	}

	// Keamanan: Cek apakah manajer ini yang punya
	managerUserID, _ := c.Get("userID")
	var managerUser models.User
	mc.DB.First(&managerUser, managerUserID)
	if evaluation.EvaluatorID != managerUser.EmployeeID {
		Response(c, http.StatusForbidden, "Anda tidak memiliki akses ke evaluasi ini", nil)
		return
	}
	
	// Ambil semua skor yang terhubung, preload Indikator dan Divisi
	var scores []models.EvaluationScore
	mc.DB.Preload("Indicator").
		Preload("Indicator.Division").
		Where("evaluation_id = ?", id).
		Find(&scores)

	// Ambil data pegawai yang dinilai
	var employee models.EmployeeDetail
	mc.DB.Model(&models.Employee{}).
		Select("employees.*, divisions.name as division_name").
		Joins("left join divisions on divisions.id = employees.division_id").
		Where("employees.id = ?", evaluation.EmployeeID).
		First(&employee)
		
	// Ambil data periode
	var period models.EvaluationPeriod
	mc.DB.First(&period, evaluation.PeriodID)

	// Gabungkan semua data
	response := gin.H{
		"evaluation_header": evaluation,
		"employee_detail":   employee,
		"period_detail":     period,
		"scores":            scores,
	}

	Response(c, http.StatusOK, "Detail evaluasi berhasil diambil", response)
}


// SubmitEvaluation: Menyimpan & mengirimkan form evaluasi
// @Route: PUT /api/manager/evaluations/:id/submit
func (mc *ManagerController) SubmitEvaluation(c *gin.Context) {
	id := c.Param("id")

	// 1. Ambil data evaluasi yang ada
	var evaluation models.Evaluation
	if err := mc.DB.First(&evaluation, id).Error; err != nil {
		Response(c, http.StatusNotFound, "Evaluasi tidak ditemukan", nil)
		return
	}
	
	// 2. Keamanan: Cek manajer
	managerUserID, _ := c.Get("userID")
	var managerUser models.User
	mc.DB.First(&managerUser, managerUserID)
	if evaluation.EvaluatorID != managerUser.EmployeeID {
		Response(c, http.StatusForbidden, "Anda tidak memiliki akses ke evaluasi ini", nil)
		return
	}
	
	// 3. Cek status (tidak bisa submit ulang)
	if evaluation.Status == db_var.EvaluationStatusSubmitted {
		Response(c, http.StatusBadRequest, "Evaluasi ini sudah disubmit sebelumnya", nil)
		return
	}

	// 4. Bind request JSON
	var req EvaluationSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid", err.Error())
		return
	}

	// 5. Mulai Transaksi
	tx := mc.DB.Begin()
	var totalWeightedScore float64 = 0

	// 6. Loop dan update setiap skor
	for _, scoreReq := range req.Scores {
		var score models.EvaluationScore
		// Ambil skor DAN indikator (untuk bobot)
		if err := tx.Preload("Indicator").Where("id = ? AND evaluation_id = ?", scoreReq.ScoreID, id).First(&score).Error; err != nil {
			tx.Rollback()
			Response(c, http.StatusBadRequest, "Data skor tidak cocok", nil)
			return
		}
		
		// Update skor
		score.Score = scoreReq.Score
		score.Notes = scoreReq.Notes
		// Konversi skor (1-5) ke poin (20-100)
		score.ConvertedScore = db_var.ScoreConversion[scoreReq.Score]
		
		if err := tx.Save(&score).Error; err != nil {
			tx.Rollback()
			Response(c, http.StatusInternalServerError, "Gagal menyimpan skor", nil)
			return
		}
		
		// 7. Hitung skor tertimbang
		// (Skor Poin * Bobot %)
		weightedScore := float64(score.ConvertedScore) * (score.Indicator.Weight / 100.0)
		totalWeightedScore += weightedScore
	}
	
	// 8. Update 'Evaluation' (header)
	evaluation.Feedback = req.Feedback
	evaluation.TotalScore = totalWeightedScore
	evaluation.Status = db_var.EvaluationStatusSubmitted // UBAH STATUS
	
	// --- INI PERBAIKANNYA ---
	now := time.Now()
	evaluation.SubmittedAt = &now // Ambil alamat memori dari 'now'
	// --- AKHIR PERBAIKAN ---
	
	if err := tx.Save(&evaluation).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusInternalServerError, "Gagal menyimpan evaluasi akhir", nil)
		return
	}

	// 9. Commit Transaksi
	if err := tx.Commit().Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menyimpan perubahan", nil)
		return
	}

	Response(c, http.StatusOK, "Evaluasi berhasil disubmit", evaluation)
}