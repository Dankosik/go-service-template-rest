---
name: test-design-session
description: "Non-obvious proof: Use when accepted behavior needs dedicated test design before planning. Own proof obligations, deterministic oracles, proving layers, and the test-design handoff; Skip when obvious inline planning proof, executable test implementation, or test review is enough."
---

# Test Design Session

Must read [Test Design](../../../docs/spec-first-workflow/phases/test-design.md); it owns proof disposition, review, and the planning boundary. Build a falsifier for every non-obvious obligation after reconstructing every affected acceptance claim, invariant, transition, failure mode, and protected side effect; retain obvious proof inline. Required Review begins only after every obligation has Test Design's canonical disposition; repair closes or reopens each finding before fresh Review. Stop only when that canonical boundary permits planning or reopens specification or design.
