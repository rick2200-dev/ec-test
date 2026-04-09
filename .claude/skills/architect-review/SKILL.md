---
name: architect-review
description: |
  Weekly architectural review agent for the EC marketplace monorepo.
  Performs deep analysis of service boundaries, coupling, shared packages, API consistency,
  multi-tenant isolation, dependency health, and frontend architecture.
  Use this skill when the user says "architect review", "architecture review", "weekly review",
  "arch review", "アーキテクチャレビュー", "週次レビュー", or wants to assess overall system health
  at the architectural level (NOT small refactoring or code style fixes).
---

# Architect Review Skill

This skill performs a comprehensive architectural review of the EC marketplace monorepo.
It focuses on **large-scale architectural concerns** — not small refactoring, linting, or code style.

The goal is to identify structural issues that, if left unaddressed, will compound into
major maintainability, scalability, or reliability problems over time.

## Monorepo Context

```
backend/
  services/          # 8 Go microservices
    gateway/         # API Gateway (BFF) — JWT, tenant resolution, routing
    auth/            # Tenant & seller management
    catalog/         # Products, SKUs, categories
    inventory/       # Stock management
    order/           # Orders, payments, commissions (Stripe)
    search/          # Product search (Vertex AI Search)
    recommend/       # Recommendations
    notification/    # Email/push notifications, event subscriptions
  pkg/               # Shared Go packages (database, tenant, middleware, errors, httputil, pagination, pubsub)
  proto/             # Protocol Buffers definitions
  gen/               # Generated gRPC stubs

frontend/
  apps/              # 3 Next.js apps (buyer :3000, seller :3001, admin :3002)
  packages/          # Shared packages (eslint-config, tsconfig, vitest-config, i18n)

infra/
  db/                # Migrations & seeds
  deploy/            # Kubernetes (Kustomize + ArgoCD)
  docker/            # Docker Compose (local dev)
```

Key design decisions:
- Multi-tenant isolation via PostgreSQL RLS (Row-Level Security)
- External: REST/JSON, Internal: gRPC, Async: Cloud Pub/Sub
- Auth0 for identity, Stripe Connect for payments
- GKE + ArgoCD for deployment

## Review Workflow

Execute the following analysis areas **sequentially**. For each area, read the relevant source files,
analyze them, and collect findings. At the end, produce a structured report.

### Area 1: Service Boundary & Coupling Analysis

**Goal:** Detect violations of service boundaries where one service depends on another's internals.

Steps:
1. For each service in `backend/services/*/`, examine its `go.mod` for import dependencies.
2. Check if any service imports another service's internal packages directly (e.g., `services/catalog/internal` imported by `services/order`). This is a boundary violation.
3. Review the gateway's `internal/grpcclient/` and `internal/proxy/` to assess how services are called. Flag any service-to-service calls that bypass the gateway without using gRPC or Pub/Sub.
4. Check `backend/pkg/` for packages that contain domain-specific logic that should live in a single service instead.

**Red flags:**
- Direct `import "...services/X/internal/..."` from service Y
- Shared packages in `pkg/` that reference service-specific domain types
- Services making HTTP calls to each other directly instead of using gRPC or Pub/Sub

### Area 2: Shared Package (`backend/pkg/`) Health

**Goal:** Ensure shared packages remain truly cross-cutting and don't become a dumping ground.

Steps:
1. Read each package under `backend/pkg/` and classify it:
   - **Cross-cutting concern** (correct): database, middleware, errors, httputil, pagination, pubsub, tenant
   - **Domain-specific** (incorrect): anything that models or operates on a specific service's domain
2. Check for circular dependencies between `pkg/` packages.
3. Assess the API surface of each package — flag overly large interfaces or god-packages.
4. Check if `pkg/` packages have appropriate tests.

**Red flags:**
- A `pkg/` package that imports from a specific service
- A single package file exceeding ~500 lines (potential god-package)
- Shared types that embed or reference service-specific entities

### Area 3: API & Proto Consistency

**Goal:** Ensure gRPC service definitions follow consistent patterns across all services.

Steps:
1. Read all `.proto` files under `backend/proto/`.
2. Verify consistent patterns:
   - Each RPC uses its own Request/Response messages (no reuse across RPCs)
   - Common types (Money, Pagination) are imported from `common/v1/`
   - Naming conventions are consistent (snake_case fields, UPPER_SNAKE_CASE enums)
3. Compare the REST routes in gateway handlers (`backend/services/gateway/internal/handler/`) with the gRPC client calls to ensure the REST-to-gRPC mapping is complete and consistent.
4. Flag any service that has HTTP handlers but no gRPC definition yet.

**Red flags:**
- Services still using HTTP proxy calls where gRPC is expected
- Inconsistent error code mapping between REST and gRPC
- Missing or incomplete proto definitions for services that should have them

### Area 4: Multi-Tenant Isolation Audit

**Goal:** Verify that tenant isolation is enforced consistently and cannot be bypassed.

Steps:
1. Check that every service that accesses PostgreSQL sets the RLS tenant context via `backend/pkg/database/` or `backend/pkg/tenant/`.
2. Verify that all DB queries go through the established patterns (no raw SQL that bypasses RLS).
3. Check migration files in `infra/db/` to ensure all tables with tenant-scoped data have RLS policies.
4. Verify that the gateway always resolves and propagates tenant context before forwarding requests.

