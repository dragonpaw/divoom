package quotes

import (
	"strings"
	"testing"
)

func TestStoicsHasKnownQuote(t *testing.T) {
	// Confirm the slice loaded and a representative line is present;
	// guards against an accidental empty-list regression.
	src := NewStoics()
	if src.Count() == 0 {
		t.Fatal("stoics: empty quote list")
	}
	const needle = "Waste no more time arguing what a good man should be"
	found := false
	for _, q := range stoics {
		if strings.Contains(q, needle) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("stoics: missing canonical Marcus Aurelius quote %q", needle)
	}
}
