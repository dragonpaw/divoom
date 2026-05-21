// Package calendar groups widgets tied to the wall calendar — day-of-year
// counters, countdowns to dated events, etc. Calendar widgets don't talk to
// any API; they're pure functions of time.Now().
package calendar

import (
	"context"
	"fmt"
	"time"
)

// DayOfYear renders "you're P.P% of the way through YYYY" — a friendly
// year-progress reminder. The "calendar" scene splits this into a big
// percentage and a smaller explanatory line.
type DayOfYear struct{}

func NewDayOfYear() *DayOfYear { return &DayOfYear{} }

func (d *DayOfYear) Name() string { return "calendar/dayofyear" }

func (d *DayOfYear) Fetch(ctx context.Context) (string, error) {
	now := time.Now()
	day := now.YearDay()
	yearDays := 365
	if isLeap(now.Year()) {
		yearDays = 366
	}
	pct := float64(day-1) / float64(yearDays) * 100
	// HEADER|BODY: header is a label, body is a friendly sentence with
	// the percentage embedded.
	return fmt.Sprintf("day of year|You're %.1f%% done with %d!", pct, now.Year()), nil
}

func isLeap(y int) bool {
	return (y%4 == 0 && y%100 != 0) || y%400 == 0
}
