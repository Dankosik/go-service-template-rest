# Engineering standard и LLM-instructions для production-ready Go микросервиса

## Scope

Этот стандарт и template подходят, когда вы хотите «boring, battle-tested» стартовую точку для нового Go-сервиса: HTTP/JSON или gRPC, деплой в контейнере (часто в Kubernetes), обязательные NFR (наблюдаемость, безопасность, управляемость, тестируемость), и ожидаете, что LLM будет генерировать код в рамках уже заданных конвенций, **без догадок о структуре репо и архитектуре**. citeturn6search28turn18search1turn7search2turn1search3

Подход особенно полезен, если вы:
- строите несколько однотипных сервисов и хотите стандартизировать базовые решения (health, metrics, tracing, graceful shutdown, конфиг, CI gates); citeturn7search2turn8search2turn8search3turn21view0
- хотите минимизировать «площадь принятия решений» для LLM (и людей) через фиксированные defaults и договорённости о код-стайле; citeturn0search3turn0search2turn0search32
- хотите встроить в репо проверяемые требования безопасности: vuln scanning, секреты, логирование, правила для API и authz/authn. citeturn1search10turn10search3turn9search0turn9search1turn7search0

Не подходит (или требует серьёзной адаптации), если:
- вам нужен heavy framework с генерацией кода, сложной DI-системой или «магией» (это плохо сочетается с предсказуемостью LLM-генерации и усложняет review); citeturn0search3turn0search2
- у вас нестандартные требования к runtime: ultra-low-latency, real-time, экстремальные ограничения памяти/CPU — потребуется другой набор дефолтов по timeouts, профилированию, нагрузочным тестам и даже протоколам; citeturn13view0turn15view0turn17search33
- сервис строго не «cloud native» (например, embedded, «one binary» без контейнеров/оркестрации). Тогда части про Kubernetes-probes/chaos, некоторые CI-стадии и контейнерные рекомендации будут лишними. citeturn7search2turn18search1turn18search3


## Recommended defaults для greenfield template

Ниже — «нормативные» defaults, которые должны быть **в самом template**, чтобы LLM могла опираться на них как на контракт.

**Версия Go и toolchain**
- Стандарт: держать project на актуальном стабильном релизе Go и фиксировать `go`/`toolchain` в `go.mod` (чтобы сборка была воспроизводимой и поведение toolchain было однозначным на CI и у разработчиков). Механизм выбора toolchain официально описан; `toolchain` может задавать предпочитаемую версию инструментария поверх минимальной версии `go`. citeturn6search1turn6search0turn4view0

**Минимально достаточная структура репозитория (без «культа папок»)**
- Официальная рекомендация по layout для модуля с internal-пакетами: `package main` в корне или в командном каталоге + `internal/` для непубличного кода приложения. В template фиксируем ровно столько структуры, сколько нужно LLM для стабильных импорт-путей и разделения слоёв. citeturn6search28turn6search0

Практический baseline:
- `cmd/<service>/main.go` — только wiring, запуск, shutdown.
- `internal/app/` — композиция зависимостей и жизненный цикл.
- `internal/httpserver/` — сервер, роутинг, middleware.
- `internal/transport/…` — клиенты (HTTP/gRPC) и их настройки.
- `internal/storage/…` — DB-слой, миграции/репозитории (если нужны).
- `internal/observability/` — tracing/metrics/log correlation.
- `internal/config/` — конфиг + валидация на старте.
- `docs/` — стандарты и LLM-инструкции (список файлов — ниже).

**Конфигурация**
- Default: конфиг хранится во внешней среде (env vars), валидируется при запуске, при ошибках — fail-fast. Это основной принцип 12-factor (конфиг как env) и уменьшает риск случайного коммита секретов в репозиторий. citeturn18search0turn9search1

**HTTP сервер по умолчанию: безопасные timeouts и лимиты**
- Template **обязан** выставлять timeouts и лимиты заголовков. В `net/http.Server` есть прямые поля `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`, `MaxHeaderBytes`, с документированным поведением (включая fallback’и при нуле/отрицательных значениях). citeturn13view0turn13view3
- Default: graceful shutdown через `Server.Shutdown(ctx)` с контекстом/таймаутом и ожиданием завершения. Поведение `Shutdown` (закрывает listeners → idle conns → ждёт активные) описано в stdlib. citeturn21view0turn21view2

