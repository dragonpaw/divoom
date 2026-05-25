package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Mark Twain" — public-domain American aphorisms. fg accent
// keeps the typography newsprint-plain.
func twainScene(widgets map[string]widget.Widget) *scene.Scene {
	return QuoteScene(QuoteSceneOpts{
		Name: "twain", Title: "Mark Twain", Weight: WeightEntertaining, BgPath: bgTwain,
		Widget:       widgets["twain"],
		Family:       FamilyMarginalia,
		Tagline:      "S.L. CLEMENS (1835—1910)",
		TaglineColor: cFgDark,
		HasAuthor:    false,
	})
}
