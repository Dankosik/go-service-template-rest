# Railway Full Production Infrastructure Sequence

Status: review-ready
Date: 2026-06-02

## High-Level Flow

```text
read-only preflight
  -> source topology proof
  -> dedicated Postgres + backup/PITR/restore proof
  -> migrations + schema read-back
  -> private Kafka-compatible broker + topics + lag proof
  -> image repair for /billing-worker
  -> private billing-worker service readiness
  -> app dependency variables + private readiness
  -> clean gonka-proxy provider contract
  -> cohort authority enablement
  -> rollback/fail-closed proof
  -> paid readiness claim
```

Every step is secret-free in artifacts and fail-closed when a dependency is not
ready.

## S0. Read-Only Preflight

1. Read back Railway project, environment, service inventory, app service ID,
   current deployment status, source repo, builder, healthcheck, domain posture,
   and variable key families without values.
2. Confirm no existing `billing-service-postgres`, `billing-worker`, or
   billing-specific broker service changes the accepted additive topology.
3. Confirm local repo origin and intended branch/root/config path against
   Railway read-back.
4. Stop before mutation if source repo, branch, root, Dockerfile, or config path
   differs from the approved target.

Failure owner: technical design or specification reopen, depending on whether
the mismatch changes topology or only an implementation-time setting.

## S1. Dedicated Postgres And Recovery Baseline

1. Create or verify private `billing-service-postgres`.
2. Set `APP__POSTGRES__DSN` as a Railway secret/reference variable for app and
   worker without exposing the DSN.
3. Prove the DSN contract through sanitized adapter/readiness output: one TCP
   host, port, database, user, non-empty password class, and `sslmode`, without
   printing values.
4. Enable daily backups.
5. Take a manual pre-cutover backup before authority enablement.
6. Enable PITR and wait for the first post-enable base backup.
7. Restore to a sibling Postgres service.
8. Verify restored schema version, dirty state, representative billing rows, and
   semantic reconciliation inputs.
9. Keep restored sibling cutover manual and closed until active exposure,
   terminal gaps, inbox/outbox state, broker evidence, and proxy evidence are
   reconciled.

Failure owner: data design or rollout planning. Paid authority remains closed.

## S2. Migration And Schema Proof

1. Build the canonical image containing `/migrate` and `/env/migrations`.
2. Railway app pre-deploy runs `/migrate` against dedicated Postgres.
3. Read back current migration version and dirty state.
4. Verify mixed-version safety during `railway.toml` overlap/drain.
5. Stop if migration cannot run before app promotion or if dirty state is true.

Failure owner: implementation for narrow migrator/image bugs; technical design
or specification if migration ownership/order changes.

## S3. Broker, Topics, And Lag Proof

1. Create or verify private persistent Kafka-compatible service
   `billing-service-kafka` from selected candidate template `kafka`.
2. Read back internal endpoint posture, persistence, storage, and no public UI or
   domain exposure unless explicitly approved.
3. Create or verify:
   - `billing.microlease.terminal.v1`;
   - `billing.microlease.checkpoint.v1`;
   - `billing.microlease.close.v1`;
   - `billing.microlease.facts.v1`.
4. Read back retention and partition policy: inbound topics at least 7 days,
   facts at least 30 days.
5. Set `APP__REDPANDA__BROKERS` and topic/group keys without printing values.
6. Prove broker probe and consumer-group lag summary without credentials.
7. Keep worker replica count at one until partition/assignment/idempotency/lag
   proof supports more.

Failure owner: specification reopen if no private persistent Kafka-compatible
service can meet the spec; otherwise technical design or planning repair.

## S4. Image Repair

1. Repair the Docker build so the final image contains:
   - `/service`;
   - `/migrate`;
   - `/billing-worker`;
   - `/env/migrations`.
2. Preserve distroless nonroot runtime and canonical Dockerfile path.
3. Run repo/container proof from `docs/build-test-and-development-commands.md`.
4. Do not create `billing-worker` service until image inspection proves the
   binary exists.

Failure owner: implementation after planning; design remains selected.

## S5. Worker Service Readiness

1. Create or verify private Railway service `billing-worker`.
2. Use the same repo/source/image lineage as the app and start command
   `/billing-worker`.
3. Disable public domains and Railway HTTP health checks unless a reopened
   design adds a private worker health surface.
4. Enable worker config only after Postgres and broker are ready:
   - Postgres enabled and DSN set by secret/reference;
   - service auth enabled because config validation requires it for worker mode;
   - Redpanda enabled with brokers/topics/group;
   - microlease worker enabled.
5. Worker process opens Postgres, builds three consumers and one producer,
   checks Postgres and broker probes, validates all seven roles, marks itself
   ready internally, and runs task loops.
