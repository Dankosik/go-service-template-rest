# Operational readiness и production diagnostics для production-ready Go микросервиса

## Scope

Этот стандарт применим, когда микросервис на Go запускается как долговременный процесс (обычно в контейнере) и должен быть управляем оркестратором через health‑checks, перезапуски, снятие/возврат в балансировку и управляемое завершение. На практике это чаще всего означает деплой в entity["organization","Kubernetes","container orchestration"], где kubelet выполняет container probes, а состояние readiness влияет на включение Pod в балансировку сервисов. citeturn4view4turn17view1turn9view0

Подход особенно полезен для сервисов:
- HTTP (REST) и/или gRPC, к которым применяется routing через Service/Ingress (readiness прямо влияет на то, будет ли Pod получать трафик через Services). citeturn27view2turn4view4turn17view1
- С внешними зависимостями (DB/queue/cache), когда важны корректные semantics для “готов принимать трафик” vs “процесс жив и не требует рестарта”. citeturn3view0turn17view1turn7view4
- С требованием к безопасной диагностике в production (pprof/expvar), не превращающейся в “случайно открыли debug наружу”. citeturn8view1turn3view7turn3view5

Стандарт **не** подходит или требует адаптации, если:
- Это batch/job, который должен завершаться сам по себе (readiness/liveness обычно не дают ценности, а нюансы shutdown и probes могут мешать). Оркестратор при этом всё равно может использовать restartPolicy и pod‑termination semantics, но “микросервисный” контроль трафика может быть нерелевантен. citeturn17view1turn2search2
- Процесс работает вне модели “оркестратор шлёт TERM и ждёт grace period” (например, специфичный init‑system/embedded). Для Kubernetes‑подобной модели важно понимать, что при удалении Pod kubelet сначала пытается остановить контейнер (TERM), а затем после grace period — принудительно (KILL). citeturn17view1turn2search2
- Требуется строгая аутентификация на health endpoint’ах: built‑in probes не поддерживают параметры аутентификации (для gRPC прямо отмечены отсутствие auth параметров и общий принцип “ошибка = probe failure”). Если вам нужен auth, придётся менять механизм (exec probe, sidecar, mTLS на уровне сетевого периметра и т.п.). citeturn27view0turn17view1

## Recommended defaults для greenfield template

Ниже — “обязательный минимум из коробки” для production‑готового шаблона. Он максимально “boring”, повторяет семантику и практики, которые уже закреплены в Kubernetes и стандартной библиотеке Go.

### Семантика health endpoint’ов и probes

**Имена и разделение endpoint’ов (default):**
- `GET /livez` — “should I be restarted?”. В Kubernetes ливнес предназначен именно для решения “когда рестартить контейнер”, и пример в документации API server трактует `/livez` как индикатор non‑recoverable состояния (например, deadlock) и необходимости рестарта. citeturn4view4turn7view3turn7view4  
- `GET /readyz` — “можно ли направлять трафик?”. Readiness определяет готовность принимать трафик; при fail readiness Kubernetes удаляет Pod из service endpoints. citeturn4view4turn7view4turn17view1  
- `GET /startupz` — “запустилось ли приложение?”. Startup probe предназначен для slow start и **отключает liveness/readiness до успеха**, чтобы контейнер не был убит преждевременно. citeturn4view4turn17view1  

**Формат ответов и “machine vs human”:**
- Машины должны опираться на **HTTP status code**, как это делает Kubernetes для своих health endpoint’ов (200 = ok). citeturn7view2turn7view4  
- Детальную диагностику делайте “для человека” через `?verbose=1` (или альтернативный endpoint), как рекомендуют для Kubernetes API health endpoint’ов: verbose предназначен для операторов и *не* должен быть машинным контрактом. citeturn7view1turn7view4  

**Dependency health reporting (рекомендуемый паттерн):**
- `GET /readyz` по умолчанию возвращает бинарный результат (200/503).  
- `GET /readyz?verbose=1` показывает список компонентных checks и их статусы (db/cache/queue и т.п.) по аналогии с verbose‑форматом Kubernetes API server; это ускоряет triage, не ломая машинный контракт. citeturn7view1turn7view4  
- Сервис должен поддерживать “readiness отличается от liveness”, включая кейс maintenance: Kubernetes прямо рекомендует readiness‑endpoint, отличный от liveness, если приложение хочет “временно снять себя с трафика”. citeturn17view1  

