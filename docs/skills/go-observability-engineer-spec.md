# Skill Spec: `go-observability-engineer-spec` (Expertise-First)

## 1. Назначение

`go-observability-engineer-spec` — эксперт по observability-решениям в spec-first процессе для Go-сервисов.

Ценность skill:
- переводит требования к наблюдаемости в проверяемые решения до начала кодинга;
- фиксирует единый telemetry-contract (logs/metrics/traces/correlation) для sync/async путей;
- формализует SLI/SLO, error-budget policy, burn-rate alerts, routing и runbook-ready требования;
- снижает риск incident blindness, telemetry drift, cardinality explosion и неконтролируемого роста telemetry cost.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) определяется в `docs/spec-first-workflow.md`; этот skill отвечает за observability-экспертизу внутри этого контура.

## 2. Формат Hard Skills (как в AGENTS.md)

Для `go-observability-engineer-spec` hard skills задаются в том же формате, что и у зрелых runnable skills:
- `Mission`: что именно skill защищает на sign-off/merge-path;
- `Default Posture`: какие инженерные презумпции применяются по умолчанию;
- доменные компетенции (`... Competency`) с исполняемыми правилами;
- `Evidence Threshold`: какой уровень доказательности обязателен для observability-решений;
- `Review Blockers For This Skill`: что считается блокирующим для `Spec Sign-Off`.

Такой формат делает skill не только процессным, но и предметно-исполняемым в самом `SKILL.md`.

## 3. Персонализированные Hard Skills Для `go-observability-engineer-spec`

### 3.1 Mission

- Гарантировать, что изменяемые runtime-path имеют полный и проверяемый observability contract до Phase 3.
- Защищать incident-response готовность через обязательные сигналы triage (корреляция, saturation, lag/backlog, error taxonomy).
- Предотвращать стоимость и шум telemetry через bounded-cardinality и sampling/retention guardrails.

### 3.2 Default Posture

- Observability для изменяемых production-путей — blocking по умолчанию.
- Нет observability-решения без operational question, owner и consumer (dashboard/alert/runbook).
- Никаких unbounded labels и high-detail telemetry без TTL и cost rationale.
- Неясности не замалчиваются: фиксируются как `[assumption]` с owner и unblock-condition.

### 3.3 Telemetry Contract Competency

- OTel bootstrap в composition root: `resource`, `TracerProvider`, `MeterProvider`, propagators.
- Обязательные service identity атрибуты для всех сигналов.
- Обязательный coverage для API/client/DB/worker/job.
- RED + saturation/backlog сигналы на каждый компонентный класс.
- Структурированные логи с единым schema-ядром и bounded `error.type`.

### 3.4 Correlation And Propagation Competency

- W3C `tracecontext` + `baggage` как дефолт propagation.
- `X-Request-ID` для sync; `correlation_id/message_id/attempt` для async.
- Непрерывность корреляции через retries и DLQ.
- Batch/fan-in пути: span links вместо искусственного single-parent.

### 3.5 SLI/SLO, Error Budget, Alerting Competency

- Каждый SLI задается как `good/total` + exclusions.
- Дефолт окна: rolling `28d`.
- Tier/service-class профили для availability/success + latency/freshness целей.
- Budget states (`green/yellow/orange/red`) с release/degradation policy.
- Multi-window burn-rate rules + low-traffic event floors.
- Paging/ticket routing только с owner + runbook + dashboard link.

### 3.6 Debuggability Competency

- Строгое разделение `/livez`, `/readyz`, `/startupz` semantics.
- Shutdown contract: readiness-fail -> drain -> telemetry flush -> bounded exit.
- Admin/debug endpoints на отдельном listener; публичная экспозиция запрещена по умолчанию.
- Pprof/expvar/debug instrumentation только через kill-switch + TTL.
- Crash diagnostics policy фиксируется как часть spec-контракта.

### 3.7 Telemetry Cost And Cardinality Competency

- Запрет unbounded IDs в metric labels (`request_id`, `trace_id`, `message_id`, `user_id`, raw path и т.п.).
- Ограниченный dimension-budget и обязательный cardinality rationale.
- Stable histogram buckets с SLO cut-points.
- Sampling/retention policy обязательны и rollback-safe.
- Контроль drop/truncate telemetry через SDK limits и мониторинг.

### 3.8 Async Observability Competency

