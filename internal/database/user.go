package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/pnj-anonymous-bot/internal/models"
)

func (d *DB) CreateUser(telegramID int64) (*models.User, error) {
	query, args, err := d.Builder.Insert("users").
		Columns("telegram_id", "created_at", "updated_at").
		Values(telegramID, time.Now(), time.Now()).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	_, err = d.Exec(strings.Replace(query, "INSERT INTO", "INSERT OR IGNORE INTO", 1), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return d.GetUser(telegramID)
}

func (d *DB) GetUser(telegramID int64) (*models.User, error) {
	user := &models.User{}
	query, args, err := d.Builder.Select(
		"id", "telegram_id", "email", "gender", "department", "year",
		"display_name", "karma", "is_verified", "is_banned",
		"report_count", "total_chats", "created_at", "updated_at",
	).From("users").Where("telegram_id = ?", telegramID).ToSql()

	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	err = d.Get(user, query, args...)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func (d *DB) UpdateUserEmail(telegramID int64, email string) error {
	query, args, err := d.Builder.Update("users").
		Set("email", email).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID).
		ToSql()
	if err != nil {
		return err
	}
	_, err = d.Exec(query, args...)
	return err
}

func (d *DB) UpdateUserVerified(telegramID int64, verified bool) error {
	query, args, err := d.Builder.Update("users").
		Set("is_verified", verified).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID).
		ToSql()
	if err != nil {
		return err
	}
	_, err = d.Exec(query, args...)
	return err
}

func (d *DB) UpdateUserGender(telegramID int64, gender string) error {
	query, args, err := d.Builder.Update("users").
		Set("gender", gender).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID).
		ToSql()
	if err != nil {
		return err
	}
	_, err = d.Exec(query, args...)
	return err
}

func (d *DB) UpdateUserDepartment(telegramID int64, dept string) error {
	query, args, err := d.Builder.Update("users").
		Set("department", dept).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID).
		ToSql()
	if err != nil {
		return err
	}
	_, err = d.Exec(query, args...)
	return err
}

func (d *DB) UpdateUserYear(telegramID int64, year int) error {
	query, args, err := d.Builder.Update("users").
		Set("year", year).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID).
		ToSql()
	if err != nil {
		return err
	}
	_, err = d.Exec(query, args...)
	return err
}

func (d *DB) UpdateUserDisplayName(telegramID int64, name string) error {
	query, args, err := d.Builder.Update("users").
		Set("display_name", name).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID).
		ToSql()
	if err != nil {
		return err
	}
	_, err = d.Exec(query, args...)
	return err
}

func (d *DB) IncrementUserKarma(telegramID int64, amount int) error {
	query, args, err := d.Builder.Update("users").
		Set("karma", squirrel.Expr("karma + ?", amount)).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID).
		ToSql()
	if err != nil {
		return err
	}
	_, err = d.Exec(query, args...)
	return err
}

func (d *DB) UpdateUserBanned(telegramID int64, banned bool) error {
	query, args, err := d.Builder.Update("users").
		Set("is_banned", banned).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID).
		ToSql()
	if err != nil {
		return err
	}
	_, err = d.Exec(query, args...)
	return err
}

func (d *DB) IncrementReportCount(telegramID int64) (int, error) {
	query, args, err := d.Builder.Update("users").
		Set("report_count", squirrel.Expr("report_count + 1")).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID).
		ToSql()
	if err != nil {
		return 0, err
	}
	_, err = d.Exec(query, args...)
	if err != nil {
		return 0, err
	}

	var count int
	query, args, _ = d.Builder.Select("report_count").From("users").Where("telegram_id = ?", telegramID).ToSql()
	err = d.Get(&count, query, args...)
	return count, err
}

func (d *DB) IncrementTotalChats(telegramID int64) error {
	query, args, err := d.Builder.Update("users").
		Set("total_chats", squirrel.Expr("total_chats + 1")).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID).
		ToSql()
	if err != nil {
		return err
	}
	_, err = d.Exec(query, args...)
	return err
}

func (d *DB) SetUserState(telegramID int64, state models.UserState, data string) error {
	query, args, err := d.Builder.Update("users").
		Set("state", string(state)).
		Set("state_data", data).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID).
		ToSql()
	if err != nil {
		return err
	}
	_, err = d.Exec(query, args...)
	return err
}

func (d *DB) GetUserState(telegramID int64) (models.UserState, string, error) {
	var res struct {
		State string `db:"state"`
		Data  string `db:"state_data"`
	}
	query, args, err := d.Builder.Select("COALESCE(state, '') as state", "COALESCE(state_data, '') as state_data").
		From("users").Where("telegram_id = ?", telegramID).ToSql()
	if err != nil {
		return models.StateNone, "", err
	}

	err = d.Get(&res, query, args...)
	if err == sql.ErrNoRows {
		return models.StateNone, "", nil
	}
	return models.UserState(res.State), res.Data, err
}

func (d *DB) GetOnlineUserCount() (int, error) {
	var count int
	query, args, _ := d.Builder.Select("COUNT(*)").From("users").
		Where(squirrel.Eq{"is_verified": true, "is_banned": false}).ToSql()
	err := d.Get(&count, query, args...)
	return count, err
}

func (d *DB) GetDepartmentUserCount(dept string) (int, error) {
	var count int
	query, args, _ := d.Builder.Select("COUNT(*)").From("users").
		Where(squirrel.Eq{"department": dept, "is_verified": true, "is_banned": false}).ToSql()
	err := d.Get(&count, query, args...)
	return count, err
}

func (d *DB) GetUsersByDepartment(dept string, excludeTelegramID int64) ([]int64, error) {
	var ids []int64
	query, args, err := d.Builder.Select("telegram_id").From("users").
		Where(squirrel.Eq{"department": dept, "is_verified": true, "is_banned": false}).
		Where(squirrel.NotEq{"telegram_id": excludeTelegramID}).ToSql()
	if err != nil {
		return nil, err
	}

	err = d.Select(&ids, query, args...)
	return ids, err
}

func (d *DB) IsUserProfileComplete(telegramID int64) (bool, error) {
	user, err := d.GetUser(telegramID)
	if err != nil || user == nil {
		return false, err
	}
	return user.IsVerified && string(user.Gender) != "" && string(user.Department) != "" && user.Year != 0, nil
}

func (d *DB) GetAllVerifiedUsers() ([]int64, error) {
	var ids []int64
	query, args, err := d.Builder.Select("telegram_id").From("users").
		Where(squirrel.Eq{"is_verified": true, "is_banned": false}).ToSql()
	if err != nil {
		return nil, err
	}

	err = d.Select(&ids, query, args...)
	return ids, err
}
