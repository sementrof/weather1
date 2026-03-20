package weather

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

type Client interface {
	FetchCurrentWeather(ctx context.Context, city string) (tempC float64, condition string, err error)
}

type openWeatherClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewOpenWeatherClient(apiKey, baseURL string) Client {
	return &openWeatherClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type openWeatherResponse struct {
	Main struct {
		Temp float64 `json:"temp"`
	} `json:"main"`
	Weather []struct {
		Description string `json:"description"`
		Main        string `json:"main"`
	} `json:"weather"`
}

func (c *openWeatherClient) FetchCurrentWeather(ctx context.Context, city string) (float64, string, error) {
	// "Dumb" URL building: just concatenate + escape.
	endpoint := c.baseURL + "/data/2.5/weather" +
		"?q=" + url.QueryEscape(city) +
		"&appid=" + url.QueryEscape(c.apiKey) +
		"&units=metric"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	var parsed openWeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return 0, "", err
	}

	condition := ""
	if len(parsed.Weather) > 0 {
		condition = parsed.Weather[0].Description
	}
	return parsed.Main.Temp, condition, nil
}
