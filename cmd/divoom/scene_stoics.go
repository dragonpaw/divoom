package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Stoics" — Marcus Aurelius / Seneca / Epictetus aphorisms.
// Green tagline picks up the laurel/marble association without
// stepping on Discworld's orange or B5's purple.
func stoicsScene(widgets map[string]widget.Widget) *scene.Scene {
	return QuoteScene(QuoteSceneOpts{
		Name: "stoics", Title: "Stoics", Weight: WeightEntertaining, BgPath: bgStoics,
		Widget:       widgets["stoics"],
		Family:       FamilyMarginalia,
		Tagline:      "memento mori",
		TaglineColor: cGreen,
		HasAuthor:    true,
	})
}
