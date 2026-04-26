---
name: "Skill Validation (Orchestrator)"
description: A/B test a Claude Code skill via blind pairwise judging using fan-out sub-workflows. Triggered manually from the Actions tab or with /skill-eval <skill-name> in a PR comment.
on:
  workflow_dispatch:
    inputs:
      skill:
        description: "Pick a known skill (or '(custom)' + fill custom_skill_name)"
        required: true
        type: choice
        options:
          - architect-review
          - ec-backend-impl
          - gh-aw
          - grpc-integration
          - "(custom)"
      custom_skill_name:
        description: "Skill name (used when skill='(custom)')"
        required: false
        type: string
      pr_number:
        description: "PR number to post results to (optional)"
        required: false
        type: string
      ref:
        description: "Git ref / SHA to evaluate (default: branch tip)"
        required: false
        type: string
  slash_command:
    name: skill-eval
    events: [pull_request_comment]
  roles: [admin, maintainer, write]
  reaction: "eyes"
  status-comment: true
concurrency:
  group: skill-eval-${{ github.event.issue.number && format('pr-{0}', github.event.issue.number) || format('dispatch-{0}', github.run_id) }}
  cancel-in-progress: false
timeout-minutes: 15
permissions:
  contents: read
  pull-requests: read
  issues: read
safe-outputs:
  add-comment:
    max: 1
  report-failure-as-issue: false
engine:
  id: copilot
  model: claude-sonnet-4.6   # Pin model. If your tier doesn't expose 4.6 ("not available" error),
                             # fall back to claude-sonnet-4.5 (broadly available default).
                             # `vars.GH_AW_MODEL_AGENT_COPILOT` would otherwise apply if set;
                             # docs (https://github.github.com/gh-aw/reference/engines/) treat
                             # frontmatter `model:` as the per-workflow override.
runtimes:
  python:
    version: "3.12"
checkout:
  fetch-depth: 0
tools:
  github:
    toolsets: [repos, pull_requests, issues]
  bash:
    - gh
    - git
    - jq
    - cat
    - head
    - tail
    - wc
    - grep
    - find
    - sort
    - uniq
    - bash
    - python3
    - mkdir
    - touch
    - date
    - base64
    - awk
    - sed
