package database

import (
	"database/sql"
	"fmt"
	"time"
)

func (d *DB) SaveCSMessage(adminMessageID int, userID int64) error {
	builder := d.Builder.Insert("cs_messages").
		Columns("admin_message_id", "user_id").
		Values(adminMessageID, userID)

	_, err := d.ExecBuilder(builder)
	return err
}

func (d *DB) GetCSUserByMessage(adminMessageID int) (int64, error) {
	var userID int64
	builder := d.Builder.Select("user_id").From("cs_messages").Where("admin_message_id = ?", adminMessageID)

	err := d.GetBuilder(&userID, builder)
	if err != nil {
		return 0, fmt.Errorf("CS message not found: %w", err)
	}
	return userID, nil
}

func (d *DB) JoinCSQueue(userID int64) error {
	builder := d.Builder.Insert("cs_queue").
		Columns("user_id", "joined_at").
		Values(userID, time.Now())

	_, err := d.InsertIgnore(builder, "user_id")
	return err
}

func (d *DB) LeaveCSQueue(userID int64) error {
	builder := d.Builder.Delete("cs_queue").Where("user_id = ?", userID)
	_, err := d.ExecBuilder(builder)
	return err
}

func (d *DB) GetNextInCSQueue() (int64, error) {
	var userID int64
	builder := d.Builder.Select("user_id").From("cs_queue").OrderBy("joined_at ASC").Limit(1)

	err := d.GetBuilder(&userID, builder)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return userID, err
}

func (d *DB) GetCSQueuePosition(userID int64) (int, error) {
	var pos int
	subQuery := d.Builder.Select("joined_at").From("cs_queue").Where("user_id = ?", userID)
	query, args, err := subQuery.ToSql()
	if err != nil {
		return 0, err
	}

	builder := d.Builder.Select("COUNT(*)").From("cs_queue").Where("joined_at < ("+query+")", args...)

	err = d.GetBuilder(&pos, builder)
	return pos + 1, err
}

func (d *DB) CreateCSSession(userID, adminID int64) error {
	builder := d.Builder.Insert("cs_sessions").
		Columns("user_id", "admin_id", "last_activity", "started_at").
		Values(userID, adminID, time.Now(), time.Now())

	_, err := d.InsertReplace(builder, "user_id", "last_activity")
	return err
}

func (d *DB) EndCSSession(userID int64) error {
	builder := d.Builder.Delete("cs_sessions").Where("user_id = ?", userID)
	_, err := d.ExecBuilder(builder)
	return err
}

func (d *DB) GetActiveCSSessionByUser(userID int64) (int64, error) {
	var adminID int64
	builder := d.Builder.Select("admin_id").From("cs_sessions").Where("user_id = ?", userID)

	err := d.GetBuilder(&adminID, builder)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return adminID, err
}

func (d *DB) GetActiveCSSessionByAdmin(adminID int64) (int64, error) {
	var userID int64
	builder := d.Builder.Select("user_id").From("cs_sessions").Where("admin_id = ?", adminID)

	err := d.GetBuilder(&userID, builder)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return userID, err
}

func (d *DB) UpdateCSSessionActivity(userID int64) error {
	builder := d.Builder.Update("cs_sessions").
		Set("last_activity", time.Now()).
		Where("user_id = ?", userID)

	_, err := d.ExecBuilder(builder)
	return err
}

func (d *DB) GetTimedOutCSSessions(timeoutMinutes int) ([]int64, error) {
	var userIDs []int64
	builder := d.Builder.Select("user_id").From("cs_sessions").
		Where("last_activity < ?", time.Now().Add(-time.Duration(timeoutMinutes)*time.Minute))

	err := d.SelectBuilder(&userIDs, builder)
	return userIDs, err
}
