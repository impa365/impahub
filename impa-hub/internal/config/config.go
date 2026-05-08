package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort         string
	ServerHost         string
	BaseURL            string
	DatabaseURL        string
	JWTSecret          string
	JWTExpirationHours int
	AdminEmail         string
	AdminPassword      string
	LogLevel           string
	CORSOrigins        string
}

var AppConfig *Config

func Load() *Config {
	_ = godotenv.Load()

	expHours, _ := strconv.Atoi(getEnv("JWT_EXPIRATION_HOURS", "24"))

	AppConfig = &Config{
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		ServerHost:         getEnv("SERVER_HOST", "0.0.0.0"),
		BaseURL:            getEnv("BASE_URL", "http://localhost:8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://impa:impa123@localhost:5432/impa_hub?sslmode=disable"),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		JWTExpirationHours: expHours,
		AdminEmail:         getEnv("ADMIN_EMAIL", "admin@impa.hub"),
		AdminPassword:      getEnv("ADMIN_PASSWORD", "admin123"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		CORSOrigins:        getEnv("CORS_ORIGINS", "*"),
	}

	return AppConfig
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
