# billing-money-core Technical Design Review

Phase: technical design review
Status: current sign-up-bonus and usage-only follow-up review complete with PASS
Latest verdict: PASS for repaired current sign-up-bonus and usage-only packet
Previous data-model verdict: PASS (historical follow-up)
Previous Redpanda verdict: PASS (historical follow-up; superseded for current
planning)
Reopen target: none
Next phase: planning for current sign-up-bonus and usage-only money-core ledger
Owner: orchestrator
Review type: local read-only technical design review with `go-design-review`
boundary/readiness lens

## Prior Review: Redpanda Event-Ingestion Addendum

Reviewed on 2026-06-01 for
`specs/billing-money-core/design/event-ingestion-redpanda.md`.

Read first and used as authority:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `specs/billing-money-core/workflow-plan.md`
- `specs/billing-money-core/workflow-plans/technical-design.md`
- `specs/billing-money-core/spec.md`
- `specs/billing-money-core/design/data-model.md`
- `specs/billing-money-core/design/event-ingestion-redpanda.md`
- `docs/PRD.md`
- `docs/critical-billing-context.md`
- `docs/repo-architecture.md`

Additional read-only provider-contract evidence checked:
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/billing/v1/index.ts`
- `/Users/daniil/Projects/GonkaGate/payments-service/docs/service-foundation/data-model-and-migration.md`
- `/Users/daniil/Projects/GonkaGate/payments-service/research/02-internal-money-flow/findings.md`

Reviewed scope:
- Redpanda/Kafka-compatible event ingestion addendum only: consumed/emitted
  topic design, consumer group strategy, partitioning and ordering rules,
  durable inbox/outbox linkage, per-event transaction order, failure handling,
  reconciliation hooks, performance posture, observability, tests, and the
  data-model delta required by the addendum.

Out of scope for this review:
- implementation code, migrations, generated SQL, `tasks.md`, runtime event
  contract schemas, runtime adapters, and broader service architecture.

### Prior Gate Summary

Verdict: FAIL.

The addendum preserves the most important money boundary: Redpanda is transport,
while Postgres ledger, balances, idempotency records, operation outcomes, inbox,
outbox, and reconciliation state remain the correctness boundary. It also
correctly rejects broker-level exactly-once as a billing guarantee and uses
offset-after-durable-outcome ordering.

Planning is still blocked because the reviewed packet leaves one
provider-contract/idempotency decision and two distributed-recovery design
decisions unresolved. These are not proof-only concerns: planning would have to
choose event identity/version semantics, poison-event durable identity, and the
owner of committed-offset retry rows before it could produce executable tasks.

Why this is not `CONCERNS`: the unresolved points are live design choices that
change schema, event identity, retry ownership, and cross-service compatibility.
They cannot be carried as later proof obligations without asking planning or
implementation to make missing architecture decisions.

Why the smallest reopen target is `specification`: REDPANDA-TDR-F01 conflicts
with the approved payment-evidence identity/idempotency decision and the current
payments-service lineage contract. Technical design should be repaired after the
specification decides the billing/payments evidence-version contract.

### Prior Findings

#### REDPANDA-TDR-F01: Payment evidence identity conflicts with payments-service versioned evidence

Classification: `blocks_planning`, `reopens_spec`

Evidence:
- `spec.md` makes `payment_evidence_id` globally unique for billing and states
  that the same evidence ID with a changed fingerprint is an evidence conflict
  that cannot mutate money.
- `design/data-model.md` models `payment_evidence_id` as the primary key and has
  no `evidence_version` or equivalent version-scoped billing selector.
- `design/event-ingestion-redpanda.md` repeats the same rule: same
  `paymentEvidenceId` plus changed fingerprint is conflict, and the open
  question asks whether `payments-service` emits one normalized evidence event
  per `paymentEvidenceId`.
- Current payments-service design states that normalized evidence is append-only
  per `paymentEvidenceId` with monotonic `evidenceVersion`, and billing
  writeback selectors must include `paymentEvidenceId` plus `evidenceVersion`,
  not the stable lineage ID alone.

Impact:
- A valid payments-side corrected or superseding normalized evidence version
  under the same `paymentEvidenceId` would be classified by billing as changed
  fingerprint conflict.
- Alternatively, planning would have to decide whether event idempotency is
  keyed by bare `paymentEvidenceId`, by `paymentEvidenceId:evidenceVersion`, or
  by newly minted immutable billing evidence IDs. That decision belongs in the
  spec or cross-service contract, not `tasks.md`.

Required repair:
- Reopen specification to decide the billing/payments evidence identity contract:
  either require provider evidence handoff to use an immutable evidence ID per
  billing-applicable economic meaning, or revise billing's evidence model to
  carry `evidenceVersion` or an equivalent version-scoped selector through
  `payment_evidence`, `billing_event_inbox`, business idempotency, and replay
  rules.
- Record the provider-contract source used for the decision.
- After specification repair, update the Redpanda technical design and run a
  follow-up technical design review.

#### REDPANDA-TDR-F02: Poison-event quarantine has no durable identity when event ID is malformed or missing

Classification: `blocks_planning`, `reopens_design`

Evidence:
- `design/event-ingestion-redpanda.md` requires every consumed critical event to
  carry `eventId`, then defines `billing_event_inbox.event_id` as required with
  `UNIQUE (topic, event_id)`.
- The transaction model validates required fields, event ID, account scope,
  operation identity, and fingerprint before inserting or locking the inbox row
  by `(topic, event_id)`.
- The failure table says unsupported schema versions and poison events with
  malformed required fields are quarantined durably and the offset is committed.

Impact:
- If the malformed required field is `eventId`, event fingerprint, account
  scope, or operation identity, the design does not say what durable key is used
  for the inbox row that allows offset commit, redrive, and operator readback.
- Planning would have to choose between blocking the partition, synthesizing an
  event ID from `(topic, partition, offset)`, relaxing the schema, or adding a
  separate poison-event receipt key. That changes schema and replay semantics.

Required repair:
- Technical design must define the canonical durable identity for rejected or
  poison events whose producer event identity or semantic payload identity is
  invalid.
- The repair must state the corresponding constraints, replay/redrive rule, and
  test obligation proving quarantine can commit the offset without storing raw
  payloads or losing operator traceability.

#### REDPANDA-TDR-F03: Committed-offset retry rows do not have a closed recovery owner

Classification: `blocks_planning`, `reopens_design`

Evidence:
- `billing_event_inbox` includes a claim index for retry or local repair
  workers.
- The failure table allows account-row lock timeouts to store a durable retry
  outcome and commit the offset.
- The batching policy says offsets advance after durable outcomes, while the
  open questions leave runtime architecture to decide whether consumers run in a
  separate binary or managed worker process.

Impact:
- Once the offset is committed for `retry_scheduled`, Redpanda will not redeliver
  the event. Correctness depends on a local inbox retry owner with lifecycle,
  claim, readiness, backoff, and stuck-row behavior.
- Planning cannot safely task event ingestion without deciding whether that
  retry owner is the same consumer loop, a separate inbox-retry worker, or a new
  binary with independent readiness and deployment policy.

Required repair:
- Technical design must close the recovery-owner decision or explicitly split
  the accepted scope so event-ingestion planning cannot implement committed-offset
  retry states until a later runtime design gate chooses the owner.
- The repair must carry tests for retry-scheduled replay, stuck-row claiming,
  and shutdown/redelivery behavior.

### Prior Orchestrator Resolution

Gate status: FAIL.

Planning remains blocked. Do not create or broaden `tasks.md` for Redpanda event
ingestion, migrations, generated SQL, runtime adapters, worker code, or event
contract schemas from this packet.

Reopen target at that time: specification.

The specification reopen should be narrow:
- decide the billing/payments normalized evidence identity and versioning
  contract that REDPANDA-TDR-F01 exposes;
- record the provider-contract evidence used for the decision;
- preserve the already approved Postgres-as-money-truth boundary unless the
  evidence-version decision directly requires changing it;
- route the repaired artifact to technical design repair for REDPANDA-TDR-F02
  and REDPANDA-TDR-F03, plus any Redpanda design changes caused by the
  specification decision.

Follow-up technical design review is required after the repaired packet.

## Historical Data-Model Reviewed Packet

Reviewed on 2026-06-01 for the billing money core data-model slice.

Read first and used as authority:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `specs/billing-money-core/workflow-plan.md`
- `specs/billing-money-core/workflow-plans/technical-design.md`
- `specs/billing-money-core/spec.md`
- `specs/billing-money-core/design/data-model.md`
- `docs/PRD.md`
- `docs/critical-billing-context.md`
- `specs/billing-money-core/research/money-math-and-settlement-context.md`
- `docs/repo-architecture.md`

Reviewed scope:
- PostgreSQL data model only: tables, columns, constrained text state/kind fields,
  amount columns, constraints, indexes, lock order, hot-path query shapes,
  reconciliation/support readbacks, legacy import compatibility, and
  data-model test obligations.

Out of scope for this review:
- implementation code, migration SQL, generated access code, `tasks.md`,
  HTTP/OpenAPI contracts, runtime adapters, workers, and broad service
  architecture.

## Initial Gate Summary (Historical FAIL)

The design preserves the approved high-level boundaries: USD-only customer
ledger, GNK inventory outside customer balance truth, `request_id` as
correlation only, qualified `inferenceId` evidence, database-backed idempotency,
stored outcomes, account-first locking, and no cross-service calls inside money
transactions.

The review still fails because several data-model details that the approved
spec requires as concrete constraints or dedupe rules are either incomplete or
under-specified. Starting planning now would force `tasks.md` or migration work
to choose schema behavior that belongs in `design/data-model.md`.

Why this is not `CONCERNS`: the remaining issues are not merely proof-only
risks. They affect exact schema constraints and uniqueness/index policy that
planning must consume rather than invent.

Why this is not a specification reopen: no reviewed issue contradicts the
approved source-of-truth, amount, settlement identity, idempotency, or
legacy-import decisions. The smallest owner is technical design.

## Initial Findings (Historical FAIL)

### TDR-F01: Reconciliation dedupe is not concrete for every approved lineage

Classification: `blocks_planning`, `reopens_design`

Evidence:
- `spec.md` requires reconciliation cases to link safe lineage fields including
  `topup_operation_id`, `payment_attempt_id`, `settlement_effect_id`,
  `qualified_inference_evidence_id`, and ledger entries.
- `design/data-model.md` includes those lineage columns, but the duplicate-open
  case partial uniqueness only covers `(reason, usage_operation_id)`,
  `(reason, payment_evidence_id)`, and `(reason, ledger_entry_id)`.
- `legacy_import_mismatch` is an approved reason, but the design does not name a
  concrete import-row or equivalent dedupe key for that reason.

Repair required:
- Define the exact duplicate-open-case uniqueness strategy for each
  reconciliation reason and eligible lineage key.
- Either add concrete partial unique indexes/equivalent rules for
  `topup_operation_id`, `payment_attempt_id`, `settlement_effect_id`,
  `qualified_inference_evidence_id`, and legacy import mismatch lineage, or
  record a design-level reason why a listed lineage must not dedupe cases.
- Ensure a follow-up review can verify that planning does not need to choose
  reconciliation dedupe semantics.

### TDR-F02: Enum-like constrained text coverage is incomplete

Classification: `blocks_planning`, `reopens_design`

Evidence:
- `design/data-model.md` states that states and kinds use constrained `TEXT`
  checks.
- Several enum-like fields are documented in column notes but lack concrete
  checks in their table constraints, including `payment_attempts.state`,
  `operation_outcomes.primary_resource_type`, `operation_outcomes.operation_kind`,
  `idempotency_records.retention_class`, and `ledger_entries.created_by_kind`.

Repair required:
- Enumerate every state/kind/class/resource-type field introduced by the data
  model.
- Add concrete `CHECK (...)` constraints for the intended closed sets, or record
  an explicit design rationale for any field intentionally left unconstrained.
- Carry corresponding constraint-test obligations in the design so planning can
  task them directly.

### TDR-F03: `usage_charge` delta checks do not forbid pending-balance mutation

Classification: `blocks_planning`, `reopens_design`

Evidence:
- `design/data-model.md` models pending movement as `topup_pending` and
  `topup_pending_release`.
- Most ledger delta-pattern checks explicitly zero unrelated deltas.
- The `usage_charge` pattern requires `amount_usd_atoms = settled_delta_usd_atoms`,
  `settled_delta_usd_atoms < 0`, and `reserved_delta_usd_atoms <= 0`, but does
  not require `pending_delta_usd_atoms = 0`.

Repair required:
- Tighten the `usage_charge` delta-pattern rule so usage finalization cannot
  mutate `pending_usd_atoms`.
- Recheck all ledger effect types for explicit zero-delta coverage where an
  effect is not allowed to touch a balance component.
- Add the resulting constraint and negative-test obligation to the design.

## Initial Orchestrator Resolution (Historical FAIL)

Gate status: FAIL.

Planning remains blocked. `tasks.md`, migration SQL, generated code, runtime
adapters, and HTTP/OpenAPI contract design must not start from this packet.

Reopen target: technical design for
`specs/billing-money-core/design/data-model.md`.

The repair should be narrow:
- keep the existing approved data-model slice boundaries;
- do not reopen HTTP/API, runtime adapter, worker, or full service architecture
  design;
- repair only the concrete schema constraint/index/dedupe gaps above unless the
  repair exposes a direct contradiction with `spec.md`.

Follow-up technical design review is required after the repaired packet. The
follow-up may be targeted to this review's findings and changed adjacent
assumptions, but it must record a new `PASS`, `CONCERNS`, or `FAIL` verdict
before planning can start.

## Follow-Up Review

Reviewed on 2026-06-01 after the targeted technical-design repair pass recorded
in `workflow-plans/technical-design.md`.

Follow-up scope:
- TDR-F01, TDR-F02, and TDR-F03 from the historical FAIL above.
- Changed adjacent assumptions needed to verify those repairs.

Preserved boundaries:
- Data model only.
- No API contracts, runtime adapters, worker design, broad service
  architecture, migration SQL, generated code, or implementation planning.

### Follow-Up Gate Summary

Verdict: PASS.

The repaired `design/data-model.md` now gives planning concrete schema policy
for the three prior blockers. It preserves the approved data-model boundaries:
USD-only customer ledger, GNK inventory outside customer balance truth,
`request_id` correlation-only, qualified inference evidence, durable
idempotency, stored outcomes, account-first locking, one short transaction per
money command, support-safe readback, and legacy import evidence-only
compatibility.

No reviewed repair requires a specification reopen. No remaining issue forces
planning, migrations, or tests to invent schema semantics that belong in the
technical design.

### Follow-Up Findings

#### TDR-F01: Reconciliation dedupe is concrete for approved lineage

Classification: `resolved`

Evidence:
- `design/data-model.md` now adds `legacy_balance_import_id` to
  `reconciliation_cases`.
- Required lineage checks map approved reasons to concrete keys:
  `usage_operation_id`, `(topup_operation_id, payment_attempt_id)`,
  `payment_evidence_id`, `settlement_effect_id`,
  `qualified_inference_evidence_id`, `ledger_entry_id`, and
  `legacy_balance_import_id`.
- Partial uniqueness rules prevent duplicate unresolved cases for usage,
  top-up/payment-attempt, payment-evidence, settlement-effect,
  qualified-inference-evidence, ledger-entry, and legacy-import lineage.
- The design records why top-up-only lineage is intentionally not a
  duplicate-open-case key for the approved reasons.

Resolution: PASS. Planning can consume the mapped uniqueness rules and test
obligations without choosing reconciliation dedupe semantics.

#### TDR-F02: Constrained-text coverage is complete for introduced closed sets

Classification: `resolved`

Evidence:
- `design/data-model.md` now has a closed constrained-text inventory.
- The previously missing fields have concrete checks:
  `payment_attempts.state`, `operation_outcomes.primary_resource_type`,
  `operation_outcomes.operation_kind`,
  `idempotency_records.retention_class`, and
  `ledger_entries.created_by_kind`.
- Adjacent closed state/kind/class/resource-type fields introduced by the data
  model are either constrained in table checks or explicitly identified as
  externally versioned/support-taxonomy fields that should not share one global
  `CHECK`.
- The test obligations require constrained-text negative tests for every field
  listed in the inventory.

Resolution: PASS. Planning can task migration and integration tests from the
enumerated closed sets.

#### TDR-F03: Ledger delta patterns forbid unintended pending-balance mutation

Classification: `resolved`

Evidence:
- The `usage_charge` delta pattern now explicitly requires
  `pending_delta_usd_atoms = 0`.
- Each listed effect type zeros balance components it is not allowed to mutate.
- The design states every effect type must match exactly one approved pattern,
  and the test obligations require negative checks proving disallowed balance
  components are zero.

Resolution: PASS. Planning can consume the ledger delta-pattern rules directly.

### Changed Adjacent Assumptions Checked

- `topup_pending` and `topup_pending_release` remain data-model-only effect
  types for durable pending-balance movement; they do not approve API contracts
  or worker behavior.
- Account-level operator adjustment queues intentionally do not use a database
  uniqueness rule beyond account/readback indexes; the design explains that
  multiple unrelated operator issues may be open for one account.
- Externally versioned fields such as source authority, schema/version, and
  support taxonomy fields are intentionally not collapsed into one shared
  constrained-text set.

No changed adjacent assumption creates a planning blocker inside the approved
data-model slice.

### Follow-Up Orchestrator Resolution

Gate status: PASS.

Named proof obligations from `CONCERNS`: none, because the follow-up verdict is
PASS. Planning must still carry the existing data-model test obligations from
`design/data-model.md`, including constraint, reconciliation dedupe, ledger
delta-pattern, idempotency, concurrency, import, and performance proof.

At that historical checkpoint, planning was permitted to start for the approved
data-model slice. `tasks.md` had not been created yet and implementation
remained blocked until planning produced an approved ledger and the post-ledger
task-review/readiness gate passed.

## Follow-Up Review: Redpanda Event-Ingestion Repaired Packet

Reviewed on 2026-06-01 after the Redpanda technical-design repair pass recorded
in `workflow-plans/technical-design.md`.

Reviewed packet:
- `specs/billing-money-core/spec.md`
- `specs/billing-money-core/design/data-model.md`
- `specs/billing-money-core/design/event-ingestion-redpanda.md`
- `specs/billing-money-core/workflow-plan.md`
- `specs/billing-money-core/workflow-plans/specification.md`
- `specs/billing-money-core/workflow-plans/technical-design.md`
- this technical-design-review record
- `docs/PRD.md`
- `docs/critical-billing-context.md`
- `docs/repo-architecture.md`
- `/Users/daniil/Projects/GonkaGate/payments-service/docs/service-foundation/data-model-and-migration.md`
- `/Users/daniil/Projects/GonkaGate/payments-service/research/02-internal-money-flow/findings.md`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/billing/v1/index.ts`

