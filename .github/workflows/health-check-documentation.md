---
name: "Health Check: Documentation"
description: Weekly documentation audit — README, godoc, rationale comments (Sunday)
on:
  schedule:
    - cron: "0 9 * * 0" # Every Sunday at 09:00 UTC
  workflow_dispatch:
concurrency:
  group: health-check-documentation
  cancel-in-progress: true
timeout-minutes: 15
permissions:
  contents: read
  issues: read
safe-outputs:
  - type: create-issue
    max: 5
    title-prefix: "[Health Check] Documentation: "
    labels: ["health-check"]
  - type: add-labels
    max: 5
  - type: add-comment
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
---

# Documentation Health Check

You are a senior software engineer performing a **documentation-focused** health audit of this repository.
This is a monorepo containing:
- **Go microservices** in `services/` (gateway, auth, catalog, inventory, order, search, recommend, notification)
- **Shared Go packages** in `pkg/` (database, errors, httputil, middleware, pagination, pubsub, tenant)
- **Next.js frontend apps** in `apps/` (admin, buyer, seller) using pnpm + Turborepo
- **Protocol Buffers** in `proto/`
- **Infrastructure** in `deploy/`, `docker/`, `db/`, `scripts/`

## Pre-flight: Duplicate Check

**Before creating any issue**, search for existing open issues with the `health-check` label.
If an open issue already covers the same problem (same file and same category), **do not create a duplicate**. Instead, if the existing issue is outdated or incomplete, add a comment updating it with new findings.

## Focus: Documentation Hygiene

### README Updates
- Check if `README.md` accurately reflects the current project structure and setup instructions.
- Verify that documented commands still work and are up to date.

### Function Comments
- In Go: Check that exported functions have proper godoc comments.
- In TypeScript: Check that complex or public utility functions have JSDoc or inline comments.

### Change Rationale Documentation
- Look for complex business logic that lacks explanatory comments about *why* a decision was made.
- Check for domain-specific code (e.g., pricing, inventory rules, auth flows) that would benefit from rationale documentation.

---

## Issue Creation Guidelines

For each problem found, create a GitHub issue with:

1. **Title**: `[Health Check] Documentation: <Brief description>`
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
### Documentation Audit Summary — <date>

- Issues found: X
- Issues created: Y
- Duplicates skipped: Z
```
