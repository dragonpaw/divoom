package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Moonphase" — moon phase name, illumination, and next-full-moon
// countdown on separate rows. Colored gruvbox blue (ambient/sky).
// 6 elements total (3 top + 3 body) collides with
// nasa/cocktail/iss/weather — the driver's same-count exclusion
// rule blocks direct transitions between them, which is fine.
func moonphaseScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "moonphase",
		Weight: 20,
		BgPath: bgMoonphase,
		Elements: []frame.DispElement{
			sceneTitle("moon"),
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 560, Width: 640, Height: 130,
				Align: 2, FontSize: 80, FontID: fontProse,
				FontColor: cBlue, BgColor: cBgHard,
			},
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 30, StartY: 730, Width: 740, Height: 110,
				Align: 2, FontSize: 72, FontID: fontMono,
				FontColor: cFgDark, BgColor: cBgHard,
			},
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 30, StartY: 890, Width: 740, Height: 90,
				Align: 2, FontSize: 40, FontID: fontProse,
				FontColor: cFgDark, BgColor: cBgHard,
			},
		},
		Widget: widgets["moonphase"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: moonPhaseName},
			{ID: idSceneSub1, Format: moonIllum},
			{ID: idSceneSub2, Format: moonNextFullMoon},
		},
	}
}
