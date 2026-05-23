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
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"

	// Side-effect imports register decoders with image.Decode so the
	// loadCachedAPODImage path can handle every APOD format on its
	// own. JPEG is registered via the image/jpeg import above; PNG
	// and GIF need explicit blank imports — the 1995-era APOD entries
	// are almost all GIF.
	_ "image/gif"
	_ "image/png"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"strconv"
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
	if err := bakeAllCocktailBackgrounds(ctx); err != nil {
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
			// Quota-exhausted means every subsequent fetch will also
			// fail — stop the whole bake instead of burning 100+ more
			// HTTP roundtrips just to hit the same wall.
			if errors.Is(err, errAPODQuotaExhausted) {
				slog.Warn("nasa quota exhausted; aborting remaining bakes",
					"completed", i, "remaining", len(nasaCuratedDates)-i, "err", err)
				return err
			}
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

// errAPODQuotaExhausted is returned by fetchAPOD when APOD signals
// (via a long Retry-After) that the daily quota is gone. Callers
// short-circuit the per-date loop on this sentinel since every
// subsequent fetch will return the same error.
var errAPODQuotaExhausted = errors.New("apod daily quota exhausted")

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

	// Prefer the standard-res `url` over `hdurl` — the display slot is
	// only 760×540 so the multi-MB HD versions are pure download cost
	// for no visible benefit, and they're the most likely target for
	// CDN throttling during a 123-image push.
	imgURL := body.URL
	if imgURL == "" {
		imgURL = body.HDURL
	}
	if imgURL == "" {
		return nil, fmt.Errorf("apod %s has no image url", date)
	}

	photo, err := loadCachedAPODImage(ctx, date, imgURL)
	if err != nil {
		return nil, fmt.Errorf("load apod image: %w", err)
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
// returns today's entry; otherwise the YYYY-MM-DD entry. Honours 429
// with Retry-After (sleeps then retries up to nasaRateLimitRetries
// times) — needed for the 123-image push run which can otherwise burn
// through the hourly quota mid-bake.
//
// Successful responses are cached to disk (one JSON per date under
// apodCacheDir) so subsequent pushes skip the network entirely. APOD
// historical entries don't change once published, so the cache has
// no TTL; delete the cache dir manually if you ever need to refetch.
func fetchAPOD(ctx context.Context, apiKey, date string) (*nasaAPODResponse, error) {
	if cached, ok := readCachedAPOD(date); ok {
		return cached, nil
	}
	const nasaRateLimitRetries = 4
	url := nasaAPIBase + "?api_key=" + apiKey
	if date != "" {
		url += "&date=" + date
	}
	client := &http.Client{Timeout: httpFetchTimeout}
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			wait := retryAfterDuration(resp.Header.Get("Retry-After"))
			resp.Body.Close()
			// Long Retry-After means the daily quota is gone, not a
			// transient rate-limit blip. DEMO_KEY caps at 50/day; a
			// real key (sign up at api.nasa.gov, 30s) is 1000/hour.
			// Bail out with a useful error rather than burning the
			// whole push waiting for the next reset.
			const maxWait = 2 * time.Minute
			if wait > maxWait {
				return nil, fmt.Errorf("%w: Retry-After %s (set NASA_API_KEY to a real key from api.nasa.gov)",
					errAPODQuotaExhausted, wait.Round(time.Second))
			}
			if attempt >= nasaRateLimitRetries {
				return nil, fmt.Errorf("http 429 after %d retries", attempt)
			}
			slog.Warn("apod rate-limited (429); sleeping before retry",
				"date", date, "wait", wait, "attempt", attempt+1)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
			continue
		}
		if resp.StatusCode/100 != 2 {
			resp.Body.Close()
			return nil, fmt.Errorf("http %d", resp.StatusCode)
		}
		raw, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		var body nasaAPODResponse
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, err
		}
		writeCachedAPOD(date, raw)
		return &body, nil
	}
}

