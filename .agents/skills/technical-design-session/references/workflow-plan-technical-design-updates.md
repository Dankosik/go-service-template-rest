# Workflow Plan Technical Design Updates

Use this file when updating the master workflow plan and the technical-design phase workflow plan. Workflow files route the work; they are not the design bundle.

## When To Load
- Load after writing or repairing required and triggered design artifacts.
- Load when technical design is blocked and workflow control must record the reopen target.
- Load when master and phase-local workflow files disagree.
- Load before ending the session so the next session can resume without chat archaeology.

## Good Design-Session Outputs
- `workflow-plan.md`: current phase `technical-design`, phase status `complete`, required design artifacts `approved`, triggered conditional artifacts `approved` or `not expected`, blockers `none`, `Session boundary reached: yes`, `Ready for next session: yes`, `Next session starts with: planning`.
- `workflow-plans/technical-design.md`: pass type, phase status, completion marker, artifact statuses, stop rule, local blockers, parallelizable follow-up if any, and planning handoff state.
- Blocked master update: current phase `technical-design`, status `blocked`, blocker names the missing spec decision, reopen target `specification`, next session starts with `specification`, and planning readiness is `no`.
- Repair pass update: records which stale design artifact was repaired and leaves unrelated artifact statuses untouched.

## Bad Design-Session Outputs
- "Updated design files; workflow state is obvious from the diff."
- `workflow-plan.md` says planning can start while `workflow-plans/technical-design.md` says sequence is still pending.
- Phase-local workflow file contains the real component map instead of linking `design/component-map.md`.
- Master workflow file omits `Session boundary reached`, `Ready for next session`, or `Next session starts with`.

## Conditional Artifact Examples
- If `test-plan.md` is triggered, record its status in both workflow files as `approved`, `draft`, or `blocked`; do not hide it under a generic design status.
- If `rollout.md` is not triggered, record `rollout.md: not expected` rather than creating a blank file.
- If `design/contracts/` is triggered but only draft, mark technical design blocked or in progress instead of planning-ready.
- If a conditional artifact is stale, record repair status and blocker separately from the required core artifact status.

## Blocked Handoff Examples
- `workflow-plan.md` and `workflow-plans/technical-design.md` disagree about phase status or next session.
- Required design artifacts are approved, but a triggered conditional artifact is missing.
- The stop rule is absent, making it unclear whether the next session may begin planning.
- A blocker is mentioned in chat but not recorded in workflow control.

## Exa Source Links
Exa MCP search and fetch were attempted before writing these examples, but the provider returned a 402 credits-limit error. Treat these fallback-verified links as calibration only; repo-local files remain authoritative.

- Google Cloud Architecture Framework: https://docs.cloud.google.com/architecture/framework
- Google Cloud Architecture decision records: https://docs.cloud.google.com/architecture/architecture-decision-records
- Azure Well-Architected Framework: https://learn.microsoft.com/en-us/azure/well-architected/what-is-well-architected-framework
- arc42 documentation: https://docs.arc42.org/home/

