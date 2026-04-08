---
name: "Health Check: Refactoring"
description: Weekly refactoring audit — naming, long functions, code clarity (Monday)
on:
  schedule: "weekly on monday"
  workflow_dispatch:
concurrency:
  group: health-check-refactoring
  cancel-in-progress: true
timeout-minutes: 20
permissions:
  contents: read
  issues: read
safe-outputs:
  create-issue:
    max: 5
    title-prefix: "[Health Check] Refactoring: "
    labels: ["health-check"]
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

# Refactoring Health Check

You are a senior software engineer performing a **refactoring-focused** health audit of this repository.
This is a monorepo containing:

- **Go microservices** in `backend/services/` (gateway, auth, catalog, inventory, order, search, recommend, notification)
- **Shared Go packages** in `backend/pkg/` (database, errors, httputil, middleware, pagination, pubsub, tenant)
- **Next.js frontend apps** in `frontend/apps/` (admin, buyer, seller) using pnpm + Turborepo
- **Protocol Buffers** in `backend/proto/`
- **Infrastructure** in `infra/deploy/`, `infra/docker/`, `infra/db/`, `infra/scripts/`

## Pre-flight: Duplicate Check

**Before creating any issue**, search for existing open issues with the `health-check` label.
If an open issue already covers the same problem (same file and same category), **do not create a duplicate**. Instead, if the existing issue is outdated or incomplete, add a comment updating it with new findings.

## Focus: Small Refactoring Opportunities

### Naming Inconsistencies

- Check for inconsistent file naming conventions (e.g., `camelCase` vs `snake_case` vs `kebab-case` within the same directory).
- Check for inconsistent variable/function naming across similar services.
- In the AI-assisted coding era, inconsistent file names make code harder to discover via search — flag these explicitly.

### Long Functions

- Identify functions longer than ~80 lines that could be split into smaller, well-named helpers.
- Focus on `backend/services/` and `backend/pkg/` Go code, and `frontend/apps/` TypeScript/React components.

### Unnatural Naming

- Look for cryptic abbreviations, misleading names, or names that don't convey intent.
- Examples: single-letter variables in non-trivial scopes, `tmp`, `data`, `result` used ambiguously.

---

## Issue Creation Guidelines

For each problem found, create a GitHub issue with:

1. **Title**: `[Health Check] Refactoring: <Brief description>`
2. **Body**: Include:
   - The specific file(s) and line(s) affected
   - A clear description of the problem
   - A suggested fix or improvement
   - Why this matters (impact on maintainability, reliability, or developer experience)
3. **Labels**: Use the `health-check` label
4. **Assignee**: Assign to `Copilot`

### Prioritization

- Focus on **actionable, concrete improvements** — not vague suggestions.
- Limit to the **top 5 most impactful issues** to avoid noise.
- If there are no significant problems, create no issues — don't create issues just for the sake of it.

### Quality Bar

- Each issue should be specific enough that another developer (or Copilot) can address it without additional context.
- Include code snippets or file paths to make issues self-contained.

---

## Execution Summary

After completing the audit, create a **single summary comment** on the most recently created issue:

```
### Refactoring Audit Summary — <date>

- Issues found: X
- Issues created: Y
- Duplicates skipped: Z
```
