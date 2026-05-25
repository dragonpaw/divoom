package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/dragonpaw/divoom/internal/render"
)

// startDailyRefresh kicks off an in-process scheduler that re-renders
// and adb-pushes the day-dependent scene backgrounds at every local
// midnight, plus once on startup. Currently handles:
//
//   - the calendar bg (today cell moves, past cells accrete)
//   - the genart bg (algorithm + seed change at the date boundary)
//
// Both refresh in the same cycle. No font reload (so divoom_app does
// NOT crash-restart). Fire-and-forget — failures log and continue.
//
// Other scene bgs are stable across days; their content (weather,
// forecast, etc.) flows through live UpdateDisplayItems text.
func startDailyRefresh(ctx context.Context) {
	go func() {
		// Startup push — covers fresh deploys / container restarts.
		if err := pushDailyBackgrounds(ctx); err != nil {
			slog.Warn("daily bg startup push failed", "err", err)
		} else {
			slog.Info("daily bgs pushed at startup")
		}

		for {
			next := nextLocalMidnight(time.Now())
			wait := time.Until(next)
			slog.Info("daily bg refresh scheduled", "next", next.Format(time.RFC3339), "in", wait.Round(time.Second))
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			if err := pushDailyBackgrounds(ctx); err != nil {
				slog.Warn("daily bg midnight push failed", "err", err)
				continue
			}
			slog.Info("daily bgs pushed at midnight")
		}
	}()
}

// pushDailyBackgrounds renders the calendar + genart bgs for the
// current instant and pushes both to the device. No font reload,
// no crash-restart — just two adb pushes of the JPGs.
func pushDailyBackgrounds(ctx context.Context) error {
	now := time.Now()
	cal, err := render.CalendarBackground(
		now,
		parseSpecialDates(os.Getenv("DIVOOM_SPECIAL_DATES")),
		usFederalHolidays(now.Year()),
		render.FormatJPEG,
	)
	if err != nil {
		return err
	}
	if err := pushBytes(ctx, cal, bgCalendar); err != nil {
		return err
	}
	gen, err := render.GenartBackground(now, render.FormatJPEG)
	if err != nil {
		return err
	}
	return pushBytes(ctx, gen, bgGenart)
}

// nextLocalMidnight returns the next 00:00:00 in the local timezone
// strictly after `from`. Handles DST transitions implicitly via
// time.Date arithmetic in the local zone.
func nextLocalMidnight(from time.Time) time.Time {
	tomorrow := from.In(time.Local).AddDate(0, 0, 1)
	return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(),
		0, 0, 0, 0, time.Local)
}