6. Evidence records bounded logs/metrics/read-backs only: role names, service
   ID, start command, replica count, dependency probe pass/fail class, lag and
   backlog buckets, admission-control freshness, and shutdown/drain proof.

If worker mode is disabled, the process exits successfully and does no work.
Therefore disabled-worker health is never a paid-readiness proof.

## S6. App Dependency Readiness

1. Set app variables coherently by key name only.
2. Enable Postgres readiness.
3. Enable Redpanda readiness through
   `APP__FEATURE_FLAGS__REDPANDA_READINESS_PROBE=true` or stricter reviewed
   proof.
4. Enable service auth with issuer, audience, and JWKS URL.
5. Keep public ingress disabled.
6. Railway `/health/ready` must fail when required dependencies fail and pass
   only after startup admission, Postgres, broker readiness, network policy, and
   app bootstrap are healthy.
7. Read back app replicas and resource policy; comments in `railway.toml` are
   not enough.

Failure owner: app implementation or technical design depending on missing
readiness behavior.

## S7. Proxy Provider Handoff

1. Verify a clean `gonka-proxy` provider contract or sibling implementation
   ledger.
2. Prove RS256 token settings by key names and non-secret posture only:
   issuer, audience, subject, `kid`, TTL, JWKS publication, and rotation
   overlap.
3. Prove private proxy-to-billing URL through Railway internal networking.
4. Prove scopes for required route classes:
   account resolve, balance read, usage read/write, microlease read/write, and
   operations read.
5. Prove microlease issue/readback/close HTTP calls where required.
6. Prove producer identity `gonka-proxy` and event production to terminal,
   checkpoint, and close topics.
7. Prove durable proxy child-debit lineage before external execution.
8. Prove no legacy `BILLING_SERVICE_AUTH_KEY` fallback, operator-adjustment
   money writer, proxy-local money writer, Redis spend authority, process-local
   reserve, or direct per-request reserve fallback for migrated cohorts.

Current dirty proxy evidence does not pass this sequence.

## S8. Authority Enablement

1. Keep `balance_usage_authority.mode=inert_expand` until all prior gates pass.
2. Move only to an approved cohort state after:
   - Postgres and migrations are current;
   - backups/PITR/restore proof exists;
   - broker/topics/lag are green;
   - worker is ready and admission-control freshness is current;
   - service auth and proxy provider proof pass;
   - rollback proof passes.
3. If broker, worker, proxy, restore, or admission-control freshness degrades,
   close new paid admission and new microlease issuance.

## S9. Rollback And Fail-Closed Sequence

1. Record pre-change deployment/resource IDs and key-only variable posture.
2. Roll back app deployment or variables only after closing authority/admission.
3. Stop or scale down worker only after drain/commit evidence for in-flight
   fetch/process/commit cycles.
4. Do not delete broker topics or shrink retention during rollback.
5. Do not switch to restored Postgres until semantic reconciliation proves
   active microleases, child debits, terminal gaps, inbox/outbox, broker offsets,
   and proxy evidence are safe.
6. On rollback, no new microleases are issued, no direct reserve fallback is
   enabled, and proxy-local money writes remain forbidden for migrated scopes.
7. Already allocated child debits must settle, write off, or reconcile through
   billing.

## Failure Point Summary

| Failure | Required response |
| --- | --- |
| Source topology mismatch | Stop before mutation; reopen specification or technical design. |
| Dedicated Postgres unavailable | Keep app default-closed; reopen specification if no reviewed equivalent exists. |
| Backup/PITR/restore proof incomplete | Block paid authority; planning may only carry a proof obligation if design review accepts it. |
| Migration dirty state | Block promotion/authority and repair migrator/data issue. |
| Kafka template cannot prove private persistence | Reopen specification. |
| Topic or lag proof missing | Keep worker/app Redpanda readiness and paid authority closed. |
| Image lacks `/billing-worker` | Block worker service deployment. |
| Worker disabled or no-op | Not paid-ready; keep migrated cohorts no-spend/read-only. |
| JWKS or proxy contract missing | Block paid readiness; do not use shared-key fallback. |
| Public domain required for proof | Requires explicit user/spec decision because `/metrics` is exposed on root router. |
| Evidence leaks secrets or payloads | Stop, redact, and rerun evidence collection safely. |

## Parallelism Notes

Later planning may parallelize read-only checks and repo proof, but mutation
sequencing is dependency ordered:

- source topology before any Railway mutation;
- Postgres/recovery before authority;
- broker/topics before worker/app Redpanda readiness;
- image repair before worker service;
- worker and proxy proof before paid authority;
- rollback proof before paid readiness.
