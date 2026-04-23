---
name: "Daily Architecture Diagram"
description: Daily Mermaid diagram of the monorepo's service and frontend architecture; no-ops on days with no structural change.
on:
  schedule: "daily around 8am utc+9"
  workflow_dispatch:
concurrency:
  group: daily-architecture-diagram
  cancel-in-progress: true
timeout-minutes: 20
permissions:
  contents: read
  issues: read
  pull-requests: read
safe-outputs:
  create-issue:
    max: 1
    title-prefix: "Architecture Diagram: "
    labels: ["architecture", "diagram"]
  update-issue:
    body:
    title:
    status:
  add-comment:
    max: 2
tools:
  cache-memory: true
  github:
    toolsets: [repos, issues]
  bash:
    - grep
    - find
    - cat
    - head
    - tail
    - ls
    - wc
    - git
---

# Daily Architecture Diagram Agent

You are a **repository cartographer**. Your job is to produce a **single, always-up-to-date architecture diagram** as a GitHub issue, describing the current service/frontend/infrastructure topology of this monorepo using a Mermaid graph. You regenerate only when the underlying structure actually changed — otherwise you no-op.

**You do NOT create one issue per run.** There is exactly **one live diagram issue** at any time; subsequent runs edit it. Close any older diagram issues and keep only the newest.

