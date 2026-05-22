package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Mark Twain" — public-domain American aphorisms. fg accent
// keeps the typography newsprint-plain.
func twainScene(widgets map[string]widget.Widget) *scene.Scene {
	return QuoteScene(QuoteSceneOpts{
		Name: "twain", Title: "Mark Twain", Weight: 20, BgPath: bgTwain,
		Widget:       widgets["twain"],
		Tagline:      "Samuel L. Clemens",
		TaglineColor: cFg,
		HasAuthor:    false,
	})
}
