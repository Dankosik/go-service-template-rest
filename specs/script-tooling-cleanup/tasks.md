# Lean repository validation tooling

status: draft
Completion: tracked tooling exposes only the retained canonical validation paths, the named ignored candidate bundle is absent, and all focused repository gates pass.
Blocked stop: preserve the current diff and report the first failing retained gate with its owning surface; reopen the specification only if fixing it requires restoring removed behavior.
Global constraints: preserve generated, security, branch-protection, migration, container, worktree-transfer, setup/doctor, and Docker parity gates; add no dependency or compatibility wrapper; leave all concurrent primary-checkout edits and every ignored `.artifacts/test` sibling outside the named template-sync bundle unchanged; execute tracked edits in `/Users/daniil/.codex/worktrees/script-tooling-cleanup` on branch `codex/script-tooling-cleanup` from immutable base `e0b18bc5e2cba988b0dc2aa18aa7e82863ca6aac`, then integrate only its bounded patch into local `main` after refreshing overlap classification. The planning bundle is tracked by that accepted base: keep byte-identical copies during execution, then include deletion of `specs/script-tooling-cleanup/spec.md` and `specs/script-tooling-cleanup/tasks.md` in the authorized bounded patch after terminal evidence is captured. Immediately after readiness PASS and before implementation, record the primary HEAD/status and both byte-identical planning-file hashes as the accepted execution-start snapshot; this verified ledger revision is baseline state rather than concurrent overlap.
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
- `assert_make docker-secret-scan 'bash ./scripts/dev/docker-tooling.sh secret-scan'`
- `assert_make docker-container-security 'bash ./scripts/dev/docker-tooling.sh container-security'`
- `assert_make docker-ci 'bash ./scripts/dev/docker-tooling.sh ci'`
The observable is rejection of the missing-before-present negative control followed by zero exit after every literal selector appears in that target's `make -n` output; any missing target, failed dry run, or missing selector fails the task.

Reusable removal oracle for T1, T2, and T4:
- Require `test ! -e` for `scripts/ci/hard-skills-check/size.go`, `scripts/ci/hard-skills-check/emission.go`, `scripts/dev/hard-skills-evals.sh`, and `scripts/ci/docs-drift-check.sh`.
- Require zero matches from `rg -n --hidden --glob '!.git/**' --glob '!.artifacts/**' --glob '!.cache/**' --glob '!specs/script-tooling-cleanup/**' '(size-report|emit-selected-evals|hard-skills-evals\.sh|docs-drift-check|docker-docs-drift-check|bootstrap-native|bootstrap-docker|secrets-scan|docker-secrets-scan|agents-check)' Makefile .github scripts docs README.md .agents`.
- Require zero matches from `rg -n '^(setup|setup-strict|setup-native|setup-native-strict|setup-docker|bootstrap-native|bootstrap-docker|secrets-scan|docker-secrets-scan|agents-check):' Makefile`.
- Require zero matches from `rg -n 'make (setup|setup-strict|setup-native|setup-native-strict|setup-docker|bootstrap-native|bootstrap-docker|secrets-scan|docker-secrets-scan|agents-check)([^[:alnum:]_-]|$)' scripts docs README.md .agents`.
Run each `rg` with `set +e`, capture its status, restore `set -e`, and require status exactly 1: status 0 means a forbidden match and status 2 means the oracle itself failed. After the planning bundle is deleted in T4, repeat the first `rg` without its `specs/script-tooling-cleanup/**` exclusion.

Reusable shell-syntax oracle for T2 and T4: define `check_shell_list() { while IFS= read -r -d '' file; do [[ ! -e "${file}" ]] || bash -n "${file}" || return 1; done; }`; prove intermediate failure first by piping a NUL list containing a temporary invalid `if`-only shell file followed by a valid `:` shell file and requiring `check_shell_list` to reject it; then require zero exit from `git ls-files --cached -z -- '*.sh' | check_shell_list`. Remove the two temporary control files afterward. The removal oracle separately proves that every skipped tracked path is an authorized deletion.

