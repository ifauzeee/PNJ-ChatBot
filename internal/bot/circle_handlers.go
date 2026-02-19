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

func (b *Bot) handleCircles(ctx context.Context, msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	rooms, err := b.room.GetActiveRooms(ctx)
	if err != nil {
		b.sendMessage(telegramID, "❌ Gagal mengambil daftar circle.", nil)
		return
	}

	kb := RoomsKeyboard(rooms)
	text := `👥 <b>Anonymous Circles (Group Chat)</b>

━━━━━━━━━━━━━━━━━━━
Gabung ke circle topik tertentu untuk ngobrol bareng mahasiswa lainnya secara anonim!

📌 <b>Pilih Circle:</b>`

	b.sendMessageHTML(telegramID, text, &kb)
}

func (b *Bot) handleLeaveCircle(ctx context.Context, msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	room, err := b.room.GetUserRoom(ctx, telegramID)
	if err != nil || room == nil {
		b.sendMessage(telegramID, "⚠️ Kamu tidak sedang berada di circle mana pun.", nil)
		return
	}

	err = b.room.LeaveRoom(ctx, telegramID)
	if err != nil {
		b.sendMessage(telegramID, "❌ Gagal keluar dari circle.", nil)
		return
	}

	b.sendMessageHTML(telegramID, fmt.Sprintf("👋 <b>Kamu telah keluar dari circle %s</b>", room.Name), nil)
	b.showMainMenu(ctx, telegramID, nil)
}

func (b *Bot) handleCircleMessage(ctx context.Context, msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	members, roomName, err := b.room.GetRoomMembers(ctx, telegramID)
	if err != nil {
		logIfErr("set_state_none_circle_err", b.db.SetUserState(ctx, telegramID, models.StateNone, ""))
		b.sendMessage(telegramID, "⚠️ Kamu tidak berada di circle aktif. Gunakan /circles untuk bergabung.", nil)
		return
	}

	user, _ := b.db.GetUser(ctx, telegramID)
	senderInfo := "Anonymous"
	if user != nil {
		senderInfo = fmt.Sprintf("%s %s", models.GenderEmoji(user.Gender), string(user.Department))
	}

	for _, memberID := range members {
		if memberID == telegramID {
			continue
		}

		if msg.Text != "" {
			text := msg.Text
			if b.profanity.IsBad(text) {
				text = b.profanity.Clean(text)
				b.sendMessage(telegramID, "⚠️ *Peringatan:* Pesan kamu mengandung kata-kata yang tidak pantas dan telah disensor.", nil)
			}
			msgOut := fmt.Sprintf("👥 <b>[%s]</b>\n👤 %s: %s", roomName, senderInfo, html.EscapeString(text))
			b.sendMessageHTML(memberID, msgOut, nil)
		} else {
			if safe, reason := b.isSafeMedia(ctx, msg); !safe {
				b.sendMessage(telegramID, "🚫 *Konten diblokir:* "+reason, nil)
				return
			}
			b.forwardMedia(memberID, msg, fmt.Sprintf("👥 [%s] 👤 %s", roomName, senderInfo))
		}
	}
}

func (b *Bot) handleRoomNameInput(ctx context.Context, msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	name := validation.SanitizeText(msg.Text)
	if errMsg := validation.ValidateText(name, validation.RoomNameLimits); errMsg != "" {
		b.sendMessage(telegramID, errMsg, nil)
		return
	}

	logIfErr("set_state_room_desc", b.db.SetUserState(ctx, telegramID, models.StateAwaitingRoomDesc, name))
	b.sendMessage(telegramID, fmt.Sprintf("📝 *Nama Circle:* %s\n\nSekarang tulis *Deskripsi Singkat* untuk circle ini:", name), nil)
}

func (b *Bot) handleRoomDescInput(ctx context.Context, msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	desc := validation.SanitizeText(msg.Text)
	_, name, _ := b.db.GetUserState(ctx, telegramID)

	if errMsg := validation.ValidateText(desc, validation.RoomDescLimits); errMsg != "" {
		b.sendMessage(telegramID, errMsg, nil)
		return
	}

	room, err := b.room.CreateRoom(ctx, name, desc)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("❌ %s", err.Error()), nil)
		logIfErr("set_state_none_room_err", b.db.SetUserState(ctx, telegramID, models.StateNone, ""))
		return
	}

	logIfErr("set_state_none_after_room", b.db.SetUserState(ctx, telegramID, models.StateNone, ""))
	b.sendMessageHTML(telegramID, fmt.Sprintf("✅ <b>Circle Berhasil Dibuat!</b>\n\nSekarang kamu dan orang lain bisa bergabung ke <b>%s</b> melalui menu /circles.", room.Name), nil)

	if _, err = b.room.JoinRoom(ctx, telegramID, room.Slug); err != nil {
		logIfErr("auto_join_room_err", err)
	}
	metrics.CircleJoinsTotal.Inc()

	kb := LeaveCircleKeyboard()
	b.sendMessageHTML(telegramID, fmt.Sprintf("🎉 Kamu otomatis bergabung ke circle <b>%s</b>. Selamat ngobrol!", room.Name), &kb)
}
