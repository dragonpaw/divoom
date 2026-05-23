package news

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dragonpaw/divoom/internal/widget"
)

// RedditTopOfDay returns one of today's top posts from a randomly-chosen
// subreddit from r.Subs. Top-of-day rather than top-of-hour or hot so the
// post is interesting enough to justify a wall-clock slot for the next
// scene rotation.
//
// Output format is a 7-segment pipe-delimited string:
//
//	"<sub>|<title>|<domain>|<score>|<author>|<age>|<comments>"
//
// where <sub> is the bare subreddit name (no "r/" prefix), <domain> is
// reddit's `data.domain` (self-posts come back as "self.<sub>" which is
// fine to render as-is), <age> is a humanised relative-time ("3h", "11h",
// "now"), and the score / comments slots are integer strings.
//
// Cache: one post per hour, regardless of how often Fetch is called. The
// scene rotates every 3 minutes; without the cache we'd hit reddit ~20x
// per hour from one IP for each configured sub, which is rude and would
// invite rate-limiting.
//
// NSFW (over_18) posts are NOT filtered — that's a deliberate choice the
// user signed off on. Stickied (announcement / pinned) posts ARE dropped
// so the rotation reflects actual top content rather than mod posts.
type RedditTopOfDay struct {
	Subs []string
	HTTP *http.Client

	mu       sync.Mutex
	rng      *rand.Rand
	cached   string
	cachedAt time.Time
}

// redditCacheTTL is how long a previously-emitted post is reused before
// Fetch hits reddit again. One hour matches the design goal of "fresh
// each hour" while staying well clear of reddit's anonymous-IP rate
// limits (~60 req/min) for the daemon's scene cadence.
const redditCacheTTL = time.Hour

// NewRedditTopOfDay returns a widget configured to rotate across the
// given subreddits. Subs should already be normalised by the caller
// (lowercased, "r/" prefix stripped); see parseSubredditList in
// cmd/divoom/scenes.go for the canonical normaliser.
func NewRedditTopOfDay(subs []string) *RedditTopOfDay {
	return &RedditTopOfDay{
		Subs: subs,
		HTTP: &http.Client{Timeout: 15 * time.Second},
		rng:  rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0xC0FFEE)),
	}
}

func (r *RedditTopOfDay) Name() string { return "reddit" }

// redditListing mirrors the shape of reddit's `/r/<sub>/top.json`
// response. Only the fields the widget actually surfaces are decoded;
// unknown fields are ignored.
type redditListing struct {
	Data struct {
		Children []struct {
			Data redditPost `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type redditPost struct {
	Subreddit   string  `json:"subreddit"`
	Title       string  `json:"title"`
	Domain      string  `json:"domain"`
	Score       int     `json:"score"`
	Author      string  `json:"author"`
	NumComments int     `json:"num_comments"`
	CreatedUTC  float64 `json:"created_utc"`
	Stickied    bool    `json:"stickied"`
}

func (r *RedditTopOfDay) Fetch(ctx context.Context) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cached != "" && time.Since(r.cachedAt) < redditCacheTTL {
		return r.cached, nil
	}

	if len(r.Subs) == 0 {
		return "", fmt.Errorf("reddit: no subreddits configured")
	}
	sub := r.Subs[r.rng.IntN(len(r.Subs))]

	url := fmt.Sprintf("https://www.reddit.com/r/%s/top.json?t=day&limit=5", sub)
	var listing redditListing
	if err := r.getJSON(ctx, url, &listing); err != nil {
		if r.cached != "" {
			return r.cached, nil
		}
		return "", fmt.Errorf("reddit r/%s: %w", sub, err)
	}

	posts := make([]redditPost, 0, len(listing.Data.Children))
	for _, c := range listing.Data.Children {
		if c.Data.Stickied {
			continue
		}
		posts = append(posts, c.Data)
	}
	if len(posts) == 0 {
		if r.cached != "" {
			return r.cached, nil
		}
		return "", fmt.Errorf("reddit r/%s: no eligible posts", sub)
	}

	picked := posts[r.rng.IntN(len(posts))]
	out := assembleRedditPipe(picked, sub, time.Now())
	r.cached = out
	r.cachedAt = time.Now()
	return out, nil
}

// assembleRedditPipe builds the 7-segment pipe string from a decoded
// post. Split out so the unit test can pin the assembly without a live
// HTTP round-trip.
func assembleRedditPipe(p redditPost, sub string, now time.Time) string {
	title := strings.TrimSpace(p.Title)
	age := ""
	if p.CreatedUTC > 0 {
		age = humanizeRedditAge(now.Sub(time.Unix(int64(p.CreatedUTC), 0)))
	}
	return strings.Join([]string{
		sub,
		title,
		p.Domain,
		strconv.Itoa(p.Score),
		p.Author,
		age,
		strconv.Itoa(p.NumComments),
	}, "|")
}

// humanizeRedditAge formats a positive duration as "Nh" up to a day,
// "Nm" under an hour, "now" for sub-minute. Days collapse back to a
// fixed "24h+" rather than rolling over to "Nd" — the scene only ever
// shows top-of-day posts, so the upper bound is naturally bounded.
func humanizeRedditAge(d time.Duration) string {
	if d < time.Minute {
		return "now"
	}
	if d < time.Hour {
		return strconv.Itoa(int(d/time.Minute)) + "m"
	}
	if d < 24*time.Hour {
		return strconv.Itoa(int(d/time.Hour)) + "h"
	}
	return "24h+"
}

func (r *RedditTopOfDay) getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := r.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

var _ widget.Widget = (*RedditTopOfDay)(nil)
