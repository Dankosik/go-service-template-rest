---
name: go-design-spec
description: "Author the smallest coherent system-mechanism and Go-ownership design after behavior is ready. Use for focused design decisions or delegated design content; skip root phase orchestration and review-loop ownership, which belong to technical-design-session."
---

# Go Design Spec

Use after behavior is ready but implementation still needs runtime or code-ownership decisions. Follow [system/integration design](../../../docs/spec-first-workflow/phases/system-integration-design.md) and [Go ownership design](../../../docs/spec-first-workflow/phases/go-code-ownership-design.md).

Choose observable mechanism before package placement. Name source of truth, contracts, sequence/failures, data/consistency, security/reliability/rollout boundaries, package/file owners, dependency direction, generated/manual authority, cleanup, test owner, and proof only where relevant.

Prefer one `design/overview.md`; split a focused artifact only when it creates a real owner or review boundary. Apply the canonical [research solution-discovery and evidence method](../../../docs/spec-first-workflow/phases/research.md#method) before selecting from a live solution choice. Reject speculative interfaces, layers, and generic shared packages.

Authoring is review-ready when planning could name files, owners, dependencies, tests, cleanup, and proof without making design decisions. Structured/orchestrated work proceeds to planning only after independent technical-design review reaches the shared convergence condition. In explicitly read-only review mode, return anchored `PASS`, `CONCERNS`, or `FAIL` findings without editing.
