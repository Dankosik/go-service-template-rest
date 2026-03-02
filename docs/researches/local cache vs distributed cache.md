# Engineering standard и LLM-инструкции для production-ready микросервиса на Go

## Scope

Этот стандарт и template предназначены для **greenfield** микросервиса на Go, который должен быть “boring” и предсказуемым: без магии, с минимальными внешними зависимостями, со встроенными guardrails для безопасности, наблюдаемости и стабильной эксплуатации. Он оптимален для сервисов, которые деплоятся как контейнеры и живут в оркестрации уровня entity["organization","Kubernetes","container orchestration"], используют стандартные health checks, масштабируются горизонтально и должны быть удобно сопровождаемы (включая разработку с помощью LLM-инструментов). citeturn2search2turn2search6turn4search0turn13search16

Подход применим, когда:
- сервис **статлес** (или почти статлес) и его состояние внешнее (БД, очереди, кэш, внешние API), что соответствует идее “backing services как attached resources”. citeturn13search16turn9view0  
- вы хотите, чтобы “clone → build → run → deploy” был максимально прямолинейным и повторяемым за счёт Go modules и управляемого toolchain. citeturn10search34turn0search1turn0search12  
- вы сознательно делаете ставку на **стандартную библиотеку** (net/http, log/slog, database/sql) как “default stack”, добавляя внешние зависимости только там, где стандартная библиотека не закрывает production-требования (например, OpenTelemetry SDK). citeturn4search1turn0search7turn0search3turn6search0turn2search1  

Подход **не подходит** или требует существенных модификаций, когда:
- нужен “hard real-time”/ультранизкие латентности и строгие SLO на tail latency, где каждый аллок/GC-пауза критичны и стандартные “boring defaults” (особенно локальные кэши и JSON) могут быть не тем уровнем контроля. citeturn16search0turn16search7turn7search6  
- сервис — не микросервис, а монолит/платформа/CLI с иными приоритетами (другой deployment model, иная наблюдаемость, другой SLA). citeturn3search33  
- вы строите публичный internet-facing API, но не готовы системно закрывать риски уровня **security misconfiguration**, ресурсного истощения и инвентаризации API (документация, версии, debug endpoints), которые entity["organization","OWASP","security foundation"] напрямую относит к критическим рискам API-безопасности. citeturn14search1turn14search3turn14search0turn14search5  
- требуется интенсивный streaming (SSE/long polling/bidi streaming) на том же http.Server, где “грубые” server-level timeout’ы могут ломать сценарии (например, WriteTimeout делает соединение конечным по времени и конфликтует со streaming). citeturn15search8turn15search3  

## Recommended defaults для greenfield template

Ниже — “набор по умолчанию”, который должен быть зафиксирован прямо в репозитории (docs + конфиги + CI), чтобы LLM не угадывала стек и правила, а считывала их как контракт.

**Версии Go и воспроизводимость**
- Target: **Go 1.26.x** как актуальный стабильный релиз (вышел в феврале 2026), чтобы использовать современную стандартную библиотеку и инструменты без “полугодового хвоста”. citeturn0search11turn0search8turn0search4  
- В `go.mod` фиксировать `go 1.26` (go directive влияет на выбор toolchain/семантику новых возможностей) и использовать механизм Go toolchains (Go ≥1.21) для повторяемых сборок/CI. citeturn0search12turn0search1turn0search8  

**Структура репозитория и модулей**
- Один модуль (один `go.mod`) на сервис; приватные пакеты — в `internal/` (официально рекомендуемый способ ограничить API поверхности пакетов внутри репозитория). citeturn3search33turn3search21  
- Если в репо несколько бинарей (например, сервис + мигратор), располагать их в отдельных директориях (пример официальной схемы “multiple commands”). citeturn3search33  
- Док-комментарии и публичные API пакетов оформлять по правилам Go doc comments (чтобы `go doc`, `pkg.go.dev` и IDE показывали документацию корректно). citeturn3search16turn3search32  

