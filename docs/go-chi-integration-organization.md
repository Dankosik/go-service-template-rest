# Организация интеграции `go-chi`: аудит черновика и скорректированный план

Дата: 2026-03-03  
Статус: draft (организационный документ до внедрения изменений в код)

> Superseded: для итогового решения используйте `docs/go-chi-final-architecture-spec.md`.

## 1. Что проверено

Проверка выполнена по фактическому состоянию репозитория:

- `docs/go-chi-integration-plan.md` (текущий черновик).
- `docs/deep-research-report (64).md` (research по `chi`).
- `AGENTS.md`, `docs/spec-first-workflow.md`, `skills/go-coder/SKILL.md`.
- runtime-файлы: `internal/infra/http/router.go`, `internal/infra/http/middleware.go`, `internal/api/oapi-codegen.yaml`, `internal/api/openapi.gen.go`.
- тесты: `internal/infra/http/router_test.go`, `internal/infra/http/openapi_contract_test.go`.

Отдельно подтверждено технически:

- `oapi-codegen v2.6.0` поддерживает `chi-server: true` вместе со `strict-server: true`.
- при `chi-server` генератор создаёт `api.ChiServerOptions` и использует `chi.Router`.
- дублирование одного и того же `GET /path` в одном `chi.Router` не паникует и приводит к тихому переопределению последней регистрацией.

## 2. Аудит текущего черновика `docs/go-chi-integration-plan.md`

## 2.1 Что в черновике уже правильно

1. Верно определены базовые точки миграции: `oapi-codegen`, router, middleware labels, тесты, skills, docs.
2. Верно отмечен риск потери route-template labels из-за текущей зависимости от `r.Pattern`.
3. Верно выделен риск деградации `/metrics` и необходимость явной политики по `404/405/OPTIONS`.
4. Верно заложена проверка через `make openapi-check`, `make test`, `go vet ./...`, `make lint`, `make skills-check`.

## 2.2 Ошибки и противоречия в черновике

1. Пункт про «исторический дрейф» skills в `docs/spec-first-workflow.md` некорректен.
   Сейчас все skill-имена из workflow присутствуют в `skills/*`.
2. В документе одновременно фигурируют `go-chi-reviewer` и `go-chi-review`.
   Нужно зафиксировать одно canonical имя до начала реализации.
3. В блоке `/metrics` не зафиксирован важный факт: в `chi` дубликат маршрута может быть тихо переопределён.
   Без явной topology-стратегии возможна незаметная регрессия.

## 2.3 Пробелы, которые нужно покрыть

1. Не покрыт `otel` span naming:
   сейчас `otelhttp.WithSpanNameFormatter` использует `r.Pattern`; после перехода на `chi` это станет `<unmatched>`, если не добавить извлечение через `chi.RouteContext`.
2. Не зафиксирована безопасная topology для `/metrics` без дубликатов на одном router.
3. Не зафиксирована единая policy для `NotFound`/`MethodNotAllowed`/`OPTIONS` (+ `Allow` header и формат problem-envelope).
4. Не зафиксирован инвариант точного порядка middleware (важно для request-id/logging/recover/body-guard).
5. Для `docs/spec-first-workflow.md` не обозначено, что `go-chi-spec` и `go-chi-review` должны быть условными (trigger-based), а не обязательными для всех фич.

## 3. Решения, которые нужно зафиксировать до реализации

## CHI-001: Canonical naming новых skills

Рекомендация:

- `go-chi-spec` (SPEC role).
- `go-chi-review` (REVIEW role, в существующем naming-паттерне `*-review`).

Если выбран `go-chi-reviewer`, это должно быть явным осознанным исключением и синхронно отражено во всех документах.

## CHI-002: Codegen baseline

- `internal/api/oapi-codegen.yaml` -> `chi-server: true` + `strict-server: true`.
- OpenAPI source of truth остаётся `api/openapi/service.yaml`.
- generated artifacts остаются в `internal/api` и не редактируются вручную.

## CHI-003: Router topology без скрытых коллизий

Рекомендуемая схема:

1. root router (`chi.NewRouter`) для внешнего middleware-стека и прямого `/metrics`.
2. OpenAPI-generated routes регистрируются в отдельный subrouter и монтируются в root.
3. Не регистрировать один и тот же `GET /metrics` дважды в одном и том же router instance.

