# Ownership map

## Source-of-truth ownership

- `internal/config` owns runtime config snapshot construction, config source precedence, config parse and validation sentinels, secret-source policy, and config helper normalization.
- `cmd/service/internal/bootstrap` owns service composition, startup/shutdown flow, dependency admission, network policy admission, and dependency initialization error taxonomy.
- `docs/configuration-source-policy.md` owns the current policy statement that `MongoProbeAddress` belongs to the guard-only config path.
- `internal/config/config_test.go` owns config package regression coverage.
- `cmd/service/internal/bootstrap/*_test.go` owns bootstrap dependency-init sentinel regression coverage.

## Dependency direction

- Bootstrap may import `internal/config`.
- `internal/config` must not import bootstrap or encode bootstrap lifecycle semantics.
- Config package tests may call unexported config helpers because they are same-package tests.
- Bootstrap tests may use an unexported bootstrap dependency-init sentinel because they are same-package tests.

## Rejected moves

- Do not move `MongoProbeAddress` to bootstrap in this task. That contradicts the current configuration source policy and would reopen a design decision without need.
- Do not use `net.LookupPort` for config validation. It would allow service names and rely on OS service databases, which is too loose for this deterministic config contract.
- Do not remove float input support from integer parsing. Existing tests cover mixed numeric inputs; keep the accepted shape and make it safe.
- Do not solve the symlink-policy ambiguity in this task. It touches security policy rather than idiomatic cleanup.

## Reopen conditions

Reopen specification or design before coding if:

- a non-bootstrap package depends on `config.ErrDependencyInit`;
- operators require service-name ports instead of numeric ports;
- local symlink behavior must change;
- `MongoProbeAddress` ownership is intentionally revisited.
