package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Devil's Dictionary" — Ambrose Bierce, 1906. Dictionary-shaped
// like the Jargon scene (headword + POS + definition), with an
// author block (Bierce baked in) and the period tagline below.
func devilScene(widgets map[string]widget.Widget) *scene.Scene {
	return DictionaryScene(DictionarySceneOpts{
		Name: "devil", Title: "Devil's Dictionary", Weight: 20, BgPath: bgDevil,
		Widget:    widgets["devil"],
		HasAuthor: true,
		Tagline:   "Cynic's Word Book, 1906",
	})
}
