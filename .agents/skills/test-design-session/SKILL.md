---
name: test-design-session
description: "Own non-obvious proof design before planning; keep obvious proof as an inline planning handoff."
---

# Test Design Session

Own the root [Test Design](../../../docs/spec-first-workflow/phases/test-design.md) macro phase through its canonical Outputs, Review, and Stop Rule when proof spans meaningful scenarios, failure modes, protected concerns, or proof boundaries.

Apply [go-qa-tester-spec](../go-qa-tester-spec/SKILL.md) for risk selection, scenario design, proof-boundary choice, and executable quality gates. Keep proof inline when the canonical persistence rule does not require `test-plan.md`. Do not write tests, edit `tasks.md`, or change approved behavior.

The owning root repairs review findings and re-reviews to the shared convergence condition, then may continue into planning and implementation in the same authorized request unless the user named Test Design or standalone QA review as the boundary.

Success means planning can map proof without inventing behavior or test strategy. Otherwise reopen the owner named by the canonical Test Design phase.
