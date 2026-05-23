package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Cocktail" — random drink from TheCocktailDB, rendered as a
// typographic recipe card. The Divoom cloud proxy whitelists only
// `f.divoom-gz.com` for Image DispElement URLs (see docs/api.md and
// memory/feedback_netdata_cloud_proxy.md), so dynamic text/image
// elements pointing at the cocktailDB API never reach the device.
// At `divoom push` time we fetch the drink JSON and bake the entire
// recipe (name, glass/category, ingredient rows with measures,
// instructions) directly into the scene's bg JPG (see
// bakeCocktailBackground in scene_baked.go). The scene has no
// Elements — every pixel is in the bg.
func cocktailScene(_ map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "cocktail",
		Weight: 20,
		BgPath: bgCocktail,
	}
}