Follow-up scope:
- REDPANDA-TDR-F01, REDPANDA-TDR-F02, and REDPANDA-TDR-F03 from the prior
  Redpanda FAIL above.
- Changed adjacent assumptions needed to verify the repaired event-ingestion
  packet is planning-ready.

Preserved boundaries:
- Redpanda/Kafka-compatible event ingestion design only.
- No code, migrations, generated SQL, `tasks.md`, runtime event contract schemas,
  runtime adapters, or broad service architecture.

### Redpanda Follow-Up Gate Summary

Verdict: PASS.

The repaired packet closes the three prior Redpanda blockers with concrete
planning inputs:

- payments-service evidence handoff is version-scoped by
  `(paymentEvidenceId, evidenceVersion)`;
- poison or quarantined events whose producer event ID or semantic identity is
  malformed get durable broker-coordinate receipts and can commit offsets without
  raw payload storage;
- committed-offset `retry_scheduled` rows have a named billing inbox retry owner
  with claim, lease, backoff, escalation, shutdown, and stale-claim recovery
  rules.

No reviewed repair requires another specification or technical-design reopen.
The remaining open items in `design/event-ingestion-redpanda.md` are planning,
contract-design, sizing, or packaging questions with fixed technical-design
boundaries. They do not force planning to invent ownership, source-of-truth,
event identity, idempotency, retry, quarantine, or money mutation semantics.

