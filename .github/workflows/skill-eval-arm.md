---
name: "Skill Validation: Arm"
description: Single-shot worker for skill-eval. Not invoked directly — only via `gh workflow run` from the skill-eval orchestrator. Handles one implementer arm OR the combined pairwise judge for 3 scenarios.
on:
  workflow_dispatch:
    inputs:
      mode:
        description: "implementer | judge"
        required: true
        type: choice
        options:
          - implementer
          - judge
      skill_name:
        description: "Skill folder name (under .claude/skills/)"
        required: true
        type: string
      ref:
        description: "Git ref (SHA) to record in artifact (not used for checkout — checkout is disabled here)"
        required: true
        type: string
      correlation_id:
        description: "Orchestrator's run id (used in artifact name and run-name)"
        required: true
        type: string
      scenario_id:
        description: "1 | 2 | 3 (implementer mode only)"
        required: false
        type: string
      variant_label:
        description: "X | Y (implementer mode only)"
        required: false
        type: string
      system_context_b64:
        description: "Base64-encoded system context file content (implementer mode only)"
        required: false
        type: string
      user_prompt_b64:
        description: "Base64-encoded user prompt (implementer mode only)"
        required: false
        type: string
      implementer_runs_b64:
        description: 'Base64-encoded JSON of implementer run_ids (judge mode only). Format: [{"scenario":"1","X":<run_id>,"Y":<run_id>}, ...]'
        required: false
        type: string
      user_prompts_b64:
        description: 'Base64-encoded JSON of all 3 user prompts (judge mode only). Format: [{"scenario":"1","prompt":"..."}, ...]'
        required: false
        type: string
  roles: [admin, maintainer, write]
run-name: "skill-eval-arm: ${{ inputs.correlation_id }}/${{ inputs.mode == 'judge' && 'judge' || format('{0}-{1}', inputs.scenario_id, inputs.variant_label) }}"
concurrency:
  group: skill-eval-arm-${{ inputs.correlation_id }}-${{ inputs.mode == 'judge' && 'judge' || format('{0}-{1}', inputs.scenario_id, inputs.variant_label) }}
  cancel-in-progress: false
timeout-minutes: 8
permissions:
  contents: read
  actions: read
engine:
  id: copilot
checkout: false   # ISOLATION: do NOT mount the repository. The implementer/judge agents
                  # must not be able to read `.claude/skills/<other>/SKILL.md` or any
                  # other repo file — that would let the "without-skill" arm leak in
                  # competing skill content via disk reads, biasing the A/B comparison
                  # toward inconclusive. All required input is base64-injected via
                  # workflow_dispatch inputs and lives under /tmp/.
runtimes:
  python:
    version: "3.12"
tools:
  bash:
    - cat
    - mkdir
    - python3
    - base64
    - gh
    - jq
    - ls
