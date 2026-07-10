---
name: specification-session
description: "Create or repair a compact spec.md that makes behavior and proof decisions explicit; review it to the risk of the change."
---

# Specification Session

Use [the specification phase](../../../docs/spec-first-workflow/phases/specification.md) when behavior, scope, invariants, constraints, or proof expectations must survive into implementation.

Write the smallest useful `spec.md`. Resolve material `TBD`s, live alternatives, source-of-truth ambiguity, hidden scope cuts, and cleanup disposition. Keep implementation mechanism out unless it is required to make the behavior decision.

Perform focused self-review. Invoke [specification review](../../../docs/spec-first-workflow/phases/specification-review.md) when independent review is required by the user or justified by impact, reversibility, ambiguity, or weak self-falsification. Repair in-scope findings and obtain any fresh re-review in the same root session; internal review never creates a next-session prompt.

Success means design/planning can proceed without inventing product meaning. Stop and reopen intake/research/user authority when that owner must supply the missing decision.
