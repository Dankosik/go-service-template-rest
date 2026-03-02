# Engineering standard и LLM-instructions для production-ready Go микросервиса

## Scope

Этот стандарт и шаблон репозитория предназначены для **greenfield микросервисов на Go**, которые:
- разворачиваются как отдельный сервис (обычно в контейнере), имеют сетевые зависимости (БД/кэш/очереди) и должны быть **наблюдаемыми и безопасными по умолчанию**; citeturn20search0turn20search1turn5search0
- требуют воспроизводимой сборки, контролируемых зависимостей и автоматической проверки качества в CI; citeturn9view2turn4search6turn7search12
- должны быть удобны для разработки с LLM-инструментами так, чтобы модель **не “додумывала” детали**, а опиралась на явные соглашения репозитория и стандарты. citeturn21search1turn21search8turn21search7turn21search0

Этот подход **не** является оптимальным, если:
- вы пишете библиотеку/SDK, где внешний API и совместимость важнее “микросервисных” соглашений (там структура, semver и compatibility должны быть иными, чем у сервиса); citeturn10search2turn10search1turn10search14
- у вас **ультра-низкая латентность / real-time** с очень специфичными требованиями, где “boring defaults” (универсальные middleware, универсальная обвязка, метрики/трейсы по умолчанию) создают неприемлемый накладной расход — в таком случае часть стандартов нужно выкинуть или сделать compile-time опциональной; citeturn3search5turn31search7turn31search16
- сервис — это “одна функция” / serverless handler, где контейнеризация, probes, полноценная обвязка и отдельные порты под debug/metrics — перерасход сложности (понадобится упрощённый профиль шаблона). citeturn20search0turn1search3

Ключевой принцип scope: **стандарт обязателен там, где цена production-инцидента выше цены “лишних” строк инфраструктурного кода** (контроль ресурсов, таймауты, ассертируемые форматы ошибок, трассировка/метрики/логи, секреты, supply-chain). citeturn22search3turn5search2turn5search3turn7search9

## Recommended defaults для greenfield template

Ниже — “boring, battle-tested defaults” как **соглашение репозитория**. Они минимизируют область догадок как для человека, так и для LLM.

### Версия toolchain и go.mod

- **Target toolchain**: Go 1.26 как baseline для шаблона (на момент 2026‑03‑02 последняя стабильная версия вышла в феврале 2026). citeturn2search3turn2search11turn2search7  
- **go.mod** должен:
  - иметь `go 1.26.0` (или минимально допустимую внутри компании версию); `go`-директива в go.mod задаёт минимальную версию Go и влияет на семантику и поведение инструментария; начиная с Go 1.21 это обязательное требование, а не “подсказка”. citeturn26view0  
  - (опционально) иметь `toolchain go1.26.0`, если проект хочет принудить developer/CI к одинаковой версии (и принять последствия auto-download toolchain). citeturn26view0turn10search0  
  - содержать минимальный набор `replace`, и только временно/локально; `replace` ломает потребителей модуля и усложняет сборку. citeturn26view0

Практический default для template:
- internal repo / единая платформа CI: **включить `toolchain go1.26.0`** (предсказуемость важнее). citeturn26view0turn10search0  
- open-source / разные окружения: **не фиксировать `toolchain`, фиксировать только `go 1.26.0`** и тестировать в CI на поддерживаемых версиях согласно release policy. citeturn2search0turn26view0

### Структура репозитория и границы пакетов

Ориентир: структура должна быть понятной без “внутренних знаний”, потому что LLM и новый инженер читают её одинаково буквально.

- **Одна точка входа**: `cmd/<service>/main.go` (или корневая директория для main-пакета, если сервис один). Для нескольких бинарей — отдельные директории. citeturn0search10turn9view0  
- **Всё “не-публичное” — в `internal/`** (границы enforced go toolchain: код под `internal` импортируем только из поддерева родителя). Это снижает риск случайного формирования “публичного API” внутри монорепы и уменьшает поверхность ошибок при рефакторинге. citeturn14view0turn0search10  
- Для чистоты слоёв:
  - `internal/app/` — композиция use-cases, dependency injection, wiring.  
  - `internal/http/` (или `internal/transport/http/`) — HTTP handlers, middleware, request/response mapping, OpenAPI glue.  
  - `internal/storage/` — все адаптеры к БД/кэшу/очередям (не “доменные” структуры).  
  - `internal/observability/` — tracer/meter init, common attributes, лог correlation. citeturn31search11turn31search0

### HTTP API: контракт, ошибки, безопасность по умолчанию

