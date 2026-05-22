package sky

import "testing"

func TestLocationFor(t *testing.T) {
	cases := []struct {
		name     string
		lat, lon float64
		want     string
	}{
		// City hits: confirm exact-coord lookups land on the expected
		// entry from the embedded table.
		{"sao paulo", -23.55, -46.63, "over São Paulo, Brazil"},
		{"tokyo", 35.69, 139.69, "over Tokyo, Japan"},
		// Mongolia interior. Ulaanbaatar is in the table at ~170 km,
		// well inside the 600 km radius. This confirms central Asia
		// is covered, not just coastal regions.
		{"mongolia near ulaanbaatar", 47, 105, "over Ulaanbaatar, Mongolia"},
		// Open-water points chosen to be > 600 km from any city in
		// the embedded table, exercising the sea/ocean fallbacks.
		{"mid pacific", 0, -150, "over Pacific"},
		{"south atlantic", -30, -20, "over Atlantic"},
		{"south indian", -40, 80, "over Indian Ocean"},
		{"high arctic", 80, 0, "over Arctic Ocean"},
		{"southern ocean", -70, 0, "over Southern Ocean"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := locationFor(c.lat, c.lon)
			if got != c.want {
				t.Errorf("locationFor(%v,%v) = %q, want %q",
					c.lat, c.lon, got, c.want)
			}
		})
	}
}

// TestNamedSea exercises the sea-box table directly because the
// city-first preference in locationFor means most named-sea points are
// shadowed by a nearby coastal city.
func TestNamedSea(t *testing.T) {
	cases := []struct {
		name     string
		lat, lon float64
		want     string
	}{
		{"mediterranean", 38, 18, "Mediterranean"},
		{"north sea", 56, 3, "North Sea"},
		{"red sea", 22, 38, "Red Sea"},
		{"caribbean", 15, -75, "Caribbean"},
		{"baltic", 58, 20, "Baltic Sea"},
		{"black sea", 43, 35, "Black Sea"},
		{"sea of japan", 40, 135, "Sea of Japan"},
		{"open pacific", 0, -150, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := namedSea(c.lat, c.lon)
			if got != c.want {
				t.Errorf("namedSea(%v,%v) = %q, want %q",
					c.lat, c.lon, got, c.want)
			}
		})
	}
}
