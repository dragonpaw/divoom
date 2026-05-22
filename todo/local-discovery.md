# Local-only Times Frame discovery

Find the frame on the LAN without calling `app.divoom-gz.com`. The
existing cloud discovery is a single point of failure outside our
control (confirmed unreachable on 2026-05-22). The dashboard runs on
the NAS at 10.0.2.201, which is not USB-attached to the frame — so
adb-based discovery is out. Must be LAN-only.

## Approaches (in order of preference)

### 1. Static IP + health-check (simplest, ship first)

Read frame IP from a `DIVOOM_FRAME_IP` env var or config. On startup,
probe `http://<ip>:9000/divoom_api` with a known command (e.g.
`Channel/GetClockInfo` — currently returns `{ClockId, Brightness}`)
to confirm we have the right host. If the probe fails, fall back to
one of the active-scan methods below.

Pros: zero scan, deterministic, instant. Almost always the right
answer in a home LAN where DHCP reservations are easy.
Cons: requires one-time config; breaks if the DHCP lease changes
and there's no reservation.

### 2. adb-over-network probe (preferred fallback)

As of 2026-05-22 the frame is listening on TCP 5555 (`adb tcpip 5555`
was issued from a USB-attached host; should persist until reboot or
factory reset). Note: the device is TinaLinux + BusyBox, NOT Android,
and `adb shell` is login-gated (no creds) — so `getprop` won't work.
Identification has to use the auth-free transports:

```
adb connect <candidate-ip>:5555     # succeeds if adbd listening
adb -s <ip>:5555 pull /etc/os-release /tmp/probe-os    # auth-free path
grep -q TinaLinux /tmp/probe-os                        # positive ID
```

`adb pull` and `adb push` bypass the login prompt (confirmed in
`docs/api.md` empirical findings 2026-05-21), so a known Divoom
file path makes a clean fingerprint. Pick a file that's small,
always present, and Divoom-specific — `/usr/share/divoom_app/...`
something is a stronger signal than `/etc/os-release` which any
TinaLinux box would have.

Walk the /24 (or just try the last known IP first), `adb connect`
each candidate with a short timeout, accept the first whose
Divoom-specific pull succeeds. The adb-server has to be installed
on the NAS (alpine package `android-tools`), which is the main
cost.

Pros: unambiguous device ID; reuses tooling we'd want anyway for
font installs / debugging from the NAS.
Cons: requires `adb tcpip 5555` to have been run once on the device
(true today, but not guaranteed after a factory reset); requires an
adb client on the dashboard host.

### 3. ARP scan + port-9000 probe (zero-tooling fallback)

Walk the local /24 (derived from the NAS's own interface), do a
quick TCP-connect to port 9000 on each live host (ARP cache + a
parallel dialer is fastest), then on each open port send the
"Only accept JSON parameters" canary: an invalid `Command` like
`Channel/GetIndex`. The Times Frame's distinctive error envelope
(`{"ReturnCode": 1, "ReturnMessage": "Only accept JSON parameters"}`)
is a strong positive ID — no other service on a home LAN should
emit that exact body.

Pros: works with zero config, survives DHCP changes.
Cons: scan latency (a few seconds on a /24); slightly noisy; the
canary is fingerprinting our own device which feels indirect.

### 4. MAC OUI filter

The device MAC observed in the cloud discovery payload starts
`4c37de…`. If that's a Divoom-assigned OUI block (verify against
the IEEE registry), the ARP scan can be narrowed: pull `arp -an`
output, filter for the OUI, done. Faster than approach 2 and more
specific. Falls back to approach 2 if no OUI match (device may
have multiple OUI ranges across hardware revisions).

### 5. mDNS / SSDP probe (worth testing once)

Unknown whether the Times Frame advertises itself via mDNS
(`_http._tcp.local`, `_divoom._tcp.local`?) or SSDP / UPnP. Worth
a one-shot `avahi-browse -a` and an SSDP M-SEARCH from the NAS to
see if either turns anything up. If yes, this is the cleanest
option — zero scan, no fingerprinting. If no, skip.

## Recommendation

Ship **#1** first (env-var IP + health check). It's a five-minute
change and covers 99% of real use. Add **#2** (adb-over-network) as
the active-discovery fallback now that the device is listening on
5555. Keep **#3 + #4** as deeper fallbacks for the case where adb
gets disabled. **#5** is a 10-minute probe worth doing once to know
whether to skip it permanently.

## Out of scope

- USB adb — NAS isn't USB-attached.
- Router API integration (Unifi / pfSense / OpenWrt to read the
  DHCP lease) — works but is bespoke per router; not portable.
- Calling the cloud discovery as a fallback — defeats the point;
  the whole reason for this todo is that it goes down.

## Verify after implementing

- With `DIVOOM_FRAME_IP=10.0.2.108` set, startup uses that and
  skips any scan.
- With the env var unset (or pointing at a dead IP), the fallback
  scan finds the frame within a few seconds and logs the IP it
  picked.
- Block egress to `app.divoom-gz.com` (e.g. `/etc/hosts` override)
  and confirm startup still succeeds end-to-end.

## Status (2026-05-22)

Implemented approach **#1** only, per CLAUDE.md (ship the smallest thing
that works; add the rest when there's evidence #1 isn't enough):

- `connectToFrame` in `cmd/divoom/display.go` now probes
  `$DIVOOM_FRAME_IP` with `Channel/GetClockInfo` before trusting it. A
  set-but-unreachable IP logs a warning and falls through to cloud
  discovery.
- Cloud-failure error now tells the user to set `DIVOOM_FRAME_IP=<ip>`
  to skip the cloud lookup entirely.
- Added `TestProbeFrameIP_DeadAddressFailsFast` to confirm the probe
  actually runs (no blind trust of the env var).

Deferred (still listed above; revisit if/when #1 proves insufficient):

- **#2** adb-over-network probe — would need `android-tools` on the NAS
  and an adb-server side-channel. Not justified until we hit a case
  where the static IP changes and cloud is also down.
- **#3** ARP scan + port-9000 canary.
- **#4** MAC OUI filter.
- **#5** mDNS / SSDP probe.
