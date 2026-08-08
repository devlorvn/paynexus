package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

func DefaultConfig() RedisConfig {
	return RedisConfig{
		Host:     "localhost",
		Port:     "6379",
		Password: "",
		DB:       0,
	}
}

type RedisClient struct {
	Client *redis.Client
	Logger *zap.Logger
}

func NewRedisClient(ctx context.Context, cfg RedisConfig, logger *zap.Logger) (*RedisClient, error) {
	logger = logger.Named("redis")

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Error("Failed to connect Redis", zap.String("addr", addr), zap.Error(err))
		return nil, err
	}

	logger.Info("Connected to Redis", zap.String("addr", addr))
	return &RedisClient{
		Client: rdb,
		Logger: logger,
	}, nil
}

func (rc *RedisClient) Close() {
	if err := rc.Client.Close(); err != nil {
		rc.Logger.Error("Failed to close Redis", zap.Error(err))
	}
	rc.Logger.Info("Closed Redis")
}