**Где уместно проверять зависимости:**
- Kubernetes допускает модель: liveness = “само приложение здорово”, readiness = “плюс проверка необходимых backend‑сервисов”, чтобы не направлять трафик на Pod, который ответит только ошибками. citeturn17view1turn3view0  
- При этом readiness в Kubernetes используется и для recovery/overload во время жизни контейнера. Значит, реализация должна быть быстрой, не блокироваться надолго и быть устойчива к частым вызовам. citeturn3view0turn27view2

**Параметры probes в манифестах (boring defaults):**
- Используйте Kubernetes‑дефолты как стартовую точку: `initialDelaySeconds=0`, `periodSeconds=10`, `timeoutSeconds=1`, `successThreshold=1`, `failureThreshold=3`. citeturn27view2turn27view3  
- Учитывайте, что пока контейнер not Ready, `ReadinessProbe` может выполняться чаще, чем `periodSeconds`, чтобы быстрее перевести Pod в Ready. Следствие: readiness handler **обязан** быть дешёвым, без вредных side effects и без утечек ресурсов. citeturn27view2turn27view3  
- Не пытайтесь “прятать” probes за HTTP auth: built‑in probes не поддерживают auth параметры. citeturn27view0turn27view3  

### Startup и защита от crash‑loop на медленном старте

Шаблон должен содержать `startupz` и пример настройки `startupProbe`:
- Startup probe отключает liveness/readiness до успеха. citeturn4view4turn17view1  
- Kubernetes рекомендует выбирать startup probe так, чтобы покрыть worst‑case startup через `failureThreshold * periodSeconds`, не “раздувая” liveness. В документации прямо дан критерий: если контейнер обычно стартует дольше `initialDelaySeconds + failureThreshold * periodSeconds`, следует добавить startup probe и поднять `failureThreshold`. citeturn17view0turn27view2  

Практический default для шаблона: `startupProbe` проверяет тот же endpoint, что и liveness (`/livez` или отдельный `/startupz`), но имеет больший бюджет по времени через `failureThreshold`. citeturn17view0turn27view2

### Graceful shutdown, draining и lifecycle

**Что Kubernetes делает при остановке Pod (важно для контракта сервиса):**
- Pod получает возможность завершиться “gracefully”, по умолчанию 30 секунд. citeturn2search2turn27view3  
- При termination kubelet обычно сначала отправляет TERM (SIGTERM) в главный процесс контейнера, ждёт grace period, затем отправляет KILL оставшимся процессам. citeturn17view1turn2search2  
- `PreStop` hook выполняется **синхронно** и должен завершиться до отправки TERM; при этом общий `terminationGracePeriodSeconds` включает и время `PreStop`, и время нормальной остановки процесса. citeturn3view3turn2search20  

**Что сервис обязан поддерживать “из коробки”:**
- На SIGTERM/SIGINT сервис должен:
  1) Перевести readiness в fail (начать draining) — так Kubernetes удаляет Pod из endpoints при fail readiness, и Pod перестаёт получать трафик через Services. citeturn4view4turn27view2turn9view0  
  2) Перестать принимать новые соединения и дождаться завершения in‑flight запросов в пределах shutdown timeout. Для net/http правильный механизм — `(*http.Server).Shutdown(ctx)`, который закрывает listeners, закрывает idle connections и ждёт активные до deadline контекста. citeturn3view4  
  3) Отдельно обработать hijacked/long‑lived connections (например, WebSocket): `Shutdown` **не** закрывает и не ждёт такие соединения, это обязанность приложения. citeturn3view4  

