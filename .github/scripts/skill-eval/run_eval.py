#!/usr/bin/env python3
"""End-to-end skill validation pipeline driver.

Replaces the multi-step bash orchestration in `skill-eval.yml` with a single
Python entry point. The workflow YAML now does just: permission/fork gate,
checkout, setup, then `python3 run_eval.py ...`, post comment, upload artifact.

Inputs (CLI args):
  --skill-name      e.g. "grpc-integration"
  --head-ref        full SHA or branch name (working tree is already at HEAD)
  --base-ref        full SHA or branch name to compare against
  --pr-number       optional, used only for the meta record
  --workdir         working directory under /tmp/skill-eval/<skill-name>
  --model           Copilot CLI --model value (e.g. "auto" or "claude-sonnet-4.5")
  --max-tokens-impl per-implementer max-tokens
  --max-tokens-judge per-judge max-tokens
  --mean-threshold  verdict threshold (signed margin)

Pipeline:
  1.  Build HEAD arm prompts (build_prompts.py logic, working tree SKILL.md)
  2.  Classify skill change vs BASE: NORMAL / NO_CHANGE / NEW_SKILL
       NO_CHANGE  -> write a "skipped" summary and exit 0
  3.  Build BASE arm prompts (git show + build_prompts logic, or no-skill stub)
  4.  Get cases:
       (a) author seeds in .claude/skills/<name>/gha-eval/case-*.md  -> verbatim
       (b) no seeds -> run orchestrator agent for one fallback generation
  5.  Random X/Y -> base/head assignment per case
  6.  Run implementer × 2N (Copilot CLI agent: skill-eval-implementer)
  7.  Run pairwise judge × 1 (Copilot CLI agent: skill-eval-judge)
  8.  Parse judge JSON array, write per-scenario judges/test-<id>.json
  9.  Compute verdict (improved/regressed/no-change/inconclusive) + summary.md
  10. Update meta.json with timestamps, verdict, token usage

Outputs (in --workdir):
  scenarios.json
  unblind.json
  users/test-<id>-user.txt
  outputs/test-<id>-{X,Y}.txt
  judges/raw_combined.txt
  judges/test-<id>.json
  meta.json
  summary.md
  head/system_prompts/...    (build_prompts artefacts for HEAD arm)
  base/system_prompts/...    (build_prompts artefacts for BASE arm)

Exit code: 0 on completion (including controlled skip). Non-zero only on
unrecoverable error (e.g. CLI auth fails, Copilot CLI not installed).
"""
from __future__ import annotations

import argparse
import datetime
import json
import os
import secrets
import shutil
import subprocess
import sys
from pathlib import Path

# Reuse logic from sibling modules
HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
from build_prompts import (  # type: ignore
    REPO_CONTEXT,
    WITHOUT_STUB,
    strip_frontmatter,
    concat_references,
    build_with_skill,
    build_without_skill,
    build_with_skill_user_prefix,
    build_without_skill_user_prefix,
    build_replay_prompt_yml,
    REPLAY_README,
)
from read_cases import collect as collect_seeds  # type: ignore
from parse_judge_output import extract_json_array, validate_entry  # type: ignore
from score import (  # type: ignore
    collect_token_usage,
    render_summary,
)


REPO_ROOT = Path.cwd()  # GitHub Actions runs at repo root by default


# ---------------------------------------------------------------------------
# Subprocess helpers
# ---------------------------------------------------------------------------

def run(cmd: list[str], **kw) -> subprocess.CompletedProcess:
    return subprocess.run(cmd, **kw)


def git_show(ref: str, path: str) -> str | None:
    """Return the file content at <ref>:<path>, or None if missing."""
    p = run(["git", "show", f"{ref}:{path}"], capture_output=True, text=True)
    if p.returncode != 0:
        return None
    return p.stdout


