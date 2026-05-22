// Package easter is the rare "easter egg" Widget — a random one-liner from
// a curated pool, picked fresh each call. Output uses HEADER|BODY so the
// ambient scene can render the label and phrase on separate lines; some
// lines compute live state (e.g. Mercury retrograde status) via
// {placeholders}.
package easter

import (
	"context"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/dragonpaw/divoom/internal/widget"
)

// Easter returns a random curated phrase. Lowest-weight source in the
// whimsy rotation so these show up infrequently.
type Easter struct {
	mu  sync.Mutex
	rng *rand.Rand
}

func New() *Easter {
	return &Easter{rng: rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0))}
}

func (e *Easter) Name() string { return "easter" }

func (e *Easter) Fetch(ctx context.Context) (string, error) {
	e.mu.Lock()
	phrase := lines[e.rng.IntN(len(lines))]
	e.mu.Unlock()
	if strings.Contains(phrase, "{mercury}") {
		phrase = strings.Replace(phrase, "{mercury}", mercuryStatus(), 1)
	}
	return phrase, nil
}

// Curated one-liners in HEADER|BODY format. Keep them short, dryly
// playful, and free of emoji the device's font might not have glyphs for.
var lines = []string{
	"musing|the universe is mostly empty space, like your inbox should be",
	"today's vibe|cautiously optimistic",
	"reminder|you have permission to close the laptop",
	"observation|the cat is judging you (yes, that one)",
	"fact|you are made of star stuff (also coffee)",
	"checklist|drink water, stretch, breathe",
	"perspective|in 4 billion years the sun will go red giant. relax.",
	"Mercury|{mercury}",
	"thought|the moon does not care about your code review",
	"today|today is the perfect day for a perfectly average day",
	"quote|between two pines is a doorway — john muir",
	"fact|you are 96% the same atoms as you were a year ago",
	"easter egg|if you can read this, the dashboard didn't crash",
	"affirmation|Iosevka thinks you're doing great",
	"wish|may your terminal stay warm and your fans stay cool",
	"easter egg|this is the rare one — congratulations",
}

// mercuryRetrogradePeriods covers 2026-2027. Extend before 2028.
// Astronomical data; conventional ISO dates (UTC).
var mercuryRetrogradePeriods = []struct{ start, end string }{
	{"2026-02-26", "2026-03-20"},
	{"2026-06-30", "2026-07-24"},
	{"2026-10-24", "2026-11-13"},
	{"2027-02-09", "2027-03-03"},
	{"2027-06-12", "2027-07-06"},
	{"2027-10-07", "2027-10-27"},
}

func mercuryStatus() string {
	today := time.Now().UTC().Format("2006-01-02")
	for _, p := range mercuryRetrogradePeriods {
		if today >= p.start && today <= p.end {
			return "retrograde (blame the planets)"
		}
	}
	return "direct (today's failures are yours alone)"
}

var _ widget.Widget = (*Easter)(nil)
