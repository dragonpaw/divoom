package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Markets" — QQQ stack: symbol on top, then a (percent, label) pair
// for week and again for month. Percents take green/red by sign;
// labels stay in fg-dark so they read as captions.
func marketsScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "markets",
		Weight: 20,
		BgPath: bgMarkets,
		Elements: []frame.DispElement{
			sceneTitle("markets"),
			// QQQ symbol header
			{
				ID: idSceneMain, Type: "Text",
				StartX: 30, StartY: 510, Width: 740, Height: 130,
				Align: 2, FontSize: 110, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Week percent (color set by formatter)
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 30, StartY: 680, Width: 740, Height: 110,
				Align: 2, FontSize: 84, FontID: fontMono,
				FontColor: cFgDark, BgColor: cBgHard,
			},
			// "week" label
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 30, StartY: 800, Width: 740, Height: 50,
				Align: 2, FontSize: 30, FontID: fontProse,
				FontColor: cFgDark, BgColor: cBgHard,
				TextMessage: "week",
			},
			// Month percent (color set by formatter)
			{
				ID: idSceneSub3, Type: "Text",
				StartX: 30, StartY: 890, Width: 740, Height: 110,
				Align: 2, FontSize: 84, FontID: fontMono,
				FontColor: cFgDark, BgColor: cBgHard,
			},
			// "month" label
			{
				ID: idSceneSub4, Type: "Text",
				StartX: 30, StartY: 1010, Width: 740, Height: 50,
				Align: 2, FontSize: 30, FontID: fontProse,
				FontColor: cFgDark, BgColor: cBgHard,
				TextMessage: "month",
			},
		},
		Widget: widgets["markets"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: qqqSymbol},
			{ID: idSceneSub1, Format: qqqWeekPct},
			{ID: idSceneSub3, Format: qqqMonthPct},
		},
	}
}