Это исключает silent override и делает поведение детерминированным.

## CHI-004: Observability labels и spans

Добавить единый helper извлечения route-template:

1. `chi.RouteContext(r.Context()).RoutePattern()` (после `next.ServeHTTP`).
2. fallback `r.Pattern`.
3. fallback `<unmatched>`.

Использовать один helper для:

- access log route field;
- metrics labels;
- otel span naming formatter.

## CHI-005: HTTP policy (404/405/OPTIONS)

Нужно явно выбрать и задокументировать:

1. формат ответа для `404` и `405` (единый problem-envelope или текущее поведение);
2. требования к `Allow` header для `405`;
3. policy для `OPTIONS` и CORS (включая preflight).

## 4. Скорректированный порядок работ

## Фаза A. Process scaffolding (без смены runtime baseline)

1. Добавить `skills/go-chi-spec/SKILL.md` и `skills/go-chi-review/SKILL.md`.
2. Запустить `make skills-sync` и `make skills-check`.
3. Обновить `docs/spec-first-workflow.md`:
   - добавить `go-chi-spec` как trigger-based SPEC роль для HTTP routing/middleware задач;
   - добавить `go-chi-review` как trigger-based REVIEW роль при изменениях router/middleware/contracts `404/405/OPTIONS`.
4. Обновить `skills/go-coder/SKILL.md`:
   - добавить `chi` router competency;
   - добавить trigger dynamic loading для `go-chi-spec`.

Примечание: в `AGENTS.md` не фиксировать `go-chi` как baseline до фактической миграции runtime.

## Фаза B. Runtime migration

1. Переключить codegen (`chi-server`), перегенерировать `internal/api/openapi.gen.go`.
2. Адаптировать `internal/infra/http/router.go` на `chi` с выбранной topology.
3. Сохранить точный порядок middleware относительно текущего behavior.
4. Обновить извлечение route labels + span names через общий helper.
5. Явно имплементировать policy `404/405/OPTIONS`.

## Фаза C. Verification and freeze

1. Обновить/добавить тесты:
   - `405` + `Allow` header;
   - `OPTIONS` policy;
   - route labels в логах/метриках;
   - отсутствие route-shadowing для `/metrics`;
   - span naming с route-template.
2. Прогнать baseline-команды (см. раздел 6).
3. После успешной миграции обновить `AGENTS.md`:
   - зафиксировать `go-chi` как transport baseline.

## 5. Файлы, которые должны попасть в change set

Обязательно:

- `internal/api/oapi-codegen.yaml`
- `internal/api/openapi.gen.go` (generated)
- `internal/infra/http/router.go`
- `internal/infra/http/middleware.go` (или новый helper-файл рядом)
- `internal/infra/http/router_test.go`
- `internal/infra/http/openapi_contract_test.go`
- `skills/go-chi-spec/SKILL.md` (new)
- `skills/go-chi-review/SKILL.md` (new, либо `go-chi-reviewer` если принято именно так)
- `skills/go-coder/SKILL.md`
- `docs/spec-first-workflow.md`
- `AGENTS.md`

Зеркала skills (через `make skills-sync`):

- `.agents/skills/*`
- `.claude/skills/*`
- `.cursor/skills/*`
- `.gemini/skills/*`
- `.github/skills/*`
- `.opencode/skills/*`

## 6. Валидация

Минимальный набор:

- `make openapi-generate`
- `make openapi-check`
- `make test`
- `go vet ./...`
- `make lint`
- `make skills-sync`
- `make skills-check`

Дополнительно:

- `make test-race` (если затронуты concurrent path-ы).

## 7. Definition of Done

Инициатива завершена, когда:

1. runtime роутинг работает на `chi` без behavioral drift по публичным endpoint.
2. `oapi-codegen` работает в режиме `chi-server + strict-server`.
3. labels/spans используют route-template и сохраняют низкую кардинальность.
4. policy `404/405/OPTIONS` зафиксирована и покрыта тестами.
5. skills `go-chi-spec` и `go-chi-review` добавлены и синхронизированы в зеркала.
6. `go-coder`, `docs/spec-first-workflow.md`, `AGENTS.md` синхронно отражают новый процесс и baseline.
