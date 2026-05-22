# Rename "sky" → "moonphase", add next-full-moon countdown

Two changes to the existing sky scene/widget:

## 1. Rename "sky" to "moonphase"

The scene only ever shows moon data (phase name + illumination
percentage). Rename for clarity:

- Widget: `internal/widget/sky/moon.go` — rename `NewMoon()` /
  the Moon struct as needed, but the package `sky` can stay (it
  already houses sunrise + ISS which are sky-adjacent). Just
  update the `Name()` method return from `"sky/moon"` to
  `"moonphase"` if the widget identity needs to match the new
  scene name in logs.
- Scene name in `cmd/divoom/scenes.go`: change `"sky"` →
  `"moonphase"`.
- `widgets[...]` map key in `serve.go`: `"sky"` → `"moonphase"`.
- Background path constant: `bgSky` → `bgMoonphase` (and the file
  name `/userdata/wallclock_bg_sky.jpg` →
  `/userdata/wallclock_bg_moonphase.jpg`).
- `SceneSky` enum in `internal/render/background.go` →
  `SceneMoonphase`. Update `drawSceneGlyph`'s switch case.
- `cmd/divoom/render.go`: `scene-sky` CLI entry → `scene-moonphase`.
- Scene title: `sceneTitle("moon")` is already in place — keep
  as-is, the title doesn't need to change.

Grep after: zero references to `bgSky`, `SceneSky`, `"sky"` (the
*scene* — `widget/sky` package stays as-is since it groups
sky-adjacent widgets including sunrise/ISS).

## 2. Add next-full-moon countdown / date

The moon scene currently shows the current phase + illumination %.
Add a third body row with the time until the next full moon —
useful timing info that complements the current state.

Computation: the synodic month is ~29.53 days. From the current
moon age (already computed by the widget for the phase name), the
days until next full moon = (14.77 - age) mod 29.53, where age is
days since last new moon.

Render in one of two formats:
- Short relative: `"full moon in 6 days"` or `"full moon in 18 hrs"`.
- Absolute date: `"next full moon: Jun 1"`.

Use relative format for ≤7 days (more visceral), absolute for
longer. Or just always use absolute — your call. The relative
version is more dynamic and re-renders meaningfully every refresh.

## Widget output format

Currently: `"<phase_emoji> · <phase_name> · <illum%>"` (separator " · ").

Add a fourth segment for the next-full-moon string:
`"<phase_emoji> · <phase_name> · <illum%> · <next_full_moon>"`.

Existing `moonPhaseName` and `moonIllum` formatters split on `" · "`
and pick segments 1 and 2; add `moonNextFullMoon` that picks
segment 3. Wire into a new scene element.

## Scene layout impact

Current sky scene: 3 top + 2 body = 5 elements. Adding the
next-full-moon row makes it 3 top + 3 body = 6 elements. That
collides with nasa / cocktail / iss / weather — fine.

```
y≈480-520    sceneTitle("moon")              (idSceneTitle)
y≈540-680    <phase_name> big blue prose     (idSceneMain)
y≈700-820    <illum%> mono                   (idSceneSub1)
y≈840-900    <next_full_moon> small dim      (idSceneSub2)
```

10% margins on all (`StartX: 80, Width: 640`).

## Glyph

`drawSceneGlyph`'s sky case currently draws a crescent (filled
circle minus offset circle). Keep that — still appropriate for a
moonphase scene. No change needed.
