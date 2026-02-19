package database

import (
	"context"
	"fmt"
	"time"

	"github.com/pnj-anonymous-bot/internal/logger"
	"github.com/pnj-anonymous-bot/internal/models"
	"go.uber.org/zap"
)

func (d *DB) CreateReport(ctx context.Context, reporterID, reportedID int64, reason, evidence string, chatSessionID int64) error {
	builder := d.Builder.Insert("reports").
		Columns("reporter_id", "reported_id", "reason", "evidence", "chat_session_id", "created_at").
		Values(reporterID, reportedID, reason, evidence, chatSessionID, time.Now())

	_, err := d.ExecBuilderContext(ctx, builder)
	if err != nil {
		return fmt.Errorf("failed to create report: %w", err)
	}
	return nil
}

func (d *DB) GetReport(ctx context.Context, reportID int64) (*models.Report, error) {
	var r models.Report
	builder := d.Builder.Select("*").From("reports").Where("id = ?", reportID)
	err := d.GetBuilderContext(ctx, &r, builder)
	return &r, err
}

func (d *DB) GetUserReportCount(ctx context.Context, telegramID int64, since time.Time) (int, error) {
	var count int
	builder := d.Builder.Select("COUNT(*)").From("reports").
		Where("reporter_id = ? AND created_at > ?", telegramID, since)

	err := d.GetBuilderContext(ctx, &count, builder)
	return count, err
}

func (d *DB) BlockUser(ctx context.Context, userID, blockedID int64) error {
	builder := d.Builder.Insert("blocked_users").
		Columns("user_id", "blocked_id", "created_at").
		Values(userID, blockedID, time.Now())

	_, err := d.InsertIgnoreContext(ctx, builder, "user_id, blocked_id")
	return err
}

func (d *DB) IsBlocked(ctx context.Context, userID, blockedID int64) (bool, error) {
	var count int
	builder := d.Builder.Select("COUNT(*)").From("blocked_users").
		Where("(user_id = ? AND blocked_id = ?) OR (user_id = ? AND blocked_id = ?)",
			userID, blockedID, blockedID, userID)

	err := d.GetBuilderContext(ctx, &count, builder)
	return count > 0, err
}

func (d *DB) SaveVerificationCode(ctx context.Context, telegramID int64, email, code string, expiresAt time.Time) error {
	deleteBuilder := d.Builder.Delete("verification_codes").Where("telegram_id = ?", telegramID)
	if _, err := d.ExecBuilderContext(ctx, deleteBuilder); err != nil {
		logger.Warn("Failed to delete old verification codes",
			zap.Int64("telegram_id", telegramID),
			zap.Error(err),
		)
	}

	builder := d.Builder.Insert("verification_codes").
		Columns("telegram_id", "email", "code", "expires_at", "created_at").
		Values(telegramID, email, code, expiresAt, time.Now())

	_, err := d.ExecBuilderContext(ctx, builder)
	return err
}

func (d *DB) VerifyCode(ctx context.Context, telegramID int64, code string) (string, bool, error) {
	var res struct {
		Email     string    `db:"email"`
		ExpiresAt time.Time `db:"expires_at"`
		Used      bool      `db:"used"`
	}

	builder := d.Builder.Select("email", "expires_at", "used").
		From("verification_codes").
		Where("telegram_id = ? AND code = ?", telegramID, code).
		OrderBy("created_at DESC").Limit(1)

	err := d.GetBuilderContext(ctx, &res, builder)
	if err != nil {
		return "", false, nil
	}

	if res.Used {
		return "", false, nil
	}

	if time.Now().After(res.ExpiresAt) {
		return "", false, nil
	}

	updateBuilder := d.Builder.Update("verification_codes").
		Set("used", true).
		Where("telegram_id = ? AND code = ?", telegramID, code)

	if _, err := d.ExecBuilderContext(ctx, updateBuilder); err != nil {
		logger.Warn("Failed to mark verification code as used",
			zap.Int64("telegram_id", telegramID),
			zap.Error(err),
		)
	}

	return res.Email, true, nil
}

func (d *DB) CreateWhisper(ctx context.Context, senderID int64, targetDept, content, senderDept, senderGender string) (int64, error) {
	builder := d.Builder.Insert("whispers").
		Columns("sender_id", "target_dept", "content", "sender_dept", "sender_gender", "created_at").
		Values(senderID, targetDept, content, senderDept, senderGender, time.Now())

	return d.InsertGetIDContext(ctx, builder, "id")
}

func (d *DB) GetUserWhisperCount(ctx context.Context, telegramID int64, since time.Time) (int, error) {
	var count int
	builder := d.Builder.Select("COUNT(*)").From("whispers").
		Where("sender_id = ? AND created_at > ?", telegramID, since)

	err := d.GetBuilderContext(ctx, &count, builder)
	return count, err
}

func (d *DB) GetUserStats(ctx context.Context, telegramID int64) (totalChats int, totalConfessions int, totalReactions int, daysSinceJoined int, err error) {
	chatsQuery := d.Builder.Select("COUNT(*)").From("chat_sessions").Where("user1_id = ? OR user2_id = ?", telegramID, telegramID)
	_ = d.GetBuilderContext(ctx, &totalChats, chatsQuery)

	confQuery := d.Builder.Select("COUNT(*)").From("confessions").Where("author_id = ?", telegramID)
	_ = d.GetBuilderContext(ctx, &totalConfessions, confQuery)

	reactQuery := d.Builder.Select("COUNT(*)").From("confession_reactions cr").
		Join("confessions c ON cr.confession_id = c.id").
		Where("c.author_id = ?", telegramID)
	_ = d.GetBuilderContext(ctx, &totalReactions, reactQuery)

	var createdAt time.Time
	userQuery := d.Builder.Select("created_at").From("users").Where("telegram_id = ?", telegramID)
	err = d.GetBuilderContext(ctx, &createdAt, userQuery)
	daysSinceJoined = int(time.Since(createdAt).Hours() / 24)

	return
}
