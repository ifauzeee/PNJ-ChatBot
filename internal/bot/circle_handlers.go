package bot

import (
	"fmt"
	"html"
	"strings"

	"github.com/pnj-anonymous-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleCircles(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	rooms, err := b.room.GetActiveRooms()
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

func (b *Bot) handleLeaveCircle(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	room, err := b.room.GetUserRoom(telegramID)
	if err != nil || room == nil {
		b.sendMessage(telegramID, "⚠️ Kamu tidak sedang berada di circle mana pun.", nil)
		return
	}

	err = b.room.LeaveRoom(telegramID)
	if err != nil {
		b.sendMessage(telegramID, "❌ Gagal keluar dari circle.", nil)
		return
	}

	b.sendMessageHTML(telegramID, fmt.Sprintf("👋 <b>Kamu telah keluar dari circle %s</b>", room.Name), nil)
	b.showMainMenu(telegramID, nil)
}

func (b *Bot) handleCircleMessage(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	members, roomName, err := b.room.GetRoomMembers(telegramID)
	if err != nil {
		b.db.SetUserState(telegramID, models.StateNone, "")
		b.sendMessage(telegramID, "⚠️ Kamu tidak berada di circle aktif. Gunakan /circles untuk bergabung.", nil)
		return
	}

	user, _ := b.db.GetUser(telegramID)
	senderInfo := "Anonymous"
	if user != nil {
		senderInfo = fmt.Sprintf("%s %s", models.GenderEmoji(user.Gender), string(user.Department))
	}

	broadcastText := fmt.Sprintf("👥 <b>[%s]</b>\n👤 %s: %s", roomName, senderInfo, html.EscapeString(msg.Text))

	for _, memberID := range members {
		if memberID == telegramID {
			continue
		}

		if msg.Text != "" {
			b.sendMessageHTML(memberID, broadcastText, nil)
		} else {
			if safe, reason := b.isSafeMedia(msg); !safe {
				b.sendMessage(telegramID, "🚫 *Konten diblokir:* "+reason, nil)
				return
			}
			b.forwardMedia(memberID, msg, fmt.Sprintf("👥 [%s] 👤 %s", roomName, senderInfo))
		}
	}
}

func (b *Bot) handleRoomNameInput(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	name := strings.TrimSpace(msg.Text)

	if name == "" || len(name) < 3 {
		b.sendMessage(telegramID, "⚠️ Nama circle terlalu pendek. Minimal 3 karakter.", nil)
		return
	}

	if len(name) > 30 {
		b.sendMessage(telegramID, "⚠️ Nama circle terlalu panjang. Maksimal 30 karakter.", nil)
		return
	}

	b.db.SetUserState(telegramID, models.StateAwaitingRoomDesc, name)
	b.sendMessage(telegramID, fmt.Sprintf("📝 *Nama Circle:* %s\n\nSekarang tulis *Deskripsi Singkat* untuk circle ini:", name), nil)
}

func (b *Bot) handleRoomDescInput(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	desc := strings.TrimSpace(msg.Text)
	_, name, _ := b.db.GetUserState(telegramID)

	if desc == "" || len(desc) < 5 {
		b.sendMessage(telegramID, "⚠️ Deskripsi terlalu pendek. Minimal 5 karakter.", nil)
		return
	}

	room, err := b.room.CreateRoom(name, desc)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("❌ %s", err.Error()), nil)
		b.db.SetUserState(telegramID, models.StateNone, "")
		return
	}

	b.db.SetUserState(telegramID, models.StateNone, "")
	b.sendMessageHTML(telegramID, fmt.Sprintf("✅ <b>Circle Berhasil Dibuat!</b>\n\nSekarang kamu dan orang lain bisa bergabung ke <b>%s</b> melalui menu /circles.", room.Name), nil)

	b.room.JoinRoom(telegramID, room.Slug)

	kb := LeaveCircleKeyboard()
	b.sendMessageHTML(telegramID, fmt.Sprintf("🎉 Kamu otomatis bergabung ke circle <b>%s</b>. Selamat ngobrol!", room.Name), &kb)
}
