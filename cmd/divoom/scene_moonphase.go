package main

import (
	"strings"
	"time"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Moonphase" — the bg JPG carries the actual moon disc painted for the
// current point in the 29.53-day synodic cycle (one of 14 pre-rendered
// variants, selected by BgPathFor from the widget's phase + illumination
// reading). The disc IS the glyph; the corner crescent that used to live
// here was a known-stale lie and is gone. Body text below the disc names
// the phase + its illumination, plus the next-full-moon countdown.
//
// Element count: sceneTitle + phase-and-illum + countdown = 3 Text,
// plus the 2 always-on Text + 1 Time = 5 Text + 1 Time. Within cap.
func moonphaseScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:      "moonphase",
		Weight:    20,
		BgPath:    moonBackgrounds[7], // fallback before first cache fill
		BgPathFor: moonBgPathFor,
		Elements: []frame.DispElement{
			// "First Quarter · 53%" — caption directly under the disc.
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 970, Width: 640, Height: 60,
				Align: 2, FontSize: 44, FontID: fontProse,
				FontColor: cBlue, BgColor: cBgHard,
			},
			// "full moon in N days" / "next full moon: Jun 1".
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 1050, Width: 640, Height: 50,
				Align: 2, FontSize: 32, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
			},
		},
		Widget: widgets["moonphase"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: moonPhaseAndIllum},
			{ID: idSceneSub1, Format: moonNextFullMoon},
		},
		// On full-moon nights, repurpose the countdown row as a dry
		// werewolf advisory: switch to mono (faux-field-note voice)
		// and bump the size a notch so the advisory reads with weight.
		// Other phases use the inherited prose-light + smaller size.
		OnActivate: moonOnActivate,
	}
}

// moonOnActivate restyles the countdown row when the widget reports
// a full moon. The Mount formatter already swaps the text + colour;
// this just changes font + size so the advisory looks like a posted
// note rather than the usual dim countdown.
func moonOnActivate(_ time.Time, raw string, elements []frame.DispElement) {
	parts := strings.Split(raw, " · ")
	if len(parts) < 2 || parts[1] != "full" {
		return
	}
	for i, e := range elements {
		if e.ID == idSceneSub1 {
			elements[i].FontSize = 38
			elements[i].FontID = fontMono
			elements[i].StartY = 1040
			return
		}
	}
}
