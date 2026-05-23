package main

import (
	"testing"
	"time"
)

// TestWeekendStatus exercises the Friday-18:00 → Monday-03:00 weekend
// window plus the dim countdown that runs outside it.
func TestWeekendStatus(t *testing.T) {
	cases := []struct {
		name      string
		when      time.Time
		wantText  string
		wantColor string
	}{
		// Outside the window — dim countdown.
		{"mon noon", time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC), "weekend+4d", cFgDark},
		{"tue noon", time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC), "weekend+3d", cFgDark},
		{"wed noon", time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC), "weekend+2d", cFgDark},
		{"thu noon", time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC), "weekend+1d", cFgDark},
		{"fri noon (pre-6pm)", time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC), "weekend+0d", cFgDark},
		{"fri 5:59pm", time.Date(2026, 5, 22, 17, 59, 0, 0, time.UTC), "weekend+0d", cFgDark},
		// Inside the window — yellow "weekend!".
		{"fri 6pm sharp", time.Date(2026, 5, 22, 18, 0, 0, 0, time.UTC), "weekend!", cYellow},
		{"fri 11pm", time.Date(2026, 5, 22, 23, 0, 0, 0, time.UTC), "weekend!", cYellow},
		{"sat noon", time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC), "weekend!", cYellow},
		{"sun noon", time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC), "weekend!", cYellow},
		{"mon 2:59am", time.Date(2026, 5, 25, 2, 59, 0, 0, time.UTC), "weekend!", cYellow},
		// Back to countdown at 3am Monday.
		{"mon 3am sharp", time.Date(2026, 5, 25, 3, 0, 0, 0, time.UTC), "weekend+4d", cFgDark},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			text, color := weekendStatus(c.when)
			if text != c.wantText || color != c.wantColor {
				t.Errorf("weekendStatus(%s) = (%q, %q), want (%q, %q)",
					c.when.Format("Mon 15:04"), text, color, c.wantText, c.wantColor)
			}
		})
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

// TestWeatherStrip: the bottom strip combines outlook + AIR/HUM/RAIN
// stats, or replaces the lot with a red hazard headline when an NWS
// alert is firing. The strip's colour is bound to the AQI band so it
// doubles as the air-quality alert lamp.
func TestWeatherStrip(t *testing.T) {
	cases := []struct {
		name, raw, wantText, wantColor string
	}{
		{"plain clear", "63°F|clear||45|62|30", "CLEAR · AIR 45 · HUM 62% · RAIN 30%", cGreen},
		{"AQI 120 → orange band", "63°F|clear||120|62|30", "CLEAR · AIR 120 · HUM 62% · RAIN 30%", cOrange},
		{"hazard wins over the stats", "78°F|hazard|Red Flag Warning|45|62|30", "⚠ Red Flag Warning", cRed},
		{"missing stats → em-dashes", "55°F|fog||||", "FOG · AIR — · HUM — · RAIN —", cFg},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text, color := weatherStrip(tc.raw)
			if text != tc.wantText || color != tc.wantColor {
				t.Errorf("weatherStrip(%q) = (%q,%q), want (%q,%q)",
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

// TestTILBody covers the prefix-stripping + defensive "that " prepend
// rules so the body always flows out of the baked "T I L" wordmark
// as a single grammatical sentence.
func TestTILBody(t *testing.T) {
	cases := []struct {
		name, raw, want string
	}{
		{"TIL that prefix", "TIL|TIL that the Iliad...", "that the Iliad..."},
		{"TIL bare prefix", "TIL|TIL the Iliad...", "that the Iliad..."},
		{"TIL colon prefix", "TIL|TIL: the Iliad...", "that the Iliad..."},
		{"already starts with that", "TIL|that the Iliad...", "that the Iliad..."},
		{"defensive prepend", "TIL|the Iliad...", "that the Iliad..."},
		{"case insensitive", "TIL|til That ...", "that ..."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, _ := tilBody(c.raw)
			if got != c.want {
				t.Errorf("tilBody(%q) = %q, want %q", c.raw, got, c.want)
			}
		})
	}
}

func TestParseTickerList(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want []string
	}{
		{"empty returns nil", "", nil},
		{"whitespace-only returns nil", "   ", nil},
		{"single symbol", "qqq", []string{"QQQ"}},
		{"multi with whitespace", " aapl, msft ,  btc-usd ", []string{"AAPL", "MSFT", "BTC-USD"}},
		{"drops empty entries", "qqq,,aapl,", []string{"QQQ", "AAPL"}},
		{"case normalised", "btc-usd,Eth-USD", []string{"BTC-USD", "ETH-USD"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseTickerList(c.env)
			if len(got) != len(c.want) {
				t.Fatalf("parseTickerList(%q) = %v, want %v", c.env, got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("parseTickerList(%q)[%d] = %q, want %q", c.env, i, got[i], c.want[i])
				}
			}
		})
	}
}

func TestMarketsSymbolPrice(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "qqq with price",
			raw:  "QQQ|$499.32|+1.2|+5.0|▁▂▃|2026-05-20",
			// 19 chars total: "QQQ" (3) + pad + "$499.32" (7) = 19 → pad 9.
			want: "QQQ         $499.32",
		},
		{
			name: "long symbol still gets at least one space",
			raw:  "REALLYLONGSYM|$1,234,567.89|+1|+1|.|2026-05-20",
			// 13 + 12 = 25 > 19 → pad clamps to 1.
			want: "REALLYLONGSYM $1,234,567.89",
		},
		{
			name: "empty raw",
			raw:  "",
			want: "",
		},
		{
			name: "single-segment raw (no price)",
			raw:  "QQQ",
			// 19 - 3 - 0 = 16 spaces
			want: "QQQ                ",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, _ := marketsSymbolPrice(c.raw)
			if got != c.want {
				t.Errorf("marketsSymbolPrice(%q) = %q, want %q", c.raw, got, c.want)
			}
		})
	}
}

