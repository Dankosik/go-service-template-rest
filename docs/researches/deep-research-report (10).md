# Engineering standard и LLM-инструкции для production-ready Go-микросервиса с упором на package design и модульные границы

## Scope

Этот стандарт рассчитан на **зелёный старт** (greenfield) и на ситуации, где нужен **универсальный, повторяемый, production-ready шаблон микросервиса на Go**, который можно клонировать и сразу развивать (в том числе с помощью LLM-инструментов). В частности, он подходит, когда сервис является **самодостаточным бинарём** (или небольшой группой бинарей), деплоится как контейнер и чаще всего живёт в оркестраторе (часто — Kubernetes) с health-checks, метриками и трассировкой. Такой профиль соответствует тому, как официальная документация Go рекомендует организовывать “server projects”: держать логику сервера в `internal/`, а команды — в `cmd/`. citeturn3view0turn13search4turn13search0

Подход намеренно “boring” и консервативный: он оптимизирован на **минимум догадок** со стороны LLM и на **предсказуемые границы** (чтобы модель не “размазывала” зависимости и не плодила мусорные пакеты вроде `util`). Это согласуется с тем, что Go поощряет ясную структуру пакетов, короткие и говорящие имена, а также отказ от бессодержательных “свалочных” пакетов. citeturn9view0turn10view0turn24view0

Не применять этот стандарт как “по умолчанию”, если ваш репозиторий — **публичная библиотека / SDK**, который должен быть импортируемым внешними потребителями, и где вы обязаны поддерживать стабильный public API. Для библиотек официальный гайд по структуре модулей предлагает другие формы организации (пакеты в корне/поддиректориях модуля), а для серверных репозиториев прямо рекомендует держать серверную логику внутри `internal/`. Если вам нужно массово переиспользовать код между репозиториями, “правильный” путь часто — вынести общую часть в отдельный модуль (или отдельный репозиторий/модуль), а не делать её внутренней и полупубличной. citeturn3view0turn4search4turn4search13

Не стоит также “насильно” внедрять слои `domain/application/infrastructure`, если сервис очень мал (условно: один HTTP handler и один клиент) и реальная архитектурная сложность отсутствует: Go-сообщество в целом предпочитает **организацию по ответственности и месту использования**, а не по абстрактной “типологии файлов” вроде `models`/`types`. Этот стандарт поэтому даёт слойную схему как **контроль границ**, но требует удерживать пакеты маленькими и предметными — иначе вы получите аналог “monorepo-mvc” внутри одного микросервиса. citeturn18view0turn9view0turn24view0

## Recommended defaults для greenfield template

Ниже — набор дефолтов, которые можно практически напрямую перенести в `docs/` и в репозиторий-шаблон.

**Версия Go и политика обновлений**

Шаблон должен фиксировать **Go 1.26** как базовую версию (в `go.mod`), потому что Go 1.26 официально выпущен **10 февраля 2026** и является “latest Go release” на текущий момент. citeturn5search3turn5search0turn5search1  
Для production-репозиториев важно следить за minor-релизами из-за security fixes: политика Go говорит, что security fixes выпускаются для двух самых свежих major-веток, а minor-релизы “backportятся” для безопасности и серьёзных проблем. citeturn6search3turn6search2turn6search0

**Один репозиторий — один модуль (по умолчанию)**

Дефолт для микросервиса: **один модуль** (`go.mod` в корне), несколько пакетов внутри. Модуль в Go — это дерево пакетов с `go.mod` в корне, где определяются module path, зависимости и версия Go. citeturn4search4turn23search7turn23search3

**Базовая структура директорий (server project)**

Официальная документация Go по организации модулей рекомендует для серверных проектов:  
- хранить команды (бинарники) вместе в `cmd/`  
- держать пакеты логики сервера в `internal/`  
- если в репозитории появляются пакеты, полезные для внешнего переиспользования, лучше выносить их в отдельные модули. citeturn3view0turn2view0turn8view0

Практически применимый минимальный “template tree”:

