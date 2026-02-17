package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pnj-anonymous-bot/internal/models"
)

func (d *DB) CreateConfession(authorID int64, content, department string) (*models.Confession, error) {
	result, err := d.Exec(
		`INSERT INTO confessions (author_id, content, department, created_at) VALUES (?, ?, ?, ?)`,
		authorID, content, department, time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create confession: %w", err)
	}

	id, _ := result.LastInsertId()
	return &models.Confession{
		ID:         id,
		AuthorID:   authorID,
		Content:    content,
		Department: department,
		LikeCount:  0,
		CreatedAt:  time.Now(),
	}, nil
}

func (d *DB) GetConfession(id int64) (*models.Confession, error) {
	c := &models.Confession{}
	err := d.QueryRow(
		`SELECT id, author_id, content, department, like_count, created_at 
		 FROM confessions WHERE id = ?`, id,
	).Scan(&c.ID, &c.AuthorID, &c.Content, &c.Department, &c.LikeCount, &c.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get confession: %w", err)
	}
	return c, nil
}

func (d *DB) GetLatestConfessions(limit int) ([]*models.Confession, error) {
	rows, err := d.Query(
		`SELECT id, author_id, content, department, like_count, created_at 
		 FROM confessions ORDER BY created_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get confessions: %w", err)
	}
	defer rows.Close()

	var confessions []*models.Confession
	for rows.Next() {
		c := &models.Confession{}
		if err := rows.Scan(&c.ID, &c.AuthorID, &c.Content, &c.Department, &c.LikeCount, &c.CreatedAt); err != nil {
			return nil, err
		}
		confessions = append(confessions, c)
	}
	return confessions, nil
}

func (d *DB) AddConfessionReaction(confessionID, telegramID int64, reaction string) error {
	_, err := d.Exec(
		`INSERT OR REPLACE INTO confession_reactions (confession_id, telegram_id, reaction, created_at) 
		 VALUES (?, ?, ?, ?)`,
		confessionID, telegramID, reaction, time.Now(),
	)
	if err != nil {
		return err
	}

	_, err = d.Exec(
		`UPDATE confessions SET like_count = (
			SELECT COUNT(*) FROM confession_reactions WHERE confession_id = ?
		) WHERE id = ?`,
		confessionID, confessionID,
	)
	return err
}

func (d *DB) HasReacted(confessionID, telegramID int64) (bool, error) {
	var count int
	err := d.QueryRow(
		`SELECT COUNT(*) FROM confession_reactions WHERE confession_id = ? AND telegram_id = ?`,
		confessionID, telegramID,
	).Scan(&count)
	return count > 0, err
}

func (d *DB) GetConfessionReactionCounts(confessionID int64) (map[string]int, error) {
	rows, err := d.Query(
		`SELECT reaction, COUNT(*) as cnt FROM confession_reactions 
		 WHERE confession_id = ? GROUP BY reaction`, confessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var reaction string
		var count int
		if err := rows.Scan(&reaction, &count); err != nil {
			return nil, err
		}
		counts[reaction] = count
	}
	return counts, nil
}

func (d *DB) GetUserConfessionCount(telegramID int64, since time.Time) (int, error) {
	var count int
	err := d.QueryRow(
		`SELECT COUNT(*) FROM confessions WHERE author_id = ? AND created_at > ?`,
		telegramID, since,
	).Scan(&count)
	return count, err
}

func (d *DB) GetTotalConfessions(telegramID int64) (int, error) {
	var count int
	err := d.QueryRow(
		`SELECT COUNT(*) FROM confessions WHERE author_id = ?`, telegramID,
	).Scan(&count)
	return count, err
}
