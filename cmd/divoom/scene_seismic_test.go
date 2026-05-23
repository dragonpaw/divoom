package main

import "testing"

// TestSeismicBandColor pins the band-colour ladder against the
// boundary magnitudes specified in the blueprint: 2.5, 3.9, 4.0, 4.9,
// 5.0, 5.9, 6.0. Each pair (mag, color) reads top-down through the
// gruvbox accent ladder cFg → cYellow → cOrange → cRed.
func TestSeismicBandColor(t *testing.T) {
	cases := []struct {
		mag  float64
		want string
	}{
		{2.5, cYellow}, // bottom of micro
		{3.9, cYellow}, // top of micro
		{4.0, cOrange}, // bottom of light
		{4.9, cOrange}, // top of light
		{5.0, cRed},    // bottom of moderate
		{5.9, cRed},    // top of moderate
		{6.0, cPurple}, // bottom of strong
	}
	for _, c := range cases {
		got := seismicBandColor(c.mag)
		if got != c.want {
			t.Errorf("seismicBandColor(%.1f) = %s, want %s", c.mag, got, c.want)
		}
	}
}

// TestSeismicCommentary verifies every band keys to a non-empty pool —
// the formatter assumes len(pool) > 0 when computing the hash-pinned
// index, so an empty band would crash with a division by zero.
func TestSeismicCommentary(t *testing.T) {
	magsByBand := map[string]float64{
		"none":     0.0,
		"micro":    3.0,
		"light":    4.5,
		"moderate": 5.5,
		"strong":   6.5,
	}
	for band, mag := range magsByBand {
		gotBand := seismicBandFor(mag)
		if gotBand != band {
			t.Errorf("seismicBandFor(%.1f) = %q, want %q", mag, gotBand, band)
		}
		pool := seismicCommentaryPool[band]
		if len(pool) == 0 {
			t.Errorf("commentary pool for band %q is empty", band)
		}
	}
}
