# Engineering standard и LLM-instructions для production-ready Go микросервиса

## Scope

Этот стандарт предназначен для **greenfield** микросервиса на Go, который собирается в **один статический (или почти статический) бинарник**, упаковывается в контейнер и запускается в среде оркестратора (типично entity["organization","Kubernetes","container orchestration"]) с прогнозируемыми L7/L4 health-check’ами, логами и метриками. Формат «сервер как продукт» хорошо ложится на рекомендации по структуре Go‑репозитория для серверных проектов: команды в `cmd/`, логика — в `internal/`, чтобы уменьшать публичную API‑поверхность и позволять рефакторинг без внешних импортеров. citeturn26view0turn7search4

Подход особенно полезен, когда важны «boring, battle-tested defaults»: стандартные инструменты цепочки Go (модули, `gofmt`, `go vet`, совместимость Go 1), минимальные и объяснимые зависимости, детерминированный запуск и наблюдаемость (logs/metrics/traces) по контракту. citeturn2search6turn21search3turn21search2turn19search0

Подход **не является оптимальным** в следующих случаях:
- Сервис не «микро» по жизненному циклу и границам ответственности (по сути модуль монолита в одном репо) — тогда часть правил о package boundaries и публичной API‑поверхности будет мешать. citeturn26view0
- Требуется специфическая среда исполнения: embedded/IoT, запрет контейнеров, жёсткие требования по размеру/зависимостям (например, нельзя тянуть entity["organization","OpenTelemetry","observability project"] SDK). В этом случае модульность и observability‑стек нужно выбирать отдельно. citeturn15search1turn15search4
- Сервис «чисто событийный» (message-driven) без HTTP/gRPC: большая часть примеров по `net/http` и аккуратной обработке тела запроса будет нерелевантна, но принципы context propagation/ошибок/границ зависимостей останутся полезными. citeturn0search15turn8search0turn15search6
- Вы делаете продуктовую публичную web‑UI поверхность (CSP, browser headers, HSTS становятся критичнее), тогда security‑заголовки/политики нужно расширять и строго тестировать под браузеры. citeturn13search0turn13search1turn13search4

## Recommended defaults для greenfield template

Ниже — набор «значений по умолчанию», которые можно почти напрямую превратить в `docs/` и репозиторные conventions. Эти defaults в первую очередь стремятся к **предсказуемости**, **безопасности** и **воспроизводимости**, а не к «самому модному стеку». citeturn19search0turn28view1turn2search6

### Базовая платформа и toolchain

- **Go версия**: целиться в текущий «latest stable» релиз (на дату запроса — Go 1.26) и отражать это в `go.mod` директивой `go 1.26`. citeturn18search0turn2search9  
- **Toolchain management**: использовать документированный механизм выбора toolchain’а и явно понимать его поведение в CI/локально (включая `GOTOOLCHAIN=local` при необходимости). Это снижает сюрпризы при сборках, но требует дисциплины по обновлениям. citeturn19search1turn18search11  
- **Совместимость Go 1**: считать «boring upgrades» нормой и регулярно обновляться; Go декларирует совместимость в рамках обещания Go 1, с оговорками (в т.ч. security‑фиксы). citeturn19search0turn19search3turn17search3

### Layout репозитория (server project)

Рекомендованный скелет (минимально достаточный):
- `cmd/<service>/main.go` — composition root (инициализация конфигурации, логгера, зависимостей, запуск серверов).
- `internal/...` — вся прикладная логика, транспорт, доступ к данным, observability.
- `docs/` — стандарты, инструкции для LLM, ADR (если нужно).
- `configs/` (опционально) — примеры конфигов для локального запуска; в проде — environment, секреты — отдельно. citeturn26view0turn4search7turn1search4

Такой «server layout» прямо рекомендован официальной документацией Go: Go‑пакеты логики держать в `internal`, команды — в `cmd`, а если появится реально переиспользуемая библиотека — выносить в отдельный модуль/репозиторий, а не публиковать «случайный API» из текущего сервиса. citeturn26view0turn7search4

### HTTP API: routing, JSON, timeouts, limits

- **HTTP стек**: стандартный `net/http` + стандартный `http.ServeMux`. Начиная с Go 1.22 он поддерживает более выразительные паттерны маршрутизации (метод в паттерне, path variables) и `Request.PathValue` для извлечения параметров. Это позволяет часто обойтись без стороннего роутера в greenfield. citeturn5search0turn5search1turn0search2  
- **JSON**: стандартный `encoding/json`, но с осознанной политикой «строгости»:
  - по умолчанию неизвестные поля в JSON при `Unmarshal` в struct **игнорируются**, что может создавать «тихие» ошибки; если сервису важна строгая схема — декодировать через `json.Decoder` и включать `DisallowUnknownFields()`. citeturn10view0turn11view0
  - помнить про неоднозначности JSON‑парсинга (дубликаты ключей, case-insensitive матчинг полей при decode в struct) и не строить security‑критичные гарантии на «как именно спарсится JSON». citeturn10view0
