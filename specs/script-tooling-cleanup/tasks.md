# Lean repository validation tooling

status: draft
Completion: tracked tooling exposes only the retained canonical validation paths, the named ignored candidate bundle is absent, and all focused repository gates pass.
Blocked stop: preserve the current diff and report the first failing retained gate with its owning surface; reopen the specification only if fixing it requires restoring removed behavior.
Global constraints: preserve generated, security, branch-protection, migration, container, worktree-transfer, setup/doctor, and Docker parity gates; add no dependency or compatibility wrapper; leave all concurrent primary-checkout edits and every ignored `.artifacts/test` sibling outside the named template-sync bundle unchanged; execute tracked edits in `/Users/daniil/.codex/worktrees/script-tooling-cleanup` on branch `codex/script-tooling-cleanup` from immutable base `50edfc5f409ce876c0fdab2c60172561f689cb86`, then integrate only its bounded patch into local `main` after refreshing overlap classification. Keep byte-identical copies of this accepted `spec.md` and `tasks.md` in both checkouts during execution; exclude the task-local planning bundle from the bounded integration patch and delete both copies after closeout.
Planned waves:
- W1: T1 and T3; hard-skills source cleanup and ignored local artifact deletion have disjoint writable ownership and proof resources.
- W2: T2; delivery-shell, Make, CI, skill, and documentation cleanup follows T1 because both update the command reference documentation.
- W3: T4; validation requires the complete tracked candidate and local cleanup result.

Reusable no-side-effect wiring oracle for T2 and T4: from the repository root, define `assert_make() { target="$1"; shift; output="$(make -n "$target")" || return 1; for expected in "$@"; do printf '%s\n' "$output" | grep -Fq -- "$expected" || return 1; done; }`, prove intermediate-selector fail-closed behavior first with `if assert_make openapi-check '__missing-before-present-selector__' 'go test ./internal/api'; then exit 1; fi`, then require zero exit from:
- `assert_make template-init 'bash ./scripts/dev/setup.sh'`
- `assert_make template-init-strict 'bash ./scripts/dev/setup.sh --strict'`
- `assert_make template-init-native 'bash ./scripts/dev/setup.sh --native'`
- `assert_make template-init-native-strict 'bash ./scripts/dev/setup.sh --native --strict'`
- `assert_make template-init-docker 'bash ./scripts/dev/setup.sh --docker'`
- `assert_make openapi-check 'bash ./scripts/ci/generated-drift-check.sh openapi' 'go test ./internal/api' "go test ./internal/infra/http -run '^TestOpenAPIRuntimeContract' -count=1" 'npx @redocly/cli@' 'go tool validate -- api/openapi/service.yaml'`
- `assert_make sqlc-check 'go tool github.com/sqlc-dev/sqlc/cmd/sqlc generate -f internal/infra/postgres/sqlc.yaml' 'bash ./scripts/ci/generated-drift-check.sh sqlc'`
- `assert_make secret-scan 'go tool gitleaks git --no-banner --redact --exit-code 1 --baseline-path .gitleaks.baseline.json .'`
- `assert_make migration-validate 'go tool migrate -path env/migrations' 'bash ./scripts/dev/docker-tooling.sh migration-validate'`
- `assert_make docker-container-security 'bash ./scripts/dev/docker-tooling.sh container-security'`
- `assert_make docker-ci 'bash ./scripts/dev/docker-tooling.sh ci'`
The observable is rejection of the missing-before-present negative control followed by zero exit after every literal selector appears in that target's `make -n` output; any missing target, failed dry run, or missing selector fails the task.

- [ ] T1: Remove the hard-skills size report and dedicated eval runner/emitter while retaining repository/scoped checks and the generic eval corpus.
  - Source: `spec.md` Scope and non-goals paragraphs 2-3; Behavior and contract delta paragraph 1.
  - Owner/surface/resources: `scripts/ci/hard-skills-check`, `scripts/dev/hard-skills-evals.sh`, and their command-reference documentation in the isolated tracked worktree.
  - Depends on: none.
  - Proof: removed hard-skills paths are unreachable outside this task-local specification/ledger and retained Go checks compile; scoped `rg` negative selectors and `go test ./scripts/ci/hard-skills-check` succeed.
  - Reopen if: a removed hard-skills path has a concrete current consumer that cannot use `workflow-behavior-evals.sh`; Specification.

- [ ] T2: Remove the docs-drift proxy, shrink self-referential checkers, and leave one canonical Make/Docker command name without weakening protected gates.
  - Source: `spec.md` Scope and non-goals paragraphs 4-8; Behavior and contract delta paragraphs 2-5.
  - Owner/surface/resources: `scripts/ci`, `scripts/dev/docker-tooling.sh`, `Makefile`, `.github/workflows`, `README.md`, `docs`, and `.agents/skills` in the isolated tracked worktree.
  - Depends on: T1.
  - Proof: scoped negative `rg`, `bash -n` over tracked shell scripts, and the reusable no-side-effect wiring oracle above all succeed; `make workflow-routing-check`, `make guardrails-check`, `make workflow-behavior-evals-check`, and `make instruction-evals-harness` pass.
  - Reopen if: a retained generated, security, branch-protection, migration, container, worktree-transfer, setup/doctor, mutation-harness, or Docker parity gate loses its canonical path; Delivery policy.

- [ ] T3: Delete only the accepted ignored template-sync candidate bundle.
  - Source: `spec.md` Scope and non-goals; Invariants and edge cases.
  - Owner/surface/resources: `.artifacts/test/template-sync-77da61e-2f72764`; exclusive local filesystem mutation.
  - Depends on: none.
  - Proof: record pre-task `git status --porcelain=v1 --untracked-files=all` and top-level `.artifacts/test` paths; after deletion the exact directory is absent, the status baseline is unchanged, and every sibling path remains present.

- [ ] T4: Prove the simplified integrated surface without widening into publication or external mutation.
  - Source: `spec.md` Success criteria and proof expectations.
  - Owner/surface/resources: isolated tracked worktree for candidate proof, then the local `main` checkout for bounded patch integration and terminal proof; local Go/shell toolchains; no external writes.
  - Depends on: T1, T2, T3.
  - Proof: isolated candidate and resulting local-main patch are byte-equivalent on the bounded paths; concurrent primary changes remain unchanged; the reusable no-side-effect wiring oracle above succeeds unchanged in both checkouts; `make workflow-routing-check`, `make guardrails-check`, `make workflow-behavior-evals-check`, `make instruction-evals-harness`, `make lint`, `make go-security`, `git diff --check`, focused negative `rg`, and final diff review all pass.
