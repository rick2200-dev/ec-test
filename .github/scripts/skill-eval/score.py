#!/usr/bin/env python3
"""Aggregate normalized judge JSON files into an overall verdict.

Reads judges/test-*.json (already normalized by run_pairwise_judge.py — this
script does NOT do raw parsing or retries) plus unblind.json, scenarios.json,
and meta.json under --indir, computes per-scenario signed margin, and emits:

  - <indir>/summary.md   (human-readable markdown report)
  - <indir>/meta.json    (updated in place with verdict, judge_failures, per_scenario)

Verdict thresholds (defaults — overridable via --mean-threshold):
  passed:       >= 2 with-win AND 0 without-win AND mean(signed_margin) >= MEAN_THRESHOLD
  failed:       >= 2 without-win
  inconclusive: otherwise
  + judge_failures >= 2  -> forced inconclusive

A `judge_failed: true` field on a judge JSON contributes signed_margin=0 (tie)
and is counted toward the failures total.
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path


MEAN_THRESHOLD_DEFAULT = 1.5


def load_json(p: Path) -> dict | list:
    return json.loads(p.read_text(encoding="utf-8"))


def collect_token_usage(workdir: Path) -> dict:
    """Sum *.tokens.json sidecar files written by call_copilot.sh.

    Returns a dict with per-phase breakdown (orchestrator / implementer / judge)
    and totals. Each phase records: input_chars, output_chars, estimated tokens,
    and (if present) actual_input_tokens / actual_output_tokens.
    """
    phases = {
        "orchestrator": [workdir / "orch_raw.txt.tokens.json"],
        "implementer": list((workdir / "outputs").glob("test-*-*.txt.tokens.json")),
        "judge": [workdir / "judges" / "raw_combined.txt.tokens.json"],
    }
    summary: dict = {"phases": {}, "totals": {
        "calls": 0,
        "input_chars": 0,
        "output_chars": 0,
        "estimated_input_tokens": 0,
        "estimated_output_tokens": 0,
        "estimated_total_tokens": 0,
        "actual_input_tokens": 0,
        "actual_output_tokens": 0,
        "has_actual": False,
    }}
    for phase, paths in phases.items():
        agg = {
            "calls": 0,
            "input_chars": 0,
            "output_chars": 0,
            "estimated_input_tokens": 0,
            "estimated_output_tokens": 0,
            "actual_input_tokens": 0,
            "actual_output_tokens": 0,
            "has_actual": False,
        }
        for p in paths:
            if not p.is_file():
                continue
            try:
                d = json.loads(p.read_text(encoding="utf-8"))
            except Exception:
                continue
            agg["calls"] += 1
            agg["input_chars"] += d.get("input_chars", 0)
            agg["output_chars"] += d.get("output_chars", 0)
            agg["estimated_input_tokens"] += d.get("estimated_input_tokens", 0)
            agg["estimated_output_tokens"] += d.get("estimated_output_tokens", 0)
            if "actual_input_tokens" in d:
                agg["actual_input_tokens"] += d["actual_input_tokens"]
                agg["has_actual"] = True
            if "actual_output_tokens" in d:
                agg["actual_output_tokens"] += d["actual_output_tokens"]
                agg["has_actual"] = True
        summary["phases"][phase] = agg
        for k in ("calls", "input_chars", "output_chars",
                 "estimated_input_tokens", "estimated_output_tokens",
                 "actual_input_tokens", "actual_output_tokens"):
            summary["totals"][k] += agg[k]
        summary["totals"]["has_actual"] = summary["totals"]["has_actual"] or agg["has_actual"]
    summary["totals"]["estimated_total_tokens"] = (
        summary["totals"]["estimated_input_tokens"]
        + summary["totals"]["estimated_output_tokens"]
    )
    return summary


def render_token_usage(usage: dict) -> list[str]:
    """Return markdown lines for the Token Usage section."""
    totals = usage["totals"]
    if totals["calls"] == 0:
        return []
    lines: list[str] = []
    lines.append("### Token usage")
    lines.append("")
    if totals["has_actual"]:
        header = "| Phase | Calls | Input chars | Output chars | Actual in tok | Actual out tok |"
        sep    = "|---|---|---|---|---|---|"
    else:
        header = "| Phase | Calls | Input chars | Output chars | Est. in tok | Est. out tok |"
        sep    = "|---|---|---|---|---|---|"
    lines.append(header)
    lines.append(sep)
    for phase in ("orchestrator", "implementer", "judge"):
        a = usage["phases"].get(phase, {})
        if not a or a.get("calls", 0) == 0:
            continue
        if totals["has_actual"]:
            lines.append(
                f"| {phase} | {a['calls']} | {a['input_chars']:,} | {a['output_chars']:,} | "
                f"{a['actual_input_tokens']:,} | {a['actual_output_tokens']:,} |"
            )
        else:
            lines.append(
                f"| {phase} | {a['calls']} | {a['input_chars']:,} | {a['output_chars']:,} | "
                f"{a['estimated_input_tokens']:,} | {a['estimated_output_tokens']:,} |"
            )
    if totals["has_actual"]:
        lines.append(
            f"| **total** | **{totals['calls']}** | **{totals['input_chars']:,}** | "
            f"**{totals['output_chars']:,}** | **{totals['actual_input_tokens']:,}** | "
            f"**{totals['actual_output_tokens']:,}** |"
        )
    else:
        lines.append(
            f"| **total** | **{totals['calls']}** | **{totals['input_chars']:,}** | "
            f"**{totals['output_chars']:,}** | **{totals['estimated_input_tokens']:,}** | "
            f"**{totals['estimated_output_tokens']:,}** |"
        )
    lines.append("")
    if not totals["has_actual"]:
        lines.append(
            "_Token figures are estimates (4 chars/token). For exact accounting, "
            "modify `call_copilot.sh` to capture the CLI's reported usage and add "
            "`actual_input_tokens` / `actual_output_tokens` to each tokens.json sidecar._"
        )
    lines.append("")
    return lines


def render_summary(
    skill_name: str,
    skill_short_sha: str,
    model: str,
    verdict: str,
    per_scenario: list[dict],
    judge_failures: int,
    mean_signed: float,
    mean_threshold: float,
    token_usage: dict | None = None,
) -> str:
    if verdict == "passed":
        verdict_emoji = "✅"
        verdict_label = "Passed"
    elif verdict == "failed":
        verdict_emoji = "❌"
        verdict_label = "Failed"
    else:
        verdict_emoji = "⚠"
        verdict_label = "Inconclusive"

    lines: list[str] = []
    lines.append(f"## {verdict_emoji} Skill Validation {verdict_label}: `{skill_name}`")
    lines.append("")
    lines.append(
        f"**Skill SHA:** `{skill_short_sha}` | **Model:** `{model}` | "
        f"**Mean signed margin:** {mean_signed:+.2f} (threshold {mean_threshold:+.2f}) | "
        f"**Judge parse failures:** {judge_failures}"
    )
    lines.append("")
    lines.append("| # | Kind | Winner | Margin | Notes |")
    lines.append("|---|---|---|---|---|")
    for row in per_scenario:
        winner_disp = row["winner_resolved"]
        if row.get("judge_failed"):
            winner_disp = f"{winner_disp} (judge_failed)"
        notes = row.get("rubric_notes", "")
        if len(notes) > 80:
            notes = notes[:77] + "..."
        lines.append(
            f"| {row['id']} | {row['scenario_kind']} | {winner_disp} | "
            f"{row['signed_margin']:+d} | {notes} |"
        )
    lines.append("")
    if token_usage is not None:
        lines.extend(render_token_usage(token_usage))
    lines.append(
        f"> ⚠️ **Note**: implementer と judge は同じモデル (`{model}`) を使っています。"
        f"これはこのモデル上での **skill の相対効果** を測定するもので、"
        f"絶対的な skill 品質ではありません。Claude Code 実環境(別モデル)では結果が異なる可能性があります。"
    )
    lines.append("")
    lines.append(
        "<details><summary>Verdict rules</summary>\n\n"
        f"- **passed**: ≥2 with-win AND 0 without-win AND mean(signed) ≥ {mean_threshold}\n"
        "- **failed**: ≥2 without-win\n"
        "- **inconclusive**: otherwise (incl. ≥2 judge parse failures)\n"
        "</details>\n"
    )
    return "\n".join(lines)


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--indir", required=True, type=Path)
    parser.add_argument("--mean-threshold", type=float, default=MEAN_THRESHOLD_DEFAULT)
    args = parser.parse_args(argv)

    indir: Path = args.indir
    judges_dir = indir / "judges"
    unblind_path = indir / "unblind.json"
    scenarios_path = indir / "scenarios.json"
    meta_path = indir / "meta.json"

    for required in (judges_dir, unblind_path, scenarios_path, meta_path):
        if not required.exists():
            print(f"missing required input: {required}", file=sys.stderr)
            return 1

    unblind = load_json(unblind_path)
    scenarios = load_json(scenarios_path)
    meta = load_json(meta_path)

    if not isinstance(unblind, dict) or not isinstance(scenarios, list) or not isinstance(meta, dict):
        print("input shape unexpected", file=sys.stderr)
        return 1

    per_scenario: list[dict] = []
    judge_failures = 0
    with_wins = 0
    without_wins = 0
    signed_total = 0

    for scenario in scenarios:
        sid = str(scenario["id"])
        kind = scenario.get("scenario_kind", "unknown")
        judge_path = judges_dir / f"test-{sid}.json"
        if not judge_path.is_file():
            print(f"missing judge file: {judge_path}", file=sys.stderr)
            return 1
        judge = load_json(judge_path)
        if not isinstance(judge, dict):
            print(f"judge {judge_path} is not an object", file=sys.stderr)
            return 1

        winner = judge.get("winner")
        margin = judge.get("margin", 0)
        judge_failed = bool(judge.get("judge_failed", False))
        if judge_failed:
            judge_failures += 1

        scenario_unblind = unblind.get(sid, {})
        if winner == "tie" or judge_failed:
            winner_resolved = "tie"
            signed = 0
        else:
            mapped = scenario_unblind.get(winner)
            if mapped == "with":
                winner_resolved = "with"
                with_wins += 1
                signed = margin
            elif mapped == "without":
                winner_resolved = "without"
                without_wins += 1
                signed = -margin
            else:
                # unblind missing — degrade safely
                winner_resolved = "unknown"
                signed = 0

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

    n = max(1, len(per_scenario))
    mean_signed = signed_total / n

    if judge_failures >= 2:
        verdict = "inconclusive"
    elif with_wins >= 2 and without_wins == 0 and mean_signed >= args.mean_threshold:
        verdict = "passed"
    elif without_wins >= 2:
        verdict = "failed"
    else:
        verdict = "inconclusive"

    token_usage = collect_token_usage(indir)

    summary_md = render_summary(
        skill_name=meta.get("skill_name", "?"),
        skill_short_sha=meta.get("skill_short_sha", "?"),
        model=meta.get("model", "?"),
        verdict=verdict,
        per_scenario=per_scenario,
        judge_failures=judge_failures,
        mean_signed=mean_signed,
        mean_threshold=args.mean_threshold,
        token_usage=token_usage,
    )

    (indir / "summary.md").write_text(summary_md, encoding="utf-8")

    meta["verdict"] = verdict
    meta["judge_failures"] = judge_failures
    meta["per_scenario"] = per_scenario
    meta["mean_signed_margin"] = mean_signed
    meta["token_usage"] = token_usage
    meta_path.write_text(json.dumps(meta, ensure_ascii=False, indent=2), encoding="utf-8")

    # Print verdict to stdout for the orchestrator to pick up
    print(verdict)
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
