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

	// 3. Simpan SP
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