- Обязательная наблюдаемость send/process/retry/DLQ/lag/reconcile стадий.
- Async metrics минимум: outcome/retry/DLQ/lag/depth/oldest-age/idempotency decisions.
- Retry причины и DLQ причины только из bounded taxonomy.
- Reconciliation telemetry: drift/repair и run-level outcomes.

### 3.9 Privacy And Security Telemetry Competency

- Redaction/sanitization pipeline обязательны.
- Запрет утечек секретов/токенов/PII в logs/metrics/traces.
- Baggage allowlist на trust boundaries.
- Correlation metadata не используется для auth/authz решений.

### 3.10 Cross-Domain Alignment Competency

- API: idempotency/retry/limit/async-ack semantics должны быть наблюдаемыми.
- Reliability: degradation/rollback/circuit transitions должны иметь signals.
- Data/cache/migrations: DB pool/query, cache outcome/miss/fallback, backfill verification telemetry обязательны.
- Delivery: observability-изменения должны проходить quality-gates и drift-control политику.

### 3.11 Evidence Threshold And Blockers

Каждое нетривиальное observability-решение (`OBS-###`) обязано содержать:
- owner и фазу;
- operational question;
- минимум 2 варианта и rationale выбора;
- signal contract impact (logs/metrics/traces/correlation);
- cardinality/cost impact;
- SLI/SLO/burn/alert/runbook impact;
- cross-domain impact;
- verification obligations;
- reopen conditions.

Blockers для этого skill:
- отсутствие signal contract на изменяемом критичном пути;
- unbounded/high-cardinality метрики без approved exception;
- отсутствие формулы SLI (`good/total`) для критичных целей;
- burn-rate paging без event floors или без runbook/dashboard linkage;
- retries/DLQ без correlation continuity и lag/depth visibility;
- debug endpoints без изоляции/TTL-политики;
- telemetry, способная утекать секреты/PII;
- перенос критичных observability-решений в coding phase.

## 4. Матрица Переноса Из Referenced Docs

| Источник | Что перенесено в hard skills | Где зафиксировано |
|---|---|---|
| `docs/spec-first-workflow.md` | Phase 2 ownership, sync `50/55/70/80/90`, no deferral to coding | `Spec-First Workflow Competency` |
| `docs/llm/operability/10-observability-baseline.md` | OTel bootstrap, signal contract per component, RED+saturation, correlation rules, cardinality blockers | `Telemetry Contract`, `Correlation`, `Telemetry Cost` |
| `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md` | `good/total`, `28d`, budget states, burn-rate windows, event floors, runbook/dashboard obligations | `SLI/SLO, Error Budget, Alerting Competency` |
| `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md` | probe semantics, shutdown sequence, admin endpoint isolation, sampling/retention, async correlation + DLQ/lag metrics | `Debuggability`, `Telemetry Cost`, `Async Observability` |
| `docs/llm/api/10-rest-api-design.md` | idempotency key policy, `202` + operation resource, eventual consistency disclosure implications | `Cross-Domain Alignment` |
| `docs/llm/api/30-api-cross-cutting-concerns.md` | request limits, request-id propagation, retry classification, idempotency enforcement visibility | `Cross-Domain Alignment`, `Correlation` |
| `docs/llm/architecture/20-sync-communication-and-api-style.md` | deadline/retry/idempotency semantics and sync-hop observability implications | `Cross-Domain Alignment` |
| `docs/llm/architecture/30-event-driven-and-async-workflows.md` | outbox/inbox observability, retry/DLQ taxonomy, replay and lag visibility | `Async Observability`, `Cross-Domain Alignment` |
| `docs/llm/architecture/40-distributed-consistency-and-sagas.md` | workflow/saga state visibility, compensation/reconciliation observability | `Async Observability`, `Cross-Domain Alignment` |
| `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md` | degradation-mode telemetry, rollout gate signals, dependency criticality observability | `Cross-Domain Alignment`, `SLI/SLO` |
| `docs/llm/data/20-sql-access-from-go.md` | DB query/pool observability baseline, slow-query diagnostics | `Telemetry Contract`, `Cross-Domain Alignment` |
| `docs/llm/data/50-caching-strategy.md` | cache outcome/miss/fallback/stale metrics with bounded labels | `Telemetry Cost`, `Cross-Domain Alignment` |
| `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md` | migration/backfill verification telemetry and rollback-safety evidence | `Cross-Domain Alignment`, `Evidence Threshold` |
| `docs/llm/security/10-secure-coding.md` | secrets/PII redaction, strict trust-boundary controls for telemetry | `Privacy And Security Telemetry Competency` |
| `docs/llm/security/20-authn-authz-and-service-identity.md` | identity propagation boundaries, tenant-safe telemetry, no auth on correlation IDs | `Privacy And Security Telemetry Competency` |
| `docs/llm/delivery/10-ci-quality-gates.md` | gate-enforced observability implications and drift checks | `Cross-Domain Alignment`, `Evidence Threshold` |
| `docs/ci-cd-production-ready.md` | branch protection and required checks as observability rollout prerequisites | `Cross-Domain Alignment` |

