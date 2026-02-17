package csbot

import (
	"fmt"
	"log"
	"time"

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
	log.Printf("🛠️ CS Bot authorized as @%s (Admin ID: %d)", b.api.Self.UserName, b.cfg.MaintenanceAccountID)

	go b.startTimeoutWorker()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		log.Printf("📥 CS Update from %d: %s", update.Message.From.ID, update.Message.Text)
		b.handleMessage(update.Message)
	}
}

func (b *CSBot) startTimeoutWorker() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		timedOutUsers, err := b.db.GetTimedOutCSSessions(5)
		if err != nil {
			continue
		}

		for _, userID := range timedOutUsers {
			b.endSession(userID, "⏰ <b>Sesi berakhir.</b> Tidak ada aktivitas selama 5 menit.")
			b.processQueue()
		}
	}
}

func (b *CSBot) handleMessage(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	if telegramID == b.cfg.MaintenanceAccountID {
		userID, _ := b.db.GetActiveCSSessionByAdmin(telegramID)
		if userID > 0 {
			b.db.UpdateCSSessionActivity(userID)
			if msg.IsCommand() && (msg.Command() == "stop" || msg.Command() == "end") {
				b.handleStop(userID)
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

	adminID, _ := b.db.GetActiveCSSessionByUser(telegramID)
	if adminID > 0 {
		b.db.UpdateCSSessionActivity(telegramID)
		if msg.IsCommand() && msg.Command() == "stop" {
			b.handleStop(telegramID)
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
			b.handleChat(telegramID)
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

func (b *CSBot) handleChat(telegramID int64) {
	activeUserID, _ := b.db.GetActiveCSSessionByAdmin(b.cfg.MaintenanceAccountID)
	if activeUserID == 0 {
		b.startSession(telegramID)
	} else {
		b.db.JoinCSQueue(telegramID)
		pos, _ := b.db.GetCSQueuePosition(telegramID)
		b.sendMessage(telegramID, fmt.Sprintf("⏳ <b>Agen sedang melayani pengguna lain.</b>\n\nKamu telah masuk ke dalam antrian. Posisi kamu saat ini: <b>#%d</b>.\nMohon tunggu sebentar, kami akan memberitahumu secara otomatis jika sudah terhubung.", pos))
	}
}

func (b *CSBot) handleStop(userID int64) {
	b.endSession(userID, "⏹️ <b>Sesi chat telah diakhiri.</b> Terima kasih telah menghubungi kami.")
	b.processQueue()
}

func (b *CSBot) startSession(userID int64) {
	log.Printf("🚀 Starting CS session: User %d with Admin %d", userID, b.cfg.MaintenanceAccountID)
	b.db.LeaveCSQueue(userID)
	err := b.db.CreateCSSession(userID, b.cfg.MaintenanceAccountID)
	if err != nil {
		log.Printf("❌ Error creating CS session: %v", err)
		return
	}

	b.sendMessage(userID, "🎧 <b>Terhubung dengan agen!</b>\nSilakan sampaikan pertanyaan atau kendala kamu.")
	b.sendMessage(b.cfg.MaintenanceAccountID, fmt.Sprintf("📩 <b>SESSION BARU</b>\nUser: %d\n\nSilakan balas pesan untuk memulai percakapan.", userID))
}

func (b *CSBot) endSession(userID int64, message string) {
	b.db.EndCSSession(userID)
	b.sendMessage(userID, message)
	b.sendMessage(b.cfg.MaintenanceAccountID, fmt.Sprintf("🛑 <b>Sesi dengan user %d berakhir.</b>", userID))
}

func (b *CSBot) processQueue() {
	nextUserID, err := b.db.GetNextInCSQueue()
	if err == nil && nextUserID > 0 {
		b.startSession(nextUserID)
	}
}

func (b *CSBot) forwardToAdmin(userID, adminID int64, msg *tgbotapi.Message) {
	log.Printf("📲 Forwarding msg from %d to admin %d", userID, adminID)
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
		log.Printf("❌ Error sending message to %d: %v", chatID, err)
	}
}
