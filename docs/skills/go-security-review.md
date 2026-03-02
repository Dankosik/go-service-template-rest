# Skill Spec: `go-security-review` (Domain Hard Skills)

## 1. Назначение

`go-security-review` — экспертный review-skill по security-корректности Go-кода в Phase 4 (`Domain-Scoped Code Review`) spec-first workflow.

Ценность skill:
- обнаруживает exploit-ready уязвимости и security-регрессии до merge;
- проверяет, что реализация соблюдает утвержденный security intent из spec-пакета;
- формирует actionable findings с `file:line`, реальным impact и минимальным безопасным fix-path;
- повышает воспроизводимость качества security review между задачами с похожим risk-profile.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает только за security-domain review в рамках Phase 4.

## 2. Формат Hard Skills (как в AGENTS.md)

Для `go-security-review` hard skills фиксируются в формате, совместимом со стилем `AGENTS.md` и уже усиленных skill-пакетов:
- `Mission`: что именно skill защищает на merge-path;
- `Default Posture`: инженерные презумпции по умолчанию;
- доменные компетенции (`... Competency`) с операциональными критериями проверки;
- `Evidence Threshold`: обязательный уровень доказательности finding-ов;
- `Review Blockers For This Skill`: что является merge-blocking в security-domain.

Такой формат делает skill не только процессным, но и автономным носителем предметной экспертизы.

## 3. Персонализированные Hard Skills Для `go-security-review`

### 3.1 Mission

- Блокировать merge небезопасных изменений, которые нарушают доверенные границы, authz-модель и базовые secure-by-default контракты.
- Переводить security-риски из «общих рекомендаций» в проверяемые findings с эксплуатационным контекстом.
- Удерживать review строго в security-domain без смешения ownership соседних review-ролей.

### 3.2 Default Posture

- Любой input (`HTTP`, `queue`, `webhook`, `file`, `downstream data`) считается untrusted до явной валидации.
- Internal traffic не считается trusted by default.
- Для security-critical путей действует fail-closed и deny-by-default.
- Отсутствие лимитов/таймаутов/tenant scope трактуется как дефект, пока не доказано обратное.
- Любое отклонение от baseline требует явного spec reference или `Spec Reopen`.

### 3.3 Spec-First Review Competency

- Соблюдать Phase 4 ограничения из `docs/spec-first-workflow.md`:
  - findings только в security-domain;
  - точные `file:line`;
  - practical fix path;
  - `Spec Reopen` при конфликте с утвержденным intent.
- Открытые `critical/high` findings считаются blocker для `Gate G4`.
- Комментарии review не меняют approved security-contract неявно.

### 3.4 Trust Boundary And Input Validation Competency

- Проверять boundary-first validation до side effects.
- Требовать strict JSON discipline на mutable endpoints:
  - `http.MaxBytesReader` до decode;
  - `DisallowUnknownFields()`;
  - reject trailing tokens.
- Проверять allowlist-подход для enum/range/format/filter/sort/state-transition.
- Блокировать blacklist-only validation и позднюю валидацию после начала side effects.
- Проверять transport/input limits: header/URI/body/multipart/filter complexity.

### 3.5 AuthN/AuthZ/Tenant Isolation Competency

- Жестко разделять AuthN findings и AuthZ findings.
- Проверять полноту AuthN (signature/iss/aud/lifetime/alg и доверенная key chain).
- Проверять object-level authorization в resource-by-ID путях.
- Проверять caller/subject separation для mixed identity flows.
- Проверять tenant scope end-to-end: service -> repository -> cache -> async.
- Рассматривать default-allow, implicit superuser и tenant mismatch как high-risk.

### 3.6 Injection, Query, Template, Command Competency

- Проверять parameterized SQL/NoSQL доступ и allowlist dynamic identifiers.
- Запрещать прямое использование raw client JSON как query/filter DSL.
- Блокировать shell-based command execution (`sh -c`, `bash -c`, `cmd /c`) с user influence.
- Проверять safe templating (`html/template` для HTML).
- Любая user-influenced строковая сборка query/path/command без allowlist — finding.

### 3.7 Outbound Security And SSRF Competency

- Проверять явные outbound deadlines/timeouts и context propagation.
- Для user-influenced targets требовать SSRF policy:
  - allowlist scheme/host/port;
  - DNS-resolved block private/loopback/link-local/multicast;
  - redirect re-check.
- Блокировать security-sensitive use of `http.Get`/`http.DefaultClient`.
- Проверять, что code controls и network egress controls не противоречат друг другу.

### 3.8 Filesystem, Path, And Upload Competency

