# Billing Service Railway Full Production Infrastructure Tasks

Status: approved for implementation with concerns
Task ledger review: PASS
Implementation readiness: CONCERNS
Date: 2026-06-02
Owner: orchestrator

## Goal Contract

Goal objective: Complete the approved Railway full production infrastructure rollout for `billing-service` by executing this ledger from T001 through final validation.
Stopping condition: all required tasks are checked, each Evidence line is current, required repository/Railway/sibling proof passes or records a concrete blocker with the exact reopen target, and ledger-owned closeout updates are current.
Read first: `AGENTS.md`, `docs/spec-first-workflow.md`, this `tasks.md`, `spec.md`, `workflow-plans/technical-design-review.md`, `design/overview.md`, all linked `design/` artifacts, `test-plan.md`, `rollout.md`, `research/*.md`, `docs/repo-architecture.md`, `railway.toml`, `docs/railway-deployment-profile.md`, `env/config/default.yaml`, `env/.env.example`, `docs/configuration-source-policy.md`, `build/docker/Dockerfile`, `docs/build-test-and-development-commands.md`, `specs/balance-usage-authority-cutover/spec.md`, `specs/balance-usage-authority-cutover/rollout.md`, and read-only provider evidence from `/Users/daniil/Projects/GonkaGate/gonka-proxy`.
Do not change: public billing ingress posture, public `/metrics` exposure, direct per-request reserve fallback rejection, proxy-local money writer rejection for migrated cohorts, Redis or memory spend-authority rejection, scoped RS256/JWKS service-auth requirement, dedicated billing-service Postgres ownership, fail-closed rollback posture, and the secret-free evidence boundary.
Progress log: after each task proof, update the task checkbox and `Evidence` line with command/read-back evidence or the concrete blocker. Keep Railway evidence key-only and secret-free.
Blocked-stop rule: if a task requires a missing architecture, ownership, contract, data, rollout, validation, public-ingress, worker-health-surface, broker-strategy, provider-contract, cohort-selection, or rollback decision, stop and record `Blocked:` under the task with the reopen target named in that task.

## Implementation Handoff

Consumes: approved `spec.md`, reviewed split `design/`, `workflow-plans/technical-design-review.md` verdict `CONCERNS`, `test-plan.md`, `rollout.md`, preserved research notes, repo-local deployment/config sources, and this ledger.
Task ledger review: PASS.
Implementation readiness: CONCERNS.
First task: T001.
Accepted concerns: `TDR-PO1` through `TDR-PO7` are accepted as executable proof obligations, not open questions.
Workflow-plan adequacy: local read-only adequacy challenge PASS. A subagent was not spawned because the available subagent tool permits spawning only when the user explicitly asks for delegation.
Separate review phase files: not expected; task-ledger review is complete in this planning pass.
Separate validation phase files: not expected; this ledger plus `test-plan.md` and `rollout.md` carry validation and closeout proof.
Reopen target: planning for task coverage/order/proof gaps; technical-design-review if a required review verdict is missing or stale; technical design for missing source-topology, broker, worker-readiness, rollout, data, validation, or evidence-surface decisions; specification for changed broker availability, public ingress, provider contract, paid-authority, money-authority, or privacy decisions.

## TDR Proof Obligation Mapping

| Obligation | Carried by | Required proof |
| --- | --- | --- |
| `TDR-PO1` source-topology and live Railway read-back before mutation | T001, T002 | Read-only project/environment/service/source/domain/variable-key read-back passes before any Railway mutation. |
| `TDR-PO2` `/billing-worker` image repair before worker service | T003, T011 | Canonical image inspection proves `/service`, `/migrate`, `/billing-worker`, and `/env/migrations` before worker service creation/start. |
| `TDR-PO3` Kafka-compatible broker, topic admin, policy, and lag proof | T008, T009, T010, T012 | Private persistent broker, topic policy, consumer group, lag/backlog thresholds, and credential-free read-back are proven before Redpanda readiness or paid authority. |
| `TDR-PO4` worker non-HTTP readiness and drain proof | T004, T011, T012 | Worker stays private with no public health surface; readiness comes from role, dependency-probe, lag/backlog, admission freshness, and shutdown/drain evidence. |
| `TDR-PO5` clean `gonka-proxy` provider contract blocks paid readiness | T015, T016, T017, T018 | Paid authority stays blocked until clean proxy contract or sibling ledger proves JWKS, scopes, private URL, event production, child-debit lineage, and no fallback. |
| `TDR-PO6` secret-free evidence boundary | Every task, especially T020 and T022 | Evidence excludes secrets, DSNs, bearer tokens, JWTs, JWKS contents, event payloads, request bodies, raw customer data, variable values, and dynamic proof URLs. |
| `TDR-PO7` fail-closed rollback semantics | T018, T019, T021 | Rollback closes admission, preserves no-fallback/no-proxy-writer rules, drains worker safely, and forbids restored-DB cutover before semantic reconciliation. |

