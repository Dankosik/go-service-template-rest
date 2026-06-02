# Balance And Usage Authority Cutover Technical Design

Phase: technical-design
Status: complete
Pass type: fresh split-design pass
Owner: orchestrator
Date: 2026-06-02

## Session Outcome

Technical design completed for
`specs/balance-usage-authority-cutover`. The session produced a review-ready
split design bundle under `design/` and did not write `tasks.md`,
`test-plan.md`, `rollout.md`, code, migrations, generated files, tests,
implementation handoff, validation output, or technical-design-review output.

## Design Artifacts

Core artifacts:

| Artifact | Status | Purpose |
| --- | --- | --- |
| `design/overview.md` | review-ready | Entry point, chosen approach, live-fork decisions, artifact index, and review readiness. |
| `design/component-map.md` | review-ready | Billing-service and proxy component responsibilities, changed surfaces, stable surfaces, and intentional non-touches. |
| `design/sequence.md` | review-ready | Account import, resolve, balance, migrated paid admission, usage convergence, terminal, readback, worker, rollback, and failure behavior. |
| `design/ownership-map.md` | review-ready | Source-of-truth, dependency direction, generated-code authority, auth ownership, runtime ownership, proxy ownership, and explicit non-owners. |

Conditional artifacts:

| Artifact | Status | Trigger rationale |
| --- | --- | --- |
| `design/data-model.md` | review-ready | Triggered by account import/parity, balance readbacks, usage operation lineage, microlease child debits, inbox/outbox, and reconciliation state. |
| `design/dependency-graph.md` | review-ready | Triggered by new app services, generated contract flow, repository boundaries, worker adapters, and proxy adapter coupling. |
| `design/contracts/http-api.md` | review-ready | Triggered by new internal account, balance, usage, operation, reconciliation, admin, and auth-scope REST contract surfaces. |
| `design/contracts/events.md` | review-ready | Triggered by Redpanda terminal, checkpoint, close, and billing-fact event ownership. |
| `design/worker-runtime.md` | review-ready | Triggered by the current no-op billing-worker bootstrap and required runtime ownership. |
| `design/rollout-validation-inputs.md` | review-ready | Triggered by mixed-mode proxy cutover, failback constraints, cross-repo proof, layered validation, and future rollout/test-plan needs. |

Later artifacts:

| Artifact | Status | Rationale |
| --- | --- | --- |
| `test-plan.md` | expected later, not written in this phase | Validation obligations are too broad for `tasks.md` alone, but this session is constrained to technical design only. |
| `rollout.md` | expected later, not written in this phase | Proxy cutover, rollback/failback, mixed-version behavior, and cohort gates require a rollout artifact, but this session is constrained to technical design only. |
| `tasks.md` | expected later, not written in this phase | Planning must wait for technical design review. |

## Accepted Assumptions Preserved

- Top-ups, payment-service integration, payment evidence, PSP flows, and
  operator credit mutation stay out of scope.
- `account_scope_key=user:<proxy_user_id>` is the first migrated account model.
- Organization charging remains reserved but inactive.
- Pricing-service supplies immutable USD-compatible pricing lineage.
- API-key-service may return policy context such as
  `spend_limit_check_required`, but final spend/account/usage authority stays
  in the proxy/billing path.
- Migrated proxy paid admission remains microlease-first with durable child
  debit before external execution and no direct reserve fallback.

## Blockers

None for starting technical design review.

## Reopen Conditions

Reopen specification if review or planning finds that the accepted design
requires any of:

- direct per-request reserve fallback for migrated proxy cohorts;
- proxy-local money writes for migrated cohorts;
- non-JWT bearer-key production auth;
- top-up/payment ownership;
- organization charging;
- Redis or memory spend authority;
- weaker privacy policy;
- runtime behavior that cannot fail closed.

Reopen technical design if review finds a planning-critical gap in ownership,
contracts, data model, sequence/failure behavior, worker/runtime wiring,
rollout inputs, or validation inputs that does not change approved scope.

## Completion Marker

Complete when:

- the design bundle exists and is internally consistent;
- this phase file records artifact statuses and blockers;
- `workflow-plan.md` routes the next session to technical design review;
- no later-phase artifacts or implementation surfaces are created.

Completion status: complete.

## Stop Rule

Stop now. The next phase is technical design review only. Do not start
technical design review, planning, implementation, validation, rollout, code,
migrations, generated files, tests, or `tasks.md` in this session.
