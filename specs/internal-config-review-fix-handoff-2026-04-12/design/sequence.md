# Sequence

## Runtime Config Loading After The Fix

1. `LoadDetailedWithContext` creates a load context and calls `loadKoanf`.
2. `loadKoanf` loads code defaults into Koanf.
3. If a base config file or overlays are present, the file-loader resolves the named file-policy mode:
   - local policy permits relative/local test paths.
   - hardened policy keeps the current non-local path, symlink, permission, and allowed-root checks.
4. `loadKoanf` collects every valid `APP__...` namespace entry from `os.Environ`, including entries whose value trims to empty.
5. Koanf loads collected namespace values last, preserving env as the final precedence layer.
6. `buildSnapshot` parses typed values:
   - explicit empty values for required strings reach validation and fail with `ErrValidate`;
   - explicit empty values for durations, ints, floats, or bools fail parse with `ErrParse`;
   - optional empty string values remain allowed when validation permits them.
7. `validateConfig` performs strict unknown-key checks using the typed-key registry rather than `defaultValues()`.
8. Mongo validation calls `MongoProbeAddress`; malformed empty or bracketed hosts now fail before bootstrap can use a bogus probe address.

## Future Implementation Order

1. Start with tests for the intended behavior changes:
   - empty env override does not silently fall back to defaults;
   - malformed Mongo host forms fail;
   - known config keys are not owned by defaults;
   - YAML baseline includes default-backed keys.
2. Implement Mongo normalization hardening.
3. Implement named config-file policy mode if touching the loader path.
4. Implement empty env collection and update test env cleanup helpers.
5. Implement typed-schema known-key derivation.
6. Update YAML and docs.
7. Run targeted and full validation.

## Failure Points To Preserve

- File-policy failures outside local mode must remain `ErrSecretPolicy` or `ErrLoad` according to current behavior.
- Malformed YAML must remain `ErrParse`.
- Invalid typed values must remain `ErrParse` before validation.
- Invalid domain constraints must remain `ErrValidate`.
- Secret source policy must continue rejecting non-empty secret-like YAML values.
