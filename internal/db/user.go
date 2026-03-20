package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type User struct {
	ID   int64
	Name string
}

type Device struct {
	ID     int64
	UserID int64
	City   string
}

type WeatherCache struct {
	DeviceID  int64
	Temp      float64
	Condition string
	FetchedAt time.Time
	ExpiresAt time.Time
}

type SettingsQuery interface {
	CreateUserWithDevice(ctx context.Context, name string, city string) (deviceID int64, err error)
	GetDeviceCity(ctx context.Context, deviceID int64) (city string, err error)
	GetFirstDevice(ctx context.Context) (deviceID int64, city string, err error)

	GetValidWeatherCache(ctx context.Context, deviceID int64, now time.Time) (*WeatherCache, bool, error)
	UpsertWeatherCache(ctx context.Context, cache *WeatherCache) error
}

type settingsQuery struct {
	runner *pgxpool.Pool
	logger *zap.Logger
}

func NewSettingsQuery(runner *pgxpool.Pool, logger *zap.Logger) SettingsQuery {
	return &settingsQuery{
		runner: runner,
		logger: logger,
	}
}

func (q settingsQuery) CreateUserWithDevice(ctx context.Context, name string, city string) (int64, error) {
	tx, err := q.runner.Begin(ctx)
	if err != nil {
		return 0, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// "user" и "device" специально оставляем как в ТЗ, экранируем кавычками.
	var userID int64
	if err := tx.QueryRow(ctx, `INSERT INTO "user"(name) VALUES ($1) RETURNING id`, name).Scan(&userID); err != nil {
		return 0, err
	}

	var deviceID int64
	if err := tx.QueryRow(
		ctx,
		`INSERT INTO "device"(user_id, city) VALUES ($1, $2) RETURNING id`,
		userID,
		city,
	).Scan(&deviceID); err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return deviceID, nil
}

func (q settingsQuery) GetDeviceCity(ctx context.Context, deviceID int64) (string, error) {
	var city string
	err := q.runner.QueryRow(ctx, `SELECT city FROM "device" WHERE id = $1`, deviceID).Scan(&city)
	if err != nil {
		return "", err
	}
	return city, nil
}

func (q settingsQuery) GetFirstDevice(ctx context.Context) (int64, string, error) {
	var deviceID int64
	var city string
	err := q.runner.QueryRow(ctx, `SELECT id, city FROM "device" ORDER BY id ASC LIMIT 1`).Scan(&deviceID, &city)
	if err != nil {
		return 0, "", err
	}
	return deviceID, city, nil
}

func (q settingsQuery) GetValidWeatherCache(ctx context.Context, deviceID int64, now time.Time) (*WeatherCache, bool, error) {
	c := &WeatherCache{}
	err := q.runner.QueryRow(
		ctx,
		`SELECT device_id, temp, condition, fetched_at, expires_at
		 FROM weather_cache
		 WHERE device_id = $1 AND expires_at > $2`,
		deviceID,
		now,
	).Scan(&c.DeviceID, &c.Temp, &c.Condition, &c.FetchedAt, &c.ExpiresAt)

	if err != nil {
		// pgx.ErrNoRows => кеш протух/отсутствует
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return c, true, nil
}

func (q settingsQuery) UpsertWeatherCache(ctx context.Context, cache *WeatherCache) error {
	_, err := q.runner.Exec(
		ctx,
		`INSERT INTO weather_cache(device_id, temp, condition, fetched_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (device_id)
		 DO UPDATE SET
		   temp = EXCLUDED.temp,
		   condition = EXCLUDED.condition,
		   fetched_at = EXCLUDED.fetched_at,
		   expires_at = EXCLUDED.expires_at`,
		cache.DeviceID,
		cache.Temp,
		cache.Condition,
		cache.FetchedAt,
		cache.ExpiresAt,
	)
	return err
}
