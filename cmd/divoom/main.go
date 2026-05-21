package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var err error
	switch os.Args[1] {
	case "probe":
		err = runProbe(ctx)
	case "display":
		err = runDisplay(ctx, os.Args[2:])
	case "render":
		err = runRender(os.Args[2:])
	case "serve":
		err = runServe(ctx)
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		slog.Error("command failed", "subcommand", os.Args[1], "err", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `divoom — Times Frame controller

USAGE
  divoom <subcommand>

SUBCOMMANDS
  probe          Discover the Times Frame on the LAN and print its current state.
  display test   Install a one-element test layout for 30s then restore.
  display ticker Install a layout with a Text element and patch it once a
                   second for 30s via UpdateDisplayItems. Validates the
                   dynamic-text channel end-to-end.
  render         Render every scene background into ./dist/scenes/*.jpg.
                   Used by CI to publish to the public assets repo.
  serve          Run the dashboard daemon: install layout, poll widgets,
                   patch Text elements via UpdateDisplayItems. Reads
                   DIVOOM_LAT and DIVOOM_LON for weather.
  help           Show this message.

ENVIRONMENT
  DIVOOM_FRAME_MAC   If set, picks the device with this MAC when more than one
                     Times Frame is present on the LAN.
  DIVOOM_FRAME_IP    If set, skips cloud discovery and talks to this IP directly.
                     Useful if you want to keep traffic off Divoom's servers.
`)
}

func runProbe(ctx context.Context) error {
	client, device, err := connectToFrame(ctx)
	if err != nil {
		return err
	}

	infoCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	info, err := client.GetClockInfo(infoCtx)
	if err != nil {
		return fmt.Errorf("GetClockInfo: %w", err)
	}
	slog.Info("frame state",
		"clock_id", info.ClockID,
		"brightness", info.Brightness,
	)

	// Only print the "save this for next time" hint when discovery actually
	// found the device — env-override probes already know these values.
	if device != nil {
		fmt.Println()
		fmt.Println("  Save this for reproducible runs:")
		fmt.Printf("    export DIVOOM_FRAME_MAC=%s\n", device.DeviceMac)
		fmt.Printf("    export DIVOOM_FRAME_IP=%s   # optional, skips cloud discovery\n", device.DevicePrivateIP)
		fmt.Println()
	}
	return nil
}
