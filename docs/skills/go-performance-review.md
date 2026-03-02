# Skill Spec: `go-performance-review` (Domain-Scoped Review)

## 1. Назначение

`go-performance-review` — экспертный review-skill по performance-регрессиям и performance-correctness в Phase 4 (`Domain-Scoped Code Review`) spec-first процесса для Go-сервисов.

Ценность skill:
- проверяет, что реализация не нарушает утвержденные performance-бюджеты и допущения;
- выявляет hot-path деградации до merge на уровнях `latency/throughput/allocations/contention/I-O`;
- требует измеримую доказательную базу (`benchmark/profile/trace`), а не интуитивные оптимизации.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; `go-performance-review` владеет только performance-domain review в рамках Phase 4.

## 2. Формат Hard Skills (как в AGENTS.md)

Для `go-performance-review` hard skills задаются в том же формате, который уже показал результат в `go-idiomatic-review` и совпадает по структуре с `AGENTS.md`:
- `Mission`: что skill защищает на merge-path;
- `Default Posture`: дефолтная инженерная позиция и границы оценок;
- доменные компетенции (`... Competency`) с исполняемыми правилами и порогом доказательности;
- `Review Blockers For This Skill`: что блокирует merge именно в performance-домене;
- явное разделение domain ownership и handoff в соседние review-skills.

Этот формат делает skill автономным носителем hard-компетенции, а не только процессным чек-листом.

## 3. Персонализированные Hard Skills Для `go-performance-review`

### 3.1 Mission

- Защищать `Gate G4` от performance-регрессий в измененных и затронутых hot paths.
- Проверять соответствие реализации утвержденным performance-budget решениям из `specs/<feature-id>`.
- Давать минимальный безопасный corrective path без скрытого архитектурного redesign.

### 3.2 Default Posture

- `Evidence-first`: без измерений нет блокирующих performance-выводов.
- Сначала проверяется измененный diff и непосредственно затронутые пути, а не случайная оптимизация соседнего кода.
- Предпочтение простому коду, пока измерения не докажут значимый выигрыш от усложнения.
- Неограниченный параллелизм, очереди, ретраи и неявные таймауты трактуются как дефект до опровержения.

### 3.3 Spec-First Review Competency

- Соблюдать ограничения Phase 4 из `docs/spec-first-workflow.md`:
  - domain-scoped findings;
  - точные `file:line`;
  - practical fix path;
  - `Spec Reopen` при конфликте с утвержденным intent.
- Не редактировать spec-файлы в review-phase.
- Считать незакрытые `critical/high` performance-findings блокерами для merge.

### 3.4 Performance Budget Conformance Competency

- Сопоставлять изменения с зафиксированными `PERF-*` решениями и бюджетными ограничениями в `20/60/70/90`.
- Если high-risk hot-path изменен без явных acceptance-критериев/измерений, фиксировать это как finding.
- Проверять бюджет по relevant измерениям:
  - `p95/p99 latency`,
  - `throughput`,
  - `allocations/GC pressure`.

### 3.5 Hot-Path And Work-Amplification Competency

- Выявлять усиление вычислительной стоимости:
  - ухудшение асимптотики,
  - nested loops/повторные сканы,
  - избыточная сериализация/десериализация.
- Выявлять per-request overhead:
  - повторные одинаковые dependency calls,
  - лишние трансформации payload,
  - чатиность I/O в критическом пути.

### 3.6 Benchmark/Profile/Trace Evidence Competency

- Локальные performance-утверждения подтверждать benchmark-данными.
- При неопределенном bottleneck требовать `pprof` (CPU/heap/allocs/mutex/block/goroutine).
- При scheduler/blocking/tail-latency проблемах требовать `go tool trace`.
- Микробенчмарки не считать достаточными для system-level выводов без дополнительного evidence.
- Проверять воспроизводимость:
  - реалистичный workload,
  - baseline vs current,
  - корректная методология запуска.

### 3.7 Allocation And Memory Pressure Competency

- Оценивать рост аллокаций и GC-cost в горячих циклах только по фактическим данным.
- Предпочитать структурные улучшения (алгоритм/потоки данных/владение памятью) микро-трюкам.
- Скептически относиться к раннему `sync.Pool`/manual reuse без доказанного выигрыша.

### 3.8 Contention, Concurrency, And Scheduler-Cost Competency

- Фиксировать unbounded concurrency/queueing как риск collapse под нагрузкой.
- Требовать bounded concurrency и cancel-path для блокирующих операций.
- Отмечать lock/contention/queue wait как tail-latency риск.
- Для race/deadlock/lifecycle correctness делать handoff в `go-concurrency-review`.

