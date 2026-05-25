package main

import (
	"testing"
	"time"
)

// TestWeightModifiers pins the time-of-day multipliers each scene applies
// to its base Weight at pick time. The exact multiplier values aren't
// load-bearing — they just bias rotation toward "right scene at right
// hour" — but the relative ordering (peak hours > off hours) is.
func TestWeightModifiers(t *testing.T) {
	// All cases use time.Local so the modifier reads match what the
	// daemon would see. Pick a Wednesday for the markets weekday test
	// and a Saturday for the weekend skip.
	wed := func(h, m int) time.Time {
		return time.Date(2026, 5, 27, h, m, 0, 0, time.Local)
	}
	sat := func(h, m int) time.Time {
		return time.Date(2026, 5, 30, h, m, 0, 0, time.Local)
	}

	type tc struct {
		name string
		fn   func(time.Time) float64
		when time.Time
		want float64
	}
	cases := []tc{
		// Markets — open Mon-Fri 06:30-13:00 PT.
		{"markets wed 10am", marketsHours, wed(10, 0), 3.0},
		{"markets wed 6:29am", marketsHours, wed(6, 29), 0.3},
		{"markets wed 6:30am", marketsHours, wed(6, 30), 3.0},
		{"markets wed 1pm", marketsHours, wed(13, 0), 0.3},
		{"markets sat 10am", marketsHours, sat(10, 0), 0.3},

		// Sunrise — favours dawn/dusk windows.
		{"sunrise 6am", sunriseModifier, wed(6, 30), 3.0},
		{"sunrise 7pm", sunriseModifier, wed(19, 0), 1.5},
		{"sunrise noon", sunriseModifier, wed(12, 0), 0.4},
		{"sunrise 2am", sunriseModifier, wed(2, 0), 0.2},

		// Moonphase — favours night.
		{"moon 11pm", moonphaseModifier, wed(23, 0), 1.8},
		{"moon 2am", moonphaseModifier, wed(2, 0), 1.8},
		{"moon noon", moonphaseModifier, wed(12, 0), 0.4},

		// NASA — favours late night.
		{"nasa 10pm", nasaModifier, wed(22, 0), 2.5},
		{"nasa 3am", nasaModifier, wed(3, 0), 2.5},
		{"nasa noon", nasaModifier, wed(12, 0), 1.0},

		// Cocktail — favours apéritif window.
		{"cocktail 7pm", cocktailModifier, wed(19, 0), 2.0},
		{"cocktail 10am", cocktailModifier, wed(10, 0), 0.7},

		// Forecast — favours morning + pre-bed.
		{"forecast 7am", forecastModifier, wed(7, 0), 2.0},
		{"forecast 9pm", forecastModifier, wed(21, 0), 2.0},
		{"forecast noon", forecastModifier, wed(12, 0), 1.0},

		// Calendar — favours morning only.
		{"calendar 7am", calendarModifier, wed(7, 0), 2.0},
		{"calendar 9pm", calendarModifier, wed(21, 0), 1.0},

		// ISS — favours visible-pass twilight, dampens midday.
		{"iss 6:30am", issModifier, wed(6, 30), 1.5},
		{"iss 8pm", issModifier, wed(20, 0), 1.5},
		{"iss noon", issModifier, wed(12, 0), 0.4},
		{"iss 2am", issModifier, wed(2, 0), 0.6},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.fn(c.when)
			if got != c.want {
				t.Errorf("%s at %s: got %v, want %v",
					c.name, c.when.Format("Mon 15:04"), got, c.want)
			}
		})
	}
}
