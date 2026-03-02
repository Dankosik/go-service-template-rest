# Engineering standard и LLM-instructions для универсального production-ready Go-микросервиса template

## Scope

Этот стандарт предназначен для репозитория-шаблона “клонируй и сразу пиши production-ready сервис”, где разработчик активно использует LLM-инструменты, а модель должна генерировать идиоматичный, безопасный, поддерживаемый и производительный Go‑код без “догадок” и без скрытых решений. Он оптимизирован под **cloud-native микросервис**, который деплоится в контейнере и чаще всего работает под **entity["organization","Kubernetes","container orchestration"]** (или совместимой платформой), с типичными требованиями: graceful shutdown, health checks, structured logging, метрики/трейсинг, безопасная работа с конфигом и секретами, минимизация supply-chain риска. citeturn8search0turn2search8turn8search1turn11search1turn16search12

Подход применять, когда:
- Нужен **greenfield** сервис (или новый bounded context) и важно, чтобы “boring defaults” были заранее зафиксированы: версия Go, структура проекта, инструментирование, CI‑гейты, политика зависимостей, паттерны HTTP/gRPC, тестовая стратегия, security baseline. citeturn3search1turn2search8turn0search17turn6search0turn5search0
- Ожидается активное использование LLM (в т.ч. в IDE/CLI) и нужно сократить риск “галлюцинаций” через жёсткие MUST/SHOULD/NEVER правила, а также через “сингл-сорс‑оф‑труф” в `docs/` и conventions репозитория. citeturn0search1turn24search1turn0search11turn0search2
- Деплой целится на Kubernetes/контейнеры и важны: probes (readiness/liveness/startup), ограничение ресурсов, securityContext, предсказуемая работа при остановке. citeturn8search5turn8search10turn8search2turn11search1turn9view0

Подход **не** применять “как есть”, когда:
- Это библиотека/SDK, CLI‑утилита, монолит, или сервис с иными нефункциональными требованиями (например, ультра‑низкие задержки, realtime/embedded, сильно stateful‑компонент), где стандартные cloud‑native дефолты могут быть неверны. citeturn8search0
- Вы сознательно выбираете иной стек: сервис‑mesh с жёсткими требованиями к telemetry, или нестандартный runtime/оркестратор, или строгая регуляторика, требующая собственного baseline (например, особые требования к крипто/сертификации, где придётся расширять security‑раздел). citeturn19search6turn16search4
- Команда не готова поддерживать “качество по умолчанию” (линтеры, сканеры, тестовые гейты): тогда шаблон превращается в спорный набор “бумажных” правил. В этом случае сначала определите минимальные неизбежные гейты (format → test → vuln scan). citeturn0search17turn14search3turn0search2

## Recommended defaults для greenfield template

Ниже — boring, battle-tested defaults. Они специально подобраны так, чтобы LLM не “изобретала” архитектуру, а заполняла заранее принятую структуру и паттерны.

**Версия Go и политика обновлений**
- Базовый шаблон должен использовать **последний стабильный релиз Go**, и фиксировать его в `go.mod` (директива `go`). На дату 2026‑03‑02 последний релиз — Go 1.26 (февраль 2026). citeturn3search1turn3search7turn3search0
- Политика поддержки должна учитывать, что backport security/critical fixes обычно идёт на **две последние ветки релизов** (в инженерном смысле это аргумент держаться близко к актуальному релизу). citeturn3search6turn3search4

**Структура репозитория и границы пакетов**
- Структура на базе официальных рекомендаций по layout модулей: `cmd/<service>/` для entrypoint, `internal/` для всего, что не должно импортироваться извне, `pkg/` — только если вы *осознанно* публикуете библиотеку для внешних потребителей. citeturn2search8turn2search4
- Код в `internal/` должен быть организован по слоям, которые уменьшают “пространство догадок” для LLM:  
  `internal/app` (wire-up), `internal/http` или `internal/grpc` (transport), `internal/domain` (модели/правила), `internal/storage` (DB), `internal/clients` (внешние вызовы). Ограничение импорта internal‑пакетов обеспечивается самим Go. citeturn2search4turn2search8

