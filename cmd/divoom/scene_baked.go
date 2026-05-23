package main

// Push-time image compositing for the NASA APOD and Cocktail scenes.
//
// Background — Divoom's cloud proxy whitelists only `f.divoom-gz.com` for
// Image DispElement URLs (see docs/api.md and
// memory/feedback_netdata_cloud_proxy.md). Fetching APOD / cocktail
// thumbnails through the device's Image element therefore silently fails.
// Workaround: at `divoom push` time, we fetch the upstream image, resize
// it to the slot the old Image element used to occupy, draw the title /
// drink name / ingredient list as raster text, JPEG-encode the result,
// and adb-push it as the scene's bg JPG. The scene definition then drops
// the Image and Text elements entirely and shows only the bg.
//
// This is specifically these two scenes. No abstraction — see CLAUDE.md.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	xdraw "golang.org/x/image/draw"

	"github.com/dragonpaw/divoom/internal/render"
)

const (
	// httpFetchTimeout caps both API calls and image downloads. Pushes
	// run from a USB-attached host so an extra few seconds is fine, but
	// we don't want a hanging endpoint to block the entire push step.
	httpFetchTimeout = 30 * time.Second

	// JPEG quality for the composited bgs. Matches render.encodeImage's 95.
	bakedJPEGQuality = 92
)

// bakeNASAandCocktailBackgrounds is called from runPush after
// pushSceneBackgrounds. Each composite is best-effort — on API or
// network failure we log a warning and continue.
func bakeNASAandCocktailBackgrounds(ctx context.Context) error {
	if err := bakeAllNASABackgrounds(ctx); err != nil {
		slog.Warn("nasa bg compositing failed", "err", err)
	}
	if err := bakeCocktailBackground(ctx); err != nil {
		slog.Warn("cocktail bg compositing failed; leaving plain scene bg", "err", err)
	}
	return nil
}

// --- NASA APOD ----------------------------------------------------------

const (
	nasaAPIBase = "https://api.nasa.gov/planetary/apod"

	// Image slot inside the bg — same rectangle the old Image DispElement
	// occupied (StartX 20, StartY 560, W 760, H 540).
	nasaImageX = 20
	nasaImageY = 560
	nasaImageW = 760
	nasaImageH = 540

	// Title slot — same rectangle the old Text DispElement occupied
	// (StartX 80, StartY 1120, W 640, H 80). FontSize 36, fontProse / cFg.
	nasaTitleX  = 80
	nasaTitleY  = 1120
	nasaTitleW  = 640
	nasaTitleH  = 80
	nasaTitleFS = 36
)

type nasaAPODResponse struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	HDURL     string `json:"hdurl"`
	Date      string `json:"date"`
	MediaType string `json:"media_type"`
}

// bakeAllNASABackgrounds iterates the curated APOD date list, bakes
// each entry into a per-index bg JPG, and adb-pushes each to its
// indexed device path (bgNASAFor(i)). The on-device NASA scene then
// uses BgPathFor to pick a random index per activation, so the wall
// display rotates through every curated picture instead of always
// showing the same one.
//
// Each per-date bake is best-effort: on fetch / encode failure we log
// and fall back to pushing the plain SceneNASA bg (the gruvbox hero
// chrome with no embedded photo) for that index, so the device never
// has a missing path. The plain fallback is rendered once at the top.
func bakeAllNASABackgrounds(ctx context.Context) error {
	apodKey := os.Getenv("NASA_API_KEY")
	if apodKey == "" {
		apodKey = "DEMO_KEY"
	}

	// Plain scene bg (hero chrome only) — used as a per-index fallback
	// when a date's APOD bake fails. Rendered once and reused.
	plainBg, err := render.SceneBackground(render.SceneNASA, render.FormatJPEG, time.Now())
	if err != nil {
		return fmt.Errorf("render plain nasa bg: %w", err)
	}

	for i, date := range nasaCuratedDates {
		path := bgNASAFor(i)
		out, err := bakeOneNASAImage(ctx, apodKey, date)
		if err != nil {
			slog.Warn("nasa bake failed; pushing plain fallback for this index",
				"index", i, "date", date, "err", err)
			if perr := pushBytes(ctx, plainBg, path); perr != nil {
				slog.Warn("nasa fallback push failed", "index", i, "err", perr)
			}
			continue
		}
		if perr := pushBytes(ctx, out, path); perr != nil {
			slog.Warn("nasa push failed", "index", i, "date", date, "err", perr)
			continue
		}
		slog.Info("nasa apod bg pushed", "index", i, "date", date)
	}
	return nil
}

