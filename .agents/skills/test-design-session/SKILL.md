---
name: test-design-session
description: "Non-obvious proof: Use when accepted behavior needs dedicated test design before planning. Own proof obligations, deterministic oracles, proving layers, and the test-design handoff; Skip when obvious inline planning proof, executable test implementation, or test review is enough."
---

# Test Design Session

Must read [Test Design](../../../docs/spec-first-workflow/phases/test-design.md); it owns proof disposition, review routing, and the planning boundary. Build a falsifier for every non-obvious obligation after reconstructing every affected acceptance claim, invariant, transition, failure mode, and protected side effect; retain obvious proof inline. Self-review is the default. Run exactly one independent read-only review only when explicitly requested or when the fixed proof design is high-impact, hard to reverse, cross-owner, or weakly falsifiable. Repair findings or reopen their owner; obtain fresh review only when that triggered review occurred and a semantic repair changed material. Stop only when that canonical boundary permits planning or reopens specification or design.