post-steps:
  - name: Upload agent artifact
    if: always()
    uses: actions/upload-artifact@v4
    with:
      name: skill-eval-agent-${{ github.run_id }}
      path: /tmp/skill-eval/**
      retention-days: 7
      if-no-files-found: ignore
jobs:
  # COMPATIBILITY NOTE: official gh-aw docs describe `jobs:` as custom jobs that run
  # BEFORE the agent. We use `needs: agent` to run AFTER the agent — works on
  # current compiler but is non-standard per docs. If upstream tightens `jobs:`
  # semantics, options:
  #   B) Move dispatch into a separate workflow triggered by `workflow_run`.
  #   C) Convert orchestrator to plain GH Actions YAML (drop gh-aw).
  # Reference: https://github.github.com/gh-aw/reference/frontmatter/
  dispatch:
    needs: agent
    if: always()
    runs-on: ubuntu-latest
    timeout-minutes: 25
    permissions:
      contents: read
      actions: write
      pull-requests: write
      issues: write    # `gh pr comment` posts via the issue-comments API
    steps:
      - name: Checkout
        uses: actions/checkout@v6
        with:
          fetch-depth: 1
      - name: Setup Python
        uses: actions/setup-python@v5
        with:
          python-version: "3.12"
      - name: Download agent artifact
        uses: actions/download-artifact@v4
        with:
          name: skill-eval-agent-${{ github.run_id }}
          path: /tmp/skill-eval-input
      - name: Dispatch sub-workflows and finalize
        env:
          GH_TOKEN: ${{ github.token }}
          ORCH_RUN_ID: ${{ github.run_id }}
          # ===== TRUST BOUNDARY =====
          # The dispatch job has actions:write + issues:write. Trust-sensitive
          # values (sub-workflow ref, PR number, skill name) MUST come from the
          # original event payload — NOT from agent-controlled artifact data.
          EVENT_NAME: ${{ github.event_name }}
          DISPATCH_PR_NUMBER: ${{ github.event.inputs.pr_number || github.event.issue.number }}
          DISPATCH_REF_INPUT: ${{ github.event.inputs.ref }}
          DISPATCH_SKILL_INPUT: ${{ github.event.inputs.skill }}
          DISPATCH_CUSTOM_SKILL: ${{ github.event.inputs.custom_skill_name }}
          COMMENT_BODY: ${{ github.event.comment.body }}
        run: |
          set -euo pipefail

          # ----- Step A: derive TRUSTED_* from event payload directly -----
          TRUSTED_PR_NUMBER=""
          TRUSTED_REF=""
          TRUSTED_SKILL_NAME=""

          if [ "$EVENT_NAME" = "workflow_dispatch" ]; then
            if [ "$DISPATCH_SKILL_INPUT" = "(custom)" ]; then
              TRUSTED_SKILL_NAME="$DISPATCH_CUSTOM_SKILL"
            else
              TRUSTED_SKILL_NAME="$DISPATCH_SKILL_INPUT"
            fi
            TRUSTED_PR_NUMBER="$DISPATCH_PR_NUMBER"
            TRUSTED_REF="$DISPATCH_REF_INPUT"
          else
            # slash_command (issue_comment-derived)
            TRUSTED_PR_NUMBER="$DISPATCH_PR_NUMBER"
            TRUSTED_SKILL_NAME=$(
              printf '%s' "$COMMENT_BODY" \
                | grep -oE '/skill-eval[[:space:]]+[^[:space:]]+' \
                | head -1 \
                | awk '{print $2}'
            )
          fi

          # ----- Step B: validate format -----
          if [ -z "$TRUSTED_SKILL_NAME" ] || ! printf '%s' "$TRUSTED_SKILL_NAME" | grep -qE '^[a-zA-Z0-9._-]+$'; then
            echo "Invalid or missing trusted skill name. Refusing to dispatch." | tee -a "$GITHUB_STEP_SUMMARY"
            exit 0
          fi
          case "$TRUSTED_SKILL_NAME" in
            *..*|/*|*/*) echo "Skill name contains traversal characters. Refusing." >> "$GITHUB_STEP_SUMMARY"; exit 0 ;;
          esac

          if [ -n "$TRUSTED_PR_NUMBER" ] && ! printf '%s' "$TRUSTED_PR_NUMBER" | grep -qE '^[0-9]+$'; then
            echo "Invalid PR number format. Refusing." >> "$GITHUB_STEP_SUMMARY"
            exit 0
          fi

          # ----- Step C: derive TRUSTED_REF -----
          if [ "$EVENT_NAME" != "workflow_dispatch" ] && [ -n "$TRUSTED_PR_NUMBER" ]; then
            TRUSTED_REF=$(gh pr view "$TRUSTED_PR_NUMBER" --json headRefOid -q .headRefOid)
          fi
          if [ -n "$TRUSTED_REF" ] && ! printf '%s' "$TRUSTED_REF" | grep -qE '^[0-9a-fA-F]{7,40}$|^[a-zA-Z0-9._/-]+$'; then
            echo "Invalid ref format. Refusing." >> "$GITHUB_STEP_SUMMARY"
            exit 0
          fi
          if [ -z "$TRUSTED_REF" ]; then
            TRUSTED_REF=$(gh api repos/${{ github.repository }} -q .default_branch)
          fi

          # ----- Step D: defense-in-depth fork PR guard -----
          if [ -n "$TRUSTED_PR_NUMBER" ]; then
            IS_FORK=$(gh pr view "$TRUSTED_PR_NUMBER" --json isCrossRepository -q .isCrossRepository 2>/dev/null || echo "false")
            if [ "$IS_FORK" = "true" ]; then
              echo "Fork PR detected at dispatch boundary. Refusing." | tee -a "$GITHUB_STEP_SUMMARY"
              exit 0
            fi
          fi

          # ----- Step E: locate agent artifact strictly under TRUSTED_SKILL_NAME -----
          INPUT_ROOT=/tmp/skill-eval-input
          if [ ! -d "$INPUT_ROOT/$TRUSTED_SKILL_NAME" ]; then
            echo "Agent artifact missing skill dir '$TRUSTED_SKILL_NAME'. Skip (early exit)." | tee -a "$GITHUB_STEP_SUMMARY"
            exit 0
          fi
          if [ ! -f "$INPUT_ROOT/$TRUSTED_SKILL_NAME/proceed.flag" ]; then
            echo "Agent did not write proceed.flag for '$TRUSTED_SKILL_NAME'. Skip." | tee -a "$GITHUB_STEP_SUMMARY"
            exit 0
          fi

          # ----- Step F: cross-check artifact's meta.json — abort on tampering -----
          ARTIFACT_SKILL=$(jq -r '.skill_name // ""' "$INPUT_ROOT/$TRUSTED_SKILL_NAME/meta.json" 2>/dev/null || echo "")
          ARTIFACT_PR=$(jq -r '.pr_number // ""' "$INPUT_ROOT/$TRUSTED_SKILL_NAME/meta.json" 2>/dev/null || echo "")
          if [ -n "$ARTIFACT_SKILL" ] && [ "$ARTIFACT_SKILL" != "$TRUSTED_SKILL_NAME" ]; then
            echo "Trust-boundary mismatch: meta.json skill='$ARTIFACT_SKILL' vs trusted='$TRUSTED_SKILL_NAME'." | tee -a "$GITHUB_STEP_SUMMARY"
            exit 0
          fi
          if [ -n "$ARTIFACT_PR" ] && [ "$ARTIFACT_PR" != "$TRUSTED_PR_NUMBER" ]; then
            echo "Trust-boundary mismatch: meta.json pr_number='$ARTIFACT_PR' vs trusted='$TRUSTED_PR_NUMBER'." | tee -a "$GITHUB_STEP_SUMMARY"
            exit 0
          fi

          # ----- Step G: bind trusted values + materialize WORKDIR -----
          SKILL_NAME="$TRUSTED_SKILL_NAME"
          REF="$TRUSTED_REF"
          PR_NUMBER="$TRUSTED_PR_NUMBER"
          export SKILL_NAME REF PR_NUMBER

          WORKDIR=/tmp/skill-eval/$SKILL_NAME
          export WORKDIR
          mkdir -p "$WORKDIR/subruns"
          cp -r "$INPUT_ROOT/$SKILL_NAME/." "$WORKDIR/"

          # Overwrite trust-sensitive fields in meta.json with trusted values
          python3 - <<'PY'
          import json, os, pathlib
          p = pathlib.Path(os.environ["WORKDIR"]) / "meta.json"
          m = json.loads(p.read_text()) if p.exists() else {}
          m["skill_name"] = os.environ["SKILL_NAME"]
          m["skill_sha"] = os.environ["REF"]
          m["pr_number"] = os.environ["PR_NUMBER"]
          p.write_text(json.dumps(m, ensure_ascii=False, indent=2))
          PY

          # ----- Step H: dispatch 6 implementer sub-workflows -----
          for id in 1 2 3; do
            for variant in X Y; do
              ARM=$(jq -r ".\"$id\".\"$variant\"" "$WORKDIR/unblind.json")
              SYS_FILE="$WORKDIR/system_prompts/${ARM}_skill.txt"
              USER_FILE="$WORKDIR/users/test-${id}-user.txt"
              SYS_B64=$(base64 -w0 "$SYS_FILE")
              USR_B64=$(base64 -w0 "$USER_FILE")

              gh workflow run skill-eval-arm.lock.yml --ref "$REF" \
                -f mode=implementer \
                -f skill_name="$SKILL_NAME" \
                -f ref="$REF" \
                -f correlation_id="$ORCH_RUN_ID" \
                -f scenario_id="$id" \
                -f variant_label="$variant" \
                -f system_context_b64="$SYS_B64" \
                -f user_prompt_b64="$USR_B64"

              echo "{\"phase\":\"implementer\",\"id\":\"$id\",\"variant\":\"$variant\",\"arm\":\"$ARM\"}" \
                >> "$WORKDIR/subruns/dispatch.log"
            done
          done

          # ----- Step I: wait for all 6 implementers -----
          python3 .github/scripts/skill-eval/wait_for_subruns.py \
            --correlation-id "$ORCH_RUN_ID" \
            --workflow skill-eval-arm.lock.yml \
            --expected-keys "1-X,1-Y,2-X,2-Y,3-X,3-Y" \
            --outdir "$WORKDIR/subruns/implementer" \
            --status-out "$WORKDIR/subruns/implementer-status.json" \
            --timeout-seconds 900 || true

          # Move outputs and record run_ids
          python3 - <<'PY'
          import json, shutil, pathlib, os
          workdir = pathlib.Path(os.environ["WORKDIR"])
          status = json.load(open(workdir / "subruns/implementer-status.json"))
          runs_for_judge = []
          for sid in ("1", "2", "3"):
              entry = {"scenario": sid}
              for v in ("X", "Y"):
                  key = f"{sid}-{v}"
                  info = status.get(key, {})
                  entry[v] = info.get("run_id")
                  src = workdir / f"subruns/implementer/{key}/response.txt"
                  dst = workdir / f"outputs/test-{sid}-{v}.txt"
                  dst.parent.mkdir(parents=True, exist_ok=True)
                  if src.is_file() and src.stat().st_size > 0:
                      shutil.copy(src, dst)
                  else:
                      dst.write_text("")
              runs_for_judge.append(entry)
          json.dump(runs_for_judge, open(workdir / "subruns/impl_runs.json", "w"), indent=2)
          PY

          EMPTY=$(find "$WORKDIR/outputs" -name "test-*-?.txt" -size 0 | wc -l)
          if [ "$EMPTY" -gt 0 ]; then
            cat > "$WORKDIR/summary.md" <<EOM
          ## ⚠ Skill Validation Inconclusive: \`$SKILL_NAME\`

          $EMPTY implementer sub-run(s) failed or returned empty output. Cannot reach a verdict.

          Implementer status:
          \`\`\`
          $(cat "$WORKDIR/subruns/implementer-status.json")
          \`\`\`
          EOM
            cat "$WORKDIR/summary.md" >> "$GITHUB_STEP_SUMMARY"
            if [ -n "$PR_NUMBER" ]; then
              gh pr comment "$PR_NUMBER" --body-file "$WORKDIR/summary.md" || true
            fi
            exit 0
          fi

          # ----- Step J: dispatch 1 judge sub-workflow -----
          IMPL_RUNS_B64=$(jq -c '.' "$WORKDIR/subruns/impl_runs.json" | base64 -w0)
          USER_PROMPTS_B64=$(jq -c '[.[] | {scenario: .id, prompt: .user_prompt}]' "$WORKDIR/scenarios.json" | base64 -w0)

          gh workflow run skill-eval-arm.lock.yml --ref "$REF" \
            -f mode=judge \
            -f skill_name="$SKILL_NAME" \
            -f ref="$REF" \
            -f correlation_id="$ORCH_RUN_ID" \
            -f implementer_runs_b64="$IMPL_RUNS_B64" \
            -f user_prompts_b64="$USER_PROMPTS_B64"

          echo "{\"phase\":\"judge\"}" >> "$WORKDIR/subruns/dispatch.log"

          # ----- Step K: wait for judge -----
          python3 .github/scripts/skill-eval/wait_for_subruns.py \
            --correlation-id "$ORCH_RUN_ID" \
            --workflow skill-eval-arm.lock.yml \
            --expected-keys "judge" \
            --outdir "$WORKDIR/subruns/judge" \
            --status-out "$WORKDIR/subruns/judge-status.json" \
            --timeout-seconds 600 || true

          JUDGE_RAW="$WORKDIR/subruns/judge/judge/response.txt"
          mkdir -p "$WORKDIR/judges"
          if [ ! -s "$JUDGE_RAW" ]; then
            cat > "$WORKDIR/summary.md" <<EOM
          ## ⚠ Skill Validation Inconclusive: \`$SKILL_NAME\`

          Judge sub-run failed or returned empty output.
          EOM
            cat "$WORKDIR/summary.md" >> "$GITHUB_STEP_SUMMARY"
            if [ -n "$PR_NUMBER" ]; then
              gh pr comment "$PR_NUMBER" --body-file "$WORKDIR/summary.md" || true
            fi
            exit 0
          fi
          cp "$JUDGE_RAW" "$WORKDIR/judges/raw_combined.txt"

          # ----- Step L: parse + score -----
          python3 .github/scripts/skill-eval/parse_judge_output.py \
            --raw-file "$WORKDIR/judges/raw_combined.txt" \
            --outdir "$WORKDIR/judges" \
            --scenario-ids "1,2,3"

          python3 .github/scripts/skill-eval/score.py \
            --indir "$WORKDIR" \
            --mean-threshold 1.5

          # Update meta.json end timestamp
          python3 - <<'PY'
          import json, datetime, os, pathlib
          p = pathlib.Path(os.environ["WORKDIR"]) / "meta.json"
          m = json.loads(p.read_text())
          m["timestamps"] = m.get("timestamps", {})
          m["timestamps"]["end"] = datetime.datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
          p.write_text(json.dumps(m, ensure_ascii=False, indent=2))
          PY

          # ----- Step M: emit results -----
          cat "$WORKDIR/summary.md" >> "$GITHUB_STEP_SUMMARY"
          if [ -n "$PR_NUMBER" ]; then
            gh pr comment "$PR_NUMBER" --body-file "$WORKDIR/summary.md" || true
          fi
      - name: Upload final artifact
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: skill-eval-${{ github.run_id }}
          path: /tmp/skill-eval/**
          retention-days: 30
          if-no-files-found: ignore
---

# Skill Validation Orchestrator (agent — scenario generation only)

You are the agent half of the **Skill Validation Orchestrator**. Your job is limited to:

1. Resolving trigger inputs and applying the fork-PR guard.
2. Validating the target skill exists.
3. Producing the normalized system prompts via `build_prompts.py`.
4. Generating 3 anti-bias test scenarios with creative thinking.
5. Randomly assigning each scenario's X/Y variants to with/without arms.
6. Writing `meta.json` and a `proceed.flag` marker that signals the dispatch job to proceed.

The follow-up `dispatch` job (declared in `jobs:` in this same workflow) runs after you exit. It dispatches the 6 implementer sub-workflows + 1 judge sub-workflow, collects their outputs, scores the result, and posts the PR comment. **You DO NOT dispatch sub-workflows yourself** — this agent job is read-only and lacks `actions: write`.

You MUST NOT modify any file in the repository. You only run helper scripts and write to `/tmp/skill-eval/<SKILL_NAME>/`. The post-step uploads that directory as an artifact named `skill-eval-agent-${{ github.run_id }}`.

## Constants

| name | value |
|---|---|
| `MODEL` | Pinned in frontmatter as `engine.model: claude-sonnet-4.6`. Both this orchestrator and `skill-eval-arm.md` (sub-workflow) declare the same model so the A/B comparison is internally consistent. To switch models, edit `engine.model` in BOTH files and recompile. `$COPILOT_MODEL` is also exported by gh-aw at runtime — bash steps can echo it for diagnostics. |
| `N_TESTS` | 3 |
| `MEAN_THRESHOLD` | 1.5 (used downstream by `score.py`) |
| `WORKDIR` | `/tmp/skill-eval/<SKILL_NAME>/` |

## Step 1: Resolve trigger inputs

### Path A — `workflow_dispatch`

```bash
SKILL_INPUT='${{ github.event.inputs.skill }}'
CUSTOM_NAME='${{ github.event.inputs.custom_skill_name }}'
if [ "$SKILL_INPUT" = "(custom)" ]; then
  SKILL_NAME="$CUSTOM_NAME"
  if [ -z "$SKILL_NAME" ]; then
    echo "skill='(custom)' was selected but custom_skill_name is empty" >> "$GITHUB_STEP_SUMMARY"
    exit 0
  fi
