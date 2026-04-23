---
name: "Health Check: Test Hygiene"
description: Weekly test hygiene audit — coverage gaps, flaky patterns, naming (Wednesday)
on:
  schedule: "weekly on wednesday"
  workflow_dispatch:
concurrency:
  group: health-check-testing
  cancel-in-progress: true
timeout-minutes: 20
permissions:
  contents: read
  issues: read
imports:
  - uses: shared/health-check-base.md
    with:
      title-prefix: "[Health Check] Test Hygiene: "
tools:
  bash:
    - go
    - node
    - pnpm
---

# Test Hygiene Health Check

You are a senior software engineer performing a **test-hygiene-focused** health audit of this repository.

{{#import shared/reporting.md}}

## Pre-flight: Duplicate Check

**Before creating any issue**, search for existing open issues with the `health-check` label.
If an open issue already covers the same problem (same file and same category), **do not create a duplicate**. Instead, if the existing issue is outdated or incomplete, add a comment updating it with new findings.

## Focus: Test Hygiene

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

---

## Issue Creation Guidelines

For each problem found, create a GitHub issue using the body template from the Reporting fragment above. Title each as `[Health Check] Test Hygiene: <Brief description>`. Label every issue with `health-check`. Assign to `Copilot`.

### Prioritization

- Focus on **actionable, concrete improvements** — not vague suggestions.
- Limit to the **top 5 most impactful issues** to avoid noise.
- If there are no significant problems, create no issues — don't create issues just for the sake of it.

### Quality Bar

- Each issue should be specific enough that another developer (or Copilot) can address it without additional context.
- Include code snippets or file paths to make issues self-contained.

When writing the execution summary comment, use this compact format:

```
### Test Hygiene Audit Summary — <date>

- Issues found: X
- Issues created: Y
- Duplicates skipped: Z
```

{{#import shared/label-taxonomy.md}}
