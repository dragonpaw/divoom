# ISS scene polish

The ISS scene currently shows just the position (lat/lon), next pass,
and a coarse region hint. Two improvements:

## 1. Add a header

Without a label, "−48.2°, 84.4°" reads as cryptic. Add a small dim
header above the body: `"ISS overhead"` or `"ISS position"`. Same
pattern as sunrise's "Today" or weather's outlook word.

Element shape: y=480, FontSize 28, cFgDark, fontProseLight, baked
`TextMessage`. NOTE: `sceneTitle("ISS overhead")` already exists and
should be used here — drop the bespoke implementation.

This bumps the scene's element count from 6 → 7. That's already
the most-crowded count (dayofyear, B5, ST, Discworld, Jargon,
zenquotes, devil, sunrise, weather/with-hazard) — accept the
same-count rotation blocks; ISS rotates with everything else fine.

## 2. Over-land: nearest city + country. Over-water: ocean/sea name.

The third body row currently shows a coarse region hint
(`"over Pacific"`, `"over South America"`). Make it concrete:

- **If the sub-satellite point is over land** → `"over <City>, <Country>"`
  for the nearest major city. E.g. `"over São Paulo, Brazil"`,
  `"over Tokyo, Japan"`, `"over Reykjavík, Iceland"`. If no city is
  within a reasonable radius (~500-800km), fall back to just the
  country name: `"over Mongolia"`.
- **If over open ocean** → `"over <Ocean>"` (Pacific, Atlantic,
  Indian, Arctic, Southern) computed from lat/lon bands.
- **If over a major sea** → name the sea (Mediterranean, Caribbean,
  North Sea, Sea of Japan, Bering Sea, etc.) — same lat/lon-band
  check.

## Data approach

**Approach A — embedded city + country list (preferred)**: bake a
small `[]struct{Name, Country string; Lat, Lon float64}` table into
the widget. Source: GeoNames `cities1000.zip` (CC-BY) filtered to
cities with population > 500k — that's ~1000 entries, ~50KB of Go
literals. Each lookup: iterate, compute great-circle (haversine)
distance, pick closest within 500km radius. For country fallback
when no city is near, use a separate `[]struct{Country string;
Polygon []LatLon}` table — or just skip the country-only case and
fall back to the ocean/sea name when no city is nearby.

- Pros: stdlib-only, deterministic, no extra HTTP call, fast.
- Cons: binary grows ~50KB; city list goes stale on decade timescale.

**Approach B — reverse-geocode fallback**: when the embedded city
list misses (no city within radius), hit OSM Nominatim:
`https://nominatim.openstreetmap.org/reverse?lat=X&lon=Y&zoom=6` with
the required User-Agent. Returns country / state / etc. so we can
say `"over Mongolia"` for thinly-populated regions.

- Pros: handles long tail (small countries, remote populated areas).
- Cons: extra HTTP per refresh; rate-limited 1 req/sec; UA dance.

**Approach C — ocean / sea bands**: hardcode lat/lon bounding boxes
for the major oceans + seas. Tiny code, no data file. Used only
when neither A nor B finds land.

Recommend **A + C** as the baseline. If users want better long-tail
land coverage later, add **B** as a final fallback before C.

## Output format

Widget output extends to: `"<lat>°, <lon>°|<next-pass>|<location>"`
where `<location>` is one of:

- `"over <City>, <Country>"` — when nearest-major-city < 500km
- `"over <Country>"` — when over land but no major city near (Approach B)
- `"over <Ocean or Sea>"` — when over water

The scene's third body element already mounts `pipeAt(2)`, no shape
change needed.

## Sample output

```
ISS overhead
-22.5°, -45.3°
next pass in 1h 4m
over São Paulo, Brazil
```

or

```
ISS overhead
14.8°, -160.2°
next pass in 47m
over Pacific
```

or

```
ISS overhead
47.1°, 105.3°
next pass in 23m
over Mongolia
```