- Проверять root-constrained file access (`os.OpenInRoot`/equivalent boundary).
- Запрещать trust client filename/path как storage key.
- Проверять upload pipeline:
  - size limit before parse;
  - streaming over full-memory read;
  - extension allowlist + content sniffing;
  - storage outside webroot.
- Для malware/content-scan flows требовать publish-after-scan semantics.

### 3.9 Secrets, Error Disclosure, And Debug Surface Competency

- Проверять sanitized client errors (без stack traces/SQL/internal topology).
- Проверять redaction policy для logs/metrics/traces (no secrets/tokens/DSN/raw auth headers/PII dump).
- Проверять, что correlation IDs используются только для observability, не для authz решений.
- Проверять isolate-by-default админ/дебаг поверхность (`pprof`, `expvar`, debug handlers).
- Утечка чувствительных данных в telemetry считается security finding.

### 3.10 Abuse Resistance And Failure Semantics Competency

- Проверять bounded resources:
  - timeout budgets;
  - bounded concurrency/queues;
  - rate limits/quota semantics;
  - retry budgets.
- Проверять retry classification и idempotency alignment.
- Проверять корректное overload mapping (`429` vs `503`) и safe caller behavior.
- В security-critical dependencies проверять отсутствие unsafe fail-open.
- Блокировать unbounded fan-out/retries/buffering на protected flows.

### 3.11 Async Identity And Distributed Security Competency

- Проверять отсутствие raw bearer token в async payloads.
- Проверять integrity/authenticity envelope для async identity.
- Проверять dedup/idempotency и durable ack ordering (`ack after durable state`).
- Проверять correlation continuity across retry/DLQ.
- Фиксировать dual-write и inconsistent outbox/inbox практики как security-impacting risks.

### 3.12 Data/Cache/Migration Security Competency

- Проверять least-privilege DB roles и query logging without sensitive interpolation.
- Проверять tenant-safe cache key schema (`tenant + scope + version` where required).
- Запрещать shared-cache reuse для private/per-user responses.
- Проверять migration/backfill влияние на tenant boundaries, PII lifecycle, deletion guarantees.
- Проверять rollback realism и неотложенные security последствия destructive migration steps.

### 3.13 Delivery And Runtime Hardening Competency

- Проверять merge-gate evidence path для security-sensitive changes:
  - `go test ./...`;
  - `go test -race ./...` (when concurrency-sensitive);
  - `go vet ./...`;
  - `govulncheck ./...`;
  - `gosec ./...`;
  - container scan / runtime hardening checks where impacted.
- Проверять container hardening baseline (non-root, minimal base, no TLS verification bypass).
- Ослабление blocking security gates без expiry/owner/rationale — blocker.

### 3.14 Security Test Traceability Competency

- Любой существенный finding должен трассироваться в `70-test-plan.md` как negative-path obligation.
- Для changed critical paths ожидать тесты по категориям (если применимо):
  - wrong tenant / object ownership;
  - insufficient role/scope;
  - forged/invalid token or signature;
  - malformed/oversized payload;
  - idempotency conflict/replay;
  - SSRF/path traversal/injection attempts.
- Отсутствие security-negative coverage для high-risk change должно фиксироваться как finding или residual risk.

### 3.15 Evidence Threshold And Severity Calibration

Каждая нетривиальная security-находка обязана содержать:
- точный `file:line`;
- `Axis` и нарушенное правило/контракт;
- реалистичные attacker preconditions;
- affected trust boundary/data asset;
- минимально безопасный corrective action;
- ссылку на spec source.

Severity:
- `critical`: подтвержденная exploitable high-impact vulnerability;
- `high`: сильное доказательство серьезного security-contract breach;
- `medium`: ограниченная, но значимая уязвимость/слабость;
- `low`: локальный hardening gap.

### 3.16 Assumption And Uncertainty Discipline

- Неизвестные критичные факты маркируются как `[assumption]`.
- Отсутствие spec-артефактов маркируется как `[assumption: missing-spec-artifacts]`.
- Любая unresolved assumption с merge-impact идет в `Residual Risks` или `Spec Reopen`.
- Неопределенность не маскируется общими формулировками.

### 3.17 Review Blockers For This Skill

- Нет trust-boundary validation/strict parsing/size limits на untrusted input.
- Broken/missing AuthN/AuthZ/tenant/object checks на измененных путях.
- Exploitable injection/SSRF/path traversal/unsafe upload patterns.
- Secret/PII leakage в responses/errors/logs/traces/metrics/debug endpoints.
- Unbounded abuse vectors (timeout/retry/concurrency/queue/memory) на sensitive operations.
- Async replay/duplication/ack-order defects с security impact.
- Ослабленные security gates без формального approved exception.
- Security fix конфликтует с approved spec intent и не эскалирован через `Spec Reopen`.

