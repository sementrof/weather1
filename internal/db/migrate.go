package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func Migrate(ctx context.Context, pool *pgxpool.Pool, logger *zap.Logger) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS "user" (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS "device" (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
			city TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS weather_cache (
			device_id BIGINT PRIMARY KEY REFERENCES "device"(id) ON DELETE CASCADE,
			temp DOUBLE PRECISION NOT NULL,
			condition TEXT NOT NULL,
			fetched_at TIMESTAMPTZ NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL
		)`,
	}

	for _, stmt := range stmts {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("migrate failed: %w", err)
		}
	}

	logger.Info("Database migration completed")
	return nil
}
