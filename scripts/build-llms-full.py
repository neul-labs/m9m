#!/usr/bin/env python3
"""Generate llms-full.txt: a single concatenated markdown dump of all docs pages.

llms-full.txt is the "full text" companion to llms.txt — large, but lets an LLM ingest
the entire documentation in one fetch. Convention: https://llmstxt.org/

Output: documentation/docs/llms-full.txt (MkDocs copies it verbatim to site root).

Run before each `mkdocs build`. Idempotent.

Usage:
  python3 scripts/build-llms-full.py
"""
from __future__ import annotations

import re
import sys
from pathlib import Path

DOCS_ROOT = Path(__file__).resolve().parent.parent / "documentation" / "docs"
OUTPUT = DOCS_ROOT / "llms-full.txt"
SITE_URL = "https://docs.neullabs.com/m9m"

HEADER = """# m9m Documentation (full text)

> The n8n alternative without the bugs — faster, more reliable workflow automation.

This is the full concatenated documentation in one file, intended for LLMs and search
agents. For a curated index, see llms.txt at the same path. For the rendered HTML, see
{SITE_URL}.

m9m is an open-source workflow automation platform written in Go. It runs n8n workflow
JSON unchanged, executes 5–10× faster, uses 70% less memory, and ships as a single 30 MB
binary with zero runtime dependencies.

---

"""

FRONTMATTER_RE = re.compile(r"^---\n.*?\n---\n+", re.DOTALL)


def page_url(path: Path) -> str:
    rel = path.relative_to(DOCS_ROOT).with_suffix("")
    if rel.name == "index":
        rel = rel.parent
    parts = [p for p in rel.parts if p]
    return f"{SITE_URL}/{'/'.join(parts)}/" if parts else f"{SITE_URL}/"


def strip_frontmatter(text: str) -> str:
    return FRONTMATTER_RE.sub("", text, count=1)


def main() -> int:
    if not DOCS_ROOT.is_dir():
        print(f"Docs root not found: {DOCS_ROOT}", file=sys.stderr)
        return 1

    parts: list[str] = [HEADER.format(SITE_URL=SITE_URL)]
    pages = sorted(p for p in DOCS_ROOT.rglob("*.md") if p.name != "llms-full.txt")

    for md in pages:
        rel = md.relative_to(DOCS_ROOT)
        text = md.read_text(encoding="utf-8")
        body = strip_frontmatter(text).strip()
        parts.append(f"\n\n<!-- ===== {rel} ===== -->\n")
        parts.append(f"# Source: {page_url(md)}\n\n")
        parts.append(body)
        parts.append("\n")

    OUTPUT.write_text("".join(parts), encoding="utf-8")
    print(f"Wrote {OUTPUT} ({OUTPUT.stat().st_size:,} bytes, {len(pages)} pages)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
