# internal/config Review Fix Handoff Spec

## Context

The completed read-only review of `internal/config` found a small set of correctness and maintainability issues around config source precedence, accepted-key ownership, Mongo probe address validation, and config-example drift.

The relevant repository contracts are:

- `internal/config` owns building one validated immutable runtime config snapshot from defaults, config files, env, and flags.
- `docs/configuration-source-policy.md` says runtime precedence is last-wins: code defaults, base config file, overlays, then `APP__...` env variables.
- `env/config/*.yaml` is the non-secret baseline config surface; `APP__...` env is the override and secret-value channel.
- Redis and Mongo remain guard-only extension stubs; config may validate and derive probe-ready addresses, but runtime adapter behavior belongs outside `internal/config`.

## Scope / Non-goals

In scope for the later implementation:

- Tighten Mongo probe host validation without changing valid Mongo URI cases currently covered by tests.
- Make `APP__...` empty-value precedence explicit and consistent with the selected policy.
- Separate accepted config keys from `defaultValues()` so defaults are not the accidental strict-mode schema registry.
- Repair the missing `otlp_traces_endpoint` baseline YAML key.
- Replace the file-policy boolean mode with a named package-local policy type if implementation touches the file-loader path.
- Add focused regression coverage for each changed behavior.

Out of scope:

- No runtime Redis or Mongo adapter behavior.
- No OpenAPI, database migration, telemetry pipeline, or bootstrap lifecycle redesign.
- No change to config key names or env namespace prefix.
- No change to `ErrorType` fallback unless a separate observability/product decision asks for a new `"unknown"` label.
- No implementation in this handoff session.

## Decisions

1. **Mongo probe validation will fail closed for empty or malformed hosts.**
   - Preserve valid cases: host with explicit numeric port, bare DNS host defaulting to `27017`, unbracketed IPv6 defaulting to `27017`, bracketed IPv6 defaulting to `27017`, and `mongodb+srv://` host defaulting to `27017`.
   - Reject empty hosts after `net.SplitHostPort`, including `mongodb://:27017/app`.
   - Replace `strings.Trim(trimmed, "[]")` behavior with balanced bracket handling using exact prefix/suffix stripping.
   - Reject `[]`, unmatched bracket forms, and hosts with stray brackets that cannot be parsed by `net.SplitHostPort`.

2. **`APP__...` env values should be real last-wins overrides, including empty strings.**
   - The review found code-policy drift: `collectNamespaceValues` currently drops empty env values, while docs say env is the final precedence layer.
   - The intended fix is to stop skipping empty env values in `collectNamespaceValues`; explicit empty env for required fields should flow into `buildSnapshot`/`validateConfig` and fail fast.
   - Keep `lookupNonEmptyEnv` behavior for environment intent hints and allowed-roots handling; this helper intentionally treats empty control hints as absent.
   - Update tests that use empty env values only as a cleanup mechanism so they truly unset or restore env vars instead of relying on loader skip behavior.
   - Clarify in `docs/configuration-source-policy.md` that an empty `APP__...` value is still an explicit override and may be rejected by parse or validation rules.

3. **Known config keys should be owned by the typed config schema, not by defaults.**
   - `knownConfigKeys()` should derive from `Config` `koanf` tags or an explicit schema registry; deriving it from `defaultValues()` makes every valid key require a default placeholder.
   - Prefer deriving from `Config` tags in package code so strict-mode accepted keys track the typed snapshot shape.
   - Keep defaults as a source of baseline values only. Tests should assert defaults are a subset of known typed keys, not equal to the accepted-key registry.
   - Preserve the existing sentinel snapshot test shape, but update it so it still proves every typed leaf is mapped into `buildSnapshot`.

4. **The baseline YAML must include `observability.otel.exporter.otlp_traces_endpoint`.**
   - Add `otlp_traces_endpoint: ""` beside `otlp_endpoint`, `otlp_headers`, and `otlp_protocol` in `env/config/default.yaml`.
   - Add or adjust a test so future keys in `defaultValues()` are not silently omitted from `env/config/default.yaml` when they belong to the YAML baseline.

5. **File config security mode should be named, not passed as a raw boolean.**
   - `localEnvironment bool` means “local file policy” versus “hardened non-local file policy.”
   - If this code path is touched during implementation, replace the raw bool with a package-local mode such as `configFilePolicyLocal` and `configFilePolicyHardened`.
   - This change must be behavior-preserving. If it changes local/non-local security semantics, reopen security/design review instead of treating it as cleanup.

6. **Do not change `ErrorType` fallback in this bundle.**
   - The review noted that unknown errors fall back to `"load"`.
   - Current call sites use `ErrorType` for errors returned by `LoadDetailedWithContext`, and the existing labels are consumed by bootstrap metrics/logging.
   - Changing the fallback to `"unknown"` would alter metric label semantics without a clear current bug. Leave it alone unless a separate observability decision approves the label change.

## Open Questions / Assumptions

- Assumption: empty `APP__...` values should be explicit overrides because that matches the published last-wins policy and fails fast for required settings.
- Assumption: deriving known keys from `Config` tags is preferable to maintaining an explicit duplicate registry.
- Assumption: `env/config/local.yaml` may remain a sparse overlay and does not need every default key.
- No user decision is currently required before implementation.

## Plan Summary / Link

Use `plan.md` and `tasks.md` in this directory for the future implementation session. The recommended implementation order is:

1. Add regression tests that expose the intended semantics.
2. Fix Mongo host normalization and file-policy naming.
3. Fix env precedence and test env cleanup helpers.
4. Split accepted-key registry from defaults.
5. Update YAML/docs and run the validation commands.

Full review-point traceability is recorded in `research/review-point-coverage.md`.

## Validation

Future implementation should prove:

- `go test ./internal/config` passes.
- Targeted Mongo tests reject empty and malformed hosts while preserving valid host normalization.
- Empty `APP__HTTP__ADDR=` or a similar required key no longer silently falls back to defaults.
- Strict-mode unknown-key validation still rejects truly unknown keys and accepts every typed key.
- `env/config/default.yaml` contains every default-backed YAML baseline key, including `observability.otel.exporter.otlp_traces_endpoint`.
- A broader `go test ./...` is recommended after the config-loader behavior change because bootstrap consumes `config.ErrorType`, `LoadDetailedWithContext`, and the typed snapshot.

## Outcome

Implemented on 2026-04-12 from `tasks.md` T001-T008.

Fresh validation evidence:

- `go test ./internal/config` passed with 116 tests.
- `go test ./cmd/service/internal/bootstrap` passed with 91 tests.
- `go test ./...` passed with 338 tests across 11 packages.
