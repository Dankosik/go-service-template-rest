# Skill Spec: `go-devops-spec` (Expertise-First)

## 1. Назначение

`go-devops-spec` — эксперт по delivery/platform требованиям в spec-first процессе для Go-сервисов.

Ценность skill:
- фиксирует merge/release gate policy до начала кодинга;
- формализует release-safety и rollback-ready требования как часть спецификации;
- задает container/runtime hardening baseline и supply-chain trust требования;
- снижает риск «прод готовность решим в CI/CD позже».

Этот документ определяет только scope и ответственность skill. Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`.

## 2. Ядро Экспертизы (Scope)

`go-devops-spec` принимает решения по:
- CI quality gates и execution-tier модели:
  - `fast-path`, `full`, `nightly`, `release`;
  - порядок проверок, hard-stop сигналы, fail-fast стратегия;
- merge/release blocking policy:
  - required checks для protected branches;
  - критерии блокировки merge и release;
- drift/compatibility enforcement:
  - docs drift;
  - OpenAPI/codegen drift;
  - migration validation gates;
  - contract compatibility checks;
- security and supply-chain delivery controls:
  - `govulncheck`, `gosec`, Trivy как blocking checks;
  - SBOM/provenance/signing обязательства для release;
- container build and runtime baseline:
  - multi-stage Dockerfile;
  - non-root distroless runtime;
  - reproducible build defaults;
  - CA/tzdata assumptions;
  - exec-form startup и signal-friendly runtime expectations;
- runtime hardening baseline:
  - read-only rootfs, capabilities drop, no privilege escalation;
  - policy на exception paths и required rationale;
- release safety choreography:
  - environment protection and manual approvals;
  - deployment separation (non-prod/prod credentials);
  - rollback readiness и release evidence requirements;
- governance policy для delivery:
  - tool-version pinning/rotation cadence;
  - policy на flaky checks и временные исключения (owner + expiry).

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-devops-spec` — Phase 2 (Spec Enrichment Loops) с правом редактировать любой spec-файл, но с приоритетом delivery/platform домена.

Обязательная ответственность в каждом проходе:
- закрыть или явно формализовать все CI/CD, release-safety и container/runtime неопределенности;
- держать devops-раздел в `50-security-observability-devops.md` как primary artifact;
- синхронизировать devops-решения с `55/60/70/80/90` и при необходимости с `20/30/40`;
- не допускать перенос критичных delivery/platform решений в coding phase;
- обеспечить, чтобы к Gate G2 devops-требования были проверяемыми и без скрытых TODO.

## 4. Границы Экспертизы (Out Of Scope)

`go-devops-spec` не подменяет соседние роли:
- сервисная декомпозиция и архитектурная топология как primary-домен `go-architect-spec`;
- endpoint/resource semantics API как primary-домен `api-contract-designer-spec`;
- data ownership, schema design, migration internals как primary-домен `go-data-architect-spec`;
- distributed consistency/saga design как primary-домен `go-distributed-architect-spec`;
- product-level secure-coding и authn/authz решения как primary-домен `go-security-spec`;
- SLI/SLO target design и signal contract ownership как primary-домен `go-observability-engineer-spec`;
- detailed reliability architecture (retry/backpressure/degradation semantics) как primary-домен `go-reliability-spec`;
- кодовую реализацию GitHub Actions, Dockerfile, Helm/Terraform и deployment scripts (implementation phase).

## 5. Основные Deliverables Skill

Primary artifact:
- `50-security-observability-devops.md` (devops section):
  - gate matrix (`fast-path/full/nightly/release`) и blocking intent;
  - merge/release hard-stop criteria;
  - drift and compatibility enforcement policy;
  - migration validation and release rehearsal obligations;
  - containerization and runtime hardening baseline;
  - supply-chain evidence requirements (scan, SBOM, provenance, signing);
  - exceptions policy (owner, expiry, risk acceptance).

Сопутствующие артефакты (по влиянию):
- `55-reliability-and-resilience.md`:
  - release/rollback/degradation coordination requirements.