```
.
├── cmd/
│   └── service/
│       └── main.go
├── internal/
│   ├── domain/          # доменная модель и инварианты
│   ├── app/             # use-cases / application services + порты (интерфейсы)
│   ├── transport/       # входные адаптеры (HTTP/gRPC)
│   ├── infra/           # выходные адаптеры (DB, очереди, внешние HTTP)
│   └── platform/        # инфраструктура процесса: config, logging, telemetry, lifecycle
├── docs/
├── api/                 # опционально: OpenAPI/proto/contract (не Go-код)
├── go.mod
└── go.sum
```

Ключевой смысл `internal/`: это **поддержанный инструментом Go механизм скрытия пакетов** от внешних импортов. Код “внутри или ниже директории `internal`” импортируется только кодом, находящимся в дереве директорий, корень которого — родитель `internal`. citeturn2view0turn8view0turn3view0

**Форматирование и импорт-менеджмент (обязательное)**

- `gofmt` — норма экосистемы; Go ожидает, что вы не спорите о стиле, а форматируете код автоматически. citeturn10view0turn24view0  
- `goimports` — надстройка над `gofmt`: умеет чинить импорты и форматировать код в стиле `gofmt`. Это особенно важно для LLM-генерации, где импорты — частый источник “мелких” ошибок. citeturn23search2turn24view0

**Безопасность зависимостей как дефолт пайплайна**

Встроить в template CI-проверку `govulncheck`: это официальный low-noise инструмент Go для поиска уязвимостей, который использует Go vulnerability database и старается показать только реально достижимые уязвимые вызовы (через анализ кода/символов). citeturn6search1turn6search5turn6search7turn6search4

**Observability и эксплуатационные эндпоинты**

- Для graceful shutdown и таймаутов сервера опираться на `net/http` и его документированное поведение для `Server.Shutdown(...)` и полей таймаутов. Важно, что `Shutdown` закрывает listeners, закрывает idle connections и ждёт, пока активные соединения станут idle; если контекст истечёт — вернёт ошибку контекста. citeturn12view1turn12view0  
- Для логирования по умолчанию использовать `log/slog` (структурные key-value логи — стандартный подход Go с 1.21; есть официальный разбор). citeturn11search3turn11search6  
- Для контейнеров в Kubernetes: предусмотреть readiness/liveness/startup probes (как минимум — endpoints) и задокументировать семантику. Kubernetes явно различает liveness/readiness/startup и описывает их поведение и назначение. citeturn13search4turn13search0turn13search23  
- Для телеметрии: если выбираете OpenTelemetry, придерживаться semantic conventions, чтобы метрики/трейсы были сопоставимы между сервисами. citeturn13search5turn13search13turn13search1  
- Если выбираете Prometheus-метрики, придерживаться их naming-практик (и осознанно отметить конфликт/схождения с OTel-неймингом: Prometheus часто требует суффиксы единиц/типов в имени метрики, а OTel-подход может отличаться). citeturn13search10turn13search2

## Decision matrix / trade-offs

Ниже — точки, где разумные команды принимают разные решения. Для каждой даны “boring defaults” и компромиссы.

**Один модуль vs несколько модулей в одном репозитории**

- Дефолт: **один модуль на микросервис**. Это проще для разработки, CI и для LLM (меньше вероятности “сломать” импорт-пути, `replace`, `go.work`). citeturn4search4turn3view0  
- Когда переходить к нескольким модулям: когда вам действительно нужно публиковать и версионировать **внешне переиспользуемые пакеты**, или у вас монорепо с несколькими сервисами/библиотеками. Go поддерживает multi-module workspaces (go.work) как инструмент совместной разработки нескольких модулей без постоянных правок `go.mod`. citeturn4search9turn4search1  
- Риск multi-module: усложнение dependency resolution и ошибок импорт-путей (включая major version suffix в путях для v2+ согласно semantic import versioning). citeturn4search13turn4search2turn4search6

**`internal/` vs “полупубличные” пакеты (`pkg/`)**

