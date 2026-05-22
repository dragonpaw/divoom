package main

import (
	"testing"
	"time"
)

// TestDaysUntilWeekend covers every weekday and both weekend days, so
// the operator footer never silently drifts when the helper changes.
func TestDaysUntilWeekend(t *testing.T) {
	// 2026-05-18 is a Monday → through 2026-05-24 (Sunday).
	base := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	want := []string{
		"weekend+4d", // Mon
		"weekend+3d", // Tue
		"weekend+2d", // Wed
		"weekend+1d", // Thu
		"weekend+0d", // Fri
		"weekend",    // Sat
		"weekend",    // Sun
	}
	for i, w := range want {
		d := base.AddDate(0, 0, i)
		if got := daysUntilWeekend(d); got != w {
			t.Errorf("daysUntilWeekend(%s = %s) = %q, want %q",
				d.Format("2006-01-02"), d.Weekday(), got, w)
		}
	}
}

// TestISOWeek pins a couple of known ISO-week values so future tweaks
// don't accidentally pull the wrong field out of time.Time.ISOWeek.
func TestISOWeek(t *testing.T) {
	// 2026-05-22 is in ISO week 21.
	if got := isoWeek(time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)); got != 21 {
		t.Errorf("isoWeek(2026-05-22) = %d, want 21", got)
	}
	// 2026-01-01 (Thursday) belongs to ISO week 1 of 2026.
	if got := isoWeek(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); got != 1 {
		t.Errorf("isoWeek(2026-01-01) = %d, want 1", got)
	}
}

// TestWeatherConditionOrHazard: hazard text wins over the outlook word,
// otherwise the outlook word renders in its outlook colour.
func TestWeatherConditionOrHazard(t *testing.T) {
	cases := []struct {
		name, raw, wantText, wantColor string
	}{
		{"plain clear", "63°F|clear||45|62|30", "clear", cYellow},
		{"plain rain", "55°F|rain||10|80|90", "rain", cBlue},
		{"hazard wins over outlook", "78°F|hazard|Red Flag Warning|45|62|30", "Red Flag Warning", cRed},
		{"empty outlook (no pipes)", "63°F", "", cFg},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text, color := weatherConditionOrHazard(tc.raw)
			if text != tc.wantText || color != tc.wantColor {
				t.Errorf("weatherConditionOrHazard(%q) = (%q,%q), want (%q,%q)",
					tc.raw, text, color, tc.wantText, tc.wantColor)
			}
		})
	}
}

// TestWeatherAQI exercises the EPA-band colour table plus the missing /
// non-numeric edges. Each band boundary is checked at both its lower
// and upper inclusive limits.
func TestWeatherAQI(t *testing.T) {
	cases := []struct {
		name, raw, wantText, wantColor string
	}{
		{"blank → em-dash dim", "63°F|clear||||", "—", cFgDark},
		{"0 → good", "63°F|clear||0||", "0", cGreen},
		{"50 → good (upper edge)", "63°F|clear||50||", "50", cGreen},
		{"51 → moderate", "63°F|clear||51||", "51", cYellow},
		{"100 → moderate (upper edge)", "63°F|clear||100||", "100", cYellow},
		{"101 → USG", "63°F|clear||101||", "101", cOrange},
		{"150 → USG (upper edge)", "63°F|clear||150||", "150", cOrange},
		{"151 → unhealthy", "63°F|clear||151||", "151", cRed},
		{"200 → unhealthy (upper edge)", "63°F|clear||200||", "200", cRed},
		{"201 → very unhealthy", "63°F|clear||201||", "201", cPurple},
		{"300 → very unhealthy (upper edge)", "63°F|clear||300||", "300", cPurple},
		{"301 → hazardous", "63°F|clear||301||", "301", cRed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text, color := weatherAQI(tc.raw)
			if text != tc.wantText || color != tc.wantColor {
				t.Errorf("weatherAQI(%q) = (%q,%q), want (%q,%q)",
					tc.raw, text, color, tc.wantText, tc.wantColor)
			}
		})
	}
}

// TestWeatherHumidityAndRain covers the present + blank cases for the
// two simple percentage formatters.
func TestWeatherHumidityAndRain(t *testing.T) {
	if text, color := weatherHumidity("63°F|clear||45|62|30"); text != "62%" || color != cBlue {
		t.Errorf("weatherHumidity present = (%q,%q), want (62%%,%q)", text, color, cBlue)
	}
	if text, color := weatherHumidity("63°F|clear||45||30"); text != "—" || color != cFgDark {
		t.Errorf("weatherHumidity blank = (%q,%q), want (—,%q)", text, color, cFgDark)
	}
	if text, color := weatherRain("63°F|clear||45|62|30"); text != "30%" || color != cAqua {
		t.Errorf("weatherRain present = (%q,%q), want (30%%,%q)", text, color, cAqua)
	}
	if text, color := weatherRain("63°F|clear||45|62|"); text != "—" || color != cFgDark {
		t.Errorf("weatherRain blank = (%q,%q), want (—,%q)", text, color, cFgDark)
	}
}
