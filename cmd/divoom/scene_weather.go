package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "weather" — three-row terminal-style readout. The widget emits
// "<temp>°<unit>|<outlook>|<hazard>|<aqi>|<humidity>|<rain>". With
// the always-on header freed up to 2 Text + Time + Week, the scene
// has 4 Text slots; we use three:
//
//   - Big temperature, colour-banded by reading.
//   - Condition / hazard row: outlook word in its outlook colour, or
//     "⚠ <hazard>" in red when an NWS alert is firing.
//   - Bottom strip: "AIR <aqi> · HUM <hum>% · RAIN <rain>%" coloured
//     by AQI band (cleared to "—" when a source fetch failed).
//
// The "weather" title row is baked into the bg by drawWeatherChrome;
// the bottom strip is vertically centred between the two hairlines
// the chrome paints at y=985 and y=1095.
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
			// Big temperature — proportional Roboto Condensed Light so
			// "63°" centres on its glyph mass. Colour set by formatter
			// (flips red when outlook == "hazard").
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 560, Width: 640, Height: 260,
				Align: 2, FontSize: 220, FontID: fontProseLight,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Condition / hazard row — outlook word in its outlook
			// colour, or the NWS alert headline when one's firing.
			// Sits in the band between the temperature and the bottom
			// strip's top hairline.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 870, Width: 640, Height: 80,
				Align: 2, FontSize: 56, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Bottom AIR/HUM/RAIN strip — vertically centred in the
			// y=985-1095 band drawn by drawWeatherChrome. The 32pt
			// mono baseline at ≈y=1052 puts the visible text height
			// (~32px) centred around the band's middle (y=1040).
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 80, StartY: 1015, Width: 640, Height: 60,
				Align: 2, FontSize: 32, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["weather"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: weatherTemp},
			{ID: idSceneSub1, Format: weatherConditionOrHazard},
			{ID: idSceneSub2, Format: weatherStats},
		},
	}
}
