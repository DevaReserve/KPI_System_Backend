package controllers

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"KPI_System_Backend/models"
)

type AchievementController struct {
	DB *gorm.DB
}

func NewAchievementController(db *gorm.DB) *AchievementController {
	return &AchievementController{DB: db}
}

// CreateAchievement: Upload file + Simpan data
func (ac *AchievementController) CreateAchievement(c *gin.Context) {
	// 1. Ambil User ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Unauthorized"})
		return
	}

	// 2. Cari Employee ID dari User ID
	var user models.User
	if err := ac.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "User not found"})
		return
	}

	// 3. Ambil Data Form (Multipart)
	title := c.PostForm("title")
	description := c.PostForm("description")
	dateStr := c.PostForm("date") // Format YYYY-MM-DD

	// Parsing Tanggal
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Format tanggal salah (Gunakan YYYY-MM-DD)"})
		return
	}

	// 4. Handle File Upload
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "File bukti wajib diupload"})
		return
	}

	// Validasi Ukuran (Max 5MB)
	if file.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "File maksimal 5MB"})
		return
	}

	// Simpan File
	ext := strings.ToLower(filepath.Ext(file.Filename))
	filename := uuid.New().String() + ext
	savePath := "./uploads/documents/" + filename // Pastikan folder ini ada

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menyimpan file"})
		return
	}

	fileURL := "http://localhost:8080/uploads/documents/" + filename

	// 5. Simpan ke Database
	achievement := models.EmployeeAchievement{
		EmployeeID:  user.EmployeeID,
		Title:       title,
		Description: description,
		Date:        date,
		FileURL:     fileURL,
		CreatedAt:   time.Now(),
	}

	if err := ac.DB.Create(&achievement).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menyimpan data ke database"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": achievement})
}

// GetMyAchievements: Lihat list prestasi sendiri
func (ac *AchievementController) GetMyAchievements(c *gin.Context) {
	userID, _ := c.Get("userID")
	var user models.User
	ac.DB.First(&user, userID)

	var achievements []models.EmployeeAchievement
	if err := ac.DB.Where("employee_id = ?", user.EmployeeID).Order("date desc").Find(&achievements).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal mengambil data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": achievements})
}

// DeleteAchievement
func (ac *AchievementController) DeleteAchievement(c *gin.Context) {
	id := c.Param("id")
	if err := ac.DB.Delete(&models.EmployeeAchievement{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menghapus"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Terhapus"})
}

func (ac *AchievementController) GetEmployeeAchievements(c *gin.Context) {
    // PENTING: Ambil ID dari URL (contoh: /admin/employees/5/achievements)
    targetEmployeeID := c.Param("id") 

    var achievements []models.EmployeeAchievement
    
    // Query berdasarkan employee_id yang didapat dari URL
    if err := ac.DB.Where("employee_id = ?", targetEmployeeID).Order("date desc").Find(&achievements).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal mengambil data"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"status": "success", "data": achievements})
}