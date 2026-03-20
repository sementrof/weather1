package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	Port          string
	DBHost        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBPort        string
	AppDBUser     string
	AppDBPassword string

	OpenWeatherAPIKey      string
	OpenWeatherBaseURL     string
	WeatherCacheTTLSeconds int64
	APISecretKey           string
}

func LoadConfig() AppConfig {
	// Try both common locations to make local runs/debug sessions work.
	if err := godotenv.Load(".env"); err != nil {
		if err2 := godotenv.Load("../.env"); err2 != nil {
			panic("failed to load .env")
		}
	}

	ttl := int64(600)
	if v := os.Getenv("WEATHER_CACHE_TTL_SECONDS"); v != "" {
		// If parsing fails - fall back to default.
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed > 0 {
			ttl = parsed
		}
	}

	return AppConfig{
		Port:          os.Getenv("PORT"),
		DBHost:        os.Getenv("DBHost"),
		DBUser:        os.Getenv("DB_USER"),
		DBPassword:    os.Getenv("DB_PASSWORD"),
		DBPort:        os.Getenv("DB_PORT"),
		DBName:        os.Getenv("POSTGRES_DB"),
		AppDBUser:     os.Getenv("APP_DB_USER"),
		AppDBPassword: os.Getenv("APP_DB_PASSWORD"),

		OpenWeatherAPIKey:      os.Getenv("OPENWEATHERMAP_API_KEY"),
		OpenWeatherBaseURL:     getOpenWeatherBaseURL(os.Getenv("OPENWEATHERMAP_BASE_URL")),
		WeatherCacheTTLSeconds: ttl,
		APISecretKey:           os.Getenv("API_SECRET_KEY"),
	}
}

func getOpenWeatherBaseURL(v string) string {
	if v == "" {
		return "https://api.openweathermap.org"
	}
	return v
}
