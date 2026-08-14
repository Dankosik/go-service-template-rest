# PostgreSQL-backed HTTP idempotency rollout design

status: ready
authority: `specs/http-idempotency-postgres/design/overview.md`
review evidence: prior review covered overview content hash `46680d10b2b633237e77392c6db0aa3a33e9530a6c5afa85eab38c412d0e96ad` and rollout content hash `89b61f51e6d577cfc743b98d640e1a3934983b6707cd0d47540306239bb841e4` (2026-08-11); fresh independent Technical Design review PASS on corrected overview candidate `e19e486bcdc5ead2a1e9c5b7261cf34dcdd27161943cbac1f870b85f07bcfaf0` with rollout candidate `f4a69683c4468a25a5f7a25804e16742bbf02059e40c6f89aae5e6c46ac13f65` (2026-08-12), followed only by readiness receipt edits
reopen review evidence: complementary Go Ownership panels and independent whole-artifact Technical Design review PASS on overview candidate `3601dd3cf5d3091a578e478ab052ba32d9f85c6118fc6a94ca77bc1580471823` with this rollout candidate `807105028f8f999914776f85f5d7ac721a14cc3a419b8530806e7ca22d3382f2` (2026-08-12). The reopened T3 system candidate `92a9dd7660b4aeb1978348b8f712b23d67e1c7ee4530ca41a30a979a56eabffb` with this rollout `8e316f84388c2dcac4b8dd85ca2e344e28a74aaed826e9eee4f727c5ff236cbb` received independent FAIL because final authority-admission rejection lacked a terminal observer; repaired candidate `01fc1cb615fbdee18fdd2e4da43c42c99e93e8a4f13e3bef033b62a8d9387bfb` with this rollout `5a53f02d20ecbbdee57bfd84761678122bb42c0e976553500200c3653048f32d` received focused independent CONCERNS with no surviving System Design blocker (2026-08-12). The movable concerns are the intentionally stale Go responsibility maps and the downstream split-finality/prompt-drain oracle update. Only review-receipt and artifact-status edits followed; target activation remains NOT READY.
TD-IDEM-015A active-image cleanup reopen: independent Technical Design review PASS on the fixed rollout candidate `80bd527176609d5a2d6d819d775a00845758d1f324987d644989dceb8fe35921` (2026-08-14). It confirmed that one disposable `psql` control session must submit `SET default_transaction_read_only = off` and the role reset as separate completed commands, preserving the active-image non-writer oracle and all runtime identities. Test Design remains valid; Planning owns the narrow `migration-validate` correction.
movement: Specification stays closed. The reopened System Design condition is closed by the disjoint admission-rejection/admitted-endpoint finality seam and prompt-drain seam in `overview.md`; Go Ownership must reconcile them into the frozen Store/bootstrap file map before Test Design repairs the invalidated T3 proof boundary. Target activation remains NOT READY.

## Release class and graph

The capability is one optional pack in the existing service and migration artifacts.
It adds no deployable. The optional outbox relay is unchanged and exists only when
an adopting endpoint appends outbox intent.

The first schema change is additive: one independent relation, its nonreused
generation source, constraints, and bounded indexes. It has no backfill and changes
no existing row or table. The canonical migration runs through `/migrate` before a
service revision may advertise an opted operation. The service never migrates at
startup.

`Down` exists only for disposable migration rehearsal. After any opted request, the
row set is contract evidence and production rollback never drops it.

## Profile and build gate

`HTTP_IDEMPOTENCY` accepts exactly `none|postgres`, defaults to `none` only when
unset, rejects explicit empty/unknown values before mutation, and requires
`DATABASE=postgres`. It is independent of authn, outbox, messaging, and transport
selection. `postgres` records `http_idempotency = "postgres"` in generated
`template.lock`; same-choice rerun is identical and changed choice is rejected.

The initialization matrix must establish:

- `none` leaves no capability runtime, migration, query/generated, config, docs, or
  test surface;
- `postgres` retains one coherent pack and runs the existing single SQLC generation
  point after profile resolution;
- health-only, OIDC-enabled, outbox, inbox, and messaging combinations compile
  without opting a health route in;
- generated-tree purity and repeatability remain exact.

No `template.lock`, migration, generated Go, or initialized fixture is created in
Technical Design.

Runtime proof uses two source identities because they prove different claims:

- the existing health-only initialized image remains the positive migration,
  profile-off/previous-revision compatibility, readiness, and shutdown carrier; its
  empty registration slice is never presented as active-idempotency evidence;
- one disposable OIDC-enabled initialized checkout receives the checked-in
  `scripts/ci/fixtures/postgres-http-idempotency-active.patch`, which supplies a
  complete protected operation and registration only for image proof. The existing
  runtime-image script builds that source once under a distinct tag and reuses the
  exact tag for missing-schema, read-only/non-writer, and
  `track_commit_timestamp=off` startup cases.

