package database

import (
	"KPI_System_Backend/config"
	"KPI_System_Backend/logger"
	"KPI_System_Backend/models" // <--- IMPORT MODELS
	"fmt"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() (*gorm.DB, error) {
	// 1. Ambil konfigurasi dari Setting.ini
	dbConfig := config.GetIniDatabase()

	// 2. Buat Data Source Name (DSN) string
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbConfig.User,
		dbConfig.Password,
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.DatabaseName,
	)

	// 3. Buka koneksi ke database
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		// Ini adalah error yang Anda dapatkan
		logger.Error("Failed to connect to database", zap.Error(err))
		return nil, err
	}

	// 4. --- INI PENAMBAHAN PENTING ---
	// AutoMigrate akan membuat/memperbarui tabel berdasarkan struct di Models
	logger.Info("Running database migrations...")
	err = db.AutoMigrate(
		&models.Division{},
		&models.Employee{},
		&models.User{},
		&models.EvaluationPeriod{},
		&models.PerformanceIndicator{},
		&models.Evaluation{},
		&models.EvaluationScore{},
	)
	if err != nil {
		logger.Error("Failed to run auto-migration", zap.Error(err))
		return nil, err
	}
	logger.Info("Database migration successful")
	// --- AKHIR PENAMBAHAN ---

	// 5. Kembalikan koneksi database yang sudah siap
	return db, nil
}