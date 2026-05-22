# Fix Dockerfile module-cache layer

```dockerfile
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download      # ← downloads
COPY . .                 # ← invalidates the previous layer on every code change
RUN CGO_ENABLED=0 go build ...
```

The `COPY go.mod ./` + `go mod download` step *is* trying to be
cache-friendly — but it's missing `go.sum`, so `go mod download`
doesn't actually pre-fetch everything; it then re-downloads when
the real build runs after `COPY . .`. Net effect: deps are
downloaded on every `make build` even though nothing changed in
go.mod.

## Fix

Two-line change:

```dockerfile
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build ...
```

Now `go mod download` actually pre-fetches the locked set, the
download layer caches on every build where go.mod/go.sum didn't
change, and the actual `go build` step finds everything ready.

## Verify

Time two consecutive `make build`s with no code change. The
second should skip the download step entirely (cached layer) and
finish in <10s. Before the fix, both runs download deps.

## Why

Smallest possible change, biggest dev-loop win for any
container-rebuild workflow. Costs nothing.
