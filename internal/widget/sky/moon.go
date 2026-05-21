// Package sky has ambient/sky-related widgets (moon phase, sunrise, ISS pass,
// etc.). Most are computable client-side from time + lat/lon so they don't
// burn API quota.
package sky

import (
	"context"
	"fmt"
	"math"
	"time"
)

// Moon is a Widget that computes the current lunar phase from astronomical
// constants — no external API, no quota concerns.
type Moon struct{}

func NewMoon() *Moon { return &Moon{} }

func (m *Moon) Name() string { return "moon" }

// Synodic month length (full new-moon to new-moon cycle), in days.
const synodicMonthDays = 29.530588853

// Reference new moon: 2000-01-06 18:14 UTC. Standard astronomical anchor;
// good to ~1 hour across centuries — fine for "what phase is the moon in."
var referenceNewMoon = time.Date(2000, 1, 6, 18, 14, 0, 0, time.UTC)

func (m *Moon) Fetch(ctx context.Context) (string, error) {
	frac, name := phase(time.Now().UTC())
	illum := illumination(frac)
	return fmt.Sprintf("moon · %s · %d%%", name, int(illum+0.5)), nil
}

// phase returns the fractional position through the synodic cycle (0 = new,
// 0.5 = full, 1 → 0 = new again) along with a human-readable phase name.
func phase(now time.Time) (float64, string) {
	days := now.Sub(referenceNewMoon).Hours() / 24.0
	f := math.Mod(days, synodicMonthDays) / synodicMonthDays
	if f < 0 {
		f += 1
	}
	switch {
	case f < 0.0270 || f > 0.9730:
		return f, "new"
	case f < 0.2230:
		return f, "waxing crescent"
	case f < 0.2770:
		return f, "first quarter"
	case f < 0.4730:
		return f, "waxing gibbous"
	case f < 0.5270:
		return f, "full"
	case f < 0.7230:
		return f, "waning gibbous"
	case f < 0.7770:
		return f, "last quarter"
	default:
		return f, "waning crescent"
	}
}

// illumination: percentage of the disc lit, derived from phase fraction.
// 0% at new moon (f=0), 100% at full moon (f=0.5), 50% at the quarters.
func illumination(f float64) float64 {
	return (1 - math.Cos(2*math.Pi*f)) / 2 * 100
}