### 3.9 I/O, DB, Cache, And API Latency Competency

- SQL access performance signals (`docs/llm/data/20-sql-access-from-go.md`):
  - `N+1`, query-in-loop, deep `OFFSET` в hot paths;
  - отсутствие дедлайнов на DB calls;
  - нарушение connection budget/pool discipline;
  - отсутствие query observability на критичных путях.
- Cache performance signals (`docs/llm/data/50-caching-strategy.md`):
  - кэш без bottleneck evidence;
  - отсутствие stampede protection/TTL jitter/fail-open fallback;
  - риск перегруза origin при деградации cache layer.
- API-level latency signals (`docs/llm/api/10-rest-api-design.md`, `docs/llm/api/30-api-cross-cutting-concerns.md`):
  - long-running (>~2s) операции без `202 + operation resource`;
  - небезопасная pagination-семантика, ведущая к latency cliffs;
  - отсутствие input/size guardrails, создающее load amplification.

### 3.10 Reliability/Overload Interaction Competency

- Проверять, что performance-чувствительные цепочки имеют явные дедлайны и propagation.
- Фиксировать retry-amplification паттерны:
  - retry-by-default,
  - unbounded retry,
  - retry non-transient failures.
- Проверять наличие backpressure/load-shedding/bulkhead controls, когда код создает queueing pressure.
- Проверять, что degradation/fallback поведение наблюдаемо и не скрывает критичные деградации.

### 3.11 Trigger-Driven Cross-Domain Signal Competency

- Concurrency-heavy diff: performance-проверка contention/scheduling + handoff в `go-concurrency-review`.
- DB/cache-heavy diff: performance-проверка round-trip/cache-cost + handoff глубокой correctness части в `go-db-cache-review`.
- Reliability-heavy diff: performance-влияние timeout/retry/backpressure + handoff policy-depth в `go-reliability-review`.
- API-shape changes: performance impact оценка + handoff contract-depth в API/design review.
- Test/measurement changes: оценка методологии evidence + handoff общей тест-стратегии в `go-qa-review`.

### 3.12 Evidence Threshold And Review Blockers

Каждая нетривиальная находка обязана включать:
- `file:line`;
- измеряемый риск (`latency`, `throughput`, `allocations`, `contention`, `I/O`);
- тип evidence (`benchmark`/`profile`/`trace`) или явно зафиксированное отсутствие обязательного evidence;
- минимальный безопасный fix path;
- как проверить исправление.

Merge-blockers для этого skill:
- high-risk hot-path изменение без обязательных измерений;
- доказанное нарушение утвержденного performance-budget;
- unbounded concurrency/queue/retry в критическом пути;
- очевидный DB/cache work amplification в hot paths;
- performance-sensitive outbound calls без explicit deadlines;
- spec-конфликт без `Spec Reopen`.

## 4. Матрица Переноса Из Referenced Docs

| Источник | Что перенесено в hard skills | Где зафиксировано |
|---|---|---|
| `docs/spec-first-workflow.md` | Domain-scoped review, Gate G4 blockers, `Spec Reopen`, findings format | `Spec-First Review Competency`, `Evidence Threshold` |
| `docs/llm/go-instructions/70-go-review-checklist.md` | review posture/order, actionable output, evidence-based performance checks | `Default Posture`, `Hot-Path`, `Evidence Threshold` |
| `docs/llm/go-instructions/60-go-performance-and-profiling.md` | measure-first workflow, benchmark/profile/trace rules, anti-guesswork baseline | `Performance Budget`, `Benchmark/Profile/Trace`, `Allocation` |
| `docs/llm/go-instructions/20-go-concurrency.md` (trigger) | bounded concurrency, cancellation path, goroutine/queue anti-pattern signals | `Contention, Concurrency`, `Trigger-Driven` |
| `docs/llm/go-instructions/40-go-testing-and-quality.md` (trigger) | benchmark methodology quality, test/race expectations | `Benchmark/Profile/Trace`, `Trigger-Driven` |
| `docs/build-test-and-development-commands.md` (trigger) | repo-native validation commands (`make test`, `make test-race`, `make test-cover`, `make lint`) | `Evidence Threshold`, deliverables validation path |
| `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md` (trigger) | timeout/retry/backpressure/bulkhead/load-shedding defaults and anti-patterns | `Reliability/Overload Interaction` |
| `docs/llm/data/20-sql-access-from-go.md` (trigger) | `N+1`/chatty SQL/pool budget/deadline/observability constraints | `I/O, DB, Cache, API Latency` |
| `docs/llm/data/50-caching-strategy.md` (trigger) | bottleneck-first cache policy, stampede protection, TTL+jitter, fail-open fallback | `I/O, DB, Cache, API Latency` |
| `docs/llm/api/10-rest-api-design.md` (trigger) | async `202` pattern for long ops, deterministic pagination, idempotency implications | `I/O, DB, Cache, API Latency` |
| `docs/llm/api/30-api-cross-cutting-concerns.md` (trigger) | input limits, retry/idempotency defaults, overload semantics (`429`/`503`) | `I/O, DB, Cache, API Latency`, `Reliability/Overload` |