**API contract**
- Для публичных/межкомандных HTTP API: **OpenAPI 3.1.x** как source of truth (генерация клиентов/контрактные тесты/документация). citeturn11search15turn11search3  
- Формат ошибок: **Problem Details (RFC 9457)**, чтобы не изобретать собственный “error envelope” и не плодить несовместимые форматы между сервисами. citeturn24search0turn24search6

**Request parsing defaults**
- Строгий JSON: `json.Decoder.DisallowUnknownFields()` для входящих payload’ов, чтобы “тихие” новые/опечатанные поля не проходили в прод и не ломали обратную совместимость незаметно. citeturn16search3  
- Ограничение размера body через `http.MaxBytesReader`, так как это прямой инструмент защиты ресурсов от случайных/враждебных больших запросов. citeturn19view0  
- Валидация входных данных должна быть “как можно раньше” и явной (whitelist/формат/диапазоны). citeturn25search2turn25search16

**Transport security**
- Для REST сервисов “secure by default”: **только HTTPS** (на уровне edge / ingress / service mesh — но контракт сервиса должен предполагать TLS как обязательный). citeturn5search1turn25search0  
- Если сервис отдаёт ответы в браузер: задать базовые security headers (HSTS и др.) и иметь возможность отключить/настроить их в зависимости от окружения. citeturn25search1turn25search5

### Таймауты, отмена, graceful shutdown

- Все входящие запросы и исходящие вызовы должны нести `context.Context` и корректно реагировать на отмену/дедлайны: это базовый контракт Go `context`. citeturn17search2turn17search12  
- В HTTP server request context отменяется при дисконнекте клиента/отмене запроса, и его нужно транзитивно прокидывать во все downstream операции (БД/HTTP/gRPC). citeturn17search15turn17search2  
- Graceful shutdown:
  - ловить SIGTERM/SIGINT через `signal.NotifyContext`; citeturn16search1  
  - завершать HTTP server через `(*http.Server).Shutdown(ctx)` (ждёт завершения активных соединений/обработчиков до истечения контекста). citeturn16search4

### Логи, метрики, трейсы

**Логи**
- Default: `log/slog` (стандартная библиотека) и структурированный вывод (ключи/значения), потому что цель — машинно-обрабатываемые логи и минимальная зависимость от внешних логгеров. citeturn0search5turn0search1turn24search2  
- Запрет на утечки: логи не должны содержать секреты/PII; правила редактирования/маскирования должны быть документированы. citeturn5search2turn20search3

**Метрики**
- Экспозиция должна соответствовать экосистеме Prometheus/OpenMetrics: OpenMetrics фиксирует Prometheus text exposition format 0.0.4 как стабильный с 2014. citeturn31search1  
- Нейминг и лейблы: следовать официальным практикам, избегать label-name в metric-name и избегать высококардинальных лейблов. citeturn31search2turn31search18  
- Если используете OpenTelemetry metrics → Prometheus export: exporter должен поддерживать text format 0.0.4. citeturn31search17

**Трейсы**
- Стандарт распространения контекста: W3C Trace Context (`traceparent`/`tracestate`) — де-факто общий формат межвендорной трассировки. citeturn11search2  
- OpenTelemetry как общий API/SDK для traces/metrics; semantic conventions дают “единые” имена атрибутов (важно для корреляции), но в некоторых областях stability может быть “mixed”, и это должно быть отражено как trade-off. citeturn1search4turn31search11turn31search7  
- Логи как сигнал в OpenTelemetry для Go на текущий момент отмечены как experimental (breaking changes возможны), поэтому **templated default — не строить production logging pipeline на OTel logs**, а использовать slog + trace correlation. citeturn31search0turn31search16

### База данных и доступ к данным

- Default интерфейс: `database/sql` (стандартная библиотека). Драйвер должен поддерживать cancel через context, иначе отмена не сработает (и это надо явно учитывать на архитектурном уровне). citeturn29search2turn17search3  
- Всегда использовать `QueryContext/ExecContext` и производные, а не контекст-нечувствительные варианты. citeturn29search9turn17search15  
- Соединения: явно задавать `SetMaxOpenConns/SetMaxIdleConns/SetConnMaxLifetime` под нагрузочный профиль; иначе вы получаете непредсказуемую конкуренцию и латентность. citeturn17search7turn17search3  
- SQL injection: использовать параметризацию и избегать конкатенации строк для запросов. citeturn29search0turn29search4turn29search3

### Build, supply chain, CI gates

