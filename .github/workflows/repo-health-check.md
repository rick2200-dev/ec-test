---
name: "Repository Health Check (Full)"
description: Full repository health audit across all categories (manual trigger only)
on:
  workflow_dispatch:
concurrency:
  group: repo-health-check
  cancel-in-progress: true
timeout-minutes: 30
permissions:
  contents: read
  issues: read
safe-outputs:
  create-issue:
    max: 10
    title-prefix: "[Health Check] "
    labels: ["health-check"]
  add-labels:
    max: 10
  add-comment:
    max: 5
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
    - pnpm
---

# Repository Health Check Agent (Full Audit)

You are a senior software engineer performing a full health audit of this repository.
This is a monorepo containing:

- **Go microservices** in `backend/services/` (gateway, auth, catalog, inventory, order, search, recommend, notification)
- **Shared Go packages** in `backend/pkg/` (database, errors, httputil, middleware, pagination, pubsub, tenant)
- **Next.js frontend apps** in `frontend/apps/` (admin, buyer, seller) using pnpm + Turborepo
- **Protocol Buffers** in `backend/proto/`
- **Infrastructure** in `infra/deploy/`, `infra/docker/`, `infra/db/`, `infra/scripts/`

## Pre-flight: Duplicate Check

**Before creating any issue**, search for existing open issues with the `health-check` label.
If an open issue already covers the same problem (same file and same category), **do not create a duplicate**. Instead, if the existing issue is outdated or incomplete, add a comment updating it with new findings.

## Your Mission

Analyze the repository across **all four dimensions** below. For **each new, concrete problem** you find, create a **separate GitHub issue** with a clear title, detailed description, and assign it to `Copilot`.

> **Note**: For scheduled runs, each category runs on a separate day of the week. This full audit is for manual, on-demand use.

---

## 1. Small Refactoring Opportunities

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

## 2. Test Hygiene

### Missing Test Cases

- Identify exported functions or public API endpoints that lack corresponding tests.
- Pay special attention to error paths and boundary conditions.

### Edge Cases

- Review existing tests and identify missing edge cases (empty inputs, nil/null values, overflow, concurrency).

### Test Naming

- Check that test names clearly describe what is being tested and the expected behavior.
- Flag generic names like `TestFunction1` or `test("works")`.

### Fixture Organization

- Check if test fixtures/helpers are well-organized and reusable.
- Look for duplicated setup code across test files.

### Flaky Test Patterns

- Identify patterns that commonly cause flaky tests:
  - Time-dependent assertions without tolerance.
  - Tests relying on external services without mocks.
  - Race conditions in concurrent test code.
  - Order-dependent tests.

## 3. Static Analysis Assistance

### Linter Warnings

- For Go: Run `go vet` conceptually — check for common issues like unchecked errors, shadow variables, unused parameters.
- For TypeScript: Look for patterns that ESLint/TypeScript compiler would flag — `any` types, unused imports, missing return types.

### Deprecated APIs

- Look for usage of deprecated standard library functions, packages, or third-party APIs.
- Check Go module versions and npm package versions for known deprecations.

### Dead Code

- Identify unexported Go functions/types that are never referenced.
- Find unused TypeScript exports, unreachable code paths, commented-out code blocks.

## 4. Documentation Hygiene

### README Updates

- Check if `README.md` accurately reflects the current project structure and setup instructions.
- Verify that documented commands still work and are up to date.

### Function Comments

- In Go: Check that exported functions have proper godoc comments.
- In TypeScript: Check that complex or public utility functions have JSDoc or inline comments.

### Change Rationale Documentation

- Look for complex business logic that lacks explanatory comments about _why_ a decision was made.
- Check for domain-specific code (e.g., pricing, inventory rules, auth flows) that would benefit from rationale documentation.

---

## Issue Creation Guidelines

For each problem found, create a GitHub issue with:

1. **Title**: `[Health Check] <Category>: <Brief description>`
   - Categories: `Refactoring`, `Test Hygiene`, `Static Analysis`, `Documentation`
2. **Body**: Include:
   - The specific file(s) and line(s) affected
   - A clear description of the problem
   - A suggested fix or improvement
   - Why this matters (impact on maintainability, reliability, or developer experience)
3. **Labels**: Use the `health-check` label
4. **Assignee**: Assign to `Copilot`

### Prioritization

- Focus on **actionable, concrete improvements** — not vague suggestions.
- Limit to the **top 10 most impactful issues** to avoid noise.
- If there are no significant problems in a category, skip it — don't create issues just for the sake of it.

### Quality Bar

- Each issue should be specific enough that another developer (or Copilot) can address it without additional context.
- Include code snippets or file paths to make issues self-contained.

---

## Execution Summary

After completing the audit, create a **single summary comment** on the most recently created issue with the following format:

```
### Health Check Summary — <date>

| Category | Issues Found | Issues Created | Duplicates Skipped |
|---|---|---|---|
| Refactoring | X | Y | Z |
| Test Hygiene | X | Y | Z |
| Static Analysis | X | Y | Z |
| Documentation | X | Y | Z |

**Total**: N new issues created, M duplicates skipped.
```
