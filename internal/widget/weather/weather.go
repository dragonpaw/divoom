// Package weather is a Widget that reads current conditions from Open-Meteo
// (free, no API key). Output looks like `68° partly cloudy`.
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client fetches current weather for a fixed lat/lon. Reuses one HTTP
// client across calls.
type Client struct {
	Lat, Lon string
	HTTP     *http.Client
}

// New builds a client. Lat/Lon are passed as strings so the caller can
// hand off env-var values without re-parsing.
func New(lat, lon string) *Client {
	return &Client{
		Lat:  lat,
		Lon:  lon,
		HTTP: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Name() string { return "weather" }

type response struct {
	Current struct {
		Temperature2m       float64 `json:"temperature_2m"`
		ApparentTemperature float64 `json:"apparent_temperature"`
		WeatherCode         int     `json:"weather_code"`
	} `json:"current"`
}

// Fetch hits Open-Meteo and returns a one-line summary.
func (c *Client) Fetch(ctx context.Context) (string, error) {
	u, err := url.Parse("https://api.open-meteo.com/v1/forecast")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("latitude", c.Lat)
	q.Set("longitude", c.Lon)
	q.Set("current", "temperature_2m,weather_code,apparent_temperature")
	q.Set("temperature_unit", "fahrenheit")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("open-meteo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("open-meteo http %d", resp.StatusCode)
	}

	var body response
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}

	return fmt.Sprintf("%d° %s",
		int(body.Current.Temperature2m+0.5),
		describeWMO(body.Current.WeatherCode),
	), nil
}

// describeWMO turns Open-Meteo's WMO weather code into a short label.
// Reference: https://open-meteo.com/en/docs#api_form (WMO Weather interpretation codes).
func describeWMO(code int) string {
	switch {
	case code == 0:
		return "clear"
	case code == 1:
		return "mostly clear"
	case code == 2:
		return "partly cloudy"
	case code == 3:
		return "overcast"
	case code == 45 || code == 48:
		return "fog"
	case code >= 51 && code <= 57:
		return "drizzle"
	case code >= 61 && code <= 67:
		return "rain"
	case code >= 71 && code <= 77:
		return "snow"
	case code >= 80 && code <= 82:
		return "rain showers"
	case code >= 85 && code <= 86:
		return "snow showers"
	case code >= 95:
		return "thunderstorm"
	default:
		return "—"
	}
}
