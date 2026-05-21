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
	SceneNow Scene = iota
	SceneMarkets
	SceneSky
	SceneAmbient
)

// SceneBackground builds the hero frame and draws the scene's glyph into
// the bottom area — same gruvbox bg + divider + year-progress bar as
// HeroBackground, plus a faint shape that hints at what's playing.
func SceneBackground(scene Scene, format Format, now time.Time) ([]byte, error) {
	img := buildHeroImage(now)
	drawSceneGlyph(img, scene)
	return encodeImage(img, format)
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
	case SceneNow:
		// Sun: filled disc, no rays (rays would need a line rasterizer).
		fillCircle(img, cx, cy, 160, c)

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

	case SceneSky:
		// Crescent: filled circle minus an offset circle in bg color.
		fillCircle(img, cx, cy, 160, c)
		fillCircle(img, cx+60, cy-30, 150, GruvBgHard)

	case SceneAmbient:
		// Sparse dots — content-agnostic, still purposeful.
		for _, p := range []image.Point{
			{cx - 130, cy - 80}, {cx + 80, cy - 30}, {cx - 40, cy + 20},
			{cx + 130, cy + 100}, {cx - 110, cy + 110},
		} {
			fillCircle(img, p.X, p.Y, 28, c)
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
