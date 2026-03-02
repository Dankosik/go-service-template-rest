# Стандарт и LLM-инструкции для production-ready Go микросервиса

## Область применения и ограничения

Этот стандарт предназначен для **greenfield** шаблона микросервиса на Go, который: разворачивается как контейнер, живёт в оркестраторе (обычно Kubernetes), имеет чётко описанные входные контракты (HTTP/OpenAPI или gRPC/Protobuf), должен быть наблюдаемым (логи/метрики/трейсы) и безопасным по умолчанию. Микросервисная архитектура в целом подразумевает разбиение приложения на независимые сервисы с узким фокусом, которые взаимодействуют по сети. citeturn17search2

Подход особенно хорошо работает, когда команда хочет:
- максимально снизить «стоимость старта» нового сервиса (клонируешь репо → запускаешь → пишешь бизнес-логику),
- стандартизировать инфраструктурные аспекты (timeouts, shutdown, health, telemetry, CI),
- использовать LLM-инструменты (ChatGPT/Codex/Claude Code и т.п.) так, чтобы модель **не гадала**, а действовала в рамках явных конвенций и генерировала идиоматичный Go-код, который проходит линтеры/тесты и не ломает контракты.

Этот стандарт **не подходит** или требует явной адаптации, если:
- сервис **не** является сетевым, долгоживущим процессом (например, CLI, batch-job, одноразовые миграции),
- требования диктуют нестандартные runtime-ограничения (встроенные устройства, экстремально низкая latency, где любые абстракции и middleware запрещены),
- есть необходимость в «framework-heavy» архитектуре с жёстким каркасом: Go-экосистема обычно предпочитает простые структуры и эволюцию «по мере роста», а не один «вечный стандартный layout». citeturn16search7turn16search2
- вы **контролируете все клиенты и их релизы** (например, один репозиторий, монорепа, единая поставка): тогда политика обратной совместимости может быть мягче, но её всё равно нужно формализовать, чтобы LLM не вносила поломки «случайно». citeturn12view0

Базовые эксплуатационные принципы (конфиг через окружение, логи как поток событий, быстрый старт и graceful shutdown) разумно брать как «boring defaults» для cloud-native сервисов. citeturn2search3

## Рекомендованные defaults для greenfield template

Ниже — **battle-tested** набор умолчаний, которые стоит зафиксировать в template репозитории и в docs/, чтобы LLM могла строго следовать им без догадок.

### Toolchain и управление зависимостями

**Версия Go и воспроизводимость**
- Зафиксировать версию Go через `go` и `toolchain` директивы в `go.mod`, чтобы локальная разработка и CI не расходились. Механизм toolchains официально поддерживается, начиная с Go 1.21. citeturn13search1turn13search12  
- Обновлять версию Go **по релизному циклу** (примерно раз в 6 месяцев) и помнить про политику поддержки: каждый major релиз поддерживается до появления двух более новых major релизов, а security/backport фокусируется на двух последних ветках. citeturn13search25turn13search4turn13search0

**Supply chain по умолчанию**
- В CI обязательно запускать `govulncheck` как low-noise поиск уязвимостей, который пытается сузить отчёт до реально вызываемых уязвимых функций. citeturn5search3turn5search15turn5search38  
- Использовать Go module proxy и checksum database как дефолтный механизм скачивания/аутентификации модулей (и документировать настройки для приватных модулей). citeturn20search2turn20search17

### Структура проекта и границы пакетов

**Layout**
- Опирайтесь на официальный подход организации модуля: отдельные каталоги для пакетов, `internal/` для приватного кода, отдельные директории для нескольких команд/бинарей. citeturn16search7  
- В шаблоне микросервиса обычно достаточно:
  - `cmd/<service>/main.go` — тонкий entrypoint;
  - `internal/` — всё, что не является публичной библиотекой;
  - `api/` или `openapi/` / `proto/` — контракты и генерируемые артефакты (по правилам репозитория).
  
