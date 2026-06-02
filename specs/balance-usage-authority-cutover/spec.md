# Balance And Usage Authority Cutover Specification

Mode: full orchestrated
Status: approved
Date: 2026-06-02
Owner: orchestrator

## Context

This specification decides the billing-service authority contract for moving
durable balance and paid-usage ownership out of `gonka-proxy` for migrated paid
cohorts.

The accepted predecessor packet,
`specs/event-driven-billing-money-performance-microleases/`, already chose
billing-issued durable microleases with zero unbacked spend exposure. That
packet is preserved authority context. This specification does not weaken that
decision and does not revive direct per-request reserve fallback for migrated
proxy cohorts.

Completed research under
`specs/balance-usage-authority-cutover/research/` found that:

- billing-service has durable USD atom balance, usage, ledger, idempotency,
  microlease, inbox/outbox, and reconciliation building blocks;
- billing-service does not yet expose or wire the broader account resolve,
  balance read, usage reserve/finalize/write-off/reversal, operation readback,
  reconciliation/admin readback, concrete HTTP app service, and real worker
  runtime surfaces needed for cutover;
- `gonka-proxy` still has live local money authority in completion and
  web-search paths through `User.balanceNgonka`, in-memory reservations,
  local balance transactions, local deductions, and local usage logging;
- the current proxy shared-balance bridge and billing-service provider
  contract do not match on paths, auth model, scopes, envelopes, and operation
  semantics;
- top-ups and payment-service integration are explicitly outside this cutover.

## Scope / Non-goals

In scope:

- account resolve behavior for mapping proxy identity to billing account scope;
- authoritative balance read behavior for migrated cohorts;
- usage reserve, finalize, write-off, reversal, and stored operation readback
  semantics;
- microlease relationship to the generic usage lifecycle;
- support-safe reconciliation and admin readbacks;
- service-to-service authentication, authorization scopes, account binding,
  and privacy constraints;
- billing HTTP runtime ownership and billing-worker ownership for Redpanda,
  inbox/outbox, stale operation repair, admission renewal, and reconciliation;
- proxy cutover behavior for migrated cohorts, including local write disablement
  and fail-closed behavior;
- validation obligations for billing-service and the required proxy cutover
  proof.

Out of scope:

- top-up lifecycle, customer deposits, bonuses, payment evidence ingestion,
  payment provider flows, payment-service integration, PSP webhooks, refunds,
  and customer credit application;
- public OpenAI-compatible route ownership;
- pricing catalog ownership, quote-source ownership, model routing, devshard
  execution, transfer-agent signing, identity lifecycle, API-key lifecycle, and
  API-key policy configuration;
- GNK, ngonka, or internal treasury inventory as customer-facing balance;
- direct per-request reserve fallback for migrated paid cohorts;
- Redis, process memory, proxy-local balance rows, or local proxy reservations
  as customer-money authority;
- operator balance mutation or manual customer-credit adjustment. This scope
  allows readbacks only unless a later approved spec adds adjustment commands.

## Constraints

- Customer money is USD. API amount fields remain decimal strings; durable
  ledger/balance math remains exact USD atom arithmetic unless a later approved
  spec reopens amount representation.
- Billing-service PostgreSQL is the customer-money source of truth for migrated
  cohorts.
- Billing-issued microlease authority remains required before proxy can admit
  migrated paid external execution.
- A billing money operation has one stable idempotency key, one stable
  fingerprint, one stored outcome, and replay-stable readback.
- `request_id` is trace correlation only. It is not a settlement key. Usage
  settlement uses billing operation IDs, usage operation IDs, microlease IDs,
  child debit IDs, terminal outcome IDs, and qualified inference IDs.
- A timeout after possible acceptance is ambiguous. Callers must retry or
  read back with the same operation identity and must not mint a new money
  operation.
- Proxy local `balanceNgonka`, local in-memory reservations, and
  `BalanceTransaction` writes are historical or legacy-only for migrated
  cohorts after cutover. They must not become fallback money authority.