### Redpanda Follow-Up Findings

#### REDPANDA-TDR-F01: Versioned payment evidence selector is carried through the repaired packet

Classification: `resolved`

Evidence:
- `spec.md:228` through `spec.md:246` now states that
  `paymentEvidenceId` is the stable payments-owned lineage and
  `(payment_evidence_id, evidence_version)` is billing's application, inbox,
  idempotency, and replay selector.
- `design/data-model.md:768` through `design/data-model.md:807` splits stable
  `payment_evidence_lineages` from versioned evidence rows and states that
  `latest_evidence_version` is readback convenience only.
- `design/data-model.md:809` through `design/data-model.md:884` gives
  `payment_evidence` an internal row ID, `UNIQUE (payment_evidence_id,
  evidence_version)`, versioned prior-evidence lineage, and replay/conflict
  rules.
- `design/data-model.md:470` through `design/data-model.md:473` requires
  `topup_evidence` idempotency keys and fingerprints to include the versioned
  selector.
- `design/event-ingestion-redpanda.md:49` through
  `design/event-ingestion-redpanda.md:51`, `design/event-ingestion-redpanda.md:288`
  through `design/event-ingestion-redpanda.md:289`, and
  `design/event-ingestion-redpanda.md:351` through
  `design/event-ingestion-redpanda.md:357` carry the same selector through inbox
  lineage, replay, conflict, reconciliation, and support readback.
