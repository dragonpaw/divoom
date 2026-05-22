package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "DayOfYear" — pretty year-progress dial. The widget emits
// "39%|Year 2026|Day 142 of 366"; the bg has a thick orange
// progress bar baked in at y=940-1000. Four body elements (7
// total with the always-on top) keep the scene count unique
// so the cache-busting same-count-exclusion rule lets us
// transition cleanly into it.
func dayofyearScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "dayofyear",
		Weight: 20,
		BgPath: bgDayOfYear,
		Elements: []frame.DispElement{
			sceneTitle("year progress"),
			// Big percentage
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 540, Width: 640, Height: 200,
				Align: 2, FontSize: 180, FontID: fontMono,
				FontColor: cOrange, BgColor: cBgHard,
			},
			// "Year 2026" — below the progress bar at y=755-815
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 850, Width: 640, Height: 70,
				Align: 2, FontSize: 56, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
			// "Day 142 of 366"
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 80, StartY: 940, Width: 640, Height: 60,
				Align: 2, FontSize: 40, FontID: fontProse,
				FontColor: cFgDark, BgColor: cBgHard,
			},
			// "year progress" caption under the body block
			{
				ID: idSceneSub3, Type: "Text",
				StartX: 80, StartY: 1080, Width: 640, Height: 50,
				Align: 2, FontSize: 28, FontID: fontProse,
				FontColor: cFgDark, BgColor: cBgHard,
				TextMessage: "year progress",
			},
		},
		Widget: widgets["dayofyear"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: pipeAt(0)},
			{ID: idSceneSub1, Format: pipeAt(1)},
			{ID: idSceneSub2, Format: pipeAt(2)},
		},
	}
}