## Tasks

- [x] T001 [Read-only preflight] Refresh live Railway and local source-topology evidence before any mutation.
  Files: read-only Railway evidence, local git/source evidence, `railway.toml`, and `docs/railway-deployment-profile.md`.
  Depends on: none.
  Proof: read back project `58904982-91dd-4780-9b70-699dc9271e18`, environment `1c272106-e6be-4547-ae8d-2e7933eca6df`, app service `64592463-335e-4be0-a8d1-330438fd61d0`, source repo, branch, root directory, config path, Dockerfile path, `Wait for CI` posture when available, domains, replica/resource settings, deployment status, and variable key families without values.
  Reopen target: specification if source owner or public-ingress assumptions change; technical design if app/worker resource or replica settings require a different design.
  Evidence: 2026-06-02 read-only preflight passed using explicit Railway IDs. `rtk railway whoami --json` authenticated as workspace `91212c39-3a7d-424e-88b9-547aac9a518a`; local checkout was unlinked, so all Railway reads used project `58904982-91dd-4780-9b70-699dc9271e18`, environment `1c272106-e6be-4547-ae8d-2e7933eca6df`, and app service `64592463-335e-4be0-a8d1-330438fd61d0`. `mcp__railway.environment_status`, `mcp__railway.list_services`, and `rtk railway service list --json --project ... --environment production` showed app deployment `8c61aefe-2f72-411d-995d-f1ac5accf259` `SUCCESS`, no `billing-service-postgres`, no `billing-worker`, and no billing-specific broker service. GraphQL `serviceInstance` read-back showed repo `Dankosik/go-service-template-rest`, `rootDirectory=null` (repo root), `railwayConfigFile=railway.toml`, `dockerfilePath=build/docker/Dockerfile`, `preDeployCommand=[/migrate]`, `healthcheckPath=/health/ready`, `healthcheckTimeout=180`, `restartPolicyType=ON_FAILURE`, `restartPolicyMaxRetries=5`, `overlapSeconds=45`, `drainingSeconds=30`, no public/service/custom domains, and latest deployment timestamp `2026-06-02T10:53:54.790Z`. Current build logs for the same deployment prove Dockerfile stages from `build/docker/Dockerfile` built `/service` and `/migrate` and copied `/env/migrations`; GraphQL `builder=RAILPACK` is an API enum mismatch with the Dockerfile build evidence and `railway.toml` `builder=DOCKERFILE`, not treated as a topology blocker. Safe key-only GraphQL variable query returned app variable keys only, including `APP__POSTGRES__ENABLED`, `APP__SERVICE_AUTH__ENABLED`, `APP__REDPANDA__ENABLED`, `APP__MICROLEASE__ENABLED`, `APP__MICROLEASE__WORKER_ENABLED`, `APP__BALANCE_USAGE_AUTHORITY__ENABLED`, `APP__BALANCE_USAGE_AUTHORITY__MODE`, and `NETWORK_PUBLIC_INGRESS_ENABLED`; no values were printed. Branch and Wait-for-CI posture were not exposed by safe CLI/MCP/GraphQL read-backs; local git branch is `main`, remote is `https://github.com/Dankosik/go-service-template-rest.git`, and HEAD is `249630de06444e5285db4fda2a345fde6bf18aad`. Live app replicas read back as one configured/running replica via `rtk railway service list --json`; this is recorded for later T013 baseline proof and does not require a pre-mutation design reopen because the app remains private/default-closed.

