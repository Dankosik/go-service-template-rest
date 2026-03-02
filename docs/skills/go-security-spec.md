# Skill Spec: `go-security-spec` (Domain Hard Skills)

## 1. Назначение

`go-security-spec` — экспертный spec-skill по security-first проектированию в Phase 2 (`Spec Enrichment Loops`) spec-first процесса.

Ценность skill:
- переводит security-требования в проверяемые решения до кодинга;
- фиксирует trust boundaries, identity/access модель и threat controls без «решим на реализации»;
- снижает риск broken access control, tenant escape, injection, SSRF, path traversal, replay abuse и secret leakage;
- делает security-решения enforceable через `SEC-###`, verification obligations и merge/release gates.

`docs/spec-first-workflow.md` задает фазовую и gate-дисциплину, а `go-security-spec` отвечает за предметную security-экспертизу внутри этого процесса.

## 2. Формат Hard Skills (как в AGENTS.md)

Для `go-security-spec` hard skills должны быть оформлены в инженерном формате, совпадающем по структуре с сильными инструкциями из `AGENTS.md` и зрелыми runnable-skills:
- `Mission`: какую зону риска skill обязан защищать до `Gate G2`;
- `Default Posture`: security-презумпции по умолчанию;
- domain-компетенции (`... Competency`) с исполняемыми правилами;
- `Evidence Threshold`: минимально достаточный уровень доказательности для security-решений;
- `Review Blockers For This Skill`: что блокирует sign-off.

Почему это критично:
- `Working Rules` задают процесс,
- но именно `Hard Skills` задают воспроизводимую глубину security-экспертизы и порог качества решений.

## 3. Персонализированные Hard Skills Для `go-security-spec`

### 3.1 Mission

- Преобразовывать security-intent в enforceable pre-coding решения по trust boundaries, identity/access и threat controls.
- Защищать `Gate G2` от скрытых security-решений и неявных trust assumptions.
- Гарантировать fail-closed, testable и incident-observable security posture для изменяемых потоков.

### 3.2 Default Posture

- Zero-trust по умолчанию: внешний и внутренний трафик считаются недоверенными, пока не доказано иное.
- `AuthN`, `AuthZ`, tenant isolation, data protection и abuse-resistance фиксируются как отдельные decision-блоки.
- Deny-by-default + least privilege: отсутствие policy трактуется как deny.
- Security-решение без enforcement-point и verification-path считается недействительным.
- Критичные неизвестные фиксируются как `[assumption]` с owner и unblock-condition.

### 3.3 Spec-First Workflow Competency

- Закрывать security-решения в Phase 2 до начала кодинга.
- Держать `50-security-observability-devops.md` primary-артефактом security-домена.
- Синхронизировать security-влияние в `30/40/55/70/80/90`.
- Считать незакрытые trust/identity/threat-control решения blocker-условием для `Gate G2`.
- Связывать каждое крупное решение с `SEC-###` и affected sections.

### 3.4 Trust Boundary And Threat Modeling Competency

- Для каждого затронутого потока фиксировать boundary class: `external`, `partner`, `internal`, `async`.
- Для каждой boundary фиксировать sensitivity class, side-effect profile, retry profile, abuse vectors.
- Для outbound путей фиксировать SSRF-aware egress policy (scheme/host/port/redirect constraints).
- Запрещать общие формулировки угроз без конкретного attacker path и impact.

### 3.5 Identity And Access Control Competency

- Вводить единый `AuthContext` с разделением caller identity и subject identity.
- Обязательные проверки AuthN: signature/issuer/audience/alg/lifetime (+ `typ`, если требуется профилем).
- Service-to-service identity: mTLS/workload identity без bypass проверок TLS trust chain.
- Модель enforcement по слоям:
  - middleware/interceptor: authentication + context build;
  - service/use-case: object-level authorization до side effects;
  - repository/data path: tenant scoping.
- Tenant scope обязателен и консистентен в DB/cache/async/audit.
- Политика propagation фиксируется явно (`forward_token`, `token_exchange`, `internal_token`).

### 3.6 Threat-Class Control Matrix Competency

- Обязательная матрица контролей по threat classes:
  - strict input validation + size limits;
  - output encoding + sanitized error disclosure;
  - SQL/NoSQL/command/template injection controls;
  - SSRF controls + redirect re-checks;
  - path traversal controls (`OpenInRoot`-class approach);
  - deserialization safety;
  - resource exhaustion controls.
