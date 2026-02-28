# Engineering standard и LLM-инструкции для production-ready Go микросервиса

## Область применения

Этот стандарт предназначен для **greenfield**-шаблона микросервиса, который:

- оказывает **сетевой API** (обычно HTTP/JSON; опционально gRPC), живёт в контейнере и должен корректно работать в оркестраторе (типично — Kubernetes: readiness/liveness, graceful shutdown); при этом сервис должен быть **наблюдаемым** (logs/metrics/traces), безопасным по умолчанию и удобным для сопровождения. citeturn2search2turn2search9turn7view0turn1search2turn10search9  
- активно развивается с помощью LLM-инструментов (IDE-агенты/чат-ассистент), поэтому репозиторий обязан давать **максимально явный контекст**: архитектурные границы, соглашения, команды, примеры правильного кода, и запреты на «догадки». citeturn19search2turn19search0turn19search17turn19search3  
- целится в “boring, battle-tested defaults”: минимум экзотики, максимум стандартной библиотеки и зрелых практик. (Это особенно хорошо сочетается с современными улучшениями стандартной библиотеки маршрутизации в net/http ServeMux, доступными начиная с Go 1.22.) citeturn15search2turn15search0turn8view0  

Не применять (или применять с оговорками), если:

- это **data/CPU-heavy** сервис (стриминг, high-throughput binary протоколы, низкие задержки), где выбор транспорта, сериализации, аллокаций, профилирования и т. п. требует отдельного дизайн-цикла; стандарт всё ещё полезен, но “defaults” могут быть неадекватны. citeturn8view0turn22search1  
- это **плагин/библиотека**, а не сервис: часть правил (health endpoints, k8s манифесты, HTTP server timeouts) не применима. citeturn4search1turn18search9  
- монолит/модульный монолит, где “микросервисная” инфраструктурная обвязка создаст лишнюю сложность (но при желании шаблон можно использовать как “service slice” внутри монорепы). citeturn4search1turn5search3  

## Рекомендуемые defaults для greenfield template

