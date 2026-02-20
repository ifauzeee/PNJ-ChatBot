package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pnj-anonymous-bot/internal/metrics"
	"github.com/pnj-anonymous-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	telegramID := callback.From.ID
	data := callback.Data

	parts := strings.SplitN(data, ":", 2)
	if len(parts) < 2 {
		return
	}

	category := parts[0]
	value := parts[1]

	if handler, exists := b.callbacks[category]; exists {
		metrics.CallbacksTotal.WithLabelValues(category).Inc()
		handler(ctx, telegramID, value, callback)
	} else {
		b.answerCallback(callback.ID, "")
	}
}

func (b *Bot) handleGenderCallback(ctx context.Context, telegramID int64, gender string, callback *tgbotapi.CallbackQuery) {
	b.deleteMessage(telegramID, callback.Message.MessageID, "delete_gender_callback")

	_, stateData, _ := b.db.GetUserState(ctx, telegramID)
	if stateData == "edit" {
		if err := b.profile.UpdateGender(ctx, telegramID, gender); err != nil {
			b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
			return
		}
		logIfErr("set_state_none_edit_gender", b.db.SetUserState(ctx, telegramID, models.StateNone, ""))
		b.sendMessage(telegramID, "✅ Profil berhasil diperbarui!", nil)
		return
	}

	err := b.profile.SetGender(ctx, telegramID, gender)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
		return
	}

	emoji := models.GenderEmoji(models.Gender(gender))
	b.sendMessage(telegramID, fmt.Sprintf("✅ Gender dipilih: %s *%s*", emoji, gender), nil)

	kb := YearKeyboard()
	b.sendMessage(telegramID, "🎓 *Pilih Tahun Angkatan (Masuk) Kamu:*", &kb)
}

func (b *Bot) handleDeptCallback(ctx context.Context, telegramID int64, dept string, callback *tgbotapi.CallbackQuery) {
	b.deleteMessage(telegramID, callback.Message.MessageID, "delete_dept_callback")

	_, stateData, _ := b.db.GetUserState(ctx, telegramID)
	if stateData == "edit" {
		if err := b.profile.UpdateDepartment(ctx, telegramID, dept); err != nil {
			b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
			return
		}
		logIfErr("set_state_none_edit_dept", b.db.SetUserState(ctx, telegramID, models.StateNone, ""))
		b.sendMessage(telegramID, "✅ Profil berhasil diperbarui!", nil)
		return
	}

	err := b.profile.SetDepartment(ctx, telegramID, dept)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
		return
	}

	emoji := models.DepartmentEmoji(models.Department(dept))
	b.sendMessage(telegramID, fmt.Sprintf("✅ Jurusan dipilih: %s *%s*", emoji, dept), nil)

	user, err := b.db.GetUser(ctx, telegramID)
	if err == nil && user != nil && user.Year == 0 {
		logIfErr("set_state_awaiting_year", b.db.SetUserState(ctx, telegramID, models.StateAwaitingYear, ""))
		kb := YearKeyboard()
		b.sendMessage(telegramID, "🎓 *Pilih Tahun Angkatan (Masuk) Kamu:*", &kb)
		return
	}

	if user != nil {
		b.showMainMenu(ctx, telegramID, user)
	}
}

