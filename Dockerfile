FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/divoom ./cmd/divoom

# Alpine for ca-certificates (HTTPS to Open-Meteo, NASA, GitHub, Reddit, et al)
# and tzdata (TZ=America/Los_Angeles in compose resolves to a real zoneinfo).
# No android-tools: this image runs `serve` only, which talks to the frame
# over LAN via divoom_api on port 9000. Background JPG pushing requires USB
# adb and is done from the dev box via `divoom push`, NOT in this image.
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/divoom /usr/local/bin/divoom

ENTRYPOINT ["/usr/local/bin/divoom"]
CMD ["serve"]
