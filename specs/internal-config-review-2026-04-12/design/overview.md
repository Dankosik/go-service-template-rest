# Design overview

## Chosen approach

Use small, local corrections rather than a config-system rewrite. The fix set should make the existing package contracts sharper:

- parsing helpers own safe type coercion;
- validation helpers own semantic ranges and host-port constraints;
- `internal/config` owns only config error taxonomy;
- bootstrap owns dependency initialization taxonomy;
- duplicated normalization and env-name mapping get one package-local source of truth.

## Artifact index

- `component-map.md`: affected files and responsibilities.
- `sequence.md`: implementation and runtime sequences.
- `ownership-map.md`: source-of-truth boundaries and rejected moves.

No conditional design artifacts are required: there is no API contract change, persisted data change, migration, cache contract, or rollout choreography.

## Design notes

- The package should keep the existing public loader shape: `Load`, `LoadWithOptions`, `LoadDetailed`, and `LoadDetailedWithContext`.
- The implementation may add unexported helpers in `internal/config` and `cmd/service/internal/bootstrap`.
- The implementation may remove `config.ErrDependencyInit`; update bootstrap code/tests to use a bootstrap-owned sentinel instead.
- The implementation should not move `MongoProbeAddress`; the repository's configuration source policy documents it as part of the current guard-only config path.
- The symlink-policy observation is left as a deferred reopen item, not an implicit coding task.

## Readiness

The design is stable for planning. Reopen specification only if the implementation discovers that service-name ports must be supported, `ErrDependencyInit` must stay exported for a non-bootstrap consumer, or local symlink behavior must change.
