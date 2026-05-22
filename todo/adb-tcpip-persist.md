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

- **Bricking risk.** Editing init scripts on an embedded device
  can leave it unbootable. Always pull a backup of the original
  file before pushing the modified version, and keep a USB cable
  handy for recovery (USB adb works even when LAN doesn't).
- **Firmware updates.** A Divoom OTA might overwrite the overlay
  or wipe `/etc/init.d/` additions. If we lose the persistence
  after an update, re-apply.
- **Security.** adb-on-LAN means anyone on the LAN can `adb push`
  files to the device. Home LAN, so accept the risk; document the
  exposure.

## Verify after implementing

- Cold power cycle the frame; wait 60 s; `adb connect 10.0.2.108:5555`
  from the NAS succeeds without any USB intervention.
- A second reboot a day later still works.
- After a Divoom firmware update (if/when one ships), confirm the
  patch survives — document if it doesn't.

## Out of scope

- Replacing adbd or installing a different adb implementation.
- Anything that involves rooting or modifying the bootloader.
