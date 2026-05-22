package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Fortune" — BSD fortune(1) cookies. Most entries are anonymous
// but those with `-- Author` lines carry attribution inline via
// the standard splitAuthor convention; HasAuthor wires the
// author slot so attributed cookies surface their source.
func fortuneScene(widgets map[string]widget.Widget) *scene.Scene {
	return QuoteScene(QuoteSceneOpts{
		Name: "fortune", Title: "fortune", Weight: 20, BgPath: bgFortune,
		Widget:       widgets["fortune"],
		Tagline:      "a fortune cookie awaits",
		TaglineColor: cAqua,
		HasAuthor:    true,
	})
}