- Provider-contract evidence remains aligned: payments-service names
  `paymentEvidenceId` plus `evidenceVersion` as the evidence writeback selector,
  while the current `gonka-proxy` v1 apply-evidence schema lacks
  `evidenceVersion` and is explicitly deferred to later contract repair.

Resolution: PASS. Planning can task schema, inbox, business idempotency,
reconciliation, support readback, and tests from the version-scoped selector
without choosing a competing evidence identity.

#### REDPANDA-TDR-F02: Poison-event quarantine has durable broker-coordinate identity

Classification: `resolved`

Evidence:
- `design/event-ingestion-redpanda.md:52` through
  `design/event-ingestion-redpanda.md:55` records the repair decision: malformed
  poison/quarantine events get a durable receipt keyed by `(topic, partition,
  offset)` and safe receipt identity.
- `design/event-ingestion-redpanda.md:270` through
  `design/event-ingestion-redpanda.md:320` defines nullable producer event IDs
  only for poison receipts, `event_receipt_identity`,
  `event_identity_basis`, broker topic/partition/offset coordinates, and
  uniqueness constraints for both receipt identity and broker coordinates.
- `design/event-ingestion-redpanda.md:342` through
  `design/event-ingestion-redpanda.md:345`,
  `design/event-ingestion-redpanda.md:392` through
  `design/event-ingestion-redpanda.md:401`, and
  `design/event-ingestion-redpanda.md:455` through
  `design/event-ingestion-redpanda.md:456` define replay, transaction order,
  quarantine, and offset-commit behavior for malformed or missing producer or
  semantic identity.
