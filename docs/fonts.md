---
name: Times Frame font list
description: Snapshot of `Device/GetTimeDialFontV2` taken 2026-05-20.
---

# Times Frame fonts

Snapshot of `POST https://appin.divoom-gz.com/Device/GetTimeDialFontV2` from 2026-05-20. Raw response is in [`fonts.json`](fonts.json).

## Stats

- **Total fonts**: 159
- **With Encryption hash (likely DRM)**: 116
- **Open (no Encryption)**: 43
- **ID range**: 6..364 (sparse)
- **Extensions**: `.bin` (34), `.otf` (5), `.ttf` (120)

## Known-good IDs

- `52` — successfully renders Time elements in our display tests.
- `126` — referenced in the docs example (p374) but not yet exercised.

## Locally-installed custom fonts

These IDs are **not in the cloud catalog** — they are adb-pushed and registered in `/divoom-config/system/font_list.cfg` on our device only. See [`api.md` → "Custom font workflow"](api.md).

| ID | Font | TTF size | File on device |
|---:|------|---------:|----------------|
| 7  | Iosevka Regular            | 10 MB      | `/usr/share/divoom_app/divoom/21/8.bin`  |
| 9  | Roboto Condensed Regular   | 505 KB     | `/usr/share/divoom_app/divoom/21/10.bin` |
| 11 | Roboto Condensed Light     | 499 KB     | `/usr/share/divoom_app/divoom/21/12.bin` |

## Full list

