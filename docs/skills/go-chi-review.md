# Skill Spec: `go-chi-review` (Domain-Scoped Review)

## 1. Назначение

`go-chi-review` — экспертный review-skill по корректности transport-routing поведения Go-кода на базе `github.com/go-chi/chi/v5` в Phase 4 (`Domain-Scoped Code Review`) spec-first workflow.

Ценность skill:
- находит дефекты и регрессии, специфичные для `chi`-роутинга и middleware-цепочек;
- подтверждает, что реализация соответствует approved routing intent из spec-пакета;
- предотвращает silent route override/shadowing и observability drift;
- дает actionable findings с `file:line`, impact и минимальным safe fix path.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает только за `chi`-routing review в рамках Phase 4.

## 2. Формат Hard Skills (как в AGENTS.md)

Для `go-chi-review` hard skills фиксируются в формате:
- `Mission`: какой merge-risk skill блокирует;
- `Default Posture`: инженерные презумпции по умолчанию;
- доменные компетенции (`... Competency`) с операциональными критериями;
- `Evidence Threshold`: обязательная доказательность findings;
- `Review Blockers For This Skill`: что блокирует `Gate G4`.

## 3. Персонализированные Hard Skills Для `go-chi-review`

### 3.1 Mission

- Защищать merge от routing/middleware регрессий при использовании `chi`.
- Делать `chi`-специфику явной: behavior на `404/405/OPTIONS`, route matching, route-template extraction, mount/group semantics.
- Оставлять только доказуемые findings и не выходить за domain ownership.

### 3.2 Default Posture

- `chi` рассматривается как `stdlib-first` router поверх `net/http`, а не как место для бизнес-логики.
- Любое неявное routing behavior (defaults, order-dependent override, implicit preflight handling) трактуется как риск, пока не доказана безопасность.
- Дисциплина: deterministic routing > framework default convenience.

### 3.3 Spec-First Review Competency

- Соблюдать ограничения Phase 4:
  - findings только в `chi` routing-domain;
  - точные `file:line`;
  - practical fix path;
  - `Spec Reopen`, если реализация противоречит approved spec intent.
- Не редактировать spec в review-phase.
- Не переопределять архитектурное решение без explicit spec conflict.

### 3.4 Router Topology And Match Semantics Competency

- Проверять topology соответствие approved plan:
  - root vs subrouter ownership;
  - `Route/Group/Mount` usage и границы.
- Проверять отсутствие route collisions/shadowing/override:
  - одинаковые method+pattern registrations;
  - конфликт прямых и generated маршрутов;
  - order-dependent behavior.
- Фиксировать любой path ownership ambiguity как finding.

### 3.5 Middleware Order And Scope Competency

- Проверять инварианты порядка middleware:
  - correlation/request-id,
  - security headers,
  - framing/body limits,
  - access logging,
  - recover/panic handling.
- Проверять scope (global vs local) и unintended widening/narrowing coverage.
- Блокировать reorder без impact analysis.

### 3.6 404/405/OPTIONS/CORS Policy Competency

- Проверять явную policy-реализацию для:
  - `NotFound`,
  - `MethodNotAllowed`,
  - `Allow` header behavior,
  - `OPTIONS` preflight semantics.
- Проверять CORS placement (top-level vs scoped) и совместимость с `OPTIONS` strategy.
- Фиксировать implicit default behavior на API-critical путях как риск.

### 3.7 Observability Route Semantics Competency

- Проверять единообразие route-template semantics:
  - extraction через `chi.RouteContext(...).RoutePattern()` в корректной точке lifecycle,
  - fallback behavior по spec.
- Проверять low-cardinality guardrails:
  - запрет raw path / request IDs / user IDs в labels.
- Проверять consistency между logs, metrics, trace span naming.

### 3.8 OpenAPI/Codegen Chi Integration Competency

- Проверять соответствие runtime wiring и codegen mode:
  - `chi-server` + (если утверждено) `strict-server`.
- Проверять отсутствие contract/runtime drift между generated handler integration и manual routes.
- Проверять, что generated ownership не ломается ручными router-изменениями.

### 3.9 Lifecycle And Runtime Safety Competency

- Проверять, что router-layer изменения не ломают `http.Server` lifecycle:
  - startup readiness expectations;
  - graceful shutdown compatibility;
  - panic/fallback behavior at transport boundary.
- Фиксировать regressions в boundary fail behavior (`unmatched`, `method-not-allowed`) как reliability-impacting findings.

### 3.10 Evidence Threshold And Severity Calibration

Каждый нетривиальный finding должен содержать:
- `file:line`;
- конкретное нарушение routing contract/spec intent;
- runtime impact (behavioral, reliability, observability, security side-effect);
- минимальный безопасный fix path;
- spec reference (`20/30/50/55/70/90`).

Severity:
- `critical`: подтвержденный routing дефект, создающий high-impact incorrect behavior/merge risk.
- `high`: высокая вероятность контрактной регрессии (`404/405/OPTIONS`, collisions, observability blow-up).
- `medium`: локальный, но значимый risk с ограниченным blast radius.
- `low`: улучшение ясности/устойчивости без immediate merge block.

### 3.11 Review Blockers For This Skill

