package database

import (
	"time"

	"github.com/pnj-anonymous-bot/internal/models"
)

func (d *DB) AddPointsAndExp(telegramID int64, points, exp int) (newLevel int, leveledUp bool, err error) {
	tx, err := d.Begin()
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback()

	var current models.User
	query, args, _ := d.Builder.Select("level", "exp", "points").From("users").Where("telegram_id = ?", telegramID).ToSql()
	err = tx.QueryRow(d.PrepareQuery(query), args...).Scan(&current.Level, &current.Exp, &current.Points)
	if err != nil {
		return 0, false, err
	}

	newExp := current.Exp + exp
	newPoints := current.Points + points
	newLevel = current.Level
	leveledUp = false

	for {
		expNeeded := newLevel * 100 * newLevel
		if newExp >= expNeeded {
			newLevel++
			leveledUp = true
		} else {
			break
		}
	}

	updateQuery, updateArgs, _ := d.Builder.Update("users").
		Set("points", newPoints).
		Set("exp", newExp).
		Set("level", newLevel).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID).ToSql()

	_, err = tx.Exec(d.PrepareQuery(updateQuery), updateArgs...)
	if err != nil {
		return 0, false, err
	}

	return newLevel, leveledUp, tx.Commit()
}

func (d *DB) UpdateDailyStreak(telegramID int64) (newStreak int, streakBonus bool, err error) {
	tx, err := d.Begin()
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback()

	var lastActive time.Time
	var currentStreak int
	query, args, _ := d.Builder.Select("last_active_at", "daily_streak").From("users").Where("telegram_id = ?", telegramID).ToSql()
	err = tx.QueryRow(d.PrepareQuery(query), args...).Scan(&lastActive, &currentStreak)
	if err != nil {
		return 0, false, err
	}

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	if lastActive.Year() == now.Year() && lastActive.YearDay() == now.YearDay() {
		return currentStreak, false, nil
	}

	newStreak = 1
	streakBonus = false

	if lastActive.Year() == yesterday.Year() && lastActive.YearDay() == yesterday.YearDay() {
		newStreak = currentStreak + 1
		streakBonus = true
	}

	updateQuery, updateArgs, _ := d.Builder.Update("users").
		Set("daily_streak", newStreak).
		Set("last_active_at", now).
		Set("updated_at", now).
		Where("telegram_id = ?", telegramID).ToSql()

	_, err = tx.Exec(d.PrepareQuery(updateQuery), updateArgs...)
	if err != nil {
		return 0, false, err
	}

	return newStreak, streakBonus, tx.Commit()
}

func (d *DB) GetLeaderboard(limit int) ([]models.User, error) {
	var users []models.User
	query, args, _ := d.Builder.Select("display_name", "level", "points", "daily_streak").
		From("users").
		Where("is_banned = FALSE AND is_verified = TRUE").
		OrderBy("points DESC", "level DESC").
		Limit(uint64(limit)).ToSql()

	err := d.Select(&users, query, args...)
	return users, err
}
