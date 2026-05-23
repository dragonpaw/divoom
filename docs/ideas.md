# Ideas

Future scene / feature ideas that aren't on the critical path. **Do
not auto-implement these.** They're recorded so we don't forget the
shape, and so a future session can pick one up after explicit
direction.

## Recorded ideas

### Calendar / agenda scene (Google Calendar)

Pull today's next event + countdown from Google Calendar via a
service account (JSON credentials in env) or an OAuth refresh token.
Shows "next: standup in 23 min" plus the event title and the
hours-until counter. Biggest utility win of the deferred ideas —
turns the dashboard into a glanceable "what's next."

Friction: auth setup. Service account is one-time and renews itself;
OAuth needs a refresh-token bootstrap from the user's browser
session. Either is acceptable; service account is the simpler runtime.

### Aurora / K-index sky-watch scene

NOAA SWPC publishes a 3-day Kp index forecast at
`services.swpc.noaa.gov/products/noaa-planetary-k-index-forecast.json`
— no auth, JSON, small. Render as a 3-day strip with Kp 0-9 bars +
colour band (cGreen <4, cYellow =5, cRed ≥6). Genuinely useful for
aurora-chasing latitudes and elegantly extends the existing
sunrise/moonphase sky-watching family. Element budget: 3-4 Text
elements; fits the 4-slot cap.

### Repo anniversaries — "this day in commits"

`git log --since/--until` to surface 1-yr-ago, 2-yr-ago, etc. commit
subjects from *this* repo, rendered as "5 years ago today you wrote
…". Niche, self-referential, charming. Zero external dependencies.
Tiny widget. Best on a repo with multiple years of history.

### Last.fm now-playing / recent scrobbles

Pull `user.getrecenttracks` for the configured handle (free API
key). Show currently-playing artist + track + album art (album art
would require a bake-on-push lookup since the device can't fetch
arbitrary image URLs). Falls through to "last played: …" when
nothing's live. Pairs beautifully with the quote scenes.

## Backlog from earlier brainstorm — out of scope unless asked

- **Tide tables** — Only useful coastal; data hygiene fragile.
- **Home Assistant integration** — Large surface area; depends on a
  user-specific HA install.
- **Stock portfolio aggregator** — Stooq is fine for one ticker;
  multi-position P/L tracking would need a persisted positions file.
- **Mastodon / Bluesky highlight** — Volatile API schemas; not worth
  the maintenance.

## Adding to this file

When the user surfaces a future scene idea, append it here under
"Recorded ideas" with:
- the working name
- one sentence on what it shows
- the data source and any auth shape
- a rough element-count estimate against the device's 4-slot
  per-scene cap

When the user picks one to implement, move it out of this file and
into a real commit. Don't auto-prioritise.