def git_ls_tree_recursive(ref: str, path: str) -> list[str]:
    """List files under <ref>:<path>/. Returns [] if path missing or empty."""
    p = run(
        ["git", "ls-tree", "-r", "--name-only", ref, "--", path],
        capture_output=True, text=True,
    )
    if p.returncode != 0:
        return []
    return [line for line in p.stdout.splitlines() if line.strip()]


def call_copilot(
    *,
    agent: str,
    user_file: Path,
    model: str,
    max_tokens: int,
    workdir: Path,
    out: Path,
) -> tuple[bool, str]:
    """Invoke the call_copilot.sh wrapper. Returns (ok, stderr)."""
    out.parent.mkdir(parents=True, exist_ok=True)
    workdir.mkdir(parents=True, exist_ok=True)
    cmd = [
        str(REPO_ROOT / ".github/scripts/skill-eval/call_copilot.sh"),
        "--agent", agent,
        "--user", str(user_file),
        "--model", model,
        "--max-tokens", str(max_tokens),
        "--workdir", str(workdir),
        "--out", str(out),
    ]
    p = run(cmd, capture_output=True, text=True)
    return p.returncode == 0, (p.stderr or "")


# ---------------------------------------------------------------------------
# Skip / early-exit summary helpers
# ---------------------------------------------------------------------------

def write_skip_summary(workdir: Path, title: str, body: str) -> None:
    """Write a minimal summary.md and meta.json for an early-exit case."""
    workdir.mkdir(parents=True, exist_ok=True)
    summary = f"## ℹ️ {title}\n\n{body}\n"
    (workdir / "summary.md").write_text(summary, encoding="utf-8")
    meta_path = workdir / "meta.json"
    meta = {}
    if meta_path.is_file():
        try:
            meta = json.loads(meta_path.read_text(encoding="utf-8"))
        except Exception:
            meta = {}
    meta["verdict"] = "skipped"
    meta["skip_reason"] = title
    meta["timestamps"] = meta.get("timestamps", {})
    meta["timestamps"]["end"] = datetime.datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
    meta_path.write_text(json.dumps(meta, ensure_ascii=False, indent=2), encoding="utf-8")


# ---------------------------------------------------------------------------
# Pipeline steps
# ---------------------------------------------------------------------------

def step_build_arm_prompts(
    *,
    skill_name: str,
    skill_md_text: str,
    references: dict[str, str],
    arm_dir: Path,
    short_sha: str,
    model: str,
    max_tokens_impl: int,
) -> None:
    """Run the build_prompts.py logic against in-memory SKILL.md text and
    references, writing to arm_dir (which becomes WORKDIR/head or WORKDIR/base).

    `references` is {filename: content}; written into arm_dir/system_prompts/
    in addition to the standard build_prompts outputs.
    """
    skill_body = strip_frontmatter(skill_md_text).strip()
    if not skill_body:
        skill_body = "(intentionally empty)"
    refs_concat_parts = []
    for fname, content in sorted(references.items()):
        refs_concat_parts.append(f"## Reference: {fname}\n\n{content.rstrip()}\n")
    references_concat = "\n".join(refs_concat_parts)

    sys_dir = arm_dir / "system_prompts"
    replay_dir = arm_dir / "replay"
    sys_dir.mkdir(parents=True, exist_ok=True)
    replay_dir.mkdir(parents=True, exist_ok=True)

    (sys_dir / "skill_body_normalized.md").write_text(skill_body + "\n", encoding="utf-8")
    (sys_dir / "references_concat.md").write_text(references_concat, encoding="utf-8")
    (sys_dir / "with_skill.txt").write_text(
        build_with_skill(skill_name, skill_body, references_concat), encoding="utf-8"
    )
    (sys_dir / "without_skill.txt").write_text(build_without_skill(), encoding="utf-8")
    (sys_dir / "with_skill_user_prefix.txt").write_text(
        build_with_skill_user_prefix(skill_name, skill_body, references_concat),
        encoding="utf-8",
    )
    (sys_dir / "without_skill_user_prefix.txt").write_text(
        build_without_skill_user_prefix(), encoding="utf-8"
    )

    (replay_dir / "prompt.yml").write_text(
        build_replay_prompt_yml(skill_name, short_sha, model, max_tokens_impl),
        encoding="utf-8",
    )
    (replay_dir / "README.md").write_text(REPLAY_README, encoding="utf-8")


