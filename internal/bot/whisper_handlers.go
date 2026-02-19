package bot

import (
	"context"
	"fmt"
	"html"

	"github.com/pnj-anonymous-bot/internal/metrics"
	"github.com/pnj-anonymous-bot/internal/models"
	"github.com/pnj-anonymous-bot/internal/validation"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleWhisper(ctx context.Context, msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	kb := WhisperDeptKeyboard()
	b.sendMessage(telegramID, `📢 *Whisper - Pesan Anonim ke Jurusan*

━━━━━━━━━━━━━━━━━━━
Kiriman pesan anonim ke semua mahasiswa di jurusan tertentu!

🎯 Pilih jurusan tujuan:`, &kb)
}

func (b *Bot) handleWhisperInput(ctx context.Context, msg *tgbotapi.Message, targetDept string) {
	telegramID := msg.From.ID

	if msg.Text == "" {
		b.sendMessage(telegramID, "⚠️ Whisper harus berupa teks.", nil)
		return
	}

	text := validation.SanitizeText(msg.Text)
	if errMsg := validation.ValidateText(text, validation.WhisperLimits); errMsg != "" {
		b.sendMessage(telegramID, errMsg, nil)
		return
	}

	content := text
	if b.profanity.IsBad(content) {
		content = b.profanity.Clean(content)
		metrics.ProfanityFiltered.Inc()
		b.sendMessage(telegramID, "⚠️ *Peringatan:* Whisper kamu mengandung kata-kata yang tidak pantas dan telah disensor.", nil)
	}

	targets, err := b.profile.SendWhisper(ctx, telegramID, targetDept, content)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
		return
	}

	logIfErr("set_state_none_after_whisper", b.db.SetUserState(ctx, telegramID, models.StateNone, ""))
	metrics.WhispersTotal.Inc()

	user, _ := b.db.GetUser(ctx, telegramID)
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