- `gofmt` обязателен (индустриальный default Go), и это должно быть автоматизировано. citeturn15search0turn15search2  
- Ошибки: wrap с `%w`, `errors.Is/As` (modern error handling). citeturn15search1turn0search4turn0search14  
- Уязвимости: `govulncheck` как обязательный шаг CI, потому что он анализирует реальные reachable вызовы и даёт “low-noise” сигнал. citeturn7search12turn7search9turn7search5  
- Сборка: использовать `-trimpath` (убирает filesystem paths из бинарей) и управлять VCS stamping через `-buildvcs` (auto/true/false) по политике org. citeturn9view1turn9view2  
- Supply-chain зрелость: целиться хотя бы в базовые требования SLSA (в зависимости от критичности) и документировать уровень/аттестации. citeturn6search0turn5search3turn5search7  
- Контейнеризация:
  - multi-stage build как default; citeturn20search0turn20search16  
  - запуск non-root / hardening (особенно в Kubernetes): ориентироваться на Pod Security Standards (Restricted как концептуальный идеал, даже если не всегда применимо). citeturn20search1turn20search7

## Decision matrix / trade-offs

Ниже — решения, которые почти всегда возникают в новом шаблоне. Важно: **шаблон должен фиксировать default**, а в `docs/decisions/` (или ADR) хранить обоснование выбранного курса.

| Тема | Default для template | Когда менять | Trade-offs / риски |
|---|---|---|---|
| `go` и `toolchain` в go.mod | `go 1.26.0`; `toolchain go1.26.0` в корпоративном шаблоне | OSS/разные окружения, где auto-download toolchain нежелателен | `go` — обязательный минимум (>=1.21) и влияет на поведение toolchain; `toolchain` фиксирует версию, но может инициировать переключение/скачивание по правилам Go toolchain selection citeturn26view0turn10search0 |
| Роутинг | stdlib `net/http` + `ServeMux` | Нужны сложные middleware/маршрутизация/совместимость с legacy | stdlib минимизирует supply chain и “галлюцинации” зависимостей; но сложные кейсы могут требовать доп. библиотек (это следует оформлять решением) citeturn18view0turn15search0 |
| Формат ошибок HTTP | RFC 9457 Problem Details | Если клиентская экосистема требует иной стандарт | RFC 9457 стандартизует machine-readable ошибки и заменяет RFC 7807; снижает фрагментацию citeturn24search0turn24search6 |
| API стиль | OpenAPI 3.1 как контракт | Внутренние RPC/стриминг — лучше gRPC | OpenAPI уменьшает двусмысленность; но для high-throughput inter-service RPC/gatewayless среды gRPC часто удобнее (типизация, стримы) citeturn11search15turn23search0turn23search12 |
| Межсервисный транспорт | HTTP/JSON для внешних и простых внутренних API | gRPC для S2S, стриминга, строгих контрактов | gRPC строится на HTTP/2 и имеет собственные ограничения (например, лимиты concurrent streams и очереди); HTTP проще интегрируется и дебажится citeturn23search0turn23search12turn23search4 |
| Логи | `log/slog` JSON | Нужен единый корпоративный логгер | `slog` — стандартная библиотека и structured logging; внешние логгеры могут дать больше фич, но увеличивают supply chain. citeturn0search5turn0search1 |
| Observability | Traces+metrics через OpenTelemetry, Prometheus-friendly /metrics | Если платформа строго Prometheus-only без OTel | OTel даёт единый подход и semantic conventions; но часть семантик и особенно logs-сигнал могут быть нестабильны (experimental/mixed) citeturn1search4turn31search0turn31search7turn31search17 |
| DB слой | `database/sql` + явные запросы | ORM при сложных доменных маппингах | `database/sql` — стандарт и простая модель; ORM может ускорить разработку, но увеличивает магию/непредсказуемость и риск N+1/скрытых запросов (решать осознанно) citeturn17search3turn29search9turn29search0 |
| Контроль ресурсов | лимиты body, таймауты, concurrency bounds | Если есть доказанные причины ослабить лимиты | OWASP API risk про unrestricted resource consumption делает это не “перфоманс-тюнингом”, а частью security posture citeturn19view0turn4search1turn22search3 |
| Контейнер и Kubernetes hardening | multi-stage + non-root + PSS mindset | Если нет Kubernetes/нужна совместимость | Multi-stage — официальный best practice; PSS Restricted повышает безопасность, но иногда ломает совместимость и требует адаптаций citeturn20search0turn20search1turn20search7 |

## Набор правил в формате MUST / SHOULD / NEVER для LLM

Этот блок задуман как содержимое, которое можно почти напрямую положить в `AGENTS.md`/`CLAUDE.md`/`.github/copilot-instructions.md`/`.cursor/rules/*.md` и использовать как общий префикс инструкций.

### MUST