## 5. Ответственность В Spec-First Workflow

Ключевая роль `go-performance-review`:
- экспертная проверка performance-correctness в Phase 4;
- контроль, что код не выходит за утвержденные performance-intent и budget-boundaries.

Обязательная ответственность в каждом проходе:
- фиксировать findings только в performance-domain;
- давать `file:line`, impact, minimal fix path, spec reference;
- разделять severity по merge-risk (`critical/high/medium/low`);
- явно фиксировать отсутствие обязательного evidence, когда это блокирует merge;
- при spec-level конфликте инициировать `Spec Reopen`, а не переопределять решения неявно.

## 6. Границы Экспертизы (Out Of Scope)

`go-performance-review` не подменяет соседние review-роли:
- не выполняет design-integrity/architecture review как primary-domain (`go-design-review`);
- не выполняет idiomatic/style review как primary-domain (`go-idiomatic-review`, `go-language-simplifier-review`);
- не выполняет full concurrency-correctness audit (`go-concurrency-review`);
- не выполняет DB/cache correctness audit (`go-db-cache-review`);
- не владеет reliability/security decision-making как primary-domain (`go-reliability-review`, `go-security-review`);
- не владеет полной тест-стратегией (`go-qa-review`).

Допустимо отмечать кросс-доменный сигнал, если он влияет на performance, но глубинная валидация передается через handoff.

## 7. Deliverables

Primary deliverable:
- записи в `reviews/<feature-id>/code-review-log.md` в формате:

```text
[severity] [go-performance-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

Обязательные секции ответа skill:
- `Findings`
- `Handoffs`
- `Spec Reopen`
- `Residual Risks`
- `Validation commands`

## 8. Матрица Документов Для Экспертизы

### 8.1 Always

- `docs/spec-first-workflow.md`
- `docs/llm/go-instructions/70-go-review-checklist.md`
- `docs/llm/go-instructions/60-go-performance-and-profiling.md`
- `specs/<feature-id>/20-architecture.md`
- `specs/<feature-id>/60-implementation-plan.md`
- `specs/<feature-id>/70-test-plan.md`
- `specs/<feature-id>/90-signoff.md`

### 8.2 Trigger-Based

- Concurrency-sensitive paths:
  - `docs/llm/go-instructions/20-go-concurrency.md`
- Benchmark/test methodology or local command expectations:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`
- Reliability/degradation/overload interactions:
  - `specs/<feature-id>/55-reliability-and-resilience.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- DB/cache bottleneck signals:
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/50-caching-strategy.md`
- API-visible latency/payload semantics:
  - `specs/<feature-id>/30-api-contract.md`
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`

## 9. Протокол Фиксации Findings

Каждую нетривиальную находку фиксировать в workflow-формате.
`PFR-###` можно использовать как внутренний идентификатор (опционально):
1. `Где`: точный `file:line`.
2. `Что`: конкретный performance-риск/регрессия.
3. `Какой измеряемый impact`: latency/throughput/allocations/contention/I-O.
4. `Evidence`: benchmark/profile/trace или явная фиксация отсутствия обязательного evidence.
5. `Как исправить`: минимальный и реалистичный fix path.
6. `Как проверить`: минимальный набор команд.
7. `Эскалация`: нужен ли handoff и/или `Spec Reopen`.

## 10. Definition Of Done Для Прохода Skill

Проход `go-performance-review` завершен, если:
- проверены performance-риски всех затронутых hot-path участков;
- все `critical/high` findings оформлены actionably и привязаны к spec obligations;
- conclusions основаны на проверяемых фактах или явно зафиксированной нехватке обязательных измерений;
- кросс-доменные риски переданы через handoff;
- нет скрытых spec-level противоречий (все вынесены в `Spec Reopen`).

## 11. Анти-Паттерны

`go-performance-review` не должен:
- давать performance-советы без измеримого риска и привязки к коду;
- блокировать merge по гипотезам без evidence-threshold;
- требовать микрооптимизации без подтвержденной пользы;
- подменять performance-review полноценным concurrency/security/design review;
- игнорировать approved spec intent и предлагать скрытый redesign вместо `Spec Reopen`.
