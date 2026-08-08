package configs

import (
	"os"
	"paynexus/internal/golibs/bootstrap"
	"paynexus/internal/golibs/database"
	"paynexus/internal/golibs/nats"
	"paynexus/internal/golibs/redis"
	"strconv"
	"time"
)

type AppConfig struct {
	Server   bootstrap.ServerConfig
	Postgres database.DBConfig
	Redis    redis.RedisConfig
	NATS     nats.NATSConfig
}

func LoadConfigFromEnv(serviceName string, defaultPort string) AppConfig {
	return AppConfig{
		Server: bootstrap.ServerConfig{
			ServiceName:     getEnv("SERVICE_NAME", serviceName),
			Port:            getEnv("SERVER_PORT", defaultPort),
			MetricsPort:     getEnv("METRICS_PORT", ":9090"),
			IsDebug:         getEnvAsBool("IS_DEBUG", true),
			ShutdownTimeout: time.Duration(getEnvAsInt("SHUTDOWN_TIMEOUT_SEC", 10)) * time.Second,
		},
		Postgres: database.DBConfig{
			Host:     getEnv("POSTGRES_HOST", "127.0.0.1"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "postgres"),
			Password: getEnv("POSTGRES_PASSWORD", "password"),
			DBName:   getEnv("POSTGRES_DB", "paynexus_db"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
			MaxConns: int32(getEnvAsInt("POSTGRES_MAX_CONNS", 20)),
			MinConns: int32(getEnvAsInt("POSTGRES_MIN_CONNS", 5)),
		},
		Redis: redis.RedisConfig{
			Host:     getEnv("REDIS_HOST", "127.0.0.1"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		NATS: nats.NATSConfig{
			URL:        getEnv("NATS_URL", "nats://localhost:4222"),
			StreamName: getEnv("NATS_STREAM_NAME", "PAYNEXUS_EVENTS"),
		},
	}
}
func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}
func getEnvAsBool(key string, fallback bool) bool {
	valStr := getEnv(key, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return fallback
}
func getEnvAsInt(key string, fallback int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return fallback
}
