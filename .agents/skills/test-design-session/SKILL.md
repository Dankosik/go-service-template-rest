---
name: test-design-session
description: "Non-obvious proof: Use when accepted behavior needs dedicated test design before planning. Own proof obligations, deterministic oracles, proving layers, and the test-design handoff; Skip when obvious inline planning proof, executable test implementation, or test review is enough."
---

# Test Design Session

Must read [Test Design](../../../docs/spec-first-workflow/phases/test-design.md); it owns proof disposition, the review trigger, and the planning boundary. Build the smallest falsifiers needed for non-obvious accepted risks and keep obvious proof inline. Apply root self-review and use independent review only when triggered. Carry dispositioned concerns forward; seek fresh review only after `FAIL` repair or material candidate change. Stop when Test Design permits planning or reopens specification or design.
