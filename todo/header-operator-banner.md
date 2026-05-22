# Always-on header redesign — Operator Banner (Direction C)

Replace the current functional-but-plain stacked day/time/date header
with a retro-terminal/operator-banner aesthetic. Lean into the
gruvbox + Iosevka heritage instead of arriving at the same wall-clock
default every digital display lands on.

Decision recorded 2026-05-22 after a `frontend-design` critique
proposed three directions (A=Almanac Page, B=Station Board,
C=Operator Banner). C wins on "most-true-to-the-project + lowest
risk + smallest layout disruption."

## Current state (what's being replaced)

`cmd/divoom/scenes.go` → `alwaysOn(now)`:
- y=20-100: Day of week, full word, 64pt fontProse, dayColors[weekday].
- y=120-340: Time, 180pt fontMono, timeColor (cAqua morning / cOrange evening).
- y=370-430: Date, 44pt fontProse, cFgDark, spelled-out month/day/year.
- y=460-462: hairline divider (baked in bg).

Critique: stacked-centered triple is the Apple-Watch-modular default.
Date row is wallpaper. Spelled-out weekday + dayColor is redundant.
No compositional voice.

## New layout

```
┌──────────────────────────────────────────────────────────┐
│                                                          │
│  > wednesday                                             │  ← prompt + lowercase
│                                                          │
│                                                          │
│                  1 4 : 3 7                               │  ← time, 160pt
│                                                          │
│                                                          │
│  ──── ── ── ─ ── ── ── ── ─ ── ── ── ── ─ ── ── ────   │  ← Morse-pattern rule
│                                                          │
│  2026-05-22  doy:142  w:21  weekend+2d                  │  ← operator footer
│                                                          │
│  ───── existing y=460 hairline ─────────────────────────│
└──────────────────────────────────────────────────────────┘
```

## Per-element spec

| ID | Type | x, y, w, h | Font | Size | Color | Notes |
|---|---|---|---|---|---|---|
| 1a | Text | x=40, y=30, w=40, h=80 | fontMono | 64 | cFgDark | `>` prompt prefix. Static. |
| 1b | Text | x=80, y=30, w=680, h=80 | fontMono | 64 | dayColor | Lowercase day name (e.g. `wednesday`). dayColors map already exists. |
| 2 | Time | x=50, y=140, w=700, h=200 | fontMono | 160 | cFg | Time. Solid cFg — drop the timeColor warm/cool gimmick; the dayColor on the prompt is the day signal. |
| 3 | (BAKED) | x=40, y=380, w=720, h=2 | — | — | cFgDark | Morse-pattern rule (see below). |
| 4 | Text | x=40, y=400, w=720, h=44 | fontMono | 28 | cFgDark | Operator footer (see below). |
| — | (BAKED) | y=460 hairline | — | — | cFgDark | Existing divider, unchanged. |

Element-count delta: was 3 (Text + Time + Date) → becomes 3 (2 Text +
1 Time) plus 1 dynamic footer Text → total 4 always-on Text + 1 Time.
Plus existing sceneTitle on every scene (1 Text). That's 5 Text + 1
Time before the scene's body starts; scene body has 6 - 4 = 2 Text
slots left. **This is tighter than today (which leaves 5 slots for
scene body).** Need to audit which scenes still fit before shipping.

Scenes most at risk:
- weather (just rebuilt to 6 elements; 4 of those are Text) — **would not fit** with a 4-Text always-on header.
- nasa / cocktail (only sceneTitle now; safe).
- DictionaryScene-based (jargon, devil, wordnik) — 4 Text bodies.
- QuoteScene-based (B5, ST, etc.) — 2 Text bodies.

**Resolution**: either drop the `>` prompt into the day name (single
combined Text element `> wednesday`), or drop the Date element from
the always-on header entirely (which we'd be doing anyway since the
operator footer subsumes it).

Recommended: combine `>` + day name into one Text element (e.g.
`"> wednesday"` with the whole thing in dayColor). Saves a slot. The
prompt-glyph-in-different-color was a nice-to-have, not load-bearing.

Final element count after combine: 1 day+prompt + Time + 1 footer + 1
sceneTitle = 3 Text + 1 Time, same as today's 3 Text + 1 Time (Day +
Date + sceneTitle + Time). Scenes keep their full 3-Text budget. ✓

## Footer content

`fmt.Sprintf("%s  doy:%d  w:%d  %s",
    now.Format("2006-01-02"),
    now.YearDay(),
    isoWeek(now),
    daysUntilWeekend(now))`

Where:
- `isoWeek(now)` — Go's `time.Time.ISOWeek()` returns (year, week).
- `daysUntilWeekend(now)` — small helper returning `"weekend+Nd"` for
  weekdays (Mon=4d to Fri=0d), `"weekend"` for Sat/Sun. Open to
  swapping in a different countdown anchor (next holiday, end-of-month,
  arbitrary date) if you have one that's more useful.

## Baked Morse-pattern rule

In `internal/render/background.go` (add to `buildHeroImage` since
it's now part of every scene, not just one):

- y=380, x=40→760, 2px tall.
- Pattern: alternating 16px dashes with 4px gaps. Every 5th gap
  replaced with a single 2px dot (so reads as `── ── ── · ── ── ── ·`
  rather than uniform `── ── ── ── ──`).
- Color: cFgDark (GruvFgDark).
- Painted into the hero bg (every scene), not the per-scene bg —
  it's part of the always-on chrome.

## Drop the `timeColor` morning/evening swap

Direction C explicitly says the time stays solid cFg. Rationale: the
dayColor on the day name already carries the "what flavor of day is
it" signal; the time-of-day swap is redundant once the day is loudly
colored. Remove `timeColor()` and inline `cFg` in the always-on
Time element.

(If you want to keep `timeColor` for nostalgic reasons, leave it but
swap to cFg — the function becomes dead and can be deleted in a
follow-up.)

## Verify after implementing

- `go build ./...`, `go vet ./...`, `go test ./...` clean.
- `go run ./cmd/divoom render -out /tmp/divoom-header-render` then
  visually inspect a couple of scene JPGs (markets, weather, nasa)
  — the Morse-pattern rule should be visible at y=380 across every
  one of them. (The y=460 hairline stays.)
- Push and run on the live frame; confirm the day name reads
  cleanly in lowercase Iosevka at 64pt, the operator footer is
  legible at 28pt, and the Morse rule isn't visually noisy.

## Out of scope

- Animated/blinking colon between HH and MM (cliché; explicit reject).
- Time as analog clock face (impossible with Image-element constraints;
  explicit reject).
- Per-scene custom headers (defeats the purpose of "always-on" giving
  the dashboard a unified identity; explicit reject).
- Weather/temperature in the header (duplicates the weather scene's
  data and clutters what should be a clean time identity; reject).

## See also

The full critique (with Direction A "Almanac Page" and Direction B
"Station Board" as the rejected alternatives) was produced by the
`frontend-design` skill on 2026-05-22; it lives in the conversation
transcript at that point. If we ever want to revisit, re-run the
skill with the same prompt.
