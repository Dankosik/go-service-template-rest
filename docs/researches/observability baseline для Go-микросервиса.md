# Observability baseline для production-ready Go-микросервиса template

## Scope

Этот baseline применим, когда вы делаете **долгоживущий production-сервис** (HTTP/gRPC API, воркеры, периодические джобы) и хотите, чтобы телеметрия была **согласованной по всему стеку**: трассы ↔ метрики ↔ логи, с едиными атрибутами и предсказуемой стоимостью хранения/обработки. Он опирается на то, что сигналы будут собираться и маршрутизироваться через единый телеметрийный протокол и/или коллектор (типичный подход в OpenTelemetry — унифицировать модели и атрибуты для всех сигналов и обогащать их в Collector). citeturn12view0turn14search7turn15view2

Этот baseline **не стоит применять “как есть”**, если:
- у вас **ультранизкая задержка** и вызовы/обновления метрик происходят в “горячих” циклах сотни тысяч раз в секунду — тогда нужно проектировать instrumentations с оглядкой на накладные расходы и кэширование лейблов/handle’ов (в Prometheus-гайдах это прямо выделено как зона внимания). citeturn23view0  
- вы делаете **одноразовый batch/CLI**, который запускается редко и быстро: там чаще уместны “джобные” метрики (время последнего успеха, длительность этапов) и/или push-модель; pull-only-инфраструктура может быть неуместна без отдельного дизайна. citeturn23view0  
- ваш сервис — **публичный интернет-edge**, где вам критично недоверие к входящим trace-заголовкам/контексту: тогда нужно отдельное решение “trust boundary”, санитаризация/дропа контекста для внешних источников и документированная политика пропагации. citeturn26view0

Ключевой принцип: baseline должен быть **boring и battle-tested**, а все “дорогие/спорные” вещи (tail sampling, Baggage с чувствительными идентификаторами, нестабильные семантические конвенции) — либо выключены по умолчанию, либо включаются строго документированным флагом с понятными последствиями. citeturn14search0turn26view0turn23view0

## Recommended defaults для greenfield template

Ниже — нормативный baseline “что должно быть встроено по умолчанию” для репозитория-шаблона.

**Transport / pipeline default: OTLP → Collector (или backend)**  
В template по умолчанию должен быть **OTLP export** для traces+metrics (и опционально logs), потому что OTLP специфицирует единый механизм доставки телеметрии между источниками/коллекторами/бэкендами и имеет стабильный статус для traces/metrics/logs. citeturn14search7turn14search7turn28view1  
Конфигурация endpoint/headers/protocol должна быть через стандартные env-переменные OTEL_EXPORTER_OTLP_* (endpoint, headers, protocol, timeout), а не “зашита” в код. Это снижает “догадки” модели и облегчает эксплуатацию. citeturn28view1

**Resource identity по умолчанию (обязательная “паспортная часть” телеметрии)**  
Template MUST задавать сервисную идентичность через Resource attributes:
- `service.name` (обязательно) — не оставлять `unknown_service`; SDK по умолчанию ставит `unknown_service`, и рекомендуется задавать явно (кодом или `OTEL_SERVICE_NAME`). citeturn16view1turn30view0  
- `deployment.environment.name` (staging/production и т.п.) — задавать через `OTEL_RESOURCE_ATTRIBUTES` и semconv-ключи. citeturn16view1turn16view2turn30view0  
- `service.version` (и/или git sha) — как часть стандарта репозитория (важно для сравнения релизов; semconv/ресурсы описывают общий принцип единых атрибутов источника телеметрии). citeturn12view0turn16view1  

**Propagation default: W3C Trace Context + W3C Baggage**  
По умолчанию propagators должны быть **`tracecontext,baggage`** (это дефолт в спецификации env vars, и Go-пакет propagators прямо указывает поддержку W3C Trace Context и W3C Baggage). citeturn30view0turn24view0turn0search3  
Для HTTP и gRPC template MUST использовать официальные instrumentation библиотеки так, чтобы контекст экстрактился/инжектился автоматически для входящих/исходящих вызовов. citeturn19view0turn18view0turn26view0