- Дефолт: **почти всё в `internal/`** для server repository — это прямо рекомендуемый путь в официальном гайде по структуре “server project”. citeturn3view0turn2view0  
- Если часть кода нужна другим репозиториям: предпочтительнее **вынести в отдельный модуль**, а не делать “случайный public API” внутри сервиса. Официальный гайд прямо советует: если в серверном репозитории появляются пакеты “для шаринга”, их лучше выделять в отдельные модули. citeturn3view0turn4search4turn4search13  
- Trade-off `pkg/`: иногда удобно иметь явную “публичную” область. Но цена — вы неявно обещаете стабильность API и повышаете риск того, что LLM начнёт использовать “публичные” пакеты как dumping ground, если правила не зафиксированы жёстко. Это противоречит советам Go об опасности `api/types/interfaces`-пакетов и бессодержательных “общих” пакетов. citeturn9view0turn24view0

**Слойная архитектура (`domain/app/infra`) vs “пакеты по ответственности/фичам”**

- Дефолт для template: **слои как механизм контроля зависимостей**, но с требованием держать пакеты предметными и маленькими.  
  - Плюс: проще задать LLM правила импорта (“domain не импортирует infra”, “transport зависит от app”). citeturn22view2turn24view0  
  - Минус: риск превратить `domain/` или `app/` в “второй util”, если не enforce-ить правила именования и когезии. Go прямо предупреждает, что пакеты `util/common/misc` разрастаются и копят зависимости, ухудшая поддержку и скорость сборки. citeturn9view0turn17view3turn10view0  
- Альтернатива: vertical slices (например `internal/orders/...`). Это улучшает локальность, но усложняет единые правила импорта и часто приводит к дублированию “platform glue” по фичам, если не аккуратно. Этот вариант стоит рассматривать, если доменные контексты слабо связаны и команда сознательно выбирает “feature-first”. citeturn18view0turn24view0

**Интерфейсы: где объявлять и как мокать**

- Дефолт: **интерфейсы объявлять в пакете-потребителе**, а пакет-реализация возвращает concrete types. Это снижает связность: реализация может расширяться методами без каскадного рефакторинга интерфейса. В Go wiki по code review comments это сформулировано как общее правило. citeturn24view0  
- Не делать интерфейсы “на стороне implementor-а для мокинга”: вместо этого проектировать API так, чтобы тестировать через публичный API реальной реализации, либо чтобы потребитель определял узкий интерфейс под свои нужды. citeturn24view0

**Generics vs интерфейсы vs дублирование**

- Дефолт: **не делать код generic, пока не появились минимум 2–3 реальных потребителя с одинаковым паттерном**. Официальный пост “When To Use Generics” специально предупреждает: это рекомендации, а не жёсткие правила, и поощряет осторожность. citeturn19search0  
- Generics полезны для структур данных и алгоритмов, реально независимых от предметной области. Но для “domain-level” абстракций generics часто ухудшают читабельность и усложняют API, особенно для LLM-кода, который склонен к преждевременной абстракции. Поэтому в template generics должны иметь явные границы: либо `internal/platform/*`, либо узкий domain-neutral пакет. citeturn19search0turn19search1

## Repo-level правила для package/module design в Go

Ниже — правила уровня репозитория (traceable, enforce-able в review) для того, как раскладывать код и удерживать границы.

**Границы модуля**

1) В корне репозитория должен быть ровно один `go.mod` (дефолт). Любые дополнительные модули добавлять только через отдельное архитектурное решение, с явной мотивацией (публикация пакета и версионирование). citeturn4search4turn23search7turn3view0  
2) `go.mod` определяет module path и зависимости; команды `go get` и `go mod tidy` поддерживают консистентность зависимостей, а `go mod tidy` “добавляет недостающие и удаляет неиспользуемые модули” и синхронизирует `go.sum`. Это нужно делать в PR при изменении зависимостей. citeturn23search15turn23search11turn4search4

**Границы пакетов и приватность**

1) Весь “сервисный” Go-код (кроме entrypoints) живёт внутри `internal/`. Это даёт инструментальную гарантию приватности: внешний код не может импортировать `internal/*`. citeturn3view0turn2view0turn8view0  
2) Если нужно скрыть код не только от внешнего мира, но и от “частей внутри монорепо/модуля”, допускается использовать **вложенные `internal/`** для усиления границ (потому что правило `internal` привязано к “родителю `internal`” и дереву импорта). Но это advanced-feature: использовать только при реальной необходимости, иначе LLM будет путаться в путях. citeturn2view0turn8view0

**Обязательный direction-of-dependencies (слои)**

