package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
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
		log.Printf("⚠️ Redis connection failed: %v", err)
	} else {
		log.Println("✅ Redis connected successfully")
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
			r.client.LRem(r.ctx, key, 0, item)
		}
	}
	return nil
}
