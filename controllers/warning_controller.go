package controllers

import (
	"KPI_System_Backend/helper"
	"KPI_System_Backend/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type WarningController struct {
	DB *gorm.DB
}

func NewWarningController(db *gorm.DB) *WarningController {
	return &WarningController{DB: db}
}

type CreateWarningRequest struct {
	EmployeeID  uint   `json:"employee_id" binding:"required"`
	Level       string `json:"level" binding:"required"` // SP1, SP2, SP3
	Reason      string `json:"reason" binding:"required"`
	Description string `json:"description"`
}

// CreateWarning: Menerbitkan SP Baru (Manager/Admin)
func (wc *WarningController) CreateWarning(c *gin.Context) {
	var req CreateWarningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Data tidak lengkap", err.Error())
		return
	}

	// 1. Ambil Data Penerbit (Issuer) dari Token
	issuerUserID, exists := c.Get("userID")
	if !exists {
		Response(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	var issuerUser models.User
	if err := wc.DB.First(&issuerUser, issuerUserID).Error; err != nil {
		Response(c, http.StatusNotFound, "User tidak valid", nil)
		return
	}

	// 2. Cek Pegawai yang akan diberi SP
	var targetEmployee models.Employee
	if err := wc.DB.First(&targetEmployee, req.EmployeeID).Error; err != nil {
		Response(c, http.StatusNotFound, "Pegawai target tidak ditemukan", nil)
		return
	}

	// 3. Validasi Level SP yang valid
	if req.Level != "SP1" && req.Level != "SP2" && req.Level != "SP3" {
		Response(c, http.StatusBadRequest, "Level SP tidak valid. Gunakan SP1, SP2, atau SP3", nil)
		return
	}

	// 4. ===== VALIDASI HAK AKSES BERDASARKAN ROLE =====
	// SP1 dan SP2 hanya boleh diterbitkan oleh Manager
	if (req.Level == "SP1" || req.Level == "SP2") && issuerUser.Role != "manager" {
		Response(c, http.StatusForbidden, "Hanya Manajer yang berhak menerbitkan SP1 dan SP2", nil)
		return
	}
	// SP3 hanya boleh diterbitkan oleh Admin/HRD
	if req.Level == "SP3" && issuerUser.Role != "admin" {
		Response(c, http.StatusForbidden, "Hanya Admin/HRD yang berhak menerbitkan SP3", nil)
		return
	}
	// ===== END VALIDASI HAK AKSES =====

	// 5. ===== VALIDASI ALUR BERTAHAP (TIDAK BOLEH LOMPAT JENJANG) =====
	sixMonthsAgo := time.Now().AddDate(0, -6, 0)

	if req.Level == "SP2" {
		// Cek apakah ada SP1 aktif (< 6 bulan) untuk pegawai ini
		var sp1Count int64
		wc.DB.Model(&models.Warning{}).
			Where("employee_id = ? AND level = ? AND issued_at >= ?", req.EmployeeID, "SP1", sixMonthsAgo).
			Count(&sp1Count)
		if sp1Count == 0 {
			Response(c, http.StatusUnprocessableEntity, "Pegawai harus mendapatkan SP 1 terlebih dahulu sebelum dapat diberikan SP 2", nil)
			return
		}
	}

	if req.Level == "SP3" {
		// Cek apakah ada SP2 aktif (< 6 bulan) untuk pegawai ini
		var sp2Count int64
		wc.DB.Model(&models.Warning{}).
			Where("employee_id = ? AND level = ? AND issued_at >= ?", req.EmployeeID, "SP2", sixMonthsAgo).
			Count(&sp2Count)
		if sp2Count == 0 {
			Response(c, http.StatusUnprocessableEntity, "Pegawai harus mendapatkan SP 2 terlebih dahulu sebelum dapat diberikan SP 3", nil)
			return
		}
	}
	// ===== END VALIDASI ALUR BERTAHAP =====

	// 6. Simpan SP
	warning := models.Warning{
		EmployeeID:  req.EmployeeID,
		IssuedByID:  issuerUser.EmployeeID, // ID Pegawai si Admin/Manager
		Level:       req.Level,
		Reason:      req.Reason,
		Description: req.Description,
		IssuedAt:    time.Now(),
	}

	if err := wc.DB.Create(&warning).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menerbitkan peringatan", nil)
		return
	}

	// 7. Kirim Notifikasi In-App ke Pegawai
	var targetUser models.User
	wc.DB.Where("employee_id = ?", targetEmployee.ID).First(&targetUser)
	if targetUser.ID != 0 {
		notif := models.Notification{
			UserID:  targetUser.ID,
			Title:   "Peringatan Baru: " + warning.Level,
			Message: "Anda mendapatkan Surat Peringatan (" + warning.Level + "). Silakan periksa di menu Riwayat SP.",
			Type:    "warning",
		}
		wc.DB.Create(&notif)
	}

	// 8. Kirim Notifikasi Email Secara Asinkron
	if targetEmployee.Email != "" {
		go func(email, name, level, reason, desc string) {
			_ = helper.SendWarningEmail(email, name, level, reason, desc)
		}(targetEmployee.Email, targetEmployee.Name, warning.Level, warning.Reason, warning.Description)
	}

	// --- [AUDIT TRAIL] ---
	// Mencatat tindakan pendisiplinan (PENTING)
	if idUint, ok := issuerUserID.(uint); ok {
		helper.LogActivity(wc.DB, idUint, "ISSUE_WARNING", "Menerbitkan "+req.Level+" untuk: "+targetEmployee.Name, c.ClientIP())
	}

	Response(c, http.StatusCreated, "Surat Peringatan berhasil diterbitkan", warning)
}

