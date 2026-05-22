package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Easter" — rare (~0.5%) treat. Just the punchline body of an
// easter-egg one-liner on top of a giant gruvbox-yellow egg
// shape baked into the bg. Element count 4 (3 top + 1 body)
// is unique so it's a valid pick after any other scene.
func easterScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "easter",
		Weight: 1,
		BgPath: bgEaster,
		Elements: []frame.DispElement{
			sceneTitle("easter egg"),
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 540, Width: 640, Height: 110,
				Align: 2, FontSize: 36, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["easter"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: pipeAt(1)},
		},
	}
}
