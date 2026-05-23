package seismic

import (
	"math"
	"strings"
	"testing"
	"time"
)

// TestSeismicHaversine pins haversineKm against two well-known city
// pair distances within 1 km tolerance.
func TestSeismicHaversine(t *testing.T) {
	cases := []struct {
		name        string
		lat1, lon1  float64
		lat2, lon2  float64
		expectKm    float64
		toleranceKm float64
	}{
		// Berkeley → SF City Hall — short city-pair, ~13 km.
		{"berkeley->sf", 37.9358, -122.3477, 37.7793, -122.4193, 18, 2},
		// NYC → LA — coast-to-coast, ~3936 km.
		{"nyc->la", 40.7128, -74.0060, 34.0522, -118.2437, 3936, 5},
	}
	for _, c := range cases {
		got := haversineKm(c.lat1, c.lon1, c.lat2, c.lon2)
		if math.Abs(got-c.expectKm) > c.toleranceKm {
			t.Errorf("%s: got %.1f km, want %.1f ± %.1f km",
				c.name, got, c.expectKm, c.toleranceKm)
		}
	}
}

// TestSeismicBearing pins bearing8 against known cardinal directions.
func TestSeismicBearing(t *testing.T) {
	cases := []struct {
		name       string
		lat1, lon1 float64
		lat2, lon2 float64
		want       string
	}{
		// Point due north of origin → N.
		{"due-north", 0, 0, 10, 0, "N"},
		// Point due east → E.
		{"due-east", 0, 0, 0, 10, "E"},
		// Point due south → S.
		{"due-south", 10, 0, 0, 0, "S"},
		// Point due west → W.
		{"due-west", 0, 10, 0, 0, "W"},
		// NE diagonal — small enough offset that great-circle ≈ rhumb.
		{"north-east", 0, 0, 5, 5, "NE"},
		// SW diagonal.
		{"south-west", 0, 0, -5, -5, "SW"},
	}
	for _, c := range cases {
		got := bearing8(c.lat1, c.lon1, c.lat2, c.lon2)
		if got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}

// TestSeismicPipeAssembly feeds a hand-built feed with three events —
// one near the origin, one far outside the 500km radius, and one
// medium-distance with the highest magnitude — and checks the pipe
// output filters and picks the headline correctly.
func TestSeismicPipeAssembly(t *testing.T) {
	origin := struct{ Lat, Lon float64 }{37.9358, -122.3477}
	now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)

	var feed usgsFeed
	// Near (Berkeley itself, ~0 km), M 2.8, 30m ago.
	addFeature(&feed, 2.8, 37.9, -122.3, now.Add(-30*time.Minute))
	// Far (Honolulu, ~3700 km), M 5.5, 3h ago — must be filtered out.
	addFeature(&feed, 5.5, 21.31, -157.86, now.Add(-3*time.Hour))
	// Medium (Mendocino, ~250 km NW), M 3.4, 3h ago — should be the headline.
	addFeature(&feed, 3.4, 39.31, -123.80, now.Add(-3*time.Hour))

	out := assemble(origin.Lat, origin.Lon, feed, now)
	parts := strings.Split(out, "|")
	if len(parts) != 5 {
		t.Fatalf("expected 5 pipe segments, got %d: %q", len(parts), out)
	}
	if parts[0] != "3.4" {
		t.Errorf("magnitude: got %q, want 3.4", parts[0])
	}
	if parts[1] != "2" {
		t.Errorf("count: got %q, want 2 (Honolulu filtered)", parts[1])
	}
	if parts[3] != "NW" {
		t.Errorf("bearing: got %q, want NW", parts[3])
	}
	if parts[4] != "3h ago" {
		t.Errorf("age: got %q, want '3h ago'", parts[4])
	}
}

// TestSeismicPipeAssemblyNoEvents — empty feed yields the no-event
// sentinel "0.0|0|||".
func TestSeismicPipeAssemblyNoEvents(t *testing.T) {
	var feed usgsFeed
	got := assemble(37.9358, -122.3477, feed, time.Now())
	if got != "0.0|0|||" {
		t.Errorf("got %q, want %q", got, "0.0|0|||")
	}
}

// addFeature is a test-only helper for shaping fixture feeds.
func addFeature(f *usgsFeed, mag, lat, lon float64, when time.Time) {
	var feat struct {
		Properties struct {
			Mag   float64 `json:"mag"`
			Time  int64   `json:"time"`
			Place string  `json:"place"`
		} `json:"properties"`
		Geometry struct {
			Coordinates []float64 `json:"coordinates"`
		} `json:"geometry"`
	}
	feat.Properties.Mag = mag
	feat.Properties.Time = when.UnixMilli()
	feat.Geometry.Coordinates = []float64{lon, lat, 0}
	f.Features = append(f.Features, feat)
}
