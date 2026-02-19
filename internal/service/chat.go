package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/pnj-anonymous-bot/internal/logger"
	"github.com/pnj-anonymous-bot/internal/models"
	"go.uber.org/zap"
)

type QueueItem struct {
	TelegramID int64  `json:"telegram_id"`
	Dept       string `json:"dept"`
	Gender     string `json:"gender"`
	Year       int    `json:"year"`
	JoinedAt   int64  `json:"joined_at"`
}

type ChatService struct {
	db                 *database.DB
	redis              *RedisService
	maxSearchPerMinute int
}

func NewChatService(db *database.DB, redis *RedisService, maxSearchPerMinute int) *ChatService {
	return &ChatService{
		db:                 db,
		redis:              redis,
		maxSearchPerMinute: maxSearchPerMinute,
	}
}

func (s *ChatService) SearchPartner(ctx context.Context, telegramID int64, preferredDept, preferredGender string, preferredYear int) (int64, error) {
	session, err := s.db.GetActiveSession(ctx, telegramID)
	if err != nil {
		return 0, fmt.Errorf("gagal memeriksa sesi: %w", err)
	}
	if session != nil {
		return 0, fmt.Errorf("kamu masih dalam sesi chat. Gunakan /stop untuk menghentikan chat saat ini")
	}

	allowed, retryAfter, rateLimitErr := s.redis.AllowPerMinute(ctx, "search", telegramID, s.maxSearchPerMinute)
	if rateLimitErr != nil {
		logger.Warn("Search rate limiter unavailable", zap.Int64("user_id", telegramID), zap.Error(rateLimitErr))
	} else if !allowed {
		return 0, fmt.Errorf("terlalu sering mencari partner. Coba lagi dalam %d detik", retryAfter)
	}

	queueKey := "chat_queue"
	items, err := s.redis.GetClient().LRange(ctx, queueKey, 0, -1).Result()
	if err != nil {
		return 0, fmt.Errorf("gagal membaca antrian: %w", err)
	}

	invalidItems := make(map[string]struct{})
	for _, raw := range items {
		var item QueueItem
		if err := json.Unmarshal([]byte(raw), &item); err != nil || item.TelegramID == 0 {
			invalidItems[raw] = struct{}{}
			continue
		}

		if item.TelegramID == telegramID {
			return 0, fmt.Errorf("kamu sudah dalam antrian pencarian.")
		}

		if s.isMatch(ctx, item, preferredDept, preferredGender, preferredYear) {

			removed, err := s.redis.GetClient().LRem(ctx, queueKey, 1, raw).Result()
			if err != nil {
				logger.Warn("Failed to remove matched user from queue", zap.Error(err))
				continue
			}
			if removed == 0 {
				logger.Debug("Partner already taken by another user", zap.Int64("partner_id", item.TelegramID))
				continue
			}

			var sessErr error
			_, sessErr = s.db.CreateChatSession(ctx, telegramID, item.TelegramID)
			if sessErr != nil {
				return 0, sessErr
			}

			if err := s.db.SetUserState(ctx, telegramID, models.StateInChat, ""); err != nil {
				logger.Warn("Failed to set user1 state to chat", zap.Error(err))
			}
			if err := s.db.SetUserState(ctx, item.TelegramID, models.StateInChat, ""); err != nil {
				logger.Warn("Failed to set user2 state to chat", zap.Error(err))
			}

			logger.Debug("Chat matched",
				zap.Int64("user1", telegramID),
				zap.Int64("user2", item.TelegramID),
			)
			return item.TelegramID, nil
		}
	}

	for raw := range invalidItems {
		if err := s.redis.GetClient().LRem(ctx, queueKey, 0, raw).Err(); err != nil {
			logger.Warn("Failed to remove invalid queue item", zap.Error(err))
		}
	}

	newItem := QueueItem{
		TelegramID: telegramID,
		Dept:       preferredDept,
		Gender:     preferredGender,
		Year:       preferredYear,
		JoinedAt:   time.Now().Unix(),
	}
	if err := s.redis.AddToQueue(ctx, telegramID, newItem); err != nil {
		return 0, fmt.Errorf("gagal menambahkan ke antrian: %w", err)
	}

	if err := s.db.SetUserState(ctx, telegramID, models.StateSearching, ""); err != nil {
		logger.Warn("Failed to set searching state", zap.Error(err))
	}
	logger.Debug("Added to queue", zap.Int64("user_id", telegramID))
	return 0, nil
}

