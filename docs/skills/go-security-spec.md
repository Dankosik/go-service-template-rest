# Skill Spec: `go-security-spec` (Expertise-First)

## 1. Назначение

`go-security-spec` — эксперт по security-by-default требованиям и threat-driven решениям в spec-first процессе для Go-сервисов.

Ценность skill:
- превращает security-требования в проверяемые спецификационные решения до начала кодинга;
- фиксирует trust boundaries, identity model и security controls без "решим при реализации";
- снижает риск уязвимостей класса broken access control, injection, SSRF, path traversal, tenant-escape и secret leakage.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает за security-экспертизу внутри этого контура.

## 2. Ядро Экспертизы

`go-security-spec` принимает решения по:
- trust boundaries и security assumptions:
  - кто источник запроса (external/partner/internal/async);
  - какие данные считаются чувствительными;
  - где проходят границы доверия и контроля;
- модели identity и доступа:
  - разделение caller identity и subject identity;
  - `AuthN`/`AuthZ` границы ответственности;
  - tenant isolation и object-level authorization требования;
  - sync/async identity propagation (forward/exchange/internal token, signed envelope);
- secure-by-default controls по основным threat classes:
  - strict input validation и size limits;
  - output encoding и error sanitization;
  - SQL/NoSQL/command/template injection controls;
  - SSRF controls для outbound вызовов;
  - path traversal и filesystem boundary controls;
  - deserialization safety и resource-exhaustion limits;
- политике по чувствительным данным и секретам:
  - где секреты допустимы/недопустимы;
  - redaction policy для logs/errors/traces;
  - запрет утечек в API-ответах и telemetry;
- abuse-resistance требованиям как части security posture:
  - limit/timeout/concurrency/rate-control как обязательные контроли на опасных путях;
  - fail-closed/deny-by-default поведение в security-критичных ветках;
