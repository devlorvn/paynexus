package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type QueryExecer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type DBWrapper struct {
	Pool   *pgxpool.Pool
	Logger *zap.Logger
}

func NewDBWrapper(ctx context.Context, cfg DBConfig, logger *zap.Logger) (*DBWrapper, error) {
	connConfig, err := pgxpool.ParseConfig(cfg.ConnectionString())
	if err != nil {
		logger.Error("Fail to parse DSN PostgreSQL", zap.Error(err))
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, connConfig)
	if err != nil {
		logger.Error("Cannot connect to Postgres ", zap.Error(err))
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		logger.Error("Ping to Postgres failed", zap.Error(err))
		return nil, err
	}
	logger.Info("Connect to Postgres successfully!", zap.String("dbname", cfg.DBName))
	return &DBWrapper{
		Pool:   pool,
		Logger: logger,
	}, nil
}

// SetRLSContext đặt Postgres session variable permission.resource_path từ Context
func SetRLSContext(ctx context.Context, tx pgx.Tx, resourcePath string) error {
	if resourcePath == "" {
		resourcePath = "/org/default/" // Mặc định nếu chưa truyền context
	}
	_, err := tx.Exec(ctx, "SELECT set_config('permission.resource_path', $1, true);", resourcePath)
	return err
}

// Close đóng toàn bộ connection pool khi shutdown server
func (db *DBWrapper) Close() {
	if db.Pool != nil {
		db.Pool.Close()
		db.Logger.Info("Đã đóng kết nối PostgreSQL Connection Pool.")
	}
}
