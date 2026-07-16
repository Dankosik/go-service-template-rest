---
name: go-test-implementation
description: "Executable proof: Use when approved behavior or test design is ready for Go test code and fixtures. Own the smallest deterministic layer and independent oracle covering required success and failure paths; Skip when production behavior must change, proof strategy is unresolved, existing tests need review, or claims need closeout verification."
---

# Go Test Implementation

Use [Test Design](../../../docs/spec-first-workflow/phases/test-design.md) for accepted proof obligations and [Implementation / Validation / Closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md) for authorized edits. When a concrete test pressure can change the proving layer, controls, oracle, or command, load [the reference selector](references/index.md) and let it choose one reference by default, adding another only for an independent pressure. Turn each obligation into executable proof at the smallest deterministic test layer with an independent oracle. Complete when every accepted proof obligation has executable proof or a named blocker. Route test review to `go-test-strategy`; hand unresolved public API behavior to `go-api-contract` and other unresolved behavior or proof strategy to its accepted owner. Return obligation dispositions, changed tests and fixtures, commands and results, and gaps.
