package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Easter" — rare (~0.5%) treat. The widget emits HEADER|BODY (e.g.
// "musing|the universe is mostly empty space, like your inbox should
// be"). The header mounts to the small dim title row, the body
// renders INSIDE the giant gruvbox-yellow egg in dark text — the
// gruvbox cBgHard-on-yellow pairing reads as printed on the shell
// rather than floating above it. The baked hairline crack across
// the egg's upper third and the "rare drop · ~1 in 200" footer
// (both in render.drawEasterEgg) carry the rarity signal that the
// plain ellipse used to lack.
//
// Element count: title + body = 2 scene Text + always-on 3 Text + 1
// Time = 5 Text + 1 Time (6 total).
func easterScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "easter",
		Weight: 1,
		BgPath: bgEaster,
		Elements: []frame.DispElement{
			// Header row — variable per phrase ("musing", "today's
			// vibe", "reminder", …). fontProseLight 26pt cFgDark
			// matches the canonical sceneTitle but is widget-driven.
			{
				ID: idSceneTitle, Type: "Text",
				StartX: 80, StartY: 480, Width: 640, Height: 40,
				Align: 2, FontSize: 26, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
			},
			// Body — sits INSIDE the egg's mid-belly. cBgHard on the
			// egg's GruvYellow fill is the gruvbox dark-on-light
			// pairing used elsewhere for high-contrast captions.
			// Rect kept inside the egg's narrowest horizontal slice
			// (egg cx=400, narrowest half-width ≈205 at y=1020) so
			// the BgColor=cYellow doesn't paint yellow corners
			// outside the shell. StartY=820 sits below the baked
			// hairline crack at y≈777 so first line clears it.
			{
				ID: idSceneMain, Type: "Text",
				StartX: 200, StartY: 820, Width: 400, Height: 200,
				Align: 2, FontSize: 38, FontID: fontProse,
				FontColor: cBgHard, BgColor: cYellow,
			},
		},
		Widget: widgets["easter"],
		Mounts: []scene.Mount{
			{ID: idSceneTitle, Format: pipeAt(0)},
			{ID: idSceneMain, Format: pipeAt(1)},
		},
	}
}