**HTTP client по умолчанию: reuse + таймауты**
- Default: один (или несколько) переиспользуемых `http.Client` на сервис/подсистему. Doc подчёркивает, что `Client` и особенно `Transport` имеют внутреннее состояние (пулы соединений), должны переиспользоваться и безопасны для concurrent use. citeturn15view3turn15view2turn15view0
- Default: `Client.Timeout` обязателен (или эквивалентные ограничения через Context), и его семантика включает connect, redirects, чтение body и отменяет запросы через Context. citeturn15view0turn15view2

**Логирование**
- Default: структурные логи через `log/slog` (stdlib), JSON в production, человекочитаемый формат — опционально локально. `slog` является стандартным пакетом структурного логирования в Go. citeturn1search3
- Security logging и запреты на утечки PII/секретов — опираемся на guidance entity["organization","OWASP","web security foundation"] (Logging Cheat Sheet / Logging Vocabulary / Secrets Management). citeturn9search0turn9search4turn9search1

**Ошибки и контракты API**
- Default для HTTP: стандартизирующий формат ошибок — Problem Details (RFC 9457), чтобы не изобретать «ещё один JSON ошибки» в каждом сервисе. RFC определяет machine-readable структуру для error details и аккуратно заменяет RFC 7807. citeturn19search0
- Default для gRPC: придерживаться официальных status codes и их семантики. citeturn19search1turn19search9turn19search29

**Наблюдаемость**
- Трейсинг/метрики: vendor-neutral подход через entity["organization","OpenTelemetry","cncf observability project"] + OTLP как протокол доставки телеметрии. OTLP (trace/metric/log) документирован как стабильный для основных сигналов. citeturn7search3turn7search15turn8search15
- Важно: в Go-экосистеме OpenTelemetry статус сигналов по документации — Traces/ Metrics stable, Logs beta. Это означает, что template должен давать трассинг и метрики «из коробки», а logs-to-OTel делать опционально/экспериментально. citeturn8search11turn7search7
- Семантика: следовать OpenTelemetry semantic conventions для HTTP spans, чтобы атрибуты/имена были совместимы между сервисами и тулзами. citeturn19search3turn19search7
- Metrics endpoint: совместимость с entity["organization","Prometheus","monitoring system project"]: conventions по naming/labels и стандартам экспозиции (Prometheus exposition format / OpenMetrics). citeturn8search0turn8search1turn8search8turn8search2

**Health endpoints и оркестратор**
- Если деплой предполагается в entity["organization","Kubernetes","container orchestration project"]: template должен иметь отдельные endpoints/режимы для liveness/readiness/startup (или эквивалентную семантику), чтобы корректно настраивать probes. Kubernetes чётко разделяет startup/readiness/liveness и описывает их назначение. citeturn7search2turn7search6

**DB и соединения**
- Default (если есть SQL): использовать `database/sql` и строго задавать лимиты пула соединений через `SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxIdleTime`/`SetConnMaxLifetime`. Официальный гайд отдельно предупреждает о рисках (в т.ч. «похоже на семафор» и возможные deadlock’и при неправильной настройке). citeturn11search3turn11search25

**Security scanning и supply chain**
- Vulnerability scanning: `govulncheck` как официальный инструмент экосистемы Go для анализа известных уязвимостей с приоритезацией по reachability. citeturn1search10turn1search21turn10search3
- Supply chain baseline (опционально, но «правильно» для production templates):
  - SBOM в формате entity["organization","SPDX","sbom standard project"]; SPDX описывает, что SBOM — коллекция элементов, описывающих состав/происхождение/лицензии/и т.д. citeturn10search2turn10search6
  - SLSA как спецификация уровней и attestations/provenance. (Важно: v1.0 помечена как retired, но сама модель уровней и provenance остаётся широко используемой точкой отсчёта; это надо явно фиксировать как trade-off и при необходимости следить за актуальными «Current activities».) citeturn10search0turn10search4
  - entity["organization","OpenSSF","open source security foundation"] Scorecard как автоматизированные supply chain checks на уровне репозитория. citeturn10search1turn10search5

