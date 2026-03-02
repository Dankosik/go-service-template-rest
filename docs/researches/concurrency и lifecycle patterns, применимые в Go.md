# Engineering standard и LLM-инструкции для production-ready Go-микросервиса

## Scope

Этот стандарт и template рассчитаны на greenfield **сетевой сервис** на Go, который будет запускаться как отдельный процесс, масштабироваться горизонтально и считаться «одноразовым» (быстрый старт/стоп), обычно в контейнерной среде (например, оркестратор с SIGTERM + grace period). Это соответствует практикам «disposability» и «stateless processes» из 12-factor, а также ожиданиям к микросервисам (независимое развертывание/масштабирование, проектирование под отказ и восстановление, максимальная статeless-ность). citeturn21search3turn21search0turn25view0

Подход **применять**, если сервис:
- обслуживает HTTP API (внутренний или внешний) и должен иметь стандартные эксплуатационные контуры: health/probes, timeouts, логирование, метрики/трейсы, graceful shutdown, CI проверки; citeturn7search3turn15view0turn17view3turn11view1turn2search5turn21search3
- имеет явные внешние зависимости (БД, очереди, downstream HTTP/gRPC) и нуждается в единых правилах по `context.Context`, таймаутам, ограничению ресурсов, предотвращению утечек горутин; citeturn14view0turn2search2turn4search19turn6search2turn25view0
- должен быть «LLM-friendly»: структура репозитория, договоренности и политики сделаны так, чтобы модель могла генерировать идиоматичный Go-код *без догадок* (явные интерфейсы, оговоренные зависимости, строгие правила). citeturn1search0turn1search4turn13search5

Подход **не применять** как «универсальную заготовку», если:
- это библиотека/SDK (здесь другая стабильность API, экспортируемые пакеты, совместимость); citeturn3search16turn30search1
- это CLI/одноразовый batch job, который не живет как сервис и не требует probes/HTTP/observability контуров (можно взять отдельный template); citeturn21search0turn7search3
- это высокоспециализированный runtime (например, ultra-low-latency, streaming/long-poll/WebSocket-centric) где стандартные «boring defaults» (например, простые HTTP handler’ы, типовые ограничения) могут оказаться неверными и потребуют отдельного профилирования/подсистем. citeturn11view1turn15view0turn5search0

## Recommended defaults для greenfield template

Ниже — «боевые дефолты», которые template должен включать сразу, чтобы сервис можно было склонировать и немедленно развивать, не решая заново базовые вопросы эксплуатации и безопасности.

**Версия Go и toolchain**
- Базовая версия: **Go 1.26** (релиз от 10 февраля 2026, актуален на 28 февраля 2026). citeturn22search0turn22search7
- В `go.mod`:  
  - `go 1.26` (language version) и **явный `toolchain go1.26.0`** для воспроизводимости сборок. Поведение `go`/`toolchain` строк и автопереключения описано в официальной документации toolchains. citeturn23view1turn22search0

**Модульность и supply chain**
- Коммитить `go.mod` и `go.sum` в репозиторий обязательно; `go.sum` используется для проверки целостности скачанных модулей. citeturn3search8turn3search5
- По умолчанию Go использует module proxy и checksum database (proxy.golang.org / sum.golang.org) для проверяемых скачиваний зависимостей; это снижает риск «подмены» исходников на стороне прокси/вендора. citeturn3search5turn3search2
- Для dependency risk management (особенно при добавлении новых OSS-зависимостей): включить практику оценки (например, через entity["organization","OpenSSF","open source security"] Scorecard как один из сигналов риска), и фиксировать решение в ADR. citeturn24search3turn26view0

**Структура репозитория и публичная поверхность**
- Структура модулей/пакетов должна быть очевидной для человека и LLM. Минимальный «boring» вариант:
  - `cmd/<service>/main.go` — только wiring (config/logger/otel/db/http server). citeturn30search1turn1search0
  - `internal/...` — вся бизнес-логика и инфраструктурные адаптеры; `internal` ограничивает импорт «снаружи» дерева и помогает держать приватные пакеты приватными. citeturn30search1
  - `docs/` — стандарты, ADR, инструкции.  
  Это согласуется с официальной рекомендацией по layout модулей/репозиториев с несколькими командами и общими `internal` пакетами. citeturn30search1

