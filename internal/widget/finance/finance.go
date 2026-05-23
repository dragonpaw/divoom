// Package finance is a Widget that reads a ticker's recent closes from
// Yahoo Finance's unofficial chart API and renders a trading-terminal
// readout: symbol, latest price, weekly + monthly percent change, a
// sparkline of the last ~35 trading days, and the close date.
package finance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dragonpaw/divoom/internal/widget"
)

// Ticker fetches a single symbol's daily closes and renders weekly +
// monthly percentage changes alongside a sparkline. The trading-day
// counts (5 and 21) are conventional proxies for "1 calendar week" and
// "1 calendar month" of market activity.
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
			Timestamp  []int64 `json:"timestamp"`
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

// sparkLen is the number of trailing closes the sparkline visualises.
// ~35 trading days ≈ 7 calendar weeks: enough context to show a trend
// without overrunning the body element's render width at 70pt mono.
const sparkLen = 35

func (t *Ticker) Fetch(ctx context.Context) (string, error) {
	// 3-month range gives ~63 trading days — more than enough for the 21-day
	// (1-month) lookback, with margin for the sparkline tail and holidays.
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

	result := body.Chart.Result[0]
	rawCloses := result.Indicators.Quote[0].Close

	// Compacted (nil-stripped) series for the percent maths — those need
	// strictly real closes.
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

	// Price formatting: "$NNN.NN" or "$N,NNN.NN" with the response's
	// currency sign when present (Yahoo emits "USD" for both stocks and
	// crypto pairs like BTC-USD, so the symbol stays "$" in practice).
	currencySym := "$"
	_ = result.Meta.Currency // reserved for future non-USD support; symbol stays $.

	price := formatPrice(latest, currencySym)

	// Sparkline: take the trailing sparkLen entries of the *raw* series
	// (preserving nil for non-trading days as spaces). If the series is
	// shorter than sparkLen, render whatever we have.
	tail := rawCloses
	if len(tail) > sparkLen {
		tail = tail[len(tail)-sparkLen:]
	}
	spark := sparkline(tail)

	// Close date: derived from the trailing timestamp (Yahoo emits one
	// timestamp per close, in Unix seconds, UTC). Fall back to "today" if
	// timestamps are missing for any reason.
	closeDate := time.Now().UTC().Format("2006-01-02")
	if len(result.Timestamp) > 0 {
		ts := result.Timestamp[len(result.Timestamp)-1]
		closeDate = time.Unix(ts, 0).UTC().Format("2006-01-02")
	}

	return strings.Join([]string{
		strings.ToUpper(t.Symbol),
		price,
		fmt.Sprintf("%+.1f", weekPct),
		fmt.Sprintf("%+.1f", monthPct),
		spark,
		closeDate,
	}, "|"), nil
}

// formatPrice renders a price as "$NNN.NN", inserting thousands
// separators for values >= 1000 ("$1,234.56"). Negative prices are not
// expected for any tradable instrument; we still handle them with a
// leading "-" for symmetry with the percent formatters.
func formatPrice(p float64, sym string) string {
	neg := p < 0
	if neg {
		p = -p
	}
	whole := int64(p)
	frac := int64((p-float64(whole))*100 + 0.5)
	// Carry rounding up across the decimal boundary.
	if frac >= 100 {
		whole++
		frac -= 100
	}
	wholeStr := withThousands(whole)
	out := fmt.Sprintf("%s%s.%02d", sym, wholeStr, frac)
	if neg {
		out = "-" + out
	}
	return out
}

// withThousands inserts commas every three digits in n. Cheap and
// allocation-light; the strconv/locale machinery would be overkill for a
// price formatter.
func withThousands(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	// Walk from the right, inserting commas every 3.
	var b strings.Builder
	rem := len(s) % 3
	if rem > 0 {
		b.WriteString(s[:rem])
		if len(s) > rem {
			b.WriteByte(',')
		}
	}
	for i := rem; i < len(s); i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < len(s) {
			b.WriteByte(',')
		}
	}
	return b.String()
}

var _ widget.Widget = (*Ticker)(nil)
