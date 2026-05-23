package finance

import (
	"context"
	"strings"
	"sync"

	"github.com/dragonpaw/divoom/internal/widget"
)

// RotatingTicker fetches one of several configured symbols per call,
// cycling through them in declaration order. Unlike the generic
// internal/widget/rotator package (which picks by weighted random),
// markets rotation is strictly round-robin so each scene activation
// shows a different ticker — the daemon's whole point.
type RotatingTicker struct {
	tickers []*Ticker

	mu     sync.Mutex
	cursor int
}

// NewRotating constructs a rotator from a list of Yahoo symbols. An
// empty list defaults to ["QQQ"] so the markets scene always has a live
// widget to fetch from. Symbols are trimmed and upper-cased on the way
// in so callers don't have to be precise about input formatting.
func NewRotating(symbols []string) *RotatingTicker {
	if len(symbols) == 0 {
		symbols = []string{"QQQ"}
	}
	rt := &RotatingTicker{}
	for _, s := range symbols {
		clean := strings.ToUpper(strings.TrimSpace(s))
		if clean == "" {
			continue
		}
		rt.tickers = append(rt.tickers, NewTicker(clean))
	}
	if len(rt.tickers) == 0 {
		rt.tickers = append(rt.tickers, NewTicker("QQQ"))
	}
	return rt
}

func (r *RotatingTicker) Name() string { return "finance/rotator" }

func (r *RotatingTicker) Fetch(ctx context.Context) (string, error) {
	r.mu.Lock()
	t := r.tickers[r.cursor]
	r.cursor = (r.cursor + 1) % len(r.tickers)
	r.mu.Unlock()
	return t.Fetch(ctx)
}

var _ widget.Widget = (*RotatingTicker)(nil)
