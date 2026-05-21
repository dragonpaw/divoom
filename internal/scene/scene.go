// Package scene rotates "scenes" on the Times Frame — different bottom-area
// layouts that swap every N seconds, sharing a common always-on top
// (time + date). Each scene reads its widgets' current values at activation
// and installs them as a single new layout.
package scene

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/widget"
)

// Mount binds a widget runner to a specific Text-element ID within a scene.
// On scene activation, the runner's Latest() value is read, optionally
// passed through Format (which can split one runner's output across
// several elements AND override per-element FontColor based on the data),
// and baked into the element.
//
// Format(raw) returns (text, color):
//   - text "" → element shows "—"
//   - color "" → keep the element's declared FontColor
type Mount struct {
	ID     int
	Runner *widget.Runner
	Format func(raw string) (text, color string)
}

// Scene is one rotating layout. Elements are the bottom-area DispList
// entries (positions, fonts, colors); Mounts pair each Text element ID
// with the widget that supplies its content. BgPath is the on-device
// path to this scene's background image (per-scene so each scene can
// have its own glyph).
type Scene struct {
	Name     string
	Duration time.Duration
	BgPath   string
	Elements []frame.DispElement
	Mounts   []Mount
}

// Driver owns the LAN client and the always-on top elements. It rotates
// through scenes forever, installing each as a fresh layout (with its
// own BgPath) when its turn comes up.
type Driver struct {
	Client   *frame.Client
	AlwaysOn []frame.DispElement
	Scenes   []Scene
}

// Run rotates through scenes forever (until ctx cancelled), holding each
// for its declared duration. Before the first scene installs, waits for
// every mounted widget runner to complete its first refresh so scenes
// never show "—" placeholders during the warmup window.
func (d *Driver) Run(ctx context.Context) error {
	if len(d.Scenes) == 0 {
		return fmt.Errorf("scene driver: no scenes")
	}

	if err := d.warmupRunners(ctx); err != nil && ctx.Err() == nil {
		// Only the parent ctx being cancelled is fatal; per-runner
		// timeouts are logged inside warmupRunners and we proceed.
		return err
	}

	for {
		for _, s := range d.Scenes {
			if err := d.show(ctx, s); err != nil {
				slog.Error("scene install failed", "scene", s.Name, "err", err)
			} else {
				slog.Info("scene active", "scene", s.Name, "duration", s.Duration)
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(s.Duration):
			}
		}
	}
}

// warmupRunners waits up to 30s for each unique runner referenced by the
// scenes to complete its first refresh. A timeout on an individual runner
// is logged but not fatal — better to start rotation with a placeholder
// than to hang forever on a broken upstream.
func (d *Driver) warmupRunners(ctx context.Context) error {
	seen := map[*widget.Runner]bool{}
	var runners []*widget.Runner
	for _, s := range d.Scenes {
		for _, m := range s.Mounts {
			if !seen[m.Runner] {
				seen[m.Runner] = true
				runners = append(runners, m.Runner)
			}
		}
	}
	waitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for _, r := range runners {
		if err := r.WaitFirstFetch(waitCtx); err != nil {
			slog.Warn("widget warmup timed out; proceeding anyway", "widget", r.Name(), "err", err)
		}
	}
	return nil
}

func (d *Driver) show(ctx context.Context, s Scene) error {
	// Bake current widget values into the bottom elements. We can't patch
	// Image elements after install, but baking Text values up-front means
	// EnterCustomControlMode is the *only* API call per scene change — no
	// follow-up UpdateDisplayItems needed.
	bottom := make([]frame.DispElement, len(s.Elements))
	copy(bottom, s.Elements)
	for _, m := range s.Mounts {
		for i, e := range bottom {
			if e.ID != m.ID {
				continue
			}
			text := m.Runner.Latest()
			color := ""
			if m.Format != nil {
				text, color = m.Format(text)
			}
			if text == "" {
				text = "—"
			}
			bottom[i].TextMessage = text
			if color != "" {
				bottom[i].FontColor = color
			}
			break
		}
	}

	elements := make([]frame.DispElement, 0, len(d.AlwaysOn)+len(bottom))
	elements = append(elements, d.AlwaysOn...)
	elements = append(elements, bottom...)

	layout := frame.CustomMode{
		BackgroundImageLocalFlag: 1,
		BackgroundImageAddr:      s.BgPath,
		DispList:                 elements,
	}

	installCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return d.Client.EnterCustomMode(installCtx, layout)
}
