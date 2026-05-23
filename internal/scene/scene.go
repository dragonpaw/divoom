// Package scene rotates "scenes" on the Times Frame — different bottom-area
// layouts that swap every N seconds, sharing a common always-on top
// (day + time + date). Each scene owns one Widget that supplies its dynamic
// text; the value is fetched on unload so it's ready for the next activation
// without blocking the transition.
package scene

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/widget"
)

// Mount maps a Text element ID to a Format function that derives its
// rendered text (and optionally an override FontColor) from the scene's
// widget output.
//
// Format(raw) returns (text, color):
//   - text "" → element shows "—" (unless AllowEmpty is true, then blank)
//   - color "" → keep the element's declared FontColor
//
// AllowEmpty: set on elements that *should* render blank when the
// format function legitimately produces no text — e.g. an author block
// for an unattributed quote.
type Mount struct {
	ID         int
	Format     func(raw string) (text, color string)
	AllowEmpty bool
	// Geometry, if set, gets called after the element's text is resolved
	// and may return an element with adjusted position/size (e.g. to
	// vertically centre a short body within its declared track). The
	// callback receives the rendered text and the element's current
	// geometry, and returns the (possibly adjusted) element.
	Geometry func(text string, e frame.DispElement) frame.DispElement
}

// SceneDuration is how long each scene holds the screen before the
// driver picks the next one. The cadence used to be per-scene but
// every scene now lives at the same rate so the variable was deleted.
const SceneDuration = 3 * time.Minute

// Scene is one rotating layout. Elements are the bottom-area DispList
// entries (positions, fonts, colors); Widget supplies the raw string that
// each Mount slices into a specific element. BgPath is the on-device path
// to this scene's background image. A nil Widget means the scene has no
// dynamic content (all elements render their declared TextMessage as-is).
//
// Weight controls how often this scene gets picked relative to its
// siblings (weighted-random selection in Driver.Run). Higher = more
// frequent. A scene with weight 0 is never picked. Use the same scale
// across scenes (small ints) for readability.
type Scene struct {
	Name   string
	Weight int
	BgPath string
	// BgPathFor, when set, is consulted at activation time with the
	// scene's cached widget text and may return a per-text background
	// path (e.g. one bg per weather outlook). An empty return value
	// falls back to BgPath. Optional; nil keeps the static BgPath.
	BgPathFor func(text string) string
	Elements  []frame.DispElement
	Widget    widget.Widget
	Mounts    []Mount

	// OnActivate, if set, runs at every scene activation after Mounts
	// have populated text/colour on the bottom slice. It receives the
	// current wall-clock and the scene's raw cached widget text, and may
	// mutate the elements slice in place (e.g. recompute an element's
	// StartX from `now`). Use only for state that must be fresh per
	// activation; per-text geometry belongs in Mount.Geometry. The
	// sunrise scene uses this to position its current-time tick along
	// the baked day-arc.
	OnActivate func(now time.Time, raw string, elements []frame.DispElement)

	mu       sync.RWMutex
	cached   string
	healthy  bool // true until first failure; flips back on next success
	wasReady bool // true once Refresh has succeeded at least once
}

// Refresh fetches the scene's widget and stores the result in the local
// cache. Called by the Driver on scene unload (and periodically by the
// retry loop for scenes currently marked unhealthy). On error the prior
// cached value is left in place and the scene is marked unhealthy so
// the picker skips it until a future Refresh succeeds. A scene that
// has never succeeded (wasReady == false) is also treated as
// unhealthy; once any Refresh succeeds, the scene becomes healthy and
// stays healthy until a subsequent failure.
func (s *Scene) Refresh(ctx context.Context) {
	if s.Widget == nil {
		// Static-content scenes have no widget; they're always healthy
		// and ready.
		s.mu.Lock()
		s.healthy = true
		s.wasReady = true
		s.mu.Unlock()
		return
	}
	fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	text, err := s.fetchWithRecover(fetchCtx)
	if err != nil {
		s.mu.Lock()
		wasHealthy := s.healthy
		s.healthy = false
		s.mu.Unlock()
		if wasHealthy {
			slog.Warn("scene marked unhealthy", "scene", s.Name, "widget", s.Widget.Name(), "err", err)
		} else {
			slog.Debug("scene still unhealthy", "scene", s.Name, "widget", s.Widget.Name(), "err", err)
		}
		return
	}
	s.mu.Lock()
	wasHealthy := s.healthy
	s.cached = text
	s.healthy = true
	s.wasReady = true
	s.mu.Unlock()
	if !wasHealthy {
		slog.Info("scene recovered", "scene", s.Name, "widget", s.Widget.Name())
	}
}

