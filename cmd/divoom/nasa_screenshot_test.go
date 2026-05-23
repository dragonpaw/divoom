package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/dragonpaw/divoom/internal/render"
)

// TestRebakeNASAScreenshot regenerates docs/screenshots/nasa.jpg from
// a cached APOD (2022-07-12, Noctilucent Clouds over Paris) so it
// picks up the current scene chrome — notably the baked
// "astronomy picture of the day" title that was added after the
// previous nasa.jpg was composited. Skip-gated on DIVOOM_PREVIEW so
// CI runs don't depend on a populated APOD cache:
//
//	DIVOOM_PREVIEW=1 go test ./cmd/divoom -run RebakeNASAScreenshot -v
func TestRebakeNASAScreenshot(t *testing.T) {
	if os.Getenv("DIVOOM_PREVIEW") == "" {
		t.Skip("set DIVOOM_PREVIEW=1 to enable")
	}
	const fixedTime = "2026-05-27T12:34:00-07:00"
	now, err := time.Parse(time.RFC3339, fixedTime)
	if err != nil {
		t.Fatal(err)
	}

	// Bake the APOD composite using the cached photo + the current
	// (re-titled) bg.
	apodKey := os.Getenv("NASA_API_KEY")
	if apodKey == "" {
		apodKey = "DEMO_KEY"
	}
	baked, err := bakeOneNASAImage(context.Background(), apodKey, "2022-07-12")
	if err != nil {
		t.Fatalf("bake nasa 2022-07-12: %v", err)
	}

	// Overlay the always-on header at the canonical fixed time.
	withHeader, err := render.BakeAlwaysOnHeaderJPEG(baked, now)
	if err != nil {
		t.Fatalf("bake header: %v", err)
	}
	if err := os.WriteFile("../../docs/screenshots/nasa.jpg", withHeader, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote nasa.jpg (%d bytes)", len(withHeader))
}
