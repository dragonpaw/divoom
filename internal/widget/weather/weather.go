// Package weather fetches current conditions from Open-Meteo and emits a
// pipe-separated "<temp>°|<outlook>|<hazard>" string for the weather
// scene. Three sources are merged in parallel:
//
//   - Open-Meteo /forecast for temperature + WMO weather code,
//   - Open-Meteo /air-quality for PM2.5 + US AQI (overrides outlook to
//     "smoke" when air quality is hazardous),
//   - api.weather.gov /alerts/active for active NWS hazards at the
//     configured point (overrides outlook to "hazard" when present;
//     non-US locations 4xx silently and are treated as "no alerts").
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Client hits Open-Meteo's current-weather endpoint for a fixed location.
// One http.Client with a 10s timeout is reused across Fetch calls.
type Client struct {
	lat, lon string
	// unit is the Open-Meteo temperature_unit value ("fahrenheit" or
	// "celsius"), picked from lat/lon at construction time. Both Fetch
	// and LoadThresholds use the same unit so the climate-calibrated
	// colour bounds stay aligned with the live reading.
	unit string
	http *http.Client

	// Climate-normals cache. LoadThresholds populates these on first call
	// (guarded by thresholdOnce) and returns the cached values on
	// subsequent calls.
	thresholdOnce sync.Once
	coldBound     int
	hotBound      int
	thresholdErr  error
}

// Lat / Lon expose the configured coordinates so callers (e.g. the
// daemon's startup logger) can report the location actually in use
// without keeping a parallel copy.
func (c *Client) Lat() string { return c.lat }
func (c *Client) Lon() string { return c.lon }

// Unit returns "F" or "C" — the single-letter form of the temperature
// unit this client is configured to fetch in. Drives the colour-band
// constants and the rendered suffix in the scene.
func (c *Client) Unit() string {
	if c.unit == "fahrenheit" {
		return "F"
	}
	return "C"
}

