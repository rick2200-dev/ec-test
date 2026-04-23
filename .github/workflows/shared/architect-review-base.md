---
# Shared chassis for weekly architect-review-{backend,api,platform,security} workflows.
#
# Usage in caller:
#   imports:
#     - uses: shared/architect-review-base.md
#       with:
#         title-prefix: "[Architecture] Backend: "
#
# Per gh-aw merge semantics: the caller keeps its own `permissions:` block (validation-only
# merge), must NOT declare `safe-outputs:` if it wants this base's version, and may add
# extra bash tools (deep-merged + deduped with the list below).
import-schema:
  title-prefix:
    type: string
    required: true
    description: "Issue title prefix, e.g. '[Architecture] Backend: '"

permissions:
  contents: read
  issues: read

safe-outputs:
  create-issue:
    max: 3
    title-prefix: "${{ github.aw.import-inputs.title-prefix }}"
    labels: ["architecture"]
  add-labels:
    max: 3
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