// bakeOneNASAImage fetches the APOD for date, composites it into the
// scene bg, and returns the encoded JPEG bytes. Returns an error on
// any fetch / encode failure; callers fall back to the plain bg.
func bakeOneNASAImage(ctx context.Context, apiKey, date string) ([]byte, error) {
	body, err := fetchAPOD(ctx, apiKey, date)
	if err != nil {
		return nil, fmt.Errorf("fetch apod: %w", err)
	}
	if body.MediaType != "image" {
		return nil, fmt.Errorf("apod %s is %q, not image", date, body.MediaType)
	}

	imgURL := body.HDURL
	if imgURL == "" {
		imgURL = body.URL
	}
	if imgURL == "" {
		return nil, fmt.Errorf("apod %s has no image url", date)
	}

	photo, err := fetchImage(ctx, imgURL)
	if err != nil {
		return nil, fmt.Errorf("download apod image: %w", err)
	}

	bgBytes, err := render.SceneBackground(render.SceneNASA, render.FormatJPEG, time.Now())
	if err != nil {
		return nil, fmt.Errorf("render scene bg: %w", err)
	}
	canvas, err := jpegToRGBA(bgBytes)
	if err != nil {
		return nil, fmt.Errorf("decode scene bg: %w", err)
	}

	pasteImage(canvas, photo, image.Rect(nasaImageX, nasaImageY, nasaImageX+nasaImageW, nasaImageY+nasaImageH))

	if err := drawCenteredText(canvas, body.Title,
		image.Rect(nasaTitleX, nasaTitleY, nasaTitleX+nasaTitleW, nasaTitleY+nasaTitleH),
		nasaTitleFS, gruvFg); err != nil {
		return nil, fmt.Errorf("draw title: %w", err)
	}

	return encodeJPEG(canvas)
}