- [x] T002 [Read-only preflight] Confirm repository command/tooling readiness and isolate unrelated dirty work.
  Files: `docs/build-test-and-development-commands.md`, `Makefile`, current git worktree, and task-local artifacts.
  Depends on: T001.
  Proof: record current git status, available Docker/tooling state, and the smallest validation bundle needed for changed surfaces. If unrelated dirty files exist, keep implementation evidence scoped and do not stage or revert unrelated changes.
  Reopen target: planning if validation command ownership is missing or the ledger cannot isolate unrelated work safely.
  Evidence: 2026-06-02 local preflight passed. `rtk git status --short --branch` showed `main...origin/main [ahead 1]` plus untracked `specs/railway-production-infrastructure/`; implementation will treat the task-local spec bundle as approved input and will not stage or revert unrelated work. Tooling is available: Railway CLI `4.66.0`, Docker CLI/daemon `29.4.0`, Go `go1.26.3 darwin/arm64`, `/usr/bin/make`, and `/usr/bin/jq`. Dry-run ownership checks confirmed the relevant validation entrypoints: `rtk make -n guardrails-check` -> `bash ./scripts/ci/required-guardrails-check.sh`; `rtk make -n docker-build` -> `docker build -f build/docker/Dockerfile -t billing-service:local .`; `rtk make -n docker-container-security` -> `bash ./scripts/dev/docker-tooling.sh container-security`; `rtk make -n test-race` -> `go test -race ./...`. Smallest required validation bundle for this rollout remains the ledger-selected repo, image, security, migration, OpenAPI, Railway, broker, worker, proxy, rollback, and privacy proof in T003-T021.

- [x] T003 [Image repair] Repair and prove the canonical Docker image ships the worker binary.
  Files: `build/docker/Dockerfile`, `cmd/billing-worker`, `docs/railway-deployment-profile.md` if policy text changes, and Docker image proof.
  Depends on: T002.
  Proof: `rtk make docker-build`; image inspection proves `/service`, `/migrate`, `/billing-worker`, and `/env/migrations`; distroless nonroot runtime and canonical Dockerfile path are preserved; `rtk make docker-container-security` runs when Docker is available.
  Reopen target: implementation for narrow Dockerfile/build defects; technical design if the canonical image strategy must change.
  Evidence: 2026-06-02 implemented in `build/docker/Dockerfile`: the canonical image now builds `./cmd/billing-worker` to `/out/billing-worker` and copies it to `/billing-worker` while preserving `/service`, `/migrate`, `/env/migrations`, distroless static Debian 12 nonroot runtime, canonical Dockerfile path, and `/service` entrypoint. `rtk make docker-build` passed and built `billing-service:local`; build output showed `/out/service`, `/out/migrate`, and `/out/billing-worker` build steps plus final-stage copies. Filesystem inspection via `rtk docker export <temp-container> | rtk tar -tf - | rtk rg '^(service|migrate|billing-worker|env/migrations($|/))'` proved `service`, `migrate`, `billing-worker`, and migrations `000001` through `000004` exist in the image. `rtk docker image inspect billing-service:local --format '{{json .Config.User}} {{json .Config.Entrypoint}}'` returned `nonroot:nonroot` and `[/service]`. `rtk make docker-container-security` passed; Trivy summary reported 0 vulnerabilities for `service:ci` Debian 12.13 and 0 vulnerabilities for `billing-worker`, `migrate`, and `service`.

- [x] T004 [Worker evidence surface] Prove or add the secret-free worker readiness and task evidence surface selected by design.
  Files: `cmd/billing-worker/internal/bootstrap`, `internal/app/microleaseworker`, `internal/infra/redpanda`, `internal/infra/postgres`, worker tests, and bounded logs/metrics/readback surfaces.
  Depends on: T003.
  Proof: targeted tests prove enabled worker mode is never a no-op, dependency probe failures exit non-zero, all seven roles are observable by bounded labels, task result/reason labels are low-cardinality, and no raw event/request/customer data is emitted.
  Reopen target: technical design if proof requires adding a private HTTP health surface or attaching any worker domain.
  Evidence: 2026-06-02 implemented a non-HTTP worker evidence surface in `cmd/billing-worker/internal/bootstrap/runtime.go`: `newWorkerRuntime` now logs the configured seven-role set and dependency probe count, then wires a `workerLogObserver` into `microleaseworker.New`; task evidence emits only bounded `worker_role`, `result`, and `reason_class` labels. The observer defensively collapses unknown role/result/reason values to `other`, so account/request/microlease-looking strings are not logged. `rtk go test ./internal/app/microleaseworker ./cmd/billing-worker/internal/bootstrap` passed with 14 tests across 2 packages, covering required seven-role validation, dependency-probe failure returning `ErrNotReady`, all seven roles running and stopping on cancellation, terminal concurrency bound, missing concrete runtime dependencies rejected, concrete terminal/checkpoint/close/inbox/outbox/stale/admission role wiring, shutdown/drain on canceled run, Postgres-open failure when enabled runtime cannot build, and bounded worker log labels with no raw account/request/microlease strings. No worker HTTP health surface or domain was added.

