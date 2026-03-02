# Skill Spec: `go-performance-review` (Domain-Scoped Review)

## 1. Назначение

`go-performance-review` — экспертный review-skill по проверке performance-рисков и performance-регрессий в Phase 4 (`Domain-Scoped Code Review`) spec-first процесса для Go-сервисов.

Ценность skill:
- подтверждает, что реализация не нарушает утвержденные performance-ограничения из спецификации;
- выявляет hot-path регрессии до merge на уровне latency/throughput/allocations/contention;
- требует проверяемую evidence-базу (bench/profile/trace), а не гипотезы «может быть медленно».

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает только за performance-review экспертизу в рамках Phase 4.

## 2. Ядро Экспертизы

`go-performance-review` принимает решения по:
- соответствию реализации performance-budget контракту, утвержденному в spec-артефактах;
- корректности hot-path реализации:
  - асимптотика и лишняя вычислительная работа;
  - лишние аллокации и избыточные копирования;
  - лишние serialization/deserialization и payload overhead;
  - ненужные I/O round-trips в критическом пути;
- рискам contention/queueing/parallelism saturation, когда они проявляются как performance-деградация;
- качеству доказательств:
  - релевантные benchmark/profile/trace данные;
  - воспроизводимость условий измерения;
  - сопоставимость baseline и текущего результата;