Дефолтная модель импорта:

- `cmd/service` импортирует только `internal/...` (и stdlib/выбранные внешние зависимости).  
- `internal/transport` зависит от `internal/app` (и stdlib/внешние transport libs), но не от `internal/infra`.  
- `internal/app` зависит от `internal/domain` и объявляет интерфейсы-порты для внешних ресурсов.  
- `internal/infra` зависит от `internal/app` и `internal/domain` (реализует порты), но `domain` никогда не зависит от `infra`.  
- `internal/platform` может импортироваться “сверху” (обычно `cmd` и `transport`), но сам `platform` должен оставаться small & cohesive, чтобы не стать новой “свалкой”.  

Главная цель — исключить циклы и “втекание” инфраструктуры в домен. В Go импорт-петли запрещены: спецификация говорит, что пакету “незаконно импортировать самого себя, прямо или косвенно”. citeturn22view2turn24view0turn3view0

**Именование пакетов и границы когезии**

1) Имена пакетов — короткие, lowercase, без underscores и без бессодержательных “общих” имён; это фиксировано в Effective Go и в Go blog. citeturn10view0turn9view0  
2) Пакеты `util`, `common`, `misc`, а также “склад интерфейсов” `api/types/interfaces` запрещены: Go blog отдельно объясняет, что такие пакеты разрастаются, копят зависимости и замедляют сборку, а также ухудшают навигацию и поддержку. citeturn9view0turn17view3  
3) Пакеты организуются **по функциональной ответственности**, а не по “типу сущностей”. Пример “`package models` — НЕ ДЕЛАТЬ” прямо приводится в практических рекомендациях по пакетам: типы должны жить ближе к месту использования (например, User рядом с сервисным API, который его использует). citeturn18view0  
4) Не экспортировать идентификаторы из `main`-пакетов без реальной причины: main-пакеты не импортируются, а значит экспорт там, как правило, бессмысленен и является признаком неправильной структуры. citeturn18view0turn22view2

**Где должны жить domain/application/infrastructure слои (конкретно)**

- `internal/domain/<bounded>`: сущности/VO/инварианты/валидации, domain errors, доменные интерфейсы только если они не про инфраструктуру. Никаких импортов DB/HTTP/логеров/метрик.  
- `internal/app/<bounded>`: use-cases, orchestration; здесь объявляются **порты** (интерфейсы) к репозиториям, брокерам, внешним сервисам; здесь фиксируются транзакционные границы и политика ретраев (если она не привязана к конкретному адаптеру). Правило Go по интерфейсам: принадлежность интерфейса — стороне использования. citeturn24view0  
- `internal/infra/<provider>` или `internal/infra/<bounded><adapter>`: реализации портов для конкретных провайдеров (`postgres`, `redis`, `kafka`, `httpclient` и т.п.).  
- `internal/transport/http` (или `grpc`): адаптеры входа; маппинг DTO/HTTP → app-команды; timeouts и контекст должны идти от границы запроса.  
- `internal/platform/*`: кросс-срез: конфиг, логирование, телеметрия, lifecycle; но не “свалка всего подряд”.

## Набор правил MUST / SHOULD / NEVER для LLM

Ниже — “LLM contract”, который можно почти напрямую положить в `docs/llm-instructions.md` или `.cursor/rules`, и использовать как префикс для генерации кода.

**MUST**

