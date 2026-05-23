package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Did you know?" — promoted out of the whimsy rotator into its own
// scene so the question-mark glyph baked into the bottom-right of
// the bg can do label duty alongside the small sceneTitle row.
// The widget emits "did you know?|<body>"; pipeAt(1) drops the
// header half and renders only the fact prose in the body track.
//
// Element count: sceneTitle + body = 2 scene Text + always-on 2
// Text + 1 Time = 4 Text + 1 Time (5 total). Same-count collisions
// are handled by Driver.pick()'s never-same-count rule.
func didyouknowScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "didyouknow",
		Weight: WeightInformational,
		BgPath: bgDidYouKnow,
		Elements: []frame.DispElement{
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