func (b *Bot) handleSearchCallback(ctx context.Context, telegramID int64, value string, callback *tgbotapi.CallbackQuery) {
	if value == "cancel" {
		b.deleteMessage(telegramID, callback.Message.MessageID, "delete_search_cancel")
		logIfErr("cancel_search_callback", b.chat.CancelSearch(ctx, telegramID))
		b.sendMessage(telegramID, "❌ Pencarian dibatalkan.", nil)
		return
	}

	if value == "any" {
		b.deleteMessage(telegramID, callback.Message.MessageID, "delete_search_any")
		b.startSearch(ctx, telegramID, "", "", 0)
		return
	}

	if value == "by_gender" {
		kb := SearchGenderKeyboard()
		editMsg := tgbotapi.NewEditMessageText(telegramID, callback.Message.MessageID, "👫 *Pilih Gender Partner:*")
		editMsg.ParseMode = "Markdown"
		editMsg.ReplyMarkup = &kb
		b.sendAPI("edit_search_gender", editMsg)
		return
	}

	if value == "by_dept" {
		kb := SearchDepartmentKeyboard()
		editMsg := tgbotapi.NewEditMessageText(telegramID, callback.Message.MessageID, "🏛️ *Pilih Jurusan Partner:*")
		editMsg.ParseMode = "Markdown"
		editMsg.ReplyMarkup = &kb
		b.sendAPI("edit_search_dept", editMsg)
		return
	}

	if value == "by_year" {
		kb := SearchYearKeyboard()
		editMsg := tgbotapi.NewEditMessageText(telegramID, callback.Message.MessageID, "🎓 *Pilih Angkatan Partner:*")
		editMsg.ParseMode = "Markdown"
		editMsg.ReplyMarkup = &kb
		b.sendAPI("edit_search_year", editMsg)
		return
	}

	if strings.HasPrefix(value, "year:") {
		b.deleteMessage(telegramID, callback.Message.MessageID, "delete_search_year")
		yearStr := strings.TrimPrefix(value, "year:")
		year, _ := strconv.Atoi(yearStr)
		b.startSearch(ctx, telegramID, "", "", year)
		return
	}

	if strings.HasPrefix(value, "gender:") {
		b.deleteMessage(telegramID, callback.Message.MessageID, "delete_search_gender")
		gender := strings.TrimPrefix(value, "gender:")
		b.startSearch(ctx, telegramID, "", gender, 0)
		return
	}

	if strings.HasPrefix(value, "dept:") {
		b.deleteMessage(telegramID, callback.Message.MessageID, "delete_search_dept_filter")
		dept := strings.TrimPrefix(value, "dept:")
		b.startSearch(ctx, telegramID, dept, "", 0)
		return
	}

	b.deleteMessage(telegramID, callback.Message.MessageID, "delete_search_default")
	b.startSearch(ctx, telegramID, value, "", 0)
}

func (b *Bot) handleChatActionCallback(ctx context.Context, telegramID int64, action string, _ *tgbotapi.CallbackQuery) {
	switch action {
	case "next":
		partnerID, err := b.chat.NextPartner(ctx, telegramID)
		if err != nil {
			b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
			return
		}
		if partnerID > 0 {
			b.sendMessage(partnerID, "👋 *Partner kamu telah memutus chat.*\n\nGunakan /search untuk mencari partner baru.", nil)
		}
		b.sendMessage(telegramID, "⏭️ *Mencari partner baru...*", nil)
		b.startSearch(ctx, telegramID, "", "", 0)

	case "stop":
		partnerID, _ := b.chat.StopChat(ctx, telegramID)
		if partnerID > 0 {
			b.sendMessage(partnerID, "👋 *Partner kamu telah memutus chat.*\n\nGunakan /search untuk mencari partner baru.", nil)
		}
		b.sendMessage(telegramID, "🛑 *Chat dihentikan.*", nil)

	case "report":
		partnerID, err := b.chat.GetPartner(ctx, telegramID)
		if err != nil || partnerID == 0 {
			b.sendMessage(telegramID, "⚠️ Tidak ada partner saat ini.", nil)
			return
		}
		logIfErr("set_state_report_callback", b.db.SetUserState(ctx, telegramID, models.StateAwaitingReport, fmt.Sprintf("%d", partnerID)))
		b.sendMessage(telegramID, "⚠️ *Laporkan Partner*\n\nTuliskan alasan kamu:\n_Ketik /cancel untuk membatalkan_", nil)

	case "block":
		partnerID, err := b.chat.GetPartner(ctx, telegramID)
		if err != nil || partnerID == 0 {
			b.sendMessage(telegramID, "⚠️ Tidak ada partner saat ini.", nil)
			return
		}
		logIfErr("block_user_callback", b.profile.BlockUser(ctx, telegramID, partnerID))
		if _, err := b.chat.StopChat(ctx, telegramID); err != nil {
			logIfErr("stop_chat_after_block_callback", err)
		}
		b.sendMessage(partnerID, "👋 *Partner kamu telah memutus chat.*", nil)
		b.sendMessage(telegramID, "🚫 *Partner telah di-block.*", nil)
	}
}

