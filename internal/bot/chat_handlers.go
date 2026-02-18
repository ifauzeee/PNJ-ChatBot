package bot

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/pnj-anonymous-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleSearch(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	state, _, _ := b.db.GetUserState(telegramID)
	if state == models.StateInChat {
		b.sendMessage(telegramID, "⚠️ Kamu masih dalam sesi chat!\nGunakan /stop untuk menghentikan atau /next untuk partner baru.", nil)
		return
	}

	if state == models.StateSearching {
		b.sendMessage(telegramID, "⏳ Kamu sudah dalam antrian pencarian. Tunggu sebentar ya!", nil)
		return
	}

	args := msg.CommandArguments()
	if args != "" {
		parts := strings.Fields(args)
		preferredDept := ""
		preferredGender := ""
		preferredYear := 0

		for _, part := range parts {
			if models.IsValidDepartment(part) {
				preferredDept = part
			} else if part == "Laki-laki" || part == "Perempuan" {
				preferredGender = part
			} else if year, err := strconv.Atoi(part); err == nil && models.IsValidEntryYear(year) {
				preferredYear = year
			}
		}
		b.startSearch(telegramID, preferredDept, preferredGender, preferredYear)
		return
	}

	kb := SearchKeyboard()
	b.sendMessage(telegramID, "🔍 *Cari Partner Chat Anonim*\n\nPilih filter pencarian:", &kb)
}

func (b *Bot) startSearch(telegramID int64, preferredDept, preferredGender string, preferredYear int) {
	if preferredDept == "any" {
		preferredDept = ""
	}

	_ = b.room.LeaveRoom(telegramID)

	matchID, err := b.chat.SearchPartner(telegramID, preferredDept, preferredGender, preferredYear)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
		return
	}

	if matchID > 0 {
		b.notifyMatchFound(telegramID, matchID)
	} else {
		queueCount, _ := b.chat.GetQueueCount()
		kb := CancelSearchKeyboard()

		searchText := fmt.Sprintf(`🔍 *Mencari Partner...*

⏳ Kamu telah masuk ke antrian
👥 Orang dalam antrian: *%d*

_Menunggu partner yang cocok..._
_Kamu akan diberi notifikasi ketika partner ditemukan_`, queueCount)

		b.sendMessage(telegramID, searchText, &kb)
	}
}

func (b *Bot) notifyMatchFound(user1ID, user2ID int64) {
	gender1, dept1, year1, _ := b.chat.GetPartnerInfo(user1ID)
	gender2, dept2, year2, _ := b.chat.GetPartnerInfo(user2ID)

	kb := ChatActionKeyboard()

	msg1 := fmt.Sprintf(`<b>🎉 Partner Ditemukan!</b>

━━━━━━━━━━━━━━━━━━━
Partner kamu:
%s %s | 🎓 %d
%s %s
━━━━━━━━━━━━━━━━━━━

💬 Mulai ngobrol sekarang!
Semua pesan akan diteruskan secara <b>anonim</b>.

<i>Ketik pesan untuk memulai...</i>`,
		models.GenderEmoji(models.Gender(gender2)), html.EscapeString(gender2), year2,
		models.DepartmentEmoji(models.Department(dept2)), html.EscapeString(dept2))

	msg2 := fmt.Sprintf(`<b>🎉 Partner Ditemukan!</b>

━━━━━━━━━━━━━━━━━━━
Partner kamu:
%s %s | 🎓 %d
%s %s
━━━━━━━━━━━━━━━━━━━

💬 Mulai ngobrol sekarang!
Semua pesan akan diteruskan secara <b>anonim</b>.

<i>Ketik pesan untuk memulai...</i>`,
		models.GenderEmoji(models.Gender(gender1)), html.EscapeString(gender1), year1,
		models.DepartmentEmoji(models.Department(dept1)), html.EscapeString(dept1))

	b.sendMessageHTML(user1ID, msg1, &kb)
	b.sendMessageHTML(user2ID, msg2, &kb)
}

func (b *Bot) handleNext(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	state, _, _ := b.db.GetUserState(telegramID)

	if state == models.StateInCircle {
		room, _ := b.room.GetUserRoom(telegramID)
		roomName := "Circle"
		if room != nil {
			roomName = room.Name
		}

		kb := ConfirmKeyboard("circle:leave_next", "circle:stay")
		b.sendMessageHTML(telegramID, fmt.Sprintf(`⚠️ <b>Kamu sedang berada di %s</b>

Perintah /next hanya digunakan untuk mencari partner Private Chat. 
Apakah kamu ingin keluar dari Circle dan mencari partner baru?`, roomName), &kb)
		return
	}

	_ = b.room.LeaveRoom(telegramID)

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
}

func (b *Bot) handleStop(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	state, _, _ := b.db.GetUserState(telegramID)
	if state == models.StateSearching {
		_ = b.chat.CancelSearch(telegramID)
		b.sendMessage(telegramID, "🛑 Pencarian dihentikan.", nil)
		return
	}

	session, _ := b.db.GetActiveSession(telegramID)
	if session == nil {
		b.sendMessageHTML(telegramID, "⚠️ <b>Tidak ada chat aktif saat ini.</b>", nil)
		return
	}

	partnerID, err := b.chat.StopChat(telegramID)
	if err != nil {
		b.sendMessageHTML(telegramID, "❌ Gagal menghentikan chat.", nil)
		return
	}

	duration := time.Since(session.StartedAt).Minutes()
	b.checkChatMarathon(session.User1ID, duration)
	b.checkChatMarathon(session.User2ID, duration)
	b.checkAchievements(session.User1ID)
	b.checkAchievements(session.User2ID)

	b.sendMessageHTML(partnerID, "👋 <b>Partner telah menghentikan chat.</b>", nil)
	b.sendMessageHTML(telegramID, "🛑 <b>Chat dihentikan.</b>\nKetik /search untuk mencari partner baru.", nil)
}

