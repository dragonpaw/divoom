package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Discworld" — GNU Terry Pratchett.
func discworldScene(widgets map[string]widget.Widget) *scene.Scene {
	return QuoteScene(QuoteSceneOpts{
		Name: "discworld", Title: "Discworld", Weight: 20, BgPath: bgDiscworld,
		Widget:       widgets["discworld"],
		Family:       FamilyFromSource,
		Tagline:      "GNU Terry Pratchett",
		TaglineColor: cOrange,
		HasAuthor:    true,
	})
}
