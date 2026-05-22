package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Cat facts" — promoted out of the whimsy rotator into its own
// scene so the cat silhouette glyph in the bottom-right corner
// gets to be the dominant visual signature. One body Text for
// the fact prose; the "cat fact" header from the widget's
// "cat fact|<body>" output is dropped — the glyph carries the
// label work. Element count 4 (3 top + 1 body) collides only
// with the rare easter scene; Driver.pick()'s same-count rule
// blocks direct easter↔catfacts transitions, which is fine.
func catfactsScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "catfacts",
		Weight: 20,
		BgPath: bgCatFacts,
		Elements: []frame.DispElement{
			sceneTitle("cat fact"),
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 540, Width: 640, Height: 560,
				Align: 2, FontSize: 38, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["catfacts"],
		Mounts: []scene.Mount{
			{ID: idSceneSub1, Format: pipeAt(1), Geometry: vCenterQuoteBody},
		},
	}
}