**Контейнеризация**
- Default: multi-stage Docker builds (сборка → минимальный runtime) как best practice, уменьшающий размер и attack surface итогового образа. Это прямо рекомендуется в Docker documentation. citeturn18search1turn18search18turn18search34

**Что из этого оформить отдельными файлами в template repo**
- `docs/engineering-standard.md` — «как мы пишем Go-сервисы»: структура, границы слоёв, ошибки, timeouts, shutdown, observability.
- `docs/llm-instructions.md` — MUST/SHOULD/NEVER для LLM + примеры.
- `docs/testing-nonfunctional.md` — NFR testing: fuzz/race/leaks/perf/load/security/chaos/flaky management + CI матрица.
- `docs/security-baseline.md` — OWASP API Top 10 + ASVS уровень (минимум) + секреты/логирование.
- `docs/observability.md` — OpenTelemetry/OTLP + Prometheus conventions + correlation policy.
- `docs/api-errors.md` — RFC 9457 (Problem Details) и политика mapping ошибок.
- `docs/adr/…` — короткие ADR на ключевые решения (HTTP vs gRPC, Prom vs OTEL metrics, slog, toolchain strategy).
- Корневые «конвенционные» файлы: `CONTRIBUTING.md`, `Makefile`, `Dockerfile`, CI workflow, и проверяемые правила форматирования/модулей/сканирования (gofmt/go mod tidy/govulncheck/etc.). citeturn6search1turn18search1turn1search10


## Decision matrix / trade-offs

Ниже — решения, где возможны разумные альтернативы. В template выбираем «по умолчанию», но фиксируем правила, **когда** переключаться.

| Зона решения | Default в template | Альтернативы | Trade-offs / когда менять |
|---|---|---|---|
| API транспорт | HTTP/JSON (net/http) | gRPC | gRPC даёт строгий контракт через Protobuf и стандартные status codes, хорош для service-to-service; HTTP проще для внешних клиентов. Выбор gRPC должен идти вместе с политикой auth (в т.ч. mTLS) и едиными кодами ошибок. citeturn19search29turn19search1turn19search2turn15view3 |
| Формат ошибок HTTP | RFC 9457 (Problem Details) | «свой JSON», RFC 7807 (устар.) | RFC 9457 снижает зоопарк ошибок и повышает предсказуемость для клиентов; кастом формат почти всегда ведёт к несовместимости между сервисами. citeturn19search0 |
| Логи | `log/slog` | zap/zerolog | `slog` снижает зависимости и даёт единый API; сторонние логгеры могут быть быстрее/фичастее, но усложняют template и LLM-генерацию (особенно при смешивании). citeturn1search3turn0search3 |
| Трейсинг | OpenTelemetry + OTLP | vendor tracing | OTLP/OTel — vendor-neutral и стандартизован для доставки сигналов; vendor SDK может быть проще «в одном вендоре», но ухудшает переносимость. citeturn7search3turn8search15 |
| Метрики | Prometheus exposition + naming | OpenTelemetry metrics end-to-end | Prometheus — де-факто стандарт для scrape-модели и имеет документированные conventions по metric/label naming и формату экспозиции. OTEL metrics тоже зрелые, но вам придётся аккуратно выбрать pipeline/экспорт. citeturn8search0turn8search1turn8search11turn7search11 |
| Конфиг | env vars + валидация | конфиг-файлы, удалённый config service | 12-factor рекомендует env: проще эксплуатация, меньше шанс коммита секретов; конфиг-файлы удобны локально, но повышают риск drift и утечек. citeturn18search0turn9search1 |
| DB доступ | `database/sql` + явные запросы/репозитории | ORM | ORM ускоряет CRUD, но часто создаёт скрытую сложность (N+1, неочевидные транзакции). Для template безопаснее и предсказуемее `database/sql` + ясный контракт пула и контекстов. citeturn11search3turn11search25turn11search1 |
| Supply chain | govulncheck + (опц.) SBOM/Scorecard | «ничего» | govulncheck — низкая стоимость и высокая ценность. SBOM/Scorecard/SLSA добавляют сложность, но повышают доверие к артефактам; имеет смысл включать минимум на релизных пайплайнах. citeturn1search10turn10search3turn10search2turn10search1turn10search0 |

