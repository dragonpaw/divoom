# 001 - No vendor directory; single module

**Status**: Accepted
**Date**: 2026-05-22

## Context

One binary (the daemon), roughly ten direct dependencies, no air-gap
requirement, and GOPROXY is reachable from both developer machines and
CI. There is no second module or second binary on the horizon.

## Decision

Stay single-module. Do not add a `vendor/` directory. Do not adopt
`go.work`.

## Consequences

- Faster repo clones and smaller dep-bump diffs to review.
- Builds depend on GOPROXY being reachable at build/CI time.
- CI's `go mod download` step in `.github/workflows/ci.yml` handles
  fetching dependencies.

## Revisit when

- A second binary lands in this repo.
- An air-gapped deployment requirement appears.
- A build-reproducibility audit demands vendored, in-tree dependencies.
