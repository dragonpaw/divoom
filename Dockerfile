FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/divoom ./cmd/divoom

# Distroless static: ca-certificates + tzdata + /etc/passwd baked in, no
# libc, no shell, no package manager. The Go binary is statically linked
# (CGO_ENABLED=0) so it brings its own runtime. This image runs `serve`
# only — talks to the frame over LAN via divoom_api on port 9000. Background
# JPG / font pushing requires USB adb and runs from the dev box via
# `divoom push`, NOT in this image.
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/divoom /usr/local/bin/divoom

ENTRYPOINT ["/usr/local/bin/divoom"]
CMD ["serve"]
