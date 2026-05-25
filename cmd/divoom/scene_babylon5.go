package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Babylon 5" — FromSource family: EarthForce transmission header is
// baked into the bg JPG (see quote_family.go); the device renders the
// quote body + attribution as plain Text elements.
func babylon5Scene(widgets map[string]widget.Widget) *scene.Scene {
	return QuoteScene(QuoteSceneOpts{
		Name: "babylon5", Title: "Babylon 5", Weight: WeightEntertaining, BgPath: bgBabylon5,
		Widget:       widgets["babylon5"],
		Family:       FamilyFromSource,
		Tagline:      "the last best hope for peace",
		TaglineColor: cPurple,
		HasAuthor:    true,
	})
}
