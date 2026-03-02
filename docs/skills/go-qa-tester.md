# Skill Spec: `go-qa-tester` (Implementation Role)

## 1. Назначение

`go-qa-tester` — implementation-skill для тестового слоя в Phase 3 (`Code-Only Implementation`) spec-first процесса.

Ценность skill:
- переводит утвержденные test obligations из `70-test-plan.md` в исполняемый тестовый код без потери spec intent;
- доказывает в коде критичные инварианты (`15`) и reliability fail-path контракты (`55`), а не оставляет их декларациями;
- удерживает test-suite детерминированным, изолированным и совместимым с merge-gates.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает за hard testing execution внутри этого контура.

## 2. Формат Hard Skills (как в AGENTS.md)

Для `go-qa-tester` hard skills задаются в том же формате, который используется в зрелых skill-пакетах и в стиле `AGENTS.md`:
- `Mission`: какой implementation-risk skill должен закрыть в Phase 3;
- `Default Posture`: какие инженерные презумпции обязательны по умолчанию;
- доменные компетенции (`... Competency`) с операциональными правилами;
- `Evidence Threshold`: что считается достаточным implementation evidence;
- `Review Blockers For This Skill`: что блокирует готовность к `Gate G3`.

Такой формат делает `go-qa-tester` не process-only, а предметно-исполняемым skill для устойчивого качества тестовой реализации.

## 3. Персонализированные Hard Skills Для `go-qa-tester`

### 3.1 Mission

- Реализовать утвержденные test obligations как исполняемое доказательство поведения, а не как формальное «покрытие строк».
- Защитить `Gate G3` от ложной готовности, когда happy-path покрыт, но критичные fail/edge/negative сценарии пропущены.
- Исключить скрытое изменение контрактов/архитектуры через «додумывание» требований в тестах.

### 3.2 Default Posture

- Obligation-first: сначала обязательства из `70`, затем расширения.
- Risk-first: fail-path сценарии имеют тот же приоритет, что и happy-path.
- Smallest-proving-layer: `unit` по умолчанию, эскалация в `integration/contract` только при необходимости доказательства boundary behavior.
- Determinism-first: flaky/timing-based тесты считаются дефектом реализации.
- No ad-hoc intent: при неоднозначности — `Spec Clarification Request`, а не локальная интерпретация.

### 3.3 Phase-3 Spec-Freeze Execution Competency

- Выполнять только Phase 3 правила из `docs/spec-first-workflow.md`:
  - `Gate G2` уже пройден;
  - активен `Spec Freeze`;
  - spec-файлы не меняются без `Spec Reopen`.
- Реализовывать только утвержденные obligations из `70`, с обязательным сохранением инвариантов из `15` и fail-path требований из `55`.
- Любой конфликт или пробел, влияющий на корректность, блокирует реализацию соответствующего сценария до уточнения.

### 3.4 Obligation-To-Test Translation Competency

- Каждый obligation из `70` переводится в конкретный тест/группу тестов с явными pass/fail assertions.
- Если в `70` уже есть `TST-###`, реализация обязана сохранять эту трассировку в naming/structure.
- Для каждого реализованного obligations-набора требуется явная фиксация scenario classes:
  - `happy`;
  - `fail`;
  - `edge`;
  - `abuse/idempotency/retry/concurrency` — когда это требуется контрактом.
- Проверять наблюдаемое поведение (статусы, ошибки, side effects, persisted state, async outcome), а не внутренние детали реализации.

### 3.5 Determinism And Isolation Competency

- Запрещены sleep-based синхронизации как основной механизм ожидания.
- Контроль времени/рандома/порядка обязателен там, где это влияет на результат.
- Никакого скрытого shared mutable state между тестами.
- Cleanup обязан быть явным (`t.Cleanup`, reset env/config/hooks).
- `t.Parallel()` используется только при подтвержденной изоляции.

### 3.6 Invariant And Fail-Path Traceability Competency

