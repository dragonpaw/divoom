package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "TIL" — top post of the day from r/todayilearned. The monumental
// "T I L" wordmark is baked into the background (see render.drawTILChrome)
// and the body Text continues the grammatical thought ("...that <fact>").
// The widget keeps its 2-segment "TIL|<title>" contract; the scene's
// tilBody formatter strips any leading "TIL that " / "TIL: " / "TIL "
// prefix and defensively ensures the body starts with "that " so the
// visual sentence completes as "TIL · that <fact>".
//
// Element count: 1 body Text + always-on 2 Text + Time = 3 Text + 1 Time.
func tilScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "til",
		Weight: WeightEntertaining,
		BgPath: bgTIL,
		Elements: []frame.DispElement{
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 630, Width: 640, Height: 540,
				Align: 0, FontSize: 34, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["til"],
		Mounts: []scene.Mount{
			{ID: idSceneSub1, Format: tilBody},
		},
	}
}
