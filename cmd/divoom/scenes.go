package main

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
	"sync"
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
	idDay     = 1
	idTime    = 2
	idFooter  = 3
	idWeekend = 4

	idSceneTitle = 9
	idSceneMain  = 10
	idSceneSub1  = 11
	idSceneSub2  = 12
	idSceneSub3  = 13
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
	// NASA APOD: bgs live at /userdata/wallclock_bg_nasa_NNN.jpg, one
	// per curated date in nasaCuratedDates. The scene's BgPathFor
	// picks an index per activation (see scene_nasa.go).
	// Cocktail: per-drink bgs live at /userdata/wallclock_bg_cocktail_NNN.jpg,
	// one per ID returned by bakeAllCocktailBackgrounds. See bgCocktailFor
	// and cocktailPoolSize.
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

// weekendStatus returns the operator-footer right-hand string and its
// FontColor. Weekend window is Friday 18:00 through Monday 03:00 (local
// time) — inside it the text becomes "weekend!" in cYellow as a small
// festive marker. Outside it, the row reverts to the dim countdown
// "weekend+Nd" (Mon-Thu after 3am: 4..1 days; Fri before 6pm: +0d).
func weekendStatus(now time.Time) (text, color string) {
	wd := now.Weekday()
	hour := now.Hour()
	weekend := false
	switch wd {
	case time.Saturday, time.Sunday:
		weekend = true
	case time.Friday:
		weekend = hour >= 18
	case time.Monday:
		weekend = hour < 3
	}
	if weekend {
		return "weekend!", cYellow
	}
	// Outside the window — countdown to Saturday morning.
	n := 5 - int(wd)
	if n < 0 {
		n = 0 // defensive; Friday-pre-6pm falls here as +0d
	}
	return fmt.Sprintf("weekend+%dd", n), cFgDark
}

