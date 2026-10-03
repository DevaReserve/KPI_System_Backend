package helper

import (
	"KPI_System_Backend/logger"
	"KPI_System_Backend/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LogActivity: Mencatat aktivitas ke Database & Zap Logger
func LogActivity(db *gorm.DB, userID uint, action string, description string, ipAddress string) {
	LogActivityWithUsername(db, userID, "", action, description, ipAddress)
}

func LogActivityWithUsername(db *gorm.DB, userID uint, username string, action string, description string, ipAddress string) {
	// 1. Simpan ke Database (Untuk Audit Trail / History di Frontend)
	logEntry := models.ActivityLog{
		Action:      action,
		Description: description,
		IpAddress:   ipAddress,
		Username:    username,
	}
	if userID > 0 {
		id := userID
		logEntry.UserID = &id
	}

	// Kita gunakan goroutine (go func) agar proses logging tidak memperlambat response API utama
	go func(entry models.ActivityLog) {
		if err := db.Create(&entry).Error; err != nil {
			logger.Error("Gagal menyimpan activity log ke DB", zap.Error(err))
		}
	}(logEntry)

	// 2. Catat juga ke System Log (Zap) untuk debugging teknis
	logger.Info("Audit Trail",
		zap.Uint("user_id", userID),
		zap.String("action", action),
		zap.String("desc", description),
	)
}