**HTTP слой: stdlib-first**
- Использовать `net/http` и `http.ServeMux` как дефолтный роутер:  
  - в Go 1.22+ `ServeMux` поддерживает шаблоны, где можно матчить method/host/path и использовать wildcard’ы; значения wildcard доступны через `Request.PathValue`. citeturn29view2turn29view0
  - `ServeMux` также делает sanitizing пути и Host header (в т.ч. удаляет порт из Host при матчингe). citeturn29view3  
  Это снижает количество зависимостей и уменьшает поверхность «LLM угадываний», при этом покрывает большинство REST API use-cases.

**Timeouts и лимиты как дефолт**
- HTTP server MUST иметь выставленные лимиты:
  - `ReadHeaderTimeout`, `WriteTimeout`, `IdleTimeout`, `MaxHeaderBytes` (или оставить дефолт и зафиксировать решение). `net/http` явно описывает семантику и trade-offs (почему часто предпочтительнее `ReadHeaderTimeout`, чем `ReadTimeout`). citeturn15view0turn15view3
- Для входящих тел запросов:
  - использовать `http.MaxBytesReader` (или `http.MaxBytesHandler`) на всех endpoint’ах, которые читают body, чтобы предотвратить случайные/злонамеренные большие payload’ы и расход ресурсов. citeturn18view0turn9search3
- HTTP client:
  - **не использовать** «голый» `http.Client{}` без таймаута: `Client.Timeout` ограничивает полный жизненный цикл запроса (connect + redirects + чтение body) и отменяет его «как будто Context закончился». citeturn17view3
  - клиентов нужно **переиспользовать** (внутреннее состояние транспорта/пул соединений), и они безопасны для concurrent use. citeturn17view3
  - всегда закрывать `Response.Body`; иначе транспорт может не переиспользовать keep-alive соединения. citeturn11view0

**Graceful shutdown и disposability**
- Root context процесса: `signal.NotifyContext` (Go 1.16+) для SIGTERM/SIGINT; semantics описаны в `os/signal`. citeturn10search0
- HTTP server shutdown: использовать `http.Server.Shutdown(ctx)` и *ждать завершения*; `Shutdown` закрывает listeners, закрывает idle conns и ждет активные, но контекст может прервать ожидание. citeturn11view1
- Помнить, что `Shutdown` **не закрывает hijacked/upgrade соединения** (например, WebSocket); это требует отдельного lifecycle управления через `Server.RegisterOnShutdown` и протокольные механизмы. citeturn11view1
- Для Kubernetes: учитывать, что `preStop` не исполняется «асинхронно от сигнала» и входит в общий `terminationGracePeriodSeconds`; зависший `preStop` удерживает Pod в Terminating до истечения grace period. Дефолтный `terminationGracePeriodSeconds` — 30 секунд. citeturn1search3turn1search7  
- Эти правила совпадают с 12-factor принципом «fast startup and graceful shutdown». citeturn21search3

**Observability: минимум, который реально помогает**
- Логирование: stdlib `log/slog` как дефолт structured logging, JSON для production, текст/человеко-читаемо для dev. `slog` официально добавлен и описан как structured logging API. citeturn2search5turn2search1
- Метрики:
  - если используется Prometheus, дефолтный эндпоинт `/metrics` и клиентская библиотека Prometheus для Go — стандартный путь, описанный в официальном руководстве Prometheus. citeturn7search2turn7search21
- Трейсинг:
  - дефолт — OpenTelemetry SDK для Go и экспорт OTLP (обычно через Collector). OTel документация подчеркивает важность context propagation для корреляции сигналов. citeturn7search4turn7search0
  - важно: в официальном Getting Started для Go отмечено, что **logs signal все еще experimental** (возможны breaking changes) — поэтому template **не должен** делать OpenTelemetry Logs обязательным контуром. citeturn7search11

