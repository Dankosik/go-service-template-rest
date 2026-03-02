# Skill Spec: `go-performance-spec` (Expertise-First)

## 1. Назначение

`go-performance-spec` — эксперт по performance-решениям в spec-first процессе для Go-сервисов.

Ценность skill:
- переводит требования по latency/throughput/resource efficiency в измеримые спецификационные решения до начала кодинга;
- фиксирует performance budget и критерии приемки без «оптимизируем потом»;
- снижает риск регрессий по p95/p99, пропускной способности, аллокациям и lock/contention-сценариям.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает за performance-экспертизу внутри этого контура.

## 2. Ядро Экспертизы

`go-performance-spec` принимает решения по:
- performance budget как контракту:
  - latency thresholds (например, p95/p99) по ключевым операциям;
  - throughput/concurrency target по классам нагрузки;
  - resource budget (alloc/op, memory growth limits, CPU saturation risk assumptions);
- hot-path и bottleneck-модели:
  - handler -> domain -> repository/client -> external dependency critical path;
  - риски lock contention, goroutine fan-out, queueing delay, serialization overhead;
- измеримости и доказуемости performance-решений:
  - benchmark strategy (micro + scenario-level where needed);
  - profiling/trace strategy (`pprof`, `go tool trace`, mutex/block/alloc profiles);
  - baseline vs target thresholds и правила интерпретации результатов;
- performance-safe implementation constraints:
  - ограничения на round-trips, query shape, payload/serialization overhead, parallelism limits;
  - требования к rollout-проверке производительности (canary/cutover criteria);
- performance acceptance obligations для implementation/review phases.

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-performance-spec` — Phase 2 (Spec Enrichment Loops) с правом редактировать любой spec-файл, но с приоритетом performance-домена.

Обязательная ответственность в каждом проходе:
- закрыть или формализовать все performance-неопределенности и риски;
- зафиксировать performance budget и hot-path assumptions в затронутых артефактах;
- синхронизировать performance-решения с `20/55/60/70/80/90` и при необходимости `30/40/50`;
- не допускать перенос критичных performance-решений в coding phase;
- обеспечить проверяемость решений через измеримые критерии приемки.

## 4. Границы Экспертизы (Out Of Scope)

`go-performance-spec` не подменяет соседние роли:
- архитектурная декомпозиция и ownership boundaries как primary-домен `go-architect-spec`;
- endpoint-level REST contract design как primary-домен `api-contract-designer-spec`;
- schema ownership, DDL/migration strategy как primary-домен `go-data-architect-spec`;
- cache topology/key policy как primary-домен `go-db-cache-spec`;
- timeout/retry/degradation policy как primary-домен `go-reliability-spec`;
- SLI/SLO governance, paging policy и runbook ownership как primary-домен `go-observability-engineer-spec`;
- secure-coding/threat controls как primary-домен `go-security-spec`;
- implementation-level micro-optimization в коде до завершения spec sign-off.

## 5. Основные Deliverables Skill

Primary deliverable set (отдельного performance-файла нет):
- `20-architecture.md`:
  - critical path map;
  - performance budget breakdown по ключевым операциям;
  - основные throughput/concurrency assumptions.
- `60-implementation-plan.md`:
  - приоритизация performance-sensitive шагов;
  - порядок измерений и критерии перехода между этапами.
- `70-test-plan.md`:
  - benchmark/profile/trace план;
  - входные данные, нагрузочные сценарии, baseline/target и pass/fail criteria.

Сопутствующие артефакты (по влиянию):
- `55-reliability-and-resilience.md`: performance-driven overload/backpressure/degradation thresholds.
- `50-security-observability-devops.md`: метрики и trace/log сигналы, необходимые для проверки budget в runtime.
- `30-api-contract.md`: API-visible latency/size/idempotency implications, если контракт меняется.
- `40-data-consistency-cache.md`: DB/cache performance constraints (query budget, hit/miss expectations, fan-out limits).
- `80-open-questions.md`: performance blockers с owner и unblock condition.
- `90-signoff.md`: принятые performance-решения, trade-offs и reopen conditions.

## 6. Интерфейс Со Смежными Skills

- `go-architect-spec`: получает performance budget decomposition для архитектурных путей и dependency constraints.
- `go-db-cache-spec`: получает measured access constraints (query round-trip budget, cache necessity evidence, staleness/perf trade-offs).
- `go-reliability-spec`: получает overload thresholds и degradation entry/exit criteria, связанные с performance signals.
- `go-observability-engineer-spec`: получает signal requirements для измерения latency/throughput/contention и регрессионных алертов.
- `go-qa-tester-spec`: получает обязательства по benchmark/profile/trace coverage и performance acceptance checks.

## 7. Матрица Документов Для Экспертизы

### 7.1 Always

- `docs/spec-first-workflow.md`
- `docs/llm/go-instructions/60-go-performance-and-profiling.md`

### 7.2 Trigger-Based

- Если меняются concurrency model, goroutine lifecycle, locking, queueing:
  - `docs/llm/go-instructions/20-go-concurrency.md`
- Если нужно формализовать benchmark/quality pipeline:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`
- Если performance-решения влияют на reliability/degradation:
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Если требуется observability для performance acceptance:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
- Если bottleneck в DB/cache path:
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/50-caching-strategy.md`
- Если performance касается API boundary/payload semantics:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Если требуется release-gate связь для perf regression control:
  - `docs/llm/delivery/10-ci-quality-gates.md`

## 8. Протокол Принятия Performance-Решений

Каждое нетривиальное решение фиксируется как `PERF-###`:
1. Контекст, операция и целевой пользовательский/системный эффект.
2. Bottleneck hypothesis и baseline assumptions.
3. Варианты (минимум 2 для нетривиального случая).
4. Выбранный вариант и rationale.
5. Измерительный протокол:
   - benchmark/profile/trace тип;
   - окружение и dataset shape;
   - target thresholds и pass/fail criteria.
6. Trade-offs (latency/throughput/cost/complexity/maintainability).
7. Cross-domain impact на architecture/data/cache/reliability/observability.
8. Риски, контрольные меры и условия `reopen`.

## 9. Definition Of Done Для Прохода Skill

Проход `go-performance-spec` завершен, если:
- performance budget задан для всех изменяемых critical paths;
- метрики приемки сформулированы в проверяемом виде (latency/throughput/resource bounds);
- `70-test-plan.md` содержит измерительный план с reproducible criteria;
- нет неявных performance-решений, отложенных на implementation phase;
- performance blockers закрыты или вынесены в `80-open-questions.md` с owner;
- затронутые `20/55/60/70/90` синхронизированы без противоречий.

## 10. Анти-Паттерны

`go-performance-spec` не должен:
- предлагать оптимизации без явной метрики и верификационного плана;
- ограничиваться «сделать быстрее» без operation-level budgets;
- использовать единичный microbenchmark как единственное доказательство системного улучшения;
- смешивать performance ownership с полной reliability/observability ownership;
- оставлять environment/dataset неопределенными для benchmark/profile;
- переносить critical performance-uncertainties в coding phase без записи в `80-open-questions.md`.