**HTTP слой (boring default без фреймворков)**
- Использовать стандартный `net/http` и **ServeMux** как дефолтный роутер. В Go 1.22 в стандартной библиотеке появились method-based patterns и wildcards, что закрывает многие причины тянуть внешний роутер для простых сервисов. citeturn4search1  
- Сервер запускать через явный `http.Server`, а не через `http.ListenAndServe*`, чтобы задать timeouts и лимиты (нулевой/дефолтный сервер не имеет таймаутов). citeturn15search29turn15search3  
- Обязательные базовые ограничения/защиты на вход:
  - `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout` и `MaxHeaderBytes` в `http.Server` — осознанно, с пониманием trade-off для streaming. citeturn15search3turn15search8turn15search29  
  - Лимит размера тела запроса через `http.MaxBytesReader` (явно описан как мера против больших/злоумышленных тел и лишних затрат ресурсов). citeturn6search1turn6search5  
  - Строгий JSON decode там, где это уместно: `json.Decoder.DisallowUnknownFields()` для предотвращения “тихого принятия” лишних полей. citeturn4search2  

**Клиенты исходящих HTTP-запросов**
- `http.Client` **переиспользовать**, потому что `Transport` держит состояние (пулы соединений), а клиенты concurrency-safe; на каждый запрос не создавать новый клиент. citeturn5search0  
- Явно задавать `Client.Timeout` или эквивалентные deadline/timeout через контекст: `Timeout=0` означает “нет таймаута”, что в production может приводить к зависающим запросам. citeturn5search10turn5search0  
- Опираться на cancellation через `Request.Context` (net/http явно связывает client cancellation с завершением контекста запроса). citeturn5search10turn11view0  

**Ошибки и контекст**
- Ошибки “обогащать” контекстом через `fmt.Errorf("...: %w", err)` и проверять причины через `errors.Is/As`, а не строковыми сравнениями. citeturn1search3  
- `context.Context` — обязательный “сквозной” параметр: входящий request создаёт контекст, исходящие вызовы принимают контекст, цепочка обязана его прокидывать; не хранить контекст в struct; `CancelFunc` всегда вызывать (иначе утечки), и `go vet` это проверяет. citeturn11view0turn6search2  
- Для параллельных подзадач использовать `errgroup` как стандартный инструмент для синхронизации, отмены и распространения ошибок. citeturn7search1  

**Логирование**
- “Boring default”: `log/slog` как стандартная structured logging API в Go 1.21+, с key-value полями и хорошей совместимостью. citeturn0search7turn0search3  
- Логи писать как event stream в stdout (12-factor), не управлять файлами логов внутри приложения. citeturn4search0turn4search0turn13search16  
- Логи и события безопасности: следовать практикам entity["organization","OWASP","security foundation"] — не логировать секреты/PII напрямую; защищать логи от подмены/несанкционированного доступа. citeturn12search1turn13search7turn13search19  

**Наблюдаемость (traces/metrics)**
- Дефолт: entity["organization","OpenTelemetry","cncf observability"] SDK (в приложении) + экспорт через OTLP (обычно в OpenTelemetry Collector). Для HTTP использовать `otelhttp`-обёртки, а атрибуты — по semantic conventions (HTTP semconv — стабильный документ). citeturn2search1turn2search5turn2search4turn2search8  
- Важно: “logs signal” в OpenTelemetry Go может быть экспериментальным; значит, логирование как сигнал наблюдаемости лучше считать отдельной задачей и не строить критически важные контракты на экспериментальном API. citeturn2search1  

**Health endpoints и эксплуатация**
- Делать отдельные endpoints под liveness/readiness/startup и документировать их контракт; это базовая механика операционной устойчивости в entity["organization","Kubernetes","container orchestration"]. citeturn2search2turn2search6  
- Graceful shutdown: `http.Server.Shutdown(ctx)` и обработка сигналов через `signal.NotifyContext`, обязательно вызывая `stop()` для восстановления поведения сигналов. citeturn4search18turn10search0turn10search8  

**База данных и безопасный доступ**
- Default подход: `database/sql` (pooling встроен; `sql.DB` concurrency-safe и управляет пулом соединений). Настраивать ограничение пула и timeouts на уровне контекста. citeturn6search0turn5search1turn6search4  
- SQL Injection защита: parameterized queries / prepared statements как первичный стандарт. citeturn12search0turn12search4  

