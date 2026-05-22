#!/usr/bin/env python3
"""Regenerate internal/widget/quotes/jargon.go from Project Gutenberg #3008.

Usage:
    python3 scripts/parse-jargon.py > internal/widget/quotes/jargon.go

The Jargon File text is structured around per-term "nodes":

    Node:abbrev, Next:ABEND, Previous:..., Up:= A
    =

    abbrev /*-breev'/, /*-brev'/ n.

    Common abbreviation for `abbreviation'.


    Node:ABEND, ...

We walk node-by-node, join the prose paragraphs into a single line, and
emit `TERM, pos. definition` strings that read like the Devil's
Dictionary entries we already ship.
"""

import re
import sys
import urllib.request


SOURCE_URL = "https://www.gutenberg.org/cache/epub/3008/pg3008.txt"


def fetch() -> str:
    raw = urllib.request.urlopen(SOURCE_URL).read().decode("utf-8-sig")
    return raw.replace("\r\n", "\n")


NODE_RE = re.compile(r"^Node:([^,]+),", re.MULTILINE)
SECTION_RE = re.compile(r"^= [A-Z0-9] =\s*$", re.MULTILINE)
# Strip jargon's `_italic_` markers and turn backtick-quotes into ASCII quotes.
ITALIC_RE = re.compile(r"_([^_\n]+)_")
BACKTICK_RE = re.compile(r"`([^']+)'")
# Real glossary heads end with a part-of-speech marker. Anything that
# doesn't match this is preface, navigation chatter, or a section
# heading, and gets dropped.
POS_RE = re.compile(
    r"(?:\b(?:n|adj|adv|v|vt|vi|vb|excl|interj|imp|conj|pron|prep|pl|sing|abbrev|num|art|comp)\.?"
    r"|\b(?:n,v|n,adj|n,vt|adj,n|adj,vt|v,n)\.?)"
    r"\s*$"
)


def split_nodes(text: str) -> list[tuple[str, str]]:
    """Return (term, body_text) tuples for every Node: in the file."""
    # Find every Node header; the body runs from the line after `=` up to
    # the next Node header (or section break).
    matches = list(NODE_RE.finditer(text))
    out: list[tuple[str, str]] = []
    for i, m in enumerate(matches):
        term = m.group(1).strip()
        body_start = text.find("\n", m.end()) + 1
        # Skip the bare `=` line that follows every Node:
        if body_start < len(text) and text[body_start:body_start + 2] == "=\n":
            body_start = text.find("\n", body_start) + 1
        body_end = matches[i + 1].start() if i + 1 < len(matches) else len(text)
        body = text[body_start:body_end]
        out.append((term, body))
    return out


def clean_body(body: str) -> str:
    """Flatten a node body into a single one-liner.

    Real glossary entries look like (with leading boilerplate above):

        abbrev /*-breev'/, /*-brev'/ n.

        Common abbreviation for `abbreviation'.

    We scan forward to the first line that ends with a recognised
    part-of-speech marker — that's the head — then take the next prose
    paragraph as the definition. Everything else (section dividers like
    "A =", navigation lines, preface text) is dropped.
    """
    lines = body.split("\n")
    head = ""
    head_idx = -1
    for i, line in enumerate(lines):
        stripped = line.strip()
        if not stripped:
            continue
        if POS_RE.search(stripped):
            head = stripped
            head_idx = i
            break
    if not head:
        return ""
    # Skip blank lines after the head; then collect one paragraph of prose.
    rest = lines[head_idx + 1:]
    while rest and not rest[0].strip():
        rest.pop(0)
    para: list[str] = []
    for line in rest:
        if not line.strip():
            break
        para.append(line.strip())
    body_text = " ".join(para).strip()
    if not body_text:
        return ""
    # Strip jargon-isms: `quoted` → "quoted", _italic_ → italic.
    body_text = BACKTICK_RE.sub(r'"\1"', body_text)
    body_text = ITALIC_RE.sub(r"\1", body_text)
    head = BACKTICK_RE.sub(r'"\1"', head)
    head = ITALIC_RE.sub(r"\1", head)
    # Collapse whitespace.
    body_text = re.sub(r"\s+", " ", body_text)
    head = re.sub(r"\s+", " ", head)
    return f"{head} {body_text}"


def main():
    raw = fetch()
    entries: list[str] = []
    for term, body in split_nodes(raw):
        line = clean_body(body)
        if not line:
            continue
        # Filter junk parses and runaway-long entries.
        if len(line) < 30 or len(line) > 800:
            continue
        entries.append(line)
    print("// Jargon File entries — Eric S. Raymond et al., 1996+.")
    print("// Sourced from Project Gutenberg #3008 and parsed with")
    print("// scripts/parse-jargon.py; regenerate when refreshing the corpus.")
    print("package quotes")
    print()
    print(f"// NewJargonFile returns a Source that draws from every prose entry in")
    print(f"// the Jargon File (~{len(entries)} entries).")
    print(f"func NewJargonFile() *Source {{ return newSource(\"Jargon File\", jargonFile) }}")
    print()
    print("var jargonFile = []string{")
    for e in entries:
        s = e.replace("\\", "\\\\").replace("\"", "\\\"")
        print(f"\t\"{s}\",")
    print("}")
    print(f"// {len(entries)} entries", file=sys.stderr)


if __name__ == "__main__":
    main()