- Запрещать boundary-validation дрейф (валидация только в бизнес-логике).
- `command execution` и `unsafe` — exception-only с явным review approval.

### 3.7 API Security Contract Competency

- В API-контракте явно фиксировать:
  - auth/error semantics (`401/403`, sanitized errors);
  - input-limit semantics (`413/414/431/415/422`);
  - rate-limit semantics (`429`, `Retry-After`);
  - retry classification + idempotency contract.
- Для retry-unsafe операций, которые клиент может ретраить, требовать `Idempotency-Key` policy (scope/TTL/conflict).
- Для long-running side effects требовать `202 + operation resource`, а не fake-sync success.
- Запрещать расхождение contract-level и runtime-level cross-cutting enforcement.

### 3.8 Async And Distributed Security Competency

- Async identity: signed envelope с минимально достаточными claims и bounded lifetime.
- Запрет raw bearer token propagation в async messages.
- Обязательные consumer-side проверки: authenticity/integrity/replay window/dedup/tenant scope.
- Для multi-step flows: security invariants и authorization semantics по шагам/compensation/failure paths.
- Outbox/inbox/idempotency обязательны для side-effecting async paths.

### 3.9 Data, Storage, Migration, And Cache Security Competency

- Service-owned DB boundary и отсутствие cross-service DB trust по умолчанию.
- Parameterized SQL + allowlisted dynamic identifiers.
- Least-privilege DB roles (runtime/migrator split), без утечки sensitive SQL details в telemetry.
- Migration security policy:
  - expand-migrate-contract;
  - mixed-version compatibility;
  - no cross-system dual writes;
  - explicit rollback limits.
- Cache security policy:
  - tenant/scope/version-safe keys;
  - no shared-cache secrets by default;
  - explicit fail-open/fail-closed domain decision.

### 3.10 Abuse-Resistance And Resilience Competency

- Abuse controls обязательны для дорогих/security-critical путей:
  - bounded timeouts;
  - bounded retries + jitter;
  - bounded queues/concurrency;
  - explicit rate/quota controls.
- Degradation/fallback режимы не должны обходить AuthZ/tenant invariants.
- Infinite timeout/retry/unbounded buffering трактуются как security blockers.

### 3.11 Security Observability And Privacy Competency

- Security-relevant events должны быть наблюдаемыми (auth failures, authorization denies, tenant violations, idempotency conflicts, abuse controls).
- Structured telemetry обязательна с correlation IDs и bounded taxonomy.
- Секреты/токены/PII запрещены в logs/metrics/traces/baggage.
- Cardinality discipline и redaction policy обязательны.
- Debug/pprof/admin endpoints должны быть изолированы и управляться TTL-based activation policy.

### 3.12 Delivery Gates And Runtime Hardening Competency

- Security-sensitive изменения обязаны проходить blocking gates:
  - contract/codegen/drift checks;
  - `govulncheck`/`gosec`;
  - container scan policy.
- Для container/runtime baseline обязательны:
  - non-root runtime;
  - minimal runtime image;
  - no embedded secrets;
  - корректный trust store;
  - запрет insecure TLS обходов.
- Security sign-off без gate-evidence path недействителен.

### 3.13 Verification And Evidence Threshold Competency

Каждое нетривиальное решение `SEC-###` обязано содержать:
1. контекст trust boundary и threat scenario;
2. минимум 2 варианта;
3. выбранный вариант + минимум один отвергнутый с явной причиной;
4. enforcement points (`contract/middleware/service/repository/infra`);
5. fail behavior (`fail_closed`, error semantics, audit obligations);
6. cross-domain impact;
7. verification obligations (`70-test-plan.md` + runtime evidence path);
8. residual risk и reopen criteria.

Решения без enforceability и verification не считаются sign-off ready.

### 3.14 Assumption And Uncertainty Discipline

- Критичные неизвестные немедленно маркировать как `[assumption]`.
- У каждого `[assumption]`: owner, validation path, deadline/condition.
- Неразрешенные critical assumptions переносить в `80-open-questions.md` как blockers.
- Запрещать формулировки вида «решим в coding phase» для security-критичных пунктов.

### 3.15 Review Blockers For This Skill

