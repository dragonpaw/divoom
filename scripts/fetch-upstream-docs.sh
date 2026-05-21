#!/usr/bin/env bash
# Refresh frozen snapshots of Divoom's upstream Times Frame docs and the font
# catalog. Run when you suspect upstream has changed (or before referencing
# something in code that depends on the docs being current).
set -euo pipefail

cd "$(dirname "$0")/.."
mkdir -p docs/upstream

# Pages that make up the TimeFrame catalog (cat_id=52 of item_id=5).
PAGES=(358 359 360 361 362 363 367 368 369 370 371 372 373 374 375 377 379)

for pid in "${PAGES[@]}"; do
  echo "  fetch p$pid"
  curl -sS "https://docin.divoom-gz.com/server/index.php?s=/api/page/info&page_id=$pid" \
    > "docs/upstream/p$pid.json"
done

echo "  fetch font list"
curl -sS -X POST 'https://appin.divoom-gz.com/Device/GetTimeDialFontV2' \
  -H 'Content-Type: application/json' --data-raw '{}' \
  > docs/fonts.json

echo "  fetch catalog tree (for new pages we don't know about yet)"
curl -sS 'https://docin.divoom-gz.com/server/index.php?s=/api/item/info&item_id=5' \
  > docs/upstream/_catalog.json

echo "done — review the JSON diffs and update docs/api.md / docs/fonts.md as needed"
