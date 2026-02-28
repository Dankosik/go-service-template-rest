# Engineering standard и LLM-instruction для production-ready template микросервиса на Go

## Scope

Этот стандарт и сопутствующие LLM-инструкции предназначены для **greenfield**-разработки **сетевых сервисов (HTTP и/или gRPC)** на Go, которые:
- собираются в **один самодостаточный бинарник** и обычно деплоятся в контейнере;
- запускаются в **оркестраторе** (типично Kubernetes) и используют liveness/readiness/startup probes, конфиги через env/ConfigMap/Secret, graceful shutdown по SIGTERM, метрики/трейсы/структурные логи; citeturn3search7turn3search23turn23search9turn16search0turn16search1
- должны позволять человеку «склонировать репозиторий и сразу писать продовый код», а LLM — генерировать **идиоматичный, безопасный, поддерживаемый** Go-код без угадываний по базовым конвенциям (layout, ошибки, контекст, наблюдаемость, supply chain). citeturn19view0turn27view0turn35view0

Не применять «как есть», если:
- это **CLI/утилита**, библиотека для внешних пользователей, или монолитный продукт с множеством бинарей и экспортируемых пакетов (потребуется иной API-стабильностный контракт и layout). Для серверных проектов Go прямо рекомендует держать логику в `internal/`, а команды — в `cmd/`, и выносить повторно используемые пакеты в отдельные модули. citeturn19view0
- сервис — **стриминговый** (SSE/WebSocket/long polling), высокочастотный (низкие p99) или требует нестандартного сетевого стека: тогда «по умолчанию» таймауты `WriteTimeout/ReadTimeout`, `http.TimeoutHandler`, прокси-буферы, логирование тела и подобные дефолты могут конфликтовать с требованиями (см. trade-offs ниже). citeturn14view0
- нужны «фреймворки как платформа» (например, тяжелый DI/кодоген), строгий DDD с многоуровневой архитектурой как обязательство, либо иной стиль, не совпадающий с Go-идиомами (в таком случае LLM-инструкции должны быть переписаны под доменную архитектуру). citeturn27view0turn33view0

## Recommended defaults для greenfield template

Ниже — **boring, battle-tested defaults**, которые минимизируют неопределенность для человека и LLM и опираются на стандартную библиотеку и зрелые практики.

### Базовая версия Go и семантика `go` directive

- **Базовая версия**: фиксировать минимальную версию в `go.mod` как **Go 1.26** (актуальный stable релиз на дату запроса). citeturn5search0turn5search4  
- Явно понимать, что `go` directive в `go.mod` — это не «комментарий», а вход в выбор toolchain и **влияет на доступность language features и поведение инструментов**; начиная с Go 1.21 `go` line — **обязательное требование минимальной версии**, и toolchains откажутся использовать модуль с более новой версией. citeturn35view0
- Использовать `go fix` как часть «обновления без фантазий»: в Go 1.26 `go fix` переписан и поддерживает модернизацию кода по правилам, привязанным к `go` directive / build constraints. citeturn5search9turn12search8

### Layout репозитория

Базовый layout для server project:
- `cmd/<service>/main.go` — точка входа, wiring, обработка сигналов, запуск серверов.
- `internal/...` — вся бизнес-логика, транспорт, хранилище, клиентские адаптеры, middleware, сериализация, валидация, observability glue. Рекомендация держать supporting packages в `internal` — официальная. citeturn19view0
- `docs/` — стандарты, LLM-инструкции, ADRs, runbooks.

Принцип: **всё, что не должно импортироваться извне**, держать в `internal/`, чтобы иметь свободу рефакторинга без обещаний внешним потребителям. citeturn19view0

### HTTP: стандартный `net/http` + ServeMux patterns

- Router по умолчанию: **`net/http.ServeMux`** (без сторонних роутеров) с pattern syntax, поддерживающим методы и wildcard’ы. Это добавлено/расширено в Go 1.22 и документировано как часть стандартной библиотеки. citeturn29search1turn29search0turn31view0  
- Учитывать, что pattern syntax и matching поведение ServeMux **существенно изменилось в Go 1.22**; для отката существует `GODEBUG=httpmuxgo121=1`, читаемый один раз при старте. Поэтому template **должен** иметь актуальный `go` directive и не полагаться на «старые правила роутинга». citeturn31view0turn34search2turn35view0
- ServeMux выполняет **санитизацию** пути и Host (в т.ч. нормализация `.`/`..`/повторных `/`) и редиректы, что важно учитывать при security review и тестах. citeturn31view0

