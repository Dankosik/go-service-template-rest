---
name: go-devops-spec
description: "Design delivery/platform-first specifications for Go services in a spec-first workflow. Use when planning or revising CI/CD quality gates, merge/release blocking policy, docs/codegen/migration/contract drift controls, container runtime hardening baseline, and release trust requirements before coding. Skip when the task is a local code fix, endpoint/resource API design, SQL schema-only modeling, distributed workflow design, or low-level implementation of pipeline/scripts/manifests."
---

# Go DevOps Spec

## Purpose
Create a clear, reviewable delivery/platform specification package before implementation. Success means CI/CD gates, release-safety controls, and container/runtime hardening rules are explicit, defensible, and directly translatable into implementation and verification tasks.

## Hard Skills

#### Mission
- Design delivery/platform specifications as enforceable operational policy, not as advisory notes.
- Convert merge and release readiness into deterministic, machine-checkable gates with explicit hard-stop conditions.
- Keep production-risk controls explicit across CI/CD, schema evolution, container runtime, and artifact trust.

#### Default posture
- Treat correctness, security, contract compatibility, migration safety, and release trust as blocking by default.
- Prefer fail-closed decisions when facts are missing; unresolved critical unknowns become explicit blockers.
- Prefer additive, backward-compatible rollout patterns over destructive or big-bang changes.
- Require evidence-first decisions: no gate policy is accepted without concrete enforcement point and artifact proof.
- Keep local and CI behavior aligned through exact repository commands and stable job mappings.

#### CI gate engineering
- Define and maintain four delivery tiers with distinct intent: `fast-path`, `full`, `nightly`, `release`.
- Preserve merge-gate execution order with fail-fast semantics:
  1. repository integrity (`-mod=readonly`, `go mod tidy -diff`, `go mod verify`)
  2. formatting drift
  3. static quality/lint
  4. OpenAPI/codegen/compatibility checks
  5. unit tests
  6. source security (`govulncheck`, `gosec`)
  7. race + integration checks
  8. container build + Trivy scan
- Keep required status check names stable and branch-protection compatible.
- Treat cancelled/timed-out required jobs as failed gates.

#### Branch protection and PR governance
- Require PR-based merge for protected branches; disable direct pushes and force-push bypasses.
- Require up-to-date branch, approved review, stale-approval dismissal, and conversation resolution.
- Include administrators in protection policy to prevent privileged bypass drift.
- Keep repository guardrails present in default branch (`CODEOWNERS`, PR template, security and contribution policy files).

#### Command fidelity and CI mapping
- Use only repository-defined commands and targets when defining gates.
- Keep policy mapped to concrete commands from `docs/build-test-and-development-commands.md`.
- Preserve CI/local parity for core checks (`mod-check`, `fmt-check`, `lint`, `test`, OpenAPI checks, security checks).
- Keep conditional gates explicit (`openapi-breaking` for PR compatibility, `migration-validate` when migrations change).
- Treat undocumented or non-reproducible command substitutions as policy defects.

#### Drift and compatibility controls
- Enforce docs drift checks for behavior-changing paths in the same PR.
- Enforce codegen drift by running generation and zero-drift verification immediately after generation.
- Enforce OpenAPI compatibility against base branch when contract can change.
- Block undocumented breaking API changes unless there is explicit approved exception with rollout/versioning plan.
- Require behavior-contract consistency between implementation, generated artifacts, and published docs.

#### Migration and data-evolution safety
- Require phased schema rollout: `Expand -> Migrate/Backfill -> Contract`.
- Require mixed-version compatibility across rolling/canary deployments until contract phase.
- Use one controlled migrator process, never migration-on-every-pod startup behavior.
- Require session-level migration safety budgets (`lock_timeout`, `statement_timeout`, `idle_in_transaction_session_timeout`).
- Require idempotent, resumable, throttled backfills with durable checkpoints and explicit abort thresholds.
- Require objective verification gates (invariants, aggregate parity, deterministic diff checks) before contract.
- Classify rollback mode explicitly (`safe`, `conditional`, `restore-based`) and document irreversible steps.
- For DB + event consistency, require outbox/atomic publication patterns; reject cross-system dual writes.
- Treat restore-drill failures and untested backup posture as release blockers.

#### Security gate policy and identity safety
- Keep `govulncheck` and `gosec` in blocking mode for merge decisions.
- Keep container security scanning blocking on policy severities (`HIGH`, `CRITICAL`) for release artifacts.
- Require suppression workflow with owner, rationale, expiry, and review trail.
- Require secure-by-default controls on changed trust boundaries:
  - bounded input size/time/concurrency
  - strict validation/decoding
  - parameterized data access
  - no secret leakage in logs/errors
- Require fail-closed authn/authz and tenant scoping behavior for delivery-significant security changes.

