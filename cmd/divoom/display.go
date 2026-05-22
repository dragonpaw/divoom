package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
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

// runDisplayLines installs one or more stacked Text elements filled with
// numbered "L1 / L2 / L3 …" wrapped content, then holds for 60s. Used to
// probe the device's Text-rendering behaviour: at what line count does a
// single tall element stop rendering, and does splitting across multiple
// stacked elements work around the cap.
//
// Usage:
//
//	divoom display lines [-n N] [-font N] [-blocks N] [-starty Y] [-height H]
//
// Defaults aim to match the whimsy/quote layout so results transfer.
func runDisplayLines(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("lines", flag.ContinueOnError)
	n := fs.Int("n", 20, "total numbered markers to spread across all blocks")
	font := fs.Int("font", 34, "FontSize for every text block")
	blocks := fs.Int("blocks", 1, "how many stacked Text elements to split content across")
	startY := fs.Int("starty", 480, "Y coordinate of the topmost block")
	totalH := fs.Int("height", 760, "total vertical area to distribute across blocks (px)")
	firstID := fs.Int("firstid", 10, "first Text-element ID (subsequent blocks use firstid+1, +2, …)")
	noReset := fs.Bool("no-reset", false, "skip the pre-Exit; reuse cached element state on the device")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *blocks < 1 || *blocks > 6 {
		return fmt.Errorf("lines: blocks must be 1..6 (got %d)", *blocks)
	}

	client, _, err := connectToFrame(ctx)
	if err != nil {
		return err
	}

	// Reset the device's per-ID cache before installing the probe layout.
	// Without this, the device appears to keep an element's FontSize /
	// Height from the previous install whenever an ID is reused — only
	// TextMessage updates reliably between EnterCustomMode calls.
	if !*noReset {
		exitCtx, ec := context.WithTimeout(ctx, 5*time.Second)
		if err := client.ExitCustomMode(exitCtx); err != nil {
			slog.Warn("pre-exit failed (continuing anyway)", "err", err)
		}
		ec()
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

	// Distribute n markers evenly across the requested number of blocks.
	// Each block gets its own contiguous range so the line numbers stay
	// in reading order even after the device wraps them.
	perBlock := (*n + *blocks - 1) / *blocks
	blockH := *totalH / *blocks

	disp := make([]frame.DispElement, 0, *blocks)
	for b := 0; b < *blocks; b++ {
		first := b*perBlock + 1
		last := first + perBlock - 1
		if last > *n {
			last = *n
		}
		if first > *n {
			break
		}
		msg := lineMarkers(first, last)
		id := *firstID + b
		disp = append(disp, frame.DispElement{
			ID: id, Type: "Text",
			StartX: 20, StartY: *startY + b*blockH, Width: 760, Height: blockH,
			Align:       2,
			FontSize:    *font,
			FontID:      52,
			FontColor:   "#ebdbb2",
			BgColor:     "#1d2021",
			TextMessage: msg,
		})
		slog.Info("block", "id", id, "first", first, "last", last,
			"starty", *startY+b*blockH, "height", blockH, "chars", len(msg))
	}

	layout := frame.CustomMode{
		BackgroundImageLocalFlag: 1,
		BackgroundImageAddr:      bgPath,
		DispList:                 disp,
	}

	enterCtx, ecancel := context.WithTimeout(ctx, 10*time.Second)
	if err := client.EnterCustomMode(enterCtx, layout); err != nil {
		ecancel()
		return fmt.Errorf("EnterCustomControlMode: %w", err)
	}
	ecancel()
	slog.Info("layout installed — note the highest LN visible in each block",
		"n", *n, "blocks", *blocks, "font", *font, "starty", *startY, "block_height", blockH)

	const hold = 60 * time.Second
	slog.Info("holding — Ctrl+C to exit early", "hold", hold)
	select {
	case <-time.After(hold):
	case <-ctx.Done():
	}
	return nil
}

// lineMarkers builds "L<first> padding… / L<first+1> padding… / …" so the
// device's text wrapper has enough text to fill many lines but each line
// still starts with a visible marker.
func lineMarkers(first, last int) string {
	const filler = " padding padding padding padding"
	var sb strings.Builder
	for i := first; i <= last; i++ {
		if sb.Len() > 0 {
			sb.WriteString(" / ")
		}
		fmt.Fprintf(&sb, "L%d", i)
		sb.WriteString(filler)
	}
	return sb.String()
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
