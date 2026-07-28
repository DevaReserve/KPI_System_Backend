package controllers

import (
	"KPI_System_Backend/db_var"
    "KPI_System_Backend/helper" // <--- Import helper
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
	ManagerID   *uint  `json:"manager_id"`
}

// --- CRUD Functions ---

// CreateDivision: Membuat divisi baru
func (dc *DivisionController) CreateDivision(c *gin.Context) {
	var req DivisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid", nil)
		return
	}

	division := models.Division{
		Name:        req.Name,
		Description: req.Description,
	}

	if req.ManagerID != nil {
		division.ManagerID = *req.ManagerID
	}

	if err := dc.DB.Create(&division).Error; err != nil {
		Response(c, http.StatusConflict, "Gagal membuat divisi, nama mungkin sudah ada", nil)
		return
	}

    // --- [AUDIT TRAIL] ---
    actorID, _ := c.Get("userID")
    if idUint, ok := actorID.(uint); ok {
	    helper.LogActivity(dc.DB, idUint, "CREATE_DIVISION", "Membuat divisi baru: "+division.Name, c.ClientIP())
    }

	Response(c, http.StatusCreated, "Divisi berhasil dibuat", division)
}

// GetAllDivisions: Mendapatkan semua divisi
// [ANTI-SPAM]: Read Only -> No Log
func (dc *DivisionController) GetAllDivisions(c *gin.Context) {
	var divisions []models.Division
	
	if err := dc.DB.Find(&divisions).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil data divisi", nil)
		return
	}

	Response(c, http.StatusOK, "Data semua divisi berhasil diambil", divisions)
}

// GetDivision: Mendapatkan satu divisi berdasarkan ID
// [ANTI-SPAM]: Read Only -> No Log
func (dc *DivisionController) GetDivision(c *gin.Context) {
	id := c.Param("id")

	var division models.Division
	if err := dc.DB.First(&division, id).Error; err != nil {
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
func (dc *DivisionController) UpdateDivision(c *gin.Context) {
	id := c.Param("id")

	var division models.Division
	if err := dc.DB.First(&division, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			Response(c, http.StatusNotFound, "Divisi tidak ditemukan", nil)
			return
		}
		Response(c, http.StatusInternalServerError, "Gagal mengambil data divisi", nil)
		return
	}

	var req DivisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid", nil)
		return
	}

	division.Name = req.Name
	division.Description = req.Description
	
	if req.ManagerID != nil {
		division.ManagerID = *req.ManagerID
	} else {
		division.ManagerID = 0 
	}

	if err := dc.DB.Save(&division).Error; err != nil {
		Response(c, http.StatusConflict, "Gagal memperbarui divisi, nama mungkin sudah ada", nil)
		return
	}

    // --- [AUDIT TRAIL] ---
    actorID, _ := c.Get("userID")
    if idUint, ok := actorID.(uint); ok {
	    helper.LogActivity(dc.DB, idUint, "UPDATE_DIVISION", "Mengupdate divisi: "+division.Name, c.ClientIP())
    }

	Response(c, http.StatusOK, "Divisi berhasil diperbarui", division)
}

// DeleteDivision: Menghapus divisi
func (dc *DivisionController) DeleteDivision(c *gin.Context) {
	id := c.Param("id")

    // Ambil data dulu untuk log
    var division models.Division
    dc.DB.First(&division, id)

	var employeeCount int64
	if err := dc.DB.Model(&models.Employee{}).Where("division_id = ?", id).Count(&employeeCount).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal memverifikasi pegawai", nil)
		return
	}

	if employeeCount > 0 {
		Response(c, http.StatusBadRequest, db_var.ErrDivisionHasEmployees, nil)
		return
	}

	if err := dc.DB.Delete(&models.Division{}, id).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menghapus divisi", nil)
		return
	}

    // --- [AUDIT TRAIL] ---
    actorID, _ := c.Get("userID")
    if idUint, ok := actorID.(uint); ok {
        // Catat nama divisi yang dihapus (menggunakan data yang diambil sebelum delete)
        desc := "Menghapus divisi ID " + id
        if division.Name != "" {
            desc = "Menghapus divisi: " + division.Name
        }
	    helper.LogActivity(dc.DB, idUint, "DELETE_DIVISION", desc, c.ClientIP())
    }

	Response(c, http.StatusOK, "Divisi berhasil dihapus", nil)
}