# Make adb-tcpip persistent on the Times Frame

Today `adb tcpip 5555` works from a USB-attached host but is wiped
on power cycle (confirmed 2026-05-22 — see `docs/api.md` empirical
findings). Making it persist would unlock adb-over-network as a
reliable LAN tool from the NAS — font installs, file pushes, and
discovery probing — without requiring a USB cable for every reboot.

## What we know about the device

- TinaLinux + BusyBox 1.33.2. NOT Android. Android-isms like
  `setprop service.adb.tcp.port` won't work.
- `adb shell` is login-gated (no creds), so we can't poke around
  interactively to find the adbd init script.
- `adb push` / `adb pull` bypass auth — so we can write/read files
  freely.

## Investigation plan

1. **Find the adbd init script.** Likely candidates:
   - `/etc/init.d/S50adbd` or `S*adbd` (BusyBox/Buildroot convention)
   - `/etc/init.d/rcS` (catch-all)
   - `/etc/inittab` (if BusyBox uses inittab here)
   - `/lib/systemd/system/adbd.service` (unlikely on BusyBox, but check)
   `adb pull` each candidate path; whichever exists, read it.

2. **Identify the adbd invocation.** Look for how adbd is started.
   The default is USB-only; we want it to also listen on TCP. The
   relevant adbd args / env var is one of:
   - `adbd -P 5555` (older adb)
   - `ADBD_PORT=5555` env
   - `service.adb.tcp.port` property (Android — probably not here)
   - A separate `adbd_tcp` service script

3. **Build a persistent patch.** Once we know the init path, write
   a tiny addition (an extra `S99adb-tcp` init script that runs
   `adb tcpip 5555` equivalent — likely something like
   `setprop service.adb.tcp.port 5555 && stop adbd && start adbd`
   or the busybox equivalent of restarting adbd in TCP mode).
   `adb push` it into the overlay (`/overlay/upper/etc/init.d/...`
   for persistence — see the font-install workflow in api.md for
   the overlay-write pattern).

4. **Reboot and verify.** `adb connect 10.0.2.108:5555` should
   succeed after a cold power cycle with no USB involvement.

## Risks

- **Bricking risk — proven, not theoretical.** First attempt
  (2026-05-22) was to uncomment the existing `#ADB_TRANSPORT_PORT=5555`
  line in `/etc/init.d/adbd` and reboot. Result: adbd dead on both
  USB and TCP after the next power cycle. The Divoom app + local
  `divoom_api:9000` kept working (separate service), but our only
  config-poking interface was gone. Recovered via the Divoom app's
  factory reset, which wipes the overlay filesystem and restores
  the original init script. **Lesson: do NOT modify `/etc/init.d/adbd`
  directly.** USB recovery cannot help if the modification kills
  adbd outright.
- **Firmware updates.** A Divoom OTA might overwrite the overlay
  or wipe `/etc/init.d/` additions. If we lose the persistence
  after an update, re-apply.
- **Security.** adb-on-LAN means anyone on the LAN can `adb push`
  files to the device. Home LAN, so accept the risk; document the
  exposure.

## Safer second-attempt strategies

Two paths that don't put adbd's startup at risk:

1. **Side-car init script.** Push a NEW file at
   `/etc/init.d/S99adb-tcp` (or similar) that runs AFTER adbd starts
   and executes the equivalent of `adb tcpip 5555` from inside the
   device — i.e. signal/restart adbd with the env var, or invoke a
   helper that opens the TCP listener. The existing
   `/etc/init.d/adbd` is unmodified, so even if the side-car
   misbehaves, adbd starts normally and USB recovery still works.
2. **rc.local entry.** Append a line to `/etc/rc.local` that runs
   after all init scripts; same idea, even smaller blast radius.

In either case, **first push a known-no-op test version** (e.g. an
init script that just `echo`s to a file) and verify it ran via
`adb pull` on the log file after a reboot. Only then add the
behaviour-changing line.

**Always push a pre-built file, never `sed` in place.** Edit the
file locally in a real editor (so you can eyeball it byte-for-byte),
keep it checked into this repo under `device-files/` or similar,
and `adb push` the exact bytes. `sed` can silently mangle CRLF, hit
unexpected matches, or transform the wrong line if the source
drifts from what you assumed.

If `ADB_TRANSPORT_PORT` really is the right env var for this adbd,
the side-car can `kill -HUP` adbd while setting the env so the new
instance picks it up — without ever touching the original launcher.

## Verify after implementing

- Cold power cycle the frame; wait 60 s; `adb connect 10.0.2.108:5555`
  from the NAS succeeds without any USB intervention.
- A second reboot a day later still works.
- After a Divoom firmware update (if/when one ships), confirm the
  patch survives — document if it doesn't.

## Out of scope

- Replacing adbd or installing a different adb implementation.
- Anything that involves rooting or modifying the bootloader.
