package main

import (
	"strings"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "agenda" — next-up peek at the user's public ICS calendar feed. The
// widget emits up to six pipe-separated fields:
//
//	NEXT_SUMMARY|NEXT_RELATIVE|NEXT_TIME|UPCOMING_SUMMARY|UPCOMING_RELATIVE|UPCOMING_TIME
//
// The big body Text is the next event's summary; a smaller row under it
// combines its relative offset and clock ("in 23m · 10:30"); a third
// dim row near the bottom shows the event-after-that ("Lunch · in 4h").
// When only one event is upcoming the upcoming-after-that row collapses
// to empty.
//
// Element count: baked title (0) + summary (1) + when (1) + after (1)
// = 3 scene Text + 2 always-on = 5 Text + 1 Time. Within the cap.
func agendaScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "agenda",
		Weight: WeightInformational,
		BgPath: bgAgenda,
		Elements: []frame.DispElement{
			// Big "next summary" — prose, multi-line tolerant.
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 560, Width: 640, Height: 320,
				Align: 0, FontSize: 48, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
			// "in 23m · 10:30" combo row.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 920, Width: 640, Height: 50,
				Align: 0, FontSize: 32, FontID: fontMono,
				FontColor: cAqua, BgColor: cBgHard,
			},
			// Then-after-that, dim.
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 80, StartY: 1100, Width: 640, Height: 50,
				Align: 0, FontSize: 28, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
			},
		},
		Widget: widgets["agenda"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: pipeAt(0)},
			{ID: idSceneSub1, Format: agendaWhen},
			{ID: idSceneSub2, Format: agendaUpcoming, AllowEmpty: true},
		},
	}
}

// agendaWhen joins the next event's relative offset (pipe[1]) and clock
// time (pipe[2]) as "in 23m · 10:30". If either segment is empty, the
// remaining one stands alone.
func agendaWhen(raw string) (text, color string) {
	rel := pipeAtRaw(raw, 1)
	clock := pipeAtRaw(raw, 2)
	switch {
	case rel == "" && clock == "":
		return "", ""
	case rel == "":
		return clock, ""
	case clock == "":
		return rel, ""
	}
	return rel + " · " + clock, ""
}

// agendaUpcoming renders the second-event line as "<summary> · <rel>"
// (no clock — saves a row of text on the dim-secondary line). Empty
// when the widget didn't supply a second event.
func agendaUpcoming(raw string) (text, color string) {
	sum := pipeAtRaw(raw, 3)
	rel := pipeAtRaw(raw, 4)
	if strings.TrimSpace(sum) == "" {
		return "", ""
	}
	if rel == "" {
		return sum, ""
	}
	return sum + " · " + rel, ""
}
