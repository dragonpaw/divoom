package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Fortune" — Terminal family: `$ fortune -s` baked into the bg, with a
// baked source/author status bar at the bottom. The widget body renders
// as a left-aligned Text element between them.
func fortuneScene(widgets map[string]widget.Widget) *scene.Scene {
	return QuoteScene(QuoteSceneOpts{
		Name: "fortune", Title: "fortune", Weight: WeightInteresting, BgPath: bgFortune,
		Widget: widgets["fortune"],
		Family: FamilyTerminal,
	})
}