- No raw prompts, completions, SSE chunks, bearer tokens, API keys, DSNs,
  payment secrets, raw provider payloads, raw event payloads, dynamic proof
  URLs, or sensitive request bodies may enter billing APIs, events, logs,
  traces, metrics, durable rows, reconciliation notes, workflow artifacts, or
  proxy local cutover rows.

## Decisions

### D1. Authoritative Contract Shape

Decision: billing-service owns the canonical internal `/internal/billing/v1`
contract for account, balance, usage lifecycle, operation readback,
microlease, reconciliation, and admin readback capabilities.

The broader cutover contract is not the proxy's existing `/api/v1/usage/*`
shared-balance bridge. Technical design must replace or adapt the proxy client
to billing-service's internal contract instead of adding a long-lived
billing-service compatibility clone of the old proxy bridge.

The contract is both:

- microlease-first for migrated proxy paid admission;
- generic enough to expose account resolve, balance read, usage lifecycle,
  operation readback, and reconciliation/admin readbacks as billing-owned
  resources.

For migrated `gonka-proxy` paid cohorts, the only accepted request-path spend
authority mode is `billing_microlease_with_proxy_child_debit` or an equivalent
name selected by technical design. Generic usage reserve/finalize/write-off and
reversal commands must preserve that authority mode and must not provide a
hidden direct billing reserve fallback for migrated cohorts.

### D2. Account Resolve

Decision: the current cutover resolves proxy-authenticated users to billing
account scopes using `account_scope_key = user:<proxy_user_id>`.

`org:<subject_id>` remains a reserved account-scope shape already supported by
the durable schema, but organization charging is not activated by this cutover.
If technical design needs active organization semantics, specification must
reopen.

Account resolve must:

- require authenticated service caller context and represented user context;
- return the canonical account scope key, billing account ID, account state,
  import/migration state, and safe balance-read eligibility metadata;
- fail closed for missing, unimported, suspended, manual-review, ambiguous, or
  unsupported account state;
- never create customer credit or mutate balance as a side effect of paid
  request-path resolve;
- expose enough readback state for proxy to distinguish `not_found`,
  `import_required`, `suspended`, `manual_review`, `reconcile_required`,
  `not_ready`, and `dependency_not_ready`.

Technical design may choose whether account records are pre-created by an
import job or by an idempotent zero-balance account bootstrap command, but paid
usage cannot proceed until the account has an initialized billing balance and
import status that passes the cutover gates.

### D3. Balance Read

Decision: balance readbacks for migrated cohorts come from billing-service
ledger, balance, hold, microlease, and reconciliation state.

The balance read contract must return at least:

- `accountScopeKey`;
- settled USD;
- reserved USD;
- available USD;
- pending/import/reconciliation exposure when applicable;
- active usage holds and active microlease exposure;
- account balance version or equivalent read model version;
- stale operation, stale microlease, terminal lag, reconciliation backlog, and
  manual-review flags needed by proxy and operators;
- safe correlation identifiers, not public request bodies or secrets.

Active microlease exposure and active usage holds subtract from visible
available balance until billing has durable terminal, close, release, write-off,
reversal, or reconciliation proof. Expiry alone does not release money.

Proxy account and balance presenters for migrated cohorts must use billing
readback or a billing-derived projection. Proxy `balanceNgonka` and
`lockedRateUsd` may be retained only for legacy cohorts, historical display, or
import parity evidence.

### D4. Usage Reserve

Decision: usage reserve is a billing-owned money command that establishes or
confirms the maximum customer exposure for one paid usage operation.

For migrated proxy cohorts, reserve must be backed by billing-issued microlease
capacity plus durable proxy child debit lineage. It must not reserve directly
from account balance as a fallback path after microlease denial.

Reserve must require:

- stable idempotency key;
- immutable reserve fingerprint;
- `usageOperationId` or design-equivalent billing operation identity;
- `accountScopeKey`;
- pricing snapshot identity, fingerprint, policy version, decision time, and
  selector or use-class context;
- max authorized USD exposure;
- caller principal, represented user context, route contract version, deadline,
  trace/correlation metadata, and safe execution metadata;
