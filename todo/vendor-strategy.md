# Decide on a vendor / go.work strategy

Currently one module, no `vendor/`, no `go.work`. Fine for one
binary today. Worth a deliberate decision before either of these
shows up:

- A second binary (e.g. an asset-renderer separate from the
  daemon, or a one-off migration tool).
- Reproducible builds in CI without trusting GOPROXY at build
  time.
- Air-gapped deploys.

## Options

- **Stay single-module, no vendor.** Default. Lowest friction.
  Re-decide if a real second binary appears.
- **Add `vendor/`.** `go mod vendor` then commit. Adds ~10 MB of
  deps to the repo but guarantees `go build` works offline and
  pinned to whatever was vendored. CI doesn't need to hit
  proxy.golang.org. Tradeoff: every dep bump is a much bigger
  diff to review.
- **Adopt `go.work`** if a second module ever lands. Don't bother
  pre-emptively — go.work only earns its keep with 2+ modules.

## Recommendation

Stay single-module + no vendor until evidence shows otherwise.
This todo exists so future-you remembers to make the decision
deliberately instead of by inertia when the moment comes.
