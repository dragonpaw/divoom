package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Did you know?" — promoted out of the whimsy rotator into its own
// scene so the bold question-mark glyph in the bottom-right corner
// gets to be the dominant visual signature. One body Text for the
// fact prose; the "did you know?" header from the widget's
// "did you know?|<body>" output is dropped — the glyph carries the
// label work. Element count 4 (3 top + 1 body) collides with the
// rare easter scene and catfacts; Driver.pick()'s same-count rule
// blocks direct transitions between them, which is fine.
func didyouknowScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "didyouknow",
		Weight: 20,
		BgPath: bgDidYouKnow,
		Elements: []frame.DispElement{
			sceneTitle("did you know?"),
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 540, Width: 640, Height: 560,
				Align: 2, FontSize: 38, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["didyouknow"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: pipeAt(1), Geometry: vCenterQuoteBody},
		},
	}
}