- Все критичные инварианты из `15` должны иметь явные proving tests.
- Все критичные reliability fail-paths из `55` должны иметь executable coverage:
  - timeout/deadline propagation;
  - retry/no-retry классы;
  - degradation/fallback outcomes;
  - shutdown/cancel behavior (если релевантно).
- Трассируемость должна быть явной в структуре тестов и итоговом отчете выполнения.

### 3.7 Error And Context Competency

- Ошибки с wrap-chain проверяются через `errors.Is`/`errors.As`, не через brittle string matching.
- Проверять `context.Canceled` и `context.DeadlineExceeded` там, где есть deadline/cancel semantics.
- Проверять propagation request-context и отсутствие подмены на `context.Background()` в request paths.
- Для derived contexts проверять корректность cancel discipline и bounded completion.

### 3.8 Concurrency And Race Competency

- Для goroutine/channel/mutex путей тесты должны доказывать:
  - отсутствие зависаний и утечек;
  - корректное завершение при cancel/shutdown;
  - соблюдение bounded concurrency ожиданий.
- Для concurrency-sensitive scope обязателен race-evidence path (`make test-race` или эквивалент).
- Тайминговая «удача» не считается доказательством корректности.

### 3.9 API And Cross-Cutting Contract Competency

Когда изменяется transport-visible поведение, тесты обязаны покрывать:
- method/status semantics;
- error-model consistency;
- idempotency/retry semantics:
  - required key behavior;
  - same-key/same-payload equivalence;
  - same-key/different-payload conflict behavior;
- boundary validation/normalization/limits (`400/413/414/431/422` where applicable);
- `429` и retry guidance semantics, если заявлены контрактом;
- correlation/request-id behavior, если часть контракта;
- `202 + operation resource` lifecycle для async/LRO путей.

### 3.10 Data, Migration, And Cache Competency

- Data access obligations:
  - transaction-boundary correctness;
  - optimistic conflict semantics;
  - deterministic pagination behavior;
  - N+1/chatty regression checks для измененных критичных путей.
- Migration-sensitive obligations:
  - mixed-version compatibility assumptions;
  - idempotent/resumable backfill expectations (если влияет на поведение);
  - verification-aware read/write transition checks.
- Cache-sensitive obligations:
  - hit/miss/expired/fallback behavior;
  - TTL+jitter semantics (where contractually relevant);
  - stampede suppression under concurrency;
  - fail-open behavior при cache degradation;
  - tenant-safe key isolation checks.

### 3.11 Security And Identity Negative-Path Competency

- Для trust-boundary изменений обязательны negative tests по:
  - strict decode + validation + size limits;
  - authn/authz fail-closed behavior;
  - tenant mismatch and cross-tenant denial;
  - invalid/forged/expired credential behavior;
  - object-level authorization denial on resource-by-ID.
- Где релевантно, проверять SSRF/path traversal/file-upload misuse controls как boundary behavior.

### 3.12 Quality Gates And Command-Parity Competency

- Реализация тестов должна иметь executable validation path через команды репозитория:
  - `make test` (обязательно);
  - `make test-race` (при concurrency-surface);
  - `make test-integration` (при boundary/integration scope);
  - `go vet ./...` / `make lint` по scope;
  - `make openapi-check` при API/runtime contract влиянии;
  - `make migration-validate` при migration-sensitive scope.
- Итог implementation-pass не считается готовым без явных результатов этих проверок.
- Локальные проверки должны быть совместимы с CI gate логикой из `docs/llm/delivery/10-ci-quality-gates.md`.

### 3.13 Evidence Threshold And Review Blockers

Для каждого реализованного obligations-набора обязательно:
- source mapping (`70/15/55` + triggered docs при использовании);
- test-layer rationale (`unit/integration/contract`);
- перечень scenario classes и ключевых assertions;
- executed quality commands и outcomes;
- если есть блокер — explicit `Spec Clarification Request` с impacted artifacts.

