package facts

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"
)

// TIL pulls the top posts from r/todayilearned (24h window) and returns
// a random one each Fetch, suppressing the most recently shown so the
// scene rotates instead of repeating. Output is "TIL|<title>" so the
// scene's pipeAt(1) formatter drops the header — the corner lightbulb
// glyph carries the labelling work.
type TIL struct {
	HTTP *http.Client

	mu     sync.Mutex
	rng    *rand.Rand
	recent []string // ring of recently-shown post IDs to avoid repeats
}

func NewTIL() *TIL {
	return &TIL{
		HTTP: defaultHTTP(),
		rng:  rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0x71C0DE)),
	}
}

func (t *TIL) Name() string { return "facts/til" }

// tilRecentHistory is how many recently-shown post IDs we remember. With
// a 10-post top-of-day window, suppressing the last 5 keeps the rotation
// feeling fresh while still always finding a candidate.
const tilRecentHistory = 5

func (t *TIL) wasRecent(id string) bool {
	for _, r := range t.recent {
		if r == id {
			return true
		}
	}
	return false
}

func (t *TIL) remember(id string) {
	t.recent = append(t.recent, id)
	if len(t.recent) > tilRecentHistory {
		t.recent = t.recent[len(t.recent)-tilRecentHistory:]
	}
}

type tilPost struct {
	ID    string
	Title string
}

func (t *TIL) Fetch(ctx context.Context) (string, error) {
	var body struct {
		Data struct {
			Children []struct {
				Data struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	const url = "https://www.reddit.com/r/todayilearned/top.json?t=day&limit=10"
	if err := getJSON(ctx, t.HTTP, url, &body); err != nil {
		return "", fmt.Errorf("reddit til: %w", err)
	}
	posts := make([]tilPost, 0, len(body.Data.Children))
	for _, c := range body.Data.Children {
		if c.Data.Title == "" || c.Data.ID == "" {
			continue
		}
		posts = append(posts, tilPost{ID: c.Data.ID, Title: c.Data.Title})
	}
	if len(posts) == 0 {
		return "", fmt.Errorf("reddit til: no posts in top-of-day window")
	}

	t.mu.Lock()
	// Filter out recently-shown posts; if that empties the pool (small
	// window, all recently shown) fall back to the full set so we always
	// return something.
	candidates := make([]tilPost, 0, len(posts))
	for _, p := range posts {
		if !t.wasRecent(p.ID) {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		candidates = posts
	}
	picked := candidates[t.rng.IntN(len(candidates))]
	t.remember(picked.ID)
	t.mu.Unlock()

	return "TIL|" + picked.Title, nil
}