// apodCacheDir returns the on-disk directory for cached APOD JSON +
// image bodies (e.g. ~/.cache/divoom/apod/). Created on first use.
// On any environment-level failure (no home dir, can't create) the
// function returns "" so callers fall through to the live fetch
// path — caching is a speedup, never a correctness requirement.
func apodCacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	dir := base + "/divoom/apod"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	return dir
}

// readCachedAPOD returns the cached APOD response for date when one
// exists on disk. The ok bool is false on miss, on read failure, or
// on JSON parse failure — in all cases the caller refetches.
func readCachedAPOD(date string) (*nasaAPODResponse, bool) {
	dir := apodCacheDir()
	if dir == "" || date == "" {
		return nil, false
	}
	raw, err := os.ReadFile(dir + "/" + date + ".json")
	if err != nil {
		return nil, false
	}
	var body nasaAPODResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, false
	}
	return &body, true
}

// writeCachedAPOD stores the raw JSON response for date. Errors are
// logged but never propagated — failure to cache shouldn't fail the
// bake (the in-memory response is already decoded and usable).
func writeCachedAPOD(date string, raw []byte) {
	dir := apodCacheDir()
	if dir == "" || date == "" {
		return
	}
	if err := os.WriteFile(dir+"/"+date+".json", raw, 0o644); err != nil {
		slog.Warn("apod cache write failed", "date", date, "err", err)
	}
}