func TestSignColor(t *testing.T) {
	cases := []struct {
		v    float64
		want string
	}{
		{1.0, cGreen},
		{0.01, cGreen},
		{-1.0, cRed},
		{-0.01, cRed},
		{0.0, cFgDark},
	}
	for _, c := range cases {
		got := signColor(c.v)
		if got != c.want {
			t.Errorf("signColor(%v) = %q, want %q", c.v, got, c.want)
		}
	}
}

func TestMarketsChange(t *testing.T) {
	cases := []struct {
		name string
		seg  int
		raw  string
		want string
	}{
		{"positive week", 2, "QQQ|$1|+1.2|-3.7|.|d", "▲ +1.2 %"},
		{"negative month", 3, "QQQ|$1|+1.2|-3.7|.|d", "▼ -3.7 %"},
		{"zero is neutral", 2, "QQQ|$1|+0.0|-3.7|.|d", "· +0.0 %"},
		{"missing segment is empty", 2, "QQQ", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, _ := marketsChange(c.seg)(c.raw)
			if got != c.want {
				t.Errorf("marketsChange(%d)(%q) = %q, want %q", c.seg, c.raw, got, c.want)
			}
		})
	}
}

// TestDictionaryStyleDispatch verifies each DictionaryStyle produces the
// expected element count and element IDs. Catches a future renumbering
// or accidental element drop in any of the three layouts.
func TestDictionaryStyleDispatch(t *testing.T) {
	cases := []struct {
		name    string
		style   DictionaryStyle
		wantIDs []int
	}{
		{"manpage (jargon)", StyleManpage, []int{idSceneSub1, idSceneSub2, idSceneSub3}},
		{"punchline (devil)", StylePunchline, []int{idSceneSub1, idSceneSub2}},
		{"ceremony (wordnik)", StyleCeremony, []int{idSceneSub1, idSceneSub2, idSceneSub3}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := DictionaryScene(DictionarySceneOpts{
				Name: "jargon", // a real registry entry so headwordX measurement doesn't barf
				Style: tc.style,
			})
			if len(s.Elements) != len(tc.wantIDs) {
				t.Fatalf("Elements: got %d, want %d", len(s.Elements), len(tc.wantIDs))
			}
			for i, want := range tc.wantIDs {
				if s.Elements[i].ID != want {
					t.Errorf("Elements[%d].ID = %d, want %d", i, s.Elements[i].ID, want)
				}
			}
			if len(s.Mounts) != len(tc.wantIDs) {
				t.Errorf("Mounts: got %d, want %d", len(s.Mounts), len(tc.wantIDs))
			}
		})
	}
}

// TestJargonSeeAlsoExtract: happy path + no-refs + multiple-pattern variants.
func TestJargonSeeAlsoExtract(t *testing.T) {
	cases := []struct {
		name, raw, want string
	}{
		{
			"see also single ref",
			"Jargon File|foo n. A thing. See also bar.|",
			"→ see also: bar",
		},
		{
			"see also multiple refs",
			"Jargon File|foo n. A thing. See also bar, baz, quux.|",
			"→ see also: bar, baz, quux",
		},
		{
			"compare ref",
			"Jargon File|foo n. A thing. Compare bar.|",
			"→ see also: bar",
		},
		{
			"cf. ref",
			"Jargon File|foo n. A thing. Cf. bar.|",
			"→ see also: bar",
		},
		{
			"no refs",
			"Jargon File|foo n. A plain definition with no refs.|",
			"",
		},
		{
			"empty raw",
			"",
			"",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := jargonSeeAlso(tc.raw)
			if got != tc.want {
				t.Errorf("jargonSeeAlso(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// TestDevilHeader: headword + POS → combined "HEADWORD, POS." string.
func TestDevilHeader(t *testing.T) {
	cases := []struct {
		name, raw, want string
	}{
		{
			"noun entry",
			"Devil's Dictionary|BEFRIEND, v.t. To make an ingrate.|Ambrose Bierce",
			"BEFRIEND, v.t.",
		},
		{
			"missing POS — headword only",
			"Devil's Dictionary|FOO bar baz|",
			"FOO",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := devilHeader(tc.raw)
			if got != tc.want {
				t.Errorf("devilHeader(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// TestWordnikHeadwordSpacing: letter-spacing transform inserts a regular
// space between adjacent letters of the headword.
func TestWordnikHeadwordSpacing(t *testing.T) {
	cases := []struct {
		name, raw, want string
	}{
		{
			"single word",
			"Word of the Day|EPHEMERAL, adj. Lasting briefly.||",
			"E P H E M E R A L",
		},
		{
			"two-letter word",
			"Word of the Day|ON, prep. A short word.||",
			"O N",
		},
		{
			"empty",
			"",
			"",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := wordnikHeadword(tc.raw)
			if got != tc.want {
				t.Errorf("wordnikHeadword(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}