{{#import shared/reporting.md}}

---

## Step 1: Load Previous State (cache-memory)

You have access to a `cache-memory` tool that persists small state between runs.

1. Attempt to read a file named `architecture-state.json` from cache-memory. Expected shape:
   ```json
   {
     "last_commit": "<40-char SHA>",
     "last_diagram_issue_number": 1234,
     "last_rendered_at": "YYYY-MM-DDTHH:MM:SSZ"
   }
   ```
2. If the file does not exist, treat this as a first run: `last_commit = null`.

## Step 2: Detect Structural Change

Structural directories that affect the diagram:

- `backend/services/` — added/removed service
- `backend/pkg/` — added/removed shared package
- `backend/proto/` — added/removed service proto
- `frontend/apps/` — added/removed app
- `frontend/packages/` — added/removed shared package
- `infra/deploy/base/` and `infra/deploy/overlays/` — deployment topology
- `infra/docker/` — local topology
- `backend/services/gateway/internal/grpcclient/` — inter-service edges

Detection:

- If `last_commit` is null → treat as first run and always regenerate.
- Otherwise run `git log --name-only --pretty=format: <last_commit>..HEAD` (via the `git` bash tool) and collect touched paths. If **no** touched path matches any of the structural directories above, **no-op**:
  1. Post a single comment on the existing `architecture` + `diagram` issue saying `No structural change since <last_commit> ({short date}). Diagram unchanged.`
  2. Do **not** edit or re-create the issue body.
  3. Update `architecture-state.json` with `last_commit` = new HEAD SHA, leave `last_diagram_issue_number` unchanged, exit.

If there IS a structural change, continue to Step 3.

## Step 3: Enumerate Topology Dynamically

Do NOT hardcode service or app counts — enumerate them at runtime and flag if `shared/reporting.md` is out of sync.

1. `ls backend/services/` → L3 services. Split out `gateway` separately as L2.
2. `ls backend/pkg/` → list of shared Go packages (not rendered as nodes, used for the sidebar note).
3. `ls frontend/apps/` → L1 frontend apps; **exclude `storybook`** (dev tooling, not deployed).
4. `ls frontend/packages/` → shared frontend packages (sidebar note only).
5. L4 Infrastructure (static set, only render if referenced in the codebase — grep to confirm):
   - `postgres` (if any `pgx` import in backend/)
   - `redis` (if `backend/pkg/redis/` exists or `redis` appears in docker-compose)
   - `pubsub` (if `backend/pkg/pubsub/` exists)
   - `vertex-ai` (if any reference to Vertex AI Search in backend/)

## Step 4: Derive Edges

1. **Frontend → gateway**: each L1 app draws one arrow to `gateway`.
2. **Gateway → backend service** (gRPC): grep `backend/services/gateway/internal/grpcclient/` — each file typically indicates one downstream service via filename or import path. Draw one arrow per detected client.
3. **Gateway → backend service** (HTTP proxy): grep `backend/services/gateway/internal/proxy/` for the same — draw these as dashed arrows so the incomplete gRPC migration is visible at a glance.
4. **Service → Pub/Sub topic → Service** (async): grep `backend/services/*/internal/**/*.go` for Pub/Sub publish calls to identify publisher→topic mappings; grep subscriber directories for subscriber→topic mappings. Render Pub/Sub as a single hub node and draw publisher → pubsub and pubsub → subscriber edges (or skip this layer if it makes the graph unreadable).
5. **Service → infra**: connect each service to the infra nodes it actually uses.

## Step 5: Render Mermaid

Use `graph TD` (top-down), short labels, and **wrap the diagram in a `<details>` block** if the total node count exceeds 40 — some GitHub viewers balk at very large diagrams.

Template:

```markdown
# Repository Architecture — <YYYY-MM-DD>

Auto-generated by `daily-architecture-diagram` workflow. Regenerated only when structural files change. Last update: <commit short SHA> (<date>).

## Diagram

<details open>
<summary>Mermaid graph</summary>

```mermaid
graph TD
  %% Legend
  %% solid arrow = gRPC / in-process
  %% dashed arrow = HTTP proxy (incomplete gRPC migration)

  subgraph L1["Frontend apps"]
    admin[admin]
    buyer[buyer]
    seller[seller]
  end

  subgraph L2["Edge"]
    gateway([gateway])
  end

  subgraph L3["Backend services"]
    auth
    cart
    catalog
    %% … (enumerate dynamically)
  end

  subgraph L4["Infrastructure"]
    postgres[(PostgreSQL)]
    redis[(Redis)]
    pubsub{{Pub/Sub}}
  end

  admin --> gateway
  buyer --> gateway
  seller --> gateway
  gateway --> auth
  gateway --> catalog
  gateway -.-> search
  %% … (edges derived from grpcclient/ and proxy/)
  auth --> postgres
  catalog --> postgres
  catalog --> redis
```

</details>

## Notes

- **Services**: {N, listed dynamically} — compare against `shared/reporting.md` and flag drift in the comment below.
- **Deployable frontend apps**: {N, excluding `storybook`}.
- **Shared Go packages**: {list}.
- **Gateway migration progress**: {X of Y services on gRPC, the rest dashed above}.
- **Observed Pub/Sub topics**: {topic list, short}.

_This is a living document. Do not edit manually — the next daily run will overwrite changes._
```

## Step 6: Update the Diagram Issue

You have both `create-issue` and `update-issue` safe-outputs. Use them exactly in this order:

1. Search open issues with **both** `architecture` and `diagram` labels. There should be at most one (call it the "live issue").
2. If one exists → use `update-issue` to rewrite its body to the new Mermaid block. Do NOT close-and-recreate; that churns notifications. Post an `add-comment` on it naming the SHAs covered (e.g., `Regenerated from <old-sha>..<new-sha>`).
3. If none exists → use `create-issue` to create the live issue with the template above.
4. If **multiple** exist (unexpected), keep the newest and use `update-issue` with `status: closed` on the older duplicates, each with a comment pointing at the survivor. Only the duplicate-cleanup path closes anything.

## Step 7: Persist State

Write `architecture-state.json` to cache-memory with:

- `last_commit`: current HEAD SHA (from `git rev-parse HEAD`)
- `last_diagram_issue_number`: the issue number used
- `last_rendered_at`: current UTC timestamp

## Failure Mode

If any step fails (e.g., cache-memory unavailable, HEAD cannot be resolved), regenerate the diagram unconditionally rather than skipping — stale > missing, and the cadence is daily so cost is bounded.

## Quality Bar

- Do **not** hardcode the service count. Always `ls backend/services/` at runtime.
- Do **not** render `storybook` as a deployable app.
- Keep labels short (service names only, no port numbers in the diagram — move them to the notes section).
- Never close the previous diagram issue as a routine step; only the duplicate-cleanup path closes.
- If the grep for gRPC clients returns nothing, assume the gateway still proxies everything and render all edges as dashed; flag this in the notes.

{{#import shared/label-taxonomy.md}}