### HTTP server guardrails: таймауты, лимиты, graceful shutdown

**Сетевые guardrails MUST быть включены по умолчанию**, чтобы LLM не «забывала» защиту от медленных/злонамеренных клиентов.

- `ReadHeaderTimeout`: включать всегда. В документации сервера прямо указано, что большинству пользователей он предпочтительнее `ReadTimeout`, потому что после чтения заголовков handler может сам решать «что слишком медленно» для body, в отличие от общего таймаута чтения всего запроса. citeturn14view0
- `IdleTimeout`: включать (ограничение ожидания следующего запроса на keep-alive). citeturn15view0
- `MaxHeaderBytes`: задавать явно; это лимит на заголовки (не body). citeturn15view0
- Лимит request body: применять `http.MaxBytesReader`/`MaxBytesHandler` на вход (до decode/parse). `MaxBytesReader` предназначен именно для ограничения incoming request bodies; Go отдельно отмечал это как защиту от класса DoS. citeturn4search3turn4search10  
  Отдельно: `multipart.ReadForm` и методы `http.Request`, вызывающие его, **не лимитируют** потребление диска временными файлами — рекомендовано ограничивать размер form data через `http.MaxBytesReader`. citeturn4search4
- Graceful shutdown: использовать `http.Server.Shutdown(ctx)` (не `Close`) и обязательно ждать завершения Shutdown; `Shutdown` сначала закрывает listeners, затем idle conns и ждёт, пока активные соединения станут idle, либо пока не истечёт контекст. citeturn14view2
- Сигналы: использовать `signal.NotifyContext(parent, ...)`, так как она возвращает контекст, который становится Done при приходе сигнала/stop/Done parent’а; `stop` требуется вызывать для освобождения ресурсов и восстановления поведения сигналов. citeturn25view0
- Kubernetes-совместимость: помнить, что при завершении Pod **grace period начинает отсчёт до выполнения `preStop`**, то есть `preStop` «съедает» часть времени; контейнер будет завершён в пределах termination grace period независимо от результата hook’а. citeturn26search1turn23search9  
  Это влияет на дефолт таймаутов shutdown и на staging/production runbook.

### Context, дедлайны и отмена

- По умолчанию: `context.Context` передаётся **явно по всей цепочке вызовов** от входящего HTTP/gRPC до исходящих запросов; это стандартный коммент code review. citeturn27view0
- Большинство функций, использующих context, принимают его **первым аргументом**; `context.Background()` допустим только для действительно «не request-specific» функций, но default — «передавать ctx». citeturn27view0
- В HTTP сервере `http.Request.Context()` отменяется, если клиент отключился/отменил запрос (в т.ч. возможно в HTTP/2); derived contexts отменяются вместе с parent’ом. citeturn4search15

### Конфигурация: env vars как основной контракт

- В Kubernetes конфигурация и параметры деплоя естественно подаются через env vars — напрямую или из ConfigMap. citeturn16search0turn16search13
- Секреты: использовать Secret-объекты, чтобы не включать конфиденциальные данные в код/образы/PodSpec напрямую. citeturn16search1  
  Админам: Kubernetes отдельно подчёркивает, что Secret по умолчанию хранится в etcd **без шифрования** и рекомендует включать encryption at rest. citeturn16search4
- В качестве «boring» инженерного принципа конфиг хранится в окружении (12-factor). Это удобно как язык/OS-agnostic механизм и снижает риск случайного коммита конфиг-файлов. citeturn16search2

### Observability: логи, метрики, трейсы

**Логи**
- Использовать стандартный `log/slog` как базовый API структурированного логирования: message + level + key-value attrs. citeturn36search0turn36search1  
- Go подчёркивает, что structured logs позволяют надёжно парсить/фильтровать/искать большие объёмы логов. citeturn36search1turn36search2  

**Метрики**
- Метрики: дефолт — endpoint `/metrics` и entity["organization","Prometheus","metrics monitoring system"] exposition через официальный Go client library (или ручная реализация формата, если хотите избежать зависимостей). citeturn3search6turn20search9turn20search1  
- Соглашения по именованию метрик/лейблов — не обязательны, но служат best practices; в template лучше закрепить единый стиль имен. citeturn32search6turn3search2

