# ISS scene polish

The ISS scene currently shows just the position (lat/lon), next pass,
and a coarse region hint. Two improvements:

## 1. Add a header

Without a label, "−48.2°, 84.4°" reads as cryptic. Add a small dim
header above the body: `"ISS overhead"` or `"ISS position"`. Same
pattern as sunrise's "Today" or weather's outlook word.

Element shape: y=480, FontSize 28, cFgDark, fontProseLight, baked
`TextMessage`.

This bumps the scene's element count from 6 → 7. That's already
the most-crowded count (dayofyear, B5, ST, Discworld, Jargon,
zenquotes, devil, sunrise, weather/with-hazard) — accept the
same-count rotation blocks; ISS rotates with everything else fine.

## 2. Closest-city lookup

Instead of the coarse `"over Pacific"` hint, surface the closest
major city when the ISS is near one. Keep the ocean/continent
fallback when no city is close (e.g. mid-Pacific).

**Approach A — embedded city list**: bake a `[]struct{Name string;
Lat, Lon float64}` of ~200-500 of the world's largest cities into
the widget. Iterate, compute great-circle distance, pick closest if
within ~500km (otherwise fall back to the broad-band region hint).

- Pros: no extra HTTP call, no auth, deterministic.
- Cons: stale list eventually; binary grows by ~10KB.
- City sources: GeoNames `cities500.zip` (CC-BY, top 500k cities, way
  too many — filter to population > 500k or 1M). Or the smaller
  curated lists in the OpenStreetMap project.

**Approach B — reverse-geocode API**: hit OSM Nominatim with the
lat/lon and pluck the nearest city from the JSON.

- Endpoint: `https://nominatim.openstreetmap.org/reverse?lat=X&lon=Y&zoom=10`
- Free, no key, but rate-limited to 1 req/sec and requires a
  meaningful User-Agent. The scene refreshes every minute, so
  rate is fine. UA pattern matches HN / NWS.
- Pros: always current, accurate.
- Cons: third API call per ISS refresh; can fail in mid-Pacific.

Recommend **Approach A** — the embedded list is stdlib-only,
deterministic, and the data update cadence (years) is fine for a
city list. Use Approach B as a fallback if Approach A's lookup
misses (e.g. small city not in the list but Nominatim knows it).

Output format change: extend the widget output to include the
closest city in segment 2 — e.g. `"<lat>°, <lon>°|<next-pass>|over
<city or region>"`. The scene's third body element already mounts
`pipeAt(2)`, so just the text content changes.

## Sample output after both fixes

```
ISS overhead
-22.5°, -45.3°
next pass in 1h 4m
over São Paulo
```

vs current:
```
-22.5°, -45.3°
next pass in 1h 4m
over South America
```
