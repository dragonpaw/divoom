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
	"sync"
	"time"
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
		drawDayOfYearProgress(img, now)
	case SceneEaster:
		drawEasterEgg(img)
	case SceneWeather:
		// No outlook supplied — fall back to the cloudy glyph so the
		// frame still renders. Production callers use SceneWeatherBackground.
		drawWeatherGlyph(img, "cloudy")
	default:
		drawSceneGlyph(img, scene)
	}
	return encodeImage(img, format)
}

// SceneWeatherBackground renders the weather scene's bg with the icon
// matching `outlook` (one of the strings produced by the weather widget:
// clear, cloudy, overcast, rain, drizzle, snow, fog, thunder). Unknown
// outlooks fall back to the cloudy icon.
func SceneWeatherBackground(outlook string, format Format, now time.Time) ([]byte, error) {
	img := buildHeroImage(now)
	drawWeatherGlyph(img, outlook)
	return encodeImage(img, format)
}

// drawEasterEgg paints a giant gruvbox-yellow egg centred in the body
// area. Built as an asymmetric ellipse (smaller radius on top, larger
// below) so the curve reads as an egg rather than a figure-8. Big
// enough to be the dominant visual feature of the scene since this is
// the rarest rotation entry and earns the spotlight.
func drawEasterEgg(img *image.RGBA) {
	const (
		cx    = CanvasW / 2
		cy    = 870
		rx    = 250
		ryTop = 250
		ryBot = 320
	)
	fillEgg(img, cx, cy, rx, ryTop, ryBot, GruvYellow)
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

// drawDayOfYearProgress paints a thick year-progress bar across the
// body area of the dayofyear scene background — orange fill on a
// bg-darker track, full width, 60 px tall. Updates only when the bg
// is re-rendered (daemon startup), but day-to-day fraction differences
// are <0.3 % so the visual stays fresh enough between restarts.
func drawDayOfYearProgress(img *image.RGBA, now time.Time) {
	const (
		barTop    = 755
		barBottom = 815
		inset     = 40
	)
	yearDays := 365
	if isLeapYear(now.Year()) {
		yearDays = 366
	}
	frac := float64(now.YearDay()-1) / float64(yearDays)
	// Track
	draw.Draw(img,
		image.Rect(inset, barTop, CanvasW-inset, barBottom),
		&image.Uniform{GruvBgDarker}, image.Point{}, draw.Src)
	// Fill
	fillW := int(frac * float64(CanvasW-2*inset))
	if fillW > 0 {
		draw.Draw(img,
			image.Rect(inset, barTop, inset+fillW, barBottom),
			&image.Uniform{GruvOrange}, image.Point{}, draw.Src)
	}
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
	const (
		// Anchor the glyph at the bottom-right, well above the year-
		// progress hairline (y=1268-1272) so the bar still reads cleanly.
		cx = CanvasW - 180
		cy = CanvasH - 240
	)
	c := GruvBgDarker

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

	case SceneMoonphase:
		// Crescent: filled circle minus an offset circle in bg color.
		fillCircle(img, cx, cy, 160, c)
		fillCircle(img, cx+60, cy-30, 150, GruvBgHard)

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
		// Discworld cosmology stack: the flat Disc on top, four world
		// elephants standing in the middle, and Great A'Tuin the star
		// turtle swimming beneath them. All in bg-darker.
		const (
			discCy     = -90 // offset from cy
			elephantsY = -65 // top of elephant rectangles, relative to cy
			elephantH  = 50
			elephantW  = 16
			turtleCy   = 60 // offset from cy
		)
		// The Disc: wide flat ellipse.
		fillEgg(img, cx, cy+discCy, 110, 15, 15, c)
		// Four elephants: narrow vertical rectangles spaced under the disc.
		elephantXs := []int{cx - 75, cx - 25, cx + 25, cx + 75}
		for _, ex := range elephantXs {
			draw.Draw(img,
				image.Rect(ex-elephantW/2, cy+elephantsY, ex+elephantW/2, cy+elephantsY+elephantH),
				&image.Uniform{c}, image.Point{}, draw.Src)
		}
		// Great A'Tuin: larger flat ellipse, fuller on the bottom for a
		// domed-shell read. Tiny head circle off to the right.
		fillEgg(img, cx, cy+turtleCy, 130, 22, 35, c)
		fillCircle(img, cx+145, cy+turtleCy-5, 12, c)

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
		// Cat silhouette: classic sitting-cat-from-behind. Round head
		// with two pointy triangular ears, oval body widening toward
		// the base, and a curled tail wrapping up the right side. The
		// body+head outline is a single closed polygon rasterised by
		// the same scanline fill as the Starfleet delta; the tail is
		// a soft arc of overlapping discs along the right side so it
		// reads as a separate appendage rather than part of the body
		// blob.
		drawCatSilhouette(img, cx, cy, 200, 200, c)

	case SceneDidYouKnow:
		// Bold typographic "?" — sourced from the Twemoji ❔ (U+2754) PNG
		// mask (see assets.go) and overpainted in c, matching the
		// mask-driven pattern used by the Starfleet delta / buddha /
		// weather icons.
		drawQuestion(img, cx, cy, c)

	case SceneSunrise:
		// Sun cresting a horizon: a long thin horizon bar, a sun disc
		// whose bottom half is carved out by a bg-hard rectangle so it
		// reads as half-risen, and three small ray discs flicking up
		// from the top arc.
		const (
			horizonHalfW = 100 // half-length of the horizon bar
			horizonH     = 6   // horizon bar thickness
			sunR         = 60  // sun radius
			sunCyOff     = -8  // sun centre relative to cy (sits just above horizon)
			rayR         = 9   // ray disc radius
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
		// Three rays flicking up around the sun's top arc.
		for _, ray := range []image.Point{
			{X: cx - 90, Y: sunCy - 40},
			{X: cx, Y: sunCy - 80},
			{X: cx + 90, Y: sunCy - 40},
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
		// Meditating figure (🧘 in lotus position). Same mask-overpaint
		// treatment as the Starfleet delta and weather icons; the source
		// is a Twemoji SVG (see assets.go).
		drawBuddha(img, cx, cy, c)

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
		drawGit(img, cx, cy, c)

	case SceneISS:
		// Stylised satellite silhouette: small central body with two long
		// thin solar panels flanking it horizontally, plus a short antenna
		// rising from the top. Rectangles only — no mask asset needed.
		const (
			bodyHalfW   = 30
			bodyHalfH   = 30
			panelW      = 120
			panelH      = 30
			panelGap    = 6 // gap between body edge and inboard panel edge
			antennaW    = 4
			antennaH    = 40
			antennaTipR = 6
		)
		// Central body.
		draw.Draw(img,
			image.Rect(cx-bodyHalfW, cy-bodyHalfH, cx+bodyHalfW, cy+bodyHalfH),
			&image.Uniform{c}, image.Point{}, draw.Src)
		// Left solar panel.
		leftRight := cx - bodyHalfW - panelGap
		draw.Draw(img,
			image.Rect(leftRight-panelW, cy-panelH/2, leftRight, cy+panelH/2),
			&image.Uniform{c}, image.Point{}, draw.Src)
		// Right solar panel.
		rightLeft := cx + bodyHalfW + panelGap
		draw.Draw(img,
			image.Rect(rightLeft, cy-panelH/2, rightLeft+panelW, cy+panelH/2),
			&image.Uniform{c}, image.Point{}, draw.Src)
		// Antenna stem rising from the top of the body, capped with a
		// small disc so it reads as a sensor rather than a stray line.
		antennaTop := cy - bodyHalfH - antennaH
		draw.Draw(img,
			image.Rect(cx-antennaW/2, antennaTop, cx+antennaW/2, cy-bodyHalfH),
			&image.Uniform{c}, image.Point{}, draw.Src)
		fillCircle(img, cx, antennaTop, antennaTipR, c)

	case SceneOnThisDay:
		// Analog clock face — the "this moment in history" motif. Built
		// as a thick ring (outer disc minus an inner disc in bg-hard),
		// four cardinal hour ticks poking inward from the ring, a short
		// hour hand pointing up-right (≈ 10 o'clock) and a longer minute
		// hand pointing up (12 o'clock), plus a tiny hub disc covering
		// the hand pivot. Total footprint ~200×200.
		const (
			faceR    = 100 // outer ring radius
			faceThk  = 14  // ring band thickness
			tickLen  = 16  // hour tick length (inward from ring)
			tickThk  = 8   // hour tick thickness
			hubR     = 8   // central pivot disc
			minHandLen = 70 // minute hand: straight up
			minHandThk = 8
			hrHandThk  = 10
			hrHandDX   = 32  // x offset of hour hand tip (rightward)
			hrHandDY   = -38 // y offset of hour hand tip (upward)
		)
		// Outer ring: filled disc, then carve out the inside.
		fillCircle(img, cx, cy, faceR, c)
		fillCircle(img, cx, cy, faceR-faceThk, GruvBgHard)
		// Four hour ticks (12 / 3 / 6 / 9) — short bars pointing inward
		// from the inner ring edge.
		innerR := faceR - faceThk
		// 12 o'clock (top)
		draw.Draw(img,
			image.Rect(cx-tickThk/2, cy-innerR, cx+tickThk/2, cy-innerR+tickLen),
			&image.Uniform{c}, image.Point{}, draw.Src)
		// 6 o'clock (bottom)
		draw.Draw(img,
			image.Rect(cx-tickThk/2, cy+innerR-tickLen, cx+tickThk/2, cy+innerR),
			&image.Uniform{c}, image.Point{}, draw.Src)
		// 3 o'clock (right)
		draw.Draw(img,
			image.Rect(cx+innerR-tickLen, cy-tickThk/2, cx+innerR, cy+tickThk/2),
			&image.Uniform{c}, image.Point{}, draw.Src)
		// 9 o'clock (left)
		draw.Draw(img,
			image.Rect(cx-innerR, cy-tickThk/2, cx-innerR+tickLen, cy+tickThk/2),
			&image.Uniform{c}, image.Point{}, draw.Src)
		// Minute hand straight up.
		draw.Draw(img,
			image.Rect(cx-minHandThk/2, cy-minHandLen, cx+minHandThk/2, cy),
			&image.Uniform{c}, image.Point{}, draw.Src)
		// Hour hand: a thick parallelogram from the hub to the upper-right.
		fillPolygon(img, []struct{ x, y int }{
			{cx - hrHandThk/2, cy - hrHandThk/2},
			{cx + hrHandThk/2, cy + hrHandThk/2},
			{cx + hrHandDX + hrHandThk/2, cy + hrHandDY + hrHandThk/2},
			{cx + hrHandDX - hrHandThk/2, cy + hrHandDY - hrHandThk/2},
		}, c)
		// Central hub disc covering the pivot.
		fillCircle(img, cx, cy, hubR, c)
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

	return img
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
