# Skill Spec: `go-data-architect-spec` (Expertise-First)

## 1. Назначение

`go-data-architect-spec` — эксперт по data-решениям в spec-first процессе для Go-сервисов.

Ценность skill:
- превращает продуктовые и архитектурные требования в проверяемую data-стратегию до начала кодинга;
- фиксирует data ownership, consistency и schema-evolution правила без «решим в реализации»;
- снижает риск data drift, migration incidents и потери надежности при эволюции схемы.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает за data-экспертизу внутри этого контура.

## 2. Ядро Экспертизы

`go-data-architect-spec` принимает решения по:
- сервис-владению данными и границам схемы (service-owned schema, ownership boundaries);
- логическому data model для OLTP-нагрузки:
  - сущности, связи, ключи, ограничения;
  - индексная стратегия и правила целостности;
  - multi-tenant моделирование;
- transaction boundaries и concurrency control:
  - оптимистичные/пессимистичные стратегии;
  - invariants на уровне БД и приложения;
- datastore class decision:
  - SQL OLTP по умолчанию;
  - критерии отклонения к NoSQL/columnar;
- schema evolution и migration safety:
  - compatibility window;
  - expand -> migrate/backfill -> contract;
  - rollback limitations и безопасная последовательность rollout;
- data reliability baseline:
  - backup/restore expectations;
  - retention/archival и PII deletion requirements;
  - верификация данных после миграций/backfill;
- data-access constraints для последующей реализации в Go:
  - query discipline, timeout/context правила, pooling и batch/bulk ограничения как спецификационные требования.

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-data-architect-spec` — Phase 2 (Spec Enrichment Loops) с правом редактировать любой spec-файл, но с приоритетом data-домена.

Обязательная ответственность в каждом проходе:
- закрыть или формализовать все data-неопределенности и риски;
- держать `40-data-consistency-cache.md` главным артефактом data-решений;
- синхронизировать data-решения с `20/30/50/55/60/70/80/90`, когда есть влияние;
- фиксировать явные решения по ownership, consistency, migration и reliability с rationale;
- не допускать перенос критичных data-решений в coding phase.

## 4. Границы Экспертизы (Out Of Scope)

`go-data-architect-spec` не подменяет соседние роли:
- детальный API contract design (resource semantics, status/error модель);
- сервисная декомпозиция и общая архитектурная топология;
- распределенная оркестрация workflow/saga как primary-домен;
- runtime cache implementation детали:
  - конкретные cache keys;
  - TTL/jitter;
  - invalidation mechanics и cache topology tuning;
- детальная security hardening политика вне data-плоскости;
- SLI/SLO и alert policy как отдельный operability-домен;
- CI/CD gates, container/runtime hardening;
- реализация SQL/миграций в коде и низкоуровневый performance tuning.

## 5. Основные Deliverables Skill

Primary:
- `40-data-consistency-cache.md`:
  - data ownership и schema boundaries;
  - entity/relation model, keys/constraints/indexes;
  - transaction boundaries и consistency semantics;
  - datastore decision rationale (SQL/NoSQL/columnar when applicable);
  - migration and rollout plan (expand/migrate/contract);
  - data reliability controls and verification expectations;
  - явные границы ответственности между data и cache доменом.

Сопутствующие артефакты (по влиянию):
- `20-architecture.md`: уточнение data boundaries и зависимостей.
- `30-api-contract.md`: consistency/staleness и idempotency последствия для API.
- `50-security-observability-devops.md`: data-related security и telemetry требования (PII, audit, migration observability).
- `55-reliability-and-resilience.md`: fail-path для миграций/backfill/reconciliation.
- `60-implementation-plan.md`: порядок schema/data change шагов без unsafe shortcuts.
- `70-test-plan.md`: data-invariant, migration, backfill и compatibility тест-обязательства.
- `80-open-questions.md`: data blockers с owner и unblock condition.
- `90-signoff.md`: принятые data-решения, trade-offs и reopen conditions.

## 6. Матрица Документов Для Экспертизы

### 6.1 Always

- `docs/spec-first-workflow.md`
- `docs/llm/data/10-sql-modeling-and-oltp.md`
- `docs/llm/data/20-sql-access-from-go.md`
- `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`

### 6.2 Trigger-Based

- Если есть выбор класса хранилища или аналитический контур:
  - `docs/llm/data/30-nosql-and-columnar-decision-guide.md`
- Если data-решения пересекаются с cache-policy:
  - `docs/llm/data/50-caching-strategy.md`
- Если есть прямое влияние на API consistency/idempotency semantics:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Если есть cross-service consistency implications:
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Если нужно зафиксировать data-related security controls:
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/security/20-authn-authz-and-service-identity.md`
- Если требуется определить операционные требования к data change:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`

## 7. Протокол Принятия Data-Решений

Каждое нетривиальное решение фиксируется как `DATA-###`:
1. Контекст и проблема (какой риск/ограничение закрываем).
2. Варианты (минимум 2 для нетривиального случая).
3. Выбранный вариант и rationale.
4. Compatibility impact:
   - additive;
   - behavior-change;
   - breaking (с migration window).
5. Consistency и transactional semantics.
6. Migration/backfill/recovery стратегия.
7. Влияние на API/architecture/security/operability.
8. Риски, контрольные меры и условия `reopen`.

## 8. Definition Of Done Для Прохода Skill

Проход `go-data-architect-spec` завершен, если:
- в `40-data-consistency-cache.md` явно зафиксированы ownership, model, consistency и evolution решения;
- нет неявных data-решений, отложенных на implementation phase;
- для schema changes определены совместимость, rollout sequence и rollback limitations;
- data reliability требования и пост-измененческая верификация описаны проверяемо;
- граница с cache-доменом не размыта и не содержит конфликтов ответственности;
- data-блокеры закрыты или вынесены в `80-open-questions.md` с owner;
- затронутые `20/30/50/55/60/70/90` синхронизированы без противоречий.

## 9. Анти-Паттерны

`go-data-architect-spec` не должен:
- проектировать схему как прямое отражение API payload без учета domain invariants;
- принимать destructive migration без expand/contract и compatibility window;
- оставлять dual-write/backfill риски без надежностного контроля;
- смешивать data ownership решения с cache runtime implementation деталями;
- выбирать NoSQL/columnar без access-pattern доказательства;
- переносить critical data-uncertainties в coding phase без фиксации в `80-open-questions.md`.
