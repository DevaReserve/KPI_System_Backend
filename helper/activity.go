package helper

import (
	"KPI_System_Backend/logger"
	"KPI_System_Backend/models"
	
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LogActivity: Mencatat aktivitas ke Database & Zap Logger
func LogActivity(db *gorm.DB, userID uint, action string, description string, ipAddress string) {
	// 1. Simpan ke Database (Untuk Audit Trail / History di Frontend)
	logEntry := models.ActivityLog{
		UserID:      userID,
		Action:      action,
		Description: description,
		IpAddress:   ipAddress,
	}

	// Kita gunakan goroutine (go func) agar proses logging tidak memperlambat response API utama
	go func() {
		if err := db.Create(&logEntry).Error; err != nil {
			logger.Error("Gagal menyimpan activity log ke DB", zap.Error(err))
		}
	}()

	// 2. Catat juga ke System Log (Zap) untuk debugging teknis
	logger.Info("Audit Trail", 
		zap.Uint("user_id", userID), 
		zap.String("action", action), 
		zap.String("desc", description),
	)
}