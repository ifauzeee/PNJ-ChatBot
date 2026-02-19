package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/pnj-anonymous-bot/internal/bot"
	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/pnj-anonymous-bot/internal/logger"
	"go.uber.org/zap"
)

func main() {
	logger.Init()
	defer func() { _ = logger.Log.Sync() }()

	banner := `
╔══════════════════════════════════════════════════╗
║                                                  ║
║   🎭  PNJ Anonymous Bot  🎭                     ║
║                                                  ║
║   Politeknik Negeri Jakarta                      ║
║   Anonymous Chat & Confession Platform           ║
║                                                  ║
║   Built with ❤️ in Go                            ║
║                                                  ║
╚══════════════════════════════════════════════════╝
`
	logger.Info(banner)

	logger.Info("📋 Loading configuration...")
	cfg := config.Load()

	logger.Info("🛡️ Initializing error tracking...")
	logger.InitSentry(cfg.SentryDSN, cfg.SentryEnv)
	defer sentry.Flush(2 * time.Second)

	logger.Info("🗄️  Initializing database...")
	db, err := database.New(cfg)
	if err != nil {
		logger.Fatal("❌ Failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	logger.Info("🤖 Creating bot instance...")
	b, err := bot.New(cfg, db)
	if err != nil {
		logger.Fatal("❌ Failed to create bot", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := b.Start(ctx); err != nil {
		logger.Fatal("Bot stopped with error", zap.Error(err))
	}
}
