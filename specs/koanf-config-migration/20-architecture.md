# 20 Architecture

## Decision Summary
- Use `koanf` as a single loader pipeline.
- Canonical configuration surface is namespace-only (`APP__...`).
- No key-translation layer.

## Loader Topology
1. Load in-memory defaults.
2. Merge optional base YAML (`--config`).
3. Merge optional overlays (`--config-overlay`, ordered).
4. Merge namespace env (`APP__...`).
5. Build typed snapshot.
6. Run validation (strict/permissive unknown keys + semantic constraints).

## Key Properties
- Deterministic precedence and deterministic output.
- Explicit stage timing for observability:
  - `config.load.defaults`
  - `config.load.file`
  - `config.load.env`
  - `config.parse`
  - `config.validate`
- No hidden compatibility behavior.

## Rationale
Template project should optimize for simplicity and predictability over compatibility migration complexity.
