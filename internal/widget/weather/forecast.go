package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dragonpaw/divoom/internal/widget"
)

// Forecast fetches the next-4-day outlook from Open-Meteo's daily
// forecast (high, low, weather code per day) for a fixed location.
// One http.Client with a 10s timeout is reused across Fetch calls.
//
// The widget output is a single pipe-separated string with one
// segment per day: "<DAY>|<HI>|<LO>|<OUTLOOK>|…" repeated for
// forecastDays days, starting with tomorrow. Today is omitted —
// that's the weather scene's job.
//
// Cache TTL keeps API usage low: daily highs/lows don't change
// minute-to-minute, so refreshing every 30 minutes is plenty for a
// wall display.
type Forecast struct {
	lat, lon string
	unit     string
	http     *http.Client

	mu        sync.Mutex
	lastFetch time.Time
	cached    string
	cachedErr error
}

// forecastDays is how far ahead we render. 4 days fits inside the
// device's 6-Text cap (4 scene Text + 2 always-on = 6).
const forecastDays = 4

// forecastCacheTTL is how stale the cache may be before refetching.
const forecastCacheTTL = 30 * time.Minute

// NewForecast returns a forecast widget for the given coordinates.
// The temperature unit mirrors useFahrenheit's lat/lon rule so it
// stays aligned with the current-weather widget's reading.
func NewForecast(lat, lon string) *Forecast {
	c := New(lat, lon) // reuse its location → unit logic
	return &Forecast{
		lat:  lat,
		lon:  lon,
		unit: c.unit,
		http: &http.Client{Timeout: 10 * time.Second, Transport: ipv4Transport},
	}
}

func (f *Forecast) Name() string { return "weather/forecast" }

type forecastResponse struct {
	Daily struct {
		Time         []string  `json:"time"`
		Max          []float64 `json:"temperature_2m_max"`
		Min          []float64 `json:"temperature_2m_min"`
		WeatherCode  []int     `json:"weathercode"`
	} `json:"daily"`
}

// Fetch returns the next-4-day forecast as a pipe-separated string:
// "DAY|HI|LO|OUTLOOK|DAY|HI|LO|OUTLOOK|…" where DAY is the lowercase
// 3-letter abbreviation ("thu"), HI/LO are integers in the configured
// unit (no degree symbol — the scene adds it), and OUTLOOK comes from
// OutlookFromCode. Tomorrow's row is segment [0..3], the day after is
// [4..7], etc.
func (f *Forecast) Fetch(ctx context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if time.Since(f.lastFetch) < forecastCacheTTL && f.lastFetch != (time.Time{}) {
		return f.cached, f.cachedErr
	}

	url := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast"+
			"?latitude=%s&longitude=%s"+
			"&daily=temperature_2m_max,temperature_2m_min,weathercode"+
			"&temperature_unit=%s&timezone=auto&forecast_days=%d",
		f.lat, f.lon, f.unit, forecastDays+1, // +1 because today is index 0; we want tomorrow onward
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return f.fail(fmt.Errorf("forecast: build request: %w", err))
	}
	resp, err := f.http.Do(req)
	if err != nil {
		return f.fail(fmt.Errorf("forecast: http: %w", err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return f.fail(fmt.Errorf("forecast: http %d", resp.StatusCode))
	}
	var body forecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return f.fail(fmt.Errorf("forecast: decode: %w", err))
	}

	// Open-Meteo returns `time` as YYYY-MM-DD strings in local TZ;
	// take the next forecastDays days starting at index 1 (today is
	// index 0).
	parts := make([]string, 0, forecastDays*4)
	for i := 1; i <= forecastDays && i < len(body.Daily.Time); i++ {
		day := shortDayName(body.Daily.Time[i])
		hi := int(math.Round(body.Daily.Max[i]))
		lo := int(math.Round(body.Daily.Min[i]))
		outlook := OutlookFromCode(body.Daily.WeatherCode[i])
		parts = append(parts,
			day,
			fmt.Sprintf("%d", hi),
			fmt.Sprintf("%d", lo),
			outlook,
		)
	}
	out := strings.Join(parts, "|")

	f.lastFetch = time.Now()
	f.cached = out
	f.cachedErr = nil
	return out, nil
}

func (f *Forecast) fail(err error) (string, error) {
	f.lastFetch = time.Now()
	f.cached = ""
	f.cachedErr = err
	return "", err
}

// shortDayName parses a YYYY-MM-DD date string and returns the
// 3-letter lowercase day name (e.g. "thu"). On parse failure
// returns the raw string — caller renders whatever they get.
func shortDayName(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return strings.ToLower(t.Weekday().String()[:3])
}

var _ widget.Widget = (*Forecast)(nil)
