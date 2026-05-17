# Research Phase Plan

Phase: research
Phase status: complete
Research mode: local
Mode rationale: the evidence surfaces are bounded to repository workflow docs, one existing task-local bundle, and public workflow documentation. No write-capable or delegated lanes were used.

## Research Tracks

| Track | Question | Evidence Target | Status |
| --- | --- | --- | --- |
| Local workflow contract | Which current phases and artifacts carry real quality? | `AGENTS.md`, `docs/spec-first-workflow.md`, `docs/subagent-contract.md`, `docs/subagent-brief-template.md`, `docs/build-test-and-development-commands.md` | complete |
| Real bundle pain map | Where does the current workflow become heavy in a real template change? | `specs/railway-auto-migrations/` workflow, spec, design, tasks, validation evidence | complete |
| External workflow comparison | How do BMAD, Superpowers, GSD, Spec Kit, Kiro, OpenSpec, Task Master, Warp, and similar systems reduce overhead? | Current public docs and GitHub pages, accessed 2026-05-17 | complete |
| Synthesis | Which simplifications preserve quality while reducing artifact and phase overhead? | Fan-in from local and external research notes | complete |

## Preserved Research Notes

- `research/external-agent-workflow-practices.md`: source-backed external comparison and patterns.
- `research/current-workflow-pain-map.md`: repo-specific quality-carrier and overhead map.

## Fan-In Summary

The must-answer-now questions are handled:
- External systems keep planning, tasking, state, and verification, but use quick modes, optional gates, persistent state files, delta specs, and executable plans to avoid forcing every phase onto every task.
- The current repo's strongest quality carriers are not the number of files. They are decision ownership, context preservation, explicit design questions, executable task slices, and fresh validation evidence.
- The main overhead comes from mandatory phase-control symmetry, split design documents for small bounded changes, repeated status fields, and challenge gates that are too often locally waived in practice.
- The audit-driven synthesis strengthens the target model: `lean local` is the bounded non-trivial default, `tasks.md` is the main lean execution artifact, and inline `Risk Challenge` replaces formal challenge lanes only when no full-orchestrated trigger exists.

## Evidence Limits

- Public docs reflect current stated workflows, not independent empirical success measurements.
- GSD source docs are fast-moving; claims here are limited to docs available on 2026-05-17.
- The local pain map uses one complete existing bundle, `specs/railway-auto-migrations/`, as the concrete repo example. More historical bundles may reveal additional patterns once this repo accumulates them.

## Stop Rule

Do not implement workflow documentation, skills, templates, or agent contract changes in this session. The research phase is complete when the research notes and `spec.md` give the user a decision-ready proposal.

## Next Action

Specification proposal review by the user. If approved, open implementation planning against the exact repo surfaces listed in `spec.md`.