// fetchAPOD calls the NASA APOD endpoint. When date is "" the API
// returns today's entry; otherwise the YYYY-MM-DD entry.
func fetchAPOD(ctx context.Context, apiKey, date string) (*nasaAPODResponse, error) {
	url := nasaAPIBase + "?api_key=" + apiKey
	if date != "" {
		url += "&date=" + date
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: httpFetchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	var body nasaAPODResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	return &body, nil
}

// --- Cocktail -----------------------------------------------------------
//
// Recipe-card redesign: the bg is now pure typography (no drink photo).
// Layout, top to bottom:
//
//	y=540  drink name           — 72pt Roboto Condensed, gruvFg, centred
//	y=620  glass · category     — 24pt Iosevka, gruvFgDark, centred
//	y=680  hairline rule        — short centred divider
//	y=730  "INGREDIENTS"        — 22pt Iosevka, gruvFgDark, left
//	y=770… ingredient rows      — 28pt Roboto, gruvFg, "<measure> <ingredient>"
//	y=1020 "METHOD"             — 22pt Iosevka, gruvFgDark, left
//	y=1060… method prose        — 24pt Roboto Light, gruvFgDark, wrapped

const (
	cocktailAPIURL = "https://www.thecocktaildb.com/api/json/v1/1/random.php"

	cocktailLeft  = 80
	cocktailRight = 720

	cocktailNameBaseY  = 540
	cocktailNameSize   = 72
	cocktailSubBaseY   = 615
	cocktailSubSize    = 24
	cocktailRuleY      = 660
	cocktailRuleHalfW  = 100

	cocktailIngLabelY    = 730
	cocktailIngLabelSize = 22
	cocktailIngFirstY    = 778
	cocktailIngRowH      = 42
	cocktailIngSize      = 28
	cocktailIngMeasureX  = cocktailLeft
	cocktailIngNameX     = cocktailLeft + 180
	cocktailIngMaxRows   = 6

	cocktailMethodLabelY    = 1020
	cocktailMethodLabelSize = 22
	cocktailMethodFirstY    = 1060
	cocktailMethodSize      = 24
	cocktailMethodLineH     = 32
	cocktailMethodMaxLines  = 5
)

type cocktailResponse struct {
	Drinks []map[string]any `json:"drinks"`
}

type recipeRow struct {
	Measure    string
	Ingredient string
}

func bakeCocktailBackground(ctx context.Context) error {
	name, glass, category, instructions, rows, err := fetchCocktail(ctx)
	if err != nil {
		return fmt.Errorf("fetch cocktail: %w", err)
	}

	bgBytes, err := render.SceneBackground(render.SceneCocktail, render.FormatJPEG, time.Now())
	if err != nil {
		return fmt.Errorf("render scene bg: %w", err)
	}
	canvas, err := jpegToRGBA(bgBytes)
	if err != nil {
		return fmt.Errorf("decode scene bg: %w", err)
	}

	if err := drawCocktailCard(canvas, name, glass, category, instructions, rows); err != nil {
		return fmt.Errorf("draw card: %w", err)
	}

	out, err := encodeJPEG(canvas)
	if err != nil {
		return fmt.Errorf("encode jpeg: %w", err)
	}
	if err := pushBytes(ctx, out, bgCocktail); err != nil {
		return fmt.Errorf("push: %w", err)
	}
	slog.Info("cocktail bg composited and pushed", "name", name)
	return nil
}

func drawCocktailCard(canvas *image.RGBA, name, glass, category, instructions string, rows []recipeRow) error {
	proseRegular, err := render.LoadFont("RobotoCondensed-Regular.ttf")
	if err != nil {
		return fmt.Errorf("load prose-regular: %w", err)
	}
	proseLight, err := render.LoadFont("RobotoCondensed-Light.ttf")
	if err != nil {
		return fmt.Errorf("load prose-light: %w", err)
	}
	mono, err := render.LoadFont("Iosevka-Regular.ttf")
	if err != nil {
		return fmt.Errorf("load mono: %w", err)
	}

	nameFace, err := newFace(proseRegular, cocktailNameSize)
	if err != nil {
		return err
	}
	defer nameFace.Close()
	drawCenteredOnCx(canvas, name, nameFace, (cocktailLeft+cocktailRight)/2, cocktailNameBaseY, gruvFg, cocktailRight-cocktailLeft)

	subFace, err := newFace(mono, cocktailSubSize)
	if err != nil {
		return err
	}
	defer subFace.Close()
	sub := cocktailSubhead(glass, category)
	if sub != "" {
		drawCenteredOnCx(canvas, sub, subFace, (cocktailLeft+cocktailRight)/2, cocktailSubBaseY, gruvFgDark, cocktailRight-cocktailLeft)
	}

	// Short hairline centred under the subhead.
	cx := (cocktailLeft + cocktailRight) / 2
	draw.Draw(canvas,
		image.Rect(cx-cocktailRuleHalfW, cocktailRuleY, cx+cocktailRuleHalfW, cocktailRuleY+1),
		&image.Uniform{gruvFgDark}, image.Point{}, draw.Src)

	ingLabelFace, err := newFace(mono, cocktailIngLabelSize)
	if err != nil {
		return err
	}
	defer ingLabelFace.Close()
	drawLeftAt(canvas, "INGREDIENTS", ingLabelFace, cocktailLeft, cocktailIngLabelY, gruvFgDark)

	ingFace, err := newFace(proseRegular, cocktailIngSize)
	if err != nil {
		return err
	}
	defer ingFace.Close()
	maxRows := cocktailIngMaxRows
	if len(rows) < maxRows {
		maxRows = len(rows)
	}
	for i := 0; i < maxRows; i++ {
		y := cocktailIngFirstY + i*cocktailIngRowH
		if rows[i].Measure != "" {
			drawLeftAt(canvas, rows[i].Measure, ingFace, cocktailIngMeasureX, y, gruvFgDark)
		}
		drawLeftAt(canvas, rows[i].Ingredient, ingFace, cocktailIngNameX, y, gruvFg)
	}

	methodLabelFace, err := newFace(mono, cocktailMethodLabelSize)
	if err != nil {
		return err
	}
	defer methodLabelFace.Close()
	drawLeftAt(canvas, "METHOD", methodLabelFace, cocktailLeft, cocktailMethodLabelY, gruvFgDark)

	methodFace, err := newFace(proseLight, cocktailMethodSize)
	if err != nil {
		return err
	}
	defer methodFace.Close()
	lines := wrapText(instructions, methodFace, cocktailRight-cocktailLeft, cocktailMethodMaxLines)
	for i, line := range lines {
		y := cocktailMethodFirstY + i*cocktailMethodLineH
		drawLeftAt(canvas, line, methodFace, cocktailLeft, y, gruvFgDark)
	}
	return nil
}

// cocktailSubhead formats the glass / category subhead row. Either field
// may be empty; if both are present they're joined with " · ".
func cocktailSubhead(glass, category string) string {
	glass = strings.TrimSpace(glass)
	category = strings.TrimSpace(category)
	switch {
	case glass != "" && category != "":
		return strings.ToLower(glass + " · " + category)
	case glass != "":
		return strings.ToLower(glass)
	case category != "":
		return strings.ToLower(category)
	}
	return ""
}

func fetchCocktail(ctx context.Context) (name, glass, category, instructions string, rows []recipeRow, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cocktailAPIURL, nil)
	if err != nil {
		return "", "", "", "", nil, err
	}
	req.Header.Set("User-Agent", "divoom-dashboard/0.1 (github.com/dragonpaw/divoom)")
	client := &http.Client{Timeout: httpFetchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", "", "", "", nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	var body cocktailResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", "", "", "", nil, err
	}
	if len(body.Drinks) == 0 {
		return "", "", "", "", nil, fmt.Errorf("no drinks in response")
	}
	d := body.Drinks[0]
	name = strings.TrimSpace(strSafe(d["strDrink"]))
	glass = strings.TrimSpace(strSafe(d["strGlass"]))
	category = strings.TrimSpace(strSafe(d["strCategory"]))
	instructions = strings.TrimSpace(strSafe(d["strInstructions"]))

	// Pair the 15 ingredient slots with their matching measure slots; the
	// first empty ingredient terminates the list.
	for i := 1; i <= 15; i++ {
		ing := strings.TrimSpace(strSafe(d[fmt.Sprintf("strIngredient%d", i)]))
		if ing == "" {
			break
		}
		measure := strings.TrimSpace(strSafe(d[fmt.Sprintf("strMeasure%d", i)]))
		rows = append(rows, recipeRow{Measure: measure, Ingredient: ing})
	}

	if name == "" {
		return "", "", "", "", nil, fmt.Errorf("missing drink name")
	}
	return name, glass, category, instructions, rows, nil
}

