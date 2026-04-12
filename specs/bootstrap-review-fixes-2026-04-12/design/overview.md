# Design Overview

## Chosen Approach

This task is a focused bootstrap maintainability fix. It keeps the existing bootstrap lifecycle and dependency admission design intact while tightening two source-of-truth seams:

- degraded dependency startup status/logging should have one local owner in the bootstrap package,
- network policy environment parsing should be explicitly documented as an operator-policy exception to normal typed runtime config, not an accidental config bypass.

No new external package, interface, service, adapter, migration, or generated-code path is required.

## Artifact Index

- `component-map.md`: affected files and stable areas.
- `sequence.md`: implementation order and runtime behavior.
- `ownership-map.md`: source-of-truth and dependency ownership.
- `../plan.md`: execution strategy for the later implementation session.
- `../tasks.md`: executable task ledger.

## Key Design Decisions

- Add a same-package helper for degraded dependency status/logging instead of duplicating Redis and Mongo branches.
- Keep `NETWORK_*` outside `internal/config.Config`, but make that exception official in docs and examples.
- Make `networkPolicyErrorLabels` production-owned through structured bootstrap logging, or move it to test-only code if that production use is rejected during implementation.
- Remove redundant branch logic in `dependencyInitFailure`; preserve error wrapping behavior.

## Readiness Summary

The design is stable for planning. The only accepted implementation risk is the small choice inside the network-policy label fix: production logging use is preferred, but moving the helper into tests is the fallback if the implementation would otherwise widen logging helper APIs too much.
