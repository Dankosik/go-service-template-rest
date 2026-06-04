# SOUL.md Orchestrator Personality Layer

Mode: full orchestrated
Status: verified
Specification date: 2026-06-04

## Context

This repository already has a strict spec-first operating contract in `AGENTS.md`, with expanded mechanics in `docs/spec-first-workflow.md`. The new `SOUL.md` layer is meant to reduce orchestrator personality drift for future Go service work without becoming a second workflow manual.

Research in `research/soul-md-agent-personality-practices.md` supports a clear boundary:

- `SOUL.md` is useful for stable identity, tone, judgment posture, and communication defaults.
- `AGENTS.md` must remain the authority for workflow rules, gates, commands, paths, role ownership, task-local artifacts, and validation discipline.
- Repo-local `SOUL.md` loading is host-dependent, so this repository must integrate it explicitly from `AGENTS.md` instead of assuming every agent host discovers it automatically.

## Scope / Non-goals

In scope:

- Add one repository-root `SOUL.md`.
- Define `SOUL.md` as the stable orchestrator personality and engineering-judgment layer for this Go service template.
- Integrate `SOUL.md` with `AGENTS.md` through an explicit lower-precedence reference and boundary rule.
- Preserve `AGENTS.md` as the controlling contract for workflow, gates, commands, paths, subagent protocol, artifact ownership, and validation requirements.
- Add validation hooks that make the new root instruction file and AGENTS bridge hard to accidentally remove.

Out of scope:

- Creating `SOUL.md` in this specification phase.
- Editing `AGENTS.md` in this specification phase.
- Creating host-specific mirrored SOUL files under `.codex/`, `.claude/`, `.cursor/`, `.gemini/`, `.opencode/`, or other runtime directories.
- Moving workflow invariants, artifact shapes, commands, route maps, service architecture, validation matrices, or task-ledger rules from `AGENTS.md` into `SOUL.md`.
- Changing runtime service behavior, APIs, generated contracts, data model, deployment policy, Go code, or CI job topology.
- Introducing a theatrical character persona, brand voice exercise, or entertainment-oriented assistant style.

## Constraints

- `AGENTS.md` stays the compact authority for repository-wide workflow rules and non-negotiable invariants.
- `docs/spec-first-workflow.md` stays the detailed workflow companion and must not be superseded by `SOUL.md`.
- Task-local `spec.md`, `design/`, and `tasks.md` artifacts remain authoritative inside their owned scopes.
- `SOUL.md` must not soften the existing production-ready default, task-ledger gate, phase boundary, subagent read-only rule, or fresh-proof requirement.
- `SOUL.md` should be concise enough to fit comfortably in model context. If it starts accumulating commands, paths, artifact templates, or policy matrices, that content belongs elsewhere.
- All shell validation commands in later phases must use the repository RTK rule, for example `rtk make guardrails-check`.

## Behavior / Contract Delta

ADDED:

- A root `SOUL.md` instruction artifact for stable orchestrator identity and communication defaults.
- An explicit `AGENTS.md` bridge that tells agents to load/apply `SOUL.md` only as lower-precedence personality guidance.
- Guardrail validation for the existence of `SOUL.md` and for the AGENTS/SOUL precedence boundary.

MODIFIED:

- Repository instruction loading becomes two-layered for hosts that honor the include/reference:
  - `AGENTS.md`: authority and operational contract.
  - `SOUL.md`: lower-precedence identity/personality layer.

UNCHANGED:

- `AGENTS.md` remains authoritative for workflow shape, gates, commands, paths, roles, subagents, artifacts, and validation.
- The spec-first workflow, phase boundaries, and production-ready target-state default remain unchanged.
- Existing agent mirror generation and subagent role files remain unchanged unless planning later finds a concrete drift check requirement.

## Decisions

- D1: Add exactly one repository-root `SOUL.md` for this scope.
  - Reason: a root file is the smallest durable repository-level surface that matches the requested orchestrator personality layer.
  - Rejected: host-specific mirrored SOUL files. They add distribution complexity without evidence that this task needs per-host runtime packaging.