- [x] T005 [Postgres resource] Create or verify dedicated private `billing-service-postgres` and key-only DSN posture.
  Files: Railway database resource read-backs, app/worker variable key posture, `docs/configuration-source-policy.md`, and Postgres adapter proof.
  Depends on: T001, T002.
  Proof: service name/ID, private connectivity posture, `APP__POSTGRES__DSN` reference/source key posture for app and worker, and sanitized DSN contract proof for one TCP target with host, port, database, user, non-empty password class, and `sslmode`, without printing the DSN.
  Reopen target: specification if no dedicated private billing Postgres or reviewed equivalent can satisfy the target; technical design if variable/reference shape changes the design.
  Evidence: 2026-06-02 created dedicated Railway service `billing-service-postgres` (`34909776-3455-4905-99e6-18b752dbbe4e`) in project `58904982-91dd-4780-9b70-699dc9271e18` / production environment `1c272106-e6be-4547-ae8d-2e7933eca6df` from image `ghcr.io/railwayapp-templates/postgres-ssl:18`, with volume `billing-service-postgres-volume` (`14c33f97-d038-4b6d-aeb5-57e929c850fc`) mounted at `/var/lib/postgresql/data`. `mcp__railway.environment_status` and GraphQL `serviceInstance` showed latest Postgres deployment `d91172b8-45d5-4985-8115-c59067685217` `SUCCESS` at `2026-06-02T12:43:34.154Z`, source image `ghcr.io/railwayapp-templates/postgres-ssl:18`, and no service/custom domains. Key-only variable read-back for the database showed `DATABASE_URL`, `PGDATA`, `PGDATABASE`, `PGHOST`, `PGPASSWORD`, `PGPORT`, `PGUSER`, `POSTGRES_DB`, `POSTGRES_PASSWORD`, `POSTGRES_USER`, and `SSL_CERT_DAYS`; values were not printed. App service `APP__POSTGRES__DSN` was set with `skipDeploys=true` as a Railway reference to `billing-service-postgres` plus `sslmode=require`; key-only unrendered read-back proved the key is present, reference-shaped, and includes `sslmode=require`. Sanitized rendered DSN contract proof reported `scheme=postgresql`, `target_count=1`, `host_present=true`, `host_private_domain=true`, `port_present=true`, `database_present=true`, `user_present=true`, `password_class=non_empty`, and `sslmode=require` without printing the DSN, host, user, database, or password. Worker-side DSN posture is deferred to T010/T011 because `billing-worker` must not be created before the approved image/broker/topic/variable gates.

