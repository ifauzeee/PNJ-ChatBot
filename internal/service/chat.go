package service

import (
	"encoding/json"
	"fmt"

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

func (s *ChatService) SearchPartner(telegramID int64, preferredDept, preferredGender string, preferredYear int) (int64, error) {
	session, err := s.db.GetActiveSession(telegramID)
	if err != nil {
		return 0, fmt.Errorf("gagal memeriksa sesi: %w", err)
	}
	if session != nil {
		return 0, fmt.Errorf("kamu masih dalam sesi chat. Gunakan /stop untuk menghentikan chat saat ini")
	}

	allowed, retryAfter, rateLimitErr := s.redis.AllowPerMinute("search", telegramID, s.maxSearchPerMinute)
	if rateLimitErr != nil {
		logger.Warn("Search rate limiter unavailable", zap.Int64("user_id", telegramID), zap.Error(rateLimitErr))
	} else if !allowed {
		return 0, fmt.Errorf("terlalu sering mencari partner. Coba lagi dalam %d detik", retryAfter)
	}

	queueKey := "chat_queue"
	items, err := s.redis.client.LRange(s.redis.ctx, queueKey, 0, -1).Result()
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

		if s.isMatch(item, preferredDept, preferredGender, preferredYear) {

			s.redis.client.LRem(s.redis.ctx, queueKey, 1, raw)

			_, err := s.db.CreateChatSession(telegramID, item.TelegramID)
			if err != nil {
				return 0, err
			}

			s.db.SetUserState(telegramID, models.StateInChat, "")
			s.db.SetUserState(item.TelegramID, models.StateInChat, "")

			logger.Debug("Chat matched",
				zap.Int64("user1", telegramID),
				zap.Int64("user2", item.TelegramID),
			)
			return item.TelegramID, nil
		}
	}

	for raw := range invalidItems {
		if err := s.redis.client.LRem(s.redis.ctx, queueKey, 0, raw).Err(); err != nil {
			logger.Warn("Failed to remove invalid queue item", zap.Error(err))
		}
	}

	newItem := QueueItem{
		TelegramID: telegramID,
		Dept:       preferredDept,
		Gender:     preferredGender,
		Year:       preferredYear,
	}
	raw, _ := json.Marshal(newItem)
	if err := s.redis.client.RPush(s.redis.ctx, queueKey, raw).Err(); err != nil {
		return 0, fmt.Errorf("gagal menambahkan ke antrian: %w", err)
	}

	s.db.SetUserState(telegramID, models.StateSearching, "")
	logger.Debug("Added to queue", zap.Int64("user_id", telegramID))
	return 0, nil
}

func (s *ChatService) isMatch(item QueueItem, prefDept, prefGender string, prefYear int) bool {

	user, _ := s.db.GetUser(item.TelegramID)
	if user == nil {
		return false
	}
	if user.IsBanned {
		return false
	}

	if prefDept != "" && string(user.Department) != prefDept {
		return false
	}
	if prefGender != "" && string(user.Gender) != prefGender {
		return false
	}
	if prefYear != 0 && user.Year != prefYear {
		return false
	}

	return true
}

func (s *ChatService) StopChat(telegramID int64) (int64, error) {
	s.redis.RemoveFromQueue(telegramID)

	session, err := s.db.GetActiveSession(telegramID)
	if err != nil {
		return 0, err
	}
	if session == nil {
		s.db.SetUserState(telegramID, models.StateNone, "")
		return 0, nil
	}

	partnerID, err := s.db.StopChat(telegramID)
	if err != nil {
		return 0, err
	}

	s.db.SetUserState(telegramID, models.StateNone, "")
	s.db.SetUserState(partnerID, models.StateNone, "")

	return partnerID, nil
}

func (s *ChatService) NextPartner(telegramID int64) (int64, error) {
	partnerID, err := s.StopChat(telegramID)
	if err != nil {
		return 0, err
	}
	return partnerID, nil
}

func (s *ChatService) GetPartner(telegramID int64) (int64, error) {
	return s.db.GetChatPartner(telegramID)
}

func (s *ChatService) GetPartnerInfo(partnerID int64) (string, string, int, error) {
	user, err := s.db.GetUser(partnerID)
	if err != nil || user == nil {
		return "", "", 0, err
	}
	return string(user.Gender), string(user.Department), user.Year, nil
}

func (s *ChatService) GetQueueCount() (int, error) {
	count, err := s.redis.client.LLen(s.redis.ctx, "chat_queue").Result()
	return int(count), err
}

func (s *ChatService) CancelSearch(telegramID int64) error {
	s.redis.RemoveFromQueue(telegramID)
	s.db.SetUserState(telegramID, models.StateNone, "")
	return nil
}

func (s *ChatService) ProcessQueueTimeout(timeoutSeconds int) ([]int64, error) {
	return nil, nil
}