**Security baseline**
- Ошибки API: возвращать клиенту «безопасные» сообщения без внутренних деталей; OWASP отдельно предупреждает об утечке информации через ошибки/stack traces. citeturn8search1turn8search5
- Логи: строить security logging осознанно, не логировать секреты/чувствительные данные, учитывать риски log injection; OWASP дает отдельный гайд по security logging. citeturn8search0turn8search2
- TLS:
  - не использовать `InsecureSkipVerify` вне тестов: документация `crypto/tls` прямо говорит, что это делает TLS уязвимым к MITM, если не включена кастомная проверка. citeturn20view0
  - TLS политика должна следовать актуальным рекомендациям (минимум версии/настройки) — OWASP TLS Cheat Sheet как baseline. citeturn8search3turn20view0
- Ограничение ресурсов/abuse:
  - rate limiting/throttling — обязательная стратегия доступности для микросервисов (NIST SP 800-204) и отдельный риск в OWASP API Security Top 10 (Unrestricted Resource Consumption). citeturn25view0turn9search3turn9search7

**Build & container**
- Dockerfile: multi-stage build как дефолт, чтобы финальный образ содержал только артефакты, необходимые для запуска; Docker официально рекомендует multi-stage как способ уменьшить размер и разделить build/run стадии. citeturn10search3turn10search11
- Важная организационная рамка: в cloud-native lifecycle security должна быть «встроена» в develop/build/test/distribute/deploy/runtime; CNCF security whitepaper подчеркивает важность security checks на ранних стадиях и supply chain практик (SBOM, attestations). citeturn26view0turn24search6

## Decision matrix / trade-offs

| Решение | Default (template) | Альтернативы | Когда менять | Риски/цена |
|---|---|---|---|---|
| Роутинг HTTP | `net/http` + `ServeMux` patterns + `PathValue` | Роутеры-фреймворки (chi/gin/echo) | Сложные middleware/маршрутизация, где stdlib неудобен | Больше зависимостей и «пространства для галлюцинаций» у LLM; у stdlib есть четкие правила конфликта pattern’ов (panic при неверной регистрации). citeturn29view2turn29view0 |
| Structured logging | `log/slog` (JSON в prod) | zap/zerolog | Очень жесткие latency требования на логировании или единый стек компании | `slog` — стандартная библиотека и проще стандартизировать. citeturn2search5turn2search1 |
| Tracing | OpenTelemetry traces | vendor SDK | Если инфраструктура не поддерживает OTel/Collector | OTel требует дисциплины context propagation. citeturn7search0turn7search4 |
| Metrics | Prometheus `/metrics` | OTel metrics only | Если платформа стандартизировала только OTel metrics pipeline | Prometheus имеет прямой, простой путь для Go сервисов. citeturn7search2turn7search21 |
| Config | 12-factor env vars + строгая валидация | config-файлы, Viper | Нужны большие конфигурационные деревья и динамика | Env vars уменьшают риск «случайно закоммитить конфиг» и portable. citeturn21search1turn21search0 |
| Shutdown | `signal.NotifyContext` + `Server.Shutdown` с timeout | кастомные lifecycle фреймворки | Очень сложный процессный оркестратор потоков | Нужно помнить про hijacked conns. citeturn10search0turn11view1 |
| Concurrency orchestration | `errgroup.WithContext` | `WaitGroup`/ручные каналы | Нужен fine-grained control без связи с ошибками | `errgroup` дает error propagation + cancellation. citeturn0search2turn0search6 |
| Bounded concurrency | `x/sync/semaphore` | channel-semaphore | Нужно «весовое» ограничение и попытки acquire | `Weighted` — официальный пакет `x/sync`, ровно для этой задачи. citeturn6search3 |
| Rate limiting | На уровне gateway/ingress по умолчанию; в сервисе — только при необходимости | Внутрисервисный limiter | Если нет API gateway или нужен per-operation лимит | NIST/OWASP считают отсутствие лимитов риском доступности. citeturn25view0turn9search7 |
| TLS verify | verify включен (дефолт) | `InsecureSkipVerify` | Только тестовые стенды + отдельное обоснование | `InsecureSkipVerify` повышает риск MITM. citeturn20view0turn8search3 |

## Набор правил в формате MUST / SHOULD / NEVER для LLM

