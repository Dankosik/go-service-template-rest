# Engineering standard и LLM-instruction docs для production-ready Go микросервиса

## Scope

Этот стандарт рассчитан на **greenfield**-микросервис на Go, который будет жить в “cloud native” окружении (контейнеры, декларативное деплой-описание, автоматизация, наблюдаемость). citeturn23search1 Он оптимизирован под ситуацию, когда разработчик **клонирует репозиторий и сразу пишет бизнес‑логику**, а LLM‑инструменты генерируют код **идиоматично, безопасно, поддерживаемо и предсказуемо**, не “угадывая” архитектуру и практики.

Подход применять, когда:
- Сервис — **самостоятельный бинарник** (обычно один main), и нет цели публиковать библиотеку как публичный API. Для серверных проектов Go‑документация рекомендует держать реализацию в `internal/`, а команды — в `cmd/`. citeturn22view0  
- Нужны “boring” дефолты: стандартная библиотека для HTTP‑сервера и роутинга (включая улучшения `ServeMux` и `PathValue`), стандартные подходы к контекстам, таймаутам, graceful shutdown, и т. п. citeturn13search0turn13search1turn15view0turn4view0  
- Требуется единый baseline для качества и review: стиль, ошибки, тесты, concurrency, безопасность, supply chain. citeturn0search1turn0search17turn0search19turn18search0turn17search0turn21search9  
- Планируется эксплуатация с базовой SLO‑логикой: метрики, трейсинг, логи, health probes. citeturn14search4turn14search2turn14search6turn14search5turn2search0  

Подход не применять (или применять с существенной адаптацией), когда:
- Нужен **сложный API gateway**, BFF с “web security headers” для браузера, SSR и т. п.: требования к кэшу, заголовкам и auth отличаются. (OWASP‑заголовки полезны, но не “серебряная пуля” и часть настроек релевантна именно браузерам.) citeturn1search8  
- Сервис — **библиотека/SDK** для внешних потребителей: вам нужно другое API‑версионирование, документация и структура пакетов (см. рекомендации Go по layout для пакетов/команд). citeturn22view0  
- Требуется строгая транзакционная согласованность и низкая латентность при записи, но кэш планируется как “система записи”: write-behind / write-back по умолчанию **не подходит** (подробно в секции про кэш). citeturn12view0turn11search2  
- Вы делаете высокорисковые домены (финансы, медицина, безопасность): этот стандарт остаётся полезным, но вам почти наверняка понадобится расширенный threat modeling и дополнительные контрольные меры beyond baseline (например, формальные security требования к API и доступам). citeturn14search3turn16search10  

Контекст-источники и “якоря” стандарта (авторитетные):
- entity["organization","Cloud Native Computing Foundation","cncf, linux foundation org"] — определение cloud native как подхода и ожидания по наблюдаемости/управляемости. citeturn23search1  
- entity["organization","OWASP","open web security project"] — практические cheat sheets по логированию, ошибкам, заголовкам, REST‑безопасности, вводу и т. п. citeturn1search11turn1search2turn1search5turn16search6turn16search1  
- entity["organization","IETF","internet engineering task force"] — стандарты HTTP caching (RFC 9111) и расширения stale content (RFC 5861). citeturn8search3turn5search1  

## Recommended defaults для greenfield template

Ниже — набор дефолтов, которые можно почти напрямую положить в `docs/engineering-standard.md` и “repo conventions”.

### Runtime и минимальная платформа

**Go version**
- MUST: фиксировать минимальную версию Go в репозитории. Рекомендуемый baseline на дату 2026‑03‑02 — **Go 1.26** (релиз 10 февраля 2026). citeturn0search0turn0search4  
- SHOULD: использовать свойства совместимости “Go 1 promise” и избегать зависимостей на нестабильные/экспериментальные API без явной необходимости. citeturn0search4  

**Структура репозитория**
- MUST: структура “server project”: `cmd/<service>/main.go` + `internal/…` для логики сервиса. citeturn22view0  
- SHOULD: держать “внешний” API пакетов минимальным (или отсутствующим): внутренние пакеты — в `internal/` чтобы не обещать стабильность внешним модулям. citeturn22view0  
- SHOULD: если появятся реально переиспользуемые части, выносить их в отдельный модуль/репозиторий (а не делать “случайный public pkg”). citeturn22view0  

