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

// TestMoonPhaseIndex covers the boundary collapses (new / full) and a
// sampling of intermediate waxing / waning illuminations so the BgPathFor
// mapping stays pinned. Variant illumination values (rounded):
//   1→5, 2→21, 3→43, 4→67, 5→87, 6→98 (waxing)
//   8→98, 9→87, 10→67, 11→43, 12→21, 13→5 (waning)
func TestMoonPhaseIndex(t *testing.T) {
	cases := []struct {
		name   string
		illum  int
		waxing bool
		want   int
	}{
		{"0% waxing → new", 0, true, 0},
		{"3% waxing → new (below threshold)", 3, true, 0},
		{"4% waxing → first waxing variant", 4, true, 1},
		{"61% waxing → variant 4 (near first-quarter sample)", 61, true, 4},
		{"67% waxing → variant 4", 67, true, 4},
		{"96% waxing → variant 6 (last sub-full)", 96, true, 6},
		{"97% waxing → full", 97, true, 7},
		{"100% waxing → full", 100, true, 7},
		{"0% waning → new", 0, false, 0},
		{"100% waning → full", 100, false, 7},
		{"61% waning → variant 10", 61, false, 10},
		{"21% waning → variant 12", 21, false, 12},
		{"5% waning → variant 13", 5, false, 13},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := moonPhaseIndex(tc.illum, tc.waxing); got != tc.want {
				t.Errorf("moonPhaseIndex(%d, waxing=%v) = %d, want %d",
					tc.illum, tc.waxing, got, tc.want)
			}
		})
	}
}

