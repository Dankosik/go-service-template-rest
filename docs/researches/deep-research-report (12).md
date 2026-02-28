# Внутренний engineering standard и LLM-instructions для production-ready Go microservice template

## Scope

Этот стандарт предназначен для **greenfield** сервисов на Go, которые:
- собираются в **один (или несколько) self-contained бинарников** и деплоятся как контейнеры (или аналогичный immutable-артефакт); citeturn13view0turn24search0  
- работают как **HTTP API и/или gRPC сервис** и должны быть “cloud-native”: управляемые таймауты, отмена запросов, health-probes, наблюдаемость; citeturn19view0turn15view0turn3search1turn5search0  
- разрабатываются с активной помощью LLM (ChatGPT/Codex/Claude Code и т.п.), поэтому нужно **минимизировать пространство догадок**: явные конвенции, явные границы, “boring defaults”; citeturn13view0turn20view0  
- хотят опираться на **первичные источники** экосистемы Go: стандартная библиотека и официальные рекомендации по стилю, ошибкам, context, структуре модуля. citeturn0search8turn20view0turn19view0turn13view0  

Этот стандарт **не лучший выбор**, когда:
- вы пишете библиотеку/SDK для внешних пользователей с публичным API (тогда структура, versioning и совместимость другие; серверный шаблон часто чрезмерен); citeturn13view0turn1search0  
- у вас уже есть “корпоративная платформа”/фреймворк/генератор и сильные обязательства (наблюдаемость, DI, трассировка, security gates) — шаблон должен быть адаптирован под неё, иначе возникнет конфликт конвенций; citeturn16search7turn11search4  
- система — не микросервис (монолит, batch, CLI), или требования радикально нестандартны (ультранизкие latencies, необычные протоколы, FIPS-режим, специфичные runtime-ограничения) — “boring defaults” могут мешать; citeturn23search0turn8search3  

Ключевая идея: у шаблона должна быть **одна “траектория по умолчанию”**, которая покрывает 80% реальных сервисов без “магии”, и набор документированных расширений для остальных 20%. citeturn13view0turn20view0  

## Recommended defaults для greenfield template

### Базовая версия Go и toolchain-пиннинг
- **Go version**: используйте **Go 1.26** как baseline, потому что это актуальный стабильный релиз февраля 2026. citeturn3search0turn3search4turn3search12  
- В `go.mod` фиксируйте минимум через `go` directive и (опционально, но практично для шаблона) фиксируйте “предпочтительную” toolchain-версию через `toolchain` directive: это соответствует текущему поведению Go toolchain management. citeturn4search9turn4search17turn4search1  
- Объяснение в инженерном стандарте: `go` directive **теперь обязательное требование** для выборки подходящей toolchain (начиная с Go 1.21), и toolchain может автоматически подтягиваться. citeturn4search9turn4search1turn2search10  

### Layout репозитория и “границы изменения”
Рекомендуемая структура (для сервера) должна соответствовать официальной рекомендации “Organizing a Go module”:
- `cmd/<service>/main.go` — **entrypoint** (инициализация, wiring зависимостей, запуск серверов). citeturn14view0  
- `internal/...` — **вся логика сервиса**, т.к. сервер обычно не экспортирует пакеты наружу, и `internal` блокирует внешние импорты, упрощая рефакторинг. citeturn14view0turn4search4  

Практический дефолт для шаблона:
- `internal/app` — сборка приложения (config → logger → telemetry → storage → transport).  
- `internal/transport/http` и/или `internal/transport/grpc` — входные протоколы.  
- `internal/service` — бизнес-операции (use cases).  
- `internal/storage` — реализации репозиториев, транзакции, клиентов внешних систем.  
- `internal/observability` — обвязка логов/метрик/трейсов (без бизнес-логики).  

Почему так: официальный гайд прямо рекомендует для “server projects” держать логику в `internal`, а команды — в `cmd`. citeturn14view0  

### Контракты API: HTTP и gRPC как “первый выбор”
**Дефолт**: HTTP+JSON через стандартный `net/http` (минимум зависимостей, предсказуемая модель). При необходимости — gRPC как расширение.

Нормативные правила для таймаутов и отмен:
- На стороне HTTP сервера обязательно задавайте `ReadHeaderTimeout`, `WriteTimeout`, `IdleTimeout`, `MaxHeaderBytes` (или явно документируйте, почему не задаёте). В `net/http` прямо указано, что большинство пользователей предпочтут `ReadHeaderTimeout`, потому что `ReadTimeout` не даёт хендлерам решений per-request по телу. citeturn15view0turn21view0  
- Используйте `http.Server.Shutdown(ctx)` для graceful shutdown; документировано, что он закрывает listeners, закрывает idle connections и ждёт активные соединения до завершения или до истечения контекста. citeturn12search2  