**Конфигурация**
- MUST: конфигурация деплой‑зависимая хранится в **окружении** (env vars), а не в коде/репозитории. citeturn1search1turn1search14  
- SHOULD: “fail fast” при старте: если критичные параметры невалидны/отсутствуют — сервис не должен стартовать. (Это не формализовано в одном источнике, но напрямую следует из практики Twelve‑Factor “config in env” и эксплуатационной предсказуемости.) citeturn1search1  

### HTTP API: роутинг, контракты, безопасность

**HTTP server**
- MUST: использовать `net/http` и задавать серверные таймауты (`ReadHeaderTimeout`, `WriteTimeout`, `IdleTimeout`, лимиты заголовков). Документация `http.Server` прямо описывает семантику таймаутов и рекомендует `ReadHeaderTimeout` как более гибкий базовый контроль скорости, чем `ReadTimeout`. citeturn3view0turn4view0  
- MUST: graceful shutdown через `Server.Shutdown(ctx)` и ожидание завершения shutdown перед `main` exit; `Shutdown` описывает порядок (закрыть listeners → idle conns → ждать active) и предупреждает не завершать процесс раньше возврата из `Shutdown`. citeturn4view0  
- SHOULD: ограничивать размер request body с `http.MaxBytesReader` (защита от случайных/злонамеренных больших тел и расхода ресурсов). citeturn19view1  

**Routing**
- SHOULD (boring default): использовать улучшенный `http.ServeMux` (Go 1.22+) с method‑based patterns и path wildcards; `Request.PathValue` предоставляет значения wildcard. citeturn13search0turn13search1  
  Это уменьшает зависимость от сторонних роутеров и делает template более “portable” и предсказуемым для LLM.

**HTTP client**
- MUST: не использовать клиент без таймаута. `http.Client.Timeout` документирован как общий лимит (connect + redirects + чтение body), причём `Timeout=0` означает “без таймаута”. citeturn20view2  
- SHOULD: переиспользовать `http.Client` (и его `Transport`), поскольку транспорт держит внутреннее состояние (keep‑alive соединения) и клиент безопасен для concurrent use. citeturn20view2  

**Валидация и ответы об ошибках**
- MUST: валидировать входные данные как можно раньше; OWASP подчёркивает необходимость проверок длины/формата/типа и недоверие к входным параметрам. citeturn16search1turn16search6  
- MUST: не возвращать пользователю детали внутренних ошибок; OWASP рекомендует “generic response” наружу и логирование деталей на сервере. citeturn1search2  

**Минимальные security‑меры**
- SHOULD: применять набор HTTP security headers (с учётом контекста: public web vs internal API). OWASP даёт обзор заголовков и рекомендуемые конфигурации. citeturn1search8  
- SHOULD: учитывать OWASP API Security Top 10 (2023), особенно object-level authorization как систематический риск. citeturn14search3turn14search7  

### Observability: логи, метрики, трейсинг, health

**Structured logging**
- MUST (боринг дефолт): использовать стандартный `log/slog` для структурированных логов (msg + level + key-value атрибуты). citeturn2search0turn2search3  
- SHOULD: следовать security logging рекомендациям (не логировать секреты, отделять аудит/безопасность от debug, обеспечивать ответственность логов). citeturn1search5  

**Метрики**
- SHOULD: соглашения по именованию метрик и label‑ов брать из практик Prometheus (они не обязательны, но являются “style guide” и best practices). citeturn14search2  
- SHOULD: экспонировать метрики в формате OpenMetrics/Prometheus exposition, если вы выбираете pull‑подход; OpenMetrics описывает требования к формату и ожидание регулярной “экспозиции” снапшота. citeturn14search6  

**Трейсинг/корреляция**
- SHOULD: использовать OpenTelemetry сигналы и корреляцию (traces/metrics/logs). Спецификация описывает общий фреймворк и принципы, а также модель логов и корреляций. citeturn14search5turn14search1turn14search22  

**Health endpoints и probes**
- MUST: иметь отдельные endpoints/режимы, как минимум для readiness/liveness; Kubernetes определяет назначение probes (liveness — перезапуск при зависании/непрогрессе, readiness — готовность к трафику, startup — корректный старт). citeturn14search4turn14search0turn14search8  

### Data access: база данных и пул соединений

- SHOULD: при использовании SQL применять `database/sql` как базовую абстракцию и помнить, что `sql.DB` — concurrency‑safe и управляет пулом соединений; `Open` обычно вызывается один раз, а `DB.Ping` используется для проверки доступности. citeturn17search0turn17search3  

### Quality gates: тестирование и уязвимости