**Трейсы**
- Трейсинг: дефолт — entity["organization","OpenTelemetry","observability project"] SDK с OTLP-экспортом; OTLP-спецификация помечает trace/metric/log signals как stable. citeturn3search8turn32search1turn32search13  
- Контекст-пропагация: OpenTelemetry поддерживает propagators, и дефолтный propagator использует заголовки, определённые W3C TraceContext. citeturn3search0  
- Инструментация `net/http`: использовать `otelhttp` wrapper. citeturn20search6  
- Настройка OTLP exporter через env vars описана в официальной документации. citeturn3search5  
- Важный boring default: **не стандартизировать “OTel logging” как основной runtime-логгер в template**, потому что в Go getting-started для OpenTelemetry отмечено, что logs signal всё ещё experimental и возможны breaking changes. citeturn32search0

### Безопасность: минимум обязательных контролей

Опорные документы: entity["organization","OWASP","web security nonprofit"] API Security Top 10 и Cheat Sheet Series.

- Для API: учитывать Top 10 рисков (например, Broken Object Level Authorization) и требовать явных авторизационных проверок для операций, использующих user-supplied IDs. citeturn10search0turn10search8
- Валидация входных данных: prefer allowlist validation как базовую технику для всех user inputs. citeturn10search2turn10search10
- SQL injection: использовать prepared statements / parameterized queries; это прямо указано как основной метод защиты. citeturn10search1turn10search5
- REST security: выдавать сервис по HTTPS (на практике termination может быть в ingress/mesh, но **сервис должен предполагать** transport security и корректную обработку заголовков прокси/forwarded). OWASP для REST подчёркивает необходимость HTTPS endpoints. citeturn10search22
- Secrets management: закрепить запрет на секреты в репозитории и практики централизованного хранения/ротации/аудита. citeturn10search7turn16search1

### Supply chain и зависимости

- Go modules: зависимости фиксируются и проверяются через module mirror и checksum database (`proxy.golang.org`, `sum.golang.org`) — это production-ready сервисы; go command использует их по умолчанию (параметризуется env). citeturn9search7turn9search1turn9search14turn9search0
- Supply chain best practices (ориентир): документ entity["organization","CNCF","cloud native foundation"] подчёркивает цель построения «resilient and verifiable supply chain» и обсуждает практики криптографической верификации, SBOM, attestations и хранение/распространение metadata. citeturn22view0turn22view1
- SLSA: provenance описывает attestation о том, что конкретная build-платформа произвела артефакты из заданного buildDefinition; применимо как модель для attestation’ов. citeturn8search0turn8search16
- Open source dependency posture: entity["organization","OpenSSF","oss security foundation"] Scorecard — automated checks для оценки security рисков OSS-проектов и зависимостей. citeturn8search1turn8search17

### Build/Container defaults

- Контейнерный билд: multi-stage builds как дефолт (уменьшение размера финального образа и attack surface; разделение build/runtime). citeturn17search2turn17search6
- Pod hardening: использовать securityContext и ориентироваться на Pod Security Standards; Kubernetes описывает security context как настройки privilege/access control. citeturn17search0turn17search1

## Decision matrix / trade-offs

Ниже — практические развилки, которые должны быть отражены в template как «опции с предсказуемыми последствиями», а не как хаотичный набор зависимостей.

**HTTP/JSON vs gRPC**
- HTTP/JSON: проще интеграция, проще дебаг, проще ingress; в Go стандартная библиотека теперь имеет более выразительный роутинг через ServeMux patterns. citeturn29search1turn31view0  
- gRPC: строгий контракт на protobuf, эффективная бинарная сериализация, встроенные механизмы; но требует proto toolchain и дисциплины дедлайнов. gRPC подчёркивает, что deadline — ключевой механизм для robust distributed systems. citeturn18search1turn18search3turn18search7  
**Boring default**: HTTP/JSON как базовый транспорт в template + опциональный модуль `grpc/` (protobuf, interceptors, health service). Для gRPC health checking есть стандартный протокол/гайд. citeturn18search2turn18search6

**API contract: OpenAPI vs “README-driven”**
- OpenAPI — формальный язык описания HTTP API, позволяющий инструментам и людям понимать интерфейс без изучения исходников. citeturn18search4  
**Trade-off**: требует дисциплины поддержания спеки в sync с кодом (подходит, если у вас много клиентов/генерация SDK/валидаторы).

