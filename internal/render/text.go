// Text rasterisation helpers for baked-in chrome (labels drawn into bg
// JPGs at render time, rather than as device Text elements). Used by
// drawWeatherChrome here and by cmd/divoom/scene_baked.go for the NASA /
// cocktail title rows.
//
// LoadFont resolves a TTF basename under fonts/ — tries the repo-root
// path first, then ../../fonts/ as a fallback for `go test ./...` runs
// whose CWD is the package directory. Cached so repeated calls don't
// re-parse the TTF.

package render

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var (
	fontMu    sync.Mutex
	fontCache = map[string]*opentype.Font{}
)

// LoadFont reads a TTF from the repo's fonts/ directory. Tries
// "fonts/<name>" first (the runtime layout for `divoom` invocations
// from the repo root), then "../../fonts/<name>" (so tests under
// cmd/divoom and internal/render both work). Results are cached.
func LoadFont(name string) (*opentype.Font, error) {
	fontMu.Lock()
	defer fontMu.Unlock()
	if f, ok := fontCache[name]; ok {
		return f, nil
	}
	candidates := []string{
		filepath.Join("fonts", name),
		filepath.Join("..", "..", "fonts", name),
	}
	var raw []byte
	var err error
	for _, p := range candidates {
		raw, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("read font %s: %w", name, err)
	}
	f, err := opentype.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse font %s: %w", name, err)
	}
	fontCache[name] = f
	return f, nil
}

// drawLabelCentered paints s in the given face, centred horizontally on
// cx with its baseline at baselineY, in colour c. Used by
// drawWeatherChrome for the column labels.
func drawLabelCentered(img *image.RGBA, s string, face font.Face, cx, baselineY int, c color.RGBA) {
	w := font.MeasureString(face, s)
	dotX := fixed.I(cx) - w/2
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.Point26_6{X: dotX, Y: fixed.I(baselineY)},
	}
	d.DrawString(s)
}

// drawLabelLeft paints s with its left edge at x and baseline at baselineY.
func drawLabelLeft(img *image.RGBA, s string, face font.Face, x, baselineY int, c color.RGBA) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(baselineY)},
	}
	d.DrawString(s)
}

// drawLabelRight paints s with its right edge at x and baseline at baselineY.
func drawLabelRight(img *image.RGBA, s string, face font.Face, x, baselineY int, c color.RGBA) {
	w := font.MeasureString(face, s)
	dotX := fixed.I(x) - w
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.Point26_6{X: dotX, Y: fixed.I(baselineY)},
	}
	d.DrawString(s)
}

// MeasureLabel returns the pixel width of s rendered in the given font/size.
// Used by callers that need to know where a baked label ends so they can
// align device Text elements next to it (e.g. terminal-family scenes
// placing a dynamic headword after a baked "$ define " prompt).
func MeasureLabel(s, fontName string, size float64) (int, error) {
	f, err := LoadFont(fontName)
	if err != nil {
		return 0, err
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size: size, DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		return 0, err
	}
	defer face.Close()
	w := font.MeasureString(face, s)
	return w.Ceil(), nil
}
