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
			// Three-column secondary stats — total PRs / open PRs /
			// years on GitHub. Each column ~240px wide, stat above
			// caption like the markets percent badges. Open PRs in
			// the middle slot recolour-flips per githubOpenPRs so
			// the live signal stands apart from the lifetime totals
			// flanking it.
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
				ID: idSceneSub4, Type: "Text",
				StartX: 520, StartY: 840, Width: 240, Height: 90,
				Align: 2, FontSize: 60, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Captions for the three stat columns — static.
			{
				ID: idSceneSub5, Type: "Text",
				StartX: 40, StartY: 950, Width: 240, Height: 30,
				Align: 2, FontSize: 22, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
				TextMessage: "total PRs",
			},
			{
				ID: idSceneSub6, Type: "Text",
				StartX: 280, StartY: 950, Width: 240, Height: 30,
				Align: 2, FontSize: 22, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
				TextMessage: "open",
			},
			{
				ID: idSceneSub7, Type: "Text",
				StartX: 520, StartY: 950, Width: 240, Height: 30,
				Align: 2, FontSize: 22, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
				TextMessage: "years",
			},
		},
		Widget: widgets["github"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: githubLifetime},
			{ID: idSceneSub1, Format: githubTotalPRs},
			{ID: idSceneSub2, Format: githubOpenPRs},
			{ID: idSceneSub4, Format: githubYears},
		},
	}
}
