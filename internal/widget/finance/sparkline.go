package finance

import "strings"

// sparkBlocks are the eight Unicode block characters used as quantised
// "bars" in a sparkline, low to high. Index 0 is the lowest level
// (▁ one-eighth) and 7 the highest (█ full block).
var sparkBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// sparkline renders a sequence of (possibly missing) prices as a string
// of Unicode block characters. Each non-nil price is quantised into one
// of 8 levels by its position in the (min, max) range; nil entries
// become a literal space so non-trading days read as gaps. A flat series
// (max == min) renders as a row of middle blocks (▄) so the bar is
// visible without implying movement.
func sparkline(prices []*float64) string {
	if len(prices) == 0 {
		return ""
	}
	// Find min/max over the real (non-nil) values only.
	first := true
	var min, max float64
	for _, p := range prices {
		if p == nil {
			continue
		}
		v := *p
		if first {
			min, max = v, v
			first = false
			continue
		}
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	if first {
		// Entirely nil — render all gaps.
		return strings.Repeat(" ", len(prices))
	}
	flat := max == min
	span := max - min

	var b strings.Builder
	b.Grow(len(prices) * 3) // each block rune is 3 bytes in UTF-8
	for _, p := range prices {
		if p == nil {
			b.WriteByte(' ')
			continue
		}
		if flat {
			b.WriteRune('▄')
			continue
		}
		// Quantise into [0, 7]. (v - min) / span lands in [0, 1];
		// multiply by 7 and round to nearest to land on an index.
		level := int((*p-min)/span*7.0 + 0.5)
		if level < 0 {
			level = 0
		} else if level > 7 {
			level = 7
		}
		b.WriteRune(sparkBlocks[level])
	}
	return b.String()
}