// TestMoonBgPathFor exercises the end-to-end parse from the widget's
// "moon · <name> · <illum>% · <countdown>" string to a variant path.
// Covers each phase name family the widget emits.
func TestMoonBgPathFor(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"new", "moon · new · 0% · full moon in 15 days", moonBackgrounds[0]},
		{"full", "moon · full · 100% · next full moon: Jun 1", moonBackgrounds[7]},
		{"first quarter at 50% (lands on variant 4 — closest sample)", "moon · first quarter · 50% · full moon in 7 days", moonBackgrounds[4]},
		{"waxing crescent low", "moon · waxing crescent · 5% · full moon in 13 days", moonBackgrounds[1]},
		{"waxing gibbous", "moon · waxing gibbous · 85% · full moon in 2 days", moonBackgrounds[5]},
		{"waning crescent", "moon · waning crescent · 5% · next full moon: Jul 1", moonBackgrounds[13]},
		{"last quarter at 50% (lands on variant 11)", "moon · last quarter · 50% · next full moon: Jul 1", moonBackgrounds[11]},
		{"malformed → safe full fallback", "garbage", moonBackgrounds[7]},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := moonBgPathFor(tc.raw); got != tc.want {
				t.Errorf("moonBgPathFor(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// TestMoonPhaseAndIllum verifies the combined formatter title-cases the
// phase name and stitches it together with the raw illum segment.
func TestMoonPhaseAndIllum(t *testing.T) {
	if text, _ := moonPhaseAndIllum("moon · first quarter · 53% · full moon in 7 days"); text != "First Quarter · 53%" {
		t.Errorf("moonPhaseAndIllum first quarter = %q, want %q", text, "First Quarter · 53%")
	}
	if text, _ := moonPhaseAndIllum("moon · new · 0% · full moon in 15 days"); text != "New · 0%" {
		t.Errorf("moonPhaseAndIllum new = %q, want %q", text, "New · 0%")
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

// TestHNFooter covers every dropout the formatter has to handle: all
// fields present, score missing (drop "▲ N" lead), comments "0" / blank
// (drop the comments segment), author blank (fall back to "unknown"),
// all metadata missing (still produces a sensible byline). Pins the
// glue characters so the rendered footer stays consistent.
func TestHNFooter(t *testing.T) {
	raw := func(score, author, age, comments string) string {
		return "Hacker News|t|d|s|" + score + "|" + author + "|" + age + "|" + comments
	}
	cases := []struct {
		name, in, want string
	}{
		{"all present",
			raw("412", "patio11", "3h", "187"),
			"▲ 412  by patio11  ·  3h  ·  187 comments"},
		{"score missing",
			raw("", "patio11", "3h", "187"),
			"by patio11  ·  3h  ·  187 comments"},
		{"comments zero",
			raw("412", "patio11", "3h", "0"),
			"▲ 412  by patio11  ·  3h"},
		{"comments blank",
			raw("412", "patio11", "3h", ""),
			"▲ 412  by patio11  ·  3h"},
		{"author missing",
			raw("412", "", "3h", "187"),
			"▲ 412  by unknown  ·  3h  ·  187 comments"},
		{"all missing",
			raw("", "", "", ""),
			"by unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := hnFooter(tc.in)
			if got != tc.want {
				t.Errorf("hnFooter = %q, want %q", got, tc.want)
			}
		})
	}
	// Defensive: a malformed raw (too few segments) returns "" so the
	// AllowEmpty mount leaves the footer blank rather than rendering
	// "—" or partial garbage.
	if got, _ := hnFooter("Hacker News|title|domain"); got != "" {
		t.Errorf("hnFooter(short) = %q, want empty", got)
	}
}

// TestSunriseTickX pins the dynamic-tick math: before-sunrise clamps to
// the left edge, after-sunset clamps to the right edge, and intermediate
// times interpolate proportionally along the arc. The arc runs x=80→720
// (640 px) and the 40px-wide tick element is centred by subtracting 20
// from the arc point, so:
//   - at sunrise:  StartX = 80  - 20 = 60
//   - at midday:   StartX = 400 - 20 = 380
//   - at sunset:   StartX = 720 - 20 = 700
func TestSunriseTickX(t *testing.T) {
	loc := time.UTC
	rise := time.Date(2026, 5, 22, 6, 0, 0, 0, loc)
	set := time.Date(2026, 5, 22, 18, 0, 0, 0, loc) // 12h span
	cases := []struct {
		name string
		now  time.Time
		want int
	}{
		{"before sunrise", time.Date(2026, 5, 22, 4, 0, 0, 0, loc), 60},
		{"at sunrise", rise, 60},
		{"midday", time.Date(2026, 5, 22, 12, 0, 0, 0, loc), 380},
		{"at sunset", set, 700},
		{"after sunset", time.Date(2026, 5, 22, 22, 0, 0, 0, loc), 700},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sunriseTickX(tc.now, rise, set)
			if got != tc.want {
				t.Errorf("sunriseTickX(%s) = %d, want %d", tc.now.Format("15:04"), got, tc.want)
			}
		})
	}
}

// TestSeasonAt pins the season name and accent colour for each month so
// the dayofyear scene's season label can't silently flip a colour or
// drop a season.
func TestSeasonAt(t *testing.T) {
	cases := []struct {
		when      time.Time
		wantName  string
		wantColor string
	}{
		{time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC), "WINTER", cAqua},
		{time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC), "WINTER", cAqua},
		{time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), "SPRING", cGreen},
		{time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC), "SPRING", cGreen},
		{time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC), "SUMMER", cYellow},
		{time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC), "SUMMER", cYellow},
		{time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC), "AUTUMN", cOrange},
		{time.Date(2026, 11, 30, 12, 0, 0, 0, time.UTC), "AUTUMN", cOrange},
		{time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC), "WINTER", cAqua},
		{time.Date(2026, 12, 31, 12, 0, 0, 0, time.UTC), "WINTER", cAqua},
	}
	for _, tc := range cases {
		gotName, gotColor := seasonAt(tc.when)
		if gotName != tc.wantName || gotColor != tc.wantColor {
			t.Errorf("seasonAt(%s) = (%q, %q), want (%q, %q)",
				tc.when.Format("2006-01-02"), gotName, gotColor, tc.wantName, tc.wantColor)
		}
	}
}