**HTTP по умолчанию**
- По умолчанию используйте стандартную библиотеку `net/http` и `http.ServeMux` с **расширенными паттернами роутинга (Go 1.22+)**: метод+путь (например `"POST /items"`), wildcard‑паттерны. Это снижает зависимость от сторонних роутеров и уменьшает “догадки” LLM при генерации роутов. citeturn3search2turn3search3
- Для сервера обязательно задавайте timeouts и лимиты заголовков (как минимум `ReadHeaderTimeout`, `WriteTimeout`, `IdleTimeout`, `MaxHeaderBytes`). Стандартная документация прямо объясняет назначение `ReadHeaderTimeout` и причины предпочесть его `ReadTimeout`. citeturn4view0

**Graceful shutdown**
- Для shutdown используйте `http.Server.Shutdown(ctx)` и signal-aware context (`signal.NotifyContext`). Документация `Shutdown` описывает алгоритм (закрытие listeners → idle conns → ожидание), ограничения (не закрывает hijacked conns), и необходимость дождаться завершения. citeturn13view0turn12search1

**Логирование**
- По умолчанию используйте `log/slog` (Go 1.21+) как стандартный structured logging API; это снижает фрагментацию и даёт единый стиль key/value‑логов. citeturn1search2turn1search6
- Логи должны учитывать security‑требования: не логировать секреты/токены/пароли/PII, логировать события аутентификации/авторизации и ошибки корректно (guidance от **entity["organization","OWASP","app security project"]**). citeturn6search2turn6search3turn6search0

**Метрики и трейсинг**
- Дефолт:  
  – Tracing через **entity["organization","OpenTelemetry","observability project"]** SDK (OTLP экспорт; семантические конвенции по возможности). citeturn5search5turn5search16turn5search4turn5search0  
  – Metrics: либо Prometheus endpoint, либо OTel metrics — выбрать один baseline (см. trade-offs). Для Prometheus используйте официальный Go‑клиент и /metrics endpoint. citeturn5search2turn5search15turn5search3  
- Важно: в OTel Go документации отмечено, что **logs signal всё ещё experimental**, поэтому “логирование через OTel” не должно быть дефолтом в production template. citeturn5search5

**Health/Readiness/Liveness**
- Шаблон должен иметь отдельные endpoints (или gRPC health) для жизнеспособности и готовности. Под **Kubernetes** semantics probes различаются: liveness определяет рестарт, readiness управляет попаданием в endpoints сервиса, startup защищает slow start. citeturn8search5turn8search1
- Если используется gRPC: применяйте стандартный gRPC Health Checking Protocol и (при Kubernetes‑деплое) известную практику с `grpc-health-probe`. citeturn15search0turn15search9turn15search1

**Конфигурация и секреты**
- Конфиг — через environment variables (12‑factor), чтобы снижать риск случайного коммита конфиг‑файлов и сохранять переносимость между средами. citeturn8search4turn8search0
- Секреты — только через секрет‑хранилища/secret manager/Kubernetes Secrets; обязателен запрет на вывод секретов в логи и на их хранение в репозитории. citeturn6search3turn6search2

**Доступ к БД**
- По умолчанию используйте `database/sql` как стандартный интерфейс, учитывая, что `sql.DB` concurrency-safe и управляет пулом соединений. citeturn14search0

**Инструменты качества и security**
- Обязательные гейты: `gofmt`, `go test`, `go vet`, `govulncheck`. `gofmt` — де-факто стандарт форматирования в экосистеме Go. citeturn0search17turn14search3turn0search2turn0search6
- Для управления dev‑tools в Go 1.24+ используйте **`tool` directive** в `go.mod` (официальная поддержка), вместо старого `tools.go` workaround. citeturn26search9turn26search3turn26search8

**Контейнеризация и runtime baseline**
- Build: multi-stage Docker builds (официальная рекомендация Docker), финальный образ — минимальный (по умолчанию distroless), чтобы уменьшить attack surface. citeturn17search3turn17search0
- Базовый стандарт контейнеров опирается на OCI image spec (для совместимости инструментов). citeturn17search5turn17search17
- Kubernetes hardening baseline: securityContext (в т.ч. `readOnlyRootFilesystem`, `allowPrivilegeEscalation`), и ориентация на Pod Security Standards `restricted`/`baseline` как reference point. citeturn9view0turn11search1

**Supply-chain security минимум**
- Использовать Go checksum database и proxy по умолчанию (это часть модели Go module security). citeturn2search6turn2search9
- Для зрелости: SBOM (SPDX или CycloneDX) и provenance/SLSA по мере роста требований. citeturn16search21turn16search3turn16search4turn16search20