Если используете gRPC:
- Обязательное правило — **всегда ставить deadline**: gRPC отдельно подчёркивает важность дедлайнов для устойчивых распределённых систем. citeturn5search0turn5search16  
- “Retries” — инструмент повышения надёжности, но требует дисциплины (idempotency, backoff, budget). gRPC docs описывают retry как ключевой паттерн. citeturn5search12  
- Контракты `proto` должны соответствовать официальным гайдам: proto3 language guide, style guide, и пониманию Go generated code. citeturn5search5turn5search9turn5search1  

### Configuration: boring, прозрачный, переносимый
**Дефолт**: конфигурация через environment variables (плюс опционально флаги для dev), без тяжёлых конфиг-фреймворков.
- “The Twelve-Factor App” рекомендует хранить конфигурацию в env vars как переносимый и независимый от языка механизм. citeturn16search0  
- Логи — в stdout как event stream (не управлять файлами логов внутри приложения). citeturn16search1  

Практический стандарт:
- Один `Config` struct, один `LoadConfig()` (валидирует, возвращает ошибку).  
- Все ключи документированы: имя env var, тип, default, допустимые значения, влияние на безопасность/стоимость.  

### Логи: структурированные, безопасные, пригодные для корреляции
**Дефолт**: стандартный `log/slog` (Go 1.21+) как базовый API для structured logging.
- Go официально позиционирует `log/slog` как структурное логирование с key-value для быстрой фильтрации и анализа. citeturn2search2turn2search13turn2search5  
- Для security-логирования следуйте OWASP guidance: logging mechanisms должны учитывать безопасность, а запись непроверенного ввода в логи может привести к log injection. citeturn1search3turn8search19  

Практический стандарт:
- JSON logs по умолчанию (prod), текстовый handler допускается только для локальной разработки. (Trade-off: читабельность vs машинная обработка; но `slog` построен под машинный разбор.) citeturn2search2turn16search1  
- Запрещено логировать секреты/токены/пароли; это отдельно закрепить правилом для LLM и review checklist. citeturn16search2turn16search6turn1search3  

### Observability: метрики, трейсы, семантика
**Дефолт**: трассировка через entity["organization","OpenTelemetry","observability framework"] + экспорт в entity["organization","OpenTelemetry Collector","telemetry pipeline"]; метрики — Prometheus endpoint `/metrics`.

- OpenTelemetry определяется как vendor-neutral framework для генерации/сбора/экспорта telemetry (traces/metrics/logs). citeturn16search3turn16search23  
- OpenTelemetry Collector — vendor-agnostic компонент для приёма/процессинга/экспорта telemetry, снижает необходимость множества агентов. citeturn16search7  
- Для согласованности атрибутов используйте Semantic Conventions: единая номенклатура снижает фрагментацию меток/атрибутов. citeturn0search7turn0search10  
- В Go-экосистеме метрик “boring default” — entity["organization","Prometheus","monitoring system"] scraping endpoint. CNCF фиксирует Prometheus как Graduated проект. citeturn17search15  
- Для метрик важно избегать высокой кардинальности лейблов; Prometheus docs прямо предупреждают не использовать user IDs/неограниченные значения в labels. citeturn17search0turn17search3  

Практический стандарт:
- `/metrics` (Prometheus exposition / OpenMetrics) — отдельный handler (часто на отдельном admin-port). citeturn17search1turn17search2  
- `/livez` и `/readyz` — health endpoints. Для kube-probes есть официальная документация про liveness/readiness/startup probes; liveness определяет перезапуск при “зависании”, readiness управляет попаданием в балансировку. citeturn3search1turn3search5turn3search13  

### Безопасность: минимум обязательных практик
**Дефолт security baseline** (должен быть частью шаблона и LLM-правил):
- Валидировать входные данные allowlist-подходом, т.к. OWASP рекомендует allowlist validation для пользовательского ввода. citeturn8search1turn8search17  
- SQL-инъекции предотвращать только параметризацией/подготовленными выражениями; OWASP явно указывает parameterized queries как основной способ. citeturn8search0turn8search4  
- Уязвимости зависимостей проверять `govulncheck`, который позиционируется как low-noise инструмент, использующий Go vulnerability database и анализирующий фактические вызовы. citeturn11search0turn11search3turn11search4  
- Go vulnerability database работает в OSV schema и предназначена для инструментального доступа. citeturn11search1turn11search2  

