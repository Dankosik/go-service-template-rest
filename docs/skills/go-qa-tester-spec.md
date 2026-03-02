# Skill Spec: `go-qa-tester-spec` (Expertise-First)

## 1. Назначение

`go-qa-tester-spec` — эксперт по спецификационному тест-дизайну в spec-first процессе для Go-сервисов.

Ценность skill:
- превращает утвержденные решения по домену/API/data/security/reliability в исполняемые test obligations до кодинга;
- фиксирует критерии достаточности тестовой стратегии в `70-test-plan.md`, а не оставляет их на implementation phase;
- снижает риск скрытых регрессий из-за непокрытых fail-path, idempotency/retry, consistency и abuse-сценариев.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает за hard testing-экспертизу внутри этого контура.

## 2. Формат Hard Skills (как в AGENTS.md)

Для `go-qa-tester-spec` hard skills задаются в том же формате, что и зрелые skill-пакеты и разделы в `AGENTS.md`:
- `Mission`: какую engineering-risk зону skill защищает до старта кодинга;
- `Default Posture`: какие тестовые презумпции действуют по умолчанию;
- доменные компетенции (`... Competency`) с проверяемыми правилами и explicit defaults;
- `Evidence Threshold`: какой уровень доказательности обязателен для test strategy решения;
- `Review Blockers For This Skill`: что блокирует `Spec Sign-Off` по тестовой части.

Такой формат делает skill не process-only, а предметно-исполняемым и воспроизводимым.

## 3. Персонализированные Hard Skills Для `go-qa-tester-spec`

### 3.1 Mission

- Зафиксировать test strategy как обязательный контракт до Phase 3, а не как «реализуем по ходу».
- Обеспечить трассируемость тестов к инвариантам (`15`) и fail-path контрактам (`55`).
- Защитить `G2` от скрытых testing-рисков, которые иначе всплывут на `G3/G4`.

### 3.2 Default Posture

- Risk-first: критичные ветки и отказные режимы важнее номинального happy-path покрытия.
- Minimal-sufficient level: сначала `unit`, эскалация в `integration/contract/e2e-smoke` только при необходимости доказательства.
- No deferred decisions: критичные тестовые решения не переносятся в coding phase.
- Testability-first: требование считается неготовым, если его нельзя проверить детерминированно и наблюдаемо.

### 3.3 Spec-First Workflow Competency

- Явно фиксировать текущую фазу, target gate и критерии готовности.
- Держать `70-test-plan.md` primary artifact, а `15/30/40/50/55/60/80/90` — синхронными по влиянию.
- Любую неоднозначность, влияющую на корректность стратегии, выносить в `80-open-questions.md` с owner/unblock condition.
- Не допускать противоречий между `70` и другими артефактами перед sign-off.

### 3.4 Test-Level Selection Competency

- По каждому существенному риску сравнивать минимум 2 варианта уровня теста и фиксировать отклоненный вариант.
- Использовать критерии выбора:
  - `unit`: локальная бизнес-логика и инварианты без внешних границ;
  - `integration`: БД/кэш/сеть/процессные границы, транзакции, timeout/cancel и деградационные ветки;
  - `contract`: HTTP/gRPC/async boundary-семантика (status/error/idempotency/correlation);
  - `e2e-smoke`: минимальная проверка критичного сквозного пути после спецификации.
- Не подменять доказательство уровня «удобством реализации».

### 3.5 Scenario Matrix Completeness Competency

Для каждого `TST-###` обязательны:
- `happy path`;
- `fail path`;
- `edge cases`;
- `abuse/negative path` при наличии trust-boundary;
- `idempotency/retry/concurrency` при side effects или параллелизме.

У каждого сценария должны быть explicit preconditions, test data, observable expected outcome и pass/fail criteria.

### 3.6 Invariant And Acceptance Traceability Competency

- Каждый критичный инвариант из `15-domain-invariants-and-acceptance.md` обязан иметь явное тестовое покрытие в `70-test-plan.md`.
- Acceptance criteria должны быть проверяемы тестом, а не декларативны.
- Разделять:
  - локальные hard invariants (строго на commit boundary);
  - cross-service process invariants (сходимость, staleness budget, reconciliation).