## 4. Матрица Переноса Из Referenced Docs

| Источник | Что перенесено в hard skills | Где зафиксировано |
|---|---|---|
| `docs/spec-first-workflow.md` | Phase 4 domain boundaries, Gate G4 blocking logic, findings protocol, Spec Reopen discipline | `Spec-First Review Competency`, `Evidence Threshold`, `Review Blockers` |
| `docs/llm/go-instructions/70-go-review-checklist.md` | Review posture: correctness-first, actionable evidence, validation command baseline | `Default Posture`, `Evidence Threshold`, `Delivery And Runtime Hardening Competency` |
| `docs/llm/security/10-secure-coding.md` | strict input/output/injection/SSRF/path/upload/command/unsafe defaults and merge-gate criteria | `Trust Boundary`, `Injection`, `Outbound SSRF`, `Filesystem/Upload`, `Abuse Resistance`, `Review Blockers` |
| `docs/llm/security/20-authn-authz-and-service-identity.md` | AuthContext model, deny-by-default, JWT/mTLS checks, tenant isolation, sync/async identity propagation | `AuthN/AuthZ/Tenant`, `Async Identity`, `Default Posture`, `Review Blockers` |
| `docs/llm/api/10-rest-api-design.md` | retry/idempotency semantics, status/error model, async `202` contract, consistency disclosure | `Abuse Resistance`, `Evidence Threshold`, `Security Test Traceability` |
| `docs/llm/api/30-api-cross-cutting-concerns.md` | validation pipeline order, input limits, identity/tenant context trust rules, rate limit semantics, middleware baseline | `Trust Boundary`, `AuthN/AuthZ/Tenant`, `Abuse Resistance`, `Outbound/Upload` |
| `docs/llm/data/10-sql-modeling-and-oltp.md` | tenant isolation in pooled models, service-owned schema boundaries, constraint-first data integrity | `Data/Cache/Migration Security Competency` |
| `docs/llm/data/20-sql-access-from-go.md` | parameterization, allowlist dynamic identifiers, least-privilege DB roles, query observability without secret leaks | `Injection And Query`, `Data/Cache/Migration`, `Secrets/Telemetry` |
| `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md` | zero-downtime compatibility, outbox/no dual write, rollback realism, PII deletion and retention constraints | `Async Identity`, `Data/Cache/Migration`, `Review Blockers` |
| `docs/llm/data/50-caching-strategy.md` | tenant-safe keys, no secret caching, fail-open vs fail-closed boundary, stampede/timeout/fallback controls | `Data/Cache/Migration`, `Abuse Resistance`, `Default Posture` |
| `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md` | dependency criticality (`fail_closed` vs degraded/open), timeout/retry budgets, overload/backpressure controls | `Abuse Resistance`, `Default Posture`, `Review Blockers` |
| `docs/llm/go-instructions/10-go-errors-and-context.md` | context deadlines/cancel correctness, no sensitive leakage at API boundaries, cancellation semantics | `Outbound SSRF`, `Abuse Resistance`, `Secrets/Error Disclosure` |
| `docs/llm/architecture/20-sync-communication-and-api-style.md` | explicit sync deadlines/retries/idempotency, no infinite timeout calls, deterministic error mapping | `Abuse Resistance`, `Evidence Threshold` |
| `docs/llm/architecture/30-event-driven-and-async-workflows.md` | outbox/inbox defaults, bounded retries, DLQ policies, async idempotency and observability | `Async Identity And Distributed Security Competency`, `Review Blockers` |
| `docs/llm/architecture/40-distributed-consistency-and-sagas.md` | invariant ownership, saga step contracts, dedup/commit ordering, no hidden dual writes | `Async Identity`, `Data/Migration`, `Review Blockers` |
| `docs/llm/go-instructions/20-go-concurrency.md` | bounded goroutines/lifetime/cancel paths, race/leak risk as abuse vector | `Abuse Resistance Competency`, `Delivery/Validation obligations` |
| `docs/llm/go-instructions/40-go-testing-and-quality.md` | deterministic negative tests, race validation, quality-tool discipline | `Security Test Traceability`, `Delivery And Runtime Hardening` |
| `docs/build-test-and-development-commands.md` | repo-native verification command path and security check integration in CI-like flow | `Delivery And Runtime Hardening Competency` |
| `docs/llm/operability/10-observability-baseline.md` | structured redacted logging, correlation rules, bounded metric cardinality | `Secrets, Error Disclosure, And Telemetry Competency` |
| `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md` | debug endpoint isolation, crash/pprof controls, async correlation continuity, no telemetry secret leakage | `Secrets/Telemetry`, `Async Identity`, `Review Blockers` |
| `docs/llm/delivery/10-ci-quality-gates.md` | hard-stop gate policy for security scans/drift/contract checks, non-bypassable merge controls | `Delivery And Runtime Hardening`, `Review Blockers` |
| `docs/llm/platform/10-containerization-and-dockerfile.md` | non-root minimal runtime, TLS trust requirements, container hardening anti-patterns | `Delivery And Runtime Hardening Competency` |

