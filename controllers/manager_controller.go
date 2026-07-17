package controllers

import (
	"KPI_System_Backend/config"
	"KPI_System_Backend/db_var"
    "KPI_System_Backend/helper" 
	"KPI_System_Backend/models"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)
	
type ManagerController struct {
	DB *gorm.DB
}

func NewManagerController(db *gorm.DB) *ManagerController {
	return &ManagerController{DB: db}
}

type EvaluationSubmitRequest struct {
	Feedback string `json:"feedback"`
	Scores   []struct {
		ScoreID uint   `json:"score_id" binding:"required"`
		Score   int    `json:"score" binding:"required,min=1,max=5"`
		Notes   string `json:"notes"`
	} `json:"scores" binding:"required"`
}

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

// GetMyTeam: [ANTI-SPAM] Read Only -> No Log
func (mc *ManagerController) GetMyTeam(c *gin.Context) {
	managerUserID, _ := c.Get("userID")

	var managerUser models.User
	if err := mc.DB.First(&managerUser, managerUserID).Error; err != nil {
		Response(c, http.StatusNotFound, "Data manajer tidak ditemukan", nil)
		return
	}
	managerEmployeeID := managerUser.EmployeeID

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

// GetTeamEvaluationStatus: [ANTI-SPAM] Read Only -> No Log
func (mc *ManagerController) GetTeamEvaluationStatus(c *gin.Context) {
	activePeriod, err := mc.getActivePeriod()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			Response(c, http.StatusBadRequest, "Saat ini tidak ada periode evaluasi yang aktif", nil)
			return
		}
		Response(c, http.StatusInternalServerError, "Gagal mendapatkan periode aktif", nil)
		return
	}

	managerUserID, _ := c.Get("userID")
	var managerUser models.User
	if err := mc.DB.First(&managerUser, managerUserID).Error; err != nil {
		Response(c, http.StatusNotFound, "Data manajer tidak ditemukan", nil)
		return
	}
	managerEmployeeID := managerUser.EmployeeID

	var team []models.Employee
	if err := mc.DB.Where("direct_supervisor_id = ? AND is_active = ?", managerEmployeeID, true).Find(&team).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil data tim", nil)
		return
	}

	type TeamStatusResponse struct {
		EmployeeID        uint    `json:"employee_id"`
		EmployeeName      string  `json:"employee_name"`
		ProfilePictureURL string  `json:"profile_picture_url"`
		EvaluationID      *uint   `json:"evaluation_id"`
		EvaluationStatus  string  `json:"evaluation_status"`
		TotalScore        float64 `json:"total_score"`
	}

	var response []TeamStatusResponse
	for _, employee := range team {
		var evaluations []models.Evaluation
		status := TeamStatusResponse{
			EmployeeID:        employee.ID,
			EmployeeName:      employee.Name,
			ProfilePictureURL: employee.ProfilePictureURL,
			EvaluationStatus:  "Belum Dibuat",
			TotalScore:        0,
		}
		
		err := mc.DB.Where("employee_id = ? AND period_id = ?", employee.ID, activePeriod.ID).Limit(1).Find(&evaluations).Error
		
		if err == nil && len(evaluations) > 0 {
			evaluation := evaluations[0]
			status.EvaluationID = &evaluation.ID
			status.EvaluationStatus = evaluation.Status
			status.TotalScore = evaluation.TotalScore
		}
		
		response = append(response, status)
	}
    
    Response(c, http.StatusOK, "Status evaluasi tim berhasil diambil", response)
}

