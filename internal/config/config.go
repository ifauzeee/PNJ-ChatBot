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
	MaxWhispersPerHour    int
	MaxRepliesPerHour     int

	AutoBanReportCount int

	MaintenanceAccountID int64
	BrevoAPIKey          string
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
		MaxWhispersPerHour:    getEnvInt("MAX_WHISPERS_PER_HOUR", 5),
		MaxRepliesPerHour:     getEnvInt("MAX_REPLIES_PER_HOUR", 10),
		AutoBanReportCount:    getEnvInt("AUTO_BAN_REPORT_COUNT", 3),
		MaintenanceAccountID:  getEnvInt64("MAINTENANCE_ID", 0),
		BrevoAPIKey:           getEnv("BREVO_API_KEY", ""),
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

	cfg.validate()

	return cfg
}

func (cfg *Config) validate() {
	warnings := []string{}

	if cfg.DBType == "postgres" {
		if cfg.DBHost == "" || cfg.DBUser == "" || cfg.DBName == "" {
			logger.Fatal("❌ PostgreSQL selected but DB_HOST, DB_USER, or DB_NAME is empty")
		}
	}

	if cfg.BrevoAPIKey == "" {
		warnings = append(warnings, "BREVO_API_KEY is not set. OTP emails will fail.")
	}
	if cfg.SMTPUsername == "" {
		warnings = append(warnings, "SMTP_USERNAME is not set. Brevo sender email may be invalid.")
	}

	if cfg.MaxSearchPerMinute <= 0 {
		cfg.MaxSearchPerMinute = 5
		warnings = append(warnings, "MAX_SEARCH_PER_MINUTE invalid, defaulting to 5")
	}
	if cfg.MaxConfessionsPerHour <= 0 {
		cfg.MaxConfessionsPerHour = 3
		warnings = append(warnings, "MAX_CONFESSIONS_PER_HOUR invalid, defaulting to 3")
	}
	if cfg.MaxReportsPerDay <= 0 {
		cfg.MaxReportsPerDay = 5
		warnings = append(warnings, "MAX_REPORTS_PER_DAY invalid, defaulting to 5")
	}
	if cfg.AutoBanReportCount <= 0 {
		cfg.AutoBanReportCount = 3
		warnings = append(warnings, "AUTO_BAN_REPORT_COUNT invalid, defaulting to 3")
	}

	if cfg.OTPLength < 4 || cfg.OTPLength > 8 {
		cfg.OTPLength = 6
		warnings = append(warnings, "OTP_LENGTH out of range (4-8), defaulting to 6")
	}
	if cfg.OTPExpiryMinutes < 1 || cfg.OTPExpiryMinutes > 60 {
		cfg.OTPExpiryMinutes = 10
		warnings = append(warnings, "OTP_EXPIRY_MINUTES out of range (1-60), defaulting to 10")
	}

	if cfg.SightengineAPIUser == "" || cfg.SightengineAPISecret == "" {
		warnings = append(warnings, "Sightengine API not configured. Image moderation disabled.")
	}

	if cfg.MaintenanceAccountID == 0 {
		warnings = append(warnings, "MAINTENANCE_ID not set. Admin commands will be disabled.")
	}

	for _, w := range warnings {
		logger.Warn("⚠️ Config warning: " + w)
	}

	logger.Info("📋 Configuration loaded",
		zap.String("db_type", cfg.DBType),
		zap.Int("workers", cfg.MaxUpdateWorkers),
		zap.Int("queue_size", cfg.MaxUpdateQueue),
		zap.Int("max_search_per_min", cfg.MaxSearchPerMinute),
		zap.Int("max_confess_per_hr", cfg.MaxConfessionsPerHour),
		zap.Int("auto_ban_threshold", cfg.AutoBanReportCount),
	)
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
