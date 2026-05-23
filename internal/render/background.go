// Package render builds the static and semi-static images we ship to the
// Times Frame as background / Image elements.
package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"log/slog"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// Canvas dimensions are fixed by the device: backgrounds MUST be 800x1280.
const (
	CanvasW = 800
	CanvasH = 1280
)

// Gruvbox dark hard palette. We anchor everything to these.
var (
	GruvBgHard   = color.RGBA{0x1d, 0x20, 0x21, 0xff}
	GruvBgDarker = color.RGBA{0x3c, 0x38, 0x36, 0xff}
	GruvFgDark   = color.RGBA{0xa8, 0x99, 0x84, 0xff}
	GruvFg       = color.RGBA{0xeb, 0xdb, 0xb2, 0xff}
	GruvRed      = color.RGBA{0xfb, 0x49, 0x34, 0xff}
	GruvGreen    = color.RGBA{0xb8, 0xbb, 0x26, 0xff}
	GruvYellow   = color.RGBA{0xfa, 0xbd, 0x2f, 0xff}
	GruvBlue     = color.RGBA{0x83, 0xa5, 0x98, 0xff}
	GruvPurple   = color.RGBA{0xd3, 0x86, 0x9b, 0xff}
	GruvAqua     = color.RGBA{0x8e, 0xc0, 0x7b, 0xff}
	GruvOrange   = color.RGBA{0xfe, 0x80, 0x19, 0xff}
)

// Format selects an output encoding for TestBackground.
type Format int

const (
	FormatPNG Format = iota
	FormatJPEG
)

// TestBackground returns a gruvbox-bg-hard field with small fg registration
// dots in each corner, an aqua cross at the canvas midpoint, and an accent-
// color swatch band along the bottom. Use to eyeball orientation, scaling,
// and color reproduction.
func TestBackground(format Format) ([]byte, error) {
	img := buildTestImage()
	var buf bytes.Buffer
	switch format {
	case FormatPNG:
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	case FormatJPEG:
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported format %d", format)
	}
	return buf.Bytes(), nil
}

// HeroBackground returns a scene-neutral background: gruvbox bg, hairline
// divider, year-progress bar, no glyph. Useful as a preview/fallback.
func HeroBackground(format Format, now time.Time) ([]byte, error) {
	return encodeImage(buildHeroImage(now), format)
}

// Scene names one of the rotating scenes; SceneBackground draws a faint
// scene-appropriate glyph into the bottom area for ambient context.
type Scene int

const (
	SceneMarkets Scene = iota
	SceneMoonphase
	SceneHN
	SceneDayOfYear
	SceneEaster
	SceneBabylon5
	SceneStarTrek
	SceneDiscworld
	SceneJargon
	SceneCatFacts
	SceneDidYouKnow
	SceneSunrise
	SceneWeather
	SceneZenQuotes
	SceneDevil
	SceneNASA
	SceneCocktail
	SceneOnThisDay
	SceneISS
	SceneGitHub
	SceneTIL
	SceneReddit
	SceneWordnik
	SceneStoics
	SceneTwain
	SceneFortune
	SceneForecast
	SceneSeismic
)

// SceneBackground builds the hero frame and draws the scene's glyph into
// the bottom area — same gruvbox bg + divider + year-progress bar as
// HeroBackground, plus a faint shape that hints at what's playing. The
// dayofyear scene gets a thick progress bar baked into the body area;
// the easter scene gets a giant centred gruvbox-yellow egg.
func SceneBackground(scene Scene, format Format, now time.Time) ([]byte, error) {
	img := buildHeroImage(now)
	switch scene {
	case SceneDayOfYear:
		// Preview / fallback path — no special dates available without
		// env access, so the grid renders with past/today/future cells
		// only. Production callers use DayOfYearBackground.
		drawDayOfYearGrid(img, now, nil)
	case SceneEaster:
		drawEasterEgg(img)
	case SceneWeather:
		// No outlook supplied — fall back to the cloudy glyph so the
		// frame still renders. Production callers use SceneWeatherBackground.
		drawWeatherGlyph(img, "cloudy")
	case SceneMoonphase:
		// No phase index supplied — fall back to the full moon (index 7)
		// so the frame still renders. Production callers use
		// SceneMoonphaseBackground to pick the right variant.
		drawMoonDisc(img, 7)
	case SceneMarkets:
		// Markets scene chrome: a hairline under the symbol+price
		// headline, baked "1 week" / "1 month" captions under the badge
		// row, and a footer hairline. The bar-chart corner glyph stays
		// as the scene's ambient mark.
		drawSceneGlyph(img, scene)
		drawMarketsChrome(img)
		drawBakedSceneTitle(img, "markets")
	case SceneHN:
		// HN scene chrome: "HACKER NEWS" wordmark + orange rule under
		// it + dim footer rule above the metadata row. The Y glyph in
		// the bottom-right corner stays as the wordmark's mirror.
		drawHNChrome(img)
		drawSceneGlyph(img, scene)
	case SceneISS:
		// ISS scene chrome: telemetry strip, hairline, equirectangular
		// world-map outline, equator + prime-meridian hairlines. The
		// corner glyph is intentionally suppressed (see drawSceneGlyph).
		DrawISSChrome(img)
	case SceneTIL:
		// TIL scene chrome: monumental "T I L" wordmark top-left, yellow
		// rule under it, footer hairline + r/todayilearned attribution.
		// The lightbulb glyph still anchors the bottom-right corner.
		drawSceneGlyph(img, scene)
		drawTILChrome(img)
	case SceneCatFacts:
		// Field-guide entry chrome: italic-ish "Felis catus" binomial,
		// short underline, taxonomic classification, pilcrow drop-marker
		// in the body's left margin, footer hairline, and an
		// observation-number + institution line in the footer. The
		// sitting-cat silhouette in the bottom-right corner stays as
		// the plate illustration.
		drawSceneGlyph(img, scene)
		obs := rand.IntN(999) + 1
		inst := catfactsInstitutions[rand.IntN(len(catfactsInstitutions))]
		DrawCatfactsChrome(img, obs, inst)
	case SceneGitHub:
		// GitHub activity card chrome: baked "GitHub" title at the
		// canonical y=480 row, "lifetime contributions" caption under
		// the hero number, and "total PRs" / "open" / "years on
		// GitHub" column labels under the stat-tile row. Baked rather
		// than carried as device Text elements because the scene's
		// dynamic numbers (hero + three stat values) already eat 4 of
		// the 6 Text slots; the labels would push the layout over the
		// per-type cap and silently get dropped on-device.
		drawSceneGlyph(img, scene)
		drawGitHubChrome(img)
	case SceneReddit:
		// Reddit scene chrome: small upvote-arrow glyph in the bottom-
		// right corner, canonical "subreddit picks" title row at the
		// top of the body area. The dynamic "r/<sub>" accent comes in
		// as a device Text element below the post title.
		drawSceneGlyph(img, scene)
		drawBakedSceneTitle(img, "subreddit picks")
	case SceneForecast:
		drawSceneGlyph(img, scene)
		drawBakedSceneTitle(img, "next 4 days")
	case SceneSeismic:
		// No corner glyph — the seismograph trace fought the commentary
		// line for the bottom-right quadrant and carried no data of its
		// own (unlike markets' sparkline, which earns its corner by
		// charting real numbers). Identity comes from the baked title,
		// the magnitude caption, and the ruled stats strip below.
		drawBakedSceneTitle(img, "seismic activity")
		drawSeismicChrome(img)
	case SceneDidYouKnow:
		drawSceneGlyph(img, scene)
		drawBakedSceneTitle(img, "did you know?")
	case SceneOnThisDay:
		drawSceneGlyph(img, scene)
		drawBakedSceneTitle(img, "on this day")
	case SceneNASA:
		drawSceneGlyph(img, scene)
		drawNASACredit(img)
	case SceneCocktail:
		// No scene glyph — the cocktail scene's body is painted at
		// `divoom push` time by bakeCocktailBackground as a typographic
		// recipe card (name, glass/category subhead, ingredient list,
		// method). A martini-glass corner mark would crash into the
		// METHOD prose, and the recipe typography carries the identity
		// without it.
	default:
		drawSceneGlyph(img, scene)
	}
	return encodeImage(img, format)
}

// MoonPhaseVariants is the number of pre-rendered moonphase variants
// covering one full synodic cycle. Index 0 is new moon, 7 is full moon;
// 1-6 wax (lit on the right), 8-13 wane (lit on the left).
const MoonPhaseVariants = 14

// SceneMoonphaseBackground builds the moonphase scene bg with the disc
// painted for phaseIndex (0-13 across one synodic cycle). Index outside
// [0, MoonPhaseVariants) is clamped — the scene's BgPathFor only emits
// valid indices, but defensiveness here keeps a stray call from
// panicking the render path.
func SceneMoonphaseBackground(phaseIndex int, format Format, now time.Time) ([]byte, error) {
	if phaseIndex < 0 {
		phaseIndex = 0
	}
	if phaseIndex >= MoonPhaseVariants {
		phaseIndex = MoonPhaseVariants - 1
	}
	img := buildHeroImage(now)
	drawMoonDisc(img, phaseIndex)
	drawBakedSceneTitle(img, "moon")
	return encodeImage(img, format)
}

// MoonIllumFractionForIndex returns the lit fraction (0..1) for variant
// i in [0, MoonPhaseVariants). 0 = new (dark), 7 = full (bright);
// 1-6 wax and 8-13 wane through the same midpoints. The shape matches
// the widget's illumination() formula sampled at i/14.
func MoonIllumFractionForIndex(i int) float64 {
	f := float64(i) / float64(MoonPhaseVariants)
	return (1 - math.Cos(2*math.Pi*f)) / 2
}

// drawMoonDisc paints a moon disc at (400, 730) with radius 200 for the
// given phaseIndex. Geometry: fill the whole disc lit, then carve an
// offset shadow circle along x. Waxing (1-6) carves the LEFT side, so
// the lit area remains on the right; waning (8-13) carves the RIGHT.
// Full (7) paints the whole disc lit; new (0) paints it dark.
func drawMoonDisc(img *image.RGBA, phaseIndex int) {
	const (
		cx     = 400
		cy     = 730
		radius = 200
	)
	lit := GruvFg
	dark := GruvBgDarker
	switch {
	case phaseIndex == 0:
		// New moon — disc is fully dark; render it so the moon "is
		// there" against the bg-hard backdrop.
		fillCircle(img, cx, cy, radius, dark)
	case phaseIndex == 7:
		fillCircle(img, cx, cy, radius, lit)
	default:
		fillCircle(img, cx, cy, radius, lit)
		illum := MoonIllumFractionForIndex(phaseIndex)
		// offset 0 → shadow centred → all dark; 2*radius → shadow off
		// the disc → all lit.
		offset := int(illum * 2 * float64(radius))
		dx := -offset // waxing: shadow on the left, lit on the right
		if phaseIndex > 7 {
			dx = offset // waning: shadow on the right, lit on the left
		}
		fillCircle(img, cx+dx, cy, radius, dark)
	}
}

// SceneWeatherBackground renders the weather scene's bg with the icon
// matching `outlook` (one of the strings produced by the weather widget:
// clear, cloudy, overcast, rain, drizzle, snow, fog, thunder). Unknown
// outlooks fall back to the cloudy icon.
func SceneWeatherBackground(outlook string, format Format, now time.Time) ([]byte, error) {
	img := buildHeroImage(now)
	drawWeatherGlyph(img, outlook)
	drawWeatherChrome(img)
	return encodeImage(img, format)
}

// SunriseBackground bakes the sunrise scene's static chrome: a
// horizontal day-arc (yellow→orange gradient) across the body area,
// three fixed reference ticks at the sunrise/noon/sunset positions,
// baked labels under each, and a bottom-LEFT sun glyph so the right
// side stays quiet for the labels. The daylight-duration headline
// (e.g. "13h 16m") and the current-time tick are dynamic device Text
// elements wired up by the scene; they are NOT baked here.
//
// pushSceneBackgrounds runs once at startup, so anything baked here
// would go stale on the first midnight rollover; the daylight value
// drifts by ~minutes across a season and used to be baked, which left
// the headline lying after the bg cache rolled over a few weeks.
func SunriseBackground(format Format, now time.Time) ([]byte, error) {
	img := buildHeroImage(now)
	drawSunriseChrome(img)
	// Sun glyph in the bottom-LEFT corner — the bottom-right area is
	// now claimed by the baked arc + labels.
	// Anchor pulled lower than the dim-era 1100 because the redesign
	// upsized the sun + rays (sunR 60→105, rays 9→14) and the higher
	// rays would crash into the "sunrise" label baseline at y=960.
	drawSceneGlyphAt(img, SceneSunrise, 180, 1150)
	drawBakedSceneTitle(img, "sun")
	return encodeImage(img, format)
}