Ниже — «контракт генерации кода»: эти правила должны лежать в общих LLM-instructions (префикс/системный промпт) и дублироваться в repo conventions. Формулировки намеренно нормативные, чтобы уменьшить «пространство для фантазии».

**MUST**
- MUST генерировать код, который проходит `gofmt` (или `go fmt`) и следует идиоматике Go (Effective Go + Code Review Comments). citeturn3search1turn3search4turn1search0turn1search4
- MUST любой I/O и долгие операции строить вокруг `context.Context` (первый параметр `ctx`), уважая cancellation/deadlines. citeturn14view0turn13search5turn12view0
- MUST вызывать `cancel()` для контекстов, созданных через `WithCancel/WithTimeout/WithDeadline`, чтобы освобождать ресурсы. citeturn14view0
- MUST управлять временем жизни горутин: каждая goroutine должна иметь явный путь завершения; утечки при блокировке на send/recv не «собираются GC». citeturn6search2turn14view0
- MUST использовать `errgroup.WithContext` для параллельных подзадач «одной операции», чтобы ошибки отменяли остальные. citeturn0search2turn0search6
- MUST ограничивать concurrency при fan-out на внешние ресурсы (bounded concurrency), используя `x/sync/semaphore` или worker pool. citeturn6search3turn25view0
- MUST для HTTP server выставлять таймауты и лимиты заголовков; MUST лимитировать request body (`MaxBytesReader/Handler`). citeturn15view0turn18view0
- MUST для outbound HTTP использовать переиспользуемый `http.Client` с установленным `Timeout`; MUST закрывать `Response.Body`. citeturn17view3turn11view0
- MUST для DB использовать `database/sql` корректно: `*sql.DB` — общий pool, открывается один раз; запросы делать через `...Context`, закрывать `Rows`, проверять `Rows.Err`. citeturn27view0turn28view0turn4search19turn2search6
- MUST не раскрывать внутренние детали в HTTP ошибках и не отдавать stack traces клиенту. citeturn8search1turn8search5
- MUST не логировать секреты/PII и учитывать log injection; security logging строить по OWASP guidance. citeturn8search0turn8search2turn19search2
- MUST реализовывать graceful shutdown через `signal.NotifyContext` и `Server.Shutdown` и ожидать завершения shutdown. citeturn10search0turn11view1turn21search3
- MUST добавлять/обновлять тесты: table-driven tests где уместно, и прогонять race detector в CI для concurrency-кода. citeturn4search5turn5search0
- MUST поддерживать go module целостность: коммитить `go.mod`/`go.sum`. citeturn3search8

**SHOULD**
- SHOULD использовать stdlib `http.ServeMux` patterns и `Request.PathValue` вместо добавления роутера-зависимости без необходимости. citeturn29view2turn29view0
- SHOULD использовать `log/slog` как единый интерфейс логирования. citeturn2search5turn2search1
- SHOULD интегрировать Prometheus `/metrics` и OpenTelemetry traces (OTLP) как базовый observability набор; logs через OTel SHOULD NOT быть обязательным из-за experimental статуса. citeturn7search2turn7search4turn7search11
- SHOULD хранить config в env vars и валидировать на старте; это portable и снижает риск «закоммитить конфиг». citeturn21search1turn21search0
- SHOULD учитывать инфраструктурные сигналы Kubernetes (liveness/readiness/startup probes) и корректно разводить semantics. citeturn7search6turn7search3
- SHOULD регулярно запускать `go vet` и использовать его как обязательную CI стадию. citeturn3search0
- SHOULD рассматривать fuzzing для критичных парсеров/валидаторов и edge-case входов, т.к. Go fuzzing позиционируется как способ находить security issues/уязвимости. citeturn4search2turn4search18
- SHOULD придерживаться supply chain практик (SBOM/attestations) в CI/CD, как рекомендует CNCF security guidance. citeturn26view0

