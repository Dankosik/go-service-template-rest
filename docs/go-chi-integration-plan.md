# План интеграции `go-chi` в шаблон и обновления skill-экосистемы

Дата: 2026-03-03  
Статус: draft (план до фактических изменений кода)

> Superseded: финальные архитектурные решения и закрытие open questions зафиксированы в `docs/go-chi-final-architecture-spec.md`.

## 1. Цель

Подготовить безопасный и предсказуемый переход template-репозитория на `go-chi` с сохранением:
- OpenAPI-first процесса;
- spec-first workflow;
- качества runtime-контрактов (middleware, метрики, error envelope, graceful shutdown);
- переносимой системы skills (`skills/*` + зеркала).

Параллельно нужно заложить отдельные skills под `chi`:
- `go-chi-spec`;
- `go-chi-reviewer` (см. решение по неймингу в разделе 5.1).

## 2. Входные данные

- `docs/deep-research-report (64).md` (исследование по `chi` и практикам использования).
- Текущий routing stack:
  - `internal/infra/http/router.go` (сейчас `http.ServeMux` + OpenAPI std-http server).
  - `internal/infra/http/middleware.go` (метрики/логи через `r.Pattern`).
  - `internal/api/oapi-codegen.yaml` (`std-http-server: true`, `strict-server: true`).
- Текущий workflow:
  - `AGENTS.md`.
  - `docs/spec-first-workflow.md`.
  - `skills/go-coder/SKILL.md`.

## 3. Текущее состояние (важно для миграции)

1. Генерация OpenAPI идёт через `oapi-codegen v2.6.0` в `std-http-server` режиме.
2. `router.go` использует `http.NewServeMux()` и method-aware паттерны Go 1.22 (`"GET /path"`).
3. Логи/метрики привязаны к `r.Pattern`; при переходе на `chi` это нужно адаптировать через `chi.RouteContext`.
4. В canonical source skills находятся в `skills/*`, зеркала синхронизируются через `make skills-sync`.
5. `docs/spec-first-workflow.md` уже содержит ссылки на некоторые review skills, которых нет в `skills/*` (исторический дрейф). Это не блокер для `chi`, но риск повторить рассинхрон.

## 4. Техническая проверка, уже подтверждённая

Локально проверено: `oapi-codegen v2.6.0` поддерживает одновременную генерацию:
- `chi-server: true`
- `strict-server: true`

Это позволяет мигрировать без отказа от strict handlers.

## 5. Решения по дизайну миграции

### 5.1 Нейминг новых skills

Рекомендация:
- spec skill: `go-chi-spec`
- review skill: `go-chi-review` (по паттерну существующих `*-review`)

Если нужно сохранить формулировку пользователя буквально, можно использовать `go-chi-reviewer`, но тогда:
- обновить `docs/spec-first-workflow.md` именно этим именем;
- явно описать исключение из naming-паттерна.

### 5.2 Архитектурный принцип для runtime

Переходить на `chi` как на transport-router, но оставить:
- совместимость с `net/http`;
- OpenAPI source of truth (`api/openapi/service.yaml`);
- strict-handler слой (`api.NewStrictHandlerWithOptions`).

### 5.3 Принцип миграции

Сначала инфраструктура и генерация, затем поведение:
1. `oapi-codegen` переключение на `chi-server`.
2. Роутер и middleware-цепочки.
3. Контрактные тесты и метрики.
4. Документация и skills.

## 6. План работ по пакетам изменений

## WP1. Зависимости и codegen

Файлы:
- `go.mod`
- `internal/api/oapi-codegen.yaml`
- `internal/api/openapi.gen.go` (generated)

Шаги:
1. Добавить `github.com/go-chi/chi/v5` в прямые зависимости.
2. В `internal/api/oapi-codegen.yaml` заменить `std-http-server: true` на `chi-server: true`.
3. Сгенерировать API заново (`make openapi-generate`).
4. Проверить, что generated API использует `api.ChiServerOptions` и `chi.Router`.

Критерий готовности:
- проект компилируется;
- runtime-контракт OpenAPI не деградировал.

## WP2. Адаптация `internal/infra/http/router.go`

Файлы:
- `internal/infra/http/router.go`

Шаги:
1. Заменить `http.ServeMux` композицию на `chi.NewRouter()`.
2. Сохранить существующий внешний middleware stack (корреляция, security headers, access log, body guard, recover).
3. Интегрировать OpenAPI-роуты через `api.HandlerWithOptions(..., api.ChiServerOptions{...})`.
4. Явно определить стратегию для `/metrics`:
   - приоритет прямого handler для избежания буферизации полного payload;
   - не допустить route collision с generated маршрутом.
5. Явно задать `NotFound`/`MethodNotAllowed` handler при необходимости единого problem-envelope.

Критерий готовности:
- поведение основных endpoint не изменилось;
- `metrics` путь не деградировал по памяти/latency;
- корректно работает `405/404` политика.

## WP3. Адаптация извлечения route pattern для логов/метрик

Файлы:
- `internal/infra/http/middleware.go`
- возможно новые helper-файлы в `internal/infra/http/`

Шаги:
1. Перестать полагаться только на `r.Pattern`.
2. Добавить helper:
   - сначала пытается взять pattern из `chi.RouteContext(r.Context()).RoutePattern()`;
   - fallback на `r.Pattern` (для совместимости/тестов/вспомогательных handler’ов);
   - fallback на `<unmatched>`.
3. Зафиксировать стабильный формат label, например `GET /api/v1/ping`.
4. Считывать route pattern после `next.ServeHTTP` (важно для `chi`).

