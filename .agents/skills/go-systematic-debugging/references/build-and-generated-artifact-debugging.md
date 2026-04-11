# Build And Generated Artifact Debugging

## When To Load
Load this reference when `go build`, `go test`, `go vet`, code generation, build tags, `GOOS` or `GOARCH`, `CGO_ENABLED`, module or workspace state, embedding, or generated files may explain the failure.

Use it before changing runtime logic.

## Commands
Start narrow:

```bash
go build ./path/to/pkg
go test ./path/to/pkg -run '^$' -count=1
go test ./path/to/pkg -run '^TestName$' -count=1 -v
go env GOVERSION GOOS GOARCH CGO_ENABLED GOMOD GOWORK GOPATH GOROOT
```

Inspect package selection and file inclusion:

```bash
go list -json ./path/to/pkg
go list -f '{{.GoFiles}} {{.IgnoredGoFiles}} {{.CgoFiles}} {{.EmbedFiles}}' ./path/to/pkg
go list -f '{{.GoFiles}} {{.IgnoredGoFiles}}' -tags 'integration' ./path/to/pkg
go list -deps ./path/to/pkg
```

When the toolchain supports structured build output:

```bash
go build -json ./path/to/pkg
go test -json ./path/to/pkg -run '^$'
```

When generated artifacts are involved:

```bash
go generate -n -v -x ./path/to/pkg
go generate -run 'stringer|mockgen|oapi' -n -v -x ./path/to/pkg
go generate -run 'stringer|mockgen|oapi' -v -x ./path/to/pkg
git diff -- . ':!vendor'
```

Use cache clearing only as a final confirmation, not as the first diagnostic:

```bash
go clean -cache -testcache
go build ./path/to/pkg
```

## Evidence To Capture
- exact failing package and whether the failure happens during build, vet, test compilation, generation, or runtime test execution
- Go version, `GOOS`, `GOARCH`, `CGO_ENABLED`, build tags, `GOMOD`, and `GOWORK`
- generated command, generator version, directive, and generated file diff
- `go list` evidence for included and ignored files
- first compiler error and import path, not only the final package summary
- whether a generated file is source-of-truth output or a hand-maintained file

## Bad Debugging Moves
- hand-editing generated output while leaving the generator input stale
- running broad `go generate ./...` before identifying the owning directive
- deleting caches first and losing the original signal
- ignoring build constraints or assuming local `GOOS/GOARCH` matches CI
- hiding a compile error with build tags instead of fixing the selected source set
- assuming `go generate` runs automatically as part of `go build` or `go test`

## Good Debugging Moves
- reproduce the narrow package build first
- use `go list` to prove which files are included or ignored
- use `go generate -n -x` to inspect commands before mutating generated outputs
- regenerate from the source-of-truth input and review the generated diff
- confirm whether generated files must be checked in for consumers
- use structured JSON output when the failure is noisy or multi-package

## Example Debugging Flow
For a CI-only compile failure in generated code:

```bash
go env GOVERSION GOOS GOARCH CGO_ENABLED GOMOD GOWORK
go list -f '{{.GoFiles}} {{.IgnoredGoFiles}}' -tags 'ci' ./internal/api
go build -json -tags 'ci' ./internal/api
go generate -run 'oapi' -n -v -x ./internal/api
go generate -run 'oapi' -v -x ./internal/api
go test ./internal/api -run '^$' -tags 'ci' -count=1
```

Interpretation:
- if `go list` changes selected files under CI tags, fix the constraint or tag expectation
- if generation changes tracked files, fix the generator input or commit the generated artifact according to repo policy
- if build JSON points at an import path mismatch, fix package ownership rather than runtime behavior

## Source Links
- [cmd/go package](https://pkg.go.dev/cmd/go)
- [go command build constraints](https://pkg.go.dev/cmd/go#hdr-Build_constraints)
- [go command build JSON](https://pkg.go.dev/cmd/go#hdr-Build__json_encoding)
- [go command generate](https://pkg.go.dev/cmd/go#hdr-Generate_Go_files_by_processing_source)
- [go/build/constraint package](https://pkg.go.dev/go/build/constraint)
- [Go blog: generating code](https://go.dev/blog/generate)