func (s *ChatService) isMatch(ctx context.Context, item QueueItem, prefDept, prefGender string, prefYear int) bool {
	user, err := s.db.GetUser(ctx, item.TelegramID)
	if err != nil || user == nil {
		logger.Warn("Failed to get user from DB for matching", zap.Int64("user_id", item.TelegramID), zap.Error(err))
		return false
	}
	if !user.IsVerified || user.IsBanned {
		logger.Debug("User in queue is not verified or is banned, skipping match",
			zap.Int64("user_id", item.TelegramID),
			zap.Bool("is_verified", user.IsVerified),
			zap.Bool("is_banned", user.IsBanned),
		)
		return false
	}

	if prefDept != "" && string(user.Department) != prefDept {
		logger.Debug("User in queue does not match searcher's preferred department",
			zap.Int64("user_id", item.TelegramID),
			zap.String("user_dept", string(user.Department)),
			zap.String("pref_dept", prefDept),
		)
		return false
	}
	if prefGender != "" && string(user.Gender) != prefGender {
		logger.Debug("User in queue does not match searcher's preferred gender",
			zap.Int64("user_id", item.TelegramID),
			zap.String("user_gender", string(user.Gender)),
			zap.String("pref_gender", prefGender),
		)
		return false
	}
	if prefYear != 0 && user.Year != prefYear {
		logger.Debug("User in queue does not match searcher's preferred year",
			zap.Int64("user_id", item.TelegramID),
			zap.Int("user_year", user.Year),
			zap.Int("pref_year", prefYear),
		)
		return false
	}

	logger.Debug("User in queue matches searcher's preferences", zap.Int64("user_id", item.TelegramID))
	return true
}

func (s *ChatService) StopChat(ctx context.Context, telegramID int64) (int64, error) {
	_ = s.redis.RemoveFromQueue(ctx, telegramID)

	session, err := s.db.GetActiveSession(ctx, telegramID)
	if err != nil {
		return 0, err
	}
	if session == nil {
		_ = s.db.SetUserState(ctx, telegramID, models.StateNone, "")
		return 0, nil
	}

	partnerID, err := s.db.StopChat(ctx, telegramID)
	if err != nil {
		return 0, err
	}

	_ = s.db.SetUserState(ctx, telegramID, models.StateNone, "")
	_ = s.db.SetUserState(ctx, partnerID, models.StateNone, "")

	return partnerID, nil
}

func (s *ChatService) NextPartner(ctx context.Context, telegramID int64) (int64, error) {
	partnerID, err := s.StopChat(ctx, telegramID)
	if err != nil {
		return 0, err
	}
	return partnerID, nil
}

func (s *ChatService) GetPartner(ctx context.Context, telegramID int64) (int64, error) {
	return s.db.GetChatPartner(ctx, telegramID)
}

func (s *ChatService) GetPartnerInfo(ctx context.Context, partnerID int64) (string, string, int, error) {
	user, err := s.db.GetUser(ctx, partnerID)
	if err != nil || user == nil {
		return "", "", 0, err
	}
	return string(user.Gender), string(user.Department), user.Year, nil
}

func (s *ChatService) GetQueueCount(ctx context.Context) (int, error) {
	count, err := s.redis.GetClient().LLen(ctx, "chat_queue").Result()
	return int(count), err
}

func (s *ChatService) CancelSearch(ctx context.Context, telegramID int64) error {
	_ = s.redis.RemoveFromQueue(ctx, telegramID)
	_ = s.db.SetUserState(ctx, telegramID, models.StateNone, "")
	return nil
}

func (s *ChatService) ProcessQueueTimeout(ctx context.Context, timeoutSeconds int) ([]int64, error) {
	if timeoutSeconds <= 0 {
		return nil, nil
	}

	queueKey := "chat_queue"
	items, err := s.redis.GetClient().LRange(ctx, queueKey, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	updatedIDs := make([]int64, 0)
	invalidItems := make(map[string]struct{})

	for _, raw := range items {
		var item QueueItem
		if err := json.Unmarshal([]byte(raw), &item); err != nil || item.TelegramID == 0 {
			invalidItems[raw] = struct{}{}
			continue
		}

		changed := false
		if item.JoinedAt <= 0 {
			item.JoinedAt = now
			changed = true
		}

		hasFilter := item.Dept != "" || item.Gender != "" || item.Year != 0
		if hasFilter && now-item.JoinedAt >= int64(timeoutSeconds) {
			item.Dept = ""
			item.Gender = ""
			item.Year = 0
			changed = true
			updatedIDs = append(updatedIDs, item.TelegramID)
		}

		if !changed {
			continue
		}

		if err := s.redis.AddToQueue(ctx, item.TelegramID, item); err != nil {
			logger.Warn("Failed to update queue item", zap.Int64("user_id", item.TelegramID), zap.Error(err))
		}
	}

	for raw := range invalidItems {
		if err := s.redis.GetClient().LRem(ctx, queueKey, 0, raw).Err(); err != nil {
			logger.Warn("Failed to remove invalid queue item", zap.Error(err))
		}
	}

	return updatedIDs, nil
}
