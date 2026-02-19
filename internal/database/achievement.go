package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/pnj-anonymous-bot/internal/models"
)

func (d *DB) AwardAchievementContext(ctx context.Context, telegramID int64, key string) (bool, error) {
	var exists bool
	existsBuilder := d.Builder.Select("1").Prefix("SELECT EXISTS(").
		From("user_achievements").Where("telegram_id = ? AND achievement_key = ?", telegramID, key).
		Suffix(")")
	err := d.GetBuilderContext(ctx, &exists, existsBuilder)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	builder := d.Builder.Insert("user_achievements").
		Columns("telegram_id", "achievement_key", "earned_at").
		Values(telegramID, key, time.Now())

	_, err = d.ExecBuilderContext(ctx, builder)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (d *DB) GetUserAchievementsContext(ctx context.Context, telegramID int64) ([]*models.UserAchievement, error) {
	var achievements []*models.UserAchievement
	builder := d.Builder.Select("telegram_id", "achievement_key", "earned_at").
		From("user_achievements").Where("telegram_id = ?", telegramID)

	err := d.SelectBuilderContext(ctx, &achievements, builder)
	return achievements, err
}

func (d *DB) GetUserPollCountContext(ctx context.Context, telegramID int64) (int, error) {
	var count int
	builder := d.Builder.Select("COUNT(*)").From("polls").Where("author_id = ?", telegramID)
	err := d.GetBuilderContext(ctx, &count, builder)
	return count, err
}

func (d *DB) GetUserMaxConfessionReactionsContext(ctx context.Context, telegramID int64) (int, error) {
	var maxReactions int
	builder := d.Builder.Select("COALESCE(MAX(like_count), 0)").From("confessions").Where("author_id = ?", telegramID)
	err := d.GetBuilderContext(ctx, &maxReactions, builder)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return maxReactions, err
}