**Security scanning и стресс-тестирование**
- В CI обязательно включить `govulncheck` (низкошумный анализ, который показывает уязвимости, реально достижимые из вашего кода, по данным базы vuln.go.dev). citeturn1search0turn1search4turn1search5  
- В CI и/или PR пайплайне гонять race detector (`go test -race`) для улавливания data races (как минимум на unit/интеграционных тестах). citeturn7search0  

**Profiling / debug endpoints**
- `net/http/pprof` полезен, но по умолчанию регистрируется на default mux и несёт риск случайного публичного экспонирования диагностических данных; endpoint’ы `/debug/pprof/` должны быть защищены (internal-only, auth, отдельный порт/сервис). citeturn10search2turn15search21turn10search15  
- Для диагностики опираться на официальные материалы по diagnostics/profiling. citeturn15search1turn15search2turn15search17  

## Decision matrix / trade-offs

Ниже — “матрица решений” (в инженерном смысле) для template и для LLM: какие решения являются boring defaults, какие требуют явного подтверждения/контекста.

**net/http ServeMux vs сторонний router**
- Default: `net/http` + улучшенный ServeMux (Go 1.22) — меньше зависимостей, меньше surface area, проще переносимость. citeturn4search1  
- Когда нужен внешний router: сложные middleware chains, advanced routing features, которые всё ещё удобнее у специализированных библиотек; но это должно быть явным ADR-решением, а не “LLM так привыкла”. (Trade-off: зависимость, скорость обновлений, security posture). citeturn4search1turn14search1  

**Timeout’ы сервера: безопасность vs streaming**
- Default: таймауты включены, потому что “нулевой сервер без таймаутов” — плохая идея для “операции в интернете/недоверенной сети”. citeturn15search29turn15search3  
- Trade-off: server-level `WriteTimeout` ломает долгоживущие ответы/стриминг, поэтому для streaming endpoint’ов нужно либо отдельный сервер, либо точечные решения (и это должно быть документировано). citeturn15search8turn15search3  

**Config через env vars vs secret manager / volumes**
- 12-factor рекомендует хранить конфигурацию в окружении (простота и переносимость). citeturn13search0turn13search16  
- Но secrets требуют отдельной дисциплины: entity["organization","Kubernetes","container orchestration"] поддерживает Secrets как объект для чувствительных данных и прямо отделяет Secrets от ConfigMaps; secrets можно прокидывать как env или как volume mount, и нужно учитывать encryption-at-rest настройки кластера и риск утечек при неправильной эксплуатации. citeturn13search1turn13search5  
- Практический boring default для template:  
  - **не-секретная** конфигурация: env vars;  
  - **секреты**: Secret store / Kubernetes Secrets, предпочитая delivery как файл (volume mount) там, где это возможно (особенно для сертификатов/ключей/бинарных данных), но фиксируя это как инфраструктурный контракт. citeturn13search5turn10search20turn12search2  

**Observability: OpenTelemetry “везде” vs точечные библиотеки**
- Default: entity["organization","OpenTelemetry","cncf observability"] — vendor-neutral стратегия, стандартизированные атрибуты/семконвы (особенно для HTTP spans). citeturn2search8turn2search4turn2search1  
- Trade-off: сложнее initial setup (SDK init, exporters, resources), но платится один раз в template; также важно помнить, что logs signal в Go может быть экспериментальным. citeturn2search9turn2search1  

**Профилирование в production: “можно” vs “опасно”**
- Default: оставить возможность подключить pprof (например, build tag или отдельный internal listener), но запретить включать его публично. citeturn10search15turn15search21turn10search2  
- Trade-off: удобство диагностики vs риск утечки и operational риск (любой debug endpoint — часть attack surface). citeturn14search1turn15search21  

**Исследование подтемы: local in-memory cache vs distributed cache**

Ниже — практический гайд, который должен лечь в template как отдельный документ и как часть LLM инструкции выбора архитектуры.

**Ключевые оси сравнения**

Latency  
- Local in-memory кэш обычно выигрывает по latency, потому что нет сетевого RTT и сериализации поверх сети; distributed cache добавляет сетевые round-trips, и “серийные операции” резко увеличивают общую латентность (поэтому нужны батчи/пакетирование). citeturn8search2turn8search5turn9view0  

