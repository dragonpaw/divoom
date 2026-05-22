package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "ZenQuotes" — sky-blue, contemplative.
func zenquotesScene(widgets map[string]widget.Widget) *scene.Scene {
	return QuoteScene(QuoteSceneOpts{
		Name: "zenquotes", Title: "zen", Weight: 20, BgPath: bgZenQuotes,
		Widget:       widgets["zenquotes"],
		Tagline:      "be here now",
		TaglineColor: cBlue,
		HasAuthor:    true,
	})
}
