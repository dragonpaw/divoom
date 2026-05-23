package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "astronomy picture of the day" — NASA APOD, rotating across the
// curated date pool (see nasaCuratedDates).
//
// Divoom's cloud proxy whitelists only `f.divoom-gz.com` for Image
// DispElement URLs (see docs/api.md and
// memory/feedback_netdata_cloud_proxy.md), so an Image element pointing
// at the live APOD URL never reaches the device. Workaround: at
// `divoom push` time we fetch every curated APOD, composite each into
// a per-index bg JPG, and adb-push each to
// /userdata/wallclock_bg_nasa_NNN.jpg (see bakeAllNASABackgrounds in
// scene_baked.go). The scene title is baked into the bg by
// render.SceneBackground (no Text element).
//
// Index mapping is stable across runs (the curated date list is
// sorted by hand → date N always lives at path N → the local APOD
// cache stays valid). Visible order is randomised per daemon start
// by newIndexWalker — every photo is shown exactly once in a fresh
// random order before any repeats.
func nasaScene(_ map[string]widget.Widget) *scene.Scene {
	walk := newIndexWalker(len(nasaCuratedDates))
	return &scene.Scene{
		Name:   "nasa",
		Weight: WeightInteresting,
		BgPath: bgNASAFor(0),
		BgPathFor: func(_ string) string {
			return bgNASAFor(walk())
		},
	}
}
