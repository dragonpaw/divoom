package main

import (
	"strings"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Reddit" — top-of-day post from a randomly-chosen configured
// subreddit. Title is the hero; the "r/<sub>" accent in aqua sits just
// below it to visually distinguish from the HN scene's orange wordmark;
// the post's link domain sits under that in dim mono (collapses to
// blank for self-posts where the "self.<sub>" string adds no
// information); a metadata footer carries score / byline / age / comments.
//
// Widget output: 7 pipe segments —
//   0:sub 1:title 2:domain 3:score 4:author 5:age 6:comments
func redditScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "reddit",
		Weight: WeightInformational,
		BgPath: bgReddit,
		Elements: []frame.DispElement{
			// Post title — hero. Left-aligned, prose font, full fg.
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 540, Width: 640, Height: 240,
				Align: 0, FontSize: 38, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
			// "r/<sub>" accent — aqua so it reads as the per-scene
			// brand mark distinct from the HN orange.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 820, Width: 640, Height: 40,
				Align: 0, FontSize: 24, FontID: fontMono,
				FontColor: cAqua, BgColor: cBgHard,
			},
			// Post's link domain — dim mono caption. AllowEmpty
			// because self-posts come back as "self.<sub>" which
			// duplicates the row above; redditDomain blanks it then.
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 80, StartY: 890, Width: 640, Height: 40,
				Align: 0, FontSize: 24, FontID: fontMono,
				FontColor: cFgDark, BgColor: cBgHard,
			},
			// Metadata footer — mono, composed from segments 3-6.
			{
				ID: idSceneSub3, Type: "Text",
				StartX: 80, StartY: 1140, Width: 640, Height: 40,
				Align: 0, FontSize: 24, FontID: fontMono,
				FontColor: cFgDark, BgColor: cBgHard,
			},
		},
		Widget: widgets["reddit"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: redditTitle},
			{ID: idSceneSub1, Format: redditSubLabel, AllowEmpty: true},
			{ID: idSceneSub2, Format: redditDomain, AllowEmpty: true},
			{ID: idSceneSub3, Format: redditFooter, AllowEmpty: true},
		},
	}
}

// redditSubLabel returns "r/<sub>" from segment 0 of the widget output.
// Returns "" when the segment is missing so the AllowEmpty mount
// collapses the row rather than rendering a bare "r/".
func redditSubLabel(raw string) (text, color string) {
	sub := pipeAtRaw(raw, 0)
	if sub == "" {
		return "", ""
	}
	return "r/" + sub, ""
}

// redditDomain returns the link domain from segment 2, blanking out
// self-posts (where reddit reports the domain as "self.<sub>") so the
// row doesn't redundantly duplicate the accent row above.
func redditDomain(raw string) (text, color string) {
	d := pipeAtRaw(raw, 2)
	if strings.HasPrefix(d, "self.") {
		return "", ""
	}
	return d, ""
}

// redditFooter composes the metadata footer "▲ <score>  by <author>  ·
// <age>  ·  <comments> comments" from the widget's 3..6 segments,
// dropping any piece that's empty so partial data renders cleanly.
// Structurally identical to hnFooter; kept as a sibling formatter
// rather than abstracted because the segment indices differ between
// the two widgets (HN is 4..7; reddit is 3..6) and the small bit of
// duplication is cheaper than a parametrised helper.
func redditFooter(raw string) (text, color string) {
	parts := strings.Split(raw, "|")
	if len(parts) < 7 {
		return "", ""
	}
	score := strings.TrimSpace(parts[3])
	author := strings.TrimSpace(parts[4])
	age := strings.TrimSpace(parts[5])
	comments := strings.TrimSpace(parts[6])

	byline := "by " + author
	if author == "" {
		byline = "by unknown"
	}
	var segs []string
	if score != "" && score != "0" {
		segs = append(segs, "▲ "+score+"  "+byline)
	} else {
		segs = append(segs, byline)
	}
	if age != "" {
		segs = append(segs, age)
	}
	if comments != "" && comments != "0" {
		segs = append(segs, comments+" comments")
	}
	return strings.Join(segs, "  ·  "), ""
}

// redditTitle is the title formatter — segment 1 of the widget output.
// Provided as a named formatter (not bare pipeAt(1)) so the scene's
// Mount reads consistently with the other reddit slots and so a future
// title-cleanup pass has a place to land.
func redditTitle(raw string) (text, color string) {
	return pipeAt(1)(raw)
}
