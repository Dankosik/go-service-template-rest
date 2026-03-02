# Engineering standard и LLM‑instructions для production‑ready template микросервиса на Go

## Scope

Этот подход уместен, когда вы делаете **greenfield микросервис** (обычно HTTP/gRPC), который будет жить в контейнерах, деплоиться через CI/CD и эксплуатироваться вместе с централизованными логами/метриками/трейсами, а разработчики активно используют LLM‑инструменты для генерации и рефакторинга кода. В таком контуре особенно важно **уменьшить пространство догадок** для LLM: жестко закрепить технологические решения, конвенции, требования к отказоустойчивости и наблюдаемости. citeturn13search0turn2search3turn8search6turn8search1turn4search12

Подход **не подходит как “универсальный шаблон на все случаи”**, если:
- вы пишете **сверхнизколатентный** сервис (HFT/Ultra‑low‑latency), где многие “boring defaults” (JSON, универсальная observability‑прослойка) не пройдут по бюджету латентности и нужно проектировать под конкретную среду; citeturn10search3turn8search3  
- у вас сервис — это **встроенный компонент** (embedded), или требуется нестандартная платформа/ABI или строгие ограничения на runtime/GC; тогда придется менять базовые исходные предположения (инструментирование, контейнеризация, профилирование, shutdown‑flow). citeturn11view0turn13search0  
- у вас **монолит** или бэк‑офисное batch‑приложение: часть стандартов сохраняется (код‑стайл, тестирование, supply chain), но шаблон микросервиса (probes, graceful termination, SLO‑метрики, cache‑fallback‑контракты) будет либо лишним, либо неполным. citeturn2search3turn17view0turn4search1

Ключевая задача (как нормативное требование): **человек клонирует репозиторий и сразу получает “правильные рельсы”**: минимальный набор обязательных компонентов production‑сервиса (timeouts, shutdown, probes, metrics/tracing/logging, тестовые контуры, supply‑chain проверки), плюс LLM‑инструкция, которая заставляет модель генерировать идиоматичный, компилируемый, безопасный, наблюдаемый Go‑код без “галлюцинаций” по зависимостям и архитектуре. citeturn1search4turn1search1turn3search8turn15search0turn8search1

## Recommended defaults для greenfield template

Ниже — набор “boring, battle‑tested defaults” для шаблона. Это можно прямо переносить в `docs/standards/*` и `repo conventions`.

### Базовая версия Go и политика совместимости

- **Версия toolchain**: фиксировать сборку на последнем стабильном релизе Go на момент создания шаблона (на сегодня это Go 1.26, февраль 2026). citeturn0search4turn11view0  
- **Поддерживаемые версии**: ориентироваться на официальную политику: релиз поддерживается до появления двух более новых major‑релизов; security‑фиксы готовятся для двух последних major‑веток. Это означает: для продакшена “по умолчанию” логично поддерживать **N и N‑1**. citeturn13search0turn13search4  
- **go.mod**: `go`‑директива должна отражать **минимально поддерживаемую** версию Go, а при желании воспроизводимости можно использовать механизм toolchain selection, чтобы разработчик/CI не “случайно” собирал другим компилятором (детали — в официальной документации по toolchains и `go`‑directive). citeturn13search7turn13search6turn11view0

Практичный default для шаблона (как правило, оптимально):  
- `go` = N‑1 (минимум),  
- `toolchain` = N (фактическая сборка в CI/локально),  
чтобы не требовать от всех разработчиков мгновенно обновляться, но иметь предсказуемую дефолтную сборку. При этом важно понимать, что в новых Go‑версиях поведение `go mod init` и выбор `go`‑версии могут изменяться, и это уже документировано в release notes. citeturn11view0turn13search7turn13search6

### Архитектурный каркас и layout репозитория

- **Modules**: только Go modules, никаких GOPATH‑предположений. citeturn3search8turn3search0  
- **Мульти‑binary layout**: официальный паттерн — отдельные директории для программ и общий `internal/` для разделяемых пакетов. Это прямо описано в “Organizing a Go module”. citeturn20search2  
- **Границы и инкапсуляция**: использовать `internal/` для ограничения импортов и сужения API‑поверхности; это поведение контролируется `go` toolchain и снижает риск “случайных” зависимостей. citeturn20search2turn20search17

Рекомендованный “универсальный” layout для шаблона (как конвенция репозитория, совместимая с go.dev guidance):  
- `cmd/<service>/main.go` (точка входа)  
- `internal/app` (инициализация зависимостей, wiring, lifecycle)  
- `internal/http` или `internal/transport/http` (router/handlers/middleware)  
- `internal/domain` (модели/правила домена — если вы действительно используете доменный слой)  
- `internal/storage` (DB/cache клиенты и репозитории)  
- `internal/observability` (метрики/трейсы/лог‑корреляция)  
- `internal/config` (чтение env/flags + валидация)  
- `docs/` (стандарты, гайды, runbooks)

### API протоколы и контракты

Default‑позиция для greenfield:  
- **HTTP/JSON как внешний контракт** (экосистема инструментов, простота, OpenAPI). Спецификация OpenAPI — стандартное описание HTTP API. citeturn7search4turn7search8  
- **gRPC как внутренний контракт** (если есть межсервисные RPC и нужна строгая схема/генерация). Для production‑сценариев важны стандартизированные health checks (gRPC health checking protocol). citeturn7search1turn7search5  

