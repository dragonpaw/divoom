package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Jargon" — dedicated scene for the Jargon File source. Shares
// the dictionary layout (source label, big headword, POS,
// definition) with the Devil's Dictionary scene via
// DictionaryScene. No author block (Jargon entries are
// communal) and no tagline.
func jargonScene(widgets map[string]widget.Widget) *scene.Scene {
	return DictionaryScene(DictionarySceneOpts{
		Name: "jargon", Title: "Jargon File", Weight: 20, BgPath: bgJargon,
		Widget: widgets["jargon"],
	})
}
