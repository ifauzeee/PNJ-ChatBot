package csbot

import (
	"fmt"
	"log"

	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/database"

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

func (b *CSBot) Start() {
	log.Printf("🛠️ CS Bot authorized as @%s", b.api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		b.handleMessage(update.Message)
	}
}

func (b *CSBot) handleMessage(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	if telegramID == b.cfg.MaintenanceAccountID && msg.ReplyToMessage != nil {
		userID, err := b.db.GetCSUserByMessage(msg.ReplyToMessage.MessageID)
		if err == nil {
			b.handleAdminReply(userID, msg)
			return
		}
	}

	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			b.sendMessage(telegramID, "👋 <b>Selamat Datang di Customer Service PNJ Chat Bot!</b>\n\nSilakan tulis kendala atau pertanyaan kamu di sini. Tim kami akan segera membalas pesanmu.")
		default:
			b.sendMessage(telegramID, "❓ Perintah tidak dikenal.")
		}
		return
	}

	b.forwardToAdmin(msg)
}

func (b *CSBot) forwardToAdmin(msg *tgbotapi.Message) {
	if b.cfg.MaintenanceAccountID == 0 {
		b.sendMessage(msg.From.ID, "❌ Maaf, layanan pengaduan sedang tidak aktif (Admin ID missing).")
		return
	}

	b.sendMessage(msg.From.ID, "📨 <b>Pesan kamu telah dikirim ke tim CS.</b>\nMohon tunggu balasan dari kami.")

	text := fmt.Sprintf("📩 <b>TICKET BARU</b>\nDari: <b>%s</b> (%d)\n\nPesan:\n%s",
		msg.From.FirstName, msg.From.ID, msg.Text)

	adminMsg := tgbotapi.NewMessage(b.cfg.MaintenanceAccountID, text)
	adminMsg.ParseMode = "HTML"

	sentMsg, err := b.api.Send(adminMsg)
	if err != nil {
		log.Printf("❌ Failed to forward to admin: %v", err)
		return
	}

	b.db.SaveCSMessage(sentMsg.MessageID, msg.From.ID)
}

func (b *CSBot) handleAdminReply(userID int64, adminMsg *tgbotapi.Message) {
	reply := tgbotapi.NewMessage(userID, fmt.Sprintf("🎧 <b>Balasan dari Customer Service:</b>\n\n%s", adminMsg.Text))
	reply.ParseMode = "HTML"

	_, err := b.api.Send(reply)
	if err != nil {
		b.sendMessage(b.cfg.MaintenanceAccountID, "❌ Gagal mengirim balasan ke user.")
		return
	}

	b.sendMessage(b.cfg.MaintenanceAccountID, "✅ Balasan terkirim ke user.")
}

func (b *CSBot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	b.api.Send(msg)
}