**Метрики: Prometheus exposition vs OTel metrics**
- Prometheus: крайне распространён, есть официальные best practices по naming/instrumentation и официальный Go client/`promhttp`. citeturn3search6turn20search1turn32search6  
- OTel metrics: единый стек с traces, OTLP стэк stable; но организационно может быть сложнее, если у вас уже Prometheus-first. citeturn32search1turn32search13  
**Boring default**: Prometheus `/metrics` + OTel traces (OTLP). Это минимизирует «взрыв неизвестного» для людей и инфраструктуры.

**Logging: slog vs сторонние логгеры**
- `log/slog` — стандартная библиотека, structured logging с уровнями, совместимая с обработчиками и экосистемой; Go явно ввёл её для structured logging. citeturn36search2turn36search1  
- Сторонние (zap/zerolog и т.п.) могут иметь отличия в API/перформансе, но увеличивают когнитивную и supply-chain нагрузку.  
**Boring default**: `slog` как frontend. Если нужна конкретная “backend” форма/корреляция — делайте handler.

**Timeout strategy: server-level vs per-request**
- `ReadTimeout/WriteTimeout` ограничивают чтение всего запроса/запись ответа, но **не дают handler’ам пер-request решений** и могут ломать стриминг/slow clients в легитимных сценариях. citeturn14view0  
**Boring default**: server-level `ReadHeaderTimeout/IdleTimeout/MaxHeaderBytes` + per-request deadlines на исходящих операциях (DB/HTTP), исходя из SLO.

## Набор правил MUST / SHOULD / NEVER для LLM

Правила ниже нужно включать в «общий префикс» (LLM instruction) и держать рядом с template, чтобы модель не гадала.

### MUST

1) **Компилируемость и инструментальные гейты**
- Генерировать код, который проходит `gofmt`/`goimports` и не содержит «TODO: implement» в продовом пути. citeturn6search0turn6search1turn27view0  
- Не добавлять новые зависимости без явной причины: зависимости фиксируются Go modules, а supply-chain риск растёт с транзитивными зависимостями. citeturn8search17turn22view1turn35view0

2) **Context и отмена**
- Принимать `ctx context.Context` первым параметром во всех функциях, где есть IO/ожидания/блокировки или где нужно пронести дедлайн/трейс. citeturn27view0  
- В HTTP handler использовать `r.Context()` и прокидывать его во все исходящие вызовы; помнить, что request context отменяется при disconnect/cancel клиента. citeturn4search15

3) **Ошибки**
- Не игнорировать ошибки (`_` только в реально исключительных случаях); Go code review comments явно запрещает «discard errors». citeturn33view2  
- Ошибки должны быть возвращаемыми значениями, а не panic (кроме действительно невосстановимых ситуаций). citeturn33view3

4) **HTTP safety-by-default**
- Любой новый handler обязан иметь лимит на body (через MaxBytesReader/handler) до парсинга и decode. citeturn4search3turn4search4  
- Сервер обязан иметь `ReadHeaderTimeout`, `IdleTimeout`, `MaxHeaderBytes` и корректный `Shutdown` путь. citeturn14view0turn15view0turn14view2

5) **Observability**
- Логи — структурированные (`slog`): message + level + поля (request_id, trace_id если доступен, latency, status). citeturn36search1turn36search0  
- Метрики должны быть экспонированы и именованы по согласованному стилю (Prometheus conventions как baseline). citeturn32search6turn20search9

### SHOULD

- Держать серверный проект в `internal/` + `cmd/` и минимизировать экспортируемые пакеты. citeturn19view0  
- Использовать `net/http.ServeMux` patterns и не тянуть фреймворк-роутер, пока не доказана необходимость. citeturn31view0turn29search1  
- Использовать `signal.NotifyContext` и корректно вызывать `stop()` после завершения shutdown. citeturn25view0  
- Для конкуррентности использовать простые синхронные API и делать lifetime goroutines очевидным; если сложно — документировать, когда/почему горутины завершаются. citeturn33view2turn33view3  
- Интерфейсы определять «со стороны потребителя», не «со стороны реализации ради моков», и не вводить интерфейс без реального кейса использования. citeturn33view0turn33view1  
- Для SQL использовать параметризацию/prepared statements. citeturn10search1  
- Для конфигов в k8s ориентироваться на env vars/ConfigMaps/Secrets. citeturn16search0turn16search1turn16search2

