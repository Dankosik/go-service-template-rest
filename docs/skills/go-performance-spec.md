# Skill Spec: `go-performance-spec` (Domain Hard Skills)

## 1. Назначение

`go-performance-spec` — экспертный spec-skill по performance-дизайну в Phase 2 (`Spec Enrichment Loops`) spec-first процесса.

Ценность skill:
- фиксирует performance-требования как контракт до кодинга;
- переводит latency/throughput/allocation/contention риски в измеримые решения;
- исключает перенос критичных performance-решений в implementation phase.

`docs/spec-first-workflow.md` задает фазовый процесс и gate-логику, а `go-performance-spec` отвечает за предметную performance-экспертизу внутри этого процесса.

## 2. Формат Hard Skills (как в AGENTS.md)

Для `go-performance-spec` hard skills должны быть оформлены в том же инженерном формате, что и сильные инструкции в `AGENTS.md`:
- `Mission`: что skill защищает на пути к `Gate G2`;
- `Default Posture`: базовые инженерные презумпции по умолчанию;
- domain-компетенции (`... Competency`) с проверяемыми правилами;
- `Evidence Threshold`: обязательный уровень доказательности решений;
- `Review Blockers For This Skill`: что блокирует sign-off.

Почему это критично для performance-spec:
- процессные шаги (`Working Rules`) обеспечивают порядок работы,
- но именно `Hard Skills` задают качество и воспроизводимость performance-решений.

## 3. Персонализированные Hard Skills Для `go-performance-spec`

### 3.1 Mission

- Преобразовывать performance-intent в enforceable decisions до начала кодинга.
- Защищать спецификацию от «оптимизируем потом» и немеряемых требований.
- Обеспечивать, что каждая оптимизация имеет измеримый критерий приемки и rollback-safe контур.

### 3.2 Default Posture

- `Measure first`: без метрики, baseline и протокола измерения решение недействительно.
- Приоритет algorithmic/data-flow/query-roundtrip улучшений над микро-трюками.
- Усложнение допускается только при подтвержденном выигрыше и явном trade-off.
- Неопределенности по workload/budget/environment считаются blocker-уровнем, пока не зафиксированы как `[assumption]` с owner.
- Совместимость и rollout-safety по умолчанию важнее «локального ускорения» без эксплуатационного контракта.

### 3.3 Spec-First Workflow Competency

- Закрывать performance-решения в Phase 2 до `Spec Sign-Off`.
- Связывать решения с `PERF-###` и синхронизировать по артефактам `20/60/70/80/90` и при необходимости `30/40/50/55`.
- Не допускать неявных performance TODO в coding phase.
- Считать незакрытые budget/acceptance gaps blocker-условием для `Gate G2`.

### 3.4 Budget Modeling Competency

- Для каждого затронутого hot path задавать operation-level budget:
  - latency (`p95`/`p99`),
  - throughput/concurrency,
  - allocation/memory,
  - contention/CPU ограничения (когда релевантно).
- Делать бюджетную декомпозицию по цепочке `api -> domain -> db/cache -> outbound dependency`.
- Не использовать «среднюю latency» как primary acceptance criterion для user-critical путей.
- Для async контуров включать lag/backlog/retry/DLQ budget-ограничения.

### 3.5 Workload & Hot-Path Normalization Competency

- До выбора вариантов фиксировать workload-модель:
  - входной профиль,
  - распределение данных и skew,
  - конкурентную нагрузку,
  - hot-key/scan сценарии.
- Проверять сценарии `warm/cold`, `steady/peak`, `cache-up/cache-down`, `dependency-degraded`.
- Отделять toy benchmark предпосылки от production-like assumptions.
- Поддерживать единый critical-path map как source of truth для performance-решений.

### 3.6 Measurement Protocol Competency

- Каждый `PERF-###` обязан включать protocol:
  - benchmark/profile/trace тип,
  - environment + dataset shape,
  - baseline/target thresholds,
  - pass/fail criteria.
- Сравнение before/after должно быть воспроизводимым и сопоставимым по условиям.
- Microbenchmark не считается достаточным доказательством системного выигрыша без profile/trace/scenario evidence.
- При scheduler/locking симптомах обязателен trace-план (`go tool trace`).

### 3.7 Benchmark/Profile/Trace Competency

- Benchmark discipline:
  - `go test -bench`,
  - setup вне measured loop,
  - `-benchmem` при allocation-целях.
- Profiling discipline:
  - symptom-driven `pprof` (`cpu`, `heap`, `allocs`, `mutex`, `block`, `goroutine`),
  - profile до и после нетривиальных изменений.