Review Blockers для `go-qa-tester`:
- in-scope obligations не реализованы и не эскалированы;
- `15`/`55` критичные сценарии не доказаны тестами;
- happy-path-only реализация при наличии fail/edge требований;
- flaky/nondeterministic tests;
- concurrency scope без race/lifecycle evidence;
- изменение API/data/security/cache semantics без соответствующего test evidence;
- отсутствие/провал обязательных quality checks;
- spec ambiguity, влияющая на корректность, не эскалирована.

## 4. Матрица Переноса Из Referenced Docs

| Источник (из `go-qa-tester` SKILL) | Что перенесено в hard skills | Где зафиксировано |
|---|---|---|
| `docs/spec-first-workflow.md` | Phase 3 rules, G2/G3 constraints, Spec Freeze discipline, escalation flow | `Phase-3 Spec-Freeze Execution Competency`, `Review Blockers` |
| `docs/llm/go-instructions/40-go-testing-and-quality.md` | deterministic testing, table/subtest style, race/fuzz/quality baseline | `Default Posture`, `Determinism And Isolation`, `Quality Gates` |
| `docs/build-test-and-development-commands.md` | repo-native executable command baseline (`make test`, `test-race`, `test-integration`, `openapi`, `migration`) | `Quality Gates And Command-Parity Competency` |
| `docs/llm/go-instructions/10-go-errors-and-context.md` | `%w`, `errors.Is/As`, cancel/deadline propagation tests | `Error And Context Competency` |
| `docs/llm/go-instructions/20-go-concurrency.md` | goroutine lifecycle, cancellation, race/deadlock/leak sensitivity | `Concurrency And Race Competency` |
| `docs/llm/api/10-rest-api-design.md` | method/status, idempotency, retry class, `202` operation-resource semantics | `API And Cross-Cutting Contract Competency` |
| `docs/llm/api/30-api-cross-cutting-concerns.md` | strict boundary validation, size limits, correlation IDs, rate-limit semantics | `API And Cross-Cutting Contract Competency`, `Security And Identity` |
| `docs/llm/data/10-sql-modeling-and-oltp.md` | transaction-local invariants, conflict semantics, pagination determinism, tenant safety | `Data, Migration, And Cache Competency` |
| `docs/llm/data/20-sql-access-from-go.md` | SQL timeout/context/retry discipline, query-shape risk (N+1), observability expectations | `Data, Migration, And Cache`, `Quality Gates` |
| `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md` | expand/backfill/contract compatibility, backfill idempotency/resumability, verification gates | `Data, Migration, And Cache Competency` |
| `docs/llm/data/50-caching-strategy.md` | cache correctness matrix, stampede controls, fail-open behavior, mandatory cache test classes | `Data, Migration, And Cache Competency` |
| `docs/llm/security/10-secure-coding.md` | strict decode/limits, abuse-resistance negative paths, secret-safe error behavior | `Security And Identity Negative-Path Competency` |
| `docs/llm/security/20-authn-authz-and-service-identity.md` | fail-closed authz, tenant isolation, object-level checks, invalid-token negative paths | `Security And Identity Negative-Path Competency` |
| `docs/llm/delivery/10-ci-quality-gates.md` | merge-gate command parity, hard-stop mindset for failed checks | `Quality Gates And Command-Parity Competency`, `Review Blockers` |

## 5. Ответственность В Spec-First Workflow

Ключевая роль `go-qa-tester` — Phase 3 исполнение test obligations без изменения spec intent.

Обязательная ответственность в каждом pass:
- реализовать in-scope тесты из `70` в коде;
- сохранить трассировку к `15` и `55`;
- обеспечить deterministic/isolated качество test-suite;
- выполнить обязательные quality checks;
- эскалировать spec ambiguity вместо ad-hoc интерпретации.

## 6. Границы Экспертизы (Out Of Scope)

