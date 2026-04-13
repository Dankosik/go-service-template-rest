# Sequence

## Quick Local

1. User runs `make check`.
2. If local Go exists, Make runs `fmt-check`, `lint`, and `test`.
3. If local Go is missing but Docker is reachable, Make runs `docker-fmt-check`, `docker-lint`, and `docker-test`.

## Quick Docker

1. User runs `make docker-check`.
2. Make runs `docker-fmt-check`, `docker-lint`, and `docker-test`.
3. Each subcommand uses the pinned Go tooling image path through `scripts/dev/docker-tooling.sh`.

## Full Local

1. User runs `make check-full`.
2. If Docker is reachable, Make runs `docker-ci`.
3. `docker-ci` routes Go-dependent guardrails through a pinned Go container, avoiding a host-Go requirement.
4. If Docker is unavailable, Make runs native `ci-local` with explicit partial-evidence wording.

## Test Reporting

1. `test-race` remains the race detector gate.
2. `test-report` produces JUnit, JSON, coverage profile, and coverage threshold evidence without running race a second time.
3. CI required contexts remain `test-race` and `test-coverage`.

## Nightly

1. Nightly repeated tests run with `-count=5 -shuffle=on`.
2. Fuzz smoke runs only when fuzz targets exist.