**Coding style как норматив**
- Минимальный нормативный слой: **Go Code Review Comments** + **Google Go Style Guide** (как “no-guess” документ для LLM) и “Effective Go” как базовый фон. citeturn16search2turn16search4turn16search12  
- Это важно именно для LLM: модель должна иметь «единственный источник правды» по именованию, ошибкам, структуре, комментариям, экспортируемым API и т.д. citeturn16search0turn16search1

### HTTP runtime: timeouts, контексты, shutdown

**net/http как boring default**
- По умолчанию используйте стандартный `net/http` (и при необходимости тонкий router). Причина: меньше магии, проще профилировать и проще модели генерировать корректный код. `net/http` явно документирует критические инварианты (например, нельзя использовать `ResponseWriter` после завершения `ServeHTTP`). citeturn4view0

**Timeouts MUST**
- Любой production HTTP server обязан иметь выставленные server timeouts. `net/http.Server` прямо подчёркивает смысл `ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout`, `IdleTimeout`, а также то, что `ReadHeaderTimeout` часто предпочтительнее `ReadTimeout`, потому что даёт handler’ам контроль над body. citeturn4view0turn4view1

**Контекст MUST**
- Все операции I/O и внешние вызовы (DB, HTTP client, брокеры) выполняются с `context.Context`. Пакет `context` предупреждает, что если вы не вызываете `CancelFunc`, можно удерживать ресурсы до отмены родителя/дедлайна; `go vet` проверяет это. citeturn3search13turn2search2  
- Начиная с Go 1.8, контекст `Request.Context()` отменяется при закрытии соединения, что критично для корректной отмены работы, если клиент «ушёл». citeturn4view1

**Graceful shutdown MUST**
- Использовать `http.Server.Shutdown(ctx)` (доступно с Go 1.8): он закрывает listeners, закрывает idle connections и ждёт завершения активных запросов до истечения контекста. citeturn14search0turn4view1

### Observability defaults

**Логи**
- Использовать структурное логирование на базе `log/slog` (стандартная библиотека, Go 1.21), чтобы логи были машинно-обрабатываемыми (фильтрация, поиск, корреляция). citeturn13search2turn13search6  
- Согласовать формат и поля логов с эксплуатацией (request_id/trace_id, service.name, environment, version). Логи трактовать как поток событий. citeturn2search3  
- Обязательно запретить утечки секретов/PII в логах и учитывать риски log injection; это отражено в рекомендациях OWASP по логированию. citeturn22search3turn22search7

**Метрики**
- Базовый дефолт: expose `/metrics` для Prometheus и использовать официальный Go client. citeturn14search8turn14search2turn1search28  
- entity["organization","Prometheus","cncf monitoring project"] — зрелый, широко используемый стек мониторинга (CNCF graduated). citeturn17search1  

**Трейсинг**
- Использовать OpenTelemetry SDK и обёртки для `net/http` (`otelhttp`) для distributed tracing. citeturn14search1turn14search7  
- entity["organization","OpenTelemetry","cncf observability project"] — CNCF incubating проект; спецификация и SDK создают vendor-neutral слой, но важно помнить про статус сигналов (например, в документации по Go отмечается, что logs signal может быть экспериментальным). citeturn18view0turn14search4turn1search27  

### Security defaults

**Базовая модель угроз**
- В качестве минимального «чек-листа рисков» для API принять entity["organization","OWASP","application security org"] API Security Top 10 (2023) как общий ориентир классов уязвимостей и зон контроля. citeturn1search29  
- Для разработки: обязательны input validation, корректная аутентификация/авторизация, защита от инъекций, корректные логи/аудит. citeturn22search10turn22search2turn22search7  

**Секреты**
- В шаблоне: запрет на хранение секретов в репозитории; документировать стратегию secrets management (источник секретов, ротация, аудит). citeturn22search24turn2search3  

**SQL и инъекции**
- Дефолтная позиция: запрещены SQL-строки через конкатенацию пользовательского ввода; только параметризация / подготовленные выражения. citeturn22search2turn22search6  

