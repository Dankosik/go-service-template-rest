# Skill Spec: `go-reliability-spec` (Domain Hard Skills)

## 1. Назначение

`go-reliability-spec` — экспертный spec-skill по reliability/resilience решениям в Phase 2 (`Spec Enrichment Loops`) spec-first процесса.

Ценность skill:
- переводит поведение при сбоях/перегрузке в явные контракты до кодинга;
- фиксирует timeout/retry/backpressure/degradation/lifecycle/rollout policy без "решим в реализации";
- снижает риск cascading failures, retry storms, probe flapping, unbounded queue growth и unsafe rollout;
- делает reliability-решения воспроизводимыми между сессиями за счет явного hard-skills ядра в самом skill-пакете.

`docs/spec-first-workflow.md` задает процесс и gate-логику; `go-reliability-spec` отвечает за предметную reliability-экспертизу внутри этого процесса.

## 2. Формат Hard Skills (как в AGENTS.md)

Для `go-reliability-spec` hard skills должны быть оформлены в том же инженерном формате, что и в `AGENTS.md` и усиленных runnable skills:
- `Mission`: какой reliability-риск skill обязан предотвращать до `Gate G2`;
- `Default Posture`: инженерные презумпции по умолчанию;
- доменные компетенции (`... Competency`) с исполняемыми правилами и default-числами;
- `Evidence Threshold`: обязательный уровень доказательности для `REL-###` решений;
- `Review Blockers For This Skill`: что блокирует sign-off.

Ключевой принцип: `Working Rules` задают порядок работы, а `Hard Skills` задают качество и enforceability решений.

## 3. Персонализированные Hard Skills Для `go-reliability-spec`

### 3.1 Mission

- Превращать reliability-intent в enforceable pre-coding contracts.
- Защищать `55-reliability-and-resilience.md` от неявных "implementation-only" решений.
- Гарантировать rollback-safe и incident-ready поведение изменяемых критичных путей.

### 3.2 Default Posture

- Сначала классифицировать dependency criticality, потом выбирать control-mechanisms.
- Explicit deadline + bounded retry + bounded concurrency — базовая норма, не опция.
- Предпочитать простые containment controls до сложных state-machine решений.
- Неопределенности по критичным reliability-фактам считать blocker до фиксации `[assumption]` + owner.
- Сохранять совместимость с mixed-version rollout (rolling/canary) по умолчанию.

### 3.3 Spec-First Workflow Competency

- Закрывать reliability-решения в spec-фазе, не переносить в coding phase.
- Держать `55-reliability-and-resilience.md` primary artifact.
- Синхронизировать последствия в `20/30/40/50/60/70/80/90`.
- Каждое нетривиальное решение маркировать `REL-###`.
- Незакрытые timeout/retry/degradation/rollback gaps считать blocker для `Gate G2`.

### 3.4 Dependency Criticality And Failure-Contract Competency

- Обязательная классификация каждой зависимости:
  - `critical_fail_closed`
  - `critical_fail_degraded`
  - `optional_fail_open`
- Для каждой критичной зависимости обязательны поля контракта:
  - timeout/deadline
  - retry class + retry budget
  - bulkhead limit + queue bound
  - fallback mode
  - circuit mode
  - observability trigger
- Для critical зависимостей обязательно: owner team, on-call route, rollback authority.

### 3.5 Timeout And Deadline Competency

- Default interactive end-to-end budget: `2500ms`.
- Reserve `100ms` for local response/cleanup.
- Fail-fast, если remaining budget < `150ms`.
- Default per-hop deadlines:
  - read/query: `300ms`
  - write/command: `1000ms`
  - absolute cap: `2000ms`
- Обязательная формула:
  - `outbound_deadline = min(per-hop default, remaining_inbound_budget - 100ms)`
- Запрещены implicit/infinite timeout defaults.

### 3.6 Retry Budget And Jitter Competency

