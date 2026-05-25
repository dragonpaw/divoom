package main

import "testing"

// TestForecastBar pins the unicode-block bar algorithm against the
// concrete cases from the design spec: full span, degenerate flat
// week, half-range bars, narrow middle range, single-point overlap.
func TestForecastBar(t *testing.T) {
	cases := []struct {
		name                          string
		dayLo, dayHi, weekLo, weekHi  int
		want                          string
	}{
		// Day range covers the whole week → every cell completely full.
		{"full span", 50, 80, 50, 80, "██████"},

		// Degenerate flat week (no spread): render a stable full bar
		// instead of dividing by zero.
		{"zero span", 60, 60, 60, 60, "██████"},

		// Day range exactly the lower half of the week → left 3 cells
		// full, right 3 cells empty.
		{"low half", 50, 65, 50, 80, "███   "},

		// Day range exactly the upper half of the week → right 3 cells
		// full, left 3 cells empty.
		{"high half", 65, 80, 50, 80, "   ███"},

		// Narrow middle range — day spans the middle third of the week
		// (cells 2-3). cellWidth = 30/6 = 5; day = [60,70] → cells 0,1
		// empty, 2,3 full, 4,5 empty.
		{"middle third", 60, 70, 50, 80, "  ██  "},

		// Single-point day (dayLo == dayHi) in a wide week. The point
		// lands at one cell boundary; nothing has positive overlap, so
		// the bar is all blanks. (Documents the boundary behaviour.)
		{"single point", 60, 60, 50, 80, "      "},
	}
	for _, c := range cases {
		got := forecastBar(c.dayLo, c.dayHi, c.weekLo, c.weekHi)
		if got != c.want {
			t.Errorf("%s: forecastBar(%d,%d,%d,%d) = %q, want %q",
				c.name, c.dayLo, c.dayHi, c.weekLo, c.weekHi, got, c.want)
		}
	}
}

// TestForecastRow exercises the new pipe-separated layout:
// "WHI|WLO|DAY|HI|LO|OUTLOOK|PRECIP|…" with 5 segments per day. Verifies
// the bar is rendered, the precip suffix appears only above the 30%
// threshold, the slash is tightened, and the row colour follows the
// outlook.
func TestForecastRow(t *testing.T) {
	// Week span 80..50 → cellWidth = 5. Day 0 ("sun") spans 53..62 →
	// crosses cells covering [50,55),[55,60),[60,65), full+full at the
	// middle, partial at edges.
	raw := "80|50|sun|62|53|sunny|30|mon|75|60|cloudy|10|tue|55|50|rainy|80|wed|70|65|partly|0"

	t.Run("row 0 sun with precip suffix", func(t *testing.T) {
		text, color := forecastRow(0)(raw)
		// 6-char bar + 2 spaces + " 62°/ 53°" + "  ·30%"
		// Bar: bar across [53,62] in week [50,80].
		want := "sun   ▃█▃      62°/ 53°  ·30%"
		if text != want {
			t.Errorf("row 0 text = %q, want %q", text, want)
		}
		if color != weatherOutlookColor("sunny") {
			t.Errorf("row 0 color = %s, want %s", color, weatherOutlookColor("sunny"))
		}
	})

	t.Run("row 1 mon no precip suffix (<30%)", func(t *testing.T) {
		text, _ := forecastRow(1)(raw)
		// precip 10 → suffix omitted entirely
		if hasSuffix(text, "%") {
			t.Errorf("row 1 should omit precip suffix, got %q", text)
		}
	})

	t.Run("row 3 wed zero precip", func(t *testing.T) {
		text, _ := forecastRow(3)(raw)
		if hasSuffix(text, "%") {
			t.Errorf("row 3 should omit precip suffix at 0%%, got %q", text)
		}
	})

	t.Run("malformed input returns empty", func(t *testing.T) {
		text, _ := forecastRow(2)("80|50|sun|62|53|sunny|30") // only one day
		if text != "" {
			t.Errorf("expected empty row for missing day, got %q", text)
		}
	})
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