**Structured logs default: JSON в stdout + корреляция с trace/span**  
В Go 1.21+ `log/slog` — стандартная structured logging API; JSONHandler даёт line-delimited JSON с ключами/значениями, пригодный для парсинга/фильтрации. Template SHOULD использовать `slog` как дефолтный логгер (не зоопарк логгеров), и выводить в stdout/stderr. citeturn20view1turn20view0  
Корреляция логов с трассами должна быть по `trace_id`/`span_id` (в OpenTelemetry logging spec: включение TraceId/SpanId в LogRecord — базовый механизм точной корреляции; также Resource должен быть единым для всех сигналов). citeturn12view0turn26view0  
Важно: в Go-экосистеме OpenTelemetry logs в SDK на момент актуальной документации помечен как **experimental** (риск breaking changes), поэтому production template по умолчанию SHOULD опираться на `slog` + корреляцию по trace/span, а “OTel Logs API/SDK” — держать как опциональный future-режим (или как отдельный профиль). citeturn25view0turn12view0  

**Security logging baseline: запрет на чувствительные данные и log-injection**  
Template MUST включать правила “данные, которые нельзя логировать” (session ids, access tokens, пароли, connection strings, ключи, платежные данные и т.д.) и санитизацию/экранирование для предотвращения log injection (CR/LF, delimiter’ы). Это прямо перечисляет OWASP Logging Cheat Sheet. citeturn22view0turn21view1

**Metrics baseline: RED + saturation, с дисциплиной кардинальности**  
Для request-driven сервисов baseline метрик должен покрывать “золотые сигналы”: latency/traffic/errors/saturation (Google SRE), а практический минимум на сервис — RED (rate/errors/duration) + saturation на ресурсах/очередях. citeturn2search2turn2search4turn2search1  
На уровне инструментов/конвенций template SHOULD следовать семантическим конвенциям:
- HTTP server duration: `http.server.request.duration` (Histogram, unit `s`) — required в HTTP metrics semconv, с рекомендованными bucket boundaries. citeturn10view1turn8view1  
- HTTP client duration: `http.client.request.duration` (Histogram, unit `s`) — required. citeturn9view1turn8view1  
- (Опционально) `http.server.active_requests`, `http.client.active_requests`, `http.client.open_connections`, `http.client.connection.duration` — полезно для saturation и диагностики, но часть из них имеет стабильность Development/optional. citeturn10view1turn10view2turn10view3  

Для Go runtime baseline SHOULD включать стандартные runtime metrics через `go.opentelemetry.io/contrib/instrumentation/runtime` (он заявлен как реализация “conventional runtime metrics” для OpenTelemetry). citeturn4search1

**Cardinality discipline как обязательное правило стоимости**  
Template MUST запрещать high-cardinality label values в метриках (user id, email, request id, необозримые значения), потому что каждое уникальное сочетание лейблов — новая time series с затратами CPU/RAM/Disk/Query. Это прямые предупреждения в Prometheus naming и instrumentation best practices. citeturn5view2turn23view0turn23view2  
В OpenTelemetry semconv аналогичная идея проводится через требования “low-cardinality” для `http.route` и запрет использовать raw URI path как target для span name. citeturn7view0turn10view1

**Distributed tracing baseline: “всё входящее/исходящее + DB + messaging + jobs”**  
Template MUST включать distributed tracing через OpenTelemetry SDK и официальные instrumentation пакеты:
- HTTP server/client: `otelhttp` — оборачивает handler/transport, создаёт span’ы и обогащает метриками; transport также инжектит span context в outbound headers. citeturn19view0  
- gRPC server/client: `otelgrpc` через `grpc.StatsHandler`/`grpc.WithStatsHandler`. citeturn18view0  
- SQL DB: `otelsql` — инструментация `database/sql` для трасс и метрик; дополнительно можно регистрировать метрики DBStats. citeturn4search2turn17view0  