- Retry default: `no retry`.
- Ретраи разрешены только для transient failures и retry-safe операций.
- Для retry-unsafe операций обязателен idempotency contract.
- Default interactive retry policy:
  - `1` retry (`2` total attempts)
  - exponential backoff + full jitter
  - base `50ms`, max `250ms`
- Retry budget per dependency обязателен:
  - extra retries `<= 20%` primary attempts в rolling `1m`
  - при исчерпании бюджета — disable retries + fail-fast
- Never-retry: validation/auth/authz/not-found/conflict/caller-canceled.

### 3.7 Overload, Backpressure, And Bulkhead Competency

- Все queue/channel/worker-lanes должны быть bounded.
- При queue depth > `80%` — включать degradation + shedding optional work.
- Явно различать rejection semantics:
  - `429` для policy/quota throttling
  - `503` для dependency/system exhaustion
- `Retry-After` обязателен при прогнозируемом горизонте восстановления.
- Обязательная per-dependency bulkhead isolation.
- Default dependency concurrency limit: `min(64, 2*GOMAXPROCS)`.

### 3.8 Circuit-Breaking And Containment Competency

- Default: `soft_retry_breaker` (retry budget + caps + shedding).
- State-machine breaker допустим только при incident evidence.
- Если state-machine включен, нужны явные thresholds (open/cooldown/half-open probes).
- Все breaker transitions должны быть observable.

### 3.9 Startup, Readiness, Liveness, And Shutdown Competency

- Строго разделять `/livez`, `/readyz`, `/startupz` semantics.
- Liveness не зависит от внешних зависимостей.
- Readiness отражает только core-traffic readiness.
- Anti-flap hysteresis обязателен.
- Shutdown sequence обязателен:
  1. set draining
  2. fail readiness
  3. stop new work
  4. drain inflight
  5. flush telemetry
  6. exit before hard kill
- Default drain timeout: `20s`.
- `terminationGracePeriodSeconds` > drain timeout + preStop budget.

### 3.10 Degradation And Fallback Competency

- Обязательная mode-модель:
  - `normal`
  - `degraded_optional_off`
  - `degraded_read_only_or_stale`
  - `emergency_fail_fast`
- Fallback rules зависят от criticality класса.
- Default stale fallback cap: `5m`.
- Deferred fallback должен использовать `202` + tracking ID.
- Activation/deactivation criteria и recovery criteria должны быть явными и наблюдаемыми.
- Dependency failure handling order обязателен: timeout/retry -> containment -> fallback -> explicit fail-fast.

### 3.11 API And Cross-Cutting Reliability Semantics Competency

- Reliability-visible behavior обязательно отражать в API contract:
  - retry class
  - idempotency requirements
  - overload response semantics
  - async acknowledgement semantics
- Default idempotency policy:
  - required key
  - dedup TTL `24h`
  - scope: tenant/account + operation + route/method
  - same key + same payload => equivalent outcome
  - same key + different payload => conflict (`409`/`ABORTED`)
- Запрещены fake-sync success responses для queued/unfinished side effects.

### 3.12 Sync, Async, And Distributed Workflow Reliability Competency

- Не добавлять sync-hop, если сценарий естественно async.
- Для state-change + publish обязателен outbox-equivalent atomic linkage.
- Для side-effecting consumers обязателен inbox/dedup contract.
- Ack/offset commit только после durable side effects.
- Async retry defaults: bounded, jittered, capped (`8` total attempts; `1s` base, factor `2`, cap `5m`).
- Non-retryable/poison сообщения — в DLQ с полным диагностическим контекстом.
- Для distributed flows обязательны:
  - invariant ownership
  - explicit state machine
  - compensation/forward-recovery path
  - reconciliation ownership/cadence
- 2PC и cross-system dual writes не являются default strategy.

### 3.13 Observability, SLO, And Budget-Gate Competency

- Reliability transitions должны иметь logs/metrics/traces contract.
- Обязательная наблюдаемость для timeout/retry/degradation/rollback/load-shedding состояний.
- SLI/SLO policy должна быть формализована через `good/total` и budget states.
- Burn-rate defaults:
  - `1h/5m @ 14.4` (page)
  - `6h/30m @ 6` (page)
  - `3d/6h @ 1` (ticket)
