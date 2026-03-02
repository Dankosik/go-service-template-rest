# Skill Spec: `go-qa-review` (Domain-Scoped Review)

## 1. Назначение

`go-qa-review` — экспертный review-skill по качеству тестов в Phase 4 (`Domain-Scoped Code Review`) spec-first процесса для Go-сервисов.

Ценность skill:
- подтверждает, что тестовая реализация соответствует утвержденному `70-test-plan.md`;
- выявляет пробелы покрытия, через которые могут пройти регрессии в критичных сценариях;
- удерживает test-suite в детерминированном, воспроизводимом и операционно надежном состоянии до merge.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает только за QA-review экспертизу в рамках Phase 4.

## 2. Формат Hard Skills (как в AGENTS.md)

Для `go-qa-review` hard skills фиксируются в стиле сильных skill-пакетов и по модели `AGENTS.md`:
- `Mission`: какой merge-risk skill обязан перехватывать;
- `Default Posture`: инженерные предпосылки по умолчанию;
- доменные компетенции (`... Competency`) с исполняемыми правилами;
- `Review Blockers For This Skill`: что считается merge-blocking в рамках QA-домена;
- явная граница domain ownership + handoff в соседние review-skills.

Такой формат делает skill автономным носителем hard-компетенции, а не только process-инструкцией.

## 3. Персонализированные Hard Skills Для `go-qa-review`

### 3.1 Mission

- Защищать `Gate G4` от ложной уверенности в качестве: тесты должны доказывать поведение, а не имитировать покрытие.
- Находить test gaps, из-за которых возможна утечка регрессий в critical paths.
- Давать минимальный и безопасный corrective path без подмены архитектурного/spec ownership.

### 3.2 Default Posture

- Начинать review от измененного поведения и соответствующих обязательств из `70-test-plan.md`.
- Приоритет: `contract safety -> deterministic reliability -> maintainability`, а не raw test count.
- Считать отсутствие критичных fail-path сценариев потенциальным merge-block до явного разрешения.
- Не выходить за QA-домен: deep архитектура/безопасность/перформанс — через handoff.

### 3.3 Spec-First QA Workflow Competency

- Применять ограничения `docs/spec-first-workflow.md` для Phase 4:
  - только domain-scoped findings;
  - точные ссылки `file:line`;
  - actionable fix path;
  - `Spec Reopen` при конфликте с утвержденным spec intent.
- Не редактировать spec-артефакты в код-ревью фазе.
- Рассматривать незакрытые `critical/high` QA-findings как blockers для `Gate G4`.

### 3.4 Coverage And Traceability Competency

- Требовать traceability changed behavior -> test obligation (`TST-*` или эквивалент из `70`).
- Проверять полноту обязательных слоев `unit/integration/contract` в затронутом scope.
- Проверять, что сценарии из `15` (инварианты) и `55` (fail-path/reliability) реально отражены в тестах через утвержденный test-plan.
- Фиксировать:
  - `orphan tests` (есть тест, нет spec-obligation),
  - `orphan obligations` (есть обязательство, нет проверяющего теста).

### 3.5 Assertion Quality And Diagnostics Competency

- Assert-ы должны проверять контрактное/наблюдаемое поведение, а не только отсутствие panic/error.
- Проверять силу проверок:
  - выходные данные,
  - side effects,
  - состояние,
  - error class/shape (там, где это часть контракта).
- Для wrapped errors требовать `errors.Is/As`, а не строковые сравнения.
- Проверять качество failure diagnostics: понятные test case names, локализация причины падения, отсутствие "opaque helper magic".

### 3.6 Determinism And Isolation Competency

- Фиксировать недетерминизм:
  - sleep-based синхронизация,
  - зависимость от scheduling luck,
  - uncontrolled time/random,
  - shared mutable state leakage,
  - внешние нестабильные зависимости без изоляции.
- `t.Parallel()` допустим только при явной изоляции данных и сайд-эффектов.
- Для concurrency-surface требовать race-suitability (`go test -race` / `make test-race`).

### 3.7 API Contract Test Competency (Trigger)

