# Production-ready template микросервиса на Go: engineering standard и LLM-инструкции

## Область применения и ограничения

Этот подход стоит применять, когда вы делаете **greenfield микросервис** на Go, который будет жить в продакшене, разворачиваться в контейнере и работать в среде оркестрации (на практике чаще всего в entity["organization","Kubernetes","container orchestration"]), с обязательными требованиями по наблюдаемости, безопасности, управляемости и устойчивости к сбоям. Такой шаблон особенно полезен, когда вы сознательно хотите «boring defaults» и низкую стоимость сопровождения: минимально необходимый набор механизмов, встроенный в repo conventions, CI и runtime-поведение. citeturn1search1turn2search7turn0search11turn15search6turn6search0

Подход особенно хорошо работает, если вы **намеренно делаете контракты и ожидания явными**, чтобы LLM-инструменты не «додумывали» (например: фиксированный стиль логов, стандартные таймауты, явная политика retries, схема эндпоинтов /health и /metrics, правила по ошибкам и idempotency). Это встраивает в процесс разработки тот же принцип, что и в промышленных практиках устойчивости: предсказуемость и явные ограничения важнее «умных» догадок. citeturn1search2turn7search0turn1search19turn2search0turn5view1

Не стоит применять этот шаблон «как есть», если:
- это **CLI/утилита**, одноразовый скрипт, или библиотека (шаблон заточен под сервисный runtime, health probes, graceful shutdown, telemetry); citeturn1search1turn2search7  
- это **высоконагруженное streaming/long-polling** или постоянно открытые соединения (SSE/WebSockets/HTTP streaming): стандартные server-level `WriteTimeout`/`ReadTimeout` могут ломать легитимные длительные ответы, поэтому нужна иная политика таймаутов и shutdown для «hijacked connections». citeturn4view0turn4view1turn0search21  
- у вас **не microservice**, а модульная монолитная система или «платформенный сервис», где стандарты должны быть на уровне более крупного архитектурного каркаса (service mesh, единый ingress/gateway, централизованные политики retries/limits), и дублировать их на уровне каждого сервиса может быть либо вредно, либо избыточно. citeturn1search15turn15search1turn7search1  
- продукт сознательно выбирает иной набор сигналов наблюдаемости (например, только metrics без traces) — тогда часть OTel-интеграций можно/нужно вырезать, но принципы (структурные логи, стабильные метрики, ограничения кардинальности, отсутствие PII) остаются. citeturn6search17turn1search0turn6search2

## Рекомендованные defaults для greenfield template

Ниже — набор «обязательных по умолчанию» решений для production-ready Go-сервиса, рассчитанный так, чтобы репозиторий можно было склонировать и сразу писать код, а LLM-генерация была максимально детерминированной.

**Версии и воспроизводимость toolchain**
- Использовать актуальный стабильный Go как baseline. На дату 28 февраля 2026 релиз — **Go 1.26**. citeturn10search0turn10search4  
- В `go.mod` фиксировать `go` directive и (рекомендуемо) `toolchain` directive, чтобы снизить расхождения среды разработчиков/CI и уменьшить шанс «устаревшего» кода от LLM. `go` directive задаёт минимальную версию Go и влияет на семантику сборки; начиная с Go 1.21 toolchain отказывается работать с модулем, если версия слишком старая. `toolchain` позволяет явно рекомендовать конкретный Go toolchain для main module. citeturn12view1turn12view0turn16search0  
- Понимать и документировать стандартные механизмы цепочки модулей: модульный прокси и checksum database (go.sum) — часть модели доверия и воспроизводимости зависимостей. citeturn2search9turn11view0turn2search17  

**Структура репозитория**
- Базовая структура модуля — из официального гайда по layout: `internal/` для приватной логики и `package main` в точке входа команды; правило импортов для `internal/` обеспечивается инструментарием Go. citeturn14search0turn14search7turn3view1  
- Практический default для микросервиса: один бинарь `cmd/service/` + приватные пакеты в `internal/` (handlers, service, storage, clients, observability, config). Это минимизирует публичную API-поверхность внутри репо и уменьшает «расползание» импорта. citeturn14search7turn14search0turn2search5  