Как правило, шаблон должен поддерживать оба режима (HTTP‑сервис сразу, gRPC — опциональный модуль), но при этом **строго фиксировать** выбор в конкретном сервисе через repo‑конфиг (иначе LLM будет “размазывать” архитектуру). citeturn7search4turn7search1

### Конфигурация и runtime‑поведение

- **Config через env** (12‑factor): конфигурация, отличающаяся между деплоями, должна быть в переменных окружения; это снижает риск утечек через репозиторий и делает деплой переносимым. citeturn4search12  
- **Graceful shutdown**:  
  - останавливать HTTP‑сервер через `Server.Shutdown(ctx)`, который прекращает принимать новые соединения и ожидает завершения активных. citeturn0search17  
  - использовать `signal.NotifyContext` для завершения по сигналам ОС (в Go 1.26 — с причиной отмены, содержащей информацию о сигнале). citeturn3search1turn12view0  
  - учитывать контейнерный shutdown‑flow: в k8s kubelet сначала посылает SIGTERM с grace period, затем SIGKILL, порядок остановки контейнеров может быть произвольным, и `PreStop` hook выполняется до отправки TERM и учитывается в grace period. citeturn6view2turn5view1  

- **HTTP server hardening**: по умолчанию задавать таймауты и лимиты (например `ReadHeaderTimeout`, `IdleTimeout`, `MaxHeaderBytes`) на уровне `http.Server`, чтобы снизить риск зависаний и медленных атак, и лимитировать размер тела запроса через `MaxBytesReader`. citeturn3search10turn16search0  
- **JSON parsing hardening**: для входящих JSON‑payload использовать `json.Decoder` и включать `DisallowUnknownFields`, чтобы избежать “молчаливого” принятия неожиданных полей (которые часто становятся источником багов и уязвимостей при эволюции API). citeturn3search3  

### База данных и внешние HTTP‑клиенты

- **database/sql как базовый слой для SQL**: `sql.DB` — это handle к пулу соединений и безопасен для конкурентного использования; его нужно создавать один раз и настраивать пул. citeturn21search0turn21search1  
- **http.Client reuse**: `http.Client` и его транспорт должны переиспользоваться (внутреннее состояние/keep‑alive); это стандартная рекомендация документации. citeturn21search13  

### Observability по умолчанию

Набор “минимально достаточных” сигналов для production:

- **Структурированные логи**: использовать `log/slog` как стандартную библиотеку для structured logging; это уменьшает зависимость от сторонних логгеров и делает формат стабильным на уровне языка. citeturn2search6turn2search2  
- **Метрики**: использовать практики именования/лейблов Prometheus (низкая кардинальность, осмысленные имена), иначе метрики становятся эксплуатационно дорогими и бесполезными. citeturn1search3turn1search11  
- **Трейсинг**: использовать entity["organization","OpenTelemetry","observability framework"] и семантические конвенции, но фиксировать версию и помнить, что некоторые области semconv имеют статус Mixed/Development и могут эволюционировать. citeturn8search5turn0search10turn10search7turn10search2  
- **Контекст трассировки**: поддерживать propagation через стандарт W3C trace context (`traceparent`/`tracestate`). citeturn8search0  
- **Профилирование**: `net/http/pprof` подключать как опциональный debug‑endpoint и закрывать его от внешнего мира (trust boundary), поскольку он публикует runtime‑профили. citeturn16search3  

Важно: в документации OpenTelemetry прямо отмечается, что сигнал **logs** может оставаться экспериментальным и подверженным breaking changes — поэтому в boring defaults логирование стоит вести через `slog`, а OTel‑логи подключать только при явной потребности и принятии риска изменений. citeturn8search9turn2search6

### Security и supply chain по умолчанию

- Модель угроз для API должна учитывать entity["organization","OWASP","security nonprofit"] API Security Top 10 (например, BOLA и др.) как чек‑лист типовых рисков. citeturn0search3turn0search7  
- Базовые гайды OWASP Cheat Sheet Series для логирования, секретов и DoS‑контролей — как минимум:  
  - не логировать чувствительные данные и проектировать security‑logging осознанно; citeturn4search3  
  - управлять секретами (минимизация доступа, ротация, алерты на misuse); citeturn4search7  
  - rate limiting и ограничения размеров/ресурсов против DoS; citeturn16search1turn16search5turn16search9  
- Для Go‑экосистемы обязательный tool: `govulncheck` как low‑noise анализатор уязвимостей зависимостей, использующий официальный Go vulnerability database. citeturn15search0turn15search1turn15search2  
- Supply chain baseline:  
  - ориентир на SLSA‑подход к гарантии целостности сборки (как минимум — понимание уровней и provenance); citeturn4search1  
  - автоматизированная оценка posture через entity["organization","OpenSSF Scorecard","oss security checks"] (как инструмент контроля безопасных практик репозитория). citeturn4search14turn4search2  

### Контейнеризация

- Делать Dockerfile по best practices: multi‑stage builds, чтобы финальный образ содержал только runtime‑артефакты и был меньше/безопаснее. citeturn7search2turn7search6  
- Опираться на OCI image format (интероперабельность инструментов). citeturn7search3turn7search11  

### Quality gates в CI

Минимально обязательные проверки:
- `gofmt` как единый форматтер; citeturn15search3turn15search6  
- `go test` (unit + integration); пакет `testing` — стандартный механизм; citeturn2search0  
- race detector: `go test -race` для конкурентного кода (находит гонки в реально исполняемых путях); citeturn14search0  
- `go vet` как базовый статический анализ; citeturn14search3  
- fuzzing (там, где есть парсинг/валидаторы/сериализация) как дополнительная техника нахождения краевых случаев и уязвимостей; citeturn14search2turn14search6  
- `govulncheck` как обязательный security gate. citeturn15search0turn15search1  