- **Лимиты запросов**:
  - ограничивать размер request body через `http.MaxBytesReader` (это отдельный инструмент именно для входящих HTTP‑тел). citeturn12search0
  - выставлять `Server.MaxHeaderBytes` для защиты от чрезмерных headers; помнить, что это не ограничивает body. citeturn9search1turn9search12
- **Timeouts** (boring default для JSON API): явно задавать server timeouts (`ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`). Значения по умолчанию в `net/http` часто «нулевые» (то есть «нет таймаута») и должны рассматриваться как небезопасный дефолт для прод‑API. citeturn9search12turn0search2  
- **Graceful shutdown**: `http.Server.Shutdown(ctx)` для корректного draining без прерывания активных соединений, в связке с `signal.NotifyContext`. citeturn0search2turn23search1turn23search0

### Конфигурация и секреты

- **Config через environment** как boring default (в духе 12‑factor): конфигурация отделяется от кода и задаётся через env vars; это упрощает повторяемые деплои одних и тех же артефактов в разные окружения. citeturn4search7turn4search11  
- **Secrets**: секреты не коммитятся в репозиторий и не «шардятся» между сервисами без необходимости; управление секретами требует централизованного хранения, аудита, ротации, и минимизации утечек. citeturn1search4turn13search17

### Observability contract: logs, metrics, traces

- **Логи**: использовать стандартный `log/slog` (структурные key-value логи) как единый API логирования внутри сервиса. citeturn5search2turn21search3  
- **Trace context propagation**: по умолчанию ориентироваться на W3C Trace Context (`traceparent`, `tracestate`) как vendor-neutral propagation стандарт; entity["organization","W3C","web standards org"] определяет формат, entity["organization","OpenTelemetry","observability project"] использует его как default propagator. citeturn4search0turn4search1  
- **SemConv**: использовать OpenTelemetry semantic conventions (единые имена атрибутов) как «контракт наблюдаемости» между сервисами и командами (дашборды/алерты переиспользуемее). citeturn1search2turn15search6  
- **Метрики**: экспортировать `/metrics` в формате, совместимом с entity["organization","Prometheus","monitoring system"] / OpenMetrics. Учитывать, что каждая комбинация label‑ов создаёт отдельный time series; не использовать метки с высокой кардинальностью (user_id, email и т.п.). citeturn1search3turn15search10

image_group{"layout":"carousel","aspect_ratio":"16:9","query":["OpenTelemetry architecture diagram context propagation traceparent tracestate","Prometheus metrics exposition format example","Kubernetes readiness liveness startup probes diagram","Go log slog structured logging example"],"num_per_query":1}

### Security baseline

- **Input validation**: валидировать входные данные на границе (HTTP handler / transport layer), используя allow-list подход там, где возможно. citeturn12search1  
- **REST over TLS**: для REST сервисов базовая рекомендация — предоставлять только HTTPS endpoints, защищая креды и данные в транзите; мTLS/клиентские сертификаты — опция для высокопривилегированных сервисов. citeturn12search2turn13search5  
- **HTTP security headers**: даже для JSON API полезно выставлять базовые защитные заголовки и не раскрывать лишние детали. citeturn13search0turn1search0  
- **Logging security**: логи должны быть пригодны для расследований, но не должны содержать секреты/чувствительные данные; придерживаться security‑ориентированных практик логирования. citeturn1search0turn1search4  
- Дополнительно: ориентироваться на практики entity["organization","OWASP","appsec foundation"] (Cheat Sheet Series) и, при необходимости формальной верификации, на ASVS как набор проверяемых требований. citeturn13search10turn1search1turn1search8

### Supply chain и dependency hygiene

