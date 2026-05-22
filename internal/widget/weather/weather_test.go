package weather

import "testing"

// TestUseFahrenheit covers the verify cases from the design note plus a
// few region edges. Richmond and Honolulu sit inside the US boxes;
// London and Sydney are firmly outside any Fahrenheit holdout.
func TestUseFahrenheit(t *testing.T) {
	cases := []struct {
		name     string
		lat, lon float64
		want     bool
	}{
		{"Richmond CA", 37.9358, -122.3477, true},
		{"Honolulu HI", 21.3069, -157.8583, true},
		{"Anchorage AK", 61.2181, -149.9003, true},
		{"San Juan PR", 18.4655, -66.1057, true},
		{"London UK", 51.5074, -0.1278, false},
		{"Sydney AU", -33.8688, 151.2093, false},
		{"Tokyo JP", 35.6762, 139.6503, false},
		// Note: the bounding boxes are deliberately coarse (per the
		// design note), so southern-Canada / northern-Mexico points
		// near the lower-48 box's edges read as Fahrenheit. That's an
		// accepted ~0.1% rate of false positives — not tested here.
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := useFahrenheit(tc.lat, tc.lon)
			if got != tc.want {
				t.Errorf("useFahrenheit(%v, %v) = %v, want %v",
					tc.lat, tc.lon, got, tc.want)
			}
		})
	}
}

// TestNewUnit checks that New translates coordinates into the right
// Unit() letter. This is the public surface the daemon depends on for
// seeding colour thresholds and rendering the suffix.
func TestNewUnit(t *testing.T) {
	cases := []struct {
		name     string
		lat, lon string
		want     string
	}{
		{"Richmond", "37.9358", "-122.3477", "F"},
		{"London", "51.5074", "-0.1278", "C"},
		{"Sydney", "-33.8688", "151.2093", "C"},
		{"Honolulu", "21.3069", "-157.8583", "F"},
		{"unparseable falls back to C", "nope", "nope", "C"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := New(tc.lat, tc.lon).Unit(); got != tc.want {
				t.Errorf("New(%q,%q).Unit() = %q, want %q",
					tc.lat, tc.lon, got, tc.want)
			}
		})
	}
}
