package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/pnj-anonymous-bot/internal/models"
)

func (d *DB) CreateConfession(authorID int64, content, department string) (*models.Confession, error) {
	now := time.Now()
	builder := d.Builder.Insert("confessions").
		Columns("author_id", "content", "department", "created_at").
		Values(authorID, content, department, now)

	id, err := d.InsertGetID(builder, "id")
	if err != nil {
		return nil, fmt.Errorf("failed to create confession: %w", err)
	}

	return &models.Confession{
		ID:         id,
		AuthorID:   authorID,
		Content:    content,
		Department: department,
		LikeCount:  0,
		CreatedAt:  now,
	}, nil
}

func (d *DB) GetConfession(id int64) (*models.Confession, error) {
	c := &models.Confession{}
	builder := d.Builder.Select("id", "author_id", "content", "department", "like_count", "created_at").
		From("confessions").Where("id = ?", id)

	err := d.GetBuilder(c, builder)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get confession: %w", err)
	}
	return c, nil
}

func (d *DB) GetLatestConfessions(limit int) ([]*models.Confession, error) {
	builder := d.Builder.Select("id", "author_id", "content", "department", "like_count", "created_at").
		From("confessions").OrderBy("created_at DESC").Limit(uint64(limit))

	var confessions []*models.Confession
	err := d.SelectBuilder(&confessions, builder)
	if err != nil {
		return nil, fmt.Errorf("failed to get confessions: %w", err)
	}
	return confessions, nil
}

func (d *DB) AddConfessionReaction(confessionID, telegramID int64, reaction string) error {
	builder := d.Builder.Insert("confession_reactions").
		Columns("confession_id", "telegram_id", "reaction", "created_at").
		Values(confessionID, telegramID, reaction, time.Now())

	_, err := d.InsertReplace(builder, "confession_id, telegram_id")
	if err != nil {
		return err
	}

	subQuery := d.Builder.Select("COUNT(*)").From("confession_reactions").Where("confession_id = ?", confessionID)
	q, args, err := subQuery.ToSql()
	if err != nil {
		return err
	}

	updateBuilder := d.Builder.Update("confessions").
		Set("like_count", squirrel.Expr("("+q+")", args...)).
		Where("id = ?", confessionID)

	_, err = d.ExecBuilder(updateBuilder)
	if err != nil {
		return err
	}

	var authorID int64
	authorQuery := d.Builder.Select("author_id").From("confessions").Where("id = ?", confessionID)
	err = d.GetBuilder(&authorID, authorQuery)
	if err == nil && authorID != telegramID {
		d.IncrementUserKarma(authorID, 1)
	}

	return err
}

func (d *DB) HasReacted(confessionID, telegramID int64) (bool, error) {
	var count int
	builder := d.Builder.Select("COUNT(*)").From("confession_reactions").
		Where("confession_id = ? AND telegram_id = ?", confessionID, telegramID)

	err := d.GetBuilder(&count, builder)
	return count > 0, err
}

func (d *DB) GetConfessionReactionCounts(confessionID int64) (map[string]int, error) {
	builder := d.Builder.Select("reaction", "COUNT(*) as cnt").From("confession_reactions").
		Where("confession_id = ?", confessionID).GroupBy("reaction")

	var items []struct {
		Reaction string `db:"reaction"`
		Count    int    `db:"cnt"`
	}
	err := d.SelectBuilder(&items, builder)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, item := range items {
		counts[item.Reaction] = item.Count
	}
	return counts, nil
}

func (d *DB) GetUserConfessionCount(telegramID int64, since time.Time) (int, error) {
	var count int
	builder := d.Builder.Select("COUNT(*)").From("confessions").
		Where("author_id = ? AND created_at > ?", telegramID, since)

	err := d.GetBuilder(&count, builder)
	return count, err
}

func (d *DB) GetTotalConfessions(telegramID int64) (int, error) {
	var count int
	builder := d.Builder.Select("COUNT(*)").From("confessions").Where("author_id = ?", telegramID)

	err := d.GetBuilder(&count, builder)
	return count, err
}

func (d *DB) CreateConfessionReply(confessionID, authorID int64, content string) error {
	builder := d.Builder.Insert("confession_replies").
		Columns("confession_id", "author_id", "content", "created_at").
		Values(confessionID, authorID, content, time.Now())

	_, err := d.ExecBuilder(builder)
	if err == nil {
		d.IncrementUserKarma(authorID, 2)
	}
	return err
}

func (d *DB) GetConfessionReplies(confessionID int64) ([]*models.ConfessionReply, error) {
	builder := d.Builder.Select("id", "confession_id", "author_id", "content", "created_at").
		From("confession_replies").Where("confession_id = ?", confessionID).OrderBy("created_at ASC")

	var replies []*models.ConfessionReply
	err := d.SelectBuilder(&replies, builder)
	return replies, err
}

func (d *DB) GetConfessionReplyCount(confessionID int64) (int, error) {
	var count int
	builder := d.Builder.Select("COUNT(*)").From("confession_replies").Where("confession_id = ?", confessionID)

	err := d.GetBuilder(&count, builder)
	return count, err
}
