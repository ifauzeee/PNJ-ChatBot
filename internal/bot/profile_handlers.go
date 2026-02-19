package bot

import (
	"context"
	"fmt"
	"html"

	"github.com/pnj-anonymous-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleProfile(ctx context.Context, msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	user, err := b.profile.GetProfile(ctx, telegramID)
	if err != nil || user == nil {
		b.sendMessage(telegramID, "❌ Gagal memuat profil.", nil)
		return
	}

	totalChats, totalConfessions, totalReactions, daysSince, _ := b.profile.GetStats(ctx, telegramID)

	earned, _ := b.db.GetUserAchievementsContext(ctx, telegramID)
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

	expNeeded := user.Level * 100 * user.Level
	if expNeeded == 0 {
		expNeeded = 100
	}
	progress := (float64(user.Exp) / float64(expNeeded)) * 10
	progressBar := ""
	for i := 0; i < 10; i++ {
		if i < int(progress) {
			progressBar += "■"
		} else {
			progressBar += "□"
		}
	}

	profileText := fmt.Sprintf(`<b>👤 Profil Kamu</b>

━━━━━━━━━━━━━━━━━━━
🏷️ <b>Nama:</b> %s
⭐ <b>Level %d</b>
📊 <code>%s</code> (%d/%d EXP)
💰 <b>Poin:</b> <b>%d</b>
🔥 <b>Daily Streak:</b> <b>%d hari</b>
✨ <b>Karma:</b> <b>%d</b>
%s <b>Gender:</b> %s
🎓 <b>Angkatan:</b> %d
%s <b>Jurusan:</b> %s
━━━━━━━━━━━━━━━━━━━
📊 <b>Statistik:</b>
💬 Total Chat: <b>%d</b>
📝 Confessions: <b>%d</b>
❤️ Reactions: <b>%d</b>
📅 Hari Aktif: <b>%d</b>%s
━━━━━━━━━━━━━━━━━━━
🛡️ Status Laporan: %d/3`,
		html.EscapeString(user.DisplayName),
		user.Level,
		progressBar, user.Exp, expNeeded,
		user.Points,
		user.DailyStreak,
		user.Karma,
		models.GenderEmoji(user.Gender), html.EscapeString(string(user.Gender)),
		user.Year,
		models.DepartmentEmoji(user.Department), html.EscapeString(string(user.Department)),
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

func (b *Bot) handleStats(ctx context.Context, msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	user, err := b.db.GetUser(ctx, telegramID)
	if err != nil || user == nil {
		b.sendMessage(telegramID, "❌ Gagal memuat profil.", nil)
		return
	}

	totalChats, totalConfessions, totalReactions, daysSince, err := b.profile.GetStats(ctx, telegramID)
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

func (b *Bot) handleEdit(ctx context.Context, msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	kb := EditProfileKeyboard()
	b.sendMessage(telegramID, "✏️ *Edit Profil*\n\nApa yang ingin kamu ubah?", &kb)
}
