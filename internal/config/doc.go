// Package config owns this service's configuration: the schema, the defaults,
// the load pipeline, parsing, validation, and the immutable snapshot a
// composition root reads. It owns no feature behavior and constructs no adapter.
//
// # Loading
//
// [LoadDetailedWithContext] is the entry point, and [LoadDetailed] is the same
// load without a caller context. The Stage constants in config.go name the steps
// a failure is reported against, in the order they run:
//
//   - StageLoadDefaults — defaults.go merges each section's own defaults over the
//     handful it declares inline.
//   - StageLoadFile — load_file.go admits the path and its overlays, and
//     secret_policy.go decides what a file-sourced value is allowed to carry.
//   - StageLoadEnv — load_environment.go maps one namespaced variable onto one
//     config key.
//   - StageParse — snapshot.go decodes the merged map into [Config], and parse.go
//     owns the scalar conversions under it, one per kind, so a bad duration and a
//     bad integer fail the same way.
//   - StageValidate — validate.go runs each section's own validator and then the
//     rules that hold only between sections.
//
// load_koanf.go is the merge those stages run inside. schema.go is what
// load_file.go and load_koanf.go both need and neither owns: which sections
// exist, and the removal of a scalar written where a section belongs. context.go
// is the cancellation check between stages, so a startup budget can stop a load
// rather than only measure it. errors.go classifies a load failure for the
// composition roots that report one.
//
// load_flags.go is not part of the pipeline. It builds [LoadOptions] from
// command-line flags before any of it runs.
//
// # Sections
//
// A section's type, defaults, and validation live together in its own
// <section>_config.go. Those three change together — adding a field touches all
// three — and a section a build profile removes then leaves with its file
// instead of being cut out of three shared ones. types.go and defaults.go keep
// the [Config] shape, the merge, and the sections small enough to read in place.
//
// A rule that spans two sections goes to whichever section depends on the other.
// validate.go keeps only the rules that belong to neither.
//
// # What it may not import
//
// This package may not import a runtime adapter, so a rule that this package and
// an adapter must agree on can live in neither. It goes to a pure leaf both
// import, and a parity test pins the two together.
// profile:authn-oidc-jwt:start
// internal/authntrust is that leaf for the issuer, JWKS, and token-profile trust
// rules.
// profile:authn-oidc-jwt:end
// internal/observability/otelconfig is that leaf for the OpenTelemetry sampler
// vocabulary and validation.
package config
