# Go-chi Integration Final Architecture Spec

Дата: 2026-03-03  
Статус: final architecture decision (pre-implementation)

## Phase And Target Gate

- Current workflow phase: Phase 1 (Baseline Architecture Frame) в терминах `docs/spec-first-workflow.md`.
- Target gate: подготовка к `Gate G2` (Spec Sign-Off) без архитектурных open questions для этой инициативы.

## Scope

- Перевести transport-router шаблона с `net/http ServeMux` на `go-chi`.
- Сохранить OpenAPI-first + strict-server подход через `oapi-codegen`.
- Формализовать skill-модель для `go-chi` (`go-chi-spec`, `go-chi-review`) и обновить `go-coder`.
- Синхронизировать `AGENTS.md` и `docs/spec-first-workflow.md` без дрейфа имен/ролей.

Out of scope:

- детальный дизайн API payloads;
- data/schema/migration изменения;
- low-level perf tuning beyond routing-related risks.

## 20-architecture.md

## ARCH-CHI-001: Canonical naming for new chi skills

- Owner: `go-architect-spec`.
- Context: нужен новый spec/review контур под routing concerns без ломки существующего naming-паттерна.
- Options:
  - A: `go-chi-spec` + `go-chi-review`.
  - B: `go-chi-spec` + `go-chi-reviewer`.
- Selected: A.
- Rejected: B, потому что это выбивается из текущего стабильного паттерна `*-review` и увеличивает риск docs/skills drift.
- Trade-offs:
  - Gain: единообразие с текущей экосистемой skills.
  - Loss: менее «человеко-читаемое» имя, чем `reviewer`.
- Impact:
  - API/Data: none.
  - Security/Operability: меньше процессного дрейфа и меньше риска ошибочного trigger-loading.
- Risks and controls:
  - Risk: пользовательские формулировки могут продолжать использовать `go-chi-reviewer`.
  - Control: явная canonical фиксация в `docs/spec-first-workflow.md` и `AGENTS.md`.
- Reopen conditions:
  - если в репозитории официально принимается новый naming standard.

## ARCH-CHI-002: Codegen mode (`oapi-codegen`) for chi migration

- Owner: `go-architect-spec`.
- Context: нужно сохранить strict handlers и OpenAPI-first при переходе на `chi`.
- Options:
  - A: `chi-server: true` + `strict-server: true`.
  - B: только `chi-server` без strict wrapper.
  - C: оставить `std-http-server`.
- Selected: A.
- Rejected:
  - B: теряется strict-contract слой и повышается риск decode/response drift.
  - C: не достигается цель миграции на chi.
- Trade-offs:
  - Gain: совместимость с целевым роутером и сохранение strict contracts.
  - Loss: требуется перегенерация и адаптация router wiring.
- Impact:
  - API: contract source of truth остается прежним (`api/openapi/service.yaml`).
  - Data/Security/Operability: none directly.
- Risks and controls:
  - Risk: генераторная несовместимость.
  - Control: зафиксировано evidence: schema `oapi-codegen` поддерживает оба флага одновременно; локальная проверка генерации выполнена.
- Reopen conditions:
  - если целевая версия `oapi-codegen` убирает или меняет совместимость `chi-server` + `strict-server`.

## ARCH-CHI-003: Router topology and `/metrics` conflict policy

- Owner: `go-architect-spec`.
- Context: нужно избежать full-payload buffering в strict `Metrics` и исключить silent route override.
- Options:
  - A: один router, регистрировать и generated `/metrics`, и прямой `/metrics`.
  - B: root `chi.Router` + отдельный mounted OpenAPI subrouter; прямой `/metrics` живет только в root.
  - C: удалить прямой `/metrics`, оставить только strict generated route.
- Selected: B.
- Rejected:
  - A: в `chi` одинаковые route registrations могут тихо переопределяться по порядку (операционный риск).
  - C: возвращает нас к буферизации полного metrics payload через strict handler path.
- Trade-offs:
  - Gain: без silent override, сохраняется memory-safe metrics path.
  - Loss: topology чуть сложнее (root + subrouter).
- Impact:
  - API: публичный путь `/metrics` не меняется.
  - Operability: остается прямой streaming-путь для Prometheus scrape.
