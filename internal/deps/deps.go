package deps

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sementrof/Weather/internal/config"
	"github.com/sementrof/Weather/internal/db"
	"github.com/sementrof/Weather/internal/logger"
	"github.com/sementrof/Weather/internal/weather"
	"go.uber.org/zap"
)

type DB struct {
	Settings db.SettingsQuery
}

type Dependencies struct {
	DB                     DB
	Pool                   *pgxpool.Pool
	Logger                 *zap.Logger
	Weather                weather.Client
	WeatherCacheTTLSeconds int64
	APISecretKey           string
}

func ProvideDependencies(ctx context.Context, cfg config.AppConfig) (*Dependencies, error) {
	logger := logger.NewLogger()

	pool, err := db.Connection(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to init db", zap.Error(err))
		return nil, err
	}

	deps := &Dependencies{
		DB: DB{
			Settings: db.NewSettingsQuery(pool, logger),
		},
		Pool:                   pool,
		Logger:                 logger,
		Weather:                weather.NewOpenWeatherClient(cfg.OpenWeatherAPIKey, cfg.OpenWeatherBaseURL),
		WeatherCacheTTLSeconds: cfg.WeatherCacheTTLSeconds,
		APISecretKey:           cfg.APISecretKey,
	}

	if err := pool.Ping(ctx); err != nil {
		logger.Fatal("Failed to ping database", zap.Error(err))
		pool.Close()
		return nil, err
	}

	// Create tables if they don't exist.
	if err := db.Migrate(ctx, pool, logger); err != nil {
		pool.Close()
		return nil, err
	}

	logger.Info("Dependencies initialized successfully")
	return deps, nil
}

func (d *Dependencies) Cleanup() {
	d.Logger.Info("Cleaning up dependencies")
	d.Logger.Sync()
	d.Pool.Close()
}
