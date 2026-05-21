// Package rotator wraps several Widgets in a single Widget that returns
// content from a randomly-chosen source each refresh. Useful for the
// "whimsy" slot where we want variety with weighted frequencies — some
// sources show often, others are rare easter eggs.
package rotator

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/dragonpaw/divoom/internal/widget"
)

// Source pairs a widget with the weight controlling how often it's picked
// relative to its siblings. A source with weight 5 fires 5x as often as
// one with weight 1.
type Source struct {
	Widget widget.Widget
	Weight int
}

// Rotator picks a Source by weighted random sample on each Fetch. If
// MaxLen > 0, the picked source's text is truncated at the last word
// boundary that fits within MaxLen bytes, with an ellipsis appended —
// the Times Frame's Text rendering caps somewhere around two line-heights
// regardless of declared Height, so this is how we keep content from
// clipping mid-character on long facts.
type Rotator struct {
	name    string
	sources []Source
	maxLen  int

	mu  sync.Mutex
	rng *rand.Rand
}

func New(name string, sources []Source) *Rotator {
	return &Rotator{
		name:    name,
		sources: sources,
		rng:     rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0)),
	}
}

// WithMaxLen returns r after setting its truncation budget. <=0 disables
// truncation (the default).
func (r *Rotator) WithMaxLen(n int) *Rotator {
	r.maxLen = n
	return r
}

func (r *Rotator) Name() string { return r.name }

func (r *Rotator) Fetch(ctx context.Context) (string, error) {
	chosen := r.pick()
	if chosen == nil {
		return "", errors.New("rotator: no sources with positive weight")
	}
	text, err := chosen.Fetch(ctx)
	if err != nil {
		return "", fmt.Errorf("%s: %w", chosen.Name(), err)
	}
	if r.maxLen > 0 && len(text) > r.maxLen {
		text = truncateAtWord(text, r.maxLen)
	}
	return text, nil
}

// truncateAtWord trims s to the last word boundary that fits within
// maxLen bytes, appending "…" so the truncation is visible.
func truncateAtWord(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	cut := s[:maxLen]
	if i := strings.LastIndexByte(cut, ' '); i > 0 {
		cut = cut[:i]
	}
	return cut + "…"
}

// pick selects one source via weighted random sample. Locked because rng
// is not goroutine-safe.
func (r *Rotator) pick() widget.Widget {
	r.mu.Lock()
	defer r.mu.Unlock()

	total := 0
	for _, s := range r.sources {
		if s.Weight > 0 {
			total += s.Weight
		}
	}
	if total == 0 {
		return nil
	}

	roll := r.rng.IntN(total)
	for _, s := range r.sources {
		if s.Weight <= 0 {
			continue
		}
		roll -= s.Weight
		if roll < 0 {
			return s.Widget
		}
	}
	return nil // unreachable when total > 0
}
