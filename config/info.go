package config

import (
	"KPI_System_Backend/global_var"
	"KPI_System_Backend/logger"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"gopkg.in/ini.v1"
)

var (
	IniConfig   *ini.File
	AppPort     string
	FrontEndURL string
	JWTSecret   string
	JWTExpiry   int

	SupabaseURL string
	SupabaseKey string
)

func InitINIConfig() {
	// 1. Load dari .env (jika ada, biasanya untuk lokal)
	err := godotenv.Load()
	if err != nil {
		logger.Warn("No .env file found or failed to read, will rely on system environment variables", zap.Error(err))
	}

	// 2. Fallback baca Setting.ini (backward compatibility)
	cfg, err := ini.Load("Setting.ini")
	if err == nil {
		IniConfig = cfg
	}
}

// Helper untuk membaca urutan: Env -> INI -> Default
func getEnvOrIni(envKey, iniSection, iniKey, defaultVal string) string {
	if val := os.Getenv(envKey); val != "" {
		return val
	}
	if IniConfig != nil {
		if sec := IniConfig.Section(iniSection); sec != nil {
			if key := sec.Key(iniKey); key != nil && key.String() != "" {
				return key.String()
			}
		}
	}
	return defaultVal
}

func GetIniDatabase() global_var.DatabaseConnection {
	return global_var.DatabaseConnection{
		Driver:       getEnvOrIni("DB_DRIVER", "MainDatabase", "Driver", "postgres"),
		Host:         getEnvOrIni("DB_HOST", "MainDatabase", "Host Name", "localhost"),
		Port:         getEnvOrIni("DB_PORT", "MainDatabase", "Port", "6543"),
		User:         getEnvOrIni("DB_USER", "MainDatabase", "User Name", "postgres"),
		Password:     getEnvOrIni("DB_PASSWORD", "MainDatabase", "Password", ""),
		DatabaseName: getEnvOrIni("DB_NAME", "MainDatabase", "Database Name", "postgres"),
		CreateDBTest: false, // Disetel false untuk default (bisa diatur lewat env jika diperlukan nanti)
	}
}

func LoadAppPort() {
	AppPort = getEnvOrIni("APP_PORT", "GlobalConfig", "AppPort", "8080")
	// Jika dari env tidak ada prefix ":", tambahkan
	if AppPort != "" && AppPort[0] != ':' {
		AppPort = ":" + AppPort
	}
	if AppPort == "" {
		AppPort = ":8080"
	}
	FrontEndURL = getEnvOrIni("FRONTEND_URL", "GlobalConfig", "FrontEndURL", "http://localhost:5173")
}

func LoadJWTConfig() {
	JWTSecret = getEnvOrIni("JWT_SECRET", "JWTConfig", "Secret", "rahasia_super_aman_cakra_123")
	
	expiryStr := getEnvOrIni("JWT_EXPIRY_HOURS", "JWTConfig", "ExpiryHours", "24")
	expiryInt, err := strconv.Atoi(expiryStr)
	if err != nil || expiryInt == 0 {
		JWTExpiry = 24
	} else {
		JWTExpiry = expiryInt
	}
}

func LoadSupabaseConfig() {
	SupabaseURL = os.Getenv("SUPABASE_URL")
	SupabaseKey = os.Getenv("SUPABASE_KEY")
}

// --- SMTP Configuration ---

var (
	SMTPHost           string
	SMTPPort           string
	SMTPSenderEmail    string
	SMTPSenderPassword string
	SMTPSenderName     string
)

func LoadSMTPConfig() {
	SMTPHost = getEnvOrIni("SMTP_HOST", "SMTPConfig", "Host", "smtp.gmail.com")
	SMTPPort = getEnvOrIni("SMTP_PORT", "SMTPConfig", "Port", "587")
	SMTPSenderEmail = getEnvOrIni("SMTP_EMAIL", "SMTPConfig", "SenderEmail", "")
	SMTPSenderPassword = getEnvOrIni("SMTP_PASSWORD", "SMTPConfig", "SenderPassword", "")
	SMTPSenderName = getEnvOrIni("SMTP_SENDER_NAME", "SMTPConfig", "SenderName", "KPI System Admin")

	// Fallback check if defined at root/DEFAULT in ini
	if SMTPHost == "smtp.gmail.com" && IniConfig != nil {
		if val := IniConfig.Section("").Key("SMTP_HOST").String(); val != "" {
			SMTPHost = val
		}
	}
	if SMTPPort == "587" && IniConfig != nil {
		if val := IniConfig.Section("").Key("SMTP_PORT").String(); val != "" {
			SMTPPort = val
		}
	}
	if SMTPSenderEmail == "" && IniConfig != nil {
		if val := IniConfig.Section("").Key("SMTP_EMAIL").String(); val != "" {
			SMTPSenderEmail = val
		}
	}
	if SMTPSenderPassword == "" && IniConfig != nil {
		if val := IniConfig.Section("").Key("SMTP_PASSWORD").String(); val != "" {
			SMTPSenderPassword = val
		}
	}
}