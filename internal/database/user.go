package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pnj-anonymous-bot/internal/models"
)

func (d *DB) CreateUser(telegramID int64) (*models.User, error) {
	_, err := d.Exec(
		`INSERT OR IGNORE INTO users (telegram_id, created_at, updated_at) VALUES (?, ?, ?)`,
		telegramID, time.Now(), time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return d.GetUser(telegramID)
}

func (d *DB) GetUser(telegramID int64) (*models.User, error) {
	user := &models.User{}
	err := d.QueryRow(
		`SELECT id, telegram_id, email, gender, department, display_name, 
		        is_verified, is_banned, report_count, total_chats, created_at, updated_at 
		 FROM users WHERE telegram_id = ?`, telegramID,
	).Scan(
		&user.ID, &user.TelegramID, &user.Email, &user.Gender, &user.Department,
		&user.DisplayName, &user.IsVerified, &user.IsBanned, &user.ReportCount,
		&user.TotalChats, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func (d *DB) UpdateUserEmail(telegramID int64, email string) error {
	_, err := d.Exec(
		`UPDATE users SET email = ?, updated_at = ? WHERE telegram_id = ?`,
		email, time.Now(), telegramID,
	)
	return err
}

func (d *DB) UpdateUserVerified(telegramID int64, verified bool) error {
	_, err := d.Exec(
		`UPDATE users SET is_verified = ?, updated_at = ? WHERE telegram_id = ?`,
		verified, time.Now(), telegramID,
	)
	return err
}

func (d *DB) UpdateUserGender(telegramID int64, gender string) error {
	_, err := d.Exec(
		`UPDATE users SET gender = ?, updated_at = ? WHERE telegram_id = ?`,
		gender, time.Now(), telegramID,
	)
	return err
}

func (d *DB) UpdateUserDepartment(telegramID int64, dept string) error {
	_, err := d.Exec(
		`UPDATE users SET department = ?, updated_at = ? WHERE telegram_id = ?`,
		dept, time.Now(), telegramID,
	)
	return err
}

func (d *DB) UpdateUserDisplayName(telegramID int64, name string) error {
	_, err := d.Exec(
		`UPDATE users SET display_name = ?, updated_at = ? WHERE telegram_id = ?`,
		name, time.Now(), telegramID,
	)
	return err
}

func (d *DB) UpdateUserBanned(telegramID int64, banned bool) error {
	_, err := d.Exec(
		`UPDATE users SET is_banned = ?, updated_at = ? WHERE telegram_id = ?`,
		banned, time.Now(), telegramID,
	)
	return err
}

func (d *DB) IncrementReportCount(telegramID int64) (int, error) {
	_, err := d.Exec(
		`UPDATE users SET report_count = report_count + 1, updated_at = ? WHERE telegram_id = ?`,
		time.Now(), telegramID,
	)
	if err != nil {
		return 0, err
	}

	var count int
	err = d.QueryRow(`SELECT report_count FROM users WHERE telegram_id = ?`, telegramID).Scan(&count)
	return count, err
}

func (d *DB) IncrementTotalChats(telegramID int64) error {
	_, err := d.Exec(
		`UPDATE users SET total_chats = total_chats + 1, updated_at = ? WHERE telegram_id = ?`,
		time.Now(), telegramID,
	)
	return err
}

func (d *DB) SetUserState(telegramID int64, state models.UserState, data string) error {
	_, err := d.Exec(
		`UPDATE users SET state = ?, state_data = ?, updated_at = ? WHERE telegram_id = ?`,
		string(state), data, time.Now(), telegramID,
	)
	return err
}

func (d *DB) GetUserState(telegramID int64) (models.UserState, string, error) {
	var state, data string
	err := d.QueryRow(
		`SELECT COALESCE(state, ''), COALESCE(state_data, '') FROM users WHERE telegram_id = ?`,
		telegramID,
	).Scan(&state, &data)
	if err == sql.ErrNoRows {
		return models.StateNone, "", nil
	}
	return models.UserState(state), data, err
}

func (d *DB) GetOnlineUserCount() (int, error) {
	var count int
	err := d.QueryRow(`SELECT COUNT(*) FROM users WHERE is_verified = TRUE AND is_banned = FALSE`).Scan(&count)
	return count, err
}

func (d *DB) GetDepartmentUserCount(dept string) (int, error) {
	var count int
	err := d.QueryRow(
		`SELECT COUNT(*) FROM users WHERE department = ? AND is_verified = TRUE AND is_banned = FALSE`,
		dept,
	).Scan(&count)
	return count, err
}

func (d *DB) GetUsersByDepartment(dept string, excludeTelegramID int64) ([]int64, error) {
	rows, err := d.Query(
		`SELECT telegram_id FROM users 
		 WHERE department = ? AND is_verified = TRUE AND is_banned = FALSE AND telegram_id != ?`,
		dept, excludeTelegramID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (d *DB) IsUserProfileComplete(telegramID int64) (bool, error) {
	user, err := d.GetUser(telegramID)
	if err != nil || user == nil {
		return false, err
	}
	return user.IsVerified && string(user.Gender) != "" && string(user.Department) != "", nil
}
