package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "NASA APOD" — Astronomy Picture of the Day. First scene to use
// the device's Image DispList element type; the widget emits
// "<url>|<title>|<date>" and the Image element's Url is wired in
// at install time via the Mount.Geometry callback (Image elements
// can't be patched via UpdateDisplayItems, but every scene
// activation is a full EnterCustomMode, so this is fine). 6
// elements total (3 always-on + 3 body) — unique among rotation
// scenes.
func nasaScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "nasa",
		Weight: 20,
		BgPath: bgNASA,
		Elements: []frame.DispElement{
			sceneTitle("NASA APOD"),
			// Full-width image — URL set by the Mount.Geometry hook.
			{
				ID: idSceneSub1, Type: "Image",
				StartX: 20, StartY: 560, Width: 760, Height: 540,
				Align: 2,
			},
			// Title underneath the image.
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 80, StartY: 1120, Width: 640, Height: 80,
				Align: 2, FontSize: 36, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["nasa"],
		Mounts: []scene.Mount{
			{
				ID:     idSceneSub1,
				Format: pipeAt(0),
				// Wire the widget's URL output (segment 0) into the
				// Image element's Url field. The element's
				// TextMessage is set by the driver but ignored by
				// the device for Image-type elements.
				Geometry: func(text string, e frame.DispElement) frame.DispElement {
					e.Url = text
					e.ImgLocalFlag = 0
					return e
				},
			},
			{ID: idSceneSub2, Format: pipeAt(1)},
		},
	}
}