### Supply chain: reproducibility, SBOM, подписи
Это зона, где “полностью boring” решений меньше, но можно задать прагматичный минимум:
- Go toolchain downloads на go.dev с Go 1.21 воспроизводимы и могут быть верифицированы (официальный rebuild-report). citeturn9search1turn9search9  
- `go` встраивает build/VCS информацию, доступную через `go version -m` или runtime/debug, что полезно для трассируемости артефактов. citeturn9search5  
- SBOM: как минимум поддержать экспорт SBOM в одном из стандартов: entity["organization","SPDX","sbom standard"] или entity["organization","CycloneDX","bom standard"]. citeturn10search5turn10search13turn10search2turn10search6  
- Для image signing можно стандартизировать entity["organization","Sigstore","software signing project"] (cosign) как “современный дефолт” keyless signing. citeturn9search3  
- Для supply-chain maturity можно ссылаться на entity["organization","SLSA","supply chain framework"] (v1.1 approved) и уровни build track как дорожную карту. citeturn10search8turn10search12  

### Контейнеризация: минимальная поверхность атаки и предсказуемость
**Дефолт**: multi-stage build + финальный образ минимальный + non-root runtime.
- Docker официально рекомендует multi-stage builds для разделения build env и runtime env. citeturn24search0turn24search4  
- Практика “run as non-root” снижает последствия компрометации контейнера. citeturn24search1  
- Минимальные runtime-образы (например distroless) уменьшают размер и часто сокращают attack surface; при этом нужно сразу договориться о стратегии дебага (debug variant / ephemeral debug containers). citeturn24search2  
- Метаданные image лучше стандартизировать через entity["organization","Open Container Initiative","container standards org"] annotations (revision/source/etc.), чтобы связать артефакт с исходниками и ревизией. citeturn24search3  

## Decision matrix / trade-offs

Ниже — решения, которые чаще всего требуют “политики по умолчанию”. Формат: **Default / когда менять / риски**.

| Область | Default (boring) | Когда менять | Главные trade-offs (и источники) |
|---|---|---|---|
| Transport | HTTP+JSON на `net/http` | Внутренние латентные RPC, строгие контракты → entity["organization","gRPC","rpc framework"] | Для gRPC критичны deadlines и дисциплина retry. citeturn5search0turn5search12turn5search16 |
| Таймауты сервера | `ReadHeaderTimeout` + `WriteTimeout` + `IdleTimeout` + `MaxHeaderBytes` | Стриминг/long-polling может конфликтовать с `WriteTimeout` | `net/http` подчёркивает ограничения per-request для `ReadTimeout/WriteTimeout`, и рекомендует `ReadHeaderTimeout`. citeturn15view0turn5search2 |
| Graceful shutdown | `http.Server.Shutdown` с контекстом | Если есть несколько серверов/воркеров — нужен coordinator | Поведение Shutdown определено в документации; важно учитывать истечение контекста. citeturn12search2turn19view0 |
| Валидация входа | ручная allowlist-валидация + тесты | Сложные формы/объекты → отдельная lib, но с явной политикой | OWASP рекомендует allowlist validation как базу. citeturn8search1turn8search17 |
| DB доступ | `database/sql` + явные query/tx | Очень сложная доменная модель → ORM (осознанно) | `database/sql` чётко документирует контекст и rollback при cancel. citeturn12search0turn12search5 |
| Ошибки | `fmt.Errorf(... %w ...)` + `errors.Is/As` | Если нужен enrich/stacktrace → доп. подход, но не ломать `Is/As` | Wrapping `%w` и `Unwrap` описаны в Go 1.13. citeturn0search0turn0search4 |
| Логи | `log/slog` structured | Если нужна совместимость со старой экосистемой логгеров — bridge | `slog` официально в stdlib и предназначен для ключ-значение логов. citeturn2search2turn2search5 |
| Метрики | Prometheus `/metrics` | Если платформа только OTel metrics — экспорт в Collector | Prometheus docs: не злоупотреблять labels и избегать high cardinality. citeturn17search0turn17search3 |
| Трейсы | OpenTelemetry SDK + OTLP export в Collector | Если есть “vendor distro” — всё равно придерживаться semantic conv | OTel — vendor-neutral; semantic conventions стандартизируют атрибуты. citeturn16search3turn0search7turn16search7 |
| Конфиги | env vars (12-factor) | Нужны dynamic config/feature flags → отдельный слой | 12-factor: config в окружении; logs в stdout. citeturn16search0turn16search1 |
| Linters/quality gates | gofmt + go vet + govulncheck | Большие команды → golangci-lint/staticcheck | go vet — часть toolchain; govulncheck — low-noise vuln scanning. citeturn1search1turn23search0turn11search0turn11search3 |
| Container image | multi-stage + non-root + минимальный runtime | Если нужен shell в prod (обычно нет) → debug images | Docker multi-stage и non-root практики описаны в official docs. citeturn24search0turn24search1turn24search2 |
| SBOM/подписи | SBOM (SPDX/CycloneDX) + Sigstore signing | Высокий комплаенс → provenance/attestations (SLSA roadmap) | SPDX/CycloneDX — стандарты SBOM; Sigstore — keyless container signing; SLSA — уровни build track. citeturn10search5turn10search2turn9search3turn10search12 |

## Набор правил в формате MUST / SHOULD / NEVER для LLM

