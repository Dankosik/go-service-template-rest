# L3-L5 API, Data, And Distributed Inputs

L3 question: What internal REST resources, idempotency rules, status mappings,
generated contracts, and compatibility constraints are required before spec
approval?

L4 question: What source-of-truth, schema, SQLC, transaction, retention, and
reconciliation questions must be decided for durable balance and usage
authority?

L5 question: What saga, inbox/outbox, Redpanda, retry, replay, ordering, and
ambiguous-outcome semantics are required across proxy and billing?

## API And Compatibility Findings

- Billing-service OpenAPI is the generated source for service HTTP bindings
  (`docs/repo-architecture.md` and `api/openapi/service.yaml`). Current routes
  are internal microlease/readback routes only.
- Proxy's older internal-money billing contract includes top-up, usage reserve,
  usage finalize, usage write-off, and treasury admission schemas
  (`/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/billing/v1/index.ts:25-39`,
  `:503-719`, `:958-1026`).
- Proxy's live shared-balance bridge calls JSON endpoints under `/api/v1/*`,
  while billing-service exposes `/internal/billing/v1/*` microlease routes.
  This is a concrete consumer/provider contract mismatch.
- Billing-service OpenAPI declares `/internal/billing/v1/operations/readback`
  with `x-route-scopes: [billing.operations.read]`
  (`api/openapi/service.yaml:228-260`), but the service-auth middleware maps
  that route to `billing.microleases.read`
  (`internal/infra/http/service_auth.go:292-307`).
- Existing microlease route validation enforces route contract IDs, contract
  version, enum values, pricing snapshot identity, idempotency key,
  accountScopeKey, trace request ID, safe metadata, and caller context
  (`internal/infra/http/handlers.go:176-317`).
- Proxy internal-money usage schemas use fields like `clientUsageRequestId`,
  `accountId`, `requestBasisFingerprint`, `terminalBasisFingerprint`,
  `qualifiedInferenceEvidence`, and result codes such as `payload_conflict`,
  `manual_review`, `reconcile_required`, and `not_ready`
  (`/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/billing/v1/index.ts:503-719`).
- Billing-service microlease schemas use microlease-specific idempotency,
  account scope, proxy allocator ownership, cap, fence, pricing snapshot, and
  deadline fields (`api/openapi/service.yaml:404-648`).

## Data Authority Findings

- `billing_accounts.account_scope_key` is unique and constrained to
  `user:<subject_id>` or `org:<subject_id>` for the current account types
  (`env/migrations/000003_billing_money_core.up.sql:1-23`).
- `account_balances` stores USD atoms and enforces non-negative settled,
  reserved, available, pending amounts, plus `available = settled - reserved`
  (`env/migrations/000003_billing_money_core.up.sql:28-54`).
- `idempotency_records` and `operation_outcomes` provide durable replay and
  stored outcome identity
  (`env/migrations/000003_billing_money_core.up.sql:56-113`).
- `usage_operations`, `usage_holds`, and `usage_terminal_outcomes` already model
  reserve/finalize/write-off/reversal/compensation lifecycle data, but research
  found generated SQLC and integration tests more than a live HTTP/app service
  for those commands (`env/migrations/000003_billing_money_core.up.sql:115-238`,
  `internal/infra/postgres/sqlcgen/billing_money_core.sql.go`).
- Ledger entries are append-only for money fields and carry USD atom deltas,
  balance versions, operation linkage, and safe metadata
  (`env/migrations/000003_billing_money_core.up.sql:385-562`).
- Microlease schema adds event inbox/outbox, admission controls,
  `spending_microleases`, `microlease_child_debits`, checkpoints, and
  reconciliation links (`env/migrations/000004_billing_microleases.up.sql:172-515`).
- Microlease repository issue and close paths lock account balances and perform
  reserve/release ledger and balance updates inside short transactions
  (`internal/infra/postgres/microlease_repository.go:184-500`).

## Distributed Flow Findings

- Microlease event protobuf defines terminal submitted, checkpoint reported,
  close reported, issued, terminal applied, closed, and admission rejected event
  shapes with safe lineage fields, fingerprints, identities, and no raw prompt
  body fields (`api/proto/events/v1/microlease_events.proto:7-120`).
- Redpanda terminal consumer applies or quarantines terminal events and commits
  offsets only after apply/quarantine behavior in the adapter code and tests
  (`internal/infra/redpanda/consumer.go`, `internal/infra/redpanda/adapter_test.go`).
- Redpanda outbox relay claims events, validates fingerprints, produces, and
  marks publish/retry via adapter code (`internal/infra/redpanda/outbox.go`).
- The billing worker runtime framework requires terminal consumer, checkpoint
  consumer, close consumer, inbox retry, outbox relay, stale reconciliation, and
  admission control renewal roles
  (`internal/app/microleaseworker/worker.go:16-24`).
- The current worker bootstrap wires those roles to no-op tasks rather than
  Redpanda/repository adapters, so distributed runtime behavior is not live
  from the command entrypoint
  (`cmd/billing-worker/internal/bootstrap/run.go:46-96`).
- Proxy completion paths currently run local reserve/deduct directly around
  external execution. The durable microlease allocator can require a durable
  child debit and terminal obligation before execution, but it is not integrated
  into those live paths in the inspected source tree.

## Must-Decide-Now Inputs For Specification

- Contract shape: microlease-first, generic usage lifecycle, or both.
- Route compatibility: whether to adapt proxy to billing-service
  `/internal/billing/v1/*`, add billing-service `/api/v1/*` compatibility, or
  explicitly replace the existing proxy shared-balance bridge.
- Account identity: whether proxy `userId` maps directly to
  `account_scope_key=user:<userId>` and how account resolve/readback handles
  missing/importing/suspended accounts.
- Idempotency: exact key and fingerprint ownership for reserve, finalize,
  write-off, reversal, microlease issue/close, child debit, terminal, checkpoint,
  and readback.
- Outcome mapping: HTTP statuses, route result codes, stored outcome readbacks,
  retryable versus terminal failures, and OpenAI-compatible proxy error mapping.
- Data write authority: when proxy local money tables become historical/read-only
  for migrated cohorts and which billing tables are the source of truth.
- Distributed ownership: which component publishes terminal/checkpoint/close
  events, who owns retry/quarantine/inbox rows, and what ambiguous outcomes
  block further paid execution.