## Decision matrix / trade-offs

Ниже — матрица решений именно для template: что выбираем по умолчанию и при каких условиях отступаем. Там, где выбор спорный, отмечены trade-offs.

| Область | Default для template | Когда выбрать иначе | Trade-offs / риски |
|---|---|---|---|
| Transport API | HTTP (`net/http` + `ServeMux` patterns) citeturn3search2turn3search3 | gRPC при high-throughput межсервисном RPC, строгих контрактах и polyglot клиентах citeturn15search12 | HTTP проще дебажить/прокси/кэшировать; gRPC даёт строгие контракты и эффективный бинарный протокол, но сложнее для “внешних” клиентов и требует protobuf toolchain. citeturn15search12turn15search0 |
| Router | Stdlib `ServeMux` (Go 1.22+ методы/wildcards) citeturn3search2turn3search3 | Сторонний router, если нужны сложные middleware/route groups/паттерны, которые неудобны в stdlib | Снижение зависимостей vs функциональность. Stdlib уменьшает “галлюцинации” LLM и риск supply-chain, но часть удобств придётся реализовать самим. citeturn2search6turn3search3 |
| Observability | Traces: OTel; metrics: Prometheus endpoint или OTel metrics (выбрать один) citeturn5search5turn5search2turn5search0 | Чистый Prometheus‑стек без OTel; или наоборот, полностью OTel pipeline | OTel — “универсальный ingestion layer” и стандарты семконвенций, но logs в Go ещё experimental. Prometheus — battle‑tested модель метрик и data model. citeturn5search5turn5search15turn5search0 |
| Logging | `log/slog` structured logging citeturn1search2turn1search6 | zap/zerolog, если нужен иной handler/ecosystem или экстремальная оптимизация | `slog` — stdlib API и единый стиль; сторонние либа могут дать более зрелый экосистемный набор, но увеличивают вариативность и “догадки”. citeturn1search2turn1search6 |
| Конфиг | Env vars (12-factor) + строгая валидация на старте citeturn8search4 | config file, если нужен сложный конфиг/динамика/локальные профили | Env упрощает деплой и снижает риск коммита секретов; файлы удобны локально, но требуют дисциплины и secret management. citeturn8search4turn6search3 |
| DB доступ | `database/sql` + конкретный драйвер citeturn14search0 | ORM, если домен очень CRUD‑heavy и команда готова к абстракциям | `database/sql` — стандарт, понятный и предсказуемый; ORM ускоряет разработку, но добавляет магию и риск неочевидной производительности. citeturn14search0 |
| Timeouts/limits | Явно задавать `ReadHeaderTimeout/WriteTimeout/IdleTimeout/MaxHeaderBytes` citeturn4view0 | Иначе только если сервис *не* HTTP server (например, worker) | Без timeouts выше риск slow-client/DoS классов проблем; doc `net/http` прямо подчёркивает, почему `ReadHeaderTimeout` предпочтительнее `ReadTimeout`. citeturn4view0turn19search3 |
| Shutdown | `signal.NotifyContext` + `Server.Shutdown` with timeout citeturn12search1turn13view0 | Отдельный lifecycle manager, если много subsystems | `Shutdown` не закрывает hijacked conns; долгоживущие соединения нужно закрывать отдельно (doc). citeturn13view0 |
| Container base image | Distroless + multi-stage build citeturn17search0turn17search3 | Scratch (если умеете правильно добавить CA certs/zoneinfo), или Alpine (если нужен shell/пакеты) | Distroless минимизирует attack surface (нет shell/package manager), но сложнее дебажить runtime; multi‑stage — официальный паттерн. citeturn17search0turn17search3 |
| Security baseline | OWASP logging/secrets/input validation + API Top 10 coverage citeturn6search2turn6search3turn19search0turn6search0 | ASVS‑level процесс, если есть formal security requirements | OWASP API Top10 даёт практический список рисков (например, resource consumption/rate limiting), ASVS — более формальная спецификация требований/проверок. citeturn6search0turn7view0 |
| Tooling deps in go.mod | `tool` directives (Go 1.24+) citeturn26search9turn26search3 | `go install tool@version` (вне модуля) или legacy tools.go для старых Go | `tool` directive официально заменяет workaround “tools.go”, делает инструменты частью модуля и доступными через `go tool`. citeturn26search9turn26search8turn26search2 |

