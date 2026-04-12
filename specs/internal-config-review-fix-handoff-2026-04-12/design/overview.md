# Design Overview

## Chosen Approach

Keep the fix package-local and source-of-truth oriented:

- `internal/config` continues to own loading, parsing, validation, and typed snapshot construction.
- No runtime dependency adapter behavior moves into `internal/config`.
- Config precedence becomes explicit: all `APP__...` entries that map to valid config keys participate as final overrides, including empty strings.
- The accepted-key registry moves away from `defaultValues()` and toward the typed `Config` schema.
- Example drift is fixed at the source surface, `env/config/default.yaml`, and protected by tests.

## Artifact Index

- `spec.md`: stable decisions and scope.
- `design/component-map.md`: affected files and responsibilities.
- `design/sequence.md`: future implementation and runtime behavior sequence.
- `design/ownership-map.md`: source-of-truth and boundary rules.
- `plan.md`: future implementation phase strategy.
- `tasks.md`: executable future task ledger.

## Rejected Options

- Do not keep empty env skipping and merely document it. That would preserve a hidden fail-open behavior where explicit empty required settings fall back to lower-precedence defaults.
- Do not keep `knownConfigKeys()` derived from `defaultValues()`. That keeps defaults as a hidden schema registry and conflicts with the documented default-addition rule.
- Do not replace Mongo URI parsing with a full Mongo driver parser in this task. The package only needs deterministic probe-address extraction for guard-only validation, not runtime adapter semantics.
- Do not change `ErrorType` fallback in this bundle. That is an observability-label decision, not required for the reviewed config correctness fixes.

## Readiness

This design is ready for lightweight implementation planning. The only policy-visible change is empty `APP__...` handling, and the selected behavior is recorded in `spec.md`.
