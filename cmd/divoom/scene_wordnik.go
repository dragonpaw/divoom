package main

import (
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "Word of the Day" — daily English vocabulary entry from
// Wordnik (WORDNIK_API_KEY) with a baked-in day-of-year
// fallback list when no key is set. Same dictionary layout
// as Jargon / Devil's; purple headword to stay distinct from
// jargon yellow and devil red.
func wordnikScene(widgets map[string]widget.Widget) *scene.Scene {
	return DictionaryScene(DictionarySceneOpts{
		Name: "wordnik", Title: "Word of the Day", Weight: 20, BgPath: bgWordnik,
		Widget:  widgets["wordnik"],
		Tagline: "wordnik.com",
	})
}