Цель этого раздела — текст, который можно почти напрямую положить в `docs/llm/instructions.md` (или аналог), чтобы LLM генерировала код без догадок и без “перетаскивания чужих фреймворков”.

### MUST
MUST означает: PR не принимается без выполнения.

**Стиль, форматирование, читаемость**
- MUST прогонять код через `gofmt` (и не спорить с форматированием). citeturn1search5turn1search1turn7search20  
- MUST соблюдать Go Code Review Comments как baseline (ошибки, контекст, интерфейсы, panic). citeturn20view0  
- MUST писать doc comments для экспортируемых сущностей и нетривиальных внутренних объектов. citeturn1search13turn20view1  

**Context и отмена**
- MUST принимать `context.Context` первым аргументом во всех функциях по трассе “входящий запрос → исходящие вызовы/DB”. citeturn19view0turn19view1turn20view0  
- MUST **не хранить Context в struct** (исключение — требования сигнатуры внешнего интерфейса). citeturn19view0turn20view0  
- MUST вызывать `cancel()` на всех путях выполнения при `WithTimeout/WithDeadline/WithCancel` (иначе утечки). citeturn19view0turn2search1turn2search20  

**Ошибки**
- MUST не игнорировать `error` (никаких `_ = err` и “best effort silently”). citeturn20view3  
- MUST использовать wrapping `%w` при добавлении контекста к ошибке, чтобы работали `errors.Is/As`. citeturn0search0turn0search4  
- MUST придерживаться правила error strings: без заглавных букв и пунктуации (кроме собственных имён), чтобы ошибки композиционировались корректно. citeturn20view1  

**HTTP server: безопасность и устойчивость**
- MUST задавать server timeouts и header limits (или документировать исключение). citeturn15view0turn21view0  
- MUST делать graceful shutdown через `Server.Shutdown(ctx)` и обеспечивать bounded shutdown timeout. citeturn12search2turn19view0  

**Логи и безопасность логов**
- MUST использовать структурированные логи через `log/slog` (единый logger, уровни). citeturn2search2turn2search5turn2search13  
- MUST учитывать OWASP guidance по security logging: логирование должно быть безопасным; нельзя писать непроверенный ввод “как есть”, чтобы избежать log injection. citeturn1search3turn8search19  

**Наблюдаемость**
- MUST иметь `/livez` и `/readyz`, совместимые с liveness/readiness probes; readiness должен отражать готовность обслуживать трафик. citeturn3search1turn3search5  
- MUST держать метрики с низкой кардинальностью label values (никаких user_id/request_id в labels). citeturn17search0turn17search3  

**Security baseline**
- MUST применять allowlist input validation для пользовательского ввода. citeturn8search1turn8search17  
- MUST использовать parameterized queries для SQL. citeturn8search0turn8search4  
- MUST запускать `govulncheck` в CI (и/или перед релизом) и чинить уязвимости, которые реально затрагивают кодовые пути. citeturn11search0turn11search3turn11search4  

### SHOULD
SHOULD означает: делаем по умолчанию, но можно отступить с обоснованием.

**Архитектура и зависимости**
- SHOULD предпочитать стандартную библиотеку (net/http, database/sql, log/slog) и добавлять зависимости только при понятной окупаемости. citeturn13view0turn2search13turn12search0  
- SHOULD держать бизнес-логику в `internal` и входные протоколы в `internal/transport/*`, а wiring — в `cmd/*`. citeturn14view0  
- SHOULD возвращать concrete types и определять интерфейсы в пакете-потребителе, а не “для моков” на стороне производителя. citeturn20view4  

**Errors & control flow**
- SHOULD придерживаться “indent error flow”: нормальный путь — без лишней вложенности, ошибки — early return. citeturn20view2  

**Config**
- SHOULD хранить конфигурацию в env vars и документировать их в одном месте. citeturn16search0  

**Трейсинг**
- SHOULD следовать OpenTelemetry semantic conventions для атрибутов (HTTP, DB, resource attributes), чтобы избежать “зоопарка” полей. citeturn0search7  

**Profiling и диагностика**
- SHOULD иметь опциональный admin endpoint для pprof (под защитой), т.к. Go поддерживает сбор profiling data через net/http/pprof, а диагностика официально описывает этот путь. citeturn18search0turn18search18  

### NEVER
NEVER означает: запрещено, кроме особых случаев с явным security/architecture review.

- NEVER использовать `panic` для нормальной обработки ошибок. citeturn20view1turn0search8  
- NEVER генерировать ключи/токены через `math/rand`; для ключей использовать `crypto/rand`. citeturn22view0  
- NEVER писать SQL через конкатенацию строк с пользовательским вводом. citeturn8search0turn8search4  
- NEVER добавлять метрики с labels высокой кардинальности (user_id, request_id, email и прочие unbounded value sets). citeturn17search3turn17search0  
- NEVER “прятать” контекст: не вызывать `context.Background()` внутри request path вместо `r.Context()`, не хранить контекст в поле структуры. citeturn19view0turn20view0  
- NEVER вводить интерфейсы “на будущее” или “для моков” без реального потребителя; это прямо запрещено в Go Code Review Comments. citeturn20view4  

