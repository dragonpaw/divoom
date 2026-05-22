package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "TIL" — top post of the day from r/todayilearned. Mirrors the
// catfacts shape exactly (single body Text, vCentered, header
// dropped via pipeAt(1)); the lightbulb glyph in the bottom-right
// corner carries the "TIL" label work. Element count 4 (3 top + 1
// body) collides with easter / catfacts / didyouknow; Driver.pick()'s
// same-count rule blocks direct transitions between them, which
// is fine.
func tilScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "til",
		Weight: 20,
		BgPath: bgTIL,
		Elements: []frame.DispElement{
			sceneTitle("today I learned"),
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 20, StartY: 540, Width: 760, Height: 560,
				Align: 2, FontSize: 38, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["til"],
		Mounts: []scene.Mount{
			{ID: idSceneSub1, Format: pipeAt(1), Geometry: vCenterQuoteBody},
		},
	}
}
