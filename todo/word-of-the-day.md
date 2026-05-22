# Word of the Day scene

Item 8 from the 1-11 brainstorm. Surface a daily English vocabulary
entry — headword + part of speech + definition — using the existing
`DictionaryScene` layout the Jargon and Devil's scenes already use.

## Sources

- **Wordnik API** — `https://api.wordnik.com/v4/words.json/wordOfTheDay`
  Requires a free API key (free tier 100 req/hr). Easy to use; returns
  `{word, definitions, examples, note}`. Cleanest single-call shape.
- **Merriam-Webster** — has a WOTD RSS / OEM but their JSON API is
  paywalled.
- **Free Dictionary API** (`https://api.dictionaryapi.dev/`) — no key,
  but no curated WOTD list; would need to seed with a curated wordlist
  and pick deterministically per day.

Recommend Wordnik — pattern matches `NASA_API_KEY` env var fallback we
already do for APOD. Skip if env var not set; fall back to a small
hardcoded wordlist as a last resort.

## Scene shape

Re-use `DictionaryScene(DictionarySceneOpts{…})` exactly — same shape
as Jargon and Devil's. Pick a headword colour distinct from those
(jargon = cYellow, devil = cRed) — try `cPurple` or `cAqua`. Optional
"Word of the Day" tagline at the bottom in matching colour.

Widget emits `"Word of the Day|<entry>|<author or empty>"` where the
entry is the `WORD pos. definition` shape the DictionaryScene parser
expects. The dictionary entry parser regex (`dictionaryEntryRE`)
should handle Wordnik's format with minimal massaging — confirm
during implementation.

## Element count

7 (3 top + 4 body, same as the other DictionaryScene-built scenes).
Pick will block consecutive transitions with same-count scenes,
which is fine.
