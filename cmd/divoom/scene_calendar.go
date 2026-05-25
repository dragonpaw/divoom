package main

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// Calendar — calendar-grid redesign. The bg bakes a 12-row × 31-col
// grid of day cells (past = orange, today = yellow border, future = dim,
// phantom cells like Feb 30 = bg-hard, US federal holidays = aqua + a
// single-letter mark, special user dates = red + a single-letter mark,
// future weekends = lifted bg-darker) plus the month-letter labels down
// the left edge. The big NN% percentage still anchors the scene; the
// season label under the grid is colour-coded per current season
// (WINTER/SPRING/SUMMER/AUTUMN → aqua/green/yellow/orange). The "Year
// 2026" row and "year progress" caption are gone — the always-on header
// carries the date already and the grid + bar at the bottom make the
// year-progress metaphor explicit.
//
// Element count: 3 body Text + always-on 2 Text + 1 Time = 5 Text + 1
// Time, one slot under the cap.
func calendarScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "calendar",
		Weight: WeightInformational,
		BgPath: bgCalendar,
		Elements: []frame.DispElement{
			// Big NN% headline.
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 510, Width: 640, Height: 200,
				Align: 2, FontSize: 180, FontID: fontMono,
				FontColor: cOrange, BgColor: cBgHard,
			},
			// Season label — left half of the body row under the grid.
			// FontColor is set per current month by calendarSeason
			// (OnActivate); the default cFg here is only a fallback.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 1050, Width: 300, Height: 60,
				Align: 0, FontSize: 32, FontID: fontProseLight,
				FontColor: cFg, BgColor: cBgHard,
			},
			// "day N of 365" — right half of the body row.
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 400, StartY: 1050, Width: 320, Height: 60,
				Align: 2, FontSize: 28, FontID: fontMono,
				FontColor: cFgDark, BgColor: cBgHard,
			},
		},
		Widget: widgets["calendar"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: pipeAt(0)},
			{ID: idSceneSub2, Format: pipeAt(2)},
		},
		OnActivate:     calendarSeason,
		WeightModifier: calendarModifier,
	}
}

// calendarSeason sets the season label's text and accent colour based
// on the current month. Mirrors the sunrise scene's OnActivate pattern.
func calendarSeason(now time.Time, _ string, elements []frame.DispElement) {
	name, color := seasonAt(now)
	for i := range elements {
		if elements[i].ID == idSceneSub1 {
			elements[i].TextMessage = name
			elements[i].FontColor = color
		}
	}
}

// parseSpecialDates parses the DIVOOM_SPECIAL_DATES env var into a
// map keyed by month*100+day. Format: "MM-DD:LETTER,MM-DD:LETTER,…".
// Whitespace around pairs and around the colon is tolerated. Malformed
// individual entries are logged and skipped — they don't poison the
// whole map. Empty / unset input returns an empty (non-nil) map.
func parseSpecialDates(env string) map[int]rune {
	out := make(map[int]rune)
	if strings.TrimSpace(env) == "" {
		return out
	}
	for _, raw := range strings.Split(env, ",") {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		colon := strings.IndexByte(entry, ':')
		if colon < 0 {
			slog.Warn("DIVOOM_SPECIAL_DATES: missing ':' in entry, skipping", "entry", entry)
			continue
		}
		datePart := strings.TrimSpace(entry[:colon])
		letterPart := strings.TrimSpace(entry[colon+1:])
		dash := strings.IndexByte(datePart, '-')
		if dash < 0 {
			slog.Warn("DIVOOM_SPECIAL_DATES: missing '-' in date, skipping", "entry", entry)
			continue
		}
		month, err1 := strconv.Atoi(strings.TrimSpace(datePart[:dash]))
		day, err2 := strconv.Atoi(strings.TrimSpace(datePart[dash+1:]))
		if err1 != nil || err2 != nil || month < 1 || month > 12 || day < 1 || day > 31 {
			slog.Warn("DIVOOM_SPECIAL_DATES: bad MM-DD, skipping", "entry", entry)
			continue
		}
		letters := []rune(letterPart)
		if len(letters) != 1 {
			slog.Warn("DIVOOM_SPECIAL_DATES: letter must be exactly one rune, skipping", "entry", entry)
			continue
		}
		out[month*100+day] = letters[0]
	}
	return out
}

// usFederalHolidays returns the 11 US federal holidays for `year` as a
// map keyed by month*100+day → a single-letter mark, mirroring the
// shape parseSpecialDates emits. Only the actual calendar date is
// marked — no observed-day weekend shifting.
func usFederalHolidays(year int) map[int]rune {
	out := make(map[int]rune, 11)
	put := func(t time.Time, letter rune) {
		out[int(t.Month())*100+t.Day()] = letter
	}
	put(time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC), 'N')
	put(nthWeekdayOfMonth(year, time.January, time.Monday, 3), 'M')
	put(nthWeekdayOfMonth(year, time.February, time.Monday, 3), 'P')
	put(lastWeekdayOfMonth(year, time.May, time.Monday), 'R')
	put(time.Date(year, time.June, 19, 0, 0, 0, 0, time.UTC), 'J')
	put(time.Date(year, time.July, 4, 0, 0, 0, 0, time.UTC), '4')
	put(nthWeekdayOfMonth(year, time.September, time.Monday, 1), 'L')
	put(nthWeekdayOfMonth(year, time.October, time.Monday, 2), 'C')
	put(time.Date(year, time.November, 11, 0, 0, 0, 0, time.UTC), 'V')
	put(nthWeekdayOfMonth(year, time.November, time.Thursday, 4), 'T')
	put(time.Date(year, time.December, 25, 0, 0, 0, 0, time.UTC), 'X')
	return out
}

// nthWeekdayOfMonth returns the nth (1-based) occurrence of `weekday`
// in (year, month). e.g. nthWeekdayOfMonth(2026, January, Monday, 3)
// returns the 3rd Monday of January 2026.
func nthWeekdayOfMonth(year int, month time.Month, weekday time.Weekday, n int) time.Time {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	offset := (int(weekday) - int(first.Weekday()) + 7) % 7
	day := 1 + offset + (n-1)*7
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// lastWeekdayOfMonth returns the last occurrence of `weekday` in
// (year, month) — e.g. the last Monday of May for Memorial Day.
func lastWeekdayOfMonth(year int, month time.Month, weekday time.Weekday) time.Time {
	// Start from the first day of the next month, step back to the
	// previous occurrence of weekday.
	next := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
	last := next.AddDate(0, 0, -1)
	offset := (int(last.Weekday()) - int(weekday) + 7) % 7
	return last.AddDate(0, 0, -offset)
}
