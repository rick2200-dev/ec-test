---
description: Shared label taxonomy reference for architect-review, health-check, and issue-triage workflows.
---

## Label Taxonomy

Use only these existing repository labels — do not invent new ones:

| Label                        | Meaning                                                 | Written by                                                  |
| ---------------------------- | ------------------------------------------------------- | ----------------------------------------------------------- |
| `architecture`               | Structural, system-level architectural concern          | `architect-review-*`, `architect-review.md`                 |
| `health-check`               | Non-architectural code-quality finding                  | `health-check-*`, `repo-health-check.md`                    |
| `triage:ready`               | Scope is clear and bounded; safe to pick up             | `issue-triage.md`                                           |
| `triage:needs-discussion`    | Requires human judgment before work can start           | `issue-triage.md`                                           |
| `wontfix`                    | Deliberately not going to be fixed                      | humans only — do not set from a workflow                    |
| `duplicate`                  | Duplicates another issue                                | humans only — do not set from a workflow                    |
| `blocked`                    | Blocked by another open issue or external dependency    | humans only — do not set from a workflow                    |
| `diagram`                    | Architecture-diagram rendering issue (single, rolling)  | `daily-architecture-diagram.md`                             |

Rules:

- When creating an architecture issue, apply exactly `architecture`.
- When creating a health-check issue, apply exactly `health-check`.
- Never apply `wontfix`, `duplicate`, or `blocked` from a workflow — those are human-only signals.
- If an issue already carries `triage:ready` or `triage:needs-discussion`, only re-triage if it has been idle > 30 days (see `issue-triage.md`).
- Every new label applied must be justified in a comment so the decision is auditable.
