package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Cat facts" rendered as a field-guide entry for Felis catus. The
// scientific name, taxonomic line, pilcrow drop-marker, footer hairline,
// and observation-number + institution footer are baked into the
// background (see render.DrawCatfactsChrome). The scene itself contributes
// a single body Text for the fact prose — the baked "Felis catus"
// binomial replaces the old sceneTitle("cat fact") header.
//
// Element count: 1 body Text + always-on 2 Text + Time = 3 Text + 1 Time.
// Same-count collisions are blocked by Driver.pick(), which is fine.
func catfactsScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "catfacts",
		Weight: WeightEntertaining,
		BgPath: bgCatFacts,
		Elements: []frame.DispElement{
			{
				// Fact body: left-aligned, sits to the right of the baked
				// pilcrow at x=80. StartX=120 leaves a small left margin
				// for the marker; the track wraps to 4-6 lines of prose.
				ID: idSceneSub1, Type: "Text",
				StartX: 120, StartY: 640, Width: 600, Height: 420,
				Align: 0, FontSize: 40, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["catfacts"],
		Mounts: []scene.Mount{
			{ID: idSceneSub1, Format: pipeAt(1)},
		},
	}
}