Consistency и staleness  
- Любой кэш — копия данных, и основная проблема — “как оставаться достаточно актуальным”. Cache-aside прямо описывает риски устаревания и необходимость стратегии инвалидирования/TTL. citeturn8search3turn9view0  
- В распределённых системах/репликациях возможны эффекты eventual consistency: один инстанс может закэшировать “старое” значение, если исходное хранилище ещё не синхронизировано; документированная проблема для cache-aside в системах с eventual consistency. citeturn9view0  

Warmup / cold start  
- Cache-aside (lazy loading) даёт “штраф первого запроса”: miss → поход в БД → заполнение → ответ; это прямо отмечается как недостаток cache-aside (дополнительные round trips). citeturn8search2turn8search3  
- Pre-warm/seed возможен, но большой seed может создать резкий всплеск нагрузки на исходное хранилище при старте — это риск, который нужно тестировать. citeturn9view0  

Memory pressure и GC (особенно для local cache на Go)  
- Local cache увеличивает live heap; Go GC управляет heap под memory limit и нагрузкой runtime, а при приближении к лимиту становится более агрессивным (больше CPU на GC). Следовательно, “просто добавим кэш в память” может ухудшить tail latency и CPU, если не ограничить размер/эвикцию и не профилировать. citeturn16search0turn16search7turn16search2  
- Практический вывод для template: local cache MUST иметь **жёсткие лимиты** (size/cost), эвикцию, TTL и мониторинг hit/miss + память/GC. citeturn9view0turn16search0turn16search22  

Horizontal scaling  
- Local cache “не шарится” между инстансами → при горизонтальном масштабировании растёт суммарная память (N копий) и divergence. Это может быть приемлемо для “read-mostly/immutable” данных, но опасно для часто меняющихся. citeturn9view0  
- Distributed cache централизует данные и лучше подходит, когда важно чтобы разные инстансы видели одинаковую кэшированную картину и когда нужно снизить нагрузку на БД при большом количестве реплик приложения. citeturn9view0turn8search3  

Eviction управляемость  
- В distributed cache типично есть глобальный лимит памяти и eviction policies: Redis явно описывает, что при превышении лимита памяти ключи будут эвиктиться согласно выбранной политике, пока использование не вернётся “под лимит”. citeturn8search0  
- В managed окружениях дефолты могут отличаться: например, AWS описывает eviction policy настройки и упоминает дефолт `volatile-lru` для ElastiCache Redis OSS. Это значит, что “как будет себя вести кэш при заполнении” — не абстракция, а конкретная настройка, которую нужно фиксировать в конвенциях деплоя. citeturn8search8  

Serialization overhead и типы данных  
- Distributed cache почти всегда требует сериализации (JSON/MsgPack/whatever) и перенос значения по сети; для bulk операций нужны batch API/пакетирование, иначе сетевой overhead доминирует. Практики по batch/операциям и опасности “одна команда — один RTT” подчёркнуты в Redis anti-patterns. citeturn8search5turn9view0  

Operational complexity и failure modes  
- Shared cache — это внешняя зависимость: приложение должно уметь **обнаруживать недоступность** shared cache и деградировать (fallback на исходное хранилище), иначе оно станет неустойчивым. Это прямо указано в guidance по кэшированию (и там же упоминается применимость Circuit Breaker). citeturn9view0  
- Вы должны учитывать replication/failover/clustering как способы HA для кэша (и понимать cost/сложность). citeturn9view0  

**Практический выбор для template (правило большого пальца)**

- Выбирай **local cache**, если:
  - данные read-heavy и **редко меняются** (или допускают staleness); citeturn9view0  
  - нужен экстремально низкий latency без сетевого RTT; citeturn8search5turn9view0  
  - кэш — “оптимизация внутри инстанса”, и divergence между репликами не критичен (например, кэш результатов вычислений, memoization). citeturn9view0  

- Выбирай distributed cache (Redis-like), если:
  - сервис масштабируется горизонтально и нужно, чтобы кэш работал “на весь флот” инстансов; citeturn9view0  
  - нужно разгрузить БД при большом количестве реплик и повторяющихся чтениях; citeturn9view0  
  - есть требования к централизованной политике eviction/TTL и наблюдаемости hit/miss на уровне всей системы; citeturn8search0turn9view0  
  - вы готовы принять operational расходы и failure mode “кэш недоступен”. citeturn9view0  

