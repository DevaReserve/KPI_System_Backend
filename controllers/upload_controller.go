package controllers

import (
	"KPI_System_Backend/helper" // <--- Import Helper
	"KPI_System_Backend/models"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"os"
	
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
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

	// Ambil data User dan Pegawai dulu untuk mendapatkan foto lama
	var user models.User
	if err := ctrl.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "User tidak ditemukan"})
		return
	}

	if user.EmployeeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Akun ini tidak terhubung dengan data pegawai"})
		return
	}

	var employee models.Employee
	if err := ctrl.DB.First(&employee, user.EmployeeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Data pegawai tidak ditemukan"})
		return
	}

	// [CLEANUP] Hapus foto lama jika ada
	if employee.ProfilePictureURL != "" {
		// Konversi URL ke path lokal
		// Contoh URL: http://localhost:8080/uploads/images/foto.jpg
		// Path lokal: ./uploads/images/foto.jpg
		oldFilename := filepath.Base(employee.ProfilePictureURL)
		oldPath := "./uploads/images/" + oldFilename
		
		// Hapus file (abaikan error jika file sudah tidak ada)
		_ = os.Remove(oldPath)
	}

	// Simpan File Baru
	filename := uuid.New().String() + ext
	savePath := "./uploads/images/" + filename

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menyimpan file ke server"})
		return
	}

	fileURL := "http://localhost:8080/uploads/images/" + filename

	// Update Database
	if err := ctrl.DB.Model(&models.Employee{}).Where("id = ?", user.EmployeeID).Update("profile_picture_url", fileURL).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal update database"})
		return
	}

	// Audit Trail
	if idUint, ok := userID.(uint); ok {
		helper.LogActivity(ctrl.DB, idUint, "UPDATE_PROFILE_PICTURE", "Mengganti foto profil", c.ClientIP())
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
	
	var achievement models.EmployeeAchievement
	if err := ac.DB.First(&achievement, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Data tidak ditemukan"})
		return
	}

	actorID, _ := c.Get("userID")

	// Update field teks
	title := c.PostForm("title")
	description := c.PostForm("description")
	dateStr := c.PostForm("date")

	if title != "" { achievement.Title = title }
	if description != "" { achievement.Description = description }
	if dateStr != "" {
		date, err := time.Parse("2006-01-02", dateStr)
		if err == nil { achievement.Date = date }
	}

	// Cek upload file baru
	file, err := c.FormFile("file")
	if err == nil && file != nil {
		if file.Size > 5*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "File maksimal 5MB"})
			return
		}

		// [CLEANUP] Hapus file dokumen lama
		if achievement.FileURL != "" {
			oldFilename := filepath.Base(achievement.FileURL)
			oldPath := "./uploads/documents/" + oldFilename
			_ = os.Remove(oldPath) // Hapus file lama
		}

		// Simpan file baru
		ext := strings.ToLower(filepath.Ext(file.Filename))
		filename := uuid.New().String() + ext
		savePath := "./uploads/documents/" + filename

		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menyimpan file baru"})
			return
		}
		
		// Update URL di struct
		achievement.FileURL = "http://localhost:8080/uploads/documents/" + filename
	}

	// Simpan ke DB
	if err := ac.DB.Save(&achievement).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal update data"})
		return
	}

	// Audit Trail
	if idUint, ok := actorID.(uint); ok {
		helper.LogActivity(ac.DB, idUint, "UPDATE_ACHIEVEMENT", "Mengupdate data prestasi: "+achievement.Title, c.ClientIP())
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": achievement})
}