**Red flags:**
- DB queries that don't set `app.current_tenant` before execution
- Tables missing RLS policies for tenant-scoped data
- Endpoints that skip tenant resolution middleware
- Hardcoded tenant IDs in any service code

### Area 5: Event-Driven Architecture Consistency

**Goal:** Ensure Pub/Sub usage follows consistent patterns and doesn't create hidden coupling.

Steps:
1. Identify all Pub/Sub publishers and subscribers across services.
2. Map the event flow: which service publishes what topic, and who subscribes.
3. Check for:
   - Events that carry too much data (should be thin events with IDs, not full payloads)
   - Missing dead-letter handling
   - Subscriber error handling patterns
4. Verify that event schemas are documented or defined in proto files.

**Red flags:**
- Circular event chains (A publishes -> B subscribes -> B publishes -> A subscribes)
- Fat events with full entity payloads instead of references
- Subscribers that make synchronous calls back to the publisher
- Missing or inconsistent error handling in subscribers

### Area 6: Dependency & Module Health

**Goal:** Assess Go module and Node.js package health across the monorepo.

Steps:
1. Check `go.work` to ensure all modules are listed.
2. For each `go.mod`, check for:
   - Outdated major dependencies
   - Replace directives that should be resolved
   - Inconsistent dependency versions across services
3. Check `package.json` files in frontend for:
   - Dependency version inconsistencies across apps
   - Overly large dependency trees
   - Missing or incorrect peer dependencies

**Red flags:**
- Different services using different versions of the same dependency
- Replace directives pointing to local paths that should be proper modules
- Frontend apps with divergent React/Next.js versions

### Area 7: Frontend Architecture Review

**Goal:** Evaluate frontend monorepo structure and cross-app consistency.

Steps:
1. Review `turbo.json` task pipeline for correctness and caching.
2. Check shared packages under `frontend/packages/` for reuse opportunities.
3. Assess whether the 3 apps (buyer, seller, admin) share components appropriately or have significant duplication.
4. Review the i18n setup for completeness.
5. Check for shared API client patterns or data fetching strategies.

**Red flags:**
- Duplicated components across apps that should be in a shared package
- Inconsistent data fetching patterns (mixing different approaches)
- Missing shared types for API responses
- Apps with significantly different architectural patterns for the same concerns

### Area 8: Infrastructure & Deployment Consistency

**Goal:** Verify that infrastructure configuration is consistent and follows best practices.

Steps:
1. Review Kubernetes manifests in `infra/deploy/` for consistency across services.
2. Check that all services have:
   - Health check endpoints
   - Resource limits and requests
   - Proper readiness/liveness probes
3. Review Docker Compose (`infra/docker/`) matches the production topology.
4. Check migration files for naming consistency and safe rollback patterns.

**Red flags:**
- Services missing health checks or probes
- Inconsistent resource allocation
- Missing environment overlays for a service
- Migrations without rollback (down) scripts

## Output Format

After completing all 8 analysis areas, produce a **structured report** as a GitHub Issue using `gh issue create`.

The issue should follow this format:

```markdown
## Weekly Architecture Review — {YYYY-MM-DD}

### Executive Summary
{2-3 sentences summarizing the overall health and most critical findings}

### Critical Issues (Action Required)
{Issues that pose immediate risk to data integrity, security, or system stability}

- **[Area N] Title**: Description and recommended action

### Architectural Debt
{Issues that won't break things today but will compound over time}

- **[Area N] Title**: Description, impact assessment, and suggested approach

### Improvement Opportunities
{Non-urgent opportunities to improve consistency, performance, or DX}

- **[Area N] Title**: Description and potential benefit

### Metrics Snapshot
| Metric | Value | Trend |
|--------|-------|-------|
| Total Go services | N | - |
| Services with gRPC | N/N | - |
| Proto definitions | N | - |
| Shared pkg/ packages | N | - |
| Frontend shared packages | N | - |
| DB migrations | N | - |
| Cross-service dependencies | N | up/down/stable |

### Next Review
Scheduled for: {next week's date}
```

Use the following `gh` command to create the issue:

```bash
gh issue create \
  --title "Weekly Architecture Review — {YYYY-MM-DD}" \
  --label "architecture,review" \
  --body "$(cat <<'EOF'
{report content}
EOF
)"
```

If labels don't exist yet, create them first:

```bash
gh label create architecture --color "0075ca" --description "Architecture review findings" 2>/dev/null || true
gh label create review --color "e4e669" --description "Review items" 2>/dev/null || true
```

## Important Guidelines

- **Focus on architecture, not style.** Do not report linting issues, formatting, naming conventions, or small code smells. Only report structural concerns that affect the system's ability to evolve.
- **Be specific.** Every finding must reference specific files and line numbers.
- **Prioritize actionability.** Each finding should include a concrete recommendation.
- **Track trends.** Compare with previous review issues (search for prior issues with the `architecture` label) and note whether issues are improving or recurring.
- **Respect service autonomy.** Services are intentionally independent — don't flag differences that are valid design choices for different domains.
- **Consider the team.** Recommendations should be achievable within a sprint, not multi-month rewrites.
