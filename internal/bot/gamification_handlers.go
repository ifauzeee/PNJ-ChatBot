package bot

import (
	"fmt"
	"html"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) processReward(telegramID int64, activity string) {
	level, leveledUp, _, _, err := b.gamification.RewardActivity(telegramID, activity)
	if err != nil {
		return
	}

	if leveledUp {
		msg := fmt.Sprintf(`🆙 <b>LEVEL UP!</b>

Selamat! Kamu sekarang mencapai <b>Level %d</b>.
Terus aktif chatting dan berinteraksi untuk mencapai level yang lebih tinggi!`, level)
		b.sendMessageHTML(telegramID, msg, nil)
	}
}

func (b *Bot) handleLeaderboard(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	users, err := b.gamification.GetLeaderboard()
	if err != nil {
		b.sendMessage(telegramID, "❌ Gagal mengambil data leaderboard.", nil)
		return
	}

	if len(users) == 0 {
		b.sendMessage(telegramID, "📋 Leaderboard masih kosong.", nil)
		return
	}

	text := "🏆 <b>LEADERBOARD MAHASISWA PALING AKTIF</b>\n"
	text += "━━━━━━━━━━━━━━━━━━━\n\n"

	for i, u := range users {
		medal := ""
		switch i {
		case 0:
			medal = "🥇 "
		case 1:
			medal = "🥈 "
		case 2:
			medal = "🥉 "
		default:
			medal = fmt.Sprintf("%d. ", i+1)
		}

		text += fmt.Sprintf("%s<b>%s</b>\n   ⭐ Level %d | 💰 %d pts | 🔥 %d days\n\n",
			medal, html.EscapeString(u.DisplayName), u.Level, u.Points, u.DailyStreak)
	}

	text += "━━━━━━━━━━━━━━━━━━━\n<i>Poin didapatkan dari chatting, confession, dan bereaksi.</i>"

	kb := BackToMenuKeyboard()
	b.sendMessageHTML(telegramID, text, &kb)
}
