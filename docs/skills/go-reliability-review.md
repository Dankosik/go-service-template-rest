# Skill Spec: `go-reliability-review` (Domain-Scoped Review)

## 1. Назначение

`go-reliability-review` — экспертный review-skill по проверке reliability/resilience-корректности в Phase 4 (`Domain-Scoped Code Review`) spec-first процесса для Go-сервисов.

Ценность skill:
- проверяет, что реализация соответствует утвержденным reliability-контрактам до merge;
- выявляет риски, которые приводят к outage-сценариям: неверные timeout/retry, overload collapse, unsafe shutdown, uncontrolled degradation;
- дает actionable findings в reliability-домене без подмены соседних reviewer-ролей.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает только за reliability-review экспертизу в рамках Phase 4.

## 2. Ядро Экспертизы

`go-reliability-review` принимает решения по:
- deadline/timeout propagation:
  - согласованность inbound и outbound deadline-контрактов;
  - отсутствие бесконтрольных или implicit infinite timeout-путей;
  - fail-fast поведение при исчерпании budget;
- retry correctness:
  - соблюдение retry eligibility и bounded retry budget;
  - наличие jitter/backoff вместо burst-retry паттернов;
  - исключение never-retry категорий ошибок из retry-loop;
  - соответствие retry-поведения idempotency-контрактам;
- backpressure и overload containment:
  - bounded queues/channels/concurrency;
  - корректность load shedding/rejection semantics (`429`/`503`, `Retry-After`) при перегрузке;
  - отсутствие путей с неограниченным ростом очереди или fan-out;
- graceful lifecycle:
  - корректность startup/readiness/liveness обязанностей;
  - shutdown drain order, cancellation propagation и bounded shutdown timeout;
  - отсутствие readiness flapping-рисков в известных fail/recover путях;
- degradation/fallback behavior:
  - явные activation/deactivation criteria для degraded modes;
  - соответствие fallback-семантики утвержденному spec intent;
  - отсутствие silent-failure поведения, маскирующего критичные сбои;
- rollout/rollback safety:
  - наличие проверяемых rollback trigger conditions и authority;
  - отсутствие изменений, которые требуют "ручного героизма" для отката;
  - соответствие feature-flag/gradual rollout ограничений reliability-требованиям;
