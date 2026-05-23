package finance

import "testing"

// ptr returns a *float64 for a literal — keeps the test tables compact.
func ptr(v float64) *float64 { return &v }

func TestSparkline(t *testing.T) {
	cases := []struct {
		name   string
		prices []*float64
		want   string
	}{
		{
			name:   "empty",
			prices: nil,
			want:   "",
		},
		{
			name:   "all-nil renders all gaps",
			prices: []*float64{nil, nil, nil},
			want:   "   ",
		},
		{
			name:   "flat series renders middle blocks",
			prices: []*float64{ptr(100), ptr(100), ptr(100), ptr(100)},
			want:   "▄▄▄▄",
		},
		{
			name:   "rising series climbs the levels",
			prices: []*float64{ptr(0), ptr(1), ptr(2), ptr(3), ptr(4), ptr(5), ptr(6), ptr(7)},
			want:   "▁▂▃▄▅▆▇█",
		},
		{
			name:   "falling series descends",
			prices: []*float64{ptr(7), ptr(6), ptr(5), ptr(4), ptr(3), ptr(2), ptr(1), ptr(0)},
			want:   "█▇▆▅▄▃▂▁",
		},
		{
			name:   "nil mid-series renders as gap",
			prices: []*float64{ptr(0), nil, ptr(7)},
			want:   "▁ █",
		},
		{
			name:   "extremes anchor min/max",
			prices: []*float64{ptr(10), ptr(20)},
			want:   "▁█",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := sparkline(c.prices)
			if got != c.want {
				t.Errorf("sparkline(%v) = %q, want %q", c.prices, got, c.want)
			}
		})
	}
}