- MUST: иметь минимальную CI‑матрицу команд: `go test ./...`, `go vet ./...`, и security check зависимостей через `govulncheck`. `govulncheck` описан как “low‑noise” инструмент, который сканирует зависимости и привязывает найденные уязвимости к реально вызываемым символам. citeturn0search19turn0search7turn0search3  
- SHOULD: регулярно использовать race detector для конкурентного кода; Go документация объясняет природу data races и зачем нужен детектор. citeturn18search0  
- SHOULD: в тестах, где уместно, использовать table-driven стиль (Go wiki документирует его как практику). citeturn18search1  
- SHOULD: для функций, обрабатывающих потенциально враждебные/сложные входы, рассмотреть fuzzing: Go‑документация подчёркивает ценность fuzzing для поиска багов и даже классов security‑уязвимостей. citeturn18search2turn18search4  

**Supply chain / модули**
- MUST: понимать, что Go по умолчанию может скачивать модули через module mirror (`proxy.golang.org`) и аутентифицировать их через checksum database (`sum.golang.org`), управляемые Go team; это описано в документации `cmd/go` и в анонсе module mirror. citeturn21search9turn21search0  
- SHOULD: для приватных модулей корректно настраивать `GOPRIVATE/GONOSUMDB/GONOPROXY`, чтобы не “утекали” приватные пути в публичную sumdb (это напрямую вытекает из описания поведения `go` с proxy/sumdb). citeturn21search9turn17search11  

Вендорные источники для caching‑best‑practices, которые удобно использовать как baseline:
- entity["company","Amazon Web Services","cloud provider"]: описывает lazy caching (cache-aside), write-through, TTL, thundering herd и TTL randomness/jitter. citeturn6view0  
- entity["company","Microsoft","technology company"]: Azure Architecture Center описывает cache-aside pattern и его роль в согласованности. citeturn5search0  

## Decision matrix / trade-offs

Цель матрицы — заранее “закрыть” спорные места, чтобы LLM не выдумывала архитектуру, а выбирала из ограниченного множества вариантов.

### HTTP routing: stdlib ServeMux vs внешний роутер
- **Stdlib ServeMux (default)**: меньше зависимостей, меньше surface area, есть метод‑роутинг и path params через `Request.PathValue`. citeturn13search0turn13search1  
  Trade-off: меньше готовых middleware‑экосистем; часть вещей (группировка роутов, сложные матчеры) придётся решать вручную.
- **Внешний роутер**: больше ergonomic sugar, middleware, community‑примеры. Trade-off: зависимость, потенциальные breaking changes, больше вариантов “как принято”, что повышает риск LLM‑галлюцинаций и неоднородности.

### Таймауты: “везде контекст” vs “таймауты на сервере/клиенте”
- **Контекст + server/client timeouts (default)**: `context` в Go предназначен для deadline/cancellation; документация требует корректной передачи контекста и предупреждает о утечках при неиспользовании cancel func. citeturn15view0turn0search2  
  Trade-off: больше кода, дисциплина.
- **Без контекстов/таймаутов**: проще старт, но повышенный риск зависаний, resource leaks и лавинообразных отказов под нагрузкой (особенно в сетевых вызовах). Это прямо противоречит смыслу `context` и семантике `http.Client.Timeout`/`http.Server` timeouts. citeturn15view0turn20view2turn3view0  

### Логирование: fmt/log.Printf vs structured logging
- **`log/slog` (default)**: стандартизирует key-value логи для машинной обработки; Go blog прямо мотивирует structured logs как поиск/фильтрацию/анализ в проде. citeturn2search3turn2search0  
  Trade-off: нужно договориться о ключах/схеме, иначе получится “JSON‑спам”.
- **printf‑логи**: быстрее начать, но хуже корреляция и анализ; выше шанс случайного логирования чувствительных данных без структуры и классификации. OWASP подчёркивает важность осмысленного security logging. citeturn1search5  

### Кэширование: cache-aside vs write-through vs write-behind
- **Cache-aside (default)**: широко используется, проще всего, хороший baseline; есть и у AWS, и у Azure как базовый паттерн для производительности и контролируемой согласованности. citeturn6view0turn5search0  
  Trade-off: miss penalty; нужно аккуратно решать stampede/TTL/инвалидацию.
- **Write-through (разрешено точечно)**: лучше hit‑rate для “точно читаемых” агрегатов; AWS описывает плюсы/минусы и рекомендует комбинировать с lazy caching. citeturn6view0turn7view0  
  Trade-off: churn, “забивание” кэша ненужным, необходимость стратегии на сбои кэша.