**Набор обязательных runtime-механизмов устойчивости (resilience)**
- **Timeouts везде**:  
  - На входе (HTTP server) выставлять server-level таймауты и отдельный пер-запросный deadline/budget. `ReadHeaderTimeout` предпочтителен для защиты от медленных/вредоносных клиентов, потому что позволяет обработчику самому решать тайм-аут на body; `ReadTimeout/WriteTimeout` не дают per-request решений. citeturn4view1turn3view0turn0search21  
  - На выходе (HTTP client) — запрещать `http.DefaultClient` без таймаута: `Client.Timeout` задаёт общий лимит, включает установление соединения, редиректы и чтение body; нулевой таймаут означает «без таймаута». Клиенты должны переиспользоваться и безопасны для конкурентного использования. citeturn5view1turn5view3  
  - Для транспортного слоя — настроить `Transport` (например, `TLSHandshakeTimeout`, `ResponseHeaderTimeout`, `IdleConnTimeout`) и при необходимости ограничивать общий параллелизм к хосту через `MaxConnsPerHost` (который блокирует dial при превышении лимита). citeturn5view3turn5view2  
  - Во всех внутренних API обязательно прокидывать `context.Context` как первый параметр, чтобы cancellation и deadlines гарантированно проходили через слои. citeturn3view1turn2search4

- **Retries по умолчанию “OFF”, кроме безопасных случаев**:  
  - Google SRE прямо предупреждает: бесконтрольные ретраи усиливают каскадные отказы; нужны ограничения “per-request” и “server-wide retry budget”. citeturn1search2turn7search5  
  - Ретрай должен быть ограничен суммарным time budget запроса, иметь capped exponential backoff и jitter (практика entity["company","AWS","cloud provider"] и их guidance по jitter). citeturn7search0turn7search8  
  - Обязательно учитывать retry-storm анти-паттерн и ограничивать число попыток/длительность; типовая ошибка — `while(true)`/бесконечный цикл. citeturn1search19turn1search11  

- **Rate limiting и ресурсные лимиты — обязательны**: отсутствие лимитов — частая причина DoS/истощения ресурсов; OWASP отдельно выделяет риски неограниченного потребления. citeturn13search7turn13search3  
  - Для простого и idiomatic Go — `golang.org/x/time/rate` (token bucket). citeturn6search3turn6search7  

- **Backpressure + load shedding без overengineering**:  
  - При перегрузке лучше «сбрасывать» часть нагрузки и деградировать функциональность, чем упасть глобально: SRE описывает load shedding и graceful degradation как способ удерживать систему от исчерпания RAM/роста latency/падения health checks. citeturn7search5turn7search1  
  - Default-механика для шаблона: ограничение concurrency (общего и по зависимости), bounded очереди, быстрый отказ 429/503 в перегрузе вместо накопления очередей и таймаутов «в хвосте». citeturn7search5turn1search3turn6search3  

- **Circuit breaker и bulkheads — как “plug-in”, включать осознанно**:  
  - Bulkhead pattern полезен для изоляции ресурсов и предотвращения взаимного заражения (например, отдельные лимиты для БД/внешнего API/кэша). citeturn1search3turn5view2  
  - Circuit breaker нужен, когда операция «скорее всего будет падать» и вы хотите быстро прекращать бесполезные попытки, сочетая его с ограниченными retry. citeturn1search7turn1search19  
  - Но: circuit breaker в коде может дублировать или конфликтовать с mesh/gateway (если они применяют свои политики). Поэтому в template лучше заложить интерфейсы/обёртки и метрики, но не навязывать сложные open/half-open машины каждому сервису без причины. citeturn1search7turn15search1turn1search15  

image_group{"layout":"carousel","aspect_ratio":"16:9","query":["circuit breaker pattern diagram microservices","bulkhead pattern diagram microservices","load shedding graceful degradation diagram","canary vs blue green deployment diagram"],"num_per_query":1}

**Graceful shutdown и “disposability”**
- Реализовать корректный shutdown: `http.Server.Shutdown(ctx)` закрывает listeners, закрывает idle connections и ждёт активные соединения до завершения; если контекст истёк — возвращает ошибку контекста; сервер нельзя повторно использовать после Shutdown. citeturn4view0turn1search1  
- Для среды entity["organization","Kubernetes","container orchestration"] учитывать lifecycle: `preStop` выполняется синхронно до посылки SIGTERM, и суммарное время `preStop` + остановка контейнера ограничено `terminationGracePeriodSeconds`; «зависший» preStop удерживает Pod в Terminating до убийства. citeturn0search3turn0search11  
- С точки зрения 12-factor, быстрый старт и корректная остановка повышают надёжность и качество деплоев. citeturn1search1turn0search11  

