# Split scenes.go into per-scene files?

`cmd/divoom/scenes.go` has grown to ~1300 lines and now holds 19+
scene blocks plus all the shared helpers (`QuoteScene`,
`DictionaryScene`, `sceneTitle`, all the format functions, the
weather/color machinery, etc.). Worth considering whether each scene
should live in its own file.

## Pros of splitting

- Easier to find a specific scene's definition without scrolling.
- Smaller diffs per change — adding/removing one scene touches one
  file instead of buried in a big one.
- Concurrent agents stomp on each other less (a recurring pain in
  this codebase — most agents have to Read/Edit `scenes.go`).
- Reviewing a single scene's PR doesn't require pulling in the
  whole 1300-line context.

## Cons / friction

- The shared helpers (sceneTitle, QuoteScene, DictionaryScene,
  pipeAt, vCenter\*, weatherTempColor, etc.) all need to live
  somewhere central — probably a `scenes_helpers.go` or stay in a
  `scenes.go` skeleton. So you don't fully escape the central file;
  it just shrinks.
- The element-ID constants (idDay, idTime, idSceneMain, etc.) and
  the font/color constants would also stay in the central file.
- `buildScenes()` itself needs to construct each scene and return a
  slice — if each scene block becomes a function, the order matters
  visually less but the helper interface is one extra layer.
- More files means more package boilerplate (every file gets a
  `package main` + imports).

## Suggested layout if we do it

```
cmd/divoom/
├── main.go
├── display.go
├── probe.go
├── render.go
├── serve.go
├── scenes.go              ← buildScenes() entry point + central
│                              constants + alwaysOn + helpers
├── scenes_helpers.go      ← (optional) sceneTitle, pipeAt*,
│                              QuoteScene, DictionaryScene,
│                              weatherTempColor, vCenter*, etc.
├── scene_markets.go       ← marketsScene() func returning *scene.Scene
├── scene_sky.go
├── scene_weather.go
├── scene_dayofyear.go
├── scene_hn.go
├── scene_easter.go
├── scene_catfacts.go
├── scene_didyouknow.go
├── scene_babylon5.go      ← (single-line: QuoteScene(...))
├── scene_startrek.go      ← (single-line)
├── scene_discworld.go     ← (single-line)
├── scene_jargon.go
├── scene_devil.go
├── scene_zenquotes.go
├── scene_sunrise.go
├── scene_iss.go
├── scene_nasa.go
├── scene_cocktail.go
└── scene_onthisday.go
```

Each `scene_<name>.go` exposes one `<name>Scene(widgets …) *scene.Scene`
function. `buildScenes()` becomes a flat list of calls:

```go
func buildScenes(widgets map[string]widget.Widget) []*scene.Scene {
    return []*scene.Scene{
        marketsScene(widgets),
        skyScene(widgets),
        weatherScene(widgets),
        // ...
    }
}
```

## Recommendation

Probably worth doing **after** the next 2-3 new scenes land (word
of the day, GitHub, TIL — they'd each be a new tiny file instead
of more bloat in `scenes.go`). The refactor itself is mechanical —
20-30 minutes of an agent — but pick a quiet moment when no other
agents are queued. Coordination/conflict cost is real, and is
exactly what this refactor is meant to reduce.
