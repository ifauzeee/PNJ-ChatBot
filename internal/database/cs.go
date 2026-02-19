package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func (d *DB) SaveCSMessage(ctx context.Context, adminMessageID int, userID int64) error {
	builder := d.Builder.Insert("cs_messages").
		Columns("admin_message_id", "user_id").
		Values(adminMessageID, userID)

	_, err := d.ExecBuilderContext(ctx, builder)
	return err
}

func (d *DB) GetCSUserByMessage(ctx context.Context, adminMessageID int) (int64, error) {
	var userID int64
	builder := d.Builder.Select("user_id").From("cs_messages").Where("admin_message_id = ?", adminMessageID)

	err := d.GetBuilderContext(ctx, &userID, builder)
	if err != nil {
		return 0, fmt.Errorf("CS message not found: %w", err)
	}
	return userID, nil
}

func (d *DB) JoinCSQueue(ctx context.Context, userID int64) error {
	builder := d.Builder.Insert("cs_queue").
		Columns("user_id", "joined_at").
		Values(userID, time.Now())

	_, err := d.InsertIgnoreContext(ctx, builder, "user_id")
	return err
}

func (d *DB) LeaveCSQueue(ctx context.Context, userID int64) error {
	builder := d.Builder.Delete("cs_queue").Where("user_id = ?", userID)
	_, err := d.ExecBuilderContext(ctx, builder)
	return err
}

func (d *DB) GetNextInCSQueue(ctx context.Context) (int64, error) {
	var userID int64
	builder := d.Builder.Select("user_id").From("cs_queue").OrderBy("joined_at ASC").Limit(1)

	err := d.GetBuilderContext(ctx, &userID, builder)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return userID, err
}

func (d *DB) GetCSQueuePosition(ctx context.Context, userID int64) (int, error) {
	var pos int
	subQuery := d.Builder.Select("joined_at").From("cs_queue").Where("user_id = ?", userID)
	query, args, err := subQuery.ToSql()
	if err != nil {
		return 0, err
	}

	builder := d.Builder.Select("COUNT(*)").From("cs_queue").Where("joined_at < ("+query+")", args...)

	err = d.GetBuilderContext(ctx, &pos, builder)
	return pos + 1, err
}

func (d *DB) CreateCSSession(ctx context.Context, userID, adminID int64) error {
	builder := d.Builder.Insert("cs_sessions").
		Columns("user_id", "admin_id", "last_activity", "started_at").
		Values(userID, adminID, time.Now(), time.Now())

	_, err := d.InsertReplaceContext(ctx, builder, "user_id", "last_activity")
	return err
}

func (d *DB) EndCSSession(ctx context.Context, userID int64) error {
	builder := d.Builder.Delete("cs_sessions").Where("user_id = ?", userID)
	_, err := d.ExecBuilderContext(ctx, builder)
	return err
}

func (d *DB) GetActiveCSSessionByUser(ctx context.Context, userID int64) (int64, error) {
	var adminID int64
	builder := d.Builder.Select("admin_id").From("cs_sessions").Where("user_id = ?", userID)

	err := d.GetBuilderContext(ctx, &adminID, builder)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return adminID, err
}

func (d *DB) GetActiveCSSessionByAdmin(ctx context.Context, adminID int64) (int64, error) {
	var userID int64
	builder := d.Builder.Select("user_id").From("cs_sessions").Where("admin_id = ?", adminID)

	err := d.GetBuilderContext(ctx, &userID, builder)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return userID, err
}

func (d *DB) UpdateCSSessionActivity(ctx context.Context, userID int64) error {
	builder := d.Builder.Update("cs_sessions").
		Set("last_activity", time.Now()).
		Where("user_id = ?", userID)

	_, err := d.ExecBuilderContext(ctx, builder)
	return err
}

func (d *DB) GetTimedOutCSSessions(ctx context.Context, timeoutMinutes int) ([]int64, error) {
	var userIDs []int64
	builder := d.Builder.Select("user_id").From("cs_sessions").
		Where("last_activity < ?", time.Now().Add(-time.Duration(timeoutMinutes)*time.Minute))

	err := d.SelectBuilderContext(ctx, &userIDs, builder)
	return userIDs, err
}