**NEVER**
- NEVER добавлять `context.Context` в поля struct «для удобства» (кроме редких случаев совместимости сигнатур интерфейса); контексты нужно передавать параметром. citeturn13search5turn13search18
- NEVER использовать `context.WithValue` для «обычных параметров функции» или произвольных string ключей; ключи должны быть безопасными от коллизий (кастомный comparable тип), и values — только request-scoped данные, которые действительно должны пересечь API границы. citeturn14view0
- NEVER запускать «fire-and-forget» goroutine без явного ownership, cancellation и точки ожидания/остановки (goroutine lifetime должен быть очевиден). citeturn6search2
- NEVER использовать `http.DefaultClient`/`http.DefaultTransport` «как есть» для production I/O без осознанной конфигурации таймаутов; отсутствие таймаута означает «нет таймаута». citeturn17view3turn16view3
- NEVER использовать `InsecureSkipVerify` в продакшене. citeturn20view0
- NEVER возвращать клиенту внутренние stack traces/SQL ошибки/подробности реализации. citeturn8search1turn8search5

## Исследование подтемы: concurrency и lifecycle patterns, применимые в Go

Цель секции — не «урок по concurrency», а **нормативный набор допустимых паттернов** для template, чтобы LLM не генерировала скрытую/опасную конкурентность и чтобы lifecycle процессов/запросов был управляемым.

**Базовый lifecycle-контракт**
- **Root context процесса** должен создаваться через `signal.NotifyContext(parent, SIGINT, SIGTERM, ...)`. Этот `ctx` является source-of-truth для остановки всего приложения. citeturn10search0
- **Каждый компонент** (HTTP server, consumers, background loops) должен запускаться как goroutine, привязанная к одному «дереву отмены» root context. Для оркестрации — `errgroup.WithContext`. citeturn0search2turn14view0
- **Shutdown path**:  
  - на `ctx.Done()` инициировать остановку HTTP server через `Server.Shutdown` с timeout; citeturn11view1turn15view0  
  - дождаться `errgroup.Wait()`;  
  - закрыть внешние ресурсы (DB, exporters). citeturn27view0turn7search4

**Контракт «время жизни горутин»**
- Любая goroutine должна иметь понятный ответ на вопрос: *когда она завершится?*  
- Утечки возможны, если горутина блокируется на send/recv, и GC ее не убьет даже если каналы недоступны. Следовательно:
  - либо goroutine читает `ctx.Done()` в `select`;
  - либо есть закрываемый ownership-канал «stop»;
  - либо есть bounded queue/worker pool с корректным закрытием и drain. citeturn6search2turn14view0

**Когда использовать какие primitives**
- `errgroup.WithContext` — дефолт для параллельных подзадач с единым результатом/ошибкой и общей отменой (например: параллельно сходить в 2 downstream и объединить ответ). citeturn0search2turn0search6
- `sync.WaitGroup` — допустим, но только если вам не нужна «первая ошибка отменяет остальных» и cancellation уже обрабатывается отдельно. В template как primitve для LLM лучше **не поощрять**, чтобы не терять error propagation. (Это design choice; опирается на тот факт, что `errgroup` именно расширяет `WaitGroup` обработкой ошибок и cancellation.) citeturn0search2
- Каналы — использовать в двух случаях:
  1) как часть pipeline/fan-out/fan-in (стрим результатов, backpressure);  
  2) как очередь задач в worker pool.  
  Для «просто сигнал остановки» предпочтительнее context/NotifyContext. citeturn6search0turn10search0turn14view0
- `x/sync/semaphore.Weighted` — дефолт для bounded concurrency, когда нужно ограничить число одновременно активных операций к ресурсу (например, максимум 20 параллельных запросов к одному downstream). citeturn6search3turn25view0

**Паттерны, допустимые в template**

1) **Bounded fan-out + errgroup** (наиболее частый «правильный» шаблон для LLM)

```go
g, ctx := errgroup.WithContext(ctx)
sem := semaphore.NewWeighted(int64(maxConcurrent))

for _, item := range items {
	item := item
	g.Go(func() error {
		if err := sem.Acquire(ctx, 1); err != nil {
			return err // ctx canceled / deadline exceeded
		}
		defer sem.Release(1)

		// ВАЖНО: любая I/O операция принимает ctx.
		return doWork(ctx, item)
	})
}

if err := g.Wait(); err != nil {
	return err
}
return nil
```

citeturn0search2turn6search3turn14view0

