# Tests for scene rotation rules

`internal/scene/`'s `Driver.pick()` (or wherever the rotation
logic lives) has at least these behaviours we'd want to lock in:

- **Same-count exclusion** — the next scene can't have the same
  element count as the one just shown (used to avoid layout flicker
  on scenes with identical bone-structure).
- **Weighted random pick** — scenes with higher `Weight` show
  proportionally more often; easter at weight 1 is rare.
- **Recovery from a panicking widget** — a widget that returns an
  error or panics shouldn't kill the rotation; the scene log line
  `"scene recovered"` we see at startup implies there's a recovery
  path. Test it actually works.
- **Recent-history avoidance** — at least some scenes (HN, TIL,
  fortune) use a ring-buffer to avoid repeating the same entry
  back-to-back; that's per-widget though, not Driver.

## What's missing

`internal/scene/` likely has no test file today. One
`scene_test.go` with a half-dozen table tests against `pick()`
covers the above. The Driver should be testable without hitting
the device — instantiate it with synthetic scenes whose `Widget`
is a stub returning canned text.

## Why bother

This is the load-bearing logic of the whole app. When the scene
count shifts (new scene added, weight changed), the same-count
exclusion behaviour can subtly change and you only notice when
you stare at the frame and realise X never shows. Tests catch
that at edit time.