// drawSunriseChrome paints the day-arc gradient, the three reference
// ticks, the sunrise/noon/sunset labels, and a "daylight" caption
// above the headline so the giant "14h 31m" number reads as
// "today's daylight hours" rather than as an unlabelled big number.
// Pulled into its own helper so SunriseBackground stays a thin façade
// and the painter can be unit-tested directly if needed.
func drawSunriseChrome(img *image.RGBA) {
	// "daylight" legend baked above the headline (Roboto Condensed
	// Light 24pt cFgDark) so the dynamic "<N>h <N>m" element below
	// has explicit context.
	if f, err := LoadFont("RobotoCondensed-Light.ttf"); err == nil {
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size: 24, DPI: 72, Hinting: font.HintingFull,
		})
		if err == nil {
			drawLabelCentered(img, "daylight", face, CanvasW/2, 585, GruvFgDark)
			face.Close()
		}
	}
	const (
		arcY      = 840
		arcLeft   = 80
		arcRight  = 720
		arcThick  = 4
		tickH     = 16 // total vertical extent of a reference tick
		tickThick = 4
		labelY    = 960
	)
	// Arc — 1px-wide vertical slices, each colour-interpolated between
	// the yellow left endpoint and the orange right endpoint. Crude but
	// reads as a smooth gradient at viewing distance.
	span := arcRight - arcLeft
	for x := arcLeft; x < arcRight; x++ {
		t := float64(x-arcLeft) / float64(span)
		c := lerpRGBA(GruvYellow, GruvOrange, t)
		draw.Draw(img,
			image.Rect(x, arcY-arcThick/2, x+1, arcY+arcThick/2),
			&image.Uniform{c}, image.Point{}, draw.Src)
	}
	// Three fixed reference ticks — sunrise (left end), noon (mid), sunset (right end).
	tickTop := arcY - tickH/2
	tickBot := arcY + tickH/2
	for _, tx := range []int{arcLeft, (arcLeft + arcRight) / 2, arcRight} {
		draw.Draw(img,
			image.Rect(tx-tickThick/2, tickTop, tx-tickThick/2+tickThick, tickBot),
			&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)
	}

	// Labels under the arc — Roboto Condensed Light, 22pt, dim.
	if f, err := LoadFont("RobotoCondensed-Light.ttf"); err == nil {
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size: 22, DPI: 72, Hinting: font.HintingFull,
		})
		if err == nil {
			defer face.Close()
			// "sunrise" left-aligned in its slot, "noon" centred,
			// "sunset" right-aligned. Slot bounds per spec.
			drawLabelLeft(img, "sunrise", face, 40, labelY, GruvFgDark)
			drawLabelCentered(img, "noon", face, (320+480)/2, labelY, GruvFgDark)
			drawLabelRight(img, "sunset", face, 760, labelY, GruvFgDark)
		} else {
			slog.Warn("sunrise chrome: label face init failed", "err", err)
		}
	} else {
		slog.Warn("sunrise chrome: label font load failed", "err", err)
	}
}

// lerpRGBA linearly interpolates between a and b at t in [0,1].
func lerpRGBA(a, b color.RGBA, t float64) color.RGBA {
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: 0xff,
	}
}

// QuoteFamily picks one of the three baked-chrome quote layouts. See
// FamilyChrome for the per-family field meanings and
// SceneFamilyBackground for how it's wired in.
type QuoteFamily int

const (
	// FamilyMarginalia is the page-of-a-book layout — book name + chapter
	// at the top, a baked drop-cap glyph in the body-left margin,
	// attribution baked at the bottom-right. Default for QuoteScene.
	FamilyMarginalia QuoteFamily = iota
	// FamilyFromSource is the in-universe-document layout — header strip
	// (stardate / earthforce transmission / press imprint) above the body
	// and a thin rule above the attribution slot at the bottom.
	FamilyFromSource
	// FamilyTerminal is the shell-session layout — baked `$ <cmd>` prompt
	// above the body and a two-line status bar at the bottom carrying
	// `source:` and `author:` lines.
	FamilyTerminal
)

// FamilyChrome carries the per-scene strings that the family painters
// bake into the bg. Fields outside a family's needs are ignored. The
// glyph (drawn by SceneBackground) is moved per family by
// glyphAnchorFor so the chrome and the glyph never collide.
type FamilyChrome struct {
	Family QuoteFamily

	// FromSource: in-universe header strip. Header is the left text
	// (e.g. "STARDATE 79341.7"); Subheader is the right text (e.g.
	// "PERSONAL LOG"). Either may be empty.
	Header    string
	Subheader string

	// Marginalia: top-of-page imprint. BookName goes top-left,
	// Chapter goes top-right. The drop cap itself is NOT baked — it's
	// a dynamic Text DispElement set at scene-activation time to the
	// body's actual first letter (see marginaliaDropCap in scenes.go),
	// so the bg only reserves the column where it will land.
	BookName string
	Chapter  string

	// Terminal: baked shell prompt above the body and two status-bar
	// lines at the bottom. ShellPrompt is the full prompt string
	// (e.g. "$ fortune -s" or "$ define"); SourceFooter / AuthorFooter
	// are the bottom two lines.
	ShellPrompt  string
	SourceFooter string
	AuthorFooter string
	// PunchlineOrnaments, when set on a FamilyTerminal chrome, bakes two
	// giant GruvFgDark pull-quote glyphs into the body area — an opening
	// curly quote in the upper-left and a closing one in the lower-right.
	// Used by the devil's dictionary scene whose aphorism body wants
	// pull-quote decoration around it.
	PunchlineOrnaments bool

	// OmitSceneGlyph suppresses the scene-identity glyph that
	// SceneFamilyBackground would otherwise bake into the body track.
	// FamilyTerminal anchors the glyph at (620, 700) which sits inside
	// the dense dictionary body — the curly-brace glyph showed through
	// the jargon definition prose. Setting this true on scenes whose
	// body fills that region keeps the shell prompt as the sole label.
	OmitSceneGlyph bool
}

// SceneFamilyBackground builds the hero frame, paints the scene's glyph
// (relocated per family so the chrome stays unobstructed), and then bakes
// the family chrome on top. Used by the quote / dictionary scenes that
// participate in the three-family redesign; other scenes keep calling
// SceneBackground.
func SceneFamilyBackground(scene Scene, chrome FamilyChrome, format Format, now time.Time) ([]byte, error) {
	img := buildHeroImage(now)
	if !chrome.OmitSceneGlyph {
		cx, cy := glyphAnchorFor(chrome.Family)
		drawSceneGlyphAt(img, scene, cx, cy)
	}
	switch chrome.Family {
	case FamilyFromSource:
		drawFromSourceChrome(img, chrome.Header, chrome.Subheader)
	case FamilyMarginalia:
		drawMarginaliaChrome(img, chrome.BookName, chrome.Chapter)
	case FamilyTerminal:
		drawTerminalChrome(img, chrome.ShellPrompt, chrome.SourceFooter, chrome.AuthorFooter)
		if chrome.PunchlineOrnaments {
			drawPunchlineOrnaments(img)
		}
	}
	return encodeImage(img, format)
}

// glyphAnchorFor returns the (cx, cy) the scene glyph should be drawn at
// for a given family. FamilyFromSource and FamilyMarginalia put the
// glyph in the bottom-LEFT (the new bottom-right is occupied by
// attribution / status text); FamilyTerminal puts it in the top-RIGHT
// (its bottom is full of the two-line status bar).
func glyphAnchorFor(family QuoteFamily) (cx, cy int) {
	switch family {
	case FamilyTerminal:
		// Top-right corner, beside the baked shell prompt, in the open
		// area before the body starts.
		return CanvasW - 180, 700
	case FamilyFromSource, FamilyMarginalia:
		// Bottom-LEFT, but pulled up enough to sit ABOVE the baked
		// bottom rule (y≈1125) so the glyph and the rule don't fight.
		return 180, 970
	default:
		return CanvasW - 180, CanvasH - 240
	}
}

// drawMarketsChrome bakes the markets scene's trading-terminal
// furniture: a 2px hairline under the symbol+price headline, "1 week"
// and "1 month" captions centred under the percent badges, and a 1px
// footer hairline near the bottom of the body track. The bar-chart
// corner glyph is painted separately by drawSceneGlyph.
func drawMarketsChrome(img *image.RGBA) {
	const (
		left          = 80
		right         = CanvasW - 80
		headlineRuleY = 680 // top of the 2px rule under the headline
		headlineRuleH = 2
		captionBase   = 880 // baseline for "1 week" / "1 month"
		footerRuleY   = 1110
	)
	// Hairline under the headline.
	draw.Draw(img, image.Rect(left, headlineRuleY, right, headlineRuleY+headlineRuleH),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)

	// "1 week" / "1 month" captions in Roboto Condensed Light 22pt.
	if f, err := LoadFont("RobotoCondensed-Light.ttf"); err == nil {
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size: 22, DPI: 72, Hinting: font.HintingFull,
		})
		if err == nil {
			defer face.Close()
			// Left column centred on x = (80+400)/2 = 240; right column on
			// (400+720)/2 = 560. Matches the badge element X spans above.
			drawLabelCentered(img, "1 week", face, (80+400)/2, captionBase, GruvFgDark)
			drawLabelCentered(img, "1 month", face, (400+720)/2, captionBase, GruvFgDark)
		} else {
			slog.Warn("markets chrome: face init failed", "err", err)
		}
	} else {
		slog.Warn("markets chrome: font load failed", "err", err)
	}

	// Footer hairline.
	draw.Draw(img, image.Rect(left, footerRuleY, right, footerRuleY+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)
}

// drawHNChrome bakes the HN scene's brand chrome: the "HACKER NEWS"
// wordmark in orange near the top of the body area, a 2px orange rule
// directly under it as a brand-color separator, and a 1px dim rule at
// y=1140 separating the body from the metadata footer. The Y glyph in
// the bottom-right corner is painted separately by drawSceneGlyph.
func drawHNChrome(img *image.RGBA) {
	const (
		left          = 80
		right         = CanvasW - 80
		wordmarkBase  = 510 // baseline for the "HACKER NEWS" wordmark
		brandRuleY    = 540 // top of the 2px orange separator
		brandRuleH    = 2
		footerRuleY   = 1140 // 1px dim rule above the metadata footer
	)
	if f, err := LoadFont("RobotoCondensed-Light.ttf"); err == nil {
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size: 32, DPI: 72, Hinting: font.HintingFull,
		})
		if err == nil {
			defer face.Close()
			drawLabelLeft(img, "HACKER NEWS", face, left, wordmarkBase, GruvOrange)
		} else {
			slog.Warn("hn chrome: face init failed", "err", err)
		}
	} else {
		slog.Warn("hn chrome: font load failed", "err", err)
	}
	// Orange brand-color separator under the wordmark (2px).
	draw.Draw(img, image.Rect(left, brandRuleY, right, brandRuleY+brandRuleH),
		&image.Uniform{GruvOrange}, image.Point{}, draw.Src)
	// Dim footer rule (1px) above the metadata footer.
	draw.Draw(img, image.Rect(left, footerRuleY, right, footerRuleY+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)
}

// drawTILChrome bakes the TIL scene's monumental wordmark and footer
// chrome: a poster-weight "T I L" in Roboto Condensed Light yellow at
// the top-left, a 4px yellow rule beneath it, a 1px footer hairline,
// and a small r/todayilearned attribution in Iosevka mono. The body
// Text element ("that <fact>") flows under the rule, completing the
// grammatical thought "TIL · that <fact>". The corner lightbulb glyph
// is painted separately by drawSceneGlyph.
func drawTILChrome(img *image.RGBA) {
	const (
		left            = 80
		right           = CanvasW - 80
		wordmarkBase    = 560
		ruleY           = 595
		ruleH           = 3
		footerRuleY     = 1180
		attributionBase = 1220
	)
	// Monumental "T I L" — each letter painted separately so the
	// letter-spacing reads as a poster instead of a word. Wordmark
	// shrunk from 180→120pt (and baseline lifted) to free vertical
	// space for longer r/todayilearned titles in the body track.
	if f, err := LoadFont("RobotoCondensed-Light.ttf"); err == nil {
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size: 120, DPI: 72, Hinting: font.HintingFull,
		})
		if err == nil {
			// Three letters across left..right with even spacing — anchor
			// the first at `left`, last at `right`, middle in the centre.
			// Centring each letter on its column keeps the optical balance
			// independent of per-glyph advance widths.
			cols := []int{left + 60, (left + right) / 2, right - 60}
			drawLabelCentered(img, "T", face, cols[0], wordmarkBase, GruvYellow)
			drawLabelCentered(img, "I", face, cols[1], wordmarkBase, GruvYellow)
			drawLabelCentered(img, "L", face, cols[2], wordmarkBase, GruvYellow)
			face.Close()
		} else {
			slog.Warn("til chrome: wordmark face init failed", "err", err)
		}
	} else {
		slog.Warn("til chrome: wordmark font load failed", "err", err)
	}
	// 4px yellow rule under the wordmark.
	draw.Draw(img, image.Rect(left, ruleY, right, ruleY+ruleH),
		&image.Uniform{GruvYellow}, image.Point{}, draw.Src)
	// Footer hairline.
	draw.Draw(img, image.Rect(left, footerRuleY, right, footerRuleY+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)
	// r/todayilearned attribution — Iosevka mono 24pt dim, left-aligned.
	if f, err := LoadFont("Iosevka-Regular.ttf"); err == nil {
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size: 24, DPI: 72, Hinting: font.HintingFull,
		})
		if err == nil {
			drawLabelLeft(img, "r/todayilearned", face, left, attributionBase, GruvFgDark)
			face.Close()
		} else {
			slog.Warn("til chrome: attribution face init failed", "err", err)
		}
	} else {
		slog.Warn("til chrome: attribution font load failed", "err", err)
	}
}

