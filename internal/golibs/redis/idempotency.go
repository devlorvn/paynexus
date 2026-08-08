package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type IdempotenceEngine struct {
	client *redis.Client
}

func NewIdempotencyEngine(client *redis.Client) *IdempotenceEngine {
	return &IdempotenceEngine{
		client: client,
	}
}

func (ie *IdempotenceEngine) LockKey(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	lockKey := fmt.Sprintf("idemp:lock:%s", key)
	return ie.client.SetNX(ctx, lockKey, "1", ttl).Result()
}

func (ie *IdempotenceEngine) SaveResponse(ctx context.Context, key string, respBytes []byte, ttl time.Duration) error {
	respKey := fmt.Sprintf("idemp:resp:%s", key)
	return ie.client.Set(ctx, respKey, respBytes, ttl).Err()
}

func (ie *IdempotenceEngine) GetResponse(ctx context.Context, key string) ([]byte, bool, error) {
	respKey := fmt.Sprintf("idemp:resp:%s", key)
	val, err := ie.client.Get(ctx, respKey).Result()

	if err == redis.Nil {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, err
	}
	return []byte(val), true, nil

}

func (ie *IdempotenceEngine) DeleteResponse(ctx context.Context, key string) error {
	respKey := fmt.Sprintf("idemp:resp:%s", key)
	return ie.client.Del(ctx, respKey).Err()
}
