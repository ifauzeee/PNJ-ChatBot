package main

import (
	"log"

	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/csbot"
	"github.com/pnj-anonymous-bot/internal/database"
)

func main() {
	cfg := config.Load()

	if cfg.CSBotToken == "" {
		log.Println("⚠️ CS_BOT_TOKEN is not set. CS Bot will not start.")
		return
	}

	db, err := database.New()
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}

	bot, err := csbot.New(cfg, db)
	if err != nil {
		log.Fatalf("❌ Failed to initialize CS Bot: %v", err)
	}

	bot.Start()
}
