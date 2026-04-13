# Spec Handoff To Technical Design

## Behavior Change Thesis
When loaded at the `spec.md -> design/` handoff or when design detail leaks into the spec, this file makes the model keep `spec.md` at behavior-decision level with explicit handoff and reopen conditions instead of stuffing component maps, sequences, ownership maps, or task lists into the spec.

## When To Load
Load this when a non-trivial spec is about to hand off to `technical design`, or when a draft starts to absorb component ownership, runtime sequence, package names, API contract detail, or implementation milestones.

## Decision Rubric
- `spec.md` decides behavior, scope, constraints, accepted risk, and proof consequences.
- `design/overview.md`, `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md` decide task-local technical shape.
- `tasks.md` decides execution order and implementation steps; optional `plan.md` carries only supplemental strategy when justified.
- A handoff note should name what design must derive, not pre-answer design with package or call-order decisions.
- Add `[reopen_spec_if_false]` when a downstream design discovery would invalidate a spec-level decision.
- If the spec cannot hand off without reopening problem framing, keep it draft or blocked.

## Imitate

```markdown
## Decisions
- Runtime token reload preserves the existing authentication contract.
- The configuration source remains the source of truth.
- Failed reloads keep the last known-good token set active and surface degraded operator state.

## Plan Summary / Link
- Technical design must derive component ownership, reload sequence, and observability placement from these decisions before planning starts.
```

Copy this: the spec fixes behavior and boundary, then points the next phase at design-owned work.

```markdown
## Open Questions / Assumptions
- [reopen_spec_if_false] If technical design finds that config is not the sole source of truth for admin tokens, return to specification before planning.
```

Copy this: design has a clear stop rule instead of silently changing the spec's core decision.

## Reject

```markdown
## Decisions
- Add `ReloadCoordinator` in `internal/auth/reload.go`.
- The handler calls config, then auth, then metrics, then logger.
- Implementation tasks:
  - Create coordinator.
  - Add tests.
  - Update docs.
```

Failure: package ownership and runtime sequence belong in `design/`; task breakdown belongs in `tasks.md`.

## Agent Traps
- Treating "handoff-ready" as "technical design already written."
- Putting canonical API contract detail in `spec.md` when a repo-owned contract source or `design/contracts/` belongs downstream.
- Loading `decision-placement-and-artifact-ownership.md` for every handoff question; use this narrower reference when approval/handoff readiness is the real pressure.
- Letting design silently rewrite a spec decision instead of reopening specification.
