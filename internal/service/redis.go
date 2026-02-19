package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pnj-anonymous-bot/internal/logger"
	"github.com/pnj-anonymous-bot/internal/metrics"
	"github.com/pnj-anonymous-bot/internal/resilience"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisService struct {
	client *redis.Client
	cb     *resilience.CircuitBreaker
}

func NewRedisService() *RedisService {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "localhost:6379"
	}

	opts := &redis.Options{
		Addr: url,
	}

	if strings.HasPrefix(url, "redis://") || strings.HasPrefix(url, "rediss://") {
		var err error
		opts, err = redis.ParseURL(url)
		if err != nil {
			logger.Warn("⚠️ Failed to parse REDIS_URL, falling back to default options", zap.Error(err))
			opts = &redis.Options{Addr: "localhost:6379"}
		}
	}

	opts.DialTimeout = 5 * time.Second
	opts.ReadTimeout = 3 * time.Second
	opts.WriteTimeout = 3 * time.Second
	opts.PoolSize = 20
	opts.MinIdleConns = 5

	client := redis.NewClient(opts)

	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		logger.Warn("⚠️ Redis connection failed", zap.Error(err))
	} else {
		logger.Info("✅ Redis connected successfully")
	}

	cbConfig := resilience.CircuitBreakerConfig{
		Name:                "redis",
		FailureThreshold:    10,
		ResetTimeout:        15 * time.Second,
		HalfOpenMaxAttempts: 3,
	}

	return &RedisService{
		client: client,
		cb:     resilience.NewCircuitBreaker(cbConfig),
	}
}

func (r *RedisService) GetClient() *redis.Client {
	return r.client
}

func (r *RedisService) Ping(ctx context.Context) error {
	return r.cb.Execute(func() error {
		_, err := r.client.Ping(ctx).Result()
		if err != nil {
			metrics.RedisErrors.WithLabelValues("ping").Inc()
		}
		return err
	})
}

func (r *RedisService) Close() error {
	return r.client.Close()
}

func (r *RedisService) AddToQueue(ctx context.Context, telegramID int64, val interface{}) error {
	return r.cb.Execute(func() error {
		key := "chat_queue"
		trackKey := "chat_queue_track"

		raw, err := json.Marshal(val)
		if err != nil {
			return err
		}

		_ = r.removeFromQueueInternal(ctx, telegramID)

		if err := r.client.RPush(ctx, key, raw).Err(); err != nil {
			metrics.RedisErrors.WithLabelValues("queue_push").Inc()
			return err
		}
		if err := r.client.HSet(ctx, trackKey, fmt.Sprintf("%d", telegramID), raw).Err(); err != nil {
			metrics.RedisErrors.WithLabelValues("queue_track").Inc()
			return err
		}
		return nil
	})
}

func (r *RedisService) GetFromQueue(ctx context.Context) ([]byte, error) {
	var result []byte
	err := r.cb.Execute(func() error {
		var err error
		result, err = r.client.LPop(ctx, "chat_queue").Bytes()
		if err != nil && err != redis.Nil {
			metrics.RedisErrors.WithLabelValues("queue_pop").Inc()
		}
		return err
	})
	return result, err
}

func (r *RedisService) RemoveFromQueue(ctx context.Context, telegramID int64) error {
	return r.cb.Execute(func() error {
		return r.removeFromQueueInternal(ctx, telegramID)
	})
}

func (r *RedisService) removeFromQueueInternal(ctx context.Context, telegramID int64) error {
	key := "chat_queue"
	trackKey := "chat_queue_track"
	idStr := fmt.Sprintf("%d", telegramID)

	raw, err := r.client.HGet(ctx, trackKey, idStr).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		metrics.RedisErrors.WithLabelValues("queue_track_get").Inc()
		return err
	}

	if err := r.client.LRem(ctx, key, 0, raw).Err(); err != nil {
		metrics.RedisErrors.WithLabelValues("queue_rem").Inc()
	}
	return r.client.HDel(ctx, trackKey, idStr).Err()
}

func (r *RedisService) AllowPerMinute(ctx context.Context, action string, telegramID int64, limit int) (bool, int, error) {
	if limit <= 0 {
		return true, 0, nil
	}

	var allowed bool
	var retryAfter int

	err := r.cb.Execute(func() error {
		key := fmt.Sprintf("rate_limit:%s:%d", action, telegramID)

		count, err := r.client.Incr(ctx, key).Result()
		if err != nil {
			metrics.RedisErrors.WithLabelValues("rate_limit_incr").Inc()
			return err
		}

		if count == 1 {
			if err := r.client.Expire(ctx, key, time.Minute).Err(); err != nil {
				metrics.RedisErrors.WithLabelValues("rate_limit_expire").Inc()
				return err
			}
		}

		if count <= int64(limit) {
			allowed = true
			retryAfter = 0
			return nil
		}

		metrics.RateLimitHitsTotal.WithLabelValues(action).Inc()

		ttl, err := r.client.TTL(ctx, key).Result()
		if err != nil {
			retryAfter = 1
			return nil
		}

		retryAfter = int(ttl.Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}

		allowed = false
		return nil
	})

	if err != nil {

		logger.Warn("Rate limiter unavailable, failing open",
			zap.String("action", action),
			zap.Error(err),
		)
		return true, 0, err
	}

	return allowed, retryAfter, nil
}
