package config

import (
	"KPI_System_Backend/global_var"
	"KPI_System_Backend/logger"

	"go.uber.org/zap"
	"gopkg.in/ini.v1"
)

var (
	IniConfig *ini.File
	AppPort   string
    FrontEndURL string
	JWTSecret string
	JWTExpiry int
)

func InitINIConfig() {
	cfg, err := ini.Load("Setting.ini")
	if err != nil {
		// Fallback jika file tidak ada (agar tidak panic saat dev)
		logger.Warn("failed to read setting.ini file, using defaults", zap.Error(err))
		JWTSecret = "rahasia_default_kpi_system_123"
		JWTExpiry = 24
		AppPort = ":8080"
		return
	}
	IniConfig = cfg
}

func GetIniDatabase() global_var.DatabaseConnection {
	if IniConfig == nil {
		return global_var.DatabaseConnection{}
	}
	
	section := IniConfig.Section("MainDatabase")
	return global_var.DatabaseConnection{
		Host:         section.Key("Host Name").String(),
		Port:         section.Key("Port").String(),
		User:         section.Key("User Name").String(),
		Password:     section.Key("Password").String(),
		DatabaseName: section.Key("Database Name").String(),
		CreateDBTest: section.Key("CreateDBTest").MustBool(false),
	}
}

func LoadAppPort() {
	if IniConfig != nil {
		AppPort = IniConfig.Section("GlobalConfig").Key("AppPort").String()
        FrontEndURL = IniConfig.Section("GlobalConfig").Key("FrontEndURL").String()
	}
	if AppPort == "" {
		AppPort = ":8080"
	}
    if FrontEndURL == "" {
        FrontEndURL = "http://localhost:5173"
    }
}

func LoadJWTConfig() {
	if IniConfig != nil {
		JWTSecret = IniConfig.Section("JWTConfig").Key("Secret").String()
		JWTExpiry = IniConfig.Section("JWTConfig").Key("ExpiryHours").MustInt(24)
	}
	// Fallback values
	if JWTSecret == "" {
		JWTSecret = "rahasia_super_aman_cakra_123"
	}
	if JWTExpiry == 0 {
		JWTExpiry = 24
	}
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
	if IniConfig != nil {
		SMTPHost = IniConfig.Section("SMTPConfig").Key("Host").String()
		SMTPPort = IniConfig.Section("SMTPConfig").Key("Port").String()
		SMTPSenderEmail = IniConfig.Section("SMTPConfig").Key("SenderEmail").String()
		SMTPSenderPassword = IniConfig.Section("SMTPConfig").Key("SenderPassword").String()
		SMTPSenderName = IniConfig.Section("SMTPConfig").Key("SenderName").String()
	}
	// Fallback values
	if SMTPHost == "" {
		SMTPHost = "smtp.gmail.com"
	}
	if SMTPPort == "" {
		SMTPPort = "587"
	}
}