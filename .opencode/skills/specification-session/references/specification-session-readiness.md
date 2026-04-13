# Specification Session Readiness

## Behavior Change Thesis
When loaded for incomplete phase context, under-researched candidate decisions, or pressure to approve `spec.md` from partial inputs, this file makes the model choose a spec-ready check, bounded assumption, or reopen/block decision instead of the likely mistake of approving by momentum.

## When To Load
Load this before deciding whether a dedicated `specification-session` may draft, repair, or approve `spec.md`.

## Decision Rubric
- Confirm phase ownership first: read `workflow-plan.md` before `workflow-plans/specification.md`, and continue only if specification is the current phase or an explicit reopen target.
- Confirm decision material second: behavior delta, scope cuts, constraints, risk hotspots, and validation expectations must be explicit enough to write stable `Decisions`.
- Treat a missing answer as a blocker when it can change scope, correctness, ownership, rollout, acceptance, or validation proof.
- Treat a remaining unknown as an assumption only when it is bounded, visible in `Open Questions / Assumptions`, and has a clear reopen condition.
- Use the clarification challenge to pressure-test candidate decisions, not to discover the basic product direction.
- If the work is tiny enough for a direct inline path, skip this wrapper instead of creating ceremony.

## Imitate
Blocked readiness:

```text
Spec readiness: blocked
Reason: candidate decisions do not define the tenant boundary for export job visibility.
Reopen target: targeted research or domain clarification before specification approval.
Spec state: draft only; do not approve.
```

Ready with bounded assumption:

```text
Spec readiness: ready with recorded assumption
Assumption: admin-only controls follow the repository's existing admin authorization pattern.
Risk: if research later contradicts that pattern, reopen specification before technical design.
```

Copy the distinction: one missing answer changes the decision record; the other is bounded by existing repo policy and a reopen condition.

## Reject
Momentum approval:

```text
Spec status: approved
Open question: tenant visibility still TBD.
```

This fails because tenant visibility can change acceptance, ownership, and validation.

Challenge misuse:

```text
Run spec-clarification-challenge to discover whether this is SAML or OIDC.
```

This fails because the challenge is for candidate decision pressure-testing, not primary product discovery.

## Agent Traps
- Reading research first and never checking whether the master workflow plan still points at specification.
- Treating `design/`, tests, or implementation as in-scope while checking readiness.
- Recording "no blockers" in chat while leaving stale blockers in workflow artifacts.
- Approving a spec because every section has prose, even though the approval-changing decision is absent.
