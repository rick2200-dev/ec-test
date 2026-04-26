---
name: gh-aw
description: |
  Create, edit, and manage GitHub Agentic Workflows for this monorepo.
  Use this skill when the user wants to: create a new agentic workflow, add a scheduled AI agent task,
  automate repository tasks with AI, write a .md workflow file for gh aw, or modify existing agentic workflows.
  Also trigger when the user mentions "gh aw", "agentic workflow", "agentic workflows",
  "ワークフロー作成", "エージェントワークフロー", or "定期実行エージェント".
---

# GitHub Agentic Workflows Skill

This skill captures the end-to-end process for creating GitHub Agentic Workflows in this monorepo.
Agentic Workflows are AI-powered GitHub Actions written in Markdown and compiled to `.lock.yml` via `gh aw compile`.

## How It Works

1. Write a `.md` file in `.github/workflows/` with YAML frontmatter + Markdown instructions
2. Run `gh aw compile` to generate a `.lock.yml` file (GitHub Actions YAML)
3. Commit both files. GitHub Actions runs the `.lock.yml`, which loads the Markdown body at runtime

**Important:** The Markdown body can be edited on GitHub.com without recompilation. Only frontmatter changes require `gh aw compile`.

## File Structure

```
.github/workflows/
  my-workflow.md          # Source (you write this)
  my-workflow.lock.yml    # Compiled (gh aw compile generates this)
```

## Workflow Template

```markdown
---
name: "Workflow Display Name"
description: One-line description of what this workflow does
on:
  schedule: "weekly on monday around 9am utc+9"
  workflow_dispatch:
concurrency:
  group: my-workflow
  cancel-in-progress: true
timeout-minutes: 20
permissions:
  contents: read
  issues: read
safe-outputs:
  create-issue:
    max: 5
    title-prefix: "[My Prefix] "
    labels: ["my-label"]
  add-labels:
    max: 5
  add-comment:
    max: 3
tools:
  github:
    toolsets: [repos, issues]
  bash:
    - grep
    - find
    - wc
    - cat
    - head
    - tail
    - sort
    - uniq
    - go
    - node
---

# Agent Instructions Title

Markdown body with natural language instructions for the AI agent.
```

## Frontmatter Reference

### Triggers (`on:`)

Standard GitHub Actions triggers with additional agentic controls:

```yaml
# Common triggers
on:
  issues:
    types: [opened]
  pull_request:
    branches: [main]
  schedule:
    - cron: '0 9 * * MON'
  workflow_dispatch:
  push:
    branches: [main]

# Agentic-specific controls
on:
  roles: [admin, maintainer, write]        # Who can trigger (default)
  bots: ["dependabot[bot]", "renovate[bot]"]
  skip-if-match: "is:open label:skip"      # Skip execution condition
  manual-approval: true                     # Require environment protection
  reaction: "rocket"                        # Emoji reaction on trigger
  status-comment: true                      # Post status comment
  stop-after: "2026-12-31"                 # Auto-disable date
```

### Schedule Syntax

**Fuzzy schedules (recommended)** — automatically distribute execution times to prevent load spikes:

```yaml
# Daily
on:
  schedule: daily
  schedule: daily on weekdays
  schedule: daily around 14:00
  schedule: daily around 9am utc+9              # 9 AM JST
  schedule: daily between 9:00 and 17:00 on weekdays

# Hourly (intervals: 1h, 2h, 3h, 4h, 6h, 8h, 12h)
on:
  schedule: hourly
  schedule: every 2h
  schedule: every 6h on weekdays

# Weekly
on:
  schedule: weekly
  schedule: weekly on monday
  schedule: weekly on friday around 5pm
  schedule: weekly on monday around 9am utc+9   # Monday 9 AM JST

# Multi-week
on:
  schedule: bi-weekly
  schedule: tri-weekly
```

**Fixed schedules** — standard cron with optional timezone:

```yaml
on:
  schedule:
    - cron: "0 0 * * 1" # Monday 00:00 UTC
    - cron: "30 9 * * 1-5"
      timezone: "Asia/Tokyo" # 9:30 AM JST weekdays
```