- **MUST читать существующие файлы репозитория и следовать им как контракту**: если есть `docs/` (архитектура, ADR, OpenAPI, конвенции) — они важнее “общих знаний” модели. citeturn21search1turn21search7turn21search8  
- **MUST не изобретать зависимости/пакеты/функции**. Любая новая зависимость требует явного мотива (security/perf/операционность) и закрепления в decision doc. Это прямой ответ на феномен “package hallucinations”. citeturn28search12turn28search1  
- **MUST генерировать идиоматичный Go-код** согласно Effective Go и Go Code Review Comments, без “квазиязыковых” паттернов из других экосистем. citeturn22search20turn15search0  
- **MUST применять gofmt к любым изменениям Go-кода**. citeturn15search0turn15search2  
- **MUST не игнорировать ошибки** и **MUST оборачивать ошибки с контекстом через `%w`**, чтобы работали `errors.Is/As` и цепочки причин. citeturn15search1turn0search0turn0search4  
- **MUST прокидывать context** через все слои, особенно в БД/внешние HTTP/gRPC вызовы, и уважать отмену и дедлайны. citeturn17search2turn17search15  
- **MUST ограничивать ресурсы входящих запросов** (размер body, таймауты сервера, лимиты параллелизма при необходимости), чтобы защищаться от Unrestricted Resource Consumption. citeturn19view0turn4search1turn22search3  
- **MUST не логировать секреты и чувствительные данные**, и **MUST соблюдать правила редактирования/маскирования**. citeturn5search2turn20search3  
- **MUST следовать безопасным практикам работы с секретами** (не хранить в репозитории; не “вшивать” в контейнер; описывать в `docs/`). citeturn20search3turn11search0  
- **MUST использовать параметризованные запросы и избегать SQL-конкатенаций**. citeturn29search0turn29search4  
- **MUST поддерживать корректный graceful shutdown** через `signal.NotifyContext` и `Server.Shutdown`. citeturn16search1turn16search4  
- **MUST добавлять/обновлять тесты** на изменения поведения и запускать минимум `go test` в рамках предложенных команд проекта (обычно `make test`). citeturn2search2turn9view0  
- **MUST учитывать риски LLM-использования**: prompt injection, insecure output handling, model DoS — особенно если сервис/репо интегрирует LLM в runtime или в CI/CD. citeturn28search0turn28search3

### SHOULD

- **SHOULD держать “source of truth”** в явных артефактах: OpenAPI, схемы, примеры запросов/ответов, список env vars, ADR’ы по ключевым решениям. Это снижает двусмысленность и уменьшает “догадки”. citeturn11search15turn11search0turn21search5  
- **SHOULD предпочитать стандартную библиотеку** для базовых задач (HTTP, logging, errors, context), если нет жёсткой причины добавить зависимость (supply chain). citeturn0search1turn18view0turn6search0  
- **SHOULD использовать RFC 9457 для ошибок HTTP API**, а не custom JSON errors, если сервис предоставляет HTTP API. citeturn24search0  
- **SHOULD документировать любые спорные изменения** (например, введение кэша, изменение consistency модели, добавление сложного middleware) в decision doc, чтобы последующие генерации кода LLM не “ломали” намерения. citeturn12search0turn30view0  
- **SHOULD держать OpenTelemetry семантики и стабильность в уме**: использовать stable/beta части, не завязывать критические интерфейсы на experimental. citeturn31search0turn31search7turn1search4  
- **SHOULD избегать high-cardinality labels** в метриках, и документировать allowed label sets. citeturn31search2turn31search18  
- **SHOULD предлагать кэш только при выполнении условий decision framework** (см. раздел про кэширование). citeturn12search0turn30view1turn13search0

### NEVER

- **NEVER добавлять зависимости ради удобства** без обоснования (производительность/безопасность/операционность) и без фиксации решения. Это прямой supply-chain риск. citeturn5search3turn6search0turn28search0  
- **NEVER вводить кэш “на всякий случай”**: кэш почти всегда добавляет сложность invalidation/consistency и новые классы инцидентов (stale data, stampede, split-brain). citeturn12search0turn13search0turn30view0  
- **NEVER логировать raw request/response целиком** (особенно headers/body) по умолчанию. Логирование должно быть селективным и безопасным. citeturn5search2turn20search3  
- **NEVER использовать строковую конкатенацию для SQL** на пользовательских данных. citeturn29search0turn29search4  
- **NEVER оставлять debug endpoints (например, pprof) открытыми наружу**. Если они есть — они должны быть gated (локалхост, отдельный port, auth, internal network). Сам net/http/pprof подчёркивает side-effect регистрацию handler’ов; это требует повышенной дисциплины. citeturn3search5turn22search9  
- **NEVER полагаться на “молчаливое” принятие данных**: запрещены “нестрогие” JSON-парсеры и бесконтрольные структуры, если это внешняя граница. citeturn16search3turn25search2

