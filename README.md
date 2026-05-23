# divoom

A custom wall-clock dashboard for the Divoom Times Frame (800×1280 portrait),
built because the stock app's preset dials are restrictive and the device is
an Allwinner TinaLinux box that quietly accepts `adb` pushes and exposes a
local JSON HTTP API at `:9000/divoom_api`. The dashboard runs as a Docker
container on a NAS, rotating every three minutes through 24 hand-designed
scenes that mix market tickers, weather + air quality + NWS alerts, sky /
moon / ISS, historical events, hand-curated quotes, useless facts, HN
headlines, and a ~300-deep rotation of typographic cocktail recipe cards
+ a 121-deep curated NASA APOD rotation.

![cocktail · margarita](docs/screenshots/cocktail.jpg)
![astronomy picture of the day · noctilucent clouds over Paris](docs/screenshots/nasa.jpg)

## What this is

The Times Frame is a 10.1" 800×1280 portrait IPS LCD that ships with a locked
set of preset dials. Its undocumented-in-broken-English local HTTP API lets
us install a custom layout: one 800×1280 background image, plus up to 6 Text
+ 10 Image + 6 NetData elements layered on top, plus built-in Time / Date /
Week / Mday / MonYear / Weather / Temperature blocks (special types — each
has its own quota and doesn't count against the 6-Text cap).

Because the device's image fetcher is cloud-proxied (it can't reach LAN URLs
and silently whitelists only `f.divoom-gz.com` for `Image` element URLs), we
don't try to host endpoints the frame polls. Instead the daemon:

1. Discovers the frame on the LAN (or talks directly to `DIVOOM_FRAME_IP`).
2. `adb`-pushes per-scene background JPGs into `/userdata/` on the device
   (one-time, from a USB-connected dev box). The NASA + cocktail scenes
   pre-bake every entry in their rotation pool into individual indexed
   bg JPGs so the device can show variety without ever touching the
   network at activation time.
3. Runs each widget (weather, markets, moon, whimsy rotator, …) in-process
   on its own refresh cadence, caching the last value.
4. Rotates through scenes every 3 minutes: at each scene change it bakes
   the current widget values into Text elements and installs the whole
   layout via `Device/EnterCustomControlMode`.

## Scenes

| Scene | What it shows |
|---|---|
| **markets** | Trading-terminal readout — symbol + price, week/month % badges with arrow + colour, ~35-day sparkline. `DIVOOM_TICKERS` rotates one symbol per activation. |
| **weather** | Big temperature (colour-banded by climate-normals-fitted thresholds), bottom strip with outlook + AIR / HUM / RAIN bound to AQI band, or "⚠ NWS alert" in red when one fires. Sources: Open-Meteo forecast + air-quality + NWS. |
| **sunrise** | Today's sunrise / sunset times + daylight hours. |
| **moonphase** | The current phase rendered as a real disc (one of 14 pre-rendered variants across the synodic cycle) + name + illumination + countdown to next full moon. |
| **iss** | Live sub-satellite-point dot drawn over a baked world-map outline + altitude + velocity. |
| **dayofyear** | 12×31 calendar grid with past / today / future cells and red letter-marks for `DIVOOM_SPECIAL_DATES` birthdays / anniversaries; season banner colour-shifts spring/summer/autumn/winter. |
| **catfacts** | Cat fact rendered as a Smithsonian-style field-guide entry — _Felis catus_ binomial, taxonomic line, pilcrow drop-marker, observation # / institution footer. |
| **didyouknow** | Random useless fact in body prose under a big bold "?" glyph. |
| **onthisday** | Historical event for today's date from Wikimedia — big orange year accent over the prose, tear-off-calendar glyph in the corner. |
| **til** | Top r/todayilearned post under a monumental "T I L" wordmark. |
| **easter** | (rare, weight 1 of ~480) Random whimsical one-liner printed dark-on-yellow _inside_ a cracked egg, with a "rare drop · ~1 in 200" caption. |
| **github** | Lifetime contributions (hero number, comma-separated, green) + three-column stat tile of total PRs / open PRs (cAqua when >0) / years on GitHub. |
| **hn** | Top Hacker News story filtered by keyword — title + domain + score / comments / author byline. |
| **nasa** | Astronomy Picture of the Day from a hand-curated pool of **121 iconic dates** (JWST releases, Hubble milestones, eclipses, Cassini, Pluto, EHT, …). One bake per date, indexed bg paths, shuffled per daemon start. |
| **cocktail** | Random drink from TheCocktailDB's Cocktail + Shot categories (~300 drinks) rendered as a typographic recipe card: drink name (huge), glass · category subhead, ingredient rows with measurements, wrapped method. Stable indexed paths, shuffled walk per daemon start. |
| **fortune / devil / wordnik / jargon** | Four dictionary / quote terminal layouts — `$ <cmd>` shell prompt + body + baked source/author footer. |
| **babylon5 / startrek / discworld** | Source-attributed quote scenes — from-source layout with attribution under the rule. |
| **stoics / twain / zenquotes** | Marginalia / page-of-a-book layout. |

