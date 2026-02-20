package csbot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/logger"
	"github.com/pnj-anonymous-bot/internal/service"
	"go.uber.org/zap"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CSBot struct {
	api        *tgbotapi.BotAPI
	cfg        *config.Config
	svc        service.CSSessionManager
	startedAt  time.Time
	background sync.WaitGroup
}

func New(cfg *config.Config, svc service.CSSessionManager) (*CSBot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.CSBotToken)
	if err != nil {
		return nil, err
	}

	api.Debug = cfg.BotDebug

	return &CSBot{
		api:       api,
		cfg:       cfg,
		svc:       svc,
		startedAt: time.Now(),
	}, nil
}

func (b *CSBot) Start(ctx context.Context) {
	username := strings.Trim(b.api.Self.UserName, "\"")

	logger.Info("🛠️ CS Bot authorized",
		zap.String("username", username),
		zap.Int64("admin_id", b.cfg.MaintenanceAccountID),
	)

	b.background.Add(1)
	go func() {
		defer b.background.Done()
		b.startHealthServer(ctx)
	}()

	go b.startTimeoutWorker(ctx)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

mainLoop:
	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping CS Bot...")
			break mainLoop
		case update, ok := <-updates:
			if !ok {
				break mainLoop
			}
			if update.Message == nil {
				continue
			}
			logger.Debug("CS update received",
				zap.Int64("from_id", update.Message.From.ID),
				zap.Bool("is_command", update.Message.IsCommand()),
				zap.Int("text_len", len(update.Message.Text)),
			)
			b.handleMessage(ctx, update.Message)
		}
	}

	logger.Info("⏳ Waiting for background tasks to finish...")
	b.background.Wait()
	logger.Info("🛑 CS Bot shutdown completed")
}

func (b *CSBot) startTimeoutWorker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			timedOutUsers, err := b.svc.GetTimedOutSessions(ctx, 5)
			if err != nil {
				continue
			}

			for _, userID := range timedOutUsers {
				b.endSession(ctx, userID, "⏰ <b>Sesi berakhir.</b> Tidak ada aktivitas selama 5 menit.")
				b.processQueue(ctx)
			}
		}
	}
}

func (b *CSBot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	if telegramID == b.cfg.MaintenanceAccountID {
		userID, _ := b.svc.GetActiveSessionByAdmin(ctx, telegramID)
		if userID > 0 {
			_ = b.svc.UpdateSessionActivity(ctx, userID)
			if msg.IsCommand() && (msg.Command() == "stop" || msg.Command() == "end") {
				b.handleStop(ctx, userID)
				return
			}
			b.handleAdminReply(userID, msg)
			return
		}

		if msg.IsCommand() {
			switch msg.Command() {
			case "start", "help":
				b.handleHelp(telegramID)
			default:
				b.sendMessage(telegramID, "💡 Kamu adalah Admin. Gunakan /chat pada akun User untuk mencoba, lalu balas dari sini.")
			}
		}
		return
	}

	adminID, _ := b.svc.GetActiveSessionByUser(ctx, telegramID)
	if adminID > 0 {
		_ = b.svc.UpdateSessionActivity(ctx, telegramID)
		if msg.IsCommand() && msg.Command() == "stop" {
			b.handleStop(ctx, telegramID)
			return
		}
		b.forwardToAdmin(telegramID, adminID, msg)
		return
	}

	if msg.IsCommand() {
		switch msg.Command() {
		case "start", "help":
			b.handleHelp(telegramID)
		case "chat":
			b.handleChat(ctx, telegramID)
		default:
			b.sendMessage(telegramID, "❓ Perintah tidak dikenal. Ketik /chat untuk bantuan.")
		}
		return
	}

	b.sendMessage(telegramID, "💡 Silakan ketik /chat untuk tersambung ke layanan Customer Service.")
}

func (b *CSBot) handleHelp(telegramID int64) {
	helpText := `🎧 <b>Customer Service PNJ Chat Bot</b>

Selamat datang di layanan bantuan resmi PNJ Anonymous Bot.

<b>Perintah:</b>
/chat - Mulai sesi chat dengan agen CS
/stop - Akhiri sesi chat aktif
/help - Tampilkan pesan bantuan ini

<i>Apabila agen sedang sibuk, kamu akan masuk ke dalam antrian. Sesuai ketentuan, sesi akan berakhir otomatis jika tidak ada aktivitas selama 5 menit.</i>`
	b.sendMessage(telegramID, helpText)
}