def step_classify_skill_change(
    *, skill_name: str, base_ref: str, head_dir: Path, base_dir_src: Path,
) -> tuple[str, str | None, dict[str, str]]:
    """Returns (classification, base_skill_md_text, base_references_dict).

    classification:
      "NORMAL"     -> base has skill, head differs
      "NO_CHANGE"  -> base and head identical (skill + references)
      "NEW_SKILL"  -> base lacks skill (PR adds it)
    """
    skill_path = f".claude/skills/{skill_name}/SKILL.md"
    base_md = git_show(base_ref, skill_path)
    if base_md is None:
        return "NEW_SKILL", None, {}

    # Check if SKILL.md AND the skill's whole tree are identical between
    # base and HEAD. If so, no-change: nothing to validate.
    head_md = (REPO_ROOT / skill_path).read_text(encoding="utf-8")
    head_tree_p = run(
        ["git", "rev-parse", f"HEAD:.claude/skills/{skill_name}"],
        capture_output=True, text=True,
    )
    base_tree_p = run(
        ["git", "rev-parse", f"{base_ref}:.claude/skills/{skill_name}"],
        capture_output=True, text=True,
    )
    if (
        head_md == base_md
        and head_tree_p.returncode == 0
        and base_tree_p.returncode == 0
        and head_tree_p.stdout.strip() == base_tree_p.stdout.strip()
    ):
        return "NO_CHANGE", base_md, {}

    # Collect base references
    refs: dict[str, str] = {}
    for ref_path in git_ls_tree_recursive(base_ref, f".claude/skills/{skill_name}/references"):
        if not ref_path.endswith(".md"):
            continue
        content = git_show(base_ref, ref_path)
        if content is not None:
            refs[Path(ref_path).name] = content
    return "NORMAL", base_md, refs


def step_get_cases(
    *,
    skill_name: str,
    workdir: Path,
    model: str,
) -> list[dict]:
    """Returns list of {"id", "user_prompt", "scenario_kind"}.

    Path A: author seeds present -> use case body verbatim, frontmatter scope
            becomes scenario_kind (default "in-scope").
    Path B: no seeds -> fallback to orchestrator agent for one generation.
    """
    skill_dir = REPO_ROOT / ".claude/skills" / skill_name
    seeds = collect_seeds(skill_dir)

    if seeds:
        cases = []
        for s in seeds:
            scope = s.get("scope") or "in-scope"
            cases.append({
                "id": str(s["id"]),
                "user_prompt": s["body"].strip(),
                "scenario_kind": scope,
            })
        # sort by integer id for stable order
        cases.sort(key=lambda c: int(c["id"]))
        return cases

    # Fallback: orchestrator agent (mode B — no seeds)
    print("No author cases found; running orchestrator agent (fallback).", file=sys.stderr)
    head_skill_body = (workdir / "head/system_prompts/skill_body_normalized.md").read_text(encoding="utf-8")
    user_prompt_lines = [
        f"Skill name (for your eyes only): {skill_name}",
        "",
        "--- BEGIN SKILL.md ---",
        head_skill_body.rstrip(),
        "--- END SKILL.md ---",
        "",
        "Now produce the JSON array of 3 scenarios per the schema and constraints in the agent body. Output the array EXACTLY, starting with [ and ending with ].",
    ]
    user_file = workdir / "orch_user.txt"
    user_file.write_text("\n".join(user_prompt_lines) + "\n", encoding="utf-8")
    raw_out = workdir / "orch_raw.txt"
    iso = workdir / "iso/orch"
    ok, err = call_copilot(
        agent="skill-eval-orchestrator",
        user_file=user_file,
        model=model,
        max_tokens=4096,
        workdir=iso,
        out=raw_out,
    )
    if not ok:
        raise RuntimeError(f"orchestrator call failed: {err}")
    raw = raw_out.read_text(encoding="utf-8")
    arr = extract_json_array(raw)
    if not isinstance(arr, list) or len(arr) == 0:
        raise RuntimeError(f"orchestrator returned non-list / empty: {raw[:300]}")
    cases = []
    for entry in arr:
        if not isinstance(entry, dict):
            continue
        if entry.get("scenario_kind") not in ("in-scope", "edge", "out-of-scope"):
            continue
        if not isinstance(entry.get("user_prompt", ""), str):
            continue
        cases.append({
            "id": str(entry.get("id", "")),
            "user_prompt": entry["user_prompt"],
            "scenario_kind": entry["scenario_kind"],
        })
    if not cases:
        raise RuntimeError("orchestrator returned no valid cases")
    return cases


