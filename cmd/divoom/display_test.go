package main

import (
	"context"
	"testing"
	"time"
)

// probeFrameIP should fail fast when nothing is listening, proving that
// connectToFrame actually verifies a $DIVOOM_FRAME_IP override rather than
// blindly trusting it. 127.0.0.1 with no Times Frame service on :9000 is a
// reliable "nothing there" target on any host (the dashboard runs in
// containers; loopback :9000 is never the device).
func TestProbeFrameIP_DeadAddressFailsFast(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	client, err := probeFrameIP(ctx, "127.0.0.1")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("probeFrameIP(127.0.0.1) succeeded unexpectedly; got client=%v", client)
	}
	// probeFrameIP uses a 3s internal timeout; allow generous slack for slow CI.
	if elapsed > 4*time.Second {
		t.Errorf("probeFrameIP took %s — should fail fast on closed port", elapsed)
	}
}
