package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/pnj-anonymous-bot/internal/models"
)

func (d *DB) CreateUser(telegramID int64) (*models.User, error) {
	builder := d.Builder.Insert("users").
		Columns("telegram_id", "created_at", "updated_at").
		Values(telegramID, time.Now(), time.Now())

	_, err := d.InsertIgnore(builder, "telegram_id")
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return d.GetUser(telegramID)
}

func (d *DB) GetUser(telegramID int64) (*models.User, error) {
	user := &models.User{}
	builder := d.Builder.Select(
		"id", "telegram_id", "email", "gender", "department", "year",
		"display_name", "karma", "is_verified", "is_banned",
		"report_count", "total_chats", "level", "points", "exp",
		"daily_streak", "last_active_at", "created_at", "updated_at",
	).From("users").Where("telegram_id = ?", telegramID)

	err := d.GetBuilder(user, builder)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func (d *DB) UpdateUserEmail(telegramID int64, email string) error {
	builder := d.Builder.Update("users").
		Set("email", email).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID)

	_, err := d.ExecBuilder(builder)
	return err
}

func (d *DB) UpdateUserVerified(telegramID int64, verified bool) error {
	builder := d.Builder.Update("users").
		Set("is_verified", verified).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID)

	_, err := d.ExecBuilder(builder)
	return err
}

func (d *DB) UpdateUserGender(telegramID int64, gender string) error {
	builder := d.Builder.Update("users").
		Set("gender", gender).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID)

	_, err := d.ExecBuilder(builder)
	return err
}

func (d *DB) UpdateUserDepartment(telegramID int64, dept string) error {
	builder := d.Builder.Update("users").
		Set("department", dept).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID)

	_, err := d.ExecBuilder(builder)
	return err
}

func (d *DB) UpdateUserYear(telegramID int64, year int) error {
	builder := d.Builder.Update("users").
		Set("year", year).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID)

	_, err := d.ExecBuilder(builder)
	return err
}

func (d *DB) UpdateUserDisplayName(telegramID int64, name string) error {
	builder := d.Builder.Update("users").
		Set("display_name", name).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID)

	_, err := d.ExecBuilder(builder)
	return err
}

func (d *DB) IncrementUserKarma(telegramID int64, amount int) error {
	builder := d.Builder.Update("users").
		Set("karma", squirrel.Expr("karma + ?", amount)).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID)

	_, err := d.ExecBuilder(builder)
	return err
}

func (d *DB) UpdateUserBanned(telegramID int64, banned bool) error {
	builder := d.Builder.Update("users").
		Set("is_banned", banned).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID)

	_, err := d.ExecBuilder(builder)
	return err
}

func (d *DB) IncrementReportCount(telegramID int64) (int, error) {
	builder := d.Builder.Update("users").
		Set("report_count", squirrel.Expr("report_count + 1")).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID)

	_, err := d.ExecBuilder(builder)
	if err != nil {
		return 0, err
	}

	var count int
	selectBuilder := d.Builder.Select("report_count").From("users").Where("telegram_id = ?", telegramID)
	err = d.GetBuilder(&count, selectBuilder)
	return count, err
}

func (d *DB) IncrementTotalChats(telegramID int64) error {
	builder := d.Builder.Update("users").
		Set("total_chats", squirrel.Expr("total_chats + 1")).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID)

	_, err := d.ExecBuilder(builder)
	return err
}

func (d *DB) SetUserState(telegramID int64, state models.UserState, data string) error {
	builder := d.Builder.Update("users").
		Set("state", string(state)).
		Set("state_data", data).
		Set("updated_at", time.Now()).
		Where("telegram_id = ?", telegramID)

	_, err := d.ExecBuilder(builder)
	return err
}

func (d *DB) GetUserState(telegramID int64) (models.UserState, string, error) {
	var res struct {
		State string `db:"state"`
		Data  string `db:"state_data"`
	}
	builder := d.Builder.Select("COALESCE(state, '') as state", "COALESCE(state_data, '') as state_data").
		From("users").Where("telegram_id = ?", telegramID)

	err := d.GetBuilder(&res, builder)
	if err == sql.ErrNoRows {
		return models.StateNone, "", nil
	}
	return models.UserState(res.State), res.Data, err
}

func (d *DB) GetOnlineUserCount() (int, error) {
	var count int
	builder := d.Builder.Select("COUNT(*)").From("users").
		Where(squirrel.Eq{"is_verified": true, "is_banned": false})

	err := d.GetBuilder(&count, builder)
	return count, err
}

func (d *DB) GetDepartmentUserCount(dept string) (int, error) {
	var count int
	builder := d.Builder.Select("COUNT(*)").From("users").
		Where(squirrel.Eq{"department": dept, "is_verified": true, "is_banned": false})

	err := d.GetBuilder(&count, builder)
	return count, err
}

func (d *DB) GetUsersByDepartment(dept string, excludeTelegramID int64) ([]int64, error) {
	var ids []int64
	builder := d.Builder.Select("telegram_id").From("users").
		Where(squirrel.Eq{"department": dept, "is_verified": true, "is_banned": false}).
		Where(squirrel.NotEq{"telegram_id": excludeTelegramID})

	err := d.SelectBuilder(&ids, builder)
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
	builder := d.Builder.Select("telegram_id").From("users").
		Where(squirrel.Eq{"is_verified": true, "is_banned": false})

	err := d.SelectBuilder(&ids, builder)
	return ids, err
}