- корректности выводов о performance-impact на основе фактов из diff-а и измерений.

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-performance-review`:
- Phase 4 review с primary-фокусом на performance correctness и regression control;
- подтверждение, что изменения безопасны относительно performance-требований, зафиксированных до coding phase.

Обязательная ответственность в каждом проходе:
- оставлять findings только в performance-domain;
- ссылаться на конкретный `file:line` и `Spec reference` (минимум `20/60/70/90`, при необходимости `55/40/30`);
- разделять:
  - доказанные регрессии/риски (`critical/high`);
  - вероятные риски с достаточной технической аргументацией (`medium`);
  - улучшения без merge-block (`low`);
- не редактировать spec-файлы в review-фазе;
- при выявлении spec-level противоречия инициировать `Spec Reopen`, а не менять требования неявно.

## 4. Scope Проверок (Что Проверяет Skill)

Обязательный проверочный scope:
- `Performance Budget Conformance`:
  - не нарушены ли утвержденные latency/throughput/resource bounds;
  - не добавлены ли новые hot-path решения без измеримых критериев;
- `Hot-Path Regression Risk`:
  - не появилась ли избыточная работа на каждом запросе/итерации;
  - не внесены ли заведомо дорогие операции в критический путь без обоснования;
- `Allocation And Memory Pressure`:
  - не выросли ли лишние аллокации/копирования в чувствительных участках;
  - нет ли признаков ускоренного memory growth/GC pressure;
- `Contention And Parallelism Cost`:
  - не ухудшены ли lock-wait/queueing/fan-out характеристики в нагрузочном пути;
  - нет ли неограниченного параллелизма, создающего performance collapse под нагрузкой;
- `I/O Efficiency Signals`:
  - нет ли явного роста round-trips/chattiness в сетевых и storage вызовах критического пути;
  - нет ли pattern-ов, которые очевидно повышают tail latency;
- `Evidence Quality`:
  - корректно ли построены benchmark/profile/trace проверки;
  - достаточно ли данных, чтобы заключение о performance было проверяемым.

## 5. Границы Экспертизы (Out Of Scope)

`go-performance-review` не подменяет соседние review-роли:
- не выполняет архитектурный redesign и проверку design-integrity как primary-domain (`go-design-review`);
- не выполняет idiomatic/style review как primary-domain (`go-idiomatic-review`, `go-language-simplifier-review`);
- не владеет полнотой тест-стратегии как primary-domain (`go-qa-review`);
- не владеет domain-invariants correctness как primary-domain (`go-domain-invariant-review`);
- не выполняет concurrency-correctness аудит (race/deadlock/lifecycle safety) как primary-domain (`go-concurrency-review`);
- не выполняет DB/cache correctness аудит как primary-domain (`go-db-cache-review`);
- не владеет reliability/security решениями как primary-domain (`go-reliability-review`, `go-security-review`).

Допустимо фиксировать кросс-доменный риск, если он влияет на performance, но глубинный анализ и финальное решение остаются у профильного review-skill.

## 6. Интерфейс Со Смежными Review Skills

- `go-qa-review`: handoff при недостаточной performance-test evidence (нет нужных benchmark/profile проверок).
- `go-db-cache-review`: handoff при подозрении на query/cache root-cause performance проблем.
- `go-concurrency-review`: handoff при рисках race/deadlock/lifecycle, где performance-симптом не является primary root cause.
- `go-reliability-review`: handoff при перегрузке, backpressure/degradation и timeout-сценариях как системной причине деградации.
- `go-design-review`: эскалация, если performance-fix требует изменения утвержденной архитектуры.
- `go-security-review`: handoff, когда proposed performance-change конфликтует с mandatory security controls.

Правило интерфейса:
- `go-performance-review` формулирует performance-impact и минимальный corrective path, не захватывая чужой primary-domain.

## 7. Deliverables И Формат Результата

Primary deliverable:
- записи в `reviews/<feature-id>/code-review-log.md` в формате workflow:

```text
[severity] [go-performance-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

Для нетривиальных performance-findings рекомендуется идентификатор `PFR-###` с минимальным набором:
1. `Где`: точный `file:line`.
2. `Что`: конкретный performance-риск/регрессия.
3. `Почему`: влияние на latency/throughput/allocations/contention.
4. `Evidence`: benchmark/profile/trace или явно обозначенное отсутствие обязательных измерений.
5. `Как исправить`: минимальный и реалистичный fix path.
6. `Эскалация`: нужен ли handoff в соседний review-domain или `Spec Reopen`.

## 8. Матрица Документов Для Экспертизы

### 8.1 Always

- `docs/spec-first-workflow.md` (Phase 4, reviewer scope, findings format, Gate G4).
- `specs/<feature-id>/20-architecture.md`.
- `specs/<feature-id>/60-implementation-plan.md`.
- `specs/<feature-id>/70-test-plan.md`.
- `specs/<feature-id>/90-signoff.md`.
- `docs/llm/go-instructions/60-go-performance-and-profiling.md`.

### 8.2 Trigger-Based

- Если затронуты goroutines/channels/mutex/worker pools:
  - `docs/llm/go-instructions/20-go-concurrency.md`
- Если проверка упирается в test/benchmark methodology:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`
- Если риск связан с timeout/retry/degradation policy:
  - `specs/<feature-id>/55-reliability-and-resilience.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Если bottleneck проходит через DB/cache path:
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/50-caching-strategy.md`
- Если performance зависит от API-shape/payload semantics:
  - `specs/<feature-id>/30-api-contract.md`
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`

## 9. Эскалация И Severity-Политика

`go-performance-review` использует стандартные severity (`critical/high/medium/low`) в performance-смысле:
- `critical`:
  - доказанная существенная деградация критичного пути;
  - отсутствие обязательной evidence-базы при явном high-risk изменении hot-path;
  - merge небезопасен без исправления или `Spec Reopen`;
- `high`:
  - сильный и обоснованный риск регрессии p95/p99/throughput/allocations в целевом сценарии;
- `medium`:
  - локальный, но заметный performance-risk с ограниченным impact radius;
- `low`:
  - точечные улучшения без блокирующего влияния.

Эскалация:
- если изменение выходит за утвержденные performance-решения или требует нового budget/acceptance-критерия, оформляется `Spec Reopen`;
- до закрытия `Spec Reopen` merge по `Gate G4` не считается завершенным.

## 10. Definition Of Done Для Прохода Skill

Проход `go-performance-review` завершен, если:
- выполнена проверка performance-рисков по измененным hot-path участкам;
- все `critical/high` findings оформлены с `file:line`, impact, suggested fix и spec-reference;
- conclusions основаны на проверяемых фактах (или явно зафиксированном отсутствии обязательной evidence-базы);
- кросс-доменные риски корректно переданы через handoff;
- нет неэскалированных spec-level конфликтов.

## 11. Анти-Паттерны

`go-performance-review` не должен:
- давать «performance advice» без конкретного риска и без привязки к коду;
- блокировать merge по гипотетическим опасениям без технической аргументации;
- подменять performance-review полноценным concurrency/security/design review;
- требовать преждевременных микрооптимизаций без доказанной пользы;
- игнорировать утвержденные spec-границы и предлагать скрытый redesign вместо `Spec Reopen`.