- **Go modules** — системная основа зависимостей; поддерживать чистый `go.mod/go.sum` и избегать «лишних» зависимостей без явной пользы. citeturn2search6turn2search9  
- **Модульный proxy + checksum DB**: `go` по умолчанию может скачивать модули через proxy и аутентифицировать их через checksum database (`sum.golang.org`). Это важная часть цепочки доверия и воспроизводимости. citeturn2search12turn2search5turn2search1  
- **Vulnerability management**: включать `govulncheck` в CI как «low-noise» проверку зависимостей и реально используемых путей вызова. citeturn2search8turn2search4turn2search11  
- **SLSA**: минимум — обеспечить воспроизводимый build и наличие provenance (Build L1); дальше — повышать уровень по мере зрелости. citeturn2search10turn2search3  
- В cloud‑native жизненном цикле особое внимание уделяется supply chain safety: регулярные сканы артефактов, обновления, криптографическая подпись и неизменяемые образы/ссылки на образы — это прямо подчёркивается в руководствах entity["organization","CNCF","cloud native foundation"]. citeturn28view1

### Container build и runtime hardening

- **Multi-stage builds**: собирать бинарник в build stage и переносить только артефакт в runtime stage, уменьшая размер и поверхность атаки. citeturn14search0turn14search4  
- **Минимальный образ**: рассматривать distroless/scratch‑подходы как опцию уменьшения attack surface (с учётом требований к CA‑certs, timezone и отладке). citeturn14search3turn14search15  
- **SecurityContext в Kubernetes**: применять минимальные привилегии как часть деплоя; Kubernetes описывает механизмы security context для контейнера/пода. citeturn14search1  
- **OCI annotations/labels**: использовать стандартные OCI‑аннотации там, где нужен метаданный след (source, revision, version). citeturn14search6turn14search2

## Decision matrix / trade-offs

Ниже — «матрица решений» для типовых развилок template‑микросервиса. Везде указан boring default и ситуации, когда он может быть неправильным.

| Область | Вариант | Когда выбирать | Trade-offs / риски |
|---|---|---|---|
| HTTP routing | `net/http` `ServeMux` (default) | Greenfield на Go ≥1.22/1.26; хочется меньше зависимостей; достаточно method+path patterns и `PathValue`. citeturn5search0turn5search1 | Меньше middleware «из коробки», чем у фреймворков; часть инфраструктурных вещей нужно написать/принести явно. citeturn0search2 |
| HTTP routing | Сторонний роутер | Нужны сложные роутинг‑фичи/экосистема middleware, которые сложно/дорого поддерживать самим. (Спорно: зависит от команды.) citeturn26view0 | Больше supply chain, больше обновлений, больше surface area для LLM‑галлюцинаций («подключил пакет — забыл обновить конфиг/версии»). citeturn2search6turn2search8 |
| Логи | `log/slog` (default) | Единый стандартный API структурных логов; хочется избежать vendor lock-in. citeturn5search2turn21search3 | Может не совпадать с исторически выбранным стеком (zap/zerolog и т.п.); миграция требует адаптеров. citeturn5search2 |
| Метрики | Prometheus/OpenMetrics endpoint (default) | Стандартная модель pull‑метрик; простая эксплуатация; совместимость с OpenMetrics. citeturn1search3turn15search10 | Нужно дисциплинированно избегать high-cardinality labels; метрики должны быть договорённостью. citeturn15search10 |
| Tracing | OpenTelemetry + W3C propagation (default) | Vendor-neutral tracing/metrics/logs; единый propagation стандарт; удобно в service mesh/collector‑мире. citeturn4search0turn4search1turn15search4 | SDK и semconv — дополнительная сложность; не всем сервисам нужно «сразу». citeturn15search1turn1search2 |
| Config | Env vars (default) | 12‑factor подход; контейнерные окружения; простая параметризация без пересборки. citeturn4search7turn4search11 | Секреты в env требуют аккуратной operational практики; иногда лучше file-mount/secret manager. citeturn1search4 |
| Data access | `database/sql` + явный SQL (default) | Нужна переносимость, простой mental model, контроль над запросами; стандартная библиотека. citeturn22search0turn31search3 | Нужно помнить правила pool tuning, закрытия rows, контекстов и т.п.; иначе легко получить leaks/latency. citeturn31search1turn30search0 |
| Data access | PostgreSQL‑специфичный драйвер/toolkit | Если вы точно в PostgreSQL и нужны фичи/скорость; пример — pgx предоставляет и нативный интерфейс, и адаптер к `database/sql`. citeturn24search1turn24search5 | Менее переносимо на другие БД; нужно договориться, каким API пользуемся (pgx напрямую или `database/sql`). citeturn24search5 |
| DI | Явные конструкторы без DI framework (default) | Лучше видны зависимости; проще тестировать; меньше магии; соответствует go.dev guidance про интерфейсы и concrete returns. citeturn25search1turn26view0 | Больше «проводочного» кода; нужно поддерживать аккуратный composition root. citeturn26view0 |
| Container image | Multi-stage + минимальный runtime (default) | Снижение размера и attack surface; меньше «лишнего» в runtime. citeturn14search0turn14search4 | Сложнее дебажить внутри контейнера; нужен план observability и отладки без shell. citeturn14search3turn5search2 |
| Supply chain | SLSA L1 + vuln scanning (default) | Минимум зрелости supply chain: автоматизируем сборку и фиксируем происхождение, плюс govulncheck. citeturn2search10turn2search8 | Для higher levels нужны подпись, хостed build platform, политики допуска артефактов; не всегда оправдано в MVP. citeturn2search10turn28view1 |