// drawBakedSceneTitle renders the canonical scene-title row baked
// into the bg at y=505 baseline, matching cmd/divoom's sceneTitle()
// helper. Frees a device Text slot for every scene that uses it —
// the title is always-static-per-scene so there's no reason to keep
// burning the slot at install time.
func drawBakedSceneTitle(img *image.RGBA, title string) {
	if title == "" {
		return
	}
	f, err := LoadFont("RobotoCondensed-Light.ttf")
	if err != nil {
		slog.Warn("scene title: font load failed", "err", err)
		return
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size: 26, DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		slog.Warn("scene title: face init failed", "err", err)
		return
	}
	defer face.Close()
	drawLabelCentered(img, title, face, CanvasW/2, 505, GruvFgDark)
}

// drawNASACredit bakes the nasa scene's title row as a two-tone
// "NASA · astronomy picture of the day" — replaces the standard
// sceneTitle so NASA gets explicit credit for the photos. NASA in
// gruvbox orange (their wordmark colour evokes the "worm" logo);
// the rest of the title in the canonical small dim grey.
func drawNASACredit(img *image.RGBA) {
	f, err := LoadFont("RobotoCondensed-Light.ttf")
	if err != nil {
		slog.Warn("nasa credit: font load failed", "err", err)
		return
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size: 26, DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		slog.Warn("nasa credit: face init failed", "err", err)
		return
	}
	defer face.Close()
	const (
		baseline = 505
		nasa     = "NASA"
		rest     = " · astronomy picture of the day"
	)
	// Measure both halves with the already-opened face so the
	// combined string centres horizontally on the canvas.
	nasaW := font.MeasureString(face, nasa).Ceil()
	restW := font.MeasureString(face, rest).Ceil()
	x := CanvasW/2 - (nasaW+restW)/2
	drawLabelLeft(img, nasa, face, x, baseline, GruvOrange)
	drawLabelLeft(img, rest, face, x+nasaW, baseline, GruvFgDark)
}

// drawSeismicChrome bakes the seismic scene's static chrome: the
// "magnitude" caption row directly under the hero number, and a pair
// of hairlines bracketing the stats strip below — mirroring the
// weather scene's bottom-strip pattern.
//
// Hero element occupies y=620..840. The caption sits at y=870 baseline
// in the dead band beneath it. The stats strip element lives at
// y=1015..1075, bracketed by hairlines at y=985 and y=1095. The gap
// between the caption baseline (y=870) and the top hairline (y=985)
// is the deliberate breathing room separating the hero from the strip.
// Roboto Condensed Light 24pt GruvFgDark for the caption; 1px
// GruvFgDark rules for the strip.
func drawSeismicChrome(img *image.RGBA) {
	const (
		colLeft   = 80
		colRight  = 720
		stripTopY = 985
		stripBotY = 1095
	)
	draw.Draw(img,
		image.Rect(colLeft, stripTopY, colRight, stripTopY+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)
	draw.Draw(img,
		image.Rect(colLeft, stripBotY, colRight, stripBotY+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)

	f, err := LoadFont("RobotoCondensed-Light.ttf")
	if err != nil {
		slog.Warn("seismic chrome: font load failed", "err", err)
		return
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size: 24, DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		slog.Warn("seismic chrome: face init failed", "err", err)
		return
	}
	defer face.Close()
	drawLabelCentered(img, "magnitude", face, CanvasW/2, 870, GruvFgDark)
}

// drawGitHubChrome bakes the GitHub scene's static chrome: title row,
// hero caption, and three stat-column labels. All five lines used to
// be device Text elements; they're baked here so the scene's four
// dynamic numbers (lifetime contributions + total PRs + open PRs +
// years) fit under the device's 6-Text cap.
//
// Layout matches the scene's element coordinates in scene_github.go:
//
//	y=505 baseline   "GitHub"               sceneTitle position
//	y=735 baseline   "lifetime contributions"  caption under hero
//	y=965 baseline   "total PRs" / "open" / "years on GitHub"  stat-column labels
//
// All baked in Roboto Condensed Light, cFgDark, matching the
// sceneTitle / caption conventions used elsewhere.
func drawGitHubChrome(img *image.RGBA) {
	f, err := LoadFont("RobotoCondensed-Light.ttf")
	if err != nil {
		slog.Warn("github chrome: font load failed", "err", err)
		return
	}
	mkFace := func(size int) (font.Face, error) {
		return opentype.NewFace(f, &opentype.FaceOptions{
			Size: float64(size), DPI: 72, Hinting: font.HintingFull,
		})
	}
	if face, err := mkFace(26); err == nil {
		drawLabelCentered(img, "GitHub", face, CanvasW/2, 505, GruvFgDark)
		face.Close()
	}
	if face, err := mkFace(24); err == nil {
		drawLabelCentered(img, "lifetime contributions", face, CanvasW/2, 735, GruvFgDark)
		face.Close()
	}
	if face, err := mkFace(22); err == nil {
		// Match scene_github.go stat-column centres at StartX + W/2:
		// col1 = 40..280 (cx=160), col2 = 280..520 (cx=400),
		// col3 = 520..760 (cx=640).
		drawLabelCentered(img, "total PRs", face, 160, 965, GruvFgDark)
		drawLabelCentered(img, "open", face, 400, 965, GruvFgDark)
		drawLabelCentered(img, "years on GitHub", face, 640, 965, GruvFgDark)
		face.Close()
	}
}

// catfactsInstitutions is the small pool of in-universe attributions the
// catfacts scene picks from per bg generation. Re-pushing produces a new
// "volume" with a fresh number + institution — a tiny delight that
// rewards repeat viewing.
var catfactsInstitutions = []string{
	"Cat Behaviour Study Group",
	"Royal Veterinary College",
	"British Feline Society",
	"Society for Cat Research",
	"Domestic Feline Council",
	"International Cat Studies",
	"Felidae Observation Trust",
}

// DrawCatfactsChrome bakes the field-guide entry chrome onto the catfacts
// scene background: the binomial "Felis catus" top-left, a short rule
// under it, the taxonomic classification beneath, a pilcrow drop-marker
// in the body's left margin, a footer hairline, and the observation
// number + institution in the footer. Italic faces fall back to regular
// since the project's fonts/ set ships only the upright variants.
func DrawCatfactsChrome(img *image.RGBA, observationNum int, institution string) {
	const (
		left          = 80
		right         = CanvasW - 80
		binomialBase  = 485
		ruleY         = 540
		ruleRightX    = 280
		classBase     = 555
		pilcrowBase   = 655
		footerRuleY   = 1140
		footerBase    = 1160
		footerRightX0 = 480
	)

	// Binomial: Roboto Condensed Light 44pt. The project's fonts/ set
	// has no italic TTF; the binomial reads as a scientific name from
	// context (followed immediately by MAMMALIA · CARNIVORA · FELIDAE).
	if f, err := LoadFont("RobotoCondensed-Light.ttf"); err == nil {
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size: 44, DPI: 72, Hinting: font.HintingFull,
		})
		if err == nil {
			drawLabelLeft(img, "Felis catus", face, left, binomialBase, GruvFg)
			face.Close()
		} else {
			slog.Warn("catfacts chrome: binomial face init failed", "err", err)
		}
	} else {
		slog.Warn("catfacts chrome: binomial font load failed", "err", err)
	}

	// Short underline rule under the binomial.
	draw.Draw(img, image.Rect(left, ruleY, ruleRightX, ruleY+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)

	// Taxonomic classification, fontProseLight-equivalent 22pt dim.
	if f, err := LoadFont("RobotoCondensed-Light.ttf"); err == nil {
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size: 22, DPI: 72, Hinting: font.HintingFull,
		})
		if err == nil {
			drawLabelLeft(img, "MAMMALIA · CARNIVORA · FELIDAE",
				face, left, classBase, GruvFgDark)
			face.Close()
		} else {
			slog.Warn("catfacts chrome: classification face init failed", "err", err)
		}
	} else {
		slog.Warn("catfacts chrome: classification font load failed", "err", err)
	}

	// Pilcrow drop-marker in the small left margin beside the fact body.
	if f, err := LoadFont("RobotoCondensed-Regular.ttf"); err == nil {
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size: 40, DPI: 72, Hinting: font.HintingFull,
		})
		if err == nil {
			drawLabelLeft(img, "¶", face, left, pilcrowBase, GruvFgDark)
			face.Close()
		} else {
			slog.Warn("catfacts chrome: pilcrow face init failed", "err", err)
		}
	} else {
		slog.Warn("catfacts chrome: pilcrow font load failed", "err", err)
	}

	// Footer hairline.
	draw.Draw(img, image.Rect(left, footerRuleY, right, footerRuleY+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)

	// Footer left: observation number.
	if f, err := LoadFont("RobotoCondensed-Light.ttf"); err == nil {
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size: 26, DPI: 72, Hinting: font.HintingFull,
		})
		if err == nil {
			drawLabelLeft(img,
				fmt.Sprintf("Observation №%d", observationNum),
				face, left, footerBase, GruvFgDark)
			face.Close()
		} else {
			slog.Warn("catfacts chrome: footer-left face init failed", "err", err)
		}
	} else {
		slog.Warn("catfacts chrome: footer-left font load failed", "err", err)
	}

	// Footer right: institution name (right-aligned within its slot).
	// Same italic note as the binomial — no italic TTF available, so
	// upright is used.
	if institution != "" {
		if f, err := LoadFont("RobotoCondensed-Light.ttf"); err == nil {
			face, err := opentype.NewFace(f, &opentype.FaceOptions{
				Size: 24, DPI: 72, Hinting: font.HintingFull,
			})
			if err == nil {
				// drawLabelRight clamps the text against the right edge
				// of its slot; the left bound (footerRightX0) is implicit
				// in the spec but not enforced — the institution names
				// are short enough that they fit comfortably.
				_ = footerRightX0
				drawLabelRight(img, institution, face, right, footerBase, GruvFgDark)
				face.Close()
			} else {
				slog.Warn("catfacts chrome: footer-right face init failed", "err", err)
			}
		} else {
			slog.Warn("catfacts chrome: footer-right font load failed", "err", err)
		}
	}
}

// ISS map geometry — the chrome and the scene's dot-positioning math
// share these constants so they can never drift apart. Keep the world
// map rect at 720x360 to match the embedded mask resolution.
const (
	ISSMapX0 = 40
	ISSMapY0 = 560
	ISSMapW  = 720
	ISSMapH  = 360
	ISSMapX1 = ISSMapX0 + ISSMapW
	ISSMapY1 = ISSMapY0 + ISSMapH
)

// worldMapMask is the decoded equirectangular continents-mask PNG;
// loaded once on first use. Same alpha-threshold treatment as the
// starfleet delta.
var (
	worldMapOnce sync.Once
	worldMapMask image.Image
)

