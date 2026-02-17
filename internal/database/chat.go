package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/pnj-anonymous-bot/internal/models"
)

func (d *DB) AddToQueue(telegramID int64, preferredDept, preferredGender string, preferredYear int) error {
	query, args, _ := d.Builder.Insert("chat_queue").
		Columns("telegram_id", "preferred_dept", "preferred_gender", "preferred_year", "joined_at").
		Values(telegramID, preferredDept, preferredGender, preferredYear, time.Now()).
		ToSql()

	query = d.PrepareQuery(strings.Replace(query, "INSERT INTO", "INSERT OR REPLACE INTO", 1))
	_, err := d.DB.Exec(query, args...)
	return err
}

func (d *DB) RemoveFromQueue(telegramID int64) error {
	query, args, _ := d.Builder.Delete("chat_queue").Where("telegram_id = ?", telegramID).ToSql()
	_, err := d.DB.Exec(query, args...)
	return err
}

func (d *DB) IsInQueue(telegramID int64) (bool, error) {
	var count int
	query, args, _ := d.Builder.Select("COUNT(*)").From("chat_queue").Where("telegram_id = ?", telegramID).ToSql()
	err := d.DB.Get(&count, query, args...)
	return count > 0, err
}

func (d *DB) FindMatch(telegramID int64, preferredDept, preferredGender string, preferredYear int) (int64, error) {
	var matchID int64

	b := d.Builder.Select("cq.telegram_id").
		From("chat_queue cq").
		Join("users u ON cq.telegram_id = u.telegram_id").
		Where("cq.telegram_id != ?", telegramID).
		Where("u.is_banned = FALSE")

	if preferredDept != "" {
		b = b.Where("u.department = ?", preferredDept)
	}

	if preferredGender != "" {
		b = b.Where("u.gender = ?", preferredGender)
	}

	if preferredYear != 0 {
		b = b.Where("u.year = ?", preferredYear)
	}

	b = b.Where(squirrel.Expr("cq.telegram_id NOT IN ("+
		"SELECT blocked_id FROM blocked_users WHERE user_id = ? "+
		"UNION "+
		"SELECT user_id FROM blocked_users WHERE blocked_id = ?)",
		telegramID, telegramID))

	b = b.OrderBy("u.karma DESC", "cq.joined_at ASC").Limit(1)

	query, args, err := b.ToSql()
	if err != nil {
		return 0, err
	}

	err = d.DB.Get(&matchID, query, args...)
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
	query, args, _ := d.Builder.Select("COUNT(*)").From("chat_queue").ToSql()
	err := d.DB.Get(&count, query, args...)
	return count, err
}

func (d *DB) CreateChatSession(user1ID, user2ID int64) (*models.ChatSession, error) {
	now := time.Now()
	query, args, _ := d.Builder.Insert("chat_sessions").
		Columns("user1_id", "user2_id", "is_active", "started_at").
		Values(user1ID, user2ID, true, now).
		ToSql()

	id, err := d.InsertGetID(query, "id", args...)

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
	query, args, err := d.Builder.Select("id", "user1_id", "user2_id", "is_active", "started_at", "ended_at").
		From("chat_sessions").
		Where("(user1_id = ? OR user2_id = ?) AND is_active = TRUE", telegramID, telegramID).
		OrderBy("started_at DESC").Limit(1).ToSql()

	if err != nil {
		return nil, err
	}

	err = d.DB.Get(session, query, args...)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active session: %w", err)
	}

	return session, nil
}

func (d *DB) EndChatSession(sessionID int64) error {
	query, args, _ := d.Builder.Update("chat_sessions").
		Set("is_active", false).
		Set("ended_at", time.Now()).
		Where("id = ?", sessionID).ToSql()
	_, err := d.DB.Exec(query, args...)
	return err
}

func (d *DB) EndAllActiveSessions(telegramID int64) error {
	query, args, _ := d.Builder.Update("chat_sessions").
		Set("is_active", false).
		Set("ended_at", time.Now()).
		Where("(user1_id = ? OR user2_id = ?) AND is_active = TRUE", telegramID, telegramID).ToSql()
	_, err := d.DB.Exec(query, args...)
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
	query, args, _ := d.Builder.Select("COUNT(*)").From("chat_sessions").
		Where("user1_id = ? OR user2_id = ?", telegramID, telegramID).ToSql()
	err := d.DB.Get(&count, query, args...)
	return count, err
}

func (d *DB) GetExpiredQueueItems(timeoutSeconds int) ([]models.ChatQueue, error) {
	threshold := time.Now().Add(-time.Duration(timeoutSeconds) * time.Second)

	query, args, err := d.Builder.Select("telegram_id", "preferred_dept", "preferred_gender", "preferred_year", "joined_at").
		From("chat_queue").
		Where("(preferred_dept != '' OR preferred_gender != '' OR preferred_year != 0)").
		Where("joined_at <= ?", threshold).ToSql()

	if err != nil {
		return nil, err
	}

	var items []models.ChatQueue
	err = d.DB.Select(&items, query, args...)
	return items, err
}

func (d *DB) ClearQueueFilters(telegramID int64) error {
	query, args, _ := d.Builder.Update("chat_queue").
		Set("preferred_dept", "").
		Set("preferred_gender", "").
		Set("preferred_year", 0).
		Where("telegram_id = ?", telegramID).ToSql()

	_, err := d.DB.Exec(query, args...)
	return err
}
