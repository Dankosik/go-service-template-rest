# Skill Spec: `go-reliability-spec` (Expertise-First)

## 1. Назначение

`go-reliability-spec` — эксперт по reliability/resilience требованиям в spec-first процессе для Go-сервисов.

Ценность skill:
- переводит поведение при сбоях и перегрузке в явные спецификационные контракты до начала кодинга;
- фиксирует timeout/retry/backpressure/degradation/shutdown/rollback policy без "решим в реализации";
- снижает риск cascading failures, retry storms, probe flapping, unbounded queue growth и unsafe rollout.

Этот документ определяет только scope и ответственность skill. Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`.

## 2. Ядро Экспертизы (Scope)

`go-reliability-spec` принимает решения по:
- классификации зависимостей и failure-contract:
  - `critical_fail_closed` / `critical_fail_degraded` / `optional_fail_open`;
  - owner/on-call/rollback authority для критичных зависимостей;
- timeout и deadline policy:
  - end-to-end budget и per-hop caps;
  - inbound->outbound deadline propagation rules;
  - fail-fast правила при недостатке оставшегося бюджета;
- retry policy:
  - retry eligibility class;
  - retry budget и jitter policy;
  - never-retry категории ошибок;
- overload control:
  - bounded queues/channels;
  - dependency bulkheads (per dependency concurrency lanes);
  - load shedding policy и rejection semantics (`429`/`503`, `Retry-After`);
- circuit and containment policy:
  - когда достаточно soft retry-breaker;
  - когда нужен state-machine breaker и какие минимальные пороги/переходы;
- graceful lifecycle:
  - startup/readiness/liveness responsibilities;
  - shutdown drain order и timeout contracts;
  - anti-flap правила для readiness;
- degradation and fallback modes:
  - mode model (`normal`, `degraded_optional_off`, `degraded_read_only_or_stale`, `emergency_fail_fast`);
  - activation/deactivation criteria и допустимые fallback semantics;
- rollout/rollback reliability safety:
  - canary progression and promotion gates;
  - rollback trigger authority и rollback-time expectations;
  - feature-flag safety requirements (owner/expiry/rollback behavior);
- reliability acceptance obligations:
  - что обязательно должно быть покрыто в `70-test-plan.md` для failure/degradation paths;
  - какие решения являются блокерами Gate G2.

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-reliability-spec` — Phase 0 и Phase 2 с правом редактировать любой spec-файл, но с приоритетом reliability-домена.

Обязательная ответственность в каждом проходе:
- в Phase 0 сформировать baseline reliability-risk сценариев и первичную заготовку `55-reliability-and-resilience.md`;
- в Phase 2 закрыть или явно формализовать все reliability-неопределенности;
- удерживать `55-reliability-and-resilience.md` как primary artifact reliability-решений;
- синхронизировать reliability-контракты с затронутыми `20/30/40/50/60/70/80/90`;
- не допускать переноса критичных reliability-решений в coding phase;
- обеспечить, чтобы к Gate G2 `55-reliability-and-resilience.md` содержал проверяемую policy по timeout/retry/backpressure/degradation/shutdown/rollback.

## 4. Границы Экспертизы (Out Of Scope)

`go-reliability-spec` не подменяет соседние роли:
- сервисная декомпозиция и ownership boundaries как primary-домен `go-architect-spec`;
- endpoint/resource modeling и payload-level API design как primary-домен `api-contract-designer-spec`;
- distributed workflow topology (orchestration/choreography, saga state model, outbox/inbox ownership) как primary-домен `go-distributed-architect-spec`;
- SQL schema design, DDL/migration execution mechanics как primary-домен `go-data-architect-spec`;
- cache topology/key strategy как primary-домен `go-db-cache-spec`;
- SLI/SLO target governance, alert routing и dashboard ownership как primary-домен `go-observability-engineer-spec`;
- secure coding и threat-control catalog как primary-домен `go-security-spec`;
- CI/CD implementation mechanics и container/runtime hardening как primary-домен `go-devops-spec`;
- code-level реализации retry wrappers, middleware, worker pools и shutdown hooks (implementation phase).

## 5. Основные Deliverables Skill

Primary artifact:
- `55-reliability-and-resilience.md`:
  - dependency criticality matrix и per-dependency failure contract;
  - timeout/deadline hierarchy и propagation rules;
  - retry budget/jitter policy + never-retry rules;
  - queue bounds, bulkheads, load shedding и overload response policy;
  - degradation mode model и fallback contract;
  - startup/readiness/liveness/shutdown policy;
  - rollout/rollback reliability gates и reopen criteria.