- Выбирай **hybrid (local + shared)**, если:
  - нужен быстрый “горячий” local cache, но при этом важно иметь shared cache как промежуточный слой и буфер при недоступности shared cache (Microsoft прямо описывает topology “local private cache + shared cache” и указывает на риск staleness и необходимость аккуратной конфигурации). citeturn9view0  

**Частые ошибки архитектурного выбора кэша (встречаются чаще всего)**
- Использовать кэш как “источник истины” для критичных данных: guidance прямо рекомендует не делать кэш авторитетным хранилищем и сохранять критичные изменения в персистентное хранилище. citeturn9view0  
- Не задать TTL/эвикцию и получить заполнение памяти/эвикции “внезапно” в самый плохой момент. citeturn9view0turn8search0  
- Cache-aside без защиты от stampede и без понимания cold misses → пики нагрузки на БД. citeturn8search2turn9view0  
- Для Redis-like кэша выполнять N запросов последовательно вместо batched операций и получить деградацию latency из-за round-trips. citeturn8search5turn9view0  

**Как LLM должна выбирать кэш (алгоритм для инструкции)**
- Сначала определить: “кэш вообще нужен?” (если есть явный bottleneck на БД/вычисления/внешние API; иначе — не добавлять). Guidance рекомендует кэшировать read-frequently / modified-infrequently и тестировать эффективность. citeturn9view0  
- Затем спросить/вытащить из контекста: требования к staleness, допустимость устаревания, критичность консистентности. citeturn8search3turn9view0  
- Определить масштабирование: один инстанс vs many replicas. citeturn9view0  
- Определить отказоустойчивость: что происходит, если кэш недоступен; fallback на origin обязателен для shared cache. citeturn9view0  
- Определить limits: память, TTL, eviction policy, max entry size, мониторинг hit/miss. citeturn8search0turn9view0turn16search0  

## Набор правил в формате MUST / SHOULD / NEVER для LLM

Эти правила должны лечь в отдельный файл “LLM instructions” и использоваться как общий префикс/контракт для ChatGPT/Codex/Claude Code. Их цель — минимизировать догадки и типовые hallucinations, заставляя модель опираться на репозиторий как источник “истины”.

**MUST**
- MUST сначала прочитать `README.md` + `docs/` (особенно engineering standard, ADR/decisions, conventions), прежде чем генерировать код или рефакторить. (Причина: уменьшить “догадки” и architectural drift — это прямой ответ на проблему hallucinations.) citeturn14search0turn14search1  
- MUST соблюдать `gofmt`/`go fmt` как единственный источник форматирования; любые style-споры решаются gofmt. citeturn3search1turn3search4turn3search32  
- MUST прокидывать `context.Context` сквозь весь call chain; `ctx` — первый параметр; не хранить context в struct; не забывать `cancel()`/`stop()` где требуется. citeturn11view0turn10search0  
- MUST использовать `fmt.Errorf(... %w ...)` и `errors.Is/As` для семантической обработки ошибок. citeturn1search3  
- MUST для исходящих HTTP запросов переиспользовать `http.Client` и задавать таймауты/дедлайны (не полагаться на нулевые значения). citeturn5search0turn5search10  
- MUST включать защитные лимиты на вход (request body limit, header limits) и документировать их как часть API контракта (для защиты от resource consumption/DoS и misconfiguration). citeturn6search1turn15search3turn14search3turn14search1  
- MUST не логировать секреты/токены/PII; при необходимости — редактировать/маскировать/хэшировать. citeturn12search1turn13search7turn13search19  
- MUST при доступе к БД использовать параметризованные запросы/prepared statements и контекстные методы (`QueryContext`/`ExecContext`). citeturn12search0turn6search0turn11view0  
- MUST добавлять/обновлять наблюдаемость: traces (и при необходимости metrics) через entity["organization","OpenTelemetry","cncf observability"], соблюдая semantic conventions. citeturn2search5turn2search4turn2search8  