## Decision matrix / trade-offs

Ниже — матрица решений, которую стоит сохранить как “decision records” (или хотя бы таблицы) в документации шаблона, чтобы LLM не “додумывала” альтернативы.

### Транспорт и контракт

| Выбор | Когда default | Плюсы | Минусы / риски |
|---|---|---|---|
| HTTP/JSON + OpenAPI | внешний API, интеграции, простой вход | стандартная спецификация API, хорошая совместимость инструментов citeturn7search4 | нет строгой схемы на уровне wire (в сравнении с protobuf), больше runtime‑валидации |
| gRPC/protobuf | внутренние RPC, строгая схема, производительность | стандартизированный health check протокол citeturn7search1turn7search5 | сложнее дебажить без tooling, нужен контроль версий proto |

### Observability: “единый стандарт” или “boring split”

| Выбор | Default‑рекомендация | Почему | Trade‑off |
|---|---|---|---|
| Logs: slog; Traces: OpenTelemetry; Metrics: Prometheus | boring split | `log/slog` — stdlib structured logging citeturn2search6; OTel — стандарт де‑факто для трейсинга, есть Go‑инструментация citeturn8search5turn8search17; Prometheus best practices для метрик стабилизированы и хорошо документированы citeturn1search3turn1search11 | “две экосистемы” (OTel+Prom) требуют аккуратной стыковки (лейблы/семантика) |
| Полностью OpenTelemetry (logs+metrics+traces) | опционально | единый pipeline и семантические конвенции citeturn0search10turn8search1 | logs сигнал может быть экспериментальным и меняться citeturn8search9; semconv в некоторых областях “Mixed/Development” citeturn10search7turn10search2 |

### JSON библиотека

| Выбор | Default | Почему | Когда менять |
|---|---|---|---|
| `encoding/json` | да | стандартная библиотека, понятные правила, поддержка `DisallowUnknownFields` citeturn3search3 | если нужна специализация/скорость — рассматривать отдельно, но фиксировать решение документом |
| экспериментальные JSON API | нет по умолчанию | экспериментальные пакеты могут менять API citeturn3search20 | только если вы сознательно принимаете риск и закрепляете версию/поведение |

### Caching strategy

| Выбор | Default | Плюсы | Минусы / причины отказаться |
|---|---|---|---|
| Cache‑aside (lazy caching) + TTL | да | наиболее распространенный паттерн; кэш заполняется только по спросу citeturn19view0 | риски stampede/thundering herd при TTL/пустом узле — нужен singleflight/locks, prewarm и jitter TTL citeturn19view0 |
| Write‑through | точечно | избегает промахов при очевидно “горячих” данных citeturn19view0 | может заполнить кэш ненужными ключами и создать churn citeturn19view0 |

### Shutdown и probes в Kubernetes

| Выбор | Default | Почему | Риск |
|---|---|---|---|
| Readiness + Liveness + Startup (при необходимости) | да | kubelet использует probes для удаления из балансировки и для рестарта (но с caution про cascading failures) citeturn5view2turn5view0 | неправильный liveness может провоцировать каскадные рестарты под нагрузкой citeturn5view2 |
| “только /health” без разделения | нет | смешение readiness и liveness → неверная реакция системы на деградации | эксплуатационный риск на проде citeturn5view2 |

## Набор правил MUST / SHOULD / NEVER для LLM

Эти правила — ядро вашего `docs/llm-instructions.md`. Они специально написаны так, чтобы **модель не додумывала** окружение и зависимости.

### MUST

**MUST: корректность и компилируемость**
- Генерируемый код **обязан компилироваться** текущим toolchain репозитория и проходить `go test ./...` без скрытых зависимостей и “магических” пакетов. Конвенции эффективного и идиоматичного Go — в Effective Go и Code Review Comments. citeturn1search4turn1search1turn3search8  
- Всегда проверять ошибки и не подавлять их “для красоты”; в Go это фундаментальный паттерн (“errors are values”). citeturn1search0  

**MUST: контекст, таймауты и graceful shutdown**
- Любая операция, которая может блокироваться (HTTP запросы, DB, cache), должна принимать `context.Context` и уважать cancel/deadline (semantics контекста описаны в документации context). citeturn1search6turn1search17  
- HTTP server должен уметь graceful shutdown через `Server.Shutdown(ctx)` и реагировать на SIGTERM/SIGINT через `signal.NotifyContext`. citeturn0search17turn3search1turn12view0  
- Код должен учитывать, что в k8s SIGTERM приходит с grace period, после чего возможен SIGKILL; `PreStop` выполняется до TERM и входит в grace period. citeturn6view2turn5view1  

**MUST: входные данные и DoS‑контроли**
- Любые входящие HTTP‑payload должны иметь лимит размера (напр. `MaxBytesReader`) и контролируемое парсинг‑поведение (строгий JSON decoder при необходимости). citeturn16search0turn3search3  
- Для API должны быть предусмотрены rate limiting/throttling механизмы (уровень сервиса или инфраструктуры), и корректная семантика ошибок (например 429) без утечки внутренних деталей. citeturn16search1turn16search5turn16search9  

