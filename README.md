# divoom

A custom wall-clock dashboard for the Divoom Times Frame (800×1280 portrait),
built because the stock app's preset dials are restrictive and the device is
an Allwinner TinaLinux box that quietly accepts `adb` pushes and exposes a
local JSON HTTP API at `:9000/divoom_api`. The dashboard runs as a Docker
container on a NAS, rotating through 24 scenes that mix market tickers,
weather, sky/moon, quotes, useless facts, HN headlines, and baked-image
scenes for the NASA APOD and a daily cocktail.

## What this is

The Times Frame is a 10.1" 800×1280 portrait IPS LCD that ships with a locked
set of preset dials. Its undocumented-in-broken-English local HTTP API lets
us install a custom layout: one 800×1280 background, plus up to 6 Text + 10
Image + 6 NetData elements layered on top, plus built-in Time / Date /
Weather / Temperature blocks.

Because the device's image fetcher is cloud-proxied (it can't reach LAN
URLs), we don't try to host endpoints the frame polls. Instead the daemon:

1. Discovers the frame on the LAN (or talks directly to `DIVOOM_FRAME_IP`).
2. `adb`-pushes per-scene background JPGs into `/userdata/` on the device
   (one-time, from a USB-connected dev box).
3. Runs each widget (weather, QQQ, moon, whimsy rotator, …) in-process on
   its own refresh cadence, caching the last value.
4. Rotates through scenes: at each scene change it bakes the current widget
   values into Text elements and installs the whole layout via
   `Device/EnterCustomControlMode`.

`Device/UpdateDisplayItems` is also used for sub-scene-cadence text patching
(see `divoom display ticker`), but the steady-state rotation is one
`EnterCustomControlMode` per scene change.

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
  ┌──────────────┐   Stooq, Open-Meteo │              │          │ 800×1280 │
  │ external     │ ◀──widgets poll──── │              │          │ IPS LCD  │
  │ APIs         │   HN, NASA, etc.    └──────────────┘          └──────────┘
  └──────────────┘
```

`push` runs occasionally (when backgrounds or fonts change). `serve` runs
forever, polling `divoom_api:9000` to set the active layout. Widgets fetch
from the open internet on their own cadences; nothing on the frame ever
reaches back into the LAN.

## Status

- LAN discovery via Divoom's `ReturnSameLANDevice` endpoint.
- Typed local-API client for `Channel/GetClockInfo`, `Channel/SetClockSelectId`,
  `Device/EnterCustomControlMode`, `Device/ExitCustomControlMode`,
  `Device/UpdateDisplayItems`.
- Scene driver rotating Markets / Sky / Whimsy / Quote with always-on
  Day-of-Week (coloured per weekday) + Time (coloured AM vs PM) + Date.
- Widgets: QQQ (Stooq), moon phase, day-of-year, cat facts, useless facts,
  HN headlines filtered by keyword, easter eggs, quotes (Devil's Dictionary,
  Jargon File, Babylon 5, Star Trek, Discworld, sassy).
- Background renderer (`divoom render`) producing gruvbox-dark-hard scene JPGs
  with a hairline divider and year-progress bar.
- Baked-image scenes for NASA APOD and a daily cocktail (image composited
  into the bg JPG at push time).
- Custom on-device fonts (Iosevka, Roboto Condensed) adb-pushed to the frame.
- Dockerfile + `docker-compose.yml` for running on a NAS with `network_mode: host`.

<!-- TODO: link a few representative `divoom render` outputs from docs/screenshots/ -->

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
directly (e.g. if you want to firewall the device off the public internet but
still want the daemon to reach it on the LAN). Set `DIVOOM_FRAME_MAC=<mac>` to
pin to a specific frame when you have more than one.

## Docs

- [`docs/api.md`](docs/api.md) — empirical notes on the Times Frame API:
  endpoints we've used, quirks we've hit, and pointers back into Divoom's
  broken-English upstream docs. The source of truth for "how does the device
  actually behave"; update it in the same commit as any change that exercises
  new behavior.
- [`docs/deploy.md`](docs/deploy.md) — the GHCR + Portainer deploy workflow
  (`make deploy` from this checkout).
- [`CLAUDE.md`](CLAUDE.md) — engineering philosophy (distilled from Kanat-
  Alexander's *Code Simplicity*) that all changes in this repo are judged by:
  reduce maintenance over implementation, keep pieces small, no speculative
  generality.
