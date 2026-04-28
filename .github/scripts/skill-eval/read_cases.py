#!/usr/bin/env python3
"""Read author-provided case seeds from a skill's `gha-eval/` directory.

Walks `.claude/skills/<name>/gha-eval/case-*.md`, extracts optional YAML
frontmatter (`scope`, `focus`) and the markdown body, and emits a JSON array
on stdout that the orchestrator agent uses as seed material.

Each case file is parsed as:

  ---
  scope: in-scope          # optional
  focus: short hint        # optional
  ---

  <free-form body — observation / task hint / draft prompt>

The seed array shape is:

  [
    {"id": "1", "scope": "in-scope" | "edge" | "out-of-scope" | null,
     "focus": "...", "body": "..."},
    ...
  ]

`id` is derived from the filename pattern `case-(\\d+).md`. Files that do not
match the pattern are skipped. Output is sorted by integer id.

If the directory does not exist OR contains zero matching files, exits 0
with `[]` on stdout — the workflow treats that as the "no seed" path.

Usage:
  python3 read_cases.py --skill-dir .claude/skills/<name>
  python3 read_cases.py --skill-dir <dir> --out cases-seed.json
"""
from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path


CASE_FILENAME_RE = re.compile(r"^case-(\d+)\.md$")
ALLOWED_SCOPES = ("in-scope", "edge", "out-of-scope")


def parse_frontmatter(text: str) -> tuple[dict, str]:
    """Return (frontmatter_dict, body_text). Frontmatter uses `key: value`
    lines (one level, no nesting). Empty frontmatter is fine."""
    fm: dict = {}
    if not text.startswith("---"):
        return fm, text.lstrip()
    lines = text.splitlines(keepends=True)
    if len(lines) < 2:
        return fm, text
    end_idx = -1
    for i in range(1, len(lines)):
        if lines[i].rstrip("\r\n") == "---":
            end_idx = i
            break
    if end_idx < 0:
        return fm, text
    for raw in lines[1:end_idx]:
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        if ":" not in line:
            continue
        key, _, val = line.partition(":")
        val = val.strip()
        # Trim surrounding quotes if present
        if len(val) >= 2 and val[0] == val[-1] and val[0] in ("'", '"'):
            val = val[1:-1]
        fm[key.strip()] = val
    body = "".join(lines[end_idx + 1:]).lstrip("\n")
    return fm, body


def load_case(path: Path, sid: str) -> dict | None:
    text = path.read_text(encoding="utf-8")
    fm, body = parse_frontmatter(text)
    body = body.rstrip()
    if not body:
        # An empty body is no useful seed; the orchestrator should skip it.
        return None
    scope = fm.get("scope")
    if scope and scope not in ALLOWED_SCOPES:
        scope = None
    focus = fm.get("focus") or ""
    return {
        "id": sid,
        "scope": scope,
        "focus": focus,
        "body": body,
    }


def collect(skill_dir: Path) -> list[dict]:
    eval_dir = skill_dir / "gha-eval"
    if not eval_dir.is_dir():
        return []
    cases: list[tuple[int, dict]] = []
    for p in eval_dir.iterdir():
        if not p.is_file():
            continue
        m = CASE_FILENAME_RE.match(p.name)
        if not m:
            continue
        sid_int = int(m.group(1))
        seed = load_case(p, str(sid_int))
        if seed is not None:
            cases.append((sid_int, seed))
    cases.sort(key=lambda t: t[0])
    return [c for _, c in cases]


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--skill-dir", required=True, type=Path,
                        help="Path to .claude/skills/<name>")
    parser.add_argument("--out", type=Path, default=None,
                        help="Optional output file path; default writes to stdout")
    args = parser.parse_args(argv)

    if not args.skill_dir.is_dir():
        print(f"skill dir not found: {args.skill_dir}", file=sys.stderr)
        return 1

    seeds = collect(args.skill_dir)
    payload = json.dumps(seeds, ensure_ascii=False, indent=2)
    if args.out:
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(payload + "\n", encoding="utf-8")
    else:
        print(payload)
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
