package config

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/pnj-anonymous-bot/internal/logger"
	"go.uber.org/zap"
)

type Config struct {
	BotToken   string
	CSBotToken string
	BotDebug   bool

	MaxUpdateWorkers int
	MaxUpdateQueue   int

	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string

	DBType     string
	DBPath     string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	OTPLength        int
	OTPExpiryMinutes int

	MaxSearchPerMinute    int
	MaxConfessionsPerHour int
	MaxReportsPerDay      int

	AutoBanReportCount int

	MaintenanceAccountID int64
	SightengineAPIUser   string
	SightengineAPISecret string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logger.Warn("Failed to load .env, using existing environment variables", zap.Error(err))
		}
	}

	cfg := &Config{
		BotToken:              getEnv("BOT_TOKEN", ""),
		CSBotToken:            getEnv("CS_BOT_TOKEN", ""),
		BotDebug:              getEnvBool("BOT_DEBUG", false),
		MaxUpdateWorkers:      getEnvInt("MAX_UPDATE_WORKERS", 16),
		MaxUpdateQueue:        getEnvInt("MAX_UPDATE_QUEUE", 256),
		SMTPHost:              getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:              getEnvInt("SMTP_PORT", 587),
		SMTPUsername:          getEnv("SMTP_USERNAME", ""),
		SMTPPassword:          getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:              getEnv("SMTP_FROM", ""),
		DBType:                getEnv("DB_TYPE", "sqlite"),
		DBPath:                getEnv("DB_PATH", "./data/pnj_anonymous.db"),
		DBHost:                getEnv("DB_HOST", "localhost"),
		DBPort:                getEnv("DB_PORT", "5432"),
		DBUser:                getEnv("DB_USER", "postgres"),
		DBPassword:            getEnv("DB_PASSWORD", ""),
		DBName:                getEnv("DB_NAME", "pnjbot"),
		OTPLength:             getEnvInt("OTP_LENGTH", 6),
		OTPExpiryMinutes:      getEnvInt("OTP_EXPIRY_MINUTES", 10),
		MaxSearchPerMinute:    getEnvInt("MAX_SEARCH_PER_MINUTE", 5),
		MaxConfessionsPerHour: getEnvInt("MAX_CONFESSIONS_PER_HOUR", 3),
		MaxReportsPerDay:      getEnvInt("MAX_REPORTS_PER_DAY", 5),
		AutoBanReportCount:    getEnvInt("AUTO_BAN_REPORT_COUNT", 3),
		MaintenanceAccountID:  getEnvInt64("MAINTENANCE_ID", 0),
		SightengineAPIUser:    getEnv("SIGHTENGINE_API_USER", ""),
		SightengineAPISecret:  getEnv("SIGHTENGINE_API_SECRET", ""),
	}

	if cfg.BotToken == "" {
		logger.Fatal("❌ BOT_TOKEN is required! Set it in .env or environment variables.")
	}

	if cfg.MaxUpdateWorkers < 1 {
		cfg.MaxUpdateWorkers = 1
	}
	if cfg.MaxUpdateQueue < 1 {
		cfg.MaxUpdateQueue = 1
	}

	return cfg
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}

func getEnvInt64(key string, defaultVal int64) int64 {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i
		}
	}
	return defaultVal
}
