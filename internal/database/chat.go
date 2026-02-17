package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pnj-anonymous-bot/internal/models"
)

func (d *DB) AddToQueue(telegramID int64, preferredDept, preferredGender string, preferredYear int) error {
	_, err := d.Exec(
		`INSERT OR REPLACE INTO chat_queue (telegram_id, preferred_dept, preferred_gender, preferred_year, joined_at) VALUES (?, ?, ?, ?, ?)`,
		telegramID, preferredDept, preferredGender, preferredYear, time.Now(),
	)
	return err
}

func (d *DB) RemoveFromQueue(telegramID int64) error {
	_, err := d.Exec(`DELETE FROM chat_queue WHERE telegram_id = ?`, telegramID)
	return err
}

func (d *DB) IsInQueue(telegramID int64) (bool, error) {
	var count int
	err := d.QueryRow(`SELECT COUNT(*) FROM chat_queue WHERE telegram_id = ?`, telegramID).Scan(&count)
	return count > 0, err
}

func (d *DB) FindMatch(telegramID int64, preferredDept, preferredGender string, preferredYear int) (int64, error) {
	var matchID int64
	var query string
	var args []interface{}

	query = `SELECT cq.telegram_id FROM chat_queue cq
			 JOIN users u ON cq.telegram_id = u.telegram_id
			 WHERE cq.telegram_id != ? 
			 AND u.is_banned = FALSE`
	args = append(args, telegramID)

	if preferredDept != "" {
		query += ` AND u.department = ?`
		args = append(args, preferredDept)
	}

	if preferredGender != "" {
		query += ` AND u.gender = ?`
		args = append(args, preferredGender)
	}

	if preferredYear != 0 {
		query += ` AND u.year = ?`
		args = append(args, preferredYear)
	}

	query += ` AND cq.telegram_id NOT IN (
				 SELECT blocked_id FROM blocked_users WHERE user_id = ?
				 UNION
				 SELECT user_id FROM blocked_users WHERE blocked_id = ?
			 )
			 ORDER BY u.karma DESC, cq.joined_at ASC
			 LIMIT 1`
	args = append(args, telegramID, telegramID)

	err := d.QueryRow(query, args...).Scan(&matchID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to find match: %w", err)
	}
	return matchID, nil
}

func (d *DB) StopChat(telegramID int64) (int64, error) {
	session, err := d.GetActiveSession(telegramID)
	if err != nil {
		return 0, err
	}
	if session == nil {
		return 0, nil
	}

	partnerID := session.User1ID
	if session.User1ID == telegramID {
		partnerID = session.User2ID
	}

	err = d.EndChatSession(session.ID)
	if err != nil {
		return 0, fmt.Errorf("failed to end chat session: %w", err)
	}

	duration := time.Since(session.StartedAt).Minutes()
	if duration >= 10 {
		d.IncrementUserKarma(session.User1ID, 2)
		d.IncrementUserKarma(session.User2ID, 2)
	} else if duration >= 5 {
		d.IncrementUserKarma(session.User1ID, 1)
		d.IncrementUserKarma(session.User2ID, 1)
	}

	return partnerID, nil
}

func (d *DB) GetQueueCount() (int, error) {
	var count int
	err := d.QueryRow(`SELECT COUNT(*) FROM chat_queue`).Scan(&count)
	return count, err
}

func (d *DB) CreateChatSession(user1ID, user2ID int64) (*models.ChatSession, error) {
	now := time.Now()
	id, err := d.InsertGetID(
		`INSERT INTO chat_sessions (user1_id, user2_id, is_active, started_at) VALUES (?, ?, TRUE, ?)`,
		"id",
		user1ID, user2ID, now,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create chat session: %w", err)
	}

	d.IncrementTotalChats(user1ID)
	d.IncrementTotalChats(user2ID)

	return &models.ChatSession{
		ID:        id,
		User1ID:   user1ID,
		User2ID:   user2ID,
		IsActive:  true,
		StartedAt: now,
	}, nil
}

func (d *DB) GetActiveSession(telegramID int64) (*models.ChatSession, error) {
	session := &models.ChatSession{}
	var endedAt sql.NullTime

	err := d.QueryRow(
		`SELECT id, user1_id, user2_id, is_active, started_at, ended_at 
		 FROM chat_sessions 
		 WHERE (user1_id = ? OR user2_id = ?) AND is_active = TRUE
		 ORDER BY started_at DESC LIMIT 1`,
		telegramID, telegramID,
	).Scan(&session.ID, &session.User1ID, &session.User2ID, &session.IsActive, &session.StartedAt, &endedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active session: %w", err)
	}

	if endedAt.Valid {
		session.EndedAt = &endedAt.Time
	}

	return session, nil
}

func (d *DB) EndChatSession(sessionID int64) error {
	now := time.Now()
	_, err := d.Exec(
		`UPDATE chat_sessions SET is_active = FALSE, ended_at = ? WHERE id = ?`,
		now, sessionID,
	)
	return err
}

func (d *DB) EndAllActiveSessions(telegramID int64) error {
	now := time.Now()
	_, err := d.Exec(
		`UPDATE chat_sessions SET is_active = FALSE, ended_at = ? 
		 WHERE (user1_id = ? OR user2_id = ?) AND is_active = TRUE`,
		now, telegramID, telegramID,
	)
	return err
}

func (d *DB) GetChatPartner(telegramID int64) (int64, error) {
	session, err := d.GetActiveSession(telegramID)
	if err != nil {
		return 0, err
	}
	if session == nil {
		return 0, nil
	}

	if session.User1ID == telegramID {
		return session.User2ID, nil
	}
	return session.User1ID, nil
}

func (d *DB) GetTotalChatSessions(telegramID int64) (int, error) {
	var count int
	err := d.QueryRow(
		`SELECT COUNT(*) FROM chat_sessions WHERE user1_id = ? OR user2_id = ?`,
		telegramID, telegramID,
	).Scan(&count)
	return count, err
}

func (d *DB) GetExpiredQueueItems(timeoutSeconds int) ([]models.ChatQueue, error) {
	query := `SELECT telegram_id, preferred_dept, preferred_gender, preferred_year, joined_at 
			  FROM chat_queue 
			  WHERE (preferred_dept != '' OR preferred_gender != '' OR preferred_year != 0)
			  AND joined_at <= ?`

	threshold := time.Now().Add(-time.Duration(timeoutSeconds) * time.Second)

	rows, err := d.Query(query, threshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.ChatQueue
	for rows.Next() {
		var item models.ChatQueue
		if err := rows.Scan(&item.TelegramID, &item.PreferredDept, &item.PreferredGender, &item.PreferredYear, &item.JoinedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (d *DB) ClearQueueFilters(telegramID int64) error {
	_, err := d.Exec(
		`UPDATE chat_queue SET preferred_dept = '', preferred_gender = '', preferred_year = 0 WHERE telegram_id = ?`,
		telegramID,
	)
	return err
}
