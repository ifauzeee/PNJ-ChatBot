package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pnj-anonymous-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleCallback(callback *tgbotapi.CallbackQuery) {
	telegramID := callback.From.ID
	data := callback.Data

	defer b.answerCallback(callback.ID, "")

	parts := strings.SplitN(data, ":", 2)
	if len(parts) < 2 {
		return
	}

	category := parts[0]
	value := parts[1]

	switch category {
	case "gender":
		b.handleGenderCallback(telegramID, value, callback)
	case "dept":
		b.handleDeptCallback(telegramID, value, callback)
	case "search":
		b.handleSearchCallback(telegramID, value, callback)
	case "chat":
		b.handleChatActionCallback(telegramID, value, callback)
	case "menu":
		b.handleMenuCallback(telegramID, value, callback)
	case "edit":
		b.handleEditCallback(telegramID, value, callback)
	case "react":
		b.handleReactionCallback(telegramID, value, callback)
	case "whisper":
		b.handleWhisperCallback(telegramID, value, callback)
	}
}

func (b *Bot) handleGenderCallback(telegramID int64, gender string, callback *tgbotapi.CallbackQuery) {
	err := b.profile.SetGender(telegramID, gender)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
		return
	}

	emoji := models.GenderEmoji(models.Gender(gender))

	deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
	b.api.Send(deleteMsg)

	b.sendMessage(telegramID, fmt.Sprintf("✅ Gender dipilih: %s *%s*", emoji, gender), nil)

	kb := DepartmentKeyboard()
	b.sendMessage(telegramID, "🏛️ *Pilih Jurusan Kamu:*\n\nPilih jurusan di bawah ini:", &kb)
}

func (b *Bot) handleDeptCallback(telegramID int64, dept string, callback *tgbotapi.CallbackQuery) {
	err := b.profile.SetDepartment(telegramID, dept)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
		return
	}

	emoji := models.DepartmentEmoji(models.Department(dept))

	deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
	b.api.Send(deleteMsg)

	b.sendMessage(telegramID, fmt.Sprintf("✅ Jurusan dipilih: %s *%s*", emoji, dept), nil)

	user, _ := b.db.GetUser(telegramID)
	if user != nil {
		completeText := fmt.Sprintf(`🎉 *Profil Kamu Sudah Lengkap!*

━━━━━━━━━━━━━━━━━━━
🏷️ Nama Anonim: *%s*
%s Gender: *%s*
%s Jurusan: *%s*
━━━━━━━━━━━━━━━━━━━

🚀 Kamu siap menggunakan PNJ Anonymous Bot!
Pilih menu di bawah untuk memulai:`,
			user.DisplayName,
			models.GenderEmoji(user.Gender), string(user.Gender),
			models.DepartmentEmoji(user.Department), string(user.Department),
		)

		kb := MainMenuKeyboard()
		b.sendMessage(telegramID, completeText, &kb)
	}
}

func (b *Bot) handleSearchCallback(telegramID int64, value string, callback *tgbotapi.CallbackQuery) {

	deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
	b.api.Send(deleteMsg)

	if value == "cancel" {
		b.chat.CancelSearch(telegramID)
		b.sendMessage(telegramID, "❌ Pencarian dibatalkan.", nil)
		return
	}

	preferredDept := ""
	if value != "any" {
		preferredDept = value
	}

	b.startSearch(telegramID, preferredDept)
}

func (b *Bot) handleChatActionCallback(telegramID int64, action string, _ *tgbotapi.CallbackQuery) {
	switch action {
	case "next":
		partnerID, err := b.chat.NextPartner(telegramID)
		if err != nil {
			b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
			return
		}
		if partnerID > 0 {
			b.sendMessage(partnerID, "👋 *Partner kamu telah memutus chat.*\n\nGunakan /search untuk mencari partner baru.", nil)
		}
		b.sendMessage(telegramID, "⏭️ *Mencari partner baru...*", nil)
		b.startSearch(telegramID, "")

	case "stop":
		partnerID, _ := b.chat.StopChat(telegramID)
		if partnerID > 0 {
			b.sendMessage(partnerID, "👋 *Partner kamu telah memutus chat.*\n\nGunakan /search untuk mencari partner baru.", nil)
		}
		b.sendMessage(telegramID, "🛑 *Chat dihentikan.*", nil)

	case "report":
		partnerID, err := b.chat.GetPartner(telegramID)
		if err != nil || partnerID == 0 {
			b.sendMessage(telegramID, "⚠️ Tidak ada partner saat ini.", nil)
			return
		}
		b.db.SetUserState(telegramID, models.StateAwaitingReport, fmt.Sprintf("%d", partnerID))
		b.sendMessage(telegramID, "⚠️ *Laporkan Partner*\n\nTuliskan alasan kamu:\n_Ketik /cancel untuk membatalkan_", nil)

	case "block":
		partnerID, err := b.chat.GetPartner(telegramID)
		if err != nil || partnerID == 0 {
			b.sendMessage(telegramID, "⚠️ Tidak ada partner saat ini.", nil)
			return
		}
		b.profile.BlockUser(telegramID, partnerID)
		b.chat.StopChat(telegramID)
		b.sendMessage(partnerID, "👋 *Partner kamu telah memutus chat.*", nil)
		b.sendMessage(telegramID, "🚫 *Partner telah di-block.*", nil)
	}
}

