package bot

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handlePoll(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	args := msg.CommandArguments()

	if args == "" {
		b.sendMessageHTML(telegramID, `<b>🗳️ Cara Membuat Polling Anonim</b>

Ketik: <code>/poll Pertanyaan | Opsi 1 | Opsi 2 | ...</code>

Contoh: <code>/poll Setuju gak harga parkir naik? | Setuju | Tidak Setuju</code>`, nil)
		return
	}

	parts := strings.Split(args, "|")
	if len(parts) < 3 {
		b.sendMessageHTML(telegramID, "⚠️ <b>Format salah.</b> Minimal harus ada pertanyaan dan 2 opsi jawaban.", nil)
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

	if len(options) < 2 {
		b.sendMessageHTML(telegramID, "⚠️ <b>Format salah.</b> Minimal harus ada 2 opsi jawaban yang valid.", nil)
		return
	}

	pollID, err := b.db.CreatePoll(telegramID, question, options)
	if err != nil {
		b.sendMessageHTML(telegramID, "❌ Gagal membuat polling.", nil)
		return
	}

	_ = b.db.IncrementUserKarma(telegramID, 3)
	b.checkAchievements(telegramID)

	b.sendMessageHTML(telegramID, fmt.Sprintf("✅ <b>Polling #%d berhasil dibuat!</b>\nSemua mahasiswa sekarang bisa memberikan suara secara anonim.", pollID), nil)
}

func (b *Bot) handleViewPolls(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	polls, err := b.db.GetLatestPolls(15)
	if err != nil {
		b.sendMessageHTML(telegramID, "❌ Gagal mengambil polling.", nil)
		return
	}

	if len(polls) == 0 {
		b.sendMessageHTML(telegramID, "📋 Belum ada polling aktif. Buat yang pertama dengan /poll!", nil)
		return
	}

	header := "<b>🗳️ Daftar Polling Terbaru</b>\n━━━━━━━━━━━━━━━━━━━\n\n"

	for _, p := range polls {
		count, _ := b.db.GetPollVoteCount(p.ID)
		header += fmt.Sprintf("📊 <b>#%d</b>: %s\n👥 <i>%d Suara</i>\n\n", p.ID, html.EscapeString(p.Question), count)
	}

	header += "━━━━━━━━━━━━━━━━━━━\n<i>Ikut memilih: ketik</i> <code>/vote_poll &lt;id&gt;</code>"

	b.sendMessageHTML(telegramID, header, nil)
}

func (b *Bot) handleVotePoll(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	args := msg.CommandArguments()

	if args == "" {
		b.sendMessageHTML(telegramID, "💡 Cara memilih: <code>/vote_poll &lt;id&gt;</code>", nil)
		return
	}

	pollID, err := strconv.ParseInt(args, 10, 64)
	if err != nil {
		b.sendMessageHTML(telegramID, "⚠️ ID polling harus berupa angka.", nil)
		return
	}

	p, err := b.db.GetPoll(pollID)
	if err != nil || p == nil {
		b.sendMessageHTML(telegramID, "❌ Polling tidak ditemukan.", nil)
		return
	}

	kb := PollVoteKeyboard(p.ID, p.Options)
	text := fmt.Sprintf("🗳️ <b>Polling #%d</b>\n\n<b>Pertanyaan:</b>\n%s\n\n<i>Pilih opsi di bawah untuk memberikan suara secara anonim:</i>", p.ID, html.EscapeString(p.Question))
	b.sendMessageHTML(telegramID, text, &kb)
}