2) **Pipeline + cancellation (fan-out/fan-in)** — использовать, когда нужно обрабатывать поток данных стадиями (парсинг → валидация → enrichment → запись). Официальный блог Go описывает fan-out/fan-in и cancellation как композиции каналов. citeturn6search0  
Нормативное требование для template: pipeline обязан иметь cancellation через context (или done channel), иначе высок риск goroutine leaks.

3) **Worker pool** — использовать, когда:
- есть бесконечный или длинный stream задач (например, чтение из очереди);
- нужно держать фиксированное число workers;
- нужен backpressure через bounded queue.  
Worker pool должен:
- иметь ограниченный буфер задач (или explicit backpressure через блокировку producers);
- завершаться по `ctx.Done()` и корректно закрывать workers/каналы. citeturn6search2turn25view0

4) **Background jobs и «ownership boundary»**
- По умолчанию template должен **не** поощрять in-process cron’ы, если можно вынести в внешний scheduler (чтобы не дублировать работу при масштабировании). Это следует из 12-factor «concurrency scale out via process model» и идеи stateless процессов. citeturn21search0turn21search1  
- Если background job все же нужен (например, периодический refresh кэша):
  - он запускается в составе `errgroup` и завершается по root ctx;
  - все I/O внутри — с контекстом и таймаутами;
  - для задач, которые должны **пережить** отмену request context (например, логирование аудита после того, как клиент закрыл соединение), допустимо использовать `context.WithoutCancel(reqCtx)` как осознанную границу владения, понимая, что `WithoutCancel` убирает Done/Deadline/Err (то есть отмена request’а вас больше не остановит). citeturn14view0turn12view0

5) **HTTP request lifecycle как источник отмены**
- В `net/http` контекст входящего запроса отменяется при закрытии соединения клиентом, отмене request (HTTP/2), или после возврата из `ServeHTTP`. Следовательно, весь request-scoped work должен использовать `req.Context()`. citeturn12view0

**Запрещенные concurrency anti-patterns (для LLM — hard ban)**
- Запуск goroutine «на фоне» без `ctx.Done()` ветки и без точки ожидания/остановки. citeturn6search2
- Unbounded fan-out (`for { go ... }`) к внешним ресурсам — прямой путь к oversubscription; NIST прямо выделяет rate limiting/throttling как стратегию доступности, а без bounded concurrency это нарушается на уровне процесса. citeturn25view0turn6search3
- Вставка `context.WithTimeout(context.Background(), ...)` глубоко в бизнес-логике «потому что так проще» — таймауты должны задаваться на boundary (HTTP handler / job runner / consumer loop), иначе получается «скрытый» SLA и трудно управлять временем жизни. (Нормативно закрепляется тем, что контекст несет deadlines/cancellation через границы API и должен передаваться явно, а не создаваться внутри произвольных слоев.) citeturn14view0turn13search5

## Concrete good / bad examples

**Пример: корректный HTTP handler с лимитом body, безопасной ошибкой и уважением cancellation**

Bad:

```go
func Create(w http.ResponseWriter, r *http.Request) {
	var req CreateReq
	_ = json.NewDecoder(r.Body).Decode(&req) // игнорируем ошибку + без лимита размера
	// ...
	w.WriteHeader(http.StatusOK)
}
```

Good:

```go
func Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
	defer r.Body.Close()

	var req CreateReq
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := doCreate(ctx, req); err != nil {
		// Клиенту — без деталей; детали уходят в лог.
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
```

citeturn18view0turn12view0turn8search1turn8search5

**Пример: корректный outbound HTTP request с таймаутом и правильным закрытием body**

Bad:

```go
resp, _ := http.Get(url) // нет таймаута
b, _ := io.ReadAll(resp.Body)
// resp.Body не закрыли
```

Good:

```go
client := &http.Client{Timeout: 3 * time.Second}

req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
if err != nil {
	return err
}

resp, err := client.Do(req)
if err != nil {
	return err
}
defer resp.Body.Close()

_, err = io.Copy(io.Discard, resp.Body) // читаем до конца (если нужно reuse)
return err
```

citeturn17view3turn11view0turn12view0

**Пример: корректный DB query с контекстом и закрытием Rows**

Bad:

```go
rows, _ := db.Query("SELECT name FROM users")
for rows.Next() { /* ... */ } // Rows.Err не проверили, Close не вызвали
```

