package bot

import (
	"fmt"
	"html"

	"github.com/pnj-anonymous-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleWhisper(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	kb := WhisperDeptKeyboard()
	b.sendMessage(telegramID, `📢 *Whisper - Pesan Anonim ke Jurusan*

━━━━━━━━━━━━━━━━━━━
Kirim pesan anonim ke semua mahasiswa di jurusan tertentu!

🎯 Pilih jurusan tujuan:`, &kb)
}

func (b *Bot) handleWhisperInput(msg *tgbotapi.Message, targetDept string) {
	telegramID := msg.From.ID

	if msg.Text == "" || len(msg.Text) < 5 {
		b.sendMessage(telegramID, "⚠️ Whisper terlalu pendek. Minimal 5 karakter.", nil)
		return
	}

	if len(msg.Text) > 500 {
		b.sendMessage(telegramID, "⚠️ Whisper terlalu panjang. Maksimal 500 karakter.", nil)
		return
	}

	content := msg.Text
	if b.profanity.IsBad(content) {
		content = b.profanity.Clean(content)
		b.sendMessage(telegramID, "⚠️ *Peringatan:* Whisper kamu mengandung kata-kata yang tidak pantas dan telah disensor.", nil)
	}

	targets, err := b.profile.SendWhisper(telegramID, targetDept, content)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
		return
	}

	_ = b.db.SetUserState(telegramID, models.StateNone, "")

	user, _ := b.db.GetUser(telegramID)
	senderDept := ""
	senderGender := ""
	if user != nil {
		senderDept = string(user.Department)
		senderGender = string(user.Gender)
	}

	for _, targetID := range targets {
		whisperMsg := fmt.Sprintf(`📢 *Whisper dari %s %s*

━━━━━━━━━━━━━━━━━━━
%s %s | %s
━━━━━━━━━━━━━━━━━━━
%s

_Pesan anonim untuk jurusan %s_`,
			models.DepartmentEmoji(models.Department(senderDept)), senderDept,
			models.GenderEmoji(models.Gender(senderGender)), senderGender,
			models.DepartmentEmoji(models.Department(senderDept)),
			escapeMarkdown(content),
			targetDept,
		)
		b.sendMessage(targetID, whisperMsg, nil)
	}

	b.sendMessageHTML(telegramID, fmt.Sprintf("✅ <b>Whisper Terkirim!</b>\n\n📤 Dikirim ke <b>%d</b> mahasiswa %s %s",
		len(targets),
		models.DepartmentEmoji(models.Department(targetDept)),
		html.EscapeString(targetDept),
	), nil)
}