post-steps:
  - name: Upload arm output
    if: always()
    uses: actions/upload-artifact@v4
    with:
      name: skill-eval-arm-${{ inputs.correlation_id }}-${{ inputs.mode == 'judge' && 'judge' || format('{0}-{1}', inputs.scenario_id, inputs.variant_label) }}
      path: /tmp/skill-eval-arm-output/**
      retention-days: 7
---

# Role

You are a single-purpose worker invoked by the `skill-eval` orchestrator. The work to do is determined by `${{ inputs.mode }}`.

You MUST write your final response to `/tmp/skill-eval-arm-output/response.txt` and stop. You MUST NOT write any other commentary, file, or response outside that file. The post-step uploads `/tmp/skill-eval-arm-output/**` as an artifact for the orchestrator to consume.

## ABSOLUTE ISOLATION RULES (apply to BOTH modes)

This run is part of a blind A/B skill validation. To preserve fairness:

- **All your inputs come from `/tmp/skill-eval-arm-input/`** (and, in judge mode only, downloaded artifact data under `/tmp/judge-input/`). Do NOT read files anywhere else.
- **Do NOT read any file under `.claude/`, `.github/`, or any path inside the repository working tree.** The repository is intentionally not checked out (`checkout: false`) so this is enforced by absence; do not try to recreate it via `git clone`, `gh api repos/.../contents/...`, `gh repo clone`, or any other means.
- **Do NOT use `gh api`, `gh repo`, `git`, `curl`, `wget`, or any command that fetches external content.** The only sanctioned external command is `gh run download <run_id>` in judge mode (used solely to fetch the implementer arms' outputs by run id, NOT repo content).
- **Do NOT browse, list, or search the filesystem outside `/tmp/skill-eval-arm-input/`, `/tmp/skill-eval-arm-output/`, and (judge mode only) `/tmp/judge-input/`.**
- All operating context you legitimately need (repo description, skill body if applicable) is already inlined into your inputs by the orchestrator. There is nothing relevant to read elsewhere.

Violating these rules biases the A/B comparison and undermines the entire purpose of this run.

## Mode = `implementer`

Triggered with: `mode=implementer`, `scenario_id`, `variant_label`, `system_context_b64`, `user_prompt_b64`.

### Setup

Run these bash commands first:

```bash
mkdir -p /tmp/skill-eval-arm-input /tmp/skill-eval-arm-output
echo "${{ inputs.system_context_b64 }}" | base64 -d > /tmp/skill-eval-arm-input/system_context.txt
echo "${{ inputs.user_prompt_b64 }}" | base64 -d > /tmp/skill-eval-arm-input/user.txt
```

### Operating context

Read the entire content of `/tmp/skill-eval-arm-input/system_context.txt`. That is your operating context for this single task: it describes the repository structure and may include loaded skill instructions. Treat it as background information and apply it when responding.

### User task

Read the entire content of `/tmp/skill-eval-arm-input/user.txt`. That is the user's task. Respond to it.

### Output requirements

- Write your full response **verbatim** to `/tmp/skill-eval-arm-output/response.txt`.
- The file must contain ONLY your response. No preamble like "Here is my response:". No postscript like "Let me know if you need more". No code-fence wrapping the whole response.
- Plain prose / code / markdown matching the task type is fine.
- After writing the file, stop. Do not produce any further output.

## Mode = `judge`

Triggered with: `mode=judge`, `implementer_runs_b64`, `user_prompts_b64`.

You will see THREE different tasks, each answered by two anonymous variants (X and Y). For each task **independently**, pick the better variant on the rubric below. If essentially equivalent, choose `tie`. Output ONLY a valid JSON array, no prose, no code fences, no commentary.

### Setup

```bash
mkdir -p /tmp/skill-eval-arm-input /tmp/skill-eval-arm-output /tmp/judge-input
echo "${{ inputs.implementer_runs_b64 }}" | base64 -d > /tmp/skill-eval-arm-input/runs.json
echo "${{ inputs.user_prompts_b64 }}" | base64 -d > /tmp/skill-eval-arm-input/prompts.json
python3 - <<'PY'
import json, subprocess, sys
runs = json.load(open('/tmp/skill-eval-arm-input/runs.json'))
for entry in runs:
    sid = str(entry['scenario'])
    for variant in ('X', 'Y'):
        run_id = str(entry[variant])
        target = f'/tmp/judge-input/{sid}-{variant}'
        r = subprocess.run(['gh', 'run', 'download', run_id, '--dir', target],
                           capture_output=True, text=True)
        if r.returncode != 0:
            sys.stderr.write(f'download failed for run {run_id}: {r.stderr}\n')
            sys.exit(1)
PY
```

### Rubric (apply all, independently per task)

1. **Factual accuracy** in this Go microservice monorepo (no fictional paths/names).
2. **Workflow fidelity** (does it follow plausible best-practice steps).
3. **Format / structure clarity**.
4. **Helpfulness** (would a developer accept this as-is).

### Inputs (for each scenario s in {1, 2, 3})

- **Task**: from `/tmp/skill-eval-arm-input/prompts.json`, the entry where `scenario == s`, field `prompt`.
- **Variant X**: contents of `/tmp/judge-input/<s>-X/response.txt`.
- **Variant Y**: contents of `/tmp/judge-input/<s>-Y/response.txt`.

### Output requirements

Write to `/tmp/skill-eval-arm-output/response.txt` a JSON ARRAY of 3 objects, ordered by scenario id ascending:

```
[
  {"scenario": "1", "winner": "X" | "Y" | "tie", "margin": 0 | 1 | 2, "reason_x": "...", "reason_y": "...", "rubric_notes": "..."},
  {"scenario": "2", ...},
  {"scenario": "3", ...}
]
```

`margin`: 0 = tie, 1 = slight, 2 = clear.

### ABSOLUTE RULES

- The file must start with `[` and end with `]`. No surrounding fences (no ```` ```json ... ``` ````). No leading/trailing prose. No preamble.
- Score each scenario INDEPENDENTLY. Do not let your judgment of one scenario influence another.
- Do NOT mention "skill", "with-skill", "without-skill", or any equivalent phrasing in `reason_x`, `reason_y`, or `rubric_notes`. The variants are anonymous to you.
- After writing the JSON array file, stop.
