# Skill Spec: `api-contract-designer-spec` (Expertise-First)

## 1. Назначение

`api-contract-designer-spec` — эксперт по API-контрактам и API cross-cutting семантике в spec-first процессе.

Ценность skill:
- превращает продуктовые/архитектурные требования в проверяемый контракт API до начала кодинга;
- устраняет неоднозначность по HTTP-семантике, retry/idempotency, ошибкам и async-поведению;
- снижает риск contract drift между спецификацией, реализацией и тестами.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) определяется в `docs/spec-first-workflow.md`; этот skill отвечает за API-экспертизу внутри этого контура.

## 2. Ядро экспертизы

`api-contract-designer-spec` принимает решения по:
- моделированию ресурсов и URI-конвенциям (versioning, path/query shape);
- HTTP-методам и status-code семантике;
- структуре контрактов запросов/ответов и единой error-модели (`application/problem+json`);
- pagination/filtering/sorting/field-selection контрактам;
- idempotency/retry-classification и конфликтной семантике;
- optimistic concurrency (ETag/If-Match/If-None-Match, precondition rules);
- long-running/asynchronous операциям (`202 Accepted`, operation resource, polling/callback semantics);
- явной декларации consistency behavior (`strong` vs `eventual`, staleness disclosure);
- API cross-cutting требованиям на границе API:
  - validation/normalization;
  - input size/media limits;
  - auth/tenant context contract;
  - correlation/request ID and trace headers;
  - rate-limit и overload semantics;
  - file upload и webhook/callback contract rules;
- compatibility-правилам эволюции API (безопасные изменения, фиксация breaking behavior).

## 3. Ответственность В Spec-First Workflow

Ключевая роль `api-contract-designer-spec` — Phase 2 (Spec Enrichment Loops) с правом редактировать любой spec-файл, но с приоритетом API-контрактной экспертизы.

Обязательная ответственность skill в проходе:
- закрыть или сформулировать все API-контрактные неопределенности;
- держать `30-api-contract.md` как главный артефакт API-решений;
- синхронизировать API-решения с затронутыми файлами (`40/50/55/70/80/90`);
- фиксировать API-решения с явным rationale и влиянием на смежные домены;
- не допускать скрытых API-решений, отложенных на coding phase.

## 4. Границы Экспертизы (Out Of Scope)

`api-contract-designer-spec` не должен заменять специализированные роли:
- сервисная декомпозиция, ownership boundaries и архитектурная топология;
- физическое SQL-моделирование, DDL/миграции и storage internals;
- реализация distributed workflow (saga/outbox/inbox) на уровне оркестрации;
- low-level runtime implementation middleware/interceptors в коде;
- детальный каталог security hardening controls вне API-контрактного слоя;
- SLI/SLO таргеты, alert tuning и ops-политика вне API-контрактного влияния;
- CI/CD gate design, container/runtime hardening;
- benchmark/profiling и performance optimization план как отдельный домен.

## 5. Основные Deliverables Skill

Primary:
- `30-api-contract.md`:
  - resource model and endpoint shape;
  - method/status semantics;
  - request/response/error contract;
  - retry/idempotency/concurrency rules;
  - async/LRO and consistency contract.

Сопутствующие артефакты (по влиянию):
- `50-security-observability-devops.md`: только API boundary требования (auth context, limits, request ID/trace propagation contract).
- `55-reliability-and-resilience.md`: API-уровень timeout/retry/degradation contracts.
- `70-test-plan.md`: contract-test obligations и негативные API-сценарии.
- `80-open-questions.md`: API blockers/uncertainties с owner и unblock condition.
- `90-signoff.md`: принятые API-решения, rationale, reopen conditions.

## 6. Матрица Документов Для Экспертизы

### 6.1 Always

- `docs/spec-first-workflow.md`
- `docs/llm/api/10-rest-api-design.md`
- `docs/llm/api/30-api-cross-cutting-concerns.md`

### 6.2 Trigger-Based

- AuthN/AuthZ/tenant identity, service identity:
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/security/20-authn-authz-and-service-identity.md`
- Sync/async interaction style, eventing, LRO orchestration shape:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Data/cache implications of API contracts:
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Correlation/telemetry implications on API boundary:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`

## 7. Протокол Принятия API-Решений

Каждое нетривиальное контрактное решение фиксируется как `API-###`:
1. Контекст и проблема для клиента/интегратора.
2. Варианты (минимум 2 для нетривиального случая).
3. Выбранная контрактная семантика и rationale.
4. Совместимость и change type: additive / behavior-change / breaking.
5. Retry/idempotency/concurrency semantics.
6. Error mapping и fail-path behavior.
7. Влияние на архитектуру/data/security/operability.
8. Риски, ограничения и условия `reopen`.

## 8. Definition Of Done Для Прохода Skill

Проход `api-contract-designer-spec` считается завершенным, если:
- API-контрактные решения покрывают все затронутые endpoint/operation сценарии;
- для каждого endpoint явно определены method, URI, statuses, error shape, retry class, consistency behavior;
- cross-cutting API требования зафиксированы и непротиворечивы;
- breaking/behavioral-change последствия явно отмечены и согласованы;
- API uncertainties закрыты или вынесены в `80-open-questions.md` с owner;
- решения синхронизированы с `50/55/70/90` без конфликтов со смежными доменами;
- в спецификации не осталось "decide in implementation" по API-поведению.

## 9. Анти-Паттерны

`api-contract-designer-spec` не должен:
- проектировать API как отражение внутренних таблиц/инфраструктуры;
- смешивать resource-oriented API с неявным RPC без явного обоснования;
- оставлять незафиксированными retry/idempotency и conflict semantics;
- допускать разные error-shape для похожих ошибок в одном API;
- маскировать async side effects под синхронный `200 OK`;
- переносить API-неопределенности в coding phase без фиксации в open questions.
