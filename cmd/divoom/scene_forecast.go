package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "forecast" — next-4-day weather strip. One row per day: short day
// name, high / low temps, outlook word, in mono so the temps align
// across rows. Tomorrow is at y=560; each row is 130px tall.
//
// The current-weather scene handles today; this scene picks up at
// tomorrow and extends the horizon out to four days. Source: a
// Forecast widget that hits Open-Meteo's daily endpoint (see
// internal/widget/weather/forecast.go).
//
// Element count: 4 scene Text rows + 2 always-on Text = 6 Text. At
// cap. Title baked into bg by render.drawForecastChrome.
func forecastScene(widgets map[string]widget.Widget) *scene.Scene {
	row := func(id, y int) frame.DispElement {
		return frame.DispElement{
			ID: id, Type: "Text",
			StartX: 80, StartY: y, Width: 640, Height: 110,
			Align: 0, FontSize: 56, FontID: fontMono,
			FontColor: cFg, BgColor: cBgHard,
		}
	}
	return &scene.Scene{
		Name:   "forecast",
		Weight: 20,
		BgPath: bgForecast,
		Elements: []frame.DispElement{
			row(idSceneMain, 560),
			row(idSceneSub1, 700),
			row(idSceneSub2, 840),
			row(idSceneSub3, 980),
		},
		Widget: widgets["forecast"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: forecastRow(0)},
			{ID: idSceneSub1, Format: forecastRow(1)},
			{ID: idSceneSub2, Format: forecastRow(2)},
			{ID: idSceneSub3, Format: forecastRow(3)},
		},
	}
}
