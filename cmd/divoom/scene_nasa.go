package main

import (
	"fmt"
	"math/rand/v2"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "NASA APOD" — Astronomy Picture of the Day, rotating across the
// curated date pool (see nasaCuratedDates).
//
// Divoom's cloud proxy whitelists only `f.divoom-gz.com` for Image
// DispElement URLs (see docs/api.md and
// memory/feedback_netdata_cloud_proxy.md), so an Image element pointing
// at the live APOD URL never reaches the device. Workaround: at
// `divoom push` time we fetch every curated APOD, composite each
// into a per-index bg JPG, and adb-push each to
// /userdata/wallclock_bg_nasa_NNN.jpg (see
// bakeAllNASABackgrounds in scene_baked.go). The scene's BgPathFor
// picks a random index per activation so a different photo lands on
// screen each time the rotator surfaces NASA.
func nasaScene(_ map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "nasa",
		Weight: 20,
		// Default to index 0 for any path where BgPathFor returns "".
		BgPath: bgNASAFor(0),
		BgPathFor: func(_ string) string {
			return bgNASAFor(rand.IntN(len(nasaCuratedDates)))
		},
		Elements: []frame.DispElement{
			sceneTitle(fmt.Sprintf("NASA APOD · %d-image rotation", len(nasaCuratedDates))),
		},
	}
}