- **Write-behind (НЕ default)**: увеличивает throughput на запись, но вводит eventual consistency и риск потерь/сложность пайплайнов; Redis описывает write-behind как async синхронизацию с backend и прямо показывает, что это отдельный слой/механизм, а не “пара строк кода”. citeturn12view0turn12view1turn11search2  

### Отказоустойчивость клиентов: retries
- **Exponential backoff + jitter (default для idempotent запросов)**: Google Cloud рекомендует exponential backoff с jitter и подчёркивает критерии (ответ + идемпотентность). citeturn2search2  
  Trade-off: нужно классифицировать ошибки и соблюдать идемпотентность; иначе можно усилить нагрузку (retry storms).
- **Без retries**: меньше сложность, но ниже resilience к transient ошибкам.

## Набор правил MUST / SHOULD / NEVER для LLM

Ниже — “инструкции для модели”, которые стоит вынести в `docs/llm/` (см. секцию про файлы). Формулировки сделаны как нормативные, чтобы уменьшить вариативность.

### MUST

LLM MUST:
- Следовать структуре “server project”: код сервиса в `internal/`, точки входа в `cmd/<service>/`. citeturn22view0  
- Делать весь публичный HTTP поверх `net/http`; если нужен роутинг — начинать со `http.ServeMux` patterns (method/path) и `Request.PathValue`. citeturn13search0turn13search1  
- Передавать `context.Context` первым аргументом во все операции, которые могут блокироваться/делать IO; **не хранить Context в структурах**, не передавать nil context, корректно вызывать cancel funcs (иначе утечки). citeturn15view0turn16search3  
- На сервере задавать таймауты (`ReadHeaderTimeout`, `WriteTimeout`, `IdleTimeout`) и лимиты (`MaxHeaderBytes`), а также ограничивать размер request body через `http.MaxBytesReader`. citeturn3view0turn19view1  
- Реализовывать graceful shutdown через `Server.Shutdown(ctx)` и ожидание завершения shutdown. citeturn4view0  
- Использовать структурированное логирование через `log/slog`. citeturn2search0turn2search3  
- Для outbound HTTP использовать `http.Client` с ненулевым `Timeout`; понимать, что `Timeout=0` — “нет таймаута”. citeturn20view2  
- Для SQL доступа учитывать, что `sql.DB` concurrency‑safe и управляет пулом; `Open` обычно вызывается один раз. citeturn17search0turn17search3  
- Валидацию входов делать явно и рано; недоверять входным параметрам. citeturn16search1turn16search6  
- Ошибки наружу отдавать безопасно (без внутренних деталей), детали логировать внутри. citeturn1search2  
- Добавлять/обновлять тесты: table-driven где уместно; для конкурентных участков — запускать (и не ломать) race detector. citeturn18search1turn18search0  
- Добавлять security gates: `govulncheck` и устранение найденного. citeturn0search19turn0search7  

### SHOULD

LLM SHOULD:
- Выбирать “boring defaults”: стандартная библиотека, минимальные зависимости, явные интерфейсы на границах. (Здесь рекомендация опирается на стремление снизить неопределённость/вариативность; источники — устойчивые практики Go layout/стиля.) citeturn22view0turn0search1turn17search1  
- В логах следовать security logging guidance: не логировать секреты/PII без основания, обеспечивать понятные события. citeturn1search5  
- Иметь readiness/liveness endpoints и логику, согласованную с Kubernetes probes (readiness — готовность обслуживать трафик, liveness — “живость”). citeturn14search4turn14search0turn14search8  
- Согласовать метрики по naming conventions и формату экспозиции (Prometheus/OpenMetrics). citeturn14search2turn14search6  
- Для retries, где это безопасно, использовать exponential backoff + jitter и уважать идемпотентность. citeturn2search2  
- При работе с кэшем применять TTL и, где нужно, TTL jitter, плюс stampede mitigation. citeturn6view0turn5search2  

### NEVER

