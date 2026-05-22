package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/dragonpaw/divoom/internal/adb"
	"github.com/dragonpaw/divoom/internal/render"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
	"github.com/dragonpaw/divoom/internal/widget/calendar"
	"github.com/dragonpaw/divoom/internal/widget/easter"
	"github.com/dragonpaw/divoom/internal/widget/facts"
	"github.com/dragonpaw/divoom/internal/widget/finance"
	"github.com/dragonpaw/divoom/internal/widget/food"
	githubw "github.com/dragonpaw/divoom/internal/widget/github"
	"github.com/dragonpaw/divoom/internal/widget/news"
	"github.com/dragonpaw/divoom/internal/widget/quotes"
	"github.com/dragonpaw/divoom/internal/widget/sky"
	"github.com/dragonpaw/divoom/internal/widget/weather"
	"github.com/dragonpaw/divoom/internal/widget/wikipedia"
)

// hnKeywords gates which HackerNews stories qualify for the hn scene.
// Tuned to Ash's interests (Claude, Linux, 3D printing, PC gaming).
var hnKeywords = []string{
	"claude", "anthropic", "llm",
	"linux", "kernel", "wayland",
	"3d print", "voron", "klipper",
	"steam", "valve", "proton",
}

// runServe installs per-scene backgrounds, then rotates scenes forever.
// Each scene's widget refreshes on unload, so the next activation
// renders from a warm cache without waiting on the network. Time + Date
// + DoW are always on top; the bottom area swaps Markets / Sky / Ambient.
func runServe(ctx context.Context) error {
	client, _, err := connectToFrame(ctx)
	if err != nil {
		return err
	}

	if err := pushSceneBackgrounds(ctx); err != nil {
		return err
	}

	weatherWidget := weather.New("37.9358", "-122.3477")
	// Auto-calibrate weather temperature colour thresholds to the
	// location's climate. Fire-and-forget: daemon startup doesn't wait
	// on the archive API (it can take 5-10s over 5 years of data).
	// Until it returns, the static defaults baked into scenes.go
	// stand; on success the next scene activation picks up the new
	// bounds via atomic load.
	go func() {
		cold, hot, err := weatherWidget.LoadThresholds(ctx)
		if err != nil {
			slog.Warn("weather threshold calibration failed; using static defaults",
				"err", err, "cold", 50, "hot", 80)
			return
		}
		SetWeatherThresholds(cold, hot)
		slog.Info("weather thresholds calibrated",
			"lat", weatherWidget.Lat(),
			"lon", weatherWidget.Lon(),
			"cold_below_F", cold,
			"hot_at_or_above_F", hot,
		)
	}()

	widgets := map[string]widget.Widget{
		"markets":    finance.NewTicker("QQQ"),
		"sky":        sky.NewMoon(),
		"hn":         news.NewHN(hnKeywords),
		"dayofyear":  calendar.NewDayOfYear(),
		"easter":     easter.New(),
		"babylon5":   quotes.NewBabylon5(),
		"startrek":   quotes.NewStarTrek(),
		"discworld":  quotes.NewDiscworld(),
		"jargon":     quotes.NewJargonFile(),
		"catfacts":   facts.NewCatFact(),
		"didyouknow": facts.NewUselessFact(),
		"sunrise":    sky.NewSunrise(),
		"weather":    weatherWidget,
		"zenquotes":  quotes.NewZenQuotes(),
		"devil":      quotes.NewDevilsDictionary(),
		"nasa":       sky.NewAPOD(),
		"cocktail":   food.New(),
		"onthisday":  wikipedia.NewOnThisDay(),
		"iss":        sky.NewISS("37.9358", "-122.3477"),
	}

	// GitHub scene is opt-in via env vars. Both must be set: without the
	// token the unauthenticated REST quota (60 req/hr) is too small for
	// the rotation cadence, and without the user there's nobody to query
	// for. When either is missing the widget isn't constructed and
	// buildScenes drops the scene from the rotation entirely.
	if ghUser, ghToken := os.Getenv("GITHUB_USER"), os.Getenv("GITHUB_TOKEN"); ghUser != "" && ghToken != "" {
		widgets["github"] = githubw.New(ghUser, ghToken)
		slog.Info("github scene enabled", "user", ghUser)
	} else {
		slog.Info("github scene disabled (set GITHUB_USER + GITHUB_TOKEN)")
	}

	driver := &scene.Driver{
		Client:   client,
		AlwaysOn: alwaysOn,
		Scenes:   buildScenes(widgets),
	}
	logStartup(driver)
	if err := driver.Run(ctx); err != nil {
		slog.Error("scene driver returned", "err", err)
	}
	return nil
}

