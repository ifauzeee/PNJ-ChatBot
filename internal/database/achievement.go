package database

import (
	"database/sql"
	"time"

	"github.com/pnj-anonymous-bot/internal/models"
)

func (d *DB) AwardAchievement(telegramID int64, key string) (bool, error) {
	var exists bool
	existsBuilder := d.Builder.Select("1").Prefix("SELECT EXISTS(").
		From("user_achievements").Where("telegram_id = ? AND achievement_key = ?", telegramID, key).
		Suffix(")")
	err := d.GetBuilder(&exists, existsBuilder)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	builder := d.Builder.Insert("user_achievements").
		Columns("telegram_id", "achievement_key", "earned_at").
		Values(telegramID, key, time.Now())

	_, err = d.ExecBuilder(builder)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (d *DB) GetUserAchievements(telegramID int64) ([]*models.UserAchievement, error) {
	var achievements []*models.UserAchievement
	builder := d.Builder.Select("telegram_id", "achievement_key", "earned_at").
		From("user_achievements").Where("telegram_id = ?", telegramID)

	err := d.SelectBuilder(&achievements, builder)
	return achievements, err
}

func (d *DB) GetUserPollCount(telegramID int64) (int, error) {
	var count int
	builder := d.Builder.Select("COUNT(*)").From("polls").Where("author_id = ?", telegramID)
	err := d.GetBuilder(&count, builder)
	return count, err
}

func (d *DB) GetUserMaxConfessionReactions(telegramID int64) (int, error) {
	var maxReactions int
	builder := d.Builder.Select("COALESCE(MAX(like_count), 0)").From("confessions").Where("author_id = ?", telegramID)
	err := d.GetBuilder(&maxReactions, builder)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return maxReactions, err
}