- `design/event-ingestion-redpanda.md:551` through
  `design/event-ingestion-redpanda.md:559` gives operator redrive rules that
  preserve business idempotency and require corrected payloads to use a new
  producer event ID while referencing the original `event_inbox_id` or
  `event_receipt_identity`.
- `design/event-ingestion-redpanda.md:745` through
  `design/event-ingestion-redpanda.md:748` requires a test for quarantine with
  malformed or missing producer event ID or semantic identity.

Resolution: PASS. Planning can implement durable quarantine and offset commit for
poison events without fabricating producer identities or storing raw payloads.

#### REDPANDA-TDR-F03: Committed-offset retry has a closed recovery owner

Classification: `resolved`

Evidence:
- `design/event-ingestion-redpanda.md:56` through
  `design/event-ingestion-redpanda.md:59` states that committed-offset
  `retry_scheduled` rows are owned by the billing event-ingestion inbox retry
  worker, not Redpanda redelivery.
- `design/event-ingestion-redpanda.md:464` through
  `design/event-ingestion-redpanda.md:497` defines when `retry_scheduled` may be
  committed, the owner, claim query, lease/generation fencing, re-entry through
  stored inbox identity and fingerprint, retry backoff, escalation, graceful
  shutdown, and stale-claim recovery.
- `design/event-ingestion-redpanda.md:515` through
  `design/event-ingestion-redpanda.md:522` links repeated committed-offset retry
  failures to reconciliation or quarantine.
