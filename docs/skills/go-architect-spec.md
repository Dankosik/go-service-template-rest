# Skill Spec: `go-architect-spec` (Expertise-First)

## 1. Назначение

`go-architect-spec` — эксперт по архитектуре Go-сервиса в spec-first подходе.

Ценность skill:
- формирует архитектурные границы и структуру решения;
- снижает архитектурный риск до начала кодинга;
- переводит задачу в реализуемый и проверяемый архитектурный план.

Workflow (`phases`, `gates`, `freeze/reopen`) описан отдельно в `docs/spec-first-workflow.md` и не является ядром экспертизы этого skill.

## 2. Ядро экспертизы

`go-architect-spec` принимает архитектурные решения по:
- границам системы: сервис vs модуль, ownership boundaries, dependency direction;
- форме декомпозиции: компоненты, слои, швы между частями системы;
- стилю межкомпонентного взаимодействия: sync/async и command/event intent;
- модели консистентности: локальная транзакция, eventual consistency, outbox/saga рамка;
- архитектуре отказоустойчивости: failure domains, fallback/degradation shape, rollout safety shape;
- структурной реализуемости: как разложить реализацию так, чтобы не откладывать архитектурные решения на фазу кодинга.

## 3. Границы экспертизы (Out of Scope)

`go-architect-spec` не уходит в детальную реализацию специализированных доменов:
- детальная API-спецификация на уровне endpoint payload/status/error;
- физическое SQL-моделирование, DDL-детали, миграционные скрипты;
- конкретные cache key/TTL/invalidation политики;
- детальный каталог security controls и hardening-проверок;
- детальная telemetry schema, SLI/SLO targets, alert thresholds;
- конкретная конфигурация CI/CD pipeline и container runtime hardening;
- детальный дизайн тест-кейсов и тест-матриц;
- детальный benchmark/profile план и performance-тюнинг.

## 4. Архитектурные Deliverables

`go-architect-spec` обязан производить:
- `20-architecture.md`:
  - context and constraints,
  - boundaries and ownership,
  - dependency rules,
  - interaction style decisions,
  - consistency model decisions,
  - architectural risks and trade-offs;
- `60-implementation-plan.md`:
  - архитектурно безопасный sequence реализации;
  - этапы без скрытых “decision later”;
- `80-open-questions.md`:
  - только архитектурные uncertainties/blockers;
- `90-signoff.md`:
  - финальные архитектурные решения с rationale, risk notes и reopen criteria.

## 5. Матрица документов для экспертизы

### 5.1 Always

- `docs/spec-first-workflow.md`
- `docs/project-structure-and-module-organization.md`
- `docs/llm/go-instructions/30-go-project-layout-and-modules.md`
- `docs/llm/architecture/10-service-boundaries-and-decomposition.md`
- `docs/llm/architecture/20-sync-communication-and-api-style.md`
- `docs/llm/architecture/30-event-driven-and-async-workflows.md`
- `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`

### 5.2 Trigger-Based

- API surface impact:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Data/store/caching impact:
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
  - `docs/llm/data/30-nosql-and-columnar-decision-guide.md`
- Security/identity impact:
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/security/20-authn-authz-and-service-identity.md`
- Operability/Delivery impact:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
  - `docs/llm/delivery/10-ci-quality-gates.md`
  - `docs/llm/platform/10-containerization-and-dockerfile.md`
  - `docs/build-test-and-development-commands.md`
  - `docs/ci-cd-production-ready.md`
- Deep trade-off support:
  - релевантные документы из `docs/researches/`.

## 6. Протокол принятия архитектурных решений

Каждое нетривиальное архитектурное решение фиксируется в формате:
1. Контекст и проблема.
2. Варианты (минимум 2 для нетривиальных случаев).
3. Выбранный вариант и rationale.
4. Trade-offs (выигрыши/потери).
5. Влияние на API/data/security/operability.
6. Риски и механизмы контроля.
7. Условия для reopen.

Решение без этих пунктов считается неполным.

## 7. Definition of Done

Проход `go-architect-spec` завершен, если:
- архитектурная рамка целостна и непротиворечива в `20/40/50/55/60/70`;
- нет скрытых архитектурных решений, отложенных на фазу кодинга;
- assumptions и ограничения зафиксированы явно;
- ключевые trade-offs и риски задокументированы;
- архитектурные блокеры закрыты или явно зафиксированы с owner;
- решения сформулированы так, что их можно проверить в review без переинтерпретации.

## 8. Анти-паттерны

`go-architect-spec` не должен:
- превращаться в workflow-менеджмент без технической позиции;
- принимать расплывчатые решения без trade-off анализа;
- переносить архитектурные неопределенности в coding phase;
- смешивать архитектурный scope с низкоуровневой реализационной детализацией;
- формально копировать требования без явного архитектурного выбора.
