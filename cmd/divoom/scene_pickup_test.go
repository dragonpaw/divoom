package main

import (
	"reflect"
	"testing"
	"time"
)

func TestParsePickupSchedule(t *testing.T) {
	got := parsePickupSchedule("trash:mon, recycle:wed,compost:wed,junk")
	want := []PickupRule{
		{Type: "trash", Day: time.Monday},
		{Type: "recycle", Day: time.Wednesday},
		{Type: "compost", Day: time.Wednesday},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParsePickupScheduleEmpty(t *testing.T) {
	if r := parsePickupSchedule(""); r != nil {
		t.Errorf("empty should return nil, got %+v", r)
	}
}

func TestPickupActive(t *testing.T) {
	rules := []PickupRule{
		{Type: "trash", Day: time.Wednesday},
		{Type: "recycle", Day: time.Wednesday},
		{Type: "compost", Day: time.Saturday},
	}
	// Times use time.Local so date arithmetic matches the daemon's
	// view; the active-window check reads now.In(time.Local).
	cases := []struct {
		name       string
		now        time.Time
		wantOK     bool
		wantPrefix string
		wantTypes  []string
	}{
		{
			name: "tue 4pm — outside window",
			now:  time.Date(2026, 5, 26, 16, 0, 0, 0, time.Local),
			wantOK: false,
		},
		{
			name: "tue 5pm — evening before wed pickups",
			now:  time.Date(2026, 5, 26, 17, 0, 0, 0, time.Local),
			wantOK: true, wantPrefix: "TOMORROW", wantTypes: []string{"trash", "recycle"},
		},
		{
			name: "wed 7am — morning of",
			now:  time.Date(2026, 5, 27, 7, 0, 0, 0, time.Local),
			wantOK: true, wantPrefix: "TODAY", wantTypes: []string{"trash", "recycle"},
		},
		{
			name: "wed 8am exactly — window closes",
			now:  time.Date(2026, 5, 27, 8, 0, 0, 0, time.Local),
			wantOK: false,
		},
		{
			name: "fri 10pm — evening before sat compost",
			now:  time.Date(2026, 5, 29, 22, 0, 0, 0, time.Local),
			wantOK: true, wantPrefix: "TOMORROW", wantTypes: []string{"compost"},
		},
		{
			name: "sun noon — nothing scheduled",
			now:  time.Date(2026, 5, 31, 12, 0, 0, 0, time.Local),
			wantOK: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prefix, types, _, ok := pickupActive(rules, c.now)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v (prefix=%q types=%v)", ok, c.wantOK, prefix, types)
			}
			if !ok {
				return
			}
			if prefix != c.wantPrefix {
				t.Errorf("prefix = %q, want %q", prefix, c.wantPrefix)
			}
			if !reflect.DeepEqual(types, c.wantTypes) {
				t.Errorf("types = %v, want %v", types, c.wantTypes)
			}
		})
	}
}
