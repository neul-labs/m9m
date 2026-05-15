#!/usr/bin/env python3
"""Add SEO frontmatter to MkDocs markdown pages that don't already have it.

For each .md page under documentation/docs/:
  - skip if YAML frontmatter is already present (line 1 is `---`)
  - skip if the file is excluded by name (CHANGELOG, etc.)
  - derive a title from the first `# H1`, or from the slug
  - derive a description from the first non-heading, non-blank paragraph (truncated)
  - inject `title:`, `description:`, `keywords:` frontmatter at the top

Idempotent. Safe to re-run.

Usage:
  python3 scripts/add-frontmatter.py
  python3 scripts/add-frontmatter.py --dry-run
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

DOCS_ROOT = Path(__file__).resolve().parent.parent / "documentation" / "docs"
EXCLUDED = {"llms.txt", "llms-full.txt", "robots.txt"}

DEFAULT_KEYWORDS = (
    "m9m, n8n alternative, workflow automation, Go, MCP server, AI agents, Claude Code, workflow engine, iPaaS"
)
SECTION_KEYWORDS = {
    "nodes": "m9m nodes, n8n nodes, workflow nodes, HTTP, database, AI, messaging, integrations",
    "api": "m9m API, REST API, workflow API, n8n API compatibility",
    "cli": "m9m CLI, command line, m9m serve, m9m exec, workflow CLI",
    "configuration": "m9m configuration, environment variables, server config, database config",
    "deployment": "m9m deployment, Docker, Kubernetes, production workflow automation",
    "architecture": "m9m architecture, workflow engine, job queue, Go runtime",
    "expressions": "m9m expressions, n8n expression syntax, workflow expressions, variables, functions",
    "credentials": "m9m credentials, secret management, API keys, workflow credentials",
    "scheduling": "m9m scheduling, cron, scheduled workflows, recurring jobs",
    "webhooks": "m9m webhooks, webhook triggers, HTTP triggers",
    "workflows": "m9m workflows, n8n workflows, workflow examples, data flow",
    "troubleshooting": "m9m troubleshooting, FAQ, common issues, debugging",
}
TAGLINE_FALLBACK = (
    "m9m is a drop-in n8n alternative in Go — 5–10× faster, 70% lower memory, deterministic execution, zero npm dependencies."
)

H1_RE = re.compile(r"^#\s+(.+?)\s*$", re.MULTILINE)


def has_frontmatter(text: str) -> bool:
    return text.startswith("---\n") or text.startswith("---\r\n")


def derive_title(text: str, fallback: str) -> str:
    m = H1_RE.search(text)
    if m:
        return m.group(1).strip()
    return fallback


def derive_description(text: str, title: str) -> str:
    """Pick the first substantive paragraph after the H1."""
    lines = text.splitlines()
    seen_h1 = False
    paragraph: list[str] = []
    for line in lines:
        stripped = line.strip()
        if not seen_h1:
            if stripped.startswith("#"):
                seen_h1 = True
            continue
        if not stripped:
            if paragraph:
                break
            continue
        if stripped.startswith(("#", "<!--", "<", "|", "-", "*", "+", ">", "```", "    ", "\t", "=")):
            if paragraph:
                break
            continue
        paragraph.append(stripped)
        if sum(len(p) for p in paragraph) > 220:
            break
    if not paragraph:
        return f"{title} — {TAGLINE_FALLBACK}"
    desc = " ".join(paragraph)
    desc = re.sub(r"\s+", " ", desc).strip()
    desc = re.sub(r"`([^`]+)`", r"\1", desc)
    desc = re.sub(r"\*\*([^*]+)\*\*", r"\1", desc)
    desc = re.sub(r"\[([^\]]+)\]\([^)]+\)", r"\1", desc)
    if len(desc) > 160:
        desc = desc[:157].rsplit(" ", 1)[0] + "…"
    return desc


def keywords_for(path: Path) -> str:
    rel = path.relative_to(DOCS_ROOT)
    if rel.parts:
        section = rel.parts[0]
        if section in SECTION_KEYWORDS:
            return SECTION_KEYWORDS[section]
    return DEFAULT_KEYWORDS


def slug_to_title(path: Path) -> str:
    name = path.stem
    if name == "index":
        name = path.parent.name or "m9m"
    return name.replace("-", " ").replace("_", " ").title()


def process(path: Path, dry_run: bool) -> str:
    text = path.read_text(encoding="utf-8")
    if has_frontmatter(text):
        return "skip-existing"

    title = derive_title(text, slug_to_title(path))
    description = derive_description(text, title)
    keywords = keywords_for(path)

    def yaml_quote(s: str) -> str:
        return '"' + s.replace("\\", "\\\\").replace('"', '\\"') + '"'

    frontmatter = (
        "---\n"
        f"title: {yaml_quote(title)}\n"
        f"description: {yaml_quote(description)}\n"
        f"keywords: {yaml_quote(keywords)}\n"
        "---\n\n"
    )
    new_text = frontmatter + text

    if dry_run:
        return "would-add"
    path.write_text(new_text, encoding="utf-8")
    return "added"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dry-run", action="store_true", help="Don't write files, just report.")
    args = parser.parse_args()

    if not DOCS_ROOT.is_dir():
        print(f"Docs root not found: {DOCS_ROOT}", file=sys.stderr)
        return 1

    counts: dict[str, int] = {"added": 0, "skip-existing": 0, "would-add": 0}
    for md in sorted(DOCS_ROOT.rglob("*.md")):
        if md.name in EXCLUDED:
            continue
        status = process(md, args.dry_run)
        counts[status] = counts.get(status, 0) + 1
        rel = md.relative_to(DOCS_ROOT)
        print(f"  {status:14s}  {rel}")

    print()
    for k, v in counts.items():
        print(f"{k}: {v}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
