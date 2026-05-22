# Weather: pick °C vs °F based on locale

The weather widget currently hardcodes `temperature_unit=fahrenheit`
in the Open-Meteo query because our test location (Richmond, CA) is
in the US. Make it locale-correct: degrees Celsius for almost the
entire world, Fahrenheit for the US (and a small handful of holdouts
— the Bahamas, Cayman Islands, Palau, FSM, Marshall Islands, Liberia).

## Where to decide

The widget knows its `lat` / `lon` already. Pick the unit from those
without an API call:

- **Approach A — built-in bounding box check**: a coarse `useFahrenheit(lat, lon) bool`
  helper that returns true if the point falls inside the lower-48
  continental US bounding box (roughly lat 24-49, lon -125 to -66),
  Alaska (lat 51-71, lon -180 to -130), Hawaii (lat 18-23, lon -160
  to -154), Puerto Rico, or the small list above. Anything else
  → Celsius. Stdlib-only, no fetch.
- **Approach B — reverse-geocode lookup**: hit Nominatim once at
  startup with the configured coords, get the ISO country code,
  switch on `country == "US"` (etc.). More accurate edge cases
  (e.g. US embassies, Saipan), but adds a one-time HTTP call and a
  User-Agent dependency.

Lean toward **A** — the bounding-box check is good enough for the
~0.1% of locations near the borders, and stays stdlib-only. The unit
choice is set at widget construction time and never changes.

## Plumbing

- Move the `temperature_unit=fahrenheit` URL parameter out of the
  forecast-fetch URL into a `c.unit` field on the Client struct,
  populated by `useFahrenheit(lat, lon)` in `New(lat, lon)`.
- Both `Fetch` and `LoadThresholds` use the same unit (otherwise the
  climate-calibration thresholds and the live temperature would
  mismatch).
- Update the colour-bucket constants in `weatherTempColor`. Today
  they're °F (50/68/76/80/85). For Celsius locations the comfort
  band 68-75°F becomes ~20-24°C; cold below ~5°C; hot ≥ 30°C. The
  auto-calibration (LoadThresholds via percentiles) already handles
  cold/hot bounds in whichever unit is fetched — only the *fixed
  comfort window* needs a unit-aware constant.
- Render the temperature with the right suffix: the widget emits
  `"<temp>°"` today; switch to `"<temp>°F"` or `"<temp>°C"` so the
  unit is unambiguous on the wall.

## Verify after implementing

- For lat=37.9358 (Richmond) → unit = F, comfort 68-75, suffix °F.
- For lat=51.5074, lon=-0.1278 (London) → unit = C, comfort 20-24,
  suffix °C.
- For lat=-33.8688, lon=151.2093 (Sydney) → unit = C, suffix °C.
- For lat=21.3069, lon=-157.8583 (Honolulu) → unit = F (Hawaii is
  US), suffix °F.
