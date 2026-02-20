package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/pnj-anonymous-bot/internal/logger"
	"github.com/pnj-anonymous-bot/internal/metrics"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type EvidenceMessage struct {
	SenderID int64  `json:"sender_id"`
	Content  string `json:"content"`
	Type     string `json:"type"`
	SentAt   int64  `json:"sent_at"`
}

type EvidenceService struct {
	db    *database.DB
	redis *redis.Client
}

func NewEvidenceService(db *database.DB, redisClient *redis.Client) *EvidenceService {
	return &EvidenceService{
		db:    db,
		redis: redisClient,
	}
}

func (s *EvidenceService) LogMessage(ctx context.Context, sessionID int64, senderID int64, content string, msgType string) {
	key := fmt.Sprintf("chat_evidence:%d", sessionID)

	msg := EvidenceMessage{
		SenderID: senderID,
		Content:  content,
		Type:     msgType,
		SentAt:   time.Now().Unix(),
	}

	raw, err := json.Marshal(msg)
	if err != nil {
		logger.Warn("Failed to marshal evidence message",
			zap.Int64("session_id", sessionID),
			zap.Error(err),
		)
		return
	}

	pipe := s.redis.Pipeline()
	pipe.RPush(ctx, key, raw)
	pipe.LTrim(ctx, key, -100, -1)
	pipe.Expire(ctx, key, 72*time.Hour)
	if _, err := pipe.Exec(ctx); err != nil {
		metrics.RedisErrors.WithLabelValues("evidence_log").Inc()
		logger.Warn("Failed to log evidence message",
			zap.Int64("session_id", sessionID),
			zap.Error(err),
		)
	}
}

func (s *EvidenceService) GetEvidence(ctx context.Context, sessionID int64) (string, error) {
	key := fmt.Sprintf("chat_evidence:%d", sessionID)
	items, err := s.redis.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		metrics.RedisErrors.WithLabelValues("evidence_get").Inc()
		return "", err
	}

	if len(items) == 0 {
		return "No recent chat logs found for this session.", nil
	}

	var evidence string
	for _, item := range items {
		var msg EvidenceMessage
		if err := json.Unmarshal([]byte(item), &msg); err == nil {
			timeStr := time.Unix(msg.SentAt, 0).Format("15:04:05")
			evidence += fmt.Sprintf("[%s] User %d: (%s) %s\n", timeStr, msg.SenderID, msg.Type, msg.Content)
		}
	}

	return evidence, nil
}

func (s *EvidenceService) ClearEvidence(ctx context.Context, sessionID int64) {
	key := fmt.Sprintf("chat_evidence:%d", sessionID)
	if err := s.redis.Del(ctx, key).Err(); err != nil {
		metrics.RedisErrors.WithLabelValues("evidence_clear").Inc()
		logger.Warn("Failed to clear evidence",
			zap.Int64("session_id", sessionID),
			zap.Error(err),
		)
	}
}