Good:

```go
rows, err := db.QueryContext(ctx, "SELECT name FROM users WHERE age = ?", age)
if err != nil {
	return err
}
defer rows.Close()

for rows.Next() {
	var name string
	if err := rows.Scan(&name); err != nil {
		return err
	}
}

if err := rows.Err(); err != nil {
	return err
}
return nil
```

citeturn28view0turn27view0turn4search19

## Anti-patterns и типичные ошибки/hallucinations LLM

**Нулевые таймауты «по умолчанию»**. LLM часто пишет `http.Client{}` или использует `http.DefaultClient`. В `net/http` таймаут 0 означает «нет таймаута», а `Timeout` определяет полный лимит времени и отменяет запрос как завершившийся `Context`. citeturn17view3

**Забытый `resp.Body.Close()` и неполное чтение body**. Это приводит к утечке ресурсов и ухудшению reuse keep-alive. Документация прямо предупреждает о невозможности re-use соединений, если body не дочитано и не закрыто. citeturn11view0

**Невидимая конкурентность**: «на всякий случай» запускать goroutine внутри handler’а для логирования/побочных эффектов. Это часто создает goroutine leaks и гонки, потому что request context отменится при disconnect и работа зависнет/оборвется. citeturn6search2turn12view0

**Хранение `context.Context` в struct**. LLM делает это для «удобства», но Go guidance прямо говорит не делать так, а передавать `ctx` параметром (кроме редких случаев совместимости с интерфейсами). citeturn13search5turn13search18

**`context.WithValue` как «универсальный контейнер параметров»**. В официальной документации контекста сказано, что values — только для request-scoped данных, пересекающих границы API/процессов, и ключи не должны быть built-in типами вроде `string`. citeturn14view0

**Unbounded fan-out** (тысячи goroutine на список задач). Без bounded concurrency вы создаете self-DoS и нарушаете базовые стратегии доступности; NIST отдельно выделяет rate limiting/throttling, а `semaphore.Weighted` — прямой инструмент ограничения. citeturn25view0turn6search3

**Небезопасные ошибки клиенту** (stack trace/SQL details). OWASP прямо описывает информационные утечки и рекомендует generic сообщения. citeturn8search1turn8search5

**Логирование секретов** (tokens, passwords, API keys). Это превращает логи в «секретный актив», усложняет compliance и повышает ущерб при утечке; OWASP secrets/logging guidance отдельно подчеркивает необходимость правильного управления секретами и логами. citeturn19search2turn8search0turn8search2

**`InsecureSkipVerify`**. LLM иногда вставляет это «чтобы заработало на dev». Документация `crypto/tls` говорит, что режим уязвим к MITM и допустим только для тестов или с кастомной проверкой. citeturn20view0

## Review checklist для PR/code review и что вынести в отдельные файлы repo

**PR / Code review checklist (короткий, но строгий)**

- Стиль и качество Go:
  - Код отформатирован gofmt; публичные API/имена идиоматичны (Effective Go, Code Review Comments). citeturn3search1turn1search0turn1search4
  - Нет хранения `Context` в struct без обоснования; `ctx` — первый параметр всех релевантных функций. citeturn13search5turn13search18
- Concurrency и lifecycle:
  - У каждой goroutine определен lifecycle (как завершится). Нет «вечных» горутин без остановки. citeturn6search2
  - Для fan-out есть bounded concurrency (`semaphore`/worker pool). citeturn6search3turn25view0
  - Используется `errgroup` там, где ошибки должны отменять параллельные подзадачи. citeturn0search2
  - Graceful shutdown корректен: `signal.NotifyContext` + `Server.Shutdown` + ожидание завершения. Учтены hijacked соединения, если есть. citeturn10search0turn11view1
- HTTP correctness и устойчивость:
  - На server выставлены timeouts/лимиты; request body ограничен. citeturn15view0turn18view0
  - Outbound HTTP: `Client.Timeout` задан, `Response.Body` закрывается, нет per-request создания клиентов без причины. citeturn17view3turn11view0
