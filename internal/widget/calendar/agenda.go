// Package calendar (continued) — Agenda widget pulls the next 1-2
// upcoming events from a public iCalendar (ICS) feed and emits a
// pipe-separated string for the agenda scene to slice into elements.
//
// Output shape (six fields, empty trailing triplet when only one event
// is upcoming):
//
//	"NEXT_SUMMARY|NEXT_RELATIVE|NEXT_TIME|UPCOMING_SUMMARY|UPCOMING_RELATIVE|UPCOMING_TIME"
//
// RELATIVE is a humanised offset like "in 23m", "in 4h", "tomorrow",
// "Thu"; TIME is the local clock time "HH:MM" (or "all-day" for
// date-only events).
//
// We do NOT expand RRULE recurrences — a recurring event whose first
// instance is in the past silently won't appear; users can publish
// non-recurring exports if that matters. See the scene spec for
// rationale (v1 simplicity over recurrence correctness).
package calendar

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dragonpaw/divoom/internal/widget"
)

// AgendaCacheTTL controls how often the ICS feed is re-fetched. Five
// minutes is small enough that "starts in 5m" reads as live, large
// enough not to hammer the publisher.
const AgendaCacheTTL = 5 * time.Minute

// Agenda fetches an ICS calendar at URL on demand (with TTL caching)
// and surfaces the next 1-2 upcoming events.
type Agenda struct {
	URL  string
	HTTP *http.Client

	mu       sync.Mutex
	cached   string
	fetchedAt time.Time
	// now is overridable for tests so the "upcoming" filter is
	// reproducible. Production: nil → time.Now().
	nowFn func() time.Time
}

// NewAgenda returns a configured Agenda widget pointed at url. A blank
// url makes Fetch always return "" — the scene wiring uses that as the
// "disabled" sentinel.
func NewAgenda(url string) *Agenda {
	return &Agenda{
		URL:  url,
		HTTP: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *Agenda) Name() string { return "calendar/agenda" }

func (a *Agenda) now() time.Time {
	if a.nowFn != nil {
		return a.nowFn()
	}
	return time.Now()
}

// Fetch returns the formatted agenda string. Cached for AgendaCacheTTL;
// network errors propagate so the scene goes unhealthy and the picker
// skips it until recovery.
func (a *Agenda) Fetch(ctx context.Context) (string, error) {
	if strings.TrimSpace(a.URL) == "" {
		return "", nil
	}
	a.mu.Lock()
	if !a.fetchedAt.IsZero() && a.now().Sub(a.fetchedAt) < AgendaCacheTTL {
		out := a.cached
		a.mu.Unlock()
		return out, nil
	}
	a.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.URL, nil)
	if err != nil {
		return "", fmt.Errorf("agenda: build request: %w", err)
	}
	resp, err := a.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("agenda: http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("agenda: http %d", resp.StatusCode)
	}
	events, err := ParseICS(resp.Body)
	if err != nil {
		return "", fmt.Errorf("agenda: parse: %w", err)
	}
	out := FormatAgenda(events, a.now())
	a.mu.Lock()
	a.cached = out
	a.fetchedAt = a.now()
	a.mu.Unlock()
	return out, nil
}

var _ widget.Widget = (*Agenda)(nil)

// Event is one parsed VEVENT — enough fields to format the agenda row.
type Event struct {
	Summary string
	Start   time.Time
	// AllDay flags a DTSTART;VALUE=DATE event. Start is set to local
	// 00:00 on that date.
	AllDay bool
}

// ParseICS extracts VEVENT blocks from an ICS stream. Each event needs
// at minimum a DTSTART; SUMMARY defaults to "(no title)" if missing.
// RRULE is ignored — see package docstring. Unfolds RFC 5545 line
// continuations (a leading space/tab on the next line is a soft join).
func ParseICS(r io.Reader) ([]Event, error) {
	lines, err := unfoldLines(r)
	if err != nil {
		return nil, err
	}
	var (
		events []Event
		cur    Event
		inEvt  bool
	)
	for _, line := range lines {
		switch {
		case line == "BEGIN:VEVENT":
			inEvt = true
			cur = Event{}
		case line == "END:VEVENT":
			if inEvt {
				if !cur.Start.IsZero() {
					if cur.Summary == "" {
						cur.Summary = "(no title)"
					}
					events = append(events, cur)
				}
				inEvt = false
			}
		case inEvt:
			key, params, value := splitICSLine(line)
			switch strings.ToUpper(key) {
			case "SUMMARY":
				cur.Summary = unescapeICS(value)
			case "DTSTART":
				t, allDay, ok := parseICSTime(value, params)
				if ok {
					cur.Start = t
					cur.AllDay = allDay
				}
			}
		}
	}
	return events, nil
}

