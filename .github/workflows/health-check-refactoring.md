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
imports:
  - uses: shared/health-check-base.md
    with:
      title-prefix: "[Health Check] Refactoring: "
tools:
  bash:
    - go
    - node
---

# Refactoring Health Check

You are a senior software engineer performing a **refactoring-focused** health audit of this repository.

{{#import shared/reporting.md}}

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

For each problem found, create a GitHub issue using the body template from the Reporting fragment above. Title each as `[Health Check] Refactoring: <Brief description>`. Label every issue with `health-check`. Assign to `Copilot`.

### Prioritization

- Focus on **actionable, concrete improvements** — not vague suggestions.
- Limit to the **top 5 most impactful issues** to avoid noise.
- If there are no significant problems, create no issues — don't create issues just for the sake of it.

### Quality Bar

- Each issue should be specific enough that another developer (or Copilot) can address it without additional context.
- Include code snippets or file paths to make issues self-contained.

When writing the execution summary comment, use this compact format:

```
### Refactoring Audit Summary — <date>

- Issues found: X
- Issues created: Y
- Duplicates skipped: Z
```

{{#import shared/label-taxonomy.md}}