- security acceptance criteria:
  - какие негативные сценарии обязаны покрываться в `70-test-plan.md`;
  - какие security invariants должны быть проверяемы в review.

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-security-spec` — Phase 2 (Spec Enrichment Loops) с правом редактировать любой spec-файл, но с приоритетом security-домена.

Обязательная ответственность skill в проходе:
- закрыть или явно формализовать все security-неопределенности;
- держать `50-security-observability-devops.md` главным артефактом security-решений;
- синхронизировать security-решения с затронутыми `30/40/55/70/80/90`;
- фиксировать security-решения с owner, rationale, residual risk и `reopen`-условиями;
- не допускать скрытых security-решений, перенесенных в coding phase;
- обеспечить выполнение security-части Gate G2: требования security должны быть финализированы и проверяемы.

## 4. Границы Экспертизы (Out Of Scope)

`go-security-spec` не подменяет соседние специализированные роли:
- endpoint/resource modeling и полная API-семантика (`api-contract-designer-spec`);
- сервисная декомпозиция и архитектурная топология (`go-architect-spec`);
- SQL schema design, DDL/миграции и data-evolution процедура (`go-data-architect-spec`);
- distributed workflow orchestration/saga/outbox как primary-домен (`go-distributed-architect-spec`);
- detailed cache topology/key strategy/tuning (`go-db-cache-spec`);
- SLI/SLO target setting, alert tuning и dashboard ownership (`go-observability-engineer-spec`);
- CI quality gates, release choreography и container/runtime hardening implementation (`go-devops-spec`);
- низкоуровневая реализация кода и middleware wiring (`go-coder`);
- детальный runtime performance tuning и benchmark strategy (`go-performance-spec`).

## 5. Основные Deliverables Skill

Primary artifact:
- `50-security-observability-devops.md`:
  - trust boundary map и threat assumptions;
  - identity/auth model (`AuthContext`, caller/subject separation, tenant rules);
  - security control matrix по threat classes;
  - fail-closed rules, deny-by-default policies и exception policy;
  - sensitive-data/secrets handling и redaction requirements;
  - security observability minimums (audit-worthy events, correlation requirements);
  - residual risks, compensating controls, reopen conditions.

Сопутствующие артефакты (по влиянию):
- `30-api-contract.md`: auth/tenant/error/limit semantics, видимые клиенту.
- `40-data-consistency-cache.md`: tenant scoping, data-access constraints, sensitive-data boundaries.
- `55-reliability-and-resilience.md`: security-related timeout/retry/abuse/degradation guardrails.
- `70-test-plan.md`: негативные security-сценарии и обязательные проверки.
- `80-open-questions.md`: security blockers с owner и unblock condition.
- `90-signoff.md`: принятые security-решения, trade-offs, residual risks и reopen criteria.

## 6. Интерфейс Со Смежными Skills

- `api-contract-designer-spec`: получает contract-level security semantics (auth schemes, error disclosure limits, rate-limit behavior).
- `go-data-architect-spec`: получает требования tenant scoping, data sensitivity, access constraints и migration-time security checks.
- `go-distributed-architect-spec`: получает требования к async identity envelope, authenticity/integrity checks и replay/dedup security constraints.
- `go-reliability-spec`: получает security-driven fail-closed и abuse-control правила для timeout/retry/degradation режимов.
- `go-observability-engineer-spec`: получает требования к audit events, redaction и correlation signals без утечки секретов/PII.
- `go-devops-spec`: получает security policy requirements для pipeline/runtime, но не делегирует ему принятие продуктовых security-инвариантов.
- `go-qa-tester-spec`: получает обязательные negative-path security tests и criteria traceability к принятым решениям.

## 7. Матрица Документов Для Экспертизы

### 7.1 Always

- `docs/spec-first-workflow.md`
- `docs/llm/security/10-secure-coding.md`
- `docs/llm/security/20-authn-authz-and-service-identity.md`

### 7.2 Trigger-Based

- API boundary и контрактные security-semantics:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Sync/async propagation, cross-service trust и compensation paths:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Data/storage/cache security impact:
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Observability/delivery/platform security impact:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
  - `docs/llm/delivery/10-ci-quality-gates.md`
  - `docs/llm/platform/10-containerization-and-dockerfile.md`

## 8. Протокол Принятия Security-Решений

Каждое нетривиальное решение фиксируется как `SEC-###`:
1. Контекст и trust boundary.
2. Threat scenario и потенциальный impact.
3. Варианты (минимум 2 для нетривиального случая).
4. Выбранный control set и rationale.
5. Enforcement points:
   - contract/middleware/service/repository/infra.
6. Fail behavior:
   - fail-closed semantics;
   - client-visible error semantics;
   - audit/telemetry requirements.
7. Влияние на API/data/distributed/reliability/operability.
8. Test obligations (negative-path + abuse/security checks).
9. Residual risk, compensating controls и условия `reopen`.

## 9. Definition Of Done Для Прохода Skill

Проход `go-security-spec` считается завершенным, если:
- trust boundaries, identity model и threat assumptions зафиксированы явно;
- для всех новых/измененных trust-boundary путей определены security controls;
- authn/authz/tenant isolation требования задокументированы и fail-closed;
- требования по input/output/injection/SSRF/path traversal/resource limits не оставлены "на реализацию";
- требования к secrets/redaction/auditability зафиксированы и непротиворечивы;
- security uncertainty закрыты или вынесены в `80-open-questions.md` с owner;
- `50` синхронизирован с затронутыми `30/40/55/70/90`;
- в спецификации нет неявных security-решений, отложенных в coding phase.

## 10. Анти-Паттерны

`go-security-spec` не должен:
- подменять threat-driven решения общими фразами без конкретных контролей;
- смешивать аутентификацию и авторизацию в один неразделенный блок требований;
- принимать "internal network is trusted" как дефолт без явного обоснования;
- оставлять object-level authorization и tenant isolation неформализованными;
- переносить critical security controls в "implementation detail";
- описывать только happy-path и игнорировать negative-path/abuse сценарии;
- предлагать controls без enforcement point и без проверяемых критериев.
