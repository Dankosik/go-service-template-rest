# internal/config review and fix-planning workflow

## Task frame

- Goal: review `/Users/daniil/Projects/Opensource/go-service-template-rest/internal/config`, prepare an implementation handoff, then apply the accepted fixes.
- Scope: accepted review findings in `internal/config` plus the narrow bootstrap ownership cleanup required by `config.ErrDependencyInit`.
- Non-goals: no broad config-loader rewrite, no config source precedence change, no new config keys, no moving `MongoProbeAddress` out of `internal/config`.
- Constraints: user explicitly wants pre-implementation context recorded in files; unrelated worktree changes stay untouched.
- Success check: task-local `spec.md`, `design/`, `plan.md`, `tasks.md`, implementation phase control, and final verification evidence exist for the accepted fixes.

## Execution control

- Execution shape: lightweight local follow-up after an agent-backed review.
- Current phase: `implementation-phase-1` complete.
- Phase-collapse waiver: `specification`, `technical design`, and `planning` are collapsed into this local pre-code pass because the behavior deltas are bounded review findings, repository evidence is already gathered, and there are no unresolved product/API/data/migration decisions.
- Research mode: local synthesis from completed review lanes plus source inspection.
- Spec clarification challenge: waived under the lightweight-local rationale; no user-only or planning-critical unanswered product decision remains.
- Workflow plan adequacy challenge for planning handoff: waived under the same lightweight-local rationale; no new subagent fan-out is needed for this no-code handoff.
- Implementation readiness: `PASS`; implementation completed subject to final verification evidence in this session.
- Existing unrelated worktree changes: deleted files under `specs/template-readiness-review`; leave untouched.

## Artifact status

- `workflow-plan.md`: complete.
- `workflow-plans/review-phase-1.md`: complete.
- `workflow-plans/planning.md`: complete.
- `workflow-plans/implementation-phase-1.md`: complete.
- `research/review-coverage.md`: complete.
- `spec.md`: approved for this bounded fix set.
- `design/overview.md`: approved.
- `design/component-map.md`: approved.
- `design/sequence.md`: approved.
- `design/ownership-map.md`: approved.
- `plan.md`: approved.
- `tasks.md`: complete.
- `test-plan.md`: not expected; validation is small enough to live in `plan.md` and `tasks.md`.
- `rollout.md`: not expected; no deployment choreography or persisted-state migration.

## Phase plans

- Completed review: `workflow-plans/review-phase-1.md`.
- Completed planning: `workflow-plans/planning.md`.
- Completed implementation: `workflow-plans/implementation-phase-1.md`.

## Challenge and synthesis status

- Review workflow adequacy challenge: reconciled; no blocking workflow-control adequacy gaps found.
- Domain review lanes: completed.
- Accepted findings: unsafe float-to-int bounds, non-finite sampler arg acceptance, `SplitHostPort` used as numeric-port validation, ineffective empty-path guard, Redis mode normalization split, namespace env key literal drift, nil `ErrorType` default, and dependency-init sentinel ownership drift.
- Filtered finding: moving `MongoProbeAddress` to bootstrap is intentionally not in scope because `docs/configuration-source-policy.md` explicitly assigns the guard-only probe-address helper to `internal/config`.
- Deferred handoff: local symlink behavior in `load_koanf.go` needs a separate security/source-policy decision if changed; this plan does not change it.

## Session boundary

- Session boundary reached: yes.
- Ready for next session: yes.
- Next session starts with: optional post-implementation review or closeout only if requested.
- Stop rule: implementation phase is complete; do not reopen scope unless a verification or review finding identifies a real spec/design gap.
