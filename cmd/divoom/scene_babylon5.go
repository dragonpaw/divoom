package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Babylon 5" — dedicated scene for the B5 quote source. See
// QuoteScene for the shared promoted-quote layout (source label,
// body, author, tagline).
func babylon5Scene(widgets map[string]widget.Widget) *scene.Scene {
	return QuoteScene(QuoteSceneOpts{
		Name: "babylon5", Title: "Babylon 5", Weight: 20, BgPath: bgBabylon5,
		Widget:       widgets["babylon5"],
		Tagline:      "the last best hope for peace",
		TaglineColor: cPurple,
		HasAuthor:    true,
	})
}
