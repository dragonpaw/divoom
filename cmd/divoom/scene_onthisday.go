package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "On this day" — historical event for today's calendar date,
// sourced from Wikimedia's free "on this day / events" feed.
// Widget emits "On <Month> <DD>|<year>: <event text>"; the
// header row carries the date label in fg-dark prose and the
// body row carries the event prose, vCentered so short events
// (one-liners) sit visually balanced. Element count 5 (3 top +
// 2 body) collides with sky / weather / aqi / hn; the driver's
// same-count rule blocks direct transitions, which is fine.
func onthisdayScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "onthisday",
		Weight: 20,
		BgPath: bgOnThisDay,
		Elements: []frame.DispElement{
			sceneTitle("on this day"),
			// Date row — "On <Month> <DD>", under the title.
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 540, Width: 640, Height: 60,
				Align: 2, FontSize: 36, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
			},
			// Body — event prose, vCentered.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 620, Width: 640, Height: 620,
				Align: 2, FontSize: 36, FontID: fontProseLight,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["onthisday"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: pipeAt(0)},
			{ID: idSceneSub1, Format: pipeAt(1)},
		},
	}
}
