package main

import (
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
)

// "pickup" — trash / recycle / compost reminder scene. Eligible only
// during the window from 17:00 the day BEFORE a configured pickup
// through 08:00 the morning OF pickup. Outside that window the scene's
// WeightModifier returns 0, making it invisible to the rotation.
//
// Config: DIVOOM_PICKUP_SCHEDULE, format:
//
//	"trash:mon,recycle:wed,compost:wed"
//
// — pairs of `<type>:<dow>` where dow is a 3-letter lowercase weekday.
// Unset → scene weight 0 → never picked.
//
// Element count: 3 body Text + always-on (2 Text + 1 Time). Within cap.
// The bg is a quiet baked frame; identity comes from the big yellow
// "▲ TOMORROW / TODAY:" headline and the pickup-type list below it.

// PickupTypeOrder is the canonical render order so "trash + recycle"
// always reads in the same direction across activations.
var pickupTypeOrder = []string{"trash", "recycle", "compost", "yard"}

// pickupDayTokens maps the 3-letter env tokens to time.Weekday.
var pickupDayTokens = map[string]time.Weekday{
	"sun": time.Sunday,
	"mon": time.Monday,
	"tue": time.Tuesday,
	"wed": time.Wednesday,
	"thu": time.Thursday,
	"fri": time.Friday,
	"sat": time.Saturday,
}

// PickupRule is one configured pickup: a type ("trash" / "recycle" /
// "compost") and the weekday it runs.
type PickupRule struct {
	Type string
	Day  time.Weekday
}

// parsePickupSchedule splits DIVOOM_PICKUP_SCHEDULE into rules. Bad
// tokens log a warning and get dropped (a typo shouldn't crash). Empty
// input returns nil so the scene factory can treat the feature as off.
func parsePickupSchedule(env string) []PickupRule {
	env = strings.TrimSpace(env)
	if env == "" {
		return nil
	}
	var rules []PickupRule
	for _, raw := range strings.Split(env, ",") {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		colon := strings.IndexByte(entry, ':')
		if colon < 0 {
			slog.Warn("DIVOOM_PICKUP_SCHEDULE: missing ':' in entry, skipping", "entry", entry)
			continue
		}
		typ := strings.ToLower(strings.TrimSpace(entry[:colon]))
		dow := strings.ToLower(strings.TrimSpace(entry[colon+1:]))
		wd, ok := pickupDayTokens[dow]
		if !ok || typ == "" {
			slog.Warn("DIVOOM_PICKUP_SCHEDULE: bad type or day, skipping", "entry", entry)
			continue
		}
		rules = append(rules, PickupRule{Type: typ, Day: wd})
	}
	return rules
}

// pickupActive returns the headline prefix ("TODAY" or "TOMORROW"),
// the sorted-and-deduped list of pickup types in the active window,
// the date of the next pickup, and ok=false when no pickup is active
// at `now`. Active window per spec: 17:00 day-before → 08:00 day-of.
func pickupActive(rules []PickupRule, now time.Time) (prefix string, types []string, pickupDate time.Time, ok bool) {
	loc := now.In(time.Local)
	hour := loc.Hour()
	today := loc.Weekday()
	tomorrow := (today + 1) % 7

	set := map[string]bool{}
	if hour < 8 {
		// Morning-of window.
		for _, r := range rules {
			if r.Day == today {
				set[r.Type] = true
			}
		}
		if len(set) > 0 {
			prefix = "TODAY"
			pickupDate = time.Date(loc.Year(), loc.Month(), loc.Day(),
				0, 0, 0, 0, loc.Location())
		}
	}
	if len(set) == 0 && hour >= 17 {
		// Evening-before window.
		for _, r := range rules {
			if r.Day == tomorrow {
				set[r.Type] = true
			}
		}
		if len(set) > 0 {
			prefix = "TOMORROW"
			t := loc.AddDate(0, 0, 1)
			pickupDate = time.Date(t.Year(), t.Month(), t.Day(),
				0, 0, 0, 0, t.Location())
		}
	}
	if len(set) == 0 {
		return "", nil, time.Time{}, false
	}
	// Render order per pickupTypeOrder; unknown types appended alpha.
	for _, t := range pickupTypeOrder {
		if set[t] {
			types = append(types, t)
			delete(set, t)
		}
	}
	leftovers := make([]string, 0, len(set))
	for t := range set {
		leftovers = append(leftovers, t)
	}
	sort.Strings(leftovers)
	types = append(types, leftovers...)
	return prefix, types, pickupDate, true
}

// pickupScene builds the optional pickup-reminder scene. Returns nil
// when DIVOOM_PICKUP_SCHEDULE is unset/empty so buildScenes drops it
// from the rotation (parallel to the github / reddit / agenda pattern).
func pickupScene() *scene.Scene {
	rules := parsePickupSchedule(os.Getenv("DIVOOM_PICKUP_SCHEDULE"))
	if len(rules) == 0 {
		return nil
	}
	return &scene.Scene{
		Name:   "pickup",
		Weight: 80, // Informational × 2 — when active, fire promptly.
		BgPath: bgPickup,
		Elements: []frame.DispElement{
			// "▲ TOMORROW:" or "▲ TODAY:" — big yellow alert.
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 600, Width: 640, Height: 120,
				Align: 2, FontSize: 70, FontID: fontProse,
				FontColor: cYellow, BgColor: cBgHard,
			},
			// "TRASH + RECYCLE" — large mono headline.
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 40, StartY: 760, Width: 720, Height: 140,
				Align: 2, FontSize: 80, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			// "wed may 27" — small dim date.
			{
				ID: idSceneSub2, Type: "Text",
				StartX: 80, StartY: 940, Width: 640, Height: 50,
				Align: 2, FontSize: 32, FontID: fontProseLight,
				FontColor: cFgDark, BgColor: cBgHard,
			},
		},
		OnActivate: pickupOnActivate(rules),
		WeightModifier: func(now time.Time) float64 {
			_, _, _, ok := pickupActive(rules, now)
			if ok {
				return 1.0
			}
			return 0
		},
	}
}

// pickupOnActivate returns an OnActivate hook closured over the
// configured rules; computes the headline / types / date from `now`
// and writes them onto the element slice.
func pickupOnActivate(rules []PickupRule) func(time.Time, string, []frame.DispElement) {
	return func(now time.Time, _ string, elements []frame.DispElement) {
		prefix, types, date, ok := pickupActive(rules, now)
		if !ok {
			// Shouldn't happen — pick() filters via WeightModifier — but
			// be defensive so a clock skew doesn't render junk.
			return
		}
		headline := "▲ " + prefix + ":"
		body := strings.ToUpper(strings.Join(types, " + "))
		dateStr := strings.ToLower(date.Format("Mon Jan 2"))
		for i := range elements {
			switch elements[i].ID {
			case idSceneMain:
				elements[i].TextMessage = headline
			case idSceneSub1:
				elements[i].TextMessage = body
			case idSceneSub2:
				elements[i].TextMessage = dateStr
			}
		}
	}
}