**Shorthand** — `on: daily` auto-expands to include `workflow_dispatch`.

**UTC offsets for JST:** Use `utc+9` in fuzzy schedules. For cron, use `timezone: "Asia/Tokyo"`.

### Permissions & Safe Outputs

Write permissions are **not granted directly**. Use `safe-outputs` instead:

```yaml
permissions:
  contents: read
  issues: read
  pull-requests: read

safe-outputs:
  create-issue:
    max: 5 # Max issues to create
    title-prefix: "[My Prefix] " # Required prefix
    labels: ["my-label"] # Auto-applied labels
  update-issue: # Enable updating existing issues
    status: # Enable status changes (open/closed) — value MUST be empty (null)
    title: # Enable title changes — value MUST be empty (null)
    body: # Enable body changes — value MUST be empty (null)
    # allowed-repos: ["owner/repo"]              # Optional: restrict which repos can be updated
    # NOTE: update-issue does NOT accept `max:` — the schema rejects it.
    # NOTE: each sub-field (status/title/body) is a toggle — presence enables it, value must be null.
  add-labels:
    max: 5
  add-comment:
    max: 3
  status-comment: true
  report-failure-as-issue: false
```

### AI Engine

Default engine is Copilot. To use Claude:

```yaml
engine:
  id: claude
  # model: claude-sonnet-4-6                    # Optional model override
  # env:
  #   ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
```

Required secret for Claude: `ANTHROPIC_API_KEY`

### Runtimes

```yaml
runtimes:
  go:
    version: "1.25"
  node:
    version: "22"
  python:
    version: "3.12"
  # Also: uv, bun, deno, ruby, java, dotnet, elixir, haskell
```

### Tools

```yaml
tools:
  github:
    toolsets: [repos, issues, pull_requests] # GitHub MCP toolsets
  bash:
    - grep
    - find
    - wc
    - cat
    - head
    - tail
    - sort
    - uniq
    - go
    - node
    - pnpm
```

### Imports (shared fragments)

Workflows can share frontmatter and/or body across callers via the `imports:` feature. Live in `.github/workflows/shared/`.

**Two syntaxes — combinable in the same caller:**

```yaml
# Frontmatter — imports another file's frontmatter (merged by the compiler) and can pass parameters
imports:
  - uses: shared/architect-review-base.md
    with:
      title-prefix: "[Architecture] Backend: "
```

```markdown
<!-- Body — inlines the shared file's body text at this location -->

{{#import shared/reporting.md}}
```

**Merge semantics (critical):**

| Field              | Merge behavior                                                        |
| ------------------ | --------------------------------------------------------------------- |
| `permissions:`     | Validation only — NOT merged. Main caller must declare them itself.   |
| `safe-outputs:`    | Each type defined once. Main caller OVERRIDES imports.                |
| `tools.bash`       | Deep-merge; lists concatenate and dedupe.                             |
| `tools.github`     | Deep-merge; toolset lists dedupe.                                     |
| `env:`             | Main takes precedence. Duplicate keys across imports fail compilation. |

**Consequence for callers:** if you want the base's `safe-outputs` to apply, do NOT redeclare `safe-outputs` in the caller. Add only extra `tools.bash` entries (e.g. `go`, `node`) — the common ones in the base merge in.

**Single-import constraint:** a file can appear at most once in an import graph. Identical imports dedupe silently; importing the same file with different `with:` values fails at compile time.

### Import schema (parameterized shared fragments)

A shared fragment declares its parameter contract via `import-schema:`. The compiler validates caller inputs and substitutes values throughout the fragment's frontmatter AND body before processing.

```yaml
---
# shared/my-base.md — has no `on:` trigger; only callable via imports
import-schema:
  title-prefix:
    type: string
    required: true
    description: "Issue title prefix"
  max-issues:
    type: number
    default: 5
  mode:
    type: choice
    options: [strict, lenient]
    required: true

safe-outputs:
  create-issue:
    max: ${{ github.aw.import-inputs.max-issues }}
    title-prefix: "${{ github.aw.import-inputs.title-prefix }}"
---

Run in ${{ github.aw.import-inputs.mode }} mode.
```

