# Skill Spec: `go-observability-engineer-spec` (Expertise-First)

## 1. Назначение

`go-observability-engineer-spec` — эксперт по observability-решениям в spec-first процессе для Go-сервисов.

Ценность skill:
- переводит требования к наблюдаемости в проверяемые спецификационные решения до начала кодинга;
- фиксирует единый контракт telemetry (logs/metrics/traces + correlation) для sync/async путей;
- устраняет неоднозначность по SLI/SLO, error budget policy, alerting и runbook-ready операционным требованиям;
- снижает риск incident blindness, telemetry drift и неконтролируемого роста стоимости telemetry.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) определяется в `docs/spec-first-workflow.md`; этот skill отвечает за observability-экспертизу внутри этого контура.

## 2. Ядро Экспертизы

`go-observability-engineer-spec` принимает решения по:
- telemetry contract на уровне спецификации:
  - обязательные поля structured logs;
  - обязательные метрики (RED + saturation + async lag/backlog);
  - trace propagation и correlation across boundaries;
- observability baseline по компонентам:
  - API handlers;
  - outbound clients;
  - DB access;
  - workers/consumers/producers;
  - scheduled/reconciliation jobs;
- SLI/SLO и error budget policy:
  - формулы `good/total`, exclusions, 28d windows;
  - burn-rate multi-window alerts;
  - paging vs ticket routing;
  - release/degradation decisions, связанные с budget state;
- debuggability contract:
  - `/livez`, `/readyz`, `/startupz` semantics;
  - admin/debug endpoint isolation;
  - crash diagnostics и graceful telemetry flush требования;
- telemetry cost and safety guardrails:
  - cardinality budgets и dimension constraints;
  - sampling defaults и incident burst-mode с TTL;
  - retention policy и redaction/sanitization requirements;
- async observability correctness:
  - correlation continuity через retries/DLQ;
  - lag/backlog visibility;
  - reconciliation observability obligations;
- observability-driven acceptance obligations для implementation и review phases.

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-observability-engineer-spec` — Phase 2 (Spec Enrichment Loops) с правом редактировать любой spec-файл, но с приоритетом observability-домена.

Обязательная ответственность в каждом проходе:
- закрыть или формализовать все observability-неопределенности и риски;
- держать observability-секцию в `50-security-observability-devops.md` как primary artifact;
- синхронизировать observability-решения с `55-reliability-and-resilience.md` (burn/degradation/shutdown contracts), `70-test-plan.md` (telemetry verification obligations), `80-open-questions.md`, `90-signoff.md`;
- при влиянии на API или data semantics синхронизировать `30-api-contract.md` и `40-data-consistency-cache.md`;
- не допускать перенос критичных observability-решений в coding phase.

## 4. Границы Экспертизы (Out Of Scope)

`go-observability-engineer-spec` не подменяет соседние роли:
- сервисная декомпозиция, ownership boundaries и архитектурная топология как primary-домен;
- endpoint-level API contract design (resource semantics, payload/schema design);
- SQL schema design, DDL, migration rollout internals и datastore class choice;
- secure coding controls за пределами telemetry/privacy boundary (authn/authz, threat controls);
- CI/CD pipeline design, image build policy и runtime hardening как primary-домен;
- detailed reliability architecture (retry/backpressure/degradation design) как самостоятельный домен;
- implementation-level код инструментирования, dashboards-as-code и alert rule syntax в конкретной платформе;
- performance optimization и profiling strategy как отдельный домен.

## 5. Основные Deliverables Skill

Primary:
- `50-security-observability-devops.md` (observability section):
  - signal contract (logs/metrics/traces/correlation);
  - SLI/SLO profile и budget-state decisions;
  - alert routing policy и runbook expectations;
  - debugability и telemetry-cost guardrails;
  - async observability obligations.

Сопутствующие артефакты (по влиянию):
- `55-reliability-and-resilience.md`: observability hooks для degradation/rollback/shutdown решений.
- `70-test-plan.md`: test obligations для telemetry correctness, propagation, cardinality guardrails, burn-rate alertability.
- `80-open-questions.md`: observability blockers с owner и unblock condition.
- `90-signoff.md`: принятые observability-решения, rationale и reopen conditions.
- `30-api-contract.md`: API boundary correlation/idempotency/timeout observability implications.
- `40-data-consistency-cache.md`: DB/query/cache/async-path observability implications.

## 6. Матрица Документов Для Экспертизы

### 6.1 Always

- `docs/spec-first-workflow.md`
- `docs/llm/operability/10-observability-baseline.md`
- `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
- `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`

### 6.2 Trigger-Based

- Если меняется API boundary contract (request ID, trace headers, idempotency/retry semantics):
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Если есть sync/async architecture implications:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Если есть data/cache observability implications:
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/50-caching-strategy.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
- Если есть privacy/security constraints для telemetry:
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/security/20-authn-authz-and-service-identity.md`
- Если нужно выровнять observability-requirements с release quality gates:
  - `docs/llm/delivery/10-ci-quality-gates.md`
  - `docs/ci-cd-production-ready.md`

## 7. Протокол Принятия Observability-Решений

Каждое нетривиальное решение фиксируется как `OBS-###`:
1. Контекст и операционный вопрос (какой incident question должен решаться).
2. Варианты (минимум 2 для нетривиального случая).
3. Выбранный вариант и rationale.
4. Signal contract impact:
   - logs fields;
   - metrics + label/cardinality shape;
   - traces/propagation semantics.
5. Cost and safety impact:
   - sampling;
   - retention;
   - privacy/redaction;
   - cardinality budget.
6. Operational usage:
   - SLI/SLO and burn-rate impact;
   - alert routing;
   - runbook/dashboard dependency.
7. Cross-domain impact на API/data/security/reliability/delivery.
8. Риски, контрольные меры и условия `reopen`.

## 8. Definition Of Done Для Прохода Skill

Проход `go-observability-engineer-spec` завершен, если:
- observability-решения в спецификации покрывают все изменяемые runtime path (sync и async);
- в `50-security-observability-devops.md` явно зафиксирован signal contract и его ограничения;
- SLI/SLO, burn-rate, paging/ticket и budget-state decisions сформулированы проверяемо;
- debugability и telemetry-cost guardrails описаны как обязательные operational contracts;
- нет неявных observability-решений, отложенных на implementation phase;
- observability blockers закрыты или вынесены в `80-open-questions.md` с owner;
- связанные `55/70/90` синхронизированы и не противоречат `50`.

## 9. Анти-Паттерны

`go-observability-engineer-spec` не должен:
- ограничиваться общими фразами без конкретного signal contract;
- дублировать security/devops/reliability решения без observability-rationale;
- допускать unbounded metric labels и нефиксированные telemetry cost assumptions;
- оставлять retry/DLQ/lag сценарии без correlation и диагностического покрытия;
- переносить observability-критичные решения в coding phase без фиксации в `80-open-questions.md`;
- подменять спецификационный дизайн реализационными деталями конкретного telemetry backend.
