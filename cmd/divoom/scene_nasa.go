package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "NASA APOD" — Astronomy Picture of the Day.
//
// Originally rendered via an Image DispElement pointing at the APOD URL,
// but Divoom's cloud proxy whitelists only `f.divoom-gz.com` for Image
// element URLs (see docs/api.md and memory/feedback_netdata_cloud_proxy.md),
// so the photo never reached the device. Workaround: at `divoom push`
// time we fetch the APOD JSON, download the photo, composite it + the
// title directly into the scene's bg JPG (see scene_baked.go), and adb-
// push that. The scene definition is then bg-only — the photo and
// title are pixels in the bg, not DispElements. sceneTitle stays as
// the only DispElement because it's a static "NASA APOD" caption row
// that the device-side renderer can handle on its own.
func nasaScene(_ map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "nasa",
		Weight: 20,
		BgPath: bgNASA,
		Elements: []frame.DispElement{
			sceneTitle("NASA APOD"),
		},
	}
}
