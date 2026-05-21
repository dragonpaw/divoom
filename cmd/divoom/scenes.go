package main

import (
	"strings"
	"time"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// Element IDs. Always-on top reserves 1-2; scene primaries start at 10.
// Each scene's layout is its own install, so re-using IDs across scenes is
// fine; we keep the IDs distinct only within a single scene.
const (
	idTime = 1
	idDate = 2

	idSceneMain = 10
	idSceneSub1 = 11
	idSceneSub2 = 12
	idSceneSub3 = 13
	idSceneSub4 = 14
)

// Font IDs on the device. Both custom-pushed via adb (see docs/api.md →
// "Fonts on disk"). Iosevka for digits/symbols, Roboto Condensed for prose.
const (
	fontMono  = 7 // Iosevka — numbers, ticker symbols, percentages
	fontProse = 9 // Roboto Condensed — labels, prose, condition words
)

// On-device per-scene background paths. We adb-push one JPG per scene so
// each can have its own glyph in the bottom area.
const (
	bgNow     = "/userdata/wallclock_bg_now.jpg"
	bgMarkets = "/userdata/wallclock_bg_markets.jpg"
	bgSky     = "/userdata/wallclock_bg_sky.jpg"
	bgAmbient = "/userdata/wallclock_bg_ambient.jpg"
)

// Gruvbox semantic colors. Reds and greens signal direction (down/up);
// yellow / blue / aqua signal weather conditions; fg / fg-dark are quiet.
const (
	cFg      = "#ebdbb2"
	cFgDark  = "#a89984"
	cRed     = "#fb4934"
	cGreen   = "#b8bb26"
	cYellow  = "#fabd2f"
	cBlue    = "#83a598"
	cAqua    = "#8ec07b"
	cPurple  = "#d3869b"
	cOrange  = "#fe8019"
	cBgHard  = "#1d2021"
)

// Vertical layout (800x1280 portrait):
//   y=120-340   Time (huge, fg-tan, always on)
//   y=370-430   Date (built-in, fg-dark, always on)
//   y=460-462   Hairline divider (rendered into bg)
//   y=480-1240  Scene-specific content
//   y=1268-1272 Year-progress bar (rendered into bg)

func alwaysOn() []frame.DispElement {
	return []frame.DispElement{
		{
			ID: idTime, Type: "Time",
			StartX: 50, StartY: 120, Width: 700, Height: 220,
			Align:     2,
			FontSize:  180,
			FontID:    fontMono,
			FontColor: cFg,
			BgColor:   cBgHard,
		},
		{
			ID: idDate, Type: "Date",
			StartX: 50, StartY: 370, Width: 700, Height: 60,
			Align:     2,
			FontSize:  44,
			FontID:    fontProse,
			FontColor: cFgDark,
			BgColor:   cBgHard,
		},
	}
}

func buildScenes(weather, qqq, moon, whimsy *widget.Runner) []scene.Scene {
	return []scene.Scene{
		// "Now" — weather. Temperature huge on top, condition word below,
		// both colored by the current condition (yellow for clear, blue
		// for rain, etc.). The widget returns "<temp>° <condition>"; the
		// Format functions split it.
		{
			Name:     "now",
			Duration: 60 * time.Second,
			BgPath:   bgNow,
			Elements: []frame.DispElement{
				{
					ID: idSceneMain, Type: "Text",
					StartX: 20, StartY: 540, Width: 760, Height: 220,
					Align: 2, FontSize: 180, FontID: fontMono,
					FontColor: cFg, BgColor: cBgHard,
				},
				{
					ID: idSceneSub1, Type: "Text",
					StartX: 20, StartY: 820, Width: 760, Height: 110,
					Align: 2, FontSize: 70, FontID: fontProse,
					FontColor: cFg, BgColor: cBgHard,
				},
			},
			Mounts: []scene.Mount{
				{ID: idSceneMain, Runner: weather, Format: weatherTemp},
				{ID: idSceneSub1, Runner: weather, Format: weatherCondition},
			},
		},

		// "Markets" — QQQ stack: symbol on top, then a (percent, label) pair
		// for week and again for month. Percents take green/red by sign;
		// labels stay in fg-dark so they read as captions.
		{
			Name:     "markets",
			Duration: 30 * time.Second,
			BgPath:   bgMarkets,
			Elements: []frame.DispElement{
				// QQQ symbol header
				{
					ID: idSceneMain, Type: "Text",
					StartX: 30, StartY: 510, Width: 740, Height: 130,
					Align: 2, FontSize: 110, FontID: fontMono,
					FontColor: cFg, BgColor: cBgHard,
				},
				// Week percent (color set by formatter)
				{
					ID: idSceneSub1, Type: "Text",
					StartX: 30, StartY: 680, Width: 740, Height: 110,
					Align: 2, FontSize: 84, FontID: fontMono,
					FontColor: cFgDark, BgColor: cBgHard,
				},
				// "week" label
				{
					ID: idSceneSub2, Type: "Text",
					StartX: 30, StartY: 800, Width: 740, Height: 50,
					Align: 2, FontSize: 30, FontID: fontProse,
					FontColor: cFgDark, BgColor: cBgHard,
					TextMessage: "week",
				},
				// Month percent (color set by formatter)
				{
					ID: idSceneSub3, Type: "Text",
					StartX: 30, StartY: 890, Width: 740, Height: 110,
					Align: 2, FontSize: 84, FontID: fontMono,
					FontColor: cFgDark, BgColor: cBgHard,
				},
				// "month" label
				{
					ID: idSceneSub4, Type: "Text",
					StartX: 30, StartY: 1010, Width: 740, Height: 50,
					Align: 2, FontSize: 30, FontID: fontProse,
					FontColor: cFgDark, BgColor: cBgHard,
					TextMessage: "month",
				},
			},
			Mounts: []scene.Mount{
				{ID: idSceneMain, Runner: qqq, Format: qqqSymbol},
				{ID: idSceneSub1, Runner: qqq, Format: qqqWeekPct},
				{ID: idSceneSub3, Runner: qqq, Format: qqqMonthPct},
			},
		},

		// "Sky" — moon phase and illumination percent on separate rows.
		// Both colored gruvbox blue (ambient/sky).
		{
			Name:     "sky",
			Duration: 30 * time.Second,
			BgPath:   bgSky,
			Elements: []frame.DispElement{
				{
					ID: idSceneMain, Type: "Text",
					StartX: 20, StartY: 620, Width: 760, Height: 150,
					Align: 2, FontSize: 80, FontID: fontProse,
					FontColor: cBlue, BgColor: cBgHard,
				},
				{
					ID: idSceneSub1, Type: "Text",
					StartX: 30, StartY: 820, Width: 740, Height: 120,
					Align: 2, FontSize: 72, FontID: fontMono,
					FontColor: cFgDark, BgColor: cBgHard,
				},
			},
			Mounts: []scene.Mount{
				{ID: idSceneMain, Runner: moon, Format: moonPhaseName},
				{ID: idSceneSub1, Runner: moon, Format: moonIllum},
			},
		},

		// "Ambient" — whimsy rotator. Each whimsy source emits HEADER|BODY;
		// header renders small at the top, body renders larger below.
		// Body height accommodates 3-4 wrapped lines at FontSize 34.
		{
			Name:     "ambient",
			Duration: 30 * time.Second,
			BgPath:   bgAmbient,
			Elements: []frame.DispElement{
				// Header label
				{
					ID: idSceneMain, Type: "Text",
					StartX: 20, StartY: 510, Width: 760, Height: 60,
					Align: 2, FontSize: 36, FontID: fontProse,
					FontColor: cFgDark, BgColor: cBgHard,
				},
				// Body / fact
				{
					ID: idSceneSub1, Type: "Text",
					StartX: 20, StartY: 600, Width: 760, Height: 540,
					Align: 2, FontSize: 34, FontID: fontProse,
					FontColor: cFg, BgColor: cBgHard,
				},
			},
			Mounts: []scene.Mount{
				{ID: idSceneMain, Runner: whimsy, Format: pipeHead},
				{ID: idSceneSub1, Runner: whimsy, Format: pipeTail},
			},
		},
	}
}

// --- weather formatters ---
//
// The weather widget returns "<temp>° <condition>" (e.g. "63° clear").
// We split it into two elements and color both by condition so the whole
// scene reads as one piece of information.

func weatherTemp(s string) (text, color string) {
	parts := strings.SplitN(s, " ", 2)
	color = weatherColor(s)
	if len(parts) == 0 || s == "" {
		return s, color
	}
	return parts[0], color
}

func weatherCondition(s string) (text, color string) {
	parts := strings.SplitN(s, " ", 2)
	color = weatherColor(s)
	if len(parts) < 2 {
		return "", color
	}
	return parts[1], color
}

// weatherColor maps a weather widget string to a gruvbox accent. The
// condition word is enough to bucket — we don't need the WMO code.
func weatherColor(text string) string {
	t := strings.ToLower(text)
	switch {
	case strings.Contains(t, "thunder"):
		return cRed
	case strings.Contains(t, "rain"), strings.Contains(t, "drizzle"):
		return cBlue
	case strings.Contains(t, "snow"):
		return cFg
	case strings.Contains(t, "fog"):
		return cAqua
	case strings.Contains(t, "overcast"), strings.Contains(t, "cloudy"):
		return cFgDark
	case strings.Contains(t, "clear"):
		return cYellow
	default:
		return cFg
	}
}

// --- QQQ formatters ---
//
// Widget output: "QQQ  -0.6% 1W   +9.2% 1M". Split into three rows;
// week and month rows take their color from the sign of their value.

func qqqSymbol(s string) (text, color string) {
	if parts := strings.Fields(s); len(parts) > 0 {
		return parts[0], ""
	}
	return s, ""
}

// qqqWeekPct returns just the week percent (e.g. "+9.2%") colored by sign.
// The "1W" suffix is dropped — the row's "week" caption supplies that.
func qqqWeekPct(s string) (text, color string) {
	parts := strings.Fields(s)
	if len(parts) < 2 {
		return s, ""
	}
	return parts[1], directionalColor(parts[1])
}

// qqqMonthPct returns just the month percent, similarly.
func qqqMonthPct(s string) (text, color string) {
	parts := strings.Fields(s)
	if len(parts) < 4 {
		return s, ""
	}
	return parts[3], directionalColor(parts[3])
}

// directionalColor: green for positive (up), red for negative (down),
// neutral for zero or unparseable.
func directionalColor(s string) string {
	switch {
	case strings.HasPrefix(s, "+"):
		return cGreen
	case strings.HasPrefix(s, "-"):
		return cRed
	default:
		return ""
	}
}

// --- HEADER|BODY splitters ---
//
// Used by ambient (whimsy rotator outputs) and calendar (day-of-year).
// Source widgets put a "|" between the small/header text and the
// large/body text; these helpers slice on the first "|".

func pipeHead(s string) (text, color string) {
	if i := strings.IndexByte(s, '|'); i >= 0 {
		return s[:i], ""
	}
	return "", ""
}

func pipeTail(s string) (text, color string) {
	if i := strings.IndexByte(s, '|'); i >= 0 {
		return s[i+1:], ""
	}
	return s, ""
}

// --- moon formatters ---

func moonPhaseName(s string) (text, color string) {
	parts := strings.Split(s, " · ")
	if len(parts) >= 2 {
		return parts[1], ""
	}
	return s, ""
}

func moonIllum(s string) (text, color string) {
	parts := strings.Split(s, " · ")
	if len(parts) >= 3 {
		return parts[2] + " lit", ""
	}
	return "", ""
}
