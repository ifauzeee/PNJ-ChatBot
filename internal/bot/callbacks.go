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

	if handler, exists := b.callbacks[category]; exists {
		handler(telegramID, value, callback)
	}
}

func (b *Bot) handleGenderCallback(telegramID int64, gender string, callback *tgbotapi.CallbackQuery) {
	deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
	_, _ = b.api.Send(deleteMsg)

	_, stateData, _ := b.db.GetUserState(telegramID)
	if stateData == "edit" {
		if err := b.profile.UpdateGender(telegramID, gender); err != nil {
			b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
			return
		}
		_ = b.db.SetUserState(telegramID, models.StateNone, "")
		b.sendMessage(telegramID, "✅ Profil berhasil diperbarui!", nil)
		return
	}

	err := b.profile.SetGender(telegramID, gender)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
		return
	}

	emoji := models.GenderEmoji(models.Gender(gender))
	b.sendMessage(telegramID, fmt.Sprintf("✅ Gender dipilih: %s *%s*", emoji, gender), nil)

	kb := YearKeyboard()
	b.sendMessage(telegramID, "🎓 *Pilih Tahun Angkatan (Masuk) Kamu:*", &kb)
}
func (b *Bot) handleDeptCallback(telegramID int64, dept string, callback *tgbotapi.CallbackQuery) {
	deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
	_, _ = b.api.Send(deleteMsg)

	_, stateData, _ := b.db.GetUserState(telegramID)
	if stateData == "edit" {
		if err := b.profile.UpdateDepartment(telegramID, dept); err != nil {
			b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
			return
		}
		_ = b.db.SetUserState(telegramID, models.StateNone, "")
		b.sendMessage(telegramID, "✅ Profil berhasil diperbarui!", nil)
		return
	}

	err := b.profile.SetDepartment(telegramID, dept)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
		return
	}

	emoji := models.DepartmentEmoji(models.Department(dept))
	b.sendMessage(telegramID, fmt.Sprintf("✅ Jurusan dipilih: %s *%s*", emoji, dept), nil)

	user, err := b.db.GetUser(telegramID)
	if err == nil && user != nil && user.Year == 0 {
		_ = b.db.SetUserState(telegramID, models.StateAwaitingYear, "")
		kb := YearKeyboard()
		b.sendMessage(telegramID, "🎓 *Pilih Tahun Angkatan (Masuk) Kamu:*", &kb)
		return
	}

	if user != nil {
		b.showMainMenu(telegramID, user)
	}
}
func (b *Bot) handleSearchCallback(telegramID int64, value string, callback *tgbotapi.CallbackQuery) {
	if value == "cancel" {
		deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
		_, _ = b.api.Send(deleteMsg)
		_ = b.chat.CancelSearch(telegramID)
		b.sendMessage(telegramID, "❌ Pencarian dibatalkan.", nil)
		return
	}

	if value == "any" {
		deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
		_, _ = b.api.Send(deleteMsg)
		b.startSearch(telegramID, "", "", 0)
		return
	}

	if value == "by_gender" {
		kb := SearchGenderKeyboard()
		editMsg := tgbotapi.NewEditMessageText(telegramID, callback.Message.MessageID, "👫 *Pilih Gender Partner:*")
		editMsg.ParseMode = "Markdown"
		editMsg.ReplyMarkup = &kb
		_, _ = b.api.Send(editMsg)
		return
	}

	if value == "by_dept" {
		kb := SearchDepartmentKeyboard()
		editMsg := tgbotapi.NewEditMessageText(telegramID, callback.Message.MessageID, "🏛️ *Pilih Jurusan Partner:*")
		editMsg.ParseMode = "Markdown"
		editMsg.ReplyMarkup = &kb
		_, _ = b.api.Send(editMsg)
		return
	}

	if value == "by_year" {
		kb := SearchYearKeyboard()
		editMsg := tgbotapi.NewEditMessageText(telegramID, callback.Message.MessageID, "🎓 *Pilih Angkatan Partner:*")
		editMsg.ParseMode = "Markdown"
		editMsg.ReplyMarkup = &kb
		_, _ = b.api.Send(editMsg)
		return
	}

	if strings.HasPrefix(value, "year:") {
		deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
		_, _ = b.api.Send(deleteMsg)
		yearStr := strings.TrimPrefix(value, "year:")
		year, _ := strconv.Atoi(yearStr)
		b.startSearch(telegramID, "", "", year)
		return
	}

	if strings.HasPrefix(value, "gender:") {
		deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
		_, _ = b.api.Send(deleteMsg)
		gender := strings.TrimPrefix(value, "gender:")
		b.startSearch(telegramID, "", gender, 0)
		return
	}

	if strings.HasPrefix(value, "dept:") {
		deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
		_, _ = b.api.Send(deleteMsg)
		dept := strings.TrimPrefix(value, "dept:")
		b.startSearch(telegramID, dept, "", 0)
		return
	}

	deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
	_, _ = b.api.Send(deleteMsg)
	b.startSearch(telegramID, value, "", 0)
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
		b.startSearch(telegramID, "", "", 0)

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
		_ = b.db.SetUserState(telegramID, models.StateAwaitingReport, fmt.Sprintf("%d", partnerID))
		b.sendMessage(telegramID, "⚠️ *Laporkan Partner*\n\nTuliskan alasan kamu:\n_Ketik /cancel untuk membatalkan_", nil)

	case "block":
		partnerID, err := b.chat.GetPartner(telegramID)
		if err != nil || partnerID == 0 {
			b.sendMessage(telegramID, "⚠️ Tidak ada partner saat ini.", nil)
			return
		}
		_ = b.profile.BlockUser(telegramID, partnerID)
		_, _ = b.chat.StopChat(telegramID)
		b.sendMessage(partnerID, "👋 *Partner kamu telah memutus chat.*", nil)
		b.sendMessage(telegramID, "🚫 *Partner telah di-block.*", nil)
	}
}