- Risks and controls:
  - Risk: при future edits возможно повторное дублирование route в одном router.
  - Control: reviewer checks + тест на route-shadowing.
- Reopen conditions:
  - если strict-handler реализация `/metrics` будет переработана на безбуферный passthrough без topology усложнения.

## ARCH-CHI-004: Route-template extraction for logs, metrics, and OTel spans

- Owner: `go-architect-spec`.
- Context: текущая логика опирается на `r.Pattern`; при chi это не гарантировано.
- Options:
  - A: оставить `r.Pattern` only.
  - B: unified helper: `chi.RouteContext(...).RoutePattern()` after `next`, fallback `r.Pattern`, fallback `<unmatched>`.
  - C: логировать raw `r.URL.Path`.
- Selected: B.
- Rejected:
  - A: теряется стабильность/полнота route template при chi.
  - C: взрыв кардинальности метрик/спанов.
- Trade-offs:
  - Gain: low-cardinality observability labels + корректные span names.
  - Loss: дополнительный helper и обязательная дисциплина порядка вызова.
- Impact:
  - Operability: единая корреляция route labels между access log, metrics и tracing.
- Risks and controls:
  - Risk: helper вызван до `next`.
  - Control: unit tests на extraction timing + code review rule в `go-chi-review`.
- Reopen conditions:
  - если `chi` API изменит семантику `RoutePattern()` относительно post-next чтения.

## ARCH-CHI-005: 404/405/OPTIONS policy for chi baseline

- Owner: `go-architect-spec`.
- Context: дефолт `chi` для `405/OPTIONS` отличается от текущих ожиданий (empty body для 405; preflight semantics требуют явной политики).
- Options:
  - A: оставить дефолт `chi`.
  - B: кастомный `NotFound` и `MethodNotAllowed` с единым `application/problem+json`, `Allow` passthrough, и явным handling policy для `OPTIONS`/CORS.
- Selected: B.
- Rejected:
  - A: несогласованный error surface и неявная preflight-политика.
- Trade-offs:
  - Gain: предсказуемый и контрактный edge behavior.
  - Loss: больше кода и тест-кейсов.
- Impact:
  - API: стабильный error contract и управляемая policy для preflight.
  - Security/Operability: прозрачные отказные ответы, меньше ambiguous behavior.
- Risks and controls:
  - Risk: случайная несовместимость с текущим runtime.
  - Control: добавить explicit tests для `404`, `405 + Allow`, `OPTIONS`.
- Reopen conditions:
  - если product/API policy решит оставить framework-native текстовые ошибки.

## ARCH-CHI-006: Process update timing for `AGENTS.md`

- Owner: `go-architect-spec`.
- Context: когда фиксировать `go-chi` как baseline в always-on contract.
- Options:
  - A: обновить `AGENTS.md` сразу до runtime миграции.
  - B: обновить `AGENTS.md` только после успешной runtime миграции и green gates.
- Selected: B.
- Rejected:
  - A: риск process/runtime drift (документ говорит одно, код еще другое).
- Trade-offs:
  - Gain: документация соответствует фактическому runtime.
  - Loss: временно часть planning-docs будет про future baseline.
- Impact:
  - Operability/Delivery: уменьшение риска неверных агентных действий между этапами.
- Risks and controls:
  - Risk: забыть post-migration update.
  - Control: включить update `AGENTS.md` в Definition of Done и CI docs-drift workflow.
- Reopen conditions:
  - если команда перейдет на policy «docs-first even before runtime switch» по ADR.

## 60-implementation-plan.md

1. Зафиксировать skills baseline:
   - добавить `skills/go-chi-spec/SKILL.md`;
   - добавить `skills/go-chi-review/SKILL.md`;
   - выполнить `make skills-sync` и `make skills-check`.
2. Обновить process docs:
   - `docs/spec-first-workflow.md` (trigger-based включение новых chi skills);
   - `skills/go-coder/SKILL.md` (chi routing competency + trigger loading `go-chi-spec`).
3. Переключить OpenAPI generation:
   - `internal/api/oapi-codegen.yaml` на `chi-server: true`, `strict-server: true`;
   - `make openapi-generate`;
   - проверить generated use of `ChiServerOptions`.
