package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "On this day" — historical event for today's calendar date, sourced
// from Wikimedia's free "on this day / events" feed. Widget emits
// "<year>|<event text>"; the year mounts to a big mono accent
// (cOrange, left-aligned) that gives the scene its visual anchor —
// the year is information the always-on header doesn't carry, so it
// earns the space — and the body row carries the event prose in
// fontProse 40pt left-aligned (matching catfacts).
//
// Element count: sceneTitle + year + body = 3 scene Text + always-on
// 2 Text + 1 Time = 5 Text + 1 Time (6 total). Same-count collisions
// are handled by Driver.pick()'s never-same-count rule.
func onthisdayScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "onthisday",
		Weight: WeightEntertaining,
		BgPath: bgOnThisDay,
		Elements: []frame.DispElement{
			// Year accent — big mono, gruvbox orange, left-aligned.
			// Sits just below the sceneTitle row at y=480; height
			// allows the 96pt glyphs to render without clipping.
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 540, Width: 640, Height: 120,
				Align: 0, FontSize: 96, FontID: fontMono,
				FontColor: cOrange, BgColor: cBgHard,
			},
			// Event prose — left-aligned, fontProse 40pt to match
			// catfacts. Tall track (y=700, h=520) so 4-6 line events
			// fit; short one-liners sit at the top and read fine
			// without vCentering tricks.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 700, Width: 640, Height: 520,
				Align: 0, FontSize: 40, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["onthisday"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: pipeAt(0)},
			{ID: idSceneSub1, Format: pipeAt(1)},
		},
	}
}