- Неявные/неполные trust boundaries или identity model для изменяемых критичных потоков.
- Отсутствие object-level authorization или tenant isolation enforcement points.
- Пропуски в threat-class controls для untrusted input.
- Retry-unsafe операция без idempotency/conflict semantics.
- Async path без authenticity/replay/dedup/tenant checks.
- Нет redaction/sanitization policy для sensitive data.
- Нет abuse controls (timeout/limit/concurrency/rate) на expensive paths.
- Security-runtime implications не отражены в delivery/platform gate obligations.
- Решения без `SEC-###`, без rejected-option rationale или без verification obligations.
- Критичная неопределенность отложена на coding phase.

## 4. Матрица Переноса Из Referenced Docs

| Источник | Что перенесено в hard skills | Где зафиксировано |
|---|---|---|
| `docs/spec-first-workflow.md` | Phase/Gate дисциплина, `Gate G2`, запрет скрытых security-решений в coding, cross-artifact synchronization | `Spec-First Workflow Competency`, `Mission`, `Review Blockers` |
| `docs/llm/security/10-secure-coding.md` | threat-class secure defaults: strict decode/limits, injection/SSRF/path/deserialization/resource controls, dangerous APIs, command/unsafe policy | `Threat-Class Control Matrix Competency`, `Default Posture` |
| `docs/llm/security/20-authn-authz-and-service-identity.md` | principal model, caller/subject split, JWT validation completeness, mTLS/service identity, object-level auth, tenant enforcement, propagation rules | `Identity And Access Control Competency` |
| `docs/llm/api/10-rest-api-design.md` | status/error semantics, idempotency policy, `202 + operation resource`, consistency disclosure, problem-details stability | `API Security Contract Competency` |
| `docs/llm/api/30-api-cross-cutting-concerns.md` | concern-to-layer mapping, boundary validation pipeline, input limits, idempotency/retry/rate-limit semantics, webhook/callback security defaults | `API Security Contract Competency`, `Threat-Class Control Matrix Competency`, `Verification And Evidence Threshold` |
| `docs/llm/architecture/20-sync-communication-and-api-style.md` | deadline/retry/idempotency rules, fail-fast behavior, error non-leak discipline for sync calls | `Abuse-Resistance And Resilience Competency`, `API Security Contract Competency` |
| `docs/llm/architecture/30-event-driven-and-async-workflows.md` | outbox/inbox/dedup defaults, bounded retries/DLQ policy, replay safety, async trace/correlation requirements | `Async And Distributed Security Competency`, `Security Observability And Privacy Competency` |
| `docs/llm/architecture/40-distributed-consistency-and-sagas.md` | explicit workflow invariants, step contracts, pivot/compensation policy, race controls, reconciliation obligations | `Async And Distributed Security Competency`, `Verification And Evidence Threshold` |
| `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md` | dependency criticality classes, timeout/retry budgets, bounded queues/bulkheads, fail-closed/fail-open rules, rollout safety gates | `Abuse-Resistance And Resilience Competency`, `Delivery Gates And Runtime Hardening Competency` |
| `docs/llm/data/10-sql-modeling-and-oltp.md` | service-owned data boundaries, tenant isolation model, invariant-centric constraints, multi-tenant pooled safety | `Data, Storage, Migration, And Cache Security Competency` |
| `docs/llm/data/20-sql-access-from-go.md` | parameterization, identifier allowlists, context/timeouts, least-privilege role separation, SQL injection and observability guardrails | `Data, Storage, Migration, And Cache Security Competency` |
| `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md` | expand-migrate-contract safety, no dual writes, verification gates, rollback limitations, backup/restore + PII deletion semantics | `Data, Storage, Migration, And Cache Security Competency`, `Verification And Evidence Threshold Competency` |
| `docs/llm/data/50-caching-strategy.md` | tenant-safe key design, secret/PII cache restrictions, stampede/fallback guardrails, observability and test obligations | `Data, Storage, Migration, And Cache Security Competency`, `Abuse-Resistance And Resilience Competency` |
| `docs/llm/operability/10-observability-baseline.md` | structured telemetry contract, correlation propagation, cardinality discipline, sensitive-data restrictions in telemetry | `Security Observability And Privacy Competency` |
| `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md` | debug endpoint isolation, shutdown/flush contracts, telemetry cost controls, async correlation/retry/DLQ observability | `Security Observability And Privacy Competency`, `Delivery Gates And Runtime Hardening Competency` |
| `docs/llm/delivery/10-ci-quality-gates.md` | blocking gate semantics, docs/codegen/migration drift controls, blocking security scans, hard-stop policy | `Delivery Gates And Runtime Hardening Competency`, `Review Blockers` |
| `docs/llm/platform/10-containerization-and-dockerfile.md` | runtime hardening baseline: non-root, minimal runtime, secret hygiene, TLS trust-store integrity, anti-pattern blockers | `Delivery Gates And Runtime Hardening Competency` |