func strSafe(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// --- shared helpers -----------------------------------------------------

// fetchImage downloads url and decodes it as PNG or JPEG.
func fetchImage(ctx context.Context, url string) (image.Image, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: httpFetchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return img, nil
}

// pasteImage resizes src to fit dst preserving aspect ratio (centre-crop
// when aspects differ) and draws it onto canvas at dst.
func pasteImage(canvas *image.RGBA, src image.Image, dst image.Rectangle) {
	sb := src.Bounds()
	srcAspect := float64(sb.Dx()) / float64(sb.Dy())
	dstAspect := float64(dst.Dx()) / float64(dst.Dy())

	// Pick the source crop rect (in src coords) that matches dst's aspect.
	var crop image.Rectangle
	if srcAspect > dstAspect {
		// Source wider than dst — crop sides.
		newW := int(float64(sb.Dy()) * dstAspect)
		off := (sb.Dx() - newW) / 2
		crop = image.Rect(sb.Min.X+off, sb.Min.Y, sb.Min.X+off+newW, sb.Max.Y)
	} else {
		// Source taller than dst — crop top/bottom.
		newH := int(float64(sb.Dx()) / dstAspect)
		off := (sb.Dy() - newH) / 2
		crop = image.Rect(sb.Min.X, sb.Min.Y+off, sb.Max.X, sb.Min.Y+off+newH)
	}
	xdraw.CatmullRom.Scale(canvas, dst, src, crop, xdraw.Over, nil)
}

func jpegToRGBA(b []byte) (*image.RGBA, error) {
	img, err := jpeg.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{}, draw.Src)
	return rgba, nil
}

func encodeJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: bakedJPEGQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Gruvbox text colours — match the originals in scenes.go (cFg, cFgDark).
var (
	gruvFg     = color.RGBA{0xeb, 0xdb, 0xb2, 0xff}
	gruvFgDark = color.RGBA{0xa8, 0x99, 0x84, 0xff}
)

// proseFontName is the basename under fonts/ that drawCenteredText
// loads via render.LoadFont. Exposed as a var so tests that run from
// the package dir don't need to override anything — render.LoadFont
// already tries ../../fonts/ as a fallback.
var proseFontName = "RobotoCondensed-Regular.ttf"

