package api

import (
	"KPI_System_Backend/config"
	"KPI_System_Backend/database"
	"KPI_System_Backend/helper"
	"KPI_System_Backend/logger"
	"KPI_System_Backend/middleware"
	"KPI_System_Backend/routes"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	App     *gin.Engine
	once    sync.Once
	initErr error
)

func init() {
	InitApp()
}

// InitApp menginisialisasi semua dependensi aplikasi (Logger, Config, DB, Router).
// Menggunakan sync.Once agar aman jika dipanggil berkali-kali.
// Tidak menggunakan panic agar Vercel serverless tidak crash saat cold-start gagal.
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
			// Catat error secara detail untuk debugging di Vercel logs
			logger.Error("CRITICAL: Failed to initialize database. Server will return 503 for all requests.",
				zap.Error(err),
				zap.String("hint", "Pastikan semua env vars DB sudah di-set di Vercel Dashboard: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME"),
			)
			initErr = err
			// Buat App tetap berjalan agar bisa return 503 (bukan crash 500)
			App = gin.Default()
			App.Use(middleware.CORSMiddleware())
			App.NoRoute(func(c *gin.Context) {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status":  "error",
					"message": "Server sedang tidak dapat terhubung ke database. Hubungi administrator.",
				})
			})
			return
		}

		// Hanya aktifkan WhatsApp jika env var ENABLE_WHATSAPP=true di-set secara eksplisit.
		// Fitur ini tidak compatible dengan Vercel serverless (butuh persistent session file).
		if os.Getenv("ENABLE_WHATSAPP") == "true" {
			go helper.InitWhatsApp()
		} else {
			logger.Info("WhatsApp service dinonaktifkan. Set env var ENABLE_WHATSAPP=true untuk mengaktifkan (hanya untuk server dedicated, bukan Vercel).")
		}

		App = gin.Default()
		App.Use(middleware.CORSMiddleware())
		routes.SetupRoutes(App, db)
		initErr = nil
	})
}

// Handler is the Vercel Serverless Function entrypoint
func Handler(w http.ResponseWriter, r *http.Request) {
	InitApp() // Ensures initialization (safe to call multiple times via sync.Once)
	if App == nil {
		http.Error(w, `{"status":"error","message":"Server tidak dapat diinisialisasi"}`, http.StatusServiceUnavailable)
		return
	}
	App.ServeHTTP(w, r)
}

