package render

import (
	"math"
	"testing"
	"time"
)

// TestRenderMoonDisc renders each of the 14 pre-rendered moonphase
// variants and verifies each one is a non-trivial JPEG (>5KB), which
// catches a render path that silently emits an empty / single-colour
// canvas. Visual correctness (waxing on the right, waning on the left)
// is checked manually via `divoom render`.
func TestRenderMoonDisc(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	for i := 0; i < MoonPhaseVariants; i++ {
		data, err := SceneMoonphaseBackground(i, FormatJPEG, now)
		if err != nil {
			t.Fatalf("SceneMoonphaseBackground(%d): %v", i, err)
		}
		if len(data) < 5*1024 {
			t.Errorf("SceneMoonphaseBackground(%d): JPEG too small (%d bytes)", i, len(data))
		}
	}
}

// TestMoonIllumFractionForIndex pins the lit-fraction sample at the
// cycle anchors (new, first quarter, full, last quarter) so the disc
// carve geometry can't silently drift.
func TestMoonIllumFractionForIndex(t *testing.T) {
	cases := []struct {
		idx       int
		wantFrac  float64
		tolerance float64
	}{
		{0, 0.0, 0.001},   // new
		{7, 1.0, 0.001},   // full
		{4, 0.611, 0.001}, // late-waxing, past first quarter
		{10, 0.611, 0.001}, // mirror on the waning side
		{11, 0.389, 0.001}, // waning, past last quarter
		{3, 0.389, 0.001},  // mirror on the waxing side
	}
	for _, tc := range cases {
		got := MoonIllumFractionForIndex(tc.idx)
		if math.Abs(got-tc.wantFrac) > tc.tolerance {
			t.Errorf("MoonIllumFractionForIndex(%d) = %.3f, want %.3f ± %.3f",
				tc.idx, got, tc.wantFrac, tc.tolerance)
		}
	}
}

