package main

import (
	"fmt"
	"time"

	"github.com/dragonpaw/divoom/internal/render"
)

// Quote-family redesign — three baked-chrome layouts shared by the ten
// quote/dictionary scenes (Babylon 5, Star Trek, Discworld, Stoics, Twain,
// ZenQuotes, Jargon, Wordnik, Devil's Dictionary, fortune). The chrome
// (in-universe header / book-page imprint / shell prompt + status bar) is
// rasterised straight into the bg JPG at push time so the device's
// 6-element DispList cap stays free for the dynamic body / author text.
//
// This file is the single registry of which scene belongs to which family
// and what strings its chrome carries. Both the bg renderer (serve.go +
// render.go) and the device-layout builders (scenes.go) read from here so
// the chrome on the JPG and the geometry of the live elements stay in
// sync.

// QuoteFamily re-exports render.QuoteFamily so the per-scene files can
// reference it without importing the render package. The numeric value
// is the same; only the spelling lives in two packages.
type QuoteFamily = render.QuoteFamily

const (
	FamilyMarginalia = render.FamilyMarginalia
	FamilyFromSource = render.FamilyFromSource
	FamilyTerminal   = render.FamilyTerminal
)

// quoteSceneRegistry lists every quote/dictionary scene that participates
// in the three-family redesign, in the same order they appear in the
// rotation list. Each entry pairs the scene's name (the same string each
// per-scene .go file uses) with its render.Scene constant (for glyph
// dispatch) and a builder for its FamilyChrome.
//
// chromeFor is a function rather than a fixed struct so dynamic chrome
// (Star Trek's stardate) can be computed at push time. now is the same
// value the surrounding hero frame uses; deterministic per-push.
var quoteSceneRegistry = []struct {
	Name      string
	Scene     render.Scene
	BgPath    string
	ChromeFor func(now time.Time) render.FamilyChrome
}{
	{
		Name: "babylon5", Scene: render.SceneBabylon5, BgPath: bgBabylon5,
		ChromeFor: func(now time.Time) render.FamilyChrome {
			return render.FamilyChrome{
				Family:    FamilyFromSource,
				Header:    "EARTHFORCE TRANSMISSION",
				Subheader: "priority 3",
			}
		},
	},
	{
		Name: "startrek", Scene: render.SceneStarTrek, BgPath: bgStarTrek,
		ChromeFor: func(now time.Time) render.FamilyChrome {
			return render.FamilyChrome{
				Family:    FamilyFromSource,
				Header:    "STARDATE " + tngStardate(now),
				Subheader: "PERSONAL LOG",
			}
		},
	},
	{
		Name: "discworld", Scene: render.SceneDiscworld, BgPath: bgDiscworld,
		ChromeFor: func(now time.Time) render.FamilyChrome {
			return render.FamilyChrome{
				Family: FamilyFromSource,
				Header: "Discworld Press — first edition",
			}
		},
	},
	{
		Name: "stoics", Scene: render.SceneStoics, BgPath: bgStoics,
		ChromeFor: func(now time.Time) render.FamilyChrome {
			return render.FamilyChrome{
				Family:       FamilyMarginalia,
				BookName:     "Meditations",
				DropCap:      "M",
				DropCapColor: cGreen,
			}
		},
	},
	{
		Name: "twain", Scene: render.SceneTwain, BgPath: bgTwain,
		ChromeFor: func(now time.Time) render.FamilyChrome {
			return render.FamilyChrome{
				Family:       FamilyMarginalia,
				BookName:     "S. L. Clemens — collected",
				DropCap:      "T",
				DropCapColor: cFgDark,
			}
		},
	},
	{
		Name: "zenquotes", Scene: render.SceneZenQuotes, BgPath: bgZenQuotes,
		ChromeFor: func(now time.Time) render.FamilyChrome {
			return render.FamilyChrome{
				Family:       FamilyMarginalia,
				DropCap:      "Z",
				DropCapColor: cBlue,
			}
		},
	},
	{
		Name: "jargon", Scene: render.SceneJargon, BgPath: bgJargon,
		ChromeFor: func(now time.Time) render.FamilyChrome {
			return render.FamilyChrome{
				Family:       FamilyTerminal,
				ShellPrompt:  "$ jargon",
				SourceFooter: "source: catb.org/jargon v4.4.7",
				AuthorFooter: "author: Eric S. Raymond et al",
			}
		},
	},
	{
		Name: "wordnik", Scene: render.SceneWordnik, BgPath: bgWordnik,
		ChromeFor: func(now time.Time) render.FamilyChrome {
			return render.FamilyChrome{
				Family:       FamilyTerminal,
				ShellPrompt:  "$ wotd " + now.Format("2006-01-02"),
				SourceFooter: "source: wordnik.com",
				AuthorFooter: "",
			}
		},
	},
	{
		Name: "devil", Scene: render.SceneDevil, BgPath: bgDevil,
		ChromeFor: func(now time.Time) render.FamilyChrome {
			return render.FamilyChrome{
				Family:             FamilyTerminal,
				ShellPrompt:        "$ define",
				SourceFooter:       "source: The Devil's Dictionary (1906)",
				AuthorFooter:       "author: Ambrose Bierce",
				PunchlineOrnaments: true,
			}
		},
	},
	{
		Name: "fortune", Scene: render.SceneFortune, BgPath: bgFortune,
		ChromeFor: func(now time.Time) render.FamilyChrome {
			return render.FamilyChrome{
				Family:       FamilyTerminal,
				ShellPrompt:  "$ fortune -s",
				SourceFooter: "source: /usr/share/games/fortunes",
				AuthorFooter: "author: anonymous",
			}
		},
	},
}

// quoteFamilyChromeByName returns the FamilyChrome for a scene name. Used
// by the per-scene builders to consult the registry without duplicating
// the per-scene strings in two places.
func quoteFamilyChromeByName(name string, now time.Time) render.FamilyChrome {
	for _, e := range quoteSceneRegistry {
		if e.Name == name {
			return e.ChromeFor(now)
		}
	}
	return render.FamilyChrome{Family: FamilyMarginalia}
}

// tngStardate returns a TNG-shaped stardate string for now, rounded to
// one decimal. Formula: year*1000 + (yearDay/yearLen)*1000. Not
// canonical — Star Trek's stardate scheme isn't real math — but the
// output reads as "looks plausible" on the in-universe header strip
// (5-digit integer + 1 decimal). Computed at push time so the value is
// fixed until the next `divoom push`; nobody's watching a clock here.
func tngStardate(now time.Time) string {
	year := now.Year()
	yearLen := 365.0
	if isLeapYear(year) {
		yearLen = 366.0
	}
	frac := float64(now.YearDay()-1) / yearLen
	sd := float64(year)*1000 + frac*1000
	return fmt.Sprintf("%.1f", sd)
}

// isLeapYear mirrors render.isLeapYear (kept private there). One-line
// helper — duplicating it here avoids exposing a public API just to
// format a stardate.
func isLeapYear(y int) bool {
	return (y%4 == 0 && y%100 != 0) || y%400 == 0
}
