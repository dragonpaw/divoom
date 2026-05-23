// Bakes scene-level Text/Time/Week elements into a rendered scene JPG.
// Used by the offline screenshot pipeline so docs/screenshots/ shows
// realistic dynamic content (real ticker readings, real quotes, a real
// ISS dot position, …) — at runtime the device renders these elements
// itself from the daemon's install payload.
//
// Sibling of BakeAlwaysOnHeaderJPEG; same decode→paint→re-encode shape.

package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"strings"
	"time"

	"github.com/dragonpaw/divoom/internal/frame"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// BakeSceneElementsJPEG decodes a scene JPG, paints the given device
// elements onto it as the wall-installed Times Frame would, and re-encodes.
//
// `now` is used to resolve Type=Time and Type=Week elements to their
// rendered strings ("15:04" and lowercase weekday respectively).
func BakeSceneElementsJPEG(in []byte, elements []frame.DispElement, now time.Time) ([]byte, error) {
	src, err := jpeg.Decode(bytes.NewReader(in))
	if err != nil {
		return nil, fmt.Errorf("decode jpeg: %w", err)
	}
	img := image.NewRGBA(src.Bounds())
	draw.Draw(img, img.Bounds(), src, image.Point{}, draw.Src)
	for _, e := range elements {
		if err := drawElement(img, e, now); err != nil {
			return nil, fmt.Errorf("element id=%d type=%s: %w", e.ID, e.Type, err)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		return nil, fmt.Errorf("encode jpeg: %w", err)
	}
	return buf.Bytes(), nil
}

// fontFileForID maps the device's FontID integers (see cmd/divoom/scenes.go)
// to TTF basenames in fonts/. Unknown IDs fall back to Roboto Condensed —
// every scene currently uses one of these three faces.
func fontFileForID(id int) string {
	switch id {
	case 7: // fontMono
		return "Iosevka-Regular.ttf"
	case 9: // fontProse
		return "RobotoCondensed-Regular.ttf"
	case 11: // fontProseLight
		return "RobotoCondensed-Light.ttf"
	case 13: // fontProseBlack
		return "RobotoCondensed-Black.ttf"
	default:
		return "RobotoCondensed-Regular.ttf"
	}
}

// parseHexColor turns "#fabd2f" into a color.RGBA. Empty / malformed
// strings fall back to GruvFg so a typo can't render invisible text.
func parseHexColor(s string) color.RGBA {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return GruvFg
	}
	var rgb [3]uint8
	for i := 0; i < 3; i++ {
		hi := hexNibble(s[i*2])
		lo := hexNibble(s[i*2+1])
		if hi < 0 || lo < 0 {
			return GruvFg
		}
		rgb[i] = uint8(hi)<<4 | uint8(lo)
	}
	return color.RGBA{rgb[0], rgb[1], rgb[2], 0xff}
}

func hexNibble(b byte) int {
	switch {
	case b >= '0' && b <= '9':
		return int(b - '0')
	case b >= 'a' && b <= 'f':
		return int(b-'a') + 10
	case b >= 'A' && b <= 'F':
		return int(b-'A') + 10
	default:
		return -1
	}
}

// faceHasAllRunes reports whether every non-ASCII-space rune in `text`
// has a glyph in the named TTF. ASCII whitespace is skipped (every face
// handles it). Returns true on an empty string and on any font load
// failure — the caller's draw will surface the real font error.
func faceHasAllRunes(fontName, text string) bool {
	ttf, err := LoadFont(fontName)
	if err != nil {
		return true
	}
	probe, err := opentype.NewFace(ttf, &opentype.FaceOptions{
		Size: 12, DPI: 72, Hinting: font.HintingNone,
	})
	if err != nil {
		return true
	}
	defer probe.Close()
	for _, r := range text {
		if r < 0x80 {
			continue
		}
		if _, ok := probe.GlyphAdvance(r); !ok {
			return false
		}
	}
	return true
}

// elementText resolves an element's rendered string. Text uses
// TextMessage verbatim; Time renders "HH:MM" from `now`; Week renders
// the lowercase weekday name from `now`. Any other type renders empty.
func elementText(e frame.DispElement, now time.Time) string {
	switch e.Type {
	case "Text":
		return e.TextMessage
	case "Time":
		return now.Format("15:04")
	case "Week":
		return strings.ToLower(now.Weekday().String())
	default:
		return ""
	}
}

func drawElement(img *image.RGBA, e frame.DispElement, now time.Time) error {
	text := elementText(e, now)
	if text == "" {
		return nil
	}
	if e.FontSize <= 0 {
		return nil
	}
	primary := fontFileForID(e.FontID)
	// Roboto Condensed (both weights) ships without the geometric-symbol
	// block that several scenes use as drop-in icons (● for the ISS dot,
	// ▼ for the sunrise tick, ▲ for the markets badge, ⚠ for hazard).
	// Reroute to Iosevka — which carries all of them — when the requested
	// text contains any rune the primary face can't render.
	chosen := primary
	if !faceHasAllRunes(primary, text) {
		chosen = "Iosevka-Regular.ttf"
	}
	ttf, err := LoadFont(chosen)
	if err != nil {
		return err
	}
	face, err := opentype.NewFace(ttf, &opentype.FaceOptions{
		Size: float64(e.FontSize), DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}
	defer face.Close()

	c := parseHexColor(e.FontColor)

	// Wrap text into lines that fit within e.Width on word boundaries.
	lines := wrapText(face, text, e.Width)

	// Vertical layout: device anchors at StartY as the top of the box.
	// Use ascent for baseline of the first line; advance by ascent+descent
	// per line. A small fixed gap matches the device's modest leading.
	metrics := face.Metrics()
	ascent := metrics.Ascent.Round()
	lineH := (metrics.Ascent + metrics.Descent).Round()

	for i, line := range lines {
		baselineY := e.StartY + ascent + i*lineH
		drawAligned(img, line, face, e.StartX, e.Width, baselineY, e.Align, c)
	}
	return nil
}

// drawAligned paints `s` inside the (startX, startX+width) horizontal
// track at the given baseline, honoring the device's Align int
// (0=left, 1=right, 2=center).
func drawAligned(img *image.RGBA, s string, face font.Face, startX, width, baselineY, align int, c color.RGBA) {
	switch align {
	case 1: // right
		drawLabelRight(img, s, face, startX+width, baselineY, c)
	case 2: // center
		drawLabelCentered(img, s, face, startX+width/2, baselineY, c)
	default: // left
		drawLabelLeft(img, s, face, startX, baselineY, c)
	}
}

// wrapText breaks `s` into lines that each fit within `width` pixels in
// the given face, splitting on whitespace boundaries. Words longer than
// the budget are emitted on their own line (no mid-word splits — the
// device clips them too, so the screenshot matches).
//
// Newlines in `s` are honored as hard breaks.
func wrapText(face font.Face, s string, width int) []string {
	var out []string
	for _, paragraph := range strings.Split(s, "\n") {
		out = append(out, wrapLine(face, paragraph, width)...)
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

func wrapLine(face font.Face, s string, width int) []string {
	if s == "" {
		return []string{""}
	}
	budget := fixed.I(width)
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	var cur string
	for _, w := range words {
		candidate := w
		if cur != "" {
			candidate = cur + " " + w
		}
		if font.MeasureString(face, candidate) <= budget {
			cur = candidate
			continue
		}
		if cur != "" {
			lines = append(lines, cur)
		}
		cur = w
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}