### 3.7 Reliability And Failure-Mode Competency

- Обязательные тест-обязательства по `55-reliability-and-resilience.md`:
  - deadline/timeout propagation;
  - bounded retry policy и no-retry классы;
  - backpressure/load-shedding behavior;
  - degradation/fallback mode transitions;
  - graceful shutdown/startup expectations.
- Для retry-safe/unsafe веток проверять idempotency/conflict semantics.
- Для async сценариев фиксировать error classification (`retryable/non-retryable/poison`) и DLQ/escalation expectations.

### 3.8 Error And Context Competency

- Проектировать проверки ошибок как часть контракта (`%w`, `errors.Is/As`, сохранение причинности).
- Явно покрывать `context.Canceled`/`context.DeadlineExceeded` и корректную propagation-модель.
- Избегать brittle string assertions на ошибки, если текст не является публичным контрактом.
- Требовать проверяемость cancel discipline для derived contexts.

### 3.9 API Contract And Cross-Cutting Competency

При API-влиянии стратегия обязана покрыть:
- method/status semantics и единый error model;
- retry classification и idempotency-key policy (dedup TTL `24h` по умолчанию);
- same-key/same-payload vs same-key/different-payload behavior;
- `202 Accepted` + operation resource lifecycle для long-running операций;
- boundary validation/normalization/size limits;
- correlation/request ID propagation и observability-visible outcomes.

### 3.10 Data, Migration, And Cache Competency

- Data-access obligations:
  - transaction boundary correctness;
  - optimistic conflict behavior;
  - deterministic pagination guarantees;
  - N+1/chatty query regression checks для измененных путей.
- Migration obligations:
  - `Expand -> Migrate/Backfill -> Contract` совместимость в mixed-version rollout;
  - backfill idempotency/resumability и verification gates;
  - rollback-limit awareness для contract phase.
- Cache obligations:
  - hit/miss/fallback correctness;
  - TTL+jitter and stampede controls;
  - fail-open behavior при деградации кэша;
  - tenant-safe keys и stale/negative semantics.

### 3.11 Security And Identity Negative-Path Competency

- Для security-sensitive изменений обязательны negative scenarios:
  - strict decode/validation/input-size enforcement;
  - authn/authz fail-closed behavior;
  - tenant mismatch и object-level authorization denial;
  - invalid/forged/expired token handling;
  - SSRF/path/file-upload misuse controls (если релевантно).
- Security obligations должны быть проверяемы на boundary behavior, а не только на внутренних предположениях.

### 3.12 Async And Distributed Consistency Competency

- Для event-driven/distributed flow фиксировать test obligations по:
  - outbox/inbox и dedup semantics;
  - replay/out-of-order tolerance;
  - ack-after-durable-state ordering;
  - compensation/forward recovery;
  - reconciliation-driven consistency checks.
- Нельзя считать workflow корректным без наблюдаемой и тестируемой модели eventual consistency.

### 3.13 Quality Gates And Execution Competency

- В стратегии обязательно задавать executable validation path через команды репозитория:
  - `make test`
  - `make test-race` (при concurrency-surface)
  - `make test-integration` (при boundary/integration рисках)
  - `go vet ./...` / `make lint` по scope изменений
  - `make openapi-check` / `make migration-validate` при contract/migration влиянии
- Требования должны быть совместимы с CI gates (`docs/llm/delivery/10-ci-quality-gates.md`).

### 3.14 Evidence Threshold And Review Blockers

Каждое нетривиальное testing-решение (`TST-###`) обязано содержать:
- owner и phase/gate context;
- риск/инвариант/контракт под проверкой;
- минимум 2 варианта и причину отклонения хотя бы одного;
- обязательный набор сценариев с observable pass/fail criteria;
- traceability к затронутым артефактам;
- явную ссылку на governing hard-skill competency для решения;
- residual risk и reopen condition.

