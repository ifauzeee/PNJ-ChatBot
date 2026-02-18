package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pnj-anonymous-bot/internal/models"
)

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
		_ = d.IncrementUserKarma(session.User1ID, 2)
		_ = d.IncrementUserKarma(session.User2ID, 2)
	} else if duration >= 5 {
		_ = d.IncrementUserKarma(session.User1ID, 1)
		_ = d.IncrementUserKarma(session.User2ID, 1)
	}

	return partnerID, nil
}

func (d *DB) CreateChatSession(user1ID, user2ID int64) (*models.ChatSession, error) {
	now := time.Now()
	builder := d.Builder.Insert("chat_sessions").
		Columns("user1_id", "user2_id", "is_active", "started_at").
		Values(user1ID, user2ID, true, now)

	id, err := d.InsertGetID(builder, "id")

	if err != nil {
		return nil, fmt.Errorf("failed to create chat session: %w", err)
	}

	_ = d.IncrementTotalChats(user1ID)
	_ = d.IncrementTotalChats(user2ID)

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
	builder := d.Builder.Select("id", "user1_id", "user2_id", "is_active", "started_at", "ended_at").
		From("chat_sessions").
		Where("(user1_id = ? OR user2_id = ?) AND is_active = TRUE", telegramID, telegramID).
		OrderBy("started_at DESC").Limit(1)

	err := d.GetBuilder(session, builder)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active session: %w", err)
	}

	return session, nil
}

func (d *DB) EndChatSession(sessionID int64) error {
	builder := d.Builder.Update("chat_sessions").
		Set("is_active", false).
		Set("ended_at", time.Now()).
		Where("id = ?", sessionID)
	_, err := d.ExecBuilder(builder)
	return err
}

func (d *DB) EndAllActiveSessions(telegramID int64) error {
	builder := d.Builder.Update("chat_sessions").
		Set("is_active", false).
		Set("ended_at", time.Now()).
		Where("(user1_id = ? OR user2_id = ?) AND is_active = TRUE", telegramID, telegramID)
	_, err := d.ExecBuilder(builder)
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
	builder := d.Builder.Select("COUNT(*)").From("chat_sessions").
		Where("user1_id = ? OR user2_id = ?", telegramID, telegramID)
	err := d.GetBuilder(&count, builder)
	return count, err
}