- microlease or child-debit lineage for migrated proxy paid admission.

Reserve returns a stored outcome. Same key and same fingerprint returns the
stored outcome. Same key and different fingerprint returns conflict. Missing
account, missing idempotency, non-USD amount, stale pricing, insufficient
capacity, missing microlease authority, expired/cutoff authority, suspended
account, manual review, dependency outage, and worker/admission-health breach
all fail closed.

### D5. Usage Finalize, Write-Off, And Reversal

Decision: billing-service owns the terminal usage lifecycle and all ledger
effects for final charges, unused-capacity release, explicit write-off, and
reversal.

Finalize must:

- reference the prior usage operation and its reserve or microlease child debit
  lineage;
- require terminal fingerprint, canonical final charge in USD, qualified
  inference or execution evidence ID, pricing lineage, and safe terminal
  metadata;
- cap customer charge at the authorized usage or child debit exposure;
- release unused reserved exposure only from durable terminal or close proof;
- store the terminal outcome and return replay-stable readback.

Write-off must be explicit when external execution may have happened but the
customer cannot be charged within the authorized cap or when terminal evidence
is missing, stale, conflicting, or unsafe. Write-off is a ledger/reconciliation
effect, not a silent balance edit.

Reversal must be explicit compensation against a previously committed effect.
It must reference the original operation/effect, carry a stable idempotency key
and fingerprint, and preserve audit lineage.

Finalize, write-off, and reversal may be driven by synchronous internal HTTP,
durable events, or reconciliation workers as selected by technical design, but
all paths must converge on the same billing operation, idempotency, ledger, and
stored-outcome authority.

### D6. Operation Readback

Decision: billing-service must expose operation readback for all cutover money
operations and ambiguous outcomes.

Operation readback must support account-bound lookup by billing operation ID,
usage operation ID, microlease ID, child debit ID, terminal outcome ID, or other
design-approved safe operation identity. It must not use public `request_id` as
the settlement lookup key.

Readback returns:

- operation kind;
- current state;
- stored outcome or conflict reason;
- retryability;
- account scope;
- safe pricing/execution lineage;
- failure class;
- reconciliation case link when one exists;
- balance version or affected ledger effect identifiers when safe.

The OpenAPI/service-auth mismatch found by research is decided here:
`/internal/billing/v1/operations/readback` requires the
`billing.operations.read` scope. Technical design and implementation must align
generated contract scopes, middleware constants, tests, and proxy caller scopes
to that decision.

### D7. Reconciliation And Admin Readbacks

Decision: this cutover requires read-only reconciliation and admin readbacks
before migrated paid cohorts can be considered production-ready.

Readbacks must cover:

- account ledger history and balance version history;
- active usage holds and active microlease exposure;
- stale reserve, stale microlease, stale child debit, missing terminal,
  ambiguous terminal, conflict, write-off, reversal, and reconciliation cases;
- import and shadow/parity state for moved proxy balances;
- worker lag, terminal lag, outbox backlog, inbox retry/quarantine, and
  admission-control state.

Admin/readback surfaces are internal, protected, scoped, support-safe, and
privacy constrained. They do not include top-up/payment evidence handling,
payment-provider payloads, or operator credit mutation in this scope.

### D8. Service Authentication And Trust Boundary

Decision: production cutover uses billing-service scoped JWT/JWKS service auth.
The current proxy `BILLING_SERVICE_AUTH_KEY` bearer-key bridge is not accepted
as the production authentication model for migrated money authority.

Required scope model:

- account resolve: `billing.accounts.resolve`;
- balance read: `billing.balances.read`;
- usage reserve/finalize/write-off/reversal: `billing.usage.write`;
- usage and operation readback: `billing.usage.read` and
  `billing.operations.read` as design assigns per route;
- microlease issue/close: `billing.microleases.write`;
- microlease readback: `billing.microleases.read`;
- reconciliation readback: `billing.reconciliation.read`;
- admin ledger/readback: `billing.admin.read`.

