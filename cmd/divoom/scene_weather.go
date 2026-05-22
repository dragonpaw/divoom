package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Weather" — current outdoor conditions consolidated with the
// hazard feeds. Widget emits
// "<temp>°<unit>|<outlook>|<hazard>|<aqi>|<humidity>|<rain>"; the
// outlook bucket carries WMO codes, smoke (PM2.5/AQI override),
// or hazard (active NWS alert at the configured point). The
// temperature row is huge proportional digits; the condition row
// is medium prose; the hazard row sits beneath in bright red and
// is blank when no NWS alert is active; the stats row at the very
// bottom packs AQI / humidity / rain-chance as " · "-separated
// fine print (each segment omitted if its source field is blank).
// Both temp and outlook colours flip to red when outlook ==
// "hazard". The bg JPG is picked per outlook via BgPathFor so the
// corner icon matches the current condition. Element count 7 (3
// top + 4 body) collides with the other 7-element scenes; the
// driver's same-count rule blocks direct transitions, which is
// fine.
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
			// Big temperature — proportional Roboto Condensed Light
			// so the "63°" centres on its glyph mass (the smaller °
			// glyph in mono Iosevka pulls the visual centre left of
			// the geometric centre, leaving the condition word below
			// looking misaligned). Colour set by formatter (flips
			// red when outlook == "hazard").
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 530, Width: 640, Height: 240,
				Align: 2, FontSize: 180, FontID: fontProseLight,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Condition word — medium prose, colour set by formatter.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 820, Width: 640, Height: 120,
				Align: 2, FontSize: 70, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Hazard message — bright red, blank unless an NWS alert
			// is active for the configured point.
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 80, StartY: 960, Width: 640, Height: 80,
				Align: 2, FontSize: 40, FontID: fontProseLight,
				FontColor: cRed, BgColor: cBgHard,
			},
			// Stats row — fine print, "AQI 45 · 62% RH · 30% rain".
			// Each segment is dropped when its source field is blank;
			// the whole row is blank when all three are missing.
			{
				ID: idSceneSub3, Type: "Text",
				StartX: 80, StartY: 1050, Width: 640, Height: 60,
				Align: 2, FontSize: 32, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
			},
		},
		Widget: widgets["weather"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: weatherTemp},
			{ID: idSceneSub1, Format: weatherCondition},
			{ID: idSceneSub2, Format: pipeAt(2), AllowEmpty: true},
			{ID: idSceneSub3, Format: weatherStats, AllowEmpty: true},
		},
	}
}
