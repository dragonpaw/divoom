package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "seismic" — most-notable recent earthquake within 500km of the
// configured location, sourced from the USGS 2.5+ "last day" feed.
// The scene reads as: big magnitude hero (band-coloured), one stats
// line (count · distance · bearing · age), one band-keyed commentary
// line. The "seismic activity" title is baked into the bg.
//
// Widget pipe shape: "<mag>|<count>|<dist_km>|<bearing>|<age>", with
// the no-event sentinel "0.0|0|||" which the formatters collapse to
// an em-dash hero + "no events in 24h" stats + a band="none" caption.
//
// Element count: 3 scene Text + 2 always-on Text + 1 Time + 1 Week.
// Five Text elements total — one slot in reserve under the 6-Text cap.
func seismicScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "seismic",
		Weight: 20,
		BgPath: bgSeismic,
		Elements: []frame.DispElement{
			// Hero magnitude — huge mono number, FontColor set by the
			// band-keyed formatter so the colour and the value stay in
			// lockstep.
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 620, Width: 640, Height: 220,
				Align: 2, FontSize: 200, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Stats row — mono, dim. "<n> events · <km>km <bearing> · <age>".
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 880, Width: 640, Height: 50,
				Align: 2, FontSize: 32, FontID: fontMono,
				FontColor: cFgDark, BgColor: cBgHard,
			},
			// Commentary — prose-light, dim. Band-keyed caption.
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 80, StartY: 1080, Width: 640, Height: 60,
				Align: 2, FontSize: 26, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
			},
		},
		Widget: widgets["seismic"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: seismicMagnitude},
			{ID: idSceneSub1, Format: seismicStats, AllowEmpty: true},
			{ID: idSceneSub2, Format: seismicCommentary, AllowEmpty: true},
		},
	}
}
