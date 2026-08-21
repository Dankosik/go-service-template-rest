---
name: go-test-implementation
description: "Go test code: Use for a test-only change or when an accepted non-routine oracle, fixture, or harness requires its own implementation method. Own test mechanics; Skip routine tests coupled to a production change, production code, and proof strategy."
metadata:
  invocation: model
  kind: method
---

# Go Test Implementation

A test is an **executable falsifier**: it exists to reject the wrong behavior at the smallest deterministic layer, through an oracle independent of the code it judges.

`accepted obligation -> proving layer -> deterministic controls -> independent oracle -> executable proof -> disposition`

Routine focused tests required by a production unit remain with `go-coder`.

Reconstruct every obligation from the accepted proof handoff and its cited behavior, then use the oracle as the anchor: inspect existing proof and choose the smallest deterministic test layer and independent observable that rejects the wrong behavior. Source-string presence substitutes for execution only when the exact text is itself the accepted output contract. When a concrete test pressure can change the proving layer, controls, oracle, or command, load [the reference selector](references/index.md) and let it choose one reference by default, adding another only for an independent pressure.

Complete when every accepted proof obligation has executable proof or a named blocker; distinguish existing from implemented proof, and carry an independent oracle with each executable proof. Route test review to `go-test-strategy` and a non-obvious or disputed claim-to-command match to `go-verification-before-completion`; hand unresolved public API behavior to `go-api-contract`, router composition to `go-chi`, and other unresolved behavior or proof strategy to its accepted owner. Return obligation dispositions, changed tests and fixtures, commands and results, and gaps.
