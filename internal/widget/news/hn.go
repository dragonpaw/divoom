// Package news fetches headlines from real-news sources, filtered by
// keywords so the dashboard surfaces stories the user actually cares
// about. Currently only HackerNews; community-curated, surfaces real news
// over press releases through reader engagement.
package news

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"math/rand/v2"
	"net/http"
	neturl "net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dragonpaw/divoom/internal/widget"
)

const userAgent = "divoom-dashboard/0.1 (github.com/dragonpaw/divoom)"

// HN scans HackerNews's top stories for one whose title matches any of
// the configured keywords (case-insensitive substring). Returns the
// highest-ranked match, augmented with the article's og:description (or
// equivalent meta summary) when one can be fetched cheaply.
//
// Output format is an 8-segment pipe-delimited string:
//
//	"Hacker News|<title>|<domain>|<summary>|<score>|<author>|<age>|<comments>"
//
// where <domain> is the URL host with a leading "www." stripped (empty
// for Ask HN / Show HN self-posts), <age> is humanised relative to
// item.time (e.g. "47m", "3h", "1d"), and <comments> is item.descendants
// (HN's total-comments-including-replies field). All fields except
// <title> may be empty for partial / unusual stories.
type HN struct {
	Keywords []string
	// Limit caps how many top stories we'll inspect; the firebase API
	// returns up to 500 ids but only the first few dozen are usually
	// interesting. 30 is enough to find a topic match most days.
	Limit int
	HTTP  *http.Client

	mu     sync.Mutex
	rng    *rand.Rand
	recent []int64 // ring of recently-shown IDs to avoid repeats
}

func NewHN(keywords []string) *HN {
	return &HN{
		Keywords: keywords,
		Limit:    30,
		HTTP:     &http.Client{Timeout: 15 * time.Second},
		rng:      rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0xC0FFEE)),
	}
}

func (h *HN) Name() string { return "news/hn" }

// wasRecent / remember manage the small ring buffer of recently-shown
// story IDs so we don't pick the same headline twice in a row when the
// keyword-matching pool is tiny. Caller must hold h.mu.
func (h *HN) wasRecent(id int64) bool {
	for _, r := range h.recent {
		if r == id {
			return true
		}
	}
	return false
}

func (h *HN) remember(id int64) {
	h.recent = append(h.recent, id)
	if len(h.recent) > recentHistory {
		h.recent = h.recent[len(h.recent)-recentHistory:]
	}
}

type hnStory struct {
	ID          int64
	Title       string
	URL         string
	Text        string
	Score       int
	By          string
	Time        int64 // Unix seconds (HN item.time)
	Descendants int   // total comments including replies
}

// recentHistory is how many recently-shown story IDs we remember and
// avoid re-picking on the next Fetch. With the top-30 window often
// containing only 2-3 keyword matches, suppressing the last ~5 makes
// the rotation feel fresh until HN's frontpage actually changes.
const recentHistory = 5

func (h *HN) Fetch(ctx context.Context) (string, error) {
	var ids []int64
	if err := h.getJSON(ctx, "https://hacker-news.firebaseio.com/v0/topstories.json", &ids); err != nil {
		return "", fmt.Errorf("hn topstories: %w", err)
	}
	if h.Limit > 0 && len(ids) > h.Limit {
		ids = ids[:h.Limit]
	}

	// Collect every keyword-matching story in the top-N window, then
	// pick one at random. Always returning the top-ranked match caused
	// the same one or two headlines to dominate the rotation.
	var matched []hnStory
	for _, id := range ids {
		var item struct {
			Title       string `json:"title"`
			Type        string `json:"type"`
			URL         string `json:"url"`
			Text        string `json:"text"`
			Score       int    `json:"score"`
			By          string `json:"by"`
			Time        int64  `json:"time"`
			Descendants int    `json:"descendants"`
		}
		url := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)
		if err := h.getJSON(ctx, url, &item); err != nil {
			continue // skip individual failures and keep scanning
		}
		if item.Type != "story" || item.Title == "" {
			continue
		}
		if !matches(item.Title, h.Keywords) {
			continue
		}
		matched = append(matched, hnStory{
			ID:          id,
			Title:       item.Title,
			URL:         item.URL,
			Text:        item.Text,
			Score:       item.Score,
			By:          item.By,
			Time:        item.Time,
			Descendants: item.Descendants,
		})
	}
	if len(matched) == 0 {
		return "", fmt.Errorf("no HN story in top %d matched keywords", len(ids))
	}

	h.mu.Lock()
	// Filter out recently-shown stories; if that empties the pool
	// (e.g. only one match exists and we've shown it), fall back to
	// the full matched set so we always return something.
	candidates := make([]hnStory, 0, len(matched))
	for _, s := range matched {
		if !h.wasRecent(s.ID) {
			candidates = append(candidates, s)
		}
	}
	if len(candidates) == 0 {
		candidates = matched
	}
	picked := candidates[h.rng.IntN(len(candidates))]
	h.remember(picked.ID)
	h.mu.Unlock()

	summary := h.summarise(ctx, picked.URL, picked.Text)
	domain := hnDomain(picked.URL)
	score := ""
	if picked.Score > 0 {
		score = strconv.Itoa(picked.Score)
	}
	comments := ""
	if picked.Descendants > 0 {
		comments = strconv.Itoa(picked.Descendants)
	}
	age := ""
	if picked.Time > 0 {
		age = humanizeHNAge(time.Since(time.Unix(picked.Time, 0)))
	}
	// Order: title | domain | summary | score | author | age | comments.
	return strings.Join([]string{
		"Hacker News",
		picked.Title,
		domain,
		summary,
		score,
		picked.By,
		age,
		comments,
	}, "|"), nil
}

