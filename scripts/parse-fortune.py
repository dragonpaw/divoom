#!/usr/bin/env python3
"""Regenerate internal/widget/quotes/fortune.go from the fortune-mod corpus.

Usage:
    python3 scripts/parse-fortune.py > internal/widget/quotes/fortune.go

Pulls the BSD-style cookie files from shlomif/fortune-mod (which preserves
the historical fortune(1) database, widely redistributable), splits them on
the `%` record separator, filters for short single-paragraph entries that
fit the device's narrow body column, drops obvious vulgar or political
material that doesn't suit the ambient dashboard, and emits a Go
string-slice literal. Output is capped so the binary doesn't bloat.

Entries that carry a trailing `-- Attribution` line get rewritten into the
existing " — Author" attribution convention used by the other quote
sources.
"""

from __future__ import annotations

import re
import sys
import urllib.request


SOURCES = [
    # The classic cookie file — short fortune-teller-style one-liners.
    "https://raw.githubusercontent.com/shlomif/fortune-mod/master/fortune-mod/datfiles/fortunes",
    # Wisdom-bucket quotes; longer and more aphoristic, attributed.
    "https://raw.githubusercontent.com/shlomif/fortune-mod/master/fortune-mod/datfiles/wisdom",
    # Computer-flavoured aphorisms that fit the dashboard's technical tone.
    "https://raw.githubusercontent.com/shlomif/fortune-mod/master/fortune-mod/datfiles/computers",
]

MAX_LEN = 200
MIN_LEN = 25
MAX_ENTRIES = 500

# Words that signal content we don't want on an ambient wall display:
# crude language, political slurs, slurs, sexual references. Conservative
# allow-by-omission filter — false positives are fine (we have 2000+
# candidates and only need 500), false negatives are the failure mode we
# care about.
BANNED = {
    "fuck", "fucking", "fucked", "shit", "bitch", "bastard", "damn",
    "hell", "ass", "asshole", "dick", "cock", "pussy", "cunt", "tits",
    "sex", "sexy", "sexual", "rape", "nigger", "nigga", "fag", "faggot",
    "queer", "jew", "kike", "spic", "chink", "gook", "nazi", "hitler",
    "trump", "biden", "obama", "clinton", "bush", "reagan", "republican",
    "democrat", "liberal", "conservative", "abortion", "gun", "kill",
    "murder", "suicide", "rapist", "porn", "naked", "horny",
}


def fetch(url: str) -> str:
    with urllib.request.urlopen(url, timeout=30) as f:
        return f.read().decode("utf-8", errors="replace")


def split_records(text: str) -> list[str]:
    """fortune cookies are separated by lines containing only `%`."""
    out: list[str] = []
    buf: list[str] = []
    for line in text.splitlines():
        if line.strip() == "%":
            if buf:
                out.append("\n".join(buf).strip())
                buf = []
        else:
            buf.append(line)
    if buf:
        out.append("\n".join(buf).strip())
    return out


ATTR_RE = re.compile(r"^\s*--\s*(.+?)\s*$")


def normalise(entry: str) -> str | None:
    """Collapse whitespace, lift trailing `-- Author` into ` — Author`."""
    lines = [l.rstrip() for l in entry.splitlines() if l.strip()]
    if not lines:
        return None
    author = ""
    # Trailing `-- Author` lines (possibly more than one) get pulled off.
    while lines and (m := ATTR_RE.match(lines[-1])):
        if not author:
            author = m.group(1)
        lines.pop()
    body = " ".join(lines).strip()
    body = re.sub(r"\s+", " ", body)
    if not body:
        return None
    if author:
        return f"{body} — {author}"
    return body


def acceptable(entry: str) -> bool:
    if not (MIN_LEN <= len(entry) <= MAX_LEN):
        return False
    lower = entry.lower()
    # Word-boundary match so "class" doesn't trip on "ass", etc.
    for w in BANNED:
        if re.search(rf"\b{re.escape(w)}\b", lower):
            return False
    # Drop entries that are just ASCII art or shell snippets.
    if entry.count(" ") < 3:
        return False
    return True


def main() -> int:
    seen: set[str] = set()
    entries: list[str] = []
    for url in SOURCES:
        try:
            raw = fetch(url)
        except Exception as exc:
            print(f"warning: {url}: {exc}", file=sys.stderr)
            continue
        for rec in split_records(raw):
            norm = normalise(rec)
            if norm is None:
                continue
            if not acceptable(norm):
                continue
            if norm in seen:
                continue
            seen.add(norm)
            entries.append(norm)
    if len(entries) > MAX_ENTRIES:
        entries = entries[:MAX_ENTRIES]

    out = sys.stdout
    print("// fortune(1) cookies — sourced from shlomif/fortune-mod's", file=out)
    print("// public-domain corpus and parsed with scripts/parse-fortune.py.", file=out)
    print("// Regenerate when refreshing the pool.", file=out)
    print("package quotes", file=out)
    print(file=out)
    print(f"// NewFortune returns a Source drawn from the BSD fortune cookie", file=out)
    print(f"// database (~{len(entries)} entries). Mixed attribution: many", file=out)
    print(f"// cookies are anonymous, those with a trailing `-- Author` line", file=out)
    print(f"// carry their attribution inline using the existing ` — Author`", file=out)
    print(f"// convention.", file=out)
    print(f"func NewFortune() *Source {{ return newSource(\"fortune\", fortuneCookies) }}", file=out)
    print(file=out)
    print("var fortuneCookies = []string{", file=out)
    for e in entries:
        s = e.replace("\\", "\\\\").replace("\"", "\\\"")
        print(f"\t\"{s}\",", file=out)
    print("}", file=out)
    print(f"// {len(entries)} entries", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
