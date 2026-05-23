package render

import (
	"math"
	"testing"
	"time"
)

// TestRenderMoonDisc renders each of the 14 pre-rendered moonphase
// variants and verifies each one is a non-trivial JPEG (>5KB), which
// catches a render path that silently emits an empty / single-colour
// canvas. Visual correctness (waxing on the right, waning on the left)
// is checked manually via `divoom render`.
func TestRenderMoonDisc(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	for i := 0; i < MoonPhaseVariants; i++ {
		data, err := SceneMoonphaseBackground(i, FormatJPEG, now)
		if err != nil {
			t.Fatalf("SceneMoonphaseBackground(%d): %v", i, err)
		}
		if len(data) < 5*1024 {
			t.Errorf("SceneMoonphaseBackground(%d): JPEG too small (%d bytes)", i, len(data))
		}
	}
}

// TestMoonIllumFractionForIndex pins the lit-fraction sample at the
// cycle anchors (new, first quarter, full, last quarter) so the disc
// carve geometry can't silently drift.
func TestMoonIllumFractionForIndex(t *testing.T) {
	cases := []struct {
		idx       int
		wantFrac  float64
		tolerance float64
	}{
		{0, 0.0, 0.001},   // new
		{7, 1.0, 0.001},   // full
		{4, 0.611, 0.001}, // late-waxing, past first quarter
		{10, 0.611, 0.001}, // mirror on the waning side
		{11, 0.389, 0.001}, // waning, past last quarter
		{3, 0.389, 0.001},  // mirror on the waxing side
	}
	for _, tc := range cases {
		got := MoonIllumFractionForIndex(tc.idx)
		if math.Abs(got-tc.wantFrac) > tc.tolerance {
			t.Errorf("MoonIllumFractionForIndex(%d) = %.3f, want %.3f ± %.3f",
				tc.idx, got, tc.wantFrac, tc.tolerance)
		}
	}
}


// TestDayOfYearGridCellState exercises the dayofyear calendar grid's
// per-cell state classifier. Past/today/future/phantom/special and
// special+today combinations are covered so the priority ordering on
// drawDayOfYearGrid can't silently drift.
func TestDayOfYearGridCellState(t *testing.T) {
	// Anchor today as 2026-05-22 (the working session's date).
	today := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	special := map[int]rune{
		522:  'T', // today is a special date
		1225: 'X', // future special
		113:  'A', // past special
	}
	cases := []struct {
		name           string
		month, day     int
		specialDates   map[int]rune
		want           dayOfYearCellState
	}{
		{"past day", 1, 15, special, dayOfYearPast},
		{"today (no special)", 5, 22, nil, dayOfYearToday},
		{"today (with special)", 5, 22, special, dayOfYearSpecial},
		{"future day", 8, 1, special, dayOfYearFuture},
		{"phantom Feb 30", 2, 30, special, dayOfYearPhantom},
		{"phantom Apr 31", 4, 31, special, dayOfYearPhantom},
		{"past special", 1, 13, special, dayOfYearPast}, // 113 not in specialDates above; rebuild
		{"future special", 12, 25, special, dayOfYearSpecial},
	}
	// Fix up the "past special" case — 113 IS in special above.
	cases[6].want = dayOfYearSpecial
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := dayOfYearCellStateFor(tc.month, tc.day, today, tc.specialDates)
			if got != tc.want {
				t.Errorf("dayOfYearCellStateFor(%d/%d) = %d, want %d", tc.month, tc.day, got, tc.want)
			}
		})
	}
}

// TestDayOfYearBackgroundRenders smoke-tests the dayofyear bg builder
// end-to-end so a refactor that breaks the JPEG path is caught.
func TestDayOfYearBackgroundRenders(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	data, err := DayOfYearBackground(now, map[int]rune{522: 'T', 1225: 'X'}, FormatJPEG)
	if err != nil {
		t.Fatalf("DayOfYearBackground: %v", err)
	}
	if len(data) < 5*1024 {
		t.Errorf("DayOfYearBackground: JPEG too small (%d bytes)", len(data))
	}
}
