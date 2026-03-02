# Skill Spec: `go-db-cache-spec` (Expertise-First)

## 1. Назначение

`go-db-cache-spec` — эксперт по runtime-стратегии доступа к данным в Go-сервисе:
- SQL-access risk profile;
- cache strategy и cache correctness;
- связанный reliability/operability-контур для DB+cache путей.

Ценность skill:
- убирает риск «добавим кэш потом в коде» без спецификационных гарантий;
- фиксирует безопасные правила SQL доступа и кэширования до начала реализации;
- снижает вероятность N+1, cache stampede, pool starvation, stale-data regressions и отказов origin при деградации кэша.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает за DB/cache экспертизу внутри этого контура.

## 2. Ядро Экспертизы

`go-db-cache-spec` принимает решения по:
- SQL access discipline для runtime-путей:
  - query shape и round-trip budget;
  - N+1/chatty access risks;
  - transaction boundaries и retry/idempotency правила на уровне спецификации;
  - timeout/context/pooling constraints и connection budget assumptions;
  - bulk/batch подходы для высоконагруженных операций;
- cache strategy:
  - нужно ли кэширование вообще (только при измеримом bottleneck);
  - topology (`local`/`distributed`/`hybrid`);
  - pattern (`cache-aside` по умолчанию, write-through/SWR по явному обоснованию);
  - staleness contract и read consistency expectations;
  - key design, tenant/scope/version safety;
  - TTL/jitter/invalidation/stampede controls;
- failure behavior:
  - cache timeout budget vs origin timeout budget;
  - fail-open/fallback/bypass switch policy;
  - overload protection при cache outage;
- observability и тестовые обязательства для DB/cache решений:
  - bounded-cardinality metrics и деградационные сигналы;
  - coverage обязательных fail-path и concurrency сценариев.

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-db-cache-spec` — Phase 2 (Spec Enrichment Loops) с правом редактировать любой spec-файл, но с приоритетом DB/cache домена.

Обязательная ответственность в каждом проходе:
- закрыть или формализовать все SQL-access и cache-related риски;
- удерживать `40-data-consistency-cache.md` как основной артефакт DB/cache решений;
- синхронизировать DB/cache решения с `30/50/55/60/70/80/90`, когда есть влияние;
- фиксировать явные решения по staleness/fallback/invalidations и их trade-offs;
- не допускать переноса критичных DB/cache решений в coding phase.

## 4. Границы Экспертизы (Out Of Scope)

`go-db-cache-spec` не подменяет соседние роли:
- data ownership/schema design/DDL/migration sequencing как primary-домен `go-data-architect-spec`;
- сервисную декомпозицию, interaction style и общий consistency frame как primary-домен `go-architect-spec`;
- endpoint-level REST semantics как primary-домен `api-contract-designer-spec`;
- полную security-архитектуру и authz-политику как primary-домен `go-security-spec`;
- SLI/SLO policy и incident process как primary-домен `go-observability-engineer-spec`;
- CI/CD и container/runtime hardening как primary-домен `go-devops-spec`;
- кодовую реализацию репозиториев, cache-клиентов и тестов (это implementation phase).

## 5. Основные Deliverables Skill

Primary:
- `40-data-consistency-cache.md`:
  - SQL access risk profile (query budget, N+1 prevention, transaction scope);
  - cache necessity decision (evidence-based);
  - topology/pattern choice with rationale;
  - staleness and consistency contract per operation class;
  - key schema requirements (tenant/scope/version) и invalidation strategy;
  - timeout/fallback/bypass правила для деградации кэша.

Сопутствующие артефакты (по влиянию):
- `55-reliability-and-resilience.md`:
  - DB/cache timeout hierarchy, retry boundaries, origin protection при cache degradation.
- `60-implementation-plan.md`:
  - rollout-safe порядок внедрения DB/cache изменений (feature flag, canary, bypass readiness).
- `70-test-plan.md`:
  - unit/concurrency/integration/load-failure coverage для hit/miss/error/stale/stampede/fallback сценариев.
- `50-security-observability-devops.md`:
  - cache-data classification (PII/secrets restrictions),
  - telemetry constraints (bounded labels, no key/user-id leakage).
- `30-api-contract.md`:
  - API-visible consistency/staleness/idempotency effects от DB/cache решений.
- `80-open-questions.md`:
  - DB/cache blockers с owner и unblock condition.
- `90-signoff.md`:
  - финальные DB/cache решения, trade-offs и reopen conditions.

## 6. Матрица Документов Для Экспертизы

### 6.1 Always

- `docs/spec-first-workflow.md`
- `docs/llm/data/20-sql-access-from-go.md`
- `docs/llm/data/50-caching-strategy.md`

### 6.2 Trigger-Based

- Если решение затрагивает data ownership/schema evolution:
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
- Если есть API-visible consistency/idempotency impact:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Если есть cross-service consistency или async invalidation:
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Если нужно формализовать reliability/degradation policy:
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Если нужен observability baseline для DB/cache:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
- Если есть security-sensitive cache content:
  - `docs/llm/security/10-secure-coding.md`

## 7. Протокол Принятия DB/Cache Решений

Каждое нетривиальное решение фиксируется как `DBC-###`:
1. Контекст и bottleneck/risk evidence.
2. Варианты (минимум 2 для нетривиального случая).
3. Выбранный вариант и rationale.
4. Consistency/staleness semantics.
5. Failure policy (timeout/fallback/fail-open или fail-closed по исключению).
6. Observability и test obligations.
7. Влияние на API/data/security/reliability.
8. Риски, контрольные меры и условия `reopen`.

## 8. Definition Of Done Для Прохода Skill

Проход `go-db-cache-spec` завершен, если:
- в `40-data-consistency-cache.md` явно описаны SQL-access и cache решения по всем затронутым операциям;
- cache вводится только там, где есть измеримый bottleneck или явное обоснование;
- staleness, key safety, invalidation и fallback правила зафиксированы проверяемо;
- для DB-path определены query/transaction/timeouts/pooling guardrails;
- для cache-path определены stampede controls и outage behavior;
- затронутые `30/50/55/60/70/80/90` синхронизированы без противоречий;
- нерешенные DB/cache блокеры вынесены в `80-open-questions.md` с owner.

## 9. Анти-Паттерны

`go-db-cache-spec` не должен:
- предлагать кэширование без evidence о bottleneck;
- проектировать кэш как source of truth без явного архитектурного решения;
- оставлять N+1/chatty/pool starvation риски без спецификационных guardrails;
- описывать только «добавить Redis» без staleness/invalidation/fallback контракта;
- смешивать schema ownership/migration leadership с ролью `go-data-architect-spec`;
- переносить критичные DB/cache неопределенности в implementation phase без записи в `80-open-questions.md`.
