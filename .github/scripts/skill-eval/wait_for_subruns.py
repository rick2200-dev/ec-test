#!/usr/bin/env python3
"""Poll skill-eval-arm sub-workflow runs until completion and download artifacts.

The orchestrator dispatches sub-workflows via `gh workflow run skill-eval-arm.lock.yml`.
NOTE: gh CLI takes the compiled workflow file (`.lock.yml`) or workflow name —
NOT the gh-aw `.md` source. GitHub Actions only registers the `.lock.yml`.

GitHub Actions does not return the new run ID synchronously, so we identify
the dispatched runs by their `run-name`, which encodes the orchestrator's
correlation_id and a per-run key (e.g. "skill-eval-arm: 12345/1-X").

This script:
  1. Polls `gh run list --workflow skill-eval-arm.lock.yml --json ...` until each
     expected run-name appears as a completed run.
  2. Downloads the artifact of each completed run to <outdir>/<key>/.
  3. Records each run's status (success/failure/timeout) plus its run_id.
  4. Writes a summary JSON to --status-out for the orchestrator.

Inputs:
  --correlation-id <id>             # orchestrator's run id, used in run-name
  --workflow <file>                 # default skill-eval-arm.lock.yml
  --expected-keys <csv>             # e.g. "1-X,1-Y,2-X,2-Y,3-X,3-Y" or "judge"
  --outdir <dir>                    # where to put downloaded artifacts
                                    #   (each key gets a subdir)
  --status-out <path>               # JSON dump of {key: {run_id, status, conclusion}, ...}
  --timeout-seconds <int>           # default 600
  --poll-interval <int>             # default 15

Exit code:
  0 if all expected runs reached `status=completed` (regardless of success).
  1 on timeout or unexpected gh CLI failure.

The orchestrator inspects --status-out to decide which runs succeeded.
"""
from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
import time
from pathlib import Path


def gh_run_list(workflow: str, limit: int = 100) -> list[dict]:
    cmd = [
        "gh", "run", "list",
        "--workflow", workflow,
        "--json", "databaseId,name,status,conclusion,createdAt",
        "--limit", str(limit),
    ]
    proc = subprocess.run(cmd, capture_output=True, text=True, encoding="utf-8", check=True)
    return json.loads(proc.stdout or "[]")


def gh_run_download(run_id: int, outdir: Path) -> bool:
    outdir.mkdir(parents=True, exist_ok=True)
    cmd = ["gh", "run", "download", str(run_id), "--dir", str(outdir)]
    proc = subprocess.run(cmd, capture_output=True, text=True, encoding="utf-8", check=False)
    if proc.returncode != 0:
        sys.stderr.write(f"gh run download failed for {run_id}: {proc.stderr}\n")
        return False
    return True


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--correlation-id", required=True)
    parser.add_argument(
        "--workflow",
        default="skill-eval-arm.lock.yml",
        help="GitHub Actions workflow file (the compiled .lock.yml, NOT the gh-aw .md source)",
    )
    parser.add_argument("--expected-keys", required=True)
    parser.add_argument("--outdir", required=True, type=Path)
    parser.add_argument("--status-out", required=True, type=Path)
    parser.add_argument("--timeout-seconds", type=int, default=600)
    parser.add_argument("--poll-interval", type=int, default=15)
    args = parser.parse_args(argv)

    expected = [k.strip() for k in args.expected_keys.split(",") if k.strip()]
    if not expected:
        sys.stderr.write("no expected keys given\n")
        return 1

    args.outdir.mkdir(parents=True, exist_ok=True)
    deadline = time.monotonic() + args.timeout_seconds

    # Map of key -> {run_id, status, conclusion}
    found: dict[str, dict] = {}

    name_pattern = re.compile(
        rf"^skill-eval-arm:\s*{re.escape(args.correlation_id)}/(?P<key>\S+)\s*$"
    )

    sleep_first = True
    while time.monotonic() < deadline:
        if sleep_first:
            time.sleep(min(30, args.poll_interval * 2))  # initial delay for runs to register
            sleep_first = False
        else:
            time.sleep(args.poll_interval)

        try:
            runs = gh_run_list(args.workflow)
        except subprocess.CalledProcessError as e:
            sys.stderr.write(f"gh run list failed: {e.stderr}\n")
            return 1

        for run in runs:
            name = run.get("name", "")
            m = name_pattern.match(name)
            if not m:
                continue
            key = m.group("key")
            if key not in expected:
                continue
            if key in found and found[key]["status"] == "completed":
                continue  # already captured
            found[key] = {
                "run_id": run["databaseId"],
                "status": run.get("status", ""),
                "conclusion": run.get("conclusion"),
            }

        completed_count = sum(
            1 for k in expected if k in found and found[k]["status"] == "completed"
        )
        sys.stderr.write(
            f"[wait_for_subruns] elapsed={int(time.monotonic() - (deadline - args.timeout_seconds))}s "
            f"completed={completed_count}/{len(expected)}\n"
        )
        if completed_count == len(expected):
            break

    # Write status (whether complete or timed out)
    status_out: dict[str, dict] = {}
    for key in expected:
        if key in found:
            status_out[key] = found[key]
        else:
            status_out[key] = {"run_id": None, "status": "timeout", "conclusion": None}

    args.status_out.parent.mkdir(parents=True, exist_ok=True)
    args.status_out.write_text(json.dumps(status_out, indent=2), encoding="utf-8")

    # Download artifacts for completed runs
    download_failures: list[str] = []
    for key, info in status_out.items():
        if info["run_id"] is None:
            continue
        if info["status"] != "completed":
            continue
        target = args.outdir / key
        ok = gh_run_download(info["run_id"], target)
        if not ok:
            download_failures.append(key)

    # Update status with download outcome
    for key in download_failures:
        status_out[key]["download_failed"] = True
    args.status_out.write_text(json.dumps(status_out, indent=2), encoding="utf-8")

    incomplete = [k for k, v in status_out.items() if v["status"] != "completed"]
    if incomplete:
        sys.stderr.write(f"[wait_for_subruns] incomplete keys: {incomplete}\n")
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