### NEVER

- NEVER хранить `context.Context` внутри struct (кроме ситуаций вынужденного интерфейсного соответствия стандартной/сторонней библиотеке); вместо этого передавать ctx в методы. citeturn27view0  
- NEVER генерировать криптоключи/токены через `math/rand`; для ключей использовать `crypto/rand`. citeturn27view0  
- NEVER использовать `log.Fatal`/`os.Exit` в request path или библиотечном коде — это ломает сервер, особенно при `context canceled` и подобных ожидаемых ошибках (в template это должно быть запрещено). (Это правило выводится из практики корректного error handling и принципа «ошибка — значение», а также из того, что `Shutdown` требует «не выйти из программы раньше времени».) citeturn14view2turn33view3  
- NEVER добавлять «слои ради слоёв»: абстракции без необходимости ведут к тестовой сложности и потере идиоматичности; особенно опасны “интерфейсы ради моков”. citeturn33view0turn33view1

## Concrete good / bad examples

Ниже — примеры, которые стоит положить в `docs/llm/examples.md` и использовать как «эталон».

### Good: HTTP handler с лимитом body, decode, контекстом, логированием

```go
package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

type Server struct {
	Log *slog.Logger
	Svc Service
}

type Service interface {
	CreateItem(ctx context.Context, in CreateItemInput) (CreateItemOutput, error)
}

type CreateItemInput struct {
	Name string `json:"name"`
}

type CreateItemOutput struct {
	ID string `json:"id"`
}

func (s *Server) register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/items", s.handleCreateItem)
}

func (s *Server) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	// Body size guardrail.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB

	var req CreateItemInput
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		s.Log.Info("bad request body", "err", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	out, err := s.Svc.CreateItem(ctx, req)
	if err != nil {
		// Map domain errors -> HTTP status.
		if errors.Is(err, ErrAlreadyExists) {
			http.Error(w, "already exists", http.StatusConflict)
			return
		}
		s.Log.Error("create item failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)

	s.Log.Info("request",
		"method", r.Method,
		"path", r.URL.Path,
		"status", http.StatusOK,
		"dur_ms", time.Since(start).Milliseconds(),
	)
}
```

Почему это «good» по стандарту:
- используется `MaxBytesReader` для защиты от больших тел; citeturn4search3turn4search4
- используется `r.Context()` как request-scoped ctx; citeturn4search15turn27view0
- structured logging через `slog`; citeturn36search1turn36search0
- ошибки не игнорируются «по умолчанию», panic не используется. citeturn33view2turn33view3

### Bad: типичный LLM-hallucination (unsafe SQL, context.Background, “магия”)

```go
func (s *Server) handleFind(w http.ResponseWriter, r *http.Request) {
	// BAD: игнорируем ctx запроса
	ctx := context.Background()

	// BAD: конкатенация SQL (SQLi), нет параметризации
	q := "SELECT id, name FROM items WHERE name = '" + r.URL.Query().Get("name") + "'"
	rows, _ := s.DB.QueryContext(ctx, q) // BAD: error discard
	defer rows.Close()

	// BAD: fatal в request path может уронить весь процесс
	if rows.Err() != nil {
		log.Fatal(rows.Err())
	}
}
```

Какие правила нарушены:
- request context не используется, отмена клиента игнорируется; citeturn4search15turn27view0
- SQL injection (нет parameterized queries); citeturn10search1
- error discard (`_`) и `log.Fatal` в server path. citeturn33view2turn14view2

## Anti-patterns и типичные ошибки/hallucinations LLM

### Переусложнение и «интерфейсный шум»
Признаки:
- интерфейсы объявлены в пакете реализации «для моков», хотя потребитель мог бы определить интерфейс сам; это прямо отмечено в Go code review comments как плохая практика. citeturn33view0turn33view1
- интерфейсы введены до появления реального usage, набор методов “угадывается” LLM. citeturn33view0
- много «слоёв» (controller/service/repository/adapter/facade/manager) без появления новых инвариантов/границ; итог — сложные тесты и непонятный dataflow.

### Hidden state и неявные side effects
Признаки:
- package-level `var` для клиентов/конфига/логгера, которые меняются в тестах; неочевидные зависимости, гонки.
- `init()` регистрирует handlers/метрики/pprof «по умолчанию». В Go это особенно опасно на примере `net/http/pprof`, который обычно импортируют ради side effect регистрации путей `/debug/pprof/`. citeturn7search1turn7search4  
  Риск: случайно экспонировать pprof в публичную сеть и утечь диагностикой/кодом. citeturn7search4