Ключевой принцип: если вы меняете default, фиксируйте это ADR-ом и обновляйте LLM-инструкции, иначе LLM продолжит «генерировать по старому контракту». citeturn0search3turn0search2


## Набор правил в формате MUST / SHOULD / NEVER для LLM

Эти правила предназначены для файла `docs/llm-instructions.md` и должны восприниматься как «контракт генерации».

**MUST**
- MUST генерировать код, который проходит `gofmt` и соблюдает рекомендации Go Code Review Comments (особенно по читаемости, именованию, ошибкам, простоте). citeturn0search3turn0search2
- MUST передавать `context.Context` через все границы (handler → сервис → storage/clients) и уважать отмену/дедлайны; для `context.WithCancel/WithTimeout/WithDeadline` MUST вызывать `CancelFunc` на всех путях, иначе утечки. Это прямо указано в доках `context`, и `go vet` проверяет использование cancel. citeturn11search1turn2search1
- MUST выставлять timeouts/лимиты на HTTP server и клиент: `ReadHeaderTimeout`, `WriteTimeout`, `IdleTimeout`, `MaxHeaderBytes`, `Client.Timeout` или эквивалент через контексты. Это documented API stdlib. citeturn13view0turn13view3turn15view0
- MUST делать graceful shutdown через `Server.Shutdown(ctx)` и обеспечить ожидание завершения shutdown перед exit (как минимум по canonical поведению `Shutdown`). citeturn21view0
- MUST использовать переиспользуемый `http.Client`/`Transport`, а не создавать их «на каждый запрос». Документация подчёркивает необходимость reuse. citeturn15view3turn15view2
- MUST использовать структурные логи (`log/slog`) и соблюдать запреты на логирование секретов/PII; ориентироваться на OWASP Logging/Secrets guidelines. citeturn1search3turn9search0turn9search1
- MUST стандартизировать ошибки HTTP через RFC 9457 Problem Details (если сервис HTTP), и использовать корректные HTTP коды/семантику. citeturn19search0
- MUST при добавлении SQL — настраивать пул соединений и документировать rationale (лимиты могут превращаться в «семафор» и приводить к ожиданиям/дедлокам при неверной конфигурации). citeturn11search3turn11search25
- MUST добавлять тесты, которые проверяют поведение и инварианты, а не только coverage. Для fuzz-тестов MUST добавлять seed inputs (`F.Add`) и/или corpus в `testdata/fuzz/...`, потому что без `-fuzz` fuzz-тест запускается как обычный тест именно на seed’ах. citeturn16search0turn16search4turn5view2

**SHOULD**
- SHOULD минимизировать новые зависимости; если нужна зависимость — объяснить, почему stdlib недостаточно, и закрепить версию/обновить `go.mod` корректно через `go` команды. Управление зависимостями и go.mod описаны в официальной документации. citeturn6search0turn6search3turn6search4
- SHOULD использовать OpenTelemetry для traces/metrics и OTLP для экспорта; следовать semantic conventions для HTTP spans. citeturn8search15turn7search3turn19search3
- SHOULD придерживаться Prometheus naming/label conventions и формата экспозиции, если отдаёте `/metrics`. citeturn8search0turn8search1turn8search8
- SHOULD запускать `govulncheck` как часть CI и исправлять уязвимости при наличии reachable путей. citeturn1search10turn10search3
- SHOULD писать ошибки как значения (не злоупотреблять паниками), использовать wrapping/контекст и “errors are values” подход. citeturn11search0turn0search2
- SHOULD учитывать модель памяти Go при конкурентном коде и использовать гонкоустойчивые паттерны; любые оптимизации «без синхронизации» запрещены. citeturn2search2turn1search2