**MUST: observability**
- Логи должны быть структурированными через `log/slog` (ключи/значения, корреляция request_id/trace_id). citeturn2search6turn2search2  
- Метрики должны соблюдать правила Prometheus по именованию/лейблам (не вводить неограниченную кардинальность). citeturn1search3turn1search11  
- При наличии трейсинга — использовать OpenTelemetry‑инструментацию для net/http и следовать семантическим конвенциям (понимая статус стабильности отдельных частей). citeturn8search17turn0search2turn10search2turn10search7  

**MUST: security и supply chain**
- Не логировать секреты/PII; следовать OWASP guidance по security logging и secrets management. citeturn4search3turn4search7  
- Любые изменения зависимостей должны проходить `govulncheck` и быть отражены в PR. citeturn15search0turn15search1  

### SHOULD

- SHOULD переиспользовать `http.Client` и не создавать новый на каждый запрос (это явно рекомендовано документацией net/http). citeturn21search13  
- SHOULD использовать `database/sql` как пул соединений, создавать `sql.DB` один раз и настраивать лимиты пула под окружение. citeturn21search0turn21search1  
- SHOULD писать тесты через пакет `testing` (табличные тесты, под‑тесты), применять race detector и при необходимости fuzzing. citeturn2search0turn14search0turn14search2  
- SHOULD учитывать ограничения liveness/readiness в k8s: readiness — для снятия из балансировки, liveness — только для действительно невосстановимых зависаний, иначе возможны каскадные рестарты. citeturn5view2turn5view0  
- SHOULD документировать выбор протокола/кэш‑стратегии и любые отклонения от defaults как “decision record” (чтобы LLM не “разводила” альтернативы). citeturn19view0turn10search7  

### NEVER

- NEVER “изобретать” новые зависимости и пакеты без явного списка разрешенных библиотек (иначе LLM легко галлюцинирует API). Этот запрет — организационный, но он напрямую улучшает воспроизводимость и снижает вероятность ложного кода. citeturn3search8turn1search1  
- NEVER создавать `sql.DB` или `http.Client` на каждый запрос. Документация прямо подчеркивает, что `sql.DB` — пул, а `http.Client` нужно переиспользовать. citeturn21search1turn21search13  
- NEVER игнорировать ошибки, паниковать в обработчиках запросов или возвращать пользователю внутренние stack traces/детали (OWASP предупреждает об утечке внутренней информации в ошибках). citeturn16search5turn1search0  
- NEVER добавлять метрики с высококардинальными лейблами (user_id, email, request_id и т.п.) — это прямо запрещено практиками Prometheus. citeturn1search3turn1search11  
- NEVER включать `/debug/pprof` “наружу” без защиты и явного решения; пакет регистрирует debug‑handlers и предназначен как profiling endpoint. citeturn16search3  

### Concrete good / bad examples

#### Graceful shutdown и корректная реакция на SIGTERM

**Good (идея):** `signal.NotifyContext` + `Server.Shutdown` + таймаут на shutdown; учитывает k8s SIGTERM‑grace. citeturn3search1turn12view0turn0search17turn6view2

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

srv := &http.Server{
	Addr:              cfg.HTTPAddr,
	Handler:           handler,
	ReadHeaderTimeout: 5 * time.Second,
	IdleTimeout:       60 * time.Second,
	MaxHeaderBytes:    1 << 20, // 1 MiB
}

go func() {
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_ = srv.Shutdown(shutdownCtx)
}()

if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
	return fmt.Errorf("http server: %w", err)
}
```

**Bad:** `http.ListenAndServe(addr, handler)` без таймаутов и без shutdown‑обработки → риск зависаний, плохая эксплуатация, проблемы при termination. citeturn3search10turn0search17turn6view2

```go
// Плохая практика: нет таймаутов, нет shutdown.
log.Fatal(http.ListenAndServe(cfg.HTTPAddr, handler))
```

#### Strict JSON + лимит размера тела

**Good:** ограничиваем размер + строгий декодер с запретом неизвестных полей. citeturn16search0turn3search3

```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB

dec := json.NewDecoder(r.Body)
dec.DisallowUnknownFields()

