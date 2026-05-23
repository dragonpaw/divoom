package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Cocktail" — typographic recipe card rotating through every drink
// in TheCocktailDB's Cocktail + Shot categories. Each drink is
// fetched, baked, and adb-pushed at `divoom push` time (see
// bakeAllCocktailBackgrounds in scene_baked.go); the device holds
// one bg per index at /userdata/wallclock_bg_cocktail_NNN.jpg.
//
// Index mapping is stable across runs (drink IDs sorted at bake
// time → the same drink always lives at the same indexed path → the
// per-drink disk cache stays valid). The visible order is randomised
// per daemon start by newIndexWalker, which yields each index
// exactly once in a fresh random order before reshuffling.
func cocktailScene(_ map[string]widget.Widget) *scene.Scene {
	walk := newIndexWalker(cocktailPoolSize())
	return &scene.Scene{
		Name:   "cocktail",
		Weight: WeightInteresting,
		BgPath: bgCocktailFor(0),
		BgPathFor: func(_ string) string {
			return bgCocktailFor(walk())
		},
	}
}