### Container и Kubernetes-ready эксплуатация

**Dockerfile**
- Multi-stage builds как дефолт: уменьшение размера и attack surface — официальная рекомендация Docker. citeturn19search0turn19search32  
- entity["company","Docker","container tooling company"] — ссылаться на официальный гайд по best practices при ревью Dockerfile. citeturn19search0  
- В качестве дефолтного рантайм-образа рассмотреть distroless: он содержит только приложение и runtime-зависимости без shell и package manager. citeturn19search1turn19search32

**Kubernetes probes**
- README/runbook должны описывать endpoints для startup/readiness/liveness и их смысл: readiness — готовность принимать трафик; liveness — «надо ли рестартовать»; startup — задержка до начала liveness/readiness для долгого старта. citeturn2search4turn2search0  
- entity["organization","Kubernetes","container orchestration project"] — источник истинных определений и примеров конфигурации probes. citeturn2search4  

**SecurityContext**
- Документировать и предустанавливать securityContext (например, запрет privileged, минимум прав), используя официальные рекомендации Kubernetes. citeturn19search3  

## Матрица решений и trade-offs

Ниже — решения, которые чаще всего «ломают» шаблон из-за неявных компромиссов. Их нужно фиксировать в template как ADR (architecture decision record) и как LLM-правила.

| Тема | Boring default | Когда менять | Основные trade-offs |
|---|---|---|---|
| HTTP stack | `net/http` + минимальный router | Сложные middleware, автогенерация REST, websockets, специфичные требования | `net/http` даёт меньше магии и чёткие инварианты; фреймворки ускоряют старт, но повышают неопределённость для LLM и усложняют контроль таймаутов/контекстов. citeturn4view0turn14search0 |
| Контракты | REST+OpenAPI **или** gRPC+Protobuf (выбрать явно) | Нужна polyglot интеграция, строгая схема, streaming | Protobuf проще эволюционировать additively (при соблюдении правил), но сложнее дебажить и требует codegen. citeturn6search28turn6search1turn7search2 |
| Метрики | Prometheus `/metrics` | Вы строите полностью OTel-native pipeline | Prometheus — зрелый дефолт (CNCF graduated); OTel даёт унификацию сигналов, но экосистема метрик имеет нюансы и «не всегда 1:1». citeturn17search1turn1search28turn1search27 |
| Трейсы | OpenTelemetry + `otelhttp` | Нет distributed tracing, жёсткий запрет зависимостей | OTel — vendor-neutral и стандартизирует трейсинг; важно учитывать стабильность сигналов/semconv и стоимость внедрения. citeturn14search1turn18view0turn1search27 |
| Логи | `log/slog` JSON handler (структурно) | Жёсткие perf-требования / единый корпоративный backend | `slog` стандартизирует API и структуру; сторонние логгеры могут быть быстрее/богаче, но добавляют вариативность. citeturn13search2turn13search6 |
| Контейнерный образ | Multi-stage + минимальный runtime (в т.ч. distroless) | Нужен shell для debug в production (обычно нежелательно) | Меньше инструментов в образе → меньше attack surface, но сложнее live-debug; компенсируется debug-образом/ephemeral containers. citeturn19search32turn19search1 |
| Версионирование API | Major-only (v1, v2), политика deprecation | Если все клиенты под вашим контролем, возможно «без версий» | Публичные/внешние API требуют строгой совместимости в пределах major; «тихие» breaking changes недопустимы. citeturn10view0turn12view0turn5search2 |

## Правила для LLM

Этот раздел нужно почти напрямую переносить в `docs/llm-instructions.md` и использовать как общий префикс/системное сообщение для модели.

### MUST

