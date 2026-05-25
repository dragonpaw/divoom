package calendar

import (
	"strings"
	"testing"
	"time"
)

func TestParseICSSingleEvent(t *testing.T) {
	ics := `BEGIN:VCALENDAR
BEGIN:VEVENT
SUMMARY:Coffee with Ash
DTSTART:20260601T093000Z
END:VEVENT
END:VCALENDAR
`
	events, err := ParseICS(strings.NewReader(ics))
	if err != nil {
		t.Fatalf("ParseICS: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Summary != "Coffee with Ash" {
		t.Errorf("summary = %q", events[0].Summary)
	}
	if events[0].AllDay {
		t.Errorf("should not be all-day")
	}
	if events[0].Start.UTC().Format(time.RFC3339) != "2026-06-01T09:30:00Z" {
		t.Errorf("start = %v", events[0].Start.UTC())
	}
}

func TestParseICSAllDay(t *testing.T) {
	ics := `BEGIN:VEVENT
SUMMARY:Friday party
DTSTART;VALUE=DATE:20260605
END:VEVENT
`
	events, err := ParseICS(strings.NewReader(ics))
	if err != nil {
		t.Fatalf("ParseICS: %v", err)
	}
	if len(events) != 1 || !events[0].AllDay {
		t.Fatalf("want one all-day event, got %+v", events)
	}
}

func TestParseICSWithTZID(t *testing.T) {
	ics := `BEGIN:VEVENT
SUMMARY:Standup
DTSTART;TZID=America/Los_Angeles:20260601T093000
END:VEVENT
`
	events, err := ParseICS(strings.NewReader(ics))
	if err != nil || len(events) != 1 {
		t.Fatalf("ParseICS: err=%v events=%+v", err, events)
	}
	// 09:30 PT == 16:30Z (in May/June, PDT = UTC-7).
	gotUTC := events[0].Start.UTC().Format("15:04")
	if gotUTC != "16:30" {
		t.Errorf("got UTC time %s, want 16:30", gotUTC)
	}
}

func TestParseICSRecurringTreatedAsSingle(t *testing.T) {
	// We intentionally don't expand RRULE; the DTSTART anchor still
	// produces one Event so the agenda always has something to show.
	ics := `BEGIN:VEVENT
SUMMARY:Weekly sync
DTSTART:20260601T100000Z
RRULE:FREQ=WEEKLY;BYDAY=MO
END:VEVENT
`
	events, _ := ParseICS(strings.NewReader(ics))
	if len(events) != 1 {
		t.Fatalf("recurring should yield 1 anchor event, got %d", len(events))
	}
}

func TestParseICSMalformedSkipped(t *testing.T) {
	ics := `THIS IS NOT ICS
just nonsense
`
	events, err := ParseICS(strings.NewReader(ics))
	if err != nil {
		t.Fatalf("err on garbage input: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("garbage gave %d events", len(events))
	}
}

func TestParseICSEmpty(t *testing.T) {
	events, err := ParseICS(strings.NewReader(""))
	if err != nil || len(events) != 0 {
		t.Errorf("empty: err=%v len=%d", err, len(events))
	}
}

func TestHumanRelative(t *testing.T) {
	now := time.Date(2026, 5, 25, 9, 0, 0, 0, time.Local)
	cases := []struct {
		when   time.Time
		allDay bool
		want   string
	}{
		{now.Add(23 * time.Minute), false, "in 23m"},
		{now.Add(4 * time.Hour), false, "in 4h"},
		{time.Date(2026, 5, 26, 9, 0, 0, 0, time.Local), false, "tomorrow"},
		{time.Date(2026, 5, 28, 9, 0, 0, 0, time.Local), false, "Thu"},
		{time.Date(2026, 6, 15, 9, 0, 0, 0, time.Local), false, "Mon 6/15"},
		{time.Date(2026, 5, 26, 0, 0, 0, 0, time.Local), true, "tomorrow"},
	}
	for _, c := range cases {
		got := humanRelative(c.when, now, c.allDay)
		if got != c.want {
			t.Errorf("humanRelative(%s, allDay=%v) = %q, want %q",
				c.when.Format(time.RFC3339), c.allDay, got, c.want)
		}
	}
}

func TestFormatAgendaSingleEvent(t *testing.T) {
	now := time.Date(2026, 5, 25, 9, 0, 0, 0, time.Local)
	events := []Event{
		{Summary: "Coffee", Start: now.Add(30 * time.Minute)},
	}
	got := FormatAgenda(events, now)
	want := "Coffee|in 30m|09:30|||"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestFormatAgendaTwoEvents(t *testing.T) {
	now := time.Date(2026, 5, 25, 9, 0, 0, 0, time.Local)
	events := []Event{
		{Summary: "Later", Start: now.Add(3 * time.Hour)},
		{Summary: "Soon", Start: now.Add(30 * time.Minute)},
	}
	got := FormatAgenda(events, now)
	if !strings.HasPrefix(got, "Soon|in 30m|09:30|Later|in 3h|") {
		t.Errorf("agenda not sorted; got %q", got)
	}
}

func TestFormatAgendaPastEventsExcluded(t *testing.T) {
	now := time.Date(2026, 5, 25, 9, 0, 0, 0, time.Local)
	events := []Event{
		{Summary: "Past", Start: now.Add(-time.Hour)},
		{Summary: "Future", Start: now.Add(time.Hour)},
	}
	got := FormatAgenda(events, now)
	if !strings.HasPrefix(got, "Future|") {
		t.Errorf("past should be filtered; got %q", got)
	}
}

func TestFormatAgendaEmpty(t *testing.T) {
	if got := FormatAgenda(nil, time.Now()); got != "" {
		t.Errorf("empty list should yield empty string, got %q", got)
	}
}
