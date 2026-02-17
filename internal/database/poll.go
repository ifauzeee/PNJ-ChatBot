package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pnj-anonymous-bot/internal/models"
)

func (d *DB) CreatePoll(authorID int64, question string, options []string) (int64, error) {
	tx, err := d.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var pollID int64
	query := d.PrepareQuery(`INSERT INTO polls (author_id, question, created_at) VALUES (?, ?, ?)`)
	if d.DBType == "postgres" {
		err = tx.QueryRow(query+" RETURNING id", authorID, question, time.Now()).Scan(&pollID)
	} else {
		res, errExec := tx.Exec(query, authorID, question, time.Now())
		if errExec == nil {
			pollID, _ = res.LastInsertId()
		}
		err = errExec
	}

	for _, opt := range options {
		_, err := tx.Exec(
			d.PrepareQuery(`INSERT INTO poll_options (poll_id, option_text) VALUES (?, ?)`),
			pollID, opt,
		)
		if err != nil {
			return 0, fmt.Errorf("failed to create poll option: %w", err)
		}
	}

	return pollID, tx.Commit()
}

func (d *DB) GetPoll(pollID int64) (*models.Poll, error) {
	poll := &models.Poll{}
	err := d.QueryRow(
		`SELECT id, author_id, question, created_at FROM polls WHERE id = ?`,
		pollID,
	).Scan(&poll.ID, &poll.AuthorID, &poll.Question, &poll.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := d.Query(
		`SELECT id, poll_id, option_text, vote_count FROM poll_options WHERE poll_id = ?`,
		pollID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		opt := &models.PollOption{}
		if err := rows.Scan(&opt.ID, &opt.PollID, &opt.OptionText, &opt.VoteCount); err != nil {
			return nil, err
		}
		poll.Options = append(poll.Options, opt)
	}

	return poll, nil
}

func (d *DB) GetLatestPolls(limit int) ([]*models.Poll, error) {
	rows, err := d.Query(
		`SELECT id, author_id, question, created_at FROM polls ORDER BY created_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var polls []*models.Poll
	for rows.Next() {
		p := &models.Poll{}
		if err := rows.Scan(&p.ID, &p.AuthorID, &p.Question, &p.CreatedAt); err != nil {
			return nil, err
		}
		polls = append(polls, p)
	}

	for _, p := range polls {
		optRows, err := d.Query(
			`SELECT id, poll_id, option_text, vote_count FROM poll_options WHERE poll_id = ?`,
			p.ID,
		)
		if err == nil {
			for optRows.Next() {
				opt := &models.PollOption{}
				if err := optRows.Scan(&opt.ID, &opt.PollID, &opt.OptionText, &opt.VoteCount); err == nil {
					p.Options = append(p.Options, opt)
				}
			}
			optRows.Close()
		}
	}

	return polls, nil
}

func (d *DB) VotePoll(pollID, telegramID, optionID int64) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRow(d.PrepareQuery(`SELECT EXISTS(SELECT 1 FROM poll_votes WHERE poll_id = ? AND telegram_id = ?)`), pollID, telegramID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("kamu sudah memberikan suara di polling ini")
	}

	var optExists bool
	err = tx.QueryRow(d.PrepareQuery(`SELECT EXISTS(SELECT 1 FROM poll_options WHERE id = ? AND poll_id = ?)`), optionID, pollID).Scan(&optExists)
	if err != nil {
		return err
	}
	if !optExists {
		return fmt.Errorf("opsi tidak valid untuk polling ini")
	}

	_, err = tx.Exec(
		d.PrepareQuery(`INSERT INTO poll_votes (poll_id, telegram_id, option_id, created_at) VALUES (?, ?, ?, ?)`),
		pollID, telegramID, optionID, time.Now(),
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(d.PrepareQuery(`UPDATE poll_options SET vote_count = vote_count + 1 WHERE id = ?`), optionID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (d *DB) GetPollVoteCount(pollID int64) (int, error) {
	var count int
	err := d.QueryRow(`SELECT COUNT(*) FROM poll_votes WHERE poll_id = ?`, pollID).Scan(&count)
	return count, err
}