Frozen-candidate process mutation oracle for T4: after committing the isolated candidate, create a temporary detached worktree at that exact commit. First require zero exit from both `bash scripts/ci/required-guardrails-check.sh` and `bash scripts/ci/workflow-instructions-check.sh`. Before each case reset and clean only that disposable worktree to the candidate commit, apply the named single mutation, require nonzero exit from the named checker, and require the diagnostic substring:
- change the `AS build` suffix on the Go builder line in `build/docker/Dockerfile`; guardrails must report `runtime build image must match go.mod`;
- delete the `iface` enablement line from `.golangci.yml`; guardrails must report `golangci-lint must enable iface`;
- change Docker tooling's `run_go "GOLANGCI_LINT_CACHE=/workspace/.cache/golangci-lint make lint"` route to invoke `make lint-broken`; guardrails must report `Docker lint must route through make lint`;
- delete the migration-directory `COPY` line from `build/docker/Dockerfile`; guardrails must report `runtime image must ship migration files`;
- rename the `repo-integrity` job key in `.github/workflows/ci.yml`; guardrails must report `CI is missing branch-protection context repo-integrity`;
- add a disposable `internal/app/health/forbidden_import_test.go` that blank-imports `github.com/jackc/pgx/v5`; guardrails must report `internal/app must not import infrastructure adapters`;
- delete `README.md`; workflow instructions must report `missing required owner: README.md`;
- rename the `## Routing` heading in `AGENTS.md`; workflow instructions must report `AGENTS.md is missing ## Routing`;
- replace the canonical `implementation-validation-closeout.md#optional-worker-execution` link in `docs/subagent-contract.md`; workflow instructions must report that the canonical link is missing;
- rename workflow manifest heading `### E66` to `### E67` in `docs/spec-first-workflow-evals.md`; workflow instructions must report `workflow behavior eval check failed`.
After all cases, reset/clean the disposable worktree, require both checkers to pass again, require its status to be empty, and remove only that temporary worktree. This is terminal proof, not new tracked test machinery.

- [ ] T1: Remove the hard-skills size report and dedicated eval runner/emitter while retaining repository/scoped checks and the generic eval corpus.
  - Source: `spec.md` Scope and non-goals paragraphs 2-3; Behavior and contract delta paragraph 1.
  - Owner/surface/resources: `scripts/ci/hard-skills-check`, `scripts/dev/hard-skills-evals.sh`, and their command-reference documentation in the isolated tracked worktree.
  - Depends on: none.
  - Proof: the reusable removal oracle succeeds for the T1 names and `go test ./scripts/ci/hard-skills-check` passes.
  - Reopen if: a removed hard-skills path has a concrete current consumer that cannot use `workflow-behavior-evals.sh`; Specification.

- [ ] T2: Remove the docs-drift proxy, shrink self-referential checkers, and leave one canonical Make/Docker command name without weakening protected gates.
  - Source: `spec.md` Scope and non-goals paragraphs 4-8; Behavior and contract delta paragraphs 2-5.
  - Owner/surface/resources: `scripts/ci`, `scripts/dev/docker-tooling.sh`, `Makefile`, `.github/workflows`, `README.md`, `docs`, and `.agents/skills` in the isolated tracked worktree.
  - Depends on: T1.
  - Proof: the reusable removal, shell-syntax, and no-side-effect wiring oracles above all succeed; `make workflow-routing-check`, `make guardrails-check`, `make workflow-behavior-evals-check`, and `make instruction-evals-harness` pass.
  - Reopen if: a retained generated, security, branch-protection, migration, container, worktree-transfer, setup/doctor, mutation-harness, or Docker parity gate loses its canonical path; Delivery policy.

- [ ] T3: Delete only the accepted ignored template-sync candidate bundle.
  - Source: `spec.md` Scope and non-goals; Invariants and edge cases.
  - Owner/surface/resources: `.artifacts/test/template-sync-77da61e-2f72764`; exclusive local filesystem mutation.
  - Depends on: none.
  - Proof: record pre-task `git status --porcelain=v1 --untracked-files=all`; before deletion, write a deterministic manifest for every `.artifacts/test` entry outside the named bundle by sorting all descendant paths and recording each path's type plus SHA-256 for regular files or link target for symlinks; after deletion require the exact directory absent, regenerate the same manifest, require byte equality with `cmp`, and require the Git status baseline unchanged.

- [ ] T4: Prove the simplified integrated surface without widening into publication or external mutation.
  - Source: `spec.md` Success criteria and proof expectations.
  - Owner/surface/resources: isolated tracked worktree for candidate proof, then the local `main` checkout for bounded patch integration and terminal proof; after capturing terminal evidence, delete the two tracked task-local planning paths on the isolated branch and integrate those deletions with the bounded patch; local Go/shell toolchains; no external writes.
  - Depends on: T1, T2, T3.
  - Proof: before integration, derive the isolated patch path set; intersect it with both the execution-start primary changed-path set and the union of primary committed/uncommitted paths changed after the snapshot. The only exemption is the two planning paths when their current hashes still equal the recorded accepted hashes and the isolated delta for them is deletion. With an empty intersection, isolated candidate and resulting local-main patch are byte-equivalent on all bounded paths and all other primary bytes remain unchanged; the reusable removal, shell-syntax, wiring, and frozen-candidate mutation oracles succeed; `make workflow-routing-check`, `make guardrails-check`, `make workflow-behavior-evals-check`, `make instruction-evals-harness`, `make lint`, `make go-security`, `git diff --check`, and final diff review all pass; after evidence capture and integration, both physical checkouts lack the two planning paths and the bounded tracked patch records only their authorized deletions.
  - Reopen if: either exact-path intersection is non-empty, either planning hash differs, or the planning-path delta is not deletion; preserve both states, do not merge automatically, and return to Planning checkout coordination for owner reconciliation.
