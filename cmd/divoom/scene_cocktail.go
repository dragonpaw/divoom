package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Cocktail" — random drink from TheCocktailDB.
//
// Same story as NASA APOD: the Divoom cloud proxy whitelists only
// `f.divoom-gz.com` for Image DispElement URLs (see docs/api.md and
// memory/feedback_netdata_cloud_proxy.md), so the drink photo never
// reached the device. At `divoom push` time we fetch the drink JSON,
// download the thumbnail, composite it + the drink name + the
// ingredient list directly into the scene's bg JPG (see
// scene_baked.go), and adb-push that. The scene definition is then
// bg-only — everything except the static "cocktail" caption row is
// pixels in the bg.
func cocktailScene(_ map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "cocktail",
		Weight: 20,
		BgPath: bgCocktail,
		Elements: []frame.DispElement{
			sceneTitle("cocktail"),
		},
	}
}