4. Реализовать chi router topology:
   - root router + mounted OpenAPI subrouter;
   - прямой `/metrics` только в root;
   - сохранить текущий middleware order.
5. Внедрить unified route-template helper:
   - использовать в access log, metrics labels, OTel span naming.
6. Имплементировать HTTP policy:
   - custom `NotFound`/`MethodNotAllowed`;
   - explicit `OPTIONS`/CORS behavior.
7. Обновить тесты:
   - `404`;
   - `405 + Allow`;
   - `OPTIONS`;
   - route labels/spans;
   - отсутствие route shadowing для `/metrics`.
8. Выполнить проверки:
   - `make openapi-check`;
   - `make test`;
   - `go vet ./...`;
   - `make lint`;
   - `make skills-check`;
   - `make test-race` (если затронута concurrency).
9. Post-migration docs finalization:
   - обновить `AGENTS.md` и зафиксировать `go-chi` transport baseline.

## 80-open-questions.md

Open questions: none.

Closed decisions:

1. Canonical review skill name: `go-chi-review`.
2. Codegen mode: `chi-server + strict-server`.
3. `/metrics` conflict strategy: root direct handler + mounted generated subrouter.
4. Route template source: unified helper with `chi.RouteContext` post-next.
5. `404/405/OPTIONS` policy: custom explicit policy (not default framework behavior).
6. `AGENTS.md` baseline update timing: only after runtime migration is completed and validated.

## 90-signoff.md

- `go-architect-spec`: accepted.
- Architecture-level blockers: none.
- Hidden “decide later” decisions: none.
- Reopen policy:
  - любой rollback по `openapi-check`/runtime contract/observability cardinality поднимает `Spec Reopen`.

## Conditional Alignment Artifacts

- `30-api-contract.md`:
  - Status: updated (в контексте этой инициативы через planning docs).
  - Изменения: фиксируется cross-cutting behavior для `404/405/OPTIONS` и `Allow`.
  - Linked decisions: `ARCH-CHI-005`.
- `40-data-consistency-cache.md`:
  - Status: no changes required.
  - Justification: migration transport-layer only, data ownership/consistency model не меняются.
  - Linked decisions: `ARCH-CHI-002`, `ARCH-CHI-003`.
- `50-security-observability-devops.md`:
  - Status: updated (в контексте инициативы).
  - Изменения: route-label cardinality control, tracing span naming consistency, docs-drift controls.
  - Linked decisions: `ARCH-CHI-004`, `ARCH-CHI-006`.
- `55-reliability-and-resilience.md`:
  - Status: updated (в контексте инициативы).
  - Изменения: deterministic behavior для `404/405/OPTIONS`, fallback policy for unmatched routes.
  - Linked decisions: `ARCH-CHI-005`.
- `70-test-plan.md`:
  - Status: updated (в контексте инициативы).
  - Изменения: обязательные тесты на route policy, observability labels/spans, route-shadowing prevention.
  - Linked decisions: `ARCH-CHI-003`, `ARCH-CHI-004`, `ARCH-CHI-005`.

## Evidence Sources

External (checked via Exa):

- `oapi-codegen` configuration schema (`chi-server`, `strict-server`):  
  https://github.com/oapi-codegen/oapi-codegen/blob/main/configuration-schema.json
- `go-chi/cors` README note about top-level middleware and `OPTIONS` caveat:  
  https://pkg.go.dev/github.com/go-chi/cors  
  https://github.com/go-chi/cors
- `chi` route override behavior discussion (duplicate route registrations):  
  https://github.com/go-chi/chi/issues/313  
  https://github.com/go-chi/chi/issues/792
- `chi` RoutePattern middleware timing context:  
  https://github.com/go-chi/chi/issues/270

Local empirical checks (2026-03-03):

- `chi v5.2.5`: `POST /foo` against `GET /foo` => `405`, `Allow: GET`, empty body.
- `chi v5.2.5`: `OPTIONS /foo` against `GET /foo` => `405`, `Allow: GET`, empty body.
- `chi v5.2.5`: duplicate route registrations in same router silently override by last registration.
- `oapi-codegen v2.6.0`: generated output contains `ChiServerOptions`, `chi.Router`, and strict server types simultaneously.
