package main

import (
	"strings"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// marketsScene — trading-terminal readout: "<SYM>            $price" on
// one big mono headline, week-/month-percent badges with arrow + colour
// underneath, and a Unicode-block sparkline of the last ~35 closes
// across the bottom. The DIVOOM_TICKERS env var rotates one symbol per
// activation; see finance.NewRotating.
//
// Element count stays at the device's 6-Text + 1-Time cap by collapsing
// the symbol and price into a single padded Text element (see
// marketsSymbolPrice); the captions "1 week" / "1 month" are baked into
// the background by drawMarketsChrome.
func marketsScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "markets",
		Weight: 20,
		BgPath: bgMarkets,
		Elements: []frame.DispElement{
			// Symbol + price on one line — left-aligned, padded so the
			// price reads as right-of-centre on the 720px text track.
			{
				ID: idSceneMain, Type: "Text",
				StartX: 40, StartY: 540, Width: 720, Height: 120,
				Align: 0, FontSize: 70, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			// Week percent badge — "▲ +1.2 %" / "▼ -3.7 %" / "· 0 %".
			// Colour set per-activation by marketsColorize.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 720, Width: 320, Height: 120,
				Align: 2, FontSize: 60, FontID: fontMono,
				FontColor: cFgDark, BgColor: cBgHard,
			},
			// Month percent badge — same shape.
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 400, StartY: 720, Width: 320, Height: 120,
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
			{ID: idSceneMain, Format: marketsSymbolPrice},
			{ID: idSceneSub1, Format: marketsChange(2)},
			{ID: idSceneSub2, Format: marketsChange(3)},
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