LLM NEVER:
- Никогда не добавлять зависимость/фреймворк “потому что так проще” без объяснения trade-off и без фиксации решения в docs/decision record. (Нормативная цель — предотвратить “архитектуру из воздуха”.) citeturn22view0turn0search1  
- Никогда не прятать `context.Context` в поле struct, global var или singleton; не использовать `context.WithValue` для передачи опциональных параметров. citeturn15view0turn16search3  
- Никогда не использовать `http.Client` без таймаута и не оставлять `Timeout=0` в прод‑коде. citeturn20view2  
- Никогда не возвращать пользователю stack trace, SQL errors, внутренние сообщения и т. п. citeturn1search2  
- Никогда не писать “магическую” кэш‑инвалидацию или write-behind “в пару строк” без явной модели согласованности и без обработки отказов (см. секцию про кэш). citeturn12view0turn6view0  

## Исследование подтемы: cache patterns и consistency trade-offs

Эта секция — “нормативный набор patterns” для template: что разрешено по умолчанию, как реализовывать в Go, и какие anti‑patterns приводят к stale data, cache storms и неконтролируемой сложности.

### Нормативно разрешённые стратегии по умолчанию

**Baseline (разрешено и рекомендовано по умолчанию): cache-aside (lazy caching)**
- AWS называет lazy caching / cache-aside “foundation” стратегии, с простым flow: read → cache get → on miss read DB → cache set → return. citeturn6view0  
- Azure описывает cache-aside как способ улучшить производительность и помочь поддерживать согласованность между кэшем и источником данных. citeturn5search0  

**Read-through (разрешено как “форма”, но реализуется как cache-aside)**
- Azure отмечает: read-through можно эмулировать через cache-aside (приложение отвечает за загрузку в кэш по требованию). citeturn8search4  

**Write-through (разрешено точечно)**
- AWS описывает write-through: обновление кэша “в реальном времени” при обновлении базы (proactive), перечисляет плюсы и минусы и прямо рекомендует комбинировать write-through с lazy caching для покрытия misses и отказов кэша. citeturn6view0turn7view0  

**Write-behind / write-back (НЕ default; только как явное решение)**
- Redis описывает write-behind как стратегию, где кэш сам асинхронно синхронизирует изменения в backing database; приложение читает/пишет только в кэш, а кэш пушит изменения асинхронно. citeturn12view0  
- IBM формулирует write-behind как async write в backend (в отличие от write-through). citeturn11search2  
Вывод для template: write-behind допустим **только** при наличии формально принятой eventual consistency модели, требований к durability (например, через журнал/очередь), и чётких отказных сценариев. В baseline template его лучше не включать.

### Stampede protection и управление свежестью

image_group{"layout":"carousel","aspect_ratio":"16:9","query":["cache-aside pattern diagram","write-through cache diagram","stale-while-revalidate diagram","thundering herd cache stampede diagram"],"num_per_query":1}

**Thundering herd / cache stampede**
- AWS описывает thundering herd: множество процессов одновременно получают miss и параллельно бьют один и тот же дорогой запрос; TTL‑истечения могут усиливать эффект. AWS рекомендует prewarming и добавление случайности в TTL, если ключи массово истекают в одном окне. citeturn6view0  

**Request coalescing (singleflight)**
- Пакет `golang.org/x/sync/singleflight` документирует механику: `Do` гарантирует, что для ключа только один вызов in‑flight, а дубликаты ждут и получают тот же результат. citeturn5search2  
Норматив для template: при cache-aside для “дорогих” ключей MUST использовать coalescing либо локальный (singleflight в процессе), либо распределённый (lock), чтобы избежать stampede.

**Stale-while-revalidate**
- RFC 5861 определяет расширения Cache-Control для stale content, включая stale-while-revalidate. citeturn5search1  
- RFC 9111 уточняет, что stale ответы нельзя выдавать, если запрещают директивы, и что stale допустим, если явно разрешено клиентом/сервером или расширениями (в т. ч. RFC 5861). citeturn10view2  
Норматив для template: stale-while-revalidate допустим только для данных, где “слегка устаревшее” приемлемо, и должен быть **ограничен окнами**: `(fresh TTL)` и `(stale-while-revalidate window)`.

**TTL и TTL jitter**
- AWS рекомендует TTL почти для всех ключей (кроме случаев write-through), а также предлагает добавлять randomness к TTL, если ключи массово истекают в одном окне, чтобы снизить herd effect. citeturn6view0  
Норматив: в template MUST быть helper для TTL jitter (например, ±N% или ±Δsec) и политика по умолчанию.

**Negative caching**
- RFC 9111 явно допускает кэширование “negative results (e.g., 404)”. citeturn10view0  
Норматив: negative caching допустим для “не существует” / “пусто” результатов, но MUST иметь короткий TTL и MUST различать “не найдено” vs “ошибка источника”, чтобы не “закэшировать аварию”.