// unfoldLines reads an ICS stream and joins RFC 5545 folded
// continuation lines (lines beginning with a single space or tab are
// appended to the previous line, sans the leading whitespace byte).
func unfoldLines(r io.Reader) ([]string, error) {
	var out []string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if line == "" {
			continue
		}
		if (line[0] == ' ' || line[0] == '\t') && len(out) > 0 {
			out[len(out)-1] += line[1:]
			continue
		}
		out = append(out, line)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// splitICSLine splits "KEY;PARAM=VAL;...:VALUE" into (key, params, value).
// The params map's keys are uppercased for case-insensitive lookup.
func splitICSLine(line string) (key string, params map[string]string, value string) {
	colon := strings.IndexByte(line, ':')
	if colon < 0 {
		return line, nil, ""
	}
	head := line[:colon]
	value = line[colon+1:]
	parts := strings.Split(head, ";")
	key = parts[0]
	if len(parts) > 1 {
		params = make(map[string]string, len(parts)-1)
		for _, p := range parts[1:] {
			if eq := strings.IndexByte(p, '='); eq > 0 {
				params[strings.ToUpper(p[:eq])] = p[eq+1:]
			}
		}
	}
	return key, params, value
}

// unescapeICS reverses RFC 5545 TEXT escapes: \\n, \\N, \\,, \\;, \\\\.
func unescapeICS(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n', 'N':
				b.WriteByte(' ')
			case ',':
				b.WriteByte(',')
			case ';':
				b.WriteByte(';')
			case '\\':
				b.WriteByte('\\')
			default:
				b.WriteByte(s[i+1])
			}
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// parseICSTime parses a DTSTART value with optional TZID/VALUE params.
// Returns (t, allDay, ok). Forms handled:
//
//	20260601T093000Z              UTC
//	20260601T093000               floating (treat as local)
//	;TZID=America/Los_Angeles:20260601T093000
//	;VALUE=DATE:20260601          all-day; t = midnight local on that day
func parseICSTime(value string, params map[string]string) (time.Time, bool, bool) {
	value = strings.TrimSpace(value)
	if v, ok := params["VALUE"]; ok && strings.EqualFold(v, "DATE") {
		t, err := time.ParseInLocation("20060102", value, time.Local)
		if err != nil {
			return time.Time{}, false, false
		}
		return t, true, true
	}
	if strings.HasSuffix(value, "Z") {
		t, err := time.Parse("20060102T150405Z", value)
		if err != nil {
			return time.Time{}, false, false
		}
		return t.In(time.Local), false, true
	}
	loc := time.Local
	if tzid, ok := params["TZID"]; ok {
		if l, err := time.LoadLocation(tzid); err == nil {
			loc = l
		}
	}
	t, err := time.ParseInLocation("20060102T150405", value, loc)
	if err != nil {
		// Some publishers omit seconds.
		t, err = time.ParseInLocation("20060102T1504", value, loc)
		if err != nil {
			return time.Time{}, false, false
		}
	}
	return t, false, true
}

// FormatAgenda selects up to 2 upcoming events from `events` (relative
// to `now`), in start-time order, and returns the pipe-separated row.
// Empty when no upcoming events.
func FormatAgenda(events []Event, now time.Time) string {
	upcoming := make([]Event, 0, len(events))
	for _, e := range events {
		if e.Start.After(now) || (e.AllDay && sameLocalDay(e.Start, now)) {
			upcoming = append(upcoming, e)
		}
	}
	sort.SliceStable(upcoming, func(i, j int) bool {
		return upcoming[i].Start.Before(upcoming[j].Start)
	})
	if len(upcoming) == 0 {
		return ""
	}
	fmt1 := func(e Event) (sum, rel, clock string) {
		sum = e.Summary
		rel = humanRelative(e.Start, now, e.AllDay)
		if e.AllDay {
			clock = "all-day"
		} else {
			clock = e.Start.In(time.Local).Format("15:04")
		}
		return
	}
	s1, r1, t1 := fmt1(upcoming[0])
	if len(upcoming) == 1 {
		return s1 + "|" + r1 + "|" + t1 + "|||"
	}
	s2, r2, t2 := fmt1(upcoming[1])
	return s1 + "|" + r1 + "|" + t1 + "|" + s2 + "|" + r2 + "|" + t2
}

func sameLocalDay(a, b time.Time) bool {
	al := a.In(time.Local)
	bl := b.In(time.Local)
	return al.Year() == bl.Year() && al.YearDay() == bl.YearDay()
}

// humanRelative renders the offset from now → when as a short, human
// phrase. Ladder:
//
//	< 0           "now"
//	< 60m         "in Nm"
//	< 24h         "in Nh" (rounded down)
//	tomorrow      "tomorrow"
//	this week     short weekday ("Thu")
//	later         "Mon 6/2"
//
// All-day events skip the "in Nm/h" rungs and go straight to day-grain.
func humanRelative(when, now time.Time, allDay bool) string {
	d := when.Sub(now)
	if d < 0 {
		return "now"
	}
	if !allDay {
		if d < time.Hour {
			m := int(d / time.Minute)
			if m < 1 {
				m = 1
			}
			return fmt.Sprintf("in %dm", m)
		}
		if d < 24*time.Hour {
			h := int(d / time.Hour)
			return fmt.Sprintf("in %dh", h)
		}
	}
	wL := when.In(time.Local)
	nL := now.In(time.Local)
	dayDiff := dayIndex(wL) - dayIndex(nL)
	switch {
	case dayDiff == 0:
		return "today"
	case dayDiff == 1:
		return "tomorrow"
	case dayDiff < 7:
		return wL.Weekday().String()[:3]
	default:
		return wL.Format("Mon 1/2")
	}
}

// dayIndex returns a monotonically increasing day count in local time so
// dayIndex(b) - dayIndex(a) gives the number of calendar days between
// two moments without DST quirks.
func dayIndex(t time.Time) int {
	t = t.In(time.Local)
	return int(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local).Unix() / 86400)
}