Blockers для этого skill:
- отсутствует полный `70-test-plan.md` с обязательными секциями;
- happy-path-only план без fail/edge/abuse покрытия;
- нет явной трассировки к `15` и `55`;
- пропущены idempotency/retry/concurrency проверки при side effects;
- изменения API/data/security/distributed semantics без соответствующих test obligations;
- quality checks не связаны с реальными командами репозитория и CI.

## 4. Матрица Переноса Из Referenced Docs

| Источник | Что перенесено в hard skills | Где зафиксировано |
|---|---|---|
| `docs/spec-first-workflow.md` | фазовые ограничения, gate criteria, sync артефактов, запрет deferred test decisions | `Spec-First Workflow Competency` |
| `docs/llm/go-instructions/40-go-testing-and-quality.md` | deterministic test principles, level selection, quality pipeline | `Default Posture`, `Test-Level Selection`, `Quality Gates` |
| `docs/llm/go-instructions/10-go-errors-and-context.md` | error/context contract testing, cancellation/deadline checks | `Error And Context Competency` |
| `docs/llm/api/10-rest-api-design.md` | method/status semantics, idempotency TTL/conflict rules, async `202` contract | `API Contract And Cross-Cutting Competency` |
| `docs/llm/api/30-api-cross-cutting-concerns.md` | boundary validation/limits, auth context, correlation IDs, retry policy | `API Contract And Cross-Cutting`, `Security And Identity` |
| `docs/llm/architecture/20-sync-communication-and-api-style.md` | deadline/retry/idempotency defaults для sync-hops | `Reliability And Failure-Mode`, `API Contract` |
| `docs/llm/architecture/30-event-driven-and-async-workflows.md` | outbox/inbox, retry/DLQ classes, replay/order assumptions, async metrics obligations | `Async And Distributed Consistency`, `Reliability` |
| `docs/llm/architecture/40-distributed-consistency-and-sagas.md` | invariant ownership, step contracts, compensation/forward recovery, reconciliation | `Invariant Traceability`, `Async And Distributed Consistency` |
| `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md` | timeout/retry/backpressure/degradation/shutdown and rollout-risk behaviors | `Reliability And Failure-Mode Competency` |
| `docs/llm/data/10-sql-modeling-and-oltp.md` | transaction/local invariant, pagination, tenant/isolation expectations | `Data, Migration, And Cache Competency` |
| `docs/llm/data/20-sql-access-from-go.md` | SQL boundary testing: retries, timeouts, pool/round-trip and N+1 discipline | `Data, Migration, And Cache` |
| `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md` | expand-backfill-contract, compatibility/verification/rollback constraints | `Data, Migration, And Cache` |
| `docs/llm/data/50-caching-strategy.md` | cache correctness/fallback/stampede/TTL+jitter/tenant-safe keys and mandatory cache tests | `Data, Migration, And Cache` |
| `docs/llm/security/10-secure-coding.md` | strict validation, abuse controls, SSRF/path/upload negative paths | `Security And Identity Negative-Path Competency` |
| `docs/llm/security/20-authn-authz-and-service-identity.md` | auth context model, tenant isolation, object-level authz, negative access cases | `Security And Identity Negative-Path Competency` |
| `docs/llm/delivery/10-ci-quality-gates.md` | merge/release gate expectations, blocking quality evidence | `Quality Gates And Execution Competency` |
| `docs/build-test-and-development-commands.md` | repo-specific executable command baseline (`make test`, `test-race`, `test-integration`, `openapi`, `migration`) | `Quality Gates And Execution Competency` |

## 5. Ответственность В Spec-First Workflow

Ключевая роль `go-qa-tester-spec` — Phase 2 (`Spec Enrichment Loops`) с фокусом на test strategy completeness и implementability.

Обязательная ответственность в каждом проходе:
- закрыть или формализовать все testing-related неопределенности;
- вести `70-test-plan.md` как primary artifact;
- синхронизировать тестовые решения с `15/30/40/50/55/60/80/90`;
- не допускать переноса критичных тестовых решений в coding phase.

## 6. Границы Экспертизы (Out Of Scope)