// fetchWithRecover invokes the widget's Fetch and converts a panic
// into an ordinary error so one buggy widget can't kill the rotation
// goroutine. The scene then takes the unhealthy path and the picker
// skips it until a future Refresh succeeds.
func (s *Scene) fetchWithRecover(ctx context.Context) (text string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("widget panic: %v", r)
		}
	}()
	return s.Widget.Fetch(ctx)
}

// isHealthy reports whether this scene is currently safe to show. A
// scene is healthy if it has ever Refresh'd successfully AND the most
// recent Refresh did not fail.
func (s *Scene) isHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.healthy && s.wasReady
}

func (s *Scene) latest() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cached
}

// Driver owns the LAN client and the always-on top elements. It rotates
// through scenes forever, installing each as a fresh layout (with its
// own BgPath) when its turn comes up. AlwaysOn is a function so the
// caller can fold time-of-day-dependent color choices (AM/PM, day of
// week) into the elements at each install.
//
// installSeq is bumped per scene install and folded into every element
// ID we send, so each install uses element IDs the device has never
// seen before. That sidesteps the per-ID property cache (see
// docs/api.md → ID caching) without needing the dial-flashing
// ExitCustomMode round-trip.
type Driver struct {
	Client   *frame.Client
	AlwaysOn func(now time.Time) []frame.DispElement
	Scenes   []*Scene

	installSeq atomic.Uint64

	pickMu sync.Mutex
	rng    *rand.Rand
}

// Run picks scenes by weighted random and holds each for its declared
// duration, forever (until ctx cancelled). Before the first activation,
// every scene's widget is fetched once so we never open on a "—"
// placeholder. Between activations, the scene that just unloaded is
// refreshed in the background so it's ready next time around.
//
// We never repeat the same scene twice in a row — variety beats the
// occasional weight-perfect distribution.
func (d *Driver) Run(ctx context.Context) error {
	if len(d.Scenes) == 0 {
		return fmt.Errorf("scene driver: no scenes")
	}
	if d.rng == nil {
		d.rng = rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0xD1AB10))
	}

	d.warmup(ctx)

	// Retry unhealthy scenes periodically so transient upstream
	// failures don't kick a scene out of rotation permanently.
	go d.retryUnhealthy(ctx)

	var last *Scene
	for {
		s := d.pick(last)
		if err := d.activate(ctx, s); err != nil {
			slog.Error("scene install failed", "scene", s.Name, "err", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(SceneDuration):
		}
		// Refresh in the background — the next scene's install must
		// not wait on this widget's network call.
		go s.Refresh(ctx)
		last = s
	}
}

// retryUnhealthy periodically re-attempts Refresh on any scene currently
// marked unhealthy, giving transient upstreams a chance to recover.
// Returns when ctx is cancelled.
func (d *Driver) retryUnhealthy(ctx context.Context) {
	const interval = 5 * time.Minute
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		for _, s := range d.Scenes {
			if s.isHealthy() {
				continue
			}
			// Sequential retries — these are slow widgets; parallel
			// would race the warmup pattern but with no upside.
			s.Refresh(ctx)
		}
	}
}

// pick returns the next scene to install via weighted random sample.
// It excludes (a) `last` so the same scene never runs twice in a row,
// and (b) any scene whose element count matches `last` — the device's
// geometry cache only invalidates when DispList length differs between
// installs, so picking a same-count scene would leave the prior
// scene's FontSize / Height in place. Scenes with Weight <= 0 are
// always skipped.
func (d *Driver) pick(last *Scene) *Scene {
	d.pickMu.Lock()
	defer d.pickMu.Unlock()

	eligible := func(s *Scene) bool {
		if s.Weight <= 0 || s == last {
			return false
		}
		if last != nil && len(s.Elements) == len(last.Elements) {
			return false
		}
		if !s.isHealthy() {
			return false
		}
		return true
	}

	total := 0
	for _, s := range d.Scenes {
		if eligible(s) {
			total += s.Weight
		}
	}
	if total == 0 {
		// Defensive fallback — every other scene is blocked (same length
		// as `last`, weight 0, or unhealthy). Prefer any healthy scene
		// that isn't `last`; otherwise tolerate one healthy scene-2x or
		// any first-scene to avoid stalling rotation.
		for _, s := range d.Scenes {
			if s.Weight > 0 && s != last && s.isHealthy() {
				return s
			}
		}
		for _, s := range d.Scenes {
			if s.Weight > 0 && s.isHealthy() {
				return s
			}
		}
		// No healthy scene at all — fall through to anything just so the
		// driver keeps cycling. Picker will reconsider on next call.
		for _, s := range d.Scenes {
			if s.Weight > 0 && s != last {
				return s
			}
		}
		return d.Scenes[0]
	}

	roll := d.rng.IntN(total)
	for _, s := range d.Scenes {
		if !eligible(s) {
			continue
		}
		roll -= s.Weight
		if roll < 0 {
			return s
		}
	}
	return d.Scenes[0] // unreachable when total > 0
}