#### Containerization and runtime hardening
- Default to multi-stage Docker builds with minimal runtime image and no build toolchain leftovers.
- Default runtime profile: distroless + non-root + exec-form `ENTRYPOINT`.
- Default linking strategy: static (`CGO_ENABLED=0`) unless cgo requirements are explicit and justified.
- Require runtime compatibility for outbound TLS and timezone behavior (CA trust + tzdata strategy).
- Require reproducible build defaults (`-trimpath`, `-mod=readonly`, deterministic ldflags, explicit Go version).
- Require `.dockerignore` to prevent context/secrets leakage.
- Require runtime hardening baseline:
  - non-root user
  - read-only root filesystem
  - no privilege escalation
  - drop Linux capabilities by default
  - no privileged/host-level execution modes without explicit exception

#### Release trust and software supply chain evidence
- Treat SBOM, provenance attestation, and artifact signing as release-gate evidence, not optional metadata.
- Require release preflight gates before publish on version tags.
- Require verifiable permissions/config for attestations, OIDC signing, and registry publishing.
- Prefer digest pinning and explicit base-image/tool version management for reproducibility.
- Reject release flows that cannot prove artifact integrity and origin.

#### Observability and SLO-aware release gating
- Require telemetry baseline for changed production paths:
  - structured logs with correlation fields
  - RED metrics plus saturation signals
  - trace propagation across sync/async boundaries
- Enforce low-cardinality telemetry dimensions; keep high-cardinality identifiers out of metrics labels.
- Require SLO policy with explicit `good/total` SLI semantics and 28-day budget accounting.
- Require multi-window burn-rate alerting with low-traffic floors.
- Tie release permissions to budget state (`green`/`yellow`/`orange`/`red`) and degrade/freeze policies.
- Require runbook and dashboard linkage for actionable paging alerts.

#### Reliability and rollout safety
- Require explicit per-dependency contracts for timeout, retry, bulkhead, fallback, and observability signals.
- Prohibit infinite deadlines/retries and unbounded queues/concurrency.
- Require overload semantics and backpressure behavior (`429`/`503` with clear policy).
- Require progressive rollout controls for risky changes (canary/blue-green/feature flag strategy).
- Require rollback ownership and objective promotion/rollback criteria.
- Treat active page-level burn alerts as rollout blockers until stability criteria recover.

#### Exception and risk acceptance governance
- Every temporary bypass must include owner, expiry, compensating controls, and reopen condition.
- No gate can be silently downgraded from blocking to informational.
- Keep all unresolved critical unknowns in `80-open-questions.md` with unblock path and owner.
- Keep accepted risk decisions and reopen criteria in `90-signoff.md`.
- Reject "fix later" wording without bounded remediation plan and accountable owner.

#### Non-negotiable never events
- Never merge destructive-first schema changes on active production paths.
- Never accept release without trust evidence (scan, SBOM, signature, provenance) required by policy.
- Never treat red CI, red nightly reliability checks, or unresolved high-risk findings as merge/release eligible.
- Never rely on manual memory instead of machine-enforced branch and CI controls.
- Never design gates that cannot be reproduced locally or verified in CI logs/artifacts.

## Scope And Boundaries
In scope:
- define CI quality-gate policy and execution tiers (`fast-path`, `full`, `nightly`, `release`)
- define merge/release blocking criteria and fail-fast decision order
- define drift and compatibility controls (docs drift, OpenAPI/codegen drift, migration validation, contract compatibility)
- define blocking source-security and container-security gate expectations (`govulncheck`, `gosec`, Trivy) at policy level
- define release trust requirements (SBOM, provenance attestation, signing, artifact evidence)
- define containerization baseline (multi-stage build, runtime base policy, non-root model, reproducible build defaults, startup command shape)
- define runtime hardening baseline and exception policy (owner, expiry, compensating controls)
- define release-safety choreography constraints (environment protection, approval gates, rollback readiness)
- produce delivery/platform deliverables that remove hidden "decide later" gaps

Out of scope:
- service/module decomposition and ownership topology as a primary domain
- endpoint/resource API semantics and payload/error schema design
- physical SQL schema design, DDL details, and migration script authoring
- distributed consistency and saga decomposition as a primary domain
- product-level secure-coding and authn/authz design as a primary domain
- SLI/SLO target ownership and telemetry signal design as a primary domain
- detailed resilience behavior design (retry/backpressure/degradation semantics) as a primary domain
- low-level implementation of CI workflow YAML, Dockerfile internals, deployment manifests, or release scripts
- benchmark/profile-driven runtime performance tuning

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting decisions.
2. Set phase-specific output targets:
   - Phase 0: seed delivery/platform assumptions and blockers in `80`; add minimum safety constraints for later design
   - Phase 1: define architecture-shaping delivery constraints for `20` and rollout-safe sequencing constraints for `60`
   - Phase 2 and later: run full devops pass; maintain `50/80/90` and update impacted `55/60/70` plus `20/30/40` when needed
