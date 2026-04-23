---
# Shared chassis for weekly health-check-{refactoring,testing,static-analysis,documentation} workflows.
#
# Usage in caller:
#   imports:
#     - uses: shared/health-check-base.md
#       with:
#         title-prefix: "[Health Check] Refactoring: "
#
# Per gh-aw merge semantics: the caller keeps its own `permissions:` block (validation-only),
# must NOT declare `safe-outputs:` if it wants this base's version, and may add extra bash
# tools (deep-merged + deduped with the list below).
import-schema:
  title-prefix:
    type: string
    required: true
    description: "Issue title prefix, e.g. '[Health Check] Refactoring: '"
  max-issues:
    type: number
    default: 5
    description: "Max issues and labels per run"

permissions:
  contents: read
  issues: read

safe-outputs:
  create-issue:
    max: ${{ github.aw.import-inputs.max-issues }}
    title-prefix: "${{ github.aw.import-inputs.title-prefix }}"
    labels: ["health-check"]
  add-labels:
    max: ${{ github.aw.import-inputs.max-issues }}
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
---
