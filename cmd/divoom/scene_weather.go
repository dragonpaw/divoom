package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Weather" — console-strip layout. The widget emits
// "<temp>°<unit>|<outlook>|<hazard>|<aqi>|<humidity>|<rain>". Six
// device elements, the maximum we can fit on one scene:
//
//   - title row ("weather", small dim)
//   - big temperature (huge, colour by reading via weatherTempColor)
//   - condition-or-hazard slot (medium prose; shows the active NWS
//     alert in red when one is firing, otherwise the outlook word in
//     its outlook colour)
//   - three small stat values in a row: AQI, humidity %, rain-chance %
//
// The three stat values sit in an "AIR | HUMIDITY | RAIN" console
// strip whose dividers and column labels are baked into the bg JPG
// by render.drawWeatherChrome (we're already at the 6-element cap, so
// the labels can't be device Text elements). Blank stat fields render
// as a dim "—" rather than "0" so a failed source lookup doesn't lie.
//
// AQI colour bands follow the US EPA scale:
//   0-50 green · 51-100 yellow · 101-150 orange ·
//   151-200 red · 201-300 purple · 301+ red.
//
// The bg JPG is picked per outlook via BgPathFor so the corner glyph
// matches the current condition.
func weatherScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "weather",
		Weight: 20,
		BgPath: bgWeatherCloudy, // fallback before first cache fill
		BgPathFor: func(raw string) string {
			return weatherBgFor(weatherOutlookFrom(raw))
		},
		Elements: []frame.DispElement{
			sceneTitle("weather"),
			// Big temperature — proportional Roboto Condensed Light so
			// "63°" centres on its glyph mass. Colour set by formatter
			// (flips red when outlook == "hazard").
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 540, Width: 640, Height: 240,
				Align: 2, FontSize: 200, FontID: fontProseLight,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Condition-or-hazard slot — medium prose. Hazard wins when
			// pipe[2] is non-empty (NWS alert active); otherwise the
			// outlook word in its outlook colour.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 790, Width: 640, Height: 80,
				Align: 2, FontSize: 70, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
			// AQI value — column 1 of the console strip (label "AIR"
			// baked into bg). Colour by EPA band.
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 80, StartY: 1025, Width: 213, Height: 60,
				Align: 2, FontSize: 56, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Humidity value — column 2 ("HUMIDITY"), always blue when
			// present.
			{
				ID: idSceneSub3, Type: "Text",
				StartX: 293, StartY: 1025, Width: 213, Height: 60,
				Align: 2, FontSize: 56, FontID: fontMono,
				FontColor: cBlue, BgColor: cBgHard,
			},
			// Rain-chance value — column 3 ("RAIN"), always aqua when
			// present.
			{
				ID: idSceneSub4, Type: "Text",
				StartX: 506, StartY: 1025, Width: 214, Height: 60,
				Align: 2, FontSize: 56, FontID: fontMono,
				FontColor: cAqua, BgColor: cBgHard,
			},
		},
		Widget: widgets["weather"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: weatherTemp},
			{ID: idSceneSub1, Format: weatherConditionOrHazard},
			{ID: idSceneSub2, Format: weatherAQI, AllowEmpty: true},
			{ID: idSceneSub3, Format: weatherHumidity, AllowEmpty: true},
			{ID: idSceneSub4, Format: weatherRain, AllowEmpty: true},
		},
	}
}
