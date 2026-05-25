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


// TestCalendarGridCellState exercises the calendar grid's per-cell
// state classifier. Past / today / future / phantom / special /
// holiday / future-weekend / collisions are covered so the priority
// ordering on drawCalendarGrid can't silently drift.
func TestCalendarGridCellState(t *testing.T) {
	// Anchor today as 2026-05-22 (a Friday).
	today := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	special := map[int]rune{
		522:  'T', // today is a special date
		1225: 'X', // future special
		113:  'A', // past special
		704:  'B', // collides with Independence Day below
	}
	holidays := map[int]rune{
		704:  '4', // Independence Day (future this year)
		1119: 'V', // an invented holiday day used to test cell state
	}
	// A known future Saturday for the weekend case: 2026-05-23.
	// A known past Saturday for the past-not-weekend case: 2026-05-16.
	cases := []struct {
		name         string
		month, day   int
		specialDates map[int]rune
		holidays     map[int]rune
		want         calendarCellState
	}{
		{"past plain day", 1, 15, nil, nil, calendarPastPlain},
		{"today (no special, no holiday)", 5, 22, nil, nil, calendarToday},
		{"today (with special, today still wins)", 5, 22, special, holidays, calendarToday},
		{"future weekday", 8, 3, nil, nil, calendarFutureWeekday}, // 2026-08-03 is Monday
		{"future weekend (Sat)", 5, 23, nil, nil, calendarFutureWeekend},
		{"past Saturday is plain past (no weekend variant)", 5, 16, nil, nil, calendarPastPlain},
		{"phantom Feb 30", 2, 30, special, holidays, calendarPhantom},
		{"phantom Apr 31", 4, 31, special, holidays, calendarPhantom},
		{"past special (Jan 13)", 1, 13, special, holidays, calendarPastSpecial},
		{"past holiday (no past holiday in this fixture — invent one)", 1, 1,
			nil,
			map[int]rune{101: 'N'},
			calendarPastHoliday},
		{"future special (Dec 25)", 12, 25, special, holidays, calendarFutureSpecial},
		{"future holiday (Jul 4)", 7, 4, nil, holidays, calendarFutureHoliday},
		{"future holiday vs special collision (special wins)", 7, 4, special, holidays, calendarFutureSpecial},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := calendarCellStateFor(tc.month, tc.day, today, tc.specialDates, tc.holidays)
			if got != tc.want {
				t.Errorf("calendarCellStateFor(%d/%d) = %d, want %d", tc.month, tc.day, got, tc.want)
			}
		})
	}
}

// TestCalendarBackgroundRenders smoke-tests the calendar bg builder
// end-to-end so a refactor that breaks the JPEG path is caught.
func TestCalendarBackgroundRenders(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	data, err := CalendarBackground(now,
		map[int]rune{522: 'T', 1225: 'X'},
		map[int]rune{704: '4', 1225: 'X'},
		FormatJPEG)
	if err != nil {
		t.Fatalf("CalendarBackground: %v", err)
	}
	if len(data) < 5*1024 {
		t.Errorf("CalendarBackground: JPEG too small (%d bytes)", len(data))
	}
}
