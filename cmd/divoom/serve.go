package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/dragonpaw/divoom/internal/adb"
	"github.com/dragonpaw/divoom/internal/render"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
	"github.com/dragonpaw/divoom/internal/widget/calendar"
	"github.com/dragonpaw/divoom/internal/widget/easter"
	"github.com/dragonpaw/divoom/internal/widget/facts"
	"github.com/dragonpaw/divoom/internal/widget/finance"
	"github.com/dragonpaw/divoom/internal/widget/news"
	"github.com/dragonpaw/divoom/internal/widget/rotator"
	"github.com/dragonpaw/divoom/internal/widget/sky"
	"github.com/dragonpaw/divoom/internal/widget/weather"
)

const (
	defaultLat = "37.9358" // Richmond, CA centroid
	defaultLon = "-122.3477"
)

// Widget data-refresh cadences — how often each source is asked for new
// data. Completely independent from how long each scene stays on screen
// (those durations live in scenes.go). To re-tune a widget's data
// freshness, change one constant here.
const (
	weatherInterval = 30 * time.Minute
	qqqInterval     = 1 * time.Hour
	moonInterval    = 6 * time.Hour
	whimsyInterval  = 60 * time.Second // fresh per Ambient turn
)

// hnKeywords gates which HackerNews stories qualify for the whimsy slot.
// Tuned to Ash's interests (Claude, Linux, 3D printing, PC gaming).
var hnKeywords = []string{
	"claude", "anthropic", "llm",
	"linux", "kernel", "wayland",
	"3d print", "voron", "klipper",
	"steam", "valve", "proton",
}

// runServe installs per-scene backgrounds, starts every widget runner in
// the background, then rotates scenes forever. Time + Date are always on
// the top; the bottom area swaps between Now / Markets / Sky / Ambient.
func runServe(ctx context.Context) error {
	lat := getenv("DIVOOM_LAT", defaultLat)
	lon := getenv("DIVOOM_LON", defaultLon)
	slog.Info("location", "lat", lat, "lon", lon)

	client, _, err := connectToFrame(ctx)
	if err != nil {
		return err
	}

	if err := pushSceneBackgrounds(ctx); err != nil {
		return err
	}

	// Build the whimsy rotator (one source picked at random on each Fetch,
	// weighted, with rare easter eggs). All HEADER|BODY sources, rendered
	// by the single "ambient" card scene.
	whimsyWidget := rotator.New("whimsy", []rotator.Source{
		{Widget: facts.NewCatFact(), Weight: 3},
		{Widget: facts.NewUselessFact(), Weight: 3},
		{Widget: news.NewHN(hnKeywords), Weight: 3},
		{Widget: calendar.NewDayOfYear(), Weight: 3},
		{Widget: easter.New(), Weight: 1},
	}).WithMaxLen(160) // ~4 lines at FontSize 26 (line height ~100px observed)

	weatherR := widget.NewRunner(weather.New(lat, lon), weatherInterval)
	qqqR := widget.NewRunner(finance.NewTicker("QQQ"), qqqInterval)
	moonR := widget.NewRunner(sky.NewMoon(), moonInterval)
	whimsyR := widget.NewRunner(whimsyWidget, whimsyInterval)

	runners := []*widget.Runner{weatherR, qqqR, moonR, whimsyR}
	var wg sync.WaitGroup
	for _, r := range runners {
		r := r
		wg.Add(1)
		go func() {
			defer wg.Done()
			slog.Info("starting widget runner", "widget", r.Name(), "interval", r.Interval)
			r.Start(ctx)
		}()
	}

	driver := &scene.Driver{
		Client:   client,
		AlwaysOn: alwaysOn(),
		Scenes:   buildScenes(weatherR, qqqR, moonR, whimsyR),
	}
	slog.Info("scene rotation starting", "scenes", len(driver.Scenes))
	if err := driver.Run(ctx); err != nil {
		slog.Error("scene driver returned", "err", err)
	}

	wg.Wait()
	return nil
}

// pushSceneBackgrounds renders each scene's bg JPG and adb-pushes it to
// the device. Done once at startup; the device will reference these paths
// via BackgroundImageLocalFlag: 1 in scene layouts.
func pushSceneBackgrounds(ctx context.Context) error {
	now := time.Now()
	bgs := []struct {
		render func() ([]byte, error)
		path   string
	}{
		{func() ([]byte, error) { return render.SceneBackground(render.SceneNow, render.FormatJPEG, now) }, bgNow},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneMarkets, render.FormatJPEG, now) }, bgMarkets},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneSky, render.FormatJPEG, now) }, bgSky},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneAmbient, render.FormatJPEG, now) }, bgAmbient},
	}
	for _, b := range bgs {
		data, err := b.render()
		if err != nil {
			return fmt.Errorf("render %s bg: %w", b.path, err)
		}
		if err := pushBytes(ctx, data, b.path); err != nil {
			return fmt.Errorf("push %s: %w", b.path, err)
		}
	}
	return nil
}

func pushBytes(ctx context.Context, data []byte, devicePath string) error {
	tmp, err := os.CreateTemp("", "wallclock-bg-*.jpg")
	if err != nil {
		return fmt.Errorf("temp file: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	tmp.Close()

	pushCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return adb.Push(pushCtx, tmp.Name(), devicePath)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
