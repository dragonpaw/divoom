package quotes

import (
	"strings"
	"testing"
)

func TestTwainHasKnownQuote(t *testing.T) {
	src := NewTwain()
	if src.Count() == 0 {
		t.Fatal("twain: empty quote list")
	}
	const needle = "If you tell the truth, you don't have to remember anything"
	found := false
	for _, q := range twain {
		if strings.Contains(q, needle) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("twain: missing canonical Twain quote %q", needle)
	}
}
