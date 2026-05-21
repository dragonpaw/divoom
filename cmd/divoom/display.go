package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dragonpaw/divoom/internal/adb"
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/render"
)

// runDisplay dispatches `divoom display <action>` subcommands.
func runDisplay(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("display: missing action (try `divoom display test`)")
	}
	switch args[0] {
	case "test":
		return runDisplayTest(ctx)
	case "ticker":
		return runDisplayTicker(ctx)
	case "lines":
		return runDisplayLines(ctx, args[1:])
	default:
		return fmt.Errorf("display: unknown action %q", args[0])
	}
}

// runDisplayLines installs a single Text element on a known background with
// a synthesized multi-line message, holds 60s, then exits. Used to probe
// how the device's Text rendering handles tall blocks — does it cap, wrap,
// scroll vertically, etc.
//
// Usage: `divoom display lines [n]` (default n=8)
func runDisplayLines(ctx context.Context, args []string) error {
	n := 8
	if len(args) > 0 {
		v, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("lines: bad count %q: %w", args[0], err)
		}
		n = v
	}

	client, _, err := connectToFrame(ctx)
	if err != nil {
		return err
	}

	bgBytes, err := render.HeroBackground(render.FormatJPEG, time.Now())
	if err != nil {
		return fmt.Errorf("render bg: %w", err)
	}
	tmp, err := os.CreateTemp("", "lines-bg-*.jpg")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	tmp.Write(bgBytes)
	tmp.Close()

	const bgPath = "/userdata/wallclock_lines_bg.jpg"
	pushCtx, pcancel := context.WithTimeout(ctx, 10*time.Second)
	if err := adb.Push(pushCtx, tmp.Name(), bgPath); err != nil {
		pcancel()
		return fmt.Errorf("push bg: %w", err)
	}
	pcancel()

	// Generate n markers, each padded so the device MUST wrap onto multiple
	// lines. Without padding, "L1 / L2 / L3" all fits on one line and
	// doesn't probe wrap/scroll behavior at all.
	const filler = " padding padding padding padding"
	var sb strings.Builder
	for i := 1; i <= n; i++ {
		if i > 1 {
			sb.WriteString(" / ")
		}
		fmt.Fprintf(&sb, "L%d", i)
		sb.WriteString(filler)
	}
	msg := sb.String()
	slog.Info("test message", "lines_target", n, "chars", len(msg), "raw", msg)

	// Generous geometry: full bottom area, big enough to plausibly hold
	// 10+ lines at FontSize 26. If the device caps, we'll see it.
	layout := frame.CustomMode{
		BackgroundImageLocalFlag: 1,
		BackgroundImageAddr:      bgPath,
		DispList: []frame.DispElement{
			{
				ID: 10, Type: "Text",
				StartX: 20, StartY: 500, Width: 760, Height: 720,
				Align:       2,
				FontSize:    26,
				FontID:      52,
				FontColor:   "#ebdbb2",
				BgColor:     "#1d2021",
				TextMessage: msg,
			},
		},
	}

	enterCtx, ecancel := context.WithTimeout(ctx, 10*time.Second)
	if err := client.EnterCustomMode(enterCtx, layout); err != nil {
		ecancel()
		return fmt.Errorf("EnterCustomControlMode: %w", err)
	}
	ecancel()
	slog.Info("layout installed — watch the wall",
		"start_y", 500, "height", 720, "font_size", 26, "lines_requested", n)

	const hold = 60 * time.Second
	slog.Info("holding — Ctrl+C to exit early", "hold", hold)
	select {
	case <-time.After(hold):
	case <-ctx.Done():
	}
	return nil
}