При изменениях API/контракта (`30-api-contract.md`, `docs/llm/api/10-rest-api-design.md`) проверять покрытие:
- HTTP method/status semantics (`200/201/202/204`, required `4xx/5xx` paths).
- `PUT` как full-replacement, `PATCH` как partial-update с корректной обработкой immutable/unknown fields.
- deterministic pagination/filter/sort semantics и reject для unsupported options.
- idempotency contract:
  - same key + same payload -> equivalent response,
  - same key + different payload -> conflict,
  - required key missing -> contract failure status.
- optimistic concurrency (`ETag`, `If-Match`, `412/428`) где применимо.
- async contract (`202 + Location + status resource`) и корректные state transitions.
- единый error-model (`application/problem+json`) без shape drift.

### 3.8 API Cross-Cutting Scenario Competency (Trigger)

По `docs/llm/api/30-api-cross-cutting-concerns.md` проверять тестовое покрытие:
- boundary validation pipeline и strict decode behavior (unknown fields/trailing JSON).
- request-size/transport guard semantics (`413`, `414`, `431`) для limit-enforced endpoints.
- auth principal/tenant propagation и object-level authorization fail paths.
- rate-limit semantics (`429`, `Retry-After`) и retry classification enforcement.
- correlation/request-id propagation в observability-visible путях.
- upload/webhook/long-running operation edge-cases, если затронуты.

### 3.9 Data Modeling And SQL Access Scenario Competency (Trigger)

По `docs/llm/data/10-sql-modeling-and-oltp.md` и `docs/llm/data/20-sql-access-from-go.md` проверять:
- покрытие DB-ограничений, выражающих доменные инварианты (uniqueness/nullability/fk assumptions).
- transaction correctness (`Begin -> defer Rollback -> Commit`) и conflict paths.
- optimistic locking conflict behavior там, где есть конкурентные апдейты.
- context deadline/cancel propagation в DB path.
- tenant isolation checks (особенно для pooled multi-tenant модели).
- отсутствие молчаливых query-per-item regressions в критичных read paths.

### 3.10 Migration And Schema Evolution Scenario Competency (Trigger)

По `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md` проверять наличие QA-доказательств для:
- mixed-version compatibility during rollout;
- корректности `expand -> backfill -> contract` assumptions;
- backfill idempotency/resumability и verification gates;
- rollback limitations на destructive этапах;
- consistency-safe evolution (без cross-system dual-write assumptions).

### 3.11 Cache Correctness And Degradation Scenario Competency (Trigger)

По `docs/llm/data/50-caching-strategy.md` проверять тест-матрицу:
- unit:
  - hit/miss/expired/negative/error/stale semantics (по применимости);
- concurrency:
  - stampede suppression, bounded origin calls, race-safety;
- integration:
  - cache-up и cache-degraded режимы;
  - fallback/bypass behavior;
- load/failure evidence:
  - деградация cache не должна приводить к неуправляемому провалу correctness/availability assumptions.

### 3.12 Security Negative-Case Test Competency (Trigger)

По `docs/llm/security/10-secure-coding.md` проверять покрытие:
- strict input validation + size limits at boundary;
- injection/SSRF/path traversal/file handling fail paths там, где релевантно;
- sanitize policy client-facing errors (без secret leakage);
- abuse-resistance guards (timeout/limit/concurrency) в expensive paths.

`go-qa-review` проверяет именно качество test coverage этих контролей; threat-depth ownership остается у `go-security-review`.

### 3.13 Performance And Concurrency Signal Competency (Trigger)

- Если изменения затрагивают concurrency behavior: требовать deterministic coordination tests и race suitability.
- Если тесты/код ссылаются на performance claims: требовать evidence path (benchmark/profile/trace), а не декларации.
- Deep-dive handoff:
  - `go-concurrency-review` для race/deadlock/leak lifecycle рисков;
  - `go-performance-review` для latency/throughput/profiling рисков.

### 3.14 Command And Quality-Gate Competency

Опора на `docs/build-test-and-development-commands.md` и `docs/llm/go-instructions/40-go-testing-and-quality.md`:
- базовый validation path:
  - `make test`
  - `go vet ./...`
  - `make lint` (или эквивалент)
