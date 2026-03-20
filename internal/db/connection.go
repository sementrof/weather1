package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sementrof/Weather/internal/config"
	"go.uber.org/zap"
)

func Connection(cfg config.AppConfig, logger *zap.Logger) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)
	poolConfig, err := pgxpool.ParseConfig(connStr)

	if err != nil {
		logger.Error("Failed to parse database config", zap.Error(err))
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	poolConfig.MaxConns = 20
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < 5; i++ {
		pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
		if err == nil {
			err = pool.Ping(ctx)
			if err == nil {
				logger.Info("Successfully connected to database",
					zap.String("host", cfg.DBHost),
					zap.String("dbname", cfg.DBName),
					zap.Int32("max_conns", poolConfig.MaxConns),
				)
				return pool, nil
			}
		}
		logger.Warn("Failed to connect to database",
			zap.Int("attempt", i+1),
			zap.Int("max_attempts", 5),
			zap.Error(err),
		)

		timex := 1 * time.Duration(1<<uint(i))
		select {
		case <-ctx.Done():
			logger.Error("Database connection timeout", zap.Error(ctx.Err()))
			return nil, fmt.Errorf("connection timeout: %w", ctx.Err())
		case <-time.After(timex):
			continue
		}
	}
	return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", 5, err)
}
