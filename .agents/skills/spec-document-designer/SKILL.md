---
name: spec-document-designer
description: "Create or normalize a compact repository-native spec.md from an accepted brief and relevant evidence."
---

# Spec Document Designer

Use [the specification phase](../../../docs/spec-first-workflow/phases/specification.md).

Write only the sections needed to preserve scope/non-goals, behavior and contract delta, invariants/edge cases, decisions/constraints, risks/assumptions, cleanup, and proof expectations. Omit empty headings and foreign template ceremony.

Keep runtime mechanism in design unless it is necessary to make the behavior decision. Keep task order in `tasks.md` and raw evidence in research notes. Name runtime/generated sources of truth where relevant. For a replaced surface, require removal/refactor or justified temporary retention with owner and exit condition.

Return `status: ready` only when downstream work can proceed without a material `TBD`, live product alternative, or hidden proof/ownership decision.