- Генерировать Go-код, который проходит `gofmt` (и предпочтительно `goimports`), не споря со стилем. citeturn10view0turn23search2turn24view0  
- Соблюдать структуру server project: entrypoints в `cmd/`, бизнес-логика в `internal/`. citeturn3view0turn2view0  
- Соблюдать правило `internal`: не импортировать `internal/*` из внешнего модуля и не предлагать “публичное использование” внутренних пакетов. citeturn2view0turn8view0  
- Соблюдать направление зависимостей слоёв: `domain` не импортирует `infra` и `transport`. Если нужна абстракция — объявлять интерфейс в потребителе (`app`), а реализацию — в `infra`. citeturn24view0turn22view2turn3view0  
- Избегать бессодержательных пакетов: не создавать `util`, `common`, `misc`, а также агрегирующие пакеты `api/types/interfaces`. Пакет должен иметь ясное назначение и когезию. citeturn9view0turn17view3turn18view0  
- Передавать `context.Context` **явно и первым аргументом** по всей цепочке вызовов от входного запроса к I/O операциям; не хранить `Context` в struct (кроме случаев совместимости с внешним интерфейсом). citeturn24view0turn11search14  
- Не использовать `os.Exit`/`log.Fatal*` вне `main()`: ошибки должны возвращаться наверх и решаться на уровне `cmd/.../main.go`. citeturn17view1  
- Не использовать `panic` для штатной обработки ошибок в production-коде. citeturn17view2turn24view0  
- При добавлении/изменении зависимостей обновлять `go.mod/go.sum` через `go mod tidy`, а не “ручными правками”. citeturn23search15turn23search11  
- Если генерируется `net/http` сервер: использовать документированные механизмы таймаутов и `Server.Shutdown(ctx)` для graceful shutdown; не оставлять поведение завершения “на удачу”. citeturn12view0turn12view1

**SHOULD**

- Использовать `log/slog` как стандартный structured logging дефолт (JSON в production), чтобы логи были пригодны для поиска/фильтрации. citeturn11search6turn11search3  
- Добавлять базовые эксплуатационные endpoints и документацию probe-логики под Kubernetes (readiness/liveness/startup), если сервис предполагается к запуску в оркестраторе. citeturn13search4turn13search0turn13search23  
- Для dependency security: включать `govulncheck` в CI и ориентироваться на практики Go security docs. citeturn6search7turn6search5turn6search1  
- Для API security: учитывать требования из entity["organization","OWASP","web app security org"] (например, OWASP API Security Top 10) и использовать их как чеклист рисков для сервисов, которые публикуют API. citeturn14search0turn14search8  
- Вводить generics только при доказанной необходимости; при сомнении — предпочитать простой неоджнерик-код. citeturn19search0  
- Писать примеры использования на уровне пакетов (godoc-friendly): это улучшает discoverability и снижает риск, что LLM “придумает” контракт пакета. citeturn18view0turn24view0

**NEVER**

- Никогда не создавать “dump” пакеты или файлы (например, `internal/utils`, `misc.go`, `helpers.go`) как место для всего подряд. Такие пакеты по определению теряют фокус и копят зависимости. citeturn9view0turn17view3turn18view0  
- Никогда не делать `domain` зависимым от конкретного провайдера (SQL драйвера, HTTP клиента, брокера, метрик/логера): это ломает тестируемость и архитектурную устойчивость. (Если нужен контракт — он в `app` как порт, реализация в `infra`.) citeturn24view0turn22view2  
- Никогда не объявлять интерфейсы “на стороне реализации” только ради моков; не проектировать ради тестов, проектировать ради API потребителя. citeturn24view0  
- Никогда не хранить `context.Context` в структуре “ради удобства” и не создавать новые root contexts внутри request path без причины (например, `context.Background()` внутри handler), потому что это ломает отмену/таймауты/трейсинг. citeturn24view0turn11search14  
- Никогда не предлагать multi-module как дефолт для одного микросервиса. Use `go.work` и несколько модулей — осознанное решение, а не генерация “по привычке”. citeturn4search9turn3view0

## Concrete good / bad examples

**Пример хорошего разбиения репозитория**

```
cmd/service/main.go                       # только wiring + lifecycle
internal/domain/account/...               # доменные типы и инварианты
internal/app/account/...                  # use-cases + порты (interfaces)
internal/infra/postgres/...               # реализации портов
internal/transport/http/...               # HTTP handlers + DTO mapping
internal/platform/logging/...             # slog setup
internal/platform/telemetry/...           # OTel init
```

Такой layout следует официальным рекомендациям для server projects (cmd + internal), использует “internal” как механизм приватности и делает явное направление зависимостей. citeturn3view0turn2view0turn8view0turn24view0

**Плохой пример структуры (типичные признаки будущего “болота”)**

```
pkg/
  common/
  utils/
internal/
  models/
  interfaces/
```

Почему плохо: `util/common/misc` и “склады интерфейсов/типов” ухудшают навигацию и поддержку, почти неизбежно разрастаются и копят зависимости. Go blog прямо перечисляет такие пакеты как антипример и поясняет последствия (включая рост зависимостей и замедление компиляции). citeturn9view0turn17view3turn18view0