### Плохая работа с context
Признаки:
- `context.Background()` внутри handler’ов/пер-request функций, вместо прокидывания request ctx; citeturn27view0turn4search15
- `context.WithTimeout` без `defer cancel()` (утечки таймеров/ресурсов);
- `Context` кладётся в struct (кроме исключений под интерфейсы). citeturn27view0

### Ошибки и логирование
Признаки:
- “логируем и продолжаем” вместо return/propagate — нарушает «indent error flow» и целостность обработки; citeturn33view3
- `panic`/`log.Fatal` в request path;
- отсутствуют корреляционные поля (request_id/trace_id) и уровни логирования, хотя `slog` это поддерживает из коробки. citeturn36search2turn36search0

### Race conditions и утечки горутин
Признаки:
- goroutine запускаются без ясного условия завершения; это прямо описано как источник goroutine leaks, GC их не убивает сам. citeturn33view2
- shared state без синхронизации (особенно, когда LLM «ускоряет» код, добавляя кеши/мапы).
- отсутствие `go test -race` в CI/локально, хотя race detector — встроенный инструмент с известным overhead, но полезный для выявления data races. citeturn12search11  
  Для корректности под нагрузкой это один из самых дешёвых гейтов.

### Allocation hot spots и скрытая неэффективность
Признаки:
- излишние аллокации в hot path (например, `fmt.Sprintf` в циклах, конкатенации строк, лишние []byte<->string).
- отсутствие профилирования, когда появляются p99/CPU/alloc problems. Go официально предлагает pprof и описывает сбор/анализ профилей. citeturn7search2turn7search17

## Review checklist для PR/code review

Этот раздел должен лечь в `docs/review/checklist.md` и быть пригодным и для человека, и для LLM-проверки (как «rubric»).

### Автоматические гейты в CI

1) **Форматирование и импорт**
- `gofmt -w` (или `go fmt`) обязательно. citeturn6search0turn6search18turn6search4  
- `goimports` как superset gofmt для управления imports. citeturn6search1turn27view0

2) **Тесты и анализ**
- `go test ./...` (включая table-driven tests, где применимо).
- `go vet ./...` как дефолтный анализатор подозрительных конструкций. citeturn6search2turn6search6
- `staticcheck ./...` как дополнительный анализатор багов/перф и упрощений (де-факто зрелый стандарт). citeturn6search3turn6search7
- `go test -race ./...` на поддерживаемых платформах (понимать overhead; но для CI это часто приемлемо хотя бы для core пакетов). citeturn12search11turn12search2

3) **Security**
- Проверка уязвимостей зависимостей (govulncheck/аналог) как отдельный job. Руководство по fuzzing прямо отмечает ценность fuzz как поиска багов/уязвимостей и содержит базовые паттерны. citeturn7search0turn7search3  
- Supply-chain: по возможности генерировать SBOM/attestations и хранить metadata (ориентир: CNCF supply chain best practices). citeturn22view1turn8search16

### Ручной checklist (строго по категориям)

**API и поведение**
- Есть ли формальный контракт (OpenAPI/Proto) или хотя бы стабильный HTTP surface? Если OpenAPI используется — соответствует ли он коду. citeturn18search4
- Валидация input: allowlist/строгие правила, отказ от “accept anything”. citeturn10search2
- Ошибки API: предсказуемые статусы/коды и отсутствие утечки внутренностей.

**HTTP безопасность**
- `ReadHeaderTimeout`, `IdleTimeout`, `MaxHeaderBytes` настроены; нет “listen and pray”. citeturn14view0turn15view0
- В каждом handler’е до парсинга установлен лимит `MaxBytesReader`; multipart/form-data ограничен. citeturn4search3turn4search4

**Context и cancellation**
- request ctx прокинут в БД/HTTP/очереди/кэш. citeturn27view0turn4search15
- Нет `context.Background()` в request path без обоснования. citeturn27view0
- Все `WithTimeout/WithCancel` имеют `defer cancel()`.

**Ошибки и логирование**
- Нет `panic`/`log.Fatal`; ошибки возвращаются и/или мапятся на доменные статусы. citeturn33view3turn14view2
- Логи через `slog`: уровни, структурные поля, отсутствие логирования секретов. citeturn36search2turn36search0

