# Context-First Artifact Audit Fixes Spec

## Context

The completed read-only audit found that the workflow contract is already strong, but several artifact handoff surfaces should be tightened so future LLM sessions can resume from files without chat memory.

Findings to fix:

- High priority: post-code implementation, review, and validation phase-control shapes are less concrete than pre-code shapes.
- High priority: `workflow-plan.md` lacks an explicit next-session context bundle field.
- Medium priority: research-backed spec decisions need a clearer optional provenance link pattern.
- Medium priority: negative artifact statuses should consistently include trigger rationale across technical design and planning guidance.
- Medium priority: `tasks.md` should allow a tiny implementation handoff header so implementation sessions do not infer entry context from scattered files.
- Low priority: the first real task-local bundle should act as a minimal end-to-end example, without adding a universal template.
- Low priority: review-phase finding disposition should have a compact orchestrator-owned shape.

## Scope / Non-goals

In scope:

- Update repository workflow documentation and existing skill/reference guidance for the findings above.
- Preserve the existing artifact authority model from `AGENTS.md`.
- Keep all format changes small and context-first.

Non-goals:

- Do not invent a new workflow philosophy.
- Do not add broad universal templates or new just-in-case artifacts.
- Do not change Go runtime behavior, API contracts, CI behavior, or subagent permissions.

## Constraints

- `spec.md` remains the canonical decision artifact.
- `workflow-plan.md` remains cross-phase control.
- `workflow-plans/<phase>.md` remains phase-local routing only.
- `design/` remains task-local technical context.
- `tasks.md` remains the executable implementation handoff.
- Post-code phases may not create missing workflow/process artifacts after implementation starts.

## Decisions

- Add a compact `Next Session Context Bundle` expectation to master workflow-plan guidance.
- Make post-code phase-control shapes more explicit in `docs/spec-first-workflow.md` and the planning reference that creates future phase files.
- Add optional provenance guidance for research-backed `spec.md` decisions, without making research notes authoritative.
- Require trigger rationale wording when technical design and planning record `not expected`, `conditional`, or `waived` artifact statuses.
- Allow a short `tasks.md` handoff header before checkbox items.
- Preserve this task-local bundle as the first minimal concrete bundle example; do not add a separate example artifact.

## Open Questions / Assumptions

Assumption: documentation and skill edits are enough for this pass; no generated mirrors need separate manual edits unless `make agents-check` or `make skills-check` says so.

## Task Breakdown / Handoff Link

See `tasks.md`.

## Validation

Claim: context-first artifact audit fixes are ready for handoff.

Scope: docs, skill guidance, skill mirrors, and this task-local bundle.

Verification Commands:

- `make skills-sync`
- `git diff --check`
- `make agents-check`
- `make skills-check`

Observed Result: all commands above completed successfully in this session.

Conclusion: verified for the docs/skills handoff scope.

Next Action: none.

## Outcome

Complete. The workflow docs and skill guidance now preserve the context-first improvements requested by the audit, and skill mirrors were synced.
