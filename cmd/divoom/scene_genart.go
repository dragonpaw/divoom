package main

import (
	"time"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/render"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "genart" — daily generative-art piece. The whole scene IS the
// background; only a single small caption Text element identifies the
// algorithm and date. The bg is re-rendered at every local midnight
// (see daily_refresh.go) so the art changes day-to-day; within a day,
// re-renders produce the identical image (date-hashed seed).
//
// Element count: 1 caption + always-on (2 Text + 1 Time) = 3 Text + 1
// Time, well within the cap.
func genartScene(_ map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "genart",
		Weight: WeightEntertaining,
		BgPath: bgGenart,
		Elements: []frame.DispElement{
			// Bottom-right caption "<algo>  YYYY-MM-DD" — quiet credit
			// so the art reads as the focal point.
			{
				ID: idSceneSub3, Type: "Text",
				StartX: 80, StartY: 1180, Width: 640, Height: 36,
				Align: 1, FontSize: 24, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
			},
		},
		Mounts: nil,
		OnActivate: func(now time.Time, _ string, elements []frame.DispElement) {
			_, name := render.GenartForDate(now)
			caption := name + "  " + now.Format("2006-01-02")
			for i := range elements {
				if elements[i].ID == idSceneSub3 {
					elements[i].TextMessage = caption
					return
				}
			}
		},
	}
}
