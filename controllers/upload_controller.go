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

type UploadController struct {
	DB *gorm.DB
}

func NewUploadController(db *gorm.DB) *UploadController {
	return &UploadController{DB: db}
}

func (ctrl *UploadController) UploadProfilePicture(c *gin.Context) {
	userID, exists := c.Get("userID") 
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Unauthorized"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "File tidak ditemukan"})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Hanya file JPG/PNG yang diperbolehkan"})
		return
	}

	if file.Size > 2*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Ukuran file maksimal 2MB"})
		return
	}

	filename := uuid.New().String() + ext
	savePath := "./uploads/images/" + filename

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menyimpan file ke server"})
		return
	}

	var user models.User
	if err := ctrl.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "User tidak ditemukan"})
		return
	}

	if user.EmployeeID == 0 { 
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Akun ini tidak terhubung dengan data pegawai"})
		return
	}

	fileURL := "http://localhost:8080/uploads/images/" + filename 

	if err := ctrl.DB.Model(&models.Employee{}).Where("id = ?", user.EmployeeID).Update("profile_picture_url", fileURL).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal update database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"url": fileURL,
		},
	})
}

func (ac *AchievementController) UpdateAchievement(c *gin.Context) {
	id := c.Param("id")
	
	// 1. Cari data prestasi lama
	var achievement models.EmployeeAchievement
	if err := ac.DB.First(&achievement, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Data tidak ditemukan"})
		return
	}

	// 2. Ambil data teks dari form (Jika ada perubahan)
	title := c.PostForm("title")
	description := c.PostForm("description")
	dateStr := c.PostForm("date")

	if title != "" { achievement.Title = title }
	if description != "" { achievement.Description = description }
	if dateStr != "" {
		date, err := time.Parse("2006-01-02", dateStr)
		if err == nil { achievement.Date = date }
	}

	// 3. Cek apakah user mengupload file baru? (LOGIKA UTAMA)
	file, err := c.FormFile("file")
	
	// Jika tidak ada error (berarti ada file baru), maka proses upload
	if err == nil && file != nil {
		// Validasi Ukuran
		if file.Size > 5*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "File maksimal 5MB"})
			return
		}

		// Simpan File Baru
		ext := strings.ToLower(filepath.Ext(file.Filename))
		filename := uuid.New().String() + ext
		savePath := "./uploads/documents/" + filename

		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menyimpan file baru"})
			return
		}

		// Hapus file lama (Opsional, agar server tidak penuh sampah)
		// os.Remove("." + achievement.FileURL) 

		// Update URL di database
		achievement.FileURL = "http://localhost:8080/uploads/documents/" + filename
	}
	// Jika user tidak upload file baru, achievement.FileURL biarkan tetap yang lama

	// 4. Simpan perubahan ke Database
	if err := ac.DB.Save(&achievement).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal update data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": achievement})
}