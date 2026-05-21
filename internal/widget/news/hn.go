// Package news fetches headlines from real-news sources, filtered by
// keywords so the dashboard surfaces stories the user actually cares
// about. Currently only HackerNews; community-curated, surfaces real news
// over press releases through reader engagement.
package news

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const userAgent = "divoom-dashboard/0.1 (github.com/dragonpaw/divoom)"

// HN scans HackerNews's top stories for one whose title matches any of
// the configured keywords (case-insensitive substring). Returns the
// highest-ranked match.
type HN struct {
	Keywords []string
	// Limit caps how many top stories we'll inspect; the firebase API
	// returns up to 500 ids but only the first few dozen are usually
	// interesting. 30 is enough to find a topic match most days.
	Limit int
	HTTP  *http.Client
}

func NewHN(keywords []string) *HN {
	return &HN{
		Keywords: keywords,
		Limit:    30,
		HTTP:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (h *HN) Name() string { return "news/hn" }

func (h *HN) Fetch(ctx context.Context) (string, error) {
	var ids []int64
	if err := h.getJSON(ctx, "https://hacker-news.firebaseio.com/v0/topstories.json", &ids); err != nil {
		return "", fmt.Errorf("hn topstories: %w", err)
	}
	if h.Limit > 0 && len(ids) > h.Limit {
		ids = ids[:h.Limit]
	}

	for _, id := range ids {
		var item struct {
			Title string `json:"title"`
			Type  string `json:"type"`
		}
		url := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)
		if err := h.getJSON(ctx, url, &item); err != nil {
			continue // skip individual failures and keep scanning
		}
		if item.Type != "story" || item.Title == "" {
			continue
		}
		if matches(item.Title, h.Keywords) {
			return "Hacker News|" + item.Title, nil
		}
	}
	return "", fmt.Errorf("no HN story in top %d matched keywords", len(ids))
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