- Burn-rate paging без event floors недопустим.
- Rollout gates должны учитывать и service SLIs, и dependency saturation.

### 3.14 Delivery And Quality-Gate Competency

- Reliability-требования должны быть переведены в проверяемые обязательства в `70-test-plan.md`.
- Для reliability-sensitive изменений обязателен репозиторный validation path (`make test`, `make test-race`, `make test-integration` и смежные contract/migration checks при влиянии).
- Рискованные изменения требуют staged rollout checkpoints и explicit rollback authority в `60-implementation-plan.md`.

### 3.15 Data Evolution And Recovery Competency

- Reliability-impacting schema changes обязаны следовать `Expand -> Migrate/Backfill -> Contract`.
- Mixed-version compatibility обязательна до завершения contract phase.
- Backfills должны быть idempotent/resumable/throttled, с kill-criteria.
- Rollback class (`safe`/`conditional`/`restore-based`) и ограничения должны быть задокументированы.
- Backup strategy считается валидной только при restore-drill evidence.

### 3.16 Evidence Threshold And Review Blockers

Каждое `REL-###` решение обязано включать:
1. контекст, failure scenario, owner;
2. criticality и invariant impact;
3. минимум 2 альтернативы;
4. selected + rejected option rationale;
5. явные contract values;
6. verification obligations;
7. cross-domain impact;
8. reopen criteria.

Blockers для `go-reliability-spec`:
- нет explicit failure-contract для критичной зависимости;
- нет bounded timeout/retry/queue policy;
- нет явных degradation/rollback semantics;
- нет lifecycle (startup/readiness/liveness/shutdown) policy;
- изменена API-visible reliability semantics без обновления contract artifacts;
- критичная неопределенность перенесена в coding phase.

## 4. Матрица Переноса Из Referenced Docs

| Источник | Что перенесено в hard skills | Где зафиксировано |
|---|---|---|
| `docs/spec-first-workflow.md` | Phase/Gate discipline, `55` как primary artifact, запрет deferral в coding | `Spec-First Workflow Competency`, `Evidence Threshold` |
| `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md` | dependency criticality classes, timeout/retry defaults, bulkheads, degradation model, rollout/budget gates | `3.4`-`3.10`, `3.13` |
| `docs/llm/go-instructions/10-go-errors-and-context.md` | context deadline propagation, cancel/canceled semantics, boundary-safe error behavior | `3.5`, `3.11` |
| `docs/llm/go-instructions/20-go-concurrency.md` | bounded goroutines/queues, shutdown unblock paths, race-safe concurrency expectations | `3.7`, `3.9`, `3.12` |
| `docs/llm/api/10-rest-api-design.md` | idempotency/retry classification, `202` operation-resource pattern, status semantics | `3.11` |
| `docs/llm/api/30-api-cross-cutting-concerns.md` | `429/503/Retry-After`, input-limit enforcement, idempotency key conflict semantics | `3.11` |
| `docs/llm/architecture/20-sync-communication-and-api-style.md` | sync-hop decision, per-hop timeout defaults, bounded retry defaults | `3.5`, `3.12` |
| `docs/llm/architecture/30-event-driven-and-async-workflows.md` | outbox/inbox, bounded async retries, DLQ policy, ack-after-durable rule | `3.12` |
| `docs/llm/architecture/40-distributed-consistency-and-sagas.md` | explicit state machine, compensation/forward recovery, reconciliation ownership | `3.12` |
| `docs/llm/operability/10-observability-baseline.md` | reliability transition observability contract, correlation and low-cardinality discipline | `3.13` |
| `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md` | `good/total`, budget states, burn-rate windows, release/degradation linkage | `3.13` |
| `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md` | probe split, shutdown telemetry flush, async retry/DLQ observability, telemetry budget guardrails | `3.9`, `3.13` |
| `docs/llm/delivery/10-ci-quality-gates.md` | CI/release blocking gates, drift/compatibility checks, merge/release hard-stop logic | `3.14` |
| `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md` | expand-contract, backfill safety, rollback classes, backup/restore/PII reliability obligations | `3.15` |