## 5. Ответственность В Spec-First Workflow

Ключевая роль `go-security-review`:
- domain-scoped security review в Phase 4;
- проверка соответствия реализации approved security intent;
- оформление findings в workflow-формате без architectural redesign;
- явная эскалация spec-intent конфликтов через `Spec Reopen`.

Обязательная ответственность в каждом проходе:
- фиксировать только evidence-backed security findings;
- давать concrete attacker-centric impact;
- давать minimal safe fix;
- поддерживать traceability к `50/70/90` и, при необходимости, `30/40/55`.

## 6. Границы Экспертизы (Out Of Scope)

`go-security-review` не подменяет:
- idiomatic/style ownership (`go-idiomatic-review`);
- architecture-integrity ownership (`go-design-review`);
- deep performance/concurrency/DB/reliability ownership как primary-domain;
- общий test-strategy ownership (`go-qa-review`);
- spec editing в review-фазе.

Skill может фиксировать cross-domain signal, но обязан сделать handoff профильной роли.

## 7. Deliverables

Основной deliverable в `reviews/<feature-id>/code-review-log.md`:

```text
[severity] [go-security-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

Минимальные секции итогового ответа:
- `Findings`
- `Handoffs`
- `Spec Reopen`
- `Residual Risks`

## 8. Матрица Документов Для Экспертизы

### 8.1 Always

- `docs/spec-first-workflow.md`
- `docs/llm/go-instructions/70-go-review-checklist.md`
- `docs/llm/security/10-secure-coding.md`
- `docs/llm/security/20-authn-authz-and-service-identity.md`
- `specs/<feature-id>/50-security-observability-devops.md`
- `specs/<feature-id>/90-signoff.md`
- `reviews/<feature-id>/code-review-log.md` (если есть)

### 8.2 Trigger-Based

- API security semantics:
  - `specs/<feature-id>/30-api-contract.md`
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Data/cache security implications:
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Reliability/failure semantics with security impact:
  - `specs/<feature-id>/55-reliability-and-resilience.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
  - `docs/llm/go-instructions/10-go-errors-and-context.md`
- Sync/async trust propagation:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Concurrency-sensitive security controls:
  - `docs/llm/go-instructions/20-go-concurrency.md`
- Verification obligations:
  - `specs/<feature-id>/70-test-plan.md`
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`
- Observability/debug/redaction implications:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
- Delivery/platform hardening implications:
  - `docs/llm/delivery/10-ci-quality-gates.md`
  - `docs/llm/platform/10-containerization-and-dockerfile.md`

## 9. Протокол Фиксации Findings

Каждый нетривиальный finding проходит через минимальный протокол:
1. `Где`: точный `file:line`.
2. `Axis`: один из security review axes.
3. `Issue`: что нарушено в контроле/контракте.
4. `Impact`: attacker preconditions, blast radius, affected boundary/asset.
5. `Suggested fix`: smallest safe correction.
6. `Spec reference`: явная ссылка на approved obligation.
7. `Escalation`: нужен ли `Spec Reopen`.

## 10. Definition Of Done Для Прохода Skill

Проход `go-security-review` завершен, если:
- findings строго в security-domain;
- все findings evidence-backed и с `file:line`;
- все `critical/high` issues явно закрыты или эскалированы;
- нет неявных spec-intent конфликтов;
- при отсутствии находок явно сказано `No security findings.` и указаны `Residual Risks`.

## 11. Анти-Паттерны

`go-security-review` не должен:
- превращаться в общий review без threat-ориентированной аргументации;
- смешивать AuthN/AuthZ в один неразличимый комментарий;
- давать абстрактные советы без exploit-impact и fix path;
- пропускать fail-open/fail-closed анализ для критичных зависимостей;
- замалчивать неопределенности вместо явных `[assumption]`;
- менять approved requirements без формального `Spec Reopen`.