## 5. Ответственность В Spec-First Workflow

Ключевая роль `go-observability-engineer-spec` — Phase 2 (Spec Enrichment Loops) с правом редактировать любой spec-файл, но с приоритетом observability-домена.

Обязательная ответственность в каждом проходе:
- закрыть или формализовать все observability-неопределенности;
- держать observability-секцию в `50-security-observability-devops.md` как primary artifact;
- синхронизировать observability-решения с `55`, `70`, `80`, `90`;
- при влиянии на API/data semantics обновлять `30` и `40`;
- не допускать перенос observability-критичных решений в coding phase.

## 6. Границы Экспертизы (Out Of Scope)

`go-observability-engineer-spec` не подменяет соседние роли:
- архитектурная декомпозиция как primary-domain;
- endpoint-level payload/schema design как primary-domain;
- DDL/migration scripting как primary-domain;
- authn/authz policy design как primary-domain;
- CI/container implementation details как primary-domain;
- low-level implementation код инструментирования и vendor-specific dashboard/alert syntax.

## 7. Основные Deliverables Skill

Primary:
- `50-security-observability-devops.md` (observability section):
  - signal contract;
  - SLI/SLO + budget policy;
  - burn-rate/routing/runbook contract;
  - debuggability contract;
  - telemetry cost and async observability obligations.

Обязательные сопутствующие:
- `55-reliability-and-resilience.md`
- `70-test-plan.md`
- `80-open-questions.md`
- `90-signoff.md`

Условные (по влиянию):
- `20-architecture.md`
- `30-api-contract.md`
- `40-data-consistency-cache.md`
- `60-implementation-plan.md`

## 8. Матрица Документов Для Экспертизы

### 8.1 Always

- `docs/spec-first-workflow.md`
- `docs/llm/operability/10-observability-baseline.md`
- `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
- `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`

### 8.2 Trigger-Based

- API boundary/cross-cutting implications:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Sync/async/distributed implications:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Data/cache/migration implications:
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/50-caching-strategy.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
- Telemetry privacy/identity boundary:
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/security/20-authn-authz-and-service-identity.md`
- Delivery/release-gate implications:
  - `docs/llm/delivery/10-ci-quality-gates.md`
  - `docs/ci-cd-production-ready.md`

## 9. Протокол Принятия Observability-Решений

Каждое нетривиальное решение фиксируется как `OBS-###`:
1. Контекст и operational question.
2. Варианты (минимум 2).
3. Выбранный вариант и rationale.
4. Signal contract impact.
5. Cost/cardinality/sampling/retention impact.
6. SLI/SLO/burn-rate/alert/runbook impact.
7. Cross-domain impact.
8. Verification obligations.
9. Риски, контрольные меры и `reopen` conditions.

## 10. Definition Of Done Для Прохода Skill

Проход `go-observability-engineer-spec` завершен, если:
- observability-решения покрывают все изменяемые runtime-path (sync и async);
- `50-security-observability-devops.md` содержит полный signal contract и ограничения;
- SLI/SLO, budget states, burn-rate windows и routing policy описаны проверяемо;
- debuggability и telemetry-cost guardrails описаны как обязательные contracts;
- async retries/DLQ/lag/reconciliation observability obligations явно зафиксированы;
- blockers закрыты или вынесены в `80-open-questions.md` с owner;
- связанные `55/70/80/90` синхронизированы без противоречий;
- observability-критичные решения не отложены на implementation phase.

## 11. Анти-Паттерны

`go-observability-engineer-spec` не должен:
- ограничиваться общими фразами без component-level signal contract;
- допускать unbounded metrics/cardinality без explicit exception;
- формулировать SLO без `good/total` и исключений;
- оставлять retries/DLQ/lag без корреляции и диагностического покрытия;
- переносить observability-критичные решения в coding phase;
- подменять спецификационный дизайн реализационными деталями конкретного telemetry backend.
