package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pnj-anonymous-bot/internal/bot"
	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/database"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)


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
	log.Println(banner)


	log.Println("📋 Loading configuration...")
	cfg := config.Load()


	log.Println("🗄️  Initializing database...")
	db, err := database.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer db.Close()


	log.Println("🤖 Creating bot instance...")
	b, err := bot.New(cfg, db)
	if err != nil {
		log.Fatalf("❌ Failed to create bot: %v", err)
	}


	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("\n🛑 Shutting down gracefully...")
		db.Close()
		os.Exit(0)
	}()


	b.Start()
}
