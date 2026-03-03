# 10 Context Goals And Non-Goals

## Context
This repository is a template service with no external client fleet that requires backward compatibility for old flat env keys.

## Goals
1. Deterministic config precedence: defaults -> file -> overlays -> namespace env.
2. Single canonical env contract: `APP__...`.
3. Fail-fast startup on invalid critical config.
4. Strict mode rejects unknown canonical keys.
5. Keep implementation and docs aligned.

## Non-Goals
- Support or deprecate non-canonical flat keys.
- Maintain phased compatibility states for config keys.
- Add runtime hot-reload.

## Constraints
- Keep public API stable: `Load() (Config, error)`.
- Keep startup behavior explicit and observable.