- `design/event-ingestion-redpanda.md:751` through
  `design/event-ingestion-redpanda.md:753` requires tests for retry-worker
  recovery without Redpanda redelivery, including stale-claim reclamation.

Resolution: PASS. Planning can task the inbox retry worker as part of the
event-ingestion slice without deciding a new recovery owner.

### Redpanda Adjacent Assumptions Checked

- Postgres remains the money truth: `design/event-ingestion-redpanda.md:38`
  through `design/event-ingestion-redpanda.md:48` keeps Redpanda as transport and
  rejects broker-level exactly-once as the billing correctness guarantee.
- Emitted events use a local outbox: `design/event-ingestion-redpanda.md:60`
  through `design/event-ingestion-redpanda.md:61` and
  `design/event-ingestion-redpanda.md:147` through
  `design/event-ingestion-redpanda.md:148` avoid a DB-write plus direct-broker
  dual write.
- Event contract schemas remain a later phase, but the planning-critical contract
  consequences are fixed: `design/event-ingestion-redpanda.md:820` through
  `design/event-ingestion-redpanda.md:847` preserves required topic envelopes,
  versioned evidence selectors, poison receipt references, and privacy-safe
  payload boundaries.
- Runtime packaging remains a later planning/architecture choice, but retry
  ownership is not open: `design/event-ingestion-redpanda.md:863` through
  `design/event-ingestion-redpanda.md:867` lets packaging choose separate binary
  or managed process while fixing the durable inbox retry owner.
- Exact partition counts and final retention sizing remain deployment inputs, but
  partition keys, minimum retention expectations, per-partition processing order,
  and offset-commit sequencing are already selected in the reviewed design.

No changed adjacent assumption creates a planning blocker for the Redpanda
event-ingestion addendum.

### Redpanda Follow-Up Orchestrator Resolution

Gate status: PASS.

Named proof obligations from `CONCERNS`: none, because the follow-up verdict is
PASS. Planning must still carry the event-ingestion test obligations from
`design/event-ingestion-redpanda.md`, including duplicate delivery, changed
fingerprint conflicts, crash-after-DB-commit replay, poison quarantine,
committed-offset retry recovery, concurrent account serialization, payment
evidence versioning, DB-timeout offset behavior, outbox retry, lag/stale
reservation reconciliation, and benchmark evidence.

At that historical checkpoint, planning was permitted to start for the Redpanda
event-ingestion addendum. That route is now superseded for current planning by
the sign-up-bonus and usage-only correction recorded below.

## Follow-Up Review: Current Sign-Up Bonus And Usage-Only Repaired Packet

Reviewed on 2026-06-01 after the current funding-source technical-design repair
recorded in `workflow-plans/technical-design.md`.

Reviewed packet:
- `specs/billing-money-core/spec.md`
- `specs/billing-money-core/design/data-model.md`
- `specs/billing-money-core/design/event-ingestion-redpanda.md`
- `specs/billing-money-core/workflow-plan.md`
- `specs/billing-money-core/workflow-plans/specification.md`
- `specs/billing-money-core/workflow-plans/technical-design.md`
- this technical-design-review record
- `docs/PRD.md`
- `docs/critical-billing-context.md`
- `docs/repo-architecture.md`

Follow-up scope:
- Current sign-up bonus grant model: grant identity, one-time credit semantics,
  ledger effect, idempotency, constraints/indexes, support readback,
  reconciliation, and tests.
- Current usage reserve/finalize/write-off data-model semantics after removing
  active payment/top-up funding from the current scope.
- Payment/top-up evidence, payments-service writeback, and Redpanda
  payment-evidence ingestion treatment as future/conditional context only.

Preserved boundaries:
- Technical design review only.
- No `tasks.md`, code, migrations, generated SQL, runtime adapters, workers,
  runtime event schemas, HTTP/OpenAPI contracts, or payment/top-up product
  design.

### Current Follow-Up Gate Summary

Verdict: PASS.