**Hot key mitigation**
- Redis описывает “Hot Keys” как анти‑паттерн: один ключ получает непропорционально большую долю трафика и становится bottleneck (особенно в cluster/sharding). citeturn8search17  
- Redis также даёт команды/механизмы для идентификации hotkeys (например, `HOTKEYS` container command). citeturn8search1  
Норматив: template должен предусматривать (а) метрики hit/miss и latency по ключам/префиксам, (б) опциональное key‑splitting / replication / локальный кэш для hot‑ключей, и (в) запрет на “один глобальный ключ на всех”.

### Конкретная реализация в Go: рекомендуемые building blocks

Ниже — примеры (good) как “встроить паттерн” в код и как (bad) обычно ломают систему. Код иллюстративный; в template стоит вынести API в `internal/cache`.

#### Good: cache-aside + singleflight + TTL jitter + negative caching

```go
package cache

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"golang.org/x/sync/singleflight"
)

// Sentinel for negative cache ("not found").
// Expose only domain-level semantics to callers; don't leak storage errors.
var ErrNotFound = errors.New("not found")

type Store[V any] interface {
	Get(ctx context.Context, key string) (V, bool, error)          // bool==found
	Set(ctx context.Context, key string, v V, ttl time.Duration) error
}

type Loader[V any] func(ctx context.Context) (V, error)

type Cache[V any] struct {
	store Store[V]
	sf    singleflight.Group
	rand  *rand.Rand
}

func (c *Cache[V]) GetOrLoad(ctx context.Context, key string, ttl time.Duration, jitterFrac float64, load Loader[V]) (V, error) {
	// 1) Fast path: cache hit
	if v, ok, err := c.store.Get(ctx, key); err == nil && ok {
		return v, nil
	}

	// 2) Coalesce concurrent misses
	vAny, err, _ := c.sf.Do(key, func() (any, error) {
		// Re-check after coalescing window (double-checked locking style)
		if v, ok, err := c.store.Get(ctx, key); err == nil && ok {
			return v, nil
		}

		v, err := load(ctx)
		if err != nil {
			// Negative caching example: cache "not found" short-lived
			if errors.Is(err, ErrNotFound) {
				_ = c.store.Set(ctx, key, v, time.Second*10) // short TTL
			}
			return v, err
		}

		ttl2 := jitter(ttl, jitterFrac, c.rand)
		_ = c.store.Set(ctx, key, v, ttl2)
		return v, nil
	})
	if err != nil {
		var zero V
		return zero, err
	}
	return vAny.(V), nil
}

func jitter(base time.Duration, frac float64, r *rand.Rand) time.Duration {
	if frac <= 0 || r == nil {
		return base
	}
	// ±(base * frac)
	delta := time.Duration(float64(base) * frac)
	if delta <= 0 {
		return base
	}
	return base - delta + time.Duration(r.Int63n(int64(2*delta+1)))
}
```

Почему это соответствует нормативу:
- singleflight подавляет дублирующиеся вычисления/загрузки и предотвращает stampede для одного ключа. citeturn5search2  
- TTL jitter как стратегия уменьшения “массового истечения” и herd эффект соответствует рекомендации AWS по добавлению randomness к TTL. citeturn6view0  
- Negative caching допустим как “negative result” в духе RFC 9111, но только с коротким TTL и с различением ошибок. citeturn10view0  

#### Bad: типовые ошибки, которые приводят к cache storms и stale data

```go
// BAD: no TTL => stale forever, memory blow-up, no self-healing on missed invalidation.
cache.Set(key, v, 0)

// BAD: synchronized TTL across many keys => mass expiry => thundering herd.
cache.Set(key, v, time.Hour)

// BAD: cache errors as "not found" => hides outages behind negative cache.
if err != nil {
	cache.Set(key, empty, time.Minute)
	return empty, nil
}

// BAD: no coalescing => 1000 concurrent misses => 1000 DB queries.
v, _ := loadFromDB()
cache.Set(key, v, time.Minute)
return v
```

Связь с источниками:
- AWS прямо предупреждает о herd effect и рекомендует TTL randomness. citeturn6view0  
- singleflight — стандартный building block подавления дублей. citeturn5search2  

### Versioned keys: как нормализовать в template (с оговорками)

