# 00 Input

## User Request
- Migrate service configuration loading to `koanf` with explicit, deterministic precedence.
- Keep template behavior simple: no migration windows, no key-translation layer, no phased cutover.
- Use only canonical namespace keys: `APP__<DOMAIN>__<FIELD>`.

## Scope
- Config loading path in runtime.
- Validation/strictness behavior.
- Startup telemetry for config lifecycle.
- Docs and examples for environment variables.

## Explicit Decision
- Flat non-canonical env keys are not supported.
- Runtime behavior is namespace-only.