func (b *Bot) handleMenuCallback(ctx context.Context, telegramID int64, action string, callback *tgbotapi.CallbackQuery) {
	b.deleteMessage(telegramID, callback.Message.MessageID, "delete_menu_callback")

	switch action {
	case "main":
		user, _ := b.db.GetUser(ctx, telegramID)
		if user != nil {
			b.showMainMenu(ctx, telegramID, user)
		}

	case "search":
		if complete, _ := b.auth.IsProfileComplete(ctx, telegramID); !complete {
			b.sendMessage(telegramID, "⚠️ Profil belum lengkap. Ketik /start untuk melengkapi.", nil)
			return
		}
		kb := SearchKeyboard()
		b.sendMessage(telegramID, "🔍 *Cari Partner Chat Anonim*\n\nPilih filter pencarian:", &kb)

	case "confess":
		logIfErr("set_state_confess_menu", b.db.SetUserState(ctx, telegramID, models.StateAwaitingConfess, ""))
		b.sendMessage(telegramID, `💬 *Tulis Confession Kamu*

📝 Ketik confession kamu sekarang...
Atau ketik /cancel untuk membatalkan.

⚠️ _Confession akan menampilkan jurusan kamu tapi TIDAK identitas kamu._`, nil)

	case "confessions":
		msg := &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}}
		b.handleConfessions(ctx, msg)

	case "polls":
		msg := &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}}
		b.handleViewPolls(ctx, msg)

	case "whisper":
		kb := WhisperDeptKeyboard()
		b.sendMessage(telegramID, "📢 *Whisper - Pesan Anonim ke Jurusan*\n\n🎯 Pilih jurusan tujuan:", &kb)

	case "circles":
		b.handleCircles(ctx, &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}})

	case "profile":
		msg := &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}}
		b.handleProfile(ctx, msg)

	case "stats":
		msg := &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}}
		b.handleStats(ctx, msg)

	case "leaderboard":
		msg := &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}}
		b.handleLeaderboard(ctx, msg)

	case "edit":
		kb := EditProfileKeyboard()
		b.sendMessage(telegramID, "✏️ *Edit Profil*\n\nApa yang ingin kamu ubah?", &kb)

	case "help":
		msg := &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}}
		b.handleHelp(ctx, msg)

	case "about":
		msg := &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}}
		b.handleAbout(ctx, msg)
	}
}

func (b *Bot) handleEditCallback(ctx context.Context, telegramID int64, field string, callback *tgbotapi.CallbackQuery) {
	b.deleteMessage(telegramID, callback.Message.MessageID, "delete_edit_callback")

	switch field {
	case "gender":
		kb := GenderKeyboard()
		b.sendMessage(telegramID, "👤 *Pilih Gender Baru:*", &kb)
		logIfErr("set_state_edit_gender", b.db.SetUserState(ctx, telegramID, models.StateAwaitingGender, "edit"))

	case "year":
		kb := YearKeyboard()
		b.sendMessage(telegramID, "🎓 *Pilih Tahun Angkatan Baru:*", &kb)
		logIfErr("set_state_edit_year", b.db.SetUserState(ctx, telegramID, models.StateAwaitingYear, "edit"))

	case "department":
		kb := DepartmentKeyboard()
		b.sendMessage(telegramID, "🏛️ *Pilih Jurusan Baru:*", &kb)
		logIfErr("set_state_edit_dept", b.db.SetUserState(ctx, telegramID, models.StateAwaitingDept, "edit"))
	}
}