**Наблюдаемость (logs/metrics/traces)**
- Structured logging: использовать `log/slog` как стандартную библиотеку structured logging; это снижает зависимость от сторонних логгеров и упрощает единый формат. citeturn6search0turn6search4  
- Security logging: логирование должно учитывать безопасность, не писать секреты/PII, быть пригодным для расследований. citeturn1search0turn1search8turn6search2  
- Metrics:  
  - Использовать официальный Go-клиент entity["organization","Prometheus","monitoring system"] (`client_golang`), экспонировать `/metrics`. citeturn15search3turn15search15turn15search11  
  - Следовать правилам именования и ограничивать cardinality: Prometheus прямо предупреждает не «перелабеливать» метрики; большинство метрик должны быть без labels, а высокую кардинальность надо избегать. citeturn6search17turn8search2  
- Tracing:  
  - Использовать entity["organization","OpenTelemetry","observability standard"] как vendor-neutral стандарт; это проект entity["organization","CNCF","cloud native foundation"], созданный для стандартизации инструментирования и доставки telemetry (traces/metrics/logs). citeturn15search6turn15search10turn15search14  
  - Инструментировать `net/http` через `otelhttp`-обёртки для server/client. citeturn16search3turn0search2  
  - Использовать semantic conventions (например, для HTTP) чтобы атрибуты и метрики были совместимы между сервисами и backend’ами. citeturn15search4turn15search0  
  - Для конфигурации exporter’ов опираться на стандартизованные env vars спецификации и OTLP exporter env-настройки. citeturn16search5turn16search1  
- Collector: рассматривать entity["organization","OpenTelemetry","observability standard"] Collector как стандартный способ приёма/обработки/экспорта telemetry и описать рекомендуемые deployment patterns (agent/sidecar/gateway) и ограничения. citeturn15search5turn15search1  

**Безопасность по умолчанию**
- TLS: базовые требования и безопасные defaults описаны в OWASP TLS cheat sheet; шаблон должен предполагать TLS на границе и возможность mTLS внутри, но не «прятать» отсутствие TLS за “dev convenience”. citeturn13search0  
- Error handling: наружу — безопасные, неразглашающие детали ошибки ответы; детали — в логах. Это напрямую рекомендует OWASP как целевую модель глобальной обработки ошибок. citeturn13search1  
- Secrets: секреты не должны попадать в repo/логи; политики хранения/ротации/аудита — по OWASP Secrets Management. citeturn1search8turn1search0  

**Качество кода и security gates в CI**
- Форматирование — `gofmt` (единый стиль, без «религиозных» споров). citeturn9search2turn9search17  
- Статический анализ — `go vet` как базовая проверка «подозрительных конструкций». citeturn9search1turn9search4  
- Тесты — `go test` как стандартный способ запуска unit/integration tests. citeturn9search0  
- Проверка уязвимостей зависимостей — `govulncheck` как официальная low-noise проверка, умеющая выявлять реально достижимые уязвимости через анализ вызовов. citeturn2search2turn2search6turn2search14  
- Управление зависимостями — `go mod tidy` как часть hygiene; фиксировать ожидания в CI (например, `-diff` или проверка отсутствия изменений). citeturn12view2turn11view0  

**Progressive delivery и эволюция**
- Базовый механизм обновления в entity["organization","Kubernetes","container orchestration"] — rolling updates (incremental replacement). citeturn16search2turn16search6  
- Для canary/blue-green как стандартизируемого механизма progressive delivery — entity["organization","Argo Rollouts","kubernetes progressive delivery"]. citeturn7search10turn7search3turn7search21  
- Для постепенной эволюции функциональности — feature flags через entity["organization","OpenFeature","feature flag standard"] как vendor-agnostic спецификацию; проект относится к entity["organization","CNCF","cloud native foundation"] ecosystem. citeturn7search2turn7search6  
- Для миграции от legacy/монолита — Strangler Fig pattern (как у entity["people","Martin Fowler","software engineer"]) и его индустриальные интерпретации (включая guidance от AWS). citeturn8search0turn8search20  

## Матрица решений и trade-offs

Ниже — матрица ключевых решений, которые лучше **зафиксировать в template** (как defaults) и **описать в docs**, чтобы LLM не «переизобретала» архитектуру под каждый PR.