- trigger-based:
  - `make test-race` для concurrency-sensitive изменений
  - `make test-integration` при интеграционных изменениях
  - `make openapi-check` при API contract/runtime изменениях.

Если команды не запускались или evidence отсутствует, это должно быть явно отражено в `Residual Risks`.

### 3.15 Evidence Threshold And Review Blockers

Каждая нетривиальная находка обязана содержать:
- `file:line`;
- нарушенное/непокрытое обязательство (желательно `TST-*`);
- конкретный regression leakage risk;
- минимальный путь исправления.

Merge-blockers для этого skill:
- отсутствуют критичные test obligations из утвержденного `70-test-plan.md`;
- test-suite системно недетерминирован и подрывает доверие к quality gates;
- assert-ы недостаточны для проверки required behavior;
- отсутствует required negative/fail-path coverage по затронутым API/data/cache/security/reliability surface;
- обнаружен spec mismatch, но `Spec Reopen` не поднят.

## 4. Матрица Переноса Из Referenced Docs

| Источник | Что перенесено в hard skills | Где зафиксировано |
|---|---|---|
| `docs/spec-first-workflow.md` | Phase 4 boundary, findings format, `Gate G4`, `Spec Reopen` | `3.3`, `3.15` |
| `docs/llm/go-instructions/70-go-review-checklist.md` | risk-first review posture, actionable findings, validation mindset | `3.2`, `3.5`, `3.15` |
| `docs/llm/go-instructions/40-go-testing-and-quality.md` | deterministic tests, table/subtest quality, anti-flaky rules | `3.5`, `3.6`, `3.14` |
| `docs/llm/api/10-rest-api-design.md` | method/status semantics, idempotency, ETag/preconditions, async `202`, error-model consistency | `3.7` |
| `docs/llm/api/30-api-cross-cutting-concerns.md` | validation pipeline, size limits, auth/tenant/correlation, rate limit, uploads/webhooks | `3.8` |
| `docs/llm/data/10-sql-modeling-and-oltp.md` | DB-encoded invariants, optimistic concurrency, deterministic pagination, tenant isolation | `3.9` |
| `docs/llm/data/20-sql-access-from-go.md` | transaction boundaries, ctx/deadline discipline, N+1 risk signals, query observability expectations | `3.9` |
| `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md` | mixed-version compatibility, phased rollout, backfill verification, rollback limits | `3.10` |
| `docs/llm/data/50-caching-strategy.md` | mandatory cache unit/concurrency/integration/failure test matrix, fallback/stampede controls | `3.11` |
| `docs/build-test-and-development-commands.md` | repo-native command baseline и CI-mapped validation path | `3.14` |
| `docs/llm/go-instructions/20-go-concurrency.md` | race-safety trigger signals and deterministic coordination expectations | `3.6`, `3.13` |
| `docs/llm/security/10-secure-coding.md` | security negative-case testing obligations and merge-gate checks | `3.12` |
| `docs/llm/go-instructions/60-go-performance-and-profiling.md` | evidence-first performance claims validation | `3.13` |

## 5. Ответственность В Spec-First Workflow

Ключевая роль `go-qa-review`:
- Phase 4 review с фокусом на test quality и соответствие `70-test-plan.md`.

Обязательная ответственность в каждом проходе:
- проверять, что тесты реально покрывают утвержденные обязательства в измененном scope;
- фиксировать findings только в QA-domain с `file:line`;
- связывать findings с конкретным spec source (`70`, и при необходимости `15/55/30/40/90`);
- давать минимальные practical fixes;
- не редактировать spec-файлы;
- поднимать `Spec Reopen`, если обнаружено spec-level рассогласование.

## 6. Scope Проверок (Что Проверяет Skill)

Обязательные QA-оси:
- `Coverage Conformance`
- `Critical Scenario Verification`
- `Assertion Quality`
- `Stability And Determinism`
- `Test Maintainability`

Сигнальные домены по trigger-документам:
- API contract / cross-cutting;
- data + SQL access;
- migrations/schema evolution;
- cache correctness and degradation;
- security negative-case coverage;
- concurrency/performance evidence handoff.

## 7. Deliverables И Формат Результата

Primary deliverable:
- записи в `reviews/<feature-id>/code-review-log.md`:

