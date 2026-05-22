package sky

import (
	"math"
	"testing"
)

// daysUntilFullMoon should give ~14.77 days from a new moon, drop to ~0
// at full moon, and roll forward to a full cycle when we pass full.
func TestDaysUntilFullMoon(t *testing.T) {
	cases := []struct {
		name string
		f    float64 // phase fraction in [0,1)
		want float64 // expected days until next full moon
	}{
		{"new moon", 0.0, synodicMonthDays / 2},
		{"first quarter", 0.25, synodicMonthDays * 0.25},
		{"full moon rolls forward", 0.5, synodicMonthDays},
		{"just past full", 0.55, synodicMonthDays * 0.95},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := daysUntilFullMoon(c.f)
			if math.Abs(got-c.want) > 0.01 {
				t.Errorf("daysUntilFullMoon(%v) = %v, want %v", c.f, got, c.want)
			}
		})
	}
}