## Набор правил MUST / SHOULD / NEVER для LLM

Это ядро, которое стоит оформить как «LLM‑instruction doc» в репозитории и использовать как общий префикс для моделей. Формулировки намеренно нормативные.

### MUST

- MUST **держать все Go‑пакеты логики сервиса в `internal/`**, а точку сборки/запуска — в `cmd/<service>/`. citeturn26view0  
- MUST **использовать `gofmt`** и не обсуждать форматирование в PR: формат — это результат `gofmt`, а не вкусовщина. citeturn21search3  
- MUST **передавать `context.Context` первым параметром** (`ctx`) во все операции, которые могут блокироваться или делать I/O (HTTP, DB, внешние вызовы), и **не передавать nil‑context** (использовать `context.TODO()` если нужно). citeturn0search15turn22search8  
- MUST **не хранить Context в struct**, кроме редких случаев совместимости с внешними интерфейсами; аргументировать исключение. citeturn0search1turn0search18  
- MUST **использовать `http.Server.Shutdown(ctx)`** для graceful shutdown и связывать shutdown с context, отменяемым по сигналу. citeturn0search2turn23search1  
- MUST **ограничивать input size** для HTTP: `MaxHeaderBytes` и `http.MaxBytesReader` (или эквивалентный механизм), прежде чем декодировать JSON. citeturn9search1turn12search0  
- MUST **явно настраивать HTTP server timeouts** (минимум `ReadHeaderTimeout`, `WriteTimeout`, `IdleTimeout`), не полагаясь на «нулевые» значения. citeturn9search12turn0search2  
- MUST **логировать структурно через `log/slog`** и не использовать `fmt.Println`/`log.Printf` в коде сервиса вне тестов/утилит, если repo‑стандарт выбрал `slog`. citeturn5search2turn21search3  
- MUST **оборачивать ошибки через `%w`** при добавлении контекста и использовать `errors.Is/As` для сопоставления по типам/цепочке. citeturn8search0turn8search1  
- MUST **следовать правилу интерфейсов**: интерфейсы, как правило, принадлежат пакету‑потребителю; реализатор должен возвращать concrete types, чтобы можно было добавлять методы без массового рефакторинга. citeturn25search1turn16search2  
- MUST **избегать высококардинальных меток метрик** (user_id, email и т.п.) и помнить, что каждая комбинация label‑ов — отдельный time series. citeturn15search10  
- MUST **использовать `govulncheck`** (или альтернативу с сопоставимым качеством) как часть CI, потому что он опирается на Go vulnerability database и снижает шум, учитывая реальные calls. citeturn2search8turn2search0

### SHOULD

- SHOULD **выбирать стандартный `ServeMux`** для роутинга в новых Go‑сервисах, если его паттернов достаточно (method patterns, wildcards, path variables). citeturn5search0turn5search1  
- SHOULD **делать JSON decoding строгим** в местах, где ошибка схемы критична (через `Decoder.DisallowUnknownFields()`), чтобы не допускать «тихих» несовпадений DTO. citeturn11view0turn10view0  
- SHOULD **использовать табличные тесты** (table-driven tests) там, где множество близких кейсов проверяются одинаковой логикой. citeturn20search0turn20search4  
- SHOULD **прогонять `go test -race` в CI** для пакетов, где есть конкурентность или shared state: race detector встроен в Go и ловит один из самых дорогих классов багов. citeturn33view0turn32search6  
- SHOULD **использовать fuzzing** для парсеров/валидаторов/протокольных обработчиков, где вход может быть атакующим: fuzzing полезен для нахождения edge cases и security‑эксплойтов. citeturn20search3turn20search7  
- SHOULD **ориентироваться на W3C Trace Context** и дефолтные propagators OpenTelemetry для межсервисной трассировки. citeturn4search0turn4search1  
- SHOULD **держать конфигурацию в env** и не «зашивать» environment‑specific значения в образ. citeturn4search7turn4search11  
- SHOULD **использовать multi-stage Docker builds** и переносить в runtime stage только собранный бинарник. citeturn14search0turn14search4  
- SHOULD **следовать практикам OWASP для логирования/секретов/валидации** как baseline security engineering guidance. citeturn1search0turn1search4turn12search1