- Trace discipline:
  - использовать tracing для blocking/scheduling/latency spikes вместо предположений.
- PGO — только как measured step после подтвержденного bottleneck analysis.

### 3.8 Concurrency And Contention Competency

- В concurrency-sensitive путях требовать explicit модель bounded parallelism:
  - лимиты fan-out,
  - queue/channel bounds,
  - lock contention assumptions,
  - cancellation/shutdown behavior.
- Привязывать concurrency-решения к validation obligations (`make test-race` или `go test -race ./...`).
- Блокировать решения с неограниченной конкуренцией, отсутствием stop-path, неучтенным backpressure.

### 3.9 DB And Cache Performance Competency

- DB performance contract обязан фиксировать:
  - query/round-trip budget,
  - N+1 prevention,
  - pool budget,
  - timeout/deadline assumptions.
- Cache performance contract обязан фиксировать:
  - cacheability/staleness class,
  - hit-ratio expectation,
  - stampede controls,
  - cache-down fallback и origin protection.
- Любой performance-выигрыш через cache/DB должен быть согласован с consistency/freshness semantics в `40-data-consistency-cache.md`.

### 3.10 API And Cross-Cutting Performance Competency

- Если performance меняет API-visible поведение, это должно быть явно отражено в контракте:
  - payload limits,
  - pagination policy,
  - retry/idempotency semantics,
  - `202` + operation resource для LRO.
- Не допускать fake-sync ответов для фактически async операций.
- При overload-зависимых envelope вводить явные деградационные исходы (`429`/`503`) и критерии их применения.

### 3.11 Observability And SLO Gating Competency

- Performance acceptance должен быть наблюдаем в runtime:
  - RED + saturation metrics,
  - bounded-cardinality labels,
  - trace/log correlation.
- Performance-цели user-facing путей должны иметь связь с SLI/SLO и budget-state decisions (когда релевантно rollout).
- Решения без production-verifiable signal contract считаются незавершенными.

### 3.12 Delivery And Quality-Gate Competency

- `70-test-plan.md` обязан содержать reproducible performance checks (benchmark/profile/trace).
- В плане должны быть конкретные команды и критерии, совместимые с репозиторным toolchain (`make test`, `make test-race`, benchmark/profile steps и т.д.).
- Для рискованных изменений нужны rollout checkpoints и rollback-safe критерии в `60-implementation-plan.md`.

### 3.13 Evidence Threshold Competency

Каждое нетривиальное performance-решение (`PERF-###`) обязано содержать:
1. контекст операции и workload;
2. baseline assumptions;
3. минимум 2 опции;
4. выбранную опцию + минимум одну отклоненную с явной причиной;
5. измерительный протокол + thresholds;
6. cross-domain impact summary;
7. acceptance и reopen criteria.

Решение без этих пунктов не считается sign-off ready.

### 3.14 Assumption & Uncertainty Discipline

- Все неизвестные критичные факты маркировать как `[assumption]`.
- Каждому предположению назначать owner и путь валидации.
- Неразрешенные critical assumptions уносить в `80-open-questions.md` как blockers.
- Не прятать неопределенность в формулировки вида «потом оптимизируем».

### 3.15 Review Blockers For This Skill

- Нет явного budget для затронутых critical paths.
- Нет reproducible measurement protocol для выбранного решения.
- Нет альтернативного сравнения и причины отклонения варианта.
- Заявление об улучшении основано только на microbenchmark или anecdotal evidence.
- Concurrency/contention path без bounded-concurrency и validation obligations.
- DB/cache optimization без query/cache/fallback contracts.
- API-visible performance semantics изменены без контрактной фиксации.
- Нет observability/SLO acceptance path для runtime-валидации.
- Critical uncertainty отложена в coding phase.

## 4. Матрица Переноса Из Referenced Docs