**Про репозиторий и источники истины**
- MUST считать **контракты и их файлы** (OpenAPI/Proto) источником истины для API: код подстраивается под контракт, а не наоборот. citeturn7search2turn6search0  
- MUST следовать структуре модуля и границам `internal/` согласно выбранному layout; новые пакеты добавлять только если это соответствует `go.dev` рекомендациям и репо-конвенциям. citeturn16search7  
- MUST следовать Go Style Guide/Code Review Comments в именовании, комментариях, экспорте символов, ошибках. citeturn16search4turn16search2  

**Про HTTP и контексты**
- MUST: каждый handler обязан соблюдать инвариант `net/http`: после возврата из `ServeHTTP` нельзя писать в `ResponseWriter` и читать `Request.Body`. citeturn4view0  
- MUST: серверные таймауты (`ReadHeaderTimeout`, `IdleTimeout`, и т.д.) должны быть заданы осознанно; предпочтение отдавать `ReadHeaderTimeout`, потому что handler может контролировать body. citeturn4view0turn4view1  
- MUST: использовать `Request.Context()` и пробрасывать `context.Context` во все долгие операции; при использовании `context.WithTimeout/WithCancel` всегда вызывать cancel на всех путях выполнения (во избежание утечек). citeturn3search13turn2search2  
- MUST: при shutdown использовать `http.Server.Shutdown(ctx)` с bounded timeout и корректно обрабатывать `ErrServerClosed`. citeturn14search0turn4view1  

**Про observability**
- MUST: логирование структурное; лог-события должны иметь стабильные ключи (уровень, сообщение + поля) и не содержать секретов/PII. citeturn13search2turn22search3turn22search24  
- MUST: в сервисе должен существовать endpoint метрик `/metrics` при выборе Prometheus. citeturn14search8turn1search28  
- MUST: при включённом tracing оборачивать server/client handlers через `otelhttp` и сохранять контекст для корреляции. citeturn14search1turn14search7  

**Про безопасность**
- MUST валидировать входные данные (schema/ограничения) на границе сервиса, а не «где-то внутри». citeturn22search10  
- MUST предотвращать SQL injection через параметризацию (никакой конкатенации пользовательского ввода в запрос). citeturn22search2turn22search6  
- MUST запускать `govulncheck` в CI и исправлять/обосновывать найденные уязвимости. citeturn5search3turn5search15  

### SHOULD

- SHOULD иметь health endpoints, согласованные с liveness/readiness/startup probes и их смыслом. citeturn2search4  
- SHOULD писать тесты в стандартном стиле `go test` и использовать рекомендации Go Test Comments для качества тестов/сообщений об ошибках. citeturn15search0turn20search1  
- SHOULD включать fuzzing для парсеров, декодеров, нормализаторов и иной логики обработки входных данных (начиная с Go 1.18 это часть toolchain). citeturn20search0turn20search6  
- SHOULD запускать race detector в CI для concurrency-heavy кода (хотя бы nightly/по метке), потому что data races тяжело отлаживаются и приводят к крашам/коррупции памяти. citeturn21search0turn21search4  
- SHOULD настраивать пул соединений `database/sql` (лимиты, idle/lifetime) и помнить, что лимит может превратить использование БД в «семантику семафора» и привести к дедлокам при неправильном использовании. citeturn22search4turn22search0  
- SHOULD использовать multi-stage Docker builds и минимальный runtime image. citeturn19search32turn19search0  

### NEVER

- NEVER менять существующий API «тихо» (семантика/дефолты/формат значений) без прохождения процесса совместимости и без документации. Это прямо запрещено как breaking change практиками обратной совместимости: менять дефолт, сериализацию дефолтов, формат значений нельзя в пределах major. citeturn12view0  
- NEVER удалять/переименовывать API элементы в пределах major: поля/методы/enum values — только депрекейт и миграция. citeturn12view0turn10view0  
- NEVER писать секреты в логи и считать данные для логов «доверенными» (риск log injection и утечек). citeturn22search3turn22search7turn22search24  
- NEVER добавлять «required» поля в существующие запросы/ресурсы без новой major версии или без очень жёстко описанного поведения по умолчанию (de facto breaking). citeturn12view0  

