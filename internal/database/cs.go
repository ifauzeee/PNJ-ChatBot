package database

import "fmt"

func (d *DB) SaveCSMessage(adminMessageID int, userID int64) error {
	_, err := d.Exec(
		`INSERT INTO cs_messages (admin_message_id, user_id) VALUES (?, ?)`,
		adminMessageID, userID,
	)
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
