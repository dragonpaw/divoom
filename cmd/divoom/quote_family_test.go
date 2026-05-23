package main

import (
	"strconv"
	"testing"
	"time"
)

// TestTNGStardate pins a couple of known dates so the stardate generator
// (year×1000 + dayOfYear-fractional × 1000) doesn't silently drift when
// the formula or the leap-year helper changes. Exact values aren't
// canonical — Star Trek's stardate scheme isn't real math — but the
// monotonic property "later date → higher stardate within the same year"
// must hold.
func TestTNGStardate(t *testing.T) {
	cases := []struct {
		when time.Time
		want string
	}{
		// 2026-01-01 — start of year, frac=0, sd = 2026*1000 = 2026000.0.
		{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), "2026000.0"},
		// 2027-01-01 — exactly one year later: sd = 2027*1000 = 2027000.0.
		{time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), "2027000.0"},
	}
	for _, tc := range cases {
		if got := tngStardate(tc.when); got != tc.want {
			t.Errorf("tngStardate(%s) = %q, want %q",
				tc.when.Format("2006-01-02"), got, tc.want)
		}
	}

	// Monotonicity within a year: a later date in 2026 must produce a
	// strictly greater stardate string when ordered lexicographically by
	// the integer part (which dominates).
	earlier, _ := strconv.ParseFloat(tngStardate(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)), 64)
	later, _ := strconv.ParseFloat(tngStardate(time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)), 64)
	if earlier >= later {
		t.Errorf("stardate non-monotonic: %.1f (Jan) >= %.1f (Nov)", earlier, later)
	}
}

// TestQuoteSceneFamilyDispatch sanity-checks that each QuoteFamily
// produces a scene with the expected element count, so a future change
// to the layout doesn't silently delete the attribution row or merge
// two families' shapes by accident. Driver.pick() relies on differing
// element counts between consecutive scenes, so this also guards that
// invariant.
func TestQuoteSceneFamilyDispatch(t *testing.T) {
	cases := []struct {
		name        string
		family      QuoteFamily
		hasAuthor   bool
		hasTagline  bool
		wantElems   int
	}{
		// FromSource: body (+ author if present, + tagline if present).
		{"fromsource bare", FamilyFromSource, false, false, 1},
		{"fromsource +author", FamilyFromSource, true, false, 2},
		{"fromsource +author +tag", FamilyFromSource, true, true, 3},

		// Marginalia: body + dynamic drop-cap (+ optional author + tagline).
		{"marginalia bare", FamilyMarginalia, false, false, 2},
		{"marginalia +author", FamilyMarginalia, true, false, 3},
		{"marginalia +author +tag", FamilyMarginalia, true, true, 4},

		// Terminal: body only — chrome (status bar) supplies source/author.
		{"terminal", FamilyTerminal, true, true, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := QuoteSceneOpts{
				Name: "t", Title: "T", Weight: 1, BgPath: "/x",
				Family:    tc.family,
				HasAuthor: tc.hasAuthor,
			}
			if tc.hasTagline {
				opts.Tagline = "test"
				opts.TaglineColor = cFg
			}
			s := QuoteScene(opts)
			if got := len(s.Elements); got != tc.wantElems {
				t.Errorf("element count = %d, want %d", got, tc.wantElems)
			}
		})
	}
}

// TestDictionarySceneTerminal: all dictionary scenes are terminal-family
// and must produce exactly three Text elements (headword, POS,
// definition). The status bar is baked into the bg.
func TestDictionarySceneTerminal(t *testing.T) {
	s := DictionaryScene(DictionarySceneOpts{
		Name: "jargon", Title: "Jargon", Weight: 1, BgPath: "/x",
	})
	if got, want := len(s.Elements), 3; got != want {
		t.Errorf("element count = %d, want %d", got, want)
	}
}

// TestMarginaliaDropCap covers the dynamic drop-cap formatter: it must
// return the first non-whitespace rune of the quote body, upper-cased,
// and "" for an empty body so the AllowEmpty mount hides the element.
// Source: "Source|body|author" pipe-shaped widget output.
func TestMarginaliaDropCap(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"happy path", "stoics|the universe is change|Marcus", "T"},
		{"lowercase body", "stoics|wisdom begins in wonder|Socrates", "W"},
		{"unicode first char", "stoics|Épictète was a slave|Epictetus", "É"},
		{"empty body", "stoics||Unknown", ""},
		{"leading whitespace", "stoics|   apple|none", "A"},
		{"missing body segment", "stoics", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := marginaliaDropCap(tc.raw)
			if got != tc.want {
				t.Errorf("marginaliaDropCap(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// TestQuoteSceneRegistryComplete: every scene name listed in the
// rotation's quote group must have a corresponding registry entry, or
// pushSceneBackgrounds will fail to bake its chrome. Catches drift if
// somebody adds a scene to scenes.go but forgets quote_family.go.
func TestQuoteSceneRegistryComplete(t *testing.T) {
	want := []string{
		"babylon5", "startrek", "discworld",
		"stoics", "twain", "zenquotes",
		"jargon", "wordnik", "devil", "fortune",
	}
	have := map[string]bool{}
	for _, e := range quoteSceneRegistry {
		have[e.Name] = true
	}
	for _, name := range want {
		if !have[name] {
			t.Errorf("quoteSceneRegistry missing %q", name)
		}
	}
}
