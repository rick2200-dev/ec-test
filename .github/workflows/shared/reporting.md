---
description: Shared repository context, issue body template, and execution summary scaffolding for architect-review and health-check workflows.
---

## Repository Context

This is a multi-tenant marketplace EC platform:

- **15 Go microservices** in `backend/services/` — `auth`, `cart`, `catalog`, `coupon`, `gateway`, `inquiry`, `inventory`, `loyalty`, `notification`, `order`, `recommend`, `review`, `search`, `shipping`, `subscription`. The gateway is the single BFF ingress; all other services sit behind it.
- **Shared Go packages** in `backend/pkg/` — `authz`, `database`, `errors`, `httputil`, `middleware`, `pagination`, `pubsub`, `redis`, `tenant` (and may grow over time; always enumerate dynamically via `ls backend/pkg/` rather than trusting this list).
- **3 Next.js apps** in `frontend/apps/` — `admin`, `buyer`, `seller`. A fourth directory `storybook` exists for shared UI development and is NOT a deployable app.
- **Shared frontend packages** in `frontend/packages/` — eslint-config, tsconfig, vitest-config, i18n (enumerate via `ls frontend/packages/`).
- **Protocol Buffers** in `backend/proto/` — per-service definitions, shared types in `common/v1/`. Generated stubs in `backend/gen/go/`.
- **Gateway gRPC clients** in `backend/services/gateway/internal/grpcclient/`; gateway HTTP handlers in `backend/services/gateway/internal/handler/`.
- **Cloud Pub/Sub** for async events — publishers and subscribers distributed across services.
- **PostgreSQL with RLS** (Row-Level Security) for multi-tenant data isolation. Tenant ID is resolved at the gateway, propagated via context, enforced at the DB level via RLS policies.
- **Kubernetes (Kustomize + ArgoCD)** deployment in `infra/deploy/` with dev/staging/prod overlays; Docker Compose in `infra/docker/` for local development; DB migrations in `infra/db/migrations/`.
- **Go Workspaces** (`go.work`) manage multiple modules; pnpm + Turborepo manage the frontend monorepo.

> ⚠ Service and package counts change. Enumerate directories at runtime (`ls backend/services/`, `ls backend/pkg/`, `ls frontend/apps/`) rather than relying on numbers in this block. Flag mismatches.

## Issue Body Template

When creating an issue, use this body structure:

```markdown
## Summary

{1–2 sentence description of the concern}

## Findings

{Specific files, line numbers, and evidence — include code snippets where helpful}

## Impact

{Why this matters — what breaks or degrades if left unaddressed}

## Recommendation

{Concrete, actionable steps achievable within a sprint}

## Trend

{New issue, recurring, or improving? Reference prior review issues if found.}
```

## Execution Summary Scaffolding

After completing analysis, create a summary comment on the most recently created issue. Use a table with one row per analysis area, with columns `Area | Status (OK/WARN/CRITICAL) | Findings`. Close with:

- **Issues created**: N new, M duplicates skipped
- **Trend**: direction since last review (improving / steady / degrading)
- **Improvement Opportunities** (no issue created): bullet list of non-critical observations

If no issues were found, still create a summary comment on the most recent open issue carrying the relevant label.
