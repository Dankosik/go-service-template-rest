# Component Map

## Tooling Config

- `.golangci.yml`: add approved low-noise linters, including `noctx` and `errchkjson` after cleanup.

## Local And Docker Commands

- `Makefile`: add `docker-check`, improve `check-full` fallback wording, make `test-report` coverage/report-focused, update help and phony targets.
- `scripts/dev/docker-tooling.sh`: route Docker guardrails through pinned Go tooling where needed, add `docker-check` support via Make target wrapper, update `test-report` race behavior.
- `build/docker/tooling-images.Dockerfile`: keep digest-pinned source of truth; no change expected unless unused image cleanup is chosen.

## CI And Nightly

- `.github/workflows/nightly.yml`: add `-shuffle=on` to repeated tests.
- `.github/workflows/ci.yml`: update step label only if needed for test-report semantics; required contexts stay unchanged.

## Go Code And Tests

- `cmd/service/internal/bootstrap`: use context-aware listen APIs where needed; add `goleak` TestMain if verification permits.
- `internal/infra/http`: handle problem JSON encode error and use request/listen APIs with context in tests.
- `internal/config`: add a stable fuzz target for parser behavior.

## Docs

- `.github/pull_request_template.md`, `CONTRIBUTING.md`, `docs/build-test-and-development-commands.md`: align beginner-facing evidence and command descriptions.