**SHOULD**
- SHOULD держаться стандартной библиотеки (net/http + ServeMux, log/slog) и добавлять новые зависимости только при явной необходимости и фиксации в ADR/decisions. citeturn4search1turn0search7turn14search1  
- SHOULD реализовывать graceful shutdown через `http.Server.Shutdown` и сигналы через `signal.NotifyContext`, чтобы корректно завершать in-flight запросы в оркестрации. citeturn4search18turn10search0  
- SHOULD включать `govulncheck` и race detector в CI (или как минимум в обязательные локальные команды), потому что это уменьшает “скрытые” дефекты и зависимостные риски. citeturn1search4turn7search0  
- SHOULD держать API-инвентарь/версионирование/документацию актуальными, иначе вы попадаете в риск API inventory management (включая забытые debug endpoints). citeturn14search0turn14search5  
- SHOULD предлагать кэш (local или distributed) только после объяснения цели, стратегии invalidation/TTL и failure modes; кэш не “магическое ускорение”. citeturn9view0turn8search3turn8search2turn8search0  

**NEVER**
- NEVER использовать `http.ListenAndServe`/default server как “полное решение” без timeouts/лимитов. citeturn15search29turn15search3  
- NEVER использовать `io/ioutil` в новом коде (deprecated с Go 1.16). citeturn3search3turn3search19  
- NEVER строить SQL строковой конкатенацией с пользовательским вводом. citeturn12search0turn12search12  
- NEVER делать кэш источником истины для критичных данных или создавать критическую зависимость от доступности shared cache (должен быть fallback на origin). citeturn9view0  
- NEVER открывать `/debug/pprof` наружу без защиты (это часть attack surface и может утекать чувствительными данными). citeturn15search21turn10search15turn10search2  
- NEVER “добавлять библиотеку потому что так принято” без указания: почему стандартная библиотека не подходит, какое влияние на security posture, и какие альтернативы. citeturn14search1turn3search32  

## Concrete good / bad examples

Ниже — примеры, которые стоит прямо включить в docs/ как “канонические паттерны”.

### Good: production-ish HTTP server с timeouts + graceful shutdown + сигналами

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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MiB (явное значение как часть контракта)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("http server starting", "addr", srv.Addr)
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server error", "err", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	logger.Info("shutdown starting")
	_ = srv.Shutdown(shutdownCtx)
	logger.Info("shutdown complete")
}
```

Почему это “good”:
- explicit `http.Server` позволяет задать timeouts и лимиты (в отличие от “нулевого сервера”) и использовать `Shutdown(ctx)` для graceful shutdown. citeturn15search29turn4search18turn15search3  
- `signal.NotifyContext` даёт простую отмену по SIGINT/SIGTERM и требует `stop()` для восстановления поведения сигналов. citeturn10search0turn10search8  
- ServeMux patterns с методами — это современная стандартная библиотека (Go 1.22+). citeturn4search1  

### Bad: “быстро подняли сервер” с дефолтами и без защиты

```go
package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Плохо: default server без timeouts/лимитов.
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Почему это “bad”:
- package-level helpers используют дефолтный сервер без timeouts; для production это риск ресурсного истощения и неконтролируемых зависаний. citeturn15search29turn14search3  

### Good: безопасный JSON decode с лимитом тела и строгими полями

```go
package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
)

const maxBody = 1 << 20 // 1 MiB

func DecodeJSONStrict(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return err
	}
	// Защита от "лишнего" JSON после первого объекта.
	if dec.More() {
		return errors.New("unexpected extra JSON tokens")
	}
	return nil
}
```

Обоснование:
- `http.MaxBytesReader` предназначен для ограничения размера тела и экономии ресурсов при атакующих/случайно больших запросах. citeturn6search1turn6search5  
- `DisallowUnknownFields` заставляет падать на неожиданных полях вместо “тихой” ошибки маппинга. citeturn4search2  

### Good: HTTP client как singleton dependency с таймаутом

```go
package outbound

import (
	"net/http"
	"time"
)

func NewClient() *http.Client {
	return &http.Client{
		Timeout: 3 * time.Second, // total timeout
	}
}
```

Обоснование:
- Клиенты должны переиспользоваться (Transport держит кеш соединений), `Timeout=0` значит “нет таймаута”. citeturn5search0turn5search10  

