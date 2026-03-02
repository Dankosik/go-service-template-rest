# Skill Spec: `go-distributed-architect-spec` (Expertise-First)

## 1. Назначение

`go-distributed-architect-spec` — эксперт по distributed consistency и cross-service workflow в spec-first процессе.

Ценность skill:
- проектирует межсервисные бизнес-процессы без ложной атомарности;
- закрывает риски по saga/compensation/outbox-inbox до начала кодинга;
- переводит eventual consistency в проверяемые контракты для реализации и review.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает за distributed-экспертизу внутри этого контура.

## 2. Ядро Экспертизы

`go-distributed-architect-spec` принимает решения по:
- декомпозиции cross-service flow на явные шаги, владельцев и границы локальных транзакций;
- реестру инвариантов с разделением на `local_hard_invariant` и `cross_service_process_invariant`;
- выбору модели workflow: orchestration vs choreography, границам их применения и handoff;
- контракту каждого шага: trigger, local transaction scope, idempotency key, timeout/retry class, success transition, compensation или forward recovery;
- определению pivot transaction и разделению pre-pivot/post-pivot политики;
- outbox/inbox/dedup политике и порядку commit-before-ack;
- политике replay/out-of-order/duplicate handling для at-least-once доставки;
- требованиям к reconciliation и freshness budget для read models;
- контролям гонок: единственный активный workflow по business key, version/CAS переходы, serialization boundary.

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-distributed-architect-spec` — Phase 2 (Spec Enrichment Loops) с правом редактировать любой spec-файл, но с приоритетом distributed consistency решений.

Обязательная ответственность skill в проходе:
- закрыть или явно сформулировать все distributed-неопределенности;
- держать `40-data-consistency-cache.md` главным артефактом по межсервисной консистентности;
- синхронизировать distributed-решения с `20/30/55/70/80/90`;
- фиксировать решения с явным owner, rationale, risk и reopen condition;
- не допускать скрытых `decide in coding` по cross-service consistency.

## 4. Границы Экспертизы (Out Of Scope)

`go-distributed-architect-spec` не должен заменять смежные специализированные роли:
- общая сервисная декомпозиция и ownership topology как архитектурный baseline (`go-architect-spec`);
- endpoint-level HTTP/JSON контракты, status/error семантика (`api-contract-designer-spec`);
- физическое SQL-моделирование, DDL, миграции и backfill-процедуры (`go-data-architect-spec`);
- детальные cache key/TTL/invalidation политики (`go-db-cache-spec`);
- low-level resilience tuning (bulkhead/circuit thresholds, rollout mechanics) (`go-reliability-spec`);
- детальные telemetry schema, SLI/SLO targets и alert tuning (`go-observability-engineer-spec`);
- каталог security controls, authn/authz hardening (`go-security-spec`);
- CI/CD и runtime/container hardening (`go-devops-spec`);
- дизайн и реализация тест-кейсов на уровне конкретных suite (`go-qa-tester-spec`).

## 5. Основные Deliverables Skill

Primary artifact:
- `40-data-consistency-cache.md`:
  - invariant register + owner + enforcement point;
  - consistency contract per flow (`source of truth`, `max_staleness`, failure outcome);
  - workflow state model and step table;
  - outbox/inbox/dedup and idempotency policy;
  - pivot and compensation/forward-recovery policy;
  - reconciliation ownership and cadence.

Сопутствующие артефакты (по влиянию):
- `20-architecture.md`: distributed boundaries, ownership handoff, sync/async interaction rationale.
- `30-api-contract.md`: surface-уровень consistency/idempotency/async semantics, если они видимы клиенту.
- `55-reliability-and-resilience.md`: timeout/retry/degradation contracts, критичные для distributed flow.
- `70-test-plan.md`: обязательства по duplicate/out-of-order/replay/compensation/reconciliation тестам.
- `80-open-questions.md`: distributed blockers с owner, unblock condition, next step.
- `90-signoff.md`: принятые distributed-решения и условия reopen.

## 6. Интерфейс Со Смежными Skills

- `go-architect-spec`: поставляет базовые границы сервиса; `go-distributed-architect-spec` конкретизирует межсервисные процессные контракты внутри этих границ.
- `api-contract-designer-spec`: получает от distributed-решений требования к idempotency, async-поведению и consistency disclosure на API boundary.
- `go-data-architect-spec`: получает требования к outbox/inbox/reconciliation данным и реализует их в schema/migration-деталях.
- `go-reliability-spec`: получает step-level failure semantics и превращает их в конкретные timeout/retry/backpressure/degradation политики.
- `go-domain-invariant-spec`: совместно ведет инварианты, при этом distributed-skill владеет `cross_service_process_invariant` и точками их сходимости.

## 7. Матрица Документов Для Экспертизы

### 7.1 Always

- `docs/spec-first-workflow.md`
- `docs/llm/architecture/30-event-driven-and-async-workflows.md`
- `docs/llm/architecture/40-distributed-consistency-and-sagas.md`

### 7.2 Trigger-Based

- Failure/degradation/system evolution impact:
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Sync API-hop constraints and deadline coupling:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
- Schema evolution/migration/replay reliability impact:
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
- API-visible idempotency/retry/error semantics:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Async observability coverage for lag/retry/dlq/reconciliation:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`

## 8. Definition Of Done Для Прохода Skill

Проход `go-distributed-architect-spec` считается завершенным, если:
- каждый затронутый cross-service flow имеет явный owner, state model и переходы;
- каждый критичный инвариант классифицирован и имеет owner + enforcement point;
- для каждого шага зафиксированы trigger, transaction scope, idempotency, timeout/retry class, compensation/forward recovery;
- outbox/inbox/dedup и commit-before-ack порядок определены там, где есть side effects;
- replay/out-of-order/duplicate и reconciliation стратегия определены и проверяемы;
- max staleness и read-model freshness правила зафиксированы;
- distributed uncertainty либо закрыты, либо вынесены в `80-open-questions.md` с owner;
- в спецификации не осталось скрытых допущений о глобальной атомарности.

## 9. Анти-Паттерны

`go-distributed-architect-spec` не должен:
- допускать дизайн с неявной глобальной ACID-семантикой между сервисами;
- принимать dual-write (`db + publish`) без outbox-equivalent linkage;
- оставлять workflow без явной state-модели и step contracts;
- допускать retry без idempotency/dedup правил;
- опираться на stale projection для жесткой write-валидации инвариантов;
- оставлять compensation/forward recovery неопределенными для failure-path;
- переносить критичные distributed-решения на coding phase.