Сопутствующие артефакты (по влиянию):
- `20-architecture.md`:
  - resilience constraints, влияющие на архитектурные границы и dependency strategy.
- `30-api-contract.md`:
  - API-visible timeout/retry/idempotency/overload semantics.
- `40-data-consistency-cache.md`:
  - data/cache consistency implications от degradation/fallback/retry policy.
- `50-security-observability-devops.md`:
  - связка reliability policy с security fail-closed и observability/devops enforcement.
- `60-implementation-plan.md`:
  - rollout-safe sequencing внедрения reliability controls.
- `70-test-plan.md`:
  - failure-path/degradation-path тестовые обязательства.
- `80-open-questions.md`:
  - reliability blockers с owner и unblock condition.
- `90-signoff.md`:
  - принятые reliability-решения, trade-offs и reopen conditions.

## 6. Интерфейс Со Смежными Skills

- `go-architect-spec` задает boundaries/dependencies; `go-reliability-spec` задает failure/degradation behavior внутри этих границ.
- `go-distributed-architect-spec` задает workflow/consistency frame; `go-reliability-spec` задает timeout/retry/fallback contracts для шагов workflow.
- `go-observability-engineer-spec` владеет telemetry/SLI-SLO policy; `go-reliability-spec` формулирует какие reliability state transitions обязаны быть наблюдаемыми.
- `go-devops-spec` владеет pipeline/release enforcement; `go-reliability-spec` задает reliability-критерии promotion/rollback decisions.
- `go-security-spec` владеет threat controls; `go-reliability-spec` синхронизирует fail-closed/fail-degraded поведение без нарушения security invariants.
- `go-performance-spec` владеет latency/throughput budgets; `go-reliability-spec` использует эти ограничения для overload/degradation policy, не подменяя performance ownership.

## 7. Матрица Документов Для Экспертизы

### 7.1 Always

- `docs/spec-first-workflow.md`
- `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`

### 7.2 Trigger-Based

- Если в задаче есть context timeout/cancellation/error contracts:
  - `docs/llm/go-instructions/10-go-errors-and-context.md`
- Если меняются goroutine lifecycle, bounded queues, worker pools, shutdown coordination:
  - `docs/llm/go-instructions/20-go-concurrency.md`
- Если reliability меняет API boundary semantics (`429/503`, `Retry-After`, idempotency/retry rules, `202` fallback):
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Если есть sync/async/distributed workflow impact:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Если reliability требует observability/budget-aware release integration:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
  - `docs/llm/delivery/10-ci-quality-gates.md`
- Если reliability policy затрагивает migration/backfill/reconciliation behavior:
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`

## 8. Протокол Принятия Reliability-Решений

Каждое нетривиальное решение фиксируется как `REL-###`:
1. Контекст и failure scenario.
2. Dependency criticality class и business impact.
3. Варианты (минимум 2 для нетривиального случая).
4. Выбранный вариант и rationale.
5. Контракт:
   - timeout/deadline;
   - retry class/budget/jitter;
   - bulkhead/queue bounds;
   - fallback/degradation mode;
   - rollback trigger and authority.
6. Cross-domain impact (architecture/api/data/security/observability/devops/performance).
7. Test obligations и verification signals.
8. Риски, compensating controls и условия `reopen`.

## 9. Definition Of Done Для Прохода Skill

Проход `go-reliability-spec` завершен, если:
- `55-reliability-and-resilience.md` содержит полную, непротиворечивую и проверяемую reliability policy для всех затронутых critical paths;
- для каждой критичной зависимости определены timeout/retry/bulkhead/fallback contracts и owner;
- overload/degradation/shutdown behavior формализованы без "implementation-only" пробелов;
- rollout/rollback reliability gates сформулированы с явными trigger conditions;
- reliability uncertainty закрыты или вынесены в `80-open-questions.md` с owner и unblock condition;
- затронутые `20/30/40/50/60/70/90` синхронизированы без противоречий.

## 10. Анти-Паттерны

`go-reliability-spec` не должен:
- ограничиваться общими фразами "добавить retries/timeouts" без budget и eligibility правил;
- допускать infinite timeouts/retries/unbounded queues как implicit defaults;
- оставлять degradation activation criteria и recovery criteria неявными;
- смешивать reliability ownership с observability SLO governance или devops implementation mechanics;
- принимать rollout без explicit rollback authority и rollback-safe plan;
- переносить critical reliability-uncertainty в coding phase без записи в `80-open-questions.md`.
