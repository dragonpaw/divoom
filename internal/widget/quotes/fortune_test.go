package quotes

import "testing"

func TestFortuneIsPopulated(t *testing.T) {
	// fortune.go is generator-output, so the precise contents shift each
	// time the script runs against an updated upstream corpus. Assert
	// the slice is non-empty and the constructor returns a usable Source
	// — that catches the empty-generation regression without freezing a
	// specific cookie that may not survive the next regeneration.
	src := NewFortune()
	if src.Count() < 100 {
		t.Fatalf("fortune: too few entries (%d), regenerate via scripts/parse-fortune.py", src.Count())
	}
}
