package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/redis/go-redis/v9"
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
	ctx   context.Context
}

func NewEvidenceService(db *database.DB, redisClient *redis.Client) *EvidenceService {
	return &EvidenceService{
		db:    db,
		redis: redisClient,
		ctx:   context.Background(),
	}
}

func (s *EvidenceService) LogMessage(sessionID int64, senderID int64, content string, msgType string) {
	key := fmt.Sprintf("chat_evidence:%d", sessionID)

	msg := EvidenceMessage{
		SenderID: senderID,
		Content:  content,
		Type:     msgType,
		SentAt:   time.Now().Unix(),
	}

	raw, _ := json.Marshal(msg)

	pipe := s.redis.Pipeline()
	pipe.RPush(s.ctx, key, raw)
	pipe.LTrim(s.ctx, key, -20, -1)
	pipe.Expire(s.ctx, key, 24*time.Hour)
	_, _ = pipe.Exec(s.ctx)
}

func (s *EvidenceService) GetEvidence(sessionID int64) (string, error) {
	key := fmt.Sprintf("chat_evidence:%d", sessionID)
	items, err := s.redis.LRange(s.ctx, key, 0, -1).Result()
	if err != nil {
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
