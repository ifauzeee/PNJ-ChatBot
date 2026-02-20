package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/csbot"
	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/pnj-anonymous-bot/internal/logger"
	"github.com/pnj-anonymous-bot/internal/service"
	"go.uber.org/zap"
)

func main() {
	logger.Init()
	defer func() { _ = logger.Log.Sync() }()

	cfg := config.Load()

	logger.InitSentry(cfg.SentryDSN, cfg.SentryEnv)
	defer sentry.Flush(2 * time.Second)

	if cfg.CSBotToken == "" {
		logger.Warn("⚠️ CS_BOT_TOKEN is not set. CS Bot will not start.")
		return
	}
	if cfg.MaintenanceAccountID == 0 {
		logger.Warn("⚠️ MAINTENANCE_ID is not set or invalid. CS Bot will not start.")
		return
	}

	db, err := database.New(cfg)
	if err != nil {
		logger.Fatal("❌ Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	csService := service.NewCSService(db)
	bot, err := csbot.New(cfg, csService)
	if err != nil {
		logger.Fatal("❌ Failed to initialize CS Bot", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	bot.Start(ctx)
}