func (b *Bot) handleMenuCallback(telegramID int64, action string, callback *tgbotapi.CallbackQuery) {

	deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
	_, _ = b.api.Send(deleteMsg)

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
		_ = b.db.SetUserState(telegramID, models.StateAwaitingConfess, "")
		b.sendMessage(telegramID, `💬 *Tulis Confession Kamu*

📝 Ketik confession kamu sekarang...
Atau ketik /cancel untuk membatalkan.

⚠️ _Confession akan menampilkan jurusan kamu tapi TIDAK identitas kamu._`, nil)

	case "confessions":
		msg := &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}}
		b.handleConfessions(msg)

	case "polls":
		msg := &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}}
		b.handleViewPolls(msg)

	case "whisper":
		kb := WhisperDeptKeyboard()
		b.sendMessage(telegramID, "📢 *Whisper - Pesan Anonim ke Jurusan*\n\n🎯 Pilih jurusan tujuan:", &kb)

	case "circles":
		b.handleCircles(&tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}})

	case "profile":
		msg := &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}}
		b.handleProfile(msg)

	case "stats":
		msg := &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}}
		b.handleStats(msg)

	case "leaderboard":
		msg := &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}}
		b.handleLeaderboard(msg)

	case "edit":
		kb := EditProfileKeyboard()
		b.sendMessage(telegramID, "✏️ *Edit Profil*\n\nApa yang ingin kamu ubah?", &kb)

	case "help":
		msg := &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}}
		b.handleHelp(msg)

	case "about":
		msg := &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}}
		b.handleAbout(msg)
	}
}

