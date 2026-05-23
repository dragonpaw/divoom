// Package github fetches a lifetime-stats summary for the configured
// user: total contributions across every year of the account's
// existence, total PRs authored across all repos, and how many years
// the account has been on GitHub. Emits a single pipe-separated string
// "<lifetime_contributions>|<total_prs>|<years_on_github>" that the
// github scene splits across hero + two small stats rows.
//
// Auth is mandatory: the search and GraphQL endpoints require a token,
// and the unauthenticated REST quota (60 req/hr) is too small for a
// rotation that fetches every few minutes. The widget reads GITHUB_TOKEN
// and GITHUB_USER from the environment; cmd/divoom/serve.go skips
// constructing the widget entirely when either is missing.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	wpkg "github.com/dragonpaw/divoom/internal/widget"
)

const (
	restSearchIssues = "https://api.github.com/search/issues"
	graphqlEndpoint  = "https://api.github.com/graphql"
	userAgent        = "divoom-dashboard/0.1 (github.com/dragonpaw/divoom)"

	// Cache TTL keeps API usage well under the 5000 req/hr authed limit.
	// Lifetime stats change slowly (only new contributions add to the
	// total, never subtract), so a longer cache is fine — refresh once
	// an hour is more than enough granularity for a wall display.
	cacheTTL = 1 * time.Hour

	// githubFoundingYear bounds the alias-per-year lifetime contributions
	// query. Years before the account existed return 0 contributions
	// from the GraphQL API, so it's harmless to query them — and it
	// lets us issue a single GraphQL roundtrip with one alias per year
	// instead of doing a createdAt fetch first.
	githubFoundingYear = 2008
)

// Widget produces the github scene's text. User and Token are captured at
// construction time from the environment; HTTP defaults to a 15s client.
type Widget struct {
	User  string
	Token string
	HTTP  *http.Client

	mu        sync.Mutex
	lastFetch time.Time
	cached    string
	cachedErr error
}