“Versioned keys” в key-value кэше — это техника инвалидации через изменение версии/ревизии в ключе (например, `user:123:v17`). Прямого “RFC для Redis‑ключей” нет; здесь корректнее мыслить через аналогию с валидаторами HTTP кэша (ETag/validators и протоколы freshness/validation). RFC 9111 подробно описывает freshness/validation и то, что кэш может хранить не только 200, но и негативные ответы, редиректы и т. п. citeturn8search3turn10view0turn10view1  

Нормативное правило для template:
- Versioned keys MAY использоваться, когда есть **надёжный источник версии** (монотонная ревизия, updated_at, hash контента), и когда стоимость “оставшихcя старых ключей” приемлема (они уйдут по TTL).
- NEVER использовать versioned keys, если версия “угадывается” или может откатываться: это приводит к “вечной” устаревшей выдаче.

## Anti-patterns и типичные ошибки/hallucinations LLM

Это — список, который стоит включить в LLM‑инструкции как “запрещённые ходы”.

### Типовые галлюцинации про Go и stdlib

- “В stdlib есть готовый middleware stack / router / DI container” — нет, и template должен опираться на `net/http` + простые композиции. В Go 1.22 появились улучшения ServeMux patterns и `PathValue`, но это всё ещё `net/http`. citeturn13search0turn13search1  
- “Можно не закрывать cancel func” — неверно: документация `context` прямо говорит, что невызываемый CancelFunc приводит к утечке до отмены parent, и `go vet` проверяет использование cancel по путям управления. citeturn15view0  
- “Можно хранить context в структуре App/Service” — запрещено правилами `context` и доп. разъясняется Go blog. citeturn15view0turn16search3  

### Типовые ошибки в сетевом коде

- `http.Client` без таймаута (или `Timeout=0` по умолчанию) — создаёт риск бесконечных зависаний под сетевыми сбоями; документация чётко говорит, что `Timeout=0` означает “no timeout”. citeturn20view2  
- Сервер без `ReadHeaderTimeout/IdleTimeout` и без лимитов — повышает уязвимость к медленным клиентам и затратам ресурсов; семантика таймаутов/лимитов определена в `http.Server`. citeturn3view0  

### Ошибки в ошибках/логах и утечки данных

- Возврат внутреннего текста ошибок клиенту (stack traces, SQL ошибки, details) — нарушает рекомендации OWASP по error handling (наружу — generic, внутрь — детали). citeturn1search2  
- Логирование “всё подряд”: секреты, токены, пароли; OWASP Logging подчёркивает security‑аспекты и необходимость сфокусированных механизмов. citeturn1search5  

### Ошибки кэширования, приводящие к stale data и cache storms

- TTL отсутствует или одинаковый для больших групп ключей → стагнация данных и “массовое истечение”. AWS рекомендует TTL почти всегда и добавление randomness, если expirations синхронизируются. citeturn6view0  
- Нет stampede protection → thundering herd. AWS описывает эффект и mitigations; singleflight даёт локальный coalescing механизм. citeturn6view0turn5search2  
- Путают negative caching и error caching: RFC 9111 допускает кэширование негативных результатов (404), но это не значит “кэшировать аварии источника”. citeturn10view0  
- Hot keys игнорируются → bottleneck; Redis описывает “hot key” как анти‑паттерн, особенно в cluster. citeturn8search17  

## Review checklist для PR/code review

Чеклист рассчитан на то, что reviewer прогоняет глазами изменения. Его удобно положить в `docs/review-checklist.md` и ссылаться из PR template.

**Go idioms / стиль**
- [ ] Имена, packages, ошибки: соответствуют Effective Go и Go Code Review Comments (например, error strings без заглавных букв и пунктуации). citeturn17search1turn17search10  
- [ ] Нет “скрытой магии”: поведение читабельно, ошибки обрабатываются явно (согласно общей философии Go error handling). citeturn17search4  

**Context / concurrency**
- [ ] `context.Context` передаётся первым аргументом; не хранится в структурах; нет `nil` context; cancel funcs вызываются. citeturn15view0turn16search3  
- [ ] Для конкурентных участков есть тесты/проверки; при необходимости — `-race` чистый. citeturn18search0  

**HTTP server/client безопасность и устойчивость**
- [ ] `http.Server` сконфигурирован с таймаутами и лимитами (`ReadHeaderTimeout`, `WriteTimeout`, `IdleTimeout`, `MaxHeaderBytes`). citeturn3view0  
- [ ] Request body лимитируется через `http.MaxBytesReader` для эндпойнтов, принимающих тело. citeturn19view1  
- [ ] `http.Client` использует `Timeout` и переиспользуется. citeturn20view2  
- [ ] Реализован graceful shutdown через `Server.Shutdown`. citeturn4view0  

