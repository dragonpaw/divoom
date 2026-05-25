// Package calendar groups widgets tied to the wall calendar — day-of-year
// counters, countdowns to dated events, etc. Calendar widgets don't talk to
// any API; they're pure functions of time.Now().
package calendar

import (
	"context"
	"fmt"
	"time"

	"github.com/dragonpaw/divoom/internal/widget"
)

// Calendar renders the year-progress as three pipe-separated segments —
// a big percentage, a year label, and a day count — that the calendar
// scene splits across separate Text elements.
//
// Output:  "39%|Year 2026|Day 142 of 366"
type Calendar struct{}

func NewCalendar() *Calendar { return &Calendar{} }

func (d *Calendar) Name() string { return "calendar" }

func (d *Calendar) Fetch(ctx context.Context) (string, error) {
	now := time.Now()
	day := now.YearDay()
	yearDays := 365
	if isLeap(now.Year()) {
		yearDays = 366
	}
	pct := float64(day-1) / float64(yearDays) * 100
	return fmt.Sprintf("%.0f%%|Year %d|Day %d of %d", pct, now.Year(), day, yearDays), nil
}

func isLeap(y int) bool {
	return (y%4 == 0 && y%100 != 0) || y%400 == 0
}

var _ widget.Widget = (*Calendar)(nil)