Technical design may refine exact names, but it must preserve separate read and
write authority and must fix the current operation-readback scope mismatch.

Every protected call must bind:

- authenticated service principal;
- represented user or account context;
- contract version;
- deadline;
- trace/correlation ID;
- safe caller and operation metadata;
- account scope in body or path according to generated OpenAPI rules.

Bearer tokens, API keys, raw customer prompts, raw model output, SSE chunks,
payment secrets, and raw provider payloads are never accepted as billing
metadata.

### D9. HTTP Runtime And Worker Ownership

Decision: billing-service owns real runtime backing for every contract it
exposes.

Technical design must replace handler-level nil-service behavior for enabled
cutover routes with concrete app services wired through bootstrap. A route may
return fail-closed `503` only when the runtime is intentionally disabled,
unhealthy, or dependency admission fails.

The billing-worker command must wire real tasks before rollout:

- terminal event consumers;
- checkpoint consumers;
- close consumers;
- inbox retry;
- outbox relay;
- stale reserve/microlease/child-debit reconciliation;
- admission control renewal and closure;
- reconciliation/admin readback maintenance.

No-op worker tasks are acceptable only when the worker/runtime feature is
disabled and readiness fails closed for migrated paid cohorts. They are not an
enabled production runtime.

Billing-service owns Redpanda consumer groups, billing inbox/outbox state,
quarantine, redrive, stored event fingerprints, and billing-side reconciliation.
Proxy owns durable child debit and terminal obligation creation before external
execution, then publishes terminal/checkpoint/close evidence through the
approved contract.

### D10. Proxy Cutover Scope

Decision: production cutover is not complete until `gonka-proxy` live paid
completion and web-search paths use billing-service authority for migrated
cohorts and stop mutating proxy-local money state for those cohorts.

The overall workflow scope includes the required proxy behavioral change and
proof. Later planning must choose one of two explicit execution routes:

- include approved cross-repo `gonka-proxy` implementation tasks and proof in
  the ledger; or
- if cross-repo writes are not authorized in that session, stop with a precise
  proxy implementation handoff and do not claim cutover completion.

For migrated cohorts:

- local `User.balanceNgonka` writes are disabled for paid usage;
- local in-memory reservations are not spend authority;
- local `BalanceTransaction` writes are historical, analytics, or migration
  evidence only;
- old shared-balance `/api/v1/usage/*` calls are replaced or adapted to the
  billing-service internal contract;
- direct per-request reserve fallback is disabled;
- missing billing capacity, stale microlease, expired/cutoff authority, worker
  backlog, ambiguous outcome, or missing operation readback fails paid admission
  closed.

Legacy cohorts may continue using current proxy-local behavior until they are
explicitly migrated. Mixed-mode support must prevent dual writes for any one
account scope.

### D11. Fail-Closed Behavior

Decision: migrated paid request admission fails closed unless billing authority
is proven before external execution.

Fail closed means no paid external model/devshard execution starts for the
migrated account scope. Public proxy surfaces must map the internal failure to
the existing OpenAI-compatible error contract, but they must not mask the
billing failure as successful unpaid or locally charged usage.

Fail-closed conditions include:

- billing-service unavailable, timeout, missing runtime, or failed readiness;
- ambiguous reserve/finalize/write-off/reversal outcome without successful
  same-identity readback;
- account resolve failure, missing account, import required, suspended account,
  manual review, or reconciliation required;
- missing or invalid service JWT, missing route scope, account binding mismatch,
  or represented-user mismatch;
- missing idempotency key, changed fingerprint, unsupported currency,
  non-canonical amount, stale pricing, or missing pricing lineage;
- insufficient balance or reserved capacity;
- missing, stale, expired, cutoff, over-cap, or conflicting microlease/child
  debit lineage;
- worker lag, terminal lag, inbox/outbox backlog, quarantine, or stale
  reconciliation outside configured gates;
- privacy metadata rejection.

After a possible accepted timeout, proxy must retry with the same operation
identity or call operation readback. It must not create a new billing operation
or fall back to proxy-local balance mutation.

