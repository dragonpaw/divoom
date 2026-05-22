# Expand README

Today's README is essentially just `Usage` — subcommands and env
vars. Anyone landing on the repo doesn't know:

- What the project IS (a custom dashboard for a Divoom Times Frame)
- What it looks like (no screenshot)
- Why it exists (you got frustrated with the official app)
- How the architecture works (NAS-side serve + dev-box push split)
- What the scene rotation actually shows

## What to add

- **Top-of-readme paragraph**: one or two sentences on what this
  is. "A custom wall-clock dashboard for the Divoom Times Frame
  (800×1280 portrait), built because the stock app is restrictive
  and the device is an Allwinner TinaLinux box that accepts adb
  pushes and a JSON HTTP API."
- **Screenshot or two**. Render output from a couple of scenes
  side-by-side. `divoom render` produces `dist/scenes/*.jpg`; pick
  three representative ones and link them in a `docs/screenshots/`
  directory. Static images, ~50 KB each.
- **Architecture section** with a small diagram:
  ```
  dev-box  ─[adb push: bgs, fonts]──> Times Frame ─[HTTP poll 9000]─> NAS divoom-dashboard
                                                                        │
                                                                        └─ widgets fetch
                                                                           external APIs
  ```
- **Engineering philosophy callout** — link to CLAUDE.md.
- **Pointers to docs/api.md and docs/deploy.md.**

## Why

If the repo ever goes public, the README is the only thing most
visitors will read. Right now it's adequate for "I already know
what this is and just need the command list" but not for
"interesting, what is this." 30 minutes of writing for the
discoverability win.