Span naming и атрибуты MUST соответствовать semconv:
- HTTP span names SHOULD быть `{method} {target}`, где target = `http.route` (server) или `url.template` (client, если доступно и low-cardinality); instrumentation MUST NOT default’ить target на URI path. citeturn7view0turn10view1  
- HTTP status → span status: для 1xx–3xx span status MUST быть unset; для server 4xx — MUST оставаться unset по дефолтным правилам (если нет доп. контекста), для client 4xx — SHOULD быть Error; 5xx — SHOULD быть Error. citeturn7view0  
- Ошибки: если операция завершилась с ошибкой, SHOULD выставлять span status = Error и SHOULD задавать `error.type`; если успешно — `error.type` не ставить. citeturn6search0turn6search2  
- DB spans: span name SHOULD быть `db.query.summary` (если доступно), а если нет — `db.operation.name {target}`; при этом non-parameterized `db.query.text` SHOULD NOT собираться по умолчанию без санитизации (из-за риска чувствительных данных). citeturn5view1turn17view0  

**Request / trace IDs (корреляция)**  
Template должен стандартизировать “что считается корреляционным идентификатором”:
- **Trace identity**: `traceparent` (W3C Trace Context) — основной переносимый идентификатор трассы. citeturn0search3turn26view0turn24view0  
- **Log correlation**: `trace_id`/`span_id` в каждом лог-сообщении в рамках request context (минимум в warn/error; часто — везде). citeturn12view0turn26view0  
- **Request ID**: допускается иметь отдельный `request_id`/`interaction_id` для удобства пользователя/саппорта, но он MUST NOT использоваться как label в метриках (high cardinality). OWASP подчёркивает ценность “interaction identifier” для связывания событий, но отдельно предупреждения по кардинальности дают Prometheus best practices. citeturn22view0turn23view0turn5view2  

image_group{"layout":"carousel","aspect_ratio":"16:9","query":["OpenTelemetry unified collection diagram logs traces metrics collector","OpenTelemetry context propagation traceparent diagram","OpenTelemetry Collector gateway deployment pattern diagram load balancing exporter","OpenTelemetry semantic conventions HTTP spans diagram"]}

## Decision matrix / trade-offs

Ниже — практичная матрица решений для template, где “default” выбирается как boring, а альтернативы показывают стоимость/риски. Факты о стандартах/ограничениях основаны на официальных спецификациях и best practices, особенно по кардинальности, W3C propagation и семантическим конвенциям. citeturn23view0turn5view2turn7view0turn28view1turn12view0turn30view0  

| Область | Default (boring) | Альтернатива | Trade-offs / когда выбрать |
|---|---|---|---|
| Метрики: сбор | OTLP metrics → Collector → backend | `/metrics` (Prometheus pull) напрямую из сервиса | Pull удобен, когда вся платформа уже на Prometheus; OTLP проще унифицирует сигналы и маршрутизацию. В обоих случаях нужно жёстко контролировать label cardinality. citeturn14search7turn23view0 |
| Трейсы: sampling | Head-based: `parentbased_traceidratio` в SDK + опционально tail sampling в Collector | Tail sampling как основной механизм | Tail sampling даёт “сохраняй ошибки/медленные”, но требует, чтобы все spans одного trace попадали в один и тот же collector instance; для этого нужен архитектурный паттерн с traceID-aware балансировкой. citeturn14search0turn15view0turn15view1 |
| Логи: эмиссия | `slog` JSON в stdout + correlation fields | OpenTelemetry Logs API/SDK в приложении | OTel logging модель сильна для унификации/обогащения/корреляции, но в Go logs signal отмечен как experimental; stdout JSON + коллектор (filelog/агент) — наиболее стабильный default. citeturn12view0turn25view0 |
| Корреляция | Trace ID как primary correlation + `trace_id`/`span_id` в логах | Отдельный request_id как primary | Request ID полезен, но легко превращается в high-cardinality метки (запрещено). Trace context стандартизирован (W3C), переносим и интегрирован в пропагацию/инструментации. citeturn0search3turn23view0turn5view2 |
| `http.route` / имена span | Route template (`http.route`) и низкая кардинальность | Raw URL path | OTel semconv прямо запрещает default’ить target на URI path и требует low-cardinality route templates; raw path ломает агрегации и взрывает кардинальность. citeturn7view0turn10view1 |
| Baggage | Очень ограниченно, только безопасные/низкорисковые ключи | Класть user_id/tenant_id повсеместно | Baggage пересылается через границы сервисов, видим в заголовках, может утечь во внешние сервисы, и нет встроенной проверки целостности; использовать только с явной политикой trust boundary и data classification. citeturn26view0turn26view1turn30view0 |
| Семантические конвенции: стабильность | Следовать текущим библиотечным дефолтам + документировать opt-in | Принудительно включать “stable semconv” везде | Для HTTP metrics semconv есть переходный механизм `OTEL_SEMCONV_STABILITY_OPT_IN` (и для некоторых инструментов типа `otelsql` это влияет на набор атрибутов). Принудительное включение может ломать совместимость со старыми дашбордами/алертами; в greenfield чаще можно включать stable, но это должно быть управляемо флагом. citeturn10view1turn17view0 |

