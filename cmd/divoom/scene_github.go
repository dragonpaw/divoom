package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "GitHub" — lifetime activity card. A giant comma-separated total of
// every contribution across every year of the account dominates the
// scene as a single legible number; underneath, two small stats —
// total PRs authored and years on GitHub — give context. Replaced the
// older today/streak/PRs layout whose "1 today" hero was confusing.
//
// Only registered when the widget is wired in (cmd/divoom/serve.go
// gates on GITHUB_USER + GITHUB_TOKEN env vars and omits the widget
// entirely when either is unset); without the conditional append the
// scene would still be in the rotation as a dead nil-widget slot.
func githubScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "github",
		Weight: 20,
		BgPath: bgGitHub,
		Elements: []frame.DispElement{
			sceneTitle("GitHub"),
			// Hero: lifetime contributions, big mono. cGreen when
			// non-zero so the wall-of-numbers reads bright.
			{
				ID: idSceneMain, Type: "Text",
				StartX: 40, StartY: 540, Width: 720, Height: 160,
				Align: 2, FontSize: 130, FontID: fontMono,
				FontColor: cFgDark, BgColor: cBgHard,
			},
			// Hero caption — small prose-light, dim. Static text.
			{
				ID: idSceneSub3, Type: "Text",
				StartX: 80, StartY: 720, Width: 640, Height: 36,
				Align: 2, FontSize: 24, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
				TextMessage: "lifetime contributions",
			},
			// Two-column secondary stats: PR count on the left,
			// years-on-github on the right. Each pairs a big mono
			// number above a small dim caption — same shape as the
			// markets percent badges so the gestalt reads "stat tile".
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 40, StartY: 840, Width: 360, Height: 90,
				Align: 2, FontSize: 70, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 400, StartY: 840, Width: 360, Height: 90,
				Align: 2, FontSize: 70, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Captions for the two stat columns — static.
			{
				ID: idSceneSub4, Type: "Text",
				StartX: 40, StartY: 950, Width: 360, Height: 30,
				Align: 2, FontSize: 22, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
				TextMessage: "PRs",
			},
			{
				ID: idSceneSub5, Type: "Text",
				StartX: 400, StartY: 950, Width: 360, Height: 30,
				Align: 2, FontSize: 22, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
				TextMessage: "years on GitHub",
			},
		},
		Widget: widgets["github"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: githubLifetime},
			{ID: idSceneSub1, Format: githubTotalPRs},
			{ID: idSceneSub2, Format: githubYears},
		},
	}
}
