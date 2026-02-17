package database

import (
	"database/sql"
	"time"

	"github.com/pnj-anonymous-bot/internal/models"
)

func (d *DB) AwardAchievement(telegramID int64, key string) (bool, error) {
	var exists bool
	err := d.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM user_achievements WHERE telegram_id = ? AND achievement_key = ?)`,
		telegramID, key,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	_, err = d.Exec(
		`INSERT INTO user_achievements (telegram_id, achievement_key, earned_at) VALUES (?, ?, ?)`,
		telegramID, key, time.Now(),
	)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (d *DB) GetUserAchievements(telegramID int64) ([]*models.UserAchievement, error) {
	rows, err := d.Query(
		`SELECT telegram_id, achievement_key, earned_at FROM user_achievements WHERE telegram_id = ?`,
		telegramID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var achievements []*models.UserAchievement
	for rows.Next() {
		ua := &models.UserAchievement{}
		if err := rows.Scan(&ua.TelegramID, &ua.AchievementKey, &ua.EarnedAt); err != nil {
			return nil, err
		}
		achievements = append(achievements, ua)
	}
	return achievements, nil
}

func (d *DB) GetUserPollCount(telegramID int64) (int, error) {
	var count int
	err := d.QueryRow(`SELECT COUNT(*) FROM polls WHERE author_id = ?`, telegramID).Scan(&count)
	return count, err
}

func (d *DB) GetUserMaxConfessionReactions(telegramID int64) (int, error) {
	var maxReactions int
	err := d.QueryRow(
		`SELECT MAX(like_count) FROM confessions WHERE author_id = ?`, telegramID,
	).Scan(&maxReactions)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return maxReactions, err
}
