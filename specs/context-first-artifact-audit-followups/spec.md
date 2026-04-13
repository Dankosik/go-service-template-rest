# Context-First Artifact Audit Followups Spec

## Context

The read-only context-first artifact audit found no remaining high-priority blocking gap after the previous `context-first-artifact-audit-fixes` pass, but it did identify a small set of medium-priority drift risks and low-priority polish items that should be tightened.

Findings to fix:

- Require `workflow-plan.md` to carry a `Next Session Context Bundle` field even when it only points at the default resume order.
- Promote the uncertainty labels used by `spec.md` guidance into central workflow and specification skill guidance, not only a lazily loaded reference.
- Make `design/overview.md` artifact indexes include status and trigger rationale for conditional artifacts when the bundle is planning-bound.
- Clarify status vocabulary so `Phase status` stays lifecycle-oriented while reopened/done remain routing or task state.
- Allow readable multi-line `tasks.md` task items when proof, dependency, or reopen detail would make one-line bullets dense.
- Record that historical task-local bundles under `specs/` are examples to study, not universal templates to copy.

## Scope / Non-goals

In scope:

- Update repository workflow documentation and existing session/deeper skill guidance for the findings above.
- Keep the existing authority model: `spec.md` for decisions, `design/` for task-local technical context, `tasks.md` for executable handoff, `workflow-plan.md` for cross-phase control, and `workflow-plans/<phase>.md` for phase-local routing.
- Keep changes small, concrete, and context-first.

Non-goals:

- Do not invent new artifacts or broad universal templates.
- Do not change Go runtime behavior, API contracts, CI behavior, or subagent permissions.
- Do not treat historical `specs/` bundles as generated templates.

## Constraints

- `AGENTS.md` remains the compact repository-wide contract.
- `docs/spec-first-workflow.md` remains the detailed workflow companion and should own central shape guidance.
- Session skills may mirror or reinforce the central rules, but must not become alternate workflow authorities.
- This pass is docs/skills-only and may use a lightweight-local same-session waiver.

## Decisions

- Make `Next Session Context Bundle` an always-present master workflow field; it may explicitly say default resume order is enough.
- Add a central uncertainty-label mini-vocabulary for `Open Questions / Assumptions`.
- Require planning-bound `design/overview.md` artifact indexes to show artifact status and trigger rationale for conditional artifacts.
- Separate `Phase status` values from `Task state` or `Routing state` values in docs and references.
- Keep `tasks.md` checkbox-ledger shape, but allow multi-line task bodies for readability when they still remain executable and proof-bound.
- Add a note that existing task-local bundles are examples of completed work, not templates or alternate authorities.

## Open Questions / Assumptions

- [assumption] Documentation and skill guidance changes are sufficient for this follow-up; no generated mirrors are expected unless validation says otherwise.
- [accepted_risk] Workflow plan adequacy challenge is waived for this lightweight-local pass because the user explicitly asked to fix all concrete findings now, the scope is docs/skills-only, and this environment may spawn subagents only on explicit delegation requests.

## Task Breakdown / Handoff Link

See `tasks.md`.

## Validation

Claim: context-first artifact audit follow-up fixes are complete for the docs/skills scope.

Scope: central workflow docs, session/deeper skill guidance, mirrored skill directories, and this task-local bundle.

Verification Commands:

- `git diff --check`
- `make agents-check`
- `make skills-check`
- `make skills-sync`
- `make skills-check`
- final `git diff --check`
- final trailing-whitespace check for new task-local files
- final `make agents-check`
- final `make skills-check`

Observed Result:

- `git diff --check` passed before closeout and after closeout updates.
- `make agents-check` passed.
- Initial `make skills-check` reported stale `.claude`, `.gemini`, `.github`, `.cursor`, and `.opencode` skill mirrors.
- `make skills-sync` completed successfully.
- Re-run and final `make skills-check` passed.
- Final trailing-whitespace check for new task-local files passed.

Conclusion: verified for the docs/skills guidance and mirrored skill-surface scope.

Next Action: none.

## Outcome

Complete. All audit follow-up findings were applied to the central workflow guidance and mirrored skill surfaces, with no Go runtime, API, CI, migration, or subagent-permission changes.
