package main

import (
	"math/rand/v2"
	"sync/atomic"

	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Cocktail" — typographic recipe card rotating through every drink
// in TheCocktailDB's Cocktail + Shot categories. Each drink is
// fetched, baked, and adb-pushed at `divoom push` time (see
// bakeAllCocktailBackgrounds in scene_baked.go); the device holds
// one bg per index at /userdata/wallclock_bg_cocktail_NNN.jpg, and
// the scene's BgPathFor walks a random index per activation.
//
// Walk strategy: the first activation picks a random starting index;
// each subsequent activation jumps to *another* random index that is
// guaranteed not to equal the previous one (so the wall never shows
// the same drink twice in a row). Drink IDs are sorted alphabetically
// at bake time so a given drink lands at the same indexed path
// across `make push-frame` runs — the per-drink JSON + JPG cache
// stays aligned with the device paths.
func cocktailScene(_ map[string]widget.Widget) *scene.Scene {
	poolSize := cocktailPoolSize()
	if poolSize < 1 {
		poolSize = 1
	}
	// current holds the index of the drink currently on screen (or
	// about to be on screen for the first activation). Stored as
	// int64 so atomic ops work without a mutex. Initial value −1
	// means "no previous pick"; the first BgPathFor call sees that
	// and picks fully randomly with no exclusion.
	var current atomic.Int64
	current.Store(-1)
	return &scene.Scene{
		Name:   "cocktail",
		Weight: 20,
		BgPath: bgCocktailFor(0),
		BgPathFor: func(_ string) string {
			prev := current.Load()
			next := int64(rand.IntN(poolSize))
			// "Different than last" — reroll once if the pool has
			// more than one slot. One reroll is enough: at pool size
			// N the probability of two consecutive collisions is
			// 1/N², negligible past N=20.
			if poolSize > 1 && next == prev {
				next = int64((int(next) + 1 + rand.IntN(poolSize-1)) % poolSize)
			}
			current.Store(next)
			return bgCocktailFor(int(next))
		},
	}
}
