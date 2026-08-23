# T001 — Durable Standard Webhooks ingress capability

Outcome:
The source template and disposable initialized outputs implement exactly one removable `INBOUND_WEBHOOKS=none|standard-webhooks` capability: the selected profile verifies exact Standard Webhooks bytes, atomically commits one PostgreSQL receipt and River job, processes only verified durable receipts through exact typed bindings, preserves terminal evidence and the reviewed privacy/finality rules, and the default or `none` profile contains no inbound residue while existing behavior remains unchanged.

Consumes:
- [`../spec.md`](../spec.md) sha256 `36554b7496075046cad501f137723c2d89a3830e29ca0b84c5f2ecbf2406fc20`.
- [`../design/overview.md`](../design/overview.md) sha256 `f034becd6cf10c4311b032ee2c1746ac5a4172b01d75189652b7ed63e6b86e3c`.
- [`../design/technical-review.md`](../design/technical-review.md) sha256 `0b4438d410490d96d8f1ac9d7f82a9c4ba3c1033267107c22585b872c9cda481` and [`../design/ownership-review.md`](../design/ownership-review.md) sha256 `41e13dd59e4d9b8c9d08ae3129181df8cbdda9e740766df20e0db377bbd294e2`.
- [`../design/transition.md`](../design/transition.md) sha256 `c9cf069c116b22f2a000adadba879590e689a7938ce4f53a3bf9295623fbe9fc`.
- [`../test-plan.md`](../test-plan.md) sha256 `0c6ca630207458f3bf4bf75f863dce075ec3814737d9f96c03717e644af4c97e` and [`../test-design-review.md`](../test-design-review.md) sha256 `2f9911843328f75b5c9dae1ef30afba802119f91d4c3b79c596694cb0868c8ed`.
- [`../transition.md`](../transition.md) sha256 `1d4a7296dfae7ae92605d2334b118eedb1cf7ce8861df2044a49a3c7df85acce`.
- Current owner seams observed during Planning at `HEAD` `8967a4ac06d4fce0515703b15ffa5db35e5378ae`; the implementation owner must refresh them before editing and keep all pre-existing dirty bytes outside the bounded candidate.

Provides:
- One locally accepted, profile-coherent source-template implementation and exact selected/default/none initialized trees.
- Repository evidence for all local Test Design rows, including real PostgreSQL transaction/concurrency proof, real River/process recovery and disclosure proof, generated OpenAPI/SQLC/migration parity, initializer removal, and runtime-image composition.
- A fixed local candidate that later TD-EXT-01 and TD-REL-01 owners may consume without satisfying or weakening either external gate.

Boundary:
Implement only the reviewed inbound capability across its neutral contract, PostgreSQL/River adapter, raw generated HTTP operation, configuration and secret projection, service/jobs-worker composition, schema/SQLC/generated outputs, telemetry, documentation, initializer/profile removal, and owned proof. Keep all layers in this unit because no partial layer is independently consumable or profile-coherent. Preserve outbound webhooks, ordinary strict HTTP behavior, existing jobs/outbox owners, and every unselected profile. Use stdlib plus installed dependencies and the existing runtime owners; add no provider adapter, protocol registry, event envelope, queue, retry engine, process, cache, limiter, timeout, payload-size setting, ordering/reconciliation engine, automatic deletion, or external action.

Mutable owners:
- inbound-webhook neutral receiver and typed processing contracts
- PostgreSQL/River inbound receipt, verification, processing, terminalization, and bounded telemetry adapter
- canonical migration and SQLC source plus generated receipt/query output
- canonical OpenAPI operation, generated bindings, raw/strict HTTP composition, Problem mapping, and route-level contract proof
- service and jobs-worker inbound configuration projection, bootstrap composition, readiness/buffer accounting, worker binding, and lifecycle proof
- inbound initializer selector, marker/removal/shared-dependency rules, `template.lock`, disposable-tree and runtime-image proof
- inbound capability documentation, examples, dependency-boundary declarations, and repository-native validation surfaces

Exclusive locks:
- inbound OpenAPI source/generated operation and raw/strict handler composition contract
- migration `000010_postgres_inbound_webhooks.sql` and its SQLC source/generated receipt contract
- inbound endpoint/secret configuration and process-projection contract
- jobs-worker `WorkersRuntime` builder/bind-before-River lifecycle contract
- `INBOUND_WEBHOOKS` initializer removal, shared River/Standard Webhooks retention, and `template.lock` contract

Accept when:
- Claim: One exact candidate satisfies TD-CFG-01, TD-MAN-01, TD-BIND-01, TD-SIG-01, TD-WIRE-01, TD-REQ-01, TD-ADM-01, TD-RSP-01, TD-HDR-01, TD-OAS-01, TD-DB-01, TD-DB-02, TD-DB-03, TD-SCHEMA-01, TD-QUAR-01, TD-HAND-01, TD-RETRY-01, TD-REC-01, TD-SEC-01, TD-OBS-01, TD-SBOOT-01, TD-WBOOT-01, TD-CAP-01, TD-GEN-01, and TD-REG-01 without claiming TD-EXT-01 or TD-REL-01.
- Focused check: Run every exact `command_or_procedure` cell and fixed oracle for the local scenario IDs above, with uncached results, non-zero matching test counts for named tests, `REQUIRE_DOCKER=1` on every Docker-backed command, and no skipped or unavailable mandatory input treated as evidence.
- Integrated check: Run in order against the same fixed candidate: `go test -count=1 ./internal/inboundwebhook ./internal/infra/postgresinboundwebhook ./internal/infra/http ./internal/config ./cmd/service/internal/bootstrap ./cmd/jobs-worker ./cmd/jobs-worker/internal/bootstrap`; `go test -count=1 -race ./internal/inboundwebhook ./internal/infra/postgresinboundwebhook ./internal/infra/http ./cmd/service/internal/bootstrap ./cmd/jobs-worker ./cmd/jobs-worker/internal/bootstrap`; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -race -tags=integration ./test -run '^Test(PostgresInboundWebhook|InboundWebhook)'`; `REQUIRE_DOCKER=1 make test-integration`; `make openapi-check`; `make sqlc-check`; `make migration-check`; `make template-init-check`; `make runtime-image-build RUNTIME_IMAGE=service:inbound-webhooks-test`; `make gosec`; `make check`.
- Observable: Every named scenario reaches its independent oracle; each `204` follows one definite receipt/job commit; duplicate/conflict, ambiguous outcome, terminalization, process recovery, and disclosure paths converge exactly as reviewed; generated and initialized trees have the exact selected/removed surfaces and shared dependencies; all 11 ladder commands are terminal-success for the same bounded candidate; required Implementation Review passes; and the acceptance receipt explicitly leaves provider/adopter and deployment facts unverified.

Reopen if:
Use the exact `reopen_owner` in the fixed Test Design row that fails. Reopen Specification only for changed wire, provider, retention, ordering, fairness, or payload-classification behavior; System / Integration Design only if raw-byte preservation, one-transaction receipt/job ownership, River recovery/finality, or rollout ordering becomes infeasible; Go Code / Ownership Design only if refreshed package, generated/manual, composition, or initializer ownership cannot preserve the reviewed map; Test Design only for a missing, infeasible, non-deterministic, or non-dispositive oracle. Reopen Planning only if the complete local capability can no longer land and be accepted as one profile-coherent result. Missing TD-EXT-01 or TD-REL-01 inputs blocks only their later external gate, not T001.