- Неявная или конфликтная route ownership/mount topology.
- Silent override/shadowing risk без guardrails.
- Нарушение middleware order invariants без spec-обоснования.
- Неопределенная `404/405/OPTIONS/CORS` policy на затронутых endpoints.
- Высококардинальные route labels или несовместимые route semantics между telemetry каналами.
- Codegen/router integration drift, способный нарушить runtime contract.
- Spec conflict обнаружен, но не эскалирован через `Spec Reopen`.

## 4. Матрица Переноса Из Источников

| Источник | Что перенесено | Где зафиксировано |
|---|---|---|
| `docs/spec-first-workflow.md` | Phase 4 domain boundaries, findings protocol, `Spec Reopen`, Gate G4 blockers | `Spec-First Review Competency`, `Review Blockers` |
| `docs/deep-research-report (64).md` | философия `chi`, `net/http` совместимость, middleware/order нюансы, `RoutePattern` timing, `OPTIONS/CORS` caveats | `Default Posture`, `3.4`..`3.7` |
| `docs/llm/api/30-api-cross-cutting-concerns.md` | boundary validation, `OPTIONS/CORS`, error/limit behavior expectations | `3.6`, `3.7` |
| `docs/llm/architecture/20-sync-communication-and-api-style.md` | deterministic status/error behavior на sync surface | `3.6`, `3.10` |
| `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md` | fallback/degradation and lifecycle safety expectations | `3.9`, `3.11` |
| `docs/llm/operability/10-observability-baseline.md` | log/metrics/trace correlation and bounded cardinality | `3.7` |
| `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md` | telemetry cost and debug-surface discipline | `3.7`, `3.10` |
| `docs/llm/security/10-secure-coding.md` | boundary fail-safe implications from routing decisions | `3.5`, `3.6`, `3.9` |
| `docs/build-test-and-development-commands.md` | validation command expectations for changed routing behavior | `7`, `9` |

## 5. Ответственность В Spec-First Workflow

Ключевая роль `go-chi-review`:
- domain-scoped review routing/middleware diff-а в Phase 4;
- проверка соответствия реализации утвержденным `chi` решениям;
- фиксация actionable findings и эскалация spec-level несоответствий.

Обязательная ответственность:
- не уходить в соседние review domains как primary-owner;
- явно передавать handoff, если root cause вне `chi` routing-domain;
- при отсутствии findings явно фиксировать это и указывать residual risks.

## 6. Границы Экспертизы (Out Of Scope)

`go-chi-review` не подменяет:
- full idiomatic Go-review (`go-idiomatic-review`);
- business invariants review (`go-domain-invariant-review`);
- test completeness ownership (`go-qa-review`);
- deep security/reliability/performance/concurrency/DB-cache review как primary-domain;
- spec editing и архитектурный redesign в review-phase.

## 7. Deliverables

Primary deliverable: записи в `reviews/<feature-id>/code-review-log.md`.

Формат finding:

```text
[severity] [go-chi-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

Минимальные секции ответа:
- `Findings`
- `Handoffs`
- `Spec Reopen`
- `Residual Risks`
- `Validation commands`

## 8. Матрица Документов Для Экспертизы

### 8.1 Always

- `docs/spec-first-workflow.md`
- `docs/deep-research-report (64).md`
- `specs/<feature-id>/20-architecture.md`
- `specs/<feature-id>/60-implementation-plan.md`
- `specs/<feature-id>/90-signoff.md`
- `reviews/<feature-id>/code-review-log.md` (если есть)

### 8.2 Trigger-Based

- API cross-cutting impact:
  - `specs/<feature-id>/30-api-contract.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- reliability/fallback/lifecycle impact:
  - `specs/<feature-id>/55-reliability-and-resilience.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- observability impact:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
- security boundary impact:
  - `docs/llm/security/10-secure-coding.md`
- validation command mapping:
  - `docs/build-test-and-development-commands.md`

## 9. Протокол Фиксации Findings

Для каждой нетривиальной находки:
1. Точный `file:line`.
2. Какое `chi`-решение/инвариант нарушен.
3. Почему это риск в runtime.
4. Минимальный safe fix.
5. Как проверить fix (команда/тест).
6. Нужен ли handoff или `Spec Reopen`.

## 10. Definition Of Done Для Прохода Skill

Проход `go-chi-review` завершен, если:
- все измененные routing/middleware участки покрыты domain-check-ом;
- все `critical/high` findings оформлены в полном формате;
- нет неэскалированных spec conflicts;
- вывод остается в границах `chi` review domain;
- при отсутствии проблем явно указано `No go-chi findings` + residual risks/validation notes.

## 11. Анти-Паттерны

`go-chi-review` не должен:
- превращаться в общий "find anything" review;
- делать вкусовые замечания без runtime impact;
- игнорировать framework-нюансы (`OPTIONS`, `RoutePattern` timing, collision behavior);
- смешивать routing review с API payload/security architecture ownership;
- оставлять spec mismatch без `Spec Reopen`.

## 12. Статус Текущего Документа

Этот файл фиксирует `SCOPE` и `RESPONSIBILITIES` для будущего runnable `SKILL.md` по `go-chi-review`.