// warmup fetches each scene's widget once before rotation begins so the
// first round of activations never renders "—". Fetches run concurrently
// — total wall-clock budget is bounded by the slowest single widget,
// not the sum. Each Refresh enforces its own internal 30s timeout; the
// outer 60s here is a hard ceiling for the whole warmup phase.
func (d *Driver) warmup(ctx context.Context) {
	warmCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for _, s := range d.Scenes {
		s := s
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Refresh(warmCtx)
		}()
	}
	wg.Wait()
}

// RenderElements applies the scene's Mounts (and OnActivate, if any) to
// the static Elements list using `raw` as the widget output and `now` as
// the wall-clock. Returns the resulting element slice without installing
// anything on a device — used by the offline screenshot baker so a scene
// can be previewed with realistic dynamic content.
func (s *Scene) RenderElements(raw string, now time.Time) []frame.DispElement {
	bottom := make([]frame.DispElement, len(s.Elements))
	copy(bottom, s.Elements)
	for _, m := range s.Mounts {
		for i, e := range bottom {
			if e.ID != m.ID {
				continue
			}
			text, color := raw, ""
			if m.Format != nil {
				text, color = m.Format(raw)
			}
			if text == "" && !m.AllowEmpty {
				text = "—"
			}
			bottom[i].TextMessage = text
			if color != "" {
				bottom[i].FontColor = color
			}
			if m.Geometry != nil {
				bottom[i] = m.Geometry(text, bottom[i])
			}
			break
		}
	}
	if s.OnActivate != nil {
		s.OnActivate(now, raw, bottom)
	}
	return bottom
}

// activate bakes the scene's cached widget value into its Text elements
// and installs the resulting layout. Logging happens here (not in
// Refresh) so the logs reflect what's actually on the wall.
func (d *Driver) activate(ctx context.Context, s *Scene) error {
	raw := s.latest()

	bottom := make([]frame.DispElement, len(s.Elements))
	copy(bottom, s.Elements)
	for _, m := range s.Mounts {
		for i, e := range bottom {
			if e.ID != m.ID {
				continue
			}
			text, color := raw, ""
			if m.Format != nil {
				text, color = m.Format(raw)
			}
			if text == "" && !m.AllowEmpty {
				text = "—"
			}
			bottom[i].TextMessage = text
			if color != "" {
				bottom[i].FontColor = color
			}
			if m.Geometry != nil {
				bottom[i] = m.Geometry(text, bottom[i])
			}
			break
		}
	}

	now := time.Now()
	if s.OnActivate != nil {
		s.OnActivate(now, raw, bottom)
	}

	top := d.AlwaysOn(now)
	elements := make([]frame.DispElement, 0, len(top)+len(bottom))
	elements = append(elements, top...)
	elements = append(elements, bottom...)

	// Fresh per-install ID offset: each element gets an ID the device
	// has never seen. Cheap, and avoids any per-ID state confusion
	// even though the property cache isn't keyed solely on ID.
	seq := d.installSeq.Add(1)
	offset := int(seq) * 100
	for i := range elements {
		elements[i].ID += offset
	}

	bg := s.BgPath
	if s.BgPathFor != nil {
		if alt := s.BgPathFor(raw); alt != "" {
			bg = alt
		}
	}
	layout := frame.CustomMode{
		BackgroundImageLocalFlag: 1,
		BackgroundImageAddr:      bg,
		DispList:                 elements,
	}

	// No ExitCustomMode here: the driver's pick() guarantees consecutive
	// scenes have different element counts, and a different DispList
	// length is empirically enough to bust the device's geometry cache
	// without the 1s preset-dial flash that Exit causes.
	installCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := d.Client.EnterCustomMode(installCtx, layout); err != nil {
		return err
	}
	slog.Info("scene active", "scene", s.Name, "duration", SceneDuration, "text", raw)
	return nil
}