// drawCenteredText rasters s centred horizontally and vertically inside
// rect at the device font-size px equivalent. We treat the DispElement's
// FontSize as a px height for the face (the device is 800x1280; our
// canvas is 1:1 with it).
func drawCenteredText(canvas *image.RGBA, s string, rect image.Rectangle, sizePx int, col color.RGBA) error {
	if s == "" {
		return nil
	}
	f, err := render.LoadFont(proseFontName)
	if err != nil {
		return err
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    float64(sizePx),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}
	defer face.Close()

	// Truncate-with-ellipsis if the string is wider than rect.
	maxW := fixed.I(rect.Dx())
	width := font.MeasureString(face, s)
	if width > maxW {
		s = ellipsize(s, face, maxW)
		width = font.MeasureString(face, s)
	}

	metrics := face.Metrics()
	textH := metrics.Ascent + metrics.Descent
	// Baseline = top of rect + (rect height - text height)/2 + ascent.
	baseline := fixed.I(rect.Min.Y) + (fixed.I(rect.Dy())-textH)/2 + metrics.Ascent
	dotX := fixed.I(rect.Min.X) + (maxW-width)/2

	d := &font.Drawer{
		Dst:  canvas,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.Point26_6{X: dotX, Y: baseline},
	}
	d.DrawString(s)
	return nil
}

// newFace wraps opentype.NewFace with the common HintingFull / DPI 72
// settings every baked-text helper here uses.
func newFace(f *opentype.Font, sizePx int) (font.Face, error) {
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size: float64(sizePx), DPI: 72, Hinting: font.HintingFull,
	})
}

// drawCenteredOnCx paints s centred horizontally on cx with its baseline
// at baselineY. maxW caps the rendered width; oversize strings are
// ellipsised.
func drawCenteredOnCx(canvas *image.RGBA, s string, face font.Face, cx, baselineY int, col color.RGBA, maxW int) {
	if s == "" {
		return
	}
	maxFx := fixed.I(maxW)
	if font.MeasureString(face, s) > maxFx {
		s = ellipsize(s, face, maxFx)
	}
	w := font.MeasureString(face, s)
	dotX := fixed.I(cx) - w/2
	(&font.Drawer{
		Dst: canvas, Src: image.NewUniform(col), Face: face,
		Dot: fixed.Point26_6{X: dotX, Y: fixed.I(baselineY)},
	}).DrawString(s)
}

// drawLeftAt paints s with its left edge at x and its baseline at
// baselineY. No truncation — callers pre-size strings to their slot.
func drawLeftAt(canvas *image.RGBA, s string, face font.Face, x, baselineY int, col color.RGBA) {
	if s == "" {
		return
	}
	(&font.Drawer{
		Dst: canvas, Src: image.NewUniform(col), Face: face,
		Dot: fixed.Point26_6{X: fixed.I(x), Y: fixed.I(baselineY)},
	}).DrawString(s)
}

// wrapText splits s into lines that each fit within maxW pixels in the
// given face. Words longer than maxW are ellipsised on their own line.
// Returns at most maxLines; the final line is ellipsised if input
// would have wrapped further.
func wrapText(s string, face font.Face, maxW, maxLines int) []string {
	s = strings.TrimSpace(s)
	if s == "" || maxLines <= 0 {
		return nil
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	maxFx := fixed.I(maxW)
	var lines []string
	var cur string
	for _, w := range words {
		candidate := cur
		if candidate != "" {
			candidate += " "
		}
		candidate += w
		if font.MeasureString(face, candidate) <= maxFx {
			cur = candidate
			continue
		}
		// Doesn't fit: flush current line, start new with this word.
		if cur != "" {
			lines = append(lines, cur)
			if len(lines) >= maxLines {
				// Re-ellipsise the last line to include the overflow marker.
				lines[len(lines)-1] = ellipsize(cur+" "+w+" …", face, maxFx)
				return lines
			}
		}
		// Single word too wide for the line — ellipsise it.
		if font.MeasureString(face, w) > maxFx {
			cur = ellipsize(w, face, maxFx)
		} else {
			cur = w
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines[maxLines-1] = ellipsize(lines[maxLines-1]+" …", face, maxFx)
	}
	return lines
}

// ellipsize trims s with a trailing "…" until it fits in maxW.
func ellipsize(s string, face font.Face, maxW fixed.Int26_6) string {
	const ell = "…"
	if font.MeasureString(face, s) <= maxW {
		return s
	}
	// Binary-search-ish trim, in characters.
	runes := []rune(s)
	for len(runes) > 1 {
		runes = runes[:len(runes)-1]
		candidate := strings.TrimRight(string(runes), " ,") + ell
		if font.MeasureString(face, candidate) <= maxW {
			return candidate
		}
	}
	return ell
}
