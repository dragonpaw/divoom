package render

import (
	"bytes"
	"testing"
	"time"
)

// TestGenartDeterminism — same date → identical bytes.
func TestGenartDeterminism(t *testing.T) {
	day := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	a, err := GenartBackground(day, FormatPNG)
	if err != nil {
		t.Fatalf("first render: %v", err)
	}
	b, err := GenartBackground(day, FormatPNG)
	if err != nil {
		t.Fatalf("second render: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Errorf("same-date renders differ (%d vs %d bytes)", len(a), len(b))
	}
}

// TestGenartRotation — different dates → different bytes, and the
// algorithm pick is itself deterministic per date.
func TestGenartRotation(t *testing.T) {
	d1 := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	a, _ := GenartBackground(d1, FormatPNG)
	b, _ := GenartBackground(d2, FormatPNG)
	if bytes.Equal(a, b) {
		t.Errorf("dates a week apart produced identical bytes")
	}
	// Algorithm pick must be stable across calls.
	_, n1a := GenartForDate(d1)
	_, n1b := GenartForDate(d1)
	if n1a != n1b {
		t.Errorf("algorithm pick not deterministic for %v: %s vs %s", d1, n1a, n1b)
	}
}