def step_random_unblind(cases: list[dict]) -> dict[str, dict[str, str]]:
    """For each case id, randomly assign X/Y -> base/head."""
    rng = secrets.SystemRandom()
    unblind: dict[str, dict[str, str]] = {}
    for c in cases:
        sid = str(c["id"])
        if rng.choice([True, False]):
            unblind[sid] = {"X": "head", "Y": "base"}
        else:
            unblind[sid] = {"X": "base", "Y": "head"}
    return unblind


def step_run_implementers(
    *,
    workdir: Path,
    cases: list[dict],
    unblind: dict,
    model: str,
    max_tokens: int,
) -> None:
    outputs_dir = workdir / "outputs"
    users_dir = workdir / "users"
    outputs_dir.mkdir(parents=True, exist_ok=True)
    users_dir.mkdir(parents=True, exist_ok=True)
    for c in cases:
        sid = str(c["id"])
        (users_dir / f"test-{sid}-user.txt").write_text(c["user_prompt"], encoding="utf-8")
        for variant in ("X", "Y"):
            arm = unblind[sid][variant]
            prefix = (workdir / arm / "system_prompts" / "with_skill_user_prefix.txt").read_text(encoding="utf-8")
            task = c["user_prompt"].rstrip()
            combined = f"{prefix.rstrip()}\n\n## Developer Task\n\n{task}\n"
            combined_file = users_dir / f"test-{sid}-{variant}-combined.txt"
            combined_file.write_text(combined, encoding="utf-8")
            out_file = outputs_dir / f"test-{sid}-{variant}.txt"
            iso = workdir / "iso" / f"impl-{sid}-{variant}"
            ok, err = call_copilot(
                agent="skill-eval-implementer",
                user_file=combined_file,
                model=model,
                max_tokens=max_tokens,
                workdir=iso,
                out=out_file,
            )
            if not ok or not out_file.is_file() or out_file.stat().st_size == 0:
                raise RuntimeError(
                    f"implementer test-{sid}-{variant} failed: {err.strip()}"
                )


def step_run_judge(
    *,
    workdir: Path,
    cases: list[dict],
    model: str,
    max_tokens: int,
) -> Path:
    judges_dir = workdir / "judges"
    judges_dir.mkdir(parents=True, exist_ok=True)
    parts = []
    for c in cases:
        sid = str(c["id"])
        task = c["user_prompt"].rstrip()
        x = (workdir / "outputs" / f"test-{sid}-X.txt").read_text(encoding="utf-8").rstrip()
        y = (workdir / "outputs" / f"test-{sid}-Y.txt").read_text(encoding="utf-8").rstrip()
        parts.append(
            f"### Scenario {sid}\n\n[Task]\n{task}\n\n[Variant X]\n{x}\n\n[Variant Y]\n{y}\n"
        )
    judge_user = workdir / "judge_user.txt"
    judge_user.write_text("\n---\n".join(parts) + "\n\nProduce the JSON array now.", encoding="utf-8")
    raw_out = judges_dir / "raw_combined.txt"
    iso = workdir / "iso/judge"
    ok, err = call_copilot(
        agent="skill-eval-judge",
        user_file=judge_user,
        model=model,
        max_tokens=max_tokens,
        workdir=iso,
        out=raw_out,
    )
    if not ok:
        raise RuntimeError(f"judge call failed: {err.strip()}")
    return raw_out