**Технические defaults для шаблона (Go/net/http):**
- Использовать `signal.NotifyContext` для отмены контекста по сигналам. citeturn8view6  
- Использовать `http.Server.Shutdown` с конечным timeout (например, 25–28 секунд при дефолтном `terminationGracePeriodSeconds=30`, оставляя запас на preStop/системные задержки). Сам факт, что shutdown должен ждать и не должен “уронить процесс раньше”, прямо отмечен в документации `Shutdown`: после вызова `Shutdown` методы Serve/ListenAndServe возвращают `ErrServerClosed`, и программа должна дождаться `Shutdown`. citeturn3view4turn2search2  
- На этапе shutdown можно отключать keep‑alive через `SetKeepAlivesEnabled(false)` (документация прямо указывает, что отключение keep‑alive уместно для “servers in the process of shutting down”). citeturn3view4  

### Admin/debug endpoints и безопасная диагностика

**Цели:**
- дать SRE/дежурному возможность быстро собрать pprof/heap/goroutine dump и минимальные runtime stats;
- не допустить “случайно открыли debug наружу” (это типичный класс security misconfiguration). citeturn3view7turn8view1turn13view0  

**pprof (обязательный инструмент диагностики производительности):**
- Go официально рассматривает pprof‑endpoint’ы как канал сбора profiling data, наряду с `go test`. citeturn13view0  
- `net/http/pprof` обычно импортируют ради side effect: handlers регистрируются, пути начинаются с `/debug/pprof/` (и, начиная с Go 1.22, требуют GET). citeturn3view5  
- Важный security‑факт: в Go ecosystem признан риск того, что `net/http/pprof` регистрирует handlers на default mux, что облегчает случайную установку потенциально небезопасных endpoint’ов и утечки данных; большие проекты сталкивались с этим и вынужденно выносили pprof на “альтернативный” канал. citeturn8view1turn0search7  

**expvar (опционально, но полезно как “минимальные runtime stats”):**
- `expvar` публикует `/debug/vars` (JSON), включает `cmdline` и `memstats` (и требует GET с Go 1.22). Это удобно для диагностики, но потенциально чувствительно (например, cmdline может раскрывать аргументы запуска). citeturn8view0  

**Безопасные defaults для шаблона:**
- Debug endpoints (`/debug/pprof/*`, `/debug/vars`) должны быть:
  - либо на отдельном admin‑listener’е, который **не** экспонируется через Service/Ingress по умолчанию (доступ только через port‑forward/внутренние каналы),
  - либо защищены сетевым периметром (NetworkPolicy, internal LB), но **не** “надеяться на удачу”.  
  Мотивация: OWASP относит включённые “лишние страницы/сервисы/порты” и выдачу stack traces/слишком подробных ошибок к классу security misconfiguration. citeturn3view7turn8view1  

### Crash diagnostics и “что делать, когда всё упало”

Шаблон должен поддерживать предсказуемую диагностику падений:
- Переменная окружения `GOTRACEBACK` управляет объёмом вывода при unrecovered panic или runtime fatal error; по умолчанию печатается stack trace текущей goroutine (и процесс завершается с exit code 2). citeturn3view6  
- `runtime/debug.SetTraceback` позволяет увеличить детализацию (например, `all`, чтобы печатать все goroutines), не уменьшая ниже уровня, заданного через `GOTRACEBACK`. citeturn14view0turn3view6  
- `runtime/debug.PrintStack` печатает stack trace в stderr (полезно для controlled dumps в обработчиках/паник‑recover). citeturn14view1  
- Начиная с Go 1.23 есть `runtime/debug.SetCrashOutput`, который позволяет дублировать вывод fatal errors в дополнительный файл (помогает организовать “crash capture” в специфичных окружениях). citeturn14view1  

### Runbooks и incident triage (минимум документации)

Шаблон должен включать “скелет” operational документации:
- Runbook/playbook должен описывать шаги на стадиях incident response (communication, triage, investigation, resolution), перечислять инструменты/ресурсы и контакты, и быть регулярно пересматриваемым. citeturn3view8  
- Наличие формализованного процесса incident response и подготовки снижает impact и ускоряет восстановление; Google SRE подчеркивает важность процесса и того, что он должен быть известен и отрепетирован, а процедуры rollback — протестированы. citeturn8view2turn8view3  

