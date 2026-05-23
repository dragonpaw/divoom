package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/render"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// CanvasW shadows render.CanvasW so scene-layout math reads naturally.
const CanvasW = render.CanvasW

// Element IDs. Always-on top reserves 1-3; scene primaries start at 10.
// Each scene's layout is its own install, so re-using IDs across scenes is
// fine; we keep the IDs distinct only within a single scene.
const (
	idDay    = 1
	idTime   = 2
	idFooter = 3

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

// moonBackgrounds is the on-device path for each of the 14 pre-rendered
// moon-disc variants spanning one synodic cycle (0 = new, 7 = full,
// 1-6 wax, 8-13 wane). Selected at scene-activation time by
// moonBgPathFor; all 14 are pushed up front by pushSceneBackgrounds.
// 14 is the design decision — covers ~2 days per variant on a 29.53-day
// cycle, finer than the ~3.4%/day human-detectable illumination change.
var moonBackgrounds = func() [render.MoonPhaseVariants]string {
	var paths [render.MoonPhaseVariants]string
	for i := 0; i < render.MoonPhaseVariants; i++ {
		paths[i] = fmt.Sprintf("/userdata/wallclock_bg_moonphase_%02d.jpg", i)
	}
	return paths
}()

// moonPhaseIndex maps an illumination percentage (0-100) plus a
// waxing/waning flag to one of the 14 variant indices. New moon (illum
// < 4) and full moon (illum > 96) collapse to indices 0 and 7
// regardless of the flag; everything else picks the nearest waxing
// (1-6) or waning (8-13) index by absolute illumination distance.
func moonPhaseIndex(illum int, waxing bool) int {
	if illum < 4 {
		return 0
	}
	if illum > 96 {
		return 7
	}
	lo, hi := 1, 6
	if !waxing {
		lo, hi = 8, 13
	}
	best := lo
	bestDiff := 101.0
	for i := lo; i <= hi; i++ {
		variantIllum := render.MoonIllumFractionForIndex(i) * 100
		diff := variantIllum - float64(illum)
		if diff < 0 {
			diff = -diff
		}
		if diff < bestDiff {
			bestDiff = diff
			best = i
		}
	}
	return best
}

// moonBgPathFor is the moonphase scene's BgPathFor callback. The widget
// emits "moon · <phase name> · <illum>% · <countdown>"; we parse out
// the illumination percent and whether the phase name is waxing /
// waning / new / full, then map to one of the 14 pre-pushed disc
// variants.
func moonBgPathFor(raw string) string {
	parts := strings.Split(raw, " · ")
	if len(parts) < 3 {
		return moonBackgrounds[7]
	}
	name := strings.TrimSpace(parts[1])
	illum := 0
	pct := strings.TrimSuffix(strings.TrimSpace(parts[2]), "%")
	if n, err := strconv.Atoi(pct); err == nil {
		illum = n
	}
	switch {
	case strings.HasPrefix(name, "new"):
		return moonBackgrounds[0]
	case strings.HasPrefix(name, "full"):
		return moonBackgrounds[7]
	}
	waxing := strings.HasPrefix(name, "waxing") || name == "first quarter"
	waning := strings.HasPrefix(name, "waning") || name == "last quarter"
	if !waxing && !waning {
		// Defensive: an unrecognised phase name still gets a real disc.
		return moonBackgrounds[7]
	}
	return moonBackgrounds[moonPhaseIndex(illum, waxing)]
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
//   y=30-110    "> dayname" prompt (Text, dayColor, fontMono)
//   y=140-340   Time (huge, solid fg, fontMono)
//   y=380-382   Morse-pattern rule (rendered into bg)
//   y=400-444   Operator footer: "YYYY-MM-DD  doy:N  w:N  weekend+Nd"
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

// isoWeek returns the ISO 8601 week number for now (the second value of
// time.Time.ISOWeek).
func isoWeek(now time.Time) int {
	_, w := now.ISOWeek()
	return w
}

// seasonAt returns the all-caps season name and its gruvbox accent for
// the month of `now`. WINTER (Dec/Jan/Feb) is cold cAqua; SPRING
// (Mar/Apr/May) is growth cGreen; SUMMER (Jun/Jul/Aug) is sun cYellow;
// AUTUMN (Sep/Oct/Nov) is foliage cOrange. Used by the dayofyear
// scene's OnActivate to colour-code the season label under the grid.
func seasonAt(now time.Time) (name, color string) {
	switch now.Month() {
	case time.December, time.January, time.February:
		return "WINTER", cAqua
	case time.March, time.April, time.May:
		return "SPRING", cGreen
	case time.June, time.July, time.August:
		return "SUMMER", cYellow
	default: // Sep, Oct, Nov
		return "AUTUMN", cOrange
	}
}

// timeColor returns the AM/PM accent for the always-on clock — cAqua
// mornings, cOrange afternoons/evenings — so the clock reads warm or
// cool at a glance.
func timeColor(now time.Time) string {
	if now.Hour() < 12 {
		return cAqua
	}
	return cOrange
}

// daysUntilWeekend returns the operator-footer countdown string. Mon-Fri
// render as "weekend+Nd" (Mon=4, Tue=3, ..., Fri=0); Sat/Sun render as
// "weekend".
func daysUntilWeekend(now time.Time) string {
	switch now.Weekday() {
	case time.Saturday, time.Sunday:
		return "weekend"
	default:
		// Monday=1 ... Friday=5; 5 - weekday days until Saturday.
		n := 5 - int(now.Weekday())
		return fmt.Sprintf("weekend+%dd", n)
	}
}

func alwaysOn(now time.Time) []frame.DispElement {
	return []frame.DispElement{
		{
			ID: idDay, Type: "Text",
			StartX: 40, StartY: 30, Width: 720, Height: 80,
			Align:       0,
			FontSize:    64,
			FontID:      fontMono,
			FontColor:   dayColors[now.Weekday()],
			BgColor:     cBgHard,
			TextMessage: "> " + strings.ToLower(now.Weekday().String()),
		},
		{
			ID: idTime, Type: "Time",
			StartX: 50, StartY: 140, Width: 700, Height: 200,
			Align:     2,
			FontSize:  160,
			FontID:    fontMono,
			FontColor: timeColor(now),
			BgColor:   cBgHard,
		},
		{
			ID: idFooter, Type: "Text",
			StartX: 40, StartY: 400, Width: 720, Height: 44,
			Align:     0,
			FontSize:  28,
			FontID:    fontMono,
			FontColor: cFgDark,
			BgColor:   cBgHard,
			TextMessage: fmt.Sprintf("%s  doy:%d  w:%d  %s",
				now.Format("2006-01-02"),
				now.YearDay(),
				isoWeek(now),
				daysUntilWeekend(now)),
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

// weatherConditionOrHazard renders the condition-or-hazard slot. When
// the hazard segment (pipe[2]) is non-empty there's an active NWS alert
// at the configured point; surface its full text in red so it's
// unmissable. Otherwise show the outlook word in its outlook colour.
func weatherConditionOrHazard(raw string) (text, color string) {
	parts := strings.Split(raw, "|")
	hazard := ""
	if len(parts) >= 3 {
		hazard = parts[2]
	}
	if hazard != "" {
		return hazard, cRed
	}
	outlook := weatherOutlookFrom(raw)
	return outlook, weatherOutlookColor(outlook)
}

// weatherPipeField pulls segment i of a pipe-separated raw string,
// returning "" when the segment is missing. Used by the three stats
// formatters below.
func weatherPipeField(raw string, i int) string {
	parts := strings.Split(raw, "|")
	if i >= len(parts) {
		return ""
	}
	return parts[i]
}

// weatherAQI renders the AQI value in column 1 of the console strip.
// Blank field (missing source / failed fetch) becomes a dim em-dash so
// it doesn't lie about clean air. Otherwise the integer is colour-coded
// by US EPA AQI band.
func weatherAQI(raw string) (text, color string) {
	v := weatherPipeField(raw, 3)
	if v == "" {
		return "—", cFgDark
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return v, cFg
	}
	return v, aqiColor(n)
}

// aqiColor maps an EPA AQI reading to its band colour. Bands are
// inclusive on the lower bound: 0-50 good, 51-100 moderate, 101-150
// unhealthy for sensitive groups, 151-200 unhealthy, 201-300 very
// unhealthy, 301+ hazardous.
func aqiColor(n int) string {
	switch {
	case n <= 50:
		return cGreen
	case n <= 100:
		return cYellow
	case n <= 150:
		return cOrange
	case n <= 200:
		return cRed
	case n <= 300:
		return cPurple
	default:
		return cRed
	}
}

// weatherHumidity renders the humidity value in column 2. Blank → dim
// em-dash; otherwise "<n>%" in blue (the column's element colour).
func weatherHumidity(raw string) (text, color string) {
	v := weatherPipeField(raw, 4)
	if v == "" {
		return "—", cFgDark
	}
	return v + "%", cBlue
}

// weatherRain renders the rain-chance value in column 3. Blank → dim
// em-dash; otherwise "<n>%" in aqua.
func weatherRain(raw string) (text, color string) {
	v := weatherPipeField(raw, 5)
	if v == "" {
		return "—", cFgDark
	}
	return v + "%", cAqua
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

// QuoteSceneOpts describes a promoted-quote scene. The Family field
// selects one of three baked-chrome layouts (FromSource, Marginalia,
// Terminal); the rest of the fields are family-dependent. See
// quote_family.go for the per-scene strings the chrome painters consume.
//
// Defaults: Family zero-value is FamilyMarginalia (the most neutral
// page-of-a-book look). HasAuthor == true keeps the author Text element.
// Tagline (when set) renders as a small caption; FamilyMarginalia puts
// it bottom-LEFT for asymmetry against the bottom-right attribution,
// FamilyFromSource centres it under the rule, FamilyTerminal omits it
// (the status-bar lines play that role).
type QuoteSceneOpts struct {
	Name         string
	Title        string
	Weight       int
	BgPath       string
	Widget       widget.Widget
	Family       QuoteFamily
	Tagline      string
	TaglineColor string
	HasAuthor    bool
}

// QuoteScene returns the *scene.Scene for a promoted-quote layout. The
// element shape varies by Family:
//   - FromSource: body LEFT-aligned, attribution Text element below the
//     baked horizontal rule.
//   - Marginalia: body LEFT-aligned (with left margin pulled in to clear
//     the baked drop cap), attribution right-aligned at the bottom,
//     optional tagline at bottom-LEFT.
//   - Terminal: body LEFT-aligned beneath the baked shell prompt; no
//     attribution / tagline Text elements — the baked status-bar lines
//     carry that information.
func QuoteScene(opts QuoteSceneOpts) *scene.Scene {
	switch opts.Family {
	case FamilyFromSource:
		return quoteSceneFromSource(opts)
	case FamilyTerminal:
		return quoteSceneTerminal(opts)
	default:
		return quoteSceneMarginalia(opts)
	}
}

// quoteSceneFromSource builds an in-universe-document layout: body left-
// aligned in the open region between the two baked rules, attribution
// (when present) just below the bottom rule, optional centred tagline
// under that.
func quoteSceneFromSource(opts QuoteSceneOpts) *scene.Scene {
	elements := []frame.DispElement{
		quoteBodyLeft(idSceneSub1, 80, 640, 580, 540),
	}
	mounts := []scene.Mount{
		{ID: idSceneSub1, Format: pipeAt(1), Geometry: vCenterQuoteBodyFromSource},
	}
	if opts.HasAuthor {
		elements = append(elements, frame.DispElement{
			ID: idSceneSub2, Type: "Text",
			StartX: 80, StartY: 1140, Width: 640, Height: 40,
			Align: 0, FontSize: 26, FontID: fontProse,
			FontColor: cFgDark, BgColor: cBgHard,
		})
		mounts = append(mounts, scene.Mount{
			ID: idSceneSub2, Format: pipeAtUpper(2), AllowEmpty: true,
		})
	}
	if opts.Tagline != "" {
		elements = append(elements, quoteTagline(idSceneSub3, opts.Tagline, opts.TaglineColor, 2, 1190))
	}
	return &scene.Scene{
		Name: opts.Name, Weight: opts.Weight, BgPath: opts.BgPath,
		Elements: elements, Widget: opts.Widget, Mounts: mounts,
	}
}

// quoteSceneMarginalia builds a page-of-a-book layout: body left-aligned
// with a left margin (120px) leaving room for the baked drop cap,
// attribution all-caps right-aligned at the bottom, optional tagline at
// the bottom-LEFT to balance.
func quoteSceneMarginalia(opts QuoteSceneOpts) *scene.Scene {
	elements := []frame.DispElement{
		quoteBodyLeft(idSceneSub1, 120, 600, 560, 540),
	}
	mounts := []scene.Mount{
		{ID: idSceneSub1, Format: pipeAt(1), Geometry: vCenterQuoteBodyMarginalia},
	}
	if opts.HasAuthor {
		elements = append(elements, frame.DispElement{
			ID: idSceneSub2, Type: "Text",
			StartX: 80, StartY: 1130, Width: 640, Height: 40,
			Align: 1, FontSize: 26, FontID: fontProse,
			FontColor: cFgDark, BgColor: cBgHard,
		})
		mounts = append(mounts, scene.Mount{
			ID: idSceneSub2, Format: pipeAtUpper(2), AllowEmpty: true,
		})
	}
	if opts.Tagline != "" {
		// Bottom-LEFT for asymmetry against the right-aligned attribution.
		elements = append(elements, quoteTagline(idSceneSub3, opts.Tagline, opts.TaglineColor, 0, 1190))
	}
	return &scene.Scene{
		Name: opts.Name, Weight: opts.Weight, BgPath: opts.BgPath,
		Elements: elements, Widget: opts.Widget, Mounts: mounts,
	}
}

// quoteSceneTerminal builds a shell-session layout: body left-aligned
// beneath the baked prompt. No author / tagline Text elements — the
// baked status-bar lines at the bottom of the bg carry "source:" and
// "author:" instead.
func quoteSceneTerminal(opts QuoteSceneOpts) *scene.Scene {
	elements := []frame.DispElement{
		quoteBodyLeft(idSceneSub1, 80, 640, 580, 520),
	}
	mounts := []scene.Mount{
		{ID: idSceneSub1, Format: pipeAt(1), Geometry: vCenterQuoteBodyTerminal},
	}
	return &scene.Scene{
		Name: opts.Name, Weight: opts.Weight, BgPath: opts.BgPath,
		Elements: elements, Widget: opts.Widget, Mounts: mounts,
	}
}

// quoteBodyLeft is the standard left-aligned body element for all three
// quote families — only the StartX / Width / StartY / Height vary.
func quoteBodyLeft(id, startX, width, startY, height int) frame.DispElement {
	return frame.DispElement{
		ID: id, Type: "Text",
		StartX: startX, StartY: startY, Width: width, Height: height,
		Align: 0, FontSize: 34, FontID: fontProseLight,
		FontColor: cFg, BgColor: cBgHard,
	}
}

// quoteTagline is a small static caption. align: 0=left, 1=right,
// 2=middle. startY pins the baseline.
func quoteTagline(id int, text, color string, align, startY int) frame.DispElement {
	return frame.DispElement{
		ID: id, Type: "Text",
		StartX: 80, StartY: startY, Width: 640, Height: 40,
		Align: align, FontSize: 22, FontID: fontProseLight,
		FontColor: color, BgColor: cBgHard,
		TextMessage: text,
	}
}

// pipeAtUpper is pipeAt(i) wrapped in strings.ToUpper — used for the
// attribution row of FromSource and Marginalia families, which the
// design crit asked for in all-caps so it reads as a typographic mark
// rather than a name in normal-case running text.
func pipeAtUpper(i int) func(raw string) (text, color string) {
	return func(raw string) (text, color string) {
		t, c := pipeAt(i)(raw)
		return strings.ToUpper(t), c
	}
}

// vCenterQuoteBodyFromSource vertically centres the body inside its
// from-source track (between the top baked rule at y=535 and the bottom
// baked rule at y=1125). Mirror of vCenterQuoteBody but with track
// bounds matching the FromSource chrome.
func vCenterQuoteBodyFromSource(text string, e frame.DispElement) frame.DispElement {
	return vCenterInTrack(text, e, 540, 1120, 30)
}

// vCenterQuoteBodyMarginalia: track between the top imprint rule at
// y=525 and the attribution row at y=1130. Slightly narrower text track
// (120..720) — char-per-line estimate folds that in.
func vCenterQuoteBodyMarginalia(text string, e frame.DispElement) frame.DispElement {
	return vCenterInTrack(text, e, 560, 1100, 28)
}

// vCenterQuoteBodyTerminal: track between the top baked rule at y=535
// and the status-bar top rule at y=1140.
func vCenterQuoteBodyTerminal(text string, e frame.DispElement) frame.DispElement {
	return vCenterInTrack(text, e, 555, 1135, 30)
}

// vCenterInTrack centres a short body in the rectangle (trackTop..trackBot),
// anchored to trackTop when the text would otherwise overflow. charsPerLine
// is the empirical wrap estimate for the body's font + width (FontSize 34,
// fontProseLight, the widths the three family builders use).
func vCenterInTrack(text string, e frame.DispElement, trackTop, trackBot, charsPerLine int) frame.DispElement {
	const lineHeight = 45
	trackH := trackBot - trackTop
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

// --- dictionary scene helper ---
//
// DictionaryScene builds a scene that renders a dictionary-shaped entry
// (Devil's Dictionary, Jargon File) as four distinct typed regions:
// source label, big mono headword, medium aqua part-of-speech, body
// definition. Optionally adds an author block (Devil's carries
// "Ambrose Bierce") and a static tagline. Shares the 10%-margin
// (StartX 80, Width 640) convention with QuoteScene.

// DictionarySceneOpts describes a dictionary-shaped scene. Dictionary
// scenes are always FamilyTerminal — the baked shell-prompt + status-bar
// chrome carries the source label and (when present) author, so the only
// device Text elements are the headword (Iosevka, scene-accent colour),
// POS (small aqua, on the same line as the headword), and definition
// (running prose).
//
// Colours are intentionally NOT options: every dictionary scene uses
// the same palette (yellow headword, aqua POS, fg definition) so they
// read as a consistent typographic family even when the source material
// differs.
type DictionarySceneOpts struct {
	Name   string
	Title  string // unused under the redesign; kept for call-site continuity
	Weight int
	BgPath string
	Widget widget.Widget
}

func DictionaryScene(opts DictionarySceneOpts) *scene.Scene {
	// Compute where the baked shell-prompt prefix ends so the headword
	// sits flush right of it on the same baseline. The fallback (80) is
	// the canvas left margin and matches the chrome-failure render path.
	headwordX := 80
	chrome := quoteFamilyChromeByName(opts.Name, time.Now())
	if chrome.ShellPrompt != "" {
		// 12 px gap between the baked prompt and the headword. Iosevka
		// 28pt is the baked face; MeasureLabel uses the same DPI/hinting
		// pair as the chrome painter, so the two stay aligned.
		if w, err := render.MeasureLabel(chrome.ShellPrompt+" ", "Iosevka-Regular.ttf", 28); err == nil {
			headwordX = 80 + w
		}
	}

	elements := []frame.DispElement{
		// Headword — Iosevka (mono), left-aligned, sitting just to the
		// right of the baked "$ <cmd>" prompt. Smaller than the previous
		// 90pt to fit on one line beside the prompt; fits the terminal-
		// session aesthetic better than the giant heading anyway.
		{
			ID: idSceneSub1, Type: "Text",
			StartX: headwordX, StartY: 490, Width: CanvasW - 80 - headwordX, Height: 50,
			Align: 0, FontSize: 36, FontID: fontMono,
			FontColor: cYellow, BgColor: cBgHard,
		},
		// Part-of-speech tag — small aqua, dropped one row below the
		// headword so the headword line stays uncluttered.
		{
			ID: idSceneSub2, Type: "Text",
			StartX: 80, StartY: 560, Width: 640, Height: 40,
			Align: 0, FontSize: 26, FontID: fontProseLight,
			FontColor: cAqua, BgColor: cBgHard,
		},
		// Definition — body prose, left-aligned, vertically centred
		// inside the body track. fitDictionaryBody auto-shrinks the font
		// for long entries.
		{
			ID: idSceneSub3, Type: "Text",
			StartX: 80, StartY: 620, Width: 640, Height: 510,
			Align: 0, FontSize: 34, FontID: fontProseLight,
			FontColor: cFg, BgColor: cBgHard,
		},
	}
	mounts := []scene.Mount{
		{ID: idSceneSub1, Format: dictionaryWord, Geometry: shrinkHeadwordTerminal},
		{ID: idSceneSub2, Format: dictionaryPOS, AllowEmpty: true},
		{ID: idSceneSub3, Format: dictionaryDefinition, Geometry: fitDictionaryBodyTerminal},
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

// shrinkHeadwordTerminal: same idea as shrinkHeadword but scaled for the
// smaller Iosevka headword used in the terminal layout. Mono characters
// are wider, so the per-char ratio is higher.
func shrinkHeadwordTerminal(text string, e frame.DispElement) frame.DispElement {
	const (
		maxFontSize    = 36
		minFontSize    = 20
		charWidthRatio = 0.55 // empirical for Iosevka regular
	)
	if text == "" {
		return e
	}
	estimated := int(float64(len(text)) * float64(maxFontSize) * charWidthRatio)
	if estimated <= e.Width {
		e.FontSize = maxFontSize
		return e
	}
	shrunk := int(float64(e.Width) / (float64(len(text)) * charWidthRatio))
	if shrunk < minFontSize {
		shrunk = minFontSize
	}
	e.FontSize = shrunk
	return e
}

// fitDictionaryBodyTerminal mirrors fitDictionaryBody but for the
// terminal-family geometry (taller track between y=620 and y=1130 since
// the chrome handles the source/author rows). Auto-shrinks the FontSize
// when long entries would overflow, then vertically centres.
func fitDictionaryBodyTerminal(text string, e frame.DispElement) frame.DispElement {
	const (
		maxFontSize    = 34
		minFontSize    = 22
		trackTop       = 620
		trackBottom    = 1130
		charWidthRatio = 0.45
		lineHeightFrac = 1.30
	)
	const trackH = trackBottom - trackTop
	if text == "" {
		e.StartY = trackTop
		e.Height = trackH
		return e
	}
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

// --- HN formatters ---
//
// Widget output is 8 pipe segments: "Hacker News|title|domain|summary|
// score|author|age|comments". hnFooter composes the bottom-row metadata
// from segments 4-7 (score, author, age, comments) into a single mono
// line. Robust to partial data — missing pieces are dropped rather
// than rendered as literal punctuation gaps.

// hnFooter composes the metadata footer "▲ <score>  by <author>  ·
// <age>  ·  <comments> comments" from the widget's 4..7 segments,
// omitting any piece that's empty so partial data doesn't produce
// "▲   by   ·   ·   comments". Returns ("", "") on a malformed raw
// string; the scene's AllowEmpty mount keeps the element blank in
// that case.
func hnFooter(raw string) (text, color string) {
	parts := strings.Split(raw, "|")
	if len(parts) < 8 {
		return "", ""
	}
	score := strings.TrimSpace(parts[4])
	author := strings.TrimSpace(parts[5])
	age := strings.TrimSpace(parts[6])
	comments := strings.TrimSpace(parts[7])

	byline := "by " + author
	if author == "" {
		byline = "by unknown"
	}
	var segs []string
	if score != "" {
		segs = append(segs, "▲ "+score+"  "+byline)
	} else {
		segs = append(segs, byline)
	}
	if age != "" {
		segs = append(segs, age)
	}
	if comments != "" && comments != "0" {
		segs = append(segs, comments+" comments")
	}
	return strings.Join(segs, "  ·  "), ""
}

// --- moon formatters ---

// moonPhaseAndIllum renders the combined "<Phase Name> · <illum>%" row
// shown under the disc (e.g. "First Quarter · 53%"). The widget emits
// the phase name in lowercase; title-case it so the row reads as a
// caption rather than continuation prose.
func moonPhaseAndIllum(s string) (text, color string) {
	parts := strings.Split(s, " · ")
	if len(parts) < 3 {
		return s, ""
	}
	return titleCasePhase(parts[1]) + " · " + parts[2], ""
}

// titleCasePhase upper-cases the first letter of each word in a moon
// phase name. Avoids cases.Title since the package can normalise in
// surprising ways; a one-liner is enough for the small input set
// ("new", "waxing crescent", "first quarter", etc.).
func titleCasePhase(s string) string {
	if s == "" {
		return s
	}
	words := strings.Fields(s)
	for i, w := range words {
		if w == "" {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
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

// --- ISS formatters and coordinate math ---

// issMapX maps a longitude (-180..+180) to a baked-map x-coordinate
// (render.ISSMapX0..render.ISSMapX0+render.ISSMapW). Out-of-range
// inputs are wrapped/clamped to the map's edges so a bad parse can't
// place the dot outside the map rect.
func issMapX(lon float64) int {
	if lon < -180 {
		lon = -180
	} else if lon > 180 {
		lon = 180
	}
	return render.ISSMapX0 + int((lon+180)*float64(render.ISSMapW)/360.0)
}

// issMapY maps a latitude (+90..-90) to a baked-map y-coordinate
// (render.ISSMapY0..render.ISSMapY0+render.ISSMapH). +90 = top,
// -90 = bottom (the equirectangular convention).
func issMapY(lat float64) int {
	if lat > 90 {
		lat = 90
	} else if lat < -90 {
		lat = -90
	}
	return render.ISSMapY0 + int((90-lat)*float64(render.ISSMapH)/180.0)
}

// parseISSCoords parses the ISS widget's pipe[0] segment ("-22.5°,
// -45.3°") into (lat, lon, ok). Tolerates leading/trailing whitespace
// and the degree-sign suffix; returns ok=false on any parse failure
// so the caller can hide the dot rather than render it at (0,0).
func parseISSCoords(s string) (lat, lon float64, ok bool) {
	parts := strings.SplitN(strings.TrimSpace(s), ",", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	clean := func(p string) string {
		return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(p), "°"))
	}
	a, err := strconv.ParseFloat(clean(parts[0]), 64)
	if err != nil {
		return 0, 0, false
	}
	b, err := strconv.ParseFloat(clean(parts[1]), 64)
	if err != nil {
		return 0, 0, false
	}
	return a, b, true
}

// formatISSCoordsNSEW reformats the widget's "-22.5°, -45.3°" into a
// human-friendly "22.5° S   45.3° W" form. Negative lat → S, positive
// → N; negative lon → W, positive → E. On parse failure the original
// string is returned unchanged so the body element still has something
// to show.
func formatISSCoordsNSEW(s string) string {
	lat, lon, ok := parseISSCoords(s)
	if !ok {
		return s
	}
	latHem := "N"
	if lat < 0 {
		latHem = "S"
		lat = -lat
	}
	lonHem := "E"
	if lon < 0 {
		lonHem = "W"
		lon = -lon
	}
	return fmt.Sprintf("%.1f° %s   %.1f° %s", lat, latHem, lon, lonHem)
}

// issCoordsAndPass is the scene mount formatter for the ISS scene's
// coords-and-pass row. It joins the reformatted coordinates from
// pipe[0] with the next-pass string from pipe[1], rewriting the
// widget's "next pass in 1h 04m" prefix to the more compact
// "next pass · 1h 04m" form the scene uses. Missing pass text is
// dropped silently so a flaky upstream just shows the coordinates.
func issCoordsAndPass(raw string) (text, color string) {
	coords := formatISSCoordsNSEW(pipeAtRaw(raw, 0))
	pass := pipeAtRaw(raw, 1)
	pass = strings.TrimPrefix(pass, "next pass in ")
	if pass != "" {
		pass = "next pass · " + pass
	}
	if pass == "" {
		return coords, ""
	}
	return coords + "       " + pass, ""
}

// pipeAtRaw returns the i-th pipe-separated segment of raw, or "" if
// the segment is missing. Plain helper for formatters that compose
// from multiple segments without going through the pipeAt() mount
// indirection.
func pipeAtRaw(raw string, i int) string {
	parts := strings.Split(raw, "|")
	if i < 0 || i >= len(parts) {
		return ""
	}
	return parts[i]
}
