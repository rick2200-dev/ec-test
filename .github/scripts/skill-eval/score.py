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

# Display label maps (Japanese). Internal identifiers (verdict strings,
# scenario_kind values, winner_resolved values) are kept in English so
# downstream consumers (meta.json, score.py exit prints, parse_judge_output.py)
# do not break.
#
# revision 11: VRT semantics — arms compare base vs head ref of SKILL.md.
#   - improved   = head version produces clearly better outputs
#   - regressed  = head version produces clearly worse outputs
#   - no-change  = no measurable difference (all-tie or near-tie)
#   - inconclusive = judge failures or mixed results
VERDICT_DISPLAY = {
    "improved": ("✅", "改善"),
    "regressed": ("❌", "劣化"),
    "no-change": ("➖", "変化なし"),
    "inconclusive": ("⚠", "判定不能"),
}
SCENARIO_KIND_JA = {
    "in-scope": "領域内",
    "edge": "境界",
    "out-of-scope": "領域外",
}
WINNER_RESOLVED_JA = {
    "head": "新版(HEAD)",
    "base": "旧版(BASE)",
    "tie": "引き分け",
    "unknown": "不明",
}
PHASE_JA = {
    "orchestrator": "シナリオ生成",
    "implementer": "応答生成",
    "judge": "判定",
}


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
    lines.append("### トークン使用量")
    lines.append("")
    if totals["has_actual"]:
        header = "| フェーズ | 呼び出し回数 | 入力文字数 | 出力文字数 | 実測入力トークン | 実測出力トークン |"
    else:
        header = "| フェーズ | 呼び出し回数 | 入力文字数 | 出力文字数 | 推定入力トークン | 推定出力トークン |"
    sep = "|---|---|---|---|---|---|"
    lines.append(header)
    lines.append(sep)
    for phase in ("orchestrator", "implementer", "judge"):
        a = usage["phases"].get(phase, {})
        if not a or a.get("calls", 0) == 0:
            continue
        phase_label = PHASE_JA.get(phase, phase)
        if totals["has_actual"]:
            lines.append(
                f"| {phase_label} | {a['calls']} | {a['input_chars']:,} | {a['output_chars']:,} | "
                f"{a['actual_input_tokens']:,} | {a['actual_output_tokens']:,} |"
            )
        else:
            lines.append(
                f"| {phase_label} | {a['calls']} | {a['input_chars']:,} | {a['output_chars']:,} | "
                f"{a['estimated_input_tokens']:,} | {a['estimated_output_tokens']:,} |"
            )
    if totals["has_actual"]:
        lines.append(
            f"| **合計** | **{totals['calls']}** | **{totals['input_chars']:,}** | "
            f"**{totals['output_chars']:,}** | **{totals['actual_input_tokens']:,}** | "
            f"**{totals['actual_output_tokens']:,}** |"
        )
    else:
        lines.append(
            f"| **合計** | **{totals['calls']}** | **{totals['input_chars']:,}** | "
            f"**{totals['output_chars']:,}** | **{totals['estimated_input_tokens']:,}** | "
            f"**{totals['estimated_output_tokens']:,}** |"
        )
    lines.append("")
    if not totals["has_actual"]:
        lines.append(
            "_トークン数は推定値です（1 トークン ≒ 4 文字として算出）。"
            "正確な数値が必要な場合は `call_copilot.sh` を修正して、"
            "CLI が報告する使用量を取得し、各 `tokens.json` に "
            "`actual_input_tokens` / `actual_output_tokens` を書き込んでください。_"
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
    base_short_sha: str | None = None,
    new_skill: bool = False,
) -> str:
    verdict_emoji, verdict_label = VERDICT_DISPLAY.get(verdict, ("⚠", "判定不能"))

    lines: list[str] = []
    lines.append(f"## {verdict_emoji} Skill 検証: {verdict_label} — `{skill_name}`")
    lines.append("")
    if new_skill:
        lines.append(
            f"> ℹ️ **新規スキル**: BASE 側に `{skill_name}` が存在しないため、"
            "BASE arm は「スキル無し」をベースラインとして比較しています。"
            "verdict は「新規スキルの導入で改善／劣化したか」を表します。"
        )
        lines.append("")
    base_label = f"`{base_short_sha}`" if base_short_sha else "（BASE 不明）"
    lines.append(
        f"**HEAD SHA:** `{skill_short_sha}` ／ **BASE SHA:** {base_label} ／ "
        f"**モデル:** `{model}` ／ "
        f"**平均符号付きマージン:** {mean_signed:+.2f}（しきい値 {mean_threshold:+.2f}）／ "
        f"**Judge パース失敗数:** {judge_failures}"
    )
    lines.append("")
    lines.append("### シナリオ別の結果")
    lines.append("")
    lines.append("| # | 種別 | 勝者 | 符号付きマージン | 講評 |")
    lines.append("|---|---|---|---|---|")
    for row in per_scenario:
        kind_label = SCENARIO_KIND_JA.get(row["scenario_kind"], row["scenario_kind"])
        winner_label = WINNER_RESOLVED_JA.get(row["winner_resolved"], row["winner_resolved"])
        if row.get("judge_failed"):
            winner_label = f"{winner_label}（判定失敗）"
        notes = row.get("rubric_notes", "")
        if len(notes) > 80:
            notes = notes[:77] + "..."
        lines.append(
            f"| {row['id']} | {kind_label} | {winner_label} | "
            f"{row['signed_margin']:+d} | {notes} |"
        )
    lines.append("")
    lines.append(
        "**符号付きマージン**: 正の値 = 新版(HEAD)が優勢、負の値 = 旧版(BASE)が優勢、"
        "0 = 引き分け／判定失敗。絶対値が大きいほど差が明確。"
    )
    lines.append("")
    if token_usage is not None:
        lines.extend(render_token_usage(token_usage))
    lines.append(
        f"> ⚠️ **注意**: 応答生成（implementer）と判定（judge）は同じモデル (`{model}`) を使っています。"
        f"これはこのモデル上での **スキルの相対効果** を測定するもので、"
        f"スキルの絶対的な品質を保証するものではありません。Claude Code 実環境（別モデル）では結果が異なる可能性があります。"
    )
    lines.append("")
    lines.append(
        "<details><summary>判定ルール</summary>\n\n"
        f"- **改善 (improved)**: HEAD 勝ちが ⌈N/2⌉ 件以上 かつ BASE 勝ちが 0 件 かつ 平均符号付きマージン ≥ {mean_threshold}\n"
        "- **劣化 (regressed)**: BASE 勝ちが ⌈N/2⌉ 件以上\n"
        "- **変化なし (no-change)**: HEAD 勝ち・BASE 勝ちがいずれも 0 件（全件引き分け／顕著な差なし）\n"
        "- **判定不能 (inconclusive)**: 上記以外（Judge パース失敗が 2 件以上の場合も含む）\n"
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
    head_wins = 0
    base_wins = 0
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
            # revision 11: arms are now `head` / `base`. Accept legacy
            # `with` / `without` from older runs for graceful migration.
            if mapped in ("head", "with"):
                winner_resolved = "head"
                head_wins += 1
                signed = margin
            elif mapped in ("base", "without"):
                winner_resolved = "base"
                base_wins += 1
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

    import math as _math
    n = len(per_scenario)
    mean_signed = signed_total / max(1, n)
    half = _math.ceil(n / 2) if n > 0 else 1

    if n == 0 or judge_failures >= max(2, half):
        verdict = "inconclusive"
    elif head_wins >= half and base_wins == 0 and mean_signed >= args.mean_threshold:
        verdict = "improved"
    elif base_wins >= half:
        verdict = "regressed"
    elif head_wins == 0 and base_wins == 0:
        verdict = "no-change"
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
        base_short_sha=meta.get("base_short_sha"),
        new_skill=bool(meta.get("new_skill", False)),
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
