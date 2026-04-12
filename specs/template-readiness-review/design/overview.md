# Design Overview

## Chosen Approach

Use a bounded, non-breaking template-hardening pass. The implementation should strengthen documentation, examples, and guardrails that teach the correct production business-code path, while avoiding speculative runtime abstractions or fake business domains.

The selected approach is:

- document the production-shaped first-feature path rather than adding a demo business domain;
- keep `ping_history` as the SQLC fixture for now but label it more strongly;
- clarify Redis/Mongo guard-only semantics without adding adapters;
- clarify protected-operation wiring without designing auth;
- add narrow guardrails for manual route registration and app/infra import direction;
- canonicalize HTTP `Allow` header behavior while route policy tests are already in scope;
- fix one concrete startup rejection log drift.

## Artifact Index

- `component-map.md`: affected files and stable surfaces.
- `sequence.md`: implementation order and proof sequence.
- `ownership-map.md`: source-of-truth and dependency ownership rules.

Conditional artifacts not created:

- `data-model.md`: no schema or persisted-state change is selected.
- `dependency-graph.md`: no dependency direction change is selected; the app/infra guardrail preserves the existing direction.
- `contracts/`: no API or generated contract change is selected.
- `test-plan.md`: validation obligations fit in `plan.md` and `tasks.md`.
- `rollout.md`: no runtime rollout, migration rollout, or compatibility choreography is selected.

## Readiness

Design is stable for implementation planning with `CONCERNS`:

- Do not add fake domain behavior.
- Do not rename `ping_history` schema/query/generated surfaces without maintainer approval.
- Do not invent auth behavior.