`go-qa-tester-spec` не подменяет соседние primary domains:
- архитектурная декомпозиция и ownership topology;
- endpoint/resource design как primary-domain;
- SQL schema/migration mechanics как primary-domain;
- security catalog/policy design как primary-domain;
- SLI/SLO/alerting policy как primary-domain;
- CI/container implementation mechanics.

Также skill не пишет production/test code и не выполняет review роль `go-qa-review`.

## 7. Основные Deliverables Skill

Primary:
- `70-test-plan.md`:
  - scope and level matrix (`unit/integration/contract/e2e-smoke`);
  - rationale выбора уровня тестов;
  - traceability к `15/55` и другим решениям;
  - scenario matrix (`happy/fail/edge/abuse`);
  - quality-check expectations;
  - residual risks and reopen criteria.

Обязательные сопутствующие:
- `80-open-questions.md`
- `90-signoff.md`

Условные (по влиянию):
- `15-domain-invariants-and-acceptance.md`
- `30-api-contract.md`
- `40-data-consistency-cache.md`
- `50-security-observability-devops.md`
- `55-reliability-and-resilience.md`
- `60-implementation-plan.md`

## 8. Матрица Документов Для Экспертизы

### 8.1 Always

- `docs/spec-first-workflow.md`
- `docs/llm/go-instructions/40-go-testing-and-quality.md`

### 8.2 Trigger-Based

- Error/context/timeout semantics:
  - `docs/llm/go-instructions/10-go-errors-and-context.md`
- API and cross-cutting behavior:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Sync/async/distributed impacts:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Data/cache/migration impacts:
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Security/identity impacts:
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/security/20-authn-authz-and-service-identity.md`
- Gate/command alignment:
  - `docs/llm/delivery/10-ci-quality-gates.md`
  - `docs/build-test-and-development-commands.md`

## 9. Протокол Принятия Test-Решений

Каждое нетривиальное решение фиксируется как `TST-###`:
1. Контекст, риск и тестируемый контракт/инвариант.
2. Варианты (минимум 2) и выбранный уровень теста.
3. Причина отклонения альтернативы.
4. Обязательные сценарии и preconditions.
5. Observable pass/fail criteria.
6. Traceability к артефактам и decision IDs.
7. Governing hard-skill competency (какое правило skill применено).
8. Residual risks и reopen conditions.

## 9.1 Legacy Alignment (после добавления Hard Skills)

Точечная адаптация legacy-инструкций выполнена так, чтобы старые секции не противоречили `Hard Skills`:
- `Context Intake` stop condition теперь учитывает не только core testing axes, но и triggered domain axes.
- `Test Decision Protocol` требует явной связи каждого `TST-###` с конкретной hard-skill competency.
- `Output/Sign-Off` теперь ожидают decision register и explicit blocker visibility.
- `Definition Of Done` дополнен условием отсутствия активных `Review Blockers For This Skill`.
- `Anti-Patterns` переведены в explicit negative form, без «рекомендательного» формата.

## 10. Definition Of Done Для Прохода Skill

Проход `go-qa-tester-spec` завершен, если:
- текущая фаза и target gate зафиксированы явно;
- `70-test-plan.md` содержит все обязательные разделы и `TST-###`;
- критичные инварианты (`15`) и reliability fail-path (`55`) покрыты тестовыми обязательствами;
- API/data/security/distributed impacts отражены в test obligations, когда релевантны;
- blockers закрыты или вынесены в `80-open-questions.md` с owner/unblock condition;
- синхронизация `15/30/40/50/55/60/80/90` выполнена без противоречий;
- активные `Review Blockers For This Skill` отсутствуют;
- нет скрытых test decisions, отложенных до coding phase.

## 11. Анти-Паттерны

`go-qa-tester-spec` не должен:
- ограничиваться формулой «добавить unit и integration тесты» без матрицы сценариев;
- оставлять fail/edge/abuse пути без explicit coverage rationale;
- подменять тест-стратегию деталями реализации тестового кода;
- дублировать архитектурные/API/data/security решения без testing rationale;
- переносить критичные testing-решения в implementation phase без blocker tracking.