## Concrete good / bad examples на Go

### HTTP сервер: таймауты, header limits, shutdown

**Good**
```go
srv := &http.Server{
	Addr:              cfg.HTTPAddr,
	Handler:           mux,
	ReadHeaderTimeout: 5 * time.Second,
	WriteTimeout:      30 * time.Second,
	IdleTimeout:       2 * time.Minute,
	MaxHeaderBytes:    1 << 20, // 1 MiB
}

go func() {
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("http server failed", "err", err)
	}
}()

shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
defer cancel()

_ = srv.Shutdown(shutdownCtx)
```
Почему: `net/http` явно описывает семантику `ReadHeaderTimeout/WriteTimeout/IdleTimeout/MaxHeaderBytes`, а `Server.Shutdown` — корректный путь graceful shutdown. citeturn15view0turn21view0turn12search2turn19view0  

**Bad**
```go
http.ListenAndServe(cfg.HTTPAddr, mux) // no timeouts, no shutdown path
```
Почему: при default-конфигурации вы теряете управляемость таймаутов и корректное завершение сервиса. citeturn15view0turn12search2  

### Context propagation + cancel discipline

**Good**
```go
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	user, err := h.repo.GetUser(ctx, parseID(r))
	if err != nil {
		// ...
	}
	// ...
}
```
Почему: правила `context` требуют передавать ctx явно, не хранить его в struct, и обязательно вызывать cancel, иначе утечки. citeturn19view0turn20view0turn2search1  

**Bad**
```go
type Handler struct {
	ctx context.Context // store ctx in struct
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background() // loses cancellation/deadlines
	_ = ctx
}
```
Почему: хранение ctx в struct и использование `Background()` в request path противоречит прямым правилам пакета context и Go Code Review Comments. citeturn19view0turn20view0  

### Ошибки: wrapping и читаемость

**Good**
```go
user, err := repo.GetUser(ctx, id)
if err != nil {
	return fmt.Errorf("get user %d: %w", id, err)
}
```
Почему: `%w` формализует wrapping и позволяет `errors.Is/As` работать на верхних уровнях. citeturn0search0turn0search4  

**Bad**
```go
user, err := repo.GetUser(ctx, id)
if err != nil {
	return fmt.Errorf("GetUser failed: %v.", err) // punctuation/capitalization + %v loses wrapping
}
```
Почему: нарушает правила error strings и теряет семантику `errors.Is/As`. citeturn20view1turn0search0  

### SQL: parameterized queries

**Good**
```go
row := db.QueryRowContext(ctx, `SELECT name FROM users WHERE id = $1`, id)
```
Почему: OWASP прямо указывает parameterized queries как базовую защиту от SQL injection. citeturn8search0turn8search4  

**Bad**
```go
q := fmt.Sprintf("SELECT name FROM users WHERE id = %s", r.URL.Query().Get("id"))
row := db.QueryRowContext(ctx, q)
```
Почему: классический SQL injection anti-pattern. citeturn8search4turn8search20  

### Логи: структурно + безопасно

**Good**
```go
logger.Info("request finished",
	"method", r.Method,
	"path", r.URL.Path,
	"status", status,
	"duration_ms", dur.Milliseconds(),
)
```
Почему: `slog` предназначен для key-value логов, которые фильтруются и анализируются надёжно. citeturn2search2turn2search5  

**Bad**
```go
log.Printf("user=%s token=%s err=%v", userEmail, authToken, err)
```
Почему: риск утечки секретов + OWASP предупреждает про security logging и log injection (если данные непроверенные). citeturn1search3turn8search19turn16search6  

### Метрики: избегать high-cardinality labels

**Good**
```go
// labels: method, route, status (bounded sets)
httpRequests.WithLabelValues(method, route, status).Inc()
```
Почему: Prometheus предупреждает, что high-cardinality labels резко увеличивают число time series и стоимость. citeturn17search0turn17search3  

**Bad**
```go
// label contains unbounded request_id
httpRequests.WithLabelValues(requestID).Inc()
```
Почему: request_id — practically unbounded. citeturn17search3turn17search0  

## Anti-patterns и типичные ошибки/hallucinations LLM

Ниже — список ошибок, которые LLM делают систематически, и которые стоит превратить в “guardrails” в документах и в CI.