func (b *Bot) handleChatMessage(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	partnerID, err := b.chat.GetPartner(telegramID)
	if err != nil || partnerID == 0 {
		_ = b.db.SetUserState(telegramID, models.StateNone, "")
		b.sendMessage(telegramID, "⚠️ Chat tidak aktif. Gunakan /search untuk mencari partner.", nil)
		return
	}

	session, _ := b.db.GetActiveSession(telegramID)
	if session != nil {
		b.logSessionEvidence(session.ID, telegramID, msg)
	}

	switch {
	case msg.Text != "":
		text := msg.Text
		if b.profanity.IsBad(text) {
			text = b.profanity.Clean(text)
			b.sendMessage(telegramID, "⚠️ *Peringatan:* Pesan kamu mengandung kata-kata yang tidak pantas dan telah disensor.", nil)
		}
		b.sendMessage(partnerID, escapeMarkdown(text), nil)
		b.processReward(telegramID, "chat_message")

	case msg.Sticker != nil, msg.Photo != nil, msg.Animation != nil:
		if safe, reason := b.isSafeMedia(msg); !safe {
			b.sendMessage(telegramID, "🚫 *Konten diblokir:* "+reason, nil)
			return
		}
		b.forwardMatchedMedia(partnerID, msg)

	case msg.Voice != nil:
		voice := tgbotapi.NewVoice(partnerID, tgbotapi.FileID(msg.Voice.FileID))
		_, _ = b.api.Send(voice)

	case msg.Video != nil:
		video := tgbotapi.NewVideo(partnerID, tgbotapi.FileID(msg.Video.FileID))
		video.Caption = "📹 *Video Sekali Lihat* (Akan terhapus dalam 15 detik)"
		if msg.Caption != "" {
			video.Caption += "\n\n" + msg.Caption
		}
		video.ParseMode = "Markdown"
		sentMsg, _ := b.api.Send(video)

		go b.deleteMessageAfterDelay(partnerID, sentMsg.MessageID, 15*time.Second)

	case msg.Document != nil:
		doc := tgbotapi.NewDocument(partnerID, tgbotapi.FileID(msg.Document.FileID))
		if msg.Caption != "" {
			doc.Caption = msg.Caption
		}
		_, _ = b.api.Send(doc)

	case msg.VideoNote != nil:
		vnCfg := tgbotapi.VideoNoteConfig{
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{ChatID: partnerID},
				File:     tgbotapi.FileID(msg.VideoNote.FileID),
			},
			Length: msg.VideoNote.Length,
		}
		_, _ = b.api.Send(vnCfg)

	default:
		b.sendMessage(telegramID, "⚠️ Tipe pesan ini tidak didukung.", nil)
	}
}

func (b *Bot) logSessionEvidence(sessionID, telegramID int64, msg *tgbotapi.Message) {
	msgType := "text"
	content := msg.Text
	switch {
	case msg.Sticker != nil:
		msgType, content = "sticker", "Sticker:"+msg.Sticker.FileID
	case msg.Photo != nil:
		msgType, content = "photo", "Photo:"+msg.Photo[len(msg.Photo)-1].FileID
	case msg.Voice != nil:
		msgType, content = "voice", "Voice:"+msg.Voice.FileID
	case msg.Video != nil:
		msgType, content = "video", "Video:"+msg.Video.FileID
	case msg.Animation != nil:
		msgType, content = "animation", "Animation:"+msg.Animation.FileID
	}
	b.evidence.LogMessage(sessionID, telegramID, content, msgType)
}

func (b *Bot) forwardMatchedMedia(partnerID int64, msg *tgbotapi.Message) {
	if msg.Sticker != nil {
		stickerCfg := tgbotapi.StickerConfig{
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{ChatID: partnerID},
				File:     tgbotapi.FileID(msg.Sticker.FileID),
			},
		}
		_, _ = b.api.Send(stickerCfg)
	} else if msg.Photo != nil {
		photos := msg.Photo
		photo := photos[len(photos)-1]
		photoMsg := tgbotapi.NewPhoto(partnerID, tgbotapi.FileID(photo.FileID))
		photoMsg.Caption = "🖼️ *Foto Sekali Lihat* (Akan terhapus dalam 10 detik)"
		if msg.Caption != "" {
			photoMsg.Caption += "\n\n" + msg.Caption
		}
		photoMsg.ParseMode = "Markdown"
		sentMsg, _ := b.api.Send(photoMsg)

		go b.deleteMessageAfterDelay(partnerID, sentMsg.MessageID, 10*time.Second)
	} else if msg.Animation != nil {
		anim := tgbotapi.NewAnimation(partnerID, tgbotapi.FileID(msg.Animation.FileID))
		_, _ = b.api.Send(anim)
	}
}

func (b *Bot) deleteMessageAfterDelay(chatID int64, messageID int, delay time.Duration) {
	time.Sleep(delay)
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	_, _ = b.api.Send(deleteMsg)
}