image_group{"layout":"carousel","aspect_ratio":"16:9","query":["Kubernetes liveness readiness startup probes explanation diagram","Kubernetes pod termination lifecycle SIGTERM grace period diagram","Go net/http pprof /debug/pprof example","Go expvar /debug/vars output example"],"num_per_query":1}

## Decision matrix / trade-offs

Ниже — ключевые развилки для template’а. Везде даны “boring defaults” и условия, когда их менять.

| Решение | Вариант A (default) | Вариант B | Когда выбирать B | Основные риски/компромиссы |
|---|---|---|---|---|
| Readiness проверяет зависимости? | `/readyz` включает **критические** зависимости с короткими таймаутами/кэшем (а `/readyz?verbose` показывает детали) citeturn17view1turn7view1 | `/readyz` проверяет только “внутреннюю готовность”, а зависимости — отдельно (`/deps`) | Когда сервис должен деградировать и продолжать принимать часть трафика при частичных отказах зависимостей | A может flapping‑ить readiness при нестабильных зависимостях и вызывать “самоотключение от трафика”; B может направлять трафик на Pod, который будет отвечать ошибками citeturn17view1turn3view0 |
| Startup: `startupProbe` vs “длинный initialDelay” | Использовать `startupProbe`, который отключает liveness/readiness до успеха citeturn4view4turn17view0turn27view2 | Увеличивать `initialDelaySeconds` у liveness | Когда нет доступа к startup endpoint’у или инициализация принципиально неотличима от normal readiness | Длинный initialDelay ухудшает реакцию на deadlock, ради которой liveness и нужен citeturn27view2turn3view0 |
| Механизм probe: httpGet / grpc / exec | `httpGet` для HTTP‑сервисов, `grpc` для gRPC | `exec` | Когда нельзя открыть порт/нет endpoint’а или нужно проверить локальное состояние без сети | `exec` дороже: форк процессов каждый раз; на плотных кластерах и малых интервалах даёт overhead узлу citeturn9view0turn10search4 |
| Health endpoint’ы и аутентификация | Probes endpoint’ы **без auth**, но минимальные по данным | Auth на endpoint’ах | Только если используете другой механизм (не built-in probe) | Built-in probes не поддерживают auth параметры; любое требование auth ломает стандартный путь citeturn27view0turn7view2 |
| pprof/expvar: где и как экспонировать | Отдельный admin port / отдельный mux, закрытый от внешнего трафика | На основном порту | Почти никогда (только если сервис гарантированно internal и есть строгий периметр) | `net/http/pprof` легко случайно установить на default mux, что признано security risk; OWASP относит “лишние страницы/порты” и stack traces к misconfiguration citeturn8view1turn3view7turn3view5 |
| PreStop hook для draining | По умолчанию — **без** preStop sleep; draining делается readiness‑fail + graceful shutdown | Добавить preStop (например, sleep/drain trigger) | Когда есть доказанная проблема “трафик продолжает приходить после SIGTERM” из‑за LB/mesh особенностей | PreStop выполняется синхронно до TERM и “съедает” `terminationGracePeriodSeconds`; зависание PreStop держит Pod в Terminating до принудительного убийства citeturn3view3turn27view3turn2search2 |

## Набор правил в формате MUST / SHOULD / NEVER для LLM

Эти правила предназначены как “LLM‑instruction docs” для генерации кода и изменений в template, чтобы избежать догадок и частых production‑ошибок.

### Health semantics и probes

- MUST реализовать **раздельные** endpoint’ы `/livez`, `/readyz`, `/startupz` с семантикой liveness/readiness/startup. citeturn4view4turn17view1turn7view4  
- MUST возвращать корректный HTTP status code (200 для success; non‑200 для failure), потому что Kubernetes и другие проверяющие системы опираются на status code. citeturn7view2turn17view1  
- MUST держать `/livez` максимально “тупым и дешёвым”: он должен сигнализировать “нужен ли рестарт”, а не “доступна ли БД”. citeturn7view3turn17view1turn3view0  
- SHOULD делать `/readyz` бинарным, а детализацию зависимостей выдавать только в `?verbose=1` (или отдельном endpoint’е), предназначенном для людей. citeturn7view1turn7view4turn17view1  
- MUST предполагать, что readiness probe может вызываться чаще, чем `periodSeconds`, пока контейнер not Ready; endpoints должны быть idempotent, быстры и без side effects. citeturn27view2turn27view3  
- NEVER требовать HTTP auth/токены/клиентские сертификаты для built‑in readiness/liveness/startup probes: built‑in probes не поддерживают auth параметры. citeturn27view0turn27view3  

