# Redpanda Event Contract Design

Status: repaired review-ready technical design for billing-issued spending leases
Authority note: design context only. Runtime event schema authority is
`api/proto/events/v1/*.proto`.
Consumes: `../sequence.md`, `../data-model.md`

## Scope

Current-scope consumed topics:

| Topic | Producer | Consumer group | Partition key | Purpose |
| --- | --- | --- | --- | --- |
| `usage.execution.terminal.v1` | `gonka-proxy` durable terminal submission relay | `billing-service.usage-terminal.v1` | `spendingLeaseId` or `accountScopeKey` | Finalize, write off, reverse, or compensate one child debit under a billing-issued lease. |
| `usage.lease.checkpoint.v1` | `gonka-proxy` durable lease allocator relay | `billing-service.lease-checkpoint.v1` | `spendingLeaseId` | Progress, close, cancel, or expiry proof for one lease generation. |
| `billing.reconciliation.command.v1` | billing admin/reconciliation tooling | `billing-service.reconciliation-command.v1` | `accountScopeKey` when account scoped, else `reconciliationCaseId` | Authorized repair/redrive command referencing existing durable billing state. |

Current-scope emitted topics:

| Topic | Producer | Partition key | Purpose |
| --- | --- | --- | --- |
| `billing.lease.outcome.v1` | billing outbox relay | `spendingLeaseId` | Lease issue/replenish/close/cancel outcome fact. |
| `billing.debit.settlement.v1` | billing outbox relay | `spendingLeaseId` or `usageOperationId` | Child debit settlement outcome fact. |
| `billing.ledger.effect.v1` | billing outbox relay | `accountScopeKey` | Derived committed ledger effect fact. |
| `billing.operation.outcome.v1` | billing outbox relay | `usageOperationId` or source operation identity | Derived operation outcome fact. |
| `billing.reconciliation.required.v1` | billing outbox relay | `accountScopeKey` or `reconciliationCaseId` | Operator/proxy visibility into durable repair cases. |
| `billing.ingestion.rejected.v1` | billing outbox relay | valid `eventId` else `eventReceiptIdentity` | Safe rejection, quarantine, or conflict signal. |

Out of current scope:

- reserve command/outcome topics for lease issuance;
- payment/top-up evidence topics;
- bearer spend-token or allowance-window topics.

## Runtime Schema Authority

Runtime Redpanda event schemas are authored as protobuf contracts under:

```text
api/proto/events/v1/
```

Required initial proto inputs:

- `usage_execution_terminal.proto` for `usage.execution.terminal.v1`.
- `usage_lease_checkpoint.proto` for `usage.lease.checkpoint.v1`.
- `billing_lease_outcome.proto` for `billing.lease.outcome.v1`.
- `billing_debit_settlement.proto` for `billing.debit.settlement.v1`.
- `billing_ledger_effect.proto` for `billing.ledger.effect.v1`.
- `billing_operation_outcome.proto` for `billing.operation.outcome.v1`.
- `billing_reconciliation_required.proto` for
  `billing.reconciliation.required.v1`.
- `billing_ingestion_rejected.proto` for `billing.ingestion.rejected.v1`.
- `billing_reconciliation_command.proto` for authorized repair/redrive
  commands.
- a shared envelope/common types file for event identity, schema version,
  producer authority, account scope, lease/debit lineage, fingerprints, safe
  correlation, USD atom/decimal amount pairs, and safe problem codes.

Proto package and Go package convention should follow the existing
`api/proto/service/v1` pattern: use package `billing.events.v1` and a
module-aligned `go_package` under
`github.com/Dankosik/billing-service/internal/api/events/v1`.

Derived surfaces:

- generated Go event DTOs under a repository-owned internal generated package
  such as `internal/api/events/v1`;
- Redpanda codec/adapters under `internal/infra/redpanda` map generated DTOs
  to `internal/app/money` command/fact types;
- app logic must not treat generated protobuf structs as business source of
  truth.

Validation and generation flow:

- planning must add repository-owned proto lint/generate/drift checks before
  runtime producers or consumers are implemented;
- compatibility checks must reject in-place breaking changes to event identity,
  amount meaning, finality, required proof, producer authenticity, and replay
  semantics;
- additive optional fields may stay in the same proto/topic version only when
  old consumers can ignore them safely and semantic validation remains
  unchanged;
- breaking semantic changes require a new versioned proto package/topic.