// runDisplayTicker proves the dynamic-text channel end-to-end:
//
//   1) Install a layout with a Time element + a Text placeholder.
//   2) Once per second for 30s, push `Device/UpdateDisplayItems` with a
//      countdown string. If the on-screen text changes in lockstep, we've
//      validated the full "dashboard daemon" pattern — backgrounds via
//      adb, layout via local API, dynamic data via UpdateDisplayItems,
//      zero cloud round-trips.
//   3) Exit + re-select the preset dial cleanly.
func runDisplayTicker(ctx context.Context) error {
	client, _, err := connectToFrame(ctx)
	if err != nil {
		return err
	}

	stateCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	initial, err := client.GetClockInfo(stateCtx)
	cancel()
	if err != nil {
		return fmt.Errorf("GetClockInfo: %w", err)
	}
	slog.Info("captured initial dial", "clock_id", initial.ClockID, "brightness", initial.Brightness)

	bgBytes, err := render.TestBackground(render.FormatJPEG)
	if err != nil {
		return fmt.Errorf("render background: %w", err)
	}
	tmp, err := os.CreateTemp("", "wallclock-bg-*.jpg")
	if err != nil {
		return fmt.Errorf("temp file: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(bgBytes); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	tmp.Close()

	pushCtx, pushCancel := context.WithTimeout(ctx, 10*time.Second)
	if err := adb.Push(pushCtx, tmp.Name(), onDeviceTestBgPath); err != nil {
		pushCancel()
		return fmt.Errorf("push background: %w", err)
	}
	pushCancel()

	// One Text element we'll patch on every tick. ID and Type both must be
	// set; UpdateDisplayItems only knows how to find a Text element by ID,
	// so the ID must match what we install here.
	const tickerID = 2

	layout := frame.CustomMode{
		BackgroundImageLocalFlag: 1,
		BackgroundImageAddr:      onDeviceTestBgPath,
		DispList: []frame.DispElement{
			{
				ID: 1, Type: "Time",
				StartX: 50, StartY: 280, Width: 700, Height: 220,
				Align:     2,
				FontSize:  180,
				FontID:    52,
				FontColor: "#ebdbb2",
				BgColor:   "#1d2021",
			},
			{
				ID: tickerID, Type: "Text",
				StartX: 50, StartY: 900, Width: 700, Height: 120,
				Align:       2,
				FontSize:    100,
				FontID:      52,
				FontColor:   "#fabd2f", // gruvbox yellow
				BgColor:     "#1d2021",
				TextMessage: "starting...",
			},
		},
	}

	slog.Info("installing ticker layout", "bg", onDeviceTestBgPath, "ticker_id", tickerID)
	enterCtx, enterCancel := context.WithTimeout(ctx, 10*time.Second)
	if err := client.EnterCustomMode(enterCtx, layout); err != nil {
		enterCancel()
		return fmt.Errorf("EnterCustomControlMode: %w", err)
	}
	enterCancel()

	defer func() {
		bg := context.Background()
		exitCtx, c := context.WithTimeout(bg, 5*time.Second)
		if err := client.ExitCustomMode(exitCtx); err != nil {
			slog.Error("ExitCustomControlMode", "err", err)
		}
		c()
		selCtx, c := context.WithTimeout(bg, 5*time.Second)
		if err := client.SelectClock(selCtx, initial.ClockID); err != nil {
			slog.Warn("could not re-select preset dial", "clock_id", initial.ClockID, "err", err)
		}
		c()
		slog.Info("restored preset dial", "clock_id", initial.ClockID)
	}()

	const (
		duration = 30 * time.Second
		interval = 1 * time.Second
	)
	slog.Info("ticking — Ctrl+C to exit early", "duration", duration, "interval", interval)

	deadline := time.Now().Add(duration)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		now := time.Now()
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil
		}
		msg := fmt.Sprintf("T-%02d", int(remaining.Round(time.Second).Seconds()))

		updCtx, c := context.WithTimeout(ctx, 3*time.Second)
		err := client.UpdateTexts(updCtx, []frame.TextUpdate{
			{ID: tickerID, TextMessage: msg},
		})
		c()
		if err != nil {
			slog.Warn("UpdateDisplayItems failed", "msg", msg, "err", err)
		} else {
			slog.Info("tick", "msg", msg, "elapsed_ms", time.Since(now).Milliseconds())
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// onDeviceTestBgPath is where we drop the test background via adb. Sits at the
// root of /userdata to match the docs' `/userdata/clock_bg.jpg` example
// (avoids the unknown question of "does the loader walk subdirs?"). For
// real scenes we'll move to /userdata/wallclock/scenes/<name>.jpg once we
// confirm subdir paths are honored.
const onDeviceTestBgPath = "/userdata/wallclock_test_bg.jpg"

// runDisplayTest:
//   1) Renders the gruvbox test background as JPEG.
//   2) adb-pushes it into /userdata on the frame.
//   3) Tells the frame to use it as a local-file background via
//      `Device/EnterCustomControlMode` with `BackgroudImageLocalFlag: 1`.
//   4) Holds 30s, then restores the preset dial. Ctrl+C-safe.
//
// No HTTP server, no cloud round-trip — pure LAN + adb. If this works,
// the whole "cloud-proxy fetches our URLs" problem is solved for static
// scene backgrounds.
func runDisplayTest(ctx context.Context) error {
	client, _, err := connectToFrame(ctx)
	if err != nil {
		return err
	}

	// Capture the active dial before we replace it so we can put the frame
	// back to exactly where it was. ExitCustomControlMode by itself leaves
	// the preset half-rendered (background returns, but the dial's widgets
	// don't re-initialize) — re-selecting the dial id kicks it cleanly.
	stateCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	initial, err := client.GetClockInfo(stateCtx)
	cancel()
	if err != nil {
		return fmt.Errorf("GetClockInfo: %w", err)
	}
	slog.Info("captured initial dial", "clock_id", initial.ClockID, "brightness", initial.Brightness)

	bgBytes, err := render.TestBackground(render.FormatJPEG)
	if err != nil {
		return fmt.Errorf("render background: %w", err)
	}

	tmp, err := os.CreateTemp("", "wallclock-bg-*.jpg")
	if err != nil {
		return fmt.Errorf("temp file: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(bgBytes); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	tmp.Close()

	pushCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	if err := adb.Push(pushCtx, tmp.Name(), onDeviceTestBgPath); err != nil {
		cancel()
		return fmt.Errorf("push background: %w", err)
	}
	cancel()

	layout := frame.CustomMode{
		BackgroundImageLocalFlag: 1,
		BackgroundImageAddr:      onDeviceTestBgPath,
		DispList: []frame.DispElement{
			{
				ID:        1,
				Type:      "Time",
				StartX:    50,
				StartY:    480,
				Width:     700,
				Height:    220,
				Align:     2,
				FontSize:  180,
				FontID:    52,
				FontColor: "#ebdbb2",
				BgColor:   "#1d2021",
			},
		},
	}

	slog.Info("installing test layout (local bg via adb)", "bg", onDeviceTestBgPath)
	enterCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := client.EnterCustomMode(enterCtx, layout); err != nil {
		return fmt.Errorf("EnterCustomControlMode: %w", err)
	}

	defer func() {
		// Always run cleanup, even if the caller's ctx was cancelled (Ctrl+C).
		bg := context.Background()

		exitCtx, exitCancel := context.WithTimeout(bg, 5*time.Second)
		if err := client.ExitCustomMode(exitCtx); err != nil {
			slog.Error("failed to exit custom mode — frame may be stuck on test layout", "err", err)
		}
		exitCancel()

		// Kick the dial back to a clean state — Exit alone leaves preset
		// widgets unrendered.
		selCtx, selCancel := context.WithTimeout(bg, 5*time.Second)
		if err := client.SelectClock(selCtx, initial.ClockID); err != nil {
			slog.Warn("could not re-select preset dial", "clock_id", initial.ClockID, "err", err)
		}
		selCancel()

		slog.Info("restored preset dial", "clock_id", initial.ClockID)
	}()

	const hold = 30 * time.Second
	slog.Info("holding test layout — Ctrl+C to exit early", "hold", hold)
	select {
	case <-time.After(hold):
	case <-ctx.Done():
		slog.Info("interrupted — restoring frame")
	}
	return nil
}

// connectToFrame resolves a Times Frame IP (env override first, cloud
// discovery otherwise) and returns a ready-to-use API client plus the
// discovered Device (nil if we used an env override). Shared by `probe`
// and `display`.
func connectToFrame(ctx context.Context) (*frame.Client, *frame.Device, error) {
	if ip := os.Getenv("DIVOOM_FRAME_IP"); ip != "" {
		slog.Info("using DIVOOM_FRAME_IP override", "ip", ip)
		return frame.New(ip), nil, nil
	}
	discoverCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	slog.Info("discovering Times Frame via cloud LAN endpoint")
	d, err := frame.FindTimesFrame(discoverCtx, os.Getenv("DIVOOM_FRAME_MAC"))
	if err != nil {
		return nil, nil, fmt.Errorf("discover: %w", err)
	}
	slog.Info("found device",
		"name", d.DeviceName,
		"id", d.DeviceID,
		"ip", d.DevicePrivateIP,
		"mac", d.DeviceMac,
		"hardware", d.Hardware,
	)
	return frame.New(d.DevicePrivateIP), &d, nil
}
