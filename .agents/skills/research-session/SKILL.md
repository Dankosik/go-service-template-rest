---
name: research-session
description: "Run a bounded current-source research pass for evidence that can change a repository task decision. Use whenever external-platform behavior, an unfamiliar mechanism, new infrastructure/dependency, or a non-trivial design choice might otherwise be inferred from model memory; persist notes only when another phase or session needs them."
---

# Research Session

Use [the research phase](../../../docs/spec-first-workflow/phases/research.md).

Start from concrete evidence questions. For each, name the decision it can change, authoritative source, minimum evidence, freshness need, and stop condition. For external or unfamiliar work, search current official docs/source plus credible real implementations or engineering writeups before design. Official sources own current contracts; real-world sources supply proven patterns and operational pitfalls. Do not fill a freshness gap from model memory. Use read-only lanes only for independent questions where separate context materially helps.

Write `research/*.md` only when reuse, conflict, freshness, or auditability justifies it. Return findings, limits, conflicts, and decision implications. Do not write the final spec/design decision.

Stop when questions are answered or honestly bounded; name the smallest missing-evidence or user-decision owner when blocked.
