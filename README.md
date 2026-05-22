# divoom

Wall-clock dashboard for the Divoom Times Frame, designed to live as a Docker
container on a NAS and drive the frame on the wall in front of it.

## What this is

The Times Frame is a 10.1" 800×1280 portrait IPS LCD that ships with a locked
set of preset dials. It also exposes an undocumented-in-broken-English local
HTTP API at `:9000/divoom_api` that lets us install a custom layout: one
800×1280 background, plus up to 6 Text + 10 Image + 6 NetData elements layered
on top, plus built-in Time / Date / Weather / Temperature blocks.

Because the device's image fetcher is cloud-proxied (it can't reach LAN URLs),
we don't try to host endpoints the frame polls. Instead the daemon:

1. Discovers the frame on the LAN (or talks directly to `DIVOOM_FRAME_IP`).
2. `adb`-pushes per-scene background JPGs into `/userdata/` on the device.
3. Runs each widget (weather, QQQ, moon, whimsy rotator) in-process on its
   own refresh cadence, caching the last value.
4. Rotates through scenes: at each scene change it bakes the current widget
   values into Text elements and installs the whole layout via
   `Device/EnterCustomControlMode`.

`Device/UpdateDisplayItems` is also used for sub-scene-cadence text patching
(see `divoom display ticker`), but the steady-state rotation is one
`EnterCustomControlMode` per scene change.

## Status

- LAN discovery via Divoom's `ReturnSameLANDevice` endpoint.
- Typed local-API client for `Channel/GetClockInfo`, `Channel/SetClockSelectId`,
  `Device/EnterCustomControlMode`, `Device/ExitCustomControlMode`,
  `Device/UpdateDisplayItems`.
- Scene driver rotating Markets / Sky / Whimsy / Quote with always-on
  Day-of-Week (coloured per weekday) + Time (coloured AM vs PM) + Date.
- Widgets: QQQ (Stooq), moon phase, day-of-year, cat facts, useless facts,
  HN headlines filtered by keyword, easter eggs, quotes (Devil's Dictionary,
  Jargon File, Babylon 5, Star Trek, sassy).
- Background renderer (`divoom render`) producing gruvbox-dark-hard scene JPGs
  with a hairline divider and year-progress bar.
- Custom on-device fonts (Iosevka, Roboto Condensed) adb-pushed to the frame.
- Dockerfile + `docker-compose.yml` for running on a NAS with `network_mode: host`.

## Usage

```
go run ./cmd/divoom probe          # discover the frame, print current dial
go run ./cmd/divoom display test   # 30s gruvbox test layout, then restore
go run ./cmd/divoom display ticker # 30s ticker via UpdateDisplayItems
go run ./cmd/divoom render         # write scene JPGs to ./dist/scenes/
go run ./cmd/divoom push           # adb-push scene backgrounds + fonts (USB host only)
go run ./cmd/divoom serve          # the dashboard daemon
```

Set `DIVOOM_FRAME_IP=<ip>` to skip cloud discovery and talk to a known device
directly (e.g. if you want to firewall the device off the public internet but
still want the daemon to reach it on the LAN). Set `DIVOOM_FRAME_MAC=<mac>` to
pin to a specific frame when you have more than one.

## API reference

See [`docs/api.md`](docs/api.md) for our running notes on the Times Frame API —
endpoints we've used, quirks we've hit, and pointers back into Divoom's
broken-English upstream docs. That file is the source of truth for "how does
the device actually behave"; update it in the same commit as any change that
exercises new behavior.

## Deploying

See [`docs/deploy.md`](docs/deploy.md) for the GHCR + Portainer deploy
workflow (`make deploy` from this checkout).
