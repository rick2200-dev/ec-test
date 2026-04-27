#!/usr/bin/env python3
"""Build normalized system prompts and replay metadata from a SKILL.md file.

Inputs:
  --skill-path <path-to-SKILL.md>
  --skill-name <slug>
  --model <model-id>
  --max-tokens-impl <int>
  --outdir <output dir, e.g. /tmp/skill-eval/<skill>/>

Outputs (created under --outdir):
  system_prompts/skill_body_normalized.md      (SKILL.md without YAML frontmatter)
  system_prompts/references_concat.md          (references/*.md concatenated, empty if none)
  system_prompts/with_skill.txt                (legacy: REPO_CONTEXT + skill body + refs)
  system_prompts/without_skill.txt             (legacy: REPO_CONTEXT + "no skill loaded" stub)
  system_prompts/with_skill_user_prefix.txt    (agent-mode user-prompt prefix; NO REPO_CONTEXT)
  system_prompts/without_skill_user_prefix.txt (agent-mode user-prompt prefix; NO REPO_CONTEXT)
  replay/prompt.yml                            (GitHub Models replay metadata)
  replay/README.md                             (explains replay/prompt.yml semantics)

The `*_user_prefix.txt` outputs are designed for use with
`.github/agents/skill-eval-implementer.agent.md` via
`call_copilot.sh --agent skill-eval-implementer`. The legacy
`with_skill.txt`/`without_skill.txt` files are kept for the
`replay/prompt.yml` artifact (which expects a system_prompt slot) and for
backward compatibility with `--system` mode.
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path


# Canonical copy of REPO_CONTEXT lives in:
#   .github/agents/skill-eval-orchestrator.agent.md
#   .github/agents/skill-eval-implementer.agent.md
# (under the `## Operating Context` section of each).
# This duplicate is kept ONLY for the legacy --system mode and for
# replay/prompt.yml metadata. Edit the agent files first; mirror here.
REPO_CONTEXT = """\
This repository is a Go microservice EC marketplace monorepo (ec-test).
- Backend services live under `backend/services/<name>/` (16 services: auth, cart, catalog, coupon, gateway, inquiry, inventory, loyalty, notification, order, recommend, review, search, shipping, subscription).
- Each service has its own proto definitions under `backend/services/<name>/api/proto/<name>/v1/*.proto`.
- Shared backend code: `backend/pkg/` (Go packages) and `backend/shared/` (e.g. `backend/shared/api/proto/common/v1/common.proto`).
- Frontend apps: `frontend/apps/{admin, buyer, seller, storybook}` (Next.js/TypeScript).
- Frontend shared packages: `frontend/packages/`.
- Other top-level: `infra/` (deployment), `docs/`, `Makefile`, `go.work`, `pnpm-workspace.yaml`.\
"""

WITHOUT_STUB = "No skill is loaded for this session."


def strip_frontmatter(text: str) -> str:
    """Remove YAML frontmatter delimited by leading and matching '---' lines.

    If no frontmatter is present, return text unchanged.
    """
    if not text.startswith("---"):
        return text
    lines = text.splitlines(keepends=True)
    if len(lines) < 2:
        return text
    for i in range(1, len(lines)):
        if lines[i].rstrip("\r\n") == "---":
            return "".join(lines[i + 1:]).lstrip("\n")
    return text


def concat_references(skill_dir: Path) -> str:
    refs_dir = skill_dir / "references"
    if not refs_dir.is_dir():
        return ""
    parts: list[str] = []
    for md in sorted(refs_dir.glob("*.md")):
        parts.append(f"## Reference: {md.name}\n\n{md.read_text(encoding='utf-8').rstrip()}\n")
    return "\n".join(parts)


def build_with_skill(skill_name: str, skill_body: str, references_concat: str) -> str:
    """Legacy --system mode: REPO_CONTEXT + skill body + references."""
    chunks = [REPO_CONTEXT, "", f"## Loaded Skill: {skill_name}", "", skill_body.rstrip()]
    if references_concat.strip():
        chunks.extend(["", references_concat.rstrip()])
    return "\n".join(chunks) + "\n"


def build_without_skill() -> str:
    """Legacy --system mode: REPO_CONTEXT + no-skill stub."""
    return f"{REPO_CONTEXT}\n\n## {WITHOUT_STUB}\n"


def build_with_skill_user_prefix(
    skill_name: str, skill_body: str, references_concat: str
) -> str:
    """Agent-mode user-prompt prefix when the implementer is in skill-loaded mode.

    Does NOT include REPO_CONTEXT — that lives in the agent body
    (.github/agents/skill-eval-implementer.agent.md). The implementer agent
    detects this prefix by the `## Loaded Skill:` header (Mode A in the agent
    body's contract).
    """
    chunks = [f"## Loaded Skill: {skill_name}", "", skill_body.rstrip()]
    if references_concat.strip():
        chunks.extend(["", references_concat.rstrip()])
    return "\n".join(chunks) + "\n"


def build_without_skill_user_prefix() -> str:
    """Agent-mode user-prompt prefix when the implementer is in no-skill mode.

    Detected by the implementer agent via the `## No skill is loaded ...`
    header (Mode B in the agent body's contract).
    """
    return f"## {WITHOUT_STUB}\n"


def build_replay_prompt_yml(
    skill_name: str,
    short_sha: str,
    model: str,
    max_tokens_impl: int,
) -> str:
    """Emit a minimal GitHub Models prompt.yml with empty expected fields.

    testData rows are placeholders; the workflow fills them with the actual
    scenarios + system prompts when generating the artifact for replay. We
    emit just the structural skeleton; the orchestrator overwrites testData
    after scenarios.json is generated.
    """
    return (
        f"name: skill-eval-{skill_name}-{short_sha}\n"
        f"description: Replay metadata. testData[].expected is intentionally empty. "
        f"See replay/README.md.\n"
        f"model: {model}\n"
        f"modelParameters:\n"
        f"  temperature: 0\n"
        f"  maxTokens: {max_tokens_impl}\n"
        f"messages:\n"
        f"  - role: system\n"
        f"    content: \"{{{{system_prompt}}}}\"\n"
        f"  - role: user\n"
        f"    content: \"{{{{user_prompt}}}}\"\n"
        f"testData: []\n"
    )


REPLAY_README = """\
# Replay metadata

This directory contains a GitHub Models prompt definition (`prompt.yml`) for
replaying the implementer calls of this skill validation run.

## Why is `expected` empty?

The standard `gh models eval` workflow populates `testData[].expected` with
reference answers for similarity / relevance scoring. **This artifact is not a
regression baseline.** It is intended as a *replay metadata* file: enough to
re-run the same `(model, system_prompt, user_prompt, temperature, max_tokens)`
combinations and inspect the new outputs.

## What to compare against

The original implementer outputs from this run are saved under
`../outputs/test-{1,2,3}-{X,Y}.txt`. Diff your replay output against those if
you want to detect drift.

## How to replay

```bash
# Re-run all 6 implementer rows (no scoring, just inference)
gh models eval replay/prompt.yml
```

The judge calls cannot be replayed via `gh models eval` because they encode a
pairwise blind comparison that is not native to the eval framework. To replay
a judge call, see `../judges/judge_system_prompt.txt` and
`../judges/judge_user_template.md` and reconstruct the call manually with
`gh models run`.
"""


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--skill-path", required=True, type=Path)
    parser.add_argument("--skill-name", required=True)
    parser.add_argument("--model", required=True)
    parser.add_argument("--max-tokens-impl", type=int, required=True)
    parser.add_argument("--short-sha", required=True)
    parser.add_argument("--outdir", required=True, type=Path)
    args = parser.parse_args(argv)

    skill_path: Path = args.skill_path
    if not skill_path.is_file():
        print(f"SKILL.md not found at {skill_path}", file=sys.stderr)
        return 1

    raw = skill_path.read_text(encoding="utf-8")
    skill_body = strip_frontmatter(raw).strip()
    if not skill_body:
        print(f"SKILL.md body is empty after stripping frontmatter: {skill_path}", file=sys.stderr)
        return 1

    references_concat = concat_references(skill_path.parent)

    outdir: Path = args.outdir
    sys_prompts_dir = outdir / "system_prompts"
    replay_dir = outdir / "replay"
    sys_prompts_dir.mkdir(parents=True, exist_ok=True)
    replay_dir.mkdir(parents=True, exist_ok=True)

    (sys_prompts_dir / "skill_body_normalized.md").write_text(skill_body + "\n", encoding="utf-8")
    (sys_prompts_dir / "references_concat.md").write_text(references_concat, encoding="utf-8")
    (sys_prompts_dir / "with_skill.txt").write_text(
        build_with_skill(args.skill_name, skill_body, references_concat), encoding="utf-8"
    )
    (sys_prompts_dir / "without_skill.txt").write_text(build_without_skill(), encoding="utf-8")
    (sys_prompts_dir / "with_skill_user_prefix.txt").write_text(
        build_with_skill_user_prefix(args.skill_name, skill_body, references_concat),
        encoding="utf-8",
    )
    (sys_prompts_dir / "without_skill_user_prefix.txt").write_text(
        build_without_skill_user_prefix(), encoding="utf-8"
    )

    (replay_dir / "prompt.yml").write_text(
        build_replay_prompt_yml(args.skill_name, args.short_sha, args.model, args.max_tokens_impl),
        encoding="utf-8",
    )
    (replay_dir / "README.md").write_text(REPLAY_README, encoding="utf-8")

    summary = {
        "skill_name": args.skill_name,
        "short_sha": args.short_sha,
        "model": args.model,
        "max_tokens_impl": args.max_tokens_impl,
        "with_skill_chars": len((sys_prompts_dir / "with_skill.txt").read_text(encoding="utf-8")),
        "without_skill_chars": len((sys_prompts_dir / "without_skill.txt").read_text(encoding="utf-8")),
        "has_references": bool(references_concat.strip()),
    }
    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
