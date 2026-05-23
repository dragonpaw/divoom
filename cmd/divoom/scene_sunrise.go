package main

import (
	"strings"
	"time"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// Sunrise scene — a horizontal day-arc with the current time pinned to it
// by a small downward triangle whose x is recomputed at every scene
// activation. Sunrise time sits under the left end of the arc in yellow,
// sunset time under the right end in orange. The arc, three reference
// ticks (sunrise / noon / sunset), and the static labels are baked into
// the bg JPG (see render.SunriseBackground). The daylight-duration
// headline used to be baked too, but the bg is only pushed once at
// daemon startup so the baked headline went stale across day boundaries;
// it now lives in a device Text element fed from pipe[2] of the widget,
// matching how the sunrise/sunset times are wired.
//
// Element count: baked title (0) + daylight (1) + sunrise time (1) +
// sunset time (1) + current-time tick (1) = 4 scene Text + 2 always-on
// Text = 6 Text + 1 Time + 1 Week, within the device's 6-Text-per-scene
// cap.
func sunriseScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "sunrise",
		Weight: 20,
		BgPath: bgSunrise,
		Elements: []frame.DispElement{
			// Daylight headline above the arc — large mono, centred,
			// pipe[2] of the widget ("13h 16m"). Re-rendered per scene
			// activation so it stays correct across midnight rollovers.
			{
				ID: idSceneMain, Type: "Text",
				StartX: 40, StartY: 600, Width: 720, Height: 120,
				Align: 2, FontSize: 96, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Sunrise time — left, yellow, just below the arc's left end.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 40, StartY: 900, Width: 240, Height: 60,
				Align: 2, FontSize: 40, FontID: fontMono,
				FontColor: cYellow, BgColor: cBgHard,
			},
			// Sunset time — right, orange, below the arc's right end.
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 520, StartY: 900, Width: 240, Height: 60,
				Align: 2, FontSize: 40, FontID: fontMono,
				FontColor: cOrange, BgColor: cBgHard,
			},
			// Current-time tick — a small downward triangle whose StartX
			// is set by OnActivate to land on the arc at the fraction of
			// today's daylight already elapsed. TextMessage is the BLACK
			// DOWN-POINTING TRIANGLE U+25BC; the device font renders it
			// inside this 40x40 slot.
			{
				ID: idSceneSub3, Type: "Text",
				StartX: 380, StartY: 800, Width: 40, Height: 40,
				Align: 2, FontSize: 24, FontID: fontProseLight,
				FontColor: cFg, BgColor: cBgHard,
				TextMessage: "▼",
			},
		},
		Widget: widgets["sunrise"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: pipeAt(2)},
			{ID: idSceneSub1, Format: pipeAt(0)},
			{ID: idSceneSub2, Format: pipeAt(1)},
		},
		OnActivate: sunrisePositionTick,
	}
}

// sunrisePositionTick walks the bottom elements, finds the current-time
// tick, and rewrites its StartX based on `now`'s position between today's
// sunrise and sunset times (parsed from the widget's raw output:
// "h:mm AM|h:mm PM|<daylight>"). The widget output is local-clock so we
// build today's sunrise / sunset moments in now.Location(). When the
// widget hasn't fetched yet, or either time fails to parse, the tick is
// hidden by clearing its TextMessage rather than drawing it at a
// misleading position.
func sunrisePositionTick(now time.Time, raw string, elements []frame.DispElement) {
	tickIdx := -1
	for i := range elements {
		if elements[i].ID == idSceneSub3 {
			tickIdx = i
			break
		}
	}
	if tickIdx < 0 {
		return
	}
	parts := strings.Split(raw, "|")
	if len(parts) < 2 {
		elements[tickIdx].TextMessage = ""
		return
	}
	local := now.Local()
	rise, ok1 := parseClockToday(parts[0], local)
	set, ok2 := parseClockToday(parts[1], local)
	if !ok1 || !ok2 || !set.After(rise) {
		elements[tickIdx].TextMessage = ""
		return
	}
	elements[tickIdx].StartX = sunriseTickX(local, rise, set)
}

// parseClockToday parses a "3:04 PM" clock string and returns the
// corresponding moment on `today`'s date in today's location. Returns
// ok=false on parse failure.
func parseClockToday(s string, today time.Time) (time.Time, bool) {
	t, err := time.Parse("3:04 PM", strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(today.Year(), today.Month(), today.Day(),
		t.Hour(), t.Minute(), 0, 0, today.Location()), true
}

// sunriseTickX returns the StartX (top-left of the 40px-wide tick
// element) that centres the tick on the day-arc point corresponding to
// `now`'s fraction of [rise, set]. The arc runs from x=80 (sunrise end)
// to x=720 (sunset end); subtracting 20 from the tick-point centres the
// glyph. Before sunrise the tick clamps to the left edge (x=60); after
// sunset it clamps to the right edge (x=700) so the indicator stays in
// view but never wanders off the arc.
func sunriseTickX(now, rise, set time.Time) int {
	const (
		arcLeft  = 80
		arcRight = 720
		halfW    = 20 // half of the tick element's width
	)
	if !now.After(rise) {
		return arcLeft - halfW
	}
	if !now.Before(set) {
		return arcRight - halfW
	}
	frac := float64(now.Sub(rise)) / float64(set.Sub(rise))
	if frac < 0 {
		frac = 0
	} else if frac > 1 {
		frac = 1
	}
	return arcLeft + int(frac*float64(arcRight-arcLeft)) - halfW
}
