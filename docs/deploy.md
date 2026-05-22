# Deploying to Portainer

The dashboard runs as a Docker container on the home NAS, managed by the
Portainer instance at `http://10.0.2.201:9000`. The image lives in GHCR
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
   `~/.config/divoom/portainer-stack-id`.

4. **Local `.env`.** Copy `.env.example` to `.env` and fill in API keys
   (`NASA_API_KEY`, `GITHUB_USER`, `GITHUB_TOKEN`) and the frame's
   `DIVOOM_FRAME_MAC` / `DIVOOM_FRAME_IP`. This file is gitignored and is
   read by the deploy command — its contents are pushed to Portainer as
   the stack's environment.

## Deploy workflow

Scene backgrounds and the three custom TTFs (Iosevka, Roboto Condensed
Regular & Light) live on the device flash and are written by `adb push`.
The NAS container has no USB connection to the frame, so the daemon
running there cannot push them — that step must happen from the
USB-attached dev box. After any scene change (new scene, new bg art,
new weather outlook tier), or after a factory reset that wipes the
overlay filesystem, run once from the dev box:

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
3. `PUT $PORTAINER_URL/api/stacks/$STACK_ID?endpointId=1` with the
   current `docker-compose.yml` contents and `.env` values. `pullImage:
   true` forces Portainer to pull the new `:latest` before recreating
   the container.

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
  required; the NAS and the frame must share a LAN segment.