**Ошибки и безопасность**
- [ ] Клиент не получает внутренних деталей; детали логируются (OWASP error handling). citeturn1search2  
- [ ] Валидация входов: длины/форматы/типы/allow-list где уместно (OWASP input validation / REST security). citeturn16search1turn16search6  

**Observability**
- [ ] Логи структурированы через `log/slog` и содержат ключевые поля для корреляции. citeturn2search0turn2search3  
- [ ] Метрики согласованы по naming conventions; формат экспозиции соблюдает ожидания OpenMetrics/Prometheus. citeturn14search2turn14search6  
- [ ] Readiness/liveness реализованы и соответствуют назначению probes. citeturn14search4turn14search8  

**Dependencies / supply chain**
- [ ] Прогнан `govulncheck`, уязвимости обработаны/зафиксированы. citeturn0search19  
- [ ] Модульная конфигурация учитывает proxy/sumdb (особенно если добавлены приватные зависимости). citeturn21search9turn17search11  

**Кэш**
- [ ] Выбранный caching pattern соответствует разрешённым стратегиям (по умолчанию cache-aside; write-through/behind — только с явным решением). citeturn6view0turn12view0  
- [ ] Есть TTL + jitter (если ключи массовые), stampede protection (singleflight/lock), и (при необходимости) SWR‑окно с явными границами. citeturn6view0turn5search2turn5search1  
- [ ] Нет “кэширования ошибок” под видом negative caching. citeturn10view0  

## Что из результата нужно оформить отдельными файлами в template repo

Ниже — предлагаемая “раскладка” того, что должно стать файлами в репозитории, чтобы стандарты реально работали для людей и LLM.

`docs/engineering-standard.md`
- Содержит: версию Go baseline, layout (`cmd/`, `internal/`), обязательные таймауты/лимиты, правила по context, logging (`slog`), error handling, тесты и security gates. Основой служат официальные Go docs по layout, context rules, net/http таймауты, slog, govulncheck. citeturn22view0turn15view0turn3view0turn2search0turn0search19  

`docs/decisions/0001-http-routing.md`
- Зафиксировать: stdlib `ServeMux` patterns + `PathValue` как default и критерии, когда разрешён внешний роутер. citeturn13search0turn13search1  

`docs/llm/prefix.md`
- “Системный префикс” для LLM: MUST/SHOULD/NEVER правила из секции выше, плюс требование не добавлять зависимости/каталоги без decision record. (Цель — снизить вариативность и галлюцинации, опираясь на Go layout и context rules.) citeturn22view0turn15view0turn0search1  

`docs/llm/tasks.md`
- Набор “скриптов задач” для LLM: “add endpoint”, “add DB query”, “add cache wrapper”, “add metrics", “write tests”, “run govulncheck” — как чеклист действий, чтобы модель не пропускала важное. Основание: govulncheck как обязательный security gate; table-driven tests и race detector как практики. citeturn0search19turn18search1turn18search0  

`docs/caching.md`
- Полностью вынести секцию “cache patterns и consistency trade-offs”: разрешённые стратегии, реализация stampede protection, TTL/jitter, SWR, negative caching, hot keys, anti‑patterns. Основные источники: AWS caching best practices, Azure cache-aside, RFC 9111/5861, singleflight, Redis hot keys guidance. citeturn6view0turn5search0turn8search3turn5search1turn5search2turn8search17  

`docs/review-checklist.md`
- Чеклист из секции review, с прямыми ссылками на CodeReviewComments/TestComments и ключевые docs. citeturn0search1turn0search17turn15view0turn3view0turn0search19  

`docs/security-baseline.md`
- Короткий baseline: OWASP error handling, logging, input validation, REST security, API Security Top 10 2023 как “напоминания” и обязательные требования (не утекать деталями, валидировать входы, помнить про object-level auth). citeturn1search2turn1search5turn16search1turn16search6turn14search3  

`CONTRIBUTING.md` + `PULL_REQUEST_TEMPLATE.md`
- Встроить: обязательные команды CI (tests, vet, govulncheck), ссылку на review checklist, правило “без decision record — не меняем фундаментальные решения”.

Дополнение: если репозиторий подразумевает работу с приватными модулями/прокси — файл `docs/deps-and-supply-chain.md` с описанием proxy/sumdb поведения `go` и безопасной настройки переменных окружения. citeturn21search9turn21search0turn17search11