**Supported types:** `string`, `number`, `boolean`, `choice` (with `options:`), `array` (with `items.type:`), `object` (with nested `properties:`).

**Referencing inputs:** `${{ github.aw.import-inputs.<key> }}` in frontmatter values (gets substituted before YAML parse) and in body text. Dotted paths for nested object fields.

**Callers supply values via `with:`:**

```yaml
imports:
  - uses: shared/my-base.md
    with:
      title-prefix: "[Foo] "
      max-issues: 10
      mode: strict
```

### Existing shared fragments in this repo

| File                                 | Shape             | Purpose                                                          |
| ------------------------------------ | ----------------- | ---------------------------------------------------------------- |
| `shared/architect-review-base.md`    | Frontmatter-only  | Chassis for the four `architect-review-*` weekly workflows       |
| `shared/health-check-base.md`        | Frontmatter-only  | Chassis for the four `health-check-*` weekly workflows           |
| `shared/reporting.md`                | Body-only         | Repo-context block (service list), issue body template, summary  |
| `shared/label-taxonomy.md`           | Body-only         | Canonical list of repo labels and which workflows set which      |

`reporting.md` is the single source of truth for the monorepo's service and app list — update it when services are added/removed and every workflow picks up the change on its next compile.

### Execution Hooks

```yaml
# Pre-agent setup steps
steps:
  - name: Install dependencies
    run: npm ci

# Post-agent cleanup
post-steps:
  - name: Upload artifacts
    uses: actions/upload-artifact@v4

# Preparatory jobs
jobs:
  setup:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
```

### Other Fields

```yaml
run-name: "Custom run name"
runs-on: ubuntu-latest # ubuntu-latest, ubuntu-24.04, ubuntu-24.04-arm
# NOTE: `max-turns:` is NOT a valid top-level frontmatter key; compile fails.
# It is Claude-engine-specific and must be nested under `engine:` if used at all.
# For `engine: copilot` it is unsupported — omit it and rely on `timeout-minutes:` to bound runtime.
checkout:
  fetch-depth: 0 # Or checkout: false
if: github.event_name == 'push' # Conditional execution
environment: production # Deployment protection
env:
  CUSTOM_VAR: "value" # WARNING: visible to AI model
secrets:
  API_TOKEN: ${{ secrets.API_TOKEN }}
```

## Workflow: Creating a New Agentic Workflow

### Step 1: Define the Purpose

Determine:

- **What** the agent should analyze or do
- **When** it should run (schedule, trigger event, manual)
- **What outputs** it needs (create issues, comments, PRs)
- **What tools** it needs (bash commands, GitHub API)

### Step 2: Write the Frontmatter

Follow this checklist:

- [ ] `name` and `description` are clear and concise
- [ ] `on` trigger is appropriate (prefer fuzzy schedules for scheduled tasks)
- [ ] `concurrency` group prevents parallel runs
- [ ] `timeout-minutes` is set (default 20, max 60)
- [ ] `permissions` are read-only (use safe-outputs for writes)
- [ ] `safe-outputs` are configured for any write operations
- [ ] `tools` include all necessary bash commands and GitHub toolsets

### Step 3: Write the Markdown Body

Structure the instructions clearly:

1. **Role declaration** — Tell the agent who it is (e.g., "You are a senior software architect")
2. **Scope boundaries** — What to analyze AND what NOT to analyze
3. **Repository context** — Brief description of the monorepo structure
4. **Pre-flight checks** — Duplicate detection, trend tracking
5. **Analysis instructions** — Detailed steps for each analysis area
6. **Output format** — Exact format for issues, comments, or other outputs
7. **Quality bar** — Criteria for what counts as a valid finding
8. **Execution summary** — How to summarize results

Best practices for the body:

- Be specific about file paths and patterns to check
- Define red flags explicitly
- Set limits (e.g., "create at most 5 issues")
- Include example output format
- Tell the agent what NOT to do (prevent scope creep)

### Step 4: Validate Against Existing Workflows

Check `.github/workflows/*.md` for overlapping concerns. This monorepo already has:

