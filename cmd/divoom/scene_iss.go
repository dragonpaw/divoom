package main

import (
	"time"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "ISS" — sub-satellite-point tracker. The scene's background bakes a
// dim equirectangular world map plus a fixed telemetry strip
// ("●  ISS  ·  408km altitude  ·  7.66km/s") so the body only has to
// install three live Text elements:
//
//   - a yellow ● dot positioned over the current lat/lon (recomputed
//     by OnActivate at every activation from the widget's pipe[0]),
//   - a "over <location>" location line, and
//   - a coords-and-next-pass row.
//
// The widget emits "<lat>°, <lon>°|<next-pass>|over <region>"; the
// pass and region segments may be empty when their respective
// upstreams flake out, so the body mounts use AllowEmpty.
func issScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "iss",
		Weight: 20,
		BgPath: bgISS,
		Elements: []frame.DispElement{
			// Sub-satellite dot — single ● glyph, positioned per
			// activation. StartX/StartY here are placeholders; the
			// real values come from OnActivate. Hidden (StartX=-100)
			// on a parse failure so a bad widget value can't render
			// a stray dot at the map's top-left corner.
			{
				ID: idSceneMain, Type: "Text",
				StartX: -100, StartY: -100, Width: 24, Height: 24,
				Align: 1, FontSize: 24, FontID: fontProse,
				FontColor: cYellow, BgColor: cBgHard,
				TextMessage: "●",
			},
			// Location line — "over <region>", prose, fg.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 970, Width: 640, Height: 70,
				Align: 2, FontSize: 44, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Coords + next-pass row — mono, dim. Composed by
			// issCoordsAndPass from pipe[0] + pipe[1].
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 80, StartY: 1050, Width: 640, Height: 60,
				Align: 2, FontSize: 28, FontID: fontMono,
				FontColor: cFgDark, BgColor: cBgHard,
			},
		},
		Widget: widgets["iss"],
		Mounts: []scene.Mount{
			// The dot element gets no Format — its TextMessage is the
			// literal "●" baked into Elements. The mount on idSceneMain
			// is unnecessary; positioning happens in OnActivate, which
			// runs after Mounts.
			{ID: idSceneSub1, Format: pipeAt(2), AllowEmpty: true},
			{ID: idSceneSub2, Format: issCoordsAndPass, AllowEmpty: true},
		},
		OnActivate: issPositionDot,
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
		// Centre the 24×24 element on the computed point.
		elements[i].StartX = x - 12
		elements[i].StartY = y - 12
		return
	}
}
