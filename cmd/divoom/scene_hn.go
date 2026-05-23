package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "HN" — HackerNews-front-page-entry-at-scale: the title is the hero,
// the source domain sits just below it as a small dim caption, an
// article summary fills the body, and a mono metadata footer (score,
// author, age, comments) anchors the bottom above the baked footer
// rule. The "HACKER NEWS" wordmark + orange separator are baked into
// the background (drawHNChrome), so this scene drops the standard
// sceneTitle row. The HN-flavoured "Y" glyph stays in the bottom-right
// corner of the bg as the wordmark's mirror.
//
// Widget output: 8 pipe segments —
//   0:"Hacker News" 1:title 2:domain 3:summary 4:score 5:author 6:age 7:comments
func hnScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "hn",
		Weight: WeightInteresting,
		BgPath: bgHN,
		Elements: []frame.DispElement{
			// Story title — left-aligned hero. Mounts to pipeAt(1).
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 580, Width: 640, Height: 240,
				Align: 0, FontSize: 46, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Source domain — small dim caption under the title.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 830, Width: 640, Height: 40,
				Align: 0, FontSize: 28, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
			},
			// Article summary — body prose.
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 80, StartY: 900, Width: 640, Height: 200,
				Align: 0, FontSize: 30, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
			},
			// Metadata footer — mono, composed from segments 4-7.
			{
				ID: idSceneSub3, Type: "Text",
				StartX: 80, StartY: 1160, Width: 640, Height: 60,
				Align: 0, FontSize: 28, FontID: fontMono,
				FontColor: cFgDark, BgColor: cBgHard,
			},
		},
		Widget: widgets["hn"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: pipeAt(1)},
			{ID: idSceneSub1, Format: pipeAt(2), AllowEmpty: true},
			{ID: idSceneSub2, Format: pipeAt(3), AllowEmpty: true},
			{ID: idSceneSub3, Format: hnFooter, AllowEmpty: true},
		},
	}
}