def step_parse_judge(
    *, raw_file: Path, outdir: Path, scenario_ids: list[str]
) -> None:
    """Inline parse_judge_output.py logic (already importable; we use the
    module's helpers but write per-scenario JSON files directly)."""
    outdir.mkdir(parents=True, exist_ok=True)

    def fallback(sid: str, reason: str, raw_excerpt: str = "") -> dict:
        return {
            "scenario": sid,
            "winner": "tie",
            "margin": 0,
            "reason_x": "",
            "reason_y": "",
            "rubric_notes": reason,
            "judge_failed": True,
            "raw_excerpt": raw_excerpt[:200],
        }

    if not raw_file.is_file():
        for sid in scenario_ids:
            (outdir / f"test-{sid}.json").write_text(
                json.dumps(fallback(sid, "judge raw file missing"), indent=2),
                encoding="utf-8",
            )
        return
    raw = raw_file.read_text(encoding="utf-8")
    parsed = extract_json_array(raw)

    by_sid: dict[str, dict] = {}
    if isinstance(parsed, list):
        for entry in parsed:
            if not isinstance(entry, dict):
                continue
            sid_val = entry.get("scenario")
            if sid_val is None:
                continue
            sid_str = str(sid_val)
            if sid_str in scenario_ids and validate_entry(entry):
                entry["scenario"] = sid_str
                by_sid[sid_str] = entry

    for sid in scenario_ids:
        out_path = outdir / f"test-{sid}.json"
        if sid in by_sid:
            out_path.write_text(json.dumps(by_sid[sid], ensure_ascii=False, indent=2), encoding="utf-8")
        else:
            out_path.write_text(
                json.dumps(
                    fallback(sid, f"scenario {sid} missing or invalid in judge output", raw),
                    ensure_ascii=False, indent=2,
                ),
                encoding="utf-8",
            )


