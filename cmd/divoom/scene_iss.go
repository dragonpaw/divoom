package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "ISS" — current sub-satellite point (lat/lon) of the
// International Space Station, plus the wall-clock time until
// its next visible pass over our location (when available) and
// a coarse "over <region>" hint. Widget emits
// "<lat>°, <lon>°|<next-pass>|over <region>"; the pass and
// region segments are AllowEmpty because the next-pass API has
// historically been flaky and the region lookup is a coarse
// continent-vs-ocean band table that may return an empty hint.
// 10% margins (StartX 80, Width 640) match the quote scenes. 6
// elements total (3 top + 3 body) collides with nasa / cocktail
// — Driver.pick()'s same-count rule blocks direct transitions
// between them, which is fine.
func issScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "iss",
		Weight: 20,
		BgPath: bgISS,
		Elements: []frame.DispElement{
			sceneTitle("ISS overhead"),
			// Big lat/lon — mono, fg.
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 520, Width: 640, Height: 140,
				Align: 2, FontSize: 80, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Next-pass row — medium prose, yellow (event-imminent
			// signal colour).
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 680, Width: 640, Height: 100,
				Align: 2, FontSize: 50, FontID: fontProseLight,
				FontColor: cYellow, BgColor: cBgHard,
			},
			// Region hint — small, dim caption.
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 80, StartY: 800, Width: 640, Height: 80,
				Align: 2, FontSize: 32, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
			},
		},
		Widget: widgets["iss"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: pipeAt(0)},
			{ID: idSceneSub1, Format: pipeAt(1), AllowEmpty: true},
			{ID: idSceneSub2, Format: pipeAt(2), AllowEmpty: true},
		},
	}
}
