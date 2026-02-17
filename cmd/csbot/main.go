package main

import (
	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/csbot"
	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/pnj-anonymous-bot/internal/logger"
	"go.uber.org/zap"
)

func main() {
	logger.Init()
	defer logger.Log.Sync()

	cfg := config.Load()

	if cfg.CSBotToken == "" {
		logger.Warn("⚠️ CS_BOT_TOKEN is not set. CS Bot will not start.")
		return
	}

	db, err := database.New()
	if err != nil {
		logger.Fatal("❌ Failed to connect to database", zap.Error(err))
	}

	bot, err := csbot.New(cfg, db)
	if err != nil {
		logger.Fatal("❌ Failed to initialize CS Bot", zap.Error(err))
	}

	bot.Start()
}
