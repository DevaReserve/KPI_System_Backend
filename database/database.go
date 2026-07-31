package database

import (
	"KPI_System_Backend/config"
	"KPI_System_Backend/helper"
	"KPI_System_Backend/logger"
	"KPI_System_Backend/models"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB() (*gorm.DB, error) {
	// 1. Ambil konfigurasi dari Setting.ini
	dbConfig := config.GetIniDatabase()

	// 2. Pilih driver dan buat DSN berdasarkan setting "Driver" di Setting.ini
	driver := strings.ToLower(strings.TrimSpace(dbConfig.Driver))

	var db *gorm.DB
	var err error

	switch driver {
	case "postgres":
		// Format DSN untuk PostgreSQL (Supabase / lokal)
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=require TimeZone=Asia/Jakarta",
			dbConfig.Host,
			dbConfig.User,
			dbConfig.Password,
			dbConfig.DatabaseName,
			dbConfig.Port,
		)
		logger.Info("Connecting to database using PostgreSQL driver...",
			zap.String("host", dbConfig.Host),
			zap.String("port", dbConfig.Port),
		)
		db, err = gorm.Open(postgres.New(postgres.Config{
			DSN:                  dsn,
			PreferSimpleProtocol: true, // Menonaktifkan implicit prepared statements (wajib untuk Supabase / pgBouncer)
		}), &gorm.Config{})

	default: // "mysql" atau tidak diisi
		// Format DSN untuk MySQL
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			dbConfig.User,
			dbConfig.Password,
			dbConfig.Host,
			dbConfig.Port,
			dbConfig.DatabaseName,
		)
		logger.Info("Connecting to database using MySQL driver...",
			zap.String("host", dbConfig.Host),
			zap.String("port", dbConfig.Port),
		)
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	}
	if err != nil {
		logger.Error("Failed to connect to database", zap.Error(err))
		return nil, err
	}

	// 4. Jalankan Auto Migrate
	// Ini akan membuat tabel baru atau mengupdate kolom yang kurang
	logger.Info("Running database migrations...")
	err = db.AutoMigrate(
		&models.Division{},
		&models.Position{},
		&models.Employee{},
		&models.User{},
		&models.EvaluationPeriod{},
		&models.PerformanceIndicator{},
		&models.Evaluation{},
		&models.EvaluationScore{},
		&models.EmployeeAchievement{},
		&models.ActivityLog{},
		&models.Warning{},
		&models.KPITarget{}, // [BARU] Tabel target KPI
		&models.Notification{},
		&models.PasswordReset{}, // [BARU] Tabel reset password OTP
	)

	if err != nil {
		logger.Error("Failed to run auto-migration", zap.Error(err))
		return nil, err
	}
	logger.Info("Database migration successful")

	// 5. Jalankan seeder untuk data awal (superadmin)
	if err := SeedSuperAdmin(db); err != nil {
		logger.Error("Failed to seed superadmin", zap.Error(err))
		// Tidak fatal, lanjutkan saja
	}

	// 6. Kembalikan koneksi database yang sudah siap
	return db, nil
}

// SeedSuperAdmin membuat akun superadmin default jika belum ada di database.
// Akun ini hanya dibuat satu kali saat database baru pertama kali digunakan.
func SeedSuperAdmin(db *gorm.DB) error {
	const superadminUsername = "superadmin"

	// Cek apakah superadmin sudah ada
	var count int64
	db.Model(&models.User{}).Where("username = ?", superadminUsername).Count(&count)
	if count > 0 {
		logger.Info("Superadmin already exists, skipping seed")
		return nil
	}

	logger.Info("Seeding superadmin account...")

	// Buat divisi default "Management" jika belum ada
	var division models.Division
	result := db.Where("name = ?", "Board of Directors").First(&division)
	if result.Error != nil {
		division = models.Division{
			Name:        "Board of Directors",
			Description: "Divisi manajemen utama",
		}
		if err := db.Create(&division).Error; err != nil {
			return fmt.Errorf("failed to create default division: %w", err)
		}
		logger.Info("Default division 'Board of Directors' created")
	}

	// Buat data Employee untuk superadmin
	joinDate, _ := time.Parse("2006-01-02", "2024-01-01")
	employee := models.Employee{
		NIP:        "SA-0001",
		Name:       "Super Admin",
		Email:      "superadmin@kpisystem.local",
		DivisionID: division.ID,
		Position:   "System Administrator",
		IsActive:   true,
		JoinDate:   joinDate,
	}
	if err := db.Create(&employee).Error; err != nil {
		return fmt.Errorf("failed to create superadmin employee: %w", err)
	}
	logger.Info("Superadmin employee record created", zap.Uint("employee_id", employee.ID))

	// Hash password default menggunakan helper yang sudah ada
	const defaultPassword = "Admin@1234"
	hashedPassword, err := helper.HashPassword(defaultPassword)
	if err != nil {
		return fmt.Errorf("failed to hash superadmin password: %w", err)
	}

	// Buat akun User superadmin
	user := models.User{
		EmployeeID:   employee.ID,
		Username:     superadminUsername,
		PasswordHash: hashedPassword,
		Role:         "admin",
		IsExecutive:  true,
		IsActive:     true,
	}
	if err := db.Create(&user).Error; err != nil {
		return fmt.Errorf("failed to create superadmin user: %w", err)
	}

	logger.Info("✅ Superadmin seeded successfully",
		zap.String("username", superadminUsername),
		zap.String("password", defaultPassword),
		zap.String("role", "admin"),
	)

	return nil
}