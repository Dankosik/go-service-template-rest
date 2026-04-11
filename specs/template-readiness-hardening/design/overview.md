# Template Readiness Hardening Design Overview

## Chosen Approach

Use narrow local fixes that make existing template conventions harder to miss:
- align the OpenAPI guard test with the existing Makefile target by renaming the test,
- document protected endpoint placement where endpoint authors already look,
- make the SQLC sample demonstrate a bounded limit,
- move Redis mode/readiness policy into config-owned helpers consumed by both validation and bootstrap.

The design intentionally avoids large abstractions. These are template-hardening fixes, not a new feature framework.

## Artifact Index

- `component-map.md`: affected package and file surfaces.
- `sequence.md`: implementation order and proof sequence.
- `ownership-map.md`: source-of-truth and dependency-boundary decisions.

No conditional data-model, contract, rollout, or separate test-plan artifact is required. The only data-access change is sample repository validation; no schema or migration changes are planned.

## Readiness Summary

Planning-ready. The implementation session can proceed from `plan.md` and `tasks.md` without reopening architecture, API, data, or config ownership decisions.

## Reopen Conditions

Reopen specification/design if implementation discovers:
- adding OpenAPI 401/403 components is necessary and would change generated artifacts or contract semantics beyond documentation,
- Redis mode handling needs a broader dependency criticality redesign,
- the ping history sample limit has external contract expectations not visible in current repository tests.