### D12. Validation Obligations

Decision: future readiness proof must cover billing-service and proxy cutover
surfaces with fresh command evidence. Prior microlease validation is useful
context but is not enough for this broader authority cutover.

Required proof classes:

- OpenAPI generation, drift, runtime contract, lint, and validation for the
  new account, balance, usage, operation, reconciliation, admin, and scope
  surfaces;
- service-auth tests for JWT/JWKS validation, route scopes, account binding,
  represented user context, 401/403 mapping, and operation-readback scope
  alignment;
- SQLC, migration, transaction, row-lock, idempotency, stored outcome,
  ledger-effect, and rollback tests for account, balance, usage, microlease,
  and reconciliation data;
- app/domain tests for USD atom parsing/formatting, non-negative balance,
  reserve/finalize/write-off/reversal invariants, replay/conflict handling,
  stale/ambiguous operation rules, and active exposure conservation;
- HTTP tests for status/result mapping, body/path identifier rules,
  ambiguous-timeout readback, bounded Problems, low-cardinality route labels,
  and no sensitive payload leakage;
- worker and Redpanda tests for terminal/checkpoint/close consume, inbox retry,
  outbox relay, quarantine/redrive, committed DB effect before offset commit,
  worker readiness, bounded concurrency, and signal-aware shutdown;
- reconciliation/admin readback tests for stale operation visibility,
  import/parity state, ledger history, and support-safe metadata;
- proxy targeted tests for completion and web-search migrated cohorts proving
  no local spend authority, durable child debit before external execution,
  terminal obligation, same-identity retry/readback, direct reserve fallback
  disablement, legacy cohort isolation, and no dual writer;
- performance benchmarks at least matching the approved microlease budgets:
  billing issue/replenish p95 under 100 ms and p99 under 250 ms, proxy durable
  child allocation p95 under 10 ms and p99 under 25 ms, cold replenishment p95
  under 250 ms and p99 under 500 ms, and first-token added latency p95 under
  25 ms unless technical design records stricter or updated budgets;
- privacy/security proof using `rtk make go-security`, `rtk make secret-scan`,
  targeted privacy assertions, and proxy proof selected under proxy repository
  rules;
- repository-owned billing validation using `rtk make check`,
  `rtk make openapi-check`, `rtk make sqlc-check`,
  `rtk make migration-validate` or Docker equivalent, targeted integration
  tests, and `rtk make check-full` when Docker and local context permit.

## Formal Clarification Gate Reconciliation

Formal clarification was required because this is full-orchestrated
protected-money work touching internal API contracts, persisted state, service
auth, worker/runtime behavior, performance, validation, rollout, and cross-repo
proxy cutover.

Method: local read-only formal clarification using the
`spec-clarification-challenge` rubric.

Scoped-down rationale: the available multi-agent tool says `spawn_agent` may be
used only when the user explicitly asks for subagents, delegation, or parallel
agent work. The user requested a specification-only phase and did not authorize
subagents, so the orchestrator ran the formal clarification locally across the
default five lenses and recorded the reconciliation here and in
`workflow-plan.md`.

| Lens | Strongest question | Resolution |
| --- | --- | --- |
| Scope and spec coherence | Does the spec decide the broader contract without reopening top-ups or replacing the approved microlease authority model? | Yes. Top-ups/payment-service stay out of scope; migrated proxy paid admission remains microlease-first; broader account, balance, usage, and readback capabilities are billing-owned. |
| Domain invariants and edge cases | Can account resolve, balance read, generic usage reserve, or fallback behavior create dual money authority? | No. Account resolve does not credit accounts; balance read derives from billing state; migrated reserve requires microlease/child debit authority; proxy-local writes are legacy/historical only for migrated cohorts. |
| Architecture ownership and dependency boundaries | Does any enabled billing route or worker role lack an owner? | The spec makes billing-service responsible for concrete HTTP app services and real worker tasks before rollout. Nil handlers and no-op worker tasks are allowed only as disabled fail-closed state. |
| API, data, compatibility, and source of truth | Is the proxy shared-balance bridge mismatch resolved enough for design? | Yes. The canonical provider contract is billing-service `/internal/billing/v1`; the old proxy `/api/v1/usage/*` bridge must be replaced or adapted, and operation readback scope is `billing.operations.read`. |
| Security, reliability, delivery, and validation proof | Are auth, fail-closed behavior, rollout proof, and validation obligations explicit enough for technical design? | Yes. The spec chooses scoped JWT/JWKS service auth, enumerates fail-closed conditions, requires worker/readiness gates, and names billing plus proxy proof classes. |