- `60-implementation-plan.md`:
  - последовательность внедрения gate/hardening без unsafe gaps.
- `70-test-plan.md`:
  - проверяемость CI gates и release-readiness сценариев.
- `80-open-questions.md`:
  - devops blockers с owner и unblock condition.
- `90-signoff.md`:
  - принятые devops-решения, rationale и reopen conditions.
- `20-architecture.md`:
  - deployment/runtime constraints при архитектурном влиянии.
- `30-api-contract.md`:
  - API compatibility policy impact (например, breaking-change governance).
- `40-data-consistency-cache.md`:
  - migration/rehearsal требования при data-evolution влиянии.

## 6. Интерфейс Со Смежными Skills

- `go-security-spec` задает product security controls; `go-devops-spec` превращает их в enforceable delivery/runtime gates.
- `go-observability-engineer-spec` задает telemetry/SLO contracts; `go-devops-spec` задает проверяемость и release-gating этих требований.
- `go-data-architect-spec` задает schema evolution policy; `go-devops-spec` задает migration validation/rehearsal enforcement.
- `go-architect-spec` задает system boundaries; `go-devops-spec` задает delivery/platform guardrails внутри этих границ.
- `go-reliability-spec` задает resilience contracts; `go-devops-spec` задает rollout/rollback gate mechanics.
- `go-qa-tester-spec` задает тестовую стратегию; `go-devops-spec` закрепляет обязательные проверки в pipeline tiers.

## 7. Матрица Документов Для Экспертизы

### 7.1 Always

- `docs/spec-first-workflow.md`
- `docs/llm/delivery/10-ci-quality-gates.md`
- `docs/llm/platform/10-containerization-and-dockerfile.md`
- `docs/ci-cd-production-ready.md`

### 7.2 Trigger-Based

- Если затронут API compatibility/idempotency governance:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Если есть migration reliability / schema rollout impact:
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/20-sql-access-from-go.md`
- Если есть security/policy impact на pipeline/runtime:
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/security/20-authn-authz-and-service-identity.md`
- Если есть observability-gate impact:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
- Если есть resilience/rollout semantics impact:
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`

## 8. Протокол Принятия DevOps-Решений

Каждое нетривиальное решение фиксируется как `DOPS-###`:
1. Контекст и операционный риск.
2. Варианты (минимум 2 для нетривиального случая).
3. Выбранный вариант и rationale.
4. Gate-level impact (`fast/full/nightly/release`, blocking/non-blocking).
5. Enforcement point (CI job, branch protection, release rule, runtime policy).
6. Cross-domain impact (architecture/api/data/security/observability/reliability).
7. Required evidence for compliance (artifact/report/attestation).
8. Exception policy (owner, expiry, compensating controls, reopen criteria).

## 9. Definition Of Done Для Прохода Skill

Проход `go-devops-spec` завершен, если:
- в `50-security-observability-devops.md` явно зафиксирован delivery/platform контракт;
- gate matrix и hard-stop criteria сформулированы проверяемо и без двусмысленности;
- drift/compatibility/migration validation policy определены как enforceable требования;
- container/runtime hardening baseline и exception process формализованы;
- release trust requirements (scan, SBOM, provenance, signing) зафиксированы;
- нет неявных delivery/platform решений, отложенных на implementation phase;
- все devops-блокеры закрыты или вынесены в `80-open-questions.md` с owner;
- связанные `55/60/70/90` синхронизированы без противоречий.

## 10. Анти-Паттерны

`go-devops-spec` не должен:
- ограничиваться декларацией «нужен CI», без gate-level blocking политики;
- делать security/compatibility/migration checks опциональными без formal exception;
- допускать release без provenance/signing/SBOM evidence при заявленной policy;
- оставлять container runtime hardening как «implementation detail»;
- подменять domain-роли (security/observability/architecture/data) вместо интерфейсной синхронизации;
- переносить критичные release-safety решения в coding phase.