- D2: `SOUL.md` owns stable orchestrator identity, not operating authority.
  - It may define role posture, engineering beliefs, ambiguity handling, communication style, and avoid-list behavior.
  - It must not define workflow rules, commands, paths, task-local artifact rules, subagent protocol, validation matrices, or repository architecture.

- D3: The accepted personality is a pragmatic senior Go service orchestrator.
  - The orchestrator should optimize for accurate, maintainable, production-ready service outcomes.
  - Complexity must earn its keep, but the agent must challenge underengineering as seriously as overengineering.
  - The desired balance is "simple enough, not simplistic": use the simplest design that satisfies the accepted scope's invariants, failure modes, ownership needs, and validation obligations.

- D4: `SOUL.md` must reinforce, not weaken, the production-ready default.
  - It should explicitly avoid MVP-now/future-hardening drift when the production-ready decision is knowable and in scope.
  - It should prefer existing repository patterns, Go-native and standard-library-first choices, and abstractions only when they remove real complexity, encode stable ownership, or protect an important policy boundary.

- D5: The accepted `SOUL.md` content shape is compact and behavioral.
  - Required sections:
    - `Role`
    - `Core Operating Beliefs`
    - `Engineering Balance`
    - `Default Behavior Under Ambiguity`
    - `Communication Style`
    - `Avoid`
    - `Boundaries`
  - The file should read as identity and judgment guidance, not as a process manual.

- D6: `AGENTS.md` integration must include both a prose precedence rule and an include/reference.
  - Add a short rule near the authority/purpose section explaining that `SOUL.md` is lower-precedence orchestrator personality guidance.
  - Add `@SOUL.md` using the repository's existing include convention.
  - Preserve the existing `@RTK.md` include and the RTK command rule.

- D7: The precedence rule must be explicit in both files.
  - `AGENTS.md` must say that `AGENTS.md`, `docs/spec-first-workflow.md`, task-local artifacts, and explicit user/system/developer instructions override `SOUL.md` for operational rules and decisions.
  - `SOUL.md` must say that if it conflicts with `AGENTS.md` or task-local artifacts, the agent follows the authoritative artifact and treats the conflict as drift to repair.

- D8: Guardrail validation is part of the accepted change.
  - `scripts/ci/required-guardrails-check.sh` should require root `SOUL.md`.
  - The guardrail should check that `AGENTS.md` references `SOUL.md` and that both files preserve the lower-precedence boundary.
  - Because this touches `scripts/ci/`, update the appropriate docs surface so docs drift rules remain satisfiable.

- D9: Separate technical design is not triggered.
  - The change is an instruction-surface/docs change with clear file ownership and no runtime, API, data, deployment, concurrency, or cross-service design.
  - The compact design decisions in this spec are sufficient for planning.
  - Technical design review is therefore not required before planning.

- D10: The next phase is planning, not implementation.
  - Planning must create `specs/orchestrator-soul-md/tasks.md` and run the required post-ledger task review/readiness gate before any file implementation starts.

## Open Questions / Assumptions

- [assumption] The repository include convention can reference `@SOUL.md` from `AGENTS.md` for hosts that process include directives. Hosts that ignore the include still receive the AGENTS precedence rule when they load `AGENTS.md`, but they may not apply the personality layer.
- [reopen_spec_if_false] If a target agent host requires mirrored or globally installed SOUL files for this repository to work as intended, reopen workflow planning and specification before adding host-specific distribution behavior.
- [assumption] No separate markdown linter exists in the current repository command surface. Validation should therefore rely on guardrails, agent mirror checks, docs drift checks when applicable, and diff sanity rather than inventing a new markdown tool.

No user decision is required for the accepted scope.

## Clarification Gate

Formal clarification is triggered because this is full-orchestrated, repository-wide instruction-surface work. A separate subagent lane was not spawned because the available `spawn_agent` tool is only permitted when the user explicitly asks for subagents, delegation, or parallel agent work. The specification pass therefore used a constrained local read-only clarification pass before file edits.