else
  SKILL_NAME="$SKILL_INPUT"
fi
PR_NUMBER='${{ github.event.inputs.pr_number }}'
REF='${{ github.event.inputs.ref }}'
```

### Path B — slash command (`pull_request_comment`)

```bash
PR_NUMBER='${{ github.event.issue.number }}'
COMMENT_TEXT="${{ steps.sanitized.outputs.text }}"
SKILL_NAME=$(echo "$COMMENT_TEXT" | grep -oE '/skill-eval[[:space:]]+[^[:space:]]+' | head -1 | awk '{print $2}')
if [ -z "$SKILL_NAME" ]; then
  AVAILABLE=$(ls .claude/skills/ 2>/dev/null | sort | tr '\n' ' ')
  # Post via safe-outputs.add-comment:
  #   "Usage: /skill-eval <skill-name>. Available: $AVAILABLE"
  exit 0    # no proceed.flag → dispatch job will skip
fi
REF=$(gh pr view "$PR_NUMBER" --json headRefOid -q .headRefOid)
```

## Step 2: Fork PR guard (BOTH paths, when PR_NUMBER is set)

```bash
if [ -n "$PR_NUMBER" ]; then
  IS_FORK=$(gh pr view "$PR_NUMBER" --json isCrossRepository -q .isCrossRepository)
  if [ "$IS_FORK" = "true" ]; then
    # safe-outputs.add-comment:
    #   "⚠️ Skill validation does not run on fork PRs. ..."
    exit 0    # no proceed.flag → dispatch job will skip
  fi
