package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Star Trek" — FromSource family: STARDATE / PERSONAL LOG header baked
// into the bg (the stardate ticks each `divoom push`).
func startrekScene(widgets map[string]widget.Widget) *scene.Scene {
	return QuoteScene(QuoteSceneOpts{
		Name: "startrek", Title: "Star Trek", Weight: WeightEntertaining, BgPath: bgStarTrek,
		Widget:       widgets["startrek"],
		Family:       FamilyFromSource,
		Tagline:      "to boldly go where no one has gone before",
		TaglineColor: cYellow,
		HasAuthor:    true,
	})
}