| ID | Ext | Encryption | URL fragment |
|---:|-----|:-:|---|
| 6 | `ttf` | ✓ | `group1/M00/02/AC/rBAAM2aXZMeEC9cHAAAAABxRByQ381.ttf` |
| 8 | `ttf` |  | `group1/M00/43/CE/eEwpPWaXZNKEPBc5AAAAAIqjT6U279.ttf` |
| 10 | `ttf` |  | `group1/M00/02/AC/rBAAM2aXZNyENeJbAAAAAIqjT6U915.ttf` |
| 12 | `ttf` |  | `group1/M00/43/CE/eEwpPWaXZOWEHC9hAAAAAIqjT6U957.ttf` |
| 14 | `ttf` |  | `group1/M00/02/AC/rBAAM2aXZPOEd-DxAAAAAHUg1vI388.ttf` |
| 16 | `ttf` |  | `group1/M00/43/CF/eEwpPWaXb0SEVylVAAAAAJOW9TA500.ttf` |
| 18 | `ttf` |  | `group1/M00/02/AC/rBAAM2aXZROEVhhZAAAAADNLHeg786.ttf` |
| 20 | `ttf` |  | `group1/M00/43/CE/eEwpPWaXZR2EadRlAAAAAIvseNY679.ttf` |
| 22 | `ttf` |  | `group1/M00/02/AC/rBAAM2aXZSeEb3PjAAAAAEfgTSc212.ttf` |
| 24 | `ttf` |  | `group1/M00/43/CE/eEwpPWaXZTGEROVMAAAAANI-ums867.ttf` |
| 26 | `ttf` |  | `group1/M00/32/97/eEwpPWW3F6eEMAizAAAAAJ5qp1s077.ttf` |
| 30 | `ttf` |  | `group1/M00/43/CF/eEwpPWaXbtmEBh_cAAAAAOGpc3k071.ttf` |
| 38 | `ttf` |  | `group1/M00/03/2C/rBAAM2bO1dyEDQPLAAAAADd9UaA558.ttf` |
| 48 | `ttf` |  | `group1/M00/04/2B/rBAAM2c1nLSEFnIoAAAAAPAwhNQ411.ttf` |
| 50 | `ttf` |  | `group1/M00/04/44/rBAAM2c77kuEJP3-AAAAACWOFoc952.ttf` |
| 52 | `ttf` |  | `group1/M00/04/E9/rBAAM2dkxoyEAy2WAAAAAMlEn4Q875.ttf` |
| 54 | `ttf` |  | `group1/M00/00/55/Ci0s8Wd87ZqERDYtAAAAAFMKIZY690.ttf` |
| 56 | `ttf` |  | `group1/M00/07/CE/rBAAM2d87dOEKCxgAAAAAH-J9-c833.ttf` |
| 58 | `ttf` |  | `group1/M00/00/55/Ci0s8Wd87giEe2ddAAAAAL812hI721.ttf` |
| 60 | `ttf` |  | `group1/M00/00/55/Ci0s8Wd87jeEVkLkAAAAAC33IBg596.ttf` |
| 62 | `ttf` |  | `group1/M00/07/CE/rBAAM2d87nyENJ_QAAAAAHsFVKQ070.ttf` |
| 64 | `ttf` |  | `group1/M00/00/56/Ci0s8Wd87qOEahATAAAAAAWMZrA891.ttf` |
| 66 | `ttf` |  | `group1/M00/0A/8A/rBAAM2ewOlaEMWC6AAAAAE0yWPU646.TTF` |
| 80 | `ttf` |  | `group1/M00/03/2F/Ci0s8We2lNaEVsQlAAAAADd9UaA789.ttf` |
| 82 | `ttf` |  | `group1/M00/03/43/Ci0s8We4Q1uEdZZbAAAAAHwIIzg560.ttf` |
| 84 | `ttf` |  | `group1/M00/0A/FB/rBAAM2e5ZByENIS7AAAAAAO10O8799.ttf` |
| 88 | `otf` |  | `group1/M00/0A/FE/rBAAM2e5jwqEf-t9AAAAAItwZJQ171.otf` |
| 90 | `ttf` |  | `group1/M00/03/52/Ci0s8We5jzqEXidbAAAAAJ-V9bY889.ttf` |
| 92 | `ttf` |  | `group1/M00/0A/FE/rBAAM2e5j_OEM610AAAAANqDc7A213.ttf` |
| 94 | `ttf` |  | `group1/M00/03/72/Ci0s8We8F5OEYXciAAAAAPH2XMc766.ttf` |
| 96 | `ttf` |  | `group1/M00/03/72/Ci0s8We8GLGEWQLjAAAAABKQy1o485.ttf` |
| 98 | `ttf` | ✓ | `group1/M00/0D/BD/rBAAM2hTgt-EAWLpAAAAAJ76Zus030.ttf` |
| 100 | `otf` |  | `group1/M00/03/80/Ci0s8We9ez-EQLZmAAAAAOo2RIQ802.otf` |
| 102 | `ttf` |  | `group1/M00/0B/32/rBAAM2e9iJiEZb0oAAAAAIgz3Zs270.ttf` |
| 104 | `otf` |  | `group1/M00/03/A0/Ci0s8WfAKPWEA9GVAAAAAPDSx4U701.otf` |
| 106 | `otf` |  | `group1/M00/0B/4F/rBAAM2fAKQaEEabPAAAAALzyb3k769.otf` |
| 108 | `ttf` |  | `group1/M00/03/BD/Ci0s8WfC2QeENhTvAAAAACtqCwo088.ttf` |
| 110 | `ttf` |  | `group1/M00/0B/C1/rBAAM2fJFHOELxKmAAAAAN1tBMI628.ttf` |
| 112 | `bin` |  | `group1/M00/04/08/Ci0s8WfJVDeEGzvjAAAAAFB25GU874.bin` |
| 114 | `ttf` | ✓ | `group1/M00/04/C6/Ci0s8WfqLTSEO4MBAAAAAMAn_80336.ttf` |
| 116 | `ttf` |  | `group1/M00/0B/D2/rBAAM2fKmS6ENIgVAAAAAHpU-Ww334.ttf` |
| 118 | `ttf` |  | `group1/M00/04/16/Ci0s8WfKvoSEBvIgAAAAANB_4Pg971.ttf` |
| 120 | `ttf` |  | `group1/M00/0B/E0/rBAAM2fLnGKEPf8YAAAAAEkd9WA036.ttf` |
| 122 | `otf` |  | `group1/M00/0C/33/rBAAM2fSn4WEZG8fAAAAAHEnRvI639.otf` |
| 124 | `ttf` |  | `group1/M00/04/6D/Ci0s8WfSpdKEZxHiAAAAAGaqq1w185.ttf` |
| 126 | `ttf` |  | `group1/M00/04/6E/Ci0s8WfSukKERa4mAAAAADIuevU965.ttf` |
| 128 | `ttf` | ✓ | `group1/M00/04/C7/Ci0s8WfqWXuEJe7YAAAAADxCP98373.ttf` |
| 132 | `ttf` | ✓ | `group1/M00/04/C7/Ci0s8WfqXG2EB2_3AAAAAACZDis138.ttf` |
| 138 | `ttf` | ✓ | `group1/M00/04/CC/Ci0s8WfrhdWEQrgmAAAAANfGFgU417.ttf` |
| 140 | `ttf` | ✓ | `group1/M00/04/CC/Ci0s8WfrlsyEak1wAAAAAJ-V9bY797.ttf` |
| 142 | `ttf` | ✓ | `group1/M00/04/CC/Ci0s8WfrmRGEQOXuAAAAANqDc7A903.ttf` |
| 144 | `ttf` | ✓ | `group1/M00/0C/92/rBAAM2frrwOEcu_JAAAAACcpfhA265.ttf` |
| 146 | `ttf` | ✓ | `group1/M00/04/D4/Ci0s8WfuPLOETRBwAAAAAPsi6S8383.ttf` |
| 148 | `ttf` | ✓ | `group1/M00/04/D4/Ci0s8WfuPOKEA_5QAAAAAPZdSxU672.ttf` |
| 152 | `ttf` | ✓ | `group1/M00/04/D5/Ci0s8WfuUi-ETkicAAAAAPnk7D8063.ttf` |
| 154 | `ttf` | ✓ | `group1/M00/0C/9A/rBAAM2fuUkqEFVOkAAAAAAnJfKI431.ttf` |
| 156 | `ttf` | ✓ | `group1/M00/04/F6/Ci0s8Wf4zSuETmr8AAAAAB7bOmc203.ttf` |
| 158 | `ttf` | ✓ | `group1/M00/05/3D/Ci0s8WgN5myEKcZgAAAAAJ_dEU0012.ttf` |
| 160 | `ttf` | ✓ | `group1/M00/0D/01/rBAAM2gN8xSESmQVAAAAAGTt4ro023.ttf` |
| 162 | `ttf` | ✓ | `group1/M00/05/41/Ci0s8WgPO1SEWW1VAAAAAIqU8ik974.ttf` |
| 164 | `ttf` | ✓ | `group1/M00/0D/05/rBAAM2gPN_6EHYTGAAAAAIkLBeQ476.ttf` |
| 166 | `ttf` | ✓ | `group1/M00/05/42/Ci0s8WgPSaCETgOGAAAAANqDc7A438.ttf` |
| 168 | `ttf` | ✓ | `group1/M00/0D/06/rBAAM2gPTBeEVHVnAAAAAIpnrUE444.ttf` |
| 170 | `ttf` | ✓ | `group1/M00/0D/42/rBAAM2gkDymEB-0XAAAAAJ2ElaE298.ttf` |
| 172 | `ttf` | ✓ | `group1/M00/0D/43/rBAAM2gkTmyEFcxUAAAAAKobP0c284.ttf` |
| 174 | `ttf` | ✓ | `group1/M00/0D/71/rBAAM2g0Lt-EGrN2AAAAADEJzS0227.ttf` |
| 176 | `ttf` | ✓ | `group1/M00/05/CA/Ci0s8WhAB7iEG596AAAAAHjN_-0703.ttf` |
| 178 | `ttf` | ✓ | `group1/M00/0D/9D/rBAAM2hGtSmEQ_5IAAAAAOQDqKs866.ttf` |
| 180 | `ttf` | ✓ | `group1/M00/0D/A7/rBAAM2hKi1CEUqQIAAAAAEaHoso495.ttf` |
| 182 | `ttf` | ✓ | `group1/M00/05/F3/Ci0s8WhP95yEcbaNAAAAAB60hXA197.ttf` |
| 184 | `ttf` | ✓ | `group1/M00/05/F9/Ci0s8WhSeBGEAx8xAAAAACPWyz8060.ttf` |
| 186 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhTbsCEcTKrAAAAALGEm84255.ttf` |
| 188 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhTfUCEXnjqAAAAAN_iCVY705.ttf` |
| 190 | `ttf` | ✓ | `group1/M00/0D/BD/rBAAM2hTfw-EQiK9AAAAAEaHoso286.ttf` |
| 192 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhTgJ-EaPsmAAAAABHtHIg205.ttf` |
| 194 | `ttf` | ✓ | `group1/M00/0D/BD/rBAAM2hTgOGEBaqzAAAAAAtVg_c909.ttf` |
| 196 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhTgP6EVH7LAAAAAOAFlks883.ttf` |
| 198 | `ttf` | ✓ | `group1/M00/0D/BD/rBAAM2hTgTSEPJAKAAAAAC229Mo124.ttf` |
| 200 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhTgWSEOvJkAAAAAB4smZs509.ttf` |
| 202 | `ttf` | ✓ | `group1/M00/0D/BD/rBAAM2hTgYCEaKCOAAAAANffYt8456.TTF` |
| 204 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhTgaiEZ9pqAAAAAPWSdds209.ttf` |
| 206 | `ttf` | ✓ | `group1/M00/0D/BD/rBAAM2hTggWETMdDAAAAAC_p3cU023.ttf` |
| 208 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhTghmESeHkAAAAABBi7ms638.ttf` |
| 210 | `ttf` | ✓ | `group1/M00/0D/BD/rBAAM2hTgiyEZX3fAAAAAGziRmU918.ttf` |
| 212 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhTglGEW1qvAAAAADHMEwM112.ttf` |
| 214 | `ttf` | ✓ | `group1/M00/0D/BD/rBAAM2hTgnKEG4NJAAAAAMf3-ck025.TTF` |
| 216 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhTgouELGGsAAAAAMO2_RM540.ttf` |
| 218 | `ttf` | ✓ | `group1/M00/0D/BD/rBAAM2hTgp6EGnu0AAAAANgN2SM990.ttf` |
| 220 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhTgriEVC2AAAAAAO_zbZE906.ttf` |
| 222 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhTgvGEGI4wAAAAAG0P-Xc557.ttf` |
| 224 | `ttf` | ✓ | `group1/M00/0D/BD/rBAAM2hTg2aEA_LGAAAAAF_Mrk8917.TTF` |
| 226 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhTg4GEaUsnAAAAAHL5tKc972.TTF` |
| 228 | `ttf` | ✓ | `group1/M00/0D/BD/rBAAM2hTg6aEX2kfAAAAANWKTaI941.ttf` |
| 230 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhTg7yEOEaJAAAAABn2OD8935.ttf` |
| 232 | `ttf` | ✓ | `group1/M00/0D/BD/rBAAM2hTg9mERTYOAAAAAKd80zI226.ttf` |
| 234 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhThBCEZrxnAAAAAKVJBZM878.ttf` |
| 236 | `ttf` | ✓ | `group1/M00/0D/BD/rBAAM2hThF2EBftdAAAAABP7SdM711.ttf` |
| 238 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhThKqEPaMUAAAAABi-R7E844.ttf` |
| 240 | `ttf` | ✓ | `group1/M00/0D/BD/rBAAM2hThRGEPGQaAAAAAK9DlpQ014.ttf` |
| 242 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhThVuEX0p7AAAAAAFyYro640.ttf` |
| 246 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhThg6EGYwHAAAAADDniXk751.ttf` |
| 248 | `ttf` | ✓ | `group1/M00/0D/BD/rBAAM2hThiqEIYETAAAAAANNqu4562.ttf` |
| 250 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhThkGETeVIAAAAANs_um0786.ttf` |
| 252 | `ttf` | ✓ | `group1/M00/0D/BD/rBAAM2hThl-EagNSAAAAAC3xP4c793.ttf` |
| 254 | `ttf` | ✓ | `group1/M00/05/FB/Ci0s8WhThraEa5CfAAAAAK3n2tY458.ttf` |
| 256 | `bin` | ✓ | `group1/M00/0D/DE/rBAAM2hfn66EKnvKAAAAADlQi6o019.bin` |
| 258 | `bin` | ✓ | `group1/M00/06/22/Ci0s8WhiM7eEM0ELAAAAANhuTqg292.bin` |
| 260 | `ttf` | ✓ | `group1/M00/06/25/Ci0s8WhjqzqEN2aUAAAAAOHUbQY816.ttf` |
| 262 | `bin` | ✓ | `group1/M00/06/25/Ci0s8WhjpfaES3HwAAAAAMKI4l8703.bin` |
| 264 | `ttf` | ✓ | `group1/M00/0D/FD/rBAAM2hs3PaEZCHhAAAAAPGC5uM680.ttf` |
| 266 | `ttf` | ✓ | `group1/M00/0D/FD/rBAAM2hs8p-EAkq0AAAAAGgmvRE253.ttf` |
| 268 | `bin` | ✓ | `group1/M00/0E/07/rBAAM2hwrg2EEjP4AAAAAN9aD1g788.bin` |
| 270 | `bin` | ✓ | `group1/M00/06/46/Ci0s8WhwrSmEcuWEAAAAAMAv_CA803.bin` |
| 272 | `ttf` | ✓ | `group1/M00/0E/04/rBAAM2hva1uEZfzoAAAAAIAxizs788.TTF` |
| 274 | `ttf` | ✓ | `group1/M00/0E/04/rBAAM2hva52Eec24AAAAAIgz3Zs950.ttf` |
| 276 | `ttf` | ✓ | `group1/M00/0E/04/rBAAM2hva-2ERNT4AAAAAPA8VBI553.ttf` |
| 278 | `bin` | ✓ | `group1/M00/0E/08/rBAAM2hwxxSECK9qAAAAAPI4NL0732.bin` |
| 280 | `bin` | ✓ | `group1/M00/0E/0B/rBAAM2hyL6WEMWzuAAAAAPejOmQ746.bin` |
| 282 | `bin` | ✓ | `group1/M00/0E/0B/rBAAM2hyEHuEUZLUAAAAANj3QqY144.bin` |
| 284 | `ttf` | ✓ | `group1/M00/0E/13/rBAAM2h0pnmEYpHTAAAAAN1bgI8619.ttf` |
| 286 | `ttf` | ✓ | `group1/M00/0E/13/rBAAM2h0yNuEeUumAAAAAFkjET8733.ttf` |
| 288 | `ttf` | ✓ | `group1/M00/0E/17/rBAAM2h197OEXGK-AAAAAJ22KE8820.ttf` |
| 290 | `bin` | ✓ | `group1/M00/06/55/Ci0s8Wh2EpiEY-2RAAAAAEvFCgc850.bin` |
| 292 | `bin` | ✓ | `group1/M00/0E/17/rBAAM2h2E9-ERzCsAAAAAKvgCbA679.bin` |
| 294 | `bin` | ✓ | `group1/M00/0E/1B/rBAAM2h3bQqEaDxcAAAAAM7QAYY256.bin` |
| 296 | `bin` | ✓ | `group1/M00/0E/1E/rBAAM2h4x3WEGQn-AAAAAJonPUM619.bin` |
| 298 | `ttf` | ✓ | `group1/M00/06/5D/Ci0s8Wh40AGEfsppAAAAAIfQMGI926.ttf` |
| 300 | `ttf` | ✓ | `group1/M00/06/60/Ci0s8Wh57lWEWaIqAAAAAJbZHQ8833.ttf` |
| 302 | `bin` | ✓ | `group1/M00/06/60/Ci0s8Wh5-1iEE7-aAAAAAF4P_ew517.bin` |
| 304 | `bin` | ✓ | `group1/M00/06/60/Ci0s8Wh5_7CEPYArAAAAAJpRhYw876.bin` |
| 306 | `bin` | ✓ | `group1/M00/0E/2C/rBAAM2h96m6EPkHKAAAAAHbuLg0491.bin` |
| 308 | `bin` | ✓ | `group1/M00/0E/30/rBAAM2h_OLSEanI2AAAAAI5IrBU384.bin` |
| 310 | `bin` | ✓ | `group1/M00/06/6F/Ci0s8Wh_OjmEKZMQAAAAAGsKkPo537.bin` |
| 312 | `ttf` | ✓ | `group1/M00/06/6F/Ci0s8Wh_OsyEUnLKAAAAAPxswkI443.ttf` |
| 314 | `bin` | ✓ | `group1/M00/06/8D/Ci0s8WiHMX6EATWMAAAAAHdzIdI216.bin` |
| 316 | `bin` | ✓ | `group1/M00/06/A1/Ci0s8WiJi0OEVKPOAAAAAIoBJWo998.bin` |
| 318 | `ttf` | ✓ | `group1/M00/0E/B0/rBAAM2iR1OGEeUQrAAAAALosBjY346.ttf` |
| 320 | `ttf` | ✓ | `group1/M00/07/57/Ci0s8Wierm2EQfVYAAAAAG5p1oY640.ttf` |
| 324 | `bin` | ✓ | `group1/M00/07/A2/Ci0s8WipXvOEACs3AAAAADqDpD8869.bin` |
| 326 | `bin` | ✓ | `group1/M00/0F/76/rBAAM2ir0d6EGUUAAAAAANCg-D4863.bin` |
| 328 | `ttf` | ✓ | `group1/M00/0F/95/rBAAM2ivwL2EIEK9AAAAAL-WmQ8835.ttf` |
| 330 | `bin` | ✓ | `group1/M00/0F/FD/rBAAM2i_4JiEOjwKAAAAAKQLfQA569.bin` |
| 332 | `bin` | ✓ | `group1/M00/08/73/Ci0s8WjJCduEJrxwAAAAAIoVP1c128.bin` |
| 334 | `bin` | ✓ | `group1/M00/08/74/Ci0s8WjJHrSEfBTnAAAAAEG0Mbs117.bin` |
| 336 | `ttf` | ✓ | `group1/M00/10/33/rBAAM2jJKDiEfHOFAAAAANAmVu8952.ttf` |
| 338 | `ttf` | ✓ | `group1/M00/09/70/Ci0s8Wj3LOWENkT3AAAAAFtdGo8249.ttf` |
| 340 | `bin` | ✓ | `group1/M00/11/38/rBAAM2j4dFOEXZybAAAAABRROIs121.bin` |
| 342 | `bin` | ✓ | `group1/M00/11/38/rBAAM2j4dvCEI23LAAAAAMumuxE963.bin` |
| 344 | `bin` | ✓ | `group1/M00/09/78/Ci0s8Wj4gWmEFEI3AAAAANXitnU228.bin` |
| 346 | `bin` | ✓ | `group1/M00/0A/6D/Ci0s8WkcHCWEN2n3AAAAABoQfUg732.bin` |
| 348 | `bin` | ✓ | `group1/M00/12/AF/rBAAM2kusK2EDYoEAAAAAACgtvI500.bin` |
| 350 | `bin` | ✓ | `group1/M00/0A/D5/Ci0s8WkqmwKEba7FAAAAAPA3dWY226.bin` |
| 352 | `bin` | ✓ | `group1/M00/0A/F0/Ci0s8WkuQMGEEANJAAAAABrtAPo828.bin` |
| 354 | `bin` | ✓ | `group1/M00/0C/0E/Ci0s8WlN5EiEBHh0AAAAAMUBq2s999.bin` |
| 356 | `bin` | ✓ | `group1/M00/14/22/rBAAM2lUkBWEU0TSAAAAAPhLyP8970.bin` |
| 358 | `ttf` | ✓ | `group1/M00/0C/6A/Ci0s8WlUzsSEP-wEAAAAAMpPjFQ480.ttf` |
| 360 | `ttf` | ✓ | `group1/M00/14/C7/rBAAM2llrpaEGYMsAAAAAIa2ehc251.ttf` |
| 362 | `ttf` | ✓ | `group1/M00/18/65/rBAAM2oFjCqEXdlNAAAAALUsmkM928.ttf` |
| 364 | `ttf` | ✓ | `group1/M00/10/B1/Ci0s8WoFjEqEK7RZAAAAALTiAU0596.ttf` |

To download a font for inspection, prepend `https://f.divoom-gz.com/` to the URL fragment.