### Good / bad примеры на Go

Хороший пример: timeouts, контекст, shutdown, структурные логи (минимальный skeleton). citeturn4view0turn14search0turn13search2turn3search13

```go
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		// Быстрый liveness: без зависимостей.
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		// Readiness может делать лёгкую проверку критичных зависимостей.
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       0,               // body контролируем на уровне handler при необходимости
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server starting", "addr", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		// graceful shutdown must be bounded
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		logger.Info("http server shutting down")
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown failed", "err", err)
		}
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "err", err)
		}
	}
}
```

Плохой пример: отсутствуют таймауты, игнорируется cancel, нет корректного shutdown. citeturn4view0turn3search13turn14search0

```go
// ПЛОХО: нет timeouts, cancel игнорируется, нет Shutdown.
func main() {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second) // cancel потерян
	_ = ctx

	http.ListenAndServe(":8080", http.DefaultServeMux) // блокирует навсегда, нет graceful shutdown
}
```

## Checklist для PR/review и что оформить файлами в template repo

### Review checklist

Этот список нужен как `docs/review-checklist.md` и как `pull_request_template.md`.

- Контракт и совместимость: нет ли breaking change (включая «скрытые» — дефолты, сериализация дефолтов, изменение формата поля, семантика поведения). citeturn12view0  
- HTTP: выставлены ли server timeouts; корректно ли используется `Request.Context()`; нет ли записи в `ResponseWriter` после return. citeturn4view0turn14search0  
- Shutdown: есть ли `Server.Shutdown(ctx)` с bounded timeout и корректной обработкой `ErrServerClosed`. citeturn14search0  
- Логи: структурные; нет секретов/PII; учтён риск log injection; есть ли достаточные поля для расследований. citeturn13search6turn22search3turn22search7  
- Метрики/трейсы: есть ли `/metrics` (если Prometheus); корректная OTel-инструментация (если включена). citeturn14search8turn14search1  
- Security: входные данные валидируются; SQL параметризован; secrets не в коде/логах; `govulncheck` зелёный. citeturn22search10turn22search2turn5search3turn22search24  
- БД: пул соединений настроен и задокументирован; нет риска дедлока из‑за MaxOpenConns и неправильного использования транзакций/коннектов. citeturn22search4turn22search0  
- Тесты: есть покрытие критических веток; тесты читаемые; при необходимости добавлены fuzz/race проверки. citeturn20search1turn20search0turn21search0  
- Контейнер: Dockerfile multi-stage; минимальный runtime; probes и securityContext согласованы с эксплуатацией. citeturn19search32turn2search4turn19search3  

### Что вынести в отдельные файлы шаблона

Ниже — рекомендуемая «раскладка» документов и конфигов, из которых LLM получает контекст и ограничения.

`docs/`
- `docs/engineering-standard.md` — этот стандарт: runtime, структуру, CI, observability, security.
- `docs/llm-instructions.md` — MUST/SHOULD/NEVER + правила совместимости.
- `docs/api-evolution-governance.md` — политика версионирования, депрекейта, sunset, процесс PR.
- `docs/observability.md` — обязательные метрики/логи/трейсы, поля, naming, примеры.
- `docs/security.md` — секреты, input validation, инъекции, логирование, минимальные требования. citeturn22search24turn22search10turn22search2turn22search3  
- `docs/runbook.md` — как запускать/диагностировать: probes, `/metrics`, shutdown, типовые алерты. citeturn2search4turn14search8turn14search0  
- `docs/adr/` — решения из матрицы trade-offs (как минимум про API стиль и версионирование). citeturn16search2  

Root / config
- `go.mod` с `go` + `toolchain` (воспроизводимость). citeturn13search12turn13search1  
- `.github/workflows/ci.yml`: `go test`, `go vet`, форматирование, `govulncheck`. citeturn15search13turn5search3  
- `Dockerfile` multi-stage. citeturn19search32turn19search0  
- Kubernetes manifests/Helm values (опционально, но полезно): probes + securityContext. citeturn2search4turn19search3  

