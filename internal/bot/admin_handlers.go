package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pnj-anonymous-bot/internal/logger"
	"github.com/pnj-anonymous-bot/internal/metrics"
	"go.uber.org/zap"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) isAdmin(telegramID int64) bool {
	return telegramID == b.cfg.MaintenanceAccountID
}

func (b *Bot) handleAdminPoll(ctx context.Context, msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	if !b.isAdmin(telegramID) {
		return
	}

	args := msg.CommandArguments()
	if args == "" {
		b.sendMessageHTML(telegramID, `<b>📢 Global Admin Poll</b>

Ketik: <code>/admin_poll Pertanyaan | Opsi 1 | Opsi 2 | ...</code>

Polling ini akan disiarkan ke <b>SELURUH</b> pengguna bot yang terverifikasi.`, nil)
		return
	}

	parts := strings.Split(args, "|")
	if len(parts) < 3 {
		b.sendMessageHTML(telegramID, "⚠️ <b>Format salah.</b> pertanyaan dan minimal 2 opsi.", nil)
		return
	}

	question := strings.TrimSpace(parts[0])
	var options []string
	for i := 1; i < len(parts); i++ {
		opt := strings.TrimSpace(parts[i])
		if opt != "" {
			options = append(options, opt)
		}
	}

	pollID, err := b.db.CreatePoll(ctx, 0, "[GLOBAL] "+question, options)
	if err != nil {
		b.sendMessageHTML(telegramID, "❌ Gagal membuat polling global.", nil)
		return
	}

	b.sendMessageHTML(telegramID, fmt.Sprintf("🚀 <b>Memulai broadcast polling global #%d...</b>", pollID), nil)

	go b.broadcastGlobalPoll(pollID)
}

func (b *Bot) handleBroadcast(ctx context.Context, msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	if !b.isAdmin(telegramID) {
		return
	}

	content := msg.CommandArguments()
	if content == "" {
		b.sendMessageHTML(telegramID, "💡 Cara broadcast: <code>/broadcast [pesan]</code>", nil)
		return
	}

	b.sendMessageHTML(telegramID, "🚀 <b>Memulai broadcast pesan ke seluruh pengguna...</b>", nil)

	go b.broadcastMessage(content)
}

func (b *Bot) broadcastGlobalPoll(pollID int64) {
	b.background.Add(1)
	defer b.background.Done()

	ctx := context.Background()
	start := time.Now()
	defer func() {
		metrics.BroadcastDuration.Observe(time.Since(start).Seconds())
	}()

	p, err := b.db.GetPoll(ctx, pollID)
	if err != nil {
		logger.Error("Failed to get poll for broadcast", zap.Int64("poll_id", pollID), zap.Error(err))
		return
	}

	users, err := b.db.GetAllVerifiedUsers(ctx)
	if err != nil {
		logger.Error("Failed to get users for broadcast", zap.Error(err))
		return
	}

	kb := PollVoteKeyboard(p.ID, p.Options)
	text := fmt.Sprintf("📢 <b>PENGUMUMAN & POLLING GLOBAL</b> 🗳️\n\n%s\n\n<i>Klik di bawah untuk memberikan suara kamu:</i>", p.Question)

	success := 0
	failed := 0

	for _, userID := range users {
		msg := tgbotapi.NewMessage(userID, text)
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = kb
		_, err := b.api.Send(msg)
		if err != nil {
			failed++
		} else {
			success++
		}
		time.Sleep(50 * time.Millisecond)
	}

	logger.Info("Global poll broadcast finished",
		zap.Int64("poll_id", pollID),
		zap.Int("success", success),
		zap.Int("failed", failed),
	)

	b.sendMessageHTML(b.cfg.MaintenanceAccountID, fmt.Sprintf("✅ <b>Broadcast Selesai!</b>\n\nBerhasil: %d\nGagal: %d", success, failed), nil)
}

func (b *Bot) broadcastMessage(content string) {
	b.background.Add(1)
	defer b.background.Done()

	ctx := context.Background()
	start := time.Now()
	defer func() {
		metrics.BroadcastDuration.Observe(time.Since(start).Seconds())
	}()

	users, err := b.db.GetAllVerifiedUsers(ctx)
	if err != nil {
		logger.Error("Failed to get users for broadcast", zap.Error(err))
		return
	}

	text := fmt.Sprintf("📢 <b>PENGUMUMAN GLOBAL</b>\n\n%s", content)

	success := 0
	failed := 0

	for _, userID := range users {
		msg := tgbotapi.NewMessage(userID, text)
		msg.ParseMode = "HTML"
		_, err := b.api.Send(msg)
		if err != nil {
			failed++
		} else {
			success++
		}
		time.Sleep(50 * time.Millisecond)
	}

	b.sendMessageHTML(b.cfg.MaintenanceAccountID, fmt.Sprintf("✅ <b>Broadcast Pesan Selesai!</b>\n\nBerhasil: %d\nGagal: %d", success, failed), nil)
}