| Workflow                          | Schedule                 | Focus                                      |
| --------------------------------- | ------------------------ | ------------------------------------------ |
| `health-check-refactoring.md`     | Weekly Monday            | Naming, long functions, code clarity       |
| `health-check-testing.md`         | Weekly Wednesday         | Test hygiene, coverage, flaky tests        |
| `health-check-static-analysis.md` | Weekly Friday            | Linting, deprecated APIs, dead code        |
| `health-check-documentation.md`   | Weekly Sunday            | README, comments, rationale docs           |
| `repo-health-check.md`            | Manual only              | Full audit across all 4 health-check areas |
| `architect-review.md`             | Manual only              | Full architecture review (all 8 areas)     |
| `architect-review-backend.md`     | Weekly Monday 9am JST    | Service boundaries, shared packages        |
| `architect-review-api.md`         | Weekly Tuesday 9am JST   | API/Proto consistency, event-driven        |
| `architect-review-security.md`    | Weekly Wednesday 9am JST | Multi-tenant isolation, dependencies       |
| `architect-review-platform.md`    | Weekly Thursday 9am JST  | Frontend architecture, infrastructure      |
| `architect-review-longevity.md`   | Weekly Friday 9am JST    | DB, scalability, extensibility, ops (Discussion) |
| `issue-triage.md`                 | Weekly Friday 10am JST   | Open-issue classification, close/label/defer |
| `daily-architecture-diagram.md`   | Daily 8am JST            | Mermaid architecture diagram; no-op on unchanged days |
| `skill-eval.md`                   | On-demand (`/skill-eval <name>` PR comment + workflow_dispatch) | Orchestrator: A/B validate `.claude/skills/<name>/SKILL.md` via blind pairwise judging — fans out to `skill-eval-arm.md` |
| `skill-eval-arm.md`               | Triggered only by `skill-eval` orchestrator | Single-shot worker: runs one implementer arm OR the combined pairwise judge for 3 scenarios |

When adding a new skill under `.claude/skills/<name>/`, optionally update `inputs.skill.options:` in `skill-eval.md` to expose it in the dispatch dropdown. This is best-effort — `(custom)` + `custom_skill_name` and the slash-command's free-form argument both let you evaluate a skill without editing the workflow.

Ensure the new workflow doesn't duplicate existing concerns.

### Step 5: Hand Off to User for Compilation

After creating the `.md` file, tell the user to run:

```bash
gh aw compile <workflow-name>
```

**Do NOT run `gh aw compile` yourself.** The user handles compilation.

### Step 6: Commit Both Files

Both the `.md` source and the generated `.lock.yml` must be committed:

```bash
git add .github/workflows/<name>.md .github/workflows/<name>.lock.yml
git commit -m "feat: add <name> agentic workflow"
```

## CLI Commands Reference

```bash
gh aw compile                    # Compile all workflows
gh aw compile my-workflow        # Compile specific workflow
gh aw compile --watch            # Auto-recompile on changes
gh aw compile --validate         # Schema validation
gh aw compile --strict           # Enforce security best practices
gh aw compile --purge            # Remove orphaned .lock.yml files
gh aw run my-workflow            # Trigger workflow run
gh aw list                       # List all workflows
gh aw status                     # Detailed workflow status
gh aw validate                   # Validate with all linters
gh aw add-wizard                 # Interactive workflow creation
gh aw logs                       # Download and analyze logs
gh aw audit                      # Analyze workflow runs
gh aw health                     # Display health metrics
```

## Constraints

- Minimum schedule interval: 5 minutes (fixed), 1 hour (fuzzy recommended)
- Runners: `ubuntu-latest`, `ubuntu-24.04`, `ubuntu-24.04-arm` only (no macOS/Windows)
- Write permissions require `safe-outputs`, not direct `permissions` grants
- Secrets in `env:` are visible to the AI model — use `secrets:` field instead
- Network: strict mode blocks wildcard domains
- `.lock.yml` files are auto-generated — never edit them manually

## Triggers and permissions — gotchas

### Slash commands (PR comment-driven)

