package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/dragonpaw/divoom/internal/render"
)

// runRender writes every known scene background to <outDir>/scenes/<name>.jpg.
// Designed to be called from CI, which then commits the output tree to a
// public sibling repo and lets GitHub Pages serve it.
func runRender(args []string) error {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	out := fs.String("out", "dist", "output directory (a scenes/ subdir will be created)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	scenesDir := filepath.Join(*out, "scenes")
	if err := os.MkdirAll(scenesDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", scenesDir, err)
	}

	now := time.Now()
	scenes := []struct {
		name   string
		render func() ([]byte, error)
	}{
		// Smoke-test pattern with corner dots + midline cross + bottom swatches.
		{name: "test", render: func() ([]byte, error) {
			return render.TestBackground(render.FormatJPEG)
		}},
		// Scene-neutral preview — just the gruvbox frame, no glyph.
		{name: "hero", render: func() ([]byte, error) {
			return render.HeroBackground(render.FormatJPEG, now)
		}},
		// Per-scene backgrounds the daemon pushes via adb.
		{name: "scene-markets", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneMarkets, render.FormatJPEG, now)
		}},
		{name: "scene-moonphase", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneMoonphase, render.FormatJPEG, now)
		}},
		{name: "scene-hn", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneHN, render.FormatJPEG, now)
		}},
		{name: "scene-dayofyear", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneDayOfYear, render.FormatJPEG, now)
		}},
		{name: "scene-easter", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneEaster, render.FormatJPEG, now)
		}},
		{name: "scene-babylon5", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneBabylon5, render.FormatJPEG, now)
		}},
		{name: "scene-startrek", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneStarTrek, render.FormatJPEG, now)
		}},
		{name: "scene-discworld", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneDiscworld, render.FormatJPEG, now)
		}},
		{name: "scene-jargon", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneJargon, render.FormatJPEG, now)
		}},
		{name: "scene-catfacts", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneCatFacts, render.FormatJPEG, now)
		}},
		{name: "scene-didyouknow", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneDidYouKnow, render.FormatJPEG, now)
		}},
		{name: "scene-sunrise", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneSunrise, render.FormatJPEG, now)
		}},
		{name: "scene-zenquotes", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneZenQuotes, render.FormatJPEG, now)
		}},
		{name: "scene-devil", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneDevil, render.FormatJPEG, now)
		}},
		{name: "scene-nasa", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneNASA, render.FormatJPEG, now)
		}},
		{name: "scene-cocktail", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneCocktail, render.FormatJPEG, now)
		}},
		{name: "scene-onthisday", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneOnThisDay, render.FormatJPEG, now)
		}},
		{name: "scene-iss", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneISS, render.FormatJPEG, now)
		}},
		{name: "scene-github", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneGitHub, render.FormatJPEG, now)
		}},
		{name: "scene-til", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneTIL, render.FormatJPEG, now)
		}},
		{name: "scene-wordnik", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneWordnik, render.FormatJPEG, now)
		}},
		{name: "scene-stoics", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneStoics, render.FormatJPEG, now)
		}},
		{name: "scene-twain", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneTwain, render.FormatJPEG, now)
		}},
		{name: "scene-fortune", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneFortune, render.FormatJPEG, now)
		}},
	}
	// One preview per weather outlook so the icon set can be spot-checked
	// without spinning up the daemon.
	for _, outlook := range []string{
		"clear", "cloudy", "overcast", "rain",
		"drizzle", "snow", "fog", "thunder",
		"smoke", "hazard",
	} {
		o := outlook
		scenes = append(scenes, struct {
			name   string
			render func() ([]byte, error)
		}{
			name: "scene-weather-" + o,
			render: func() ([]byte, error) {
				return render.SceneWeatherBackground(o, render.FormatJPEG, now)
			},
		})
	}

	if len(scenes) == 0 {
		return errors.New("no scenes defined")
	}

	for _, s := range scenes {
		data, err := s.render()
		if err != nil {
			return fmt.Errorf("render %q: %w", s.name, err)
		}
		path := filepath.Join(scenesDir, s.name+".jpg")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		slog.Info("rendered scene", "name", s.name, "path", path, "bytes", len(data))
	}
	slog.Info("render complete", "scenes", len(scenes), "out", scenesDir)
	return nil
}
