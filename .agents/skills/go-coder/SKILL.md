---
name: go-coder
description: "Go change: Use for an authorized implementation outcome. Own production code, tests, cleanup, and proof; Skip open decisions and diagnosis/test/verification-only work."
---

# Go Coder

One authorized outcome becomes one **surgical change**: the smallest diff at the earliest valid owner that makes every accepted criterion true, provable, and clean.

Measure smallest across repository-owned behavior rather than changed lines: when a new case is evidence of the same policy, extend or refactor its existing owner instead of cloning the path.

`accepted criteria -> earliest valid owner -> smallest change -> required tests -> cleanup -> proof -> return`

Direct work follows the root [Direct Work](../../../AGENTS.md#direct-work)
contract. Load [Implementation / Validation / Closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md)
only for structured work or one of its conditional validation, deployment,
review, integration, or blocked-closeout boundaries.

Reconstruct every accepted criterion and named source before touching code. Map
each to production change, test, cleanup, evidence-backed unchanged path,
proof-only action, or blocker while preserving unrelated work and generated
authority. Load one matching [implementation reference](references/index.md)
only when a concrete pressure changes the method.

Before returning, check the far side of each touched boundary: affected caller,
concurrent reader, or version that must coexist.

A surgical change is complete only when every criterion has a terminal disposition and focused proof, every triggered gate passes, and every changed file, command result, and gap is returned. Stop and reopen the owner on unresolved behavior, ownership, policy, or proof — an invented decision is a defect even when the code works.
