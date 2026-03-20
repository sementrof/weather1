package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/sementrof/Weather/internal/deps"
	"go.uber.org/zap"

	"github.com/jackc/pgx/v5"
	"github.com/sementrof/Weather/internal/db"
)

type ApiInterface interface {
	CreateUsersPost(w http.ResponseWriter, r *http.Request)
	GetWeather(w http.ResponseWriter, r *http.Request)
}

type ApiImplemented struct {
	deps *deps.Dependencies
}

func NewApi(deps *deps.Dependencies) *ApiImplemented {
	return &ApiImplemented{
		deps: deps,
	}
}

func (im *ApiImplemented) CreateUsersPost(w http.ResponseWriter, r *http.Request) {
	type createUserRequest struct {
		Name string `json:"name"`
		City string `json:"city"`
	}

	var input createUserRequest
	ctx := context.Background()
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	if input.Name == "" || input.City == "" {
		http.Error(w, "name and city are required", http.StatusBadRequest)
		return
	}

	if len(input.Name) > 100 || len(input.City) > 100 {
		http.Error(w, "name and city must be under 100 characters", http.StatusBadRequest)
		return
	}

	deviceID, err := im.deps.DB.Settings.CreateUserWithDevice(ctx, input.Name, input.City)
	if err != nil {
		im.deps.Logger.Error("Failed to create user/device", zap.Error(err))
		http.Error(w, "Failed to create user/device", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"device_id": deviceID,
	})
}

func (im *ApiImplemented) GetWeather(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	deviceIDStr := r.URL.Query().Get("device_id")
	if deviceIDStr == "" {
		deviceIDStr = r.Header.Get("X-Device-Id")
	}

	now := time.Now().UTC()
	var deviceID int64
	var city string
	var err error

	if deviceIDStr != "" {
		deviceID, err = strconv.ParseInt(deviceIDStr, 10, 64)
		if err != nil {
			http.Error(w, "device_id must be an integer", http.StatusBadRequest)
			return
		}

		city, err = im.deps.DB.Settings.GetDeviceCity(ctx, deviceID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, "device not found", http.StatusNotFound)
				return
			}
			im.deps.Logger.Error("Failed to get device city", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	} else {
		deviceID, city, err = im.deps.DB.Settings.GetFirstDevice(ctx)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, "no devices configured", http.StatusNotFound)
				return
			}
			im.deps.Logger.Error("Failed to get first device", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	cache, ok, err := im.deps.DB.Settings.GetValidWeatherCache(ctx, deviceID, now)
	if err != nil {
		im.deps.Logger.Error("Failed to get weather cache", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if ok && cache != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"city":       city,
			"temp":       cache.Temp,
			"condition":  cache.Condition,
			"from_cache": true,
		})
		return
	}

	tempC, condition, err := im.deps.Weather.FetchCurrentWeather(ctx, city)
	if err != nil {
		im.deps.Logger.Error("OpenWeatherMap request failed", zap.Error(err))
		http.Error(w, "Failed to fetch weather", http.StatusBadGateway)
		return
	}

	ttl := time.Duration(im.deps.WeatherCacheTTLSeconds) * time.Second
	expiresAt := now.Add(ttl)
	cacheToSave := &db.WeatherCache{
		DeviceID:  deviceID,
		Temp:      tempC,
		Condition: condition,
		FetchedAt: now,
		ExpiresAt: expiresAt,
	}

	if err := im.deps.DB.Settings.UpsertWeatherCache(ctx, cacheToSave); err != nil {
		im.deps.Logger.Error("Failed to upsert weather cache", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"city":       city,
		"temp":       tempC,
		"condition":  condition,
		"from_cache": false,
	})
}