- DB:
  - `*sql.DB` создается один раз и переиспользуется как pool; используются `...Context`; `Rows` закрываются и проверяется `Rows.Err`. citeturn27view0turn28view0turn4search19
- Security:
  - Клиенту не отдаются внутренние детали ошибок; в логах нет секретов/PII, есть базовые security logging события. citeturn8search1turn8search0turn19search2
  - Нет `InsecureSkipVerify` и иных «временных» insecure обходов без явного guard’а. citeturn20view0
  - Для API есть стратегия resource-consumption (payload limits, timeouts, rate limiting на уровне gateway или в сервисе при необходимости). citeturn9search7turn25view0turn18view0
- Observability:
  - Логи структурированы (`slog`), есть correlation ids/trace context propagation по возможности (если включен OTel). citeturn2search5turn7search0
  - Есть `/metrics` (если Prometheus) и минимальные health endpoints для probes. citeturn7search2turn7search3
- Тесты и CI:
  - Есть/обновлены unit-тесты (table-driven где уместно). citeturn4search5
  - CI запускает `go test` + race detector для concurrency критичного кода; `go vet` включен. citeturn5search0turn3search0turn4search0
- Supply chain:
  - `go.mod`/`go.sum` в порядке; зависимости добавлены осознанно и задокументированы (ADR), по возможности оценены по supply chain практикам. citeturn3search8turn26view0turn24search3

**Что оформить отдельными файлами в template repo**

Ниже — набор файлов, которые стоит положить в `docs/` и корень репозитория, чтобы LLM могла «читать контракт» из репо и не додумывать:

- `docs/engineering-standard.md`  
  Нормы: структура проекта, зависимости, API conventions, error model, logging/metrics/tracing, security baseline, CI правила. Опирается на Effective Go/CodeReviewComments, OWASP cheat sheets, NIST, Kubernetes, Docker, 12-factor. citeturn1search0turn1search4turn8search0turn25view0turn7search3turn10search11turn21search0
- `docs/llm-instructions.md`  
  MUST/SHOULD/NEVER правила (как выше), плюс «workflow генерации изменений»: какие файлы можно трогать, как писать тесты, как писать миграции, как обновлять конфиг и т.д. Основа — правила из официальных Go docs и security guidance. citeturn14view0turn6search2turn3search1turn3search0turn8search1
- `docs/concurrency-and-lifecycle.md`  
  Нормативный каталог паттернов (errgroup, bounded concurrency, pipeline, worker pool, cancellation, shutdown в Kubernetes). citeturn0search2turn6search3turn6search0turn1search7turn10search0
- `docs/adr/` + `docs/adr/0001-template-stack.md`  
  Зафиксировать ключевые выборы (stdlib routing, slog, Prometheus, OTel traces, Go toolchain pinning) и критерии изменения. Практика ADR не «про обзор», а про снижение догадок LLM и людей. (Опора на toolchain/modules guidance и supply chain рекомендации.) citeturn23view1turn3search8turn26view0
- `CONTRIBUTING.md`  
  Как запускать тесты/линты локально, как формировать PR, что считается «готово». Минимум: gofmt, go test, go vet, race detector. citeturn3search1turn4search0turn3search0turn5search0
- `SECURITY.md`  
  Политики secret management, логирования инцидентов, рекомендации по TLS, error handling. citeturn19search2turn8search0turn8search3turn8search1
- `Makefile` (или `taskfile.yml`)  
  Стандартизированные цели: `fmt`, `test`, `vet`, `race`, `run`, `docker-build`. (Опора на gofmt/go vet/go test). citeturn3search1turn3search0turn4search0
- `.github/workflows/ci.yml` (или эквивалент в вашей CI)  
  Пайплайн: gofmt-check, go test, go vet, race; supply chain шаги при наличии (SBOM/attestations), как рекомендует CNCF security whitepaper. citeturn26view0turn5search0turn3search0turn3search1
- `Dockerfile` (multi-stage)  
  Сборка бинаря и минимальный runtime stage; Docker официально описывает multi-stage и его пользу. citeturn10search3turn10search11
- `go.mod` / `go.sum`  
  `go 1.26` + `toolchain go1.26.0`. Поведение строк `go`/`toolchain` описано официально. citeturn23view1turn22search0