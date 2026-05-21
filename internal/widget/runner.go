package widget

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Runner wraps a Widget with a background refresh goroutine + a cached
// "last known value." Decouples data-source freshness (the Interval) from
// display cadence (whoever reads Latest). Scene rotation reads Latest at
// activation time and bakes it into the installed layout; the Runner keeps
// refreshing in the background on its own schedule.
type Runner struct {
	Widget   Widget
	Interval time.Duration

	mu     sync.RWMutex
	latest string

	firstDone chan struct{}
	once      sync.Once
}

// NewRunner builds a runner. Latest() returns "" until the first refresh
// completes; WaitFirstFetch can be used to gate work that needs the cache
// populated.
func NewRunner(w Widget, interval time.Duration) *Runner {
	return &Runner{
		Widget:    w,
		Interval:  interval,
		firstDone: make(chan struct{}),
	}
}

// Start loops forever (until ctx cancelled), refreshing on the configured
// interval. The first refresh fires immediately so Latest() is non-empty
// quickly.
func (r *Runner) Start(ctx context.Context) {
	r.refresh(ctx)
	r.once.Do(func() { close(r.firstDone) })

	ticker := time.NewTicker(r.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.refresh(ctx)
		}
	}
}

// WaitFirstFetch blocks until Start has attempted its first refresh. The
// refresh itself may have failed — Latest() can still return "" — but at
// least one fetch attempt is complete. Bound the wait with ctx.
func (r *Runner) WaitFirstFetch(ctx context.Context) error {
	select {
	case <-r.firstDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Latest returns the most recently fetched value. Empty string until the
// first refresh succeeds. Callers should substitute a placeholder ("—") as
// appropriate.
func (r *Runner) Latest() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.latest
}

// Name forwards to the wrapped Widget; useful for logging in scene code.
func (r *Runner) Name() string { return r.Widget.Name() }

func (r *Runner) refresh(ctx context.Context) {
	fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	text, err := r.Widget.Fetch(fetchCtx)
	if err != nil {
		slog.Warn("widget fetch failed", "widget", r.Widget.Name(), "err", err)
		return
	}
	r.mu.Lock()
	r.latest = text
	r.mu.Unlock()
	slog.Info("widget refreshed", "widget", r.Widget.Name(), "text", text)
}