**NEVER**
- NEVER хардкодить секреты, токены, пароли, приватные ключи, connection strings в коде/тестах/логах; следовать OWASP Secrets Management. citeturn9search1turn9search0
- NEVER логировать данные, которые могут быть эксплуатационно чувствительными (секреты, персональные данные, raw payloads без необходимости). citeturn9search0turn9search4
- NEVER игнорировать ошибки (особенно от I/O, DB, криптографии, сериализации). Это ломает надёжность и усложняет диагностику. citeturn11search0turn0search3
- NEVER использовать `context.Background()` внутри request path «просто чтобы работало»: контексты должны приходить сверху (из handler’а / входящего RPC). citeturn11search1turn21view0
- NEVER добавлять “магические” production изменения без тестов/observability (метрики/трейсы/логи). citeturn17search33turn8search15turn1search3


## Concrete good / bad examples

Примеры ниже предназначены для прямого включения в `docs/llm-instructions.md` и `docs/engineering-standard.md`.

### HTTP server: timeouts + max headers + graceful shutdown

**Bad (типично для LLM-галлюцинаций):**
```go
// ❌ Нет timeouts, нет MaxHeaderBytes, нет graceful shutdown.
func main() {
	http.ListenAndServe(":8080", myHandler())
}
```

**Good:**
```go
func main() {
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           myHandler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MiB (можно использовать DefaultMaxHeaderBytes)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
```
Почему это «good»: семантика таймаутов/лимитов и `Shutdown` документирована в stdlib и должна быть дефолтом template. citeturn13view0turn13view3turn21view0

### context: предотвращение утечек CancelFunc

**Bad:**
```go
func handler(w http.ResponseWriter, r *http.Request) {
	ctx, _ := context.WithTimeout(r.Context(), 2*time.Second) // ❌ cancel игнорируется
	_ = doWork(ctx)
}
```

**Good:**
```go
func handler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel() // ✅ обязательно
	_ = doWork(ctx)
}
```
Документация прямо говорит, что не-вызов `CancelFunc` течёт (утечки дерева контекстов/таймеров), и `go vet` это проверяет. citeturn11search1turn5view3

### http.Client: reuse + timeout

**Bad:**
```go
func call(ctx context.Context, url string) error {
	c := &http.Client{}          // ❌ новый клиент каждый вызов
	_, err := c.Get(url)         // ❌ нет timeout/контекста запроса
	return err
}
```

**Good:**
```go
type Client struct {
	http *http.Client
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{
			Timeout: 5 * time.Second,
			// Transport: можно настроить/переиспользовать при необходимости
		},
	}
}

func (c *Client) Call(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
```
Почему: `Client`/`Transport` должны переиспользоваться (пулы соединений) и `Timeout` имеет чёткую семантику отмены через Context. citeturn15view3turn15view0turn15view2

### Fuzz tests: seeds, которые реально работают в CI

**Good (минимальный шаблон):**
```go
func FuzzParseSomething(f *testing.F) {
	// ✅ seeds для запуска без -fuzz (обычный go test прогонит их)
	f.Add([]byte("ok"))
	f.Add([]byte(""))
	f.Add([]byte("{malformed"))

	f.Fuzz(func(t *testing.T, b []byte) {
		_, _ = ParseSomething(b) // проверяйте инварианты, отсутствие panic, корректные ошибки
	})
}
```
Почему: без режима fuzzing (`-fuzz`) fuzz цель выполняется на seed’ах и corpus, что делает такие тесты полезными как regression tests. citeturn16search0turn16search4turn5view2


## Anti-patterns и типичные ошибки/hallucinations LLM

Эти пункты стоит вынести в отдельный раздел `docs/llm-instructions.md` как «запрещённые шаблоны».

