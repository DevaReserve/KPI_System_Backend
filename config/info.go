package config

import (
	"log"
	"KPI_System_Backend/global_var"
	"KPI_System_Backend/logger"

	"go.uber.org/zap"
	"gopkg.in/ini.v1"
)

var (
	IniConfig *ini.File
	AppPort   string
	JWTSecret string
	JWTExpiry int
)

func InitINIConfig() {
	cfg, err := ini.Load("Setting.ini")
	if err != nil {
		logger.Error("failed to read setting.ini file", zap.Error(err))
		log.Panic("Failed to read file:", err)
	}
	IniConfig = cfg
}

func GetIniDatabase() global_var.DatabaseConnection {
	HostName := IniConfig.Section("MainDatabase").Key("Host Name").String()
	Port := IniConfig.Section("MainDatabase").Key("Port").String()
	UserName := IniConfig.Section("MainDatabase").Key("User Name").String()
	Password := IniConfig.Section("MainDatabase").Key("Password").String()
	DatabaseName := IniConfig.Section("MainDatabase").Key("Database Name").String()
	CreateDBTest := IniConfig.Section("MainDatabase").Key("CreateDBTest").MustBool(false)

	return global_var.DatabaseConnection{
		Host:         HostName,
		Port:         Port,
		User:         UserName,
		Password:     Password,
		DatabaseName: DatabaseName,
		CreateDBTest: CreateDBTest,
	}
}

func LoadAppPort() {
	AppPort = IniConfig.Section("GlobalConfig").Key("AppPort").String()
}

func LoadJWTConfig() {
	JWTSecret = IniConfig.Section("JWTConfig").Key("Secret").String()
	JWTExpiry = IniConfig.Section("JWTConfig").Key("ExpiryHours").MustInt(24)
}