func (b *Bot) handleYearCallback(ctx context.Context, telegramID int64, value string, callback *tgbotapi.CallbackQuery) {
	year, err := strconv.Atoi(value)
	if err != nil {
		b.sendMessage(telegramID, "⚠️ Tahun angkatan tidak valid.", nil)
		return
	}

	b.deleteMessage(telegramID, callback.Message.MessageID, "delete_year_callback")

	_, stateData, _ := b.db.GetUserState(ctx, telegramID)
	if stateData == "edit" {
		err := b.profile.UpdateYear(ctx, telegramID, year)
		if err != nil {
			b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
			return
		}
		logIfErr("set_state_none_edit_year", b.db.SetUserState(ctx, telegramID, models.StateNone, ""))
		b.sendMessage(telegramID, "✅ Profil berhasil diperbarui!", nil)
		return
	}

	err = b.profile.SetYear(ctx, telegramID, year)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
		return
	}

	b.sendMessage(telegramID, fmt.Sprintf("✅ Angkatan dipilih: *%d*", year), nil)

	user, _ := b.db.GetUser(ctx, telegramID)
	if user != nil && string(user.Department) == "" {
		kb := DepartmentKeyboard()
		b.sendMessage(telegramID, "🏛️ *Pilih Jurusan Kamu:*\n\nPilih jurusan di bawah ini:", &kb)
		return
	}

	if user != nil {
		b.showMainMenu(ctx, telegramID, user)
	}
}

func (b *Bot) handleVoteCallback(ctx context.Context, telegramID int64, value string, callback *tgbotapi.CallbackQuery) {
	parts := strings.Split(value, ":")
	if len(parts) < 2 {
		return
	}

	pollID, _ := strconv.ParseInt(parts[0], 10, 64)
	optionID, _ := strconv.ParseInt(parts[1], 10, 64)

	err := b.db.VotePoll(ctx, pollID, telegramID, optionID)
	if err != nil {
		b.answerCallback(callback.ID, "⚠️ "+err.Error())
		return
	}

	poll, err := b.db.GetPoll(ctx, pollID)
	if err == nil && poll != nil {
		kb := PollVoteKeyboard(poll.ID, poll.Options)
		editMsg := tgbotapi.NewEditMessageReplyMarkup(telegramID, callback.Message.MessageID, kb)
		b.sendAPI("edit_poll_vote", editMsg)
	}

	b.answerCallback(callback.ID, "✅ Suara kamu berhasil direkam!")
}

func (b *Bot) handleReactionCallback(ctx context.Context, telegramID int64, data string, callback *tgbotapi.CallbackQuery) {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) < 2 {
		return
	}

	confessionID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return
	}

	reaction := parts[1]

	err = b.confession.ReactToConfession(ctx, confessionID, telegramID, reaction)
	if err != nil {
		b.answerCallback(callback.ID, "❌ Gagal menambahkan reaksi")
		return
	}

	confession, _ := b.db.GetConfession(ctx, confessionID)
	if confession != nil {
		b.checkAchievements(ctx, confession.AuthorID)
	}
	b.checkAchievements(ctx, telegramID)
	b.processReward(ctx, telegramID, "reaction_given")

	counts, _ := b.confession.GetReactionCounts(ctx, confessionID)
	newKb := ConfessionReactionKeyboard(confessionID, counts)

	editMsg := tgbotapi.NewEditMessageReplyMarkup(
		telegramID,
		callback.Message.MessageID,
		newKb,
	)
	b.sendAPI("edit_reaction_keyboard", editMsg)

	b.answerCallback(callback.ID, fmt.Sprintf("Kamu react %s", reaction))
}

func (b *Bot) handleWhisperCallback(ctx context.Context, telegramID int64, dept string, callback *tgbotapi.CallbackQuery) {
	b.deleteMessage(telegramID, callback.Message.MessageID, "delete_whisper_callback")

	logIfErr("set_state_awaiting_whisper", b.db.SetUserState(ctx, telegramID, models.StateAwaitingWhisper, dept))

	emoji := models.DepartmentEmoji(models.Department(dept))
	b.sendMessage(telegramID, fmt.Sprintf(`📢 *Whisper ke %s %s*

Tulis pesan anonim kamu untuk mahasiswa %s:

_Ketik /cancel untuk membatalkan_`, emoji, dept, dept), nil)
}

