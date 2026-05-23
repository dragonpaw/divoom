package main

import (
	"time"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "ISS" — sub-satellite-point tracker. The scene's background bakes a
// dim equirectangular world map outline plus a single hairline under
// the always-on top zone; the telemetry strip
// ("● ISS · 408km altitude · 7.66km/s") above that hairline is now a
// live Text element fed by the widget's altitude + velocity segments,
// so the readings update with the position instead of lying.
//
// Body Text elements (4):
//
//   - a colourful ● dot positioned over the current lat/lon (recomputed
//     by OnActivate at every activation from the widget's pipe[0]),
//   - the live telemetry strip ("● ISS · <alt>km altitude · <vel>km/s"),
//   - an "over <location>" location line,
//   - a combined coords + next-pass row in monospaced text; the whole
//     row turns aqua when the next pass is imminent (within 60
//     minutes), cFgDark otherwise.
//
// The widget emits
// "<lat>°, <lon>°|<next-pass>|over <region>|<altitude>|<velocity>";
// the pass and region segments may be empty when their respective
// upstreams flake out, so the body mounts use AllowEmpty.
//
// Element count: dot (1) + telemetry (1) + location (1) +
// coords/pass (1) = 4 scene Text + 2 always-on = 6 Text. At the
// device's per-type cap.
func issScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "iss",
		Weight: 20,
		BgPath: bgISS,
		Elements: []frame.DispElement{
			// Sub-satellite dot — single ● glyph at FontSize 44 (≈1.8×
			// the previous 24px) so it pops against the dim world-map
			// outline. Yellow against the cFgDark map keeps the live
			// reading unmistakable. StartX/StartY here are placeholders;
			// the real values come from OnActivate. Hidden (StartX=-100)
			// on a parse failure so a bad widget value can't render a
			// stray dot at the map's top-left corner.
			{
				ID: idSceneMain, Type: "Text",
				StartX: -100, StartY: -100, Width: 44, Height: 44,
				Align: 1, FontSize: 44, FontID: fontProse,
				FontColor: cYellow, BgColor: cBgHard,
				TextMessage: "●",
			},
			// Telemetry strip — Iosevka mono 28pt, sits where the old
			// baked text was (left-aligned at x=80, y=510). Always
			// cFgDark — subordinate to the live ● dot.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 510, Width: 640, Height: 30,
				Align: 1, FontSize: 28, FontID: fontMono,
				FontColor: cFgDark, BgColor: cBgHard,
			},
			// Location line — "over <region>", prose, fg.
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 80, StartY: 970, Width: 640, Height: 70,
				Align: 2, FontSize: 44, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Coords + next-pass row — mono. Colour set by
			// issColorizePass: cAqua when the pass is within 60 minutes
			// ("look up!"), cFgDark otherwise.
			{
				ID: idSceneSub3, Type: "Text",
				StartX: 80, StartY: 1080, Width: 640, Height: 50,
				Align: 2, FontSize: 28, FontID: fontMono,
				FontColor: cFgDark, BgColor: cBgHard,
			},
		},
		Widget: widgets["iss"],
		Mounts: []scene.Mount{
			// The dot element gets no Format — its TextMessage is the
			// literal "●" baked into Elements. Positioning happens in
			// OnActivate, which runs after Mounts.
			{ID: idSceneSub1, Format: issTelemetryStrip},
			{ID: idSceneSub2, Format: pipeAt(2), AllowEmpty: true},
			{ID: idSceneSub3, Format: issCoordsAndPass, AllowEmpty: true},
		},
		OnActivate: issOnActivate,
	}
}

// issOnActivate runs both the dot-positioning logic and the next-pass
// colour rule. Combined into one OnActivate because Scene only supports
// a single OnActivate hook.
func issOnActivate(now time.Time, raw string, elements []frame.DispElement) {
	issPositionDot(now, raw, elements)
	issColorizePass(now, raw, elements)
}

// issColorizePass sets the combined coords+next-pass row's FontColor to
// cAqua when the widget's pass duration is within 60 minutes — a "look
// up, it's about to fly over" signal — and leaves it at cFgDark
// otherwise. Missing or unparseable pass values keep the dim default;
// the row's text comes from issCoordsAndPass, which surfaces just the
// coords (no pass suffix) for those cases.
func issColorizePass(_ time.Time, raw string, elements []frame.DispElement) {
	dur, ok := parseISSPassDuration(pipeAtRaw(raw, 1))
	if !ok {
		return
	}
	for i := range elements {
		if elements[i].ID != idSceneSub3 {
			continue
		}
		if dur <= 60*time.Minute {
			elements[i].FontColor = cAqua
		}
		return
	}
}

// issPositionDot recomputes the ISS dot element's StartX/StartY from
// the widget's current lat/lon (pipe[0]). On parse failure the dot is
// hidden by parking it at StartX=-100 so the body element renders
// off-screen rather than at the map's origin.
func issPositionDot(_ time.Time, raw string, elements []frame.DispElement) {
	lat, lon, ok := parseISSCoords(pipeAtRaw(raw, 0))
	for i := range elements {
		if elements[i].ID != idSceneMain {
			continue
		}
		if !ok {
			elements[i].StartX = -100
			elements[i].StartY = -100
			return
		}
		x := issMapX(lon)
		y := issMapY(lat)
		// Centre the 44×44 element on the computed point.
		elements[i].StartX = x - 22
		elements[i].StartY = y - 22
		return
	}
}
