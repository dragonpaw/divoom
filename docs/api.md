# Divoom Times Frame API

A living reference. Update in the same commit as any change that exercises new device behavior.

Our test device:

| field        | value          |
|--------------|----------------|
| DeviceName   | `Timesframe`   |
| DeviceId     | `300380193`    |
| Hardware     | `510`          |
| MAC          | `4c37deff59df` |
| LAN IP       | `10.0.2.108`   |

Upstream source of truth (when Divoom doesn't lie): [`https://docin.divoom-gz.com/web/#/5/<page_id>`](https://docin.divoom-gz.com/web/#/5/). The portal renders client-side via Vue + XHR — plain HTTP scrapes return an empty SPA shell. Fetch the underlying JSON instead:

```sh
# one page
curl 'https://docin.divoom-gz.com/server/index.php?s=/api/page/info&page_id=374'

# the whole catalog tree (TimeFrame is item_id=5, cat_id=52)
curl 'https://docin.divoom-gz.com/server/index.php?s=/api/item/info&item_id=5'
```

We keep frozen snapshots of every TimeFrame page in [`docs/upstream/`](upstream/) so we can diff future Divoom edits against what we read when this doc was written.

---

## TL;DR — what actually works

| Concern | Verdict |
|---|---|
| Send the frame a custom layout (background + widgets) | ✅ via local API `:9000` |
| Patch text contents in place, no re-layout | ✅ via `Device/UpdateDisplayItems` |
| Frame fetches our **LAN** URL for `NetData` / `Image` / background | ❌ — these are cloud-proxied (see [Empirical findings](#empirical-findings)) |
| Frame fetches a **public** URL for `NetData` / images | ✅ |
| Push static files to the device for offline use | likely ✅ via adb-push to `/userdata/` (p358); not yet exercised |

Architectural consequence: **dynamic data goes via `Device/UpdateDisplayItems`** (text patches over the LAN API); **assets must come from publicly-reachable URLs** (Cloudflare Tunnel, public web hosting, …) **or `/userdata/` local files** pushed via adb.

---

## Discovery

`GET https://app.divoom-gz.com/Device/ReturnSameLANDevice`

Cloud endpoint that returns every Divoom device whose WAN IP matches the caller's. Acts as a LAN-membership probe. **GET and POST both work**; no body needed.

```json
{
  "ReturnCode": 0,
  "ReturnMessage": "",
  "DeviceList": [
    {
      "DeviceName": "Timesframe",
      "DeviceId": 300380193,
      "DevicePrivateIP": "10.0.2.108",
      "DeviceMac": "4c37deff59df",
      "Hardware": 510
    }
  ]
}
```

Quirks:

- **Server-side dedup / freshness window**. Two GETs in quick succession from the same source can return an empty `DeviceList` on the second. Cache the result; never poll.
- Discovery is the only thing we call against Divoom's cloud once we have the device's IP. Treat it as a one-shot lookup at startup, with a long cache and a re-discover-on-failure fallback.
- **2026-05-22: endpoint observed unreachable** — `app.divoom-gz.com` failed DNS/connect from two independent vantage points. The local device API on the frame itself kept working fine. Reinforces the need for a local-only discovery fallback ([todo](../todo/local-discovery.md)) since the cloud endpoint is a single point of failure outside our control.

---

## Local device API protocol

`GET http://<frame-ip>:9000/divoom_api`, `Content-Type: application/json`, JSON body whose `Command` field selects the operation.

All responses share an envelope:

```json
{ "ReturnCode": 0, "ReturnMessage": "" }
```

Non-zero `ReturnCode` means the device rejected the command. Some responses additionally echo `DeviceId`, `PacketFlag`, `DeviceType`; those can be ignored.

### Quirks of the protocol

- **GET with a JSON body** is non-standard. Many HTTP clients refuse to send a body on GET. `curl -G` will *not* send a body; use `curl -X GET --data-raw '…'`. Go's `net/http` is fine — pass a body reader to `http.NewRequestWithContext(..., MethodGet, ...)`.
- **Invalid commands return `{"ReturnCode": 1, "ReturnMessage": "Only accept JSON parameters"}`** even when the JSON is perfectly valid. The error text is misleading — it really means "I don't recognize that Command string for this device." We've reproduced this with `Channel/GetIndex` (a real Pixoo command) on the Times Frame. Treat "Only accept JSON parameters" as "command not supported on Times Frame."
- **Field-name typos in Divoom's docs are real and must be preserved**. Most painful: `BackgroudImageAddr` (no `n`) and `BackgroudImageLocalFlag` in `Device/EnterCustomControlMode`. Sending the correctly-spelled `BackgroundImageAddr` silently does the wrong thing — mirror Divoom's typos exactly in any payload.
- **Method is GET for state-changing commands**. Yes, really.

---

## Command URL & format (p358, p24)

Per the docs:

> You can use adb to push files to the device so that local files can be used on custom dials.
>
> Requested URL: `IP:9000/divoom_api`
> Method: GET
> Body: JSON with a `Command` field.

All commands accept JSON only (p24).

---

## Custom Control (cat_id=56)

The headline category. This is how we install custom dashboards.

### `Device/EnterCustomControlMode` (p374)

Installs a layout: one 800×1280 background plus a `DispList` of layered display elements.

| Field                       | Type    | Notes |
|-----------------------------|---------|-------|
| `Command`                   | string  | `"Device/EnterCustomControlMode"` |
| `BackgroudImageLocalFlag`   | number  | `0` = `BackgroudImageAddr` is a URL; `1` = it's an absolute path on the device, e.g. `/userdata/clock_bg.jpg`. |
| `BackgroudImageAddr`        | string  | Image must be exactly **800×1280** portrait. JPG and PNG both work for the docs example URL. **Must be a publicly-reachable URL** when `LocalFlag=0`; LAN URLs do not work. |
| `DispList`                  | array   | Per-type limits: ≤6 `Text`, ≤10 `Image`, ≤6 `NetData`. Other types appear unconstrained. |

#### `DispList` elements

Every element has the geometry fields:

| Field      | Type   | Notes |
|------------|--------|-------|
| `ID`       | number | >0. **Originally believed to key a property cache (2026-05-21); superseded 2026-05-25 — the cache is actually keyed on (Type, position-in-DispList). See Empirical findings rows 2026-05-21 and 2026-05-25.** |
| `Type`     | string | See below. |
| `StartX`   | number | Canvas X coordinate. |
| `StartY`   | number | Canvas Y coordinate. |
| `Width`    | number | Display area width. |
| `Height`   | number | Display area height. |
| `Align`    | number | `0`=left, `1`=right, `2`=middle. |

Per-type additional fields:

| Type          | Fields | Behavior |
|---------------|--------|----------|
| `Text`        | `FontSize`, `FontID`, `FontColor`, `BgColor`, `TextMessage` | Static text. **Updatable in place via `Device/UpdateDisplayItems`** — primary mechanism for dynamic data without re-sending the layout. |
| `Image`       | `Url`, optional `ImgLocalFlag` (`1` = `/userdata/…`) | URL or local path. **Cannot be patched live** — re-send the whole layout to change. |
| `NetData`     | `Url`, `RuleInfo`, `RequestTime`, font/color | Frame polls `Url` every `RequestTime` seconds (≥10), extracts a value via `RuleInfo`, renders as text. **Cloud-proxied — public URLs only** (see findings). |
| `Time`        | font/color | Built-in HH:MM clock. Self-updating. |
| `Date`        | font/color | Built-in. |
| `MonYear`     | font/color | `"2025-08"` format. |
| `Mday`        | font/color | Day-of-month integer. |
| `Year`        | font/color | Year integer. |
| `Month`       | font/color | Month integer. |
| `Week`        | font/color | Weekday. |
| `Weather`     | `Url` to a 10-frame webp | Frame selects which frame based on its current weather state. Frames in order: sunny day, cloudy day, rainy day, snow day, fog day, sunny night, cloudy night, rainy night, snow night, fog night. |
| `Temperature` | font/color | The device's current weather widget temperature. |

Colors are CSS hex strings, e.g. `"#ebdbb2"`. `BgColor` paints behind the element.

#### Minimal example (URL background)

```json
{
  "Command": "Device/EnterCustomControlMode",
  "BackgroudImageLocalFlag": 0,
  "BackgroudImageAddr": "https://example.com/bg.jpg",
  "DispList": [
    {
      "ID": 1,
      "Type": "Time",
      "StartX": 50, "StartY": 480, "Width": 700, "Height": 220,
      "Align": 2,
      "FontSize": 180, "FontID": 52,
      "FontColor": "#ebdbb2", "BgColor": "#1d2021"
    }
  ]
}
```

#### Local-file example

```json
{
  "Command": "Device/EnterCustomControlMode",
  "BackgroudImageLocalFlag": 1,
  "BackgroudImageAddr": "/userdata/clock_bg.jpg",
  "DispList": [ … ]
}
```

The full upstream example (with a dozen elements, NetData rows, all the types) lives in [`upstream/p374.json`](upstream/p374.json).

### `Device/UpdateDisplayItems` (p377)

Patch the `TextMessage` of existing `Text` elements without re-sending the layout. **Image elements cannot be patched this way.** This is how all dynamic data should flow — daemon polls its sources, then pushes text updates over the local API.

```json
{
  "Command": "Device/UpdateDisplayItems",
  "DispList": [
    { "ID": 13, "TextMessage": "hello world!" }
  ]
}
```

### `Device/ExitCustomControlMode` (p375)

Restores the previously-selected preset dial. Always pair with `EnterCustomControlMode` — including on Ctrl+C — so test runs never leave the device stuck.

```json
{ "Command": "Device/ExitCustomControlMode" }
```

---

## System settings (cat_id=54)

### `Channel/SetBrightness` (p362)

```json
{ "Command": "Channel/SetBrightness", "Brightness": 100 }
```

`Brightness`: 0–100. We've observed our frame at 90 by default.

### `Sys/LogAndLat` — set weather location (p363)

> All weather data comes from openweathermap.org.

```json
{ "Command": "Sys/LogAndLat", "Longitude": "30.29", "Latitude": "20.58" }
```

Note: `Longitude`/`Latitude` are **strings**, not numbers. Drives the built-in `Weather` and `Temperature` elements.

### `Channel/OnOffScreen` (p367)

```json
{ "Command": "Channel/OnOffScreen", "OnOff": 1 }
```

`OnOff`: `1` = on, `0` = off.

### `Device/SetMirrorMode` (p368)

```json
{ "Command": "Device/SetMirrorMode", "Mode": 0 }
```

`Mode`: `0` = disable, `1` = enable. **Resets on power-off** — not persistent.

### `Device/SetTime24Flag` (p369)

```json
{ "Command": "Device/SetTime24Flag", "Mode": 0 }
```

`Mode`: `1` = 24-hour, `0` = 12-hour. **Resets on power-off.**

---

## Dial control (cat_id=55)

### `Channel/MyClockGetList` — list available dials (p360)

⚠️ Cloud endpoint, not local. Requires the cloud's view of which dials this device has.

```
GET https://appin.divoom-gz.com/Channel/MyClockGetList
```

| Field         | Type   | Notes |
|---------------|--------|-------|
| `DeviceId`    | number | The device ID from discovery. |
| `StartNum`    | number | ≥1, paginated. |
| `EndNum`      | number | ≥1. |
| `DeviceType`  | string | Must be `"Frame"` for Times Frame. |

Returns `ClockList`: `[{ClockId, ClockName, ImagePixelId, ClockType, Position}]`. `ImagePixelId` is prefixed with `https://fin.divoom-gz.com/` (sic, the docs alternately say `f.divoom-gz.com`) to get the preview image URL.

The list is per-device — what's installed depends on what the user has favorited/purchased. Our device's full list is not snapshotted here because it's account-specific.

### `Channel/SetClockSelectId` — switch dial (p361)

Local API. Selects a preset dial by `ClockId`.

```json
{ "Command": "Channel/SetClockSelectId", "ClockId": 923 }
```

Our test device was on `ClockId 923` when we started.

---

## Tools (cat_id=53)

These control built-in widget tools — when set, the frame enters the corresponding tool screen.

### `Tools/SetTimer` — countdown (p370)

```json
{ "Command": "Tools/SetTimer", "Minute": 1, "Second": 0, "Status": 1 }
```

`Status`: `1` = start, `0` = stop.

### `Tools/SetStopWatch` — stopwatch (p371)

```json
{ "Command": "Tools/SetStopWatch", "Status": 1 }
```

`Status`: `2` = reset, `1` = start, `0` = stop.

### `Tools/SetScoreBoard` — scoreboard (p372)

```json
{ "Command": "Tools/SetScoreBoard", "BlueScore": 100, "RedScore": 79 }
```

Scores: 0–999.

### `Tools/SetNoiseStatus` — noise/ambient (p373)

```json
{ "Command": "Tools/SetNoiseStatus", "NoiseStatus": 1 }
```

`NoiseStatus`: `1` = start, `0` = stop.

---

## Misc (cat_id=52)

### `Device/SysReboot` (p359)

```json
{ "Command": "Device/SysReboot" }
```

> It will work at V510086.

Our test device reports `Hardware: 510` — likely matches but we haven't exercised this.

### Command URL & format (p358)

See [Command URL & format](#command-url--format-p358-p24) above. Notes that adb-push'd files in `/userdata/` can be referenced by custom dials with `*LocalFlag: 1`.

---

## NetData / `RuleInfo` syntax (p145)

`RuleInfo` is a dotted path through the JSON response, comma-separated, with the leaf prefixed `n:` (number) or `s:` (string). Case-sensitive.

```
n:Score                              top-level number "Score"
s:Nickname                           top-level string "Nickname"
dispData,n:LikeCnt                   dispData.LikeCnt as number
RetData,UserInfo,dispData,n:LikeCnt  deeply nested
```

`RequestTime` minimum is 10 seconds. The frame ignores lower values.

**Coverage caveat**: NetData returns a *single scalar* per element. For multi-value widgets (sparkline, list of tickers, weather strip) you must either burn one slot per scalar (max 6 per layout) or render server-side as a PNG and use an `Image` element instead (also URL-based, also cloud-proxied — but cacheable).

---

## Fonts (p379)

`POST https://appin.divoom-gz.com/Device/GetTimeDialFontV2`

Returns the device's font catalog as `FontList: [{id, type, url, charSet, Encryption}]`. We've saved a full snapshot at [`docs/fonts.json`](fonts.json) and a human-readable index at [`docs/fonts.md`](fonts.md).

Cardinality (snapshot 2026-05-20):

- 159 font entries, IDs 6–364 (sparse).
- Extensions: `.ttf`, `.otf`, `.bin` (probably proprietary glyph packs).
- ~73% of entries have an `Encryption` SHA-1; these may be DRM-protected and unusable as raw TTFs.

The docs examples use FontID `52` and `126`. FontID `52` is what we've successfully rendered with in our display tests.

Custom fonts (Iosevka, Roboto Condensed) likely require adb-push'ing TTFs onto the device's font directory and identifying their assigned IDs. Not yet exercised — see TODO in `docs/adb.md` (when we have one).

---

## Empirical findings

Things we learned from running real tests against the device — distinct from what the docs claim.

| Date       | Finding | Evidence |
|------------|---------|----------|
| 2026-05-20 | The local API does respond to GET-with-JSON-body. | First display test rendered a Time element. |
| 2026-05-20 | `Channel/GetClockInfo` returns `{ClockId, Brightness}`. | Probe returned `ClockId: 923, Brightness: 90`. |
| 2026-05-20 | Invalid Command strings return `{"ReturnCode": 1, "ReturnMessage": "Only accept JSON parameters"}`. | `Channel/GetIndex` (Pixoo cmd) returned that error. |
| 2026-05-20 | Discovery responds to GET as well as POST, with empty body. | `curl https://app.divoom-gz.com/Device/ReturnSameLANDevice` returned full device list. |
| 2026-05-20 | The frame fetched the docs example `BackgroundImageAddr` from `f.divoom-gz.com` correctly. | First display test rendered the docs' example background image (looked like a "score" Divoom profile dial). |
| 2026-05-20 | **NetData URLs are fetched server-side by Divoom's cloud, not from the device.** | 3-way probe: docs example URL → rendered `2101`; `jsonplaceholder.typicode.com/posts/1` → rendered `1`; `http://10.0.2.2:8080/probe.json` → screen rendered "Err URL" while our local HTTP server saw **zero** TCP connects from `10.0.2.108`. ICMP and ARP confirm L2 reachability — only a server-side proxy explains the asymmetry. |
| 2026-05-20 | Self-hosted `BackgroundImageAddr` URLs over HTTP fail silently (same cloud-proxy root cause). | Self-hosted `http://10.0.2.2:<port>/bg.{png,jpg}` never reached our server during multiple test runs. |
| 2026-05-20 | `adb connect 10.0.2.108:5555` returns "Connection refused". | Tried over TCP; adb-over-network not enabled by default. |
| 2026-05-22 | **`adb tcpip 5555` from the USB host enables adb-over-network, but does NOT persist across a power cycle.** After reboot the device is back on USB-only adbd; the TCP listener is gone until `adb tcpip 5555` is re-issued from a USB host. Unblocks LAN-side adb workflows *while the current session lasts*, but unreliable as a primary discovery path. A persistent fix would require editing the TinaLinux init scripts on the device. | Issued `adb tcpip 5555` over USB; adbd restarted in TCP mode. After unplug/replug, `adb devices` no longer showed the TCP entry; on power cycle, the device returned as USB-only. |
| 2026-05-22 | **Init system is OpenWrt-style procd.** `/etc/init.d/adbd` is a `#!/bin/sh /etc/rc.common` script using `procd_set_param env / command`, controlling adbd at `/bin/adbd` with `-D`. Contains a commented `#ADB_TRANSPORT_PORT=5555` line that *looks* like the natural switch to enable TCP adbd. | `adb pull /etc/init.d/adbd`; readable contents per [todo/adb-tcpip-persist.md](../todo/adb-tcpip-persist.md). |
| 2026-05-22 | **DO NOT just uncomment `ADB_TRANSPORT_PORT=5555` in `/etc/init.d/adbd` and reboot.** Doing so killed adbd entirely after the next power cycle — frame was unreachable on both USB and TCP. The Divoom app and local `divoom_api` on port 9000 kept working (independent service), so the device wasn't bricked, but adb access was gone until a **factory reset via the Divoom app** wiped the overlay filesystem and restored the original init script. | Pushed patched `adbd` (one-line diff: uncomment), power-cycled, adbd unreachable on both transports. Recovered via Divoom app's factory reset. |
| 2026-05-22 | **Cloud proxy whitelists only `f.divoom-gz.com` for `Image` element URLs.** Image elements work end-to-end on the Times Frame (verified with `divoom display image` against the docs' example URL — image renders alongside a Time element as expected). But the same subcommand against `nasa.gov`, `thecocktaildb.com`, `raw.githubusercontent.com`, and `upload.wikimedia.org` all silently fail — the device renders the background + other elements but the Image slot stays blank with no error. The cloud proxy must enforce a host whitelist that only covers Divoom's own CDN. Workaround for third-party images: download server-side, `adb push` to `/userdata/...`, reference with `ImgLocalFlag: 1`. | `display image` against five hosts; only the `f.divoom-gz.com` URL rendered. |
| 2026-05-22 | **Image DispElements require Font/Color fields even though they're semantically meaningless for images.** Our nasa+cocktail Image elements originally omitted `FontSize/FontID/FontColor/BgColor` and the `omitempty` JSON tags dropped them from the payload; the frame rendered the background + other elements but skipped the Image slot. The docs' working example (`docs/upstream/p374.json`) sets these on every element regardless of type, including Image. Adding stub values to Image elements (`scenes_*.go`) is necessary for the device parser to accept them. | nasa/cocktail scenes never rendered an image until `b9b3ac2` added stub Font/Color fields. |
| 2026-05-22 | **adb-over-TCP is not feasible on this device without replacing the adbd binary.** Confirmed via a sidecar `/etc/rc.local` investigation pass that this `/bin/adbd` (55 KB ARM ELF, recognises `ADB_TRANSPORT_PORT`, `ADB_AUTH_ENABLE`, etc.) accepts the `tcpip:5555` service request (host says `restarting in TCP mode port: 5555`) but then **fails to actually bind any TCP transport listener** — the USB adbd dies during the restart and nothing on the LAN responds. Currently-listening ports per `/proc/net/tcp` are 9000 (divoom_api), 5037 (adbd smart-socket bound to 0.0.0.0), and 5005 (mystery); 5037 and 5005 both accept TCP but speak no adb-transport protocol, so `adb connect host:PORT` returns "offline". Conclusion: this adbd build's TCP transport is broken (likely a TinaLinux-specific build with TCP support compiled out or wired to USB-only). Use the `DIVOOM_FRAME_IP` static-IP discovery path instead; keep USB for font installs and other on-device pokes. | rc.local sidecar dump of adbd args/env + listening sockets; `adb tcpip 5555` followed by `adb connect 10.0.2.202:5555` → connection refused; USB device disappeared post-restart. |
| 2026-05-22 | **Cloud discovery `app.divoom-gz.com` observed unreachable.** DNS/connect failed from two independent vantage points. Local device API on the frame kept working. Reinforces the need for local-only discovery. | Two simultaneous `curl` failures; no resolution. |
| 2026-05-21 | **adb-over-USB works.** Device shows up in `adb devices -l`, serial `20080411`. | USB-C cable from host to frame. |
| 2026-05-21 | Device runs **TinaLinux + BusyBox 1.33.2**, not Android. | `adb shell` falls into `/bin/login`; `getprop` and other Android-isms don't exist. |
| 2026-05-21 | `adb shell` is gated by a login prompt we don't have credentials for. | Banner: `TinaLinux login:`. `adb shell -c …` errors with `/bin/login: invalid option -- 'c'`. |
| 2026-05-21 | **`adb push` / `adb pull` work without authentication** — they bypass the shell. | 13-byte round-trip clean; 26 KB JPG pushed in <1 ms. |
| 2026-05-21 | `BackgroundImageLocalFlag: 1` + `/userdata/<file>.jpg` works end-to-end. | Pushed gruvbox test pattern; frame rendered it with all 4 corner dots visible, midline cross intact, color swatches present along the bottom. |
| 2026-05-21 | `Device/ExitCustomControlMode` alone leaves the previously-selected dial **half-rendered** — background returns but the dial's widgets don't re-initialize. Following Exit with `Channel/SetClockSelectId` to the same dial ID fixes it. | First `display test` left "pixel arcade" preset visible without any text. Capturing `ClockInfo.ClockId` before entering custom mode and re-selecting on exit produces a clean restore. |
| 2026-05-21 | Device's preset images are **800×1280 webp**, not JPG. JPG is also accepted (we proved both ways), but webp is the native format. | `file app_pic/default_pic/1@1x.webp` → `Web/P image, ICC profile, 800x1280`. |
| 2026-05-21 | Device phones home via MQTT (topic `divoom/2/DeviceHeart`). | Visible in `/userdata/debug.txt`. Confirms that `NetData` / image URL fetches are proxied through Divoom's MQTT-backed cloud, not done from the device. |
| 2026-05-21 | **Superseded 2026-05-25 — see the row dated 2026-05-25 for the corrected cache-key model. Original claim retained for historical accuracy:** **`EnterCustomControlMode` caches element properties by `ID`. Re-installing with the same ID only updates `TextMessage` — `FontSize`, `Height`, `StartY`, etc. retain the first install's value.** Calling `Device/ExitCustomControlMode` first clears the cache, after which the next install's declared geometry takes effect. | Probe via `divoom display lines`: running back-to-back with `-font=20` then `-font=60` (same ID 10) showed identical text size on the wall. Running each preceded by `ExitCustomMode` produced visibly different sizes. The same caching effect explained the live scene driver's "stuck" body geometry — Height edits in code never reached the device. |
| 2026-05-21 | At `FontSize=34` with `Width=760`, the device wraps text at roughly **45 px per line** within a Text element's declared `Height`. There is *no* hidden per-element line cap — `Height` directly determines how many wrapped lines render. | Probe data from `divoom display lines` (with ExitCustomMode reset): `-height=1280 -n=40` rendered through L28 (28 lines × ~46 px = 1280). `-starty=540 -height=480 -n=20` rendered through L12 (12 × ~40 px = 480). Earlier "cap at ~y=1024 / ~10 lines" observations were the ID-cache bug, not a rendering limit. |
| 2026-05-21 | **Roboto Condensed Light installed as FontID 11.** Third custom font, same workflow as Iosevka (ID 7) and Roboto Condensed Regular (ID 9). RobotoCondensed-Light.ttf (511,264 bytes, from `https://github.com/googlefonts/roboto/raw/main/src/hinted/RobotoCondensed-Light.ttf`) pushed to `/usr/share/divoom_app/divoom/21/12.bin`; new `{"id": 11, "type": 1, "url": "group1/M00/02/AC/roboto_condensed_light_local.ttf", "charset": ""}` entry added to `/divoom-config/system/font_list.cfg` (slot between catalog IDs 10 and 12). Daemon reload via the `Device/GetTimeDialFontV2` crash-and-restart trick succeeded; post-restart a `Device/EnterCustomControlMode` render at `FontID: 11, FontSize: 80` returned `ReturnCode: 0` with no error. Pre-modification cfg backup at [`docs/upstream/font_list_device_system_pre_roboto_light_backup.cfg`](upstream/font_list_device_system_pre_roboto_light_backup.cfg). | End-to-end install + verify per the existing "Custom font workflow" procedure. |
| 2026-05-23 | **The ID-property cache (row above, 2026-05-21) leaks across scene rotations even with the documented workarounds.** `internal/scene/scene.go` defends two ways: `pick()` excludes scenes whose `len(Elements)` matches the previous scene's, and the driver applies a `+ seq*100` ID offset per install so no ID is ever reused. Both together still let the cache pollute — observed on `forecast` (idSceneMain inherited FontSize ~180 from a prior scene), `markets` (idSceneSub3 sparkline wrapped at oversized cached FontSize), and `weather` (idSceneMain 220pt headline rendered as small top-left text instead of the giant centred number). In every case the only reliable reset was the divoom_app crash-restart at the end of `divoom push` (font reload). Open hypotheses: the cache key isn't just ID (possibly `(type, position-in-DispList)`); `pick()` only excludes against the immediately-previous scene, not transitively; the defensive-fallback branch can bypass the count rule. Worth a real audit — currently the workaround is "push when geometry looks wrong." | Real-user reports + screenshots; correlated with scene-switch sequences. |
| 2026-05-23 | **Pushing fonts from the container entrypoint causes an infinite serve-crash loop.** `divoom push` always ends with `reloadFontsViaCrashRestart` (intentional crash of `divoom_app` to reload `font_list.cfg`), making `:9000/divoom_api` unreachable for ~30s. If push runs on container start, `divoom serve` then fails to reach the frame and crash-exits; Docker `restart: unless-stopped` brings the container back; entrypoint re-runs push; repeat. Solution: entrypoint must be `exec /usr/local/bin/divoom "$@"` only, with no pre-serve push step. Daily bg refresh wants a NAS-side cron calling `docker exec divoom-dashboard divoom push`, not entrypoint logic. | Observed afternoon of 2026-05-23 — 5-minute push churn for ~10 minutes before noticed. |
| 2026-05-25 | **The element-property cache (row 2026-05-21) is keyed on (Type, position-in-DispList), NOT on element ID.** The previous record at 2026-05-21 claimed ID-based caching ("re-installing with the same ID only updates TextMessage"). That was wrong — it's the *position* in DispList, regardless of ID, that determines whether the device treats an element as "re-use cached props". Verified via paired Enter installs with `--no-reset` chained: (E1) different IDs at same count=1 → font unchanged from prior install; (E2) count change 1→2 → both positions render at declared values; (E3) same count=2 with new IDs → font unchanged; (E4) count change 2→1 → declared values apply; (E5) round-trip 2→1→2 → fully resets (no per-count history). Filler element experiment confirmed: a non-Text built-in (`Type: "Year"`, off-screen 1×1) added to DispList counts toward total length for cache purposes and busts cache on length change. The Divoom docs (p374) explicitly claim ID-based overwrite — they lie about this device's actual behavior. | Six paired-install probes from a Docker container with `display lines --no-reset`, observed visually on the frame. |
| 2026-05-25 | **6-Text cap is real and silent.** Sending a DispList with 7 Text elements returns `ReturnCode: 0` (no error) but only the first 6 render — the 7th is silently dropped. Use built-in Types (`Year`, `Week`, etc.) for filler elements that don't count toward the Text cap. | Direct curl POST with 7 colored Text rows; visual count showed 6. |
| 2026-05-25 | **Over-cap Text elements still count in the device's cache-key length.** Sent 7 Text (only 6 rendered), then 6 Text at a different FontSize — the new FontSize took effect, meaning the device's cache saw the prior install as length=7 and the new one as length=6, so the length-change reset rule fired. Useful for understanding cache mechanics; the engineering choice still uses a `Year` filler (structurally distinct from body Text) over an empty Text filler. | Two paired curl POSTs; visible font-size change confirmed cache reset. |
| 2026-05-25 | **`Driver.install()` now appends a cache-buster filler element** when the new DispList length would equal the previous install's length. The previous `+seq*100` ID-offset defence is removed (dead code given the corrected cache-key understanding above). `pick()`'s count-different exclusion remains as belt-and-suspenders. Filler is a 1×1 off-screen built-in `Year` element. | Driver edit landed 2026-05-25 after the probe sequence above. |

---

## adb-push: pushing assets to the device

The Times Frame is a TinaLinux/BusyBox board. Interactive `adb shell` is gated by a login prompt we don't have credentials for; trying `adb shell -c '…'` errors out because BusyBox's `login` doesn't accept `-c`. **But `adb push` and `adb pull` go through adbd's filesystem channel and don't need a shell login**, so we can freely move files into `/userdata/` and reference them with `*LocalFlag: 1`.

```sh
# enable over-USB (no settings change needed; just plug a USB-C cable)
adb devices -l                       # confirms the device appears
adb push my_bg.jpg /userdata/my_bg.jpg
adb pull /userdata/clock_bg.jpg ./   # round-trip works too
```

The full `/userdata/` tree (paths only, no binary contents) is in [`docs/userdata-tree.txt`](userdata-tree.txt). Notable directories:

- `app_pic/default_pic/` — the 91 webp images and 6 mp4 animated dials Divoom ships as preset dial backgrounds.
- `divoom_thumb/` — dial preview thumbnails.
- `divoom_cloud_photo/` — where the Divoom app's "upload to your frame" puts user photos.
- `app_api/`, `app_para/` — empty in our snapshot; presumably runtime state.
- `debug.txt`, `bt_debug.txt`, `divoom_run.log` — readable runtime logs (visible via `adb pull`).
- `pic_db.bin`, `strings.xml` — image catalog binary + localization strings.

Our convention (proposed): put everything we push under a single root like `/userdata/wallclock_*` (or a sub-tree `/userdata/wallclock/`) to keep our stuff visibly separate from Divoom's. We haven't yet confirmed whether sub-paths under `/userdata/` are honored by the image loader — the docs example uses files at the root.

## Fonts on disk

The 159 fonts in the catalog (p379) are **not pre-cached on the device**. They are downloaded from `https://f.divoom-gz.com/<url-fragment>` to the device's local cache on first use of each `(FontID, FontSize)` pair. Probing the running device's filesystem against every catalog ID found:

- **Cache path**: `/usr/share/divoom_app/divoom/21/<file_id>.bin` (directory `21` is the font sub-cache; other numbered subdirs hold image/asset caches — e.g. dir `15` is JPEG dial backgrounds, dir `28` is something else compressed, dir `29` is animated webp). The path lives on the overlay filesystem, so writes go to `/overlay/upper/usr/share/divoom_app/divoom/21/` and persist across power cycles.
- **Filename convention**: `<catalog_font_id + 1>.bin`. The `+1` offset is consistent across every entry we sampled (font 6 → `7.bin`, font 8 → `9.bin`, font 286 → `287.bin`, font 358 → `359.bin`, font 364 → `365.bin`). We did not derive the offset from any data structure — it is just what the device writes.
- **File format**: **plain TTF**, despite the `.bin` extension. `file(1)` reports `TrueType Font data` (matches the catalog `type:1` entries). For `type:0` entries (image-font glyph packs with a `charSet`) the `.bin` is the device's proprietary compressed glyph format; we have not reverse-engineered it.
- **Integrity check**: the `Encryption` field in p379 is the SHA-1 of the file. Despite the field name, it is a checksum, not a key. We verified by `sha1sum` matching the cached file against the catalog field for multiple IDs.
- **No DRM that we can find**: the cached `.bin` for entries with a non-empty `Encryption` field is just the raw TTF; FreeType (`libfreetype.so.6.17.0` is loaded by `divoom_app`) opens it directly.

### Custom font workflow (verified end-to-end with Iosevka 34.5.0 and Roboto Condensed)

`divoom_app` reads `/divoom-config/system/font_list.cfg` exactly once at startup (function `divoom_system_font_init`), then keeps the parsed list in RAM forever. Pushing a `.bin` to the cache directory is necessary but **not sufficient** — the device also needs an entry in `font_list.cfg` for the FontID, or the runtime falls back to the cloud (`Device/GetSomeFontInfoV2` over MQTT) and gets `FontList: []` for an unknown ID, then loops retrying.

The procedure that worked:

1. Back up `/divoom-config/system/font_list.cfg` before changing it. (Snapshots taken 2026-05-21 of all three on-device copies — `/divoom-config/font_list.cfg`, `/divoom-config/system/font_list.cfg`, `/usr/share/divoom_app/font_list.cfg` — are in [`docs/upstream/font_list_device_*backup.cfg`](upstream/).)
2. Pick an unused FontID (we used `7` because it's in the gap between catalog IDs 6 and 8). `adb push <local.ttf> /usr/share/divoom_app/divoom/21/<id+1>.bin`.
3. Edit `font_list.cfg` to add an entry `{"id": <id>, "type": 1, "url": "...placeholder...", "charset": ""}`. The `url` field can be anything — the device only fetches the URL if the cached file is missing. `update_time` can be bumped or left alone; `check_flag: 2` must remain.
4. `adb push` the modified `font_list.cfg` to `/divoom-config/system/font_list.cfg`.
5. **Force `divoom_app` to re-read the config**. Without an ssh shell we can't `kill -HUP` or restart it cleanly. Empirically, sending `{"Command": "Device/GetTimeDialFontV2"}` to the local API at `:9000/divoom_api` causes `divoom_app` to crash; `procd` restarts it within ~5 seconds, and `divoom_system_font_init` reads the new config (boot log now shows `font num: 160` instead of `font num: 159`). The restart restores the previously-selected dial. **This is fragile and ugly — but it works.** It is the only known way to refresh `font_list.cfg` short of a power cycle. **NEVER rely on this on a production frame** — `Device/GetTimeDialFontV2` against the local API is undocumented behavior that happens to crash the process; future firmware updates may not crash and instead silently misbehave.
6. Send a render with `FontID: <your id>`. The log will show `we will remove font: <id>,<size>,...` on subsequent layout changes, confirming FreeType loaded our TTF at the requested size with no `Error loading font file:` message and no `GetSomeFontInfoV2` retry storm.

End-to-end: Iosevka-Regular.ttf (10 MB) pushed as `/usr/share/divoom_app/divoom/21/8.bin`, registered as FontID 7 in `font_list.cfg`, and rendered with `Device/EnterCustomControlMode` at FontSize 120 — the device accepted the layout cleanly (no error log lines, font tracking confirmed `font: 7,120` was in the LVGL cache after render).

Re-verified 2026-05-21 with RobotoCondensed-Regular.ttf (505 KB, from `https://github.com/googlefonts/roboto/raw/main/src/hinted/RobotoCondensed-Regular.ttf`) pushed as `/usr/share/divoom_app/divoom/21/10.bin` and registered as FontID 9 (gap in catalog between IDs 8 and 10). Post-reload `divoom_system_font_init--714` logged `font num: 161` (up from 160 after the Iosevka install), and a `Device/EnterCustomControlMode` render at FontSize 80 with `FontID: 9` returned `ReturnCode: 0` with no `Error loading font file:` or `GetSomeFontInfoV2` retry-storm entries. The procedure transferred 1:1 from Iosevka; no per-font tweaks needed. Pre-modification backup of the on-device cfg is at [`docs/upstream/font_list_device_system_pre_roboto_backup.cfg`](upstream/font_list_device_system_pre_roboto_backup.cfg).

Re-re-verified 2026-05-21 with RobotoCondensed-Light.ttf (511,264 bytes, same Google Fonts hinted source) pushed as `/usr/share/divoom_app/divoom/21/12.bin` and registered as FontID 11 (gap in catalog between IDs 10 and 12). Daemon-reload crash-trick succeeded; post-restart `Device/EnterCustomControlMode` at FontSize 80 with `FontID: 11` returned `ReturnCode: 0`. Pre-modification cfg backup at [`docs/upstream/font_list_device_system_pre_roboto_light_backup.cfg`](upstream/font_list_device_system_pre_roboto_light_backup.cfg). Third successful application of the procedure with no modifications.

### Caveats and unknowns

- **Visual confirmation is missing**: we have no screenshot or screen-readback API, and no eyes-on report from the device for this test session. The absence of errors plus the LVGL cache trace strongly suggests Iosevka actually rendered, but a human still needs to look at the frame to confirm glyph shapes.
- **Roboto Condensed verified 2026-05-21** at FontID 9 (Regular) and FontID 11 (Light), both at FontSize 80. Same procedure applied without modification.
- **The cache directory is on the overlay filesystem.** A factory reset (`format user_data` / `format divoom_config` actions in `divoom_main.c`) would presumably wipe our pushed TTFs and config edits. We have not exercised this.
- **The crash-to-reload trick is not documented and not safe.** If we wanted a robust workflow we'd want to find a cleaner reload path (a `procd` signal, a hidden `Sys/Reload` command, etc.) — none found yet in the binary strings.
- **Font IDs outside `[1, 999]` and conflicts with future Divoom-issued IDs**: Divoom adds new fonts every few months (the catalog grew from ~120 to 159 entries within the last year). If we register FontID 7 locally, then Divoom later issues FontID 7, the next `Device/GetTimeDialFontV2` sync would overwrite our entry. Pick IDs well above the current max (`364`); we suggest the `1000+` range. We deliberately used `7` for this test to verify the "gap in catalog" case works.
- **`divoom_app` reads from `/divoom-config/system/font_list.cfg`** — the other two copies (`/divoom-config/font_list.cfg` at the root, `/usr/share/divoom_app/font_list.cfg` in the rom-shadow) appear to be staging/backup; modifying them did not affect the running process.

## Pointers

- Frozen JSON snapshots of every upstream TimeFrame page: [`docs/upstream/`](upstream/) (one file per `page_id`).
- Font list snapshot: [`docs/fonts.json`](fonts.json) (verbatim API response).
- Font readable index: [`docs/fonts.md`](fonts.md).
- Upstream catalog tree: `curl 'https://docin.divoom-gz.com/server/index.php?s=/api/item/info&item_id=5'`.
- A page by ID: `curl 'https://docin.divoom-gz.com/server/index.php?s=/api/page/info&page_id=<id>'`.
