// Package seismic emits the most-notable recent earthquake within a
// fixed 500km radius of a configured (lat, lon), pulled from the USGS
// near-real-time feed.
package seismic

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/dragonpaw/divoom/internal/widget"
)

// USGS 2.5+ "all earthquakes in the last day" GeoJSON summary feed. The
// 2.5+ tier is small (~50KB) and pre-filtered to events the public
// usually feels; the scene's notability check filters again by radius.
const feedURL = "https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/2.5_day.geojson"

const (
	// radiusKm is the fixed reach of the dashboard's seismic alerting.
	// 500km covers the local fault network around the configured point
	// without overpowering distant unrelated swarms; not parameterised
	// because no second use case has surfaced (CLAUDE.md flaw 3).
	radiusKm = 500.0

	// cacheTTL throttles fetches to once per five minutes. USGS publishes
	// updates more often than that, but the scene activates well below
	// five-minute cadence and we don't want to hammer their CDN.
	cacheTTL = 5 * time.Minute

	userAgent = "divoom-dashboard/0.1 (github.com/dragonpaw/divoom)"
)

// Seismic emits "<mag>|<count>|<dist_km>|<bearing>|<age>" — the
// magnitude of the most-notable recent event in the 500km radius, the
// total event count in that radius, and the distance + 8-point compass
// bearing + age of the headline event.
//
// Empty/no-event case: "0.0|0|||".
type Seismic struct {
	Lat, Lon string
	HTTP     *http.Client

	mu       sync.Mutex
	cached   string
	cachedAt time.Time
}

// New constructs a Seismic widget anchored at (lat, lon). Lat/lon are
// strings to match the dashboard's other location-anchored widgets
// (sky.NewISS, weather.NewForecast) so the call sites stay uniform.
func New(lat, lon string) *Seismic {
	return &Seismic{
		Lat:  lat,
		Lon:  lon,
		HTTP: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *Seismic) Name() string { return "seismic" }

// usgsFeed is the slice of the USGS GeoJSON we actually read. The
// upstream structure is much larger; this captures only the fields the
// scene cares about.
type usgsFeed struct {
	Features []struct {
		Properties struct {
			Mag   float64 `json:"mag"`
			Time  int64   `json:"time"`
			Place string  `json:"place"`
		} `json:"properties"`
		Geometry struct {
			Coordinates []float64 `json:"coordinates"` // [lon, lat, depth]
		} `json:"geometry"`
	} `json:"features"`
}

// Fetch returns the pipe-shaped widget string. Concurrent callers
// serialize on the mutex; within cacheTTL, the cached value is
// returned without an HTTP call.
func (s *Seismic) Fetch(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cached != "" && time.Since(s.cachedAt) < cacheTTL {
		return s.cached, nil
	}

	lat, err := strconv.ParseFloat(s.Lat, 64)
	if err != nil {
		return "", fmt.Errorf("seismic: bad lat %q: %w", s.Lat, err)
	}
	lon, err := strconv.ParseFloat(s.Lon, 64)
	if err != nil {
		return "", fmt.Errorf("seismic: bad lon %q: %w", s.Lon, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return "", fmt.Errorf("seismic: build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("seismic: http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("seismic: http %d", resp.StatusCode)
	}
	var body usgsFeed
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("seismic: decode: %w", err)
	}

	out := assemble(lat, lon, body, time.Now())
	s.cached = out
	s.cachedAt = time.Now()
	return out, nil
}

// assemble walks the feed once: filters by radius, counts qualifiers,
// and tracks the running max-magnitude headline event. Output shape:
// "<mag>|<count>|<dist_km>|<bearing>|<age>", with no-event case
// "0.0|0|||" (trailing empties; scene formatters tolerate).
func assemble(originLat, originLon float64, feed usgsFeed, now time.Time) string {
	count := 0
	var (
		bestMag  float64
		bestDist float64
		bestBrg  string
		bestAge  time.Duration
		haveBest bool
	)
	for _, f := range feed.Features {
		if len(f.Geometry.Coordinates) < 2 {
			continue
		}
		evLon := f.Geometry.Coordinates[0]
		evLat := f.Geometry.Coordinates[1]
		dist := haversineKm(originLat, originLon, evLat, evLon)
		if dist > radiusKm {
			continue
		}
		count++
		if !haveBest || f.Properties.Mag > bestMag {
			haveBest = true
			bestMag = f.Properties.Mag
			bestDist = dist
			bestBrg = bearing8(originLat, originLon, evLat, evLon)
			bestAge = now.Sub(time.UnixMilli(f.Properties.Time))
		}
	}
	if !haveBest {
		return "0.0|0|||"
	}
	return fmt.Sprintf("%.1f|%d|%d|%s|%s",
		bestMag, count, int(math.Round(bestDist)), bestBrg, formatAge(bestAge))
}

// haversineKm — great-circle distance in km. Earth radius 6371.0 km,
// same constant used elsewhere in the project (sky/iss_geo.go).
func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLon := (lon2 - lon1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// bearing8 returns one of N/NE/E/SE/S/SW/W/NW — the initial-course
// great-circle bearing from (lat1, lon1) to (lat2, lon2), rounded to
// the nearest 45° compass octant.
func bearing8(lat1, lon1, lat2, lon2 float64) string {
	rad := math.Pi / 180
	φ1 := lat1 * rad
	φ2 := lat2 * rad
	Δλ := (lon2 - lon1) * rad
	y := math.Sin(Δλ) * math.Cos(φ2)
	x := math.Cos(φ1)*math.Sin(φ2) -
		math.Sin(φ1)*math.Cos(φ2)*math.Cos(Δλ)
	θ := math.Atan2(y, x) / rad // -180..180
	if θ < 0 {
		θ += 360
	}
	// Octant index 0..7 — round to nearest 45°, mod 8.
	i := int(math.Round(θ/45)) % 8
	return [...]string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}[i]
}

// formatAge renders a Duration as "47m ago", "3h ago", "2d ago", with
// "just now" for anything under one minute. Negative durations (clock
// skew between client and USGS) collapse to "just now" too.
func formatAge(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

var _ widget.Widget = (*Seismic)(nil)