### Startup и time budgets

- MUST поддерживать startup semantics: если настроен startup probe, liveness/readiness должны быть отключены до его успеха (или эквивалентное поведение), чтобы избежать ложных рестартов на медленном старте. citeturn4view4turn17view1  
- SHOULD выбирать параметры startup probe через `failureThreshold * periodSeconds` ≥ worst‑case startup time (Kubernetes прямо рекомендует этот подход, без “раздувания” liveness). citeturn17view0turn27view2  

### Shutdown lifecycle и draining

- MUST обрабатывать SIGTERM/SIGINT через `signal.NotifyContext` (или эквивалент), чтобы запускать детерминированный shutdown. citeturn8view6  
- MUST на начале shutdown переводить сервис в not ready (чтобы Pod убирался из service endpoints и не получал новый трафик). citeturn4view4turn27view2turn9view0  
- MUST вызывать `(*http.Server).Shutdown(ctx)` и **дождаться** его завершения; документация прямо предупреждает, что после `Shutdown` серверные методы возвращают `ErrServerClosed`, а программа должна не завершаться раньше `Shutdown`. citeturn3view4  
- MUST отдельно закрывать/дожидаться hijacked/long‑lived connections (WebSocket и т.п.), так как `Shutdown` этого не делает. citeturn3view4  
- SHOULD учитывать `terminationGracePeriodSeconds` и `PreStop` как общий бюджет времени: PreStop выполняется до TERM и входит в grace period. citeturn3view3turn27view3turn2search2  
- NEVER делать `PreStop` hook “длинным и безусловным” без строгого обоснования: зависание PreStop держит Pod в Terminating до принудительного убийства. citeturn3view3turn2search20  

### Admin/debug endpoints

- MUST предусмотреть pprof как production diagnostic инструмент (включаемый безопасно), т.к. Go официально поддерживает сбор profiling данных через net/http/pprof endpoints. citeturn13view0turn3view5  
- MUST изолировать pprof от публичного трафика (отдельный port/mux/периметр) и документировать безопасный доступ (port-forward, internal‑only). citeturn8view1turn3view7  
- NEVER “слепо” добавлять `import _ "net/http/pprof"` в публичный HTTP‑server на default mux: это признанный security risk (утечки через профили/trace/stack), и Go community обсуждает изменения API именно из‑за этого. citeturn8view1turn0search7turn3view5  
- SHOULD добавлять `expvar` только если вы готовы защищать `/debug/vars` и осознаёте, что там есть `cmdline` и `memstats`. citeturn8view0turn3view7  

### Crash diagnostics и runbooks

- MUST документировать ожидаемое поведение при panic/fatal error (куда пишется stack trace, какие env vars влияют) и как это собирать в production. citeturn3view6turn14view0turn14view1  
- SHOULD сделать настройку `GOTRACEBACK` частью “операционного профиля” сервиса (через env), чтобы при краше получать полезный объём данных. citeturn3view6  
- MUST иметь runbook/playbook‑скелет с triage/investigation/resolution и поддерживать его актуальность. citeturn3view8turn8view2  

## Concrete good / bad examples

Ниже — примеры, которые можно почти напрямую переносить в template (упрощены, но сохраняют критичные свойства).

### Good: раздельные health endpoint’ы, draining и безопасный debug‑порт

