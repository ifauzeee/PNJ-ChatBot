package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pnj-anonymous-bot/internal/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisService struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisService() *RedisService {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr: url,
	})

	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		logger.Warn("⚠️ Redis connection failed", zap.Error(err))
	} else {
		logger.Info("✅ Redis connected successfully")
	}

	return &RedisService{
		client: client,
		ctx:    ctx,
	}
}

func (r *RedisService) AddToQueue(telegramID int64, data string) error {
	key := "chat_queue"
	val := fmt.Sprintf("%d:%s", telegramID, data)
	return r.client.RPush(r.ctx, key, val).Err()
}

func (r *RedisService) GetFromQueue() (string, error) {
	key := "chat_queue"
	return r.client.LPop(r.ctx, key).Result()
}

func (r *RedisService) RemoveFromQueue(telegramID int64) error {
	key := "chat_queue"

	items, err := r.client.LRange(r.ctx, key, 0, -1).Result()
	if err != nil {
		return err
	}

	for _, item := range items {
		if strings.HasPrefix(item, fmt.Sprintf("%d:", telegramID)) {
			if err := r.client.LRem(r.ctx, key, 0, item).Err(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *RedisService) AllowPerMinute(action string, telegramID int64, limit int) (bool, int, error) {
	if limit <= 0 {
		return true, 0, nil
	}

	key := fmt.Sprintf("rate_limit:%s:%d", action, telegramID)

	count, err := r.client.Incr(r.ctx, key).Result()
	if err != nil {
		return false, 0, err
	}

	if count == 1 {
		if err := r.client.Expire(r.ctx, key, time.Minute).Err(); err != nil {
			return false, 0, err
		}
	}

	if count <= int64(limit) {
		return true, 0, nil
	}

	ttl, err := r.client.TTL(r.ctx, key).Result()
	if err != nil {
		return false, 1, nil
	}

	retryAfter := int(ttl.Seconds())
	if retryAfter < 1 {
		retryAfter = 1
	}

	return false, retryAfter, nil
}