def step_score(
    *, workdir: Path, threshold: float, meta: dict
) -> str:
    """Inline score.py main logic. Returns the verdict string."""
    import math as _math

    judges_dir = workdir / "judges"
    unblind = json.loads((workdir / "unblind.json").read_text(encoding="utf-8"))
    scenarios = json.loads((workdir / "scenarios.json").read_text(encoding="utf-8"))

    per_scenario: list[dict] = []
    judge_failures = 0
    head_wins = 0
    base_wins = 0
    signed_total = 0

    for s in scenarios:
        sid = str(s["id"])
        kind = s.get("scenario_kind", "unknown")
        judge_path = judges_dir / f"test-{sid}.json"
        judge = json.loads(judge_path.read_text(encoding="utf-8"))
        winner = judge.get("winner")
        margin = judge.get("margin", 0)
        judge_failed = bool(judge.get("judge_failed", False))
        if judge_failed:
            judge_failures += 1
        scenario_unblind = unblind.get(sid, {})
        if winner == "tie" or judge_failed:
            winner_resolved, signed = "tie", 0
        else:
            mapped = scenario_unblind.get(winner)
            if mapped in ("head", "with"):
                winner_resolved, signed = "head", margin
                head_wins += 1
            elif mapped in ("base", "without"):
                winner_resolved, signed = "base", -margin
                base_wins += 1
            else:
                winner_resolved, signed = "unknown", 0
        signed_total += signed
        per_scenario.append({
            "id": sid,
            "scenario_kind": kind,
            "winner": winner,
            "winner_resolved": winner_resolved,
            "margin": margin,
            "signed_margin": signed,
            "rubric_notes": judge.get("rubric_notes", ""),
            "judge_failed": judge_failed,
        })

    n = len(per_scenario)
    mean_signed = signed_total / max(1, n)
    half = _math.ceil(n / 2) if n > 0 else 1

    if n == 0 or judge_failures >= max(2, half):
        verdict = "inconclusive"
    elif head_wins >= half and base_wins == 0 and mean_signed >= threshold:
        verdict = "improved"
    elif base_wins >= half:
        verdict = "regressed"
    elif head_wins == 0 and base_wins == 0:
        verdict = "no-change"
    else:
        verdict = "inconclusive"

    token_usage = collect_token_usage(workdir)

    summary_md = render_summary(
        skill_name=meta.get("skill_name", "?"),
        skill_short_sha=meta.get("skill_short_sha", "?"),
        model=meta.get("model", "?"),
        verdict=verdict,
        per_scenario=per_scenario,
        judge_failures=judge_failures,
        mean_signed=mean_signed,
        mean_threshold=threshold,
        token_usage=token_usage,
        base_short_sha=meta.get("base_short_sha"),
        new_skill=bool(meta.get("new_skill", False)),
    )
    (workdir / "summary.md").write_text(summary_md, encoding="utf-8")

    meta["verdict"] = verdict
    meta["judge_failures"] = judge_failures
    meta["per_scenario"] = per_scenario
    meta["mean_signed_margin"] = mean_signed
    meta["token_usage"] = token_usage
    return verdict


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--skill-name", required=True)
    parser.add_argument("--head-ref", required=True)
    parser.add_argument("--base-ref", required=True)
    parser.add_argument("--pr-number", default="")
    parser.add_argument("--workdir", required=True, type=Path)
    parser.add_argument("--model", default="auto")
    parser.add_argument("--max-tokens-impl", type=int, default=2048)
    parser.add_argument("--max-tokens-judge", type=int, default=1024)
    parser.add_argument("--mean-threshold", type=float, default=1.5)
    args = parser.parse_args(argv)

    workdir: Path = args.workdir
    workdir.mkdir(parents=True, exist_ok=True)
    for sub in ("head", "base", "iso", "users", "outputs", "judges"):
        (workdir / sub).mkdir(parents=True, exist_ok=True)

    skill_path_rel = f".claude/skills/{args.skill_name}/SKILL.md"
    skill_path = REPO_ROOT / skill_path_rel
    if not skill_path.is_file():
        write_skip_summary(
            workdir,
            "Skill not found in HEAD",
            f"`{args.skill_name}` does not exist at HEAD. Available: "
            + ", ".join(sorted(p.name for p in (REPO_ROOT / ".claude/skills").iterdir()
                                if p.is_dir())) + ".",
        )
        return 0

    head_full = subprocess.run(["git", "rev-parse", "HEAD"], capture_output=True, text=True).stdout.strip()
    head_short = subprocess.run(["git", "rev-parse", "--short", "HEAD"], capture_output=True, text=True).stdout.strip()
    base_full = subprocess.run(["git", "rev-parse", args.base_ref], capture_output=True, text=True).stdout.strip()
    base_short = subprocess.run(["git", "rev-parse", "--short", args.base_ref], capture_output=True, text=True).stdout.strip()

    # ----- Step 1: HEAD arm prompts -----
    head_md_text = skill_path.read_text(encoding="utf-8")
    head_refs: dict[str, str] = {}
    head_refs_dir = skill_path.parent / "references"
    if head_refs_dir.is_dir():
        for p in head_refs_dir.glob("*.md"):
            head_refs[p.name] = p.read_text(encoding="utf-8")
    step_build_arm_prompts(
        skill_name=args.skill_name,
        skill_md_text=head_md_text,
        references=head_refs,
        arm_dir=workdir / "head",
        short_sha=head_short,
        model=args.model,
        max_tokens_impl=args.max_tokens_impl,
    )

    # ----- Step 2: Classify skill change -----
    classification, base_md, base_refs = step_classify_skill_change(
        skill_name=args.skill_name,
        base_ref=args.base_ref,
        head_dir=workdir / "head",
        base_dir_src=workdir / "base_src",
    )
    new_skill = classification == "NEW_SKILL"

    # initial meta
    meta = {
        "skill_name": args.skill_name,
        "skill_sha": head_full,
        "skill_short_sha": head_short,
        "base_sha": base_full,
        "base_short_sha": base_short,
        "new_skill": new_skill,
        "model": args.model,
        "max_tokens_impl": args.max_tokens_impl,
        "max_tokens_judge": args.max_tokens_judge,
        "mean_threshold": args.mean_threshold,
        "pr_number": args.pr_number,
        "execution_order": [],
        "timestamps": {"start": datetime.datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ"), "end": None},
        "verdict": None,
    }
    (workdir / "meta.json").write_text(json.dumps(meta, ensure_ascii=False, indent=2), encoding="utf-8")

    if classification == "NO_CHANGE":
        write_skip_summary(
            workdir,
            f"検証スキップ: `{args.skill_name}` の内容が BASE と HEAD で同一",
            f"BASE (`{base_short}`) と HEAD (`{head_short}`) で skill 本体・references ともに差分がないため、"
            "比較するものがありません。SKILL.md か references/ を変更した PR で再度実行してください。",
        )
        return 0

    # ----- Step 3: BASE arm prompts -----
    if classification == "NEW_SKILL":
        # base arm = no skill loaded. Reuse head's stub prefix.
        base_sys_dir = workdir / "base/system_prompts"
        base_sys_dir.mkdir(parents=True, exist_ok=True)
        stub = (workdir / "head/system_prompts/without_skill_user_prefix.txt").read_text(encoding="utf-8")
        (base_sys_dir / "with_skill_user_prefix.txt").write_text(stub, encoding="utf-8")
    else:
        step_build_arm_prompts(
            skill_name=args.skill_name,
            skill_md_text=base_md or "",
            references=base_refs,
            arm_dir=workdir / "base",
            short_sha=base_short,
            model=args.model,
            max_tokens_impl=args.max_tokens_impl,
        )

    # ----- Step 4: Get cases (author seeds OR orchestrator fallback) -----
    cases = step_get_cases(
        skill_name=args.skill_name,
        workdir=workdir,
        model=args.model,
    )
    (workdir / "scenarios.json").write_text(
        json.dumps(cases, ensure_ascii=False, indent=2), encoding="utf-8"
    )

    # ----- Step 5: Random X/Y unblind -----
    unblind = step_random_unblind(cases)
    (workdir / "unblind.json").write_text(
        json.dumps(unblind, ensure_ascii=False, indent=2), encoding="utf-8"
    )

    # ----- Step 6: Implementers -----
    step_run_implementers(
        workdir=workdir, cases=cases, unblind=unblind,
        model=args.model, max_tokens=args.max_tokens_impl,
    )

    # ----- Step 7: Judge -----
    step_run_judge(
        workdir=workdir, cases=cases,
        model=args.model, max_tokens=args.max_tokens_judge,
    )

    # ----- Step 8: Parse judge output -----
    step_parse_judge(
        raw_file=workdir / "judges/raw_combined.txt",
        outdir=workdir / "judges",
        scenario_ids=[str(c["id"]) for c in cases],
    )

    # ----- Step 9: Score + summary.md -----
    verdict = step_score(workdir=workdir, threshold=args.mean_threshold, meta=meta)

    # finalize timestamps
    meta["timestamps"]["end"] = datetime.datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
    (workdir / "meta.json").write_text(
        json.dumps(meta, ensure_ascii=False, indent=2), encoding="utf-8"
    )

    print(verdict)
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