## Набор правил MUST / SHOULD / NEVER для LLM

Этот раздел — “нормативка” для `docs/llm/instructions.md`. Он должен читаться как контракт: если LLM нарушает правила — PR не принимается.

### MUST

**Код и стиль**
- MUST генерировать Go‑код, который проходит `gofmt` (и не вносить ручной формат), потому что gofmt — стандартное средство enforcing layout в Go‑экосистеме. citeturn0search17  
- MUST следовать idiomatic Go (в т.ч. по именованию/структуре), опираясь на Effective Go и Go Code Review Comments как базовый reference. citeturn0search0turn0search1  
- MUST добавлять doc comments ко всем exported именам (конвенции doc comments описаны официально). citeturn24search1  

**Контексты, таймауты, shutdown**
- MUST принимать `context.Context` в публичных методах, которые могут блокироваться/делать I/O, и MUST прокидывать его вниз по стеку (server->client->db), как требует документация `context`. citeturn1search1  
- MUST на HTTP сервере задавать осмысленные timeouts (`ReadHeaderTimeout` предпочтительнее для контроля slow headers; допустимо использовать оба), и ограничивать `MaxHeaderBytes`. citeturn4view0  
- MUST реализовывать graceful shutdown через `signal.NotifyContext` и `http.Server.Shutdown` с дедлайном, и MUST дождаться завершения `Shutdown` (doc требует “не выходить из программы раньше”). citeturn12search1turn13view0  

**Ошибки**
- MUST использовать error wrapping и inspection (`errors.Is/As`, `%w`, `errors.Join` где нужно) согласно Go 1.13+ и Go 1.20+ semantics. citeturn18search0turn18search1turn18search2turn18search3  

**Безопасность**
- MUST не логировать секреты/ключи/токены и MUST следовать принципам secure logging и secrets management. citeturn6search2turn6search3  
- MUST выполнять input validation на границе доверия (раньше в потоке данных), и для SQL использовать parameterized queries/защиты от injection. citeturn19search0turn19search1  
- MUST учитывать OWASP API Security Top 10 2023 риски, особенно authorization, authn и resource consumption (rate limiting/ограничения), когда генерируется публичное API. citeturn6search0turn19search3  

**Tooling/security scanning**
- MUST запускать `go vet` и исправлять предупреждения как часть “green” качества. citeturn14search3  
- MUST запускать `govulncheck` (или эквивалентный интеграционный шаг) и не мерджить изменения, которые вводят известные уязвимости, затрагивающие реально вызываемый код. citeturn0search2turn0search6turn0search10  

### SHOULD

- SHOULD минимизировать внешние зависимости и предпочитать stdlib/официальные проекты, чтобы снижать supply-chain риск и уменьшать пространство решений для LLM (Go module mirror + checksum DB являются частью модели доверия, но не заменяют дисциплину зависимостей). citeturn2search6turn2search9turn16search12  
- SHOULD использовать `log/slog` и структурированные поля (корреляция request_id/trace_id), потому что structured logs лучше для поиска/фильтрации и официально поддерживаются в stdlib. citeturn1search2turn1search6  
- SHOULD использовать `errgroup` для параллельных подзадач с отменой по контексту, вместо ручного WaitGroup+каналов, когда есть управляемые subtask‑ы. citeturn14search1  
- SHOULD, если выбран gRPC, включать стандартный health checking, совместимый с Kubernetes практиками. citeturn15search0turn15search9  
- SHOULD хранить dev‑tools в `go.mod` через `tool` directives (Go 1.24+) и запускать их через `go tool …` для воспроизводимости. citeturn26search9turn26search3turn26search12  
- SHOULD использовать multi-stage Docker build и минимальный runtime образ (например distroless) для уменьшения attack surface. citeturn17search3turn17search0  

### NEVER

