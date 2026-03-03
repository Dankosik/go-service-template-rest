# 60 Implementation Plan

## WP-1 Loader Core
- Build koanf pipeline with precedence:
  - defaults -> base file -> overlays -> namespace env.
- Remove key-translation layer entirely.

## WP-2 Validation
- Keep strict/permissive unknown-key behavior for canonical keys.
- Enforce required-if-enabled and value ranges.
- Enforce secret-source policy for YAML.

## WP-3 Runtime Wiring
- Keep config lifecycle stage spans/metrics without translation stage.
- Keep startup summary and failure classification stable.

## WP-4 Tests
- Update tests to namespace-only contract.
- Remove compatibility matrix for non-canonical key translation and related conflict assertions.
- Add explicit tests that flat non-canonical keys are ignored.

## WP-5 Docs
- Update `README.md` and `env/.env.example` to namespace-only examples.
- Remove migration narrative for non-canonical key support.