func (b *Bot) handleMenuCallback(telegramID int64, action string, callback *tgbotapi.CallbackQuery) {

	deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
	b.api.Send(deleteMsg)

	switch action {
	case "main":
		user, _ := b.db.GetUser(telegramID)
		if user != nil {
			b.showMainMenu(telegramID, user)
		}

	case "search":

		if complete, _ := b.auth.IsProfileComplete(telegramID); !complete {
			b.sendMessage(telegramID, "⚠️ Profil belum lengkap. Ketik /start untuk melengkapi.", nil)
			return
		}
		kb := SearchKeyboard()
		b.sendMessage(telegramID, "🔍 *Cari Partner Chat Anonim*\n\nPilih filter pencarian:", &kb)

	case "confess":
		b.db.SetUserState(telegramID, models.StateAwaitingConfess, "")
		b.sendMessage(telegramID, `💬 *Tulis Confession Kamu*

📝 Ketik confession kamu sekarang...
Atau ketik /cancel untuk membatalkan.

⚠️ _Confession akan menampilkan jurusan kamu tapi TIDAK identitas kamu._`, nil)

	case "confessions":
		msg := &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}}
		b.handleConfessions(msg)

	case "whisper":
		kb := WhisperDeptKeyboard()
		b.sendMessage(telegramID, "📢 *Whisper - Pesan Anonim ke Jurusan*\n\n🎯 Pilih jurusan tujuan:", &kb)

	case "profile":
		msg := &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}}
		b.handleProfile(msg)

	case "stats":
		msg := &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}}
		b.handleStats(msg)

	case "edit":
		kb := EditProfileKeyboard()
		b.sendMessage(telegramID, "✏️ *Edit Profil*\n\nApa yang ingin kamu ubah?", &kb)

	case "help":
		msg := &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}}
		b.handleHelp(msg)
	}
}

func (b *Bot) handleEditCallback(telegramID int64, field string, callback *tgbotapi.CallbackQuery) {
	deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
	b.api.Send(deleteMsg)

	switch field {
	case "gender":
		kb := GenderKeyboard()
		b.sendMessage(telegramID, "👤 *Pilih Gender Baru:*", &kb)
		b.db.SetUserState(telegramID, models.StateAwaitingGender, "edit")

	case "department":
		kb := DepartmentKeyboard()
		b.sendMessage(telegramID, "🏛️ *Pilih Jurusan Baru:*", &kb)
		b.db.SetUserState(telegramID, models.StateAwaitingDept, "edit")
	}
}

func (b *Bot) handleReactionCallback(telegramID int64, data string, callback *tgbotapi.CallbackQuery) {

	parts := strings.SplitN(data, ":", 2)
	if len(parts) < 2 {
		return
	}

	confessionID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return
	}

	reaction := parts[1]

	err = b.confession.ReactToConfession(confessionID, telegramID, reaction)
	if err != nil {
		b.answerCallback(callback.ID, "❌ Gagal menambahkan reaksi")
		return
	}

	counts, _ := b.confession.GetReactionCounts(confessionID)
	newKb := ConfessionReactionKeyboard(confessionID, counts)

	editMsg := tgbotapi.NewEditMessageReplyMarkup(
		telegramID,
		callback.Message.MessageID,
		newKb,
	)
	b.api.Send(editMsg)

	b.answerCallback(callback.ID, fmt.Sprintf("Kamu react %s", reaction))
}

func (b *Bot) handleWhisperCallback(telegramID int64, dept string, callback *tgbotapi.CallbackQuery) {
	deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
	b.api.Send(deleteMsg)

	b.db.SetUserState(telegramID, models.StateAwaitingWhisper, dept)

	emoji := models.DepartmentEmoji(models.Department(dept))
	b.sendMessage(telegramID, fmt.Sprintf(`📢 *Whisper ke %s %s*

Tulis pesan anonim kamu untuk mahasiswa %s:

_Ketik /cancel untuk membatalkan_`, emoji, dept, dept), nil)
}
