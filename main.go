package main

import (
    "KPI_System_Backend/config"
    "KPI_System_Backend/database"
    "KPI_System_Backend/helper"
    "KPI_System_Backend/logger"
    "KPI_System_Backend/middleware"
    "KPI_System_Backend/routes"
    "os"

    "go.uber.org/zap"
    "github.com/gin-gonic/gin"
)

func main() {
    // 1. Inisialisasi logger
    logger.InitLogger()
    defer logger.Sync()

    // 2. Inisialisasi konfigurasi
    config.InitINIConfig()
    config.LoadAppPort()
    config.LoadJWTConfig()
    config.LoadSMTPConfig()
    
    // 3. Inisialisasi database
    db, err := database.InitDB()
    if err != nil {
        logger.Error("Failed to initialize database", zap.Error(err))
        panic("Database connection failed")
    }
    
    if _, err := os.Stat("./uploads/images"); os.IsNotExist(err) {
        os.MkdirAll("./uploads/images", os.ModePerm)
    }
    if _, err := os.Stat("./uploads/documents"); os.IsNotExist(err) {
        os.MkdirAll("./uploads/documents", os.ModePerm)
    }

    // 4. Inisialisasi WhatsApp client (untuk pengiriman OTP)
    // Session disimpan di wa_session.db - jika belum login, QR Code akan tampil di terminal.
    go helper.InitWhatsApp()
    
// 4. Setup Gin
    router := gin.Default()
    router.Use(middleware.CORSMiddleware())

    // 5. Setup routes
    routes.SetupRoutes(router, db)
    
    // 6. Start server
    logger.Info("Starting KPI System Backend", zap.String("port", config.AppPort))
    router.Run(":" + config.AppPort)
}