// loadCachedAPODImage returns the decoded APOD photo for date, using
// the on-disk cache when available. On cache miss the image is
// downloaded from imgURL, written to the cache, and decoded.
func loadCachedAPODImage(ctx context.Context, date, imgURL string) (image.Image, error) {
	dir := apodCacheDir()
	cachePath := ""
	if dir != "" && date != "" {
		cachePath = dir + "/" + date + ".img"
		if raw, err := os.ReadFile(cachePath); err == nil {
			if img, _, derr := image.Decode(bytes.NewReader(raw)); derr == nil {
				return img, nil
			}
			// Corrupt cache entry — fall through to re-download.
			slog.Warn("apod cached image failed to decode; refetching", "date", date)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imgURL, nil)
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
		return nil, fmt.Errorf("read body: %w", err)
	}
	if cachePath != "" {
		if err := os.WriteFile(cachePath, raw, 0o644); err != nil {
			slog.Warn("apod image cache write failed", "date", date, "err", err)
		}
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return img, nil
}

// retryAfterDuration parses an HTTP Retry-After header into a sleep
// duration. The header may be either an integer count of seconds
// ("60") or an HTTP-date ("Wed, 21 Oct 2026 07:28:00 GMT"). Returns a
// 60-second default when the header is empty or unparseable. No
// upper cap — if APOD asks us to wait an hour for the quota to
// reset, we wait an hour; ctx.Done() still aborts on Ctrl-C.
func retryAfterDuration(header string) time.Duration {
	const defaultWait = 60 * time.Second
	header = strings.TrimSpace(header)
	if header == "" {
		return defaultWait
	}
	if secs, err := strconv.Atoi(header); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(header); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return defaultWait
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
	// TheCocktailDB's free tier uses literal key "1" in the path. The
	// list endpoint returns ID + name only (filter by category); the
	// lookup endpoint returns the full drink record by ID.
	cocktailListAPIBase   = "https://www.thecocktaildb.com/api/json/v1/1/filter.php?c="
	cocktailLookupAPIBase = "https://www.thecocktaildb.com/api/json/v1/1/lookup.php?i="

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

// cocktailCategories drives the rotation pool. Cocktail covers the
// canonical "fancy drink" half; Shot covers the silly / weird /
// shooter half (Brain Hemorrhage, Cement Mixer, Liquid Cocaine, etc.).
// Mocktails excluded per design — the user wants the bar, not the
// soda fountain.
var cocktailCategories = []string{"Cocktail", "Shot"}

// bakeAllCocktailBackgrounds fetches the full Cocktail + Shot drink
// list from TheCocktailDB, looks up each drink, bakes the recipe-card
// bg, and pushes each to its indexed device path. The scene's
// BgPathFor picks a random index per activation. Per-drink data is
// cached under ~/.cache/divoom/cocktail/ so subsequent pushes skip
// the network entirely.
func bakeAllCocktailBackgrounds(ctx context.Context) error {
	ids, err := fetchCocktailIDList(ctx)
	if err != nil {
		return fmt.Errorf("fetch cocktail id list: %w", err)
	}
	if len(ids) == 0 {
		return fmt.Errorf("empty cocktail id list")
	}

	plainBg, err := render.SceneBackground(render.SceneCocktail, render.FormatJPEG, time.Now())
	if err != nil {
		return fmt.Errorf("render plain cocktail bg: %w", err)
	}

	for i, id := range ids {
		path := bgCocktailFor(i)
		out, err := bakeOneCocktail(ctx, id)
		if err != nil {
			slog.Warn("cocktail bake failed; pushing plain fallback for this index",
				"index", i, "id", id, "err", err)
			if perr := pushBytes(ctx, plainBg, path); perr != nil {
				slog.Warn("cocktail fallback push failed", "index", i, "err", perr)
			}
			continue
		}
		if perr := pushBytes(ctx, out, path); perr != nil {
			slog.Warn("cocktail push failed", "index", i, "id", id, "err", perr)
			continue
		}
	}
	slog.Info("cocktail bgs pushed", "count", len(ids))
	return nil
}

// bakeOneCocktail looks up drink id, composites the recipe card into
// the plain scene bg, and returns the encoded JPEG bytes.
func bakeOneCocktail(ctx context.Context, id string) ([]byte, error) {
	name, glass, category, instructions, rows, err := fetchCocktailByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetch drink: %w", err)
	}

	bgBytes, err := render.SceneBackground(render.SceneCocktail, render.FormatJPEG, time.Now())
	if err != nil {
		return nil, fmt.Errorf("render scene bg: %w", err)
	}
	canvas, err := jpegToRGBA(bgBytes)
	if err != nil {
		return nil, fmt.Errorf("decode scene bg: %w", err)
	}
	if err := drawCocktailCard(canvas, name, glass, category, instructions, rows); err != nil {
		return nil, fmt.Errorf("draw card: %w", err)
	}
	return encodeJPEG(canvas)
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

// fetchCocktailIDList queries TheCocktailDB filter endpoint for every
// category in cocktailCategories, merges the result lists, and
// returns a deduped, shuffled slice of drink IDs. The category list
// is itself cached on disk (per-category JSON) so subsequent runs
// skip the filter calls — only newly-added drinks would be missed,
// which is fine until you manually clear the cache.
//
// Shuffle order: a fresh PCG seeded from time.Now per call means each
// `make push-frame` run pushes the same drinks to *different* indexed
// slots. The scene picks a random index per activation anyway, but
// shuffling here means an interrupted push doesn't leave the device
// with "all the A drinks" — interrupt at index N and you've got a
// random subset of the catalogue, not the alphabetical prefix.
func fetchCocktailIDList(ctx context.Context) ([]string, error) {
	seen := make(map[string]bool)
	var ids []string
	for _, cat := range cocktailCategories {
		catIDs, err := fetchCocktailCategoryIDs(ctx, cat)
		if err != nil {
			return nil, fmt.Errorf("category %q: %w", cat, err)
		}
		for _, id := range catIDs {
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			ids = append(ids, id)
		}
	}
	rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0xC0CC7A11))
	rng.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
	return ids, nil
}