// StartEvaluation: [ANTI-SPAM] Ini hanya inisialisasi draft, tidak perlu dicatat agar log tidak penuh.
func (mc *ManagerController) StartEvaluation(c *gin.Context) {
	var req struct {
		EmployeeID uint `json:"employee_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid, 'employee_id' diperlukan", nil)
		return
	}

	activePeriod, err := mc.getActivePeriod()
	if err != nil {
		Response(c, http.StatusBadRequest, "Tidak ada periode evaluasi yang aktif", nil)
		return
	}
	
	managerUserID, _ := c.Get("userID")
	var managerUser models.User
	mc.DB.First(&managerUser, managerUserID)
	evaluatorEmployeeID := managerUser.EmployeeID
	
	var employee models.Employee
	if err := mc.DB.First(&employee, req.EmployeeID).Error; err != nil {
		Response(c, http.StatusNotFound, "Pegawai tidak ditemukan", nil)
		return
	}
	
	if employee.DirectSupervisorID == nil || *employee.DirectSupervisorID != evaluatorEmployeeID {
		Response(c, http.StatusForbidden, "Anda bukan atasan langsung dari pegawai ini", nil)
		return
	}
	
	var existingEval models.Evaluation
	if err := mc.DB.Where("employee_id = ? AND period_id = ?", req.EmployeeID, activePeriod.ID).First(&existingEval).Error; err == nil {
		Response(c, http.StatusConflict, "Evaluasi untuk pegawai ini di periode ini sudah ada", existingEval)
		return
	}

	var indicators []models.PerformanceIndicator
	if err := mc.DB.Where(
		mc.DB.Where("indicator_type = ?", db_var.IndicatorTypeUmum).
		Or("indicator_type = ? AND (division_id = ? OR id IN (SELECT performance_indicator_id FROM indicator_divisions WHERE division_id = ?))", db_var.IndicatorTypeSpesifik, employee.DivisionID, employee.DivisionID),
	).Find(&indicators).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil indikator penilaian", nil)
		return
	}

	if len(indicators) == 0 {
		Response(c, http.StatusBadRequest, "Tidak ada indikator penilaian yang di-set untuk divisi ini atau umum", nil)
		return
	}
	
	tx := mc.DB.Begin()
	
	evaluation := models.Evaluation{
		EmployeeID:  req.EmployeeID,
		EvaluatorID: evaluatorEmployeeID,
		PeriodID:    activePeriod.ID,
		TotalScore:  0,
		Status:      db_var.EvaluationStatusDraft,
	}
	if err := tx.Create(&evaluation).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusInternalServerError, "Gagal membuat header evaluasi", nil)
		return
	}

	var scores []models.EvaluationScore
	for _, indicator := range indicators {
		score := models.EvaluationScore{
			EvaluationID:   evaluation.ID,
			IndicatorID:    indicator.ID,
			Score:          0,
			ConvertedScore: 0,
		}
		scores = append(scores, score)
	}

	if err := tx.Create(&scores).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusInternalServerError, "Gagal membuat detail skor evaluasi", nil)
		return
	}

	tx.Commit()

	Response(c, http.StatusCreated, "Draf evaluasi berhasil dibuat", gin.H{"evaluation_id": evaluation.ID})
}

// GetEvaluationDetail: [ANTI-SPAM] Read Only -> No Log
func (mc *ManagerController) GetEvaluationDetail(c *gin.Context) {
	id := c.Param("id")
	
	var evaluation models.Evaluation
	if err := mc.DB.First(&evaluation, id).Error; err != nil {
		Response(c, http.StatusNotFound, "Evaluasi tidak ditemukan", nil)
		return
	}

	managerUserID, _ := c.Get("userID")
	var managerUser models.User
	mc.DB.First(&managerUser, managerUserID)
	if evaluation.EvaluatorID != managerUser.EmployeeID {
		Response(c, http.StatusForbidden, "Anda tidak memiliki akses ke evaluasi ini", nil)
		return
	}
	
	var scores []models.EvaluationScore
	mc.DB.Preload("Indicator").
		Preload("Indicator.Division").
		Where("evaluation_id = ?", id).
		Find(&scores)

	var employee models.EmployeeDetail
	mc.DB.Model(&models.Employee{}).
		Select("employees.*, divisions.name as division_name").
		Joins("left join divisions on divisions.id = employees.division_id").
		Where("employees.id = ?", evaluation.EmployeeID).
		First(&employee)
		
	var period models.EvaluationPeriod
	mc.DB.First(&period, evaluation.PeriodID)

	response := gin.H{
		"evaluation_header": evaluation,
		"employee_detail":   employee,
		"period_detail":     period,
		"scores":            scores,
	}

	Response(c, http.StatusOK, "Detail evaluasi berhasil diambil", response)
}

// SubmitEvaluation: Menyimpan & mengirimkan form evaluasi
func (mc *ManagerController) SubmitEvaluation(c *gin.Context) {
	id := c.Param("id")

	var evaluation models.Evaluation
	if err := mc.DB.First(&evaluation, id).Error; err != nil {
		Response(c, http.StatusNotFound, "Evaluasi tidak ditemukan", nil)
		return
	}
	
	managerUserID, _ := c.Get("userID")
    var managerUser models.User
    mc.DB.First(&managerUser, managerUserID)
    
    // <--- TAMBAHKAN 1 BARIS INI --->
    evaluatorEmployeeID := managerUser.EmployeeID 
    
    if evaluation.EvaluatorID != managerUser.EmployeeID {
        Response(c, http.StatusForbidden, "Anda tidak memiliki akses ke evaluasi ini", nil)
        return
    }
	
	if evaluation.Status == db_var.EvaluationStatusSubmitted {
		Response(c, http.StatusBadRequest, "Evaluasi ini sudah disubmit sebelumnya", nil)
		return
	}

    // Ambil nama pegawai yang dinilai untuk log (opsional, tapi informatif)
    var targetEmployee models.Employee
    mc.DB.First(&targetEmployee, evaluation.EmployeeID)

	var req EvaluationSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid", err.Error())
		return
	}

	tx := mc.DB.Begin()
	var totalWeightedScore float64 = 0

	for _, scoreReq := range req.Scores {
		var score models.EvaluationScore
		if err := tx.Preload("Indicator").Where("id = ? AND evaluation_id = ?", scoreReq.ScoreID, id).First(&score).Error; err != nil {
			tx.Rollback()
			Response(c, http.StatusBadRequest, "Data skor tidak cocok", nil)
			return
		}
		
		score.Score = scoreReq.Score
		score.Notes = scoreReq.Notes
		score.ConvertedScore = db_var.ScoreConversion[scoreReq.Score]
		
		if err := tx.Save(&score).Error; err != nil {
			tx.Rollback()
			Response(c, http.StatusInternalServerError, "Gagal menyimpan skor", nil)
			return
		}
		
		weightedScore := float64(score.ConvertedScore) * (score.Indicator.Weight / 100.0)
		totalWeightedScore += weightedScore
	}
	
	evaluation.Feedback = req.Feedback
	evaluation.TotalScore = totalWeightedScore
	evaluation.Status = db_var.EvaluationStatusSubmitted
	
	now := time.Now()
	evaluation.SubmittedAt = &now
	
	if err := tx.Save(&evaluation).Error; err != nil {
        tx.Rollback()
        Response(c, http.StatusInternalServerError, "Gagal menyimpan evaluasi akhir", nil)
        return
    }

	// =================================================================
	// ---> JALANKAN LOGIKA DETEKSI SP OTOMATIS BERJENJANG DI SINI <---
	// =================================================================
	err := CheckAndGenerateAutomaticSP(tx, evaluation.EmployeeID, evaluation.TotalScore, evaluatorEmployeeID)
	if err != nil {
		tx.Rollback() // Batalkan submit nilai jika pembuatan SP mengalami kegagalan sistem
		Response(c, http.StatusInternalServerError, "Gagal memproses pembuatan SP otomatis", nil)
		return
	}
	// =================================================================

if err := tx.Commit().Error; err != nil {
        Response(c, http.StatusInternalServerError, "Gagal menyimpan perubahan", nil)
        return
    }

    // =================================================================
    // ---> BUAT NOTIFIKASI IN-APP KE PEGAWAI <---
    // =================================================================
    var targetUser models.User
    mc.DB.Where("employee_id = ?", targetEmployee.ID).First(&targetUser)
    if targetUser.ID != 0 {
        var period models.EvaluationPeriod
        mc.DB.First(&period, evaluation.PeriodID)
        notif := models.Notification{
            UserID: targetUser.ID,
            Title:  "Evaluasi Selesai",
            Message: fmt.Sprintf("Evaluasi kinerja Anda untuk %s telah selesai dinilai.", period.Name),
            Type:   "evaluation",
        }
        mc.DB.Create(&notif)
    }

    // =================================================================
    // ---> KIRIM EMAIL NOTIFIKASI KE PEGAWAI SECARA ASINKRON <---
    // =================================================================
    // Kita menggunakan perintah 'go func()' agar proses email berjalan
    // di latar belakang (background). Dengan begini, loading aplikasi
    // tidak akan tertahan (nge-lag) saat menunggu respon server Gmail.
go func() {
        // Pastikan pegawai memiliki email di database
        if targetEmployee.Email != "" {
            // 1. Cari nama manajer (evaluator) di database
            var evaluator models.Employee
            mc.DB.First(&evaluator, evaluatorEmployeeID)
            evaluatorName := evaluator.Name

            // 2. Cari nama periode evaluasi
            var period models.EvaluationPeriod
            mc.DB.First(&period, evaluation.PeriodID)
            periodName := period.Name

            // 3. Setup Link Aplikasi (Ganti localhost dengan IP WiFi Anda jika ingin diuji di HP)
            appLink := config.FrontEndURL

            subject := fmt.Sprintf("Pemberitahuan: Evaluasi Kinerja %s Telah Selesai", periodName)
            htmlBody := fmt.Sprintf(`
                <div style="font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; color: #333; max-width: 600px; margin: auto; border: 1px solid #e2e8f0; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 6px rgba(0,0,0,0.05);">
                    <div style="background-color: #1e40af; padding: 20px; text-align: center;">
                        <h2 style="color: #ffffff; margin: 0; font-size: 20px;">PT. Cakra Media Data</h2>
                    </div>
                    <div style="padding: 30px;">
                        <p style="font-size: 16px;">Halo, <strong>%s</strong>,</p>
                        <p style="font-size: 15px; line-height: 1.6; color: #475569;">
                            Proses penilaian kinerja (KPI) Anda untuk <strong>%s</strong> telah selesai dievaluasi oleh <strong>%s</strong>.
                        </p>
                        <div style="background-color: #f8fafc; border-left: 4px solid #3b82f6; padding: 15px; margin: 20px 0;">
                            <p style="margin: 0; font-size: 14px; color: #334155;">
                                Data rapor kinerja, nilai akhir, serta catatan <i>feedback</i> dari manajer Anda sudah dapat diakses melalui portal sistem internal perusahaan.
                            </p>
                        </div>
                        <div style="text-align: center; margin-top: 30px;">
                            <a href="%s" style="background-color: #2563eb; color: #ffffff; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold; font-size: 14px; display: inline-block;">Lihat Rapor Kinerja</a>
                        </div>
                    </div>
                    <div style="background-color: #f1f5f9; padding: 15px; text-align: center; border-top: 1px solid #e2e8f0;">
                        <p style="font-size: 12px; color: #64748b; margin: 0;">Pesan ini dihasilkan otomatis oleh Sistem KPI. Harap tidak membalas email ini.</p>
                    </div>
                </div>
            `, targetEmployee.Name, periodName, evaluatorName, appLink)

            // Panggil fungsi pembantu kita
            helper.SendEmailNotification(targetEmployee.Email, subject, htmlBody)
        }
    }()
    // =================================================================

    // --- [AUDIT TRAIL] ---
    if idUint, ok := managerUserID.(uint); ok {
        helper.LogActivity(mc.DB, idUint, "SUBMIT_EVALUATION", "Menilai pegawai: "+targetEmployee.Name, c.ClientIP())
    }

    Response(c, http.StatusOK, "Evaluasi berhasil disubmit", evaluation)
}
// CheckAndGenerateAutomaticSP bertugas memeriksa riwayat nilai dan menerbitkan SP 1, 2, atau 3
func CheckAndGenerateAutomaticSP(tx *gorm.DB, employeeID uint, currentScore float64, evaluatorEmployeeID uint) error {
	// 1. Batasan Kinerja Buruk: Jika skor >= 2.50 (Grade A, B, C), maka pegawai AMAN.
	if currentScore >= 2.50 {
		return nil
	}

	// 2. Hitung berapa kali pegawai ini mendapat nilai buruk secara berturut-turut
	// Kita akan mengambil riwayat evaluasi terakhir yang sudah berstatus 'submitted' sebelum evaluasi saat ini
	var previousEvaluations []models.Evaluation
	// PERBAIKAN: Gunakan submitted_at IS NOT NULL untuk exclude evaluasi yang baru saja dibuat
	// dalam transaksi ini (yang submitted_at-nya baru di-set tapi belum commit).
	// Limit(2) untuk mengambil maksimal 2 periode ke belakang.
	err := tx.Where("employee_id = ? AND status = ? AND submitted_at IS NOT NULL", employeeID, "submitted").
		Order("submitted_at DESC").
		Limit(2).
		Find(&previousEvaluations).Error

	if err != nil {
		return err
	}

	// 3. Tentukan Tingkat SP berdasarkan konsistensi nilai buruknya
	spLevel := "SP1" // Defaultnya jika ini pelanggaran pertama kali
	reasonMessage := fmt.Sprintf("Surat Peringatan 1 diterbitkan otomatis oleh sistem karena total skor evaluasi Anda pada periode ini berada di bawah standar (Skor: %.2f).", currentScore)

	// Cek kondisi berturut-turut
	if len(previousEvaluations) >= 1 && previousEvaluations[0].TotalScore < 2.50 {
		// Jika periode lalu JUGA buruk, naik pangkat jadi SP2
		spLevel = "SP2"
		reasonMessage = fmt.Sprintf("Surat Peringatan 2 diterbitkan otomatis oleh sistem karena Anda mendapatkan skor di bawah standar selama dua periode berturut-turut (Skor Periode Ini: %.2f).", currentScore)

		if len(previousEvaluations) == 2 && previousEvaluations[1].TotalScore < 2.50 {
			// Jika 2 periode lalu JUGA buruk (Total 3 periode hancur berturut-turut), naik jadi SP3
			spLevel = "SP3"
			reasonMessage = fmt.Sprintf("Surat Peringatan TERAKHIR (SP3) diterbitkan otomatis oleh sistem karena Anda mendapatkan skor di bawah standar selama tiga periode berturut-turut (Skor Periode Ini: %.2f). Silakan hubungi HRD untuk evaluasi kelanjutan kontrak.", currentScore)
		}
	}

	// 4. Masukkan data SP baru ke dalam tabel warnings sesuai struct models.Warning Anda
	newWarning := models.Warning{
		EmployeeID:  employeeID,
		IssuedByID:  evaluatorEmployeeID, // ID Manajer yang mensubmit nilai
		Level:       spLevel,
		Reason:      reasonMessage,
		Description: "Diterbitkan otomatis oleh Sistem KPI Penilaian Kinerja.",
		IssuedAt:    time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := tx.Create(&newWarning).Error; err != nil {
		return err
	}

	// 5. Ambil data pegawai untuk kebutuhan notifikasi
	var targetEmployee models.Employee
	if err := tx.First(&targetEmployee, employeeID).Error; err == nil {
		// Kirim Notifikasi In-App ke Pegawai
		var targetUser models.User
		tx.Where("employee_id = ?", employeeID).First(&targetUser)
		if targetUser.ID != 0 {
			notif := models.Notification{
				UserID:  targetUser.ID,
				Title:   "Peringatan Baru: " + newWarning.Level,
				Message: "Anda mendapatkan Surat Peringatan (" + newWarning.Level + "). Silakan periksa di menu Riwayat SP.",
				Type:    "warning",
			}
			_ = tx.Create(&notif).Error
		}

		// Kirim Notifikasi Email Secara Asinkron
		if targetEmployee.Email != "" {
			go func(email, name, level, reason, desc string) {
				_ = helper.SendWarningEmail(email, name, level, reason, desc)
			}(targetEmployee.Email, targetEmployee.Name, newWarning.Level, newWarning.Reason, newWarning.Description)
		}
	}

	return nil
}

// ResolveAppeal: Manajer menolak komplain dan mengembalikan status menjadi final (submitted)
// @Route: PUT /api/manager/evaluations/:id/resolve-appeal
func (mc *ManagerController) ResolveAppeal(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid", err.Error())
		return
	}

	var evaluation models.Evaluation
	if err := mc.DB.First(&evaluation, id).Error; err != nil {
		Response(c, http.StatusNotFound, "Evaluasi tidak ditemukan", nil)
		return
	}

	// Cek apakah ini benar-benar evaluasi milik manajer yang sedang login
	managerUserID, _ := c.Get("userID")
	var managerUser models.User
	mc.DB.First(&managerUser, managerUserID)
	if evaluation.EvaluatorID != managerUser.EmployeeID {
		Response(c, http.StatusForbidden, "Anda tidak memiliki akses untuk meresolusi evaluasi ini", nil)
		return
	}

	// Jika ditolak (rejected): kembalikan status ke "submitted" agar form terkunci (Read Only).
	// Jika diterima (approved): set ke "appealed" agar form penilaian Manajer terbuka untuk revisi,
	// tanpa memberikan akses edit kepada Pegawai (yang hanya bisa edit saat status = "draft").
	if req.Status == "rejected" {
		evaluation.Status = db_var.EvaluationStatusSubmitted
	} else if req.Status == "approved" {
		evaluation.Status = "appealed" // Status khusus untuk mode revisi oleh Manajer
	}

	if err := mc.DB.Save(&evaluation).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal memperbarui status sanggahan", nil)
		return
	}

	// --- LOG ACTIVITY ---
	if idUint, ok := managerUserID.(uint); ok {
        actionMsg := "Menolak sanggahan"
        if req.Status == "approved" {
            actionMsg = "Menerima sanggahan"
        }
		helper.LogActivity(mc.DB, idUint, "RESOLVE_APPEAL", actionMsg+" untuk evaluasi ID: "+id, c.ClientIP())
	}

    // --- BUAT NOTIFIKASI IN-APP KE PEGAWAI ---
    var employeeUser models.User
    mc.DB.Where("employee_id = ?", evaluation.EmployeeID).First(&employeeUser)

    if employeeUser.ID != 0 {
        statusMsg := "Sanggahan Anda telah ditolak. Nilai ditetapkan secara permanen."
        if req.Status == "approved" {
            statusMsg = "Sanggahan Anda diterima. Nilai akan direvisi oleh manajer."
        }
        notif := models.Notification{
            UserID:  employeeUser.ID,
            Title:   "Hasil Sanggahan",
            Message: statusMsg,
            Type:    "appeal",
        }
        mc.DB.Create(&notif)
    }

	Response(c, http.StatusOK, "Sanggahan berhasil diresolusi", nil)
}