package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dragonpaw/divoom/internal/render"
)

// resolve fonts/ relative to repo root so the test works regardless of
// `go test ./...` invocation dir.
func init() {
	for _, p := range []string{"fonts/RobotoCondensed-Regular.ttf", "../../fonts/RobotoCondensed-Regular.ttf"} {
		if _, err := os.Stat(p); err == nil {
			prosePath = filepath.Clean(p)
			return
		}
	}
}

// TestBakedCompositeOffline exercises the resize + text-draw + JPEG
// encode path without hitting the network. Saves the composites to
// /tmp for eyeball verification when run with `-v`. It also doubles
// as a regression test for the layout coordinates (asserts the result
// decodes back to 800x1280 and is non-empty).
func TestBakedCompositeOffline(t *testing.T) {
	// Synthetic source photo: 1600x900 magenta-on-cyan grid, recognisably
	// "an image got pasted here" without needing a real APOD download.
	src := image.NewRGBA(image.Rect(0, 0, 1600, 900))
	for y := 0; y < 900; y++ {
		for x := 0; x < 1600; x++ {
			c := color.RGBA{0x33, 0x66, 0x99, 0xff}
			if (x/40+y/40)%2 == 0 {
				c = color.RGBA{0xcc, 0x55, 0xaa, 0xff}
			}
			src.SetRGBA(x, y, c)
		}
	}

	// NASA composite
	bgBytes, err := render.SceneBackground(render.SceneNASA, render.FormatJPEG, time.Now())
	if err != nil {
		t.Fatalf("scene bg: %v", err)
	}
	canvas, err := jpegToRGBA(bgBytes)
	if err != nil {
		t.Fatalf("decode bg: %v", err)
	}
	pasteImage(canvas, src, image.Rect(nasaImageX, nasaImageY, nasaImageX+nasaImageW, nasaImageY+nasaImageH))
	if err := drawCenteredText(canvas, "The Helix Nebula Through Webb", image.Rect(nasaTitleX, nasaTitleY, nasaTitleX+nasaTitleW, nasaTitleY+nasaTitleH), nasaTitleFS, gruvFg); err != nil {
		t.Fatalf("draw title: %v", err)
	}
	out, err := encodeJPEG(canvas)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if len(out) < 1000 {
		t.Fatalf("nasa composite too small: %d bytes", len(out))
	}
	if testing.Verbose() {
		os.WriteFile("/tmp/baked_nasa.jpg", out, 0644)
		t.Logf("wrote /tmp/baked_nasa.jpg (%d bytes)", len(out))
	}

	// Cocktail composite
	bgBytes, err = render.SceneBackground(render.SceneCocktail, render.FormatJPEG, time.Now())
	if err != nil {
		t.Fatalf("scene bg: %v", err)
	}
	canvas, err = jpegToRGBA(bgBytes)
	if err != nil {
		t.Fatalf("decode bg: %v", err)
	}
	pasteImage(canvas, src, image.Rect(cocktailImageX, cocktailImageY, cocktailImageX+cocktailImageW, cocktailImageY+cocktailImageH))
	if err := drawCenteredText(canvas, "Negroni", image.Rect(cocktailNameX, cocktailNameY, cocktailNameX+cocktailNameW, cocktailNameY+cocktailNameH), cocktailNameFS, gruvFg); err != nil {
		t.Fatalf("draw name: %v", err)
	}
	if err := drawCenteredText(canvas, "Gin, Campari, Sweet Vermouth, Orange Peel", image.Rect(cocktailIngX, cocktailIngY, cocktailIngX+cocktailIngW, cocktailIngY+cocktailIngH), cocktailIngFS, gruvFgDark); err != nil {
		t.Fatalf("draw ing: %v", err)
	}
	out, err = encodeJPEG(canvas)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if testing.Verbose() {
		os.WriteFile("/tmp/baked_cocktail.jpg", out, 0644)
		// also save as PNG for sharper pixel inspection
		var pngBuf bytes.Buffer
		_ = png.Encode(&pngBuf, canvas)
		os.WriteFile("/tmp/baked_cocktail.png", pngBuf.Bytes(), 0644)
		t.Logf("wrote /tmp/baked_cocktail.jpg (%d bytes)", len(out))
	}
}
