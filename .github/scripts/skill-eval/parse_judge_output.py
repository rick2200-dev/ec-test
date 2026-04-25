#!/usr/bin/env python3
"""Parse the judge sub-run's combined raw output into per-scenario JSON files.

The judge sub-run is expected to write a JSON array of 3 verdict objects,
one per scenario, ordered by scenario id ascending. This script:

  1. Strips code fences and surrounding prose.
  2. Parses the JSON array.
  3. Validates each element's schema.
  4. Writes one normalized JSON file per scenario to <outdir>/test-<sid>.json.
  5. For any scenario id missing or with invalid schema, writes a fallback
     {"judge_failed": true, ...} JSON in the same shape so score.py can treat
     it uniformly.

Inputs:
  --raw-file <path>           # judge sub-run's response.txt
  --outdir <dir>              # where to write test-{sid}.json files
  --scenario-ids <csv>        # default "1,2,3"

Exit code 0 always (failures are encoded in the per-scenario JSON files via
judge_failed: true). The orchestrator/score.py detect failures, not this
script's exit code.
"""
from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path


_FENCE_RE = re.compile(r"```(?:json)?\s*\n?(.*?)```", re.DOTALL | re.IGNORECASE)
_ARRAY_RE = re.compile(r"\[[\s\S]*\]")


def extract_json_array(raw: str) -> list | None:
    """Return the parsed JSON array if extractable, else None."""
    candidates: list[str] = []

    fences = _FENCE_RE.findall(raw)
    if fences:
        candidates.append(fences[-1].strip())
    candidates.append(raw.strip())

    for cand in candidates:
        m = _ARRAY_RE.search(cand)
        if not m:
            continue
        blob = m.group(0)
        try:
            obj = json.loads(blob)
        except json.JSONDecodeError:
            continue
        if isinstance(obj, list):
            return obj
    return None


def validate_entry(entry: object) -> bool:
    if not isinstance(entry, dict):
        return False
    if entry.get("winner") not in ("X", "Y", "tie"):
        return False
    if entry.get("margin") not in (0, 1, 2):
        return False
    for key in ("reason_x", "reason_y", "rubric_notes"):
        if key not in entry or not isinstance(entry[key], str):
            return False
    return True


def fallback_for(sid: str, reason: str, raw_excerpt: str = "") -> dict:
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


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--raw-file", required=True, type=Path)
    parser.add_argument("--outdir", required=True, type=Path)
    parser.add_argument("--scenario-ids", default="1,2,3")
    args = parser.parse_args(argv)

    scenario_ids = [s.strip() for s in args.scenario_ids.split(",") if s.strip()]
    args.outdir.mkdir(parents=True, exist_ok=True)

    if not args.raw_file.is_file():
        # No raw output at all → all fallback
        for sid in scenario_ids:
            (args.outdir / f"test-{sid}.json").write_text(
                json.dumps(fallback_for(sid, "judge raw file missing"), indent=2),
                encoding="utf-8",
            )
        print(json.dumps({"parsed": 0, "fallback": len(scenario_ids), "ids": scenario_ids}))
        return 0

    raw = args.raw_file.read_text(encoding="utf-8")
    parsed_array = extract_json_array(raw)

    if parsed_array is None:
        # Total failure: all fallback
        for sid in scenario_ids:
            (args.outdir / f"test-{sid}.json").write_text(
                json.dumps(
                    fallback_for(sid, "judge output unparseable as JSON array", raw),
                    indent=2,
                ),
                encoding="utf-8",
            )
        print(json.dumps({"parsed": 0, "fallback": len(scenario_ids), "ids": scenario_ids}))
        return 0

    # Index parsed entries by scenario id (string-coerced)
    by_sid: dict[str, dict] = {}
    for entry in parsed_array:
        if not isinstance(entry, dict):
            continue
        sid_val = entry.get("scenario")
        if sid_val is None:
            continue
        sid_str = str(sid_val)
        if sid_str in scenario_ids and validate_entry(entry):
            entry["scenario"] = sid_str  # normalize to string
            by_sid[sid_str] = entry

    parsed_count = 0
    fallback_count = 0
    for sid in scenario_ids:
        out_path = args.outdir / f"test-{sid}.json"
        if sid in by_sid:
            out_path.write_text(json.dumps(by_sid[sid], ensure_ascii=False, indent=2), encoding="utf-8")
            parsed_count += 1
        else:
            out_path.write_text(
                json.dumps(
                    fallback_for(sid, f"scenario {sid} missing or invalid in judge output", raw),
                    ensure_ascii=False,
                    indent=2,
                ),
                encoding="utf-8",
            )
            fallback_count += 1

    print(json.dumps({"parsed": parsed_count, "fallback": fallback_count, "ids": scenario_ids}))
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
