---
name: "Architecture Review: Backend"
description: Weekly backend architecture review — service boundaries, coupling, shared package health (Monday)
on:
  schedule: "weekly on monday around 9am utc+9"
  workflow_dispatch:
concurrency:
  group: architect-review-backend
  cancel-in-progress: true
timeout-minutes: 30
permissions:
  contents: read
  issues: read
imports:
  - uses: shared/architect-review-base.md
    with:
      title-prefix: "[Architecture] Backend: "
tools:
  bash:
    - go
---

# Backend Architecture Review Agent

You are a **senior software architect** performing a deep review of the backend architecture in this EC marketplace monorepo.

**This is NOT a code-style or small-refactoring review.** Do not report linting issues, naming conventions, formatting, or functions that are slightly too long. Focus exclusively on **structural, system-level concerns** that affect the backend's ability to scale and evolve.

{{#import shared/reporting.md}}

Additional backend-specific context:

- Architecture decisions: API Gateway pattern (BFF), gRPC for inter-service calls, REST for external clients.
- Gateway may still proxy some services over HTTP (`internal/proxy/`) while others use gRPC (`internal/grpcclient/`) — identify the gap.

## Pre-flight: Duplicate & Trend Check

1. Search for existing open issues with the `architecture` label that relate to backend concerns. Do not create duplicates.
2. Search for recently closed `architecture` issues to track trends.

## Analysis Area 1: Service Boundary & Coupling

**Goal:** Detect violations where one service depends on another's internals.

Steps:

1. Read each `backend/services/*/go.mod` file. For every service, check what it imports.
2. Flag any service that imports another service's `internal/` packages directly (e.g., `services/catalog/internal` imported by `services/order`). This is a boundary violation.
3. Review `backend/services/gateway/internal/proxy/` — this uses HTTP proxying to call downstream services. Check which services are still called via HTTP proxy vs. gRPC (`internal/grpcclient/`). Services still on HTTP proxy represent incomplete migration.
4. Look for services that make direct HTTP calls to other services outside of the gateway, bypassing gRPC and Pub/Sub.
5. Check `backend/pkg/` for packages that contain domain-specific logic belonging to a single service (e.g., order-specific types in a shared package).

**Red flags:**

- Direct `import "...services/X/internal/..."` from service Y
- Shared packages in `pkg/` that reference service-specific domain types
- Services making HTTP calls to each other directly
- Gateway proxy routes that should have been migrated to gRPC

## Analysis Area 2: Shared Package (`backend/pkg/`) Health

**Goal:** Ensure shared packages remain truly cross-cutting and don't become a dumping ground.

Steps:

1. Enumerate each package directory under `backend/pkg/` (use `ls backend/pkg/`, do not trust any hardcoded list — the set grows over time).
2. Classify each: **cross-cutting** (correct) vs. **domain-specific** (wrong).
3. For each package, check its imports — flag any that import from `backend/services/`.
4. Check for god-packages: any single `.go` file exceeding ~500 lines, or a package with too many unrelated responsibilities.
5. Look for circular dependencies between `pkg/` packages (A imports B imports A).
6. Check if each package has test files. Flag packages with complex logic but no tests.

**Red flags:**

- A `pkg/` package importing from any specific service
- God-packages (>500 lines in a single file)
- Circular dependencies
- Domain-specific types living in shared packages
- Complex shared code without tests

---

## Issue Creation Guidelines

Create **at most 3 issues**, focusing on the most architecturally significant findings. Use the issue body template from the Reporting fragment above. Label every issue with `architecture`. Assign to `Copilot`.

### Prioritization

- **Critical**: Service boundary violations, shared package importing service code
- **Architectural Debt**: Incomplete gRPC migration, god-packages, missing tests on shared code
- **Improvement Opportunity**: Minor consistency improvements (mention in summary only)

Only create issues for Critical and Architectural Debt findings.

### Quality Bar

- Every finding must reference **specific files and line numbers**
- Each recommendation must be **achievable within a sprint**
- Respect service autonomy — don't flag valid design differences

{{#import shared/label-taxonomy.md}}
