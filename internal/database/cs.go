package database

import (
	"database/sql"
	"fmt"
	"time"
)

func (d *DB) SaveCSMessage(adminMessageID int, userID int64) error {
	_, err := d.Exec(`INSERT INTO cs_messages (admin_message_id, user_id) VALUES (?, ?)`, adminMessageID, userID)
	return err
}

func (d *DB) GetCSUserByMessage(adminMessageID int) (int64, error) {
	var userID int64
	err := d.QueryRow(
		`SELECT user_id FROM cs_messages WHERE admin_message_id = ?`,
		adminMessageID,
	).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("CS message not found: %w", err)
	}
	return userID, nil
}

func (d *DB) JoinCSQueue(userID int64) error {
	_, err := d.Exec(`INSERT OR IGNORE INTO cs_queue (user_id, joined_at) VALUES (?, ?)`, userID, time.Now())
	return err
}

func (d *DB) LeaveCSQueue(userID int64) error {
	_, err := d.Exec(`DELETE FROM cs_queue WHERE user_id = ?`, userID)
	return err
}

func (d *DB) GetNextInCSQueue() (int64, error) {
	var userID int64
	err := d.QueryRow(`SELECT user_id FROM cs_queue ORDER BY joined_at ASC LIMIT 1`).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return userID, err
}

func (d *DB) GetCSQueuePosition(userID int64) (int, error) {
	var pos int
	err := d.QueryRow(`SELECT COUNT(*) FROM cs_queue WHERE joined_at < (SELECT joined_at FROM cs_queue WHERE user_id = ?)`, userID).Scan(&pos)
	return pos + 1, err
}

func (d *DB) CreateCSSession(userID, adminID int64) error {
	_, err := d.Exec(`INSERT OR REPLACE INTO cs_sessions (user_id, admin_id, last_activity, started_at) VALUES (?, ?, ?, ?)`,
		userID, adminID, time.Now(), time.Now())
	return err
}

func (d *DB) EndCSSession(userID int64) error {
	_, err := d.Exec(`DELETE FROM cs_sessions WHERE user_id = ?`, userID)
	return err
}

func (d *DB) GetActiveCSSessionByUser(userID int64) (int64, error) {
	var adminID int64
	err := d.QueryRow(`SELECT admin_id FROM cs_sessions WHERE user_id = ?`, userID).Scan(&adminID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return adminID, err
}

func (d *DB) GetActiveCSSessionByAdmin(adminID int64) (int64, error) {
	var userID int64
	err := d.QueryRow(`SELECT user_id FROM cs_sessions WHERE admin_id = ?`, adminID).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return userID, err
}

func (d *DB) UpdateCSSessionActivity(userID int64) error {
	_, err := d.Exec(`UPDATE cs_sessions SET last_activity = ? WHERE user_id = ?`, time.Now(), userID)
	return err
}

func (d *DB) GetTimedOutCSSessions(timeoutMinutes int) ([]int64, error) {
	rows, err := d.Query(`SELECT user_id FROM cs_sessions WHERE last_activity < ?`, time.Now().Add(-time.Duration(timeoutMinutes)*time.Minute))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			userIDs = append(userIDs, id)
		}
	}
	return userIDs, nil
}