## Concrete good / bad examples, anti-patterns и типичные LLM-ошибки

### Контекст и ошибки

**Good: оборачивание ошибок и проверка причины**
```go
if err := repo.Save(ctx, user); err != nil {
    return fmt.Errorf("save user %s: %w", user.ID, err)
}

if errors.Is(err, sql.ErrNoRows) {
    // ...
}
```

Почему это good: `%w` даёт правильную unwrap-цепочку и совместимость с `errors.Is/As`. citeturn15search1turn0search4

**Bad: потеря причины и невозможность errors.Is**
```go
if err := repo.Save(ctx, user); err != nil {
    return fmt.Errorf("save user %s: %v", user.ID, err) // %v вместо %w
}
```
Проблема: причина становится “текстом”, вы теряете программную классификацию ошибок. citeturn0search0turn0search4

### Безопасный HTTP parsing: лимиты и строгий JSON

**Good: MaxBytesReader + DisallowUnknownFields**
```go
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
    r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
    defer r.Body.Close()

    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()

    if err := dec.Decode(dst); err != nil {
        return err
    }
    return nil
}
```
Это напрямую соответствует назначению `MaxBytesReader` как лимитера body и `DisallowUnknownFields` как строгого режима JSON. citeturn19view0turn16search3

**Bad: чтение body без лимита + нестрогий decode**
```go
body, _ := io.ReadAll(r.Body)   // может быть DoS
_ = json.Unmarshal(body, &dst)  // неизвестные поля тихо игнорируются
```
Проблема: неограниченное чтение — класс DoS на ресурсы, а “тихое” игнорирование неизвестных полей ухудшает совместимость и дебаг. citeturn19view0turn22search3turn16search3

### Graceful shutdown

**Good: signal.NotifyContext + Server.Shutdown**
```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

srv := &http.Server{Addr: ":8080", Handler: h}

go func() {
    <-ctx.Done()
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    _ = srv.Shutdown(shutdownCtx)
}()

if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
    return err
}
```
`NotifyContext` отменяет контекст по сигналу, а `Shutdown` делает мягкую остановку активных соединений до дедлайна. citeturn16search1turn16search4

**Bad: убийство процесса/отсутствие Shutdown**
```go
log.Fatal(http.ListenAndServe(":8080", h))
```
Проблема: `log.Fatal` завершает процесс сразу, без graceful shutdown, бросая in-flight запросы. citeturn16search4turn0search5

### Метрики: label кардинальность

**Good: ограниченный набор label values**
```go
httpRequestsTotal.WithLabelValues(method, route, status).Inc()
// route — шаблонный путь, а не raw URL
```

**Bad: high-cardinality labels**
```go
httpRequestsTotal.WithLabelValues(r.URL.Path, userID, requestID).Inc()
```
Проблема: Prometheus практики прямо предупреждают про избыточные/высококардинальные лейблы; такие метрики ломают storage и запросы. citeturn31search2turn31search18

### Типичные LLM hallucinations и как шаблон их должен блокировать

**Hallucinated packages / API symbols**
- LLM может “придумать” пакет, который выглядит правдоподобно (особенно вокруг наблюдаемости, конфигов и “middleware”), но не существует или не подходит по версии. Это описывается как отдельный класс “package hallucinations”. citeturn28search12turn28search1  
Контрмера в шаблоне: правило “NEVER invent dependencies”, плюс явные команды `make lint/test`, плюс pinned deps и docs/contract.

**Неправильные или устаревшие рекомендации**
- Пример: модель может предложить завязаться на OpenTelemetry logs как на стабильный канал (хотя в текущей Go-документации logs signal отмечен как experimental). citeturn31search0turn31search16  
Контрмера: явно прописать в LLM правилах, что OTel logs не default.

**Security blind spots**
- LLM часто “забывает” лимиты ресурсов, таймауты, запрет на логирование секретов, безопасное хранение секретов. Это пересекается с OWASP API Security рисками (Unrestricted Resource Consumption) и практиками безопасного логирования/секретов. citeturn22search3turn5search2turn20search3  
Контрмера: checklist-гейты в PR + обязательные helper-функции в template (decodeJSON с лимитом и строгим парсингом, стандартные timeouts, redaction логов).

**Prompt injection / insecure output handling (если сервис использует LLM в runtime)**
- Если микросервис сам использует LLM, он наследует класс рисков OWASP LLM Top 10 (prompt injection и др.). Даже если сейчас сервис LLM не использует, template должен включать “do not accidentally create an agentic surface” через небезопасное выполнение предложенных моделью команд и т.п. citeturn28search0turn28search3turn21search2