fi
```

## Step 3: Checkout the evaluation ref

```bash
if [ -n "$REF" ]; then
  git fetch origin "$REF" || true
  git checkout "$REF"
fi
SHORT_SHA=$(git rev-parse --short HEAD)
FULL_SHA=$(git rev-parse HEAD)
```

## Step 4: Validate skill exists

```bash
SKILL_PATH=".claude/skills/${SKILL_NAME}/SKILL.md"
if [ ! -f "$SKILL_PATH" ]; then
  AVAILABLE=$(ls .claude/skills/ 2>/dev/null | sort | tr '\n' ' ')
  # safe-outputs.add-comment: "Skill not found: \`$SKILL_NAME\`. Available: $AVAILABLE"
  exit 0    # no proceed.flag
fi
WORKDIR="/tmp/skill-eval/${SKILL_NAME}"
mkdir -p "$WORKDIR/users"
```

## Step 5: Build normalized system prompts + replay metadata

```bash
MODEL="${COPILOT_MODEL:-claude-sonnet-4.6}"   # Mirrors engine.model in frontmatter; bash fallback for diagnostics only.
python3 .github/scripts/skill-eval/build_prompts.py \
  --skill-path "$SKILL_PATH" \
  --skill-name "$SKILL_NAME" \
  --model "$MODEL" \
  --max-tokens-impl 2048 \
  --short-sha "$SHORT_SHA" \
  --outdir "$WORKDIR"
