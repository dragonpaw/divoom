package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Sunrise" — today's sunrise / sunset / daylight hours. Three
// big mono rows under a small "Today" label; sunrise in yellow
// (morning) and sunset in orange (evening) so the colour pair
// carries the meaning without needing inline captions. 7
// elements total (3 always-on + 4 body); collides only with the
// other 7-element scenes (dayofyear, B5, ST, Discworld, jargon),
// and Driver.pick()'s same-count rule blocks direct transitions
// between them.
func sunriseScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "sunrise",
		Weight: 20,
		BgPath: bgSunrise,
		Elements: []frame.DispElement{
			sceneTitle("today"),
			// "sunrise" legend.
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 555, Width: 640, Height: 40,
				Align: 2, FontSize: 30, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
				TextMessage: "sunrise",
			},
			// Sunrise time — big, yellow.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 600, Width: 640, Height: 120,
				Align: 2, FontSize: 84, FontID: fontMono,
				FontColor: cYellow, BgColor: cBgHard,
			},
			// "sunset" legend.
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 80, StartY: 740, Width: 640, Height: 40,
				Align: 2, FontSize: 30, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
				TextMessage: "sunset",
			},
			// Sunset time — big, orange.
			{
				ID: idSceneSub3, Type: "Text",
				StartX: 80, StartY: 785, Width: 640, Height: 120,
				Align: 2, FontSize: 84, FontID: fontMono,
				FontColor: cOrange, BgColor: cBgHard,
			},
			// Daylight duration — medium, fg.
			{
				ID: idSceneSub4, Type: "Text",
				StartX: 80, StartY: 940, Width: 640, Height: 100,
				Align: 2, FontSize: 50, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["sunrise"],
		Mounts: []scene.Mount{
			{ID: idSceneSub1, Format: pipeAt(0)},
			{ID: idSceneSub3, Format: pipeAt(1)},
			{ID: idSceneSub4, Format: pipeAt(2)},
		},
	}
}