## Набор правил в формате MUST / SHOULD / NEVER для LLM

Ниже — “LLM-инструкции” как нормативный слой, чтобы модель не додумывала, а действовала в рамках стандартов.

**Cross-cutting MUST (контекст, идентичность, стандарты)**
- MUST настраивать propagation как W3C Trace Context + W3C Baggage (по умолчанию `tracecontext,baggage`) и не изобретать собственные заголовки для distributed tracing. citeturn30view0turn24view0turn0search3  
- MUST обеспечивать перенос `context.Context` через все слои: handler → service → repo/DB → client calls → goroutines (иначе break trace/log correlation и теряется причинность). Это — базовая модель context propagation в OpenTelemetry. citeturn26view0turn19view0turn18view0  
- MUST задавать `service.name` (через `OTEL_SERVICE_NAME` и/или Resource в коде) и `deployment.environment.name` через `OTEL_RESOURCE_ATTRIBUTES`. Не оставлять `unknown_service`. citeturn16view1turn30view0turn16view2  
- SHOULD использовать semantic conventions (атрибуты/метрики/имена span) вместо произвольных ключей — это уменьшает “локальную диалектность” и повышает переносимость в разных бэкендах. citeturn0search4turn7view0turn10view1  

**Structured logging MUST/NEVER (безопасность и корреляция)**
- MUST логировать структурировано (ключ-значение), предпочтительно JSON (`slog.JSONHandler`). citeturn20view0turn20view1  
- MUST добавлять `trace_id` и `span_id` (из текущего контекста) в логи, как минимум для warn/error, лучше — для всех логов в request scope. Это — основной механизм log ↔ trace correlation в OpenTelemetry. citeturn12view0turn26view0  
- NEVER логировать: session ids, access tokens, пароли, connection strings, ключи шифрования, payment данные; MUST маскировать/хэшировать/санитизировать при необходимости. citeturn22view0turn21view1  
- NEVER логировать “сырой” request/response body по умолчанию; если очень нужно для отладки — только под явным флагом, с redaction и ограничением размера (OWASP относит request body к extended details и отдельно подчёркивает “data to exclude”). citeturn22view0turn21view1  
- MUST защищаться от log injection: санитизировать CR/LF и разделители, корректно кодировать под формат вывода. citeturn22view0turn21view1  

**Metrics MUST/NEVER (RED/USE и кардинальность)**
- MUST иметь метрики, покрывающие минимум RED: rate, errors, duration на уровне сервиса, и saturation на уровне ресурсов/очередей (и ориентироваться на golden signals). citeturn2search4turn2search2turn2search1  
- MUST соблюдать единицы измерения: duration в секундах (s), размер в байтах; Prometheus и OTel semconv подчёркивают base units. citeturn10view1turn5view2turn6search3  
- MUST гарантировать `http.route` как route template (low-cardinality) **если** технически возможно; MUST NOT подменять его URI path. citeturn10view1turn7view0  
- NEVER добавлять в labels/attributes метрик значения с высокой кардинальностью: user_id, email, request_id, UUID, IP и т.п. Это прямой запрет в Prometheus naming practices. citeturn5view2turn23view0  
- SHOULD начинать с минимального набора labels (или без labels) и добавлять измерения только под конкретные вопросы/алерты. citeturn23view0turn23view2  
- NEVER генерировать “динамические имена метрик” (часть имени собирается программно); вместо этого использовать labels (Prometheus прямо запрещает процедурную генерацию частей имени, кроме редких исключений). citeturn23view0  