// DrawISSChrome bakes the ISS scene's static chrome onto img:
//
//   - a hairline under the always-on top zone (the telemetry strip
//     above it is now a live Text element painted by the scene — see
//     issTelemetryStrip in scenes.go),
//   - the equirectangular world-map outline filling the body area
//     (loaded from the embedded mask and painted in GruvFgDark),
//   - hairlines marking the equator (horizontal mid-line of the map)
//     and the prime meridian (vertical mid-line of the map).
//
// The dynamic ISS sub-satellite dot is NOT baked here — the scene
// installs it as a Text element whose StartX/StartY are recomputed at
// every activation from the current lat/lon.
func DrawISSChrome(img *image.RGBA) {
	const (
		left      = 80
		right     = CanvasW - 80
		hairlineY = 535
	)
	draw.Draw(img, image.Rect(left, hairlineY, right, hairlineY+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)

	// World map outline — every above-threshold mask pixel paints a
	// GruvFgDark pixel at the same offset inside the map rect.
	worldMapOnce.Do(func() {
		m, err := png.Decode(bytes.NewReader(worldMapEquirectPNG))
		if err != nil {
			panic(fmt.Errorf("render: decode embedded world map: %w", err))
		}
		worldMapMask = m
	})
	paintMaskAt(img, worldMapMask, ISSMapX0, ISSMapY0, GruvFgDark)

	// Equator hairline across the map (latitude 0).
	equatorY := ISSMapY0 + ISSMapH/2
	draw.Draw(img,
		image.Rect(ISSMapX0, equatorY, ISSMapX1, equatorY+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)
	// Prime-meridian hairline down the map (longitude 0).
	meridianX := ISSMapX0 + ISSMapW/2
	draw.Draw(img,
		image.Rect(meridianX, ISSMapY0, meridianX+1, ISSMapY1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)
}

// paintMaskAt is paintMask's top-left-anchored cousin: every above-
// threshold pixel of mask paints a pixel at (originX+px, originY+py).
// Used for the world-map outline where centring math would just hide
// the explicit rect the chrome already declares.
func paintMaskAt(img *image.RGBA, mask image.Image, originX, originY int, c color.RGBA) {
	mb := mask.Bounds()
	mw, mh := mb.Dx(), mb.Dy()
	bounds := img.Bounds()
	for py := 0; py < mh; py++ {
		dy := originY + py
		if dy < bounds.Min.Y || dy >= bounds.Max.Y {
			continue
		}
		for px := 0; px < mw; px++ {
			_, _, _, a := mask.At(mb.Min.X+px, mb.Min.Y+py).RGBA()
			if a>>8 <= starfleetDeltaAlphaThreshold {
				continue
			}
			dx := originX + px
			if dx < bounds.Min.X || dx >= bounds.Max.X {
				continue
			}
			img.SetRGBA(dx, dy, c)
		}
	}
}

// drawFromSourceChrome bakes the in-universe header strip: Header on the
// left, Subheader on the right, both in fontProseLight 28pt cFgDark with
// a thin rule below them and a matching rule above the attribution slot
// at the bottom.
func drawFromSourceChrome(img *image.RGBA, header, subheader string) {
	const (
		left      = 80
		right     = CanvasW - 80
		baselineY = 510
		topRuleY  = 525
		botRuleY  = 1125
	)
	if header != "" || subheader != "" {
		f, err := LoadFont("RobotoCondensed-Light.ttf")
		if err == nil {
			face, err := opentype.NewFace(f, &opentype.FaceOptions{
				Size: 26, DPI: 72, Hinting: font.HintingFull,
			})
			if err == nil {
				defer face.Close()
				if header != "" {
					drawLabelLeft(img, header, face, left, baselineY, GruvFgDark)
				}
				if subheader != "" {
					drawLabelRight(img, subheader, face, right, baselineY, GruvFgDark)
				}
			} else {
				slog.Warn("from-source chrome: face init failed", "err", err)
			}
		} else {
			slog.Warn("from-source chrome: font load failed", "err", err)
		}
	}
	draw.Draw(img, image.Rect(left, topRuleY, right, topRuleY+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(left, botRuleY, right, botRuleY+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)
}

// drawMarginaliaChrome bakes the book-page imprint: BookName top-left,
// Chapter top-right, a thin rule beneath them, plus a thin bottom-right
// rule under where the attribution Text element will render. The drop
// cap itself is painted at push time as a dynamic Text DispElement
// (the body's actual first letter), not baked here — see
// marginaliaDropCap in cmd/divoom/scenes.go.
func drawMarginaliaChrome(img *image.RGBA, bookName, chapter string) {
	const (
		left         = 80
		right        = CanvasW - 80
		imprintBase  = 510
		topRuleY     = 525
		bottomRuleY  = 1175
		bottomRuleX0 = 380
	)
	if bookName != "" || chapter != "" {
		f, err := LoadFont("RobotoCondensed-Light.ttf")
		if err == nil {
			face, err := opentype.NewFace(f, &opentype.FaceOptions{
				Size: 26, DPI: 72, Hinting: font.HintingFull,
			})
			if err == nil {
				defer face.Close()
				if bookName != "" {
					drawLabelLeft(img, bookName, face, left, imprintBase, GruvFgDark)
				}
				if chapter != "" {
					drawLabelRight(img, chapter, face, right, imprintBase, GruvFgDark)
				}
			} else {
				slog.Warn("marginalia chrome: face init failed", "err", err)
			}
		} else {
			slog.Warn("marginalia chrome: font load failed", "err", err)
		}
	}
	draw.Draw(img, image.Rect(left, topRuleY, right, topRuleY+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)
	// Decorative bottom-right rule under the attribution slot.
	draw.Draw(img, image.Rect(bottomRuleX0, bottomRuleY, right, bottomRuleY+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)
}

// drawTerminalChrome bakes the shell-session frame: ShellPrompt baked in
// fontMono 28pt at the top, a thin top rule below it, and a two-line
// status bar (source: / author:) at the bottom in fontMono 22pt with
// thin rules bracketing it.
func drawTerminalChrome(img *image.RGBA, prompt, sourceFooter, authorFooter string) {
	const (
		left          = 80
		right         = CanvasW - 80
		promptBase    = 515
		topRuleY      = 535
		statusTopRule = 1140
		sourceBase    = 1170
		authorBase    = 1200
		statusBotRule = 1215
	)
	if prompt != "" {
		f, err := LoadFont("Iosevka-Regular.ttf")
		if err == nil {
			face, err := opentype.NewFace(f, &opentype.FaceOptions{
				Size: 28, DPI: 72, Hinting: font.HintingFull,
			})
			if err == nil {
				defer face.Close()
				drawLabelLeft(img, prompt, face, left, promptBase, GruvFgDark)
			} else {
				slog.Warn("terminal chrome: face init failed", "err", err)
			}
		} else {
			slog.Warn("terminal chrome: font load failed", "err", err)
		}
	}
	draw.Draw(img, image.Rect(left, topRuleY, right, topRuleY+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(left, statusTopRule, right, statusTopRule+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)
	if sourceFooter != "" || authorFooter != "" {
		f, err := LoadFont("Iosevka-Regular.ttf")
		if err == nil {
			face, err := opentype.NewFace(f, &opentype.FaceOptions{
				Size: 22, DPI: 72, Hinting: font.HintingFull,
			})
			if err == nil {
				defer face.Close()
				if sourceFooter != "" {
					drawLabelLeft(img, sourceFooter, face, left, sourceBase, GruvFgDark)
				}
				if authorFooter != "" {
					drawLabelLeft(img, authorFooter, face, left, authorBase, GruvFgDark)
				}
			}
		}
	}
	draw.Draw(img, image.Rect(left, statusBotRule, right, statusBotRule+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)
}

// drawPunchlineOrnaments bakes the framed-epitaph shape used by the
// devil's dictionary scene: two 1-pixel GruvFgDark horizontal rules
// bracketing the centred aphorism body, plus a small (~80px) corner
// devil-head silhouette anchored bottom-right so it identifies the
// scene without competing with the framed body text.
//
// Top rule sits at y=590 (just above the body's top edge at y=740);
// bottom rule sits at y=1080 (just below the body's bottom edge at
// y=1060). Together they read as the rules above and below a
// nineteenth-century epitaph or title-page motto — instantly Bierce.
func drawPunchlineOrnaments(img *image.RGBA) {
	const (
		ruleLeft   = 80
		ruleRight  = 720
		ruleTopY   = 590
		ruleBotY   = 1080
		devilSize  = 80
		devilCX    = CanvasW - 80 - devilSize/2 // 680 — flush with the right rule end
		devilCY    = CanvasH - 200               // above the baked status bar
	)
	draw.Draw(img, image.Rect(ruleLeft, ruleTopY, ruleRight, ruleTopY+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(ruleLeft, ruleBotY, ruleRight, ruleBotY+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)

	devilOnce.Do(func() {
		m, err := png.Decode(bytes.NewReader(devilPNG))
		if err != nil {
			panic(fmt.Errorf("render: decode embedded devil: %w", err))
		}
		devilMask = m
	})
	paintMaskScaled(img, devilMask, devilCX, devilCY, devilSize, devilSize, GruvFgDark)
}

// drawWeatherChrome bakes the weather scene's static chrome: a small
// "weather" title row at y=480 (replacing the device sceneTitle Text
// element so the scene stays at the device's 6-Text cap) and a pair
// of hairlines bracketing the bottom strip where the dynamic stat
// row renders.
//
// Three separate per-column values (AIR / HUMIDITY / RAIN) used to
// live here; they're now folded into a single combined Text element
// driven by weatherStrip, so the chrome only needs to mark the strip
// region — no column labels or vertical dividers required.
func drawWeatherChrome(img *image.RGBA) {
	const (
		colLeft = 80
		colRight = 720
		stripTopY = 985
		stripBotY = 1095
	)
	// Top + bottom horizontal hairlines, 1px tall — bracket the
	// strip element below.
	draw.Draw(img,
		image.Rect(colLeft, stripTopY, colRight, stripTopY+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)
	draw.Draw(img,
		image.Rect(colLeft, stripBotY, colRight, stripBotY+1),
		&image.Uniform{GruvFgDark}, image.Point{}, draw.Src)

	// Baked "weather" title — Roboto Condensed Light 26pt cFgDark
	// centred at y=480, matching the cmd/divoom sceneTitle helper so
	// the title row is visually identical to every other scene.
	f, err := LoadFont("RobotoCondensed-Light.ttf")
	if err != nil {
		slog.Warn("weather chrome: font load failed; skipping title", "err", err)
		return
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size: 26, DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		slog.Warn("weather chrome: face init failed; skipping title", "err", err)
		return
	}
	defer face.Close()
	drawLabelCentered(img, "weather", face, CanvasW/2, 505, GruvFgDark)
}

// drawEasterEgg paints the rare-treat scene's centrepiece — a giant
// gruvbox-yellow egg, a hairline zigzag crack across its upper third,
// and a small "rare drop · ~1 in 200" caption beneath it. The body
// Text renders DARK on the yellow (cBgHard on GruvYellow — a real
// gruvbox pairing) so the text reads as printed on the egg rather
// than floating above it.
//
// ryBot shrunk 320→280 so the egg + caption fit cleanly above the
// scene rotator; the crack and caption together carry the "this is
// the rare one" signal that the plain ellipse used to lack.
func drawEasterEgg(img *image.RGBA) {
	const (
		cx    = CanvasW / 2
		cy    = 860
		rx    = 250
		ryTop = 250
		ryBot = 280
	)
	fillEgg(img, cx, cy, rx, ryTop, ryBot, GruvYellow)
	drawEasterCrack(img, cx, cy, rx, ryTop)
	drawEasterCaption(img, "rare drop  ·  ~1 in 200", cx, 1210)
}

// drawEasterCrack draws a thin dark zigzag hairline across the upper
// third of the egg — a chip in the shell that signals "rare drop"
// without spelling it out. Built from a sequence of (x, y) waypoints
// joined by 3-px-thick line segments so the crack reads at glance
// distance. Y is anchored at egg-top + ryTop/3 (roughly one-third
// down from the egg's apex).
func drawEasterCrack(img *image.RGBA, cx, cy, rx, ryTop int) {
	const (
		thick = 3
		amp   = 14 // zigzag vertical amplitude
	)
	yBase := ryTop / 3 // offset above cy
	// Six waypoints across the egg's mid-upper band, alternating
	// above/below yBase so the crack zig-zags.
	xs := []int{
		cx - rx*7/10,
		cx - rx*4/10,
		cx - rx*1/10,
		cx + rx*2/10,
		cx + rx*5/10,
		cx + rx*8/10,
	}
	ys := []int{
		cy - yBase + amp,
		cy - yBase - amp,
		cy - yBase + amp/2,
		cy - yBase - amp,
		cy - yBase + amp,
		cy - yBase - amp/2,
	}
	for i := 0; i < len(xs)-1; i++ {
		drawThickLine(img, xs[i], ys[i], xs[i+1], ys[i+1], thick, GruvBgHard)
	}
}

// drawEasterCaption bakes a small dim mono caption centred on (cx,
// baselineY). Used for the "rare drop" footer that announces the
// scene's 0.5% weight.
func drawEasterCaption(img *image.RGBA, s string, cx, baselineY int) {
	f, err := LoadFont("Iosevka-Regular.ttf")
	if err != nil {
		slog.Warn("easter caption: font load failed", "err", err)
		return
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size: 22, DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		slog.Warn("easter caption: face init failed", "err", err)
		return
	}
	defer face.Close()
	drawLabelCentered(img, s, face, cx, baselineY, GruvFgDark)
}

// drawThickLine paints a width-thick line from (x0, y0) to (x1, y1)
// using a Bresenham walk that stamps a small filled rect per pixel
// step. Thickness is implemented as a thick×thick square brush; not
// perfectly round but invisible at this scale.
func drawThickLine(img *image.RGBA, x0, y0, x1, y1, thick int, c color.RGBA) {
	dx := x1 - x0
	if dx < 0 {
		dx = -dx
	}
	dy := -(y1 - y0)
	if -dy < 0 {
		dy = y1 - y0
	}
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	half := thick / 2
	for {
		draw.Draw(img,
			image.Rect(x0-half, y0-half, x0-half+thick, y0-half+thick),
			&image.Uniform{c}, image.Point{}, draw.Src)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// fillEgg rasterises a filled egg shape — an ellipse with a smaller
// vertical radius above the equator and a larger one below, so the
// outline reads as the classic narrower-top / wider-bottom curve.
// Integer math (dx²·ry² + dy²·rx² ≤ rx²·ry²) keeps it self-contained.
func fillEgg(img *image.RGBA, cx, cy, rx, ryTop, ryBot int, c color.RGBA) {
	bounds := img.Bounds()
	rx2 := rx * rx
	ryTop2 := ryTop * ryTop
	ryBot2 := ryBot * ryBot
	for y := cy - ryTop; y <= cy+ryBot; y++ {
		if y < bounds.Min.Y || y >= bounds.Max.Y {
			continue
		}
		dy := y - cy
		ry2 := ryTop2
		if dy >= 0 {
			ry2 = ryBot2
		}
		for x := cx - rx; x <= cx+rx; x++ {
			if x < bounds.Min.X || x >= bounds.Max.X {
				continue
			}
			dx := x - cx
			if dx*dx*ry2+dy*dy*rx2 <= rx2*ry2 {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

// DayOfYearBackground builds the dayofyear scene bg with the calendar
// grid baked in (12 rows × 31 cols of day cells, plus month-letter
// labels down the left edge). specialDates maps month*100+day → a
// single-rune mark that paints the cell in red with the letter
// centred; nil / empty just produces the past/today/future grid.
func DayOfYearBackground(now time.Time, specialDates map[int]rune, format Format) ([]byte, error) {
	img := buildHeroImage(now)
	drawDayOfYearGrid(img, now, specialDates)
	return encodeImage(img, format)
}

// dayOfYearCellState describes how one (month, day) cell paints in the
// dayofyear grid. The five states form the priority order documented on
// drawDayOfYearGrid; cellColorFor returns one of these for a given date
// + special-mark presence + today.
type dayOfYearCellState int

const (
	dayOfYearPhantom dayOfYearCellState = iota // dayOfMonth > days in month
	dayOfYearSpecial                           // user-defined special date
	dayOfYearToday                             // today, not special
	dayOfYearPast                              // past day this year
	dayOfYearFuture                            // future day this year
)

// dayOfYearCellState returns the visual state for the cell at (month,
// dayOfMonth) given today's date and the set of special dates. Priority
// order: phantom > special > today > past > future. When today IS a
// special date the cell is still classified as special — the painter
// adds the cYellow border on top of the cRed fill separately.
func dayOfYearCellStateFor(month, dayOfMonth int, today time.Time, specialDates map[int]rune) dayOfYearCellState {
	if dayOfMonth > daysInMonth(today.Year(), month) {
		return dayOfYearPhantom
	}
	if _, ok := specialDates[month*100+dayOfMonth]; ok {
		return dayOfYearSpecial
	}
	if month == int(today.Month()) && dayOfMonth == today.Day() {
		return dayOfYearToday
	}
	tMonth := int(today.Month())
	if month < tMonth || (month == tMonth && dayOfMonth < today.Day()) {
		return dayOfYearPast
	}
	return dayOfYearFuture
}

// daysInMonth returns the number of days in (year, month).
func daysInMonth(year, month int) int {
	// time.Date normalises so day 0 of (month+1) is the last day of month.
	return time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC).Day()
}

// drawDayOfYearGrid bakes the 12×31 calendar grid into the dayofyear
// scene bg. Cells are 18×18 with a 2px gap (stride 20); the grid origin
// is (130, 750), occupying x=130..750 / y=750..990. Month labels go in
// the left margin at x=60.
//
// Cell painting priority (highest first):
//   1. phantom (dayOfMonth > days in month): cBgHard — invisible.
//   2. special date: cRed fill + letter centred in cFg.
//   3. today: 2px cYellow border around the cell, fill underneath
//      stays whatever past/future colour applies.
//   4. past: cOrange fill.
//   5. future: cBgDarker fill.
// Today + special: cRed fill + letter + cYellow border (both signals).
func drawDayOfYearGrid(img *image.RGBA, now time.Time, specialDates map[int]rune) {
	const (
		gridX = 130
		gridY = 750
		cell  = 18
		gap   = 2
		// Stride: cell + gap = 20.
	)
	stride := cell + gap
	today := now

	// Month labels — fontMono 16pt cFgDark, vertically centred on each row.
	monthFace, _ := loadFace("Iosevka-Regular.ttf", 16)
	if monthFace != nil {
		defer monthFace.Close()
	}
	letterFace, _ := loadFace("Iosevka-Regular.ttf", 14)
	if letterFace != nil {
		defer letterFace.Close()
	}

	monthAbbrev := []string{
		"JAN", "FEB", "MAR", "APR", "MAY", "JUN",
		"JUL", "AUG", "SEP", "OCT", "NOV", "DEC",
	}

	for monthIdx := 0; monthIdx < 12; monthIdx++ {
		month := monthIdx + 1
		cellY := gridY + monthIdx*stride
		if monthFace != nil {
			// Baseline ~4 px down from cell top so the cap-line sits
			// roughly centred against the 18px cell.
			drawLabelLeft(img, monthAbbrev[monthIdx], monthFace, 60, cellY+cell-4, GruvFgDark)
		}
		for d := 1; d <= 31; d++ {
			cellX := gridX + (d-1)*stride
			rect := image.Rect(cellX, cellY, cellX+cell, cellY+cell)
			state := dayOfYearCellStateFor(month, d, today, specialDates)
			switch state {
			case dayOfYearPhantom:
				draw.Draw(img, rect, &image.Uniform{GruvBgHard}, image.Point{}, draw.Src)
			case dayOfYearSpecial:
				draw.Draw(img, rect, &image.Uniform{GruvRed}, image.Point{}, draw.Src)
				if letterFace != nil {
					letter := string(specialDates[month*100+d])
					drawLabelCentered(img, letter, letterFace, cellX+cell/2, cellY+cell-4, GruvFg)
				}
				if month == int(today.Month()) && d == today.Day() {
					drawCellBorder(img, rect, 2, GruvYellow)
				}
			case dayOfYearToday:
				// Paint the underlying past/future fill, then border.
				underlying := GruvBgDarker
				tMonth := int(today.Month())
				if month < tMonth || (month == tMonth && d < today.Day()) {
					underlying = GruvOrange
				}
				draw.Draw(img, rect, &image.Uniform{underlying}, image.Point{}, draw.Src)
				drawCellBorder(img, rect, 2, GruvYellow)
			case dayOfYearPast:
				draw.Draw(img, rect, &image.Uniform{GruvOrange}, image.Point{}, draw.Src)
			case dayOfYearFuture:
				draw.Draw(img, rect, &image.Uniform{GruvBgDarker}, image.Point{}, draw.Src)
			}
		}
	}
}

// loadFace loads a TTF from fonts/ at the given point size, returning
// nil on failure (callers skip the label rather than fail the whole
// render — same defensive pattern the chrome painters use).
func loadFace(filename string, size float64) (font.Face, error) {
	f, err := LoadFont(filename)
	if err != nil {
		slog.Warn("dayofyear grid: font load failed", "file", filename, "err", err)
		return nil, err
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size: size, DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		slog.Warn("dayofyear grid: face init failed", "file", filename, "err", err)
		return nil, err
	}
	return face, nil
}

// drawCellBorder paints a `thick`-pixel border in colour c around rect.
// Used for the today-cell highlight.
func drawCellBorder(img *image.RGBA, rect image.Rectangle, thick int, c color.RGBA) {
	u := &image.Uniform{c}
	// Top
	draw.Draw(img, image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+thick), u, image.Point{}, draw.Src)
	// Bottom
	draw.Draw(img, image.Rect(rect.Min.X, rect.Max.Y-thick, rect.Max.X, rect.Max.Y), u, image.Point{}, draw.Src)
	// Left
	draw.Draw(img, image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+thick, rect.Max.Y), u, image.Point{}, draw.Src)
	// Right
	draw.Draw(img, image.Rect(rect.Max.X-thick, rect.Min.Y, rect.Max.X, rect.Max.Y), u, image.Point{}, draw.Src)
}

func encodeImage(img *image.RGBA, format Format) ([]byte, error) {
	var buf bytes.Buffer
	switch format {
	case FormatPNG:
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	case FormatJPEG:
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported format %d", format)
	}
	return buf.Bytes(), nil
}

// drawSceneGlyph paints a chunky scene-specific shape in the bottom-right
// corner. Bigger and more visible than the previous centered version (per
// Ash's design feedback) — the glyph is now ambient decoration, not a
// subliminal hint. Color is gruvbox bg-darker (~step lighter than bg-hard)
// so the shape is clearly present without overpowering the text.
func drawSceneGlyph(img *image.RGBA, scene Scene) {
	drawSceneGlyphAt(img, scene, CanvasW-180, CanvasH-240)
}

// drawSceneGlyphAt paints the scene's glyph centred on (cx, cy). The
// public-API drawSceneGlyph wraps this with the long-standing bottom-
// right anchor; the three-family quote redesign uses this directly via
// glyphAnchorFor to move the glyph out from under family chrome.
// SceneGlyphColor is the canonical paint colour for every scene's
// corner glyph. **Rule:** every case in drawSceneGlyphAt MUST paint
// in this colour (or via the local `c` parameter that's seeded from
// it) — no per-scene overrides. The family's visual weight depends
// on every glyph reading at the same darkness against
// gruvbox-dark-hard; one bright outlier breaks the gestalt. If a
// scene needs a different visual treatment, redesign the glyph
// shape, not the colour.
//
// Declared as `var` because color.RGBA is a struct (Go forbids
// struct consts), but treat it as a constant — never reassign it.
var SceneGlyphColor = GruvBgDarker

// drawSceneGlyphAt paints a scene's identifying corner glyph at
// (cx, cy). Every case MUST paint in `c` (the SceneGlyphColor) so
// the family of scene glyphs reads at consistent visual weight.
// See the SceneGlyphColor docstring for the rationale.
func drawSceneGlyphAt(img *image.RGBA, scene Scene, cx, cy int) {
	c := SceneGlyphColor

	switch scene {
	case SceneMarkets:
		// Bar chart: 4 bars rising left-to-right, anchored to a baseline.
		const barW, gap = 50, 22
		baseY := cy + 130
		heights := []int{90, 160, 120, 200}
		totalW := 4*barW + 3*gap
		startX := cx - totalW/2
		for i, h := range heights {
			x := startX + i*(barW+gap)
			draw.Draw(img,
				image.Rect(x, baseY-h, x+barW, baseY),
				&image.Uniform{c}, image.Point{}, draw.Src)
		}

	case SceneHN:
		// Bold blocky "Y" — Y Combinator-inspired, reading as the
		// "Hacker News" home. Two angled arms meeting a vertical stem
		// at the centre of the glyph. Built as three filled polygons
		// (left arm, right arm, stem) so the joints meet cleanly.
		const (
			armLen   = 90  // length of each diagonal arm
			armThick = 26  // arm thickness (perpendicular)
			stemH    = 100 // vertical stem length below the junction
			stemW    = 26  // stem width
			junctY   = -20 // y offset of the arm/stem junction from cy
		)
		jx, jy := cx, cy+junctY
		// Left arm — diagonal rectangle from upper-left down to the
		// junction. Rasterise as a parallelogram via fillPolygon.
		fillPolygon(img, []struct{ x, y int }{
			{jx - armLen, jy - armLen - armThick/2},
			{jx - armLen + armThick, jy - armLen - armThick/2},
			{jx + armThick/2, jy},
			{jx - armThick/2, jy},
		}, c)
		// Right arm — mirror of the left.
		fillPolygon(img, []struct{ x, y int }{
			{jx + armLen - armThick, jy - armLen - armThick/2},
			{jx + armLen, jy - armLen - armThick/2},
			{jx + armThick/2, jy},
			{jx - armThick/2, jy},
		}, c)
		// Vertical stem dropping from the junction.
		draw.Draw(img,
			image.Rect(jx-stemW/2, jy, jx+stemW/2, jy+stemH),
			&image.Uniform{c}, image.Point{}, draw.Src)

	case SceneBabylon5:
		// "Babylon 5" 1994 title-card wordmark — large numeral 5 with
		// BABYLON across it. Rasterised from a PD-shape SVG on Wikimedia
		// Commons (see assets.go) and overpainted in c via the same
		// mask-paint pattern as the Starfleet delta. Replaces the older
		// hand-rasterised side-view station silhouette, which was
		// readable to fans but very simplified.
		drawBabylon5(img, cx, cy, c)

	case SceneDiscworld:
		// Great A'Tuin silhouette: the world turtle carrying the four
		// world elephants who in turn carry the flat Disc. Sourced from
		// a hand-composed PD-shape SVG (see assets.go) and overpainted
		// in c via the same mask-paint pattern as the Starfleet delta.
		// Replaces the older fillEgg+rect composition, which read as
		// "blob/bars/blob" at glyph scale rather than the iconic stack.
		drawDiscworld(img, cx, cy, c)

	case SceneJargon:
		// Curly braces { } framing an empty middle — programmer/lexicon
		// motif for the Jargon File. Each brace is a vertical bar with
		// short flanges at the top, middle, and bottom; the middle
		// flange points inward (toward the gap) so the pair reads as
		// the typographic curly. Built from rectangles plus small discs
		// at the tips for softly-rounded corners.
		const (
			braceH      = 160 // total brace height
			barW        = 12  // vertical bar thickness
			flangeW     = 28  // horizontal flange length
			flangeH     = 12  // horizontal flange thickness
			gap         = 90  // gap between the two braces
			tipR        = 6   // rounding-disc radius at the brace tips
		)
		braceTop := cy - braceH/2
		braceBot := cy + braceH/2
		// Left brace { — bar inset from the right, flanges pointing left
		// at top + bottom, and a middle-left bump.
		lBarX := cx - gap/2 - barW
		draw.Draw(img,
			image.Rect(lBarX, braceTop, lBarX+barW, braceBot),
			&image.Uniform{c}, image.Point{}, draw.Src)
		// Top flange pointing left.
		draw.Draw(img,
			image.Rect(lBarX-flangeW, braceTop, lBarX, braceTop+flangeH),
			&image.Uniform{c}, image.Point{}, draw.Src)
		// Bottom flange pointing left.
		draw.Draw(img,
			image.Rect(lBarX-flangeW, braceBot-flangeH, lBarX, braceBot),
			&image.Uniform{c}, image.Point{}, draw.Src)
		// Middle-left bump.
		draw.Draw(img,
			image.Rect(lBarX-flangeW, cy-flangeH/2, lBarX, cy+flangeH/2),
			&image.Uniform{c}, image.Point{}, draw.Src)
		// Tip rounding.
		fillCircle(img, lBarX-flangeW, braceTop+flangeH/2, tipR, c)
		fillCircle(img, lBarX-flangeW, braceBot-flangeH/2, tipR, c)
		fillCircle(img, lBarX-flangeW, cy, tipR, c)

		// Right brace } — mirrored.
		rBarX := cx + gap/2
		draw.Draw(img,
			image.Rect(rBarX, braceTop, rBarX+barW, braceBot),
			&image.Uniform{c}, image.Point{}, draw.Src)
		draw.Draw(img,
			image.Rect(rBarX+barW, braceTop, rBarX+barW+flangeW, braceTop+flangeH),
			&image.Uniform{c}, image.Point{}, draw.Src)
		draw.Draw(img,
			image.Rect(rBarX+barW, braceBot-flangeH, rBarX+barW+flangeW, braceBot),
			&image.Uniform{c}, image.Point{}, draw.Src)
		draw.Draw(img,
			image.Rect(rBarX+barW, cy-flangeH/2, rBarX+barW+flangeW, cy+flangeH/2),
			&image.Uniform{c}, image.Point{}, draw.Src)
		fillCircle(img, rBarX+barW+flangeW, braceTop+flangeH/2, tipR, c)
		fillCircle(img, rBarX+barW+flangeW, braceBot-flangeH/2, tipR, c)
		fillCircle(img, rBarX+barW+flangeW, cy, tipR, c)

	case SceneCatFacts:
		// Cat — Twemoji full-body cat (U+1F408) silhouette; replaces
		// the hand-rasterised cat-from-behind which read as a fat
		// pikachu blob rather than a recognisable cat.
		drawCat(img, cx, cy, c)

	case SceneDidYouKnow:
		// Bold typographic "?" — sourced from the Twemoji ❔ (U+2754) PNG
		// mask (see assets.go) and overpainted in c, matching the
		// mask-driven pattern used by the Starfleet delta / buddha /
		// weather icons.
		drawQuestion(img, cx, cy, c)

	case SceneReddit:
		// Twemoji thumbs-up (U+1F44D) — distinctive fist+thumb
		// silhouette that reads as "upvote" without involving
		// Reddit's trademarked Snoo. Replaces the earlier alien-face
		// mask, which rendered as a featureless teardrop at
		// silhouette-only.
		drawThumbsup(img, cx, cy, c)

	case SceneTIL:
		// Lightbulb (idea / new knowledge) — sourced from the Heroicons
		// light-bulb solid SVG (see assets.go) and overpainted in c,
		// matching the mask-driven pattern used by the Starfleet delta /
		// buddha / question icons.
		drawTIL(img, cx, cy, c)

	case SceneSunrise:
		// Sun cresting a horizon: a long thin horizon bar, a sun disc
		// whose bottom half is carved out by a bg-hard rectangle so it
		// reads as half-risen, and a halo of ray discs flicking up
		// from the top arc. All painted in c (SceneGlyphColor) so the
		// sun reads at the same darkness as every other corner glyph —
		// the family-rule, not a per-scene override.
		const (
			horizonHalfW = 160 // half-length of the horizon bar (was 100)
			horizonH     = 8   // horizon bar thickness (was 6)
			sunR         = 105 // sun radius (was 60; ~1.75× to earn its space)
			sunCyOff     = -14 // sun centre relative to cy (sits just above horizon)
			rayR         = 14  // ray disc radius (was 9)
		)
		// Horizon bar — centred on cy.
		draw.Draw(img,
			image.Rect(cx-horizonHalfW, cy-horizonH/2, cx+horizonHalfW, cy+horizonH/2),
			&image.Uniform{c}, image.Point{}, draw.Src)
		// Sun disc, then carve away everything at or below the horizon
		// line so only the upper half stays visible.
		sunCy := cy + sunCyOff
		fillCircle(img, cx, sunCy, sunR, c)
		draw.Draw(img,
			image.Rect(cx-sunR-2, cy-horizonH/2, cx+sunR+2, cy+sunR+sunCyOff+2),
			&image.Uniform{GruvBgHard}, image.Point{}, draw.Src)
		// Five rays flicking up around the sun's top arc.
		for _, ray := range []image.Point{
			{X: cx - 150, Y: sunCy - 70},
			{X: cx - 80, Y: sunCy - 130},
			{X: cx, Y: sunCy - 150},
			{X: cx + 80, Y: sunCy - 130},
			{X: cx + 150, Y: sunCy - 70},
		} {
			fillCircle(img, ray.X, ray.Y, rayR, c)
		}

	case SceneStarTrek:
		// Starfleet delta insignia. The canonical silhouette is rasterised
		// from an embedded PNG mask (derived from a PD SVG — see
		// assets.go) rather than hand-coded, so the shape matches the
		// real emblem. Every opaque pixel of the mask is painted in c.
		drawStarfleetDelta(img, cx, cy, c)

	case SceneZenQuotes:
		// Lotus flower (🪷) — the canonical zen-meditation mark. Replaces
		// the earlier 🧘 figure, which read as a scrawny stick at
		// silhouette. The lotus is chunky and symmetric, so it stays
		// legible at the family's dim SceneGlyphColor without needing
		// a brightness override — exactly why we picked it over the
		// meditator silhouette.
		drawLotus(img, cx, cy, c)

	case SceneNASA:
		// Saturn-with-ring: a filled planet disc plus a thin elliptical
		// ring around it. The ring is built from two concentric flat
		// ellipses with the inner one carved back out in bg-hard, then
		// the planet body painted on top so the ring appears to pass
		// behind it. Reads as the iconic "space photography" motif for
		// NASA's APOD without needing a recognisable spiral.
		const (
			planetR     = 70  // planet body radius
			ringRX      = 150 // ring horizontal radius
			ringRY      = 26  // ring vertical radius (flattened ellipse)
			ringThick   = 10  // ring band thickness
		)
		// Outer ring fill, then inner ellipse carved away so only a
		// band remains.
		fillEgg(img, cx, cy, ringRX, ringRY, ringRY, c)
		fillEgg(img, cx, cy, ringRX-ringThick, ringRY-ringThick, ringRY-ringThick, GruvBgHard)
		// Planet body on top, occluding the front half of the ring.
		fillCircle(img, cx, cy, planetR, c)

	case SceneCocktail:
		// Martini glass silhouette: a downward-pointing triangular bowl
		// sitting on a short vertical stem and a wide flat base. Triangle
		// is drawn as a scanline-narrowing rectangle stack from wide top
		// to a single-pixel apex. Total height ~180 px.
		const (
			bowlTopHalfW = 90 // half-width of the triangle bowl at the top
			bowlH        = 110
			stemW        = 10
			stemH        = 50
			baseHalfW    = 50
			baseH        = 10
		)
		// Bowl: scan from top down, narrowing toward the apex. The bowl's
		// top sits at cy-bowlH/2, apex at cy+bowlH/2.
		bowlTop := cy - bowlH/2
		for y := 0; y < bowlH; y++ {
			// Fraction of remaining width as we descend toward the apex.
			frac := float64(bowlH-y) / float64(bowlH)
			halfW := int(float64(bowlTopHalfW) * frac)
			if halfW < 1 {
				continue
			}
			draw.Draw(img,
				image.Rect(cx-halfW, bowlTop+y, cx+halfW, bowlTop+y+1),
				&image.Uniform{c}, image.Point{}, draw.Src)
		}
		// Stem dropping from the bowl's apex.
		stemTop := cy + bowlH/2
		draw.Draw(img,
			image.Rect(cx-stemW/2, stemTop, cx+stemW/2, stemTop+stemH),
			&image.Uniform{c}, image.Point{}, draw.Src)
		// Base: wide flat rectangle under the stem.
		baseTop := stemTop + stemH
		draw.Draw(img,
			image.Rect(cx-baseHalfW, baseTop, cx+baseHalfW, baseTop+baseH),
			&image.Uniform{c}, image.Point{}, draw.Src)

	case SceneDevil:
		// Imp / horned-devil head (👿). Same mask-overpaint treatment as
		// the Starfleet delta and buddha; source is a Twemoji SVG (see
		// assets.go). Reads as the cover motif of Bierce's Devil's
		// Dictionary.
		drawDevil(img, cx, cy, c)

	case SceneGitHub:
		// Git branch-diamond from Bootstrap Icons (see assets.go). Same
		// mask-overpaint treatment as the Starfleet delta. Reads as
		// "version control" without invoking GitHub's trademarked octocat.
		// Painted ~90px lower than the canonical corner anchor because
		// the github scene's column-label row sits at y=950 baseline;
		// the default anchor at cy=CanvasH-240 (1040) would put the
		// glyph's top edge at y=940 and clip the labels.
		drawGit(img, cx, cy+90, c)

	case SceneISS:
		// No corner glyph for the ISS scene — the baked world map + dynamic
		// dot in the body IS the visualisation, and a corner satellite
		// would compete with it. Handled fully by DrawISSChrome.

	case SceneStoics:
		// Greek column: square capital + plinth at the top, fluted shaft
		// below, square plinth at the base. Three stacked rectangles
		// roughly proportioned 1 : 4 : 1 in height so the silhouette
		// reads as a classical pillar.
		const (
			capW   = 110 // top capital / base block width
			capH   = 22
			shaftW = 80
			shaftH = 180
		)
		// Capital (top).
		draw.Draw(img,
			image.Rect(cx-capW/2, cy-shaftH/2-capH, cx+capW/2, cy-shaftH/2),
			&image.Uniform{c}, image.Point{}, draw.Src)
		// Shaft.
		draw.Draw(img,
			image.Rect(cx-shaftW/2, cy-shaftH/2, cx+shaftW/2, cy+shaftH/2),
			&image.Uniform{c}, image.Point{}, draw.Src)
		// Base (bottom).
		draw.Draw(img,
			image.Rect(cx-capW/2, cy+shaftH/2, cx+capW/2, cy+shaftH/2+capH),
			&image.Uniform{c}, image.Point{}, draw.Src)

	case SceneTwain:
		// Twemoji ship (U+1F6A2) — riverboat silhouette is the
		// canonical Twain reference ("Mark Twain" is a steamboat
		// sounding-call for two fathoms). Replaces the hand-
		// rasterised pen-stroke slash, which read as a meaningless
		// diagonal mark at glance distance.
		drawSteamboat(img, cx, cy, c)

	case SceneFortune:
		// Twemoji fortune cookie (U+1F960) — replaces the hand-
		// rasterised asymmetric-crescent which read as a generic
		// blob rather than a cookie at silhouette.
		drawFortuneCookie(img, cx, cy, c)

	case SceneOnThisDay:
		// Tear-off calendar page — "an entry in the historical record"
		// motif. A clock signified "current time," which the always-on
		// header already shows; the calendar page signifies a dated
		// event without overlapping with the live clock. Two binder
		// ring tabs poke above a chunky top header band; below sits a
		// hollow page body with a single ruled line suggesting "an
		// entry written here." Total footprint ~200×200, no date
		// glyphs baked in so the bg never needs re-upload.
		const (
			pageHalfW = 90 // half-width of the page rectangle
			pageTop   = -100
			pageBot   = 100
			headerH   = 36 // height of the filled top header band
			ringW     = 14
			ringH     = 18
			ringInset = 32 // horizontal offset of each ring from cx
			borderThk = 6  // page-body border thickness
			ruleY     = 30 // y-offset of the single ruled "entry" line
			ruleHalfW = 50
			ruleThk   = 6
		)
		// Two binder ring tabs sitting above the header band.
		for _, dx := range []int{-ringInset, ringInset} {
			draw.Draw(img,
				image.Rect(cx+dx-ringW/2, cy+pageTop-ringH, cx+dx+ringW/2, cy+pageTop),
				&image.Uniform{c}, image.Point{}, draw.Src)
		}
		// Filled top header band.
		draw.Draw(img,
			image.Rect(cx-pageHalfW, cy+pageTop, cx+pageHalfW, cy+pageTop+headerH),
			&image.Uniform{c}, image.Point{}, draw.Src)
		// Page body: filled rect, then carve out the inside so only a
		// thick border remains (same outer-minus-inner pattern as the
		// clock ring this replaced).
		draw.Draw(img,
			image.Rect(cx-pageHalfW, cy+pageTop+headerH, cx+pageHalfW, cy+pageBot),
			&image.Uniform{c}, image.Point{}, draw.Src)
		draw.Draw(img,
			image.Rect(
				cx-pageHalfW+borderThk, cy+pageTop+headerH+borderThk,
				cx+pageHalfW-borderThk, cy+pageBot-borderThk,
			),
			&image.Uniform{GruvBgHard}, image.Point{}, draw.Src)
		// Single ruled "entry" line inside the page body.
		draw.Draw(img,
			image.Rect(cx-ruleHalfW, cy+ruleY-ruleThk/2, cx+ruleHalfW, cy+ruleY+ruleThk/2),
			&image.Uniform{c}, image.Point{}, draw.Src)

	case SceneWordnik:
		// Open book (📖). Same mask-overpaint treatment as the devil /
		// buddha / question / hazard glyphs; source is a Twemoji SVG
		// (see assets.go). Reads as the dictionary motif for the
		// Word of the Day scene.
		drawBook(img, cx, cy, c)

	}
}

// starfleetDeltaMask is the decoded silhouette PNG; loaded once on first
// use. Pixels with alpha above starfleetDeltaAlphaThreshold count as part
// of the shape.
var (
	starfleetDeltaOnce sync.Once
	starfleetDeltaMask image.Image
)

const starfleetDeltaAlphaThreshold = 128

// drawStarfleetDelta paints the Starfleet insignia silhouette centred on
// (cx, cy). The shape comes from an embedded PNG mask (see assets.go);
// every pixel of the mask whose alpha exceeds the threshold is painted
// in c. Pixels that fall outside img's bounds are skipped.
func drawStarfleetDelta(img *image.RGBA, cx, cy int, c color.RGBA) {
	starfleetDeltaOnce.Do(func() {
		m, err := png.Decode(bytes.NewReader(starfleetDeltaPNG))
		if err != nil {
			// The PNG is embedded at build time; a decode failure means
			// the asset is corrupt, which is a programmer error.
			panic(fmt.Errorf("render: decode embedded starfleet delta: %w", err))
		}
		starfleetDeltaMask = m
	})
	paintMask(img, starfleetDeltaMask, cx, cy, c)
}

// buddhaMask is the decoded meditating-figure silhouette PNG; loaded
// once on first use. Same alpha-threshold treatment as the starfleet
// delta.
var (
	buddhaOnce sync.Once
	buddhaMask image.Image
)

// drawBuddha paints the meditating-figure silhouette centred on (cx, cy)
// in colour c. Mirror of drawStarfleetDelta — embedded PNG decoded once,
// then handed to paintMask.
func drawBuddha(img *image.RGBA, cx, cy int, c color.RGBA) {
	buddhaOnce.Do(func() {
		m, err := png.Decode(bytes.NewReader(buddhaPNG))
		if err != nil {
			panic(fmt.Errorf("render: decode embedded buddha: %w", err))
		}
		buddhaMask = m
	})
	paintMask(img, buddhaMask, cx, cy, c)
}

// lotusMask is the decoded lotus-flower silhouette PNG; loaded once on
// first use. Replaces the buddha glyph for SceneZenQuotes — see the
// SceneZenQuotes branch of drawSceneGlyphAt for why.
var (
	lotusOnce sync.Once
	lotusMask image.Image
)

// drawLotus paints the lotus-flower silhouette centred on (cx, cy) in
// colour c. Mirror of drawBuddha — embedded PNG decoded once, then
// handed to paintMask.
func drawLotus(img *image.RGBA, cx, cy int, c color.RGBA) {
	lotusOnce.Do(func() {
		m, err := png.Decode(bytes.NewReader(lotusPNG))
		if err != nil {
			panic(fmt.Errorf("render: decode embedded lotus: %w", err))
		}
		lotusMask = m
	})
	paintMask(img, lotusMask, cx, cy, c)
}

// devilMask is the decoded imp-face silhouette PNG; loaded once on first
// use. Same alpha-threshold treatment as the starfleet delta.
var (
	devilOnce sync.Once
	devilMask image.Image
)

// drawDevil paints the imp-face silhouette centred on (cx, cy) in colour
// c. Mirror of drawBuddha — embedded PNG decoded once, then handed to
// paintMask.
func drawDevil(img *image.RGBA, cx, cy int, c color.RGBA) {
	devilOnce.Do(func() {
		m, err := png.Decode(bytes.NewReader(devilPNG))
		if err != nil {
			panic(fmt.Errorf("render: decode embedded devil: %w", err))
		}
		devilMask = m
	})
	paintMask(img, devilMask, cx, cy, c)
}

// catMask is the decoded Twemoji full-body cat silhouette (U+1F408),
// loaded once on first use. Replaces the hand-rasterised cat-from-
// behind drawCatSilhouette which read as a round blob.
var (
	catOnce sync.Once
	catMask image.Image
)

// drawCat paints the cat silhouette centred on (cx, cy) in colour c.
func drawCat(img *image.RGBA, cx, cy int, c color.RGBA) {
	catOnce.Do(func() {
		m, err := png.Decode(bytes.NewReader(catPNG))
		if err != nil {
			panic(fmt.Errorf("render: decode embedded cat: %w", err))
		}
		catMask = m
	})
	paintMask(img, catMask, cx, cy, c)
}

// fortuneCookieMask is the decoded Twemoji fortune-cookie silhouette
// (U+1F960), loaded once on first use. Replaces the hand-rasterised
// asymmetric-crescent that didn't read as a cookie at silhouette.
var (
	fortuneCookieOnce sync.Once
	fortuneCookieMask image.Image
)

// drawFortuneCookie paints the fortune-cookie silhouette centred on
// (cx, cy) in colour c.
func drawFortuneCookie(img *image.RGBA, cx, cy int, c color.RGBA) {
	fortuneCookieOnce.Do(func() {
		m, err := png.Decode(bytes.NewReader(fortuneCookiePNG))
		if err != nil {
			panic(fmt.Errorf("render: decode embedded fortune cookie: %w", err))
		}
		fortuneCookieMask = m
	})
	paintMask(img, fortuneCookieMask, cx, cy, c)
}

// thumbsupMask is the decoded Twemoji thumbs-up silhouette
// (U+1F44D), loaded once on first use. Replaces the alien-face mask
// for the reddit scene — the alien rendered as a featureless teardrop
// at silhouette; the thumbs-up has a distinctive fist+thumb outline
// that reads as "upvote" without involving Reddit's trademarked Snoo.
var (
	thumbsupOnce sync.Once
	thumbsupMask image.Image
)

// drawThumbsup paints the thumbs-up silhouette centred on (cx, cy)
// in colour c.
func drawThumbsup(img *image.RGBA, cx, cy int, c color.RGBA) {
	thumbsupOnce.Do(func() {
		m, err := png.Decode(bytes.NewReader(thumbsupPNG))
		if err != nil {
			panic(fmt.Errorf("render: decode embedded thumbsup: %w", err))
		}
		thumbsupMask = m
	})
	paintMask(img, thumbsupMask, cx, cy, c)
}

// steamboatMask is the decoded Twemoji ship silhouette (U+1F6A2),
// loaded once on first use. Used by SceneTwain — riverboat is the
// canonical Twain reference ("Mark Twain" is a steamboat sounding-
// call). Replaces the hand-rasterised pen-stroke slash that read
// as a confusing slash mark.
var (
	steamboatOnce sync.Once
	steamboatMask image.Image
)

// drawSteamboat paints the ship silhouette centred on (cx, cy) in
// colour c.
func drawSteamboat(img *image.RGBA, cx, cy int, c color.RGBA) {
	steamboatOnce.Do(func() {
		m, err := png.Decode(bytes.NewReader(steamboatPNG))
		if err != nil {
			panic(fmt.Errorf("render: decode embedded steamboat: %w", err))
		}
		steamboatMask = m
	})
	paintMask(img, steamboatMask, cx, cy, c)
}

// alienMask is the decoded Twemoji alien-face silhouette PNG (U+1F47D),
// loaded once on first use. Stands in for the reddit Snoo mascot in the
// reddit scene — Twemoji is CC BY 4.0 + matches the existing mask-paint
// pattern; Snoo proper is trademarked.
var (
	alienOnce sync.Once
	alienMask image.Image
)

// drawAlien paints the alien-face silhouette centred on (cx, cy) in
// colour c. Same shape as drawDevil / drawBuddha — embedded PNG decoded
// once, then handed to paintMask.
func drawAlien(img *image.RGBA, cx, cy int, c color.RGBA) {
	alienOnce.Do(func() {
		m, err := png.Decode(bytes.NewReader(alienPNG))
		if err != nil {
			panic(fmt.Errorf("render: decode embedded alien: %w", err))
		}
		alienMask = m
	})
	paintMask(img, alienMask, cx, cy, c)
}

// questionMask is the decoded question-mark silhouette PNG; loaded once
// on first use. Same alpha-threshold treatment as the starfleet delta.
var (
	questionOnce sync.Once
	questionMask image.Image
)

// drawQuestion paints the question-mark silhouette centred on (cx, cy)
// in colour c. Mirror of drawDevil — embedded PNG decoded once, then
// handed to paintMask.
func drawQuestion(img *image.RGBA, cx, cy int, c color.RGBA) {
	questionOnce.Do(func() {
		m, err := png.Decode(bytes.NewReader(questionPNG))
		if err != nil {
			panic(fmt.Errorf("render: decode embedded question: %w", err))
		}
		questionMask = m
	})
	paintMask(img, questionMask, cx, cy, c)
}

// gitMask is the decoded git branch-diamond silhouette PNG; loaded once
// on first use. Same alpha-threshold treatment as the starfleet delta.
var (
	gitOnce sync.Once
	gitMask image.Image
)

// drawGit paints the git branch-diamond silhouette centred on (cx, cy)
// in colour c. Mirror of drawQuestion — embedded PNG decoded once, then
// handed to paintMask.
func drawGit(img *image.RGBA, cx, cy int, c color.RGBA) {
	gitOnce.Do(func() {
		m, err := png.Decode(bytes.NewReader(gitPNG))
		if err != nil {
			panic(fmt.Errorf("render: decode embedded git: %w", err))
		}
		gitMask = m
	})
	paintMask(img, gitMask, cx, cy, c)
}

// babylon5Mask is the decoded "Babylon 5" wordmark silhouette PNG;
// loaded once on first use. Same alpha-threshold treatment as the
// starfleet delta.
var (
	babylon5Once sync.Once
	babylon5Mask image.Image
)

// drawBabylon5 paints the Babylon 5 title-card wordmark centred on
// (cx, cy) in colour c. Mirror of drawQuestion — embedded PNG decoded
// once, then handed to paintMask.
func drawBabylon5(img *image.RGBA, cx, cy int, c color.RGBA) {
	babylon5Once.Do(func() {
		m, err := png.Decode(bytes.NewReader(babylon5PNG))
		if err != nil {
			panic(fmt.Errorf("render: decode embedded babylon5: %w", err))
		}
		babylon5Mask = m
	})
	paintMask(img, babylon5Mask, cx, cy, c)
}

// discworldMask is the decoded Great A'Tuin / elephants / disc
// silhouette PNG; loaded once on first use. Same alpha-threshold
// treatment as the starfleet delta.
var (
	discworldOnce sync.Once
	discworldMask image.Image
)

// drawDiscworld paints the Discworld cosmology silhouette (turtle +
// four elephants + disc) centred on (cx, cy) in colour c. Mirror of
// drawBabylon5 — embedded PNG decoded once, then handed to paintMask.
func drawDiscworld(img *image.RGBA, cx, cy int, c color.RGBA) {
	discworldOnce.Do(func() {
		m, err := png.Decode(bytes.NewReader(discworldPNG))
		if err != nil {
			panic(fmt.Errorf("render: decode embedded discworld: %w", err))
		}
		discworldMask = m
	})
	paintMask(img, discworldMask, cx, cy, c)
}

// tilMask is the decoded lightbulb silhouette PNG; loaded once on first
// use. Same alpha-threshold treatment as the starfleet delta.
var (
	tilOnce sync.Once
	tilMask image.Image
)

// drawTIL paints the lightbulb silhouette centred on (cx, cy) in colour
// c. Mirror of drawQuestion — embedded PNG decoded once, then handed to
// paintMask.
func drawTIL(img *image.RGBA, cx, cy int, c color.RGBA) {
	tilOnce.Do(func() {
		m, err := png.Decode(bytes.NewReader(tilPNG))
		if err != nil {
			panic(fmt.Errorf("render: decode embedded til: %w", err))
		}
		tilMask = m
	})
	paintMask(img, tilMask, cx, cy, c)
}

// bookMask is the decoded open-book silhouette PNG; loaded once on first
// use. Same alpha-threshold treatment as the starfleet delta.
var (
	bookOnce sync.Once
	bookMask image.Image
)

// drawBook paints the open-book silhouette centred on (cx, cy) in colour
// c. Mirror of drawQuestion — embedded PNG decoded once, then handed to
// paintMask.
func drawBook(img *image.RGBA, cx, cy int, c color.RGBA) {
	bookOnce.Do(func() {
		m, err := png.Decode(bytes.NewReader(bookPNG))
		if err != nil {
			panic(fmt.Errorf("render: decode embedded book: %w", err))
		}
		bookMask = m
	})
	paintMask(img, bookMask, cx, cy, c)
}

// weatherMasks holds the decoded weather-icon silhouettes, keyed by
// outlook string. Each mask is loaded once on first use; pixels with
// alpha above starfleetDeltaAlphaThreshold count as part of the shape.
var (
	weatherMasksOnce sync.Once
	weatherMasks     map[string]image.Image
)

func loadWeatherMasks() {
	sources := map[string][]byte{
		"clear":    weatherClearPNG,
		"cloudy":   weatherCloudyPNG,
		"overcast": weatherOvercastPNG,
		"rain":     weatherRainPNG,
		"drizzle":  weatherDrizzlePNG,
		"snow":     weatherSnowPNG,
		"fog":      weatherFogPNG,
		"thunder":  weatherThunderPNG,
		"smoke":    weatherSmokePNG,
		"hazard":   hazardPNG,
	}
	weatherMasks = make(map[string]image.Image, len(sources))
	for outlook, raw := range sources {
		m, err := png.Decode(bytes.NewReader(raw))
		if err != nil {
			// Embedded at build time; a decode failure is a programmer
			// error, same as the starfleet delta.
			panic(fmt.Errorf("render: decode embedded weather icon %q: %w", outlook, err))
		}
		weatherMasks[outlook] = m
	}
}

// drawWeatherGlyph paints the icon for `outlook` in the bottom-right
// corner using the same mask-and-overpaint approach as the Starfleet
// delta. Unknown outlooks fall back to the cloudy icon so the frame
// always renders something. Colour is gruvbox bg-darker (ambient).
func drawWeatherGlyph(img *image.RGBA, outlook string) {
	const (
		cx = CanvasW - 180
		cy = CanvasH - 240
	)
	weatherMasksOnce.Do(loadWeatherMasks)
	mask, ok := weatherMasks[outlook]
	if !ok {
		mask = weatherMasks["cloudy"]
	}
	paintMask(img, mask, cx, cy, GruvBgDarker)
}

// paintMaskScaled is paintMask's resampling cousin: it paints the mask
// scaled to (dstW × dstH) pixels, centred on (cx, cy), in colour c.
// Sampling is nearest-neighbour — the source masks are silhouettes
// where alpha is effectively binary, so fancier filters add nothing
// the threshold won't immediately quantise back out.
func paintMaskScaled(img *image.RGBA, mask image.Image, cx, cy, dstW, dstH int, c color.RGBA) {
	mb := mask.Bounds()
	mw, mh := mb.Dx(), mb.Dy()
	if mw == 0 || mh == 0 || dstW <= 0 || dstH <= 0 {
		return
	}
	originX := cx - dstW/2
	originY := cy - dstH/2
	bounds := img.Bounds()
	for py := 0; py < dstH; py++ {
		dy := originY + py
		if dy < bounds.Min.Y || dy >= bounds.Max.Y {
			continue
		}
		sy := py * mh / dstH
		for px := 0; px < dstW; px++ {
			sx := px * mw / dstW
			_, _, _, a := mask.At(mb.Min.X+sx, mb.Min.Y+sy).RGBA()
			if a>>8 <= starfleetDeltaAlphaThreshold {
				continue
			}
			dx := originX + px
			if dx < bounds.Min.X || dx >= bounds.Max.X {
				continue
			}
			img.SetRGBA(dx, dy, c)
		}
	}
}

// paintMask paints every above-threshold pixel of mask into img, centred
// on (cx, cy), in colour c. Pixels falling outside img's bounds are
// skipped. Shared by the starfleet delta and weather glyphs.
func paintMask(img *image.RGBA, mask image.Image, cx, cy int, c color.RGBA) {
	mb := mask.Bounds()
	mw, mh := mb.Dx(), mb.Dy()
	originX := cx - mw/2
	originY := cy - mh/2
	bounds := img.Bounds()
	for py := 0; py < mh; py++ {
		dy := originY + py
		if dy < bounds.Min.Y || dy >= bounds.Max.Y {
			continue
		}
		for px := 0; px < mw; px++ {
			_, _, _, a := mask.At(mb.Min.X+px, mb.Min.Y+py).RGBA()
			if a>>8 <= starfleetDeltaAlphaThreshold {
				continue
			}
			dx := originX + px
			if dx < bounds.Min.X || dx >= bounds.Max.X {
				continue
			}
			img.SetRGBA(dx, dy, c)
		}
	}
}

// drawCatSilhouette rasterises a sitting-cat-from-behind silhouette: a
// round head with two pointy triangular ears poking up, an oval body
// widening toward a flat base, and a curled tail arcing up the right
// side. The head+body+ears outline is a single closed polygon; the
// tail is a sequence of overlapping discs forming a soft arc so it
// reads as a distinct appendage. w/h set the bounding box in pixels,
// centred on (cx, cy); the polygon is laid out in normalised [-1,1]
// coords (-1 = top/left of box, +1 = bottom/right) and projected to
// pixels at fill time.
func drawCatSilhouette(img *image.RGBA, cx, cy, w, h int, c color.RGBA) {
	// Closed outline, clockwise from the upper-left of the head where
	// the left ear meets the round skull. Coords are normalised:
	// x in [-1,1] across the box, y in [-1,1] top-to-bottom.
	outline := []struct{ x, y float64 }{
		// Top of left ear: outer base on the head, up to the pointy
		// tip, down into the valley between the ears.
		{-0.55, -0.50}, // left ear outer base (on head)
		{-0.65, -0.75},
		{-0.50, -1.00}, // left ear tip
		{-0.30, -0.75},
		{-0.20, -0.55}, // valley between ears
		// Mirror over to the right ear.
		{0.20, -0.55},
		{0.30, -0.75},
		{0.50, -1.00}, // right ear tip
		{0.65, -0.75},
		{0.55, -0.50}, // right ear outer base (on head)
		// Right side of head curving down to shoulder.
		{0.62, -0.35},
		{0.60, -0.15}, // right side of head/neck
		// Shoulder flaring out to the body.
		{0.72, 0.05},
		{0.85, 0.35},
		{0.88, 0.65},
		// Lower-right of the body sloping in to the base.
		{0.78, 0.90},
		{0.65, 1.00}, // base, right corner
		// Across the base.
		{-0.65, 1.00}, // base, left corner
		// Up the left side, mirror of the right.
		{-0.78, 0.90},
		{-0.88, 0.65},
		{-0.85, 0.35},
		{-0.72, 0.05},
		{-0.60, -0.15}, // left side of head/neck
		{-0.62, -0.35},
		// (closes back to the first point)
	}

	hw := float64(w) / 2
	hh := float64(h) / 2
	poly := make([]struct{ x, y int }, len(outline))
	for i, p := range outline {
		poly[i] = struct{ x, y int }{
			x: cx + int(p.x*hw),
			y: cy + int(p.y*hh),
		}
	}
	fillPolygon(img, poly, c)

	// Tail: an arc of overlapping discs sweeping from the lower-right
	// of the body up and curling back over the cat's right hip. Each
	// point is in the same normalised box coords as the outline.
	tail := []struct {
		x, y float64
		r    int
	}{
		{0.95, 0.55, 14},
		{1.00, 0.30, 14},
		{1.00, 0.05, 13},
		{0.95, -0.18, 12},
		{0.82, -0.32, 11},
		{0.65, -0.38, 10},
	}
	for _, t := range tail {
		fillCircle(img,
			cx+int(t.x*hw),
			cy+int(t.y*hh),
			t.r, c)
	}
}

// fillPolygon rasterises a closed polygon by scanline fill. Points are
// integer pixel coords; the polygon is implicitly closed (last vertex
// joins back to the first). Half-open scanline rule on y avoids
// double-counting shared vertices.
func fillPolygon(img *image.RGBA, poly []struct{ x, y int }, c color.RGBA) {
	if len(poly) < 3 {
		return
	}
	bounds := img.Bounds()
	minY, maxY := poly[0].y, poly[0].y
	for _, p := range poly[1:] {
		if p.y < minY {
			minY = p.y
		}
		if p.y > maxY {
			maxY = p.y
		}
	}
	if minY < bounds.Min.Y {
		minY = bounds.Min.Y
	}
	if maxY >= bounds.Max.Y {
		maxY = bounds.Max.Y - 1
	}
	for y := minY; y <= maxY; y++ {
		var xs []int
		for i := 0; i < len(poly); i++ {
			a := poly[i]
			b := poly[(i+1)%len(poly)]
			if a.y == b.y {
				continue
			}
			y0, y1 := a.y, b.y
			x0, x1 := a.x, b.x
			if y0 > y1 {
				y0, y1 = y1, y0
				x0, x1 = x1, x0
			}
			if y < y0 || y >= y1 {
				continue
			}
			t := float64(y-y0) / float64(y1-y0)
			xs = append(xs, x0+int(t*float64(x1-x0)))
		}
		if len(xs) < 2 {
			continue
		}
		for i := 1; i < len(xs); i++ {
			for j := i; j > 0 && xs[j-1] > xs[j]; j-- {
				xs[j-1], xs[j] = xs[j], xs[j-1]
			}
		}
		for i := 0; i+1 < len(xs); i += 2 {
			x0 := xs[i]
			x1 := xs[i+1]
			if x0 < bounds.Min.X {
				x0 = bounds.Min.X
			}
			if x1 >= bounds.Max.X {
				x1 = bounds.Max.X - 1
			}
			for x := x0; x <= x1; x++ {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

// fillCircle is a tiny stdlib-only filled-disc rasterizer. The render
// package doesn't depend on x/image, so we roll a small version here
// rather than pulling in a graphics library for one shape.
func fillCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	bounds := img.Bounds()
	r2 := r * r
	for y := cy - r; y <= cy+r; y++ {
		if y < bounds.Min.Y || y >= bounds.Max.Y {
			continue
		}
		dy := y - cy
		for x := cx - r; x <= cx+r; x++ {
			if x < bounds.Min.X || x >= bounds.Max.X {
				continue
			}
			dx := x - cx
			if dx*dx+dy*dy <= r2 {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func buildHeroImage(now time.Time) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, CanvasW, CanvasH))
	draw.Draw(img, img.Bounds(), &image.Uniform{GruvBgHard}, image.Point{}, draw.Src)

	// Hairline divider at y=460 separating the always-on "clock+date" zone
	// above from the rotating scene area below. Inset 60px from each side
	// so the rule reads as a composition mark, not a horizon line.
	draw.Draw(img, image.Rect(60, 460, CanvasW-60, 462),
		&image.Uniform{GruvBgDarker}, image.Point{}, draw.Src)

	// Year-progress bar along the very bottom edge. Track in bg-darker, fill
	// in orange to the elapsed fraction of the year. Subtle ambient marker
	// of where you are in the year.
	const (
		barH       = 4
		barOffsetY = 8
	)
	trackTop := CanvasH - barOffsetY - barH
	trackBot := CanvasH - barOffsetY
	draw.Draw(img, image.Rect(0, trackTop, CanvasW, trackBot),
		&image.Uniform{GruvBgDarker}, image.Point{}, draw.Src)

	yearDays := 365
	if isLeapYear(now.Year()) {
		yearDays = 366
	}
	frac := float64(now.YearDay()-1) / float64(yearDays)
	filledW := int(frac * float64(CanvasW))
	if filledW > 0 {
		draw.Draw(img, image.Rect(0, trackTop, filledW, trackBot),
			&image.Uniform{GruvOrange}, image.Point{}, draw.Src)
	}

	drawMorseRule(img)

	// Baked "> " prompt to the left of the day-of-week. The day name
	// itself renders as a device Week element (built-in type — saves
	// one Text slot, doesn't count against the 6-Text cap). The prompt
	// glyph stays here in cFgDark mono so the always-on header still
	// reads as "> wednesday" rather than just "Wednesday".
	if f, err := LoadFont("Iosevka-Regular.ttf"); err == nil {
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size: 64, DPI: 72, Hinting: font.HintingFull,
		})
		if err == nil {
			drawLabelLeft(img, "> ", face, 40, 90, GruvFgDark)
			face.Close()
		}
	}

	return img
}

// drawMorseRule paints a dashed separator at y=380 between the time and
// the operator footer. Alternating 16px dashes with 4px gaps; every 5th
// gap is replaced with a 2px dot so the line reads as a deliberate
// rhythm break rather than uniform stippling.
func drawMorseRule(img *image.RGBA) {
	const (
		y       = 380
		thick   = 2
		xStart  = 40
		xEnd    = 760
		dashW   = 16
		gapW    = 4
		dotW    = 2
		dotEvery = 5 // every Nth gap becomes a dot
	)
	c := &image.Uniform{GruvFgDark}
	x := xStart
	gapIdx := 0
	for x < xEnd {
		// Dash
		x1 := x + dashW
		if x1 > xEnd {
			x1 = xEnd
		}
		draw.Draw(img, image.Rect(x, y, x1, y+thick), c, image.Point{}, draw.Src)
		x = x1
		if x >= xEnd {
			break
		}
		// Gap (with a dot painted into it on every Nth iteration)
		gapIdx++
		if gapIdx%dotEvery == 0 {
			dotX := x + (gapW-dotW)/2
			dotX1 := dotX + dotW
			if dotX1 > xEnd {
				dotX1 = xEnd
			}
			if dotX < xEnd {
				draw.Draw(img, image.Rect(dotX, y, dotX1, y+thick), c, image.Point{}, draw.Src)
			}
		}
		x += gapW
	}
}

func isLeapYear(y int) bool {
	return (y%4 == 0 && y%100 != 0) || y%400 == 0
}

func buildTestImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, CanvasW, CanvasH))
	draw.Draw(img, img.Bounds(), &image.Uniform{GruvBgHard}, image.Point{}, draw.Src)

	// 7x7 registration dots inset 20px from each corner.
	for _, p := range []image.Point{
		{20, 20}, {CanvasW - 20, 20},
		{20, CanvasH - 20}, {CanvasW - 20, CanvasH - 20},
	} {
		drawSquare(img, p, 3, GruvFg)
	}

	// Aqua mid-line stripes (horizontal + a short vertical) to spot rotation.
	draw.Draw(img, image.Rect(0, CanvasH/2-1, CanvasW, CanvasH/2+1), &image.Uniform{GruvAqua}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(CanvasW/2-1, CanvasH/2-100, CanvasW/2+1, CanvasH/2+100), &image.Uniform{GruvAqua}, image.Point{}, draw.Src)

	// Gruvbox accent palette swatches along the bottom.
	swatches := []color.RGBA{GruvRed, GruvGreen, GruvYellow, GruvBlue, GruvPurple, GruvAqua, GruvOrange}
	const swH = 12
	swW := CanvasW / len(swatches)
	for i, c := range swatches {
		r := image.Rect(i*swW, CanvasH-swH-20, (i+1)*swW, CanvasH-20)
		draw.Draw(img, r, &image.Uniform{c}, image.Point{}, draw.Src)
	}
	return img
}

// drawSquare paints a (2r+1)×(2r+1) filled square centered on p.
func drawSquare(img *image.RGBA, p image.Point, r int, c color.RGBA) {
	rect := image.Rect(p.X-r, p.Y-r, p.X+r+1, p.Y+r+1)
	draw.Draw(img, rect, &image.Uniform{c}, image.Point{}, draw.Src)
}