```go
package main

import (
	"context"
	"errors"
	"expvar"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type HealthState struct {
	started      atomic.Bool
	ready        atomic.Bool
	shuttingDown atomic.Bool
}

func (h *HealthState) Livez(w http.ResponseWriter, _ *http.Request) {
	// Liveness: только "процесс жив и не в фатальном состоянии".
	if h.shuttingDown.Load() {
		// Решение спорное: иногда livez в shutdown оставляют 200, чтобы не рестартить.
		// Выбирайте по политике. По дефолту показываем, что процесс не для рестарта,
		// а для планового завершения.
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *HealthState) Readyz(w http.ResponseWriter, r *http.Request) {
	// Readiness: не принимать новый трафик в shutdown.
	if h.shuttingDown.Load() || !h.ready.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not ready"))
		return
	}
	// Вариант: если r.URL.Query().Has("verbose") — показать детали зависимостей.
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}

func (h *HealthState) Startupz(w http.ResponseWriter, _ *http.Request) {
	if !h.started.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("starting"))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("started"))
}

func main() {
	// 1) Контекст, отменяемый по SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	health := &HealthState{}

	// 2) Основной mux (НЕ DefaultServeMux).
	appMux := http.NewServeMux()
	appMux.HandleFunc("/hello", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	})

	// 3) Health endpoints в доступном для probes месте (обычно тот же порт, что и traffic).
	// Здесь для примера — на том же appMux.
	appMux.HandleFunc("/livez", health.Livez)
	appMux.HandleFunc("/readyz", health.Readyz)
	appMux.HandleFunc("/startupz", health.Startupz)

	appSrv := &http.Server{
		Addr:    ":8080",
		Handler: appMux,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	// 4) Admin/debug server на отдельном порту.
	adminMux := http.NewServeMux()
	// expvar публикует переменные через HTTP; добавим кастомную.
	expvar.NewString("service").Set("example")

	adminMux.HandleFunc("/debug/pprof/", pprof.Index)
	adminMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	adminMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	adminMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	adminMux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// NB: expvar по умолчанию вешается на DefaultServeMux, но мы можем просто проксировать:
	adminMux.Handle("/debug/vars", http.DefaultServeMux)

	adminSrv := &http.Server{
		Addr:    "127.0.0.1:9090", // безопасный default; при необходимости меняется конфигом
		Handler: adminMux,
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// 5) Старт серверов.
	go func() {
		defer wg.Done()
		if err := appSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("app server error: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := adminSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("admin server error: %v", err)
		}
	}()

	// 6) Инициализация (условно).
	health.started.Store(true)
	health.ready.Store(true)

	// 7) Ждём сигнал.
	<-ctx.Done()

	// 8) Draining: сначала снимаем readiness, затем выключаем keep-alive и делаем graceful shutdown.
	health.shuttingDown.Store(true)
	health.ready.Store(false)

	appSrv.SetKeepAlivesEnabled(false)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 28*time.Second)
	defer cancel()

	_ = adminSrv.Shutdown(shutdownCtx) // опционально: можно оставить admin доступным дольше.
	_ = appSrv.Shutdown(shutdownCtx)

	wg.Wait()
}
```

Почему это “good” по стандарту:
- Разведены semantics `/livez`/`/readyz`/`/startupz` и есть явный draining. citeturn4view4turn17view1turn9view0  
- Используется `signal.NotifyContext`. citeturn8view6  
- Используется `Server.Shutdown`, с учётом того, что нужно дождаться результата и что hijacked‑connections требуют отдельной обработки (в примере отмечено). citeturn3view4  
- pprof вынесен на отдельный admin‑mux/порт с безопасным default и без “слепого” default mux на публичном listener’е. citeturn8view1turn3view5turn3view7  

### Bad: один `/health`, зависит от БД, pprof на публичном порту, shutdown “в никуда”

```go
package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// BAD: один endpoint для liveness+readiness, да ещё и с внешними зависимостями.
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// imagine: ping DB, ping Redis, ping Kafka...
		// Любой кратковременный сбой зависимостей -> liveness fail -> рестарты.
		w.WriteHeader(http.StatusOK)
	})

	// BAD: pprof зарегистрирован на DefaultServeMux и доступен на том же порту что и API.
	go func() { log.Fatal(http.ListenAndServe(":8080", nil)) }()

	// BAD: ловим SIGTERM, но не делаем readiness fail и не делаем Shutdown().
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch

	// BAD: просто выходим -> обрываем соединения, не дожидаемся in-flight.
}
```