### Пример: “интерфейсы в пакете-потребителе” (good)

```go
// internal/app/account/ports.go
package account

import "context"

type Repository interface {
	Get(ctx context.Context, id string) (*Account, error)
	Save(ctx context.Context, a *Account) error
}
```

```go
// internal/infra/postgres/account_repo.go
package postgres

import (
	"context"

	"example.com/service/internal/app/account"
)

type AccountRepo struct {
	// db *sql.DB или pgxpool.Pool — конкретика не важна для примера
}

var _ account.Repository = (*AccountRepo)(nil)

func (r *AccountRepo) Get(ctx context.Context, id string) (*account.Account, error) {
	// ...
	return nil, nil
}
```

Обоснование: Go wiki по code review comments рекомендует располагать интерфейсы в пакете, который их использует, а реализации возвращать concrete types; это делает API менее хрупким при эволюции реализации. citeturn24view0

### Пример: “не делай package models” (bad)

```go
// internal/models/user.go
package models // DON'T
type User struct{ /* ... */ }
```

Вместо этого типы размещаются рядом с логикой, которая ими оперирует, организуя код по ответственности. Это явно рекомендуется в практиках про package organization. citeturn18view0

### Пример: корректное использование server shutdown (good)

```go
srv := &http.Server{
	Addr:              cfg.HTTPAddr,
	Handler:           h,
	ReadHeaderTimeout: 5 * time.Second,
	IdleTimeout:       60 * time.Second,
}

go func() {
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}()

if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
	return fmt.Errorf("listen: %w", err)
}
```

`ReadHeaderTimeout`, `IdleTimeout` и поведение `Shutdown(ctx)` — документированная часть `net/http`. citeturn12view0turn12view1

## Anti-patterns, типичные LLM-ошибки, review checklist и что вынести в отдельные файлы

**Anti-patterns и типичные hallucinations LLM**

1) **“Свалочные пакеты”**: LLM часто создаёт `utils/common/helpers`, потому что это распространённый паттерн в других языках. В Go это напрямую считается плохим дизайном пакета (плохие имена, рост зависимостей, низкая когезия). citeturn9view0turn17view3turn18view0  
2) **Интерфейсы “для мокинга” на стороне implementor-а**: модель может создавать `internal/infra/interfaces.go` и т.п. Это противоречит общему правилу Go: интерфейсы должны жить у потребителя. citeturn24view0  
3) **Неправильное обращение с `context`**: `context.Background()` внутри handler, хранение `ctx` в struct “для удобства”, пропуск `cancel()` — типичная ошибка. Go code review comments дают прямые правила (ctx первым аргументом, не хранить в struct, не создавать кастомный Context). citeturn24view0turn11search14  
4) **`log.Fatal`/`os.Exit` в библиотечном коде**: LLM может “обрубать” выполнение на глубине стека. Это плохая практика: завершение процесса — ответственность `main`. Это прямо сформулировано в стиле entity["company","Uber","rideshare company"]. citeturn17view1turn17view2  
5) **`init()` как dependency injection**: модель может пытаться “автоматически регистрировать” зависимости через `init()` и side-effects. Это плохо для предсказуемости, тестируемости и порядка инициализации; style guide рекомендует избегать `init()` и не спавнить горутины в `init()`. citeturn17view0turn22view2  
6) **Случайное создание публичного API**: LLM может начать экспортировать типы из `main` или создавать “полу-SDK” внутри сервиса. Это бессмысленно (main не импортируется) и обычно означает неверную структуру. citeturn18view0turn22view2  
7) **Преждевременная абстракция и generics “ради красоты”**: модель может обобщать всё подряд. Официальные рекомендации по generics призывают к осторожности и подчёркивают, что это не “универсальный подход”, а инструмент по необходимости. citeturn19search0turn19search1

**Review checklist для PR / code review**

