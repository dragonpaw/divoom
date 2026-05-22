package main

import "testing"

// TestWeatherStats covers the AQI / humidity / rain packing of the
// stats-row formatter. The widget output has six pipe segments; this
// helper only reads segments 3-5 and drops any that are blank, so the
// leading temp / outlook / hazard fields are present here just to keep
// the raw shape realistic.
func TestWeatherStats(t *testing.T) {
	cases := []struct {
		name, raw, want string
	}{
		{
			name: "all three present",
			raw:  "63°F|clear||45|62|30",
			want: "AQI 45 · 62% RH · 30% rain",
		},
		{
			name: "AQI missing (air-quality fetch failed)",
			raw:  "63°F|clear|||62|30",
			want: "62% RH · 30% rain",
		},
		{
			name: "humidity missing",
			raw:  "63°F|clear||45||30",
			want: "AQI 45 · 30% rain",
		},
		{
			name: "rain missing",
			raw:  "63°F|clear||45|62|",
			want: "AQI 45 · 62% RH",
		},
		{
			name: "all three missing",
			raw:  "63°F|clear||||",
			want: "",
		},
		{
			name: "legacy 3-segment output (no stats fields at all)",
			raw:  "63°F|clear|",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := weatherStats(tc.raw)
			if got != tc.want {
				t.Errorf("weatherStats(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}