- [ ] T006 [Postgres recovery] Enable and prove backups, PITR, restore-to-sibling, and semantic reconciliation inputs.
  Files: Railway backup/PITR/read-back evidence, restored sibling service evidence, `env/migrations`, reconciliation queries/checks, and support-safe data summaries.
  Depends on: T005.
  Proof: daily backup schedule, manual pre-cutover backup, PITR enabled state, first post-enable base backup, restore to sibling Postgres, restored migration version and dirty state, representative billing-state summaries, and semantic reconciliation inputs for active microleases, child debits, terminal gaps, inbox/outbox, broker offsets, proxy evidence, reconciliation cases, and admission-control freshness.
  Reopen target: technical design if restore/reconciliation sequence needs a different design; specification if dedicated recovery policy cannot be met.
  Evidence: Partial 2026-06-02 recovery setup proof completed before blocker; continuation proof refreshed 2026-06-02 and blocked-audit recheck repeated 2026-06-02. Resolved the new volume instance via GraphQL `environment.volumeInstances`: volume instance `468ec790-c311-4725-aaf4-57c8e94103d8`, external ID `vol_dvcapf5r9rdf1d5y`, volume ID `14c33f97-d038-4b6d-aeb5-57e929c850fc`, service ID `34909776-3455-4905-99e6-18b752dbbe4e`, mount path `/var/lib/postgresql/data`, state `READY`, size `50000` MB, current size `0` MB. `volumeInstanceBackupScheduleUpdate(kinds=[DAILY])` returned true; fresh schedule read-back returned daily schedule `8891978c-a660-452d-9e9f-dcbf4f2b0640`, cron `12 13 * * *`, retention `518400` seconds, created `2026-06-02T12:48:27.934Z`. Manual pre-cutover backup workflow `createVolumeInstanceBackup/468ec790-c311-4725-aaf4-57c8e94103d8` was accepted; fresh backup list read-back returned backup `9253effb-a7b8-4a7a-9c85-b75ea5e2f92c`, name `billing-service-precutover-2026-06-02`, created `2026-06-02T12:48:49.200Z`, `expiresAt=null`, `usedMB=1`, `referencedMB=1106`. `rtk railway volume --project 58904982-91dd-4780-9b70-699dc9271e18 --environment production list --json` showed `billing-service-postgres-volume` mounted on `billing-service-postgres` at `/var/lib/postgresql/data`; `rtk railway volume --help` exposed volume list/add/delete/update/detach/files/browse/attach only, with no PITR or backup subcommand. Official Railway PITR docs fetched 2026-06-02 still state enablement is performed from the Postgres service Backups tab and creates a `Postgres-PITR` bucket, sets `WAL_ARCHIVE_*` variables, and redeploys. Public API docs fetched 2026-06-02 expose volume backup list/create/restore/lock/delete and schedule list; live GraphQL introspection exposed `volumeInstanceBackupCreate`, `volumeInstanceBackupRestore`, `volumeInstanceBackupScheduleUpdate`, and `volumeInstancePITRRestore`, but no PITR-enable mutation or enable-shaped WAL/archive mutation. Fresh key-only Postgres variable read-back returned 11 keys and no `WAL`, `ARCHIVE`, `PITR`, `BACKREST`, or `RECOVER` key names. Blocked: PITR enablement, first post-enable base backup, PITR restore-to-sibling, restored schema/dirty-state read-back, and semantic reconciliation inputs cannot be completed from the current safe tool/API/CLI surface without inventing the dashboard's unreviewed `WAL_ARCHIVE_*` patch. Exact reopen target: technical design for a reviewed PITR enablement and restore evidence path, or specification if the dedicated PITR recovery policy cannot be met safely on Railway without dashboard/operator action.

- [ ] T007 [Migration proof] Promote schema through `/migrate` against dedicated Postgres and prove schema state.
  Files: `railway.toml`, `build/docker/Dockerfile`, `env/migrations`, migration read-backs, and migration validation output.
  Depends on: T003, T005.
  Proof: `/migrate` remains Railway app pre-deploy, migrations run against `billing-service-postgres`, current migration version matches repository latest, dirty state is false, and `rtk make migration-validate` or `rtk make docker-migration-validate` passes. Mixed-version safety under overlap/drain is recorded.
  Reopen target: implementation for narrow migrator defects; technical design if schema promotion order or mixed-version policy changes.
  Evidence: Pending.

- [ ] T008 [Broker resource] Prove the selected Kafka-compatible broker candidate before or while creating `billing-service-kafka`.
  Files: Railway broker/template read-backs, broker service evidence, and `design/contracts/service-auth-and-broker.md`.
  Depends on: T001.
  Proof: selected candidate template/service `kafka` or exact equivalent proves private persistent Kafka-compatible operation, internal endpoint posture, storage/persistence, no public UI/domain by default, and credential-free admin/read-back path before worker/app Redpanda readiness.
  Reopen target: specification if no private persistent Kafka-compatible Railway broker can meet the approved requirements; technical design if a different concrete template/image/topic admin path is needed while preserving the spec.
  Evidence: Pending.

- [ ] T009 [Broker topics and lag] Create or verify topics, retention, partitions, consumer group, and lag/backlog gates.
  Files: broker admin/read-back evidence, Railway variables by key name, `internal/infra/redpanda`, and broker validation notes.
  Depends on: T008.
  Proof: topics `billing.microlease.terminal.v1`, `billing.microlease.checkpoint.v1`, `billing.microlease.close.v1`, and `billing.microlease.facts.v1` exist; inbound topics retain at least 7 days; facts retain at least 30 days; partition count and keying policy preserve microlease/child-debit/account-scope ordering; consumer group is `billing-service-microleases`.
  Required gates: paid-readiness `green` requires no critical terminal/checkpoint/close lag, no retry-eligible outbox/inbox backlog, admission-control freshness <= 45s, and synthetic proof drained to zero; `warning` is nonzero lag/backlog with oldest item <= 60s and count < 100 during shadow/internal proof only; `critical` is oldest item > 60s, count >= 100, admission freshness > 45s, or missing topic/group proof.
  Reopen target: technical design if observed runtime capacity cannot support these first-rollout gates without a different partitioning, assignment, or worker-scale design.
  Evidence: Pending.

