package quotes

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const userAgent = "divoom-dashboard/0.1 (github.com/dragonpaw/divoom)"

// ZenQuotes fetches a random quote from the ZenQuotes.io API.
type ZenQuotes struct {
	HTTP *http.Client
}

func NewZenQuotes() *ZenQuotes {
	return &ZenQuotes{
		HTTP: &http.Client{Timeout: 10 * time.Second},
	}
}

func (z *ZenQuotes) Name() string { return "quotes/ZenQuotes" }

type zenQuotesResp struct {
	Quote  string `json:"q"`
	Author string `json:"a"`
}

func (z *ZenQuotes) Fetch(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://zenquotes.io/api/random", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := z.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("zenquotes request: %w", err)
	}
	defer resp.Body.Close()

	// Rate limit: ZenQuotes free tier is ~5 req per 30s per IP.
	if resp.StatusCode == http.StatusTooManyRequests {
		slog.Warn("zenquotes rate limit hit")
		return "", fmt.Errorf("zenquotes rate limited")
	}
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("zenquotes http %d", resp.StatusCode)
	}

	var results []zenQuotesResp
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return "", fmt.Errorf("zenquotes decode: %w", err)
	}
	if len(results) == 0 {
		return "", fmt.Errorf("zenquotes empty response")
	}

	q := results[0]
	return "ZenQuotes|" + q.Quote + "|" + q.Author, nil
}