Scoped-down rationale:

- The approval risk is concentrated in one seam: preventing `SOUL.md` from becoming a second authority or weakening `AGENTS.md`.
- API, data, security, reliability, rollout, and runtime lenses cannot change approval for this accepted scope because the change has no runtime service behavior, generated contract, persisted state, deployment, or cross-service effect.
- The remaining validation question is answered by guardrail and docs-drift obligations in this spec.

Clarification result: PASS.

Resolved questions:

- Could `SOUL.md` weaken the spec-first workflow contract?
  - Resolution: no, because `SOUL.md` is constrained to lower-precedence personality guidance, with duplicate precedence boundaries in `AGENTS.md`, `SOUL.md`, and guardrail checks.
- Could repo-local `SOUL.md` fail to load in some hosts?
  - Resolution: yes, host loading is not universal; this is accepted for the current scope and mitigated by the `AGENTS.md` reference. Host-specific mirroring is out of scope unless reopened.
- Could "do not overengineer" push future agents toward under-designed services?
  - Resolution: the personality wording must say "complexity earns its keep" and "simple enough, not simplistic," preserving AGENTS.md's production-ready target-state default.
- Could validation be too weak for a standing instruction artifact?
  - Resolution: validation must include guardrail requirements for `SOUL.md` existence and AGENTS/SOUL precedence, plus normal repository drift checks for touched docs/scripts.

## Task Breakdown / Handoff Link

Use `specs/orchestrator-soul-md/tasks.md` after the planning phase creates and reviews it.

Planning must consume:

- this approved `spec.md`;
- `workflow-plan.md`;
- `workflow-plans/specification.md`;
- `AGENTS.md`;
- `docs/spec-first-workflow.md`;
- `research/soul-md-agent-personality-practices.md`;
- `Makefile` and `scripts/ci/required-guardrails-check.sh` for validation tasking.

Planning must not reopen the SOUL purpose, precedence boundary, or design-depth decision unless it finds a concrete contradiction with these artifacts.

## Validation

Implementation proof recorded on 2026-06-04:

- Root `SOUL.md` exists and targeted reads verified the required sections, lower-precedence personality boundary, and absence of command/template/path instructions.
- `AGENTS.md` references `SOUL.md`, includes `@SOUL.md`, preserves `@RTK.md`, and states that `AGENTS.md`, `docs/spec-first-workflow.md`, task-local artifacts, and explicit user/system/developer instructions override `SOUL.md` for operational authority.
- `scripts/ci/required-guardrails-check.sh` requires root `SOUL.md` and checks the AGENTS/SOUL reference and lower-precedence boundary in both files.
- `docs/build-test-and-development-commands.md` documents that `make guardrails-check` enforces root `SOUL.md` and the AGENTS/SOUL lower-precedence boundary.
- `rtk make guardrails-check` passed with `required repository guardrails check passed`.
- `rtk make agents-check` passed with `agents check complete`.
- `rtk git diff --check` passed before closeout updates.
- `rtk make docs-drift-check BASE_REF=origin/main HEAD_REF=46390788a1122e123e461cb18917b8b9b9a8a512` passed. The head ref was a temporary tracked-worktree commit created with `rtk proxy git stash create docs-drift-current-worktree` because branch `HEAD` did not include the uncommitted `scripts/ci/` and docs edits.
- Runtime Go tests were not required because the implementation touched only repository instruction, guardrail, docs, and task-local closeout surfaces.

## Outcome

Implementation verified. The repository now has a root-only `SOUL.md` personality layer, an explicit lower-precedence `AGENTS.md` bridge, guardrail coverage for SOUL presence and boundary drift, paired command documentation, and ledger-owned validation evidence.

No runtime Go code, API contract, generated contract, persisted data, deployment policy, CI job topology, or host-specific SOUL mirror was changed.

The workflow is complete; there is no next workflow session for this task.
