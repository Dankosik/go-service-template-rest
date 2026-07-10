---
name: technical-design-session
description: "Own the user-started technical-design macro phase: run system-integration design, Go code ownership design, independent technical review, repair, and fresh re-review before handing off."
---

# Technical Design Session

## Eligibility And Outcome

Use only from a current specification-review-approved `spec.md` when separate technical design is triggered. The invariant internal route is:

`system-integration-design -> go-code-ownership-design -> technical design review`

Do not use it for spec repair, compact lean design already sufficient in the spec, test design, planning, implementation, or validation. It owns technical design review internally.

The outcome is one reviewed technical design bundle with both triggered checkpoint contributions and a current technical-design-review verdict, or an exact blocker/reopen target.

## Canonical Owners

- [System / Integration Design](../../../docs/spec-first-workflow/phases/system-integration-design.md) owns observable behavior, external systems, data/source-of-truth, sequence, failures, validation, rollout, and triggered contract design.
- [Go Code / Ownership Design](../../../docs/spec-first-workflow/phases/go-code-ownership-design.md) owns package/file responsibility, dependency direction, Go-native abstractions, cleanup, and test ownership while preserving the prior checkpoint.
- [Technical Design Review](../../../docs/spec-first-workflow/phases/technical-design-review.md) owns the distinct review gate.
- [Artifact model](../../../docs/spec-first-workflow/shared/artifact-model.md) owns design depth, typed state, routing identity, and phase-control eligibility.
- [Subagents and handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md) owns fan-out gates, authorization wording, resume order, and final prompt rendering.

Use `go-design-spec` for integrated design assembly. Load only the domain specialist skill or bundled reference whose decision frontier can change the active checkpoint. Read [repository architecture](../../../docs/repo-architecture.md) only when repository ownership or source-of-truth boundaries matter.

## Allowed Side Effects

This session may create or repair both ordered checkpoints' approved design surfaces, record internal technical-design review cycles, update existing `workflow-plan.md`, and update checkpoint phase-control files only when ROUTING-PHASE-CONTROL allows them.

It must not edit `spec.md`, another checkpoint's decisions, technical-design-review verdicts, `test-plan.md`, `tasks.md`, code, tests, migrations, generated output, or implementation handoff.

## Unique Method

1. Verify current spec approval, active checkpoint, routing identity, design-depth trigger, prior checkpoint status, and required contract/provider evidence.
2. Record checkpoint-scoped `Design fan-out: complete | scoped_down | local_only | blocked`. Use lanes only for concrete independent bounded questions whose separate context materially improves speed or quality. Full shape, FULL-* evidence, and domain count determine design depth, not automatic fan-out. Default to no more than three concurrently active subagent lanes.
3. In system/integration design, close behavior and mechanism seams without selecting Go file layout. Resolve contract design as created, compact_sufficient, not_expected with evidence, or blocked.
4. In Go code/ownership design, consume system decisions unchanged and name owner packages/files, rejected placements, focused responsibilities, dependency direction, cleanup/removal, abstractions, and test ownership.
5. Before selecting a non-trivial design shape, confirm Pattern Fit Diligence. If evidence is missing, open a Pattern Fit research or review lane rather than inventing the answer here.
6. Use `go-design-spec` to assemble coherent design context without creating a second specification.
7. After both checkpoints are review-ready, invoke fresh read-only technical-design review. Repair design-owned findings in the owning checkpoint, mark the old verdict stale, and launch fresh re-review at the same or stronger model tier.

Repository-standing authorization covers read-only design and review lanes. If the primary surface is unavailable, use the configured independent fallback; do not convert mandatory review to local-only.

## Success, Blocked Stop, And Reopen

System/integration success continues in the same root session to Go code/ownership design. Go code/ownership success continues to independent technical design review. Macro-phase success requires current `PASS` or eligible `CONCERNS` over the final combined design revision.

Stop blocked for stale or missing spec approval, wrong checkpoint order, unresolved public/provider contract, ambiguous source owner, incomplete required fan-out, missing Pattern Fit evidence, or a decision that belongs to specification.

Reopen specification for behavior or scope, research for evidence, or workflow planning for routing. Repair the active design checkpoint and re-review inside this session. Render a handoff only after technical design closes, through [Subagents and handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md).
