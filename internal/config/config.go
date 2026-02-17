package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken string
	BotDebug bool

	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string

	DBPath string

	OTPLength        int
	OTPExpiryMinutes int

	MaxSearchPerMinute    int
	MaxConfessionsPerHour int
	MaxReportsPerDay      int

	AutoBanReportCount int
}

func Load() *Config {

	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, using environment variables")
	}

	cfg := &Config{
		BotToken:              getEnv("BOT_TOKEN", ""),
		BotDebug:              getEnvBool("BOT_DEBUG", false),
		SMTPHost:              getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:              getEnvInt("SMTP_PORT", 587),
		SMTPUsername:          getEnv("SMTP_USERNAME", ""),
		SMTPPassword:          getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:              getEnv("SMTP_FROM", ""),
		DBPath:                getEnv("DB_PATH", "./data/pnj_anonymous.db"),
		OTPLength:             getEnvInt("OTP_LENGTH", 6),
		OTPExpiryMinutes:      getEnvInt("OTP_EXPIRY_MINUTES", 10),
		MaxSearchPerMinute:    getEnvInt("MAX_SEARCH_PER_MINUTE", 5),
		MaxConfessionsPerHour: getEnvInt("MAX_CONFESSIONS_PER_HOUR", 3),
		MaxReportsPerDay:      getEnvInt("MAX_REPORTS_PER_DAY", 5),
		AutoBanReportCount:    getEnvInt("AUTO_BAN_REPORT_COUNT", 3),
	}

	if cfg.BotToken == "" {
		log.Fatal("❌ BOT_TOKEN is required! Set it in .env or environment variables.")
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
