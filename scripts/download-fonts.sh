#!/usr/bin/env bash
# Fetch the three custom TTFs that `divoom push` installs on the Times
# Frame. Run once on the USB-attached host before the first `divoom push`.
# Files land in ./fonts/, which is gitignored — the TTFs are big (Iosevka
# is ~10 MB) and don't belong in the repo.
#
# Canonical sources per docs/api.md "Custom font workflow":
#   - Iosevka-Regular.ttf       — Iosevka 34.5.0 GitHub release
#   - RobotoCondensed-Regular.ttf — googlefonts/roboto hinted sources
#   - RobotoCondensed-Light.ttf   — googlefonts/roboto hinted sources
set -euo pipefail

cd "$(dirname "$0")/.."
mkdir -p fonts

fetch() {
  local url="$1" dst="$2"
  if [ -s "$dst" ]; then
    echo "  have $dst"
    return
  fi
  echo "  fetch $dst"
  curl -fsSL --retry 3 -o "$dst" "$url"
}

# Iosevka ships its TTF inside a zip per release.
IOSEVKA_VERSION="34.5.0"
IOSEVKA_ZIP_URL="https://github.com/be5invis/Iosevka/releases/download/v${IOSEVKA_VERSION}/PkgTTF-Iosevka-${IOSEVKA_VERSION}.zip"
if [ ! -s fonts/Iosevka-Regular.ttf ]; then
  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT
  echo "  fetch Iosevka ${IOSEVKA_VERSION} zip"
  curl -fsSL --retry 3 -o "$tmp/iosevka.zip" "$IOSEVKA_ZIP_URL"
  unzip -q -o -d "$tmp" "$tmp/iosevka.zip" "Iosevka-Regular.ttf"
  mv "$tmp/Iosevka-Regular.ttf" fonts/Iosevka-Regular.ttf
  rm -rf "$tmp"
  trap - EXIT
  echo "  wrote fonts/Iosevka-Regular.ttf"
else
  echo "  have fonts/Iosevka-Regular.ttf"
fi

fetch \
  "https://github.com/googlefonts/roboto/raw/main/src/hinted/RobotoCondensed-Regular.ttf" \
  fonts/RobotoCondensed-Regular.ttf

fetch \
  "https://github.com/googlefonts/roboto/raw/main/src/hinted/RobotoCondensed-Light.ttf" \
  fonts/RobotoCondensed-Light.ttf

echo "done — run \`go run ./cmd/divoom push\` to install on the frame"
