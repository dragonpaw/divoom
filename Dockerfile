FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/divoom ./cmd/divoom

# Alpine (not distroless/static) because the daemon shells out to `adb` to
# push background JPGs into /userdata/ on the frame (see internal/adb/adb.go).
# ca-certificates covers HTTPS to Open-Meteo, NASA, GitHub, Reddit, et al.
# tzdata so TZ=America/Los_Angeles in compose resolves to a real zoneinfo.
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata android-tools
COPY --from=build /out/divoom /usr/local/bin/divoom

# Default to the frame we discovered during dev. Override with `-e` or in
# docker-compose.yml if you ever swap devices or have more than one on the LAN.
ENV DIVOOM_FRAME_MAC=4c37deff59df

ENTRYPOINT ["/usr/local/bin/divoom"]
CMD ["help"]
