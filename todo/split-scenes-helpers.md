# Finish splitting cmd/divoom/scenes.go

The 2026-05-22 scene-per-file refactor took `scenes.go` from 1599
to ~895 lines. Most of what's left is helper code, not scene
definitions, and it's the same "monolith two agents would
trample on" problem in miniature.

## What's still in `scenes.go`

- Element-ID, font-ID, bg-path constants
- Gruvbox palette + `dayColors` / `timeColor`
- `alwaysOn` + `sceneTitle`
- `pipeAt*`, `vCenter*` formatters
- `weatherTempColor`, `parseWeatherTemp`, `weatherBgFor` + the
  atomic threshold state and `init()`
- `QuoteScene` + `QuoteSceneOpts` + author/tagline rendering
- `DictionaryScene` + `DictionarySceneOpts` + headword fit
- `stripQuotes`, `shrinkHeadword`, `fitDictionaryBody`

## Suggested split

- `scenes.go` — `buildScenes()` entry, central constants,
  `alwaysOn`, `sceneTitle`. Stays in the cmd/divoom package because
  it references font/color identifiers and Scene types directly.
- `scenes_helpers.go` — formatters (`pipeAt*`, `vCenter*`,
  `stripQuotes`, `shrinkHeadword`, `fitDictionaryBody`).
- `scenes_weather_color.go` — `weatherTempColor`,
  `parseWeatherTemp`, `weatherBgFor`, threshold atomics + `init`.
- `quote_scene.go` — `QuoteScene` + `QuoteSceneOpts`.
- `dictionary_scene.go` — `DictionaryScene` + `DictionarySceneOpts`.

All stay in `package main`; no public-API surface changes. Same
verification as the previous split: render parity diff.

## Why bother

Same reason as the first split — concurrent agents stop trampling
each other. The QuoteScene helper is the most common change vector
when adding a new quote source; splitting it out means new-source
agents touch one tiny file plus `serve.go` plus `scenes.go`'s
buildScenes call.