- [ ] T010 [Variable posture] Configure app and worker dependency variables coherently by key name only.
  Files: Railway app/worker variable key read-backs, `env/.env.example`, `env/config/default.yaml`, and `internal/config`.
  Depends on: T005, T009, T015.
  Proof: app and worker key posture shows Postgres, service auth, Redpanda, microlease, worker-readiness assertion, authority mode, feature readiness probes, and `NETWORK_PUBLIC_INGRESS_ENABLED=false` are coherent. Variable evidence is key-name/non-secret posture only; values are never copied.
  Reopen target: technical design if config validation requires a different dependency chain; specification if public ingress or weaker auth is required.
  Evidence: Pending.

- [ ] T011 [Worker service] Create or verify private Railway service `billing-worker` only after image and dependency gates pass.
  Files: Railway worker service read-backs and source config evidence.
  Depends on: T003, T005, T009, T010.
  Proof: service name/ID, same repo/branch/root/image lineage as app unless reviewed otherwise, start command `/billing-worker`, one replica for first rollout, no public domain, no Railway HTTP healthcheck unless design reopens, deployment status, source config path, Dockerfile path, and resource settings.
  Reopen target: technical design if worker readiness requires HTTP health or resource/replica settings materially change.
  Evidence: Pending.

- [ ] T012 [Worker readiness] Prove non-HTTP worker readiness, roles, probes, lag/backlog, admission freshness, and shutdown/drain.
  Files: worker logs/metrics/read-backs, broker lag/read-backs, Postgres backlog/read-backs, and worker tests.
  Depends on: T011.
  Proof: all seven roles are present (`terminal_consumer`, `checkpoint_consumer`, `close_consumer`, `inbox_retry`, `outbox_relay`, `stale_reconciliation`, `admission_control_renewal`); Postgres and broker probes pass; terminal/checkpoint/close consumers and facts producer construct; task evidence is bounded; admission-control freshness <= 45s; shutdown/drain completes within configured timeout with safe offset/commit behavior; disabled/no-op worker is not accepted as readiness.
  Reopen target: technical design if these proofs cannot be produced without a new worker health surface or different worker lifecycle design.
  Evidence: Pending.

- [ ] T013 [App dependency readiness] Configure and prove the app remains private and dependency-gated before paid traffic.
  Files: Railway app service settings, app variable key posture, `/health/ready` evidence, logs, `railway.toml`, and network policy evidence.
  Depends on: T007, T010, T012.
  Proof: app variables enable Postgres, service auth, Redpanda, Redpanda readiness probe or stricter proof, microlease runtime, worker-readiness assertion only after T012, and authority remains disabled or inert until T018. `/health/ready` fails closed when required dependencies fail and passes only after startup admission, Postgres, broker, network policy, and bootstrap health. Replica/resource read-back matches approved baseline or records a design reopen.
  Reopen target: specification if public billing ingress or public `/metrics` becomes required; technical design if readiness behavior or resource baseline changes.
  Evidence: Pending.

- [ ] T014 [App deployment] Deploy/redeploy the app through the canonical Railway policy and prove protected readiness.
  Files: Railway app deployment read-backs, app logs, health evidence, and key-only variable posture.
  Depends on: T013.
  Proof: latest deployment succeeds, `/migrate` ran before promotion, `/health/ready` passes through private/non-public proof path, no public service/custom domain exists unless separately approved, no public `/metrics` exposure exists, app logs show no startup/config/readiness/network-policy errors, and deployment/resource IDs are recorded.
  Reopen target: implementation for narrow startup bugs; specification or technical design if proof requires public ingress or a different topology.
  Evidence: Pending.

- [ ] T015 [Billing service auth] Prove billing-service RS256/JWKS verifier, route scopes, and protected-route behavior.
  Files: `api/openapi/service.yaml`, `internal/api`, `internal/infra/http/service_auth.go`, route-scope tests, and app service-auth key posture.
  Depends on: T010, T014.
  Proof: `rtk make openapi-check` when contract/generated surfaces change; targeted tests prove RS256 only, non-empty `kid`, issuer/audience enforcement, JWKS fetch/rotation behavior without printing JWKS, missing token/invalid token/missing scope failures, and required scopes from the service-auth contract.
  Reopen target: specification if auth model or scope boundary changes; technical design if JWKS/readiness evidence surface changes.
  Evidence: Pending.