// GetEmployeeWarnings: Admin/Manager melihat riwayat SP seorang pegawai
// @Route: GET /api/admin/employees/:id/warnings
func (wc *WarningController) GetEmployeeWarnings(c *gin.Context) {
	employeeID := c.Param("id")

	var warnings []models.Warning
	if err := wc.DB.Preload("IssuedBy").Where("employee_id = ?", employeeID).Order("issued_at desc").Find(&warnings).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil data peringatan", nil)
		return
	}

	Response(c, http.StatusOK, "Riwayat peringatan berhasil diambil", warnings)
}

// GetMyWarnings: Pegawai melihat SP miliknya sendiri
// @Route: GET /api/employee/warnings
func (wc *WarningController) GetMyWarnings(c *gin.Context) {
	userID, _ := c.Get("userID")

	var user models.User
	wc.DB.First(&user, userID)

	var warnings []models.Warning
	// Preload IssuedBy agar pegawai tau siapa yang memberi SP
	if err := wc.DB.Preload("IssuedBy").Where("employee_id = ?", user.EmployeeID).Order("issued_at desc").Find(&warnings).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil data", nil)
		return
	}

	Response(c, http.StatusOK, "Riwayat peringatan Anda berhasil diambil", warnings)
}

// DeleteWarning: Menghapus SP (Jika salah input) - Admin Only
func (wc *WarningController) DeleteWarning(c *gin.Context) {
	id := c.Param("id")

	var warning models.Warning
	if err := wc.DB.Preload("Employee").First(&warning, id).Error; err != nil {
		Response(c, http.StatusNotFound, "Data tidak ditemukan", nil)
		return
	}

	// ===== VALIDASI: LARANGAN HAPUS JIKA ADA SP DI JENJANG LEBIH TINGGI =====
	// SP1 tidak bisa dihapus jika sudah ada SP2 milik pegawai yang sama
	if warning.Level == "SP1" {
		var sp2Count int64
		wc.DB.Model(&models.Warning{}).
			Where("employee_id = ? AND level = ?", warning.EmployeeID, "SP2").
			Count(&sp2Count)
		if sp2Count > 0 {
			Response(c, http.StatusConflict,
				"SP 1 tidak dapat dihapus karena pegawai ini sudah memiliki riwayat SP 2. Hapus SP 2 terlebih dahulu.",
				nil)
			return
		}
	}
	// SP2 tidak bisa dihapus jika sudah ada SP3 milik pegawai yang sama
	if warning.Level == "SP2" {
		var sp3Count int64
		wc.DB.Model(&models.Warning{}).
			Where("employee_id = ? AND level = ?", warning.EmployeeID, "SP3").
			Count(&sp3Count)
		if sp3Count > 0 {
			Response(c, http.StatusConflict,
				"SP 2 tidak dapat dihapus karena pegawai ini sudah memiliki riwayat SP 3. Hapus SP 3 terlebih dahulu.",
				nil)
			return
		}
	}
	// ===== END VALIDASI LARANGAN HAPUS =====

	if err := wc.DB.Delete(&warning).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menghapus", nil)
		return
	}

	// --- [AUDIT TRAIL] ---
	actorID, _ := c.Get("userID")
	if idUint, ok := actorID.(uint); ok {
		helper.LogActivity(wc.DB, idUint, "DELETE_WARNING", "Menghapus "+warning.Level+" milik: "+warning.Employee.Name, c.ClientIP())
	}

	Response(c, http.StatusOK, "Peringatan berhasil dihapus", nil)
}

// GetTeamWarnings: Manajer melihat SP yang diberikan kepada timnya
func (wc *WarningController) GetTeamWarnings(c *gin.Context) {
	managerUserID, _ := c.Get("userID")
	var managerUser models.User
	if err := wc.DB.First(&managerUser, managerUserID).Error; err != nil {
		Response(c, http.StatusNotFound, "Data manajer tidak ditemukan", nil)
		return
	}

	levelFilter := c.Query("level")       // "SP1", "SP2", "SP3"
	employeeID  := c.Query("employee_id") // ID Pegawai

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

	query := wc.DB.Table("warnings w").
		Select(`w.id, w.employee_id, 
			emp.name as employee_name, 
			d.name as division_name,
			issuer.name as issued_by_name,
			w.level, w.reason, w.description,
			w.issued_at`).
		Joins("LEFT JOIN employees emp ON emp.id = w.employee_id").
		Joins("LEFT JOIN divisions d ON d.id = emp.division_id").
		Joins("LEFT JOIN employees issuer ON issuer.id = w.issued_by_id").
		Where("emp.direct_supervisor_id = ?", managerUser.EmployeeID).
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

	Response(c, http.StatusOK, "Data SP tim berhasil diambil", results)
}