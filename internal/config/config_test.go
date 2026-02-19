package config

import (
	"os"
	"testing"

	"github.com/pnj-anonymous-bot/internal/logger"
)

func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}

func TestLoadDefaultValues(t *testing.T) {

	os.Setenv("BOT_TOKEN", "test-token-12345")
	defer os.Unsetenv("BOT_TOKEN")

	cfg := Load()

	if cfg.BotToken != "test-token-12345" {
		t.Errorf("BotToken = %q, want %q", cfg.BotToken, "test-token-12345")
	}

	if cfg.MaxUpdateWorkers != 16 {
		t.Errorf("MaxUpdateWorkers = %d, want 16", cfg.MaxUpdateWorkers)
	}
	if cfg.MaxUpdateQueue != 256 {
		t.Errorf("MaxUpdateQueue = %d, want 256", cfg.MaxUpdateQueue)
	}
	if cfg.OTPLength != 6 {
		t.Errorf("OTPLength = %d, want 6", cfg.OTPLength)
	}
	if cfg.DBType != "sqlite" {
		t.Errorf("DBType = %q, want %q", cfg.DBType, "sqlite")
	}
}

func TestLoadCustomValues(t *testing.T) {
	os.Setenv("BOT_TOKEN", "test-token")
	os.Setenv("MAX_UPDATE_WORKERS", "8")
	os.Setenv("MAX_UPDATE_QUEUE", "128")
	os.Setenv("OTP_LENGTH", "4")
	os.Setenv("MAX_SEARCH_PER_MINUTE", "10")
	defer func() {
		os.Unsetenv("BOT_TOKEN")
		os.Unsetenv("MAX_UPDATE_WORKERS")
		os.Unsetenv("MAX_UPDATE_QUEUE")
		os.Unsetenv("OTP_LENGTH")
		os.Unsetenv("MAX_SEARCH_PER_MINUTE")
	}()

	cfg := Load()

	if cfg.MaxUpdateWorkers != 8 {
		t.Errorf("MaxUpdateWorkers = %d, want 8", cfg.MaxUpdateWorkers)
	}
	if cfg.MaxUpdateQueue != 128 {
		t.Errorf("MaxUpdateQueue = %d, want 128", cfg.MaxUpdateQueue)
	}
	if cfg.OTPLength != 4 {
		t.Errorf("OTPLength = %d, want 4", cfg.OTPLength)
	}
	if cfg.MaxSearchPerMinute != 10 {
		t.Errorf("MaxSearchPerMinute = %d, want 10", cfg.MaxSearchPerMinute)
	}
}

func TestValidateClampsBadValues(t *testing.T) {
	cfg := &Config{
		BotToken:              "test",
		MaxUpdateWorkers:      4,
		MaxUpdateQueue:        32,
		OTPLength:             2,
		OTPExpiryMinutes:      99,
		MaxSearchPerMinute:    -1,
		MaxConfessionsPerHour: 0,
		MaxReportsPerDay:      0,
		AutoBanReportCount:    0,
	}

	cfg.validate()

	if cfg.OTPLength != 6 {
		t.Errorf("OTPLength = %d, want 6 (clamped)", cfg.OTPLength)
	}
	if cfg.OTPExpiryMinutes != 10 {
		t.Errorf("OTPExpiryMinutes = %d, want 10 (clamped)", cfg.OTPExpiryMinutes)
	}
	if cfg.MaxSearchPerMinute != 5 {
		t.Errorf("MaxSearchPerMinute = %d, want 5 (clamped)", cfg.MaxSearchPerMinute)
	}
	if cfg.MaxConfessionsPerHour != 3 {
		t.Errorf("MaxConfessionsPerHour = %d, want 3 (clamped)", cfg.MaxConfessionsPerHour)
	}
	if cfg.MaxReportsPerDay != 5 {
		t.Errorf("MaxReportsPerDay = %d, want 5 (clamped)", cfg.MaxReportsPerDay)
	}
	if cfg.AutoBanReportCount != 3 {
		t.Errorf("AutoBanReportCount = %d, want 3 (clamped)", cfg.AutoBanReportCount)
	}
}

func TestGetEnvHelpers(t *testing.T) {
	os.Setenv("TEST_STRING", "hello")
	os.Setenv("TEST_INT", "42")
	os.Setenv("TEST_BOOL", "true")
	os.Setenv("TEST_INT64", "9999999999")
	defer func() {
		os.Unsetenv("TEST_STRING")
		os.Unsetenv("TEST_INT")
		os.Unsetenv("TEST_BOOL")
		os.Unsetenv("TEST_INT64")
	}()

	if v := getEnv("TEST_STRING", "default"); v != "hello" {
		t.Errorf("getEnv = %q, want %q", v, "hello")
	}
	if v := getEnv("NONEXISTENT", "default"); v != "default" {
		t.Errorf("getEnv default = %q, want %q", v, "default")
	}
	if v := getEnvInt("TEST_INT", 0); v != 42 {
		t.Errorf("getEnvInt = %d, want 42", v)
	}
	if v := getEnvInt("NONEXISTENT", 99); v != 99 {
		t.Errorf("getEnvInt default = %d, want 99", v)
	}
	if v := getEnvBool("TEST_BOOL", false); v != true {
		t.Errorf("getEnvBool = %v, want true", v)
	}
	if v := getEnvInt64("TEST_INT64", 0); v != 9999999999 {
		t.Errorf("getEnvInt64 = %d, want 9999999999", v)
	}
}
