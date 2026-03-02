# CI quality gates instructions for LLMs

## Load policy
- Load: Optional.
- Use when:
  - Defining or updating CI/CD quality gates for merge and release.
  - Designing fast-path vs full pipeline vs nightly vs release checks.
  - Reviewing blocking rules for formatting, lint, static analysis, tests, security, contract checks, and drift checks.
  - Defining or reviewing docs drift, codegen drift, migrations validity, and contract compatibility controls.
- Do not load when: Task is local implementation work with no CI, release, or quality-gate impact.

## Purpose
- This document defines the repository baseline for CI quality gates and blocking rules.
- Goal: deterministic merge/release decisions with low ambiguity, low noise, and high signal.
- Treat this as an LLM contract: checks must be explicit, ordered, and enforceable.

## Baseline assumptions
- Stack: Go microservice with Makefile-driven checks and GitHub Actions.
- Contract-first API workflow exists (`api/openapi/service.yaml` + `internal/api` generation).
- SQL migrations are stored in `env/migrations/`.
- Container artifact is built from `build/docker/Dockerfile`.
- Security baseline includes `govulncheck`, `gosec`, and container scan (Trivy).

## Required inputs before changing gate policy
Resolve these first. If unknown, apply defaults and document assumptions.

- Branch model and protected branch policy (which checks are required).
- Runtime budget for PR feedback (target duration for fast-path and full pipeline).
- Risk profile of service (internal-only vs external/public).
- Release cadence (continuous vs scheduled release trains).
- Migration policy (forward-only vs up/down rollback support).
- Contract compatibility policy (breaking changes allowed or blocked by default).

## Pipeline tiers and blocking intent
- `fast-path`: short feedback for contributor velocity; runs on every PR update.
- `full pipeline`: merge gate; required before merge to protected branch.
- `nightly`: long-running reliability checks; blocks release, not day-to-day PR iteration.
- `release`: artifact trust and production readiness checks; hard gate for release tag/deploy.

Default rule: jobs may run in parallel, but decision order is logical and fail-fast.

## Mandatory merge gates and required execution order
Use this order for merge decisions.

1. Repository integrity gate.
- Set `GOFLAGS=-mod=readonly`.
- Run `go mod tidy -diff`.
- Run `go mod verify`.
- Fail if any diff or verification error appears.

2. Formatting gate.
- Run `make fmt`.
- Run `git diff --exit-code`.
- Fail on any formatting drift.

3. Static quality gate.
- Run `make lint`.
- Fail on any linter/vet/staticcheck error.

4. Contract and codegen gate.
- Run `make openapi-generate`.
- Run `go test ./internal/api`.
- Run `make openapi-validate`.
- Run `make openapi-lint`.
- Run `make openapi-drift-check` (tracked + untracked codegen artifacts).
- On PR, run `BASE_OPENAPI=<base-spec> make openapi-breaking`.
- Fail on any validation/lint/breaking/codegen drift signal.

5. Unit test gate.
- Run `make test`.
- Fail on any test failure, panic, timeout, or flake rerun mismatch.

6. Source security gate.
- Run `govulncheck ./...` in default mode (not JSON/SARIF-only mode for gating).
- Run `gosec ./...`.
- Fail on non-zero exit code.

7. Extended correctness gate.
- Run `make test-race`.
- Run `make test-integration`.
- Fail on race detection or integration failure.

8. Container security gate.
- Build image: `docker build -f build/docker/Dockerfile -t service:ci .`.
- Scan image with Trivy (`CRITICAL,HIGH`, non-zero exit on findings).
- Fail on container build/scan failure.

## Fast-path vs full vs nightly vs release matrix

| Tier | Trigger | Default scope | Blocking target |
|---|---|---|---|
| Fast-path | every PR update | `go mod tidy -diff`, `go mod verify`, `make fmt` + no diff, `make lint`, `make test`, `make openapi-generate`, `make openapi-validate`, `make openapi-lint`, codegen drift check, docs drift check, `govulncheck` | blocks PR update status |
| Full pipeline | required before merge | fast-path + `make openapi-breaking`, `make test-race`, `make test-integration`, `gosec`, container build + Trivy | hard stop for merge |
| Nightly | schedule | full pipeline + long fuzz (`go test -fuzz ... -fuzztime ...`), flake detection runs (`-count`), leak checks, heavier integration matrix | blocks release, not routine PR merge |
| Release | tag/release branch | nightly + SBOM, provenance attestations, artifact signing, release-doc consistency and incident-readiness checks | hard stop for release |

## Drift and compatibility checks

### Docs drift (mandatory)
Default policy: changes that alter behavior, contract, CI process, or operations must update docs in same PR.

Trigger paths (minimum):
- `api/openapi/service.yaml`
- `env/migrations/**`
- `Makefile`
- `.github/workflows/**`
- `cmd/**`, `internal/app/**`, `internal/infra/http/**` when runtime behavior changes

Required docs update paths:
- `docs/**` or `README.md`

Minimum CI guard example:
```bash
BASE_REF="${BASE_REF:-origin/main}"
CHANGED="$(git diff --name-only "$BASE_REF"...HEAD)"

if echo "$CHANGED" | grep -Eq '^(api/openapi/service.yaml|env/migrations/|Makefile|\.github/workflows/|cmd/|internal/app/|internal/infra/http/)'; then
  echo "$CHANGED" | grep -Eq '^(docs/|README\.md$)' || {
    echo "docs drift: behavior changed without docs update"
    exit 1
  }
fi
```

### Codegen drift (mandatory)
- Run generation in CI: `make openapi-generate`.
- Immediately verify no uncommitted changes in generated files:
  - `make openapi-drift-check`.
- Compile generated package:
  - `go test ./internal/api`.
