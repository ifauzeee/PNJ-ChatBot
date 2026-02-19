package csbot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/pnj-anonymous-bot/internal/logger"
	"go.uber.org/zap"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CSBot struct {
	api *tgbotapi.BotAPI
	cfg *config.Config
	db  *database.DB
}

func New(cfg *config.Config, db *database.DB) (*CSBot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.CSBotToken)
	if err != nil {
		return nil, err
	}

	api.Debug = cfg.BotDebug

	return &CSBot{
		api: api,
		cfg: cfg,
		db:  db,
	}, nil
}

func (b *CSBot) Start(ctx context.Context) {
	username := strings.Trim(b.api.Self.UserName, "\"")

	logger.Info("🛠️ CS Bot authorized",
		zap.String("username", username),
		zap.Int64("admin_id", b.cfg.MaintenanceAccountID),
	)

	go b.startTimeoutWorker(ctx)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping CS Bot...")
			return
		case update, ok := <-updates:
			if !ok {
				return
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
}

func (b *CSBot) startTimeoutWorker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			timedOutUsers, err := b.db.GetTimedOutCSSessions(ctx, 5)
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
		userID, _ := b.db.GetActiveCSSessionByAdmin(ctx, telegramID)
		if userID > 0 {
			_ = b.db.UpdateCSSessionActivity(ctx, userID)
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

	adminID, _ := b.db.GetActiveCSSessionByUser(ctx, telegramID)
	if adminID > 0 {
		_ = b.db.UpdateCSSessionActivity(ctx, telegramID)
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
	activeUserID, _ := b.db.GetActiveCSSessionByAdmin(ctx, b.cfg.MaintenanceAccountID)
	if activeUserID == 0 {
		b.startSession(ctx, telegramID)
	} else {
		_ = b.db.JoinCSQueue(ctx, telegramID)
		pos, _ := b.db.GetCSQueuePosition(ctx, telegramID)
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
	_ = b.db.LeaveCSQueue(ctx, userID)
	err := b.db.CreateCSSession(ctx, userID, b.cfg.MaintenanceAccountID)
	if err != nil {
		logger.Error("❌ Error creating CS session", zap.Error(err))
		return
	}

	b.sendMessage(userID, "🎧 <b>Terhubung dengan agen!</b>\nSilakan sampaikan pertanyaan atau kendala kamu.")
	b.sendMessage(b.cfg.MaintenanceAccountID, fmt.Sprintf("📩 <b>SESSION BARU</b>\nUser: %d\n\nSilakan balas pesan untuk memulai percakapan.", userID))
}

func (b *CSBot) endSession(ctx context.Context, userID int64, message string) {
	_ = b.db.EndCSSession(ctx, userID)
	b.sendMessage(userID, message)
	b.sendMessage(b.cfg.MaintenanceAccountID, fmt.Sprintf("🛑 <b>Sesi dengan user %d berakhir.</b>", userID))
}

func (b *CSBot) processQueue(ctx context.Context) {
	nextUserID, err := b.db.GetNextInCSQueue(ctx)
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