func (b *Bot) handleLegalCallback(ctx context.Context, telegramID int64, value string, callback *tgbotapi.CallbackQuery) {
	if value == "agree" {
		b.deleteMessage(telegramID, callback.Message.MessageID, "delete_legal_callback")

		b.startEmailVerif(ctx, telegramID)
	}
}

func (b *Bot) handleCircleCallback(ctx context.Context, telegramID int64, data string, callback *tgbotapi.CallbackQuery) {
	parts := strings.SplitN(data, ":", 2)
	action := parts[0]

	switch action {
	case "join":
		if len(parts) < 2 {
			return
		}
		slug := parts[1]

		state, _, _ := b.db.GetUserState(ctx, telegramID)
		if state == models.StateInChat {
			kb := ConfirmKeyboard(fmt.Sprintf("circle:confirm_join:%s", slug), "circle:stay_chat")
			b.sendMessageHTML(telegramID, `⚠️ <b>Kamu sedang dalam Private Chat aktif</b>

Bergabung ke circle akan mengakhiri chat kamu saat ini secara otomatis. Apakah kamu yakin?`, &kb)
			return
		}

		room, err := b.room.JoinRoom(ctx, telegramID, slug)
		if err != nil {
			b.answerCallback(callback.ID, "❌ "+err.Error())
			return
		}

		b.deleteMessage(telegramID, callback.Message.MessageID, "delete_circle_join")

		kb := LeaveCircleKeyboard()
		text := fmt.Sprintf(`🎉 <b>Berhasil Terhubung ke Circle %s</b>

━━━━━━━━━━━━━━━━━━━
Sekarang semua pesan yang kamu ketik akan dikirim ke semua anggota circle ini secara anonim.

💡 Gunakan /leave_circle atau klik tombol di bawah untuk keluar.

<i>Mulai ngobrol sekarang...</i>`, room.Name)

		b.sendMessageHTML(telegramID, text, &kb)

	case "create":
		b.deleteMessage(telegramID, callback.Message.MessageID, "delete_circle_create")

		logIfErr("set_state_room_name", b.db.SetUserState(ctx, telegramID, models.StateAwaitingRoomName, ""))
		b.sendMessage(telegramID, "➕ *Buat Circle Baru*\n\nTuliskan *Nama Circle* yang ingin kamu buat:\n(Contoh: Pejuang Kopi PNJ)\n\n_Ketik /cancel untuk membatalkan_", nil)

	case "confirm_join":
		if len(parts) < 2 {
			return
		}
		slug := parts[1]

		b.deleteMessage(telegramID, callback.Message.MessageID, "delete_circle_confirm_join")

		partnerID, _ := b.chat.StopChat(ctx, telegramID)
		if partnerID > 0 {
			b.sendMessage(partnerID, "👋 *Partner kamu telah memutus chat.*", nil)
		}

		room, err := b.room.JoinRoom(ctx, telegramID, slug)
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
		b.deleteMessage(telegramID, callback.Message.MessageID, "delete_circle_stay_chat")
		b.answerCallback(callback.ID, "👌 Oke, private chat dilanjutkan.")

	case "leave":
		b.deleteMessage(telegramID, callback.Message.MessageID, "delete_circle_leave")
		b.handleLeaveCircle(ctx, &tgbotapi.Message{From: &tgbotapi.User{ID: telegramID}})

	case "leave_next":
		b.deleteMessage(telegramID, callback.Message.MessageID, "delete_circle_leave_next")

		logIfErr("leave_room_for_next", b.room.LeaveRoom(ctx, telegramID))
		b.sendMessageHTML(telegramID, "👋 <b>Kamu telah keluar dari circle.</b>\n⏭️ <i>Mencari partner baru...</i>", nil)

		b.startSearch(ctx, telegramID, "", "", 0)

	case "stay":
		b.deleteMessage(telegramID, callback.Message.MessageID, "delete_circle_stay")
		b.answerCallback(callback.ID, "👌 Oke, kamu tetap di circle.")
	}
}
