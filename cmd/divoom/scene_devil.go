package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Devil's Dictionary" — Terminal family: `$ define` baked into the bg,
// with the source line + "author: Ambrose Bierce" baked into the bottom
// status bar (see quote_family.go).
func devilScene(widgets map[string]widget.Widget) *scene.Scene {
	return DictionaryScene(DictionarySceneOpts{
		Name: "devil", Title: "Devil's Dictionary", Weight: 20, BgPath: bgDevil,
		Widget: widgets["devil"],
	})
}
