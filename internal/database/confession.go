package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/pnj-anonymous-bot/internal/logger"
	"github.com/pnj-anonymous-bot/internal/models"
	"go.uber.org/zap"
)

func (d *DB) CreateConfession(ctx context.Context, authorID int64, content, department string) (*models.Confession, error) {
	now := time.Now()
	builder := d.Builder.Insert("confessions").
		Columns("author_id", "content", "department", "created_at").
		Values(authorID, content, department, now)

	id, err := d.InsertGetIDContext(ctx, builder, "id")
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

func (d *DB) GetConfession(ctx context.Context, id int64) (*models.Confession, error) {
	c := &models.Confession{}
	builder := d.Builder.Select("id", "author_id", "content", "department", "like_count", "created_at").
		From("confessions").Where("id = ?", id)

	err := d.GetBuilderContext(ctx, c, builder)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get confession: %w", err)
	}
	return c, nil
}

func (d *DB) GetLatestConfessions(ctx context.Context, limit int) ([]*models.Confession, error) {
	builder := d.Builder.Select("id", "author_id", "content", "department", "like_count", "created_at").
		From("confessions").OrderBy("created_at DESC").Limit(uint64(limit))

	var confessions []*models.Confession
	err := d.SelectBuilderContext(ctx, &confessions, builder)
	if err != nil {
		return nil, fmt.Errorf("failed to get confessions: %w", err)
	}
	return confessions, nil
}

func (d *DB) AddConfessionReaction(ctx context.Context, confessionID, telegramID int64, reaction string) error {
	builder := d.Builder.Insert("confession_reactions").
		Columns("confession_id", "telegram_id", "reaction", "created_at").
		Values(confessionID, telegramID, reaction, time.Now())

	_, err := d.InsertReplaceContext(ctx, builder, "confession_id, telegram_id")
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

	_, err = d.ExecBuilderContext(ctx, updateBuilder)
	if err != nil {
		return err
	}

	var authorID int64
	authorQuery := d.Builder.Select("author_id").From("confessions").Where("id = ?", confessionID)
	err = d.GetBuilderContext(ctx, &authorID, authorQuery)
	if err == nil && authorID != telegramID {
		if err := d.IncrementUserKarma(ctx, authorID, 1); err != nil {
			logger.Warn("Failed to increment user karma after confession reaction", zap.Error(err))
		}
	}

	return err
}

func (d *DB) HasReacted(ctx context.Context, confessionID, telegramID int64) (bool, error) {
	var count int
	builder := d.Builder.Select("COUNT(*)").From("confession_reactions").
		Where("confession_id = ? AND telegram_id = ?", confessionID, telegramID)

	err := d.GetBuilderContext(ctx, &count, builder)
	return count > 0, err
}

func (d *DB) GetConfessionReactionCounts(ctx context.Context, confessionID int64) (map[string]int, error) {
	builder := d.Builder.Select("reaction", "COUNT(*) as cnt").From("confession_reactions").
		Where("confession_id = ?", confessionID).GroupBy("reaction")

	var items []struct {
		Reaction string `db:"reaction"`
		Count    int    `db:"cnt"`
	}
	err := d.SelectBuilderContext(ctx, &items, builder)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, item := range items {
		counts[item.Reaction] = item.Count
	}
	return counts, nil
}

func (d *DB) GetUserConfessionCount(ctx context.Context, telegramID int64, since time.Time) (int, error) {
	var count int
	builder := d.Builder.Select("COUNT(*)").From("confessions").
		Where("author_id = ? AND created_at > ?", telegramID, since)

	err := d.GetBuilderContext(ctx, &count, builder)
	return count, err
}

func (d *DB) GetTotalConfessions(ctx context.Context, telegramID int64) (int, error) {
	var count int
	builder := d.Builder.Select("COUNT(*)").From("confessions").Where("author_id = ?", telegramID)

	err := d.GetBuilderContext(ctx, &count, builder)
	return count, err
}

func (d *DB) CreateConfessionReply(ctx context.Context, confessionID, authorID int64, content string) error {
	builder := d.Builder.Insert("confession_replies").
		Columns("confession_id", "author_id", "content", "created_at").
		Values(confessionID, authorID, content, time.Now())

	_, err := d.ExecBuilderContext(ctx, builder)
	if err == nil {
		if err := d.IncrementUserKarma(ctx, authorID, 2); err != nil {
			logger.Warn("Failed to increment user karma after confession reply", zap.Error(err))
		}
	}
	return err
}

func (d *DB) GetConfessionReplies(ctx context.Context, confessionID int64) ([]*models.ConfessionReply, error) {
	builder := d.Builder.Select("id", "confession_id", "author_id", "content", "created_at").
		From("confession_replies").Where("confession_id = ?", confessionID).OrderBy("created_at ASC")

	var replies []*models.ConfessionReply
	err := d.SelectBuilderContext(ctx, &replies, builder)
	return replies, err
}

func (d *DB) GetConfessionReplyCount(ctx context.Context, confessionID int64) (int, error) {
	var count int
	builder := d.Builder.Select("COUNT(*)").From("confession_replies").Where("confession_id = ?", confessionID)

	err := d.GetBuilderContext(ctx, &count, builder)
	return count, err
}
