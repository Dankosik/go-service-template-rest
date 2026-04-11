# Clarification-Gate Reconciliation

Load this file after a `spec-clarification-challenge` pass returns questions or when reviewing whether non-trivial `spec.md` approval is legitimate.

The challenge is advisory. The orchestrator reconciles it, and `spec.md` stores only final resolved outcomes.

## Good: Reconciled Blocking Finding

Challenge finding:

```text
blocks_spec_approval: The spec does not decide whether failed webhook delivery is retried or surfaced only in logs.
next_action: answer_from_existing_evidence
```

Repo-native reconciliation:

```markdown
## Decisions
- Failed webhook delivery uses the existing retry policy owned by the outbound delivery component; this change does not introduce a new retry budget.

## Validation
- Tests prove the new webhook path delegates failures to the existing retry policy.
```

Why this works: the raw challenge is not pasted into the spec; the final decision and proof consequence are placed where downstream design needs them.

## Good: Reconciled With Deferral

```markdown
## Open Questions / Assumptions
- [defer_to_design] The exact retry metric label belongs in `design/overview.md` and observability design; the spec only decides that existing retry semantics are preserved.
```

Why this works: the spec keeps the behavior decision stable while routing implementation-shaped detail downstream.

## Bad: Transcript Dump

```markdown
## Clarification Challenge Transcript
Reviewer: What about webhook retry?
Orchestrator: Maybe existing retry?
Reviewer: Please check.
```

Why this fails: transcripts are not final decisions and force later phases to infer authority from conversation.

## Bad: Unreconciled Approval

```markdown
## Outcome
Spec approved.

## Open Questions / Assumptions
- Need to decide retry behavior later.
```

Why this fails: the unresolved item changes correctness and design, so non-trivial spec approval is blocked or must record an explicit accepted risk with proof consequences.

## Foreign-Template Translation Examples

| Foreign practice | Repo-native translation |
|---|---|
| "Append reviewer comments to the spec." | Keep comments out; write only reconciled outcomes in `Decisions`, `Open Questions / Assumptions`, or `Validation`. |
| "Use a sign-off checklist." | Record the clarification gate status in workflow-control artifacts; use `spec.md` only for final decisions and remaining assumptions. |
| "Ask the user every open question." | Answer from repository evidence first; ask the user only for true external product or business policy. |
| "Turn challenge output into tasks." | Do not create tasks in this pass; route implementation work to planning after approved spec and design. |

## Exa / External Source Links

Exa MCP was attempted before authoring (`web_search_exa` and `web_fetch_exa`) but returned a 402 credits-limit error. The links below were gathered with browser fallback and are calibration only; the repository clarification gate is the source of truth.

- Frattini et al., "Requirements Quality Research: a harmonized Theory, Evaluation, and Roadmap": https://arxiv.org/abs/2309.10355
- NASA, "Appendix C: How to Write a Good Requirement": https://www.nasa.gov/reference/appendix-c-how-to-write-a-good-requirement/
- IREB CPRE Online Glossary: https://cpre.ireb.org/en/downloads-and-resources/glossary
