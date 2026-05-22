# Add GitHub Actions CI

No `.github/workflows/` exists. Every commit on main is trust-the-
local-build. The moment a `go vet` regression or test failure lands
unnoticed, it'll bite.

## Minimum viable

`.github/workflows/ci.yml`:

```yaml
name: ci
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - run: go vet ./...
      - run: go test ./...
      - run: go build ./...
```

That's it. ~30s per push. Catches the regressions you'd otherwise
discover on your dev box after a context switch.

## Extras (later, if useful)

- `golangci-lint run` — wires in staticcheck, errcheck, ineffassign
  for free. One more workflow step, one config file.
- `make deploy` smoke from CI — probably overkill since it needs
  Portainer creds; skip.
- Container build + push to GHCR on tag — automate what `make
  build push` does manually today.

## Verify

Push a deliberately broken commit (unused import, failing test) to
a branch; CI should fail and the PR should show the red X. Then
revert.
