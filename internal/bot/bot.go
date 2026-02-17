package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/pnj-anonymous-bot/internal/email"
	"github.com/pnj-anonymous-bot/internal/models"
	"github.com/pnj-anonymous-bot/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api        *tgbotapi.BotAPI
	cfg        *config.Config
	db         *database.DB
	auth       *service.AuthService
	chat       *service.ChatService
	confession *service.ConfessionService
	profile    *service.ProfileService
	room       *service.RoomService
	startedAt  time.Time
}

func New(cfg *config.Config, db *database.DB) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, err
	}

	api.Debug = cfg.BotDebug

	emailSender := email.NewSender(cfg)

	bot := &Bot{
		api:        api,
		cfg:        cfg,
		db:         db,
		auth:       service.NewAuthService(db, emailSender, cfg),
		chat:       service.NewChatService(db),
		confession: service.NewConfessionService(db, cfg),
		profile:    service.NewProfileService(db, cfg),
		room:       service.NewRoomService(db),
		startedAt:  time.Now(),
	}

	log.Printf("🤖 Bot authorized as @%s", api.Self.UserName)
	return bot, nil
}

type HealthResponse struct {
	Status     string `json:"status"`
	Bot        string `json:"bot"`
	Uptime     string `json:"uptime"`
	GoVersion  string `json:"go_version"`
	MemoryMB   uint64 `json:"memory_mb"`
	Goroutines int    `json:"goroutines"`
	Users      int    `json:"registered_users"`
	Queue      int    `json:"queue_size"`
	Timestamp  string `json:"timestamp"`
}

func (b *Bot) startHealthServer() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		userCount, _ := b.db.GetOnlineUserCount()
		queueCount, _ := b.chat.GetQueueCount()

		health := HealthResponse{
			Status:     "ok",
			Bot:        fmt.Sprintf("@%s", b.api.Self.UserName),
			Uptime:     time.Since(b.startedAt).Round(time.Second).String(),
			GoVersion:  runtime.Version(),
			MemoryMB:   memStats.Alloc / 1024 / 1024,
			Goroutines: runtime.NumGoroutine(),
			Users:      userCount,
			Queue:      queueCount,
			Timestamp:  time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(health)
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {

		if err := b.db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"not_ready","error":"%s"}`, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ready"}`)
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		userCount, _ := b.db.GetOnlineUserCount()
		queueCount, _ := b.chat.GetQueueCount()

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "# HELP pnj_bot_uptime_seconds Bot uptime in seconds\n")
		fmt.Fprintf(w, "pnj_bot_uptime_seconds %.0f\n", time.Since(b.startedAt).Seconds())
		fmt.Fprintf(w, "# HELP pnj_bot_goroutines Number of goroutines\n")
		fmt.Fprintf(w, "pnj_bot_goroutines %d\n", runtime.NumGoroutine())
		fmt.Fprintf(w, "# HELP pnj_bot_memory_bytes Memory usage in bytes\n")
		fmt.Fprintf(w, "pnj_bot_memory_bytes %d\n", memStats.Alloc)
		fmt.Fprintf(w, "# HELP pnj_bot_registered_users Total registered users\n")
		fmt.Fprintf(w, "pnj_bot_registered_users %d\n", userCount)
		fmt.Fprintf(w, "# HELP pnj_bot_queue_size Current search queue size\n")
		fmt.Fprintf(w, "pnj_bot_queue_size %d\n", queueCount)
	})

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	log.Println("🏥 Health check server listening on :8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("⚠️  Health check server error: %v", err)
	}
}

func (b *Bot) startQueueWorker() {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			updatedIDs, err := b.chat.ProcessQueueTimeout(60)
			if err != nil {
				log.Printf("⚠️ Queue worker error: %v", err)
				continue
			}

			for _, telegramID := range updatedIDs {
				msg := `⏳ *Belum menemukan partner...*

Karena belum ada partner yang cocok dengan kriteria kamu, sekarang bot akan mencari partner secara *acak* agar lebih cepat.

_Mohon tunggu sebentar ya..._`
				b.sendMessage(telegramID, msg, nil)
			}
		}
	}()
}

func (b *Bot) Start() {
	log.Println("🚀 Starting PNJ Anonymous Bot...")

	go b.startHealthServer()
	b.startQueueWorker()

	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "🎭 Mulai bot / Menu utama"},
		{Command: "regist", Description: "📝 Registrasi akun baru"},
		{Command: "search", Description: "🔍 Cari partner chat anonim"},
		{Command: "next", Description: "⏭️ Skip ke partner berikutnya"},
		{Command: "stop", Description: "🛑 Hentikan chat saat ini"},
		{Command: "confess", Description: "💬 Kirim confession anonim"},
		{Command: "confessions", Description: "📋 Lihat confession terbaru"},
		{Command: "react", Description: "❤️ Reaksi ke confession"},
		{Command: "reply", Description: "Balas confession (contoh: /reply 1 Hallo!)"},
		{Command: "view_replies", Description: "Lihat balasan confession (contoh: /view_replies 1)"},
		{Command: "poll", Description: "🗳️ Buat polling anonim"},
		{Command: "polls", Description: "📊 Lihat daftar polling"},
		{Command: "vote_poll", Description: "🗳️ Ikut memilih polling (contoh: /vote_poll 1)"},
		{Command: "whisper", Description: "📢 Kirim whisper ke jurusan"},
		{Command: "circles", Description: "👥 Gabung Group Circle Anonim"},
		{Command: "profile", Description: "👤 Lihat profil kamu"},
		{Command: "stats", Description: "📊 Statistik kamu"},
		{Command: "edit", Description: "✏️ Edit profil"},
		{Command: "about", Description: "⚖️ Informasi hukum & disclaimer"},
		{Command: "help", Description: "❓ Bantuan & panduan"},
		{Command: "cancel", Description: "❌ Batalkan aksi saat ini"},
	}
	cmdCfg := tgbotapi.NewSetMyCommands(commands...)
	b.api.Send(cmdCfg)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		go b.handleUpdate(update)
	}
}