Privacy-safe fixtures must use synthetic IDs, bounded safe amounts, safe
problem codes, and fingerprints only. They must not include raw prompts,
completions, SSE chunks, bearer tokens, API keys, DSNs, raw provider payloads,
payment secrets, dynamic provider proof URLs, or raw event payload dumps from
production.

## Common Envelope

Every processable event carries:

- `eventId`
- `eventSchemaVersion`
- `eventFingerprint`
- `producerAuthority`
- `producerInstanceId` or safe producer reference
- `producedAt`
- trace ID or safe trace context
- `accountScopeKey` when account-scoped
- `spendingLeaseId`, generation/fence, and `proxyLeaseOwnerId` when
  lease-scoped
- `debitAuthorizationId` and `usageOperationId` when child-debit-scoped
- payload fingerprint fields for semantic replay

Billing-computed receipt fields:

- `topic`
- `partition`
- `offset`
- `eventReceiptIdentity`
- `receivedAt`

Poison events with malformed or missing producer `eventId` or semantic
identity are recorded by broker-coordinate receipt identity. They cannot mutate
money.

## Producer Authenticity

Required controls:

- Redpanda ACLs allow only `gonka-proxy` to produce
  `usage.execution.terminal.v1` and `usage.lease.checkpoint.v1`.
- Redpanda ACLs allow only billing-service outbox relay to produce
  `billing.*.v1` facts.
- Consumer validates topic-to-producer authority allowlist before business
  processing.
- Redrive or repair commands require operator/admin authority and durable audit.

Envelope producer identity is checked against topic allowlists. It is not a
substitute for broker ACLs.

If a future topic needs multiple producer authorities or untrusted network
producers, technical design must reopen security for envelope signing or an
equivalent stronger authenticity proof.

## `usage.execution.terminal.v1`

Purpose: terminal fact for one child debit authorization under a
billing-issued lease.

Required fields:

- common envelope fields;
- `accountScopeKey`;
- `spendingLeaseId`;
- `spendingLeaseGeneration` / `leaseFence`;
- `proxyLeaseOwnerId`;
- `debitAuthorizationId`;
- `usageOperationId`;
- `clientUsageRequestId`;
- `childCapUsd`;
- `terminalKind`: `finalize`, `write_off`, `reversal`, or `compensation`;
- `operationFingerprint`;
- `terminalFingerprint`;
- `pricingSnapshotId`;
- `pricingSnapshotFingerprint`;
- `terminalObservedAt`;
- `proxyTerminalSubmissionId`;
- `proxyTerminalSubmissionFingerprint`;

For `finalize`:

- `meteredFactsFingerprint`;
- final charge USD;
- base execution USD when available;
- fee/rate policy versions;
- optional qualified inference evidence: `inferenceId`, `providerFamily`,
  `verificationSurface`, `proofScope`, `evidenceFingerprint`, and optional safe
  provider proof reference.

For `write_off`:

- `writeOffReasonCode`;
- `evidenceGapClass`;
- `cancellationOrTimeoutFactsFingerprint`;
- `usageWriteOffPolicyVersion`;
- optional explicit write-off USD or release-only classification.

Forbidden fields:

- raw prompt;
- raw completion;
- SSE chunk;
- raw provider response;
- API key or bearer token;
- DSN;
- payment secret;
- raw webhook body;
- dynamic `verifyUrl` to be dereferenced by billing.

Billing may store a safe proof reference or fingerprint for evidence locators.
It must not dereference dynamic URLs unless a later security design restricts
them to fixed provider allowlists.

## `usage.lease.checkpoint.v1`

Purpose: progress, close, cancel, or expiry evidence for one proxy-held lease
generation.

Required fields:

- common envelope fields;
- `accountScopeKey`;
- `spendingLeaseId`;
- `spendingLeaseGeneration` / `leaseFence`;
- `proxyLeaseOwnerId`;
- `checkpointSequence`;
- `checkpointKind`: `progress`, `close_requested`, `cancel_requested`,
  `expired_scan`, or `operator_repair`;
- `allocatedChildCapSumUsd`;
- `terminalSubmittedChildCapSumUsd`;
- `localRemainingUsd`;
- `openChildDebitCount`;
- `oldestOpenChildDeadlineAt` when open children exist;
- `checkpointFingerprint`;
- `observedAt`.

Billing releases unused capacity only after validating lineage, monotonic
sequence, fingerprint, and release safety. Incomplete proof opens
reconciliation.

## Terminal And Checkpoint Outcomes

