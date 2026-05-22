package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Cocktail" — random drink from TheCocktailDB. Same shape as NASA
// APOD: a full-width Image element gets its Url wired in at install
// time via Mount.Geometry (Image elements can't be patched live, but
// every scene activation is a full EnterCustomMode, so a fresh URL
// lands on every show). Name goes in big prose underneath the
// photo; ingredient list as small fg-dark caption. The bottom-right
// glass glyph carries the "this is a cocktail" labelling work. 6
// elements total (3 always-on + 3 body) — matches NASA, and
// Driver.pick()'s same-count rule blocks direct nasa↔cocktail
// transitions, which is fine.
func cocktailScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "cocktail",
		Weight: 20,
		BgPath: bgCocktail,
		Elements: []frame.DispElement{
			sceneTitle("cocktail"),
			// Full-width image — URL set by the Mount.Geometry hook.
			{
				ID: idSceneMain, Type: "Image",
				StartX: 20, StartY: 540, Width: 760, Height: 480,
				Align: 2,
			},
			// Drink name — big prose.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 1040, Width: 640, Height: 100,
				Align: 2, FontSize: 60, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Ingredient list — small, dim.
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 80, StartY: 1160, Width: 640, Height: 70,
				Align: 2, FontSize: 28, FontID: fontProse,
				FontColor: cFgDark, BgColor: cBgHard,
			},
		},
		Widget: widgets["cocktail"],
		Mounts: []scene.Mount{
			{
				ID:     idSceneMain,
				Format: pipeAt(0),
				// Wire the widget's URL output (segment 0) into the
				// Image element's Url field. TextMessage gets set by
				// the driver too but the device ignores it for Image
				// elements.
				Geometry: func(text string, e frame.DispElement) frame.DispElement {
					e.Url = text
					e.ImgLocalFlag = 0
					return e
				},
			},
			{ID: idSceneSub1, Format: pipeAt(1)},
			{ID: idSceneSub2, Format: pipeAt(2)},
		},
	}
}
