FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/divoom ./cmd/divoom

# Debian slim base so we get adb (android-tools-adb) alongside the static
# Go binary. `serve` talks to the frame over LAN via divoom_api on port
# 9000; `push` uploads backgrounds and fonts over USB-adb to /userdata/ —
# the latter only works when the host running this container has the
# frame on a USB port and the container is started with /dev/bus/usb
# bind-mounted and --privileged (see docs/scene-rules.md).
FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends android-tools-adb ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/divoom /usr/local/bin/divoom
COPY scripts/entrypoint.sh /entrypoint.sh
# Fonts copied into /opt/divoom/fonts; WORKDIR set to /opt/divoom so
# LoadFont's "fonts/<name>" relative path resolves correctly at runtime
# (see internal/render/text.go).
COPY fonts/ /opt/divoom/fonts/
# device-files/ holds the local copy of font_list.cfg that `divoom push`
# pushes to /divoom-config/system/font_list.cfg on the frame so it
# learns about our custom TTFs (see cmd/divoom/fonts.go).
COPY device-files/ /opt/divoom/device-files/
WORKDIR /opt/divoom

# Entrypoint runs `divoom push` (best-effort) before exec'ing the
# requested subcommand, so a fresh deploy / restart refreshes the frame's
# baked backgrounds + fonts whenever the container comes up.
ENTRYPOINT ["/entrypoint.sh"]
CMD ["serve"]
