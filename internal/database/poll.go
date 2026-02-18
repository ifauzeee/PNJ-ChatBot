package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/pnj-anonymous-bot/internal/models"
)

func (d *DB) CreatePoll(authorID int64, question string, options []string) (int64, error) {
	tx, err := d.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	builder := d.Builder.Insert("polls").
		Columns("author_id", "question", "created_at").
		Values(authorID, question, time.Now())

	query, args, err := builder.ToSql()
	if err != nil {
		return 0, err
	}

	var pollID int64
	if d.DBType == "postgres" {
		err = tx.QueryRow(query+" RETURNING id", args...).Scan(&pollID)
	} else {
		res, errExec := tx.Exec(query, args...)
		if errExec == nil {
			pollID, _ = res.LastInsertId()
		}
		err = errExec
	}
	if err != nil {
		return 0, err
	}

	for _, opt := range options {
		optBuilder := d.Builder.Insert("poll_options").
			Columns("poll_id", "option_text").
			Values(pollID, opt)
		optQuery, optArgs, err := optBuilder.ToSql()
		if err != nil {
			return 0, err
		}
		_, err = tx.Exec(optQuery, optArgs...)
		if err != nil {
			return 0, fmt.Errorf("failed to create poll option: %w", err)
		}
	}

	return pollID, tx.Commit()
}

func (d *DB) GetPoll(pollID int64) (*models.Poll, error) {
	poll := &models.Poll{}
	builder := d.Builder.Select("id", "author_id", "question", "created_at").
		From("polls").Where("id = ?", pollID)

	err := d.GetBuilder(poll, builder)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	optBuilder := d.Builder.Select("id", "poll_id", "option_text", "vote_count").
		From("poll_options").Where("poll_id = ?", pollID)

	err = d.SelectBuilder(&poll.Options, optBuilder)
	if err != nil {
		return nil, err
	}

	return poll, nil
}

func (d *DB) GetLatestPolls(limit int) ([]*models.Poll, error) {
	builder := d.Builder.Select("id", "author_id", "question", "created_at").
		From("polls").OrderBy("created_at DESC").Limit(uint64(limit))

	var polls []*models.Poll
	err := d.SelectBuilder(&polls, builder)
	if err != nil {
		return nil, err
	}

	for _, p := range polls {
		optBuilder := d.Builder.Select("id", "poll_id", "option_text", "vote_count").
			From("poll_options").Where("poll_id = ?", p.ID)
		_ = d.SelectBuilder(&p.Options, optBuilder)
	}

	return polls, nil
}

func (d *DB) VotePoll(pollID, telegramID, optionID int64) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	defer func() { _ = tx.Rollback() }()

	var exists bool
	existsQuery, existsArgs, _ := d.Builder.Select("1").Prefix("SELECT EXISTS(").
		From("poll_votes").Where("poll_id = ? AND telegram_id = ?", pollID, telegramID).
		Suffix(")").ToSql()

	err = tx.QueryRow(existsQuery, existsArgs...).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("kamu sudah memberikan suara di polling ini")
	}

	var optExists bool
	optExistsQuery, optExistsArgs, _ := d.Builder.Select("1").Prefix("SELECT EXISTS(").
		From("poll_options").Where("id = ? AND poll_id = ?", optionID, pollID).
		Suffix(")").ToSql()

	err = tx.QueryRow(optExistsQuery, optExistsArgs...).Scan(&optExists)
	if err != nil {
		return err
	}
	if !optExists {
		return fmt.Errorf("opsi tidak valid untuk polling ini")
	}

	voteBuilder := d.Builder.Insert("poll_votes").
		Columns("poll_id", "telegram_id", "option_id", "created_at").
		Values(pollID, telegramID, optionID, time.Now())
	voteQuery, voteArgs, _ := voteBuilder.ToSql()

	_, err = tx.Exec(voteQuery, voteArgs...)
	if err != nil {
		return err
	}

	updateBuilder := d.Builder.Update("poll_options").
		Set("vote_count", squirrel.Expr("vote_count + 1")).
		Where("id = ?", optionID)
	updateQuery, updateArgs, _ := updateBuilder.ToSql()

	_, err = tx.Exec(updateQuery, updateArgs...)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (d *DB) GetPollVoteCount(pollID int64) (int, error) {
	var count int
	builder := d.Builder.Select("COUNT(*)").From("poll_votes").Where("poll_id = ?", pollID)
	err := d.GetBuilder(&count, builder)
	return count, err
}
