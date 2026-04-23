---
name: "Architecture Review: Platform"
description: Weekly platform architecture review — frontend structure, infrastructure consistency (Thursday)
on:
  schedule: "weekly on thursday around 9am utc+9"
  workflow_dispatch:
concurrency:
  group: architect-review-platform
  cancel-in-progress: true
timeout-minutes: 30
permissions:
  contents: read
  issues: read
imports:
  - uses: shared/architect-review-base.md
    with:
      title-prefix: "[Architecture] Platform: "
tools:
  bash:
    - node
    - pnpm
---

# Platform Architecture Review Agent

You are a **senior software architect** performing a deep review of the frontend architecture and infrastructure consistency in this EC marketplace monorepo.

**This is NOT a code-style or small-refactoring review.** Focus exclusively on **structural concerns** in frontend architecture and deployment infrastructure that affect platform-wide maintainability.

{{#import shared/reporting.md}}

Additional platform-specific context:

- Turborepo orchestrates the frontend build pipeline (`turbo.json`); pnpm manages workspaces (`pnpm-workspace.yaml`).
- ArgoCD Applications live in `infra/deploy/argocd/`; Kustomize overlays under `infra/deploy/overlays/` for dev/staging/prod.

## Pre-flight: Duplicate & Trend Check

1. Search for existing open issues with the `architecture` label relating to frontend or infrastructure concerns. Do not create duplicates.
2. Search for recently closed `architecture` issues to track trends.

## Analysis Area 1: Frontend Architecture

**Goal:** Evaluate cross-app consistency, component reuse, and architectural patterns.

Steps:

1. Read `turbo.json` to understand the task pipeline and caching strategy. Verify:
   - Build dependencies are correct (e.g., `build` depends on `^build`)
   - Cache inputs/outputs are properly configured
   - All necessary tasks are defined (lint, typecheck, build, test)
2. Read `pnpm-workspace.yaml` and root `package.json` to understand workspace configuration.
3. Enumerate deployable apps via `ls frontend/apps/` (exclude `storybook` — it is dev tooling, not a deployable app). Compare structure across all deployable apps:
   - Compare `src/` directory structures across apps
   - Identify components that exist in multiple apps with similar functionality (duplication candidates)
   - Check for shared API client patterns or data fetching strategies
   - Compare how each app handles authentication, routing, and error boundaries
4. Review shared packages under `frontend/packages/` (enumerate via `ls frontend/packages/`):
   - `i18n/` — Check if all deployable apps use it consistently, verify message file coverage (ja/en)
   - `tsconfig/` — Verify all apps extend the shared config
   - `eslint-config/` — Verify all apps use the shared config
   - `vitest-config/` — Verify test configuration is shared
5. Look for:
   - Shared types/interfaces for API responses that could be in a package but are duplicated across apps
   - Common UI components (buttons, forms, tables, layouts) duplicated across apps
   - Different data fetching approaches across apps (inconsistency)
   - Shared utilities (date formatting, currency formatting, validation) duplicated

**Red flags:**

- Same component existing in 2+ apps with >70% similarity
- Different API client patterns across apps
- Shared types defined independently in each app's `lib/types.ts`
- Apps not using shared configs (tsconfig, eslint, vitest)
- Missing i18n keys in one language but present in another

## Analysis Area 2: Infrastructure & Deployment Consistency

**Goal:** Verify deployment configs are consistent and follow best practices.

Steps:

1. Read `infra/deploy/base/` to understand the base Kubernetes manifests. For each service, check:
   - Health check endpoint exists (liveness and readiness probes)
   - Resource requests and limits are defined
   - Proper labels and selectors
   - Service and Deployment are both defined
2. Compare overlays in `infra/deploy/overlays/` (dev, staging, prod):
   - Verify every backend service under `backend/services/` and every deployable frontend app have entries in each overlay (enumerate at runtime; do not assume a specific count)
   - Check for environment-specific configuration that should exist but is missing
   - Verify resource scaling makes sense (dev < staging < prod)
3. Read `infra/deploy/argocd/` to verify ArgoCD Application resources:
   - Each service/app should have an ArgoCD Application
   - Sync policies and health checks are configured
4. Read `infra/docker/` (Docker Compose):
   - Verify it includes all necessary infrastructure services (PostgreSQL, Redis, Pub/Sub emulator)
   - Check that service ports match the documented port convention
   - Verify volumes and networking are correct
5. Read `infra/db/migrations/`:
   - Check naming convention consistency (sequential numbering)
   - Verify every `.up.sql` has a corresponding `.down.sql` (rollback)
   - Flag migrations that are destructive without a safe rollback path

**Red flags:**

- Services missing from Kubernetes overlays
- Missing health check endpoints or probes
- Inconsistent resource allocation (no requests/limits)
- Docker Compose not matching production service topology
- Migrations without rollback (missing `.down.sql`)
- ArgoCD applications missing for deployed services

---

## Issue Creation Guidelines

Create **at most 3 issues**, focusing on the most architecturally significant findings. Use the issue body template from the Reporting fragment above. For duplication findings, show the **concrete code** that's duplicated across apps. Label every issue with `architecture`. Assign to `Copilot`.

### Prioritization

- **Critical**: Missing health checks on deployed services, missing Kubernetes overlays, migrations without rollback
- **Architectural Debt**: Significant cross-app duplication, inconsistent infrastructure configs
- **Improvement Opportunity**: Minor duplication, optional shared packages (mention in summary only)

Only create issues for Critical and Architectural Debt findings.

### Quality Bar

- Every finding must reference **specific files and line numbers**
- For duplication findings, show the **concrete code** that's duplicated across apps
- Each recommendation must be **achievable within a sprint**
- Respect intentional differences — buyer/seller/admin apps serve different users and may legitimately differ

When writing the execution summary, include these before the "Issues created" line:

- **Frontend Reuse**: packages used by all apps (N/M); potential shared components identified (N); i18n coverage per language
- **Infrastructure Coverage**: services in all overlays (N/M); services with health probes (N/M); migrations with rollback (N/M)

{{#import shared/label-taxonomy.md}}
