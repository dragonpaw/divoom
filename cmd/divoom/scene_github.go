package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "GitHub" — today's commit count, current contribution streak, and
// open-PR count for the configured user. Only registered when the
// widget is wired in (cmd/divoom/serve.go gates on the GITHUB_USER +
// GITHUB_TOKEN env vars and omits the widget entirely when either is
// unset); without the conditional append the scene would still be in
// the rotation as a dead nil-widget slot showing the static "GitHub"
// title and nothing else. buildScenes() handles that gate.
func githubScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "github",
		Weight: 20,
		BgPath: bgGitHub,
		Elements: []frame.DispElement{
			sceneTitle("GitHub"),
			// Today's commit count — big mono. cGreen when non-zero so
			// a productive day reads bright; cFgDark when zero so a
			// quiet day fades into the background.
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 540, Width: 640, Height: 160,
				Align: 2, FontSize: 130, FontID: fontMono,
				FontColor: cFgDark, BgColor: cBgHard,
			},
			// Current streak — medium mono. cYellow above 7 days as a
			// "you're on a roll" signal; cFgDark below so short streaks
			// don't shout for attention.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 720, Width: 640, Height: 120,
				Align: 2, FontSize: 70, FontID: fontMono,
				FontColor: cFgDark, BgColor: cBgHard,
			},
			// Open PRs — small prose, aqua. Always rendered with the
			// "PRs" suffix so the unit is unambiguous next to the
			// numeric streak above.
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 80, StartY: 860, Width: 640, Height: 120,
				Align: 2, FontSize: 36, FontID: fontProse,
				FontColor: cAqua, BgColor: cBgHard,
			},
		},
		Widget: widgets["github"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: githubCommits},
			{ID: idSceneSub1, Format: githubStreak},
			{ID: idSceneSub2, Format: githubPRs},
		},
	}
}