| Область | Варианты | Риски/выгоды | Бордерлайн / когда менять | Default для template |
|---|---|---|---|---|
| Transport API | HTTP/JSON vs gRPC | gRPC поощряет deadlines/отмену и контрактность; HTTP проще и меньше tooling. | gRPC если нужна строгая схема, low-latency RPC, много внутренних вызовов. | HTTP как минимум; gRPC как “доп. трек” с явным документом про deadlines. |
| Таймауты inbound | Только server-level vs + per-request budget | Только server-level не даёт per-request решений; но server-level нужны против slowloris и hung clients. | Streaming endpoints требуют иной политики (часто `WriteTimeout=0`). | Server-level + per-request budget. |
| Retries | Retry everywhere vs retry only safe/idempotent | Retry storms и каскадные отказы при «повсеместных» ретраях. | Допускается ограниченно для идемпотентных операций и transient ошибок. | Retry по умолчанию “OFF”, точечно “ON”. |
| Backoff/jitter | Fixed delay vs exponential + jitter | Без jitter возможна синхронизация повторов (“thundering herd”). | Всегда, где есть ретраи. | Capped exponential backoff + jitter. |
| Retry budget | Нет vs есть (глобальный/на dependency) | Без budget ретраи съедают capacity и усиливают деградацию. | Почти всегда при включённых ретраях. | Есть budget (простая реализация). |
| Circuit breaker | В коде vs в mesh/gateway vs отсутствует | Circuit breaker может конфликтовать с другими слоями, но помогает fail-fast. | Нужен при persistently failing dependency и дорогих попытках. | Интерфейс+обёртка в коде, включение — конфигом. |
| Bulkheads | Нет vs per-dependency concurrency limits | Общий pool может «заразиться» одной зависимостью. | Почти всегда полезно для внешних зависимостей. | Лимиты per dependency (простые). |
| Rate limiting | Только на edge vs и в сервисе | Только edge не защищает от внутренних «петлей» и ошибок клиентов внутри сети. | Для internal сервисов — хотя бы глобальный лимит и лимиты на дорогие операции. | Middleware на сервисе + поддержка edge. |
| Observability | Только logs vs logs+metrics vs +tracing | Без metrics сложно SLO/alerting; tracing критичен в распределённых запросах. | Tracing может быть выключаемым, но дизайн должен поддерживать. | logs+metrics обязательно, tracing включаемый env-конфигом. |
| Feature flags | Vendor SDK vs OpenFeature | Vendor SDK lock-in; OpenFeature — нейтральная абстракция. | Если уже есть единый корпоративный стандарт. | OpenFeature-совместимый интерфейс. |

Обоснования ключевых trade-offs опираются на практики: timeouts/клиенты и семантика таймаутов в net/http citeturn4view1turn5view1turn5view2, ретраи/бюджеты/каскадные отказы citeturn1search2turn7search0turn1search19, circuit breaker/bulkhead как паттерны отказоустойчивости citeturn1search7turn1search3, а также требования к rate limiting и ограничению ресурсов в OWASP API Security citeturn13search7turn13search3. Progressive delivery и эволюция — через Argo Rollouts, OpenFeature и Strangler Fig. citeturn7search10turn7search2turn8search0  

## Правила для LLM в формате MUST / SHOULD / NEVER

Цель этих правил — сделать генерацию кода **идиоматичной, безопасной и предсказуемой**, а также минимизировать «догадки». Формулировки ниже рассчитаны на копирование в `docs/llm/` и на использование как “system prompt”/repo instructions.

**MUST**
- MUST прокидывать `context.Context` как **первый аргумент** во все публичные методы, которые делают I/O или могут блокироваться; MUST использовать `r.Context()` как базовый контекст внутри HTTP handlers; MUST обеспечивать, что горутины завершаются при отмене контекста/таймауте. citeturn3view1turn2search4turn4view0  
- MUST вызывать `cancel()` для `context.WithTimeout/WithCancel/...` на всех путях выполнения (иначе утечки). citeturn3view1turn2search0  
- MUST задавать таймауты для всех исходящих HTTP вызовов через `http.Client.Timeout` и/или `Transport`-таймауты; MUST переиспользовать `http.Client`, не создавать его на каждый запрос. citeturn5view1turn5view2  
- MUST выставлять server-level timeouts (минимум `ReadHeaderTimeout`) и ограничивать размере/время чтения входящих запросов; MUST учитывать ограничения server-level `ReadTimeout/WriteTimeout` (не дают per-request решений). citeturn4view1turn3view0turn0search21  
- MUST реализовать graceful shutdown через `Server.Shutdown(ctx)` и гарантировать, что процесс ждёт завершения Shutdown перед выходом; MUST корректно обрабатывать долгоживущие hijacked connections отдельно. citeturn4view0turn0search11  
- MUST избегать бесконечных retries; MUST ограничивать retries по количеству попыток и по общему времени запроса; MUST использовать capped exponential backoff + jitter и соблюдать retry budget. citeturn1search2turn7search0turn1search19  
- MUST избегать утечек чувствительных данных: не логировать секреты/PII; не класть их в distributed baggage/trace context. citeturn1search0turn1search8turn6search2  
- MUST использовать `gofmt`-совместимый формат и стандартные go tools в CI (`go fmt`, `go vet`, `go test`, `govulncheck`). citeturn9search17turn9search1turn9search0turn2search6  
- MUST оборачивать ошибки с контекстом через `%w`, чтобы сохранять цепочку причин и позволять `errors.Is/As`. citeturn0search16  
- MUST соблюдать правила `internal/` и не расширять публичную API-поверхность модуля без явной причины. citeturn14search7turn14search0  

