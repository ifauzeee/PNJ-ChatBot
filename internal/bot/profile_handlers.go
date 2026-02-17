package bot

import (
	"fmt"
	"html"

	"github.com/pnj-anonymous-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleProfile(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	user, err := b.profile.GetProfile(telegramID)
	if err != nil || user == nil {
		b.sendMessage(telegramID, "❌ Gagal memuat profil.", nil)
		return
	}

	totalChats, totalConfessions, totalReactions, daysSince, _ := b.profile.GetStats(telegramID)

	earned, _ := b.db.GetUserAchievements(telegramID)
	badgeStr := ""
	if len(earned) > 0 {
		badgeStr = "\n🏆 <b>Lencana:</b> "
		allAch := models.GetAchievements()
		for _, ua := range earned {
			if ach, ok := allAch[ua.AchievementKey]; ok {
				badgeStr += ach.Icon + " "
			}
		}
		badgeStr += "\n"
	}

	profileText := fmt.Sprintf(`<b>👤 Profil Kamu</b>

━━━━━━━━━━━━━━━━━━━
🏷️ <b>Nama Anonim:</b> %s
✨ <b>Karma:</b> <b>%d</b>
%s <b>Gender:</b> %s
🎓 <b>Angkatan:</b> %d
%s <b>Jurusan:</b> %s
📧 <b>Email:</b> %s
━━━━━━━━━━━━━━━━━━━
📊 <b>Statistik:</b>
💬 Total Chat: <b>%d</b>
📝 Confessions: <b>%d</b>
❤️ Reactions Diterima: <b>%d</b>
📅 Hari Aktif: <b>%d</b>%s
━━━━━━━━━━━━━━━━━━━
⚠️ Report Count: %d/3`,
		html.EscapeString(user.DisplayName),
		user.Karma,
		models.GenderEmoji(user.Gender), html.EscapeString(string(user.Gender)),
		user.Year,
		models.DepartmentEmoji(user.Department), html.EscapeString(string(user.Department)),
		html.EscapeString(maskEmail(user.Email)),
		totalChats,
		totalConfessions,
		totalReactions,
		daysSince,
		badgeStr,
		user.ReportCount,
	)

	kb := BackToMenuKeyboard()
	b.sendMessageHTML(telegramID, profileText, &kb)
}

func (b *Bot) handleStats(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	user, err := b.db.GetUser(telegramID)
	if err != nil || user == nil {
		b.sendMessage(telegramID, "❌ Gagal memuat profil.", nil)
		return
	}

	totalChats, totalConfessions, totalReactions, daysSince, err := b.profile.GetStats(telegramID)
	if err != nil {
		b.sendMessage(telegramID, "❌ Gagal memuat statistik.", nil)
		return
	}

	statsText := fmt.Sprintf(`<b>📊 Statistik Kamu</b>

━━━━━━━━━━━━━━━━━━━
✨ Total Karma: <b>%d</b>
💬 Total Chat: <b>%d</b>
📝 Confession Dibuat: <b>%d</b>
❤️ Reactions Diterima: <b>%d</b>
📅 Hari Sejak Bergabung: <b>%d</b>
━━━━━━━━━━━━━━━━━━━

<i>Terus berinteraksi untuk meningkatkan statistik kamu!</i> 🚀`,
		user.Karma, totalChats, totalConfessions, totalReactions, daysSince)

	kb := BackToMenuKeyboard()
	b.sendMessageHTML(telegramID, statsText, &kb)
}

func (b *Bot) handleEdit(msg *tgbotapi.Message) {
	kb := EditProfileKeyboard()
	b.sendMessage(msg.From.ID, "✏️ *Edit Profil*\n\nApa yang ingin kamu ubah?", &kb)
}