## Review checklist для PR / code review

Этот чеклист ориентирован на review изменений, включая изменения, сгенерированные LLM.

**Корректность и поддерживаемость**
- Код отформатирован (gofmt), читаем, в стиле Effective Go / CodeReviewComments. citeturn15search0turn22search20  
- Ошибки не игнорируются; есть wrap `%w`, `errors.Is/As` применимы. citeturn15search1turn0search4  
- Нет скрытых глобальных состояний; зависимости явны в wiring слое (`internal/app`). citeturn14view0turn0search10  

**Контракты внешних границ**
- HTTP API: OpenAPI обновлён (если применимо), error responses соответствуют RFC 9457. citeturn11search15turn24search0  
- Входные payload’ы: лимит body + строгий JSON + input validation. citeturn19view0turn16search3turn25search2  
- Таймауты/отмена: контекст прошит до БД/внешних вызовов. citeturn17search2turn17search15  

**Безопасность**
- Нет утечек секретов/PII в логах; редактирование/маскирование соблюдено. citeturn5search2turn20search3  
- SQL — параметризованный, без конкатенаций. citeturn29search0turn29search4  
- Ресурсы ограничены (body limits, rate/concurrency ограничения где нужно, таймауты). citeturn22search3turn19view0turn4search1  
- Debug endpoints не экспонированы наружу (pprof gated). citeturn3search5turn22search9  

**Observability**
- Метрики не содержат высококардинальных label’ов; имена и единицы корректны. citeturn31search2turn31search18  
- Трейсы: контекст распространяется (W3C Trace Context), атрибуты не содержат секреты; использование OTel semconv учитывает их stability. citeturn11search2turn31search11turn31search7  
- Логи структурированы (slog), есть корреляция с trace_id (если включены трейсы). citeturn0search5turn24search2  

**CI / supply-chain**
- `go test` (и при необходимости `-race`) проходит; добавлены тесты на изменения поведения. citeturn2search2turn9view0  
- `govulncheck` пройден. citeturn7search12turn7search9  
- Build flags (`-trimpath`, `-buildvcs`) соответствуют политике. citeturn9view1turn9view2  
- По критичным артефактам есть шаги к SLSA/provenance, если это требование организации. citeturn6search0turn5search7  

## Что из результата нужно оформить отдельными файлами в template repo

Ниже — список файлов, которые обеспечивают “без догадок” разработку человеком и LLM. Для каждого файла — цель.

### Root-level инструкции для LLM инструментов

- `AGENTS.md` — общий контракт для Codex-агентов: команды сборки/тестов, структура, стандарты, запреты, workflow. Codex официально читает AGENTS.md перед работой и поддерживает layered overrides. citeturn21search1turn21search3  
- `CLAUDE.md` (или `/.claude/CLAUDE.md`) — проектные инструкции для Claude Code: build/test команды, конвенции, архитектурные решения. citeturn21search8  
- `.github/copilot-instructions.md` — репозиторные инструкции для GitHub Copilot (Chat/Review/Agent). citeturn21search7turn21search4  
- `.cursor/rules/*.mdc` — правила для Cursor (официальная механика rules в `.cursor/rules`). citeturn21search0turn21search9  

Практический совет для template: **держать в этих файлах одинаковое ядро MUST/SHOULD/NEVER**, а специфичность раскладывать “progressive disclosure” по подпапкам (например, rules по глобавм путей). Это снижает контекстный шум и улучшает соблюдение правил. citeturn21search19turn21search8

### Документы стандарта и соглашений

- `docs/engineering/standard.md` — “официальная конституция” сервиса: принципы, язык, структура, ошибки, логирование, observability, безопасность.  
- `docs/engineering/go-style.md` — ссылка/выжимка Effective Go и CodeReviewComments + локальные дополнения. citeturn22search20turn15search0  
- `docs/api/openapi.yaml` (или `/openapi/…`) — контракт API. citeturn11search15turn11search3  
- `docs/api/errors.md` — как сервис использует RFC 9457 (поля, mapping HTTP status ↔ error types). citeturn24search0  
- `docs/ops/runbook.md` — как запускать/диагностировать сервис, какие метрики/логи/трейсы, что делать при инцидентах. (Опора: slog/OTel/Prometheus/Kubernetes probes). citeturn0search5turn31search1turn1search3  
- `docs/decisions/` — ADR/decision records (особенно: storage, очередь, кэш, консистентность, auth). Это ключевое место, которое уменьшает “догадки” LLM. citeturn26view0turn12search0turn30view0  

### Репозиторные конвенции и automation