**SHOULD**
- SHOULD использовать `log/slog` для структурных логов и единые ключи (trace_id/span_id, request_id, service, env), чтобы логи хорошо индексировались. citeturn6search0turn6search4turn1search1  
- SHOULD инструментировать HTTP server/client через `otelhttp`, а telemetry настраивать env vars согласно OTel спецификациям. citeturn16search3turn16search5turn16search1  
- SHOULD публиковать метрики Prometheus, соблюдая правила naming и ограничения cardinality; SHOULD избегать labels с user_id/email/uuid и любых «неограниченных» значений. citeturn6search17turn8search2  
- SHOULD реализовывать rate limiting на уровне сервиса (минимум глобальный, лучше — на дорогие эндпоинты/операции), используя token bucket limiter. citeturn13search7turn6search3  
- SHOULD иметь простую реализацию bulkheads: отдельные лимиты concurrency для разных dependency-клиентов (DB, external HTTP, cache). citeturn1search3turn5view2  
- SHOULD описывать progressive delivery: rolling update как baseline и (если нужно) canary/blue-green через Argo Rollouts; изменения поведения через feature flags (OpenFeature). citeturn16search2turn7search10turn7search2  

**NEVER**
- NEVER использовать `http.Get()`/`http.DefaultClient` без явных таймаутов в production-коде. citeturn5view1  
- NEVER делать “retry by default” для неидемпотентных операций (создание заказа/платёж/команда без idempotency key) и NEVER ретраить бесконечно. citeturn1search19turn7search0turn1search2  
- NEVER хранить `context.Context` в struct и NEVER прокидывать `nil` context. citeturn3view1  
- NEVER логировать секреты/ключи/токены, и NEVER помещать чувствительные данные в baggage. citeturn1search8turn6search2turn1search0  
- NEVER добавлять метрики с неограниченной кардинальностью labels. citeturn6search17turn8search2  
- NEVER возвращать пользователю «сырые» внутренние ошибки/stack traces; детали должны уходить в логи. citeturn13search1turn13search12  

## Concrete good / bad examples

Ниже примеры специально подобраны так, чтобы их можно было почти напрямую переносить в `docs/` и в код template.

**Пример: таймаут на исходящий HTTP вызов + правильное использование context**

```go
// GOOD: общий бюджет задаётся контекстом, client переиспользуется, timeout задан.
type FooClient struct {
	http *http.Client
	base string
}

func NewFooClient(base string) *FooClient {
	return &FooClient{
		base: base,
		http: &http.Client{
			Timeout: 3 * time.Second, // hard cap на всю операцию
			Transport: &http.Transport{
				ResponseHeaderTimeout: 2 * time.Second,
				TLSHandshakeTimeout:   5 * time.Second,
				IdleConnTimeout:       90 * time.Second,
				MaxConnsPerHost:       64, // bulkhead на уровне коннектов
			},
		},
	}
}

func (c *FooClient) GetFoo(ctx context.Context, id string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/foo/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
```

`http.Client.Timeout` задаёт общий лимит и нулевое значение означает отсутствие таймаута; клиенты нужно переиспользовать и они потокобезопасны. citeturn5view1turn5view2

```go
// BAD: без таймаутов и без контекста -> зависания, утечки goroutines, cascading failures.
func GetFooBad(id string) ([]byte, error) {
	resp, err := http.Get("https://example.com/foo/" + id)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
```

**Пример: контекст и отмена — избежать утечек CancelFunc**

```go
// GOOD: cancel вызывается всегда.
func (s *Service) Handle(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return s.dep.Do(ctx)
}
```

Контекстные функции возвращают CancelFunc; если его не вызвать, утечки сохраняются до отмены родителя; `go vet` умеет проверять корректность использования cancel. citeturn3view1turn9search1