**Tracing MUST/SHOULD/NEVER (семантика, ошибки, шум)**
- MUST инструментировать HTTP handlers и HTTP clients через `otelhttp` (server handler wrapper и client transport wrapper), чтобы span’ы/метрики/пропагация работали согласованно. citeturn19view0  
- MUST инструментировать gRPC через `otelgrpc` (StatsHandler для server/client). citeturn18view0  
- MUST инструментировать SQL через `otelsql` (traces+metrics), не писать самодельные span’ы вокруг `database/sql`, если можно использовать библиотеку. citeturn17view0turn4search2  
- MUST именовать HTTP span’ы по semconv: `{method} {target}`; для server target = `http.route`; MUST NOT использовать raw URI path как target. citeturn7view0turn10view1  
- MUST записывать ошибки по правилам: если ошибка → span status Error + `error.type`; если нет ошибки → span status unset, `error.type` не задавать. citeturn6search0turn6search2  
- SHOULD использовать head-based sampling в SDK и управлять им через `OTEL_TRACES_SAMPLER`/`OTEL_TRACES_SAMPLER_ARG` (например, `parentbased_traceidratio`), а “всегда семплировать” — только в dev. citeturn30view0turn29view0  
- SHOULD рассматривать tail sampling в Collector, если нужны политики “сохраняй ошибки/медленные”, но ONLY при наличии архитектуры, гарантирующей что все spans одного trace попадут в один collector instance (traceID-aware балансировка). citeturn15view0turn15view1turn14search0  
- NEVER класть в span attributes/metrics attributes секреты/PII “ради удобства”; если нужен идентификатор для разреза — использовать безопасную классификацию/бакеты/хэш с понятными рисками. При использовании Baggage помнить, что оно уходит в headers и может утечь наружу. citeturn26view0turn26view1turn22view0  

## Concrete good / bad examples и ключевые anti-patterns

Ниже — примеры специально “template-формата”: их можно почти напрямую переносить в `docs/` и в `LLM-instruction.md`.

### Good: HTTP handler + logs + трассы/метрики через otelhttp

```go
// HTTP server wiring (boring default):
// - otelhttp.NewHandler: traces + metrics + context propagation
// - request-scoped logger: adds trace_id/span_id into every log record

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	// Business handler:
	mux.Handle("/v1/orders", http.HandlerFunc(s.handleCreateOrder))

	// Wrap with OTel HTTP instrumentation at the edge.
	return otelhttp.NewHandler(mux, "http.server")
}

func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// loggerFromContext should add trace_id/span_id if span context exists.
	log := loggerFromContext(ctx)

	// Example: structured log with stable keys (no PII, no body dumps).
	log.Info("create order request received",
		"http.method", r.Method,
		"http.route", "/v1/orders", // route template, not raw path
	)

	// ... do work, errors should become span errors and logs
	if err := s.app.CreateOrder(ctx); err != nil {
		span := trace.SpanFromContext(ctx)
		span.RecordError(err)
		span.SetStatus(codes.Error, "") // keep description empty or non-sensitive

		log.Error("create order failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
```

Почему это good (нормативно): `otelhttp` предназначен для оборачивания `http.Handler`, создаёт span’ы/метрики, а client transport — инжектит контекст в заголовки; HTTP semconv требует low-cardinality route templates и запрещает использовать raw URI path как target по умолчанию; ошибки должны выставлять status Error и `error.type`. citeturn19view0turn7view0turn10view1turn6search0

### Bad: span name и метрики с raw path / request id → кардинальность и стоимость

```go
// BAD: span name uses raw path with IDs, explodes cardinality
ctx, span := tracer.Start(ctx, "GET "+r.URL.Path) // e.g. "GET /users/123/orders/987"
defer span.End()

// BAD: request_id becomes a metric label → one time series per request
requestsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String("request_id", uuid.NewString())))
```

Почему это bad: OTel HTTP semconv говорит, что span name target не должен default’иться на URI path (он высококардинален), а `http.route` должен быть low-cardinality template; Prometheus best practices прямо запрещают high-cardinality labels вроде user IDs / request IDs. citeturn7view0turn10view1turn5view2turn23view0

### Good: DB instrumentation через otelsql + безопасная семантика