// fetchCocktailCategoryIDs hits filter.php?c=<category> and returns
// the idDrink values. Cached by category name under
// ~/.cache/divoom/cocktail/cat_<category>.json.
func fetchCocktailCategoryIDs(ctx context.Context, category string) ([]string, error) {
	dir := cocktailCacheDir()
	cachePath := ""
	if dir != "" {
		cachePath = dir + "/cat_" + sanitiseCacheKey(category) + ".json"
		if raw, err := os.ReadFile(cachePath); err == nil {
			if ids, ok := parseCocktailIDList(raw); ok {
				return ids, nil
			}
		}
	}
	endpoint := cocktailListAPIBase + url.QueryEscape(category)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "divoom-dashboard/0.1 (github.com/dragonpaw/divoom)")
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
	ids, ok := parseCocktailIDList(raw)
	if !ok {
		return nil, fmt.Errorf("could not parse drink list")
	}
	if cachePath != "" {
		if err := os.WriteFile(cachePath, raw, 0o644); err != nil {
			slog.Warn("cocktail list cache write failed", "category", category, "err", err)
		}
	}
	return ids, nil
}

func parseCocktailIDList(raw []byte) ([]string, bool) {
	var body cocktailResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, false
	}
	out := make([]string, 0, len(body.Drinks))
	for _, d := range body.Drinks {
		out = append(out, strSafe(d["idDrink"]))
	}
	return out, true
}

// fetchCocktailByID hits lookup.php?i=<id> and decodes the full drink
// record. The raw JSON is cached under
// ~/.cache/divoom/cocktail/<id>.json so subsequent pushes skip the
// network entirely.
func fetchCocktailByID(ctx context.Context, id string) (name, glass, category, instructions string, rows []recipeRow, err error) {
	dir := cocktailCacheDir()
	cachePath := ""
	if dir != "" && id != "" {
		cachePath = dir + "/" + id + ".json"
		if raw, rerr := os.ReadFile(cachePath); rerr == nil {
			if d, ok := parseCocktailDrink(raw); ok {
				return cocktailFields(d)
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cocktailLookupAPIBase+id, nil)
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
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", "", nil, err
	}
	d, ok := parseCocktailDrink(raw)
	if !ok {
		return "", "", "", "", nil, fmt.Errorf("lookup parse failed")
	}
	if cachePath != "" {
		if err := os.WriteFile(cachePath, raw, 0o644); err != nil {
			slog.Warn("cocktail cache write failed", "id", id, "err", err)
		}
	}
	return cocktailFields(d)
}

func parseCocktailDrink(raw []byte) (map[string]any, bool) {
	var body cocktailResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, false
	}
	if len(body.Drinks) == 0 {
		return nil, false
	}
	return body.Drinks[0], true
}

// cocktailFields extracts the name, glass, category, instructions and
// (measure, ingredient) pairs from a parsed drink map.
func cocktailFields(d map[string]any) (name, glass, category, instructions string, rows []recipeRow, err error) {
	name = strings.TrimSpace(strSafe(d["strDrink"]))
	glass = strings.TrimSpace(strSafe(d["strGlass"]))
	category = strings.TrimSpace(strSafe(d["strCategory"]))
	instructions = strings.TrimSpace(strSafe(d["strInstructions"]))
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

// cocktailCacheDir mirrors apodCacheDir but under cocktail/.
func cocktailCacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	dir := base + "/divoom/cocktail"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	return dir
}

// cocktailPoolSize counts cached drink JSONs in cocktailCacheDir so
// the scene's BgPathFor knows how many indexed paths exist. Returns 0
// if the cache dir is missing or unreadable, in which case the
// scene falls back to bgCocktailFor(0).
func cocktailPoolSize() int {
	dir := cocktailCacheDir()
	if dir == "" {
		return 0
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "cat_") {
			continue // category list cache, not a drink
		}
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		n++
	}
	return n
}

// sanitiseCacheKey makes a category name safe for use as a filename
// (no path separators, no spaces).
func sanitiseCacheKey(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
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
