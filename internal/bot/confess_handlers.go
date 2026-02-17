package bot

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/pnj-anonymous-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleConfess(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	b.db.SetUserState(telegramID, models.StateAwaitingConfess, "")

	b.sendMessage(telegramID, `💬 *Tulis Confession Kamu*

━━━━━━━━━━━━━━━━━━━
Kirim confession anonim yang bisa dibaca semua pengguna.

📝 Ketik confession kamu sekarang...
Atau ketik /cancel untuk membatalkan.

⚠️ _Confession akan menampilkan jurusan kamu tapi TIDAK identitas kamu._`, nil)
}

func (b *Bot) handleConfessionInput(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	if msg.Text == "" {
		b.sendMessage(telegramID, "⚠️ Confession harus berupa teks.", nil)
		return
	}

	if len(msg.Text) < 10 {
		b.sendMessage(telegramID, "⚠️ Confession terlalu pendek. Minimal 10 karakter.", nil)
		return
	}

	if len(msg.Text) > 1000 {
		b.sendMessage(telegramID, "⚠️ Confession terlalu panjang. Maksimal 1000 karakter.", nil)
		return
	}

	confession, err := b.confession.CreateConfession(telegramID, msg.Text)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
		return
	}

	b.db.SetUserState(telegramID, models.StateNone, "")

	b.sendMessage(telegramID, fmt.Sprintf(`✅ *Confession Terkirim!*

📝 Confession #%d berhasil dikirim.
Confession kamu sekarang bisa dilihat semua pengguna melalui /confessions.`, confession.ID), nil)
}

func (b *Bot) handleConfessions(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	confessions, err := b.confession.GetLatestConfessions(10)
	if err != nil {
		b.sendMessage(telegramID, "❌ Gagal mengambil confession.", nil)
		return
	}

	if len(confessions) == 0 {
		b.sendMessage(telegramID, "📋 Belum ada confession. Jadilah yang pertama dengan /confess!", nil)
		return
	}

	header := "<b>📋 Confession Terbaru</b>\n━━━━━━━━━━━━━━━━━━━\n\n"

	for _, c := range confessions {
		emoji := models.DepartmentEmoji(models.Department(c.Department))
		counts, _ := b.confession.GetReactionCounts(c.ID)
		replyCount, _ := b.db.GetConfessionReplyCount(c.ID)

		reactionStr := ""
		for r, count := range counts {
			reactionStr += fmt.Sprintf("%s%d ", r, count)
		}

		replyStr := ""
		if replyCount > 0 {
			replyStr = fmt.Sprintf("💬 %d Replies", replyCount)
		}

		text := fmt.Sprintf(`💬 <b>#%d</b> | %s %s
%s
%s %s
─────────────────
`, c.ID, emoji, html.EscapeString(c.Department), html.EscapeString(c.Content), reactionStr, replyStr)

		header += text
	}

	header += "\n<i>React: ketik</i> /react &lt;id&gt; &lt;emoji&gt;\n<i>Balas: ketik</i> /reply &lt;id&gt; &lt;pesan&gt;\n<i>Lihat: ketik</i> /view_replies &lt;id&gt;"

	b.sendMessageHTML(telegramID, header, nil)
}

func (b *Bot) handleReact(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	args := msg.CommandArguments()

	if args == "" {
		b.sendMessage(telegramID, "💡 Cara menggunakan: `/react <id> <emoji>`\nContoh: `/react 1 ❤️`", nil)
		return
	}

	parts := strings.Fields(args)
	if len(parts) < 2 {
		b.sendMessage(telegramID, "⚠️ Format salah. Contoh: `/react 1 ❤️`", nil)
		return
	}

	confessionID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		b.sendMessage(telegramID, "⚠️ ID confession harus berupa angka.", nil)
		return
	}

	reaction := parts[1]
	err = b.confession.ReactToConfession(confessionID, telegramID, reaction)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("❌ %s", err.Error()), nil)
		return
	}

	b.sendMessage(telegramID, fmt.Sprintf("✅ Berhasil menambahkan reaksi %s ke confession #%d", reaction, confessionID), nil)
}

func (b *Bot) handleReply(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	args := msg.CommandArguments()

	if args == "" {
		b.sendMessage(telegramID, "💡 Cara membalas: `/reply <id> <pesan>`\nContoh: `/reply 1 Semangat ya!`", nil)
		return
	}

	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		b.sendMessage(telegramID, "⚠️ Format salah. Contoh: `/reply 1 Halo!`", nil)
		return
	}

	confessionID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		b.sendMessage(telegramID, "⚠️ ID confession harus berupa angka.", nil)
		return
	}

	content := parts[1]
	err = b.db.CreateConfessionReply(confessionID, telegramID, content)
	if err != nil {
		b.sendMessage(telegramID, "❌ Gagal mengirim balasan.", nil)
		return
	}

	confession, _ := b.db.GetConfession(confessionID)
	if confession != nil && confession.AuthorID != telegramID {
		b.db.IncrementUserKarma(confession.AuthorID, 1)
		b.checkAchievements(confession.AuthorID)
	}

	b.checkAchievements(telegramID)

	b.sendMessage(telegramID, fmt.Sprintf("✅ Berhasil membalas confession #%d", confessionID), nil)
}

func (b *Bot) handleViewReplies(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	args := msg.CommandArguments()

	if args == "" {
		b.sendMessage(telegramID, "💡 Cara melihat: `/view_replies <id>`", nil)
		return
	}

	confessionID, err := strconv.ParseInt(args, 10, 64)
	if err != nil {
		b.sendMessage(telegramID, "⚠️ ID confession harus berupa angka.", nil)
		return
	}

	confession, err := b.db.GetConfession(confessionID)
	if err != nil || confession == nil {
		b.sendMessage(telegramID, "❌ Confession tidak ditemukan.", nil)
		return
	}

	replies, err := b.db.GetConfessionReplies(confessionID)
	if err != nil {
		b.sendMessage(telegramID, "❌ Gagal mengambil balasan.", nil)
		return
	}

	if len(replies) == 0 {
		b.sendMessage(telegramID, fmt.Sprintf("📋 Belum ada balasan untuk confession #%d.", confessionID), nil)
		return
	}

	response := fmt.Sprintf("<b>📋 Balasan Confession #%d</b>\n", confessionID)
	response += fmt.Sprintf("&gt; <i>%s</i>\n", html.EscapeString(confession.Content))
	response += "━━━━━━━━━━━━━━━━━━━\n\n"

	for i, r := range replies {
		response += fmt.Sprintf("<b>%d.</b> %s\n\n", i+1, html.EscapeString(r.Content))
	}

	b.sendMessageHTML(telegramID, response, nil)
}
