package main

import "testing"

// TestWeatherConditionOrHazard: hazard text wins over the outlook word,
// otherwise the outlook word renders in its outlook colour.
func TestWeatherConditionOrHazard(t *testing.T) {
	cases := []struct {
		name, raw, wantText, wantColor string
	}{
		{"plain clear", "63°F|clear||45|62|30", "clear", cYellow},
		{"plain rain", "55°F|rain||10|80|90", "rain", cBlue},
		{"hazard wins over outlook", "78°F|hazard|Red Flag Warning|45|62|30", "Red Flag Warning", cRed},
		{"empty outlook (no pipes)", "63°F", "", cFg},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text, color := weatherConditionOrHazard(tc.raw)
			if text != tc.wantText || color != tc.wantColor {
				t.Errorf("weatherConditionOrHazard(%q) = (%q,%q), want (%q,%q)",
					tc.raw, text, color, tc.wantText, tc.wantColor)
			}
		})
	}
}

// TestWeatherAQI exercises the EPA-band colour table plus the missing /
// non-numeric edges. Each band boundary is checked at both its lower
// and upper inclusive limits.
func TestWeatherAQI(t *testing.T) {
	cases := []struct {
		name, raw, wantText, wantColor string
	}{
		{"blank → em-dash dim", "63°F|clear||||", "—", cFgDark},
		{"0 → good", "63°F|clear||0||", "0", cGreen},
		{"50 → good (upper edge)", "63°F|clear||50||", "50", cGreen},
		{"51 → moderate", "63°F|clear||51||", "51", cYellow},
		{"100 → moderate (upper edge)", "63°F|clear||100||", "100", cYellow},
		{"101 → USG", "63°F|clear||101||", "101", cOrange},
		{"150 → USG (upper edge)", "63°F|clear||150||", "150", cOrange},
		{"151 → unhealthy", "63°F|clear||151||", "151", cRed},
		{"200 → unhealthy (upper edge)", "63°F|clear||200||", "200", cRed},
		{"201 → very unhealthy", "63°F|clear||201||", "201", cPurple},
		{"300 → very unhealthy (upper edge)", "63°F|clear||300||", "300", cPurple},
		{"301 → hazardous", "63°F|clear||301||", "301", cRed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text, color := weatherAQI(tc.raw)
			if text != tc.wantText || color != tc.wantColor {
				t.Errorf("weatherAQI(%q) = (%q,%q), want (%q,%q)",
					tc.raw, text, color, tc.wantText, tc.wantColor)
			}
		})
	}
}

// TestWeatherHumidityAndRain covers the present + blank cases for the
// two simple percentage formatters.
func TestWeatherHumidityAndRain(t *testing.T) {
	if text, color := weatherHumidity("63°F|clear||45|62|30"); text != "62%" || color != cBlue {
		t.Errorf("weatherHumidity present = (%q,%q), want (62%%,%q)", text, color, cBlue)
	}
	if text, color := weatherHumidity("63°F|clear||45||30"); text != "—" || color != cFgDark {
		t.Errorf("weatherHumidity blank = (%q,%q), want (—,%q)", text, color, cFgDark)
	}
	if text, color := weatherRain("63°F|clear||45|62|30"); text != "30%" || color != cAqua {
		t.Errorf("weatherRain present = (%q,%q), want (30%%,%q)", text, color, cAqua)
	}
	if text, color := weatherRain("63°F|clear||45|62|"); text != "—" || color != cFgDark {
		t.Errorf("weatherRain blank = (%q,%q), want (—,%q)", text, color, cFgDark)
	}
}