### Bad: новый http.Client на каждый запрос и без таймаута

```go
func Fetch(url string) (*http.Response, error) {
	c := &http.Client{} // Timeout=0 => может висеть вечно
	return c.Get(url)
}
```

Почему это “bad”:
- Нулевой timeout означает отсутствие таймаута; создание клиента “на каждый запрос” ухудшает управление соединениями/ресурсами. citeturn5search10turn5search0  

## Anti-patterns и типичные ошибки/hallucinations LLM

Ниже — список “типовых провалов” LLM в Go-микросервисах, которые стоит явно запретить/перехватывать ревью и линтингом.

**Неявные или устаревшие практики**
- Использование `io/ioutil` (deprecated) вместо `os.*` / `io.*`. citeturn3search19turn3search3  
- “Случайные” `context.Background()` внутри request handler вместо `r.Context()` и отсутствующая прокидка `ctx` вниз по stack. citeturn11view0  
- Игнорирование `CancelFunc` / `stop()` → утечки контекстов/таймеров либо неправильная работа сигналов; Go прямо предупреждает, что невызов cancel утечёт до отмены родителя, а `go vet` это ловит. citeturn11view0turn10search0  

**Сеть и устойчивость**
- Использование `http.ListenAndServe`/default server без timeouts и лимитов (частая LLM-галлюцинация “так быстрее”). citeturn15search29turn15search3  
- Использование `http.DefaultClient`/клиента без таймаутов для внешних API (навешивает риск зависаний и истощения ресурсов). citeturn5search10turn5search0  

**Безопасность**
- Логирование токенов/паролей/секретов “для дебага”; OWASP прямо рекомендует не писать чувствительные данные в логи и защищать лог-файлы как высокоценный актив. citeturn12search1turn13search7turn13search19  
- “Временный” публичный `/debug/pprof` “на минуту” — но минут не бывает: pprof по дизайну добавляет стандартные эндпойнты и может привести к утечкам, а также есть известный security risk из-за default mux. citeturn15search21turn10search2turn10search15  
- Конкатенация SQL строкой с пользовательским вводом (SQLi). citeturn12search0turn12search12  

**Кэширование (особенно частые архитектурные ошибки)**
- Добавить local cache “без лимитов” → рост heap → рост GC нагрузки/CPU, деградация tail latency. citeturn16search0turn16search2turn9view0  
- Использовать distributed cache синхронно/серийно (N round trips) вместо батчей/пакетов. citeturn8search5turn9view0  
- Делать shared cache обязательной зависимостью без fallback на origin. citeturn9view0  

## Review checklist для PR/code review

Этот чеклист стоит прямо встроить в `CONTRIBUTING.md` и использовать в PR template.

Корректность и идиоматичность Go  
Проверить, что код отформатирован gofmt и следует базовым Go conventions; gofmt — предписанный стандарт, а Go Code Review Comments — канонический справочник типовых замечаний. citeturn3search4turn3search1turn3search32  

Контексты, отмена и утечки  
Проверить: `ctx` прокинут вниз, нет `nil` contexts, `CancelFunc` не забыты, контекст не хранится в struct, request-scoped values не используются как “опциональные параметры”. citeturn11view0  

HTTP сервер и входные лимиты  
Проверить: нет `ListenAndServe`/нулевых дефолтов; таймауты и `MaxHeaderBytes` заданы; request body лимитируется (`MaxBytesReader`); для JSON — строгий decode там, где это часть API контракта. citeturn15search29turn15search3turn6search1turn4search2  

HTTP клиенты и внешние вызовы  
Проверить: клиенты переиспользуются, таймауты/дедлайны заданы, отмена корректна через контекст. citeturn5search0turn5search10turn11view0  

Ошибки и логирование  
Проверить: ошибки оборачиваются через `%w`, нет строковых сравнений; логи структурированы (slog), не содержат секретов/PII/токенов и защищены по политике организации. citeturn1search3turn0search7turn12search1turn13search7  

База данных  
Проверить: `sql.DB` используется как shared handle (pool), нет открытия “на каждый запрос”; запросы параметризованы; контекст применяется. citeturn6search0turn5search1turn12search0  

