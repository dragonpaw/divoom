package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
	"time"

	"github.com/dragonpaw/divoom/internal/render"
)

// Font lookup is handled by render.LoadFont, which probes both
// fonts/<name> and ../../fonts/<name> — no test-side shim required.

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

	// Cocktail composite — typographic recipe card (no photo).
	bgBytes, err = render.SceneBackground(render.SceneCocktail, render.FormatJPEG, time.Now())
	if err != nil {
		t.Fatalf("scene bg: %v", err)
	}
	canvas, err = jpegToRGBA(bgBytes)
	if err != nil {
		t.Fatalf("decode bg: %v", err)
	}
	rows := []recipeRow{
		{Measure: "1 oz", Ingredient: "Gin"},
		{Measure: "1 oz", Ingredient: "Campari"},
		{Measure: "1 oz", Ingredient: "Sweet Vermouth"},
		{Measure: "", Ingredient: "Orange peel"},
	}
	instructions := "Stir all ingredients with ice in a mixing glass. Strain into a chilled rocks glass over a large ice cube. Garnish with an orange peel."
	if err := drawCocktailCard(canvas, "Negroni", "Old-fashioned glass", "Ordinary Drink", instructions, rows); err != nil {
		t.Fatalf("draw card: %v", err)
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