```go
// Boring default: use otelsql.Open + RegisterDBStatsMetrics.
// Add stable DB semantic attributes; avoid logging DSN or query parameters.

attrs := append(
	otelsql.AttributesFromDSN(dsn),
	semconv.DBSystemNamePostgreSQL,
)

db, err := otelsql.Open("pgx", dsn, otelsql.WithAttributes(attrs...))
if err != nil { return err }

reg, err := otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(attrs...))
if err != nil { return err }
defer reg.Unregister()
```

Почему это good: `otelsql` инструментирует `database/sql` и даёт traces+metrics; DB semconv задаёт правила span naming (через `db.query.summary`/`db.operation.name`) и подчёркивает, что non-parameterized `db.query.text` не должно собираться по умолчанию без санитизации (из-за чувствительных данных). citeturn17view0turn5view1

### Bad: DB query text и секреты в логах

```go
// BAD: logs DSN and raw SQL with literals (leaks secrets/PII; also noisy)
log.Error("db error", "dsn", dsn, "sql", "SELECT * FROM users WHERE email='a@b.com'")
```

Почему это bad: OWASP перечисляет connection strings, access tokens, PII и иные секреты как данные, которые нельзя писать “как есть”; DB semconv отдельно говорит, что non-parameterized query text не собирать по умолчанию без санитизации. citeturn22view0turn5view1

### Anti-patterns и типичные ошибки/hallucinations LLM

**Hallucination: “придуманные стандарты” ключей/метрик**  
LLM часто смешивает старые/чужие ключи (`http.method` вместо `http.request.method`, `http.path` вместо `http.route`) или выдумывает “похоже-OTel” имена метрик. Нормативный контроль: всегда сверяться с semantic conventions для HTTP spans/metrics и использовать перечисленные ключи (`http.request.method`, `http.route`, `http.response.status_code`, `error.type` и т.д.). citeturn10view1turn7view0turn6search0  

**Ошибка: “добавим побольше лейблов — будет лучше”**  
Это приводит к тихому росту time series и стоимости. Prometheus прямо предупреждает: кардинальность перемножается по измерениям, и большинство метрик вообще должно быть без labels; отдельная рекомендация — держать кардинальность ниже ~10 как правило большого пальца. citeturn23view0turn23view2  

**Ошибка: “протащим user_id через baggage, чтобы везде видеть пользователя”**  
Документация OpenTelemetry одновременно описывает, что Baggage можно использовать для распространения контекстных данных, но подчёркивает security последствия: оно уходит в заголовках, может попасть в third-party API, нет встроенной проверки целостности, входящий контекст может быть подделан. В template это должно быть либо запрещено, либо строго регламентировано (например, только tenant tier/traffic class, а не персональные идентификаторы). citeturn26view0turn26view1turn30view0  

**Ошибка: “логируем request/response body для отладки”**  
Это обычно превращается в утечки PII/секретов и неконтролируемый объём логов. OWASP явно относит body к extended details и перечисляет категории данных, которые не должны попадать в логи “как есть”. citeturn22view0turn21view1  

**Ошибка: “tail sampling включим на сервисе без изменения архитектуры”**  
Tail sampling требует, чтобы все spans одного trace были обработаны вместе; OpenTelemetry описывает gateway-паттерн и traceID-aware load balancing (двухуровневый Collector), иначе tail sampling становится некорректным/не масштабируется. citeturn15view0turn15view1  

## Review checklist для PR/code review и что вынести в файлы template repo

Этот раздел — почти прямой “PR checklist” для `docs/` и CODEOWNERS/PR template. Он следует best practices по кардинальности, семантическим конвенциям HTTP/DB/errors, безопасному логированию и стандартной конфигурации OTEL_* env vars. citeturn23view0turn5view2turn7view0turn6search0turn22view0turn28view1turn30view0  

**PR / code review checklist (Observability/SRE)**
- Логи:
  - Все новые log statements структурированные (ключ-значение) и проходят политику “data to exclude” (нет токенов/DSN/PII/секретов). citeturn22view0turn21view1  
  - В request scope логи коррелируются с trace (есть `trace_id`/`span_id` хотя бы в warn/error). citeturn12view0turn26view0  
  - Нет “body dump” и нет логов, которые легко превратить в log injection (CR/LF/делимитеры санитизируются или формат вывода безопасен). citeturn22view0turn21view1  