```go
// BAD: cancel никогда не вызывается (утечки таймеров/дочерних контекстов).
func (s *Service) Handle(ctx context.Context) error {
	ctx, _ = context.WithTimeout(ctx, 2*time.Second)
	return s.dep.Do(ctx)
}
```

**Пример: server timeouts + graceful shutdown**

```go
// GOOD: ReadHeaderTimeout + Shutdown с контекстом.
srv := &http.Server{
	Addr:              cfg.HTTPAddr,
	Handler:           handler,
	ReadHeaderTimeout: 5 * time.Second,
	ReadTimeout:       30 * time.Second,
	WriteTimeout:      30 * time.Second,
	IdleTimeout:       60 * time.Second,
}

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

`ReadHeaderTimeout/ReadTimeout/WriteTimeout/IdleTimeout` имеют чёткую семантику; `Shutdown` закрывает listeners, закрывает idle connections и ждёт активные до завершения либо до таймаута контекста; hijacked connections надо закрывать отдельно. citeturn4view1turn4view0turn0search21

**Пример: безопасные retries с бюджетом и jitter (псевдокод политики)**

```go
// GOOD (policy sketch):
// - retries только для идемпотентных операций и transient ошибок
// - maxAttempts = 3
// - capped exponential backoff + jitter
// - общий time budget ограничен контекстом
// - retry budget (например, 10% от успешных запросов за окно)
```

SRE рекомендует ограничивать ретраи и использовать retry budget, иначе ретраи могут превратить локальную проблему в глобальный каскадный отказ. Amazon guidance описывает exponential backoff, cap и jitter как практику для устойчивых клиентских библиотек. citeturn1search2turn7search0turn7search8

```go
// BAD: бесконечный ретрай без задержки и без бюджета.
for {
	if err := call(); err == nil {
		break
	}
}
```

Azure описывает retry storm как анти-паттерн; бесконечные ретраи ухудшают ситуацию и могут удерживать систему в деградации. citeturn1search19turn7search5

**Пример: метрики Prometheus — низкая кардинальность**

```go
// GOOD: ограниченные labels: method, route, status.
httpRequestsTotal.WithLabelValues(r.Method, routePattern, strconv.Itoa(status)).Inc()
```

Prometheus рекомендует не злоупотреблять labels и держать cardinality низкой; также есть guidance по именованию метрик и labels. citeturn6search17turn8search2

```go
// BAD: user_id в label => взрыв кардинальности.
httpRequestsTotal.WithLabelValues(userID).Inc()
```

## Анти‑паттерны и типичные ошибки/hallucinations LLM

Ниже — ошибки, которые LLM систематически «галлюцинируют» в Go сервисах, и почему это опасно именно для микросервисной устойчивости.

Самая частая группа — **неявные бесконечности**: отсутствие таймаутов на HTTP клиентах, нулевые таймауты по умолчанию, бесконечные ретраи, бесконечные очереди/каналы. Эти дефекты обычно не видны на dev окружении, но в продакшене превращаются в медленные каскадные отказы: запросы копятся, горутины висят, соединения не освобождаются. net/http прямо фиксирует: `Client.Timeout=0` значит «без таймаута», а transport и server поля с нулём/отрицательными значениями часто означают «нет таймаута». citeturn5view1turn4view1turn7search5

Вторая группа — **контекст и утечки**: LLM часто забывает `defer cancel()`, прячет `context.Context` в struct, или создаёт `context.Background()` внутри handler’а вместо `r.Context()`. Это ломает отмену запросов, противоречит правилам пакета context и приводит к утечкам ресурсов до тех пор, пока родительский контекст не будет отменён. citeturn3view1turn2search4

Третья группа — **retry storm в чистом виде**: LLM склонна «чинить» transient ошибки увеличением retries, но редко добавляет jitter, cap, retry budget и ограничения по общему времени. SRE и Azure отдельно выделяют ретраи как источник каскадных отказов и рекомендуют строгие лимиты. citeturn1search2turn7search0turn1search19

Четвёртая группа — **метрики с высокой кардинальностью** и «случайные labels»: user_id/email/request_id/uuid в labels, динамические route строки, raw error messages в labels. Prometheus предупреждает, что каждая уникальная комбинация labels — отдельная time series со стоимостью RAM/CPU/disk/network. citeturn6search17turn8search2

Пятая группа — **утечки секретов и PII** через логи/trace/baggage: LLM может логировать весь request/headers, добавлять токены в поля, или класть чувствительные параметры в baggage. OWASP требует осторожного security logging и корректного secrets management; OpenTelemetry предупреждает не класть чувствительные данные в baggage из‑за распространения через границы сервисов. citeturn1search0turn1search8turn6search2

Шестая группа — **неправильный shutdown**: `os.Exit` в ошибке, `log.Fatal` из глубины библиотечного кода, отсутствие `Server.Shutdown`, игнорирование особенностей hijacked connections. Это приводит к потере запросов и проблемам при rolling update. net/http документирует семантику Shutdown и ограничения. citeturn4view0turn16search2

Седьмая группа — **“cargo cult” паттерны на каждый чих**: circuit breaker везде, сложные state machines без метрик, самописные фреймворки retries вместо минимальной политики с прозрачными параметрами. Azure patterns подчёркивают, что паттерны комбинируются осознанно (bulkhead+retry+circuit breaker+throttling), а не автоматически. citeturn1search3turn1search7turn1search11

## Review checklist для PR/code review

Этот чек-лист предназначен для PR template и для внутренних engineering review — с фокусом на устойчивость и на типовые failure modes.

Проверить контракты и контекст:
- Все публичные функции, делающие I/O или потенциально блокирующиеся, принимают `context.Context` первым аргументом; в HTTP handlers используется `r.Context()`. citeturn3view1turn2search4  
- Все `context.WithTimeout/WithCancel/...` имеют `defer cancel()` на всех путях. citeturn3view1  

Проверить таймауты и ресурсные лимиты:
- HTTP server имеет `ReadHeaderTimeout` и согласованную политику `ReadTimeout/WriteTimeout/IdleTimeout`; учтены streaming кейсы. citeturn4view1turn0search21  
- HTTP clients переиспользуются, имеют `Client.Timeout` и при необходимости `Transport`-таймауты; `MaxConnsPerHost`/лимиты concurrency применены для bulkhead. citeturn5view1turn5view2turn1search3  
- Есть механика backpressure/load shedding: bounded очереди/лимиты concurrency/быстрый отказ при перегрузе, а не накопление. citeturn7search5turn7search1  

Проверить retries/circuit breaker:
- Retries применяются только к идемпотентным операциям и transient ошибкам; есть cap/jitter; retries ограничены общим budget запроса; есть retry budget. citeturn1search2turn7search0turn7search8  
- Нет “retry storm” поведения (бесконечные циклы, агрессивные повторы, синхронизация повторов). citeturn1search19turn7search14  
- Circuit breaker (если включён) имеет метрики/логирование и не конфликтует с mesh/gateway политиками. citeturn1search7turn15search1  

Проверить graceful shutdown и readiness:
- Реализован `Server.Shutdown(ctx)` и корректная обработка SIGTERM; приложение не выходит, пока Shutdown не завершился; учтены hijacked connections. citeturn4view0turn0search11  
- Readiness/Liveness/Startup probes соответствуют семантике entity["organization","Kubernetes","container orchestration"], нет «вечной readiness=true при недоступных зависимостях» или «liveness=проверить базу данных» без причины. citeturn2search7turn2search3  

Проверить наблюдаемость:
- Логи структурные (`slog`), без секретов/PII, пригодны для расследований. citeturn6search0turn1search0turn1search8  
- Метрики Prometheus соблюдают naming и cardinality; нет labels с неограниченными значениями. citeturn6search17turn8search2  
- Tracing (если включено) использует OTel, `otelhttp`, стандартные env vars конфигурации и semantic conventions. citeturn15search6turn16search3turn16search5turn15search4  

Проверить security hygiene:
- Error handling: наружу — безопасные сообщения, внутрь — детали в логах; нет утечек внутренних деталей. citeturn13search1turn13search12  
- Secrets не хардкодятся и не попадают в логи; есть документированный способ передачи секретов (env/secret store). citeturn1search8turn1search0  
- Есть минимальные security gates: `govulncheck` в CI, а также `go vet`/tests/format. citeturn2search6turn9search1turn9search2  

## Что из результата нужно оформить отдельными файлами в template repo

Ниже — структура конкретных файлов, которые стоит положить в репозиторий-шаблон так, чтобы это было «нормативно» и напрямую поддерживало LLM-генерацию.

**В корне репозитория**
- `README.md`: что за сервис, как запустить локально, минимальные SLO/ожидания, описание `/live`, `/ready`, `/metrics`, базовый troubleshooting. citeturn2search7turn15search15turn1search1  
- `CONTRIBUTING.md`: правила кода (gofmt, go vet, go test), требования к PR, минимальные тесты, стиль ошибок (`%w`), политика контекстов. citeturn9search2turn9search1turn9search0turn0search16turn3view1  
- `SECURITY.md`: политика секретов, логирования, TLS ожидания, и обязательный `govulncheck`/dependency hygiene. citeturn1search8turn1search0turn13search0turn2search6  
- `go.mod`: `go 1.26` + `toolchain go1.26.0` (или актуальная patch), чтобы унифицировать toolchain. citeturn10search0turn12view1turn12view0turn16search0  
- `Makefile` (или `taskfile.yml`): `fmt`, `vet`, `test`, `lint` (опционально), `govulncheck`, `run`, `docker-build`. Поддержать “one command” для CI локально. Действия `go fmt`/`gofmt` документированы как стандартные инструменты. citeturn9search17turn9search2turn9search1turn2search6  

**CI/CD**
- `.github/workflows/ci.yml` (или аналог):  
  - шаги: `go fmt`/проверка, `go vet`, `go test`, `govulncheck`, `go mod tidy -diff`/эквивалент. citeturn9search17turn9search1turn9search0turn2search6turn12view2  
- `PULL_REQUEST_TEMPLATE.md`: встроить чек-лист из раздела выше (timeouts/retries/metrics/security). citeturn1search2turn6search17turn13search1  

**docs/**
- `docs/engineering-standard.md`: конвенции репозитория, структура модулей (`internal/`), правила контекстов, стиль ошибок, базовые quality gates. citeturn14search7turn3view1turn0search16turn0search0  
- `docs/resilience.md`: нормативные правила timeouts/retries/jitter/budget, bulkheads, circuit breakers, rate limiting, backpressure, load shedding, graceful degradation; типовые failure modes и запреты (retry storm). citeturn1search2turn7search0turn1search19turn7search5turn1search3turn1search7turn13search7  
- `docs/observability.md`: стандарты логов (`slog`), метрик (Prometheus naming/cardinality), tracing (OTel + semantic conventions), конфигурация exporter’ов env vars, рекомендации по Collector deployment patterns. citeturn6search0turn6search17turn8search2turn15search6turn15search4turn16search5turn15search1  
- `docs/kubernetes-runtime.md`: probes семантика, graceful shutdown, `preStop`/`terminationGracePeriodSeconds`, ожидания для rolling update и (если используется) canary/blue-green. citeturn2search7turn0search3turn0search11turn16search2turn7search21turn7search3  
- `docs/security-baseline.md`: OWASP-ориентированные правила по secrets/error handling/TLS/logging, минимальные требования по rate limiting/ресурсам. citeturn1search8turn13search1turn13search0turn1search0turn13search7  
- `docs/progressive-delivery.md`: канареечные/blue-green стратегии (Argo Rollouts), feature flags (OpenFeature), и шаблоны постепенной миграции (Strangler Fig) — с примерами того, что должен сделать сервис (например, метрики влияния флага, поведение при деградации). citeturn7search10turn7search2turn8search0turn8search20  

**LLM-инструкции**
- `docs/llm/system-instructions.md`: MUST/SHOULD/NEVER правила из этого отчёта (включая “не угадывать требования”, “предлагать варианты с defaults”, “обновлять тесты/доки при изменениях”). Опора на строгие правила контекстов/таймаутов/ретраев критична, потому что это главные источники каскадных сбоев. citeturn3view1turn5view1turn1search2turn1search19  
- `docs/llm/codegen-recipes.md`: «рецепты» генерации типовых компонентов (HTTP handler с deadline budget, client с timeouts, rate limiting middleware, metrics instrumentation, otelhttp wrapping). Ссылки на канонические источники по инструментарию и семантикам. citeturn4view1turn5view1turn6search3turn16search3  

**Кодовая база (минимум)**
- `cmd/service/main.go`: wiring, config, logger, telemetry init, http server, shutdown. Семантика `Shutdown` и политика таймаутов должны быть в коде по умолчанию. citeturn4view0turn4view1turn6search0  
- `internal/config/`: config struct + validation (env-first, 12-factor). citeturn1search1turn12view1  
- `internal/httpapi/`: маршрутизация, middleware: request-id/trace, timeouts budget, rate limiting, recovery+error mapping (без раскрытия деталей). citeturn13search1turn6search3turn15search0  
- `internal/observability/`: slog setup, metrics registry, OTel SDK setup (включаемый конфигом), semantic conventions. citeturn6search4turn15search4turn16search1turn16search5  
- `internal/resilience/`: небольшие, тестируемые обёртки: retry policy (budget+jitter), bulkhead limiter, простая интеграция с circuit breaker интерфейсом; всё с метриками. citeturn1search2turn7search0turn1search3turn1search7