// Package wikipedia fetches text snippets from Wikimedia's public feed
// APIs. Currently only the "On this day / events" feed, which returns
// the day's historical events sourced from the English Wikipedia.
package wikipedia

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dragonpaw/divoom/internal/widget"
)

const userAgent = "divoom-dashboard/0.1 (github.com/dragonpaw/divoom)"

// OnThisDay fetches today's "On this day in history" events from
// Wikimedia's free REST feed and returns one at random per Fetch. No
// auth required; Wikimedia asks only for a meaningful User-Agent.
type OnThisDay struct {
	HTTP *http.Client

	mu  sync.Mutex
	rng *rand.Rand
}

func NewOnThisDay() *OnThisDay {
	return &OnThisDay{
		// 30s — the Wikimedia feed is occasionally slow to first byte.
		HTTP: &http.Client{Timeout: 30 * time.Second},
		rng:  rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0xB00C5)),
	}
}

func (o *OnThisDay) Name() string { return "wikipedia/onthisday" }

type otdEvent struct {
	Text string `json:"text"`
	Year int    `json:"year"`
}

type otdResponse struct {
	Events []otdEvent `json:"events"`
}

// Fetch returns "<year>|<event text>" for a randomly picked event from
// today's "on this day" feed. The scene surfaces year and prose as
// separate visual elements (big mono accent + body prose); the device's
// always-on header already carries today's date, so the widget no
// longer emits a date label.
func (o *OnThisDay) Fetch(ctx context.Context) (string, error) {
	now := time.Now()
	month := int(now.Month())
	day := now.Day()
	url := fmt.Sprintf(
		"https://api.wikimedia.org/feed/v1/wikipedia/en/onthisday/events/%02d/%02d",
		month, day,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := o.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("onthisday: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("onthisday: http %d", resp.StatusCode)
	}
	var payload otdResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("onthisday decode: %w", err)
	}
	if len(payload.Events) == 0 {
		return "", fmt.Errorf("onthisday: no events for %02d/%02d", month, day)
	}

	o.mu.Lock()
	picked := payload.Events[o.rng.IntN(len(payload.Events))]
	o.mu.Unlock()

	text := strings.TrimSpace(picked.Text)
	return fmt.Sprintf("%d|%s", picked.Year, text), nil
}

var _ widget.Widget = (*OnThisDay)(nil)
