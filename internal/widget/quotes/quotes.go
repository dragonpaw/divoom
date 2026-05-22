// Package quotes serves random one-liners drawn from a curated set of
// sources (Devil's Dictionary, Jargon File, Star Trek, Babylon 5, sassy
// one-liners). Each source is its own Widget that emits
// "Source|body|author", where the third pipe-separated segment may be
// empty when no attribution exists for the picked quote. The scene that
// renders this splits each segment into its own Text element.
package quotes

import (
	"context"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/dragonpaw/divoom/internal/widget"
)

// Source returns a random quote from a fixed in-memory pool. One Source
// per distinct origin; combine with the rotator package to mix them at
// fixed weights regardless of pool size. defaultAuthor is used for
// quotes whose entry text does not carry its own trailing
// " — Author" suffix.
type Source struct {
	label         string
	defaultAuthor string
	quotes        []string

	mu  sync.Mutex
	rng *rand.Rand
}

func newSource(label string, quotes []string) *Source {
	return newAuthoredSource(label, "", quotes)
}

func newAuthoredSource(label, defaultAuthor string, quotes []string) *Source {
	return &Source{
		label:         label,
		defaultAuthor: defaultAuthor,
		quotes:        quotes,
		rng:           rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(len(quotes)))),
	}
}

func (s *Source) Name() string { return "quotes/" + s.label }

// Label returns the source's display label — exposed so the daemon can
// log the configured quote sources at startup.
func (s *Source) Label() string { return s.label }

// Count returns how many quote entries this Source has.
func (s *Source) Count() int { return len(s.quotes) }

func (s *Source) Fetch(ctx context.Context) (string, error) {
	s.mu.Lock()
	raw := s.quotes[s.rng.IntN(len(s.quotes))]
	s.mu.Unlock()
	body, author := splitAuthor(raw)
	if author == "" {
		author = s.defaultAuthor
	}
	return s.label + "|" + body + "|" + author, nil
}

// splitAuthor partitions an entry on its trailing " — Author" suffix.
// We use LastIndex so em-dashes embedded inside the quote body don't
// steal the split. When the entry carries no separator, body is the
// whole string and author is empty.
func splitAuthor(s string) (body, author string) {
	const sep = " — "
	if i := strings.LastIndex(s, sep); i >= 0 {
		return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+len(sep):])
	}
	return strings.TrimSpace(s), ""
}

var _ widget.Widget = (*Source)(nil)
