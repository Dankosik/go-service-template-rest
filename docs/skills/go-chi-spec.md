# Skill Spec: `go-chi-spec` (Expertise-First)

## 1. Назначение

`go-chi-spec` — эксперт по архитектуре HTTP transport-layer для Go-сервисов на базе `github.com/go-chi/chi/v5` в spec-first процессе.

Ценность skill:
- фиксирует router/middleware решения до кодинга;
- предотвращает runtime-drift по `404/405/OPTIONS`, route conflicts и observability labels;
- делает интеграцию `chi` с OpenAPI/codegen предсказуемой и проверяемой.

## 1.1 Философия `go-chi`

`go-chi` в этом репозитории трактуется как:
- `stdlib-first` роутер поверх `net/http`, а не «полный web framework»;
- инструмент композиции transport-слоя для растущего API (`Route/Group/Mount`, локальные middleware);
- способ сохранить совместимость с `http.Handler`-экосистемой и при этом снизить сложность маршрутизации.

`go-chi-spec` должен защищать именно эту философию:
- бизнес-логика не утекает в роутер;
- роутинг-решения остаются явными и проверяемыми;
- framework-нюансы (`OPTIONS`, `Allow`, `RoutePattern`, порядок middleware) формализуются до coding phase.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает за `chi`-архитектуру и routing policy внутри этого контура.

## 2. Ядро Экспертизы

`go-chi-spec` принимает решения по:
- router topology:
  - root router vs mounted subrouters;
  - `Route/Group/Mount` стратегия;
  - предотвращение route shadowing и silent override;
- middleware layering:
  - порядок и scope (`global` vs route-local);
  - deterministic behavior для correlation/logging/recover/body-limits/security headers;
- OpenAPI integration:
  - `oapi-codegen` режим для `chi` (`chi-server` + `strict-server`);
  - wiring generated handlers в `chi` topology;
- HTTP policy:
  - `NotFound`, `MethodNotAllowed`, `Allow` header behavior;
  - `OPTIONS`/CORS policy и preflight semantics;
- observability semantics:
  - route-template extraction (`chi.RouteContext(...).RoutePattern()`);
  - low-cardinality labels для metrics/logs/traces;
  - span naming consistency;
- transport resilience shape:
  - безопасность path-level fallback behavior;
  - сохранение корректного shutdown/serve lifecycle на `net/http` уровне.

Дополнительно skill обязан явно учитывать `chi`-семантику:
- глобальный `Use()` исполняется до полного разрешения route context;
- `RoutePattern()` для telemetry следует читать после `next`;
- дефолтные `405/OPTIONS` semantics не считаются «достаточными» без явной policy-фикисации;
- дублирующие route registrations рассматриваются как риск и требуют guardrails.

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-chi-spec` — Phase 2 (Spec Enrichment Loops), когда изменения затрагивают HTTP router/middleware behavior.

Обязательная ответственность skill:
- закрыть все routing-related open questions до coding phase;
- удерживать transport decisions в `20-architecture.md` и синхронизировать их с `30/50/55/60/70/80/90`;
- явно фиксировать policy для `404/405/OPTIONS/CORS`, если меняется router behavior;
- не допускать решений вида "настроим роутер в коде по месту".

## 4. Границы Экспертизы (Out Of Scope)

`go-chi-spec` не подменяет соседние роли:
- endpoint payload/status/error model как primary-domain `api-contract-designer-spec`;
- authz/tenant isolation threat model как primary-domain `go-security-spec`;
- SLI/SLO/alerting политика как primary-domain `go-observability-engineer-spec`;
- detailed reliability policy (retry budget, bulkheads) как primary-domain `go-reliability-spec`;
- тест-стратегия как primary-domain `go-qa-tester-spec`;
- низкоуровневая реализация кода и тестов (Phase 3 scope).

## 5. Основные Deliverables Skill

Primary:
- `20-architecture.md`:
  - router topology decision;
  - middleware ordering model;
  - route conflict prevention rules;
  - runtime fallback policy at transport boundary.

Сопутствующие артефакты (по влиянию):
- `30-api-contract.md`:
  - API-visible effects of `404/405/OPTIONS` policy.
- `50-security-observability-devops.md`:
  - route-label cardinality guardrails;
  - correlation/span naming constraints.
- `55-reliability-and-resilience.md`:
  - transport failure behavior for unmatched/method-not-allowed paths.
- `60-implementation-plan.md`:
  - deterministic sequence migration (`ServeMux` -> `chi`) без скрытых решений.
- `70-test-plan.md`:
  - обязательные tests для routing policy и label correctness.
- `80-open-questions.md`:
  - только `chi`/routing blockers с owner.
- `90-signoff.md`:
  - финальные `chi` decisions + reopen conditions.

## 6. Матрица Документов Для Экспертизы

### 6.1 Always

- `docs/spec-first-workflow.md`
- `docs/project-structure-and-module-organization.md`
- `docs/llm/go-instructions/30-go-project-layout-and-modules.md`
- `docs/llm/architecture/10-service-boundaries-and-decomposition.md`
- `docs/llm/api/30-api-cross-cutting-concerns.md`

### 6.2 Trigger-Based

- Если нужен source-backed разбор философии/нюансов `chi`:
  - `docs/deep-research-report (64).md`
- Если меняется sync HTTP behavior и error mapping:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
- Если затрагиваются degradation/failure semantics маршрутизации:
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Если затрагивается observability route labeling/span naming:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
- Если затрагиваются boundary security controls:
  - `docs/llm/security/10-secure-coding.md`
- Если есть OpenAPI/codegen implications:
  - `api/openapi/service.yaml`
  - `internal/api/oapi-codegen.yaml`
  - `internal/api/README.md`

## 7. Протокол Принятия `chi` Решений

Каждое нетривиальное решение фиксируется как `CHI-###`:
1. Контекст и routing-проблема.
2. Варианты (минимум 2 для нетривиальных случаев).
3. Выбранный вариант и rationale.
4. Trade-offs (выигрыши/потери).
5. API/security/operability impact.
6. Риски + контрольные меры.
7. Reopen conditions.

Обязательные decision classes:
- topology and mounting strategy;
- middleware order invariants;
- `404/405/OPTIONS/CORS` policy;
- observability route-template policy;
- generated vs direct route coexistence policy.

## 8. Definition Of Done Для Прохода Skill

Проход `go-chi-spec` завершен, если:
- router topology и middleware order зафиксированы и не противоречат другим spec artifacts;
- `404/405/OPTIONS` policy определена явно и не оставлена на coding phase;
- политика route-template labels/spans зафиксирована с low-cardinality guardrails;
- риски route overlap/shadowing формализованы и покрыты implementation/test obligations;
- `60/70` содержат проверяемые шаги без "решим в коде";
- все `chi`-related blockers либо закрыты, либо внесены в `80-open-questions.md` с owner.

## 9. Анти-Паттерны

`go-chi-spec` не должен:
- сводить решение к "просто заменим `ServeMux` на `chi`" без topology/policy;
- оставлять middleware order неявным;
- игнорировать `OPTIONS/CORS` path semantics;
- допускать raw-path labels в метриках/трейсах с высокой кардинальностью;
- переносить решения о route conflict/override в coding phase;
- дублировать домены API/security/QA вместо явного handoff профильным skill.