- NEVER “придумывать” несуществующие пакеты/функции/поля стандартной библиотеки. Если не уверен — свериться с `pkg.go.dev` или исходниками и выбрать минимальный безопасный вариант (либо оставить TODO с точной ссылкой на решение). citeturn4view0turn1search6turn1search1  
- NEVER использовать `context.Background()` внутри handler/запросного пути вместо `r.Context()`; это ломает cancellation и дедлайны. citeturn1search1  
- NEVER читать request body без лимита и без обработки ошибочных/враждебных входов; это напрямую связано с “unrestricted resource consumption”. citeturn19search3turn19search0  
- NEVER логировать raw credentials/secrets или “подробные” внутренние ошибки так, что они раскрывают чувствительные детали клиенту. (Логи и ошибки — разные каналы). citeturn6search2turn6search3  
- NEVER отключать timeouts на HTTP сервере “потому что так проще”; документация `net/http` описывает риски и рекомендуемые поля. citeturn4view0  

## Concrete good / bad examples, anti-patterns и типичные ошибки LLM

Ниже примеры, которые стоит прямо включить в `docs/engineering-standards.md` как “канонические”.

### HTTP server: timeouts + graceful shutdown

**Good (канонично)**

```go
srv := &http.Server{
    Addr:              cfg.HTTPAddr,
    Handler:           handler,
    ReadHeaderTimeout: 5 * time.Second,
    WriteTimeout:      30 * time.Second,
    IdleTimeout:       2 * time.Minute,
    MaxHeaderBytes:    1 << 20, // 1 MiB
}

ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

go func() {
    <-ctx.Done()
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
    defer cancel()
    _ = srv.Shutdown(shutdownCtx)
}()

err := srv.ListenAndServe()
if err != nil && !errors.Is(err, http.ErrServerClosed) {
    return err
}
```

Смысл: `ReadHeaderTimeout/WriteTimeout/IdleTimeout/MaxHeaderBytes` — явные; shutdown делается через `Server.Shutdown`, который не прерывает активные conns и требует дождаться завершения, а `ListenAndServe` после shutdown возвращает `ErrServerClosed`. citeturn4view0turn13view0turn12search1

**Bad (анти‑пример)**

```go
http.ListenAndServe(":8080", handler) // no timeouts, no shutdown
```

Это оставляет сервер без управляемых ограничений времени/ресурсов и без корректного поведения при SIGTERM в Kubernetes (pod termination). citeturn4view0turn8search5

### Request body: лимиты и input validation

**Good**

```go
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
    defer r.Body.Close()

    var req CreateRequest
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()

    if err := dec.Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid JSON")
        return
    }
    if err := req.Validate(); err != nil {
        writeError(w, http.StatusBadRequest, "validation error")
        return
    }
    // ...
}
```

Решает сразу две категории рисков: ранняя валидация на границе доверия и предотвращение неконтролируемого потребления ресурсов (DoS/стоимость). citeturn19search0turn19search3

**Bad**

```go
b, _ := io.ReadAll(r.Body)           // unbounded
json.Unmarshal(b, &req)              // ignores unknown fields, ignores decode errors
```

Проблема: unbounded read + слабая обработка ошибок повышают риск resource consumption и ошибок валидации. citeturn19search3turn19search0

### Errors: wrapping, inspection, multi-error

**Good**

```go
if err := doThing(); err != nil {
    return fmt.Errorf("do thing: %w", err)
}
```

+ для нескольких ошибок:

```go
return errors.Join(errA, errB)
```

Это соответствует Go error wrapping semantics (1.13+) и multi-error (1.20+). citeturn18search0turn18search1turn18search3turn18search2

**Bad**

```go
if err != nil && strings.Contains(err.Error(), "timeout") { ... } // stringly-typed
```

Строковые сравнения ломают стабильность обработки и противоречат идее инспекции через `errors.Is/As`. citeturn18search11turn18search3

### Логирование: structured и безопасное

**Good (slog)**

```go
logger.Info("request finished",
    "method", r.Method,
    "path", r.URL.Path,
    "status", status,
    "latency_ms", latency.Milliseconds(),
)
```

Structured logging — часть stdlib (`log/slog`) и предназначен для надёжного парсинга/фильтрации. citeturn1search2turn1search6

**Bad**

```go
logger.Info("token=" + token) // секрет в логе
```

Это нарушение secure logging и secrets management guidance. citeturn6search2turn6search3

### Типичные anti-patterns и “галлюцинации” LLM

1) **Игнорирование контекста**: LLM часто создаёт goroutine или делает запросы в БД/HTTP без `ctx`, либо подставляет `context.Background()` “для простоты”. Это ломает cancellation и дедлайны, которые стандартно должны распространяться по цепочке вызовов. citeturn1search1  