Почему это “bad”:
- Смешаны readiness и liveness; при fail readiness Kubernetes убирает Pod из endpoints, а liveness предназначен для решения “рестартить ли контейнер”. Смешение ведёт к рестартам при временной неготовности. citeturn4view4turn17view1turn7view3  
- `net/http/pprof` на default mux признан security risk (легко случайно открыть потенциально небезопасные endpoint’ы наружу). citeturn8view1turn3view7turn3view5  
- Нет `Server.Shutdown`; документация подчёркивает, что `Shutdown` нужен для корректного graceful shutdown и ожидания завершения активных соединений. citeturn3view4turn17view1  

## Anti-patterns и типичные ошибки/hallucinations LLM

Следующие ошибки встречаются часто именно в LLM‑генерируемом коде/конфигах и делают сервис “неоперабельным”.

Смешение liveness и readiness в один endpoint — особенно если внутри “пингуются зависимости”. Kubernetes допускает readiness‑проверку backend‑сервисов, но liveness должен отвечать на вопрос “нужен ли рестарт контейнера”. Если направить в оба probe один и тот же “dependency health”, временный outage зависимостей может привести к рестартам вместо снятия с трафика. citeturn17view1turn4view4turn7view3  

Игнорирование startup semantics: LLM часто “лечит” slow start увеличением `initialDelaySeconds` у liveness, вместо startup probe. Kubernetes прямо описывает startup probe как средство избежать убийства slow‑starting контейнеров и отключить liveness/readiness до успеха. citeturn4view4turn17view0turn27view2  

“Verbose как контракт”: LLM может предложить отдавать JSON со множеством полей и требовать, чтобы мониторинг его парсил. В Kubernetes verbose‑опции для health endpoint’ов предназначены для людей и не должны быть машинным контрактом; машины должны полагаться на HTTP status code. citeturn7view1turn7view2turn7view4  

Readiness handler с side effects (создание процессов, файлы, миграции): Kubernetes прямо предупреждает, что неправильная реализация readiness может приводить к росту числа процессов и starvation ресурсов; плюс readiness может выполняться чаще, чем `periodSeconds`. citeturn27view2turn27view3  

Использование `exec` probe для “сложных проверок” без понимания цены: exec‑probe подразумевает создание/форк процессов при каждом выполнении и может давать CPU overhead на ноде при высокой плотности Pod и малых интервалах. citeturn9view0turn10search4  

Публикация pprof/expvar на публичном порту “для удобства”: `net/http/pprof` легко случайно воткнуть в default mux; в Go‑репозитории это прямо признаётся security risk с утечками через профили/trace/stack. В OWASP A05 отдельно выделяются “ненужные страницы/сервисы/порты” и выдача stack traces как типичный misconfiguration. citeturn8view1turn3view7turn3view5turn8view0  

Неполный shutdown: LLM‑код часто ловит сигнал и завершает процесс, не вызывая `Server.Shutdown` и не дожидаясь его. Документация `Shutdown` подчёркивает необходимость ожидания завершения, а также то, что hijacked connections не закрываются автоматически. citeturn3view4turn8view6  

Непонимание `PreStop`: LLM может предложить “preStop sleep 30s”, не учитывая, что `PreStop` выполняется **до** TERM и входит в общий grace period; зависание `PreStop` удерживает Pod в Terminating до принудительного убийства по `terminationGracePeriodSeconds`. citeturn3view3turn2search20turn27view3  

## Review checklist для PR/code review и что оформить отдельными файлами в template repo

### PR / code review checklist