Implementation extends the existing `migration-validate` owner; it does not add a
second migration harness or required context. The exact aggregate remains
`make runtime-image-build RUNTIME_IMAGE=service:ci` followed by
`make migration-validate RUNTIME_IMAGE=service:ci`. `migration-validate` reuses the
supplied health-only tag, invokes the existing runtime-image builder once for the
distinct active fixture source, and reuses that active tag for all three rejection
cases. The two builds are justified by distinct source identities; neither identity
is rebuilt inside the aggregate.

The TD-IDEM-015A non-writer schedule has two fixture-local owners: the active image
under the `app` role observes the injected read-only authority, while a separate
disposable `psql` control session owns that role setting's restoration. Setting
`default_transaction_read_only = on` makes a new `app` session read-only, so cleanup
must not submit `ALTER ROLE app RESET default_transaction_read_only` directly from
such a session. After the rejection assertion, that control session first submits
`SET default_transaction_read_only = off` as one completed command, then submits the
role reset as a second command in the same session; the reset therefore begins a
writable transaction. The reset must succeed before the commit-timestamp case. On
any earlier failure, the existing disposable Compose volume cleanup is the sole
fallback and no role mutation survives the rehearsal. This changes neither image
identity, active registration, runtime startup ordering, nor the non-writer oracle.

The active rejection cases occur before OIDC trust initialization. They therefore
exercise the production image and PostgreSQL admission path without a fake
principal or a local-issuer bypass. Local positive active composition uses the
real-PostgreSQL bootstrap integration harness and test-only runtime wiring; final
positive production-image evidence is the adopter's authenticated activation
canary against its real public HTTPS issuer. A second local trust path is forbidden.

## Migration and mixed-version sequence

1. Publish the contract-preserving image and additive migration.
2. Run `/migrate` to completion on the authoritative writer.
3. Start the new service revision with schema/writer admission closed until its
   selected-profile checks pass.
4. Prove the previous revision remains healthy against the additive schema during
   the configured old/new overlap.
5. Keep opted operation traffic gated until every receiving revision enforces the
   contract and all endpoint/deployment inputs are present.
6. Activate the new/versioned operation or complete its accepted client migration.

Before first activation, application rollback may leave the additive schema unused.
After activation, an unguarded revision is never a safe target while evidence may
exist. Roll back only to a contract-preserving revision or gate that operation at
ingress and roll forward. Never down-migrate or delete evidence as incident rollback.

A future constraint tightening or schema contraction is a later
expand -> migrate/backfill -> switch -> contract release. Migration history remains
append-only.

## Deployment admission

Repository-local implementation can prove the reusable mechanism, not a live
topology. Operation activation remains `NOT READY` until the adopter records:

- one writer-only route and one authoritative clock domain;
- acknowledged-commit preservation and commit-timestamp continuity across supported
  failover, with promotion fencing and no opted admission during uncertainty;
- environment/region namespace and maximum service replicas;
- one positive `http_idempotency.owner_recovery_delay` from the ordinary typed
  deployment config path, plus session-loss, writer-authority, and bounded-
  classification evidence that closes `T_owner_recovery`;
- every endpoint quantity, authenticated authority, admission owner, semantic
  version/codec, and external-effect recovery;
- cleanup batch/cadence/maximum lag, storage ceiling/reserve, representative churn,
  autovacuum, and commit-epoch materialization headroom;
- privacy classification, lawful guarded evidence, erasure authority, backup
  extinction horizon, and a restore-surviving erasure source;
- alert thresholds/SLOs, private diagnostics collection, activation route, and
  contract-preserving rollback route.

With no registered operation, startup does not construct or admit the capability.
With at least one registration, startup checks selected schema and writer state and
rejects absent operation quantities, missing or nonpositive active runtime config,
or `track_commit_timestamp=off`. Runtime writer loss or uncertain promotion closes
new ownership; safe retained-state reads continue only when their authority remains
decisive.

## Maintenance and process lifecycle

Terminal telemetry follows the existing HTTP split. An authenticated, valid opted
request rejected by authority admission records one `failed` outcome in the envelope
and returns before endpoint invocation. An admitted request records nothing there;
its endpoint/application adapter records exactly once only after the final Store,
transaction, and reconciliation result. Active bootstrap supplies both paths the
same observation-only Store seam. No request token or second metric owner is added.

With no registered operation, bootstrap constructs no idempotency store, probe,
telemetry instruments, or maintenance task and performs no capability
schema/writer/config check. With at least one registration, each service process
runs one supervised loop. Before Store construction, bootstrap conditionally
validates its active-only typed config and maps `owner_recovery_delay` with the
other Store-owned fields into the one `StoreOptions` passed to
`postgresidempotency.NewStore`; operation
registration and OpenAPI carry no recovery delay. The Store uses it only to derive
writer-clock `recover_after`; a held row lock still wins, and activation evidence
still owns the full `T_owner_recovery` bound. A cycle prioritizes commit-epoch
materialization, then strips expired replay results, deletes eligible finite guards,
and refreshes aggregate observations in one configured total batch budget. Writer
`SKIP LOCKED` arbitration makes replicas cooperate without a leader; total
concurrency is bounded by the declared maximum service replicas.