func (b *Bot) handleEditCallback(telegramID int64, field string, callback *tgbotapi.CallbackQuery) {
	deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
	_, _ = b.api.Send(deleteMsg)

	switch field {
	case "gender":
		kb := GenderKeyboard()
		b.sendMessage(telegramID, "👤 *Pilih Gender Baru:*", &kb)
		_ = b.db.SetUserState(telegramID, models.StateAwaitingGender, "edit")

	case "year":
		kb := YearKeyboard()
		b.sendMessage(telegramID, "🎓 *Pilih Tahun Angkatan Baru:*", &kb)
		_ = b.db.SetUserState(telegramID, models.StateAwaitingYear, "edit")

	case "department":
		kb := DepartmentKeyboard()
		b.sendMessage(telegramID, "🏛️ *Pilih Jurusan Baru:*", &kb)
		_ = b.db.SetUserState(telegramID, models.StateAwaitingDept, "edit")
	}
}
func (b *Bot) handleYearCallback(telegramID int64, value string, callback *tgbotapi.CallbackQuery) {
	year, err := strconv.Atoi(value)
	if err != nil {
		b.sendMessage(telegramID, "⚠️ Tahun angkatan tidak valid.", nil)
		return
	}

	deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
	_, _ = b.api.Send(deleteMsg)

	_, stateData, _ := b.db.GetUserState(telegramID)
	if stateData == "edit" {
		err := b.profile.UpdateYear(telegramID, year)
		if err != nil {
			b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
			return
		}
		_ = b.db.SetUserState(telegramID, models.StateNone, "")
		b.sendMessage(telegramID, "✅ Profil berhasil diperbarui!", nil)
		return
	}

	err = b.profile.SetYear(telegramID, year)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
		return
	}

	b.sendMessage(telegramID, fmt.Sprintf("✅ Angkatan dipilih: *%d*", year), nil)

	user, _ := b.db.GetUser(telegramID)
	if user != nil && string(user.Department) == "" {
		kb := DepartmentKeyboard()
		b.sendMessage(telegramID, "🏛️ *Pilih Jurusan Kamu:*\n\nPilih jurusan di bawah ini:", &kb)
		return
	}

	if user != nil {
		b.showMainMenu(telegramID, user)
	}
}
func (b *Bot) handleVoteCallback(telegramID int64, value string, callback *tgbotapi.CallbackQuery) {
	parts := strings.Split(value, ":")
	if len(parts) < 2 {
		return
	}

	pollID, _ := strconv.ParseInt(parts[0], 10, 64)
	optionID, _ := strconv.ParseInt(parts[1], 10, 64)

	err := b.db.VotePoll(pollID, telegramID, optionID)
	if err != nil {
		b.answerCallback(callback.ID, "⚠️ "+err.Error())
		return
	}

	poll, err := b.db.GetPoll(pollID)
	if err == nil && poll != nil {
		kb := PollVoteKeyboard(poll.ID, poll.Options)
		editMsg := tgbotapi.NewEditMessageReplyMarkup(telegramID, callback.Message.MessageID, kb)
		_, _ = b.api.Send(editMsg)
	}

	b.answerCallback(callback.ID, "✅ Suara kamu berhasil direkam!")
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

	confession, _ := b.db.GetConfession(confessionID)
	if confession != nil {
		b.checkAchievements(confession.AuthorID)
	}
	b.checkAchievements(telegramID)
	b.processReward(telegramID, "reaction_given")

	counts, _ := b.confession.GetReactionCounts(confessionID)
	newKb := ConfessionReactionKeyboard(confessionID, counts)

	editMsg := tgbotapi.NewEditMessageReplyMarkup(
		telegramID,
		callback.Message.MessageID,
		newKb,
	)
	_, _ = b.api.Send(editMsg)

	b.answerCallback(callback.ID, fmt.Sprintf("Kamu react %s", reaction))
}

func (b *Bot) handleWhisperCallback(telegramID int64, dept string, callback *tgbotapi.CallbackQuery) {
	deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
	_, _ = b.api.Send(deleteMsg)

	_ = b.db.SetUserState(telegramID, models.StateAwaitingWhisper, dept)

	emoji := models.DepartmentEmoji(models.Department(dept))
	b.sendMessage(telegramID, fmt.Sprintf(`📢 *Whisper ke %s %s*

Tulis pesan anonim kamu untuk mahasiswa %s:

_Ketik /cancel untuk membatalkan_`, emoji, dept, dept), nil)
}

