package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Weather" — console-strip layout. The widget emits
// "<temp>°<unit>|<outlook>|<hazard>|<aqi>|<humidity>|<rain>". The
// scene has two dynamic Text elements (the device caps Text at 6;
// always-on already uses 3, leaving 3 for the scene including the
// baked title — we use 2):
//
//   - Big temperature (huge, colour by reading via weatherTempColor).
//     The outlook is communicated visually by the bg's corner glyph
//     so we don't burn an element on the word.
//   - Bottom strip: when an NWS alert is firing, "⚠ <hazard>" in red;
//     otherwise "<COND> · AIR <aqi> · HUM <hum>% · RAIN <rain>%".
//
// The "weather" title row (cFgDark 26pt Roboto Condensed Light) is
// baked into the bg by render.drawWeatherChrome — moving it out of
// the device-side elements bought us the cap headroom needed for the
// always-on weekend split.
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
				StartX: 80, StartY: 560, Width: 640, Height: 320,
				Align: 2, FontSize: 240, FontID: fontProseLight,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Combined bottom strip — outlook word + three stats, or
			// the hazard headline when an NWS alert is active. Sits in
			// the y=1000-1080 band between the two hairlines baked by
			// drawWeatherChrome.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 1000, Width: 640, Height: 80,
				Align: 2, FontSize: 32, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["weather"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: weatherTemp},
			{ID: idSceneSub1, Format: weatherStrip},
		},
	}
}