// TestParseSpecialDates exercises happy-path parsing, whitespace
// tolerance, malformed-entry skipping, and the empty-input case.
func TestParseSpecialDates(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if got := parseSpecialDates(""); len(got) != 0 {
			t.Errorf("parseSpecialDates(empty) = %v, want empty map", got)
		}
		if got := parseSpecialDates("   "); len(got) != 0 {
			t.Errorf("parseSpecialDates(whitespace) = %v, want empty map", got)
		}
	})
	t.Run("happy", func(t *testing.T) {
		got := parseSpecialDates("01-13:A,03-22:B,12-25:C")
		want := map[int]rune{113: 'A', 322: 'B', 1225: 'C'}
		if len(got) != len(want) {
			t.Fatalf("parseSpecialDates: got %v, want %v", got, want)
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("parseSpecialDates: key %d = %q, want %q", k, got[k], v)
			}
		}
	})
	t.Run("whitespace tolerance", func(t *testing.T) {
		got := parseSpecialDates("  01-13 : A , 03-22 :B,12-25:C  ")
		if got[113] != 'A' || got[322] != 'B' || got[1225] != 'C' {
			t.Errorf("parseSpecialDates whitespace: got %v", got)
		}
	})
	t.Run("malformed dropped", func(t *testing.T) {
		// Missing letter, missing colon, bad month/day, multi-rune letter.
		got := parseSpecialDates("01-13:A,bad,02-30,99-01:X,03-15:AB,04-04:D")
		if got[113] != 'A' || got[404] != 'D' {
			t.Errorf("parseSpecialDates malformed: missing valid entries, got %v", got)
		}
		// Bad entries should not appear; 99-01 is invalid month, 03-15:AB is multi-rune.
		if _, ok := got[315]; ok {
			t.Errorf("parseSpecialDates: multi-rune entry should not be kept, got %v", got)
		}
		if _, ok := got[9901]; ok {
			t.Errorf("parseSpecialDates: bad month should not be kept, got %v", got)
		}
	})
}

// TestISSMapXY pins the lat/lon → baked-map coordinate projection so a
// drift in either axis (e.g. inverted Y, flipped longitude) fails loudly
// instead of placing the dot in the wrong ocean.
func TestISSMapXY(t *testing.T) {
	// Map rect from render.ISSMap{X0,Y0,W,H}: x ∈ [40,760], y ∈ [560,920].
	cases := []struct {
		lat, lon       float64
		wantX, wantY   int
	}{
		{0, 0, 400, 740},       // equator + prime meridian → centre
		{90, 0, 400, 560},      // north pole → top centre
		{-90, 0, 400, 920},     // south pole → bottom centre
		{0, -180, 40, 740},     // dateline west → left edge mid-height
		{0, 180, 760, 740},     // dateline east → right edge mid-height
	}
	for _, tc := range cases {
		gotX := issMapX(tc.lon)
		gotY := issMapY(tc.lat)
		if gotX != tc.wantX || gotY != tc.wantY {
			t.Errorf("lat=%g lon=%g → (%d,%d); want (%d,%d)",
				tc.lat, tc.lon, gotX, gotY, tc.wantX, tc.wantY)
		}
	}
}

// TestFormatISSCoordsNSEW pins the N/S/E/W reformat used in the
// coords-and-pass row so negative-degree readings can never silently
// render as a positive northern/eastern reading.
func TestFormatISSCoordsNSEW(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"22.5°, 45.3°", "22.5° N   45.3° E"},
		{"-22.5°, -45.3°", "22.5° S   45.3° W"},
		{"0.0°, 0.0°", "0.0° N   0.0° E"},
		{"-1.0°, 179.9°", "1.0° S   179.9° E"},
	}
	for _, tc := range cases {
		if got := formatISSCoordsNSEW(tc.in); got != tc.want {
			t.Errorf("formatISSCoordsNSEW(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

// TestParseISSCoords covers the happy path and a few malformed shapes
// the upstream API could return; ok=false is the only signal the scene
// uses to hide the dot.
func TestParseISSCoords(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		lat, lon, ok := parseISSCoords("-22.5°, -45.3°")
		if !ok || lat != -22.5 || lon != -45.3 {
			t.Fatalf("parse failed: lat=%g lon=%g ok=%v", lat, lon, ok)
		}
	})
	t.Run("no-degree-sign", func(t *testing.T) {
		lat, lon, ok := parseISSCoords("12.0, 34.0")
		if !ok || lat != 12.0 || lon != 34.0 {
			t.Fatalf("parse failed: lat=%g lon=%g ok=%v", lat, lon, ok)
		}
	})
	t.Run("missing-comma", func(t *testing.T) {
		if _, _, ok := parseISSCoords("12.0 34.0"); ok {
			t.Fatal("expected ok=false")
		}
	})
	t.Run("empty", func(t *testing.T) {
		if _, _, ok := parseISSCoords(""); ok {
			t.Fatal("expected ok=false")
		}
	})
	t.Run("non-numeric", func(t *testing.T) {
		if _, _, ok := parseISSCoords("north, east"); ok {
			t.Fatal("expected ok=false")
		}
	})
}
