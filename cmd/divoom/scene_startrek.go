package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Star Trek" — Starfleet command gold tagline.
func startrekScene(widgets map[string]widget.Widget) *scene.Scene {
	return QuoteScene(QuoteSceneOpts{
		Name: "startrek", Title: "Star Trek", Weight: 20, BgPath: bgStarTrek,
		Widget:       widgets["startrek"],
		Tagline:      "to boldly go where no one has gone before",
		TaglineColor: cYellow,
		HasAuthor:    true,
	})
}