Gate status: complete.

No unresolved clarification blocker remains for specification approval. Reopen
specification if technical design requires direct per-request reserve fallback,
proxy-local money writes for migrated cohorts, non-JWT bearer-key production
auth, top-up/payment ownership, organization charging, Redis/memory spend
authority, weaker privacy policy, or a worker/runtime shape that cannot fail
closed.

## Open Questions / Assumptions

- [defer_to_design] Exact route names, schemas, field names, Problem/result
  envelopes, generated type names, status-code mapping, and package layout
  belong to technical design and OpenAPI authoring. They must preserve the
  decisions above.
- [defer_to_design] Technical design must decide whether generic usage commands
  are synchronous HTTP commands, event-ingestion commands, or both. All paths
  must converge on the same billing operation, ledger, idempotency, and stored
  outcome authority.
- [defer_to_design] Technical design must define the account import/parity flow
  and the exact point where proxy `balanceNgonka` becomes historical/read-only
  for a migrated account scope.
- [defer_to_design] Technical design must choose concrete worker topology,
  Redpanda topic names, consumer groups, retry/backoff budgets, readiness
  gates, and reconciliation SLAs.
- [defer_to_design] Technical design must choose whether proxy changes are
  included in this repository's eventual implementation ledger as approved
  cross-repo tasks or emitted as an exact proxy handoff. Cutover completion
  cannot be claimed without proxy proof.
- [assumption] Current proxy user IDs are stable enough for
  `account_scope_key=user:<proxy_user_id>` during the first cutover. Reopen
  specification if identity ownership requires organization-first,
  tenant-scoped, or multi-subject account attribution.
- [assumption] Pricing-service can continue to provide USD-compatible immutable
  pricing snapshot identity, fingerprint, policy version, decision time, and
  selector/use-class context. Reopen specification if that evidence cannot be
  persisted before money lineage.
- [reopen_spec_if_false] Reopen specification if web-search paid execution
  cannot be represented as microlease-backed usage reserve, terminal
  finalize/write-off/reversal, and reconciliation without proxy-local fallback.
- [reopen_spec_if_false] Reopen specification if migrated public proxy error
  mapping cannot preserve OpenAI-compatible error shape while failing paid
  admission closed.

## Task Breakdown / Handoff Link

Next phase: technical design.

Expected technical-design output:

- task-local `design/` bundle for account resolve, balance read, usage
  lifecycle, operation readback, reconciliation/admin readbacks, service auth,
  worker/runtime wiring, proxy cutover, fail-closed behavior, validation, and
  rollout;
- explicit relationship between generic usage commands and the approved
  microlease-first paid-admission model;
- OpenAPI/source-of-truth design for `/internal/billing/v1` capabilities and
  route scopes, including the operation-readback scope fix;
- data model and transaction design for account import, balance read model,
  usage operations, holds, terminal outcomes, write-offs, reversals,
  idempotency, stored outcomes, ledger entries, reconciliation cases, and
  active microlease exposure;
- sequence/failure design for account resolve, balance read, reserve,
  finalize, write-off, reversal, readback, ambiguous timeout, worker replay,
  reconciliation, and proxy cutover;
- service-auth design for scoped JWT/JWKS, proxy token issuance or acquisition,
  account binding, represented user context, and failure mapping;
- worker/runtime design for concrete HTTP app services, billing-worker
  Redpanda tasks, inbox/outbox, admission renewal, stale repair, readiness,
  and shutdown;