- `Makefile` или `justfile`: `make test`, `make lint`, `make vuln`, `make build`, `make fmt`, `make run`.  
- `.github/workflows/ci.yml`:
  - gofmt/go vet/`go test`/`govulncheck`; citeturn7search12turn15search0turn9view0  
  - сборка с `-trimpath -buildvcs=auto` (или политика org). citeturn9view1turn9view2  
- `Dockerfile` (multi-stage) и `deploy/` примеры (k8s manifests/helm values) — по необходимости. citeturn20search0turn1search3  

## Исследование подтемы: decision framework — когда кеш нужен, а когда нет

Этот раздел — калиброванная рамка решений, предназначенная для прямой вставки в template как “policy”: LLM должна **предлагать кэш только при выполнении условий**, иначе — явно **сдерживаться**.

### Входные параметры решения о кэше

Решение о кэшировании — это не “оптимизация”, а изменение архитектуры с балансом между latency/cost и complexity/consistency.

Обязательные вопросы перед предложением кэша:

- **Latency goal**: какой SLO/SLA по p95/p99 для конкретного endpoint/job? Если цель уже достигается, кэш не нужен. (Кэш не “улучшает всё” — он добавляет ветвления и ошибки.) citeturn12search0turn30view0  
- **Cost optimization**: есть ли измеримый драйвер стоимости (DB CPU, внешние API calls, egress), который покрывает стоимость кэша? citeturn12search0turn12search3  
- **Hot reads**: есть ли высокая доля повторяющихся чтений (часто одинаковые ключи) — то есть ожидаемая высокая hit-rate? Cache-aside имеет смысл при реальной доле hit’ов. citeturn12search0turn13search18  
- **Data volatility**: как часто данные меняются и сколько допустимо устаревание? Если изменения частые и staleness неприемлем — кэш становится источником багов. citeturn30view0turn13search18  
- **Consistency requirements**: нужна ли строгая консистентность (read-your-writes / linearizability) или достаточно eventual? Если нужна строгая, кэш обычно либо запрещён, либо требует сложных механизмов (инвалидация, write-through, транзакционные паттерны). citeturn12search0turn30view0  
- **Cacheability criteria**: можно ли описать cache key детерминированно, включая всё, что влияет на ответ (версия схемы, auth scope, locale и т.п.)? Для HTTP caching это соответствует требованиям cache key + `Vary`. citeturn30view0turn30view0  
- **Security/privacy**: разрешено ли хранить эти ответы/данные в shared cache? HTTP caching стандарт прямо говорит о `no-store` и ограничениях кэширования authenticated ответов shared cache’ом; также подчёркивает, что `no-store` не является “достаточной” гарантией приватности. citeturn30view1turn30view0

### Когда LLM должна предлагать кэш

LLM **должна** предлагать кэш (как вариант решения), если выполняется большинство условий:

- Есть доказанный bottleneck на чтении (профилирование/метрики) и повторяемые запросы (hot keys), а кэшируемые данные относительно стабильны. citeturn12search0turn13search18  
- Сервис делает дорогие запросы к БД/внешнему API, и cache-aside снижает нагрузку и улучшает латентность при высоком hit rate. Cache-aside описан как наиболее распространённый паттерн: сначала читаем из кэша, при miss идём в БД и заполняем кэш. citeturn12search0turn13search18  
- Допустима controlled staleness (например, TTL 30–300 секунд) и бизнес-логика явно допускает неидеально свежие данные. citeturn13search18turn30view0  
- Есть необходимость “negative caching” (кэшировать 404/5xx на короткий TTL) для защиты origin от перегрузки: CDN-кэши вроде CloudFront явно поддерживают кэширование ошибок с отдельным TTL, и документация подчёркивает, что слишком маленький TTL увеличивает нагрузку на origin, а при 5xx может усугублять проблему. citeturn12search1turn12search4  
- Высокая конкурентность + cache miss/expiry может вызвать stampede; тогда **вместе с кэшем** LLM должна предложить suppression (например, `golang.org/x/sync/singleflight`) для подавления дублирующих вызовов по ключу. citeturn13search0turn12search0  

### Когда кэш запрещён по умолчанию

LLM **должна запрещать кэширования в предложении**, если:

- Данные строго чувствительны (секреты/персональные данные/персонализированные ответы), и нет гарантии, что кэш будет только private-per-user и корректно изолирован. Ошибка в keying приводит к leakage между пользователями. (Даже в HTTP мире различают private/shared caches, и стандарт предупреждает о приватности и ограничениях, а `no-store` не является “достаточным” механизмом.) citeturn30view1turn30view0  
- Read-after-write консистентность критична (например, финансовые операции, права доступа, инварианты), а вы не внедряете сложные протоколы инвалидации/версионирования. citeturn30view0turn12search0  
- Hit rate неизвестен и не измеряется; отсутствуют метрики cache hit/miss, latency, evictions. Если нельзя наблюдать эффект — кэш становится “непрозрачной магией”. citeturn31search1turn31search2  
- Ответы не детерминированны или зависят от множества скрытых факторов (auth scopes, время, A/B флаги), и вы не можете корректно построить cache key. citeturn30view0turn11search15  
- Потребность — “ускорить тяжёлый compute”, но compute детерминирован и лучше решается **precompute/materialization** (см. ниже), либо нужно масштабировать compute, а не кэшировать. citeturn12search3turn30view0

### Precompute vs cache

Шаблонное правило:

- **Cache** — “хранить результат того, что уже посчитали”, на ограниченный TTL/объём, с риском stale data и invalidation. citeturn13search18turn30view0  
- **Precompute/materialize** — “считать заранее” и хранить как новый источник чтения (например, агрегаты/индексы), если:
  - запросы предсказуемы и повторяются “по расписанию/событиям”;  
  - важно стабильное время ответа без TTL-краёв и stampede;  
  - данные меняются, но вы можете обновлять материализацию event-driven или по окнам. citeturn12search3turn13search14

Нормативное решение для template: **по умолчанию предлагать кэш только как временный слой ускорения, а “долгоживущие” ускорения оформлять как precompute/материализацию с отдельным ADR**. citeturn12search0turn30view0

### Типовые сценарии кэширования для API, workers, data-heavy сервисов

**API (read-heavy)**
- Cache-aside для GET-эндпоинтов по ключу ресурса (например, `GET /users/{id}`), если допускается staleness и ключ однозначен. citeturn12search0turn13search18  
- HTTP caching headers (Cache-Control/ETag/Last-Modified) — если сервис отдаёт реально cacheable ответы и вы хотите делегировать caching прокси/CDN; следовать RFC 9111. citeturn30view0turn30view1  
- Negative caching для 404 при высокой частоте miss’ов (например, “проверка существования”), но с коротким TTL и с учётом бизнес-логики появления сущности. citeturn12search4turn12search1  

**Workers / job processing**
- Кэшировать стоит редко: worker обычно работает по очереди событий и делает writes; кэш больше помогает как **локальный “read-through” справочник** (например, настройки, справочники), если эти данные меняются редко и есть версия/TTL. citeturn12search0turn17search15  
- Если worker делает массовые одинаковые вызовы к внешнему API, cache-aside может снизить cost/latency и rate-limit проблемы, но важно учитывать срок годности данных. citeturn12search0turn12search3  

**Data-heavy сервисы**
- Для тяжёлых агрегатов/поиска: кэш оправдан при высокой повторяемости запросов и возможности стабильного keying (хэш параметров запроса). Redis-экосистема описывает query caching как cache-aside с ключом на базе параметров. citeturn13search19turn12search0  
- При риске stampede обязательно добавлять suppression (singleflight) и/или jitter TTL, иначе hot key expiry превращается в avalanche. `singleflight` предоставляет механизм подавления дублирующих вызовов. citeturn13search0turn12search0  

### Нормативные правила для LLM: “кэш-полиси” как алгоритм

Эти правила предназначены для вставки в template как часть LLM instructions.

- **MUST** сначала предложить измерение и проверку гипотезы (метрики latency, DB time, hit rate потенциального кэша), а не сразу архитектурный кэш. citeturn31search1turn12search0  
- **MUST** описать:
  - cache key (включая auth/tenant/locale/version) и почему он корректен; citeturn30view0  
  - TTL и допустимую staleness;  
  - стратегию инвалидации (TTL-only, write-through, event-driven) и её ограничения; citeturn13search18turn12search0  
  - защиту от stampede (singleflight и/или jitter/backoff). citeturn13search0  
- **MUST NOT** предлагать кэширование ответов/данных, если присутствуют персональные/секретные данные без строгой изоляции, или если требуется строгая консистентность. citeturn30view1turn20search3  
- **SHOULD** предпочитать cache-aside как стартовую стратегию, потому что она описана как наиболее распространённая и проста для внедрения, но обязательно фиксировать trade-off consistency. citeturn12search0turn13search18  
- **SHOULD** использовать negative caching только с контролируемыми TTL и с пониманием влияния на origin (слишком маленький TTL увеличивает нагрузку; для 5xx может усугубить инцидент). citeturn12search1turn12search4  
- **NEVER** внедрять кэш без observability (hit/miss, latency, evictions) и без плана rollback. citeturn31search1turn31search2turn30view0