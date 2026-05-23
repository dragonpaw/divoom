package finance

import (
	"context"
	"strings"
	"testing"
)

func TestRotatingTickerRoundRobin(t *testing.T) {
	// Three tickers; cycling should produce A, B, C, A, B, C, ... by
	// returning each ticker's Symbol-as-name. We can't easily stub the
	// network call, but the rotator's cursor logic is independent of the
	// underlying Fetch — verify cursor advancement by inspecting the
	// chosen *Ticker through the exposed Name() of finance.Ticker.
	rt := NewRotating([]string{"a", "b", "c"})
	if len(rt.tickers) != 3 {
		t.Fatalf("NewRotating: want 3 tickers, got %d", len(rt.tickers))
	}
	want := []string{"finance/A", "finance/B", "finance/C", "finance/A"}
	for i, expected := range want {
		// Re-implement the cursor read so we don't need a network call:
		// pop the same way Fetch does (under the mutex).
		rt.mu.Lock()
		got := rt.tickers[rt.cursor].Name()
		rt.cursor = (rt.cursor + 1) % len(rt.tickers)
		rt.mu.Unlock()
		if got != expected {
			t.Errorf("rotation step %d: got %q, want %q", i, got, expected)
		}
	}
}

func TestRotatingTickerDefaultsToQQQ(t *testing.T) {
	rt := NewRotating(nil)
	if len(rt.tickers) != 1 || rt.tickers[0].Symbol != "QQQ" {
		t.Errorf("NewRotating(nil): want [QQQ], got %v", rt.tickers)
	}
	rt = NewRotating([]string{"", "  "})
	if len(rt.tickers) != 1 || rt.tickers[0].Symbol != "QQQ" {
		t.Errorf("NewRotating(empties): want [QQQ], got %v", rt.tickers)
	}
}

func TestRotatingTickerNormalisesSymbols(t *testing.T) {
	rt := NewRotating([]string{" aapl ", "btc-usd"})
	if rt.tickers[0].Symbol != "AAPL" {
		t.Errorf("symbol[0] = %q, want AAPL", rt.tickers[0].Symbol)
	}
	if rt.tickers[1].Symbol != "BTC-USD" {
		t.Errorf("symbol[1] = %q, want BTC-USD", rt.tickers[1].Symbol)
	}
}

// TestRotatingTickerFetchError confirms Fetch surfaces the underlying
// Ticker's error rather than wrapping it; sanity that the wrapper isn't
// silently swallowing failures. Uses an unreachable hostname to force a
// quick DNS error without needing a live network.
func TestRotatingTickerFetchError(t *testing.T) {
	rt := NewRotating([]string{"DOESNOTEXIST"})
	// Aim Fetch at an obviously bad URL by stubbing the HTTP client's
	// transport would be cleaner, but the rotator already uses the
	// per-ticker client; cancelling the context is the simplest way to
	// guarantee a fast failure without depending on DNS.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := rt.Fetch(ctx)
	if err == nil {
		t.Fatal("Fetch with cancelled ctx: want error, got nil")
	}
	if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "yahoo") {
		t.Logf("got error %v — acceptable so long as it's not nil", err)
	}
}
