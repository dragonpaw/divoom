package main

import (
	"sync"

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
// by newIndexWalker, which yields each index exactly once in a fresh
// random order before reshuffling.
//
// The pool is re-counted on every activation rather than once at
// startup: the serve container comes up with an empty cache (pool 0
// → index 000 forever, which is Mojito), and a later `divoom push`
// fills it. A changed count starts a fresh walk over the new pool.
func cocktailScene(_ map[string]widget.Widget) *scene.Scene {
	var mu sync.Mutex
	poolSize := -1
	var walk func() int
	return &scene.Scene{
		Name:   "cocktail",
		Weight: WeightEntertaining,
		BgPath: bgCocktailFor(0),
		BgPathFor: func(_ string) string {
			n := cocktailPoolSize()
			mu.Lock()
			defer mu.Unlock()
			if n != poolSize {
				poolSize = n
				walk = newIndexWalker(n)
			}
			return bgCocktailFor(walk())
		},
		WeightModifier: cocktailModifier,
	}
}
