# Deploying to plugger

The dashboard runs **natively** on **plugger** (the M920q, 10.0.2.203), where the
Times Frame is USB-attached, as `divoom.service` (since 2026-11-16; before that a
Docker container, before that a Portainer stack). `make deploy` builds a static
binary and copies it with `fonts/`, `device-files/`, the unit and the udev rule
(`deploy/`) to `/opt/divoom`, then restarts the service. Env: `/srv/divoom/stack.env`
(root 0600). Cache: `/var/cache/divoom`. Logs: `journalctl -u divoom`.

## One-time setup

1. **Go** (`go build`) and the fonts: `scripts/download-fonts.sh` fills `fonts/`.

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
The frame is USB-attached to **plugger**; the udev rule gives the `divoom` user
the device, so on-device pushes run there as the service's user with its env:

```
make push-frame      # ssh plugger: systemd-run ... /opt/divoom/divoom push
```

(Or attach the frame to the dev box and run `go run ./cmd/divoom push`
locally.) Run it after any scene change (new scene, new bg art, new weather
outlook tier), or after a factory reset that wipes the overlay filesystem.
The frame restarts at the end of `push` so divoom_app re-reads
`/divoom-config/system/font_list.cfg` — see docs/api.md "Custom font
workflow" for the mechanism.

Deploy the daemon (and `make` alone does `deploy` then `push-frame`):

```
scripts/download-fonts.sh     # one-time, populates ./fonts/ from upstream
make deploy
```

1. `go build` a static binary (`CGO_ENABLED=0`) to `dist/divoom`.
2. If `.env` is present here, copy it (minus comments and any `PORTAINER_*`
   lines; an empty `GITHUB_TOKEN` is filled from `GHCR_PAT`/`GITHUB_TOKEN`) to
   `/srv/divoom/stack.env` (root 0600); otherwise the host's copy is kept.
3. rsync the binary, `fonts/`, `device-files/` and `deploy/` to `/opt/divoom`
   (root-owned), install `divoom.service` and the udev rule, restart.

Override the target with `DEPLOY_HOST=...` on the make command line.

## Troubleshooting

- *Service runs but can't push to the frame* — check the USB device is group
  `divoom` (`ls -l /dev/bus/usb/*/*`; the udev rule matches `18d1:d002`) and that
  `sudo -u divoom env HOME=/var/lib/divoom adb devices` lists serial `20080411`.
  plugger and the frame must share a LAN segment for the HTTP API.
