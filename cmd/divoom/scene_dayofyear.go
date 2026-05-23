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

// DayOfYear — calendar-grid redesign. The bg bakes a 12-row × 31-col
// grid of day cells (past = orange, today = yellow border, future = dim,
// phantom cells like Feb 30 = bg-hard, special user dates = red + a
// single-letter mark) plus the month-letter labels down the left edge.
// The big NN% percentage still anchors the scene; the season label
// under the grid is colour-coded per current season (WINTER/SPRING/
// SUMMER/AUTUMN → aqua/green/yellow/orange). The "Year 2026" row and
// "year progress" caption are gone — the always-on header carries the
// date already and the grid + bar at the bottom make the year-progress
// metaphor explicit.
//
// Element count: 3 body Text + always-on 2 Text + 1 Time = 5 Text + 1
// Time, one slot under the cap.
func dayofyearScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "dayofyear",
		Weight: WeightInformational,
		BgPath: bgDayOfYear,
		Elements: []frame.DispElement{
			// Big NN% headline.
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 510, Width: 640, Height: 200,
				Align: 2, FontSize: 180, FontID: fontMono,
				FontColor: cOrange, BgColor: cBgHard,
			},
			// Season label — left half of the body row under the grid.
			// FontColor is set per current month by dayofyearSeason
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
		Widget: widgets["dayofyear"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: pipeAt(0)},
			{ID: idSceneSub2, Format: pipeAt(2)},
		},
		OnActivate: dayofyearSeason,
	}
}

// dayofyearSeason sets the season label's text and accent colour based
// on the current month. Mirrors the sunrise scene's OnActivate pattern.
func dayofyearSeason(now time.Time, _ string, elements []frame.DispElement) {
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
