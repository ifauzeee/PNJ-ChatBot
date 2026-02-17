package bot

import (
	"fmt"

	"github.com/pnj-anonymous-bot/internal/models"
)

func (b *Bot) checkAchievements(telegramID int64) {
	maxReactions, err := b.db.GetUserMaxConfessionReactions(telegramID)
	if err == nil && maxReactions >= 5 {
		b.awardAchievement(telegramID, "POPULAR_AUTHOR")
	}

	user, err := b.db.GetUser(telegramID)
	if err == nil && user != nil && user.Karma >= 50 {
		b.awardAchievement(telegramID, "KARMA_MASTER")
	}

	pollCount, err := b.db.GetUserPollCount(telegramID)
	if err == nil && pollCount >= 3 {
		b.awardAchievement(telegramID, "POLL_MAKER")
	}
}

func (b *Bot) awardAchievement(telegramID int64, key string) {
	newlyEarned, err := b.db.AwardAchievement(telegramID, key)
	if err != nil || !newlyEarned {
		return
	}

	achievement := models.GetAchievements()[key]

	msg := fmt.Sprintf(`🎊 <b>ACHIEVEMENT UNLOCKED!</b> 🎊

━━━━━━━━━━━━━━━━━━━
🏆 <b>%s %s</b>
📜 <i>%s</i>
━━━━━━━━━━━━━━━━━━━

Selamat! Kamu baru saja mendapatkan lencana baru. Cek lencana kamu di /profile.`,
		achievement.Icon, achievement.Name, achievement.Description)

	b.sendMessageHTML(telegramID, msg, nil)
}

func (b *Bot) checkChatMarathon(telegramID int64, durationMinutes float64) {
	if durationMinutes >= 60 {
		b.awardAchievement(telegramID, "CHAT_MARATHON")
	}
}