**Concurrency**
- Жизненный цикл горутин очевиден; есть остановка по ctx.Done() или явному stop. citeturn33view2turn11search1
- Нет гонок на shared state; добавлены тесты/гонки детектятся. citeturn12search11turn11search0

**Security fundamentals**
- SQL параметризован (prepared/parameterized). citeturn10search1turn10search5
- Секреты не хардкожены; путь получения секретов соответствует Kubernetes Secrets / секрет-менеджменту. citeturn16search1turn10search7
- Учитываются OWASP API Top 10 риски (особенно authz на object-level). citeturn10search0turn10search8

## Что оформить отдельными файлами в template repo

Минимальная нарезка документов и конвенций, чтобы LLM и человек имели один «источник правды» (и чтобы модель не домысливала):

- `docs/engineering/standard.md`  
  Норматив: layout (`cmd/`, `internal/`), код-стайл, границы зависимостей, правила ошибок/контекста/конкурентности. Основа — Go Code Review Comments и Organizing a Go module. citeturn27view0turn19view0

- `docs/engineering/http-server-defaults.md`  
  Обязательные настройки `http.Server` (timeouts, MaxHeaderBytes), лимиты body, graceful shutdown, запрет публичного pprof. citeturn14view0turn15view0turn14view2turn7search4turn7search1turn4search4

- `docs/engineering/observability.md`  
  `slog` контракт полей, метрики Prometheus (/metrics), трейсы OpenTelemetry (context propagation, OTLP env vars), позиция по OTel logs (экспериментальность). citeturn36search1turn20search9turn3search0turn3search5turn32search0

- `docs/security/baseline.md`  
  OWASP API Top 10 (как чеклист угроз), input validation allowlist, SQLi prevention, secrets management, REST over HTTPS. citeturn10search0turn10search2turn10search1turn10search7turn10search22

- `docs/review/checklist.md`  
  Чеклист выше (авто + ручной).

- `docs/llm/prefix.md`  
  «Общий префикс» (system/developer-level instruction) с MUST/SHOULD/NEVER, запретом на домысливание требований и требованием писать код, совместимый с repo defaults. Основа — CodeReviewComments + go.mod semantics + net/http docs. citeturn27view0turn35view0turn14view0turn31view0

- `docs/llm/examples.md`  
  Good/bad примеры, как выше.

- `docs/supply-chain.md`  
  Минимум: зависимость-политика (Go proxy/sumdb), SBOM/attestation ориентиры, Scorecard. citeturn9search7turn9search1turn22view1turn8search17

## Refactoring heuristics и code review rubric для LLM-generated Go

Этот раздел — «d)» и должен лечь в отдельный файл (например, `docs/llm/refactoring-rubric.md`) и использоваться как для ручной ревизии, так и как «LLM self-review» чеклист.

### Признаки хорошего LLM-кода (положительные сигналы)

- **Мало магии**: минимум глобального состояния, нет скрытых side effects (особенно `init()`), зависимости передаются явно (структуры/конструкторы), wiring в `main`. citeturn19view0turn7search4
- **Контекст прозрачен**: `ctx` — первый параметр, прокидывается до границ IO; нет хранения ctx в struct; отмена реально останавливает работу. citeturn27view0turn4search15
- **Очевидная жизнь горутин**: либо синхронные функции, либо явный lifecycle (ctx.Done, errgroup, stop). citeturn33view2turn33view3turn11search1
- **Ошибки — значения**: нет panic для обычных ошибок, нет discard errors, нормальный “indent error flow”. citeturn33view3turn33view2
- **HTTP guardrails присутствуют**: timeouts/limits/shutdown реализованы и тестируемы. citeturn14view0turn14view2turn4search3
- **Структурные логи**: `slog`, уровни, ключевые поля. citeturn36search2turn36search0

### Признаки переусложнения и “LLM smell”

**Интерфейсный шум / premature abstraction**
- Есть интерфейсы без необходимости (“на всякий случай”, “для моков”), объявлены в пакете-реализаторе; это противоречит guidance: интерфейсы обычно должны жить в пакете-потребителе. citeturn33view0  
- Интерфейсы добавлены до появления реального примера использования. citeturn33view0

**Лишние слои**
- Слои не вводят новых инвариантов (security boundary, transaction boundary, retry semantics), но увеличивают количество типов/файлов.

