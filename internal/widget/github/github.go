// Package github fetches a small "today on GitHub" activity summary for a
// configured user: how many commits they made today, how long their
// current daily-contribution streak is, and how many of their open PRs
// are still open. Emits a single pipe-separated string
// "<today_commits>|<streak_days>|<open_prs>" that the github scene splits
// across three text rows.
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
	restSearchCommits = "https://api.github.com/search/commits"
	restSearchIssues  = "https://api.github.com/search/issues"
	graphqlEndpoint   = "https://api.github.com/graphql"
	userAgent         = "divoom-dashboard/0.1 (github.com/dragonpaw/divoom)"

	// Cache TTL keeps API usage well under the 5000 req/hr authed limit:
	// the scene driver fetches on every 5-min refresh cycle, so a 5-min
	// cache means at most one Fetch per cycle even if a future caller
	// hits the widget more aggressively.
	cacheTTL = 5 * time.Minute
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

// Fetch returns "<today_commits>|<streak_days>|<open_prs>". Results are
// cached for cacheTTL; concurrent callers wait on the same mutex so we
// only ever have one in-flight Fetch at a time per widget instance.
func (w *Widget) Fetch(ctx context.Context) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if time.Since(w.lastFetch) < cacheTTL && w.lastFetch != (time.Time{}) {
		return w.cached, w.cachedErr
	}

	commits, cerr := w.todayCommits(ctx)
	streak, serr := w.currentStreak(ctx)
	prs, perr := w.openPRs(ctx)

	// If every call failed, surface the first error so the driver can log
	// it and leave the previous cached value on screen. If at least one
	// succeeded, render what we have (zeros for the failed segments).
	if cerr != nil && serr != nil && perr != nil {
		w.lastFetch = time.Now()
		w.cached = ""
		w.cachedErr = fmt.Errorf("github: %w", cerr)
		return w.cached, w.cachedErr
	}

	out := fmt.Sprintf("%d|%d|%d", commits, streak, prs)
	w.lastFetch = time.Now()
	w.cached = out
	w.cachedErr = nil
	return out, nil
}

// todayCommits returns the number of commits authored by the configured
// user since 00:00 UTC today, via the REST /search/commits endpoint.
func (w *Widget) todayCommits(ctx context.Context) (int, error) {
	// Use UTC so the day boundary matches GitHub's own search semantics.
	since := time.Now().UTC().Format("2006-01-02")
	q := fmt.Sprintf("author:%s author-date:>=%s", w.User, since)
	url := restSearchCommits + "?per_page=1&q=" + queryEscape(q)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	w.setAuth(req)
	// Commit search needs the cloak-preview Accept header on older API
	// versions; harmless on current ones.
	req.Header.Set("Accept", "application/vnd.github.cloak-preview+json")
	var body struct {
		TotalCount int `json:"total_count"`
	}
	if err := w.do(req, &body); err != nil {
		return 0, fmt.Errorf("today commits: %w", err)
	}
	return body.TotalCount, nil
}

// openPRs returns the count of open pull requests authored by the user
// across all repositories, via the REST /search/issues endpoint.
func (w *Widget) openPRs(ctx context.Context) (int, error) {
	q := fmt.Sprintf("is:open author:%s type:pr", w.User)
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
		return 0, fmt.Errorf("open prs: %w", err)
	}
	return body.TotalCount, nil
}

// currentStreak walks the GraphQL contribution calendar from today
// backward and counts consecutive days with at least one contribution.
// Today counts when it has contributions; if today has zero we step
// straight back to yesterday so the streak doesn't reset until a full
// missed day has passed.
func (w *Widget) currentStreak(ctx context.Context) (int, error) {
	const query = `query($login: String!) {
  user(login: $login) {
    contributionsCollection {
      contributionCalendar {
        weeks {
          contributionDays {
            date
            contributionCount
          }
        }
      }
    }
  }
}`
	payload := map[string]any{
		"query":     query,
		"variables": map[string]any{"login": w.User},
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlEndpoint, bytes.NewReader(buf))
	if err != nil {
		return 0, err
	}
	w.setAuth(req)
	req.Header.Set("Content-Type", "application/json")

	var body struct {
		Data struct {
			User struct {
				ContributionsCollection struct {
					ContributionCalendar struct {
						Weeks []struct {
							ContributionDays []struct {
								Date              string `json:"date"`
								ContributionCount int    `json:"contributionCount"`
							} `json:"contributionDays"`
						} `json:"weeks"`
					} `json:"contributionCalendar"`
				} `json:"contributionsCollection"`
			} `json:"user"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := w.do(req, &body); err != nil {
		return 0, fmt.Errorf("streak: %w", err)
	}
	if len(body.Errors) > 0 {
		return 0, fmt.Errorf("streak: graphql: %s", body.Errors[0].Message)
	}

	// Flatten weeks into a single date-ordered slice of (date, count).
	type day struct {
		date  string
		count int
	}
	var days []day
	for _, wk := range body.Data.User.ContributionsCollection.ContributionCalendar.Weeks {
		for _, d := range wk.ContributionDays {
			days = append(days, day{d.Date, d.ContributionCount})
		}
	}
	if len(days) == 0 {
		return 0, nil
	}

	// Walk from the latest day backward. The calendar may include future
	// days for the current week with count=0 — skip those at the head so
	// the count starts from today (or the last completed day).
	today := time.Now().UTC().Format("2006-01-02")
	i := len(days) - 1
	for i >= 0 && days[i].date > today {
		i--
	}
	// If today has zero contributions, allow the streak to start from
	// yesterday — a day in progress doesn't break the streak.
	if i >= 0 && days[i].date == today && days[i].count == 0 {
		i--
	}
	streak := 0
	for ; i >= 0; i-- {
		if days[i].count == 0 {
			break
		}
		streak++
	}
	return streak, nil
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