### Архитектурные галлюцинации
- “Clean Architecture по книжке” с десятками пакетов, интерфейсов и абстракций **без потребителя**: противоречит Go Code Review Comments (“не определять интерфейсы до использования”, “не делать интерфейсы для моков на стороне producer”). citeturn20view4  
- Путаница в границах модулей: несколько `go.mod` “на всякий случай”, экспортируемые `pkg/*` без реальной необходимости. Официальный модульный гайд подчёркивает, что server projects обычно не имеют экспортируемых пакетов и рекомендует `internal` + `cmd`. citeturn14view0turn1search4  

### Ошибки вокруг context/таймаутов
- `context.Background()` внутри request path → отмена не доходит до DB/HTTP клиента → висящие операции. Правила `context` требуют явной передачи ctx, предпочтение “err on the side of passing a Context”, и запрет на хранение Context в struct. citeturn19view0turn20view0  
- `WithTimeout` без `defer cancel()` → утечки derived contexts, о чём предупреждает документация. citeturn19view0turn2search1  

### Ошибки обработки ошибок
- Игнорирование ошибок (`_ = err`, “best effort”) — прямо запрещено. citeturn20view3  
- Потеря wrapping (использование `%v` вместо `%w`) → невозможно делать `errors.Is/As`. citeturn0search0turn0search4  
- “Красивые” error strings с заглавных букв и точками — ломают композицию сообщений, это отдельно выделено в Go Code Review Comments. citeturn20view1  

### Наблюдаемость-hostile решения
- Метрики с labels высокой кардинальности (user_id/request_id/email) — прямой anti-pattern в Prometheus best practices. citeturn17search3turn17search0  
- Несогласованные имена атрибутов/меток в логах и трейcах (“reqId”, “request_id”, “rid”) — приводит к плохой корреляции; OTel semantic conventions существуют, чтобы стандартизировать схемы. citeturn0search7  

### Security-ошибки
- `math/rand` для токенов (“временно”) — прямо запрещено рекомендациями Go code review: использовать `crypto/rand`. citeturn22view0  
- SQL через конкатенацию строк — OWASP квалифицирует как основной класс инъекций и рекомендует parameterization. citeturn8search4turn8search0  
- Логирование непроверенного ввода без учёта log injection. OWASP прямо описывает log injection как атаки “непроверенный ввод → запись в лог”. citeturn8search19turn1search3  

### Tooling-галлюцинации
- “Придуманные” флаги `go test`/`go vet`/`gofmt`, несуществующие пакеты stdlib. Стандарт должен требовать ссылаться на docs при добавлении тулов и флагов. `go vet` и его смысл описаны в документации cmd/vet. citeturn23search0  
- “Случайные” зависимости ради удобства: вместо этого — минимальный осознанный набор quality gates (gofmt, go vet, govulncheck). citeturn1search1turn23search0turn11search0  

## Review checklist для PR/code review и что вынести в файлы repo

### Review checklist
Эту секцию удобно вынести в `docs/review-checklist.md` и ссылать из PR template.

**Go correctness**
- Код отформатирован `gofmt`; нет “ручных” отклонений. citeturn1search5turn1search1  
- Ошибки не игнорируются; соблюдены `Indent error flow` и правила error strings. citeturn20view3turn20view2turn20view1  
- При добавлении новых публичных сущностей есть doc comments. citeturn20view1turn1search13  

**Context, timeouts, cancellation**
- `context.Context` прокинут по цепочке; нет хранения ctx в struct; cancel() вызывается. citeturn19view0turn20view0  
- HTTP сервер имеет разумные timeouts и header limits, или есть документированное исключение. citeturn15view0turn21view0  
- Есть корректный shutdown path через `Server.Shutdown`. citeturn12search2  

**Security**
- Валидация входа соответствует allowlist-подходу. citeturn8search1  
- SQL/DB: только parameterized queries; транзакции и отмена через context учитываются. citeturn8search0turn12search0  
- В логах нет секретов/PII по умолчанию; учтён риск log injection. citeturn1search3turn8search19turn16search6  
- В CI/локально предусмотрен `govulncheck` и результаты обработаны. citeturn11search0turn11search3  

**Observability**
- Метрики: labels bounded, без high-cardinality; endpoint `/metrics` работает. citeturn17search3turn17search1  
- Health endpoints: `/livez` и `/readyz` отражают реальное состояние; соответствуют смыслу probes. citeturn3search1turn3search5  
- Трейсы/атрибуты: если добавлены — следуют semantic conventions. citeturn0search7  

**Testing**
- Unit tests на критические ветки, включая отмену (ctx timeout/cancel) и ошибки инфраструктуры. (База — пакет `testing` и стандартные механизмы subtests/parallelism). citeturn12search3turn23search9  
- Если компонент “опасен” (парсинг, декодинг, обработка внешнего ввода) — рассмотреть fuzzing через `go test -fuzz`. citeturn23search1  

