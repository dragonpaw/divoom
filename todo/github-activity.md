# GitHub activity scene

Item 9 from the 1-11 brainstorm. Show today's commit count + this-year
streak + open-PR count for the configured GitHub user. Tech/personal
metric scene.

## Source

GitHub REST API v3. Endpoints used:

- **Today's commits**: query `https://api.github.com/search/commits`
  with `author:<user> author-date:>=<today>` — note this requires the
  preview Accept header `application/vnd.github.cloak-preview+json`
  for unauthenticated, or a PAT for clean access. Authed is preferred
  to avoid the 30 req/min anonymous rate limit.
- **Year-to-date contribution count + streak**: GitHub's GraphQL API
  (v4) has `user.contributionsCollection.contributionCalendar` with
  daily counts; compute current streak from the tail. Requires a PAT.
- **Open PRs authored by user**: `search/issues` with
  `is:open author:<user> type:pr`.

Auth: read GitHub PAT from `GITHUB_TOKEN` env var. Fall back to
unauthenticated where possible (much lower rate limit; some endpoints
won't work).

## Scene shape

Layout — 3 top + 3 body = **6 elements** (collides with nasa /
cocktail / onthisday / weather — fine):

```
y≈540-700   <today commits>  big mono, FontSize 130, cGreen if >0 else cFgDark   (idSceneMain)
y≈720-840   <streak days>    medium mono, FontSize 70, cYellow if >7d else cFgDark (idSceneSub1)
y≈860-980   <open PRs>       small prose, FontSize 36, cAqua                     (idSceneSub2)
```

Labels are baked into the visual context — big numbers + the corner
glyph (octocat? cat-like silhouette?) carry meaning.

Output format: `"<today_commits>|<streak_days>|<open_prs>"` from the
widget; scene splits with `pipeAt(0..2)`.

## Corner glyph

The GitHub octocat-style silhouette is recognisable but the trademark
is murky. Safer alternatives:
- Open-source "git" branch icon — three dots with connecting lines
- Generic terminal-prompt `>_` rendering
- Bootstrap Icons `git` (MIT) — a clean branch + commit shape

Lean toward the branch-icon SVG via the `paintMask` pattern.

## Privacy / env-var caveat

The default config should not bake in a specific user. Either:
- Read `GITHUB_USER` env var alongside `GITHUB_TOKEN`
- Skip the scene entirely (don't add to `buildScenes`) when env vars
  are unset
- Log "github scene disabled (set GITHUB_USER + GITHUB_TOKEN)" at startup
