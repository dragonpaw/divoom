# Add dev targets to the Makefile

Today's `Makefile` is deploy-only: `build`, `login`, `push`,
`deploy`, `stacks`. Nothing for the day-to-day Go workflow. Means
every new contributor (or future-you) reinvents the commands.

## Targets to add

```make
.PHONY: test vet lint fmt run probe render-out

test:
	go test ./...

vet:
	go vet ./...

# Optional — only if golangci-lint is on PATH.
lint:
	@command -v golangci-lint >/dev/null && golangci-lint run \
	    || echo "golangci-lint not installed; skipping"

fmt:
	gofmt -w .

# Run the daemon locally against the configured frame.
run:
	go run ./cmd/divoom serve

probe:
	go run ./cmd/divoom probe

# Render every scene background JPG to ./dist/scenes/ for inspection.
render-out:
	go run ./cmd/divoom render
```

## Why

Documents the canonical commands inline with the rest of the build
plumbing. `make test` is muscle-memory across most languages; not
having it is a small papercut every time.