**Health endpoints и probes**
- [ ] Есть `/livez`, `/readyz`, `/startupz` (или эквивалентная разделённая семантика), и они возвращают корректные HTTP status code (200 ok; non‑200 fail). citeturn4view4turn7view2turn17view1  
- [ ] `/readyz` при fail действительно снимает Pod с трафика (readiness влияет на endpoints/Service routing). citeturn4view4turn27view2turn17view1  
- [ ] Реализация readiness быстрая и без side effects; учитывает, что readiness может вызываться чаще `periodSeconds`, пока контейнер not Ready. citeturn27view2turn27view3  
- [ ] Если readiness включает dependency checks, они bounded по времени (timeouts), а детали доступны только для человека (например, `?verbose=1`). citeturn17view1turn7view1turn7view4  
- [ ] Probes не требуют аутентификации на уровне HTTP/gRPC параметров (built‑in probes этого не поддерживают). citeturn27view0turn7view2  

**Startup**
- [ ] Для slow start предусмотрен `startupProbe`, который отключает liveness/readiness до успеха; параметры выбраны через `failureThreshold * periodSeconds` ≥ worst‑case startup. citeturn4view4turn17view0turn27view2  

**Shutdown / draining**
- [ ] На SIGTERM/SIGINT сервис переводит readiness в fail (draining), затем выполняет `http.Server.Shutdown(ctx)` с deadline и ожидает завершения. citeturn9view0turn3view4turn8view6  
- [ ] Учтено, что `Shutdown` не закрывает hijacked connections (WebSocket и т.п.) — они закрываются отдельно. citeturn3view4  
- [ ] Тайминг shutdown согласован с `terminationGracePeriodSeconds` (по умолчанию 30с) и возможным `PreStop`, который выполняется до TERM и входит в общий budget. citeturn2search2turn3view3turn27view3  

**Admin/debug и безопасность**
- [ ] pprof/expvar не доступны с публичного listener’а по умолчанию; есть безопасный способ доступа (например, отдельный admin‑порт, закрытый периметром). citeturn8view1turn3view7turn13view0  
- [ ] Нет “слепого” `import _ "net/http/pprof"` на default mux публичного HTTP‑сервера; если pprof нужен — handlers регистрируются осознанно на admin mux. citeturn8view1turn3view5turn0search7  

**Crash diagnostics и runbooks**
- [ ] Документировано/настроено поведение при panic (GOTRACEBACK/traceback detail), и понятно, где искать crash output. citeturn3view6turn14view0turn14view1  
- [ ] В репозитории есть runbook/playbook‑скелет с triage/investigation/resolution и контактами; есть правило регулярно обновлять. citeturn3view8turn8view2  

### Что из результата оформить отдельными файлами в template repo

Рекомендуемая нарезка (так, чтобы это было почти “прямо в docs/”):

- `docs/engineering-standards/operational-readiness.md`  
  Семантика `/livez`/`/readyz`/`/startupz`, требования к dependency health reporting, правила для shutdown/draining, рекомендуемые probe defaults и как их подбирать. citeturn4view4turn27view2turn17view0turn3view4  

- `docs/engineering-standards/admin-debug-endpoints.md`  
  Правила по pprof/expvar, “почему нельзя на публичном порту”, безопасные варианты доступа (admin port, периметр), примеры команд для сбора профилей и предупреждения про DefaultServeMux. citeturn8view1turn3view5turn8view0turn3view7turn13view0  

- `docs/engineering-standards/graceful-shutdown.md`  
  Lifecycle в Kubernetes (TERM → grace → KILL), влияние PreStop, согласование с `terminationGracePeriodSeconds`, как правильно делать `http.Server.Shutdown`. citeturn17view1turn3view3turn2search2turn3view4  

- `docs/runbooks/README.md`  
  Как писать runbooks/playbooks, требования к структуре (triage/investigation/resolution/communication), правило ревью/актуализации. citeturn3view8turn8view2  

- `docs/runbooks/incident-triage-template.md`  
  Шаблон triage: “проверить readiness/liveness, последние релизы, собрать pprof/goroutine dump, оценить зависимость, решить mitigation/rollback”. (Шаблон как документ — с ссылками на internal tools.) Основание: требования к стадиям triage/investigation/resolution и важность процесса. citeturn3view8turn8view3  

- `docs/llm-instructions/observability-operational-readiness.md`  
  Сжатый MUST/SHOULD/NEVER набор (раздел выше) в форме “как генерировать изменения в коде/манифестах без догадок”. citeturn27view3turn3view4turn8view1turn3view8