- [ ] T016 [Proxy provider contract] Verify clean `gonka-proxy` provider contract or sibling ledger before paid readiness.
  Files: `/Users/daniil/Projects/GonkaGate/gonka-proxy` read-only evidence, sibling tasks/contract artifacts if used, and sanitized proxy proof.
  Depends on: T009, T015.
  Proof: clean provider contract or approved sibling ledger proves JWKS publication, key rotation, issuer, audience, subject, token TTL, `kid`, exact variable key names, private billing URL, route scopes, microlease issue/readback/close calls, terminal/checkpoint/close event production, producer identity `gonka-proxy`, durable child-debit lineage before external execution, no `BILLING_SERVICE_AUTH_KEY` fallback, no proxy-local money writer, no Redis spend authority, and no direct reserve fallback for migrated cohorts.
  Reopen target: specification if the clean provider contract cannot meet these requirements; planning if proof exists but this ledger omitted a required verification step.
  Evidence: Pending.

- [ ] T017 [End-to-end provider proof] Prove private proxy-to-billing reachability, scoped calls, event flow, and readback without public ingress.
  Files: billing-service app read-backs, broker/topic/lag read-backs, proxy proof, and support-safe readback summaries.
  Depends on: T012, T014, T016.
  Proof: proxy reaches billing-service through Railway private networking, scoped JWT calls cover account/balance/usage/microlease/operation readback, terminal/checkpoint/close events arrive on approved topics, worker settles or reconciles them, billing facts outbox relay works, child-debit lineage is durable before external execution, and ambiguous outcomes use same-identity retry/readback.
  Reopen target: technical design if private reachability, event flow, or readback proof needs a different sequence; specification if proxy cannot satisfy the approved provider contract.
  Evidence: Pending.

- [ ] T018 [Authority gate] Enable only the already-approved authority mode and cohort state from the balance/usage cutover artifacts.
  Files: Railway variable key posture, `specs/balance-usage-authority-cutover/spec.md`, `specs/balance-usage-authority-cutover/rollout.md`, proxy cohort proof, billing readbacks, and rollback notes.
  Depends on: T006, T009, T012, T014, T017.
  Proof: authority remains `inert_expand` unless the balance/usage cutover artifacts and current proxy proof identify an approved `shadow_no_spend`, `internal_cohort`, or `migrated` state. For any enabled cohort, prove import/parity, active account state, old proxy writer disabled for that scope, direct reserve fallback disabled, scoped service auth, worker readiness, broker lag green, admission freshness <= 45s, and operator readbacks.
  Reopen target: specification if a new cohort, public paid rollout boundary, or fallback behavior must be decided; planning if the existing cross-spec gate is omitted from implementation proof.
  Evidence: Pending.

- [ ] T019 [Rollback and fail-closed proof] Prove rollback closes admission and does not revive forbidden authority.
  Files: Railway deployment/resource IDs, app/worker variable key posture, worker drain evidence, broker/topic posture, Postgres restore/reconciliation evidence, and proxy proof.
  Depends on: T018.
  Proof: pre-change deployment/resource IDs are recorded; authority/admission close happens before rollback; no new microleases after close; worker drains or stops only after safe in-flight/offset/commit evidence; broker topics/retention are not destructively changed; restored DB cutover remains manual and blocked until semantic reconciliation; direct reserve fallback, proxy-local money writer, Redis spend authority, and shared-key auth remain disabled for migrated scopes.
  Reopen target: specification if rollback requires proxy-local money writes, direct reserve fallback, public ingress, or weaker money authority; technical design if drain/reconciliation sequence changes.
  Evidence: Pending.

- [ ] T020 [Secret-free evidence audit] Audit all planning, ledger, logs, read-backs, and closeout evidence for privacy and secret boundaries.
  Files: `tasks.md`, `spec.md` validation/outcome updates, sanitized logs/read-backs, and any implementation-owned evidence notes.
  Depends on: T001 through T019.
  Proof: targeted privacy scan/classification plus `rtk make secret-scan` prove no raw secrets, DSNs, bearer tokens, JWTs, JWKS contents, private keys, broker credentials, Railway variable values, event payloads, request bodies, raw customer identifiers, dynamic proof URLs, prompts, completions, SSE chunks, provider payloads, or OTLP headers are recorded.
  Reopen target: planning if evidence capture method is unsafe; implementation if a narrow emitted-evidence bug can be fixed inside the ledger.
  Evidence: Pending.

