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
	// Initialize configuration
	config.InitINIConfig()
	config.LoadAppPort()
	config.LoadJWTConfig()
	
	// Initialize logger
	logger.InitLogger()
	defer logger.Sync()
	
	// Initialize database
	db, err := database.InitDB()
	if err != nil {
		logger.Error("Failed to initialize database", zap.Error(err))
		panic("Database connection failed")
	}
	
	// Setup Gin
	router := gin.Default()
	router.Use(middleware.CORSMiddleware())
	
	// Setup routes
	routes.SetupRoutes(router, db)
	
	// Start server
	logger.Info("Starting KPI System Backend", zap.String("port", config.AppPort))
	router.Run(":" + config.AppPort)
}