### NEVER

- NEVER **придумывать зависимости «из воздуха»**: если пакет/модуль не присутствует в `go.mod`, нельзя ссылаться на него как на существующий. Любое добавление dependency должно быть осознанным, минимальным и отражённым в `go.mod/go.sum`. citeturn2search6turn2search9  
- NEVER **создавать интерфейсы «на всякий случай» до появления реального потребителя**: без реального usage слишком легко сделать неверный контракт. citeturn16search2  
- NEVER **делать service locator/глобальные синглтоны зависимостей** внутри `internal/...`: зависимости должны быть явными параметрами конструкторов/функций (исключения — строго обоснованы). Этот принцип напрямую следует из подхода Go к интерфейсам и тестируемости через consumer-side contracts. citeturn25search1turn26view0  
- NEVER **хранить секреты в логах** (токены, пароли, приватные ключи) или «случайно» логировать полные request/response без редактирования. citeturn1search0turn1search4  
- NEVER **отключать лимиты/таймауты «потому что мешают тестам»** — если лимит мешает, значит сервис не моделирует реальность или не предоставляет корректный конфиг‑override. citeturn9search12turn12search0  
- NEVER **использовать `io/ioutil` в новом коде**: пакет deprecated с Go 1.16, функциональность перенесена в `io`/`os`. citeturn29search0turn29search1  
- NEVER **использовать `context.WithValue` как механизм передачи бизнес‑данных** между слоями: context предназначен для request-scoped метаданных/сигналов отмены; бизнес‑данные передаются параметрами/структурами. citeturn0search18turn0search15

## Идиоматичные для Go паттерны проектирования кода

Эта секция — итог Theme #2 в виде практических правил и объяснений для LLM: как строить зависимости, где объявлять интерфейсы, и какие OO‑привычки (Java/C#) переносить нельзя.

### Composition и границы пакетов

- **Composition over inheritance** в Go реализуется через:
  - структурирование кода пакетами и явными зависимостями;
  - «встраивание» (embedding) там, где оно действительно упрощает API, а не создаёт скрытую магию. Общая цель — выразительность без усложнения. citeturn0search0turn26view0  
- Структура `internal/` — не «косметика», а механизм управления видимостью и границами: внешние модули не могут импортировать `internal/...`, что позволяет активно рефакторить внутреннюю архитектуру. citeturn26view0turn7search4turn7search19

**LLM‑правило**: компоненты должны формироваться вокруг функциональной связности (cohesion): пакет отвечает за одну ясную область ответственности; если пакет начинает «собирать всё подряд», это сигнал к разбиению. Это согласуется с целью server layout держать переиспользуемое наружу отдельно, а внутренности — в `internal`. citeturn26view0

### Интерфейсы: маленькие, по месту использования, без «загрязнения»

Ключевые идиомы из Go guidance:
- **Интерфейсы принадлежат потребителю**, а не реализатору; реализатор чаще возвращает concrete (struct/pointer), чтобы было возможно расширять реализацию без «ломающих» изменений контрактов. citeturn25search1  
- **Не определять интерфейс до появления использования** — иначе сложно понять, нужен ли интерфейс вообще и какие методы в нём должны быть. citeturn16search2

