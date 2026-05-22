package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "HN" — promoted out of the now-empty whimsy rotator into its own
// scene. Widget emits "Hacker News|<title> — <summary>"; the small
// dim header sits above the body, which carries the headline and
// summary together. The HN-flavoured "Y" glyph in the bottom-right
// corner labels the scene. 5 elements total (3 top + 2 body) —
// matches weather/aqi; the driver's same-count rule blocks
// direct transitions between them, which is fine.
func hnScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "hn",
		Weight: 20,
		BgPath: bgHN,
		Elements: []frame.DispElement{
			sceneTitle("Hacker News"),
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 540, Width: 640, Height: 580,
				Align: 2, FontSize: 34, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["hn"],
		Mounts: []scene.Mount{
			{ID: idSceneSub1, Format: pipeAt(1), Geometry: vCenterQuoteBody},
		},
	}
}
