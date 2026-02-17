package database

import (
	"fmt"
	"time"
)

func (d *DB) CreateReport(reporterID, reportedID int64, reason string, chatSessionID int64) error {
	_, err := d.Exec(
		`INSERT INTO reports (reporter_id, reported_id, reason, chat_session_id, created_at) VALUES (?, ?, ?, ?, ?)`,
		reporterID, reportedID, reason, chatSessionID, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to create report: %w", err)
	}
	return nil
}

func (d *DB) GetUserReportCount(telegramID int64, since time.Time) (int, error) {
	var count int
	err := d.QueryRow(
		`SELECT COUNT(*) FROM reports WHERE reporter_id = ? AND created_at > ?`,
		telegramID, since,
	).Scan(&count)
	return count, err
}

func (d *DB) BlockUser(userID, blockedID int64) error {
	_, err := d.Exec(
		`INSERT OR IGNORE INTO blocked_users (user_id, blocked_id, created_at) VALUES (?, ?, ?)`,
		userID, blockedID, time.Now(),
	)
	return err
}

func (d *DB) IsBlocked(userID, blockedID int64) (bool, error) {
	var count int
	err := d.QueryRow(
		`SELECT COUNT(*) FROM blocked_users WHERE 
		 (user_id = ? AND blocked_id = ?) OR (user_id = ? AND blocked_id = ?)`,
		userID, blockedID, blockedID, userID,
	).Scan(&count)
	return count > 0, err
}

func (d *DB) SaveVerificationCode(telegramID int64, email, code string, expiresAt time.Time) error {

	_, _ = d.Exec(
		`DELETE FROM verification_codes WHERE telegram_id = ?`, telegramID,
	)

	_, err := d.Exec(
		`INSERT INTO verification_codes (telegram_id, email, code, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		telegramID, email, code, expiresAt, time.Now(),
	)
	return err
}

func (d *DB) VerifyCode(telegramID int64, code string) (string, bool, error) {
	var email string
	var expiresAt time.Time
	var used bool

	err := d.QueryRow(
		`SELECT email, expires_at, used FROM verification_codes 
		 WHERE telegram_id = ? AND code = ? 
		 ORDER BY created_at DESC LIMIT 1`,
		telegramID, code,
	).Scan(&email, &expiresAt, &used)

	if err != nil {
		return "", false, nil
	}

	if used {
		return "", false, nil
	}

	if time.Now().After(expiresAt) {
		return "", false, nil
	}

	_, _ = d.Exec(
		`UPDATE verification_codes SET used = TRUE WHERE telegram_id = ? AND code = ?`,
		telegramID, code,
	)

	return email, true, nil
}

func (d *DB) CreateWhisper(senderID int64, targetDept, content, senderDept, senderGender string) (int64, error) {
	return d.InsertGetID(
		`INSERT INTO whispers (sender_id, target_dept, content, sender_dept, sender_gender, created_at) 
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"id",
		senderID, targetDept, content, senderDept, senderGender, time.Now(),
	)
}

func (d *DB) GetUserStats(telegramID int64) (totalChats int, totalConfessions int, totalReactions int, daysSinceJoined int, err error) {

	d.QueryRow(`SELECT COUNT(*) FROM chat_sessions WHERE user1_id = ? OR user2_id = ?`, telegramID, telegramID).Scan(&totalChats)

	d.QueryRow(`SELECT COUNT(*) FROM confessions WHERE author_id = ?`, telegramID).Scan(&totalConfessions)

	d.QueryRow(`SELECT COUNT(*) FROM confession_reactions cr 
				JOIN confessions c ON cr.confession_id = c.id 
				WHERE c.author_id = ?`, telegramID).Scan(&totalReactions)

	var createdAt time.Time
	d.QueryRow(`SELECT created_at FROM users WHERE telegram_id = ?`, telegramID).Scan(&createdAt)
	daysSinceJoined = int(time.Since(createdAt).Hours() / 24)

	return
}