The repaired packet is planning-ready for the current sign-up-bonus and
usage-only money-core scope. The sign-up bonus is modeled as a first-class
billing grant and explicit ledger effect with stable grant identity, exact USD
atom amount, duplicate prevention, durable idempotency, conflict reconciliation,
support readback, concurrency posture, and data-model proof obligations. Usage
reserve/finalize/write-off remains current and independent from payment/top-up
activation. Payment/top-up evidence, payments-service writeback, and Redpanda
payment-evidence ingestion are clearly marked future/conditional and must not be
planned unless the workflow explicitly reopens that scope.

No reviewed repair requires another specification or technical-design reopen.
Planning can consume the current packet without inventing schema, idempotency,
reconciliation, source-of-truth, or payment-scope decisions.

### Current Follow-Up Findings

#### CURRENT-TDR-F01: Sign-up bonus grant model is concrete and planning-ready

Classification: `resolved`

Evidence:
- `spec.md:232` through `spec.md:269` defines the current sign-up bonus as the
  only approved balance-increase path, names the exact `$10.00` /
  `1,000,000,000 usd_atoms` grant, requires one grant per admitted account and
  policy, and excludes payment/top-up planning from this slice.
- `design/data-model.md:723` through `design/data-model.md:799` defines
  `signup_bonus_grants`, including `signup_bonus_grant_id`,
  `admission_authority`, `admission_reference_id`,
  `signup_bonus_policy_version`, fixed amount, grant fingerprint, state,
  idempotency/outcome/ledger links, uniqueness by account, subject, admission
  lineage, and application invariants.
- `design/data-model.md:340` through `design/data-model.md:426` includes
  `signup_bonus_credit` in the append-only ledger effect model and constrains it
  to a positive settled delta with zero reserved and pending deltas.
- `design/data-model.md:459` through `design/data-model.md:514` makes
  `signup_bonus_grant` a durable idempotency operation kind whose key and
  fingerprint include account, grant, policy, admission lineage, and amount.
- `design/data-model.md:1006` through `design/data-model.md:1139` gives
  `signup_bonus_conflict` a concrete reconciliation lineage and duplicate-open
  dedupe key by `signup_bonus_grant_id`.
- `design/data-model.md:1331` through `design/data-model.md:1361` gives the
  O(1) hot-path query shape and duplicate/conflict handling for sign-up bonus
  grant application.
- `design/data-model.md:1553` through `design/data-model.md:1658` carries exact
  bonus atom, uniqueness, idempotency, concurrency, reconciliation, ledger
  conservation, and performance proof obligations.

Resolution: PASS. Planning can task the sign-up bonus schema/query/test work
without making hidden data-model decisions.

#### CURRENT-TDR-F02: Usage reserve/finalize/write-off remains current and independent of payment activation

Classification: `resolved`

Evidence:
- `spec.md:187` through `spec.md:230` keeps durable holds and usage operations as
  the lifecycle authority for reserve, finalize, write-off, reversal, and
  compensation; none of those decisions depend on top-up or payment evidence
  activation.
- `spec.md:362` through `spec.md:416` keeps sign-up bonus, reserve, finalize,
  and write-off as expected O(1) hot-path transactions with account-row locking,
  durable idempotency, no cross-service calls inside DB transactions, and named
  proof obligations.
- `design/data-model.md:554` through `design/data-model.md:690` defines
  `usage_operations`, `usage_holds`, and `usage_terminal_outcomes` with the
  required states, uniqueness, terminal outcome protection, and hold invariants.
- `design/data-model.md:1266` through `design/data-model.md:1309` preserves the
  account-first lock order and forbids outbound HTTP, pricing lookup, payment
  lookup, or provider calls while holding a transaction.
- `design/data-model.md:1363` through `design/data-model.md:1418` gives the
  current reserve, finalize, and write-off access paths and required checks
  without requiring payment/top-up activation.

Resolution: PASS. Planning can include current usage decrement/release work from
the repaired data model without first reopening payment/top-up or event-ingestion
scope.

#### CURRENT-TDR-F03: Payment, top-up, payments-service writeback, and Redpanda payment evidence are future/conditional

Classification: `resolved`

Evidence:
- `spec.md:31` through `spec.md:54` and `spec.md:260` through `spec.md:269`
  exclude customer payment/top-up product flow, normalized payment evidence
  application, payments-service writeback, and Redpanda payment-evidence
  ingestion from the current implementation slice.
- `design/data-model.md:17` through `design/data-model.md:43` marks top-up and
  normalized payment evidence tables as dormant context, not current
  balance-increase paths.
- `design/data-model.md:800` through `design/data-model.md:1005` keeps
  top-up/payment/evidence tables as dormant future context and explicitly
  prevents them from becoming current balance-increase inputs.
- `design/data-model.md:1420` through `design/data-model.md:1448` labels top-up
  evidence application as historical/future design context only and requires
  explicit payment/top-up scope reopen plus live payments-service contract
  revalidation before planning.
