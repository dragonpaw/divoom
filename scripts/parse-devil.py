#!/usr/bin/env python3
"""Regenerate internal/widget/quotes/devil.go from Project Gutenberg #972.

Usage:
    python3 scripts/parse-devil.py > internal/widget/quotes/devil.go

Pipes the UTF-8 text of *The Devil's Dictionary* through a heading-detector
that recognises lines like ``WORD, n.`` / ``WORD, adj.`` etc., joins each
prose definition into a single line, and emits a Go string-slice literal.
Trailing verse stanzas attached to some entries are dropped — only the
prose part (up to the first blank line) is kept.
"""

from pathlib import Path
import re
import sys
import urllib.request


SOURCE_URL = "https://www.gutenberg.org/files/972/972-0.txt"

HEAD = re.compile(
    r"^[A-Z][A-Z'\-]+(?:\s+[A-Z][A-Z'\-]+)?,\s+"
    r"(?:n\.|adj\.|v\.t\.|v\.i\.|pp\.|adv\.|interj\.|imp\.|conj\.|pron\.|prep\.|\[.*\])"
)


def fetch_body() -> str:
    raw = urllib.request.urlopen(SOURCE_URL).read().decode("utf-8")
    lines = raw.splitlines()
    state = "pre"
    body = []
    for line in lines:
        if state == "pre" and "START OF THE PROJECT" in line:
            state = "body"
            continue
        if state == "body" and "END OF THE PROJECT" in line:
            break
        if state == "body":
            body.append(line)
    return "\n".join(body)


def parse(text: str) -> list[str]:
    entries: list[str] = []
    buf: list[str] = []

    def flush():
        if not buf:
            return
        joined = " ".join(s.strip() for s in buf).strip()
        joined = re.sub(r"\s+", " ", joined)
        joined = joined.replace("_", "")
        if joined:
            entries.append(joined)

    for line in text.splitlines():
        if HEAD.match(line):
            flush()
            buf = [line]
        elif buf:
            if line.strip() == "":
                flush()
                buf = []
            else:
                buf.append(line)
    flush()
    # Drop obvious mis-parses.
    return [e for e in entries if 30 <= len(e) <= 600]


def main():
    body = fetch_body()
    entries = parse(body)
    print(f"// Devil's Dictionary entries — Ambrose Bierce, 1906. Public domain.")
    print(f"// Sourced from Project Gutenberg #972 and parsed with")
    print(f"// scripts/parse-devil.py; regenerate when refreshing the corpus.")
    print(f"package quotes")
    print()
    print(f"// NewDevilsDictionary returns a Source that draws from every prose")
    print(f"// entry in Bierce's dictionary (~{len(entries)} entries).")
    print(f"func NewDevilsDictionary() *Source {{ return newSource(\"Devil's Dictionary\", devilsDictionary) }}")
    print()
    print("var devilsDictionary = []string{")
    for e in entries:
        s = e.replace("\\", "\\\\").replace("\"", "\\\"")
        print(f"\t\"{s}\",")
    print("}")
    print(f"// {len(entries)} entries", file=sys.stderr)


if __name__ == "__main__":
    main()