**Build & container**
- Dockerfile использует multi-stage build; runtime user — non-root; минимальная финальная стадия. citeturn24search0turn24search1turn24search2  
- Артефакты связываются с ревизией/исходниками через OCI annotations (где применимо). citeturn24search3  

### Что из результата оформить отдельными файлами в template repo
Ниже — список файлов/доков, которые обеспечивают “минимум догадок” для LLM и человека.

**Файлы в `docs/`**
- `docs/engineering-standard.md` — этот стандарт: архитектурные дефолты, границы, принципы. citeturn14view0turn20view0turn19view0  
- `docs/llm/instructions.md` — MUST/SHOULD/NEVER правила для LLM (включая запрет на лишние зависимости, требования к gofmt, context и error wrapping). citeturn19view0turn20view0turn0search0  
- `docs/observability.md` — как включать логи/метрики/трейсы, политика labels, ссылки на semantic conventions, режимы prod/dev. citeturn2search2turn17search3turn0search7turn16search7  
- `docs/security-baseline.md` — input validation, SQL parameterization, logging security, `govulncheck`, секреты/конфиги. citeturn8search1turn8search0turn1search3turn11search0turn16search6  
- `docs/review-checklist.md` — checklist для PR/CR (из секции выше). citeturn20view0turn15view0turn17search0turn11search3  
- `docs/patterns-go.md` — карта паттернов из следующей секции. citeturn6search2turn6search1turn7search0turn20view4  

**Файлы в корне репозитория**
- `go.mod` / `go.sum` с Go 1.26 и (опционально) toolchain directive. citeturn3search0turn4search9turn4search17  
- `Makefile` (или `justfile`) с целями: `fmt`, `test`, `vet`, `vuln`, `lint`, `docker-build`, `run`. (Ссылки на источники — в доках про инструменты.) citeturn23search0turn11search0turn1search5  
- `Dockerfile` multi-stage + non-root; опционально — distroless final stage и debug-образ для отладки. citeturn24search0turn24search1turn24search2  
- `.golangci.yml` (опционально) или документ с обоснованием, почему ограничились gofmt+go vet+govulncheck. citeturn23search6turn23search0turn11search3  
- `.github/workflows/ci.yml` (или другой CI): шаги gofmt-check, go test, go vet, govulncheck. citeturn1search1turn23search0turn11search3  
- `README.md` с “quick start” и ссылками на `docs/*`; структура `cmd/` + `internal/` как официально рекомендуемая для серверов. citeturn14view0  

## Расширяемость и “классические паттерны” в Go

Цель: дать LLM “карту соответствий”, чтобы она не переносила OO-паттерны 1:1, а выражала их идиоматично через интерфейсы, композицию и функциональные типы.

### База идиоматичности: интерфейсы, потребитель-владелец, и generics “по делу”
- Интерфейсы в Go обычно должны жить в пакете-потребителе; не создавайте интерфейсы “для моков” на стороне producer; возвращайте concrete types. citeturn20view4  
- Generics применять осознанно: официальный гайд “When To Use Generics” даёт рекомендации, когда generics помогают, а когда перегружают дизайн. citeturn7search0turn7search3  

### Карта “pattern → идиоматичная Go-реализация → когда применять → когда запрещать”
Ниже — материал, который удобно перенести почти напрямую в `docs/patterns-go.md` и в LLM instructions.

