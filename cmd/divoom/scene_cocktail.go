package main

import (
	"math/rand/v2"

	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Cocktail" — typographic recipe card rotating through every drink
// in TheCocktailDB's Cocktail + Shot categories (~300 drinks). Each
// drink is fetched, baked, and adb-pushed at `divoom push` time (see
// bakeAllCocktailBackgrounds in scene_baked.go); the device holds
// one bg per index at /userdata/wallclock_bg_cocktail_NNN.jpg, and
// the scene's BgPathFor picks a random index per activation.
//
// Pool size is determined from the local on-disk cache count — the
// number of cocktail JSON files in ~/.cache/divoom/cocktail/. That's
// captured once at scene-config time so each activation just does an
// O(1) random pick. If the cache is empty (first run before push),
// the scene falls back to index 0.
func cocktailScene(_ map[string]widget.Widget) *scene.Scene {
	poolSize := cocktailPoolSize()
	if poolSize < 1 {
		poolSize = 1
	}
	return &scene.Scene{
		Name:   "cocktail",
		Weight: 20,
		BgPath: bgCocktailFor(0),
		BgPathFor: func(_ string) string {
			return bgCocktailFor(rand.IntN(poolSize))
		},
	}
}