2) **Выдуманные API стандартной библиотеки**: особенно в `net/http`, `slog`, `database/sql`. Зафиксируйте правило: если код не компилируется — правка обязательна; если LLM не уверена — выбираем stdlib-путь и проверяем на `pkg.go.dev`. citeturn4view0turn1search6turn14search0  

3) **Отсутствие server timeouts**: LLM часто оставляет `ListenAndServe` как в туториалах. Для production template это запрет, потому что `net/http` предоставляет явные поля timeouts и лимиты и документирует их смысл. citeturn4view0  

4) **Небезопасное логирование**: LLM любит “распечатать весь request/headers/body”. Это почти гарантированный утечка токенов/PII, что противоречит OWASP рекомендациям по логированию/секретам. citeturn6search2turn6search3  

5) **Неправильные ожидания от `Server.Shutdown`**: часто предполагается, что shutdown “закроет всё”. Документация явно говорит: hijacked conns (например WebSocket) не закрываются автоматически; требуется отдельная логика. citeturn13view0  

## Review checklist для PR/code review и что оформить отдельными файлами

### Review checklist

Этот чеклист лучше хранить как `docs/review-checklist.md` и частично дублировать в PR template.

**Корректность и поддерживаемость**
- Код отформатирован `gofmt`, нет “ручного” форматирования. citeturn0search17  
- Экспортируемые сущности имеют doc comments по правилам Go doc comments. citeturn24search1  
- Ошибки не теряют контекст; используется wrapping `%w`, инспекция через `errors.Is/As`, для multi-error — `errors.Join` при необходимости. citeturn18search0turn18search1turn18search3  

**Контексты, конкуррентность, shutdown**
- Все I/O операции принимают/прокидывают `context.Context` (нет `context.Background()` внутри request path). citeturn1search1  
- Для параллельных подзадач корректно управляется отмена/ошибки (предпочтительно `errgroup` при наличии групп задач). citeturn14search1  
- HTTP сервер имеет timeouts и лимиты; graceful shutdown реализован через `Server.Shutdown` и signal-aware контекст. citeturn4view0turn12search1turn13view0  

**Security**
- Нет секретов/токенов в логах, нет секретов в репозитории; соблюдены рекомендации secure logging / secrets management. citeturn6search2turn6search3  
- Валидация входов на границе доверия, защита от injection (parameterized queries), throttling/лимиты ресурсов для рискованных endpoints. citeturn19search0turn19search1turn19search3  
- Запущены `go vet` и `govulncheck`, результаты “зелёные”. citeturn14search3turn0search6turn0search2  

**Observability**
- Логи структурированные (`slog`), ключи стабильны, нет “спама” на hot path. citeturn1search2turn1search6  
- Трейсы/метрики используют согласованные semantic conventions (где применимо). citeturn5search4turn5search0  

**Kubernetes readiness**
- Есть отдельные endpoints/health checks для readiness/liveness/startup (или gRPC health). citeturn8search5turn15search0turn15search9  
- Контейнерный baseline учитывает securityContext и Pod Security Standards как reference. citeturn9view0turn11search1  

### Что оформить отдельными файлами в template repo

Минимальный набор файлов, который “фиксирует решения” и снижает догадки LLM:

- `docs/engineering-standards.md` — этот стандарт (defaults, архитектурные решения, правила ошибок/контекстов/HTTP). citeturn0search0turn0search1turn4view0turn1search1  
- `docs/llm/instructions.md` — MUST/SHOULD/NEVER правила + канонические примеры + запрет на галлюцинации (ссылки на pkg.go.dev/официальные доки). citeturn0search17turn24search1turn0search6  
- `docs/testing-strategy.md` — раздел ниже (матрица тестов, границы ответственности, правила выбора тестов под изменение). citeturn22search1turn20search5turn0search3  
- `docs/security-baseline.md` — аккуратная выжимка из OWASP: logging, secrets, input validation, API Top 10 карты рисков. citeturn6search2turn6search3turn19search0turn6search0  
- `docs/observability.md` — выбранный baseline (OTel traces + metrics strategy), что именно логируем/трейсим/метрим, что запрещено. citeturn5search0turn5search5turn5search15turn1search2  
- `docs/review-checklist.md` и `.github/pull_request_template.md` — review гейты. citeturn0search1turn0search2  
- `Makefile` (или `taskfile`) — стандартные команды: fmt, test, vet, vuln, lint. База должна опираться на go toolchain. citeturn14search3turn0search6turn0search17  
- `go.mod` с `tool` directives для инструментов (Go 1.24+), чтобы в репо были зафиксированы версии dev‑tools и их можно было запускать через `go tool`. citeturn26search9turn26search3turn26search12  
- `SECURITY.md` — политика уязвимостей + обязательность `govulncheck`. citeturn0search2turn0search10  
- `Dockerfile` с multi-stage build и минимальным runtime образом; `deploy/` или `helm/` с probes и securityContext примерами. citeturn17search3turn17search0turn8search1turn9view0  
- `CONTRIBUTING.md` — правила изменений, тестовая дисциплина, release/версионирование.