| Источник | Что перенесено в hard skills | Где зафиксировано |
|---|---|---|
| `docs/spec-first-workflow.md` | Phase/Gate дисциплина, запрет переноса решений в coding, обязательность артефактной синхронизации | `Spec-First Workflow Competency`, `Review Blockers` |
| `docs/llm/go-instructions/60-go-performance-and-profiling.md` | Measure-first, benchmark/profile/trace workflow, anti-guesswork, PGO как поздний шаг | `Default Posture`, `Measurement Protocol`, `Benchmark/Profile/Trace` |
| `docs/llm/go-instructions/20-go-concurrency.md` | bounded concurrency, lifecycle/cancel rules, contention awareness, race obligations | `Concurrency And Contention Competency` |
| `docs/llm/go-instructions/40-go-testing-and-quality.md` | benchmark discipline, deterministic validation, quality command expectations | `Benchmark/Profile/Trace`, `Delivery And Quality-Gate` |
| `docs/build-test-and-development-commands.md` | repo-native команды и воспроизводимый validation path | `Delivery And Quality-Gate Competency` |
| `docs/llm/architecture/20-sync-communication-and-api-style.md` | sync budget rules, timeout/retry/idempotency связь с latency envelope | `Budget Modeling`, `API And Cross-Cutting Performance` |
| `docs/llm/architecture/30-event-driven-and-async-workflows.md` | lag/backlog/retry/DLQ performance envelope, async throughput/freshness constraints | `Budget Modeling`, `Workload & Hot-Path`, `Observability` |
| `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md` | backpressure/load-shedding/circuit/bulkhead влияние на performance acceptance и rollout safety | `API And Cross-Cutting`, `Observability`, `Delivery And Quality-Gate` |
| `docs/llm/data/20-sql-access-from-go.md` | query round-trip budget, N+1 prevention, pool/timeouts, SQL performance observability | `DB And Cache Performance Competency` |
| `docs/llm/data/50-caching-strategy.md` | cache topology/staleness/hit-ratio/stampede/fallback contracts | `DB And Cache Performance Competency` |
| `docs/llm/api/10-rest-api-design.md` | payload/pagination/idempotency/LRO (`202`) как performance-visible API semantics | `API And Cross-Cutting Performance Competency` |
| `docs/llm/api/30-api-cross-cutting-concerns.md` | input limits, retry/idempotency, rate-limit/overload semantics | `API And Cross-Cutting Performance Competency` |
| `docs/llm/operability/10-observability-baseline.md` | RED+saturation baseline, low-cardinality, correlation contract | `Observability And SLO Gating Competency` |
| `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md` | SLI/SLO and error-budget linkage для performance decisions и rollout constraints | `Observability And SLO Gating Competency` |
| `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md` | telemetry cost/cardinality, async correlation, diagnostics contract | `Observability And SLO Gating Competency` |
| `docs/llm/delivery/10-ci-quality-gates.md` | quality gates, hard-stop criteria, reproducible evidence in CI/release flow | `Delivery And Quality-Gate Competency`, `Evidence Threshold` |

## 5. Ответственность В Spec-First Workflow

`go-performance-spec` в каждом проходе обязан:
- формировать измеримые performance-решения в своей primary-domain зоне;
- синхронизировать последствия в смежных артефактах без перехвата ownership у соседних skills;
- фиксировать unresolved blockers в `80-open-questions.md`;
- фиксировать принятые решения и reopen criteria в `90-signoff.md`.

## 6. Границы Экспертизы (Out Of Scope)

`go-performance-spec` не подменяет:
- primary архитектурную декомпозицию (`go-architect-spec`),
- API semantic ownership (`api-contract-designer-spec`),
- schema/migration ownership (`go-data-architect-spec`),
- cache implementation ownership (`go-db-cache-spec`),
- reliability policy ownership (`go-reliability-spec`),
- observability policy ownership (`go-observability-engineer-spec`),
- security control ownership (`go-security-spec`),
- implementation-level coding/optimization до sign-off.

## 7. Deliverables

Минимальный набор deliverables в performance-проходе:
- `20-architecture.md`: critical path + budget decomposition;
- `60-implementation-plan.md`: performance-sensitive sequencing + rollout checkpoints;
- `70-test-plan.md`: benchmark/profile/trace obligations + pass/fail thresholds;
- `80-open-questions.md`: performance blockers;
- `90-signoff.md`: принятые `PERF-###` решения и reopen criteria.

## 8. Definition Of Done Для Прохода Skill

Проход считается завершенным, если:
- все затронутые hot paths имеют явные budgets;
- каждое нетривиальное решение оформлено как `PERF-###` по evidence-threshold;
- измерительный протокол воспроизводим и связан с acceptance thresholds;
- нет скрытых performance-решений, отложенных в coding phase;
- unresolved blockers явно зафиксированы с owner/unblock condition;
- нет активных пунктов из `Review Blockers For This Skill`.

## 9. Анти-Паттерны

`go-performance-spec` не должен:
- писать «оптимизировать потом» без контрактного решения;
- делать выбор только на основе intuition/микробенча;
- игнорировать workload realism и degradation scenarios;
- предлагать complexity-first решение без доказанного выигрыша;
- оставлять performance acceptance без runtime telemetry и validation path.