## 5. Ответственность В Spec-First Workflow

`go-reliability-spec` в каждом проходе обязан:
- формировать reliability-решения в своей primary-domain зоне;
- синхронизировать последствия в затронутых артефактах, не перехватывая ownership у соседних skills;
- фиксировать unresolved blockers в `80-open-questions.md`;
- фиксировать принятые решения и reopen criteria в `90-signoff.md`.

## 6. Границы Экспертизы (Out Of Scope)

`go-reliability-spec` не подменяет:
- primary архитектурную декомпозицию (`go-architect-spec`),
- endpoint-level API design ownership (`api-contract-designer-spec`),
- data-model/DDL ownership (`go-data-architect-spec`),
- cache topology ownership (`go-db-cache-spec`),
- SLI/SLO governance ownership (`go-observability-engineer-spec`),
- secure-coding/threat catalog ownership (`go-security-spec`),
- CI/container implementation ownership (`go-devops-spec`),
- implementation-level coding до spec sign-off.

## 7. Deliverables

Минимальный набор deliverables в reliability-проходе:
- `55-reliability-and-resilience.md`: полный reliability policy package (including circuit-breaking/containment contract);
- `80-open-questions.md`: reliability blockers/assumptions с owner;
- `90-signoff.md`: accepted `REL-###` decisions + reopen conditions;
- по влиянию: `20/30/40/50/60/70` с явным статусом `updated` или `no changes required`.

## 8. Матрица Документов Для Экспертизы

### 8.1 Always

- `docs/spec-first-workflow.md`
- `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`

### 8.2 Trigger-Based

- `docs/llm/go-instructions/10-go-errors-and-context.md`
- `docs/llm/go-instructions/20-go-concurrency.md`
- `docs/llm/api/10-rest-api-design.md`
- `docs/llm/api/30-api-cross-cutting-concerns.md`
- `docs/llm/architecture/20-sync-communication-and-api-style.md`
- `docs/llm/architecture/30-event-driven-and-async-workflows.md`
- `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- `docs/llm/operability/10-observability-baseline.md`
- `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
- `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
- `docs/llm/delivery/10-ci-quality-gates.md`
- `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`

## 9. Протокол Принятия Reliability-Решений

Каждое нетривиальное решение фиксируется как `REL-###`:
1. Context + failure scenario.
2. Dependency criticality + invariant impact.
3. Минимум 2 варианта.
4. Selected + rejected rationale.
5. Contract values (timeout/retry/bulkhead/fallback/circuit/observability trigger/lifecycle/rollout).
6. Verification obligations (tests + signals).
7. Cross-domain impact.
8. Reopen conditions.

## 10. Definition Of Done Для Прохода Skill

Проход завершен, если:
- `55-reliability-and-resilience.md` содержит полный и непротиворечивый reliability-contract;
- каждая critical dependency имеет explicit timeout/retry/bulkhead/fallback/circuit/observability trigger/owner contract;
- overload/degradation/shutdown behavior формализованы и тестопригодны;
- startup/readiness/liveness semantics формализованы и anti-flap by policy;
- rollout/rollback gates имеют trigger + authority semantics;
- unresolved assumptions переведены в `80-open-questions.md` с owner/unblock condition;
- затронутые `20/30/40/50/60/70/90` синхронизированы без противоречий;
- нет активных пунктов из `Review Blockers For This Skill`.

## 11. Анти-Паттерны

`go-reliability-spec` не должен:
- оставлять timeout/retry/backpressure/degradation как общие слова без чисел и policy;
- допускать infinite timeout/retry/unbounded queues как implicit defaults;
- оставлять circuit/containment policy неявной для зависимостей с повторяющимися отказами;
- путать ownership reliability с ownership observability/security/devops;
- делать rollout без explicit rollback authority и gate-criteria;
- переносить critical reliability uncertainty в coding phase без blocker-tracking.