| Pattern | Идиоматичная реализация в Go | Когда применять | Когда запрещать/осторожно |
|---|---|---|---|
| Adapter | “Функция-адаптер” или wrapper type, который удовлетворяет целевому интерфейсу. Классический пример — `http.HandlerFunc` как адаптер обычной функции к `http.Handler`. citeturn6search2 | Когда нужно привести чужой API к вашему интерфейсу без изменения логики | Не плодить адаптеры ради “архитектурных слоёв”; сначала убедиться, что есть реальный конфликт интерфейсов citeturn20view4 |
| Decorator | Композиция “оборачивающих” объектов/функций. В HTTP — middleware вида `func(http.Handler) http.Handler`; в streams — обёртки над `io.Reader/io.Writer`. citeturn6search1turn6search2 | Наблюдаемость (логирование/метрики вокруг handler), cross-cutting concerns | Осторожно с порядком декораторов и побочными эффектами; обязательно тестировать, чтобы не ломать cancellation/timeouts citeturn19view0turn3search1 |
| Strategy | Функциональные типы (`type Fn func(...) ...`) или маленький интерфейс “один метод”. По смыслу `http.Handler`/`HandlerFunc` — “стратегия обработки запроса”. citeturn6search2turn20view4 | Когда поведение должно подменяться (например, retry-policy, backoff, выбор алгоритма) | Не делать “стратегию” ради DI; если один вариант — оставьте обычную функцию/метод citeturn20view4 |
| Option pattern (functional options) | Конвенционально: `type Option func(*T)` + `NewT(opts ...Option)`. Это **популярный community-идиом**, но не формализован в official docs; применять как инструмент эволюции конструктора без взрыва параметров. citeturn6search9turn6search0 | Публичные конструкторы/SDK, где нужно безболезненно добавлять параметры со временем | Внутри сервиса (не библиотека) чаще достаточно `Config` struct: меньше магии, легче читать. Также избегать options, если они делают жизненный цикл/валидацию неявными citeturn16search0turn20view0 |
| Builder alternatives | В Go часто вместо Builder используют: (1) struct literal + defaults, (2) `Config` struct + `Validate()`, (3) functional options (см. выше). 12-factor конфиг через env часто делает Builder лишним. citeturn16search0turn19view0 | Когда объект действительно сложный и нужна поэтапная сборка, либо для читабельной инициализации больших структур | Не переносить “fluent builder” из OO ради цепочек вызовов; ухудшает ясность и усложняет тестирование без нужды citeturn20view0 |
| State pattern | Обычно: явный `type State int` + switch, или интерфейс “state machine” с ограниченным числом методов. Важно: состояние и переходы должны быть тестируемы. | Протоколы/парсеры/воркфлоу с явными конечными состояниями | Не делать “OO state objects” с множеством мелких типов, если достаточно enum+switch; риск избыточности |
| Factory patterns | В Go чаще “factory function” — это просто `NewX(...) (*X, error)` или `func New(...)` + dependency wiring в `internal/app`. Это сочетается с правилом “return concrete types”. citeturn20view4turn14view0 | Когда нужно централизовать создание со сложной валидацией и ошибками | Запрещать “abstract factory” ради DI-абстракций без потребителя; чаще приводит к лишним интерфейсам citeturn20view4 |
| io.Reader/io.Writer composition | Стандартная композиция интерфейсов (`ReadWriter`, `ReadWriteCloser` и т.п.) показывает “Go way” комбинировать поведения через embedding интерфейсов. citeturn6search1 | Потоки данных, декораторы (compression/encryption/buffering), test doubles | Не “оборачивать” I/O без управления контекстом/таймаутами там, где это важно (HTTP, DB), иначе утечки и зависания citeturn15view0turn12search0turn19view0 |
| Function types as strategy | Используйте функции как значения для внедрения поведения (например, `Clock func() time.Time` для тестов). Это обычно проще, чем интерфейс. | Подмена мелких зависимостей, тестируемость | Осторожно: не превращать всё в function fields; если нужна группа операций — лучше маленький интерфейс, определённый потребителем citeturn20view4 |
| Generics vs interfaces | Generics — когда нужен алгоритм, одинаковый для множества типов, и интерфейсная динамика не нужна. Интерфейсы — когда нужны разные реализации поведения/DI. Официальный текст “When To Use Generics” задаёт ориентиры. citeturn7search0turn7search3turn20view4 | Generics: контейнеры/утилиты/алгоритмы. Интерфейсы: границы компонентов, внешние клиенты, тестируемость с consumer-owned интерфейсом | Не использовать generics, чтобы “заменить DI”: generics не дают подмену реализации в runtime так, как интерфейсы. И не создавать интерфейсы “про запас” citeturn20view4turn7search0 |

### Какие классические OO patterns в Go чаще вредят, чем помогают
Это важно сформулировать именно как “LLM-guardrail”, потому что модели склонны переносить OO-канонические решения.

- **Singleton**: в Go почти всегда деградирует в глобальное состояние и скрытые зависимости; лучше явная передача зависимостей через `internal/app` wiring и concrete types. (Признак: глобальные `var DB *sql.DB` и т.п.). Параллельно, Go style guide подталкивает к явным зависимостям и избеганию “скрытых” контекстов/данных. citeturn20view0turn14view0  
- **Abstract Factory / Service Locator**: часто вынуждают заранее определить интерфейсы “на стороне producer” (запрещённый стиль) и вводят лишние уровни косвенности. citeturn20view4  
- **Visitor**: редко оправдан; обычно приводит к сложной иерархии типов и “двойной диспетчеризации”, которую Go не пытается сделать идиоматичной. Предпочитайте явные функции/методы и простые интерфейсы по месту использования. citeturn20view4turn0search8  
- **Inheritance-heavy patterns**: Go ориентирован на композицию и маленькие интерфейсы; “эмуляция наследования” через embedding без чёткой причины ухудшает читаемость и приводит к неявным method sets. Базовая рекомендация — держать интерфейсы небольшими и использовать композицию. citeturn6search1turn20view4turn0search8  

Итоговая формулировка для LLM instructions (как “политика”):
- “Если паттерн в OO решает проблему через наследование/иерархии — в Go сначала ищи решение через **композицию, маленький интерфейс у потребителя или функциональную стратегию**. Вводить сложные фабрики/инжекторы/локаторы — только при наличии реального потребителя и явного выигрыша.” citeturn20view4turn7search0turn6search2