- Метрики:
  - Изменения покрывают минимум RED (rate/errors/duration) и/или golden signals, где применимо. citeturn2search4turn2search2  
  - Нет high-cardinality labels; новый label имеет ограниченный словарь значений и документирован. citeturn5view2turn23view0turn23view2  
  - Duration метрики — в секундах; bucket boundaries не “случайные”, а соответствуют SLO/или рекомендуемым практикам. citeturn6search3turn10view1turn5view2  

- Трейсы:
  - Все новые handlers/clients/DB calls instrumentированы через официальные библиотеки (`otelhttp`, `otelgrpc`, `otelsql`) или ручная инструментация строго следует semconv. citeturn19view0turn18view0turn17view0  
  - HTTP span names и `http.route` используют **route template**, а не raw path; нет утечек высококардинальных значений в имена span/атрибуты. citeturn7view0turn10view1  
  - Ошибки записаны корректно: span status unset при успехе; при ошибке — Error + `error.type`, без чувствительных “description”. citeturn6search0turn6search2turn7view0  

- Конфигурация и эксплуатация:
  - Service identity задана (`OTEL_SERVICE_NAME`, `deployment.environment.name` в `OTEL_RESOURCE_ATTRIBUTES`); не используется `unknown_service`. citeturn16view1turn30view0  
  - OTLP endpoint/headers/протокол настраиваются переменными `OTEL_EXPORTER_OTLP_*`, а не константами. citeturn28view1  
  - Sampling: для prod задан `OTEL_TRACES_SAMPLER` (обычно `parentbased_traceidratio`) и аргумент вероятности; для dev допускается always_on. citeturn30view0turn29view0  
  - Если предлагается tail sampling — PR должен включать архитектурную заметку/diagram, подтверждающую traceID-aware routing (иначе решение неполное). citeturn15view0turn15view1  

**Что из результата оформить отдельными файлами в template repo**

Минимальный набор файлов, который снижает количество “догадок” LLM и превращает baseline в репозиторную норму:

- `docs/observability-baseline.md`  
  Норматив: какие сигналы обязательны, какие env vars обязательны, какие поля/метрики обязательны, какие запрещены (PII/секреты, high cardinality).

- `docs/observability-naming-and-cardinality.md`  
  Единый стандарт именования и “cardinality discipline”: что можно в labels/attributes, что нельзя, примеры “route template vs raw path”, запрет request_id/user_id в метриках. citeturn5view2turn23view0turn7view0turn10view1  

- `docs/observability-instrumentation-patterns.md`  
  Как инструментировать handlers/clients/workers/DB/messaging: ссылочно на `otelhttp`/`otelgrpc`/`otelsql`, правила span naming/errors. citeturn19view0turn18view0turn17view0turn6search0turn7view0  

- `docs/logging-security.md`  
  OWASP-профиль: “data to exclude”, log injection, уровни логов, политика body/log sampling. citeturn22view0turn21view1  

- `docs/sampling-policy.md`  
  Дефолтные значения для dev/staging/prod, как задавать `OTEL_TRACES_SAMPLER(_ARG)`, когда нужен tail sampling и какие архитектурные требования. citeturn30view0turn29view0turn15view0turn15view1  

- `docs/llm/observability.rules.md`  
  Текст для LLM “MUST/SHOULD/NEVER” (раздел выше) + короткий “don’t hallucinate semconv keys” блок.

- `internal/observability/` (код)  
  1) инициализация OTel SDK (TracerProvider/MeterProvider, propagators, resource attributes); 2) `slog` JSON logger + добавление trace/span IDs; 3) middleware/интерцепторы для HTTP/gRPC; 4) обёртки/фабрики для HTTP clients и DB.

- `deploy/local-observability/`  
  Docker Compose/манифесты для локального запуска Collector + минимальный backend (по выбору команды), чтобы разработчик мог клонировать и сразу увидеть traces/metrics/logs end-to-end через OTLP. (Collector-подход и gateway deployment pattern описаны как стандартная практика). citeturn15view0turn14search7turn12view0