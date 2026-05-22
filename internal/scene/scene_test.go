package scene

import (
	"context"
	"errors"
	"math"
	"math/rand/v2"
	"testing"

	"github.com/dragonpaw/divoom/internal/frame"
)

// stubWidget is a synthetic widget for Driver tests: returns a fixed
// string, a fixed error, or panics — whichever the test asked for.
type stubWidget struct {
	name  string
	text  string
	err   error
	panic any
}

func (w *stubWidget) Name() string { return w.name }
func (w *stubWidget) Fetch(ctx context.Context) (string, error) {
	if w.panic != nil {
		panic(w.panic)
	}
	if w.err != nil {
		return "", w.err
	}
	return w.text, nil
}

// newTestScene builds a scene that's already healthy (Refresh succeeds
// on the stub widget) and has `n` placeholder elements so the picker's
// same-count exclusion can distinguish it from siblings of other sizes.
func newTestScene(t *testing.T, name string, weight int, elementCount int) *Scene {
	t.Helper()
	s := &Scene{
		Name:     name,
		Weight:   weight,
		Elements: make([]frame.DispElement, elementCount),
		Widget:   &stubWidget{name: name, text: name + "-text"},
	}
	s.Refresh(context.Background())
	if !s.isHealthy() {
		t.Fatalf("test scene %q failed to warm up", name)
	}
	return s
}

// newDriver gives every test a deterministic rng so weighted-pick is
// reproducible. The seed is fixed; tests that need a different seed
// override d.rng themselves.
func newDriver(scenes ...*Scene) *Driver {
	return &Driver{
		Scenes: scenes,
		rng:    rand.New(rand.NewPCG(1, 2)),
	}
}

func TestPickExcludesSameCountAsLast(t *testing.T) {
	// Two scenes share element count 3; one has count 5. After showing
	// either count-3 scene, pick must return the count-5 scene — never
	// the other count-3 (would leave the device's geometry cache stale).
	cases := []struct {
		name string
		last *Scene
	}{
		{"last is count-3 A", nil}, // filled in below
		{"last is count-3 B", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := newTestScene(t, "a3", 1, 3)
			b := newTestScene(t, "b3", 1, 3)
			c := newTestScene(t, "c5", 1, 5)
			d := newDriver(a, b, c)
			last := a
			if tc.name == "last is count-3 B" {
				last = b
			}
			for i := 0; i < 200; i++ {
				got := d.pick(last)
				if got == last {
					t.Fatalf("pick returned `last` (%s)", got.Name)
				}
				if len(got.Elements) == len(last.Elements) {
					t.Fatalf("pick returned same-count scene %s (count %d) after %s",
						got.Name, len(got.Elements), last.Name)
				}
			}
		})
	}
}

func TestPickWeightedDistribution(t *testing.T) {
	// Scenes have different element counts so same-count exclusion is
	// never the reason a scene is skipped — the only filter at play is
	// the weighted random pick (and the "never twice in a row" rule,
	// which we factor out by *not* passing `last`).
	rare := &Scene{Name: "rare", Weight: 1, Elements: make([]frame.DispElement, 2),
		Widget: &stubWidget{name: "rare", text: "x"}}
	mid := &Scene{Name: "mid", Weight: 3, Elements: make([]frame.DispElement, 3),
		Widget: &stubWidget{name: "mid", text: "x"}}
	common := &Scene{Name: "common", Weight: 6, Elements: make([]frame.DispElement, 4),
		Widget: &stubWidget{name: "common", text: "x"}}
	for _, s := range []*Scene{rare, mid, common} {
		s.Refresh(context.Background())
	}
	d := newDriver(rare, mid, common)

	const iters = 10000
	counts := map[string]int{}
	for i := 0; i < iters; i++ {
		s := d.pick(nil)
		counts[s.Name]++
	}

	totalWeight := float64(rare.Weight + mid.Weight + common.Weight)
	want := map[string]float64{
		"rare":   float64(rare.Weight) / totalWeight,
		"mid":    float64(mid.Weight) / totalWeight,
		"common": float64(common.Weight) / totalWeight,
	}
	// 3% absolute tolerance over 10k draws is comfortably above the
	// expected stddev (~0.5% for p=0.1) and tight enough to catch a
	// real weight bug.
	const tol = 0.03
	for name, frac := range want {
		got := float64(counts[name]) / float64(iters)
		if math.Abs(got-frac) > tol {
			t.Errorf("scene %q: got freq %.3f, want ~%.3f (tol %.2f)", name, got, frac, tol)
		}
	}
}

func TestPickSkipsUnhealthyScene(t *testing.T) {
	// A scene whose widget always errors stays unhealthy and must never
	// be picked while a healthy alternative exists.
	bad := &Scene{
		Name:     "bad",
		Weight:   100, // huge weight; if eligibility ignored it'd dominate
		Elements: make([]frame.DispElement, 2),
		Widget:   &stubWidget{name: "bad", err: errors.New("nope")},
	}
	bad.Refresh(context.Background())
	if bad.isHealthy() {
		t.Fatalf("bad scene should be unhealthy after failed Refresh")
	}
	good := newTestScene(t, "good", 1, 3)
	d := newDriver(bad, good)

	for i := 0; i < 100; i++ {
		got := d.pick(nil)
		if got == bad {
			t.Fatalf("pick returned unhealthy scene")
		}
		if got != good {
			t.Fatalf("pick returned unexpected scene %s", got.Name)
		}
	}
}

func TestRefreshRecoversFromPanickingWidget(t *testing.T) {
	// A widget that panics must not crash the Refresh goroutine. The
	// scene flips to unhealthy and the picker skips it — rotation
	// continues on the healthy sibling.
	boom := &Scene{
		Name:     "boom",
		Weight:   100,
		Elements: make([]frame.DispElement, 2),
		Widget:   &stubWidget{name: "boom", panic: "kaboom"},
	}
	// Must not panic out of the test goroutine.
	boom.Refresh(context.Background())
	if boom.isHealthy() {
		t.Fatalf("panicking widget should leave scene unhealthy")
	}

	good := newTestScene(t, "good", 1, 3)
	d := newDriver(boom, good)
	for i := 0; i < 50; i++ {
		got := d.pick(nil)
		if got == boom {
			t.Fatalf("pick returned scene whose widget panics")
		}
	}
}

func TestRefreshRecoveryRestoresHealth(t *testing.T) {
	// After a transient error, a subsequent successful Refresh must
	// flip the scene back to healthy so the picker re-includes it.
	w := &stubWidget{name: "flaky", err: errors.New("down")}
	s := &Scene{
		Name:     "flaky",
		Weight:   1,
		Elements: make([]frame.DispElement, 2),
		Widget:   w,
	}
	s.Refresh(context.Background())
	if s.isHealthy() {
		t.Fatalf("scene should start unhealthy after first failed Refresh")
	}
	w.err = nil
	w.text = "ok"
	s.Refresh(context.Background())
	if !s.isHealthy() {
		t.Fatalf("scene should be healthy after successful Refresh")
	}
	if s.latest() != "ok" {
		t.Fatalf("cached text = %q, want %q", s.latest(), "ok")
	}
}