## 5. Ответственность В Spec-First Workflow

`go-security-spec` в каждом проходе обязан:
- закрывать или формализовать все security-неопределенности;
- поддерживать `50-security-observability-devops.md` как source of truth по security;
- синхронизировать обязательные изменения в `80-open-questions.md` и `90-signoff.md`;
- синхронизировать `30/40/55/70` при влиянии на API/data/reliability/testing;
- не допускать переноса security-критичных решений в coding phase.

## 6. Границы Экспертизы (Out Of Scope)

`go-security-spec` не подменяет соседние primary-domain роли:
- endpoint/resource modeling как primary-domain (`api-contract-designer-spec`);
- architecture decomposition (`go-architect-spec`);
- physical SQL schema/migration mechanics (`go-data-architect-spec`);
- distributed orchestration как primary-domain (`go-distributed-architect-spec`);
- cache topology tuning как primary-domain (`go-db-cache-spec`);
- SLI/SLO tuning ownership (`go-observability-engineer-spec`);
- CI/runtime implementation mechanics (`go-devops-spec`);
- implementation coding (`go-coder`);
- perf tuning как primary-domain (`go-performance-spec`).

## 7. Deliverables

Минимальный набор deliverables в security-проходе:
- `50-security-observability-devops.md`:
  - trust boundaries and threat assumptions;
  - identity/AuthN/AuthZ/tenant requirements;
  - threat-class control matrix;
  - secrets/redaction policy;
  - abuse-resistance and fail-closed rules;
  - verification obligations;
  - residual risks and reopen criteria.
- `80-open-questions.md`: security blockers + owner + unblock condition.
- `90-signoff.md`: принятые `SEC-###` решения и reopen criteria.
- по влиянию: `30-api-contract.md`, `40-data-consistency-cache.md`, `55-reliability-and-resilience.md`, `70-test-plan.md`.

## 8. Матрица Документов Для Экспертизы

### 8.1 Always

- `docs/spec-first-workflow.md`
- `docs/llm/security/10-secure-coding.md`
- `docs/llm/security/20-authn-authz-and-service-identity.md`

### 8.2 Trigger-Based

- API boundary and contract semantics:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Sync/async/distributed workflow implications:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Data/storage/migration/cache implications:
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Observability/delivery/platform implications:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
  - `docs/llm/delivery/10-ci-quality-gates.md`
  - `docs/llm/platform/10-containerization-and-dockerfile.md`

## 9. Протокол Принятия Security-Решений

Каждое нетривиальное решение фиксируется как `SEC-###`:
1. контекст и trust boundary;
2. threat scenario и impact;
3. варианты (минимум 2);
4. выбранный вариант + минимум 1 отклоненный с причиной;
5. enforcement points (`contract/middleware/service/repository/infra`);
6. fail behavior (`fail_closed`, error semantics, audit obligations);
7. cross-domain impact;
8. verification obligations;
9. residual risk и reopen criteria.

## 10. Definition Of Done Для Прохода Skill

Проход `go-security-spec` завершен, если:
- trust boundaries, identity/access model и threat assumptions явно зафиксированы;
- все изменяемые boundary paths имеют threat-class controls и enforcement points;
- `AuthN/AuthZ/tenant` требования fail-closed и непротиворечивы;
- секреты/redaction/error disclosure/telemetry privacy требования формализованы;
- negative/abuse-path test obligations синхронизированы с `70-test-plan.md`;
- blockers закрыты или вынесены в `80-open-questions.md` с owner;
- связанные `30/40/55/70/80/90` синхронизированы без противоречий;
- нет активных пунктов из `Review Blockers For This Skill`;
- нет security-решений, отложенных в coding phase.

## 11. Анти-Паттерны

`go-security-spec` не должен:
- заменять threat-driven дизайн общими фразами без control matrix;
- смешивать AuthN и AuthZ в один неразделенный блок;
- принимать internal trust как default без explicit justification;
- оставлять object-level authorization и tenant isolation «на реализацию»;
- описывать только happy-path и игнорировать negative/abuse-path;
- предлагать controls без enforcement point и без verification obligations;
- переносить security-критичные решения в coding phase.