```text
[severity] [go-qa-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

Требования к findings:
- `Issue`: конкретный test-gap/weakness;
- `Impact`: реальный риск регрессии/ложной уверенности;
- `Suggested fix`: минимально достаточное исправление;
- `Spec reference`: ссылка на обязательство из `70` (и связанный артефакт при необходимости).

Обязательные секции ответа skill:
- `Findings`
- `Handoffs`
- `Spec Reopen`
- `Residual Risks`
- `Validation commands`

## 8. Эскалация И Severity-Политика

Severity:
- `critical`:
  - отсутствуют критичные test obligations;
  - тесты системно недетерминированы и дискредитируют quality gate;
- `high`:
  - значимый пробел в обязательном покрытии;
  - assert-ы не подтверждают required behavior;
- `medium`:
  - важная edge/fail-path неполнота с ограниченным краткосрочным риском;
- `low`:
  - локальные улучшения диагностики/поддерживаемости.

Эскалация:
- найдено рассогласование code vs approved spec intent -> `Spec Reopen`;
- merge через `Gate G4` невозможен до закрытия `Spec Reopen`.

## 9. Интерфейс Со Смежными Review Skills

- `go-domain-invariant-review`: инварианты как бизнес-контракты.
- `go-reliability-review`: timeout/retry/degradation/shutdown semantics.
- `go-concurrency-review`: race/deadlock/leak/lifecycle анализ.
- `go-performance-review`: benchmark/profile-driven correctness.
- `go-db-cache-review`: deep consistency/query/cache correctness.
- `go-security-review`: deep threat-model/security controls.
- `go-design-review`: итоговая design integrity проверка.

Правило:
- `go-qa-review` формулирует QA-risk и fix path, но не захватывает чужой primary-domain.

## 10. Матрица Документов Для Экспертизы

### 10.1 Always

- `docs/spec-first-workflow.md`
- `docs/llm/go-instructions/70-go-review-checklist.md`
- `docs/llm/go-instructions/40-go-testing-and-quality.md`
- `specs/<feature-id>/70-test-plan.md`
- `reviews/<feature-id>/code-review-log.md` (если есть)

### 10.2 Trigger-Based

- Invariants: `specs/<feature-id>/15-domain-invariants-and-acceptance.md`
- Reliability: `specs/<feature-id>/55-reliability-and-resilience.md`
- API contract/cross-cutting:
  - `specs/<feature-id>/30-api-contract.md`
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Data/cache/evolution:
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Validation commands baseline:
  - `docs/build-test-and-development-commands.md`
- Cross-domain risk handoff docs:
  - concurrency: `docs/llm/go-instructions/20-go-concurrency.md`
  - security: `docs/llm/security/10-secure-coding.md`
  - performance: `docs/llm/go-instructions/60-go-performance-and-profiling.md`

## 11. Протокол Фиксации Findings

Опциональный internal tracking ID: `QAR-###`.

Для каждой нетривиальной находки:
1. `Где`: точный `file:line`.
2. `Что не покрыто/слабо`: обязательство или quality-rule.
3. `Почему важно`: regression leakage risk.
4. `Как исправить`: минимальный corrective path.
5. `Как проверить`: релевантные команды/tests.
6. `Эскалация`: нужен ли handoff или `Spec Reopen`.

## 12. Definition Of Done Для Прохода Skill

Проход `go-qa-review` завершен, если:
- все критичные QA-оси проверены для измененного scope;
- `critical/high` findings оформлены с `file:line`, impact, fix, spec reference;
- flaky/nondeterminism риски явно выявлены или обоснованно отсутствуют;
- нет неэскалированных spec-level конфликтов;
- вывод остается в QA-domain, а cross-domain риски переданы через handoff;
- в результате явно указан `Validation commands` набор по измененному scope.

## 13. Анти-Паттерны

`go-qa-review` не должен:
- подменять проверку качества тестов подсчетом "количества тестов";
- принимать субъективные решения без привязки к regression-risk;
- захватывать primary-domain соседних review skills;
- предлагать архитектурный redesign под видом тестового замечания;
- оставлять критичные test gaps без явной эскалации;
- принимать spec-level решения в Phase 4 без `Spec Reopen`.