Наблюдаемость и эксплуатация  
Проверить: есть trace propagation/инструментация через OpenTelemetry, соблюдены semconv; health endpoints соответствуют ожиданиям оркестратора; graceful shutdown корректен. citeturn2search5turn2search4turn2search2turn4search18turn10search0  

Security/Vuln и конкурентность  
Проверить: PR не добавляет “случайные” зависимости; CI включает `govulncheck`; тесты запускаются с race detector там, где это возможно. citeturn1search4turn7search0  

Кэширование (если добавлено/изменено)  
Проверить: описана стратегия (cache-aside/write-through и т.п.), TTL/эвикция/лимиты, failure modes, отсутствие критической зависимости от shared cache, и оценён эффект на память/GC для локального кэша. citeturn8search2turn8search0turn9view0turn16search0  

## Что из результата нужно оформить отдельными файлами в template repo

Ниже — практическая нарезка, чтобы этот результат “почти напрямую” стал содержимым `docs/` и repo conventions.

**Root-level contracts**
- `README.md`: quickstart (локальный запуск, тесты, линтеры, переменные окружения, health endpoints), “what’s included / what’s not”. Основание: 12-factor про прозрачный запуск и stdout логи; Kubernetes probes; Go modules/toolchain. citeturn4search0turn2search2turn0search1turn10search34  
- `CONTRIBUTING.md`: PR checklist (раздел выше), правила gofmt, go vet, govulncheck, race detector. citeturn3search4turn6search2turn1search4turn7search0  
- `docs/engineering-standard.md`: весь “Recommended defaults” + “Decision matrix”, как нормативный стандарт. citeturn15search29turn3search1turn11view0turn9view0turn14search1  

**LLM-инструкции**
- `docs/llm/instructions.md`: MUST/SHOULD/NEVER + список запрещённых анти-паттернов + “как добавлять зависимости” (только через ADR). citeturn11view0turn3search19turn15search21turn14search1  
- `docs/llm/prefix.md`: копипастимый префикс для LLM, включающий:
  - “сначала прочитай docs/ и следуй стандарту”;  
  - “не угадывай”;  
  - “если вводные неполны — предлагай boring default и фиксируй предположения”, особенно для кэша (алгоритм выбора local vs distributed). citeturn9view0turn14search0turn14search1  

**Архитектура и эксплуатация**
- `docs/architecture.md`: high-level схема (HTTP API, зависимости, observability, shutdown, health). citeturn4search18turn2search2turn2search5  
- `docs/observability.md`: OpenTelemetry setup, required attributes/semconv, что экспортируем, где sampling, как связывать trace/span IDs с логами (без привязки к экспериментальному logs signal). citeturn2search1turn2search4turn2search8  
- `docs/security.md`: OWASP API Security Top 10 2023 как чеклист угроз (misconfiguration, resource consumption, inventory); правила логирования/секретов; запрет публичного pprof; правила параметризации SQL. citeturn14search6turn10search3turn12search0turn15search21  
- `docs/cache.md`: “local vs distributed cache” (раздел выше) + требования к любой реализации кэша (TTL, eviction, мониторинг hit/miss, fallback). citeturn9view0turn8search0turn8search2turn16search0  

**Runbooks и диагностика**
- `docs/runbook.md`: как проверять readiness/liveness, как снимать профили (если включено), как безопасно включать pprof (internal-only). Использовать официальные материалы по diagnostics/pprof. citeturn15search1turn10search2turn2search2  

**Repo conventions / tooling (configs)**
- `Makefile` или `taskfile`: цели `fmt`, `vet`, `test`, `test-race`, `vuln`, `lint` (lint опционально, но vet обязателен). citeturn6search2turn7search0turn1search4turn3search12  
- `.github/workflows/ci.yml` (или аналог): enforce gofmt/go vet/go test/govulncheck/race detector минимум на main ветке. citeturn6search2turn7search0turn1search4  

**Примечание о расширяемости template**
- Любые “не-boring” решения (другой роутер, другой логгер, иной формат API, иной кэш) должны оформляться как ADR. Это одновременно снижает риск security misconfiguration и риск “improper inventory management” в части дрейфа API/инструментов. citeturn14search1turn14search0turn14search5