## Governance: versioning, backward compatibility, deprecation и эволюция API contracts

Этот раздел следует практически напрямую положить в `docs/api-evolution-governance.md` и «прибить гвоздями» в PR-процесс.

### Принципы совместимости и определения

**Определение breaking change (практическое)**
- Breaking change — любое изменение, которое может сломать существующий клиент при обновлении сервера (в пределах одной major версии API). Это включает не только схему, но и семантику (поведение) и сериализацию. citeturn12view0  
- Даже когда «кажется, что изменение additive», оно может быть breaking для части клиентов (например, строгие JSON-парсеры). Поэтому политика должна явно определять «что считаем совместимым», и LLM должна трактовать сомнительные изменения как потенциально breaking. citeturn12view0turn7search9

**Три измерения совместимости (полезно фиксировать в PR)**
- Source compatibility (клиентский код компилируется/генерируется).
- Wire compatibility (старые клиенты корректно общаются с новым сервером).
- Semantic compatibility (ожидаемое поведение не меняется «удивляющим» образом). citeturn12view0

### Стратегия версионирования API

**Boring default для template**
- Использовать **только major версии API**: `v1`, `v2`, без `v1.1`/`v1.2` на внешнем контракте, если вы не готовы поддерживать матрицу совместимости и множественные ветки поведения. Такой подход отражён в практике крупных API-провайдеров: major обновляется «in place» совместимыми изменениями. citeturn10view0turn5search2  
- Для REST размещать major версию в начале URI path (`/v1/...`) и отражать её в контракте и документации. citeturn10view0  
- Для gRPC/Protobuf отражать major версию в `package` (например `foo.v1`) и не допускать несовместимых протокольных изменений, даже при major bump (требование совместной работы старых и новых). citeturn6search1turn10view0

**Trade-offs URL versioning vs media-type versioning**
- URL versioning (`/v1`) проще обнаруживать, документировать, логировать и маршрутизировать, но «засоряет» URI и может приводить к version sprawl.
- Media-type/header versioning опирается на `Accept`/content negotiation, но повышает сложность для клиентов, кэшей и отладки; требует дисциплины `Vary: Accept` и строгой инфраструктуры. HTTP семантика `Accept` и content negotiation описана в RFC 9110, но это не равно «удобной политике версий» — нужно отдельно определить правила. citeturn7search3turn7search31

### Политика deprecation и sunset

**Runtime-сигналы депрекейта**
- Использовать стандартный HTTP заголовок `Deprecation` (RFC 9745) для уведомления клиентов о том, что ресурс/endpoint депрекейтится. Важно: сам заголовок **не меняет поведение** ресурса, он только сигнализирует жизненный цикл. citeturn5search5  
- Использовать `Sunset` (RFC 8594), чтобы заранее сообщать момент, когда ресурс станет недоступен. citeturn5search19  

**Окна депрекейта**
- Для beta поверхностей разумный boring default — не менее ~180 дней до удаления (это прямо предлагается как рекомендация в версиях для beta каналов). citeturn10view0  
- Для stable/GA поверхностей окно обычно должно быть больше и определяется продуктовой/контрактной политикой; но даже для внутренних API окно должно быть явно прописано (иначе LLM «удалит поле», считая это уборкой). citeturn10view0turn12view0

### Правила эволюции контрактов: совместимые vs несовместимые изменения

#### REST/JSON/OpenAPI

**Additive changes (обычно совместимы, но требуют осторожности)**
- Добавление новых optional полей в responses часто считается совместимым, если клиенты игнорируют неизвестные поля; этот подход встречается в рекомендациях по web API дизайну. citeturn7search9  
- Добавление новых endpoints/ресурсов обычно совместимо.