func (b *Bot) handleLegalCallback(telegramID int64, value string, callback *tgbotapi.CallbackQuery) {
	if value == "agree" {
		deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
		_, _ = b.api.Send(deleteMsg)

		b.startEmailVerif(telegramID)
	}
}

func (b *Bot) handleCircleCallback(telegramID int64, data string, callback *tgbotapi.CallbackQuery) {
	parts := strings.SplitN(data, ":", 2)
	action := parts[0]

	switch action {
	case "join":
		if len(parts) < 2 {
			return
		}
		slug := parts[1]

		state, _, _ := b.db.GetUserState(telegramID)
		if state == models.StateInChat {
			kb := ConfirmKeyboard(fmt.Sprintf("circle:confirm_join:%s", slug), "circle:stay_chat")
			b.sendMessageHTML(telegramID, `⚠️ <b>Kamu sedang dalam Private Chat aktif</b>

Bergabung ke circle akan mengakhiri chat kamu saat ini secara otomatis. Apakah kamu yakin?`, &kb)
			return
		}

		room, err := b.room.JoinRoom(telegramID, slug)
		if err != nil {
			b.answerCallback(callback.ID, "❌ "+err.Error())
			return
		}

		deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
		_, _ = b.api.Send(deleteMsg)

		kb := LeaveCircleKeyboard()
		text := fmt.Sprintf(`🎉 <b>Berhasil Terhubung ke Circle %s</b>

━━━━━━━━━━━━━━━━━━━
Sekarang semua pesan yang kamu ketik akan dikirim ke semua anggota circle ini secara anonim.

💡 Gunakan /leave_circle atau klik tombol di bawah untuk keluar.

<i>Mulai ngobrol sekarang...</i>`, room.Name)

		b.sendMessageHTML(telegramID, text, &kb)

	case "create":
		deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
		_, _ = b.api.Send(deleteMsg)

		_ = b.db.SetUserState(telegramID, models.StateAwaitingRoomName, "")
		b.sendMessage(telegramID, "➕ *Buat Circle Baru*\n\nTuliskan *Nama Circle* yang ingin kamu buat:\n(Contoh: Pejuang Kopi PNJ)\n\n_Ketik /cancel untuk membatalkan_", nil)

	case "confirm_join":
		if len(parts) < 2 {
			return
		}
		slug := parts[1]

		deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
		_, _ = b.api.Send(deleteMsg)

		partnerID, _ := b.chat.StopChat(telegramID)
		if partnerID > 0 {
			b.sendMessage(partnerID, "👋 *Partner kamu telah memutus chat.*", nil)
		}

		room, err := b.room.JoinRoom(telegramID, slug)
		if err != nil {
			b.sendMessage(telegramID, "❌ "+err.Error(), nil)
			return
		}

		kb := LeaveCircleKeyboard()
		text := fmt.Sprintf(`🎉 <b>Berhasil Terhubung ke Circle %s</b>

━━━━━━━━━━━━━━━━━━━
Pesan kamu sekarang dikirim ke circle ini. Private chat sebelumnya telah dihentikan.

<i>Mulai ngobrol sekarang...</i>`, room.Name)
		b.sendMessageHTML(telegramID, text, &kb)

	case "stay_chat":
		deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
		_, _ = b.api.Send(deleteMsg)
		b.answerCallback(callback.ID, "👌 Oke, private chat dilanjutkan.")

	case "leave":
		deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
		_, _ = b.api.Send(deleteMsg)
		b.handleLeaveCircle(&tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}})

	case "leave_next":
		deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
		_, _ = b.api.Send(deleteMsg)

		_ = b.room.LeaveRoom(telegramID)
		b.sendMessageHTML(telegramID, "👋 <b>Kamu telah keluar dari circle.</b>\n⏭️ <i>Mencari partner baru...</i>", nil)

		b.startSearch(telegramID, "", "", 0)

	case "stay":
		deleteMsg := tgbotapi.NewDeleteMessage(telegramID, callback.Message.MessageID)
		_, _ = b.api.Send(deleteMsg)
		b.answerCallback(callback.ID, "👌 Oke, kamu tetap di circle.")
	}
}
