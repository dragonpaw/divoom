package quotes

import "testing"

// TestBabylon5MinimumCount guards against accidental truncation of the
// curated quote list. The BU source yields ~80 single-speaker quotes that fit
// the device's display, so anything well below that floor signals breakage.
func TestBabylon5MinimumCount(t *testing.T) {
	src := NewBabylon5()
	if got := src.Count(); got < 80 {
		t.Errorf("babylon5: only %d quotes, expected at least 80", got)
	}
}