| Event condition | Billing outcome |
| --- | --- |
| Same `(topic, eventId)` and same fingerprint | Stored inbox outcome replay. |
| Same `(topic, eventId)` with changed fingerprint | Inbox conflict, rejected fact emitted, no money mutation. |
| Same debit ID and same operation/terminal fingerprint | Stored child/terminal outcome replay. |
| Finalize for valid child cap under original lease | Charge capped at child and lease authority, release unused child capacity, store settlement. |
| Write-off/release for valid child cap | Release or write off explicitly, store settlement. |
| Terminal after lease expiry | Settle against original lease/debit authority if lineage is valid. |
| Missing, stale, or invalid lease/debit lineage | Reconciliation, no customer charge beyond verified authority. |
| Proxy over-debit beyond lease budget | Cap customer charge at valid lease authority and open reconciliation/write-off for excess. |
| Checkpoint close with incomplete proof | Keep disputed capacity reserved and open reconciliation. |

## Emitted Billing Facts

### `billing.lease.outcome.v1`

Derived from `spending_leases` and stored outcomes.

Required fields:

- `spendingLeaseId`;
- `accountScopeKey`;
- `proxyLeaseOwnerId`;
- generation/fence;
- outcome kind;
- issued/replenished/released/reserved USD amounts;
- state;
- stored outcome identity;
- ledger reference when applicable;
- safe problem code when rejected/conflicted.

### `billing.debit.settlement.v1`

Derived from child debit settlement lineage.

Required fields:

- `spendingLeaseId`;
- `debitAuthorizationId`;
- `usageOperationId`;
- terminal kind;
- child cap USD;
- charged/released/write-off USD amounts;
- terminal outcome identity;
- settlement effect identity when present;
- safe reconciliation reference when required.

### `billing.ledger.effect.v1`

Derived from `ledger_entries`.

Required fields:

- `ledgerEntryId`;
- `settlementEffectId` when present;
- `accountScopeKey`;
- `effectType`;
- USD atom and decimal amount fields;
- after-balance version/reference;
- source operation identity;
- `spendingLeaseId` and `debitAuthorizationId` when relevant;
- `createdAt`;
- event fingerprint.

### `billing.operation.outcome.v1`

Derived from `operation_outcomes`.

Required fields:

- `storedOutcomeId`;
- `operationKind`;
- `outcomeStatus`;
- `primaryResourceType`;
- `primaryResourceId`;
- `spendingLeaseId`, `debitAuthorizationId`, or `usageOperationId` when
  relevant;
- safe failure class/problem code;
- account scope when account-scoped.

### `billing.reconciliation.required.v1`

Derived from `reconciliation_cases`.

Required fields:

- `reconciliationCaseId`;
- `reason`;
- `severity`;
- `state`;
- `accountScopeKey`;
- lease/debit/usage lineage when relevant;
- safe failure class/problem code;
- `openedAt` or `updatedAt`.

### `billing.ingestion.rejected.v1`

Derived from `billing_event_inbox`.

Required fields:

- `eventInboxId`;
- `eventReceiptIdentity`;
- valid producer `eventId` when available;
- `topic`, `partition`, `offset`;
- safe rejection class;
- safe problem code;
- operation identity when valid.

No rejected event fact includes raw payload.

## Retention And Lag

- `usage.execution.terminal.v1`: minimum 14 days hot replay.
- `usage.lease.checkpoint.v1`: minimum 14 days.
- `billing.reconciliation.command.v1`: minimum 14 days.
- `billing.lease.outcome.v1`: minimum 30 days or copied to downstream owned
  store.
- `billing.debit.settlement.v1`: minimum 30 days or copied to downstream owned
  store.
- `billing.ledger.effect.v1`: minimum 30 days or copied to downstream owned
  store.
- `billing.operation.outcome.v1`: minimum 14 days.
- `billing.reconciliation.required.v1`: minimum 30 days.
- `billing.ingestion.rejected.v1`: minimum 30 days.

Initial critical terminal lag budgets:

- warning: oldest unprocessed terminal/checkpoint event age above 60 seconds
  for 5 minutes;
- critical: oldest unprocessed terminal/checkpoint event age above 5 minutes
  for 5 minutes;
- stale lease/debit reconciliation eligibility no later than 5 minutes after
  lease expiry, child terminal deadline, or configured terminal-lag breach.

## Contract Evolution

- Additive optional fields may stay in the same topic version only when old
  consumers can ignore them safely.
- Changes to identity, amount meaning, finality, required proof, auth model, or
  replay semantics require a new versioned proto package/topic.
- Payment/top-up topics require specification reopen.
- Lease reserve command/outcome topics over Redpanda require specification
  reopen.