`go-qa-tester` не подменяет:
- `go-qa-tester-spec` (дизайн тест-стратегии);
- архитектурные/API/data/security/reliability решения как primary-domain;
- `go-qa-review` (review и оценка качества diff как отдельная роль);
- изменение spec-артефактов в `Spec Freeze`.

## 7. Основные Deliverables Skill

Primary deliverable:
- реализованный тестовый код по obligations из `70-test-plan.md`.

Обязательные сопутствующие deliverables в ответе skill:
- `Implemented Obligations` с source mapping;
- `Quality Checks` с фактами выполнения;
- `Escalations` (`Spec Clarification Request`, если есть);
- `Residual Risks` (если есть, иначе явно `none`).

## 8. Матрица Документов Для Экспертизы

### 8.1 Always

- `docs/spec-first-workflow.md`
- `docs/llm/go-instructions/40-go-testing-and-quality.md`
- `docs/build-test-and-development-commands.md`
- `specs/<feature-id>/70-test-plan.md`
- `specs/<feature-id>/15-domain-invariants-and-acceptance.md`
- `specs/<feature-id>/55-reliability-and-resilience.md`
- `specs/<feature-id>/60-implementation-plan.md`

### 8.2 Trigger-Based

- Error/context semantics:
  - `docs/llm/go-instructions/10-go-errors-and-context.md`
- Concurrency-surface:
  - `docs/llm/go-instructions/20-go-concurrency.md`
- API/cross-cutting semantics:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Data/cache/migration semantics:
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Security/identity semantics:
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/security/20-authn-authz-and-service-identity.md`
- Merge-gate alignment:
  - `docs/llm/delivery/10-ci-quality-gates.md`

## 9. Протокол Реализации Test Obligations

Для каждого нетривиального obligations-набора в реализации:
1. Определи source obligation (`70`, и при необходимости связанный `15/55`).
2. Выбери test level (`unit/integration/contract`) с кратким rationale.
3. Зафиксируй scenario classes (`happy/fail/edge/...`) и ключевые assertions.
4. Реализуй deterministic setup/cleanup и dependency isolation.
5. Проверь traceability (название теста/структура/комментарий) к source obligation.
6. Выполни scope-обязательные quality команды.
7. Если найден blocker/ambiguity — зафиксируй `Spec Clarification Request` с impacted artifacts.

## 9.1 Legacy Alignment (после добавления Hard Skills)

Точечная адаптация legacy-инструкций для `go-qa-tester` зафиксирована так:
- `Working Rules` теперь явно требуют применять `Hard Skills` как норму исполнения, а не только workflow-шаги.
- Добавлен explicit precedence rule: при конфликте старых формулировок и `Hard Skills` решающими считаются `Hard Skills`.
- `Output Expectations` синхронизированы с новым evidence-подходом (`Implemented Obligations`, `Quality Checks`, `Escalations`, `Residual Risks`).
- `Definition Of Done` дополнен условием отсутствия активных blocker-пунктов из `Review Blockers For This Skill`.
- `Anti-Patterns` оставлены в explicit negative форме, чтобы не конфликтовать с обязательными competencies.

## 10. Definition Of Done Для Прохода Skill

Проход `go-qa-tester` завершен, если:
- in-scope obligations реализованы или явно эскалированы;
- критичные `15` invariants и `55` fail-paths покрыты исполняемыми тестами;
- тесты детерминированы и изолированы;
- quality checks выполнены и результаты зафиксированы;
- нет скрытых spec/contract решений, внесенных через тесты;
- отсутствуют активные пункты из `Review Blockers For This Skill`.

## 11. Анти-Паттерны

`go-qa-tester` не должен:
- ограничиваться happy-path тестами при наличии обязательных fail/edge obligations;
- писать тесты без source traceability к `70`;
- закрывать spec-gaps ad hoc-логикой в тестах;
- оставлять timing/shared-state флакки без стабилизации или эскалации;
- заявлять readiness без фактического выполнения обязательных quality commands;
- размывать role boundaries в сторону spec-authoring или review-доменов.
