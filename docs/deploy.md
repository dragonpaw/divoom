# Deploying to plugger

The dashboard runs as a Docker container on **plugger** (the M920q, 10.0.2.203),
where the Times Frame is USB-attached. It's plain Docker Compose in
`/srv/divoom` on plugger (`docker-compose.yml` + `stack.env`, root 0600).
Portainer was retired 2026-11-16. The image lives in GHCR
(`ghcr.io/dragonpaw/divoom`, public). Deploys are manual: build locally, push
to GHCR, then copy the compose file and env to plugger over ssh and recreate.

## One-time setup

1. **GHCR login.** Create a GitHub PAT with `write:packages` and
   `read:packages`, then:

   ```
   echo "$GHCR_PAT" | podman login ghcr.io -u <github-user> --password-stdin
   ```

2. **ssh to plugger** as `ash` with passwordless sudo (`DEPLOY_HOST`,
   default `ash@10.0.2.203`).

3. **Local `.env`.** Copy `.env.example` to `.env` and fill in API keys
   (`NASA_API_KEY`, `GITHUB_USER`, `GITHUB_TOKEN`) and the frame's
   `DIVOOM_FRAME_MAC` / `DIVOOM_FRAME_IP`. This file is gitignored and is
   read by the deploy command — its contents are copied to plugger as
   `/srv/divoom/stack.env`, the stack's environment.

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
3. Copy `.env` (minus comments and any `PORTAINER_*` lines; an empty
   `GITHUB_TOKEN` is filled from `GHCR_PAT`/`GITHUB_TOKEN`) to
   `/srv/divoom/stack.env` and `docker-compose.yml` to `/srv/divoom/`, then
   `docker compose -p divoom --env-file stack.env pull && ... up -d` there.

Override the target with `DEPLOY_HOST=...` on the make command line.

## Troubleshooting

- *Container didn't pick up the new image* — `make deploy` runs `pull`
  before `up -d`; check `ssh ash@10.0.2.203 'sudo docker images ghcr.io/dragonpaw/divoom'`.
- *Container starts but can't reach frame* — `network_mode: host` is
  required; plugger and the frame must share a LAN segment, and the USB
  cable must be in one of plugger's ports (`docker exec divoom-dashboard
  adb devices` should list serial `20080411`).
