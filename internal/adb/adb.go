// Package adb wraps the `adb` CLI for the few operations we need against
// the Times Frame: pushing rendered backgrounds and fonts into /userdata/.
// The device runs TinaLinux + BusyBox; interactive `adb shell` is gated by
// a login prompt we don't have credentials for, but `adb push` / `adb pull`
// bypass the shell entirely and work without authentication.
package adb

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

// Push copies a local file to the given path on the device. `adb` must be on
// PATH; the device must be connected (USB or `adb connect`). When multiple
// devices are attached, set ADB_SERIAL in the environment and we'll target
// that one.
func Push(ctx context.Context, src, dst string) error {
	args := []string{}
	if serial := os.Getenv("ADB_SERIAL"); serial != "" {
		args = append(args, "-s", serial)
	}
	args = append(args, "push", src, dst)

	cmd := exec.CommandContext(ctx, "adb", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("adb push %s -> %s: %w (output: %s)", src, dst, err, out)
	}
	slog.Info("adb pushed", "src", src, "dst", dst, "out", strings.TrimRight(string(out), "\r\n "))
	return nil
}
