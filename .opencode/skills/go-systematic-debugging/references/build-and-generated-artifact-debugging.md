# Build And Generated Artifact Debugging

## Behavior Change Thesis
When loaded for build or generated-artifact failures, this file makes the model prove selected files, tags, toolchain, and generator source of truth instead of changing runtime logic or hand-editing generated output.

## When To Load
Load when `go build`, `go test` compilation, `go vet`, code generation, build tags, `GOOS` or `GOARCH`, `CGO_ENABLED`, module or workspace state, embedding, or generated files may explain the failure.

## Decision Rubric
- Classify the failure stage first: build, vet, test compilation, generation, or runtime test execution.
- Start with the narrow package build or test compile before runtime debugging.
- Use `go list` to prove which files are included, ignored, embedded, or tag-selected.
- Use `go generate -n -x` to inspect the generator command before mutating files.
- Regenerate from the source-of-truth input; do not hand-edit generated output unless the repo explicitly treats it as maintained source.
- Clear caches only as final confirmation, not as the first diagnostic.

## Imitate

```bash
go env GOVERSION GOOS GOARCH CGO_ENABLED GOMOD GOWORK
go list -f '{{.GoFiles}} {{.IgnoredGoFiles}}' -tags 'ci' ./internal/api
go build -json -tags 'ci' ./internal/api
go generate -run 'oapi' -n -v -x ./internal/api
go generate -run 'oapi' -v -x ./internal/api
go test ./internal/api -run '^$' -tags 'ci' -count=1
```

Copy the order: prove environment and selected source set, inspect generation, then mutate through the generator if needed.

## Reject

```bash
go generate ./...
```

This is too broad before the owning directive is known and can create noisy, unrelated diffs.

```text
Fixed by editing api.gen.go directly.
```

This leaves the generator input stale and sets up the next regeneration to reintroduce the failure.

## Agent Traps
- Assuming `go generate` runs automatically during `go build` or `go test`.
- Ignoring CI `GOOS`, `GOARCH`, `CGO_ENABLED`, `GOWORK`, or build tags.
- Hiding a compile error behind a new build tag instead of fixing the selected source set.
- Deleting build caches before preserving the original signal.
- Reporting only the final package summary instead of the first compiler error and import path.

## Validation Shape
Capture the exact package, build tags, toolchain/env values, first compiler or generator error, included/ignored file evidence, generator command, generated diff when relevant, and the narrow build or test-compile command that now passes.
