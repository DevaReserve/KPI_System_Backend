package controllers

import (
	"KPI_System_Backend/helper" // <--- Import Helper
	"KPI_System_Backend/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PositionController struct {
	DB *gorm.DB
}

func NewPositionController(db *gorm.DB) *PositionController {
	return &PositionController{DB: db}
}

type PositionRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	DivisionID  uint   `json:"division_id" binding:"required"`
}

// CreatePosition
func (pc *PositionController) CreatePosition(c *gin.Context) {
	var req PositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Data tidak lengkap", nil)
		return
	}

	// --- Validasi Duplikasi ---
	var count int64
	pc.DB.Model(&models.Position{}).
		Where("LOWER(name) = LOWER(?) AND division_id = ?", req.Name, req.DivisionID).
		Count(&count)
	if count > 0 {
		Response(c, http.StatusConflict, "Jabatan dengan nama tersebut sudah ada di divisi ini", nil)
		return
	}

	position := models.Position{
		Name:        req.Name,
		Description: req.Description,
		DivisionID:  req.DivisionID,
	}

	if err := pc.DB.Create(&position).Error; err != nil {
		Response(c, http.StatusConflict, "Gagal membuat jabatan", nil)
		return
	}

    // --- [AUDIT TRAIL] ---
    actorID, _ := c.Get("userID")
    if idUint, ok := actorID.(uint); ok {
	    helper.LogActivity(pc.DB, idUint, "CREATE_POSITION", "Membuat jabatan baru: "+position.Name, c.ClientIP())
    }

	Response(c, http.StatusCreated, "Jabatan berhasil dibuat", position)
}

// GetAllPositions: Read Only -> No Log
func (pc *PositionController) GetAllPositions(c *gin.Context) {
	var positions []models.Position
	if err := pc.DB.Preload("Division").Find(&positions).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil data", nil)
		return
	}
	Response(c, http.StatusOK, "Data jabatan berhasil diambil", positions)
}

// UpdatePosition
func (pc *PositionController) UpdatePosition(c *gin.Context) {
	id := c.Param("id")
	
	var position models.Position

	if err := pc.DB.First(&position, id).Error; err != nil {
		Response(c, http.StatusNotFound, "Jabatan tidak ditemukan", nil)
		return
	}

	var req PositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid", nil)
		return
	}

	// --- Validasi Duplikasi (kecuali jabatan itu sendiri) ---
	var count int64
	pc.DB.Model(&models.Position{}).
		Where("LOWER(name) = LOWER(?) AND division_id = ? AND id != ?", req.Name, req.DivisionID, position.ID).
		Count(&count)
	if count > 0 {
		Response(c, http.StatusConflict, "Jabatan dengan nama tersebut sudah ada di divisi ini", nil)
		return
	}

	position.Name = req.Name
	position.Description = req.Description
	position.DivisionID = req.DivisionID

	if err := pc.DB.Save(&position).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal memperbarui jabatan", nil)
		return
	}

    // --- [AUDIT TRAIL] ---
    actorID, _ := c.Get("userID")
    if idUint, ok := actorID.(uint); ok {
	    helper.LogActivity(pc.DB, idUint, "UPDATE_POSITION", "Mengupdate jabatan: "+position.Name, c.ClientIP())
    }

	Response(c, http.StatusOK, "Jabatan berhasil diperbarui", position)
}

// DeletePosition
func (pc *PositionController) DeletePosition(c *gin.Context) {
	id := c.Param("id")
    
    // Ambil data sebelum hapus
    var position models.Position
    pc.DB.First(&position, id)

	if err := pc.DB.Delete(&models.Position{}, id).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menghapus jabatan", nil)
		return
	}

    // --- [AUDIT TRAIL] ---
    actorID, _ := c.Get("userID")
    if idUint, ok := actorID.(uint); ok {
        desc := "Menghapus jabatan ID " + id
        if position.Name != "" { desc = "Menghapus jabatan: " + position.Name }
	    helper.LogActivity(pc.DB, idUint, "DELETE_POSITION", desc, c.ClientIP())
    }

	Response(c, http.StatusOK, "Jabatan berhasil dihapus", nil)
}