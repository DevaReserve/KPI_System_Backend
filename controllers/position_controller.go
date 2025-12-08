package controllers

import (
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

func (pc *PositionController) CreatePosition(c *gin.Context) {
	var req PositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Data tidak lengkap", nil)
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
	Response(c, http.StatusCreated, "Jabatan berhasil dibuat", position)
}

func (pc *PositionController) GetAllPositions(c *gin.Context) {
	var positions []models.Position
	if err := pc.DB.Preload("Division").Find(&positions).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil data", nil)
		return
	}
	Response(c, http.StatusOK, "Data jabatan berhasil diambil", positions)
}

func (pc *PositionController) UpdatePosition(c *gin.Context) {
	id := c.Param("id")
	
	// FIX: Deklarasi variabel di luar blok if
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

	position.Name = req.Name
	position.Description = req.Description
	position.DivisionID = req.DivisionID

	if err := pc.DB.Save(&position).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal memperbarui jabatan", nil)
		return
	}
	Response(c, http.StatusOK, "Jabatan berhasil diperbarui", position)
}

func (pc *PositionController) DeletePosition(c *gin.Context) {
	id := c.Param("id")
	if err := pc.DB.Delete(&models.Position{}, id).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menghapus jabatan", nil)
		return
	}
	Response(c, http.StatusOK, "Jabatan berhasil dihapus", nil)
}