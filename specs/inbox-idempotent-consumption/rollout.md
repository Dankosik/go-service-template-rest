# Rollout: idempotent PostgreSQL inbox adoption
status: draft
Affected graph: generated profile + PostgreSQL schema + one service-specific worker handler, duplicate-tolerant handler -> atomic claim/effect handler
Irreversible boundary: the last rollback-safe point is before the first inbox claim commits; after that, dropping or renaming the claim identity can reapply an effect, so stop and roll forward.

| Gate | Owner; node or edge | Prerequisite | Action | Success and distinct safe failure signal | Duration or behavior-changing horizon | Rollback or roll-forward | Proof |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1. Fix identity, history, and effect boundary | Service owner; durable consumer -> feature adapter | Stable stream/consumer names and same-PostgreSQL effect confirmed | Bind `stream/consumer` plus logical `MessageID`; keep claim/effect in one transaction; prove no pre-cutover applied ID can replay without a claim, or complete a service-owned claim seed/retain existing domain idempotency | Compile/lint exemplar and authoritative history inspection close both the transaction and forward-only adoption boundary; failure blocks adoption | Identity is permanent; a later rename is a new consumer | Change code freely before any claim commits | **TD-INBOX-010** target identity/history procedure plus **TD-INBOX-007** (`REQUIRE_DOCKER=1 go test -tags=integration ./examples/reference-service -run '^TestPostgresInboxAdapterPlacement$' -count=1`; `make lint`) |
| 2. Add schema | Database owner; PostgreSQL | Gate 1; migration number does not collide in the derived service | Apply additive claim-table migration | Catalog/constraint readback matches the two-column key; failure rolls back the transactional migration | One schema transaction | Roll back only while the table is empty | **TD-INBOX-008** (`make sqlc-check`; `make migration-check`; `make migration-validate`) plus **TD-INBOX-011** writer-primary catalog readback |
| 3. Replace the durable consumer | Service operator; old/new worker replicas | Schema ready; old handler drain budget known | Stop and drain every old replica for this durable consumer, then start inbox-enabled replicas | No old replica remains and the new worker admits the same durable consumer; failure leaves consumption paused | One worker drain plus admission cycle | After first claim, roll forward; never overlap old and new handlers on one durable consumer | **TD-INBOX-011** schema/readiness/deployment-identity cutover procedure |
| 4. Verify durable suppression | Service/operator owner; NATS -> handler -> PostgreSQL | New worker admitted | Deliver/redeliver one known event and observe one feature effect plus one claim before `../outbox-production-closure/rollout.md` Gate 5 publishes production events to this consumer | Second delivery returns success without another effect; failure stops the new worker and blocks the outbox relay cutover | One handler timeout plus redelivery/ack cycle | Roll forward after diagnosing; retain claims | **TD-INBOX-012** consumer-before-relay canary, backed by **TD-INBOX-006** (`REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresInboxNATSLogicalIdentityAndAcknowledgement$' -count=1`) |

Completion: the selected worker consumes through the concrete adapter, one
PostgreSQL transaction commits the claim and feature effect, a redelivery with
the same logical ID is acknowledged without another effect, and no old handler
shares the durable consumer. Test Design bindings are fixed above. This record
stays draft until an adopting service supplies the target identity, historical
authority, deployment inventory, and canary readbacks required by
TD-INBOX-010 through TD-INBOX-012.