// New returns a weather client for the given coordinates. Coordinates are
// strings so callers can pass them straight from config without going
// through float parsing on our side. The temperature unit is picked from
// the coordinates: Fahrenheit for the US (lower-48, Alaska, Hawaii,
// Puerto Rico) and a small handful of other holdouts, Celsius everywhere
// else. Unparseable coordinates fall back to Celsius (the global default).
func New(lat, lon string) *Client {
	unit := "celsius"
	latF, errLat := strconv.ParseFloat(lat, 64)
	lonF, errLon := strconv.ParseFloat(lon, 64)
	if errLat == nil && errLon == nil && useFahrenheit(latF, lonF) {
		unit = "fahrenheit"
	}
	return &Client{
		lat:  lat,
		lon:  lon,
		unit: unit,
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

// useFahrenheit reports whether a (lat, lon) point falls inside a
// region that conventionally uses Fahrenheit for everyday weather: the
// US (lower-48 bounding box, Alaska, Hawaii, Puerto Rico) and the
// handful of other holdouts (Bahamas, Cayman Islands, Belize, Liberia,
// Palau, FSM, Marshall Islands). The boxes are deliberately coarse —
// for the ~0.1% of locations near a border that's good enough, and it
// keeps the check stdlib-only.
func useFahrenheit(lat, lon float64) bool {
	in := func(lat1, lat2, lon1, lon2 float64) bool {
		return lat >= lat1 && lat <= lat2 && lon >= lon1 && lon <= lon2
	}
	switch {
	case in(24, 49, -125, -66): // US lower-48
		return true
	case in(51, 71, -180, -130): // Alaska
		return true
	case in(18, 23, -160, -154): // Hawaii
		return true
	case in(17.8, 18.6, -67.4, -65.2): // Puerto Rico
		return true
	case in(20.8, 27.3, -79.0, -72.7): // Bahamas
		return true
	case in(19.2, 19.8, -81.5, -79.7): // Cayman Islands
		return true
	case in(15.8, 18.5, -89.3, -87.7): // Belize
		return true
	case in(4.3, 8.6, -11.5, -7.3): // Liberia
		return true
	case in(2.8, 8.1, 131.1, 134.7): // Palau
		return true
	case in(1.0, 10.0, 138.0, 163.1): // Federated States of Micronesia
		return true
	case in(4.5, 14.7, 160.8, 172.2): // Marshall Islands
		return true
	}
	return false
}

func (c *Client) Name() string { return "weather" }

// User-Agent for NWS api.weather.gov. They require a contact-bearing UA
// and 403 anonymous traffic.
const nwsUserAgent = "divoom-dashboard/0.1 (github.com/dragonpaw/divoom)"

// PM2.5 and US AQI thresholds above which we override the outlook to
// "smoke". The US AQI threshold (>150) matches the EPA "Unhealthy" band;
// the PM2.5 threshold (>35 µg/m³) matches the EPA 24-hour fine-particulate
// standard.
const (
	smokePM25Threshold = 35.0
	smokeAQIThreshold  = 150
)

// hazardHeadlineMaxLen caps the alert headline length so it fits the
// device's body element. Truncation adds an ellipsis.
const hazardHeadlineMaxLen = 50

// currentWeatherResponse is the slice of the Open-Meteo /forecast JSON we
// care about — just the current_weather block. Open-Meteo's "weathercode"
// follows the WMO weather interpretation table.
type currentWeatherResponse struct {
	CurrentWeather struct {
		Temperature float64 `json:"temperature"`
		WeatherCode int     `json:"weathercode"`
	} `json:"current_weather"`
}

type airQualityResponse struct {
	Current struct {
		PM25  float64 `json:"pm2_5"`
		USAQI float64 `json:"us_aqi"`
	} `json:"current"`
}

type nwsAlertsResponse struct {
	Features []struct {
		Properties struct {
			Event    string `json:"event"`
			Severity string `json:"severity"`
		} `json:"properties"`
	} `json:"features"`
}

// Fetch returns a pipe-separated "<temp>°|<outlook>|<hazard>" string.
// The hazard segment is empty unless an NWS alert is active for the
// configured point.
func (c *Client) Fetch(ctx context.Context) (string, error) {
	var (
		wg      sync.WaitGroup
		fcResp  currentWeatherResponse
		fcErr   error
		aqResp  airQualityResponse
		aqErr   error
		nwsResp nwsAlertsResponse
	)

	wg.Add(3)
	go func() {
		defer wg.Done()
		fcErr = c.fetchForecast(ctx, &fcResp)
	}()
	go func() {
		defer wg.Done()
		aqErr = c.fetchAirQuality(ctx, &aqResp)
	}()
	go func() {
		defer wg.Done()
		// NWS errors (incl. 4xx for non-US points) are swallowed by
		// fetchAlerts; an empty features list means "no alerts".
		c.fetchAlerts(ctx, &nwsResp)
	}()
	wg.Wait()

	if fcErr != nil {
		return "", fcErr
	}

	temp := int(math.Round(fcResp.CurrentWeather.Temperature))
	outlook := OutlookFromCode(fcResp.CurrentWeather.WeatherCode)
	hazardMsg := ""

	// NWS takes top precedence — an active warning trumps both the
	// air-quality smoke override and the WMO code's outlook.
	if alert := mostSevereAlert(nwsResp.Features); alert != "" {
		outlook = "hazard"
		hazardMsg = truncateHeadline(alert, hazardHeadlineMaxLen)
	} else if aqErr == nil && isSmoke(aqResp.Current.PM25, aqResp.Current.USAQI) {
		outlook = "smoke"
	}

	return fmt.Sprintf("%d°%s|%s|%s", temp, c.Unit(), outlook, hazardMsg), nil
}

func (c *Client) fetchForecast(ctx context.Context, out *currentWeatherResponse) error {
	url := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast"+
			"?latitude=%s&longitude=%s"+
			"&current_weather=true&temperature_unit=%s&timezone=auto",
		c.lat, c.lon, c.unit,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("weather: build request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("weather: http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("weather: http %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("weather: decode: %w", err)
	}
	return nil
}

func (c *Client) fetchAirQuality(ctx context.Context, out *airQualityResponse) error {
	url := fmt.Sprintf(
		"https://air-quality-api.open-meteo.com/v1/air-quality"+
			"?latitude=%s&longitude=%s&current=pm2_5,us_aqi",
		c.lat, c.lon,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("weather: build aq request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("weather: aq http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("weather: aq http %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("weather: aq decode: %w", err)
	}
	return nil
}

// fetchAlerts queries NWS for active alerts at the configured point.
// Errors (network, 4xx for non-US points, parse) are intentionally
// swallowed — the alerts feed is best-effort context; a failure here
// must not block the forecast. On any failure `out` is left zero
// (empty Features), which downstream code reads as "no alerts".
func (c *Client) fetchAlerts(ctx context.Context, out *nwsAlertsResponse) {
	url := fmt.Sprintf(
		"https://api.weather.gov/alerts/active?point=%s,%s",
		c.lat, c.lon,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", nwsUserAgent)
	req.Header.Set("Accept", "application/geo+json")
	resp, err := c.http.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}
	_ = json.NewDecoder(resp.Body).Decode(out)
}

// isSmoke reports whether the air-quality readings cross either of the
// "smoky outdoors" thresholds (PM2.5 > 35 µg/m³ or US AQI > 150).
func isSmoke(pm25 float64, usAQI float64) bool {
	return pm25 > smokePM25Threshold || int(usAQI+0.5) > smokeAQIThreshold
}

// severityRank orders NWS alert severity strings so we can pick the
// "most severe" feature in a multi-alert response. Unknown / missing
// severities sort to the bottom.
func severityRank(s string) int {
	switch s {
	case "Extreme":
		return 4
	case "Severe":
		return 3
	case "Moderate":
		return 2
	case "Minor":
		return 1
	default:
		return 0
	}
}

// mostSevereAlert returns the Event string of the highest-ranked alert
// in features, or "" if the slice is empty. Ties (equal severity) keep
// the first-encountered alert — NWS returns them in issuance order.
func mostSevereAlert(features []struct {
	Properties struct {
		Event    string `json:"event"`
		Severity string `json:"severity"`
	} `json:"properties"`
}) string {
	bestEvent := ""
	bestRank := -1
	for _, f := range features {
		r := severityRank(f.Properties.Severity)
		if r > bestRank && f.Properties.Event != "" {
			bestRank = r
			bestEvent = f.Properties.Event
		}
	}
	return bestEvent
}

// truncateHeadline shortens s to at most max characters, adding a single
// trailing ellipsis when truncation actually happens. Whitespace at the
// truncation boundary is trimmed so the ellipsis doesn't sit after a
// dangling space.
func truncateHeadline(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	cut := strings.TrimRightFunc(s[:max-1], unicodeIsSpaceOrPunct)
	return cut + "…"
}

func unicodeIsSpaceOrPunct(r rune) bool {
	switch r {
	case ' ', '\t', ',', ';', ':', '-':
		return true
	}
	return false
}

// OutlookFromCode buckets a WMO weather code into one of eight outlook
// strings. The buckets are deliberately coarse — the scene only needs
// enough resolution to pick an icon, a colour, and a label word.
//
// WMO code ranges (per Open-Meteo docs):
//
//	0          clear sky
//	1, 2       mainly clear, partly cloudy
//	3          overcast
//	45, 48     fog, depositing rime fog
//	51, 53, 55 drizzle
//	56, 57     freezing drizzle
//	61, 63, 65 rain
//	66, 67     freezing rain
//	71, 73, 75 snow fall
//	77         snow grains
//	80, 81, 82 rain showers
//	85, 86     snow showers
//	95         thunderstorm
//	96, 99     thunderstorm with hail
func OutlookFromCode(code int) string {
	switch code {
	case 0:
		return "clear"
	case 1, 2:
		return "cloudy"
	case 3:
		return "overcast"
	case 45, 48:
		return "fog"
	case 51, 53, 55, 56, 57:
		return "drizzle"
	case 61, 63, 65, 66, 67, 80, 81, 82:
		return "rain"
	case 71, 73, 75, 77, 85, 86:
		return "snow"
	case 95, 96, 99:
		return "thunder"
	default:
		return "cloudy"
	}
}

// archiveResponse is the slice of Open-Meteo's /v1/archive JSON we use
// for climate-normals fitting. Both arrays are parallel daily series.
type archiveResponse struct {
	Daily struct {
		Time []string  `json:"time"`
		Max  []float64 `json:"temperature_2m_max"`
		Min  []float64 `json:"temperature_2m_min"`
	} `json:"daily"`
}

// LoadThresholds fetches 5 years of historical daily highs/lows from
// Open-Meteo's archive API and returns (coldBound, hotBound) where:
//
//	coldBound = 15th-percentile of daily LOWS  (rounded to nearest int)
//	hotBound  = 85th-percentile of daily HIGHS (rounded to nearest int)
//
// Both in whichever unit the client was configured for (see Unit). The
// archive request uses the same temperature_unit as Fetch so the bounds
// stay aligned with the live reading. Results are cached internally;
// second and later calls return the cached values immediately without
// re-fetching.
//
// To keep the fixed comfort band (68-75°F / 20-24°C) non-empty,
// coldBound is clamped just below the band's lower edge and hotBound
// just above its upper edge so the aqua/yellow/orange/red bands always
// have at least one degree each.
//
// Returns an error on network / parse / empty-sample failure. The
// caller is expected to log and fall back to static defaults.
func (c *Client) LoadThresholds(ctx context.Context) (cold, hot int, err error) {
	c.thresholdOnce.Do(func() {
		c.coldBound, c.hotBound, c.thresholdErr = c.fetchThresholds(ctx)
	})
	return c.coldBound, c.hotBound, c.thresholdErr
}

func (c *Client) fetchThresholds(ctx context.Context) (cold, hot int, err error) {
	// The archive API trails real time by ~5 days; "yesterday" is the
	// safe upper bound. 5 years back gives ~1825 samples, enough for a
	// stable 15th/85th percentile.
	end := time.Now().AddDate(0, 0, -1)
	start := end.AddDate(-5, 0, 0)
	url := fmt.Sprintf(
		"https://archive-api.open-meteo.com/v1/archive"+
			"?latitude=%s&longitude=%s"+
			"&start_date=%s&end_date=%s"+
			"&daily=temperature_2m_max,temperature_2m_min"+
			"&temperature_unit=%s&timezone=auto",
		c.lat, c.lon,
		start.Format("2006-01-02"),
		end.Format("2006-01-02"),
		c.unit,
	)

	// Archive responses over 5 years can be slow — give it a generous
	// budget, separate from c.http's short Fetch timeout.
	httpCli := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("weather: build archive request: %w", err)
	}
	resp, err := httpCli.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("weather: archive http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("weather: archive http %d", resp.StatusCode)
	}
	var body archiveResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, 0, fmt.Errorf("weather: archive decode: %w", err)
	}
	if len(body.Daily.Max) == 0 || len(body.Daily.Min) == 0 {
		return 0, 0, fmt.Errorf("weather: archive returned empty daily series")
	}

	highs := append([]float64(nil), body.Daily.Max...)
	lows := append([]float64(nil), body.Daily.Min...)
	sort.Float64s(highs)
	sort.Float64s(lows)

	cold = int(math.Round(lows[int(float64(len(lows))*0.15)]))
	hot = int(math.Round(highs[int(float64(len(highs))*0.85)]))

	// Clamp so the fixed comfort band (68-75°F / 20-24°C) always has
	// room above (yellow) and below (aqua) it.
	comfortLo, comfortHi := 68, 75
	if c.unit == "celsius" {
		comfortLo, comfortHi = 20, 24
	}
	if cold >= comfortLo {
		cold = comfortLo - 1
	}
	if hot <= comfortHi {
		hot = comfortHi + 1
	}
	return cold, hot, nil
}