// hnDomain extracts the bare host from a story URL, stripping a leading
// "www.". Returns "" for empty / unparseable URLs (Ask HN, Show HN self-
// posts have no URL); the scene mounts the domain element with
// AllowEmpty so a blank value renders as nothing rather than "—".
func hnDomain(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := neturl.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}
	host := strings.ToLower(u.Host)
	host = strings.TrimPrefix(host, "www.")
	return host
}

// humanizeHNAge formats a positive duration the way HN's frontpage does:
// "<1m" under a minute, "Nm" under an hour, "Nh" under a day, else "Nd".
// Negative / zero durations (clock skew, future timestamps) clamp to
// "<1m" — the frontpage never shows negative ages.
func humanizeHNAge(d time.Duration) string {
	if d < time.Minute {
		return "<1m"
	}
	if d < time.Hour {
		return strconv.Itoa(int(d/time.Minute)) + "m"
	}
	if d < 24*time.Hour {
		return strconv.Itoa(int(d/time.Hour)) + "h"
	}
	return strconv.Itoa(int(d/(24*time.Hour))) + "d"
}

// summarise returns a short description of the linked article, or "" if
// no summary is available cheaply. For Ask-HN / Show-HN self-posts the
// HN item's own `text` is used (stripped of HTML); for link posts we
// fetch the article's HTML and pull `og:description` (or its twitter /
// standard <meta> siblings). All failures are silent — we always have
// the title as a fallback.
func (h *HN) summarise(ctx context.Context, articleURL, hnText string) string {
	if hnText != "" {
		return clamp(stripHTML(hnText), 400)
	}
	if articleURL == "" {
		return ""
	}
	desc := h.fetchMetaDescription(ctx, articleURL)
	return clamp(desc, 400)
}

var metaContentRE = regexp.MustCompile(
	`(?is)<meta\s+[^>]*?(?:property|name)\s*=\s*["'](?:og:description|twitter:description|description)["'][^>]*?content\s*=\s*["']([^"']+)["']`,
)
var metaContentReversedRE = regexp.MustCompile(
	`(?is)<meta\s+[^>]*?content\s*=\s*["']([^"']+)["'][^>]*?(?:property|name)\s*=\s*["'](?:og:description|twitter:description|description)["']`,
)

func (h *HN) fetchMetaDescription(ctx context.Context, url string) string {
	fetchCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := h.HTTP.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return ""
	}
	// 256 KB is more than enough for any reasonable <head>; we stop
	// reading after the first match so most fetches won't even hit it.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return ""
	}
	for _, re := range []*regexp.Regexp{metaContentRE, metaContentReversedRE} {
		if m := re.FindSubmatch(body); m != nil {
			return strings.TrimSpace(html.UnescapeString(string(m[1])))
		}
	}
	return ""
}

// stripHTML removes the simple HTML markup the HN API emits for self
// posts ( <p>, <a href>, <i>, etc. ) and decodes entities. Not a
// general-purpose sanitiser — HN's markup vocabulary is tiny.
var tagRE = regexp.MustCompile(`<[^>]+>`)

func stripHTML(s string) string {
	s = tagRE.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

// clamp returns s truncated to at most n bytes, ending on a word
// boundary with a trailing "…" when truncation actually happens.
func clamp(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := s[:n]
	if i := strings.LastIndexByte(cut, ' '); i > 0 {
		cut = cut[:i]
	}
	return cut + "…"
}

func (h *HN) getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := h.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func matches(title string, kws []string) bool {
	lower := strings.ToLower(title)
	for _, kw := range kws {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

var _ widget.Widget = (*HN)(nil)
