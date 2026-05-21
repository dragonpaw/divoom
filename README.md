# divoom

Wall-clock dashboard for the Divoom Times Frame, designed to live as a Docker container on a NAS and drive the frame on the wall in front of it.

## What this is

The Times Frame is a 10.1" 800×1280 portrait IPS LCD that ships with a locked set of preset dials. It also exposes an undocumented-in-broken-English local HTTP API at `:9000/divoom_api` that lets us install a custom layout: one 800×1280 background, plus up to 6 Text + 10 Image + 6 NetData elements layered on top, plus built-in Time / Date / Weather / Temperature blocks.

The trick is that **the frame polls our data sources itself**. We declare element positions and URLs once via `Device/EnterCustomControlMode`, and the device pulls our JSON endpoints on its own (≥10 s interval). For sub-10 s text updates we patch in place via `Device/UpdateDisplayItems`. Image elements aren't patchable — those require re-sending the layout.

So this repo is not a "render-PNG-and-push" daemon. It's a small service that:

1. Discovers the frame on the LAN at startup.
2. Hosts a handful of tiny JSON widget endpoints (weather, market, calendar, …).
3. Decides which "scene" should be visible and installs its layout on the frame.
4. Pushes text updates between scene changes when something interesting happens.

## Status

Early scaffolding. Working:

- LAN discovery via Divoom's `ReturnSameLANDevice` endpoint.
- Typed local-API client for `Channel/GetClockInfo`, `Channel/SetClockSelectId`, `Device/EnterCustomControlMode`, `Device/ExitCustomControlMode`, `Device/UpdateDisplayItems`.
- `divoom probe` — discovers the frame, prints its current dial + brightness.

Not built yet: scenes, widget HTTP server, background renderer, Docker image, scene scheduler, ADB-pushed custom fonts.

## Usage

```
go run ./cmd/divoom probe
```

Set `DIVOOM_FRAME_IP=<ip>` to skip cloud discovery and talk to a known device directly (e.g. if you want to firewall the device off the public internet but still want the daemon to reach it on the LAN). Set `DIVOOM_FRAME_MAC=<mac>` to pin to a specific frame when you have more than one.

## API reference

See [`docs/api.md`](docs/api.md) for our running notes on the Times Frame API — endpoints we've used, quirks we've hit, and pointers back into Divoom's broken-English upstream docs. That file is the source of truth for "how does the device actually behave"; update it in the same commit as any change that exercises new behavior.

## Where backgrounds live

The Times Frame's image fetcher is cloud-proxied through Divoom's servers (see [docs/api.md](docs/api.md) → Empirical findings), so any URL the frame consumes must be publicly reachable. We split the project:

- **This repo (private)**: code, layout definitions, the daemon that pushes text updates over the LAN.
- **`<owner>/divoom-assets` (public)**: rendered scene backgrounds, served by GitHub Pages at `https://<owner>.github.io/divoom-assets/scenes/<name>.jpg` (or via a custom domain CNAME).

A workflow [`.github/workflows/publish-assets.yml`](.github/workflows/publish-assets.yml) re-renders the scenes via `go run ./cmd/divoom render` and pushes them to the public repo on every change to `internal/render/` or `cmd/divoom/render.go`.

### One-time setup

1. Create the public assets repo (default name `divoom-assets`) on GitHub. Empty is fine.
2. In that repo's **Settings → Pages**, enable Pages for `main` branch, root.
3. Generate a **fine-grained PAT** at <https://github.com/settings/tokens?type=beta> scoped only to the assets repo with **Contents: Read and write**.
4. In this repo's **Settings → Secrets and variables → Actions**, add the PAT as `ASSETS_TOKEN`.
5. If your handle isn't `dragonpaw`, edit `ASSETS_REPO` in `.github/workflows/publish-assets.yml`.
6. Push to main (or trigger the workflow manually) and confirm scenes appear in the public repo.