Active startup obtains the first writer/schema/capacity observation before opening
admission. Thereafter maintenance publishes one atomic snapshot that owns the
Store's private first-execution predicate. Reservation classifies an existing row
before consulting it, so replay, mismatch, in-progress, and expired remain
available; only writer-confirmed absence returns unavailable without insertion when
the snapshot is unobserved, stale, terminal, non-writer, epoch-lost, or inside the
configured safety reserve. No request performs a capacity query and no exported
capacity-specific seam is added. Bootstrap uses only the general
`Store.Maintain(context.Context) error` cycle plus the Store's `Name`/`Check`
`health.Probe` shape; the atomic snapshot and capacity predicate stay private.

Recoverable cleanup failure retains evidence, records the closed failure event, and
retries on the declared cadence. A full batch leaves the next cycle at that cadence
and never queues another cycle concurrently; representative capacity proof must show
that fixed rate catches up. New first executions close before storage safety
headroom is exhausted. Safe replay/mismatch/expired reads
continue while writer authority is available.

Terminal task failure, stale safety observation, evidence-preservation breach,
missing schema, or writer-authority loss becomes cached-unready. The readiness HTTP
handler performs no I/O. Liveness remains process-only. Existing drain order stays:

1. readiness disabled and propagation delay;
2. API drain;
3. maintenance cancel and bounded join;
4. PostgreSQL close;
5. telemetry flush.

Request discovery of `ErrEpochLost` or `ErrIntegrityConflict` first publishes the
exact first terminal error in the Store's atomic safety snapshot, then sends it once
on the Store-owned capacity-one notification channel. The existing supervised
maintenance task is the sole receiver and returns that error without waiting for
the next cadence tick, so the existing background-failure path starts the same
readiness-off and clamped drain above. The notification is a wakeup only: it adds no
goroutine, poll, task, readiness source, or shutdown owner; zero registrations add
nothing. A maintenance database call already in progress remains bounded by its
existing context and must return before its task can consume the notification.

Loss of an unmaterialized row's PostgreSQL commit timestamp is a terminal integrity
incident, not recoverable cleanup. Requests receive 503 without replay, expiry,
deletion, or execution; the task failure drains the process. Before activation, the
database owner names an exact commit-timestamp recovery source and RTO. Recovery
restores that epoch from a surviving synchronous member or backup/WAL authority,
materializes it, and reruns schema/writer/expiry/erasure admission. If no exact epoch
survives, the deployment remains blocked and Specification R9 reopens; operators do
not substitute statement time or delete the guard.

Do not add a cleanup binary, cron, elected leader, or database-backed scheduler.
Reopen topology only if measured cleanup cannot fit service-replica headroom, replica
count is operationally unbounded, or maintenance must continue while service replicas
are zero.

## Repository publication gates

The existing exact-head aggregate remains authoritative. Implementation must extend
the relevant existing gates rather than add a new required context:

- repository integrity, project structure, secret scan, OpenAPI, SQLC drift,
  migration history/source, format/lint/race/integration/security, and template-init
  proof;
- the existing CI change-scope route includes the runtime-image script, active
  fixture patch, and HTTP-idempotency profile paths so a fixture-only change cannot
  skip migration validation; publication continues to run that existing gate
  unconditionally;
- the upstream deterministic health-only image proves generic image lifecycle, and
  the distinct disposable active-registration fixture selects PostgreSQL
  idempotency plus OIDC so active container proof cannot pass on an empty
  registration slice;
- migration validation builds each distinct source identity once and reuses its
  exact tag: the health-only tag proves up/down/up, migrate-then-ready,
  old/profile-off compatibility, and clean hardened shutdown; the active tag proves
  missing schema, read-only/non-writer authority, and commit-timestamp rejection
  before OIDC initialization. Only the adopter activation canary claims positive
  active production-image readiness;
- publication keeps the existing digest, vulnerability/SBOM, signature,
  attestation, migration-history, and mutable-tag promotion chain;
- a managed source build is identified by its platform build/deployment record;
  registry attestations do not certify an independently rebuilt image.

Changing required-key behavior, scope, fingerprint versions, semantic result,
stable headers, horizons, writer region, or Problem/status/retry advice is a
client-visible compatibility release, not runtime config drift.

## Rollback and restore falsifiers

Activation fails if any of these is true:

- old code can receive opted traffic after the first request;
- failover can lose an acknowledged completed row or its exact commit epoch;
- a restore can expose replay material already expired or erased at current writer
  time;
- cleanup can delete an active/unknown row or required guard;
- capacity admission allows required evidence to cross its reserve;
- a rollback plan depends on down-migration or evidence deletion;
- a direct external effect is described as covered without endpoint recovery.

These are deployment blockers, not reasons to weaken the reusable contract.