- «Успокоительные» timeouts отсутствуют: сервис без `ReadHeaderTimeout/WriteTimeout/MaxHeaderBytes` уязвим к медленным клиентам и ресурсному истощению; template должен заставлять LLM выставлять эти значения. citeturn13view0turn13view3
- Неправильная отмена контекстов: `context.WithTimeout` без `cancel()` — прямой путь к утечкам. citeturn11search1
- `http.Client` создаётся в цикле/на каждый запрос: ломает преимущества connection pooling и может приводить к большому числу открытых соединений. Документация явно советует переиспользовать transports/clients. citeturn15view3turn15view2
- «Паники как control flow»: LLM часто генерирует `panic(err)` в production path. Политика должна быть: ошибки — значения, обогащайте контекстом и возвращайте наверх, а `panic` — только для truly unrecoverable / programmer bugs. citeturn11search0turn0search2
- Логи с секретами/PII: токены, пароли, raw request body, персональные идентификаторы — запрещены; ориентируйтесь на OWASP guidance по security logging и secrets management. citeturn9search0turn9search1turn7search0
- Метрики с неправильными labels: частая LLM-ошибка — пытаться «засунуть всё в label», особенно ID/уникальные значения. Хотя Prometheus docs формально описывают правила имён/labels, в template это должно быть правилом «не делать высокую кардинальность». Минимум: не кодировать label names в metric name и следовать naming conventions. citeturn8search0turn8search8
- Fuzz/benchmarks «ради галочки»: LLM генерирует fuzz/bench без инвариантов, без seed’ов или с нестабильными данными. В итоге CI не ловит баги и только тратит время. Правило: fuzz/bench должны ловить сбои/регрессии, а не увеличивать coverage. citeturn16search0turn2search3turn17search2
- Фальшивые зависимости и несуществующие API: LLM может «придумать» пакет/функцию. Требование: если API не найден в текущем репо и stdlib — остановиться и предложить минимальный вариант или явный запрос на подтверждение, а не галлюцинировать. (Это практическое правило для LLM; оно критично для поддерживаемости и review.) citeturn0search3turn6search0


## Review checklist для PR/code review

Этот чеклист стоит поместить в `docs/review-checklist.md` и ссылаться на него из PR template.

- **Контракты и совместимость**
  - Публичные эндпоинты/контракты документированы; ошибки HTTP в формате RFC 9457 (если HTTP). citeturn19search0
  - Для gRPC: корректные status codes и согласованная политика ошибок. citeturn19search1turn19search9

- **Контекст, таймауты, отмена**
  - Контексты не теряются между слоями; нет `context.Background()` в request path. citeturn11search1
  - Timeouts выставлены на server/client; ограничения заголовков и размеры учтены. citeturn13view0turn15view0
  - Graceful shutdown реализован через `Server.Shutdown(ctx)` и корректно ожидается. citeturn21view0

- **Безопасность**
  - Нет секретов/ключей/токенов в коде, тестах, логах; соблюдены OWASP logging/secrets правила. citeturn9search1turn9search0
  - В CI присутствует `govulncheck` и нет необъяснённых игнорирований. citeturn1search10turn10search3
  - Валидация входных данных присутствует, выполняется как можно раньше; ориентир — OWASP input validation. citeturn9search2turn9search12

- **Наблюдаемость**
  - Логи структурные (slog), содержат полезные ключи, не содержат чувствительных данных. citeturn1search3turn9search0
  - Трейсы/метрики подключены согласно OpenTelemetry/Prometheus conventions; имена/атрибуты согласованы. citeturn19search3turn7search3turn8search0

- **Тестируемость**
  - Тесты проверяют поведение/инварианты; fuzz-тесты имеют seeds и/или corpus. citeturn16search0turn16search4
  - Нет флейки из-за времени/параллелизма; при необходимости используется `-shuffle`/`-count` стратегии (см. ниже). citeturn5view1turn5view2

- **Производительность и ресурсы**
  - Если изменения в hot path: есть бенчмарки и (на release/nightly) сравнение результатов `benchstat`. citeturn2search3turn17search2
  - При подозрении на регрессии — предусмотрены профили (pprof endpoints в безопасном режиме). citeturn17search33turn17search1turn17search0


## Исследование подтемы: non-functional testing и устойчивость test suite