Ниже — набор “по умолчанию”, который **можно прямо перенести в docs/** и репозиторные соглашения. Он опирается на первичные источники (go.dev, стандартная библиотека, спецификации, CNCF/OWASP/Prometheus/Kubernetes и т. п.). citeturn0search1turn0search0turn3search5turn2search2turn11search2turn10search0  

### Версии, модульность и воспроизводимость

- Целевая версия языка/тулчейна: **Go 1.26.0** (релиз от 2026‑02‑10). citeturn2search4turn2search1  
- В `go.mod` фиксировать:
  - `go 1.26.0` (минимально требуемая версия) — начиная с Go 1.21 это уже *строгое требование*; тулчейн откажется работать с модулем, если версия выше установленной. citeturn3search13turn3search1turn3search17  
  - `toolchain go1.26.0` (рекомендованный тулчейн) для консистентности локальной разработки и CI. citeturn3search5turn3search17turn3search1  
- Держать `go.mod` и `go.sum` в репозитории: `go.sum` используется для проверки целостности скачанных модулей. citeturn6search19turn6search11turn6search15  
- Минимизировать зависимости и документировать их назначение: шаблон должен быть понятен без знания конкретных фреймворков. (В Go это естественно: форматирование и большая часть tooling — стандартные.) citeturn3search6turn3search2turn0search1  

### Структура репозитория

Опирайтесь на официальное руководство по layout модулей: оно прямо говорит про несколько программ и top-level `internal/` для разделяемых внутренних пакетов. citeturn4search1turn18search2  

Рекомендуемый скелет:

- `cmd/<service>/main.go` — точка входа, wiring (конфиг, зависимости, запуск/останов).  
- `internal/` — приватный код сервиса (не для импортов извне), включая HTTP handlers, use-cases, repo-адаптеры. Механизм `internal` обеспечивается самим tooling Go. citeturn18search1turn18search2  
- `api/` — контракт: OpenAPI (по умолчанию) и/или protobuf. OpenAPI даёт язык-агностичное описание HTTP API и поддерживает генерацию клиентов/тестов. citeturn9search4turn9search8turn9search12  
- `docs/` — стандарты и “LLM context” (см. ниже).  
- `deploy/` (или `k8s/`) — минимальные манифесты/Helm/Kustomize (если целитесь в Kubernetes). Пробы и жизненный цикл подов — из официальной доки Kubernetes (а не “как у кого-то в блог-посте”). citeturn2search2turn11search0turn12search0  
- `.github/workflows/` — CI: gofmt/go vet/tests/govulncheck. citeturn3search4turn3search8turn0search4  

**Важно про project-layout**: популярный репозиторий “Standard Go Project Layout” полезен как ориентир, но *не является официальным стандартом*; официальная позиция Go — “layout зависит от типа проекта”, и есть официальный гайд по структуре модуля. В шаблоне это стоит явно указать, чтобы LLM не “догадалась”, что `pkg/` обязателен. citeturn4search1turn4search5  

### Транспорт и маршрутизация

- По умолчанию: **net/http + ServeMux** с паттернами Go 1.22+ (методы и wildcards). Это снижает зависимость от роутеров и делает шаблон “boring” и долгоживущим. citeturn15search2turn15search0turn15search6turn15search1  
- Паттерны ServeMux **существенно изменились** в Go 1.22; существует режим совместимости через `GODEBUG=httpmuxgo121=1`, но для greenfield на Go 1.26 разумнее принять новое поведение и зафиксировать версию в go.mod. citeturn15search1turn15search2turn3search13  

### HTTP server: timeouts, лимиты, shutdown

Для production сервис **обязан** задавать явные лимиты на входящие запросы и корректно завершаться:

- В `http.Server` использовать по меньшей мере:
  - `ReadHeaderTimeout` (часто предпочтительнее, чем `ReadTimeout`, потому что позволяет handler’ам решать дедлайны для body). citeturn8view0  
  - `WriteTimeout` и `IdleTimeout` (с пониманием trade-off: стриминг/long-poll может требовать отдельных настроек). citeturn8view0turn0search2  
  - `MaxHeaderBytes` как базовую защиту от чрезмерных заголовков. citeturn8view0  
- Graceful shutdown делать через `Server.Shutdown(ctx)`: метод закрывает listeners, закрывает idle connections и ждёт завершения активных соединений до таймаута контекста. citeturn7view0  
- В Kubernetes помнить, что `preStop` hook **входит** в `terminationGracePeriodSeconds` и выполняется **не асинхронно** относительно SIGTERM (hook должен завершиться до отправки TERM). Это критично для правильной настройки “drain” и shutdown логики. citeturn11search0turn11search3turn2search6  

### HTTP client: таймауты и переиспользование

- `http.Client` и `Transport` нужно **переиспользовать**, потому что transport хранит состояние (кэш TCP соединений), и клиент безопасен для конкурентного использования. citeturn17view0  
- У `Client.Timeout` значение 0 означает **отсутствие таймаута**, что в проде часто приводит к зависшим запросам и исчерпанию ресурсов; таймаут должен быть явным “по умолчанию”. citeturn17view0  
- По умолчанию Transport уже имеет ряд настроек (MaxIdleConns, IdleConnTimeout, TLSHandshakeTimeout и т. п.), но **таймаут уровня запроса** всё равно должен быть задан (через `Client.Timeout` и/или контекст). citeturn16search0turn17view0  
- Всегда закрывать `Response.Body` и (для keep-alive) корректно дочитывать до EOF, иначе соединение может не переиспользоваться. citeturn17view0  

### Логи, метрики, трейсинг

- Логи: структурированные (ключ‑значение) через `log/slog` (Go 1.21+), логгер — зависимость, а не глобальная магия; output ориентировать на stdout/stderr. citeturn1search2turn2search3  
- Следовать принципу 12‑Factor “treat logs as event streams”: приложение не управляет файлами логов, пишет поток в stdout, а окружение маршрутизирует/хранит. citeturn2search3  
- Security logging: логирование должно учитывать риск утечек (PII/секреты), и быть устойчивым к инъекциям в лог-канал. citeturn16search2turn16search13  

- Метрики: если используете Prometheus-совместимую модель, ключевые правила:
  - не злоупотреблять labels; каждая комбинация label set — отдельный time series с затратами RAM/CPU/disk/network; guideline — кардинальность метрик держать низкой, большинство метрик — без labels. citeturn10search0  
  - не использовать высококардинальные значения (user_id, email, request_id) в label’ах. citeturn10search1  

- Tracing: ориентироваться на vendor-neutral подход (OpenTelemetry) и стандарт пропагации контекста W3C Trace Context (`traceparent`, `tracestate`). citeturn10search3turn10search9turn9search15  
  - Важно: в OTel документации отмечено, что signal logs всё ещё **experimental** и может меняться; это влияет на “defaults” (для логов можно использовать `slog` как базу, а OTel logs подключать опционально/позже). citeturn9search15turn10search9  

### API errors: формат и безопасность

- Для HTTP API ошибок использовать стандарт **Problem Details** (RFC 9457), который предназначен для машинно-читаемого формата ошибок и **обsoletes RFC 7807**. citeturn13search0turn13search7  
- Статус-коды и их значения опираются на HTTP Semantics (RFC 9110) и реестр IANA; это снижает “свободное творчество” и помогает договорам между сервисами. citeturn13search2turn13search6  
- Ошибки не должны раскрывать чувствительные детали (SQL, внутренние ID, stack traces), особенно наружу. Это согласуется и с практиками безопасного API дизайна/обработки ошибок. citeturn13search17turn16search13  

### Security-by-default

- Для REST/HTTP сервисов: “Secure REST services must only provide HTTPS endpoints” — базовое требование. entity["organization","OWASP","web app security nonprofit"] формулирует это прямо в REST Security Cheat Sheet. citeturn11search2  
- В качестве практического “чек-листа требований” для web/app security удобно использовать entity["organization","OWASP ASVS","application security standard"]: он задаёт структуру требований и уровни. citeturn3search7turn3search3turn3search19  
- Для API-специфических рисков ориентироваться на OWASP API Security Top 10 2023 (внутренний стандарт должен отражать хотя бы ключевые классы: авторизация по объектам, misconfiguration, и т. п.). citeturn0search3  

### Container defaults

- Dockerfile: multi-stage build — официальная рекомендация Docker для поддерживаемых и компактных образов. entity["company","Docker","container tooling company"] описывает multi-stage как базовый механизм. citeturn12search3  
- Base image: distroless уменьшает attack surface (нет shell/package manager). Это зрелый подход, но он ухудшает “отладку в контейнере”, поэтому в шаблоне нужно прописать trade-off и практику (например, отладка через отдельно запускаемый debug-образ, ephemeral containers, или через окружение). citeturn12search2  

## Матрица решений и trade-offs

Этот раздел — то, что LLM должна “видеть” как **разрешённые развилки**. Он критичен, чтобы модель не изобретала стек и не добавляла случайные зависимости.

### HTTP/JSON vs gRPC

- HTTP/JSON + OpenAPI:
  - плюсы: проще вход, легко дебажить, “язык-агностичный” контракт, большие экосистемы клиентов/инструментов. citeturn9search4turn9search12  
  - минусы: типизация слабее, сложнее эволюционировать сложные схемы, performance overhead сериализации (обычно не критично на старте). citeturn9search4  

- gRPC + protobuf:
  - плюсы: строгая типизация и контракт, эффективная бинарная сериализация, стандартные статусы/интерфейсы, хорош для internal RPC.  
  - минусы: требует инфраструктуры/инструментов, сложнее для внешних клиентов без gRPC, иной подход к ошибкам и middleware (interceptors). citeturn9search1turn9search21turn13search11  

**Default**: начинать с HTTP/JSON, а gRPC держать как “заранее предусмотренный” второй транспорт (не обязательно включать в MVP-шаблон, но архитектура должна позволять добавить transport без переписывания бизнес-логики). Это напрямую соответствует идее separation transport vs service layer, которую используют зрелые toolkits (например, Go kit). citeturn5search14turn15search2turn9search4  

### Router/library vs stdlib ServeMux

- stdlib ServeMux (Go 1.22+):
  - плюсы: минимум зависимостей, долгосрочная поддержка, достаточно выразительные паттерны (методы, wildcards). citeturn15search2turn15search0  
  - минусы: меньше “готовых батареек” для групп роутов/мидлварей (хотя в Go middleware легко реализуется поверх `http.Handler`). citeturn7view0  

- сторонние роутеры:
  - плюсы: часто удобнее composition, группы, параметры, middleware.  
  - минусы: зависимость, разные “школы” и несовпадение паттернов, больше места для LLM-галлюцинаций (“я видел так в другом проекте”).  

**Default**: stdlib ServeMux (Go 1.26) + явная middleware chain, фиксированная в шаблоне. citeturn15search2turn7view0  

### Observability: Prometheus vs OpenTelemetry

- Prometheus-first:
  - плюсы: простая модель, зрелая документация best practices по labels и naming. citeturn10search0turn10search1  
  - минусы: трейсинг и логи — отдельные решения/интеграции; vendor-neutral “end-to-end” сложнее.  

- OpenTelemetry:
  - плюсы: единая модель signals, семантические соглашения, стандартный путь через Collector (receivers/processors/exporters). citeturn10search2turn10search9turn1search11  
  - минусы: сложнее на старте; logs signal может быть experimental. citeturn9search15  

**Default**: traces через OpenTelemetry (вход/выход HTTP instrumentation), метрики — Prometheus endpoint (или OTel metrics → Prometheus exporter/bridge через Collector, если в компании принят OTel pipeline). Внутренний стандарт должен фиксировать один “primary path”, иначе LLM будет смешивать подходы. citeturn9search3turn10search9turn10search0  

### Errors: «свои JSON ошибки» vs RFC 9457 Problem Details

- Свой формат:
  - плюсы: можно сделать минимальным и “внутренним”.  
  - минусы: каждый сервис делает по-своему; сложно стандартизировать клиентов.

- RFC 9457:
  - плюсы: стандарт, хорошая совместимость, понятная структура для машин, меньше “велосипедов”. citeturn13search0turn13search7  
  - минусы: нужно дисциплинированно определить `type`-URI/каталог problem types и правила маппинга ошибок.  

**Default**: RFC 9457. citeturn13search0turn13search7  

## Архитектура приложения внутри Go-сервиса

Ниже — конкретные, практические правила про handlers/controllers, transport, service/use-case, repository, DTO vs domain model, middleware, транзакции, validation, mapping и cross-cutting concerns — с учётом Go-идиоматики и типичных перегибов “чистой архитектуры”.

image_group{"layout":"carousel","aspect_ratio":"16:9","query":["hexagonal architecture ports and adapters diagram","clean architecture diagram dependency rule","golang microservice layered architecture diagram","middleware chain net/http diagram"],"num_per_query":1}

### Сравнение подходов: layered vs clean vs hexagonal

- **Hexagonal / Ports-and-Adapters** (А. Кокберн): идея — отделить application core от внешних “устройств” (UI, DB, транспорт), чтобы приложение одинаково драйвилось людьми/программами/тестами и тестировалось в изоляции. citeturn5search0  
- **Clean Architecture** (Р. Мартин): акцент на независимость от фреймворков/БД/UI, тестируемость и правило зависимостей (внутрь, к бизнес-правилам). citeturn5search1  
- **Pragmatic layered architecture**: обычно транспорт → сервис → репозитории, минимум формализма. В Go это часто наиболее устойчивый вариант, потому что язык поощряет простоту интерфейсов и композицию пакетов; а чрезмерная “архитектурность” быстро превращается в boilerplate. Это согласуется с тем, что Go style-guides стремятся уменьшать “guesswork” и поощряют читаемость/простоту. citeturn4search2turn0search1turn0search0  

Отдельно полезно упомянуть Go kit как пример “прагматичной слоистости” (transport/endpoint/service): даже если вы не используете библиотеку, сама декомпозиция хорошо объясняет границы. citeturn5search14  

### Рекомендуемый default-pattern для template

**Default**: “упрощённая hexagonal/clean” в форме **трёх слоёв** (transport → application/usecase → adapters/repository), но без “религии”:

- **Transport layer (HTTP/gRPC)**:  
  отвечает за:
  - маршрутизацию, authn/authz middleware, rate limiting (если есть), correlation ID, request logging, recovery;  
  - decoding/encoding (JSON/Proto), нормализацию входа, базовую валидацию формы (типы, обязательные поля, формат), установку ограничений на body;  
  - маппинг application errors → HTTP/gRPC status + Problem Details. citeturn7view0turn11search2turn13search0turn9search1  

- **Application layer (service/use-case)**:  
  отвечает за:
  - бизнес-правила и оркестрацию;  
  - транзакционные границы;  
  - использование портов (интерфейсов) для внешних зависимостей;  
  - idempotency/reties (если релевантно), но *без* транспортно-специфичных деталей. citeturn5search0turn5search1turn6search2  

- **Adapters layer (repositories, внешние клиенты)**:  
  отвечает за:
  - конкретную реализацию портов: SQL/HTTP clients/message brokers;  
  - маппинг storage model ↔ domain/application model;  
  - observability instrumentation на границе IO. citeturn6search2turn9search3turn17view0  

### DTO vs domain model: границы и маппинг

**Default правило**: DTO (transport) ≠ domain/application model.

- DTO существуют, чтобы быть стабильными для внешнего контракта (OpenAPI/proto) и управляться правилами сериализации. citeturn9search4turn15search2  
- Domain/application types выражают бизнес-правила и инварианты и не должны “знать” о JSON тэге, HTTP статусе или SQL-колонке. citeturn5search1turn5search0  

Практический компромисс для Go: не вводить “богатый домен” через OOP-методы и сложные иерархии; вместо этого — **простые структуры + функции**, с явными проверками инвариантов в application layer. Это снижает риск, что LLM создаст лишнюю сложность. citeturn0search1turn4search2  

### Validation placement

- В transport: “форма и безопасность” (валидность JSON, обязательные поля, размеры строк/массивов, ограничения на body). net/http даёт инструменты для ограничения (например, через MaxBytesReader/handlers) и server-level limits. citeturn8view0turn7view0  
- В application/use-case: “смысл и бизнес-инварианты” (например, переходы статусов, права на объект). Это напрямую связано с типовыми API security проблемами: object-level authorization должна проверяться на каждом доступе к данным по ID из запроса. citeturn0search3turn11search9  

### Transaction boundaries

**Default**: транзакция — ответственность application layer, а не репозитория “в вакууме”.

- В `database/sql` контекст, переданный в `BeginTx`, действует до commit/rollback; при отмене контекста пакет `sql` откатывает транзакцию, а `Commit` вернёт ошибку. Это важно учитывать, чтобы корректно связать lifecycle транзакции с request context. citeturn6search2turn14view0  
- Документация Go по транзакциям явно описывает работу через `sql.Tx` и операции commit/rollback. citeturn6search5  
- Для PostgreSQL помнить дефолтный isolation (Read Committed) и то, что разные уровни изоляции имеют реальные semantic trade-offs; стандарт должен фиксировать default и критерии повышения уровня (например, для денежных операций). citeturn6search3turn6search10  

Практичный шаблонный паттерн: `type Querier interface{ ExecContext(...); QueryContext(...); QueryRowContext(...) }` который реализуют и `*sql.DB`, и `*sql.Tx`; use-case решает “в транзакции или нет”, а репозиторий принимает `Querier`. Это минимизирует duplication и не заставляет LLM “городить UnitOfWork фреймворк”.

### Middleware / interceptors и cross-cutting concerns

- В HTTP: опираться на `http.Handler` composition (цепочки middleware). Это естественная модель net/http. citeturn7view0  
- В gRPC: использовать interceptors (client/server) — официально описанный механизм. citeturn9search1turn9search21  
- Observability: instrumentation делать на границе транспорта и IO (wrap handler, wrap RoundTripper, DB instrumentation) при использовании семантических соглашений. citeturn9search3turn1search11turn17view0  

## Набор правил MUST / SHOULD / NEVER для LLM

Эти правила задуманы как **основа для файлов инструкций** (Copilot/Claude/Codex) и как “контракт” между репозиторием и моделью: что можно делать, а что запрещено.

### MUST

1) **Следовать Go-идиомам и официальным гайдам**: форматирование gofmt; стиль и читаемость — по Effective Go и CodeReviewComments; избегать “универсальных паттернов из других языков”. citeturn0search1turn0search0turn3search2turn0search4  

2) **Не делать скрытых предположений**: если требуется решение (errno mapping, таймауты, транзакция/не транзакция, формат ошибки), а стандарт/код уже содержит правило — следовать ему; если правило отсутствует — добавлять `TODO(standard)` + короткий список допущений и выбрать наиболее boring вариант, не вводя новых зависимостей.

3) **Пропагировать `context.Context` сверху вниз**:
- context — первый параметр; не хранить context в struct; не передавать nil context; `WithCancel/WithTimeout` всегда закрывать cancel на всех путях, чтобы не утекали таймеры/дети. citeturn14view0  

4) **Все сетевые операции должны иметь таймауты**:
- сервер: `ReadHeaderTimeout`/`WriteTimeout`/`IdleTimeout`/`MaxHeaderBytes` (или эквивалент, если транспорт иной). citeturn8view0turn7view0  
- клиент: `http.Client.Timeout` ≠ 0; client переиспользуется; body закрывается. citeturn17view0  

5) **Ошибки оборачивать корректно и проверяемо**:
- использовать wrapping из Go 1.13 (`%w`, `errors.Is/As/Unwrap`) для причинно-следственной цепочки. citeturn1search1  
- маппить application errors → RFC 9457 Problem Details и корректные HTTP status codes. citeturn13search0turn13search2turn13search6  

6) **Security-by-default**:
- не добавлять HTTP endpoints без TLS-требования в документации/деплое (если сервис предполагается наружу); не логировать секреты/PII; авторизация на уровне объектов должна быть явной. citeturn11search2turn16search2turn0search3  

7) **Изменения должны быть проверяемыми**:
- добавлять/обновлять тесты; запускать go test в CI (минимум `./...`), включать проверку data races там, где это применимо (race detector — официальный инструмент Go). citeturn21search12turn23search1turn23search4  
- проверять уязвимости зависимостей через govulncheck (официальный инструмент Go команды безопасности). citeturn3search4turn3search8turn3search12  

### SHOULD

1) **Держать зависимости минимальными** (stdlib-first), а новые зависимости добавлять только при явной необходимости и с описанием “зачем” в docs. citeturn15search2turn0search1turn4search2  

2) **Делить код по границам**:
- transport не содержит бизнес-правил; use-case не содержит HTTP деталей; репозитории не возвращают “сырой SQL error наружу”; маппинг делается на границе слоя. citeturn5search1turn5search0turn13search0  

3) **Логи структурированные** через `slog`, correlation/request id как поле контекста логгера; логировать события как поток (stdout). citeturn1search2turn2search3  

4) **Метрики и labels**: имена, типы и labels следуют Prometheus best practices; избегать high-cardinality labels. citeturn10search1turn10search0turn10search4  

### NEVER

1) **Никогда не использовать `panic` для ожидаемых ошибок** (валидация, not found, конфликт, upstream timeout). Ошибки должны быть выражены значениями и возвращены вверх. Это фундаментальная идиома Go error handling. citeturn0search1turn1search1  

2) **Никогда не игнорировать ошибки** (включая `defer rows.Close()`, `tx.Rollback()` при ошибке пути, `resp.Body.Close()`, ошибки сериализации/кодирования). Подобные “мелочи” — типичная причина деградации прод-систем и LLM-халлюцинаций. citeturn17view0turn6search5turn6search2  

3) **Никогда не логировать секреты/PII или токены**; не включать в error responses внутренние детали (SQL, stack traces). citeturn16search2turn16search13turn13search17  

4) **Никогда не вводить “архитектурный фреймворк”** (DI-контейнер, генераторы, абстракции ради абстракций) без явной потребности: Go архитектура должна оставаться читаемой. citeturn4search2turn0search0turn0search1  

## Concrete good / bad examples и типичные LLM-анти‑паттерны

Ниже — примеры, которые стоит включить в docs как “canonical”.

### Good: server timeouts + graceful shutdown

```go
srv := &http.Server{
	Addr:              cfg.HTTP.Addr,
	Handler:           handler,
	ReadHeaderTimeout: 5 * time.Second,
	WriteTimeout:      15 * time.Second,
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

Почему это good: поля таймаутов и лимитов соответствуют назначению `http.Server`; shutdown делается через `Server.Shutdown(ctx)` с таймаутом, что соответствует семантике стандартной библиотеки. citeturn8view0turn7view0  

### Bad: “безлимитный” сервер и exit без shutdown

```go
http.ListenAndServe(":8080", handler) // nolint
// нет таймаутов, нет graceful shutdown, ошибку игнорируем
```

Почему это bad: отсутствие таймаутов/лимитов и игнорирование возврата ошибки противоречит назначению server timeouts и семантике ошибок/закрытия сервера. citeturn8view0turn7view0  

### Good: http.Client с таймаутом и закрытием body

```go
client := &http.Client{
	Timeout: 5 * time.Second,
}

req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
if err != nil {
	return fmt.Errorf("new request: %w", err)
}

resp, err := client.Do(req)
if err != nil {
	return fmt.Errorf("do request: %w", err)
}
defer resp.Body.Close()
```

Почему это good: `Timeout` задан (0 означает “нет таймаута”), request использует контекст, а body закрывается. citeturn17view0turn14view0  

### Bad: default client без таймаута и утечка соединений

```go
resp, _ := http.Get(url) // ошибки игнорируются
// resp.Body не закрыт
```

Почему это bad: нарушаются требования `Client.Timeout` и закрытия `Response.Body`, что влияет на keep-alive reuse и может утекать ресурсы. citeturn17view0  

### Good: RFC 9457 Problem Details как единый контракт ошибок

```go
type Problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func writeProblem(w http.ResponseWriter, status int, p Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}
```

Почему это good: `application/problem+json` — стандартный media type для Problem Details; структурировано и машинно-читаемо. citeturn13search0turn13search7  

### Частые LLM-анти‑паттерны и галлюцинации

1) **“Интерфейс на каждый чих”** (interface pollution): модель создаёт десятки интерфейсов для структур, которые нигде не подменяются. В Go это ухудшает читаемость и усложняет навигацию; интерфейсы должны выражать реальные границы/порты. citeturn0search0turn4search2turn0search1  

2) **Смешивание слоёв**: handler напрямую формирует SQL или use-case возвращает `http.Status*`/JSON DTO. Это ломает тестируемость и делает невозможной смену транспорта. Такая проблема прямо противоречит целям clean/hexagonal (независимость от фреймворков/БД). citeturn5search1turn5search0turn5search14  

3) **Неправильные границы транзакций**: открытие транзакции в репозитории, который внутри вызывает другие репозитории и теряет контекст, либо забывает rollback на ошибках. В `database/sql` контекст влияет на rollback/commit; это нужно учитывать строго. citeturn6search2turn6search5  

4) **Неправильная работа с context**:
- `context.Background()` внутри request path вместо проброса request context;  
- хранение context в struct;  
- `WithTimeout` без `cancel()` → утечки. citeturn14view0  

5) **Observability-ошибки**:
- высококардинальные labels (request_id/user_id);  
- разные сервисы используют разные имена и семантики атрибутов;  
- попытка “автоматически логировать всё”, включая секреты. citeturn10search1turn1search11turn16search2  

## Review checklist и что оформить отдельными файлами в template repo

### Review checklist (PR / code review)

Этот чек‑лист стоит хранить как `docs/review-checklist.md` и как `.github/pull_request_template.md`.

- **Контракт и API**
  - OpenAPI/proto обновлены синхронно с кодом; backward compatibility осознанна. citeturn9search4turn13search8  
  - Ошибки соответствуют RFC 9457, статусы — по HTTP semantics, не утекли внутренние детали. citeturn13search0turn13search2turn13search17  

- **Контекст, таймауты, отмена**
  - context корректно проброшен во все IO; `WithTimeout/WithCancel` закрыт cancel’ом; нет хранения context в struct. citeturn14view0  
  - server/client timeouts заданы; нет “нулевых” бесконечных таймаутов по умолчанию. citeturn8view0turn17view0  

- **Транзакции и данные**
  - транзакционные границы определены в application layer; rollback гарантирован при ошибках; уровень изоляции PostgreSQL осознан. citeturn6search2turn6search3  

- **Security**
  - HTTPS-only допущение отражено в deploy/ingress; нет логирования секретов; авторизация по объектам/ролям проверяется явно. citeturn11search2turn16search2turn0search3  

- **Observability**
  - logs структурированы; метрики не имеют high-cardinality labels; trace context пропагируется. citeturn1search2turn10search1turn10search3  

- **Качество и поддерживаемость**
  - код отформатирован gofmt; соответствует Effective Go/CodeReviewComments; тесты обновлены. citeturn0search1turn0search0turn21search12  
  - запущен govulncheck; результаты учтены. citeturn3search4turn3search8  

### Какие файлы вынести в template repo

Ниже — практический список файлов, чтобы LLM-инструменты получали контекст автоматически, а разработчик “клонировал и работал”.

- `docs/engineering-standard.md`  
  Содержит: coding standard, dependency policy, error model (RFC 9457), timeouts policy, logging/metrics/tracing policy, security baseline (ASVS/OWASP API Top 10), конфиг/секреты, k8s ожидания. citeturn3search7turn0search3turn13search0turn8view0turn1search2  

- `docs/architecture.md`  
  Содержит: рекомендованную layered/clean/hexagonal “упрощённую” архитектуру, границы DTO↔domain, где валидация, где транзакции, как устроены middleware/interceptors. (Ссылки на первоисточники clean/hexagonal как “идеологический фундамент”, но с чёткими “не перегибать”.) citeturn5search0turn5search1turn5search14turn6search2  

- `docs/runbook.md`  
  Содержит: health endpoints, readiness/liveness, graceful shutdown в Kubernetes, как дебажить (логирование, метрики, pprof опционально). citeturn2search2turn11search0turn22search1turn7view0  

- `docs/review-checklist.md`  
  PR checklist (см. выше).  

- `docs/llm/` (или аналог) — “single source of truth” для LLM контекста:  
  - `docs/llm/context.md` — краткий контекст проекта + команды (`make test`, `go test ./...`, `govulncheck ./...`), принципы слоёв, запреты, error model. citeturn3search4turn14view0turn13search0  
  - `docs/llm/examples.md` — canonical good/bad snippets.  
  - `docs/llm/decision-matrix.md` — разрешённые развилки (HTTP vs gRPC, OTel vs Prometheus-first, и т. п.).  

- Tool-specific instruction entrypoints (опционально, но очень полезно):
  - `CLAUDE.md` — файл, который entity["organization","Anthropic","claude code vendor"] Claude Code читает в начале каждой сессии. citeturn19search2  
  - `.github/copilot-instructions.md` — репозиторные инструкции для entity["company","GitHub","code hosting platform"] Copilot. citeturn19search0  
  - `AGENTS.md` — инструкции для entity["company","OpenAI","ai lab company"] Codex, как описано в документации. citeturn19search17turn19search3turn19search7  
  - Cursor Rules — через механизм rules (документация Cursor). citeturn19search1  

- CI/automation:
  - `Makefile` (или `taskfile.yml`) с единым интерфейсом команд для людей и LLM (fmt/vet/test/vuln/build/run). (В Go tooling это особенно эффективно из-за стандартных команд gofmt/go tool/go test.) citeturn3search2turn3search8turn3search4  
  - `.github/workflows/ci.yml` — gofmt, go test, govulncheck. citeturn3search4turn0search4  

### Минимальный “LLM preamble”, который стоит поместить в CLAUDE.md / AGENTS.md / copilot-instructions

Смысл: дать модели **непротиворечивые правила** и **точки входа** (где что лежит), чтобы она не фантазировала архитектуру.

- “Проект использует Go 1.26.0; см. go.mod (go/toolchain).” citeturn2search4turn3search5  
- “HTTP routing — stdlib ServeMux patterns (Go 1.22+).” citeturn15search2turn15search0  
- “Ошибки HTTP — RFC 9457 Problem Details; не invent other formats.” citeturn13search0  
- “Контекст: правила из pkg.go.dev/context; cancel funcs must be called.” citeturn14view0  
- “Набор команд: make fmt/test/vuln; перед PR — gofmt/go test/govulncheck.” citeturn3search4turn3search8turn3search2  
- “Новые зависимости запрещены без явного RFC/ADR внутри repo.”  

## Приложение: опорные первоисточники (на которые стоит ссылаться в docs)

Этот список полезно включить в `docs/engineering-standard.md` как “authoritative baseline”.

- Go style/идиомы: Effective Go; Go Code Review Comments; Google Go Style Guide. (Последний прямо ставит целью уменьшать “guesswork”.) entity["company","Google","tech company"] citeturn0search1turn0search0turn4search2  
- Go modules/toolchain: Go Toolchains; Go Modules Reference; go.mod reference; управление зависимостями и `go.sum`. citeturn3search1turn3search5turn3search13turn6search19  
- Context: pkg.go.dev/context (правила, cancel leakage). citeturn14view0  
- net/http: Server timeouts/Shutdown; Client.Timeout и правила reuse/Body.Close; ServeMux routing enhancements. citeturn8view0turn17view0turn15search2turn7view0  
- Ошибки: Go 1.13 error wrapping (errors.Is/As/Unwrap). citeturn1search1  
- Логирование: slog (Go 1.21+); OWASP Logging Cheat Sheet; 12‑Factor Logs. citeturn1search2turn16search2turn2search3  
- API security: OWASP REST Security Cheat Sheet; OWASP API Security Top 10 2023; ASVS. citeturn11search2turn0search3turn3search7  
- Error contract: RFC 9457 (Problem Details); HTTP semantics RFC 9110; IANA status registry. entity["organization","IETF","internet standards body"] entity["organization","IANA","internet numbers authority"] citeturn13search0turn13search2turn13search6  
- Observability: OpenTelemetry Collector architecture/config; W3C Trace Context. entity["organization","W3C","web standards consortium"] citeturn10search9turn10search2turn10search3  
- Prometheus metrics: instrumentation practices; naming; data model. citeturn10search0turn10search1turn10search4  
- Kubernetes production semantics: probes; lifecycle hooks; security context; pod security standards. entity["organization","CNCF","cloud native foundation"] (как umbrella для cloud-native практик) и официальная дока Kubernetes. citeturn2search2turn11search0turn12search0turn12search1turn5search3