## Исследование подтемы: полная testing strategy для Go-микросервиса

Цель стратегии — не “пирамида тестов”, а **decision system**: какие типы тестов обязательны по умолчанию, где границы ответственности, и как LLM выбирает минимально достаточный набор тестов под конкретное изменение.

### Базовые принципы (для template)

1) **`go test` — центр**: unit/integration/benchmark/fuzz должны быть доступны через `go test` (или дополняться внешними harness, но запуск должен быть стандартизирован). `testing` пакет — официальный фундамент тестирования, поддерживает тесты, бенчмарки, fuzzing. citeturn22search1turn20search5  
2) **Race как safety net**: конкурентные баги ловятся поздно и дорого; использовать race detector как обязательный слой для CI (хотя он дороже по времени). citeturn0search3  
3) **Fuzz как security усилитель**: Go fuzzing coverage-guided и прямо рассматривается как способ находить уязвимости/edge cases, включая security‑классы багов. citeturn20search5turn20search8turn20search13  
4) **Граница ответственности теста важнее названия**: unit тест не должен требовать внешних систем; интеграционный тест обязан поднимать/использовать реальную зависимость (DB/queue) или стабильный test double, но должен фиксировать контракт интеграции. citeturn14search0turn1search1  

### Матрица типов тестирования для template

| Тип тестов | Default обязательность | Граница ответственности | Когда добавлять/обновлять | Как запускать в CI |
|---|---|---|---|---|
| Unit | Обязателен по умолчанию | Чистая бизнес‑логика, валидация, маппинги, error handling; без сети/диска/БД | Любое изменение логики, валидации, форматирования ответов | `go test ./...` (быстро) citeturn22search1 |
| Integration | Обязателен, если есть внешние зависимости | Реальная БД, реальный HTTP client к mock‑server, реальные миграции | Изменения SQL, транзакций, миграций, клиентов внешних API | `go test ./... -tags=integration` или отдельный job |
| Contract | Условно обязателен | API контракт (OpenAPI/proto), версии, backwards compatibility | Любое изменение HTTP/gRPC API (поля/коды/семантика) | Генерация/валидация контрактов + тесты |
| End-to-end | Не обязателен по умолчанию (дорого) | Сквозной путь через сеть/деплой, включая конфиг, auth, real deps | Критичные флоу, релизные проверки, крупные рефакторы | Nightly/Release pipeline |
| Smoke | Обязателен для релизного шаблона | Минимальная проверка “сервис жив и отвечает” после деплоя | Каждый деплой в staging/prod | Отдельный короткий набор |
| Regression | Автоматически покрывается unit/integration, но выделяется практикой | Повторяемый набор баг‑кейсов | После фикса багов (добавить тест, который reproduces bug) | Включить в стандартные suites citeturn22search1 |
| Migration tests | Обязателен, если есть БД | Миграции вперёд/назад* (если поддерживается), совместимость схемы | Любое изменение migrations | В integration job (поднять DB, прогнать миграции) |
| Load/perf | Не обязателен по умолчанию, но benchmark‑каркас обязателен | Производительность hot paths, latency/allocations, throughput | При изменениях в hot path, при SLO нарушениях | `go test -bench` локально/в perf pipeline citeturn23search5turn22search1 |
| Fuzz | Рекомендуется по умолчанию для парсеров/валидаторов | Coverage-guided fuzz targets для функций с “враждебным” вводом | Любой новый парсер/декодер/валидатор/протокол | Отдельный CI job с лимитом времени citeturn20search5turn20search13 |
| Security tests | Обязателен baseline (vuln scan), расширяется по рискам | SAST/линт, dependency vuln scan, security unit/integration | Любые изменения auth, crypto, десериализация, рискованные endpoints | `govulncheck` + targeted suites citeturn0search6turn0search2turn6search0 |
| Chaos | Не обязателен по умолчанию | Устойчивость к отказам (latency, timeouts, kill pods, network) | После стабилизации сервиса и появления SLO | Отдельная среда; по принципам chaos engineering citeturn20search3turn20search11 |