3. Load context using this skill's dynamic loading rules and stop when four delivery axes are source-backed: gate policy, release safety, container/runtime baseline, compliance evidence.
4. Normalize affected delivery surface: branch protection assumptions, required checks, contract/migration change paths, artifact trust obligations, runtime hardening expectations.
5. For each nontrivial devops decision, compare at least two options and select one explicitly.
6. Assign decision ID (`DOPS-###`) and owner for each major devops decision.
7. Record trade-offs and cross-domain impact (architecture, API, data, security, observability, reliability).
8. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate in the current pass or move to `80-open-questions.md` with owner and unblock condition.
9. If uncertainty blocks merge/release safety decisions, record it in `80-open-questions.md` with concrete next step.
10. Keep `50-security-observability-devops.md` as primary artifact and synchronize devops implications in impacted artifacts.
11. Verify internal consistency: no contradictory gate policy, no unresolved exception process, and no critical delivery decisions deferred to coding.

## DevOps Decision Protocol
For every major devops decision, document:
1. decision ID (`DOPS-###`) and current phase
2. owner role
3. context and operational/release risk
4. options (minimum two)
5. selected option with rationale
6. at least one rejected option with explicit rejection reason
7. gate-level impact (`fast-path` / `full` / `nightly` / `release`, blocking vs informational)
8. enforcement points (`ci job`, `branch protection`, `release rule`, `runtime policy`)
9. required compliance evidence (report/artifact/attestation)
10. exception policy (owner, expiry, compensating controls, reopen conditions)
11. affected artifacts and linked open-question IDs (if any)

## Output Expectations
- Primary artifact:
  - `50-security-observability-devops.md` with mandatory devops sections:
    - `CI Gate Matrix And Blocking Policy`
    - `Merge And Release Hard-Stop Criteria`
    - `Drift, Compatibility, And Migration Validation Policy`
    - `Containerization And Runtime Hardening Baseline`
    - `Release Trust Evidence Requirements`
    - `Exception And Risk-Acceptance Policy`
- Required core artifacts per pass:
  - `80-open-questions.md` with devops blockers/uncertainties
  - `90-signoff.md` with accepted devops decisions and reopen criteria
- Conditional alignment artifacts (update when impacted):
  - `55-reliability-and-resilience.md`
  - `60-implementation-plan.md`
  - `70-test-plan.md`
  - `20-architecture.md`
  - `30-api-contract.md`
  - `40-data-consistency-cache.md`
- Conditional artifact status format for `55/60/70/20/30/40`:
  - include one explicit status: `Status: updated` or `Status: no changes required`
  - for `no changes required`, add one sentence justification with linked `DOPS-###`
  - for `updated`, list changed sections and linked `DOPS-###`
- Language: match user language when possible.
- Detail level: concrete and reviewable with explicit policy semantics and enforcement points.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when four delivery axes are covered with source-backed inputs: gate policy, release safety, container/runtime baseline, compliance evidence.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Artifacts`, current phase subsection, and target gate criteria first
  - load additional sections only when unresolved delivery decisions require them
- `docs/llm/delivery/10-ci-quality-gates.md`
- `docs/llm/platform/10-containerization-and-dockerfile.md`
- `docs/ci-cd-production-ready.md`

Load by trigger:
- Repository command/gate mapping requires exact local commands:
  - `docs/build-test-and-development-commands.md`
- API compatibility governance or idempotency/contract gate implications:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Migration rollout/rehearsal policy implications:
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/20-sql-access-from-go.md`
- Security policy implications for delivery/runtime:
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/security/20-authn-authz-and-service-identity.md`
- Observability/SLO gate implications:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
- Reliability/rollout/degradation policy implications:
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.
- If conflict persists, preserve latest accepted decision in `90-signoff.md` and add reopen blocker in `80-open-questions.md`.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Resolve each `[assumption]` by source validation in the current pass or by promoting it to `80-open-questions.md` with owner and unblock condition.

## Definition Of Done
- Current phase and target gate are explicitly stated.
- `50-security-observability-devops.md` contains all mandatory devops sections from this skill.
- Every major devops decision includes `DOPS-###`, owner, selected option, and at least one rejected option with reason.
- Gate matrix and blocking semantics are explicit and testable.
- Drift, compatibility, and migration validation policy is explicit and enforceable.
- Container/runtime hardening baseline and exception process are explicit and consistent.
- Release trust evidence requirements are explicit for release decisions.
- Every `[assumption]` is either source-validated in the current pass or tracked in `80-open-questions.md` with owner and unblock condition.
- Devops blockers are closed or tracked in `80-open-questions.md` with owner and unblock condition.
- Impacted `55/60/70/20/30/40` artifacts have explicit status with decision links and no contradictions.
- No hidden delivery/platform decisions are deferred to coding.

## Anti-Patterns
Use these preferred patterns to prevent anti-pattern drift:
- express CI/CD policy as enforceable gate rules with explicit blocking semantics
- keep contract, migration, and security checks mandatory by default and define formal exception flow when needed
- tie release-trust requirements to concrete evidence artifacts and attestation outputs
- define runtime hardening as an explicit baseline contract, not as implicit implementation detail
- align with security, observability, and reliability skills through explicit interface contracts and ownership boundaries
- require owner, expiry, and compensating controls for any temporary bypass
- close release-safety decisions in spec artifacts or track them as explicit open questions with owner and unblock condition
