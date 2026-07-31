package main

import (
	"KPI_System_Backend/api"
	"KPI_System_Backend/config"
	"KPI_System_Backend/logger"
	"os"

	"go.uber.org/zap"
)

func main() {
	// Sync logger at exit
	defer logger.Sync()

	// Pastikan folder lokal tetap terbuat jika masih running local,
	// namun upload akan tetap diarahkan ke Supabase (jika dikonfigurasi).
	if _, err := os.Stat("./uploads/images"); os.IsNotExist(err) {
		os.MkdirAll("./uploads/images", os.ModePerm)
	}
	if _, err := os.Stat("./uploads/documents"); os.IsNotExist(err) {
		os.MkdirAll("./uploads/documents", os.ModePerm)
	}

	logger.Info("Starting KPI System Backend (Local Mode)", zap.String("port", config.AppPort))
	
	// Jalankan aplikasi menggunakan instance App yang telah diinisialisasi secara otomatis
	// oleh package "api" saat startup.
	if err := api.App.Run(config.AppPort); err != nil {
		logger.Error("Server failed to start", zap.Error(err))
	}
}