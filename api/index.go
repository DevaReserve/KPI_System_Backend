package api

import (
	"KPI_System_Backend/config"
	"KPI_System_Backend/database"
	"KPI_System_Backend/helper"
	"KPI_System_Backend/logger"
	"KPI_System_Backend/middleware"
	"KPI_System_Backend/routes"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	App  *gin.Engine
	once sync.Once
)

func init() {
	InitApp()
}

// InitApp menginisialisasi semua dependensi aplikasi (Logger, Config, DB, Router).
// Menggunakan sync.Once agar aman jika dipanggil berkali-kali.
func InitApp() {
	once.Do(func() {
		logger.InitLogger()

		config.InitINIConfig()
		config.LoadAppPort()
		config.LoadJWTConfig()
		config.LoadSMTPConfig()
		config.LoadSupabaseConfig()

		db, err := database.InitDB()
		if err != nil {
			logger.Error("Failed to initialize database", zap.Error(err))
			panic("Database connection failed")
		}

		go helper.InitWhatsApp()

		App = gin.Default()
		App.Use(middleware.CORSMiddleware())
		routes.SetupRoutes(App, db)
	})
}

// Handler is the Vercel Serverless Function entrypoint
func Handler(w http.ResponseWriter, r *http.Request) {
	InitApp() // Ensures initialization
	App.ServeHTTP(w, r)
}