```

## Step 6: Generate 3 anti-bias scenarios

Read `$SKILL_PATH` carefully (frontmatter description + body). Generate exactly 3 test scenarios and write them to `$WORKDIR/scenarios.json` as a JSON array of objects with keys `id`, `user_prompt`, `scenario_kind`.

**Constraints (you MUST follow all of them):**

1. Each scenario reads like a **realistic developer task in this monorepo** (no toy projects, no invented services).
2. **Do NOT use skill-specific vocabulary in `user_prompt`.** The user prompt must be plain natural language describing what the developer wants. The skill's body is what the with-skill arm sees in its context; the user prompt should not echo it.
   - Example: validating the `gh-aw` skill — DO NOT write "I want to add an agentic workflow that..." Instead: "I want to set up something that automatically reviews pull requests every Friday."
3. **Composition** (one of each):
   - `scenario_kind: "in-scope"` — a task where the skill clearly should help.
   - `scenario_kind: "edge"` — adjacent to the skill's domain.
   - `scenario_kind: "out-of-scope"` — unrelated to the skill, to detect over-fire.
4. **At least one scenario must test a failure / misdirection mode** — a task where naïve application of the skill could produce a wrong recommendation.
5. Each scenario MUST be in scope of THIS repo (its services, frontend, tooling).

After saving `scenarios.json`, also write each scenario's `user_prompt` to `$WORKDIR/users/test-<id>-user.txt`.

## Step 7: Random X/Y assignment per scenario

```bash
python3 - <<'PY'
import json, secrets, os
rng = secrets.SystemRandom()
unblind = {}
for sid in ("1", "2", "3"):
    if rng.choice([True, False]):
        unblind[sid] = {"X": "with", "Y": "without"}
    else:
        unblind[sid] = {"X": "without", "Y": "with"}