- proxy integration design or exact proxy handoff strategy;
- triggered `test-plan.md` and `rollout.md` design inputs, because validation
  and mixed-version cutover are too broad for `tasks.md` alone.

Technical design must not write `tasks.md`, code, migrations, generated files,
tests, or implementation handoff. Planning is blocked until technical design
and mandatory technical design review complete with `PASS` or eligible
`CONCERNS`.

## Validation

Implementation validation recorded on 2026-06-02:

- Billing contract/generated proof passed with temporary-index drift checks for
  `rtk make openapi-check`, `rtk make proto-check`, and
  `rtk make sqlc-check`.
- Billing runtime, worker, repository, event, and integration proof passed with
  `rtk make check`, `rtk make migration-validate`,
  `rtk go test -tags=integration ./test -count=1`, targeted package tests for
  HTTP/bootstrap/app/Postgres/Redpanda/worker surfaces, `rtk make go-security`,
  and `rtk make secret-scan`.
- Billing performance proof passed the approved microlease budgets with
  integration p95/p99 measurements for issue/replenish, terminal ingestion,
  checkpoint/close cadence, cold replenishment, stale reconciliation scan, and
  account contention.
- Proxy targeted cutover proof passed with `rtk bun test` over shared-balance
  authority, balance local-writer disablement, web-search maintenance cutover,
  durable microlease allocator, and pricing/API-key lineage tests.
- Proxy durable allocation performance proof passed the approved p95/p99
  budgets without Redis or memory spend authority.
- Coverage remediation proof passed after focused authority/runtime behavioral
  tests raised the repository coverage gate from starting output
  `coverage 51.60% is below threshold 80.00%` to final `rtk make test-report`
  success and `rtk make coverage-check` output
  `coverage 80.00% meets threshold 80.00%`. The final proof also passed
  `rtk make fmt` and `rtk make check`.

Remaining blockers:

- Proxy `rtk bun run typecheck` is blocked by unrelated existing TypeScript
  errors outside billing cutover files:
  `src/errors/normalization/api/api-error-formatter.ts`,
  `src/middleware/anthropic-api-key-credential-extractor.ts`,
  `src/routes/dashboard/activity-usage/_shared/numbers.ts`,
  `src/routes/dashboard/activity-usage/plugins/activity-usage-workaround.plugin.ts`,
  `src/services/admin-user.service.ts`,
  `src/services/api-keys/api-key-mutation-parser.ts`, and
  `src/utils/max-tokens.ts`.

Privacy evidence: `rtk make go-security`, `rtk make secret-scan`, and a targeted
privacy `rg` scan found no prohibited raw prompts, completions, SSE chunks,
bearer tokens, API keys, DSNs, payment secrets, raw provider payloads, raw event
payloads, dynamic proof URLs, or sensitive request bodies in the changed
billing surfaces or task-local artifacts.

## Outcome

Implementation completed on 2026-06-02 for the approved balance and usage
authority cutover ledger.

Billing-service now owns internal account, balance, usage lifecycle, operation
readback, reconciliation, and admin-readback authority for migrated paid
cohorts. Migrated request-path spend authority remains
`billing_microlease_with_proxy_child_debit`; direct per-request reserve
fallback, proxy-local money writes for migrated cohorts, Redis/memory spend
authority, top-ups, payment-service integration, organization charging, and
operator credit mutation remain out of scope.

Proxy cutover work is included for the ledger-owned seams: scoped service JWT
calls to billing `/internal/billing/v1`, `billing.operations.read` operation
readback, durable microlease child-debit lineage, local proxy money-writer
disablement while shared-balance cutover is enabled, and web-search
money-touching maintenance fail-closed behavior. Proxy broad typecheck remains
blocked by unrelated existing errors listed in Validation.

The billing repository-owned coverage blocker is resolved at the current 80.00%
threshold. Billing proof, proxy targeted proof, privacy/security proof,
migration/contract/generated checks, and performance budget proof listed above
are current. Proxy broad typecheck remains the only recorded non-billing
blocker.