func alwaysOn(now time.Time) []frame.DispElement {
	weekendText, weekendColor := weekendStatus(now)
	return []frame.DispElement{
		{
			// Week is a device built-in (renders the day name from
			// the device's own clock). Doesn't count against the
			// 6-Text cap. The "> " prompt to its left is baked into
			// every scene bg by buildHeroImage; this element only
			// owns the day name itself. StartX shifted right of the
			// baked prompt; FontColor still picks up the per-day
			// chroma so each weekday has its own colour.
			ID: idDay, Type: "Week",
			StartX: 110, StartY: 30, Width: 650, Height: 80,
			Align:     0,
			FontSize:  64,
			FontID:    fontMono,
			FontColor: dayColors[now.Weekday()],
			BgColor:   cBgHard,
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
		// Left half of the footer row — date / day-of-year / iso-week,
		// dim mono left-aligned.
		{
			ID: idFooter, Type: "Text",
			StartX: 40, StartY: 400, Width: 720, Height: 44,
			Align:     0,
			FontSize:  28,
			FontID:    fontMono,
			FontColor: cFgDark,
			BgColor:   cBgHard,
			TextMessage: fmt.Sprintf("%s  doy:%d  w:%d",
				now.Format("2006-01-02"),
				now.YearDay(),
				isoWeek(now)),
		},
		// Right half of the footer row — weekend status, right-aligned
		// so it can carry its own colour (cYellow during the Fri 6pm →
		// Mon 3am window, cFgDark otherwise) without recolouring the
		// numeric metadata to its left.
		{
			ID: idWeekend, Type: "Text",
			StartX: 40, StartY: 400, Width: 720, Height: 44,
			Align:       1,
			FontSize:    28,
			FontID:      fontMono,
			FontColor:   weekendColor,
			BgColor:     cBgHard,
			TextMessage: weekendText,
		},
	}
}

// bgNASAFor returns the on-device path for the NASA APOD bg at index i
// (0..len(nasaCuratedDates)-1). Each path holds the baked APOD for one
// curated date; the scene's BgPathFor picks a random index per
// activation so the wall display rotates through the curated set.
func bgNASAFor(i int) string {
	return fmt.Sprintf("/userdata/wallclock_bg_nasa_%03d.jpg", i)
}

// bgCocktailFor mirrors bgNASAFor for the cocktail rotation pool.
// The bake pushes one JPG per drink from the Cocktail + Shot
// categories; scene_cocktail.go's BgPathFor picks an index per
// activation. Pool size is dynamic — see cocktailPoolSize().
func bgCocktailFor(i int) string {
	return fmt.Sprintf("/userdata/wallclock_bg_cocktail_%03d.jpg", i)
}

// newIndexWalker returns a function that yields 0..n-1 in a freshly
// shuffled order, then reshuffles once the order is exhausted and
// starts over. Returns 0 forever when n < 1.
//
// Shared by the NASA and cocktail scenes: each scene needs to
// rotate through every cached bg before repeating any (so a given
// drink/photo always sits at the same indexed path → the per-item
// disk cache stays valid across pushes) while still producing a
// fresh ordering on every daemon restart.
func newIndexWalker(n int) func() int {
	if n < 1 {
		return func() int { return 0 }
	}
	var mu sync.Mutex
	order := rand.Perm(n)
	next := 0
	return func() int {
		mu.Lock()
		defer mu.Unlock()
		if next >= len(order) {
			order = rand.Perm(n)
			next = 0
		}
		i := order[next]
		next++
		return i
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
// Widget output: "<lifetime_contributions>|<total_prs>|<open_prs>|<years_on_github>",
// e.g. "14238|287|4|11". The hero row gets the lifetime contributions
// with thousands separators; the three small stat rows render total
// PRs, open PRs, and years. Open PRs colour-shifts: cAqua when >0
// (you have outstanding work) so the live reading stands apart from
// the slower-changing lifetime totals.

func githubLifetime(raw string) (text, color string) {
	parts := strings.Split(raw, "|")
	if len(parts) < 1 {
		return "0", cFgDark
	}
	n, _ := strconv.Atoi(parts[0])
	if n > 0 {
		return withThousands(n), cGreen
	}
	return "0", cFgDark
}

func githubTotalPRs(raw string) (text, color string) {
	parts := strings.Split(raw, "|")
	if len(parts) < 2 {
		return "0", cFg
	}
	n, _ := strconv.Atoi(parts[1])
	return withThousands(n), cFg
}

func githubOpenPRs(raw string) (text, color string) {
	parts := strings.Split(raw, "|")
	if len(parts) < 3 {
		return "0", cFgDark
	}
	n, _ := strconv.Atoi(parts[2])
	c := cFgDark
	if n > 0 {
		c = cAqua
	}
	return strconv.Itoa(n), c
}

func githubYears(raw string) (text, color string) {
	parts := strings.Split(raw, "|")
	if len(parts) < 4 {
		return "0", cFg
	}
	return parts[3], cFg
}

// withThousands inserts comma separators every three digits for the
// large GitHub stats so "14238" reads as "14,238" at glance distance.
func withThousands(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	rem := len(s) % 3
	if rem > 0 {
		b.WriteString(s[:rem])
		if len(s) > rem {
			b.WriteByte(',')
		}
	}
	for i := rem; i < len(s); i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < len(s) {
			b.WriteByte(',')
		}
	}
	return b.String()
}

// --- markets formatters ---
//
// Widget output: "<SYM>|<price>|<week_pct>|<month_pct>|<sparkline>|<close_date>".
// Six pipe-separated fields. The ticker symbol and price ride two
// separate Text elements (left/right of one row) via pipeAt(0) and
// pipeAt(1). The week + month badges live on a single combined row
// rendered by marketsChangeBoth. The sparkline is surfaced as-is via
// pipeAt(4). marketsColorize sets the combined-badge colour at
// activation time based on the sign of the week percent.

// marketsChangeBoth renders the week + month percent badges on one
// mono-padded row: "▲ +1.2 %   ▼ -3.7 %" / "· 0 %   ▼ -0.1 %" etc. The
// two halves are joined by three spaces so the row reads as two
// visually-separated badges within the device's 640px text track at
// FontSize 60. Either half collapses to empty (and the gap closes) when
// its widget segment is missing — defensive against pre-fetch state.
func marketsChangeBoth(raw string) (text, color string) {
	week := marketsChangeBadge(pipeAtRaw(raw, 2))
	month := marketsChangeBadge(pipeAtRaw(raw, 3))
	switch {
	case week == "" && month == "":
		return "", ""
	case week == "":
		return month, ""
	case month == "":
		return week, ""
	}
	return week + "   " + month, ""
}

// marketsChangeBadge renders a single signed-percent value as a badge —
// "▲ +1.2 %" for positive, "▼ -3.7 %" for negative, "· 0 %" for zero or
// unparseable. Returns "" when v is empty so the caller can collapse a
// missing half.
func marketsChangeBadge(v string) string {
	if v == "" {
		return ""
	}
	f, ok := parseSignedFloat(v)
	arrow := "·"
	switch {
	case ok && f > 0:
		arrow = "▲"
	case ok && f < 0:
		arrow = "▼"
	}
	return arrow + " " + v + " %"
}

// parseSignedFloat parses a "+1.2" / "-3.7" / "0" string. Tolerant of a
// leading "+" (strconv.ParseFloat is happy with that on modern Go but the
// helper keeps the call sites simple).
func parseSignedFloat(s string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.TrimPrefix(s, "+"), 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// signColor returns the gruvbox accent for a directional value:
// green positive, red negative, dim neutral.
func signColor(v float64) string {
	switch {
	case v > 0:
		return cGreen
	case v < 0:
		return cRed
	default:
		return cFgDark
	}
}

// marketsColorize sets the combined week+month badge FontColor from the
// sign of the WEEK percent — week is the primary signal (more recent
// than month, less noisy than day-of). The month sign reads from the
// glyph (▲/▼) inside the badge text. Runs as the markets scene's
// OnActivate, after the Mounts have set TextMessage but before the
// layout ships to the device.
func marketsColorize(_ time.Time, raw string, elements []frame.DispElement) {
	week, weekOK := parseSignedFloat(pipeAtRaw(raw, 2))
	if !weekOK {
		return
	}
	for i := range elements {
		// IDs are offset per-install (see Driver.activate); match by the
		// low-order ID since OnActivate runs before the offset is added.
		if elements[i].ID == idSceneSub2 {
			elements[i].FontColor = signColor(week)
		}
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

// tilBody extracts and normalizes the fact body from the TIL widget's
// "TIL|<title>" output. Strips any leading "TIL that " / "TIL: " / "TIL "
// prefix (case-insensitive) so the body flows out of the monumental
// "T I L" wordmark baked above it. Ensures the result starts with "that "
// so the visual sentence completes as "TIL · that <fact>".
func tilBody(raw string) (text, color string) {
	body, _ := pipeAt(1)(raw)
	body = strings.TrimSpace(body)
	lower := strings.ToLower(body)
	for _, prefix := range []string{"til that ", "til: ", "til, ", "til "} {
		if strings.HasPrefix(lower, prefix) {
			body = strings.TrimSpace(body[len(prefix):])
			break
		}
	}
	if !strings.HasPrefix(strings.ToLower(body), "that ") {
		body = "that " + body
	}
	return body, ""
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

// weatherPipeField pulls segment i of a pipe-separated raw string,
// returning "" when the segment is missing.
func weatherPipeField(raw string, i int) string {
	parts := strings.Split(raw, "|")
	if i >= len(parts) {
		return ""
	}
	return parts[i]
}

// weatherStrip renders the bottom console strip. When an NWS alert
// fires (pipe[2] non-empty) the strip becomes a red "⚠ <hazard>"
// warning that takes over the whole row. Otherwise it shows the
// outlook word + the three stats joined by middots:
// "FOG · AIR 50 · HUM 96% · RAIN 2%". Missing stat segments render
// as "—" so a failed upstream lookup doesn't lie about clean air.
// The colour is bound to AQI band when present so the strip doubles
// as the air-quality alert lamp (red at AQI>200, etc.).
func weatherStrip(raw string) (text, color string) {
	hazard := weatherPipeField(raw, 2)
	if hazard != "" {
		return "⚠ " + hazard, cRed
	}
	outlook := strings.ToUpper(weatherOutlookFrom(raw))
	aqi := weatherPipeField(raw, 3)
	hum := weatherPipeField(raw, 4)
	rain := weatherPipeField(raw, 5)

	dash := func(v, suffix string) string {
		if v == "" {
			return "—"
		}
		return v + suffix
	}
	parts := []string{outlook,
		"AIR " + dash(aqi, ""),
		"HUM " + dash(hum, "%"),
		"RAIN " + dash(rain, "%"),
	}
	c := cFg
	if n, err := strconv.Atoi(aqi); err == nil {
		c = aqiColor(n)
	}
	return strings.Join(parts, " · "), c
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
		{ID: idSceneSub1, Format: pipeAt(1), Geometry: vTopQuoteBodyFromSource},
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
// between the baked imprint rules, attribution all-caps right-aligned
// at the bottom, optional tagline at the bottom-LEFT to balance. The
// dynamic drop-cap was removed (read poorly at 90pt on the dim track —
// see git history); the body now uses the full left margin and the
// track lifts upward by ≈80px to fill the void the cap had occupied.
func quoteSceneMarginalia(opts QuoteSceneOpts) *scene.Scene {
	elements := []frame.DispElement{
		quoteBodyLeft(idSceneSub1, 80, 640, 480, 620),
	}
	mounts := []scene.Mount{
		{ID: idSceneSub1, Format: pipeAt(1), Geometry: vTopQuoteBodyMarginalia},
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
		{ID: idSceneSub1, Format: pipeAt(1), Geometry: vTopQuoteBodyTerminal},
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

// vTopQuoteBodyFromSource pins the body to the top of the from-source
// track (between the top baked rule at y=535 and the bottom baked
// rule at y=1125). Top-anchored rather than centred so that short
// quotes don't drift up and away from the source-label rule above.
func vTopQuoteBodyFromSource(_ string, e frame.DispElement) frame.DispElement {
	return vTopInTrack(e, 540, 1120)
}

// vTopQuoteBodyMarginalia pins the body to the top of the marginalia
// track. The drop-cap used to live at the top of this track; with it
// removed, trackTop sits at 480 so the body still starts close to
// the imprint rule above without floating in dead space.
func vTopQuoteBodyMarginalia(_ string, e frame.DispElement) frame.DispElement {
	return vTopInTrack(e, 480, 1100)
}

// vTopQuoteBodyTerminal pins the body to the top of the terminal
// track (between the top baked rule at y=535 and the status-bar top
// rule at y=1140). Matches the "command output starts at the prompt
// row, not floating in the middle of the screen" gestalt.
func vTopQuoteBodyTerminal(_ string, e frame.DispElement) frame.DispElement {
	return vTopInTrack(e, 555, 1135)
}

// vTopInTrack anchors a body element at trackTop with full track
// height. Top-aligned: short quotes still start at the top of the
// track and run downward, never centred or pushed away from the
// source-label / shell-prompt that introduces them.
func vTopInTrack(e frame.DispElement, trackTop, trackBot int) frame.DispElement {
	e.StartY = trackTop
	e.Height = trackBot - trackTop
	return e
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
// DictionaryStyle picks one of three per-source body layouts that share
// the FamilyTerminal chrome (shell prompt + status-bar footer). The
// styles tune typography and proportions for the different shapes of
// the underlying corpora:
//   - StyleManpage   — jargon: header line, body, see-also footer.
//   - StylePunchline — devil's: big centred aphorism in pull-quote ornaments.
//   - StyleCeremony  — wordnik: monumental centred headword, tiny POS, breathy body.
type DictionaryStyle int

const (
	// StyleManpage is the zero value — jargon's three-row layout (header,
	// body, see-also). Default for scenes that don't set Style explicitly.
	StyleManpage DictionaryStyle = iota
	StylePunchline
	StyleCeremony
)

type DictionarySceneOpts struct {
	Name   string
	Title  string // unused under the redesign; kept for call-site continuity
	Weight int
	BgPath string
	Widget widget.Widget
	Style  DictionaryStyle
}

// DictionaryScene dispatches on opts.Style to one of three per-source
// layouts. All share the FamilyTerminal chrome baked into the bg JPG.
func DictionaryScene(opts DictionarySceneOpts) *scene.Scene {
	switch opts.Style {
	case StylePunchline:
		return dictionarySceneDevil(opts)
	case StyleCeremony:
		return dictionarySceneWordnik(opts)
	default:
		return dictionarySceneJargon(opts)
	}
}

// dictionaryHeadwordX computes where the headword should start on the
// shell-prompt line — flush right of the baked "$ <cmd> " prompt. The
// fallback (80) is the canvas left margin and matches the chrome-failure
// render path. Shared by the StyleManpage and StylePunchline layouts;
// StyleCeremony centres the headword instead and doesn't call this.
func dictionaryHeadwordX(name string) int {
	chrome := quoteFamilyChromeByName(name, time.Now())
	if chrome.ShellPrompt == "" {
		return 80
	}
	// Iosevka 28pt is the baked face used by drawTerminalChrome; the
	// trailing space gives a ~12px visual gap between prompt and headword.
	w, err := render.MeasureLabel(chrome.ShellPrompt+" ", "Iosevka-Regular.ttf", 28)
	if err != nil {
		return 80
	}
	return 80 + w
}

// dictionarySceneJargon builds the StyleManpage layout: a one-line
// header (headword + pronunciation + POS) flush right of the baked
// shell prompt, the body in the middle, and a small "see also" footer.
func dictionarySceneJargon(opts DictionarySceneOpts) *scene.Scene {
	headwordX := dictionaryHeadwordX(opts.Name)
	elements := []frame.DispElement{
		{
			ID: idSceneSub1, Type: "Text",
			StartX: headwordX, StartY: 490, Width: CanvasW - 80 - headwordX, Height: 50,
			Align: 0, FontSize: 36, FontID: fontMono,
			FontColor: cYellow, BgColor: cBgHard,
		},
		{
			ID: idSceneSub2, Type: "Text",
			StartX: 80, StartY: 600, Width: 640, Height: 480,
			Align: 0, FontSize: 36, FontID: fontProse,
			FontColor: cFg, BgColor: cBgHard,
		},
		{
			ID: idSceneSub3, Type: "Text",
			StartX: 80, StartY: 1100, Width: 640, Height: 36,
			Align: 0, FontSize: 24, FontID: fontProseLight,
			FontColor: cFgDark, BgColor: cBgHard,
		},
	}
	mounts := []scene.Mount{
		{ID: idSceneSub1, Format: jargonHeader},
		{ID: idSceneSub2, Format: dictionaryDefinition, Geometry: fitDictionaryBody},
		{ID: idSceneSub3, Format: jargonSeeAlso, AllowEmpty: true},
	}
	return &scene.Scene{
		Name: opts.Name, Weight: opts.Weight, BgPath: opts.BgPath,
		Elements: elements, Widget: opts.Widget, Mounts: mounts,
	}
}

// dictionarySceneDevil builds the StylePunchline layout: one-line header
// (HEADWORD, POS) flush right of the baked prompt, then a giant centred
// aphorism body filling the middle. The two GIANT curly-quote ornaments
// are baked into the bg JPG by drawPunchlineOrnaments (see quote_family.go).
func dictionarySceneDevil(opts DictionarySceneOpts) *scene.Scene {
	headwordX := dictionaryHeadwordX(opts.Name)
	elements := []frame.DispElement{
		{
			ID: idSceneSub1, Type: "Text",
			StartX: headwordX, StartY: 490, Width: CanvasW - 80 - headwordX, Height: 60,
			Align: 0, FontSize: 44, FontID: fontProseLight,
			FontColor: cYellow, BgColor: cBgHard,
		},
		{
			ID: idSceneSub2, Type: "Text",
			StartX: 160, StartY: 780, Width: 520, Height: 280,
			Align: 2, FontSize: 60, FontID: fontProse,
			FontColor: cFg, BgColor: cBgHard,
		},
	}
	mounts := []scene.Mount{
		{ID: idSceneSub1, Format: devilHeader},
		{ID: idSceneSub2, Format: dictionaryDefinition, Geometry: fitDevilBody},
	}
	return &scene.Scene{
		Name: opts.Name, Weight: opts.Weight, BgPath: opts.BgPath,
		Elements: elements, Widget: opts.Widget, Mounts: mounts,
	}
}

// dictionarySceneWordnik builds the StyleCeremony layout: a monumental
// letter-spaced headword centred at the top of the body area, a tiny
// POS+pronunciation row beneath it, and the body's prose centred below.
// The shell prompt (with today's date) is baked into the bg chrome but
// the headword is centre-aligned in its own track and doesn't sit
// alongside it — the negative space IS the design.
func dictionarySceneWordnik(opts DictionarySceneOpts) *scene.Scene {
	elements := []frame.DispElement{
		{
			ID: idSceneSub1, Type: "Text",
			StartX: 40, StartY: 620, Width: 720, Height: 160,
			Align: 2, FontSize: 110, FontID: fontProseLight,
			FontColor: cYellow, BgColor: cBgHard,
		},
		{
			ID: idSceneSub2, Type: "Text",
			StartX: 40, StartY: 800, Width: 720, Height: 50,
			Align: 2, FontSize: 32, FontID: fontProseLight,
			FontColor: cFgDark, BgColor: cBgHard,
		},
		{
			ID: idSceneSub3, Type: "Text",
			StartX: 80, StartY: 940, Width: 640, Height: 200,
			Align: 2, FontSize: 44, FontID: fontProseLight,
			FontColor: cFg, BgColor: cBgHard,
		},
	}
	mounts := []scene.Mount{
		{ID: idSceneSub1, Format: wordnikHeadword, Geometry: fitWordnikHeadword},
		{ID: idSceneSub2, Format: wordnikPosPron, AllowEmpty: true},
		{ID: idSceneSub3, Format: dictionaryDefinition, Geometry: fitDictionaryBody},
	}
	return &scene.Scene{
		Name: opts.Name, Weight: opts.Weight, BgPath: opts.BgPath,
		Elements: elements, Widget: opts.Widget, Mounts: mounts,
	}
}

// --- per-style dictionary formatters ---

// dictionaryEntryWithPronRE captures (headword, pronunciation list, pos,
// definition) — mirror of dictionaryEntryRE but with the pronunciation
// segment exposed as its own group so jargonHeader can surface it.
var dictionaryEntryWithPronRE = regexp.MustCompile(
	`^(.+?),?\s+((?:/[^/]+/(?:,\s*/[^/]+/)*)?)\s*` +
		`((?:n|v|vi|vt|adj|adv|prep|conj|pp|interj|pron|num|art|excl|pl|i|t|imp|abbrev)` +
		`(?:\.?[.,](?:n|v|vi|vt|adj|adv|prep|conj|pp|interj|pron|num|art|excl|pl|i|t|imp|abbrev))*)` +
		`\.?\s+(.+)$`,
)

// jargonHeader builds the StyleManpage header line: headword + (optional)
// pronunciation + POS joined with tabs. Falls back to the existing
// dictionaryWord/POS shape when the entry doesn't carry a pronunciation.
func jargonHeader(raw string) (text, color string) {
	body := dictionaryBody(raw)
	if m := dictionaryEntryWithPronRE.FindStringSubmatch(body); m != nil {
		parts := []string{m[1]}
		if m[2] != "" {
			parts = append(parts, m[2])
		}
		parts = append(parts, m[3]+".")
		return strings.Join(parts, "  "), ""
	}
	w, _ := dictionaryWord(raw)
	p, _ := dictionaryPOS(raw)
	if p == "" {
		return w, ""
	}
	return w + "  " + p, ""
}

// jargonSeeAlsoRE matches the Jargon File's trailing cross-reference
// patterns ("See also X, Y.", "Compare X.", "Cf. X."). The capture
// group holds the bare reference list, comma-separated.
var jargonSeeAlsoRE = regexp.MustCompile(
	`(?i)\s*(?:see\s+also|compare|cf\.)\s*:?\s*([^.]+?)\s*\.?\s*$`,
)

// jargonSeeAlso extracts trailing "see also" / "compare" / "Cf." reference
// patterns from the entry body and emits "→ see also: <refs>". Empty
// string when no such pattern is present so the scene's AllowEmpty mount
// leaves the footer slot blank.
func jargonSeeAlso(raw string) (text, color string) {
	def, _ := dictionaryDefinition(raw)
	if m := jargonSeeAlsoRE.FindStringSubmatch(def); m != nil {
		refs := strings.TrimSpace(m[1])
		if refs == "" {
			return "", ""
		}
		return "→ see also: " + refs, ""
	}
	return "", ""
}

// devilHeader returns "HEADWORD, POS." on one line — the devil's
// dictionary scene's compact header above its monumental aphorism body.
func devilHeader(raw string) (text, color string) {
	w, _ := dictionaryWord(raw)
	p, _ := dictionaryPOS(raw)
	if p == "" {
		return w, ""
	}
	return w + ", " + p, ""
}

// wordnikHeadword returns the headword with thin-space letter-spacing
// for the StyleCeremony layout, e.g. "EPHEMERAL" → "E P H E M E R A L".
// Thin space (U+2009) is narrower than a regular space so the letters
// read as one word rather than dissociated columns.
func wordnikHeadword(raw string) (text, color string) {
	w, _ := dictionaryWord(raw)
	if w == "" {
		return "", ""
	}
	var b strings.Builder
	runes := []rune(w)
	for i, r := range runes {
		if i > 0 {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
	}
	return b.String(), ""
}

// wordnikPosPron combines POS + pronunciation into one line for
// StyleCeremony's small caption row. The widget's third pipe segment
// (when present) carries the IPA pronunciation; otherwise just POS.
func wordnikPosPron(raw string) (text, color string) {
	p, _ := dictionaryPOS(raw)
	pron := pipeAtRaw(raw, 3)
	switch {
	case p != "" && pron != "":
		return p + "    " + pron, ""
	case p != "":
		return p, ""
	case pron != "":
		return pron, ""
	default:
		return "", ""
	}
}

// fitDevilBody auto-shrinks the punchline FontSize for long aphorisms.
// Pattern: 60pt fits the ~95% of entries that are one or two sentences;
// 44pt picks up the longer ones; 32pt covers the rare paragraph-length
// entry so it doesn't clip. Vertically centres in the 280px track.
func fitDevilBody(text string, e frame.DispElement) frame.DispElement {
	const (
		trackTop       = 780
		trackBottom    = 1060
		charWidthRatio = 0.45 // px per char ≈ FontSize * this
		lineHeightFrac = 1.20
	)
	const trackH = trackBottom - trackTop
	if text == "" {
		e.StartY = trackTop
		e.Height = trackH
		return e
	}
	// Tier through 60 → 44 → 32 picking the largest that fits.
	tiers := []int{60, 44, 32}
	fs := tiers[len(tiers)-1]
	rendered := trackH
	for _, size := range tiers {
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

// fitWordnikHeadword auto-shrinks the monumental headword FontSize when
// a long letter-spaced word would overflow the 720px track. The thin
// spaces between letters make every word ~2x its bare letter count, so
// the budget shrinks fast for longer words.
func fitWordnikHeadword(text string, e frame.DispElement) frame.DispElement {
	const (
		maxFontSize    = 110
		minFontSize    = 56
		charWidthRatio = 0.45 // empirical for Roboto Condensed Light at this size
	)
	if text == "" {
		return e
	}
	runes := []rune(text)
	estimated := int(float64(len(runes)) * float64(maxFontSize) * charWidthRatio)
	if estimated <= e.Width {
		e.FontSize = maxFontSize
		return e
	}
	shrunk := int(float64(e.Width) / (float64(len(runes)) * charWidthRatio))
	if shrunk < minFontSize {
		shrunk = minFontSize
	}
	e.FontSize = shrunk
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

// issCoordsOnly is the scene mount formatter for the ISS scene's coords
// row. Surfaces just the reformatted "12.3° N   45.6° E" string; the
// next-pass field lives on its own row (see issNextPass).
func issCoordsOnly(raw string) (text, color string) {
	return formatISSCoordsNSEW(pipeAtRaw(raw, 0)), ""
}

// issNextPass is the scene mount formatter for the ISS scene's next-pass
// row. Rewrites the widget's "next pass in 1h 04m" string into the
// compact "next pass · 1h 04m" form the scene uses; returns "" when the
// widget's pass segment is empty so the row collapses cleanly.
func issNextPass(raw string) (text, color string) {
	pass := strings.TrimPrefix(pipeAtRaw(raw, 1), "next pass in ")
	if pass == "" {
		return "", ""
	}
	return "next pass · " + pass, ""
}

// parseISSPassDuration parses the widget's pass segment ("next pass in
// 1h 23m" or "1h 23m" or "47m") into a time.Duration. Returns ok=false
// for any input that doesn't match either shape so callers don't have
// to guess.
func parseISSPassDuration(s string) (time.Duration, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "next pass in ")
	if s == "" {
		return 0, false
	}
	var h, m int
	if i := strings.Index(s, "h "); i > 0 {
		v, err := strconv.Atoi(strings.TrimSpace(s[:i]))
		if err != nil {
			return 0, false
		}
		h = v
		s = strings.TrimSpace(s[i+2:])
	}
	if i := strings.Index(s, "m"); i > 0 {
		v, err := strconv.Atoi(strings.TrimSpace(s[:i]))
		if err != nil {
			return 0, false
		}
		m = v
	} else {
		return 0, false
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute, true
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