- [ ] T021 [Validation bundle] Run the repository, Railway, database, broker, worker, service-auth, proxy, network, and rollback proof selected by `test-plan.md`.
  Files: changed source/config/generated files, Railway read-backs, broker/database proof, proxy proof, and validation logs.
  Depends on: T003, T004, T006, T007, T009, T012, T015, T017, T019, T020.
  Proof: run scope-matched commands from `docs/build-test-and-development-commands.md`, including `rtk make guardrails-check`, `rtk make fmt-check`, `rtk make lint`, `rtk make test`, `rtk make test-race`, `rtk make test-integration` or Docker equivalent when required, `rtk make migration-validate` or Docker equivalent, `rtk make sqlc-check` when SQL/migrations change, `rtk make openapi-check` when API/auth changes, `rtk make go-security`, `rtk make secret-scan`, `rtk make docker-build`, image inspection, and `rtk make docker-container-security` when Docker is available. Name skipped Docker/GitHub-only checks as limits, not proof.
  Reopen target: implementation for narrow failing proof inside approved scope; planning if proof expectations are incomplete; technical design/specification if failure exposes a missing decision.
  Evidence: Pending.

- [ ] T022 [Closeout] Update ledger-owned closeout surfaces after proof is current.
  Files: this `tasks.md` and `spec.md` `Validation`/`Outcome`; do not update `workflow-plan.md` merely because it exists after implementation starts.
  Depends on: T021.
  Proof: all required checkboxes and Evidence lines are current; `spec.md` records privacy-safe validation/outcome; `rtk git diff --check` passes; final evidence names any residual blocked proof with exact reopen target; no new workflow/process artifacts were created during implementation.
  Reopen target: planning if closeout proves the ledger missed required tasking; validation/implementation if proof failed inside approved scope.
  Evidence: Pending.

## Task-Ledger Review

Review result: PASS.

Reviewed against:

- `spec.md` approved full infrastructure decisions, constraints, non-goals, reopen conditions, and forward proof obligations;
- `workflow-plans/technical-design-review.md` eligible `CONCERNS` verdict and `TDR-PO1` through `TDR-PO7`;
- `design/overview.md`, `design/component-map.md`, `design/sequence.md`, `design/ownership-map.md`, `design/data-model.md`, `design/dependency-graph.md`, and `design/contracts/service-auth-and-broker.md`;
- `test-plan.md` validation families and privacy guard;
- `rollout.md` rollout, rollback, failback, and fail-closed choreography;
- `research/*.md` evidence and limits;
- repo-local source material named in `workflow-plan.md`;
- `specs/balance-usage-authority-cutover/spec.md` and `specs/balance-usage-authority-cutover/rollout.md` for migrated cohort authority-mode gates.

Coverage result:

- Source topology and live Railway preflight are represented by T001 and T002.
- Worker image repair is represented by T003.
- Worker non-HTTP readiness and evidence are represented by T004, T011, and T012.
- Dedicated Postgres, backups, PITR, restore, migrations, and semantic reconciliation are represented by T005, T006, and T007.
- Kafka-compatible broker, topics, topic policy, consumer group, and lag/backlog gates are represented by T008, T009, and T010.
- App private dependency-gated readiness is represented by T013 and T014.
- RS256/JWKS service auth and route scopes are represented by T015.
- Clean `gonka-proxy` provider contract and private end-to-end proof are represented by T016 and T017.
- Authority mode/cohort gating is represented by T018 without inventing a new cohort decision.
- Rollback/fail-closed proof is represented by T019.
- Secret-free evidence is represented across all tasks and audited by T020.
- Final validation and closeout are represented by T021 and T022.

Implementation readiness: CONCERNS.

Rationale: implementation may start because the ledger is dependency-ordered and does not require implementation to choose hidden architecture, ownership, contract, sequencing, rollout, or validation policy. Readiness remains `CONCERNS` because production paid readiness depends on live/external proof obligations that must be satisfied during execution: live source topology (`TDR-PO1`), worker image/readiness (`TDR-PO2`, `TDR-PO4`), broker/topic/lag proof (`TDR-PO3`), clean proxy provider contract (`TDR-PO5`), secret-free evidence (`TDR-PO6`), and fail-closed rollback (`TDR-PO7`).
