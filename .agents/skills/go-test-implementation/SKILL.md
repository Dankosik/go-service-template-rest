---
name: go-test-implementation
description: "Use when approved behavior or test design is ready for executable Go proof; Own the smallest deterministic test code and fixtures covering required success and failure paths; Skip when production behavior must change, proof strategy is unresolved, existing tests need review, or claims need closeout verification."
---

# Go Test Implementation

Use [Test Design](../../../docs/spec-first-workflow/phases/test-design.md) for accepted proof obligations and [Implementation / Validation / Closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md) for authorized edits. Load [the reference selector](references/index.md) only when its pressure changes the proving layer or controls. Translate obligations into the smallest deterministic test layer and independent oracle. Return coverage, changed tests, results, and gaps; stop when behavior or proof layer is unresolved.

Reference entry points: [obligations](references/obligation-to-test-translation.md), [proving layer](references/minimal-proving-layer-selection.md), [Go tests](references/go-test-construction-patterns.md), [concurrency/time](references/deterministic-concurrency-and-time-tests.md), [errors/context](references/error-context-and-cancellation-tests.md), [API contracts](references/api-contract-test-patterns.md), [data/cache](references/data-cache-integration-test-patterns.md), and [verification](references/verification-command-evidence.md). Select through the index above.
