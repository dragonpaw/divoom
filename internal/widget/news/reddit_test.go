package news

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestHumanizeRedditAge pins the three format bands plus the upper-bound
// clamp. Top-of-day posts can be up to 24h old; anything older shouldn't
// appear in a t=day listing but the clamp keeps a clock-skew edge case
// from producing a nonsense "0h".
func TestHumanizeRedditAge(t *testing.T) {
	cases := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"zero", 0, "now"},
		{"30s", 30 * time.Second, "now"},
		{"1m", time.Minute, "1m"},
		{"47m", 47 * time.Minute, "47m"},
		{"1h", time.Hour, "1h"},
		{"11h", 11 * time.Hour, "11h"},
		{"23h", 23 * time.Hour, "23h"},
		{"25h clamps", 25 * time.Hour, "24h+"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := humanizeRedditAge(c.d); got != c.want {
				t.Errorf("humanizeRedditAge(%s) = %q, want %q", c.d, got, c.want)
			}
		})
	}
}

// TestRedditPipeAssembly decodes a fixture listing and pins the pipe-
// assembled output a Fetch round-trip would emit. Covers: sticky post
// dropped from the candidate pool, NSFW post kept (not filtered), and
// the 7-segment field order.
func TestRedditPipeAssembly(t *testing.T) {
	// Two posts: one stickied (must be dropped), one regular NSFW
	// (must be kept). After filtering, only the regular post remains;
	// the assembly should reflect its fields exactly.
	const fixture = `{
        "data": {
            "children": [
                {"data": {
                    "subreddit": "pcgaming",
                    "title": "MOD ANNOUNCEMENT - read me",
                    "domain": "self.pcgaming",
                    "score": 9999,
                    "author": "AutoModerator",
                    "num_comments": 0,
                    "created_utc": 1716480000,
                    "stickied": true,
                    "over_18": false
                }},
                {"data": {
                    "subreddit": "pcgaming",
                    "title": "  Steam Deck OLED review thread  ",
                    "domain": "rockpapershotgun.com",
                    "score": 4321,
                    "author": "tester",
                    "num_comments": 187,
                    "created_utc": 1716480000,
                    "stickied": false,
                    "over_18": true
                }}
            ]
        }
    }`

	var listing redditListing
	if err := json.Unmarshal([]byte(fixture), &listing); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	var kept []redditPost
	for _, c := range listing.Data.Children {
		if c.Data.Stickied {
			continue
		}
		kept = append(kept, c.Data)
	}
	if len(kept) != 1 {
		t.Fatalf("after sticky drop: got %d posts, want 1", len(kept))
	}

	now := time.Unix(1716480000, 0).Add(3 * time.Hour)
	out := assembleRedditPipe(kept[0], "pcgaming", now)

	parts := strings.Split(out, "|")
	if len(parts) != 7 {
		t.Fatalf("segment count = %d, want 7 (raw=%q)", len(parts), out)
	}
	want := []string{
		"pcgaming",
		"Steam Deck OLED review thread",
		"rockpapershotgun.com",
		"4321",
		"tester",
		"3h",
		"187",
	}
	for i, w := range want {
		if parts[i] != w {
			t.Errorf("segment %d = %q, want %q", i, parts[i], w)
		}
	}
}
