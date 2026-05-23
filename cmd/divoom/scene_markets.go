package main

import (
	"strings"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// marketsScene — trading-terminal readout: ticker symbol on the left and
// price on the right of one big mono headline row, a combined week/month
// percent badge underneath (mono-padded so the two halves align), and a
// Unicode-block sparkline of the last ~35 closes across the bottom. The
// DIVOOM_TICKERS env var rotates one symbol per activation; see
// finance.NewRotating.
//
// Element count: baked title (0) + symbol (1) + price (1) + combined
// week/month badge (1) + sparkline (1) = 4 scene Text. Plus 2 always-on
// Text = 6 Text. At the device's per-type cap. The captions "1 week" /
// "1 month" sit under the badge row, baked by drawMarketsChrome; the
// ticker symbol itself is NOT baked because DIVOOM_TICKERS may rotate
// among several symbols and one bg per ticker would multiply the bake
// pool unnecessarily.
func marketsScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "markets",
		Weight: 20,
		BgPath: bgMarkets,
		Elements: []frame.DispElement{
			// Ticker symbol — left-aligned, ticker-coloured (cYellow as a
			// quiet accent that doesn't compete with the green/red badge
			// signal below). On the same y-row as the price element.
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 540, Width: 640, Height: 120,
				Align: 0, FontSize: 80, FontID: fontMono,
				FontColor: cYellow, BgColor: cBgHard,
			},
			// Price — right-aligned, white. Shares the headline row with
			// the symbol so the eye reads "QQQ ... $478.21" left-to-right.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 540, Width: 640, Height: 120,
				Align: 1, FontSize: 80, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Combined week + month percent badge — "▲ +1.2 %   ▼ -3.7 %",
			// mono-padded so the two halves visually split the row.
			// Colour is set per-activation by marketsColorize from the
			// sign of the week percent (the month sign is the secondary
			// signal; one colour per row keeps the read fast).
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 80, StartY: 720, Width: 640, Height: 120,
				Align: 2, FontSize: 60, FontID: fontMono,
				FontColor: cFgDark, BgColor: cBgHard,
			},
			// Sparkline strip — last ~35 closes as Unicode blocks.
			{
				ID: idSceneSub3, Type: "Text",
				StartX: 40, StartY: 950, Width: 720, Height: 120,
				Align: 2, FontSize: 70, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["markets"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: pipeAt(0)},
			{ID: idSceneSub1, Format: pipeAt(1)},
			{ID: idSceneSub2, Format: marketsChangeBoth},
			{ID: idSceneSub3, Format: pipeAt(4)},
		},
		OnActivate: marketsColorize,
	}
}

// parseTickerList splits a comma-separated DIVOOM_TICKERS env value
// into a list of normalised symbols. Whitespace is trimmed around each
// entry; entries are upper-cased; empties are dropped. Empty/unset input
// returns nil so finance.NewRotating can apply its default ("QQQ").
func parseTickerList(env string) []string {
	if strings.TrimSpace(env) == "" {
		return nil
	}
	var out []string
	for _, raw := range strings.Split(env, ",") {
		sym := strings.ToUpper(strings.TrimSpace(raw))
		if sym == "" {
			continue
		}
		out = append(out, sym)
	}
	return out
}