- reliability fail-path test readiness:
  - соответствие coverage-obligations из `70-test-plan.md` для timeout/retry/overload/degradation/shutdown путей.

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-reliability-review`:
- выполнять reliability-focused review в Phase 4;
- подтверждать, что код не нарушает принятые решения из `55-reliability-and-resilience.md` и связанных spec-артефактов.

Обязательная ответственность в каждом проходе:
- оставлять findings только в reliability-domain;
- ссылаться на конкретный `file:line` и `Spec reference`;
- давать practical fix path, а не абстрактные советы;
- не редактировать spec-файлы в review-фазе;
- инициировать `Spec Reopen`, если safe fix невозможен без изменения утвержденного spec intent.

## 4. Scope Проверок (Что Проверяет Skill)

Обязательный проверочный scope:
- `Timeout/Deadline Conformance`:
  - не нарушены ли per-hop и end-to-end timeout contracts;
  - не исчезла ли propagation входного `context.Context` в критичных вызовах;
- `Retry Budget And Eligibility`:
  - retries ограничены и не приводят к retry storm;
  - retry применяются только к допустимым error-классам;
  - retry-поведение не дублирует side effects там, где нет idempotency guarantees;
- `Overload And Backpressure Safety`:
  - нагрузка ограничивается bounded механизмами;
  - при перегрузке система деградирует предсказуемо, а не зависает/раздувает очереди;
- `Startup/Readiness/Liveness/Shutdown Correctness`:
  - readiness отражает реальную готовность, а не "процесс жив";
  - shutdown-путь завершает критичные операции и останавливается в bounded time;
- `Degradation And Fallback Correctness`:
  - fallback режимы не ломают утвержденные reliability-invariants;
  - transition в degraded/recovery режимы формализованы и безопасны;
- `Rollout/Rollback Reliability Safety`:
  - risky changes имеют понятный rollback path;
  - отсутствуют rollout-шаги, создающие необратимые side effects без recovery strategy;
- `Reliability Test Traceability`:
  - для критичных fail-path есть проверяемые тестовые сигналы в объеме `70-test-plan.md`.

## 5. Границы Экспертизы (Out Of Scope)

`go-reliability-review` не подменяет соседние review-роли:
- не выполняет полный idiomatic/language-style review (`go-idiomatic-review`, `go-language-simplifier-review`);
- не выполняет архитектурный integrity-review как primary-domain (`go-design-review`);
- не выполняет deep performance profiling ownership (`go-performance-review`);
- не выполняет primary concurrency-аудит race/deadlock/lifecycle-механики (`go-concurrency-review`);
- не выполняет primary DB/cache correctness review (`go-db-cache-review`);
- не выполняет primary security-аудит (`go-security-review`);
- не выполняет общий test-strategy ownership (`go-qa-review`);
- не выполняет primary domain-correctness review (`go-domain-invariant-review`).

Также вне scope:
- redesign утвержденной архитектуры без явного spec-конфликта;
- редактирование spec-артефактов в review-фазе;
- блокирующие замечания без четкого reliability-impact.

## 6. Интерфейс Со Смежными Review Skills

`go-reliability-review` передает handoff:
- в `go-concurrency-review`, если root cause в race/deadlock/channel/goroutine lifecycle, а reliability-симптом вторичен;
- в `go-performance-review`, если решение требует benchmark/profile-level доказательств в hot paths;
- в `go-db-cache-review`, если reliability-risk вызван transaction/query/cache semantics;
- в `go-security-review`, если fallback/degradation или fail-open поведение создает security-риск;
- в `go-domain-invariant-review`, если retry/degradation/shutdown путь ломает бизнес-инварианты;
- в `go-qa-review`, если основной gap в отсутствии failure-oriented тестов;
- в `go-design-review`, если исправление выходит за рамки утвержденного design intent.

Правило интерфейса:
- `go-reliability-review` формулирует reliability-impact и минимальный safe fix,
- но не захватывает primary-domain другого review-skill.

## 7. Deliverable Формат Для Review-Лога

Primary deliverable:
- записи в `reviews/<feature-id>/code-review-log.md` в формате workflow:

```text
[severity] [go-reliability-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

Минимальные требования к finding:
- `Issue`: конкретный reliability-дефект или риск;
- `Impact`: как это влияет на отказоустойчивость, предсказуемость fail-path и merge-safety;
- `Suggested fix`: минимально достаточный реалистичный путь исправления;
- `Spec reference`: ссылка на релевантный spec-артефакт (обычно `55/70/90`, при необходимости `20/30/40/50/60`).

## 8. Матрица Документов Для Экспертизы

### 8.1 Always

- `docs/spec-first-workflow.md` (Phase 4, Reviewer Focus Matrix, Findings Format, Gate G4)
- `specs/<feature-id>/55-reliability-and-resilience.md`
- `specs/<feature-id>/60-implementation-plan.md`
- `specs/<feature-id>/70-test-plan.md`
- `specs/<feature-id>/90-signoff.md`
- `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`

### 8.2 Trigger-Based

- Если затронуты timeout/error wrapping/cancellation semantics:
  - `docs/llm/go-instructions/10-go-errors-and-context.md`
- Если затронуты goroutines/channels/worker pools/bounded queues:
  - `docs/llm/go-instructions/20-go-concurrency.md`
- Если reliability-semantics видны на API boundary:
  - `specs/<feature-id>/30-api-contract.md`
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Если reliability зависит от async/distributed flow:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Если reliability-impact связан с data/cache consistency:
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Если нужен контекст quality gates и release safety:
  - `docs/llm/delivery/10-ci-quality-gates.md`
- Если нужен контекст тестовой доказательной базы:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`

## 9. Severity И Эскалация

Severity-интерпретация в reliability-domain:
- `critical`:
  - подтвержденный риск outage/cascading failure/data-loss path;
  - unbounded retry/queue/timeout behavior в критичном пути;
  - unsafe shutdown/degradation/rollback behavior, блокирующий безопасный merge;
- `high`:
  - высокая вероятность срыва availability/SLO в типичном failure-сценарии;
  - существенное расхождение с утвержденными reliability-контрактами;
- `medium`:
  - локальный reliability-risk с ограниченным blast radius;
- `low`:
  - локальные улучшения надежности и предсказуемости без немедленного merge-block.

Эскалация:
- если safe fix требует изменения утвержденного spec intent, инициируется `Spec Reopen`;
- до закрытия `Spec Reopen` merge по `Gate G4` не считается завершенным.

## 10. Definition Of Done Для Прохода Skill

Проход `go-reliability-review` завершен, если:
- проверены все измененные reliability-sensitive пути по обязательному scope;
- все `critical/high` findings оформлены с `file:line`, impact, suggested fix и spec reference;
- кросс-доменные риски переданы через handoff в профильные reviewer-роли;
- нет неэскалированных spec-level конфликтов;
- при отсутствии проблем явно зафиксировано, что reliability findings не обнаружены.

## 11. Анти-Паттерны

`go-reliability-review` не должен:
- ограничиваться общими замечаниями "добавить retry/timeout" без budget/eligibility/bounds;
- считать приемлемыми implicit infinite timeout/retry/unbounded queue defaults;
- игнорировать graceful shutdown и degradation behavior, проверяя только happy-path;
- смешивать reliability review с полноценным performance/security/design review;
- блокировать merge без четкого failure-impact и без привязки к конкретному коду;
- оставлять spec-level mismatch без явной эскалации через `Spec Reopen`.

## 12. Статус Текущего Документа

Этот файл фиксирует `SCOPE` и `RESPONSIBILITIES` для будущего `SKILL.md` по `go-reliability-review`.
Runtime-инструкция (`Working Rules`, `Context Intake`, execution protocol) будет оформлена отдельным шагом.
