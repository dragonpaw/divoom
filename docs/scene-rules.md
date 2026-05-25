# Scene rules

The Times Frame is a deeply constrained display target. These rules
encode the constraints that have been hit (and the workarounds that have
been tried and ruled out) so future scene work doesn't relearn them. Pair
with [api.md](api.md) for protocol details and CLAUDE.md for the
engineering philosophy.

---

## The hard constraints

### 1. The 6-element layout cap

A single `Device/EnterCustomControlMode` payload can hold at most **6
display elements total** (Text + Time + Image, combined). The always-on
chrome consumes 3 (2 Text + 1 Time), leaving **3 slots for scene body**.

- Don't fight the cap by splitting concepts across multiple elements;
  bake static content into the background instead.
- For multi-value content (sparkline, ticker list, multi-cell grid), the
  options are: pick ≤3 values to elevate, or **render the whole thing
  server-side as a baked bg**.

### 2. Backgrounds are mostly frozen at deploy

Scene backgrounds are pushed at startup via `adb push`, then referenced
by local path with `BackgroundImageLocalFlag: 1`. The only bgs that
refresh on a live device are the **calendar** and **genart** scenes,
which `startDailyRefresh` (cmd/divoom/daily_refresh.go) re-renders and
adb-pushes at every local midnight (and once at startup). Everything
else is immutable between deploys.

- A baked bg must not encode the current date, current time-of-day, or
  any other state that changes faster than the deploy cadence — **unless
  the scene is wired into the daily refresh path**.
- Static decorations, axis labels, axis ticks, season art, in-universe
  chrome (book page, shell prompt, etc.) — fine.
- Today's sun position, today's stock prices — **not fine** in the bg.
  Put those in foreground elements.

### 3. Daily refresh exists for the date-sensitive scenes only

`startDailyRefresh` covers calendar + genart. It runs inside `divoom
serve` (no separate cron, no host USB needed at runtime) and re-pushes
those two JPGs without reloading fonts — so divoom_app does NOT
crash-restart and serve keeps running.

The original two refresh paths considered for general bg refresh are
both still ruled out, which is why the daily refresh is opt-in
per-scene rather than blanket:

- **adb-over-TCP** — confirmed broken on this device. docs/api.md:396–401.
  Summary: `adb tcpip 5555` doesn't persist across power cycles; editing
  `/etc/init.d/adbd` to enable TCP at boot bricked adbd entirely (only
  recoverable via factory reset); the shipped adbd binary fails to bind
  any TCP listener even when asked. **Not feasible without replacing the
  adbd binary.**
- **Remote-URL background fetch** — confirmed broken. docs/api.md:399.
  The Times Frame's cloud proxy whitelists only `f.divoom-gz.com` for
  Image element URLs. Self-hosted public URLs (NASA, raw.githubusercontent,
  Wikimedia, our own host) all silently fail — the device renders the
  rest of the layout with the image slot blank, no error.

The daily refresh works only because the container itself has adb +
USB-passthrough to the device. Without that USB path you're back to
immutable-bgs land.

### 4. Dynamic data must come through foreground elements

For values that change between deploys, the live channels are:

- **Text elements**, updated via `Device/UpdateDisplayItems` —
  unconstrained content, drawn server-side by us each tick.
- **NetData** — frame polls a URL on its own and extracts a single
  scalar per RuleInfo. Cloud-proxied; **public URLs only**, same whitelist
  caveat as Image elements (see api.md:143 + the Image whitelist note).
  One scalar per element, so multi-value widgets burn one slot per value.
- **Image elements** — same whitelist limitation. Only useful for assets
  hosted at `f.divoom-gz.com`, which in practice means we don't use them
  for our own content.

Default to Text elements driven by our server. Reach for NetData only
when the value genuinely lives behind a Divoom-CDN URL.

### 5. Field-name typos must be preserved

Divoom's protocol mis-spells `BackgroudImageAddr` (no `n`) and
`BackgroudImageLocalFlag` in `Device/EnterCustomControlMode`. Sending the
correctly-spelled name silently does the wrong thing. Mirror the typos
exactly. (api.md:89)