func (b *Bot) handleUpdate(update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ Panic recovered: %v", r)
		}
	}()

	if update.CallbackQuery != nil {
		b.handleCallback(update.CallbackQuery)
		return
	}

	if update.Message == nil {
		return
	}

	if update.Message.IsCommand() {
		b.handleCommand(update.Message)
		return
	}

	b.handleMessage(update.Message)
}

func (b *Bot) handleCommand(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	switch msg.Command() {
	case "start":
		b.handleStart(msg)
	case "regist":
		b.handleRegist(msg)
	case "help":
		b.handleHelp(msg)
	case "about":
		b.handleAbout(msg)
	case "cancel":
		b.handleCancel(msg)
	default:

		if !b.requireVerification(msg) {
			return
		}

		if banned, _ := b.auth.IsBanned(telegramID); banned {
			b.sendMessage(telegramID, "🚫 *Akun kamu telah di-banned.*\n\nKamu tidak bisa menggunakan bot ini karena telah melanggar aturan.", nil)
			return
		}

		switch msg.Command() {
		case "search":
			b.handleSearch(msg)
		case "next":
			b.handleNext(msg)
		case "stop":
			b.handleStop(msg)
		case "confess":
			b.handleConfess(msg)
		case "confessions":
			b.handleConfessions(msg)
		case "react":
			b.handleReact(msg)
		case "reply":
			b.handleReply(msg)
		case "view_replies":
			b.handleViewReplies(msg)
		case "poll":
			b.handlePoll(msg)
		case "polls":
			b.handleViewPolls(msg)
		case "vote_poll":
			b.handleVotePoll(msg)
		case "whisper":
			b.handleWhisper(msg)
		case "profile":
			b.handleProfile(msg)
		case "stats":
			b.handleStats(msg)
		case "edit":
			b.handleEdit(msg)
		case "report":
			b.handleReport(msg)
		case "block":
			b.handleBlock(msg)
		case "circles":
			b.handleCircles(msg)
		case "leave_circle":
			b.handleLeaveCircle(msg)
		default:
			b.sendMessage(telegramID, "❓ Perintah tidak dikenali. Ketik /help untuk bantuan.", nil)
		}
	}
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	state, stateData, err := b.db.GetUserState(telegramID)
	if err != nil {
		log.Printf("Error getting user state: %v", err)
		return
	}

	switch state {
	case models.StateAwaitingEmail:
		b.handleEmailInput(msg)
	case models.StateAwaitingOTP:
		b.handleOTPInput(msg)
	case models.StateInChat:
		b.handleChatMessage(msg)
	case models.StateAwaitingConfess:
		b.handleConfessionInput(msg)
	case models.StateAwaitingReport:
		b.handleReportInput(msg)
	case models.StateAwaitingWhisper:
		b.handleWhisperInput(msg, stateData)
	case models.StateInCircle:
		b.handleCircleMessage(msg)
	case models.StateAwaitingRoomName:
		b.handleRoomNameInput(msg)
	case models.StateAwaitingRoomDesc:
		b.handleRoomDescInput(msg)
	default:

		if msg.Text != "" {
			b.sendMessage(telegramID, "💡 Gunakan /start untuk membuka menu utama atau /help untuk bantuan.", nil)
		}
	}
}

func (b *Bot) requireVerification(msg *tgbotapi.Message) bool {
	telegramID := msg.From.ID

	if b.cfg.MaintenanceAccountID != 0 && telegramID == b.cfg.MaintenanceAccountID {
		user, _ := b.db.GetUser(telegramID)
		if user == nil {
			b.db.CreateUser(telegramID)
			b.db.UpdateUserDisplayName(telegramID, "🛠️ Maintenance Account")
			b.db.UpdateUserVerified(telegramID, true)
			b.db.UpdateUserGender(telegramID, "Maintenance")
			b.db.UpdateUserDepartment(telegramID, "System")
		}
		return true
	}

	user, err := b.db.GetUser(telegramID)
	if err != nil || user == nil {
		b.sendMessage(telegramID, "⚠️ Kamu belum terdaftar. Ketik /start untuk memulai.", nil)
		return false
	}

	if !user.IsVerified {
		b.sendMessage(telegramID, "⚠️ *Email belum diverifikasi!*\n\nKetik /regist dan ikuti proses verifikasi email PNJ kamu.", nil)
		return false
	}

	if string(user.Gender) == "" || string(user.Department) == "" {
		b.sendMessage(telegramID, "⚠️ *Profil belum lengkap!*\n\nKetik /start untuk melengkapi profil kamu.", nil)
		return false
	}

	return true
}

func (b *Bot) sendMessage(chatID int64, text string, keyboard *tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	if keyboard != nil {
		msg.ReplyMarkup = keyboard
	}

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Error sending message to %d: %v", chatID, err)
	}
}

func (b *Bot) sendMessageHTML(chatID int64, text string, keyboard *tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	if keyboard != nil {
		msg.ReplyMarkup = keyboard
	}

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Error sending HTML message to %d: %v", chatID, err)
	}
}

func (b *Bot) answerCallback(callbackID string, text string) {
	callback := tgbotapi.NewCallback(callbackID, text)
	if _, err := b.api.Request(callback); err != nil {
		log.Printf("Error answering callback: %v", err)
	}
}