- `design/data-model.md:1660` through `design/data-model.md:1676` states that no
  implementation, migration, API contract, runtime adapter, runtime event schema,
  or task ledger is approved by the design, and that payment/top-up,
  payments-service evidence writeback, and Redpanda payment-evidence ingestion
  remain future/conditional.
- `design/event-ingestion-redpanda.md:11` through
  `design/event-ingestion-redpanda.md:21`,
  `design/event-ingestion-redpanda.md:186` through
  `design/event-ingestion-redpanda.md:203`, and
  `design/event-ingestion-redpanda.md:913` through
  `design/event-ingestion-redpanda.md:922` demote Redpanda payment-evidence
  ingestion and runtime event schemas to future/conditional context for current
  planning.

Resolution: PASS. Planning must not create tasks for payment/top-up evidence,
payments-service writeback, inbox/outbox event-ingestion deltas, or runtime event
contracts from the current packet.

### Current Adjacent Assumptions Checked

- The broad PRD still describes top-up and payment evidence as target product
  responsibilities, but the approved current spec supersedes that broad target
  for this planning slice by explicitly limiting current balance increases to
  the sign-up bonus.
- Dormant future top-up/payment tables remaining in `design/data-model.md` do not
  create current work because each active planning section labels them future or
  conditional and ties any use to explicit scope reopen.
- Repository architecture constraints remain satisfied for this review phase:
  no runtime adapter, worker, OpenAPI, migration, generated SQL, or service
  package design is approved here.

No changed adjacent assumption creates a planning blocker for the current
sign-up-bonus and usage-only money-core scope.

### Current Follow-Up Orchestrator Resolution

Gate status: PASS.

Named proof obligations from `CONCERNS`: none, because the follow-up verdict is
PASS. Planning must still carry the current data-model test obligations from
`design/data-model.md`, including exact money representation, sign-up bonus
constraints, usage reserve/finalize/write-off idempotency, ledger conservation,
non-negative balances, reconciliation dedupe, concurrency, privacy-safe
readback, and performance proof.

Planning is permitted to start for the current sign-up-bonus and usage-only
money-core ledger. The existing `tasks.md` is approved only for a prior slice and
must not be treated as an implementation handoff for the repaired current scope
until planning repairs or replaces it and records task-ledger review/readiness.

## Workflow State

Current phase: follow-up technical design review for the current sign-up-bonus
and usage-only repair.
Phase status: complete with PASS.
Next phase: planning for the current sign-up-bonus and usage-only money-core
ledger.
Reopen target: none.
Planning status for current scope: permitted to start.
`tasks.md` status: existing prior ledger must be repaired or replaced by
planning before current sign-up-bonus/usage-only implementation can start.
Required next gate: planning must produce a reviewed task ledger with
task-ledger review/readiness `PASS` or eligible `CONCERNS` before coding,
migrations, generated SQL, runtime adapters, workers, payment/top-up work, or
event contract work.

## Recommended Next-Session Prompt

```text
Work in `/Users/daniil/Projects/GonkaGate/billing-service`.

Next phase: planning for the current sign-up-bonus and usage-only money-core ledger.

Read first:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `specs/billing-money-core/workflow-plan.md`
- `specs/billing-money-core/workflow-plans/specification.md`
- `specs/billing-money-core/workflow-plans/technical-design.md`
- `specs/billing-money-core/workflow-plans/technical-design-review.md`
- `specs/billing-money-core/spec.md`
- `specs/billing-money-core/design/data-model.md`
- `specs/billing-money-core/design/event-ingestion-redpanda.md`
- `specs/billing-money-core/tasks.md` if present, only to determine whether the
  prior ledger should be repaired or replaced for current scope
- `docs/PRD.md`
- `docs/critical-billing-context.md`
- `docs/repo-architecture.md`

Goal:
Create or repair the planning ledger from the approved current sign-up-bonus and
usage-only specification/design packet and the follow-up technical-design-review
PASS. The ledger must cover sign-up bonus grant schema/query/test work, current
usage reserve/finalize/write-off preservation, reconciliation/support readback,
and the named proof obligations without planning payment/top-up evidence,
payments-service writeback, Redpanda payment-evidence ingestion, runtime event
schemas, or runtime adapter/worker work.

Expected output:
- update `specs/billing-money-core/tasks.md` or the appropriate planning
  artifact so current sign-up-bonus and usage-only work is executable without
  hidden design choices;
- run and record the post-ledger task-review/readiness gate as `PASS`,
  eligible `CONCERNS`, or `FAIL`;
- update `specs/billing-money-core/workflow-plan.md` with the planning outcome;
- produce the next recommended prompt.

Stop rule:
Complete planning and task-ledger review/readiness only, then stop with updated
workflow state and the next recommended prompt. Do not write code, create
migrations, generate SQL, implement runtime adapters or workers, design runtime
event contract schemas, or reopen payment/top-up scope in this session unless
the planning review verdict requires reopening an earlier phase.
```