**Hidden state**
- package-level singleton’ы (DB, logger, config) → сложные тесты и возможные гонки.

**Context misuse**
- `context.Background()` в handler path, “потеря” ctx при переходе транспорт→сервис→репозиторий. citeturn27view0turn4search15
- Нет max deadline для исходящих вызовов (в итоге зависания при проблемах сети/БД).

**Ошибки/логирование**
- Логирование секретов/PII; отсутствие уровней; mixed responsibility (логируем вместо возврата ошибки).
- `log.Fatal`/`os.Exit` в request path.

**Race conditions**
- Shared maps/slices без мьютекса/канала/атомиков; фоновые горутины без остановки. citeturn11search0turn33view2

**Allocation hot spots**
- “Красивые” helpers, которые аллоцируют и форматируют строки в hot path, без профилирования; либо преждевременные микрооптимизации без данных (в Go прямой совет — профилировать). citeturn7search2turn7search17

### Строгий refactoring checklist (для автоматической и ручной валидации)

**A. Снижение сложности (обязательный порядок)**
1. Удалить интерфейсы без потребителя (если нет места, где реально нужна подмена реализации). citeturn33view0  
2. Схлопнуть слои, которые не дают инвариантов (перенести логику ближе к месту использования).  
3. Удерживать API синхронным, пока не доказана необходимость асинхронности. citeturn33view3  
4. Убедиться, что `internal/` содержит основную логику, а `cmd/` — только wiring. citeturn19view0

**B. Контекст и отмена**
- Все функции с IO/ожиданием имеют `ctx` первым аргументом. citeturn27view0  
- Нет ctx в struct; исключения документированы (совпадение с интерфейсом stdlib/3rd party). citeturn27view0  
- Все goroutines слушают `ctx.Done()` или имеют явный stop.

**C. Ошибки**
- Нет discard errors. citeturn33view2  
- Нет panic для обычных ошибок. citeturn33view3  
- Ошибки оборачиваются/классифицируются так, чтобы вызывающий код мог мапить их на поведение (retry/no retry, status code).

**D. HTTP safety**
- `ReadHeaderTimeout`, `IdleTimeout`, `MaxHeaderBytes` установлены. citeturn14view0turn15view0  
- `MaxBytesReader` применяется на вход. citeturn4search3  
- `Shutdown` реализован с `signal.NotifyContext` и ожиданием завершения. citeturn14view2turn25view0

**E. Concurrency и память**
- Запуск горутин документирует lifecycle или использует `errgroup`. citeturn11search1turn33view2  
- CI/локальная проверка `-race` для ключевых пакетов. citeturn12search11  
- При жалобах на CPU/alloc есть pprof путь и методика анализа. citeturn7search2turn7search17

**F. Security**
- Parameterized queries для SQL. citeturn10search1  
- Allowlist validation на входах. citeturn10search2  
- Секреты не логируются и не хардкожены; используются Secrets/секрет-менеджмент. citeturn16search1turn10search7  
- OWASP API Top 10 использован как линза при ревью авторизации и поверхности API. citeturn10search0turn10search8

### Добавка к общему префиксу (готовый блок для LLM)

Текст ниже стоит вставить в `docs/llm/prefix.md` как «Refactoring & Review addendum»:

> **Перед тем как выдать финальный код/дифф, выполни self-review по rubric:**
> - Удали интерфейсы “для моков” и преждевременные абстракции; интерфейсы вводи только со стороны потребителя и при наличии реального примера использования. citeturn33view0  
> - Проверь, что `ctx` прокинут везде, где есть IO/ожидание; `ctx` — первый параметр; не храни ctx в struct. citeturn27view0turn4search15  
> - В HTTP handlers обязателен `MaxBytesReader`, server timeouts, и корректный Shutdown через NotifyContext + Server.Shutdown. citeturn4search3turn14view0turn14view2turn25view0  
> - Не игнорируй ошибки и не используй panic для обычного контроля потока; следуй “indent error flow”. citeturn33view2turn33view3  
> - Горутины должны иметь очевидный lifecycle и отмену; иначе документируй. citeturn33view2turn11search1  
> - Логи — `slog` (структурные поля и уровни), метрики — `/metrics` (Prometheus conventions), трейсы — OpenTelemetry context propagation. citeturn36search1turn20search9turn3search0