# Agent Instruction Hygiene Workflow Plan

## Execution Shape

- Shape: lightweight local.
- Rationale: the user provided a concrete structural audit with eight actionable findings, and the affected surfaces are repository instruction/config/docs/scripts rather than service runtime behavior.
- Waiver: collapse specification, design, planning, implementation, and validation in one session. This is accepted because the work is bounded to agent/skill/tooling hygiene, no subagent fan-out was explicitly requested, and the audit already supplies the research frame.
- Research mode: local only. No subagent lanes because the current Codex environment only permits delegated agents when the user explicitly requests delegation.
- Workflow-plan adequacy challenge: waived for this same-session local pass. Risk is controlled by small artifact bundle, script checks, mirror checks, and fresh validation.

## Current Phase

- Current phase: done.
- Phase status: completed.
- Session boundary: waived for this lightweight local pass.
- Next session starts with: no follow-up required unless CI exposes an environment-specific issue.

## Scope

Fix the audit findings by:
- centralizing repeated subagent boilerplate in a shared contract and brief template,
- adding agents mirror sync/check tooling,
- closing delivery/distributed/observability review-skill coverage,
- recording explicit Codex agent model/reasoning tiers and a lower fan-out ceiling,
- unifying fan-in output expectations,
- documenting registry-style `.codex/config.toml` compatibility,
- updating README/build docs so the operational model matches the tooling.

Non-goals:
- service runtime behavior changes,
- Go application package changes,
- wholesale rewrite of every agent into a new prompt style,
- introducing write-capable subagents.

## Artifact Status

- `workflow-plan.md`: approved for lightweight local pass.
- `workflow-plans/implementation-phase-1.md`: approved for same-session implementation routing.
- `workflow-plans/validation-phase-1.md`: approved for closeout routing.
- `spec.md`: approved.
- `design/overview.md`: approved.
- `design/component-map.md`: approved.
- `design/sequence.md`: approved.
- `design/ownership-map.md`: approved.
- `plan.md`: approved.
- `tasks.md`: in progress.
- `test-plan.md`: not expected; validation is command and diff based.
- `rollout.md`: not expected; no runtime rollout.

## Blockers And Assumptions

- Assumption: the official Codex config reference remains authoritative for `agents.<name>.config_file`, so the registry layer is retained and checked rather than removed.
- Assumption: new review skills can start as concise instruction-only skills without bundled references; they may grow references after real review usage shows repeatable pressure points.
- Blockers: none.

## Phase Plan Links

- Implementation: `workflow-plans/implementation-phase-1.md`
- Validation: `workflow-plans/validation-phase-1.md`
