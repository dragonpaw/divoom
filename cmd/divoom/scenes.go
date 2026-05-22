package main

import (
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// Element IDs. Always-on top reserves 1-3; scene primaries start at 10.
// Each scene's layout is its own install, so re-using IDs across scenes is
// fine; we keep the IDs distinct only within a single scene.
const (
	idDay  = 1
	idTime = 2
	idDate = 3

	idSceneTitle = 9
	idSceneMain  = 10
	idSceneSub1  = 11
	idSceneSub2  = 12
	idSceneSub3  = 13
	idSceneSub4  = 14
	idSceneSub5  = 15
)

// Font IDs on the device. All custom-pushed via adb (see docs/api.md →
// "Fonts on disk"). Iosevka for digits/symbols, Roboto Condensed for prose.
const (
	fontMono       = 7  // Iosevka — numbers, ticker symbols, percentages
	fontProse      = 9  // Roboto Condensed — labels, prose, condition words
	fontProseLight = 11 // Roboto Condensed Light — secondary prose, fine print
)

// On-device per-scene background paths. We adb-push one JPG per scene so
// each can have its own glyph in the bottom area.
const (
	bgMarkets    = "/userdata/wallclock_bg_markets.jpg"
	bgMoonphase  = "/userdata/wallclock_bg_moonphase.jpg"
	bgHN         = "/userdata/wallclock_bg_hn.jpg"
	bgDevil      = "/userdata/wallclock_bg_devil.jpg"
	bgDayOfYear  = "/userdata/wallclock_bg_dayofyear.jpg"
	bgEaster     = "/userdata/wallclock_bg_easter.jpg"
	bgBabylon5   = "/userdata/wallclock_bg_babylon5.jpg"
	bgStarTrek   = "/userdata/wallclock_bg_startrek.jpg"
	bgDiscworld  = "/userdata/wallclock_bg_discworld.jpg"
	bgJargon     = "/userdata/wallclock_bg_jargon.jpg"
	bgCatFacts   = "/userdata/wallclock_bg_catfacts.jpg"
	bgDidYouKnow = "/userdata/wallclock_bg_didyouknow.jpg"
	bgSunrise    = "/userdata/wallclock_bg_sunrise.jpg"
	bgZenQuotes  = "/userdata/wallclock_bg_zenquotes.jpg"
	bgNASA       = "/userdata/wallclock_bg_nasa.jpg"
	bgCocktail   = "/userdata/wallclock_bg_cocktail.jpg"
	bgOnThisDay  = "/userdata/wallclock_bg_onthisday.jpg"
	bgISS        = "/userdata/wallclock_bg_iss.jpg"
	bgGitHub     = "/userdata/wallclock_bg_github.jpg"
	bgTIL        = "/userdata/wallclock_bg_til.jpg"
	bgWordnik    = "/userdata/wallclock_bg_wordnik.jpg"
	bgStoics     = "/userdata/wallclock_bg_stoics.jpg"
	bgTwain      = "/userdata/wallclock_bg_twain.jpg"
	bgFortune    = "/userdata/wallclock_bg_fortune.jpg"

	// One bg per weather outlook so the icon in the bottom-right corner
	// matches the current condition. Selected at activation time via
	// Scene.BgPathFor; all eight are pre-pushed at startup.
	bgWeatherClear    = "/userdata/wallclock_bg_weather_clear.jpg"
	bgWeatherCloudy   = "/userdata/wallclock_bg_weather_cloudy.jpg"
	bgWeatherOvercast = "/userdata/wallclock_bg_weather_overcast.jpg"
	bgWeatherRain     = "/userdata/wallclock_bg_weather_rain.jpg"
	bgWeatherDrizzle  = "/userdata/wallclock_bg_weather_drizzle.jpg"
	bgWeatherSnow     = "/userdata/wallclock_bg_weather_snow.jpg"
	bgWeatherFog      = "/userdata/wallclock_bg_weather_fog.jpg"
	bgWeatherThunder  = "/userdata/wallclock_bg_weather_thunder.jpg"
	bgWeatherSmoke    = "/userdata/wallclock_bg_weather_smoke.jpg"
	bgWeatherHazard   = "/userdata/wallclock_bg_weather_hazard.jpg"
)

// weatherOutlooks lists every outlook bucket the weather widget can emit,
// in the same order used for log output / preview rendering. Each entry's
// bg path matches one of the bgWeather* constants above.
var weatherOutlooks = []struct {
	Outlook string
	BgPath  string
}{
	{"clear", bgWeatherClear},
	{"cloudy", bgWeatherCloudy},
	{"overcast", bgWeatherOvercast},
	{"rain", bgWeatherRain},
	{"drizzle", bgWeatherDrizzle},
	{"snow", bgWeatherSnow},
	{"fog", bgWeatherFog},
	{"thunder", bgWeatherThunder},
	{"smoke", bgWeatherSmoke},
	{"hazard", bgWeatherHazard},
}

// weatherBgFor maps an outlook string to its on-device bg path, used by
// the weather scene's BgPathFor callback. Unknown outlooks (e.g. an empty
// cache before first fetch) fall back to the cloudy bg.
func weatherBgFor(outlook string) string {
	for _, o := range weatherOutlooks {
		if o.Outlook == outlook {
			return o.BgPath
		}
	}
	return bgWeatherCloudy
}

// Gruvbox semantic colors. Reds and greens signal direction (down/up);
// yellow / blue / aqua signal weather conditions; fg / fg-dark are quiet.
const (
	cFg     = "#ebdbb2"
	cFgDark = "#a89984"
	cRed    = "#fb4934"
	cGreen  = "#b8bb26"
	cYellow = "#fabd2f"
	cBlue   = "#83a598"
	cAqua   = "#8ec07b"
	cPurple = "#d3869b"
	cOrange = "#fe8019"
	cBgHard = "#1d2021"
)

// Vertical layout (800x1280 portrait):
//   y=20-100    Day of week (Text, color picked from weekday)
//   y=120-340   Time (huge, color picked from AM vs PM)
//   y=370-430   Date (built-in, fg-dark)
//   y=460-462   Hairline divider (rendered into bg)
//   y=480-1240  Scene-specific content
//   y=1268-1272 Year-progress bar (rendered into bg)

// dayColors picks a gruvbox accent per weekday, sweeping the palette
// through the week so each day reads distinctly at a glance.
var dayColors = map[time.Weekday]string{
	time.Sunday:    cPurple,
	time.Monday:    cRed,
	time.Tuesday:   cOrange,
	time.Wednesday: cYellow,
	time.Thursday:  cGreen,
	time.Friday:    cAqua,
	time.Saturday:  cBlue,
}

// timeColor returns the AM/PM accent for the clock — cAqua mornings,
// cOrange afternoons/evenings — so the clock reads warm or cool at a
// glance.
func timeColor(now time.Time) string {
	if now.Hour() < 12 {
		return cAqua
	}
	return cOrange
}

func alwaysOn(now time.Time) []frame.DispElement {
	return []frame.DispElement{
		{
			ID: idDay, Type: "Text",
			StartX: 50, StartY: 20, Width: 700, Height: 80,
			Align:       2,
			FontSize:    64,
			FontID:      fontProse,
			FontColor:   dayColors[now.Weekday()],
			BgColor:     cBgHard,
			TextMessage: now.Weekday().String(),
		},
		{
			ID: idTime, Type: "Time",
			StartX: 50, StartY: 120, Width: 700, Height: 220,
			Align:     2,
			FontSize:  180,
			FontID:    fontMono,
			FontColor: timeColor(now),
			BgColor:   cBgHard,
		},
		{
			ID: idDate, Type: "Date",
			StartX: 50, StartY: 370, Width: 700, Height: 60,
			Align:     2,
			FontSize:  44,
			FontID:    fontProse,
			FontColor: cFgDark,
			BgColor:   cBgHard,
		},
	}
}

// sceneTitle returns the canonical scene-title element — small, dim,
// Roboto Condensed Light, centred at y=480 with 10% margins. Every
// scene should use this so the title row looks identical across the
// rotation. Pass a short label like "did you know?" or "ISS overhead".
func sceneTitle(text string) frame.DispElement {
	return frame.DispElement{
		ID: idSceneTitle, Type: "Text",
		StartX: 80, StartY: 480, Width: 640, Height: 40,
		Align: 2, FontSize: 26, FontID: fontProseLight,
		FontColor: cFgDark, BgColor: cBgHard,
		TextMessage: text,
	}
}

// buildScenes returns the configured scene rotation. `widgets` maps a
// scene's Name to the Widget that supplies its dynamic text; scenes
// not present in the map render with a nil Widget (static content
// only). The github scene is gated on its widget being wired in (see
// cmd/divoom/serve.go).
func buildScenes(widgets map[string]widget.Widget) []*scene.Scene {
	scenes := []*scene.Scene{
		marketsScene(widgets),
		moonphaseScene(widgets),
		hnScene(widgets),
		dayofyearScene(widgets),
		babylon5Scene(widgets),
		startrekScene(widgets),
		discworldScene(widgets),
		jargonScene(widgets),
		zenquotesScene(widgets),
		wordnikScene(widgets),
		stoicsScene(widgets),
		twainScene(widgets),
		fortuneScene(widgets),
		devilScene(widgets),
		catfactsScene(widgets),
		tilScene(widgets),
		didyouknowScene(widgets),
		onthisdayScene(widgets),
		sunriseScene(widgets),
		weatherScene(widgets),
		nasaScene(widgets),
		cocktailScene(widgets),
		easterScene(widgets),
		issScene(widgets),
	}
	if widgets["github"] != nil {
		scenes = append(scenes, githubScene(widgets))
	}
	return scenes
}

// --- github formatters ---
//
// Widget output: "<today_commits>|<streak_days>|<open_prs>", e.g. "3|42|7".
// Each formatter pulls one segment and tags the right colour/label so the
// scene's three rows read with their meaning attached.

func githubCommits(raw string) (text, color string) {
	parts := strings.Split(raw, "|")
	if len(parts) < 1 {
		return "0", cFgDark
	}
	n, _ := strconv.Atoi(parts[0])
	if n > 0 {
		return parts[0], cGreen
	}
	return parts[0], cFgDark
}

func githubStreak(raw string) (text, color string) {
	parts := strings.Split(raw, "|")
	if len(parts) < 2 {
		return "0d", cFgDark
	}
	n, _ := strconv.Atoi(parts[1])
	c := cFgDark
	if n > 7 {
		c = cYellow
	}
	return parts[1] + "d streak", c
}

func githubPRs(raw string) (text, color string) {
	parts := strings.Split(raw, "|")
	if len(parts) < 3 {
		return "0 open PRs", cAqua
	}
	return parts[2] + " open PRs", cAqua
}

// --- QQQ formatters ---
//
// Widget output: "QQQ  -0.6% 1W   +9.2% 1M". Split into three rows;
// week and month rows take their color from the sign of their value.

func qqqSymbol(s string) (text, color string) {
	if parts := strings.Fields(s); len(parts) > 0 {
		return parts[0], ""
	}
	return s, ""
}

// qqqWeekPct returns just the week percent (e.g. "+9.2%") colored by sign.
// The "1W" suffix is dropped — the row's "week" caption supplies that.
func qqqWeekPct(s string) (text, color string) {
	parts := strings.Fields(s)
	if len(parts) < 2 {
		return s, ""
	}
	return parts[1], directionalColor(parts[1])
}

// qqqMonthPct returns just the month percent, similarly.
func qqqMonthPct(s string) (text, color string) {
	parts := strings.Fields(s)
	if len(parts) < 4 {
		return s, ""
	}
	return parts[3], directionalColor(parts[3])
}

// directionalColor: green for positive (up), red for negative (down),
// neutral for zero or unparseable.
func directionalColor(s string) string {
	switch {
	case strings.HasPrefix(s, "+"):
		return cGreen
	case strings.HasPrefix(s, "-"):
		return cRed
	default:
		return ""
	}
}

// --- pipe-separated segment formatters ---
//
// Sources emit pipe-separated fields ("Label|body" for whimsy,
// "Source|body|author" for quotes). pipeAt picks the i-th segment;
// pipeAtColor does the same and tags the rendered text with a fixed
// color so the scene can highlight the segment (e.g. quote authors in
// gruvbox aqua).

func pipeAt(i int) func(raw string) (text, color string) {
	return func(raw string) (text, color string) {
		parts := strings.Split(raw, "|")
		if i < 0 || i >= len(parts) {
			return "", ""
		}
		return parts[i], ""
	}
}

func pipeAtColor(i int, c string) func(raw string) (text, color string) {
	return func(raw string) (text, color string) {
		parts := strings.Split(raw, "|")
		if i < 0 || i >= len(parts) {
			return "", c
		}
		return parts[i], c
	}
}

// --- dictionary entry formatters ---
//
// Devil's Dictionary and Jargon File both emit "Source|<entry>|<author>"
// where the entry is shaped as "HEADWORD[,] [pronunciation]? POS. definition".
// Devil's uses an uppercase headword followed by a comma; Jargon uses a
// lowercase headword with an optional slash-bracketed pronunciation list.
// dictionaryEntryRE captures (headword, pos, definition) from either; the
// dictionaryWord / dictionaryPOS / dictionaryDefinition formatters surface
// one piece each to its own Text element so the scene reads as a dictionary
// entry instead of one undifferentiated paragraph. The formatters return
// "" for color — the scene's element-level FontColor carries the visual
// styling (yellow for Jargon, red for Devil's).
//
// POS atoms are an explicit allow-list of the tokens actually used by the
// two corpora (n, v, vi, vt, adj, adv, prep, conj, pp, interj, pron, num,
// art, excl, pl, i, t, imp, abbrev); compound forms like "n.,vi" and
// "v.t." are built by joining atoms with "." or ",". A short allow-list
// prevents the headword group from greedily eating a real noun when the
// POS marker happens to be missing its trailing dot (e.g. "code monkey n
// 1. A person...").
var dictionaryEntryRE = regexp.MustCompile(
	`^(.+?),?\s+(?:/[^/]+/(?:,\s*/[^/]+/)*\s+)?` +
		`((?:n|v|vi|vt|adj|adv|prep|conj|pp|interj|pron|num|art|excl|pl|i|t|imp|abbrev)` +
		`(?:\.?[.,](?:n|v|vi|vt|adj|adv|prep|conj|pp|interj|pron|num|art|excl|pl|i|t|imp|abbrev))*)` +
		`\.?\s+(.+)$`,
)

// dictionaryBody returns segment 1 of the widget's "Source|body|author"
// output — the dictionary entry text itself.
func dictionaryBody(raw string) string {
	parts := strings.Split(raw, "|")
	if len(parts) < 2 {
		return raw
	}
	return parts[1]
}

func dictionaryWord(raw string) (text, color string) {
	body := dictionaryBody(raw)
	if m := dictionaryEntryRE.FindStringSubmatch(body); m != nil {
		return m[1], ""
	}
	// Fallback: first word up to a space, so the slot is never empty.
	if i := strings.IndexByte(body, ' '); i > 0 {
		return body[:i], ""
	}
	return body, ""
}

func dictionaryPOS(raw string) (text, color string) {
	body := dictionaryBody(raw)
	if m := dictionaryEntryRE.FindStringSubmatch(body); m != nil {
		return m[2] + ".", ""
	}
	return "", ""
}

func dictionaryDefinition(raw string) (text, color string) {
	body := dictionaryBody(raw)
	if m := dictionaryEntryRE.FindStringSubmatch(body); m != nil {
		return m[3], ""
	}
	return body, ""
}

// vCenterQuoteBody shifts the quote body's StartY downward so a short
// (one- or two-line) quote sits visually centred within its declared
// track between the source label above and the author block below.
// Long quotes that would fill or overflow the track are left anchored
// at the track top so the device's wrapping/clipping behaves as before.
//
// charsPerLine and lineHeight are empirical estimates for the body's
// FontSize 34 + width 760 + prose font combination (see docs/api.md).
// Adjust here when the device's rendering math is better understood.
func vCenterQuoteBody(text string, e frame.DispElement) frame.DispElement {
	const (
		charsPerLine = 32
		lineHeight   = 45
		trackTop     = 540
		trackBottom  = 1120
	)
	const trackH = trackBottom - trackTop
	lines := (len(text) + charsPerLine - 1) / charsPerLine
	if lines < 1 {
		lines = 1
	}
	rendered := lines * lineHeight
	if rendered >= trackH {
		e.StartY = trackTop
		e.Height = trackH
		return e
	}
	e.StartY = trackTop + (trackH-rendered)/2
	e.Height = rendered
	return e
}

// shrinkHeadword reduces a dictionary headword's FontSize when the text
// would otherwise wrap. Estimates the rendered width as
// `len(text) * FontSize * charWidthRatio`; if that exceeds the 640 px
// budget, scales the FontSize down proportionally (clamped to a sane
// minimum). With a condensed font this rarely fires — most headwords
// fit at the default 90 px.
func shrinkHeadword(text string, e frame.DispElement) frame.DispElement {
	const (
		maxFontSize    = 90
		minFontSize    = 44
		widthBudget    = 640
		charWidthRatio = 0.45 // empirical for Roboto Condensed Light
	)
	if text == "" {
		return e
	}
	estimated := int(float64(len(text)) * float64(maxFontSize) * charWidthRatio)
	if estimated <= widthBudget {
		e.FontSize = maxFontSize
		return e
	}
	shrunk := int(float64(widthBudget) / (float64(len(text)) * charWidthRatio))
	if shrunk < minFontSize {
		shrunk = minFontSize
	}
	e.FontSize = shrunk
	return e
}

// fitDictionaryBody shrinks the dictionary scene's definition FontSize
// (within a clamped range) so even long entries fit inside the body
// track without the device clipping them, then vertically centres
// what's left so short definitions don't anchor to the top.
//
// The track is y=720..1100 (380px), below the headword+POS rows.
// charWidthRatio and lineHeightFrac are empirical estimates for
// Roboto Condensed Light at the FontSizes we walk through; tune via
// device probing if entries still clip.
func fitDictionaryBody(text string, e frame.DispElement) frame.DispElement {
	const (
		maxFontSize    = 34
		minFontSize    = 22
		trackTop       = 720
		trackBottom    = 1100
		charWidthRatio = 0.45 // px per char ≈ FontSize * this
		lineHeightFrac = 1.30 // px per line ≈ FontSize * this
	)
	const trackH = trackBottom - trackTop
	if text == "" {
		e.StartY = trackTop
		e.Height = trackH
		return e
	}

	// Walk FontSize from the preferred max down to the minimum, picking
	// the first one whose estimated rendered height fits inside the
	// track. Falls back to the minimum if even that overflows.
	fs := minFontSize
	rendered := trackH
	for size := maxFontSize; size >= minFontSize; size-- {
		charsPerLine := int(float64(e.Width) / (float64(size) * charWidthRatio))
		if charsPerLine < 1 {
			charsPerLine = 1
		}
		lines := (len(text) + charsPerLine - 1) / charsPerLine
		if lines < 1 {
			lines = 1
		}
		h := int(float64(lines*size) * lineHeightFrac)
		if h <= trackH {
			fs = size
			rendered = h
			break
		}
	}

	e.FontSize = fs
	if rendered >= trackH {
		e.StartY = trackTop
		e.Height = trackH
		return e
	}
	e.StartY = trackTop + (trackH-rendered)/2
	e.Height = rendered
	return e
}

// --- weather formatters ---
//
// Widget output: "<temp>°|<outlook>|<hazard>", e.g. "63°|clear|" or
// "78°|hazard|Red Flag Warning". weatherTempFrom / weatherOutlookFrom
// pick the two leading segments; the hazard segment is wired directly
// via pipeAt(2) on the scene mount. weatherCondition colours its half
// by outlook (red for "hazard", orange for "smoke", etc.); weatherTemp
// colours its half by temperature value, except outlook "hazard"
// forces red regardless of the temperature reading.

// weatherTempFrom returns the leading "<temp>°" segment, or the whole
// raw string when no pipe is present (defensive fallback for stale
// pre-merge widget output).
func weatherTempFrom(raw string) string {
	if i := strings.IndexByte(raw, '|'); i >= 0 {
		return raw[:i]
	}
	return raw
}

// weatherOutlookFrom returns the outlook segment (the second pipe-
// separated field). Empty string when the segment is missing.
func weatherOutlookFrom(raw string) string {
	parts := strings.Split(raw, "|")
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

// weatherOutlookColor returns the gruvbox accent for a given outlook —
// yellow for clear, blue for rain/drizzle, fg for snow, aqua for fog,
// fg-dark for cloudy/overcast, red for thunder, orange for smoke, red
// for hazard. Reproduces the old "now" scene's colour coding plus the
// two consolidated hazard buckets.
func weatherOutlookColor(outlook string) string {
	switch outlook {
	case "clear":
		return cYellow
	case "rain", "drizzle":
		return cBlue
	case "snow":
		return cFg
	case "fog":
		return cAqua
	case "cloudy", "overcast":
		return cFgDark
	case "thunder":
		return cRed
	case "smoke":
		return cOrange
	case "hazard":
		return cRed
	default:
		return cFg
	}
}

// Weather threshold defaults — overridden once
// weather.Client.LoadThresholds returns a climate fit for the configured
// location. The Fahrenheit defaults match Bay Area weather (where this
// dashboard was first calibrated); SetWeatherThresholds replaces them
// with location-specific 15th/85th percentile bounds in whichever unit
// the weather client is configured for.
var (
	weatherColdBound atomic.Int32 // temp below which reading is "blue"
	weatherHotBound  atomic.Int32 // temp at/above which reading is "orange"
)

func init() {
	weatherColdBound.Store(50)
	weatherHotBound.Store(80)
}

// SetWeatherThresholds replaces the dynamic cold/hot bounds used by
// weatherTempColor. Called from serve.go once the weather widget's
// LoadThresholds fetch completes (or up front with sensible seed
// values for the configured unit). The fixed comfort band
// (68-75°F / 20-24°C) is unaffected.
func SetWeatherThresholds(cold, hot int) {
	weatherColdBound.Store(int32(cold))
	weatherHotBound.Store(int32(hot))
}

// weatherTempColor maps an integer temperature reading to a gruvbox
// accent, scaling from cold (blue) through comfortable (green) up to
// hot (red). The cold and hot bounds are dynamic (auto-calibrated to
// the configured location's climate via SetWeatherThresholds); the
// comfort band in the middle is fixed: 68-75°F for "F" or 20-24°C for
// anything else.
func weatherTempColor(temp int, unit string) string {
	cold := int(weatherColdBound.Load())
	hot := int(weatherHotBound.Load())
	comfortLo, comfortHi := 68, 75
	hotOverhead := 5
	if unit != "F" {
		comfortLo, comfortHi = 20, 24
		hotOverhead = 3 // ~5°F in °C
	}
	switch {
	case temp < cold:
		return cBlue
	case temp < comfortLo:
		return cAqua
	case temp <= comfortHi:
		return cGreen
	case temp <= hot:
		return cYellow
	case temp <= hot+hotOverhead:
		return cOrange
	default:
		return cRed
	}
}

// weatherTemp colours the temperature segment by value, except an
// outlook of "hazard" forces red so the alert reading is unmissable
// regardless of the actual temperature.
func weatherTemp(raw string) (text, color string) {
	temp := weatherTempFrom(raw)
	if weatherOutlookFrom(raw) == "hazard" {
		return temp, cRed
	}
	// temp looks like "63°F" or "20°C". The unit letter drives the
	// comfort band; strip it (and the degree sign) before atoi.
	n, unit, ok := parseWeatherTemp(temp)
	if !ok {
		return temp, cFg
	}
	return temp, weatherTempColor(n, unit)
}

// parseWeatherTemp pulls the integer reading and the unit letter ("F"
// or "C") out of a "<n>°<unit>" string. Returns ok=false for any input
// that doesn't match the shape — callers fall back to the default
// foreground colour rather than guess.
func parseWeatherTemp(s string) (n int, unit string, ok bool) {
	i := strings.Index(s, "°")
	if i < 0 {
		return 0, "", false
	}
	num, err := strconv.Atoi(s[:i])
	if err != nil {
		return 0, "", false
	}
	return num, s[i+len("°"):], true
}

func weatherCondition(raw string) (text, color string) {
	outlook := weatherOutlookFrom(raw)
	return outlook, weatherOutlookColor(outlook)
}

// --- promoted-quote scene helper ---
//
// The six promoted quote/dictionary scenes (babylon5, startrek, discworld,
// zenquotes, devil, plus jargon's structural sibling) share a near-identical
// four-body-element shape: small source label, large body, optional author
// block, optional static tagline. QuoteScene assembles one from a few
// declarative options so the per-scene blocks stay short and obviously
// correct. All text uses fontProseLight (Roboto Condensed Light); margins
// are 10% of the 800px canvas — StartX 80, Width 640 — on every element.

// QuoteSceneOpts describes a promoted-quote scene. Tagline == "" omits the
// fourth element; HasAuthor == false omits the author mount. Title is the
// short label shown in the canonical sceneTitle row at the top of the body
// area (typically the source name — "Babylon 5", "Star Trek", etc.).
type QuoteSceneOpts struct {
	Name         string
	Title        string
	Weight       int
	BgPath       string
	Widget       widget.Widget
	Tagline      string
	TaglineColor string
	HasAuthor    bool
}

// QuoteScene returns the *scene.Scene for a promoted-quote layout. IDs are
// assigned sequentially: title -> idSceneTitle, body -> idSceneSub1,
// author -> idSceneSub2 (when HasAuthor), tagline -> idSceneSub3 (when
// Tagline != "").
func QuoteScene(opts QuoteSceneOpts) *scene.Scene {
	elements := []frame.DispElement{
		sceneTitle(opts.Title),
		quoteBody(idSceneSub1),
	}
	mounts := []scene.Mount{
		{ID: idSceneSub1, Format: pipeAt(1), Geometry: vCenterQuoteBody},
	}
	if opts.HasAuthor {
		elements = append(elements, quoteAuthor(idSceneSub2))
		mounts = append(mounts, scene.Mount{
			ID: idSceneSub2, Format: pipeAtColor(2, cAqua), AllowEmpty: true,
		})
	}
	if opts.Tagline != "" {
		elements = append(elements, quoteTagline(idSceneSub3, opts.Tagline, opts.TaglineColor))
	}
	return &scene.Scene{
		Name:     opts.Name,
		Weight:   opts.Weight,
		BgPath:   opts.BgPath,
		Elements: elements,
		Widget:   opts.Widget,
		Mounts:   mounts,
	}
}

func quoteBody(id int) frame.DispElement {
	return frame.DispElement{
		ID: id, Type: "Text",
		StartX: 80, StartY: 540, Width: 640, Height: 520,
		Align: 2, FontSize: 34, FontID: fontProseLight,
		FontColor: cFg, BgColor: cBgHard,
	}
}

func quoteAuthor(id int) frame.DispElement {
	return frame.DispElement{
		ID: id, Type: "Text",
		StartX: 80, StartY: 1080, Width: 640, Height: 70,
		Align: 2, FontSize: 32, FontID: fontProseLight,
		FontColor: cAqua, BgColor: cBgHard,
	}
}

func quoteTagline(id int, text, color string) frame.DispElement {
	return frame.DispElement{
		ID: id, Type: "Text",
		StartX: 80, StartY: 1160, Width: 640, Height: 50,
		Align: 2, FontSize: 26, FontID: fontProseLight,
		FontColor: color, BgColor: cBgHard,
		TextMessage: text,
	}
}

// --- dictionary scene helper ---
//
// DictionaryScene builds a scene that renders a dictionary-shaped entry
// (Devil's Dictionary, Jargon File) as four distinct typed regions:
// source label, big mono headword, medium aqua part-of-speech, body
// definition. Optionally adds an author block (Devil's carries
// "Ambrose Bierce") and a static tagline. Shares the 10%-margin
// (StartX 80, Width 640) convention with QuoteScene.

// DictionarySceneOpts describes a dictionary-shaped scene. HasAuthor adds
// the author element + mount (segment 2 of the widget output). Tagline
// adds a static caption beneath everything else.
//
// Colours are intentionally NOT options: every dictionary scene uses
// the same palette (yellow headword, aqua POS/author, dim tagline) so
// they read as a consistent typographic family even when the source
// material differs.
type DictionarySceneOpts struct {
	Name      string
	Title     string
	Weight    int
	BgPath    string
	Widget    widget.Widget
	HasAuthor bool
	Tagline   string
}

func DictionaryScene(opts DictionarySceneOpts) *scene.Scene {
	elements := []frame.DispElement{
		sceneTitle(opts.Title),
		// Headword (big condensed, scene-chosen colour). Height tightened
		// to just clear FontSize 90 so the POS sits right under it
		// rather than 50px of empty padding away. The Geometry callback
		// auto-shrinks the FontSize for long headwords so they fit on
		// one line instead of wrapping; condensed font means this rarely
		// triggers in practice.
		{
			ID: idSceneSub1, Type: "Text",
			StartX: 80, StartY: 540, Width: 640, Height: 100,
			Align: 2, FontSize: 90, FontID: fontProseLight,
			FontColor: cYellow, BgColor: cBgHard,
		},
		// Part of speech (medium prose, aqua).
		{
			ID: idSceneSub2, Type: "Text",
			StartX: 80, StartY: 650, Width: 640, Height: 50,
			Align: 2, FontSize: 36, FontID: fontProseLight,
			FontColor: cAqua, BgColor: cBgHard,
		},
		// Definition (body prose, fg, vertically centred within its
		// own track so long entries never bleed up into the headword).
		{
			ID: idSceneSub3, Type: "Text",
			StartX: 80, StartY: 720, Width: 640, Height: 380,
			Align: 2, FontSize: 34, FontID: fontProseLight,
			FontColor: cFg, BgColor: cBgHard,
		},
	}
	mounts := []scene.Mount{
		{ID: idSceneSub1, Format: dictionaryWord, Geometry: shrinkHeadword},
		{ID: idSceneSub2, Format: dictionaryPOS, AllowEmpty: true},
		{ID: idSceneSub3, Format: dictionaryDefinition, Geometry: fitDictionaryBody},
	}
	if opts.HasAuthor {
		elements = append(elements, frame.DispElement{
			ID: idSceneSub4, Type: "Text",
			StartX: 80, StartY: 1110, Width: 640, Height: 50,
			Align: 2, FontSize: 32, FontID: fontProseLight,
			FontColor: cAqua, BgColor: cBgHard,
		})
		mounts = append(mounts, scene.Mount{
			ID: idSceneSub4, Format: pipeAt(2), AllowEmpty: true,
		})
	}
	if opts.Tagline != "" {
		elements = append(elements, frame.DispElement{
			ID: idSceneSub5, Type: "Text",
			StartX: 80, StartY: 1170, Width: 640, Height: 50,
			Align: 2, FontSize: 26, FontID: fontProseLight,
			FontColor: cFgDark, BgColor: cBgHard,
			TextMessage: opts.Tagline,
		})
	}
	return &scene.Scene{
		Name:     opts.Name,
		Weight:   opts.Weight,
		BgPath:   opts.BgPath,
		Elements: elements,
		Widget:   opts.Widget,
		Mounts:   mounts,
	}
}

// --- moon formatters ---

func moonPhaseName(s string) (text, color string) {
	parts := strings.Split(s, " · ")
	if len(parts) >= 2 {
		return parts[1], ""
	}
	return s, ""
}

func moonIllum(s string) (text, color string) {
	parts := strings.Split(s, " · ")
	if len(parts) >= 3 {
		return parts[2] + " lit", ""
	}
	return "", ""
}

// moonNextFullMoon picks the fourth segment — the precomputed
// "full moon in N days" / "next full moon: Jun 1" countdown string.
func moonNextFullMoon(s string) (text, color string) {
	parts := strings.Split(s, " · ")
	if len(parts) >= 4 {
		return parts[3], ""
	}
	return "", ""
}
