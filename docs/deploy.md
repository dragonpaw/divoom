# Deploying to Portainer

The dashboard runs as a Docker container on **plugger** (the M920q), where the
Times Frame is USB-attached since the 2026-06-04 fleet migration. It's managed
through the Portainer **hub**, which also runs on plugger (moved off the ADM NAS
2026-06-04) at `http://10.0.2.203:19900`; plugger is the hub's **agent endpoint
5**, so deploys target `endpointId=5` (the Makefile default). The image lives in GHCR
(`ghcr.io/dragonpaw/divoom`, public). Deploys are manual: build locally,
push to GHCR, then PUT the compose file at the Portainer stack API.

Portainer CE does not honour webhooks or git-backed redeploys reliably, so
the deploy command treats the stack as "editor-style" and replaces the
compose contents on every deploy.

## One-time setup

1. **GHCR login.** Create a GitHub PAT with `write:packages` and
   `read:packages`, then:

   ```
   echo "$GHCR_PAT" | podman login ghcr.io -u <github-user> --password-stdin
   ```

2. **Portainer API key.** In the Portainer UI, *My account* →
   *Access tokens* → *Add access token*. Save the token to
   `~/.config/divoom/portainer-key` (chmod 600).

3. **Portainer stack ID.** Create the stack once in the UI by importing
   `docker-compose.yml` from this repo with the environment vars from
   `.env`. After creation, copy the stack ID from the URL
   (`/#!/<endpoint>/docker/stacks/<id>`) into
   The stack itself doesn't need to exist beforehand — `make deploy` finds the stack named `divoom` (override with `STACK_NAME=...`) or creates it if it's not there.

4. **Local `.env`.** Copy `.env.example` to `.env` and fill in API keys
   (`NASA_API_KEY`, `GITHUB_USER`, `GITHUB_TOKEN`) and the frame's
   `DIVOOM_FRAME_MAC` / `DIVOOM_FRAME_IP`. This file is gitignored and is
   read by the deploy command — its contents are pushed to Portainer as
   the stack's environment.

## Deploy workflow

Scene backgrounds and the three custom TTFs (Iosevka, Roboto Condensed
Regular & Light) live on the device flash and are written by `adb push`.
The frame is USB-attached to **plugger**, and the container there ships
`adb` + `/dev/bus/usb` passthrough, so on-device pushes can run inside the
running container:

```
ssh root@10.0.2.203 'docker exec divoom-dashboard divoom push'
```

(Or attach the frame to the dev box and run `go run ./cmd/divoom push`
locally.) After any scene change (new scene, new bg art, new weather
outlook tier), or after a factory reset that wipes the overlay filesystem,
run once:

```
scripts/download-fonts.sh     # one-time, populates ./fonts/ from upstream
go run ./cmd/divoom push      # bgs + fonts; frame restarts at the end
```

The download script only fetches what's missing; it is safe to re-run.
The frame restarts at the end of `push` so divoom_app re-reads
`/divoom-config/system/font_list.cfg` — see docs/api.md "Custom font
workflow" for the mechanism.

Then deploy the daemon:

```
make deploy
```

That runs `build` → `push` → `deploy`:

1. `podman build` tagged with both `:latest` and `:$(git describe)`.
2. `podman push` of both tags to GHCR.
3. `PUT $PORTAINER_URL/api/stacks/$STACK_ID?endpointId=5` (plugger) with
   the current `docker-compose.yml` contents and `.env` values. `pullImage:
   true` forces Portainer to pull the new `:latest` before recreating
   the container. (`make deploy` scopes its stack-name lookup to the target
   endpoint, so the stale NAS-side `divoom` stack on endpoint 3 is ignored.)

Override the Portainer endpoint with `PORTAINER_URL=...`,
`PORTAINER_API_KEY=...`, `PORTAINER_STACK_ID=...`, or
`PORTAINER_ENDPOINT=...` on the make command line.

## Troubleshooting

- *401 from Portainer* — API key expired or wrong file. Re-issue from the
  UI and rewrite `~/.config/divoom/portainer-key`.
- *Stack updates but container doesn't pull* — confirm `pullImage: true`
  made it into the request body; check `/tmp/portainer-deploy.out` for
  the response.
- *Container starts but can't reach frame* — `network_mode: host` is
  required; plugger and the frame must share a LAN segment, and the USB
  cable must be in one of plugger's ports (`docker exec divoom-dashboard
  adb devices` should list serial `20080411`).
