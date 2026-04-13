# Go Code Quality Automation Research

## Question

How well does this reusable Go service template enforce Go code quality through local commands, CI, scripts, pinned tooling, and beginner-friendly workflows, and what practical improvements should be considered next?

## Current Strengths

- `make check` is a beginner-friendly quick path: it runs `fmt-check`, `lint`, and `test`, with Docker fallback when local Go is missing (`Makefile:93-108`).
- Formatting is strong for a Go template: `fmt-check` verifies both `goimports` and `gofumpt`, while excluding generated outputs from `gofumpt` (`Makefile:5-6`, `Makefile:185-197`).
- Linting is pinned and schema-checked: `.golangci.yml` uses v2 config with `default: standard` plus semantic/resource linters, and `make lint` runs `go tool golangci-lint config verify` before `run` (`.golangci.yml:1-23`, `Makefile:293-295`).
- `go tool golangci-lint linters` confirms the configured set includes `bodyclose`, `contextcheck`, `errcheck`, `errorlint`, `gocritic`, `govet`, `ineffassign`, `nilerr`, `nilnil`, `nolintlint`, `predeclared`, `sqlclosecheck`, `staticcheck`, `thelper`, `unconvert`, `unused`, and `wastedassign`.
- The Go toolchain and tools are reproducible: `go.mod` declares Go `1.26.2` and pins tools including `golangci-lint`, `goimports`, `gofumpt`, `gotestsum`, `govulncheck`, `gosec`, `gitleaks`, `sqlc`, `mockgen`, `stringer`, `validate`, and `oasdiff` (`go.mod:3`, `go.mod:505-519`).
- `make check-full` prefers `docker-ci`, and `docker-ci` is the closest zero-setup CI parity path (`Makefile:110-117`, `Makefile:460`; `scripts/dev/docker-tooling.sh:515-544`).
- CI enforces repo integrity, lint, unit test plus vet, race, coverage/report artifacts, integration, migration validation, Go security, secret scan, and container scan (`.github/workflows/ci.yml:21-398`).
- Nightly keeps heavier checks out of the everyday loop: repeated tests, fuzz smoke, race, integration, OpenAPI, Go security, and image scan (`.github/workflows/nightly.yml:41-86`).
- Branch protection expectations are explicit and guarded: the helper requires named CI contexts, and the guardrails script checks the helper/workflow alignment (`scripts/dev/configure-branch-protection.sh:39-64`, `scripts/ci/required-guardrails-check.sh:102-118`).

## Gaps And Weak Spots

- Hidden zero-setup bug: `scripts/dev/docker-tooling.sh guardrails-check` runs the host `scripts/ci/required-guardrails-check.sh`, and that script calls `go list`. With `PATH=/usr/bin:/bin`, the Docker wrapper failed with `go: command not found`, so `docker-ci` still leaks a host-Go dependency through `guardrails-check` (`scripts/dev/docker-tooling.sh:491-492`, `scripts/dev/docker-tooling.sh:516-518`, `scripts/ci/required-guardrails-check.sh:65-66`).
- `make check-full` falls back to native `ci-local` when Docker is unavailable; that path later skips Docker-backed integration, migration rehearsal, and container scan with a message. Docs disclose this, but the command name can still imply stronger parity than it proves (`Makefile:110-117`, `Makefile:313-321`, `docs/build-test-and-development-commands.md:639-644`).
- There is no short advertised Docker quick-check command equivalent to `make check`; users can run `docker-fmt-check`, `docker-lint`, and `docker-test`, but there is not a single `make docker-check` target (`Makefile:25-33`, `Makefile:199-203`, `Makefile:272-285`, `Makefile:323-324`).
- Fuzz smoke is wired but currently has no fuzz targets. A targeted search found 26 test files and 0 `func Fuzz` targets, so the gate currently proves graceful skip behavior rather than parser/property coverage (`Makefile:242-253`, `.github/workflows/nightly.yml:56-57`).
- Nightly repeated tests use `go test -count=5 ./...` but not `-shuffle=on`; `go help testflag` confirms `-shuffle=on` randomizes test order (`.github/workflows/nightly.yml:53-54`).
- The PR template asks for `fmt-check`, `lint`, `test`, and conditional race/integration, but not `make check-full`, `make test-report`, or coverage artifact/CI coverage evidence (`.github/pull_request_template.md:12-25`).
- Complexity/style linters need calibration before gating: default `cyclop/gocyclo/nestif` produced 44 issues, `funlen/gocognit` produced 45 issues, `revive` produced 50 issues, and `paralleltest/tparallel/usetesting` produced 67 issues in local candidate probes.

## Candidate Linter Evidence

Baseline command evidence from this session:

- `go tool golangci-lint config verify`: passed with no output.
- `go tool golangci-lint version`: v2.10.1 built with Go 1.26.2.
- `make check`: passed; lint reported `0 issues` and `go test ./...` passed.

Low-noise candidate probes that returned `0 issues`:

- `durationcheck`, `rowserrcheck`, `nilnesserr`, `nosprintfhostport`, `usestdlibvars`, `exptostd`, `makezero`
- `asciicheck`, `bidichk`, `dupword`, `misspell`, `gocheckcompilerdirectives`, `gomoddirectives`
- `loggercheck`, `promlinter`

Useful but non-zero candidate probes:

- `noctx`, `errchkjson`, `forcetypeassert`, `copyloopvar`, `canonicalheader`: 16 issues. Some are useful cleanup candidates (`noctx`, `errchkjson`), while others look churny or policy-sensitive (`copyloopvar`, `canonicalheader`, `forcetypeassert` in tests).
- `cyclop`, `gocyclo`, `nestif`: 44 issues. This is too noisy as a default beginner gate without thresholds or cleanup.
- `funlen`, `gocognit`: 45 issues. Too noisy for default gating.
- `revive`: 50 issues, mostly exported-symbol/package comment findings. Too noisy for default gating.
- `paralleltest`, `tparallel`, `usetesting`: 67 issues. Too opinionated for a beginner-friendly default.

## Recommendations

### Do Now

1. Fix the Docker zero-setup guardrails leak.
   - Likely files: `scripts/dev/docker-tooling.sh`, possibly `scripts/ci/required-guardrails-check.sh`.
   - Run location: `docker-guardrails-check`, `docker-ci`, and `make check-full` when Docker is present.
   - Benefit: restores the documented zero-setup promise for users without local Go.
   - Cost/noise risk: low; this is workflow plumbing rather than stricter code policy.

2. Add a small zero-baseline linter bundle to `.golangci.yml`.
   - Candidate bundle: `durationcheck`, `nilnesserr`, `rowserrcheck`, `gocheckcompilerdirectives`, `bidichk`, `loggercheck`, `promlinter`.
   - Optional after policy choice: `nosprintfhostport`, `gomoddirectives`.
   - Run location: normal `make lint`, therefore `make check`, `make check-full`, CI lint, and Docker lint.
   - Benefit: catches duration math mistakes, nil/error contract issues, row-iteration errors, invalid compiler directives, dangerous Unicode, structured logging mistakes, and Prometheus metric naming issues.
   - Cost/noise risk: low based on local zero-issue candidate probes, but still needs user approval because it expands the default gate.

3. Add `-shuffle=on` to the nightly repeated test run.
   - Likely file: `.github/workflows/nightly.yml`.
   - Run location: nightly only, not `make check` or required PR CI.
   - Benefit: catches hidden test order/global-state coupling without slowing the beginner daily loop.
   - Cost/noise risk: low to moderate; failures would be nightly-only and Go prints the seed.

4. Improve beginner-facing evidence prompts.
   - Likely files: `.github/pull_request_template.md`, `docs/build-test-and-development-commands.md`, possibly `CONTRIBUTING.md`.
   - Suggested content: mention `make check-full` before PRs and `make test-report`/CI `test-coverage` evidence when coverage matters.
   - Benefit: less-experienced users are more likely to provide the same evidence CI expects.
   - Cost/noise risk: low; documentation/template only.

### Maybe Later

- Add `make docker-check` as a quick pinned path for `docker-fmt-check`, `docker-lint`, and `docker-test`.
  - Benefit: one obvious zero-setup daily command.
  - Cost/noise risk: low, but it is an ergonomics improvement rather than a correctness gap.
- Tighten `check-full` fallback messaging, or require an explicit opt-in when Docker is missing and only partial native evidence can run.
  - Benefit: reduces false confidence.
  - Cost/noise risk: low to medium depending on strictness.
- Trial `noctx` and `errchkjson` after cleanup.
  - Benefit: catches missing context propagation and ignored JSON encode errors.
  - Cost/noise risk: moderate because current baseline has findings that need policy review.
- Add targeted fuzz tests for stable parser/normalizer surfaces such as config, URL, or DSN parsing.
  - Benefit: turns existing fuzz-smoke wiring into real parser hardening.
  - Cost/noise risk: moderate because fuzz failures require seed triage.
- Consider `goleak` in goroutine-heavy bootstrap/server packages after a local trial.
  - Benefit: extends leak detection beyond `internal/infra/http`.
  - Cost/noise risk: moderate because legitimate background goroutines may require ignores.
- Decide whether `test-report` should keep running race in addition to the dedicated `test-race` job.
  - Benefit: potential full-check speedup if race duplication is reduced.
  - Cost/noise risk: moderate because CI artifacts and branch-protection expectations need care.

### Skip For Now

- Do not enable default `cyclop`, `gocyclo`, `nestif`, `funlen`, `gocognit`, `revive`, `paralleltest`, `tparallel`, `testpackage`, `containedctx`, `canonicalheader`, `fatcontext`, `forcetypeassert`, full `modernize`, or `spancheck` as blocking beginner-facing gates.
- Do not move nightly flake/fuzz checks into `make check` or required PR CI by default.
- Do not aggressively raise the global coverage floor above the existing 65 percent without package/scenario-specific reasoning.

## Approval Points

- The user should approve the desired strictness before enabling any new lint bundle, even the zero-baseline one.
- The user should choose how strict `check-full` should be when Docker is unavailable: clearer warning, explicit opt-in, or hard failure.
- Any follow-up implementation should start with the zero-setup guardrails bug, because it is a workflow correctness gap rather than a style preference.

## Validation Evidence

- `make check`: passed in this session.
- `go tool golangci-lint config verify`: passed in this session.
- `go tool golangci-lint linters`: inspected in this session.
- `go tool golangci-lint` candidate probes: ran in this session; summarized above.
- `env PATH=/usr/bin:/bin bash scripts/dev/docker-tooling.sh guardrails-check`: failed with `go: command not found`, confirming the hidden host-Go dependency.
- `bash -n scripts/dev/docker-tooling.sh scripts/dev/configure-branch-protection.sh scripts/ci/required-guardrails-check.sh`: passed.
- `go help testflag | rg -- "-shuffle|-count|-fuzz|-fuzztime"`: confirmed the relevant Go test flags.
- Heavy full CI (`make check-full`, `make docker-ci`, CI jobs) was not run in this research pass.