Цель: template должен содержать практический стандарт того, **какие NFR-проверки запускаются где**, и как LLM должна генерировать тесты «на баги», а не «на coverage».

### Базовые принципы устойчивости test suite

- Делайте тесты воспроизводимыми: `go test -shuffle=on` сообщает seed для воспроизводимости, а `-shuffle=N` позволяет повторить порядок. Это ключевой инструмент для выявления скрытых зависимостей тестов от порядка. citeturn5view1
- Управляйте флейками через повторные прогоны: `-count n` повторяет тесты/bench/seed прогоны; полезно для выявления flaky поведения, но дорого по времени. citeturn5view2
- Всегда задавайте `-timeout` на CI; по умолчанию 10m, но для PR pipeline часто разумно ужесточать, иначе зависшие тесты съедают весь бюджет CI. citeturn5view1turn5view3
- Понимайте тестовый кэш: «idiomatic способ» явно отключить reuse кэша — `-count=1`. Это помогает, когда тесты зависят от внешней среды/файлов/переменных окружения и вы хотите гарантировать реальный прогон. citeturn5view2turn5view3

### Матрица проверок: PR CI vs nightly vs release

Ниже — практическая матрица для `docs/testing-nonfunctional.md`.

**Проверки, которые должны быть встроены в PR CI (gating)**
- Unit tests + seed-run fuzz tests: `go test ./... -shuffle=on -count=1 -timeout=…`  
  Обоснование: `-shuffle` ловит порядок-зависимости; `-count=1` гарантирует прогон; fuzz-тесты без `-fuzz` всё равно прогоняют seeds/corpus и становятся regression barrier. citeturn5view1turn5view2turn16search0turn5view3
- Race detection (как минимум на Linux/amd64): `go test -race ./...`  
  Race detector — официальный инструмент для поиска data races; data races — частые и крайне сложные баги конкурентных систем. citeturn1search2turn1search6turn4view0
- Static checks (минимум): `go vet ./...` и/или оставить `go test` с включённым vet (по умолчанию `go test` запускает curated list vet checks, управляется `-vet`). citeturn2search1turn5view3
- Vulnerability scanning: `govulncheck ./...`  
  Это «низкая стоимость, высокая отдача» и официальный путь в Go экосистеме, использующий Go vulnerability database (OSV schema). citeturn1search10turn10search3
- Formatting/modules gate: проверка `gofmt` и отсутствия незакоммиченных изменений после `go mod tidy` (как минимум через CI-скрипт). Управление зависимостями через `go mod tidy` описано официально. citeturn6search3turn4view0turn0search32

**Проверки для nightly (или периодических) прогонов**
- Fuzzing «по-настоящему»: `go test -fuzz=… -fuzztime=…` (на выбранных пакетах/целях)  
  Флаги `-fuzz` и `-fuzztime` документированы в `go test` flags; это позволяет находить краши/edge cases за пределами seeds. citeturn5view2turn16search4turn0search1
- Leak detection (goroutine/resource leaks):  
  - Базовый уровень: тесты обязаны корректно отменять контексты (иначе утечки). citeturn11search1  
  - Практический промышленный инструмент: `go.uber.org/goleak` (mature, но сторонний) — полезно для выявления goroutine leaks, которые очень типичны для микросервисов. citeturn17search3turn17academia39
- Performance regression suite (микробенчи): `go test -bench …` + сравнение через `benchstat`  
  `benchstat` официально описывает статистически устойчивые сравнения и рекомендует многократные прогоны (>=10) для значимости. citeturn2search3turn17search2
- Profiling hooks: сбор pprof профилей на representative нагрузке и анализ через `go tool pprof`  
  Есть официальные материалы по профилированию (Diagnostics, blog pprof) и стандартные пакеты `runtime/pprof` и `net/http/pprof`. citeturn17search33turn17search0turn17search2turn17search1
- Конкурентные санитайзеры (экспериментально, платформенно-зависимо): `-msan`/`-asan`  
  Эти флаги описаны в `go` build flags и имеют жёсткие ограничения по OS/arch и toolchain (Clang/GCC). Их имеет смысл выносить из PR gating и запускать на специальных runner’ах. citeturn4view0

