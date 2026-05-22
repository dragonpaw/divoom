// Package facts has the light/whimsy fact widgets — cat facts, useless
// facts. Each returns a single short string ready for a Text element.
package facts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dragonpaw/divoom/internal/widget"
)

const userAgent = "divoom-dashboard/0.1 (github.com/dragonpaw/divoom)"

func defaultHTTP() *http.Client { return &http.Client{Timeout: 15 * time.Second} }

// CatFact returns a random cat fact from catfact.ninja.
type CatFact struct{ HTTP *http.Client }

func NewCatFact() *CatFact { return &CatFact{HTTP: defaultHTTP()} }

func (c *CatFact) Name() string { return "facts/cat" }

func (c *CatFact) Fetch(ctx context.Context) (string, error) {
	var body struct {
		Fact string `json:"fact"`
	}
	if err := getJSON(ctx, c.HTTP, "https://catfact.ninja/fact", &body); err != nil {
		return "", err
	}
	// Output uses HEADER|BODY so the ambient scene can render the label
	// and the fact on separate lines.
	return "cat fact|" + body.Fact, nil
}

// UselessFact returns a random "did you know" fact from uselessfacts.jsph.pl.
type UselessFact struct{ HTTP *http.Client }

func NewUselessFact() *UselessFact { return &UselessFact{HTTP: defaultHTTP()} }

func (u *UselessFact) Name() string { return "facts/useless" }

func (u *UselessFact) Fetch(ctx context.Context) (string, error) {
	var body struct {
		Text string `json:"text"`
	}
	if err := getJSON(ctx, u.HTTP, "https://uselessfacts.jsph.pl/api/v2/facts/random", &body); err != nil {
		return "", err
	}
	return "did you know?|" + body.Text, nil
}

func getJSON(ctx context.Context, h *http.Client, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := h.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

var (
	_ widget.Widget = (*CatFact)(nil)
	_ widget.Widget = (*UselessFact)(nil)
)