`on.slash_command:` accepts either a **string shorthand** or an **object** form. Both are officially supported (do NOT mistake `on.command:` — that one is wrong).

**String shorthand** (works when you only need the command name):

```yaml
on:
  slash_command: "skill-eval"
```

**Object form** (when you need extra settings like restricting to PR-comment-only):

```yaml
on:
  slash_command:
    name: skill-eval
    events: [pull_request_comment]    # restrict to PR comments only (skip plain issue comments)
  roles: [admin, maintainer, write]   # who can invoke
  reaction: "eyes"
  status-comment: true
```

To read the triggering comment's body in your workflow, use the auto-injected sanitized text rather than the raw GitHub event payload — gh-aw runs basic injection-prevention sanitization:

```bash
COMMENT_TEXT="${{ steps.sanitized.outputs.text }}"
SKILL_NAME=$(echo "$COMMENT_TEXT" | grep -oE '/skill-eval[[:space:]]+[^[:space:]]+' | head -1 | awk '{print $2}')
```

The local SKILL above predates this trigger; refer to upstream `gh-aw` docs (`reference/command-triggers/`) for current syntax. `gh aw compile --validate` will catch wrong keys.

### GitHub Models (`gh models run` / `gh models eval`)

- There is **no `models` permission scope** in `permissions:`. Writing `permissions: { models: read }` will fail compilation.
- GitHub Models access is gated at the **repo / org admin level** — Settings → Models → enable. If it's disabled, every `gh models run` call in your bash steps will fail.
- `gh models run` does not have a verified `--json` structured-output flag. Treat its stdout as free-form text and parse defensively.
- `gh models eval` evaluators score each `testData` row independently — there is no native pairwise (X vs Y) compare. For blind A/B, build the pairwise prompt yourself with `gh models run`.

### Copilot engine model selection

Per the official `engines/` and `environment-variables/` reference, two layers control the model. Frontmatter takes precedence:

1. **Frontmatter `engine.model:` (highest priority, per-workflow):**

   ```yaml
   engine:
     id: copilot
     model: claude-sonnet-4.6   # Copilot identifier uses dots (4.6, not 4-6)
   ```

2. **Repo variable `GH_AW_MODEL_AGENT_COPILOT` (applies when frontmatter omits `engine.model:`):**

   ```
   Settings → Variables → New repository variable → Name: GH_AW_MODEL_AGENT_COPILOT → Value: claude-sonnet-4.6
   ```

3. **Neither set:** the compiler emits `COPILOT_MODEL: ${{ vars.GH_AW_MODEL_AGENT_COPILOT || '' }}` — an **empty string** at runtime — and the Copilot agent picks its own current default (Claude Sonnet 4.5 at time of writing). NOT `claude-sonnet-4.6` despite earlier docs implying otherwise; verify in your compiled `.lock.yml`.

At runtime gh-aw exports `$COPILOT_MODEL` for diagnostic bash inspection.

**Identifier format gotcha**: Copilot CLI / agent rejects hyphen-separated forms (`claude-sonnet-4-6`) with `Model not available`. Use the dot-separated form (`claude-sonnet-4.6`). The Claude engine (Anthropic API) uses the opposite — see line 191 above for hyphen-separated `engine: id: claude` example.

**Tier gating**: `claude-sonnet-4.6` may show `not available` even on Copilot Pro+. See https://github.com/orgs/community/discussions/192198. Fall back to `claude-sonnet-4.5` if so.

**Symmetric multi-workflow setups** (e.g. orchestrator + sub-workflows that must use the same model for valid A/B): set `engine.model:` to the same value in EACH frontmatter — explicit, immune to a missing repo variable, easier to audit.

### Fork PRs

- `roles:` filters who can *trigger* the workflow but does not control whether *fork-contributed code* gets evaluated. If your workflow checks out PR head SHA and feeds the diff to an LLM, fork PRs may exfiltrate fork code into a third-party model.
- For workflows that read PR content, defensively check `gh pr view "$PR_NUMBER" --json isCrossRepository` and exit early if `true`. Apply the same guard to BOTH slash-command and `workflow_dispatch + pr_number` paths so dispatch isn't a bypass.