### 6. Element properties are cached by (Type, position-in-DispList)

Re-installing a layout where the new DispList has the **same total
length** as the previous install means the device treats it as an
update — only `TextMessage` round-trips; `FontSize`, `StartY`,
`Height`, `Align` retain their prior values. The cache is keyed on
slot position by Type, **not** on element ID (verified 2026-05-25;
see api.md rows 2026-05-25). The Divoom docs claim ID-based
addressing but that's wrong for this firmware.

- The scene driver (`internal/scene/scene.go`) handles this
  automatically: it tracks the previous install's DispList length
  and appends an off-screen `Year`-type filler element when the
  current install would otherwise match. Length differs → device
  reallocates all slots fresh.
- For one-off installs outside the driver (e.g. `divoom display
  lines`), the `--no-reset` flag toggles between "Exit then Enter"
  (clean slate, brief preset-dial flash) and "Enter only" (preserves
  cache — useful for probing or for chained installs that don't need
  geometry changes).
- Within a stable layout, only `TextMessage` round-trips reliably via
  `UpdateDisplayItems`. Geometry changes need a length change OR an
  explicit `ExitCustomControlMode`.

### 7. Text wrapping is height-driven

At `FontSize=34` with `Width=760`, the device wraps at roughly **45–46 px
per line**. There is no hidden per-element line cap — `Height` directly
determines how many wrapped lines render. Size `Height` to the visible
line count you want. (api.md:412)

---

## Workflow rules

### Scene additions go through brainstorming and designer review

- For any new scene or visual change, dispatch a designer-review agent
  *before* implementation. Relay the verdict, wait for confirmation,
  then dispatch a separate implementer. (memory:
  [feedback_designer_review_pattern](../.../...)).
- New scenes default to `WeightEntertaining` (20). Promote to
  `WeightInformational` (40) only if the content depends on *when*
  you look (clock, calendar, current weather, current markets). The
  rotation aims for ~50/50 between the two tiers.

### Substantial scene work runs as a background agent

Multi-file feature work (a new scene end-to-end, a cross-scene refactor)
runs as `Agent({...run_in_background: true})`. "Kick off bg agent" is the
cue.

### Living docs

When a scene work session uncovers something new about the device — a
quirk, a limit, a workaround that worked, a workaround that didn't —
update **docs/api.md** in the same change. The codebase is the only
authoritative record of what we know.

### Screenshots are pipeline-generated

Never hand-edit files in `docs/screenshots/`. Use `divoom render` +
`TestRefreshScreenshots` (memory:
[feedback_screenshot_workflow](../.../...)).

### Scene-glyph colour rule

Every corner glyph paints in `SceneGlyphColor`. If a glyph reads scrawny
at dim, redesign the **shape**, not the colour.

### Engineering philosophy

CLAUDE.md governs. The short version for scenes:

- The smallest scene that does *something* useful, then iterate. Don't
  pre-build configuration knobs.
- Specific before generic. Two scenes that share a pattern is not enough
  to justify a framework — wait for the third concrete use case.
- Don't refactor "while you're in there." Drive-by refactors inflate
  diffs and inflate bugs.
- Fix only what has evidence of being broken. Aesthetic discomfort
  doesn't count.

---

## Quick decision tree for new scenes

1. **What changes between deploys?** That content goes in foreground
   Text elements driven by our server.
2. **What never changes (or only changes per deploy)?** Bake into the
   background image. Includes axis art, chrome, decorative content,
   season-scoped imagery if the scene rotates seasonally.
3. **What changes *within* a deploy?** That content cannot be in the
   bg. If it can't fit in ≤3 foreground elements, the scene needs a
   different design — not a different infrastructure.
4. **Need a third-party image?** Either it lives on `f.divoom-gz.com`
   (it almost certainly doesn't), or you adb-push it server-side and
   reference it with `*LocalFlag: 1`. Public-URL Image elements do not
   work for us.