More screenshots — every scene's baked chrome is in [`docs/screenshots/`](docs/screenshots/).
The dynamic Text overlays only show up once the frame installs the layout
live, so the bg-only renders look sparse for data-heavy scenes (markets,
weather, github). The cocktail + NASA scenes are fully baked, so their
screenshots match what you'd see on the wall.

## Architecture

Responsibility is split between a dev-box one-shot (`push`) that loads
static assets over USB-adb, and a NAS-side long-running daemon (`serve`)
that drives the frame over HTTP and pulls live data from external APIs.

```
  dev-box (USB)                        NAS                       Times Frame
  ┌──────────────┐   adb push          ┌──────────────┐  HTTP    ┌──────────┐
  │ divoom push  │ ──bgs + fonts────▶  │              │ ────────▶│ :9000    │
  └──────────────┘   to /userdata/     │ divoom serve │  scene   │ JSON API │
                                       │  (container) │  swaps   │          │
  ┌──────────────┐   open APIs:        │              │          │ 800×1280 │
  │ external     │ ◀──widgets poll──── │              │          │ IPS LCD  │
  │ APIs         │   Open-Meteo, NWS,  └──────────────┘          └──────────┘
  └──────────────┘   NASA APOD, NYT,
                     HN, Wikimedia, GitHub,
                     TheCocktailDB, Stooq
```

`push` runs occasionally — after scene-design changes, font changes,
or factory resets. The NASA + cocktail bakes are slow (hundreds of HTTP
fetches + ImageMagick-style compositing + adb pushes per bake) but they're
fully cached under `~/.cache/divoom/` so subsequent pushes are network-free.

`serve` runs forever, polling `divoom_api:9000` to set the active layout.
Widgets fetch from the open internet on their own cadences; nothing on the
frame ever reaches back into the LAN.

## Usage

```
go run ./cmd/divoom probe          # discover the frame, print current dial
go run ./cmd/divoom display test   # 30s gruvbox test layout, then restore
go run ./cmd/divoom display ticker # 30s ticker via UpdateDisplayItems
go run ./cmd/divoom render         # write scene JPGs to ./dist/scenes/
go run ./cmd/divoom push           # adb-push scene backgrounds + fonts (USB host only;
                                   # prereq: scripts/download-fonts.sh once)
go run ./cmd/divoom serve          # the dashboard daemon
```

Set `DIVOOM_FRAME_IP=<ip>` to skip cloud discovery and talk to a known device
directly (e.g. if you've firewalled the device off the public internet but
still want the daemon to reach it on the LAN). Set `DIVOOM_FRAME_MAC=<mac>`
to pin to a specific frame when you have more than one. See `.env.example`
for the full list of widget keys + deploy settings (NASA / GitHub / Wordnik
API keys, Portainer credentials, etc.).

A typical end-to-end deploy is `make` from the repo root, which builds the
container, pushes it to GHCR, redeploys the Portainer stack on the NAS, and
then runs `push-frame` to refresh scene backgrounds and fonts via adb.

## Docs

- [`docs/api.md`](docs/api.md) — empirical notes on the Times Frame API:
  endpoints we've used, quirks we've hit (the cloud-proxy URL whitelist,
  the per-type element caps, the ID-cache geometry bug, the Image-element
  Font-field requirement), and pointers back into Divoom's broken-English
  upstream docs at `docs/upstream/`. The source of truth for "how does the
  device actually behave"; update it in the same commit as any change that
  exercises new behaviour.
- [`docs/deploy.md`](docs/deploy.md) — the GHCR + Portainer deploy workflow
  (`make deploy` from this checkout).
- [`CLAUDE.md`](CLAUDE.md) — engineering philosophy (distilled from
  Kanat-Alexander's *Code Simplicity*) that all changes in this repo are
  judged by: reduce maintenance over implementation, keep pieces small,
  no speculative generality.