Практическая трактовка (anti-Java/anti-C#):
- В Go **не надо** заводить интерфейс «на каждый сервис/репозиторий», если сейчас у вас один реализатор и нет реальной причины для полиморфизма. Это ведёт к interface pollution и усложняет чтение кода; Go‑гайд прямо предостерегает от интерфейсов ради моков. citeturn25search1turn16search2  
- «Mock-friendly дизайн» в Go достигается тем, что интерфейс задаётся **на стороне потребителя** и обычно очень мал (ровно те методы, которые реально вызываются). Тогда тест легко подставляет fake/stub. citeturn16search2turn25search1

**LLM‑правило (конкретное)**:
- Если пакет A вызывает пакет B, **интерфейс для B объявляется в A**, рядом с местом использования, и включает только методы, которые реально нужны A. citeturn16search2turn25search1  
- Конструкторы для B возвращают **конкретный тип**, а A принимает интерфейс как зависимость. Это стабилизирует API и соответствует guidance «implementing package should return concrete types». citeturn25search1

### Dependency boundaries и «явные зависимости»

- **Composition root** — это `cmd/<service>/main.go`: только там разрешено собирать граф зависимостей, создавать конкретные реализации, и «склеивать» transport→service→storage. Этот подход следует из server layout рекомендации держать команды вместе и из идеи ограничивать публичную поверхность. citeturn26view0  
- **Запрещённые OO‑переносы**:
  - Service locator (глобальный контейнер зависимостей) — скрывает связи и усложняет тестирование; Go‑гайд по интерфейсам подталкивает к обратному: интерфейс у потребителя и concrete return у реализатора. citeturn25search1turn16search2  
  - DI frameworks «ради DI» часто создают неявность и усложняют трассировку ошибок; boring default — простые конструкторы и явные параметры. (Это рекомендация уровня engineering practice; источником служит общий Go‑подход к интерфейсам и структуре server projects.) citeturn25search1turn26view0

### Zero-value friendliness

Хотя «Make the zero value useful» известен как Go‑принцип, практическая импликация для template‑сервиса такая: структуры должны иметь безопасный нулевой смысл (или явно запрещать его), чтобы LLM не генерировала «конструкторную магию» и не плодила nil‑panic’и. Это особенно важно для middleware/конфиг‑структур и маленьких утилит. citeturn16search0turn0search0

## Concrete good / bad examples на Go

Ниже — примеры, которые стоит включить в docs как «образцы для LLM» (и как эталон для code review).

### Example: безопасный JSON handler с лимитами и строгим decoding

**Good**

```go
package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

type CreateWidgetRequest struct {
	Name string `json:"name"`
}

type WidgetService interface {
	CreateWidget(r *http.Request, name string) error
}

func CreateWidgetHandler(log *slog.Logger, svc WidgetService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Срок жизни запроса: привязываем к request context.
		ctx := r.Context()
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		// Ограничиваем тело запроса (например 1 MiB).
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		var req CreateWidgetRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		if err := dec.Decode(&req); err != nil {
			log.Info("bad request", "err", err)
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		if err := svc.CreateWidget(r, req.Name); err != nil {
			// Пример: разделяем доменные ошибки и 5xx.
			if errors.Is(err, ErrConflict) {
				http.Error(w, "conflict", http.StatusConflict)
				return
			}
			log.Error("create widget failed", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
```

Почему это good:
- неизвестные поля JSON не игнорируются «тихо», потому что используется `Decoder.DisallowUnknownFields()`. citeturn11view0turn10view0  
- размер request body ограничен `http.MaxBytesReader`, что прямо предназначено для защиты входящих тел. citeturn12search0  
- таймаут контролируется через контекст как стандартная модель отмены. citeturn0search15turn22search8

**Bad**

```go
func handler(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	_ = json.NewDecoder(r.Body).Decode(&req) // игнорируем ошибку
	name := req["name"].(string)             // panic при неожиданных типах
	doWork(context.Background(), name)       // теряем отмену по client disconnect
	fmt.Println("ok")                        // неструктурный лог
}
```

Что здесь плохо:
- игнорирование ошибок декодирования + потенциальный panic — прямое нарушение базовых практик работы с входными данными. citeturn12search1turn10view0  
- `context.Background()` ломает cancellation цепочку: `http.Request.Context()` отменяется при disconnect/cancel клиента и это должно «протекать» в I/O. citeturn22search8turn0search15

### Example: интерфейс на стороне потребителя и concrete return

**Good (consumer-side interface)**

```go
// internal/httpapi/handler.go
package httpapi

type UserReader interface {
	FindUser(ctx context.Context, id string) (User, error)
}

type Handler struct {
	users UserReader
}

func NewHandler(users UserReader) *Handler { // возвращаем concrete
	return &Handler{users: users}
}
```

```go
// internal/storage/postgres/users.go
package postgres

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// Store реализует интерфейс UserReader неявно.
func (s *Store) FindUser(ctx context.Context, id string) (User, error) { /* ... */ }
```

Почему это good: соответствует guidance, что интерфейсы обычно принадлежат пакету использования, а реализатор возвращает concrete типы, чтобы не «замораживать» API. citeturn25search1turn16search2

**Bad (interface pollution + implementor-side mocking)**

```go
// internal/storage/users.go
type UserStore interface {
	FindUser(ctx context.Context, id string) (User, error)
	CreateUser(ctx context.Context, u User) error
	DeleteUser(ctx context.Context, id string) error
	// ... ещё 20 методов "на будущее"
}

func NewUserStore(...) UserStore { ... } // возвращаем интерфейс
```

Проблема: интерфейс объявлен «на стороне реализатора» и раздут заранее; Go‑гайд прямо предостерегает от интерфейсов до использования и рекомендует возвращать concrete types. citeturn16search2turn25search1

### Example: ошибки и оборачивание

**Good**

```go
if err := doThing(); err != nil {
	return fmt.Errorf("doThing: %w", err)
}
```

Это даёт цепочку ошибок с `Unwrap`, пригодную для `errors.Is/As`. citeturn8search0turn8search1

**Bad**

```go
if err != nil {
	return errors.New("failed") // теряем первопричину
}
```

Потеря причинности ухудшает диагностику и обработку ошибок. citeturn8search0turn8search1

## Anti-patterns и типичные ошибки/hallucinations LLM

Ниже — практический список того, что чаще всего «ломает прод» в Go‑микросервисах и что LLM склонны генерировать, если репо‑стандарт не фиксирует правила.

### Неправильная работа с context и отменой

- Замена `r.Context()` на `context.Background()` «чтобы было проще» ломает контроль отмены и таймаутов; в HTTP сервере контекст запроса отменяется при disconnect/cancel клиента и это должно приводить к отмене внутренних операций (особенно DB). citeturn22search8turn0search15  
- Хранение context в struct «для удобства» противоречит правилам `context` и go.dev guidance; исключения редки и должны быть обоснованы совместимостью. citeturn0search15turn0search1turn0search18

### Отсутствие лимитов и таймаутов

- Не выставлены `ReadHeaderTimeout`/`WriteTimeout`/`MaxHeaderBytes`, не ограничен body — это делает сервис уязвимым к простым DoS‑паттернам и ресурсному истощению. citeturn9search12turn12search0

### Ошибки в JSON decoding и «тихие» несовпадения схемы

- `encoding/json` по умолчанию игнорирует неизвестные поля при `Unmarshal` в struct; LLM часто забывают включать `DisallowUnknownFields` там, где нужна строгая схема. citeturn10view0turn11view0  
- Игнорирование parsing‑нюансов JSON (дубликаты ключей, case-insensitive matching) при security‑критичных решениях. citeturn10view0

### Interface pollution и OO‑переносы

- Интерфейсы «для моков» на стороне реализатора и «God interfaces» из 10+ методов. Go guidance: не определять интерфейсы до использования; интерфейсы принадлежат пакету‑потребителю; реализатор возвращает concrete. citeturn16search2turn25search1  
- Сервис‑локатор и глобальные синглтоны зависимостей, из-за которых неясно, откуда берутся зависимости, и невозможны локальные тесты без сложной среды. Этот антипаттерн конфликтует с Go‑подходом «consumer-side interface» и с server layout, где `cmd/` — composition root. citeturn25search1turn26view0

### Проблемы с метриками и кардинальностью

- Метки вида `user_id`, `email`, `request_id` в Prometheus метриках создают неограниченную кардинальность и могут «убить» storage/ресурсы. Prometheus прямо предупреждает не использовать labels с высокой кардинальностью. citeturn15search10

### Deprecated и неактуальные куски standard library

- Использование `io/ioutil` в новом коде (частое у LLM из старых примеров): пакет deprecated с Go 1.16, нужно использовать `io`/`os`. citeturn29search0turn29search1

## Review checklist для PR / code review

Этот список стоит положить в репозиторий как обязательный чек‑лист ревью для изменений, с акцентом на типичные failure modes.

### Correctness и API-contract

Проверить, что:
- handlers и сервисные методы корректно используют `ctx` из `r.Context()`; нет `context.Background()` внутри request flow без явного обоснования. citeturn22search8turn0search15  
- ошибки не теряют причинность: при добавлении контекста используется `%w`, а сопоставление делается через `errors.Is/As`. citeturn8search0turn8search1  
- JSON decoding: там, где это важно, включён `DisallowUnknownFields`; body ограничен. citeturn11view0turn12search0

### Надёжность: timeouts, shutdown, Kubernetes probes

Проверить, что:
- `http.Server` сконфигурирован с разумными таймаутами и лимитами заголовков/тел. citeturn9search12turn9search1turn12search0  
- graceful shutdown реализован через `signal.NotifyContext` (или эквивалент) и `Server.Shutdown(ctx)`. citeturn23search1turn0search2  
- readiness/liveness разделены корректно (readiness — готовность обслуживать трафик; liveness — «жив ли процесс»), и probes не делают тяжёлых проверок. citeturn3search4turn3search0  
- если используются lifecycle hooks (preStop), учтено, что preStop выполняется в рамках termination grace period. citeturn3search1turn3search5

### Observability

Проверить, что:
- логирование делается через `log/slog` и ключи/поля согласованы (нет случайного `fmt.Printf`). citeturn5search2turn21search3  
- trace context propagируется по W3C Trace Context, если сервис участвует в distributed tracing. citeturn4search0turn4search1  
- метрики не содержат high-cardinality labels; имена метрик/лейблов соответствуют Prometheus практикам. citeturn15search10turn1search3

### Security

Проверить, что:
- входные данные валидируются на границе и не происходит «слепого» доверия к JSON/params. citeturn12search1  
- логи не содержат секреты/PII; нет логирования сырых токенов/паролей. citeturn1search0turn1search4  
- HTTPS/TLS требования отражены в инфраструктурной документации; для REST сервисов базовая рекомендация — только HTTPS. citeturn12search2turn13search5

### Tooling и supply chain

Проверить, что:
- `go.mod/go.sum` обновлены корректно; нет лишних зависимостей «просто потому что модель так написала». citeturn2search6turn2search9  
- CI включает `govulncheck` и базовые проверки (`gofmt`, `go vet`). citeturn2search8turn21search3turn21search2  
- при конкурентности есть покрытие `go test -race` (хотя бы в nightly или на важных пакетах). citeturn33view0

## Что из результата оформить отдельными файлами в template repo

Ниже — рекомендуемая «раскладка документов» и repo conventions, которую можно почти напрямую перенести в `docs/` и корень репозитория.

### Документы в `docs/`

- `docs/engineering-standards.md`  
  Свод стандартов: layout (`cmd/` + `internal/`), правила по `context`, error handling, logging, timeouts, dependency policy. citeturn26view0turn0search15turn8search0turn5search2turn9search12

- `docs/llm/instructions.md`  
  MUST/SHOULD/NEVER для LLM (раздел «Набор правил…»), плюс «Definition of Done» для PR. citeturn16search2turn25search1turn2search8turn21search3

- `docs/architecture.md`  
  Короткая модель слоёв и границ: transport → application/service → storage; где объявляются интерфейсы; что считается composition root. citeturn26view0turn25search1turn16search2

- `docs/observability.md`  
  Observability contract: поля логов, соглашение по trace/propagation, базовые метрики и правила кардинальности, ссылки на semconv. citeturn5search2turn4search0turn1search2turn15search10

- `docs/security.md`  
  Минимальный security baseline: input validation, logging & secrets, TLS требования, SSRF/redirect правила для исходящих HTTP‑вызовов (если есть), базовые headers. citeturn12search1turn1search0turn1search4turn13search5turn13search0turn13search3

- `docs/operations.md`  
  Runbook‑уровень: graceful shutdown, probes, ожидания по сигналам/termination, настройки таймаутов и лимитов, контейнерные практики. citeturn23search1turn0search2turn3search4turn14search0

- `docs/supply-chain.md`  
  Dependency policy (proxy/sumdb), `govulncheck`, минимальные требования SLSA/provenance, политика контейнерных образов и multi-stage build. citeturn2search12turn2search8turn2search10turn14search4turn28view1

### Репозиторные conventions (в корне)

- `go.mod` (и при необходимости `toolchain` директива по политике команды) — фиксирует язык/зависимости. citeturn2search9turn19search1  
- `Makefile` или `task`‑runner: стандартизованные команды `fmt`, `test`, `test-race`, `lint`, `vuln`, `build`, `docker-build`. (Выбор инструмента — локальная политика; важна воспроизводимость.) citeturn21search3turn2search8turn33view0  
- `Dockerfile` (multi-stage) + политика тегов/лейблов (OCI annotations). citeturn14search0turn14search6  
- `.github/workflows/ci.yml` (или эквивалент в вашей CI системе): `gofmt`, `go vet`, `go test ./...`, `govulncheck`, опционально `go test -race` и статический анализ. citeturn21search2turn2search8turn33view0  
- `.golangci.yml` (опционально) если команда выбирает entity["company","GitHub","code hosting company"]‑ориентированный линтинг через golangci-lint; иначе достаточно `go vet` + точечных инструментов. citeturn21search0turn21search2  
- `CONTRIBUTING.md`: как запускать локально, как добавлять зависимости, как писать тесты (table-driven), как соблюдать стандарты. citeturn20search0turn2search6turn21search3  
- `CODEOWNERS`/политика code review («four eyes principle») — особенно важно, потому что изменения инфраструктуры/кода могут иметь далеко идущие security‑эффекты; cloud‑native security guidance отдельно подчёркивает ценность такого контроля. citeturn28view1