**Проверки для release / pre-release стадий**
- Нагрузочные тесты (smoke/stress/soak) в staging с чёткими pass/fail thresholds  
  Практический default: entity["company","Grafana Labs","observability company"] k6 как современный load-testing инструмент; документация описывает API load testing и типы тестов (smoke/stress/soak/spike). citeturn18search12turn18search9turn18search6
- Chaos experiments (если деплой в Kubernetes): сценарии отказов (pod kill, network delay/partition, resource stress)  
  Практический выбор: Chaos Mesh (Kubernetes-native инструмент), у него документирована архитектура и примеры инъекций. Это не для каждого сервиса, но для критичных потоков даёт сильную уверенность в устойчивости. citeturn18search3turn18search32turn7search2
- Supply chain артефакты (если требуется): SBOM (SPDX) + политики уровня SLSA/Scorecard  
  SPDX определяет SBOM как стандартный представимый набор элементов; Scorecard даёт автоматизированные supply chain checks; SLSA описывает уровни и provenance. citeturn10search2turn10search1turn10search0

### Как LLM должна создавать non-functional тесты, которые ловят проблемы

Норматив для `docs/llm-instructions.md` (секция “Testing”):

- **Fuzzing**
  - LLM MUST выбирать цели fuzzing там, где есть парсинг/десериализация/валидаторы/преобразования (JSON, base64, UUID, query parsing, кастомные протоколы).  
  - LLM MUST формулировать инварианты: «не паниковать», «ошибка возвращается консистентно», «round-trip сохраняет свойства», «валидатор не принимает запрещённое».  
  - LLM MUST добавлять несколько `F.Add` seeds (минимум: empty, typical valid, typical invalid) и при найденном крэше — фиксировать его вход как regression (в `testdata/fuzz/...` или как seed). citeturn16search0turn16search4turn5view2

- **Property-based testing**
  - Default в template: делать ставку на встроенный fuzzing (как более «нативный» и поддерживаемый путь). citeturn16search4  
  - Если нужна классическая property-based модель: допустимо использовать `testing/quick` (stdlib, но пакет frozen — не ждать развития) либо явно подключать зрелую библиотеку и закреплять её версию (это уже осознанный trade-off). citeturn16search3

- **Race**
  - LLM SHOULD добавлять тесты, которые реально создают concurrency (goroutines, каналы, параллельные обработчики) и проверять корректность под `-race`. Race detector — официальный путь ловить data races. citeturn1search2turn4view0

- **Leak detection**
  - LLM MUST проектировать goroutine lifecycle: каждая goroutine должна завершаться при отмене контекста или закрытии канала.  
  - Для тестов: использовать `defer cancel()`, ограничивать ожидания таймаутами, избегать «вечных» select без ctx.Done(). Эти требования напрямую связаны с тем, что отмена контекстов и таймеры могут течь при неправильном использовании. citeturn11search1turn2search2

- **Performance regression**
  - LLM SHOULD добавлять микробенчмарки только для действительно критичных функций и использовать стабильные датасеты. Сравнение результатов — через `benchstat`, поскольку он делает статистически устойчивые сравнения. citeturn2search3turn17search2

- **Load/security/chaos**
  - LLM SHOULD писать load-тесты как сценарии поведения (не «один запрос»), с threshold’ами и отделением тестовых данных от production secrets. k6 прямо ориентирован на такие тесты. citeturn18search12turn18search9turn9search1
  - LLM MUST не подменять security-тестирование “сканером зависимостей”: нужен как минимум `govulncheck`, а также негативные тесты на input validation/authz в критичных местах, учитывая OWASP API Top 10. citeturn1search10turn7search0turn9search2

Эта модель (PR gating + nightly + release) даёт практический баланс: быстрые проверки удерживают качество на ежедневном цикле, а «дорогие» проверки (fuzz длительный, perf regressions, load/chaos) ищут глубокие баги без разрушения developer velocity. citeturn5view1turn18search12turn18search3turn16search4turn2search3