package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dragonpaw/divoom/internal/render"
	"github.com/dragonpaw/divoom/internal/widget"
)

// TestRefreshScreenshots regenerates docs/screenshots/*.jpg by baking
// realistic widget output onto each scene's already-rendered bg JPG.
//
// Gated behind DIVOOM_PREVIEW=1 (mirrors the existing preview test
// pattern) so a plain `go test ./...` doesn't write files. Run with:
//
//	DIVOOM_PREVIEW=1 go test ./cmd/divoom -run TestRefreshScreenshots
//
// The bake reads cmd/divoom/../../dist/scenes/scene-<name>.jpg (the
// output of `divoom render`), paints the always-on header on top, then
// paints every device-side Text/Time/Week element using a per-scene
// fixture string that the device would normally fetch from a widget.
//
// Skipped scenes:
//   - easter (no screenshot per project decision)
//   - cocktail, nasa (already real-content composites baked from
//     upstream — leave them as-is)
//   - moonphase variants other than 07 (only moonphase-07.jpg ships
//     in docs/screenshots/)
//   - weather variants other than cloudy (only weather-cloudy.jpg
//     ships in docs/screenshots/)
//
// The fixture table lives inline so a human can audit / iterate
// without going through a separate config file.
func TestRefreshScreenshots(t *testing.T) {
	if os.Getenv("DIVOOM_PREVIEW") != "1" {
		t.Skip("set DIVOOM_PREVIEW=1 to regenerate docs/screenshots/")
	}

	// Locate repo root from the test's working dir (cmd/divoom).
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	bgDir := filepath.Join(repoRoot, "dist", "scenes")
	outDir := filepath.Join(repoRoot, "docs", "screenshots")
	if _, err := os.Stat(bgDir); err != nil {
		t.Fatalf("dist/scenes missing — run `divoom render` first: %v", err)
	}

	// Hardcoded timestamp matches cmd/divoom/render.go so the always-on
	// header layout (day name, time, footer, weekend status) lines up
	// across every screenshot.
	now := time.Date(2026, time.May, 27, 12, 34, 0, 0,
		time.FixedZone("local", -7*3600))

	// Fixtures — one realistic raw-widget output per scene. Drawn from
	// the curated quote lists / real widget shapes, not "TEST". The
	// scene's mounts + OnActivate transform these into the final
	// per-element text just as the daemon does.
	type entry struct {
		// Scene name as used by buildScenes (also the docs/screenshots
		// filename stem and the dist/scenes/scene-<name>.jpg basename).
		Name string
		// Raw widget output; "" for scenes whose only dynamic content
		// is the always-on header (everything else is baked into bg).
		Raw string
	}
	fixtures := []entry{
		{"markets", "QQQ|$478.21|+1.2|+5.0|▁▂▃▅▆▇█▇▅▃▂▁▂▃▅▆▇█▇▆▅▄▃▂▁▂▃▄|2026-05-26"},
		{"moonphase-07", "moon · full · 100% · next full moon in 30 days"},
		{"hn", "Hacker News|Claude Code adds 1M context tokens for Opus 4.7|anthropic.com|Anthropic shipped a million-token context window today, available now in Claude Code with the long-context beta flag.|412|simonw|3h|187"},
		{"dayofyear", "40%|Year 2026|Day 147 of 365"},
		{"babylon5", "Babylon 5|The avalanche has started, it is too late for the pebbles to vote.|Kosh"},
		{"startrek", "Star Trek|Logic is the beginning of wisdom, not the end.|Spock"},
		{"discworld", "Discworld|It is well known that a vital ingredient of success is not knowing that what you're attempting can't be done.|Terry Pratchett"},
		{"jargon", "Jargon File|yak shaving /yak shay'ving/ n. Any apparently useless activity which, by allowing you to overcome intermediate difficulties, allows you to solve a larger problem. Also used to describe recursive yak-shaving expeditions which spiral out of control. See also: hack value, deep magic."},
		{"zenquotes", "ZenQuotes|The mind is everything. What you think you become.|Buddha"},
		{"wordnik", "Word of the Day|ephemeral adj. Lasting a very short time; short-lived; transitory.||/ɪˈfɛm(ə)rəl/"},
		{"stoics", "Stoics|You have power over your mind — not outside events. Realize this, and you will find strength.|Marcus Aurelius"},
		{"twain", "Mark Twain|The two most important days in your life are the day you are born and the day you find out why.|Mark Twain"},
		{"fortune", "fortune|Any sufficiently advanced technology is indistinguishable from magic.|Arthur C. Clarke"},
		{"devil", "Devil's Dictionary|CYNIC, n. A blackguard whose faulty vision sees things as they are, not as they ought to be.|Ambrose Bierce"},
		{"catfacts", "cat fact|A cat's purr vibrates at a frequency between 25 and 150 Hertz, the same range scientists have shown can promote tissue regeneration and bone healing in mammals."},
		{"til", "TIL|that octopuses have three hearts, blue blood, and nine brains — a central brain and one in each of their eight arms that can act independently."},
		{"didyouknow", "did you know?|Bananas are berries, but strawberries are not. Botanically, a berry has seeds inside its flesh — bananas, tomatoes, and grapes all qualify; strawberries do not."},
		{"onthisday", "1969|Apollo 10 returned to Earth after a successful mission as a dress rehearsal for the first lunar landing, splashing down in the Pacific Ocean east of American Samoa."},
		{"sunrise", "5:52 AM|8:23 PM|14h 31m"},
		{"weather-cloudy", "63°F|cloudy||45|62|30"},
		{"iss", "-22.5°, -45.3°|next pass in 47m|over South America"},
		{"github", "14238|287|4|11"},
		{"reddit", "pcgaming|Half-Life 3 finally confirmed at The Game Awards|polygon.com|2841|gabe_irl|2h|1247"},
	}

	// Build the scene set. Provide a non-nil stub for the github widget
	// so buildScenes includes the github scene.
	widgets := map[string]widget.Widget{
		"github": stubWidget{},
		"reddit": stubWidget{},
	}
	scenes := buildScenes(widgets)
	sceneByName := map[string]int{}
	for i, s := range scenes {
		sceneByName[s.Name] = i
	}

	for _, fx := range fixtures {
		t.Run(fx.Name, func(t *testing.T) {
			// Resolve bg file: most scenes are scene-<name>.jpg; the
			// weather and moonphase variants embed the variant suffix
			// in the screenshot filename (weather-cloudy → scene-
			// weather-cloudy.jpg, moonphase-07 → scene-moonphase-07.jpg).
			bgPath := filepath.Join(bgDir, "scene-"+fx.Name+".jpg")
			bg, err := os.ReadFile(bgPath)
			if err != nil {
				t.Fatalf("read bg %s: %v", bgPath, err)
			}

			// dist/scenes/scene-*.jpg is the output of `divoom render`,
			// which already runs BakeAlwaysOnHeaderJPEG over the bg
			// before writing — see cmd/divoom/render.go. Just paint the
			// scene's dynamic elements on top.
			//
			// Look up the scene and render its elements (mounts +
			// OnActivate). Strip the variant suffix for weather /
			// moonphase since buildScenes only registers the base scene.
			sceneName := baseSceneName(fx.Name)
			idx, ok := sceneByName[sceneName]
			if !ok {
				t.Fatalf("scene %q not in buildScenes", sceneName)
			}
			elements := scenes[idx].RenderElements(fx.Raw, now)
			out, err := render.BakeSceneElementsJPEG(bg, elements, now)
			if err != nil {
				t.Fatalf("bake elements: %v", err)
			}

			outPath := filepath.Join(outDir, fx.Name+".jpg")
			if err := os.WriteFile(outPath, out, 0o644); err != nil {
				t.Fatalf("write %s: %v", outPath, err)
			}
			t.Logf("wrote %s (%d bytes)", outPath, len(out))
		})
	}
}

// baseSceneName strips the variant suffix from screenshot names so the
// "weather-cloudy" and "moonphase-07" screenshots resolve to the
// "weather" and "moonphase" scenes that buildScenes registers.
func baseSceneName(s string) string {
	switch {
	case len(s) > len("weather-") && s[:len("weather-")] == "weather-":
		return "weather"
	case len(s) > len("moonphase-") && s[:len("moonphase-")] == "moonphase-":
		return "moonphase"
	default:
		return s
	}
}

// stubWidget is a placeholder Widget used to satisfy buildScenes's
// gating on the github widget being present. It's never actually
// fetched during the screenshot bake — the test supplies its own raw
// fixture string straight to Scene.RenderElements.
type stubWidget struct{}

func (stubWidget) Name() string                                  { return "stub" }
func (stubWidget) Fetch(ctx context.Context) (string, error)     { return "", nil }
