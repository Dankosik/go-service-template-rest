# Decision Placement And Artifact Ownership

Load this file when a spec draft mixes decisions with evidence, design detail, implementation tasks, or review transcripts.

Use it to preserve the repository's single-source discipline: `spec.md` owns final decisions, not every useful fact.

## Placement Rule Of Thumb

| Content | Home |
|---|---|
| final behavior, scope, accepted constraints | `spec.md` |
| raw comparisons, source links, benchmarks, transcript evidence | `research/*.md` |
| component map, sequence, ownership, source-of-truth design | `design/` |
| execution order, phase breakdown, task IDs | `plan.md` and `tasks.md` |
| unresolved assumptions or blockers | `spec.md` `Open Questions / Assumptions` |

## Good: Decision-Level Spec Snippet

```markdown
## Decisions
- The import endpoint remains synchronous for files below the existing request-size limit.
- Duplicate external IDs are rejected with the existing validation-error response shape.
- Audit events are emitted only after the import transaction commits.

## Validation
- API tests cover duplicate external IDs and transaction rollback.
- Audit-event tests prove no event is emitted for rejected imports.
```

Why this works: the spec fixes acceptance semantics and proof hooks without describing SQL statements, handler call order, or task IDs.

## Bad: Evidence Dump In `Decisions`

```markdown
## Decisions
- Researcher A said option 2 looked safer.
- The database package has files `store.go`, `tx.go`, and `queries.go`.
- Task 1: edit the handler. Task 2: edit repository tests.
- Use the following transaction sequence: begin, insert rows, insert audit, commit.
```

Why this fails: evidence belongs in `research/`, file topology belongs in design if it matters, and execution sequence belongs in planning.

## Good: Rejected Alternative

```markdown
## Decisions
- Reject asynchronous import for this change because the current API contract is synchronous and no queue ownership exists in the approved scope.
```

Why this works: the rejected path is stable enough to prevent planning drift, but it does not become a design section.

## Foreign-Template Translation Examples

| Foreign section | Repo-native placement |
|---|---|
| "Architecture overview" | If it decides behavior, summarize in `Decisions`; otherwise move to `design/overview.md`. |
| "Research findings" | Preserve in `research/*.md`; put only the chosen outcome in `Decisions`. |
| "Implementation plan" | Move to `plan.md`; keep only a link or compact summary in `Plan Summary / Link`. |
| "Traceability matrix" | Keep proof-impacting expectations in `Validation`; use `tasks.md` or planning artifacts for task traceability when needed. |

## Exa / External Source Links

Exa MCP was attempted before authoring (`web_search_exa` and `web_fetch_exa`) but returned a 402 credits-limit error. The links below were gathered with browser fallback and are calibration only; repository artifact ownership remains authoritative.

- IEEE SA, "IEEE 830-1984: IEEE Guide for Software Requirements Specifications": https://standards.ieee.org/ieee/830/1220/
- NASA, "Appendix C: How to Write a Good Requirement": https://www.nasa.gov/reference/appendix-c-how-to-write-a-good-requirement/
- Frattini et al., "Requirements Quality Research: a harmonized Theory, Evaluation, and Roadmap": https://arxiv.org/abs/2309.10355