if err := dec.Decode(&req); err != nil {
	http.Error(w, "invalid request", http.StatusBadRequest)
	return
}
```

**Bad:** `json.Unmarshal` без лимита размера и без контроля неизвестных полей → риск DoS по памяти/CPU и “тихие” несовместимости API. citeturn16search0turn3search3

```go
b, _ := io.ReadAll(r.Body)
_ = json.Unmarshal(b, &req)
```

#### Переиспользование http.Client

**Good:** один клиент (или ограниченное число), общий транспорт, читаем тело ответа до конца и закрываем. citeturn21search13

```go
resp, err := httpClient.Do(req)
if err != nil { return err }
defer resp.Body.Close()
io.Copy(io.Discard, resp.Body)
```

**Bad:** `http.Client{}` на каждый запрос → потеря keep‑alive состояния и эксплуатационные проблемы. citeturn21search13

```go
client := &http.Client{}
resp, _ := client.Get(url)
```

## Anti-patterns и типичные ошибки/hallucinations LLM

Ниже — то, что стоит вынести отдельной секцией в `docs/llm-instructions.md` как “запрещенные типовые решения”.

LLM‑типовые ошибки с высокой вероятностью:
- **“Контекст теряется”**: использование `context.Background()` внутри request‑цепочки вместо `r.Context()` или переданного `ctx`, из‑за чего отмена по client disconnect / deadline не работает. Это противоречит назначению `context` как механизма cancellation/deadline. citeturn1search6turn1search17  
- **Нет таймаутов на сервере/клиенте**: отсутствие `ReadHeaderTimeout` и лимитов делает сервис уязвимым к медленным клиентам и зависаниям; shutdown становится непредсказуемым. citeturn3search10turn0search17turn16search0  
- **Неверные probes**: liveness проверяет зависимости (DB/cache) и начинает “убивать” поды при деградациях зависимостей, создавая cascading restarts — Kubernetes прямо предупреждает о каскадных сбоях от неверных liveness. citeturn5view2turn5view0  
- **Метрики с высокой кардинальностью**: request_id/user_id как label → экспоненциальный рост time series. Prometheus это прямо запрещает в practices. citeturn1search3turn1search11  
- **“Открываем DB на каждый запрос”**: LLM может создать `sql.Open` в handler и закрывать после каждого запроса. Это нарушает идею `sql.DB` как пула. citeturn21search0turn21search1  
- **Логирование секретов**: печать токенов/паролей/PII “для дебага”. Это запрещено практиками security logging и secrets management. citeturn4search3turn4search7  
- **Ложная “универсальная” обработка ошибок**: возвращение внутренних деталей клиенту (stack traces, SQL ошибки целиком). OWASP прямо предупреждает о риске раскрытия внутренней информации через ошибки. citeturn16search5turn0search3  
- **Гонки и небезопасная конкуррентность**: запуск горутин без ограничений и без учета отмены, отсутствие race‑прогонов. Race detector официально рекомендуется запускать как минимум на тестах, но он видит только исполняемые пути. citeturn14search0turn1search6  
- **Галлюцинации по OTel semconv**: модель “придумывает” атрибуты/имена метрик. Нужна ссылка на конкретную версию semantic conventions, потому что часть конвенций имеет статус Mixed/Development. citeturn10search7turn0search2turn10search2  

Чтобы минимизировать это, LLM‑инструкция должна включать:  
- список разрешенных библиотек и версий (pinned),  
- требование писать код “в стиле репозитория” (структура пакетов, error wrapping, лог‑ключи),  
- чек‑лист обязательных non‑functional требований (timeouts, metrics, shutdown),  
- запрет на высокорисковые “автодогадки”. citeturn1search1turn1search4turn13search7

## Review checklist для PR/code review и список файлов для template repo

### Review checklist

Это стоит вынести в `docs/review-checklist.md` и (короткой версией) в PR template.

**Build & tooling**
- Код отформатирован `gofmt`. citeturn15search3turn15search6  
- `go test ./...` проходит (включая нужные integration tests). citeturn2search0  
- `go test -race ./...` (или хотя бы для пакетов конкурентного кода) проходит; замечания о том, что race detector покрывает только исполняемые пути, зафиксированы в доках. citeturn14search0  
- `go vet ./...` проходит. citeturn14search3  
- `govulncheck ./...` проходит, изменения зависимостей осознаны. citeturn15search0turn15search1  

**Runtime safety**
- У HTTP‑сервера стоят таймауты/лимиты (`ReadHeaderTimeout`, `IdleTimeout`, `MaxHeaderBytes` и т.п.). citeturn3search10  
- У входящих payload есть лимит (например `MaxBytesReader`), парсинг строгий там, где это нужно. citeturn16search0turn3search3  
- Есть graceful shutdown через `Server.Shutdown` и сигнал‑контекст через `NotifyContext`. citeturn0search17turn3search1turn12view0  
- Учитывается k8s termination flow (SIGTERM → grace → SIGKILL), `PreStop` и различие readiness/liveness. citeturn6view2turn5view2turn5view1  

**Observability**
- Логи структурированные через `log/slog`, ключи стабильны, чувствительные данные не логируются. citeturn2search2turn4search3turn4search7  
- Метрики не имеют высокой кардинальности; naming/labels соответствуют практикам Prometheus. citeturn1search3turn1search11  
- Трейсы/пропагация соответствуют OpenTelemetry и W3C Trace Context (если включено). citeturn8search17turn8search0turn0search10  

**API security**
- Ошибки наружу не раскрывают внутренние детали; статус‑коды и семантика ошибок соответствуют OWASP REST Security guidance. citeturn16search5turn0search3  
- Присутствуют меры против DoS/abuse (rate limiting, лимиты размеров, аккуратные retries). citeturn16search1turn16search9turn17view0  

**Supply chain**
- Dockerfile использует multi‑stage builds; финальный образ минимален. citeturn7search2turn7search6  
- Зафиксированы версии и политика поддержки Go. citeturn13search0turn13search4  
- Включены базовые supply chain практики и автоматические проверки (SLSA/Scorecard — как минимум на уровне репозитория). citeturn4search1turn4search14  

### Что оформить отдельными файлами в template repo

Минимальный пакет документов/конвенций, который лучше сделать отдельными файлами, чтобы LLM могла ссылаться на конкретные тексты:

- `docs/engineering-standard.md`  
  Норматив: версии Go, layout, принципы API, error handling, контракты observability, правила по зависимостям, ожидания по perf/allocations (в разумной мере), правила shutdown, подход к k8s Probes. citeturn1search4turn1search1turn20search2turn6view2turn5view2  

- `docs/llm-instructions.md`  
  MUST/SHOULD/NEVER, список разрешенных библиотек, запрет на “угадывание”, требование компилировать/тестировать, anti‑patterns, и формат запросов к LLM (“если не хватает контекста — запроси конкретные файлы/типы/интерфейсы”). Основание: уменьшение риска галлюцинаций и следование idiomatic Go. citeturn1search4turn3search8turn1search1  

- `docs/observability.md`  
  Логи на `log/slog`, метрики по Prometheus practices, трейсинг по OpenTelemetry (с pinned semconv версиями и оговоркой статуса Mixed/Development), W3C trace context propagation, правила кардинальности, обязательные RED/USE‑метрики. citeturn2search6turn1search3turn8search0turn0search10turn10search7  

- `docs/security.md`  
  OWASP API Top 10 как чек‑лист рисков, OWASP cheat sheets (logging, secrets, DoS), политика ошибок наружу, `govulncheck` как gate, подход к секретам. citeturn0search3turn4search3turn4search7turn16search1turn15search0  

- `docs/testing.md`  
  Структура тестов на `testing`, race detector, fuzzing, принципы интеграционных тестов, тестирование concurrency (в т.ч. при необходимости синхронизация). citeturn2search0turn14search0turn14search2turn2search8  

- `docs/caching.md`  
  Политики cache‑aside/write‑through, TTL и jitter, thundering herd, наблюдаемость кэша (hit ratio, miss reasons), поведение при отказах, правильные fallback/degraded режимы, и обязательные тесты/метрики (см. следующий раздел). citeturn19view0turn17view0turn17view1turn9search4turn9search0  

- `docs/runbook.md` (или `docs/operations.md`)  
  Как диагностировать деградацию (SLO, latency, saturation), что смотреть при инциденте (метрики, логи, трейсы), как понимать симптомы проблем с кэшем/DB. Основание: SRE‑практики о перегрузке, каскадных сбоях и деградации. citeturn17view0turn17view1turn5view2  

Плюс репо‑конфиги (как часть “template conventions”):  
- CI workflow (gofmt/go test/race/vet/govulncheck),  
- Dockerfile (multi‑stage),  
- PR template с checklist,  
- CODEOWNERS/ownership,  
- `Makefile`/task runner,  
- `.editorconfig`. citeturn7search6turn15search6turn14search0turn15search0  

## Исследование подтемы: observability, testing и отказоустойчивость кеша

Дальше — итог в формате production guide: метрики и тесты, обязательные для caching layer, и поведение сервиса при деградациях/отказах кэша.

### Базовый контракт кэша

Для шаблона лучше явно описать, что кэш — **ускоритель**, а не источник истины (если явно не выбран иной дизайн). Отсюда стандартный default‑паттерн — **cache‑aside / lazy caching**: при запросе сначала читаем кэш, при miss идем в primary store, затем пишем в кэш. Это прям “foundation” паттерн в AWS caching best practices. citeturn19view0  

Обязательный baseline для ключей:
- На все ключи ставить TTL (кроме строго write‑through случаев), чтобы ошибки инвалидации не превращались в вечную неконсистентность. AWS отдельно рекомендует TTL почти везде как “страховку от багов”. citeturn19view0  
- При использовании TTL учитывать, что TTL может усугублять thundering herd, поэтому нужно проектировать защиту. citeturn19view0  

### Метрики кэша: что обязательно в сервисе

Ниже — метрики, которые **обязательны на уровне приложения**, потому что сами Redis/Memcached не знают “смысл” вашего кэша (почему вы bypass, что такое “stale”, что такое “fallback”).

#### Метрики “cache request outcomes”
Обязательная базовая метрика:
- `cache_requests_total{cache="<name>", op="get|set|delete", outcome="hit|miss|error|bypass|stale_hit|negative_hit"}`

Принципиально важно: outcome‑лейбл должен быть **ограниченным** множеством значений. Prometheus практики прямо предупреждают, что высокая кардинальность лейблов резко увеличивает стоимость хранения и нагрузку. citeturn1search3turn1search11  

Derived KPI:
- **hit ratio** = hits / (hits + misses) по кешу и по ключевым `cache`/`op`. На уровне Redis есть агрегированные `keyspace_hits`/`keyspace_misses` как базовые счетчики, но они не дают бизнес‑семантики и могут быть “слишком общими” (несколько logical caches в одной БД). citeturn9search4turn9search7  

#### Miss reason taxonomy (обязательная таксономия)
Чтобы понимать, *почему* miss происходит, нужен отдельный лейбл с ограниченным словарем:

- `cache_misses_total{cache="<name>", reason="cold|expired|evicted|invalidated|not_found|negative_cached_absent|bypass_disabled|bypass_too_large|dependency_error|serialization_error"}`

Смысл причин:
- `cold` — ключ никогда не был прогрет или новый узел после масштабирования/фейловера. Факт “пустого узла” и необходимость prewarm — отдельная тема в AWS best practices. citeturn19view0  
- `expired` — TTL истек. TTL‑стратегии и побочные эффекты TTL описаны у AWS. citeturn19view0  
- `evicted` — вытеснен по памяти/eviction policy (прямой маркер давления памяти и неправильного sizing). Redis имеет eviction политики и метрики по вытеснениям; AWS отдельно говорит “evictions обычно означают, что надо scale up/out”, если это не сознательный LRU‑кейс. citeturn9search0turn19view0  
- `invalidated` — ключ явно удален после записи в primary store (ваша логика инвалидации).  
- `not_found` / `negative_cached_absent` — полезно различать “в кэше нет, потому что отсутствует в мире” (negative caching) и “нет по случайности”; особенно важно для защиты от stampede по отсутствующим объектам.  
- `bypass_*` — приложение сознательно решило не использовать cache (например, “payload слишком большой” или “кэш отключен в деградированном режиме”). Это критично для интерпретации hit ratio.  
- `dependency_error` — сам кэш недоступен/timeout/ошибка сети; такие misses нельзя смешивать с “нормальными”.  
- `serialization_error` — ошибка (де)сериализации, схемы или формата.

Эта таксономия должна быть договором команды: фиксировать словарь и запрещать ad‑hoc значения, иначе вы нарушите правила кардинальности. citeturn1search3turn1search11  

#### Latency и ошибки cache operations
- `cache_op_duration_seconds` как histogram по операциям (`get/set/delete`) — с bucket’ами, согласованными с SLO (Prometheus отдельно объясняет, как под SLO выбирать buckets для histogram). citeturn10search3turn10search3  
- `cache_errors_total{cache="<name>", class="timeout|conn_refused|protocol|auth|other"}` — bounded классы ошибок.

#### “Stale reads” и fallback visibility
Если вы используете stale‑while‑revalidate / serve stale on error (частая практичная стратегия под перегрузкой), то нужно отдельное наблюдение:
- `cache_stale_served_total{cache="<name>", reason="revalidate_inflight|backend_overload|cache_error"}`
- `cache_refresh_total{cache="<name>", outcome="success|error"}` и latency refresh’а
- `cache_fallback_total{cache="<name>", to="primary|degraded_response", reason="cache_down|timeout|circuit_open|overload"}`

“Serve degraded responses” — базовая SRE‑стратегия при перегрузке: лучше отдать менее точный/устаревший ответ, чем положить систему. Это напрямую применимо к кэш‑слою (serve stale вместо “ударить в primary store в шторм”). citeturn17view1turn17view0  

### Метрики кэша: что обязательно собирать с backend (Redis/Memcached)

Это метрики “снаружи” сервиса, но гайд должен требовать их наличия в мониторинге, иначе вы не увидите, что кэш стал источником проблем.

#### Redis
Минимум:
- `keyspace_hits`, `keyspace_misses` — основа hit ratio на уровне Redis. citeturn9search4  
- `expired_keys`, `evicted_keys` — маркеры TTL‑истечений и вытеснений; Redis документация и материалы по observability выделяют их как “metrics of note”. citeturn9search7turn9search4  
- мониторинг eviction политики и факта превышения maxmemory (Redis INFO включает eviction exceeded time; плюс есть документ по eviction policy). citeturn9search0turn9search4  

#### Memcached
Минимум:
- `get_hits`, `get_misses`, `evictions` (и различение evictions vs reclaimed важно для понимания pressure). Для AWS ElastiCache (Memcached) эти метрики описаны и формализованы; также есть определения evictions в memcached docs. citeturn9search5turn9search2  

Если вы используете managed cache (например, ElastiCache), гайд должен требовать также cloud‑метрики доступности/фейловеров и “read availability temporarily if node fails” (AWS explicitly предупреждает про временную недоступность чтений при отказе ноды и про распределение чтений). citeturn9search3  

### Как сервис должен вести себя при отказе кэша

Нужно заранее выбрать и закрепить policy, потому что “по умолчанию” LLM может сделать ретраи/блокировки, которые приведут к каскадным сбоям.

#### Default policy: fail‑open + bounded timeouts
Для большинства кэш‑ускорителей default — **fail open**: ошибка кэша должна превращаться в miss и вести к чтению из primary store *при строгом ограничении времени и параллелизма*. Иначе при падении кэша вы мгновенно перегрузите primary store и получите cascading failure (SRE подробно описывает каскадные сбои и важность load shedding/degraded responses). citeturn17view0turn17view1  

Практически это означает:
- у кэш‑клиента жесткий таймаут (меньше, чем у primary store),  
- при серии ошибок включается **circuit breaker / cache disable window**: сервис временно перестает трогать cache и сразу идет в primary store (или отдает degraded), чтобы не тратить ресурсы на повторяющиеся timeouts,  
- ретраи к кэшу крайне ограничены, с backoff+jitter, чтобы не усилить перегрузку (в SRE есть прямые указания про экспоненциальный backoff+jitter; и предупреждения о retry‑паттернах как триггере каскадных сбоев). citeturn17view0turn16search1  

#### Thundering herd/stampede protection как часть “поведения при отказе”
AWS отдельно выделяет thundering herd (dog piling) как типовую проблему при TTL и при добавлении нового пустого узла, и перечисляет практики: prewarm и TTL jitter (случайная добавка к TTL, чтобы ключи не истекали синхронно). citeturn19view0  

Шаблон должен закрепить один из “boring способов” защиты:
- request coalescing (на уровне процесса) для горячих ключей: один запрос делает refresh, остальные ждут/получают stale;  
- randomized TTL jitter для распределения истечений; citeturn19view0  
- prewarm при масштабировании/замене узлов; citeturn19view0  
- при сильной перегрузке — “serve stale” как degraded ответ, если это допустимо контрактом (см. SRE). citeturn17view1  

### Cache correctness: обязательные тесты

Цель: не просто “покрыть код”, а проверять **корректность поведения** в условиях деградаций.

#### Unit‑тесты: детерминированный контракт кэш‑обертки
Обязательные тест‑кейсы (как спецификация поведения):
- hit → не вызывает primary store, корректно десериализует, записывает метрики outcome=hit;  
- miss(cold/not_found) → вызывает primary store, записывает в cache, outcome=miss, reason=cold/not_found;  
- expired → ведет себя как miss с reason=expired;  
- cache error/timeout → ведет себя как miss с reason=dependency_error, outcome=error + fallback_total увеличивается;  
- singleflight/coalescing: при конкурентных запросах к одному ключу refresh выполняется один раз и не вызывает шторм в primary;  
- stale‑serve (если включено): при cache_error или backend_overload отдает stale и маркирует `cache_stale_served_total`.

Структурно эти тесты пишутся через стандартный `testing`. citeturn2search0  

Если кэш‑обертка конкурентная, то обязателен прогон с race detector. citeturn14search0  

#### Integration‑тесты: совместимость с реальным backend
Минимум два режима:
- “с кэшем” (реальный Redis/Memcached поднимается локально/в CI)  
- “без кэша/кэш недоступен” (симулировать отказ: timeout/connection refused)  

Цель — проверить, что fail‑open/circuit‑breaker действительно защищают primary store. При постановке таких тестов нужно помнить, что Kubernetes и реальные системы могут давать SIGTERM и ограниченное время на shutdown; интеграционные тесты shutdown‑поведения — отдельный плюс. citeturn6view2turn0search17  

#### Consistency verification: выборочная проверка “кэш портит данные”
Поскольку кэш по определению может отдавать устаревшее, важно измерять **насколько** и **почему**:
- выборочно (например, 0.1% запросов на чтение) после получения значения из cache делать read‑after‑read из primary store и сравнивать (если это допустимо по нагрузке);  
- метрика `cache_value_mismatch_total{cache="<name>", class="stale|corrupt|schema_mismatch"}`;  
- отдельный счетчик `cache_bypassed_consistency_check_total` если проверку пришлось отключить.

Эта “проверка консистентности” должна быть feature‑flagged и с жестким budget’ом, иначе она может сама стать нагрузкой. Логика соответствует SRE‑идее: тестировать поведение системы в неблагоприятных режимах и знать свои breaking points. citeturn17view0  

### Load testing с кэшем: что измерять и почему

SRE прямо рекомендует: **load test до отказа и beyond**, и тестировать failure mode при overload — это ключ к предотвращению каскадных отказов. citeturn17view0  

Для кэша это означает три обязательных профиля нагрузочного теста:
- **Warm cache**: предварительно прогретые горячие ключи → измерить p50/p95/p99 latency и нагрузку на primary store.  
- **Cold cache / new node**: пустой кэш → измерить stampede‑защиту, как быстро система стабилизируется, какие miss reasons доминируют (должен быть `cold`). citeturn19view0  
- **Cache outage / high error rate**: симулировать недоступность кэша → убедиться, что система не “самоубивается” ретраями/таймаутами и корректно переходит в degraded mode / circuit open. Это прямо связано с SRE‑предупреждениями о cascading failures из‑за overload и неверных retry‑паттернов. citeturn17view0turn17view1  

Что обязательно измерять:
- hit ratio + распределение miss reasons,  
- tail latency (p95/p99) на endpoint’ах,  
- рост QPS/latency на primary store во всех трех режимах,  
- rate evictions/expired keys (backend‑метрики) как признак, что TTL/размер/политика не подходят. citeturn9search4turn9search7turn9search0turn19view0  

### Признаки, что кэш ухудшает систему, а не помогает

Эту секцию стоит вынести в `docs/caching.md` и `docs/runbook.md` как “when to turn it off”.

Кэш часто “вредит”, если наблюдаются такие симптомы:

- **Hit ratio низкий, но стоимость высокая**: много cache calls, но мало hits; при этом растет tail latency из‑за сетевых round‑trip’ов к кэшу. Это видно по `cache_requests_total` (много outcome=miss/bypass/error) и `cache_op_duration_seconds`. citeturn1search11turn10search3  
- **Memory pressure и evictions растут**: рост `evicted_keys`/evictions → вы фактически постоянно вытесняете полезные ключи, и кэш превращается в “мельницу”, создавая лишнюю нагрузку на primary store. Redis eviction и метрики истечений/вытеснений документированы. citeturn9search0turn9search4turn9search7  
- **Thundering herd/stampede**: всплески нагрузки на БД в моменты TTL‑истечения или при добавлении нового узла. AWS описывает проблему и рекомендует prewarm и TTL jitter. citeturn19view0  
- **Неверная деградация в Kubernetes**: если readiness/liveness завязаны на кэш, поды начинают перезапускаться “в унисон” и усугубляют инцидент (Kubernetes прямо предупреждает о cascade при неверном liveness). citeturn5view2  
- **Кэш нарушает ожидания консистентности**: рост `cache_value_mismatch_total` (или ручные инциденты “читаем старое/битое”) — сигнал пересмотреть TTL, write‑path инвалидацию или отказаться от кэша для этого типа данных.  
- **Retry storm**: при деградации кэша сервис начинает агрессивно ретраить и тратить ресурсы, что соответствует SRE‑паттернам каскадных отказов. Лечение: лимиты, backoff+jitter, circuit‑breaker, ранний отказ/деградация. citeturn17view0turn17view1  

Default remediation steps (как “runbook actions”):
- включить “cache disable window” (быстро перевести сервис в режим bypass кэша),  
- уменьшить load на primary store (load shedding / degraded responses),  
- включить/усилить stampede protection и TTL jitter,  
- пересчитать sizing/eviction policy, если evictions объективно высокие. citeturn17view1turn19view0turn9search0