package controllers

import (
	"KPI_System_Backend/db_var"
	"KPI_System_Backend/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DivisionController struct {
	DB *gorm.DB
}

// NewDivisionController adalah "constructor" untuk controller
func NewDivisionController(db *gorm.DB) *DivisionController {
	return &DivisionController{DB: db}
}

// --- Struct untuk Request Binding ---

type DivisionRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	// Kita gunakan pointer (*uint) agar bisa menerima 'null' atau 0
	// Ini PENTING untuk mengatasi masalah "ayam & telur"
	// Kita bisa buat divisi dulu, baru tunjuk manajernya nanti (saat update)
	ManagerID *uint `json:"manager_id"`
}

// --- CRUD Functions ---

// CreateDivision: Membuat divisi baru
// @Route: POST /api/admin/divisions
func (dc *DivisionController) CreateDivision(c *gin.Context) {
	var req DivisionRequest
	// Bind JSON ke struct request
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid", nil)
		return
	}

	// Buat objek model
	division := models.Division{
		Name:        req.Name,
		Description: req.Description,
	}

	// Hanya set ManagerID jika nilainya diberikan (tidak null)
	if req.ManagerID != nil {
		division.ManagerID = *req.ManagerID
	}

	// Simpan ke database
	if err := dc.DB.Create(&division).Error; err != nil {
		// Handle error jika nama divisi sudah ada (unique constraint)
		Response(c, http.StatusConflict, "Gagal membuat divisi, nama mungkin sudah ada", nil)
		return
	}

	Response(c, http.StatusCreated, "Divisi berhasil dibuat", division)
}

// GetAllDivisions: Mendapatkan semua divisi
// @Route: GET /api/admin/divisions
func (dc *DivisionController) GetAllDivisions(c *gin.Context) {
	var divisions []models.Division
	
	// Ambil semua data dari tabel divisions
	if err := dc.DB.Find(&divisions).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil data divisi", nil)
		return
	}

	Response(c, http.StatusOK, "Data semua divisi berhasil diambil", divisions)
}

// GetDivision: Mendapatkan satu divisi berdasarkan ID
// @Route: GET /api/admin/divisions/:id
func (dc *DivisionController) GetDivision(c *gin.Context) {
	// Ambil ID dari URL parameter
	id := c.Param("id")

	var division models.Division
	// Cari divisi berdasarkan ID
	if err := dc.DB.First(&division, id).Error; err != nil {
		// Handle jika data tidak ditemukan
		if err == gorm.ErrRecordNotFound {
			Response(c, http.StatusNotFound, "Divisi tidak ditemukan", nil)
			return
		}
		Response(c, http.StatusInternalServerError, "Gagal mengambil data divisi", nil)
		return
	}

	Response(c, http.StatusOK, "Data divisi berhasil diambil", division)
}

// UpdateDivision: Memperbarui divisi
// @Route: PUT /api/admin/divisions/:id
func (dc *DivisionController) UpdateDivision(c *gin.Context) {
	// Ambil ID dari URL
	id := c.Param("id")

	var division models.Division
	// Cek apakah divisi ada
	if err := dc.DB.First(&division, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			Response(c, http.StatusNotFound, "Divisi tidak ditemukan", nil)
			return
		}
		Response(c, http.StatusInternalServerError, "Gagal mengambil data divisi", nil)
		return
	}

	// Bind JSON request ke struct
	var req DivisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid", nil)
		return
	}

	// Update data
	division.Name = req.Name
	division.Description = req.Description
	
	if req.ManagerID != nil {
		division.ManagerID = *req.ManagerID
	} else {
		// Jika request-nya null, kita set jadi 0 (atau null di DB)
		division.ManagerID = 0 
	}

	// Simpan perubahan
	if err := dc.DB.Save(&division).Error; err != nil {
		Response(c, http.StatusConflict, "Gagal memperbarui divisi, nama mungkin sudah ada", nil)
		return
	}

	Response(c, http.StatusOK, "Divisi berhasil diperbarui", division)
}

// DeleteDivision: Menghapus divisi
// @Route: DELETE /api/admin/divisions/:id
func (dc *DivisionController) DeleteDivision(c *gin.Context) {
	// Ambil ID dari URL
	id := c.Param("id")

	// PENTING! Cek Keamanan: Jangan hapus divisi jika masih ada pegawai di dalamnya.
	var employeeCount int64
	if err := dc.DB.Model(&models.Employee{}).Where("division_id = ?", id).Count(&employeeCount).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal memverifikasi pegawai", nil)
		return
	}

	if employeeCount > 0 {
		// Gunakan pesan error dari db_var yang sudah Anda buat
		Response(c, http.StatusBadRequest, db_var.ErrDivisionHasEmployees, nil)
		return
	}

	// Jika aman (tidak ada pegawai), lanjutkan proses hapus
	if err := dc.DB.Delete(&models.Division{}, id).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menghapus divisi", nil)
		return
	}

	Response(c, http.StatusOK, "Divisi berhasil dihapus", nil)
}