\*Rollback миграций — спорно: многие команды делают only-forward migrations. Это нужно явно зафиксировать в template decision (и тогда migration tests проверяют только forward). Trade-off: rollback упрощает “быстрый откат”, но усложняет миграции и дисциплину; only-forward требует blue/green и backward-compatible changes. (Это сознательное архитектурное решение, которое следует описать в `docs/testing-strategy.md` как policy.)

### Как LLM должна выбирать минимально достаточный набор тестов под изменение

Правило: **LLM не выбирает “все тесты всегда”**. Она выбирает минимальный набор, который защищает риски изменения, и добавляет тесты только там, где появилась новая логика/контракт/интеграция.

Нормативный алгоритм:

1) **Определи поверхность изменения** (что меняется):  
   a) чистая логика/валидация/маппинг;  
   b) транспорт (HTTP/gRPC handlers, статус-коды, схемы);  
   c) интеграция (DB/очередь/внешний HTTP);  
   d) конкуррентность/фоновые задачи;  
   e) security‑критичное (authn/authz, токены, крипто, десериализация).  
   Контекст и request-scoped значения должны сохраняться при любых изменениях request path. citeturn1search1  

2) **Минимальный обязательный набор по умолчанию**:  
   - Всегда: unit тесты для новой/изменённой логики (`go test ./...`). citeturn22search1  
   - Если добавлена/изменена конкурентность или goroutine orchestration: добавить тест, который проявляет гонку, и прогнать `-race` хотя бы на пакете/модуле. citeturn0search3  
   - Если изменён публичный API: добавить/обновить contract test(s) и unit тесты на сериализацию/валидацию. citeturn3search2turn15search0  

3) **Условия для integration**:  
   - Любое изменение SQL, транзакций, обработчиков ошибок DB, миграций → обязателен integration тест с реальной БД (или максимально близкой). Поскольку `sql.DB` — пул и concurrency‑safe, ошибки часто проявляются на уровне настройки пула/таймаутов/контекста, что unit тест не ловит. citeturn14search0turn1search1  

4) **Условия для fuzz**:  
   - Если добавлен новый парсер/валидатор (JSON decode, URL parsing, custom protocol, regex‑heavy logic) или повышается риск edge cases → добавить fuzz target. Go fuzzing coverage-guided и рассматривается как способ находить security‑баги. citeturn20search5turn20search8  

5) **Условия для load/perf**:  
   - Изменения на hot path (парсинг, сериализация, heavy allocations) → добавить benchmark или обновить существующий, потому что `testing` пакет поддерживает бенчмарки через `-bench`. citeturn23search5turn22search1  

6) **Security baseline обязателен всегда**:  
   - `govulncheck` должен быть зелёным на каждом PR; это low-noise инструмент для обнаружения уязвимостей в реально вызываемом коде. citeturn0search6turn0search2turn0search10  

7) **Документируй “почему этих тестов достаточно”**:  
   - В PR описание: “изменение затрагивает X → добавлен unit; затрагивает DB → добавлен integration; затрагивает parser → добавлен fuzz”. Это снижает риск, что LLM “просто добавила тесты ради тестов”.

### Инструментальные рекомендации для реализации стратегии в репо

- Использовать встроенные возможности Go: fuzzing (`go test -fuzz`), бенчмарки (`-bench`), и базовый `testing` пакет. citeturn20search5turn23search5turn22search1  
- Добавить отдельный CI job для `-race` (дороже, но строгий сигнал для конкурентности). citeturn0search3  
- Включить coverage как сигнал качества (без “религиозных” порогов, но с контролем деградации). Go tooling эволюционирует (например, covdata с Go 1.20+ для работы с coverage data files). citeturn14search2turn22search1  
- Для chaos testing: фиксировать цели/гипотезы и steady state метрики; опираться на принципы chaos engineering (эксперименты как дисциплина, а не случайный “kill -9”). citeturn20search3turn20search11