# Go Code Quality Automation Improvements Spec

## Context

The completed research pass found the template already has strong Go quality automation, but the user approved implementing all actionable recommendations to improve zero-setup reliability, lint coverage, nightly checks, test proving power, and beginner-facing workflow guidance.

## Scope / Non-goals

In scope:
- Fix the Docker zero-setup `guardrails-check` host-Go leak.
- Add a quick `docker-check` path.
- Improve `check-full` fallback messaging without removing native partial mode.
- Enable low-noise lint candidates and clean up `noctx`/`errchkjson` findings so those can become default gates.
- Add nightly `-shuffle=on`.
- Add at least one stable fuzz target so fuzz smoke does real work.
- Add `goleak` coverage to a goroutine-heavy bootstrap package if it verifies cleanly.
- Remove race duplication from `test-report` while preserving the dedicated `test-race` gate.
- Update PR/development docs to guide less-experienced users toward the right evidence.

Out of scope:
- Enabling noisy default linters identified as skip-for-now in research.
- Moving nightly flake/fuzz checks into required PR CI.
- Raising global coverage threshold.
- Architecture, API, database, deployment, or OpenAPI policy changes beyond Go quality workflow effects.

## Constraints

- Keep beginner-facing defaults template-friendly and low-noise.
- Preserve zero-setup Docker support for users without local Go.
- Keep local/CI/Docker command names and documentation aligned.
- Do not hand-edit generated files.

## Decisions

- Treat the user's “all recommendations” approval as approval to implement all actionable `do now` and `maybe later` recommendations; preserve `skip for now` recommendations by not enabling those checks.
- Use a soft `check-full` fallback warning rather than a hard failure when Docker is unavailable, preserving the existing native fallback.
- Make `test-race` the canonical race gate and make `test-report` coverage/report-focused to avoid duplicated race execution in full validation.
- Enable `noctx` and `errchkjson` after cleaning up the current findings.

## Validation

- `go tool golangci-lint config verify`: passed.
- `make lint`: passed.
- `go test ./internal/config ./internal/infra/http ./cmd/service/internal/bootstrap`: passed.
- `make test-fuzz-smoke FUZZ_TIME=2s`: passed.
- `make test-report COVERAGE_MIN=65.0`: passed.
- `make test-race`: passed.
- `make check`: passed.
- `env PATH=/usr/local/bin:/usr/bin:/bin bash scripts/dev/docker-tooling.sh guardrails-check`: passed without host `go` in PATH.
- `make docker-check`: passed.
- `bash -n scripts/dev/docker-tooling.sh scripts/ci/required-guardrails-check.sh scripts/dev/configure-branch-protection.sh`: passed.
- `git diff --check`: passed.
- `make check-full` / `make docker-ci`: not run.

## Outcome

Implemented the approved Go code quality automation improvements.
