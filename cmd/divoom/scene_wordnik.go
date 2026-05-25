package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Word of the Day" — daily English vocabulary entry from
// Wordnik (WORDNIK_API_KEY) with a baked-in day-of-year
// fallback list when no key is set. Shares the manpage chassis
// with jargon (header + body + footer-row); wordnik's variant
// uses the footer slot for the IPA pronunciation instead of a
// see-also reference.
func wordnikScene(widgets map[string]widget.Widget) *scene.Scene {
	return DictionaryScene(DictionarySceneOpts{
		Name: "wordnik", Title: "Word of the Day", Weight: WeightEntertaining, BgPath: bgWordnik,
		Widget: widgets["wordnik"], Style: StyleManpage,
	})
}