// New returns a *Widget configured for the given user and token. Caller
// is responsible for skipping construction when either is empty (so the
// scene is left out of the rotation cleanly).
func New(user, token string) *Widget {
	return &Widget{
		User:  user,
		Token: token,
		HTTP:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (w *Widget) Name() string { return "github/activity" }

// Fetch returns "<lifetime_contributions>|<total_prs>|<years_on_github>".
// Results are cached for cacheTTL; concurrent callers wait on the same
// mutex so we only ever have one in-flight Fetch at a time per widget
// instance.
func (w *Widget) Fetch(ctx context.Context) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if time.Since(w.lastFetch) < cacheTTL && w.lastFetch != (time.Time{}) {
		return w.cached, w.cachedErr
	}

	contributions, years, lerr := w.lifetimeContributions(ctx)
	prs, perr := w.totalPRs(ctx)

	// If every call failed, surface the first error so the driver can log
	// it and leave the previous cached value on screen. If at least one
	// succeeded, render what we have (zeros for the failed segments).
	if lerr != nil && perr != nil {
		w.lastFetch = time.Now()
		w.cached = ""
		w.cachedErr = fmt.Errorf("github: %w", lerr)
		return w.cached, w.cachedErr
	}

	out := fmt.Sprintf("%d|%d|%d", contributions, prs, years)
	w.lastFetch = time.Now()
	w.cached = out
	w.cachedErr = nil
	return out, nil
}

// totalPRs returns the lifetime count of PRs authored by the user
// (open + merged + closed), via the REST /search/issues endpoint.
func (w *Widget) totalPRs(ctx context.Context) (int, error) {
	q := fmt.Sprintf("author:%s type:pr", w.User)
	url := restSearchIssues + "?per_page=1&q=" + queryEscape(q)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	w.setAuth(req)
	var body struct {
		TotalCount int `json:"total_count"`
	}
	if err := w.do(req, &body); err != nil {
		return 0, fmt.Errorf("total prs: %w", err)
	}
	return body.TotalCount, nil
}

// lifetimeContributions issues one GraphQL request with one alias per
// year from githubFoundingYear to the current year, summing
// contributionCalendar.totalContributions across all aliases. Pre-
// account years return 0 from the API, so no createdAt-first roundtrip
// is needed. Also returns years-on-github computed from user.createdAt
// (year delta — partial years round down).
func (w *Widget) lifetimeContributions(ctx context.Context) (contributions, years int, err error) {
	currentYear := time.Now().UTC().Year()

	// Build the query with one alias per year: y2008, y2009, ..., yYYYY.
	var b strings.Builder
	b.WriteString("query($login: String!) {\n")
	b.WriteString("  user(login: $login) {\n")
	b.WriteString("    createdAt\n")
	for y := githubFoundingYear; y <= currentYear; y++ {
		fmt.Fprintf(&b,
			"    y%d: contributionsCollection(from: \"%d-01-01T00:00:00Z\", to: \"%d-12-31T23:59:59Z\") { contributionCalendar { totalContributions } }\n",
			y, y, y)
	}
	b.WriteString("  }\n")
	b.WriteString("}\n")

	payload := map[string]any{
		"query":     b.String(),
		"variables": map[string]any{"login": w.User},
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return 0, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlEndpoint, bytes.NewReader(buf))
	if err != nil {
		return 0, 0, err
	}
	w.setAuth(req)
	req.Header.Set("Content-Type", "application/json")

	// Generic decode: the per-year aliases are dynamic, so unmarshal into
	// a map and walk the alias keys.
	var body struct {
		Data struct {
			User map[string]json.RawMessage `json:"user"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := w.do(req, &body); err != nil {
		return 0, 0, fmt.Errorf("lifetime: %w", err)
	}
	if len(body.Errors) > 0 {
		return 0, 0, fmt.Errorf("lifetime: graphql: %s", body.Errors[0].Message)
	}

	// Parse createdAt for years-on-github.
	if raw, ok := body.Data.User["createdAt"]; ok {
		var createdAt string
		if err := json.Unmarshal(raw, &createdAt); err == nil && createdAt != "" {
			if t, perr := time.Parse(time.RFC3339, createdAt); perr == nil {
				years = currentYear - t.UTC().Year()
				if years < 0 {
					years = 0
				}
			}
		}
	}

	// Sum totalContributions across every per-year alias.
	type yearBlock struct {
		ContributionCalendar struct {
			TotalContributions int `json:"totalContributions"`
		} `json:"contributionCalendar"`
	}
	for key, raw := range body.Data.User {
		if !strings.HasPrefix(key, "y") {
			continue
		}
		var yb yearBlock
		if err := json.Unmarshal(raw, &yb); err != nil {
			continue
		}
		contributions += yb.ContributionCalendar.TotalContributions
	}
	return contributions, years, nil
}

// setAuth attaches the standard GitHub headers, including the token. The
// User-Agent is required by the API; the Accept header pins to the v3
// JSON variant for REST calls (the commit search Accept override happens
// in the caller for that one endpoint).
func (w *Widget) setAuth(req *http.Request) {
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+w.Token)
}

// do executes the request and decodes the JSON body into out. Non-2xx
// responses are wrapped with the status code so caller logs are useful
// without needing to inspect the body.
func (w *Widget) do(req *http.Request, out any) error {
	resp, err := w.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// queryEscape URL-encodes a search query. Avoids pulling in net/url for
// one call — the search query only contains a small set of ASCII
// punctuation (`:`, `>=`, `-`, space) plus the user name.
func queryEscape(q string) string {
	var b strings.Builder
	for i := 0; i < len(q); i++ {
		c := q[i]
		switch {
		case c == ' ':
			b.WriteByte('+')
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~':
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

var _ wpkg.Widget = (*Widget)(nil)