// counter is the optional Count() interface implemented by static quote
// sources (`*quotes.Source`). Widgets that fetch from the network or
// rotate across sub-widgets don't implement it and log as "live".
type counter interface {
	Count() int
}

// logStartup reports the rotation config: one line per scene with its
// weight, share %, and entry count ("live" for HTTP-fetching widgets and
// rotators, an integer for static quote sources, "—" for scenes with no
// widget). Operators reading the daemon logs see exactly what's wired up
// without cracking open the source.
func logStartup(d *scene.Driver) {
	slog.Info("scene rotation starting", "scenes", len(d.Scenes), "duration", scene.SceneDuration)
	totalWeight := 0
	for _, s := range d.Scenes {
		totalWeight += s.Weight
	}
	for _, s := range d.Scenes {
		share := 0.0
		if totalWeight > 0 {
			share = float64(s.Weight) / float64(totalWeight) * 100
		}
		entries := "live"
		switch w := s.Widget.(type) {
		case nil:
			entries = "—"
		case counter:
			entries = strconv.Itoa(w.Count())
		default:
			_ = w
		}
		slog.Info("scene configured",
			"name", s.Name,
			"weight", s.Weight,
			"share_pct", fmt.Sprintf("%.0f", share),
			"entries", entries,
		)
	}
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
		{func() ([]byte, error) { return render.SceneBackground(render.SceneMarkets, render.FormatJPEG, now) }, bgMarkets},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneSky, render.FormatJPEG, now) }, bgSky},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneHN, render.FormatJPEG, now) }, bgHN},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneDayOfYear, render.FormatJPEG, now) }, bgDayOfYear},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneEaster, render.FormatJPEG, now) }, bgEaster},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneBabylon5, render.FormatJPEG, now) }, bgBabylon5},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneStarTrek, render.FormatJPEG, now) }, bgStarTrek},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneDiscworld, render.FormatJPEG, now) }, bgDiscworld},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneJargon, render.FormatJPEG, now) }, bgJargon},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneCatFacts, render.FormatJPEG, now) }, bgCatFacts},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneDidYouKnow, render.FormatJPEG, now) }, bgDidYouKnow},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneSunrise, render.FormatJPEG, now) }, bgSunrise},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneZenQuotes, render.FormatJPEG, now) }, bgZenQuotes},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneDevil, render.FormatJPEG, now) }, bgDevil},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneNASA, render.FormatJPEG, now) }, bgNASA},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneCocktail, render.FormatJPEG, now) }, bgCocktail},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneOnThisDay, render.FormatJPEG, now) }, bgOnThisDay},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneISS, render.FormatJPEG, now) }, bgISS},
		{func() ([]byte, error) { return render.SceneBackground(render.SceneGitHub, render.FormatJPEG, now) }, bgGitHub},
	}
	// One bg per weather outlook, each carrying the matching icon in the
	// bottom-right corner; the scene's BgPathFor picks among these at
	// activation time based on the current widget value.
	for _, o := range weatherOutlooks {
		outlook, path := o.Outlook, o.BgPath
		bgs = append(bgs, struct {
			render func() ([]byte, error)
			path   string
		}{
			render: func() ([]byte, error) {
				return render.SceneWeatherBackground(outlook, render.FormatJPEG, now)
			},
			path: path,
		})
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
