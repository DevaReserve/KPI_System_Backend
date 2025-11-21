package main

import (
	"KPI_System_Backend/config"
	"KPI_System_Backend/database"
	"KPI_System_Backend/logger"
	"KPI_System_Backend/middleware"
	"KPI_System_Backend/routes"

	"go.uber.org/zap"
	"github.com/gin-gonic/gin"
)

func main() {
	// --- INI PERBAIKANNYA ---
	// 1. Inisialisasi logger DULU
	logger.InitLogger()
	defer logger.Sync()
	// --- AKHIR PERBAIKAN ---

	// 2. Baru inisialisasi konfigurasi
	// Jika config.InitINIConfig() gagal, logger sudah siap mencatatnya
	config.InitINIConfig()
	config.LoadAppPort()
	config.LoadJWTConfig()
	
	// 3. Inisialisasi database
	db, err := database.InitDB()
	if err != nil {
		logger.Error("Failed to initialize database", zap.Error(err))
		panic("Database connection failed")
	}
	
	// 4. Setup Gin
	router := gin.Default()
	router.Use(middleware.CORSMiddleware())
	
	// 5. Setup routes
	routes.SetupRoutes(router, db)
	
	// 6. Start server
	logger.Info("Starting KPI System Backend", zap.String("port", config.AppPort))
	router.Run(":" + config.AppPort)
}