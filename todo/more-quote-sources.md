# Additional quote-source suggestions

Original brief asked: "suggest other sources we could include". Parking
them here so we don't lose the list. Each would slot in as another
`*quotes.Source` with weight 1 alongside the existing five.

Candidates ordered by how well they fit the dashboard's tone (dry,
self-aware, technical-adjacent):

- **Stoics** — Marcus Aurelius (*Meditations*), Seneca, Epictetus.
  Public domain. Strong overlap with the "perspective" mood.
- **Discworld** — Pratchett one-liners. Strict fair use; ~50-100
  well-known lines, attribution per book/character.
- **Hitchhiker's Guide to the Galaxy** — Adams. Same caveat as Pratchett.
- **Knuth / Dijkstra / Hoare** — programming aphorisms. Some are
  already in the Jargon File but a focused "computing forefathers"
  source would be denser.
- **Calvin & Hobbes** — Watterson, fair-use small set.
- **Mark Twain** — public domain, lots of quotable lines.
- **Oscar Wilde** — public domain, world-class wit.
- **Office Space / The Office / Parks and Rec** — workplace humor;
  small fair-use sets.
- **fortune(1) cookies** — the BSD fortune database is widely
  redistributable and full of one-liners; could be the easiest bulk
  source. <https://www.shlomifish.org/open-source/projects/fortune-mod/>

Implementation pattern matches existing sources:
- Hand-curated → just a `[]string` in `internal/widget/quotes/<name>.go`
- Bulk → fetch script in `scripts/` + generated Go file (see
  `parse-devil.py` for the pattern)

Per-quote attribution uses the existing `" — Author"` suffix
convention; entries without attribution leave the author block blank.
