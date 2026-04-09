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
    - cron: "0 0 * * 1"                         # Monday 00:00 UTC
    - cron: "30 9 * * 1-5"
      timezone: "Asia/Tokyo"                     # 9:30 AM JST weekdays
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
    max: 5                                       # Max issues to create
    title-prefix: "[My Prefix] "                 # Required prefix
    labels: ["my-label"]                         # Auto-applied labels
  update-issue:
    status: true
    title: true
    body:
      operation: append                          # append|prepend|replace|replace-island
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
    toolsets: [repos, issues, pull_requests]    # GitHub MCP toolsets
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
runs-on: ubuntu-latest                           # ubuntu-latest, ubuntu-24.04, ubuntu-24.04-arm
max-turns: 20                                    # AI chat iterations (Claude-specific)
checkout:
  fetch-depth: 0                                 # Or checkout: false
if: github.event_name == 'push'                  # Conditional execution
environment: production                          # Deployment protection
env:
  CUSTOM_VAR: "value"                            # WARNING: visible to AI model
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

| Workflow | Schedule | Focus |
|----------|----------|-------|
| `health-check-refactoring.md` | Weekly Monday | Naming, long functions, code clarity |
| `health-check-testing.md` | Weekly (other day) | Test hygiene, coverage, flaky tests |
| `health-check-static-analysis.md` | Weekly (other day) | Linting, deprecated APIs, dead code |
| `health-check-documentation.md` | Weekly (other day) | README, comments, rationale docs |
| `repo-health-check.md` | Manual only | Full audit across all 4 health-check areas |
| `architect-review.md` | Weekly Monday 9am JST | Large-scale architecture (8 areas) |

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