func (b *CSBot) handleChat(ctx context.Context, telegramID int64) {
	activeUserID, _ := b.svc.GetActiveSessionByAdmin(ctx, b.cfg.MaintenanceAccountID)
	if activeUserID == 0 {
		b.startSession(ctx, telegramID)
	} else {
		_ = b.svc.JoinQueue(ctx, telegramID)
		pos, _ := b.svc.GetQueuePosition(ctx, telegramID)
		b.sendMessage(telegramID, fmt.Sprintf("⏳ <b>Agen sedang melayani pengguna lain.</b>\n\nKamu telah masuk ke dalam antrian. Posisi kamu saat ini: <b>#%d</b>.\nMohon tunggu sebentar, kami akan memberitahumu secara otomatis jika sudah terhubung.", pos))
	}
}

func (b *CSBot) handleStop(ctx context.Context, userID int64) {
	b.endSession(ctx, userID, "⏹️ <b>Sesi chat telah diakhiri.</b> Terima kasih telah menghubungi kami.")
	b.processQueue(ctx)
}

func (b *CSBot) startSession(ctx context.Context, userID int64) {
	logger.Info("🚀 Starting CS session",
		zap.Int64("user_id", userID),
		zap.Int64("admin_id", b.cfg.MaintenanceAccountID),
	)
	_ = b.svc.LeaveQueue(ctx, userID)
	err := b.svc.CreateSession(ctx, userID, b.cfg.MaintenanceAccountID)
	if err != nil {
		logger.Error("❌ Error creating CS session", zap.Error(err))
		return
	}

	b.sendMessage(userID, "🎧 <b>Terhubung dengan agen!</b>\nSilakan sampaikan pertanyaan atau kendala kamu.")
	b.sendMessage(b.cfg.MaintenanceAccountID, fmt.Sprintf("📩 <b>SESSION BARU</b>\nUser: %d\n\nSilakan balas pesan untuk memulai percakapan.", userID))
}

func (b *CSBot) endSession(ctx context.Context, userID int64, message string) {
	_ = b.svc.EndSession(ctx, userID)
	b.sendMessage(userID, message)
	b.sendMessage(b.cfg.MaintenanceAccountID, fmt.Sprintf("🛑 <b>Sesi dengan user %d berakhir.</b>", userID))
}

func (b *CSBot) processQueue(ctx context.Context) {
	nextUserID, err := b.svc.GetNextInQueue(ctx)
	if err == nil && nextUserID > 0 {
		b.startSession(ctx, nextUserID)
	}
}

func (b *CSBot) forwardToAdmin(userID, adminID int64, msg *tgbotapi.Message) {
	logger.Debug("Forwarding message to admin",
		zap.Int64("from_user", userID),
		zap.Int64("to_admin", adminID),
		zap.Int("text_len", len(msg.Text)),
	)
	text := fmt.Sprintf("👤 <b>USER %d</b>\n\n%s", userID, msg.Text)
	b.sendMessage(adminID, text)
}

func (b *CSBot) handleAdminReply(userID int64, adminMsg *tgbotapi.Message) {
	reply := tgbotapi.NewMessage(userID, fmt.Sprintf("🎧 <b>Customer Service:</b>\n\n%s", adminMsg.Text))
	reply.ParseMode = "HTML"

	_, err := b.api.Send(reply)
	if err != nil {
		b.sendMessage(b.cfg.MaintenanceAccountID, "❌ Gagal mengirim balasan ke user.")
	}
}

func (b *CSBot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	_, err := b.api.Send(msg)
	if err != nil {
		logger.Error("❌ Error sending message",
			zap.Int64("chat_id", chatID),
			zap.Error(err),
		)
	}
}

type HealthResponse struct {
	Status     string `json:"status"`
	Bot        string `json:"bot"`
	Uptime     string `json:"uptime"`
	GoVersion  string `json:"go_version"`
	MemoryMB   uint64 `json:"memory_mb"`
	Goroutines int    `json:"goroutines"`
	Timestamp  string `json:"timestamp"`
}

func (b *CSBot) startHealthServer(ctx context.Context) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		health := HealthResponse{
			Status:     "ok",
			Bot:        fmt.Sprintf("@%s (CS)", b.api.Self.UserName),
			Uptime:     time.Since(b.startedAt).Round(time.Second).String(),
			GoVersion:  runtime.Version(),
			MemoryMB:   memStats.Alloc / 1024 / 1024,
			Goroutines: runtime.NumGoroutine(),
			Timestamp:  time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(health)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Warn("CS Health server shutdown error", zap.Error(err))
		}
	}()

	logger.Info("🏥 CS Health check server listening on :" + port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("⚠️ CS Health check server error", zap.Error(err))
	}
}