- Структура: entrypoints находятся в `cmd/`, логика — в `internal/`; не появилось ли “случайного” `pkg/` без цели публикации? citeturn3view0turn2view0  
- Имена пакетов: lowercase, кратко, предметно; нет ли новых пакетов/файлов вида `util/common/misc/models/types/interfaces`? citeturn10view0turn9view0turn18view0turn17view3  
- Границы: `domain` не импортирует `infra/transport`; порты объявлены в потребителе; не возник ли новый цикл импорта (в том числе “скрытый” через рефакторинг)? citeturn22view2turn24view0  
- Контекст: `context.Context` идёт первым аргументом в I/O; не хранится в struct; нет `context.Background()` в request path без явной причины. citeturn24view0turn11search14  
- Ошибки: нет `panic` для штатных ситуаций; нет `log.Fatal/os.Exit` вне `main`; ошибки возвращаются и оборачиваются контекстом. citeturn17view2turn17view1turn24view0  
- HTTP server: есть `ReadHeaderTimeout`/`IdleTimeout`/`Shutdown(ctx)` (или эквивалент с документированным смыслом); корректно обрабатывается `ErrServerClosed`. citeturn12view0turn12view1  
- Зависимости: при изменении импортов выполнен `go mod tidy`; `go.sum` обновлён; нет случайных “тяжёлых” зависимостей без ADR/обоснования. citeturn23search15turn23search11turn4search4  
- Безопасность: есть (или не ломается) `govulncheck` в CI; обновления Go учитывают security policy и minor releases. citeturn6search7turn6search5turn6search3  
- Observability: если используется OTel — semantic conventions соблюдены; если Prometheus — naming практики соблюдены и документированы. citeturn13search13turn13search1turn13search2  
- Документация пакетов: новые пакеты имеют краткую package doc и/или примеры, если API неочевиден. citeturn18view0turn24view0

**Что оформить отдельными файлами в template repo**

Рекомендуемый набор файлов, который превращает этот стандарт в реальный “production-ready template”:

- `docs/engineering-standard.md` — конвенции проекта: структура, зависимости, тесты, CI, релизы. (Основание: официальные рекомендации по структуре модулей и общие code review comments.) citeturn3view0turn24view0  
- `docs/package-design.md` — правила package/module design (material из разделов про `internal`, naming, interfaces-in-consumer, запрет `util/common`). citeturn2view0turn9view0turn24view0turn18view0  
- `docs/llm-instructions.md` — MUST/SHOULD/NEVER контракт для LLM (как в этом документе). citeturn24view0turn10view0turn9view0  
- `docs/security.md` — политика обновления Go и зависимостей, `govulncheck`, ориентиры entity["organization","OWASP","web app security org"] (API Top 10, ASVS/cheat sheets по необходимости). citeturn6search7turn6search3turn6search5turn14search0turn14search1  
- `docs/observability.md` — logging (slog), health endpoints, probes, метрики/трейсы (OTel/Prometheus) и их naming/semconv. citeturn11search6turn13search4turn13search5turn13search2turn13search13  
- `.golangci.yml` (если выбираете golangci-lint) и/или строго зафиксированные `go vet`, `go test`, `govulncheck` шаги в CI. (Если linter спорный — зафиксировать линтерный набор и причины.) citeturn23search0turn23search4turn6search5turn24view0  
- `Makefile` или `Taskfile.yml` — детерминированные команды: `fmt`, `lint`, `test`, `vuln`, `build`, `run`. Основание: необходимость стандартизировать `gofmt/goimports`, `go mod tidy` и security checks. citeturn10view0turn23search2turn23search15turn6search7  
- `.editorconfig` + рекомендации IDE (как минимум: gofmt/goimports on save), чтобы LLM-генерируемый код не “рассыпался” от разных форматтеров. citeturn10view0turn23search2turn24view0

**Отдельно про источники “стандарта”**

- Нормативная часть про структуру модулей, `cmd/`/`internal/` и server projects — из официальной документации Go. citeturn3view0turn2view0turn8view0  
- Запрет `util/common/api/types` как “плохих пакетов” — из официального Go blog + подтверждён в индустриальных style guides. citeturn9view0turn17view3  
- Правила по `context` и “интерфейсы живут у потребителя” — из Go Code Review Comments. citeturn24view0  
- Остальная “production hygiene” (security scanning, release policy) — из официальных материалов Go security. citeturn6search7turn6search3turn6search5