Критерий готовности:
- текущие метрики сохраняют низкую кардинальность;
- тест на labels проходит без drift.

## WP4. Обновление тестов

Файлы:
- `internal/infra/http/router_test.go`
- `internal/infra/http/openapi_contract_test.go`
- при необходимости дополнительные тесты 404/405/OPTIONS

Шаги:
1. Обновить ожидания там, где меняется поведение роутера.
2. Добавить тесты на:
   - `MethodNotAllowed` (`405`) и `Allow` header;
   - route pattern в метриках/логах;
   - отсутствие route conflict для `/metrics`.
3. Проверить, что проблема из `deep-research-report` про `OPTIONS` обработана явной политикой.

Критерий готовности:
- `make test` и `make openapi-check` зелёные;
- контрактные runtime-тесты стабильны.

## WP5. Новые skills (`go-chi-spec` и review skill)

Файлы (canonical):
- `skills/go-chi-spec/SKILL.md`
- `skills/go-chi-review/SKILL.md` или `skills/go-chi-reviewer/SKILL.md`

Зеркала:
- `.agents/skills/...`
- `.claude/skills/...`
- `.cursor/skills/...`
- `.gemini/skills/...`
- `.github/skills/...`
- `.opencode/skills/...`

Содержание `go-chi-spec`:
- проектирование router topology (`Route/Group/Mount`);
- middleware layering и порядок применения;
- 404/405/OPTIONS/CORS policy;
- правила route label extraction для observability;
- ограничения body/headers/timeouts на boundary.

Содержание review skill:
- проверка route conflicts/overlap;
- проверка middleware order и fail-closed границ;
- проверка trace/request-id continuity;
- проверка `chi.RouteContext` использования и кардинальности метрик;
- проверка graceful shutdown совместимости.

Операционный шаг:
- после добавления skills выполнить `make skills-sync` и `make skills-check`.

Критерий готовности:
- новые skills доступны из `skills/*` и зеркал;
- trigger-description и scope не пересекаются с существующими ролями.

## WP6. Обновление `go-coder` skill под `chi`

Файлы:
- `skills/go-coder/SKILL.md`

Изменения:
1. Добавить explicit competency про `chi` + OpenAPI chi-server integration.
2. Зафиксировать инварианты:
   - route labels без высокой кардинальности;
   - middleware order не меняется без явной причины;
   - `NotFound/MethodNotAllowed` контракты сохраняются;
   - no hidden router globals.
3. Добавить trigger-based загрузку `go-chi-spec` материалов при transport-routing задачах.

Критерий готовности:
- `go-coder` не допускает ad-hoc решений по роутингу в обход `go-chi-spec`.

## WP7. Документация процесса (`AGENTS.md`, workflow)

Файлы:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- опционально `docs/skills/*` спецификации под новые skills

Изменения в `AGENTS.md`:
1. Явно указать, что transport-router baseline в template — `go-chi`.
2. Добавить правило динамической загрузки `go-chi-spec`/review skill для HTTP-router задач.
3. Уточнить, что `skills/*` — canonical, зеркала обновляются через `make skills-sync`.

Изменения в `docs/spec-first-workflow.md`:
1. Добавить `go-chi-spec` в SPEC skills.
2. Добавить review skill в REVIEW skills и `Reviewer Focus Matrix`.
3. Уточнить точку в Phase 2, где закрываются router/middleware/404-405/OPTIONS решения.
4. Уточнить точку в Phase 4, где проверяется route/middleware correctness.

Критерий готовности:
- workflow и skills согласованы по именам и ролям;
- нет новых несоответствий между документами и runnable skills.

## 7. Порядок выполнения (рекомендуемый)

1. Сначала WP1 (codegen + deps).
2. Потом WP2 и WP3 (runtime router + observability labels).
3. Потом WP4 (tests).
4. Потом WP5 и WP6 (skills и обновление go-coder).
5. В конце WP7 (AGENTS/workflow/docs sync).

Так минимизируется риск смешать инфраструктурные и process-изменения.

## 8. Валидация

Минимальный набор:
- `make openapi-generate`
- `make openapi-check`
- `make test`
- `go vet ./...`
- `make lint`
- `make skills-check`

Дополнительно при затрагивании параллелизма/горутин:
- `make test-race`

## 9. Риски и меры

1. Риск: изменение формата route label в метриках.
   - Мера: фиксированный helper + тест на labels.
2. Риск: регрессия `405/Allow/OPTIONS`.
   - Мера: явные тесты и явно заданная policy в роутере.
3. Риск: конфликт `/metrics` между generated и прямым handler.
   - Мера: детерминированная схема mount order + тест.
4. Риск: рассинхрон skills между `skills/*` и зеркалами.
   - Мера: `make skills-sync` + `make skills-check` в конце.
5. Риск: дрейф между workflow и фактическими skill именами.
   - Мера: единый canonical naming decision перед созданием новых skill папок.

## 10. Definition of Done для этой инициативы

Инициатива считается завершённой, когда:
1. Runtime работает на `chi` и проходит baseline quality gates.
2. OpenAPI generation использует `chi-server` + `strict-server`.
3. Метрики/логи сохраняют route-template semantics и низкую кардинальность.
4. Добавлены и синхронизированы новые skills для `chi`.
5. `go-coder` обновлён и учитывает `chi` инварианты.
6. `AGENTS.md` и `docs/spec-first-workflow.md` отражают новую skill-модель и router baseline.