workdir = os.environ["WORKDIR"]
with open(f"{workdir}/unblind.json", "w") as f:
    json.dump(unblind, f, indent=2)
PY
```

(Make sure `WORKDIR` is exported as an env var before running the heredoc, e.g. `export WORKDIR`.)

## Step 8: Initialize meta.json

```bash
cat > "$WORKDIR/meta.json" <<EOF
{
  "skill_name": "$SKILL_NAME",
  "skill_sha": "$FULL_SHA",
  "skill_short_sha": "$SHORT_SHA",
  "model": "$MODEL",
  "n_tests": 3,
  "max_tokens_impl": 2048,
  "max_tokens_judge": 1024,
  "mean_threshold": 1.5,
  "correlation_id": "${{ github.run_id }}",
  "pr_number": "$PR_NUMBER",
  "execution_order": [],
  "timestamps": {"start": "$(date -u +%FT%TZ)", "end": null},
  "verdict": null
}
EOF
```

## Step 9: Write the proceed marker

After all the above succeeded, write the marker file. The dispatch job uses this to detect a successful agent run.

```bash
touch "$WORKDIR/proceed.flag"
```

## What happens next

When you exit, the post-step uploads `/tmp/skill-eval/**` as an artifact. The `dispatch` job (declared in this workflow's `jobs:` section) then dispatches sub-workflows, collects results, and posts the PR comment. You do not need to do any of those steps. Just generate good scenarios, write the marker, and exit.

## Constraints recap

- Do not invoke `claude` or `copilot` CLI directly — model access goes through gh-aw's engine.
- Do not dispatch sub-workflows. The dispatch job handles that.
- Do not run `gh aw compile`. The user compiles locally.
- Do not modify any file in the repository.
- All early-exit messages (skill not found, fork PR block, missing skill arg) go via `safe-outputs.add-comment`. Do NOT write `proceed.flag` in those cases — the dispatch job will then skip silently.