**Potentially breaking even if “looks harmless” (LLM должна считать риск)**
- Изменение дефолтов (поведение ресурса меняется через server-side defaults). citeturn12view0  
- Изменение формата значений (например, строка IPv4 → IPv6) или алгоритма формирования значения. citeturn12view0  
- Добавление обязательности (required) для существующих параметров/полей.
- Изменение сериализации дефолтов: раньше поле отсутствовало, теперь приходит с дефолтным значением (или наоборот). citeturn12view0  
- Изменение поведения pagination/ordering/filtering так, что старые клиенты получают меньше/по-другому (частный случай прямо разобран как риск). citeturn12view0  

**OpenAPI как контрактный артефакт**
- OpenAPI Specification сама по себе использует semver и допускает редкие несовместимости даже в minor версий спецификации, поэтому для вас важно не «верить» автоматически minor/patch OpenAPI как гарантии совместимости вашего API; совместимость — ваша политика и ваши diff‑гейты. citeturn7search10turn7search2  

#### Protobuf/gRPC

**Базовые гарантии**
- При соблюдении «простых практик» старый код читает новые сообщения, игнорируя неизвестные поля; удалённые поля будут иметь дефолтные значения. citeturn6search28  

**Жёсткие правила эволюции**
- Нельзя переиспользовать номера удалённых полей: нужно добавлять их в `reserved`, иначе возможна порча данных и тяжёлые ошибки. citeturn6search18  
- Некоторые изменения схемы ведут к потере данных или некорректной интерпретации (например, `repeated` → `scalar`). citeturn6search3  
- В пределах major версии нельзя удалять/переименовывать компоненты и менять типы: это ломает сгенерированный код и/или семантику. citeturn12view0  

**gRPC совместимость**
- Даже при major bump gRPC подчёркивает необходимость сохранения совместимости протокола: старые клиенты должны продолжать взаимодействовать с новыми серверами и наоборот; несовместимые протокольные изменения запрещены. citeturn6search1  

### Consumer-driven contract testing

Boring default для межкомандной микросервисной среды: внедрять consumer-driven contract testing в стиле Pact, где consumer формализует ожидания тестом и публикует контракт, а provider верифицирует его в CI, предотвращая breaking changes до релиза. citeturn6search2  

### Процесс принятия решений и оформление PR/release notes

**В PR обязательно**
- Явно указать тип изменения контракта: `additive`, `behavioral`, `breaking`, `deprecation-only`.
- Для `deprecation` приложить:
  - дату и причины,
  - план миграции,
  - дату `Sunset`,
  - появление заголовков `Deprecation`/`Sunset` в ответах (если HTTP). citeturn5search5turn5search19  
- Для любого изменения, затрагивающего «дефолты/формат/сериализацию», считать это потенциально breaking и требовать явного решения (либо новая major, либо expand/contract миграция). citeturn12view0  

**В release notes обязательно**
- Раздел “API Changes”: что добавили, что депрекейтнули, что удалили.
- Для депрекейтов — сроки и заголовки `Deprecation`/`Sunset`, и ссылка на миграционный гайд (внутренний).

**LLM должна считать breaking (минимальный список-триггер)**
- Любое удаление/переименование поля/метода/endpoint. citeturn12view0  
- Любая смена дефолта или сериализации дефолтов. citeturn12view0  
- Любая смена формата значения (даже если тип “string” не менялся). citeturn12view0  
- Любая смена типа protobuf поля или перемещение в/из `oneof`. citeturn12view0  
- Любое переиспользование protobuf tag number или смена `repeated`/`scalar`. citeturn6search18turn6search3  
- Любое поведенческое изменение, которое может удивить «разумного разработчика» (semantic break). citeturn12view0  

**Примечание о спорных моментах (фиксировать как policy)**
- «Добавление нового JSON поля в response» — в одних организациях считается совместимым (клиенты игнорируют), в других — нет (строгие модели/генераторы). Поэтому это должно быть явно зафиксировано как ваша политика, иначе LLM будет действовать по “средней температуре по больнице”. citeturn7search9turn12view0