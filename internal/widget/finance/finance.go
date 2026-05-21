// Package finance is a Widget that reads a ticker's recent closes from
// Yahoo Finance's unofficial chart API and renders week + month percentage
// changes. Calibrated for long-term investors who don't want intraday noise.
package finance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Ticker fetches a single symbol's daily closes and renders weekly + monthly
// percentage changes. The trading-day counts (5 and 21) are conventional
// proxies for "1 calendar week" and "1 calendar month" of market activity.
type Ticker struct {
	Symbol string
	HTTP   *http.Client
}

func NewTicker(symbol string) *Ticker {
	return &Ticker{
		Symbol: symbol,
		HTTP:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (t *Ticker) Name() string { return "finance/" + t.Symbol }

type yahooResp struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				Currency           string  `json:"currency"`
			} `json:"meta"`
			Indicators struct {
				Quote []struct {
					// *float64 lets us distinguish null (missing close) from
					// an actual 0.0 — Yahoo emits null for non-trading days
					// and for the in-progress trading day before close.
					Close []*float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error any `json:"error"`
	} `json:"chart"`
}

func (t *Ticker) Fetch(ctx context.Context) (string, error) {
	// 3-month range gives ~63 trading days — more than enough for the 21-day
	// (1-month) lookback, with margin for holidays.
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=3mo", t.Symbol)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	// Yahoo will sometimes 401 a "default" UA; identify ourselves politely.
	req.Header.Set("User-Agent", "divoom-dashboard/0.1 (github.com/dragonpaw/divoom)")

	resp, err := t.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("yahoo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("yahoo http %d", resp.StatusCode)
	}

	var body yahooResp
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}
	if body.Chart.Error != nil {
		return "", fmt.Errorf("yahoo error: %v", body.Chart.Error)
	}
	if len(body.Chart.Result) == 0 || len(body.Chart.Result[0].Indicators.Quote) == 0 {
		return "", fmt.Errorf("yahoo: no chart data for %s", t.Symbol)
	}

	rawCloses := body.Chart.Result[0].Indicators.Quote[0].Close
	closes := make([]float64, 0, len(rawCloses))
	for _, c := range rawCloses {
		if c != nil {
			closes = append(closes, *c)
		}
	}
	if len(closes) < 22 {
		return "", fmt.Errorf("yahoo: insufficient history for %s (%d closes)", t.Symbol, len(closes))
	}

	const (
		oneWeek  = 5  // trading days
		oneMonth = 21 // trading days
	)
	latest := closes[len(closes)-1]
	weekBack := closes[len(closes)-1-oneWeek]
	monthBack := closes[len(closes)-1-oneMonth]

	weekPct := (latest - weekBack) / weekBack * 100
	monthPct := (latest - monthBack) / monthBack * 100

	return fmt.Sprintf("%s  %+.1f%% 1W   %+.1f%% 1M", t.Symbol, weekPct, monthPct), nil
}
