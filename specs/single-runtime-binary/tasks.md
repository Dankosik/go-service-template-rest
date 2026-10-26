# Goal
status: done
Completion: The assembled change satisfies [spec.md](spec.md) R1 through R9 and its Success table (M1 at most 50 MB and at most 40% of baseline M1; M2 not larger than baseline M2; M3 at most baseline M3 + 1 MB; M1 gzip reported) through the mechanism in [design/system.md](design/system.md) and [design/ownership-map.md](design/ownership-map.md), with the spec's Proof expectations executed at one final validation and no known in-scope defect.
Global constraints: Baseline is commit `aeda2664f` plus the uncommitted edits to `build/docker/Dockerfile`, `go.mod`, `go.sum`, `scripts/ci/runtime-image-build.sh`, and `internal/problem/problem_test.go`; build on them and never revert them. Accepted decisions live in spec.md and the design; do not reopen them. Heavy and full gates follow [AGENTS.md](../../AGENTS.md#validation-budget) (`ALLOW_HEAVY=1`/`ALLOW_FULL=1`, never concurrently). Push, pull request, template sync fan-out to derived repositories, and deployment are not authorized by this ledger.

## Tasks

- [x] T001: The template ships one `argv[0]`-dispatched runtime executable for every selected role, with unchanged entry paths, local `make` behavior, profile stripping, role seams, synced-tooling behavior, and a template-owned migration guide.
  - Depends on: none
  - Provides: the complete single-executable candidate in the working tree, plus the baseline snapshot that final validation measures against.
  - Packet: tasks/T001-single-runtime-executable.md
  - Resume: Baseline snapshot `/Users/daniil/Projects/Opensource/go-service-template-rest.single-runtime-baseline` (detached worktree at `aeda2664f` plus the five baseline edits); patch copy `/Users/daniil/Projects/Opensource/go-service-template-rest.single-runtime-baseline.patch`.

## Results

Implemented: T001; verification: final validation below; candidate: uncommitted working tree on `aeda2664f`.
Accepted: Completion; evidence: linux/amd64 PGO off — M1 48,555,074 B vs baseline 140,338,738 B (34.6%), gzip 15,748,992 B vs 48,996,346 B; M2 29,349,079 B vs 48,420,179 B; M3 24,777,943 B vs 24,773,847 B (+4,096 B); one ELF plus role symlinks; `make template-init-check` 12/12 profiles; `runtime-image-check`, `migration-validate`, `test-integration-process` (5 PASS) on the candidate; role exec parity with baseline; unmigrated derived layout fails naming the guide; `--no-cache` rebuild digest-identical; stale PGO manifest fails closed; zoom-out review PASS (follow-ups 1, 2, 4 and the duplicate `go list` in 5 applied afterwards, rechecked with `go test ./cmd/entrypoint`, shellcheck, the changed-surfaces self-test and the isolation guard); lint, `lint-deep` and `make verify` pending CI; candidate: working tree on `aeda2664f`, not committed.
