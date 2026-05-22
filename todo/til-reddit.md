# r/TodayILearned top post scene

Item 10 from the 1-11 brainstorm. Pull the top post from
r/todayilearned (24-hour window) and render the title as a fact scene
— same shape as catfacts / didyouknow / easter, which is the
1-body-text "single fact" pattern.

## Source

Reddit JSON: `https://www.reddit.com/r/todayilearned/top.json?t=day&limit=10`

- No auth required for read-only JSON.
- Requires a User-Agent with a meaningful identifier per Reddit's API
  rules: `divoom-dashboard/0.1 (github.com/dragonpaw/divoom)`. Same
  pattern as the HN and NWS widgets.
- Response shape: `data.children[].data.title` — already cleaned by
  Reddit (no markdown to strip).

Pick a random post from the top 10 each Refresh so the scene rotates
between several headlines per day. Use the same `recentHistory`
ring-buffer pattern HN uses to avoid showing the same post twice in
a row.

## Scene shape

Mirror **catfacts** exactly — 4 elements (3 top + 1 body). Single
prose Text element with `Geometry: vCenterQuoteBody`, no header or
author block. The corner glyph carries the "TIL" branding.

Widget emits `"TIL|<post title>"`; scene uses `Format: pipeAt(1)` to
drop the header.

## Corner glyph

Recognisable "did-you-know" alternatives:
- Lightbulb (idea / new knowledge) — Heroicons `light-bulb` (MIT) or
  Bootstrap `lightbulb` (MIT)
- A stylised "TIL" text rendered as graphics — would need careful
  glyph composition from rectangles
- Open book — Bootstrap `book-open`

Lean toward lightbulb. Same `paintMask` embed pattern as the existing
SVG-sourced glyphs (Trek delta, Buddha, weather icons, OnThisDay clock).

Note: visually distinct from didyouknow's `?` glyph and catfacts' cat
silhouette since this is a separate scene.
