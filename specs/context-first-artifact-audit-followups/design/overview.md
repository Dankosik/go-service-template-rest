# Design Overview

## Approach

Patch the central workflow documentation first, then mirror the same behavior in the session/deeper skill guidance that future LLM sessions actually load. Keep each edit small and field-oriented rather than introducing new templates.

## Artifact Index

- `design/component-map.md`: changed documentation and skill surfaces.
- `design/sequence.md`: edit and validation order.
- `design/ownership-map.md`: authority split preserved by the edits.
- `tasks.md`: executable docs/skills ledger for this pass.

Conditional artifacts:

- Supplemental strategy note: not expected; the work is one bounded docs/skills checkpoint.
- `test-plan.md`: not expected; validation obligations fit in `tasks.md`.
- `rollout.md`: not expected; no runtime deployment, migration, compatibility, or operator rollout behavior changes.

## Readiness

Planning and implementation can proceed in this lightweight-local session because the user explicitly asked to fix all concrete audit findings, the scope is docs/skills-only, and the changes preserve the existing artifact authority model.
