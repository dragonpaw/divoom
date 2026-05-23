package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "GitHub" — lifetime activity card. A giant comma-separated total of
// every contribution dominates the scene; underneath, three small
// stat tiles read the live "what's outstanding" state alongside the
// slower-changing lifetime totals: total PRs, currently-open PRs
// (cAqua when non-zero so it stands out as the live signal), and
// years on GitHub.
//
// The title, hero caption, and three stat-column labels are baked
// into the bg by render.drawGitHubChrome — the dynamic numbers alone
// already use 4 of the device's 6 Text slots, so the labels can't
// also be Text elements without the layout silently dropping them.
//
// Only registered when the widget is wired in (cmd/divoom/serve.go
// gates on GITHUB_USER + GITHUB_TOKEN env vars and omits the widget
// entirely when either is unset); without the conditional append the
// scene would still be in the rotation as a dead nil-widget slot.
func githubScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "github",
		Weight: WeightInformational,
		BgPath: bgGitHub,
		Elements: []frame.DispElement{
			// Hero: lifetime contributions, big mono. cGreen when
			// non-zero so the wall-of-numbers reads bright.
			{
				ID: idSceneMain, Type: "Text",
				StartX: 40, StartY: 540, Width: 720, Height: 160,
				Align: 2, FontSize: 130, FontID: fontMono,
				FontColor: cFgDark, BgColor: cBgHard,
			},
			// Three-column secondary stats: total PRs / open PRs /
			// years on GitHub. Open PRs in the middle slot recolour-
			// flips per githubOpenPRs so the live signal stands apart
			// from the lifetime totals flanking it. Column labels
			// are baked by render.drawGitHubChrome at y=965.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 40, StartY: 840, Width: 240, Height: 90,
				Align: 2, FontSize: 60, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 280, StartY: 840, Width: 240, Height: 90,
				Align: 2, FontSize: 60, FontID: fontMono,
				FontColor: cFgDark, BgColor: cBgHard,
			},
			{
				ID: idSceneSub3, Type: "Text",
				StartX: 520, StartY: 840, Width: 240, Height: 90,
				Align: 2, FontSize: 60, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["github"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: githubLifetime},
			{ID: idSceneSub1, Format: githubTotalPRs},
			{ID: idSceneSub2, Format: githubOpenPRs},
			{ID: idSceneSub3, Format: githubYears},
		},
	}
}