- Fail merge if generation changes files not included in PR.

### Migrations validity (mandatory when `env/migrations/**` changed)
Default rule:
- Migration changes require automated DB validation in CI.
- If repository has no migration validation command, migration PR is blocked until one is added.

Minimum validation contract:
1. Apply all forward migrations on clean ephemeral DB.
2. Verify schema reaches expected version.
3. If rollback policy supports down migrations: test down+up for latest migration.
4. Run integration tests against migrated DB.

Example gate:
```bash
if git diff --name-only "$BASE_REF"...HEAD | grep -Eq '^env/migrations/'; then
  if grep -q '^migration-validate:' Makefile; then
    make migration-validate
  else
    echo "missing migration-validate target for migration changes"
    exit 1
  fi
fi
```

### Contract compatibility (mandatory)
- Contract source of truth: `api/openapi/service.yaml`.
- Merge gate for PR:
  - extract base spec from target branch,
  - run `BASE_OPENAPI=<base-spec> make openapi-breaking`.
- Breaking change policy default:
  - breaking changes are blocked unless explicit approved exception is provided in review with rollout plan.

## Hard stop signals for merge and release

### Hard stop for merge
Any one signal below blocks merge.

- Any required CI job fails, times out, or is cancelled.
- `go mod tidy -diff` or `go mod verify` fails.
- Formatting drift exists after `make fmt`.
- `make lint` fails.
- Unit/race/integration tests fail or are flaky without quarantine decision.
- OpenAPI generate/validate/lint/breaking check fails.
- Codegen drift is detected.
- Docs drift is detected by configured path rules.
- Migration validation fails or is missing for migration-changing PR.
- `govulncheck` returns non-zero (default mode gate).
- `gosec` returns non-zero without approved suppression process.
- Trivy reports blocked severities (`HIGH`/`CRITICAL`) based on policy.

### Hard stop for release
Any one signal below blocks release.

- Any full-pipeline gate is red.
- Nightly reliability checks are red for current release candidate.
- Unresolved high-risk security findings without documented risk acceptance.
- Missing SBOM or provenance attestation for release artifact.
- Missing artifact signature verification.
- Contract compatibility violations not covered by approved versioning plan.
- Migration rehearsal failure for release schema changes.
- Missing incident/readiness updates for security-sensitive or operationally sensitive changes.

## Decision rules
Apply in order.

1. If a check protects correctness, security, contract compatibility, or migration safety, it is blocking.
2. If a check is fast and deterministic, keep it in fast-path.
3. If a check is expensive but high value (race, integration, long fuzz), keep it in full pipeline or nightly by risk.
4. If a check is noisy, tune or quarantine; do not silently disable.
5. If a change touches OpenAPI, run codegen + validate + lint + breaking checks.
6. If a change touches migrations, run migration validation against ephemeral DB.
7. If a change touches delivery workflow/commands, require docs update in same PR.
8. If security scan output is non-blocking by tool format, add a separate blocking invocation.
9. If breaking API change is intentional, require explicit approval and rollout/backward-compatibility plan.
10. If required evidence is missing, reject merge/release.

## Anti-patterns
Treat these as review blockers unless explicitly approved.

- Marking unstable or informational checks as required status checks.
- Running `govulncheck` only in JSON/SARIF mode and assuming it gates merge.
- Running `go generate` without enforcing zero drift after generation.
- Running OpenAPI lint/validate without compatibility check against base branch.
- Allowing migration SQL changes without DB migration rehearsal.
- Treating docs updates as optional for behavior/contract/CI changes.
- Disabling required checks "temporarily" without expiry and owner.
- Accepting flaky tests instead of quarantine + ticket + owner + due date.
- Merging with red security jobs based on "will fix later" comments only.
- Release without signed artifacts and provenance evidence.

## MUST / SHOULD / NEVER

### MUST
- MUST define required status checks in branch protection and keep names stable.
- MUST enforce merge gates via CI status, not manual memory.
- MUST enforce dependency integrity (`-mod=readonly`, tidy diff, mod verify).
- MUST enforce contract checks and codegen drift checks.
- MUST enforce migration validation when migrations change.
- MUST enforce security scans as blocking where policy says blocking.
- MUST keep docs drift policy automated and deterministic.

### SHOULD
- SHOULD keep fast-path under a strict time budget and fail-fast order.
- SHOULD keep heavy checks in nightly/release when they degrade PR velocity.
- SHOULD pin tool versions for reproducibility (`golangci-lint`, `govulncheck`, `gosec`, Trivy).
- SHOULD publish coverage/security artifacts for debugging failed gates.
- SHOULD maintain a quarantine workflow for flaky tests with strict expiration.

### NEVER
- NEVER bypass required checks with force-merge on protected branches.
- NEVER silently downgrade blocking checks to informational.
- NEVER accept contract-breaking changes without explicit versioning and rollout plan.
- NEVER merge migration changes without automated validation evidence.
- NEVER release artifacts without security and provenance evidence.

## Review checklist (merge/release gate)
Before approving CI or gate changes, verify:

- Required checks list matches actual workflow jobs.
- Gate order is fail-fast and documented.
- Fast-path, full, nightly, and release scopes are explicitly defined.
- Docs drift rule has path triggers and automated enforcement.
- Codegen drift check runs generation plus `git diff --exit-code`.
- Migration validity rule is automated and enforced for migration changes.
- Contract compatibility check compares current spec with base branch spec.
- `govulncheck` is run in blocking mode; SARIF/JSON output is not the only gate.
- Security findings suppression process requires owner, rationale, and expiry.
- Hard stop conditions for merge and release are explicit and machine-enforced.
- No anti-pattern (manual bypass, noisy required checks, silent gate removal) is introduced.
