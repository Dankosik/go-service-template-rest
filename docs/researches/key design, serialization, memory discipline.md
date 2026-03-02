# Engineering standard и LLM-instructions для production-ready Go-микросервиса

## Scope

Этот стандарт предназначен для greenfield **микросервиса на Go**, который:
- деплоится как отдельный сервис (контейнер) и живёт в типичной cloud-native среде (часто под entity["organization","Kubernetes","container orchestration project"]); health checks, graceful shutdown и ограничение ресурсов — обязательная часть контракта. citeturn4search2turn4search6turn10view5
- имеет сетевой API (HTTP/JSON и/или entity["organization","gRPC","rpc framework"]), и требует предсказуемого, повторяемого поведения под нагрузкой (timeouts, лимиты входных данных, наблюдаемость). citeturn10view2turn10view4turn4search3
- использует базовые backing services (БД, кэш/очередь), где важно «boring, battle-tested» поведение: явные TTL, выбранная политика eviction, контролируемая кардинальность ключей/лейблов. citeturn20view2turn19view1turn16search0
- разрабатывается с активным использованием LLM-инструментов, поэтому решения/допущения должны быть **жёстко зафиксированы** в repo conventions и docs, чтобы модель не «догадывалась». (Это — цель вашего запроса, а не отдельная «best practice».)

Этот стандарт **не подходит** или требует существенной адаптации, если:
- вы делаете библиотеку/SDK (тогда публичный API и семантическое версионирование важнее шаблона микросервиса; структура repo и экспортируемые пакеты будут другими). citeturn2search1
- вы пишете batch/ETL или CLI (другие приоритеты: простота запуска, работа с файлами/стримами, иные SLO; многие решения вокруг HTTP/k8s probes лишние). citeturn4search2
- у вас уже есть платформенные стандарты (единый логгер, трейсер, service mesh, централизованные шаблоны деплоя), и «универсальный» шаблон должен им соответствовать — тогда этот документ становится *baseline* и должен быть переопределён ADR-ами.

## Recommended defaults для greenfield template

Ниже — **стартовые дефолты**, которые можно «почти напрямую» превратить в `docs/` и соглашения репозитория. Везде, где решение может быть спорным, оно будет отражено в matrix ниже, вместе с альтернативами.

**Версия Go и политика совместимости**
- Зафиксировать актуальный стабильный toolchain: на дату **2026‑03‑02** последняя major-версия — **Go 1.26.0**, релиз **2026‑02‑10**. citeturn6view2turn6view1  
- В `go.mod` использовать связку:
  - `go 1.25.0` как минимальную совместимую версию (в момент релиза 1.26 сам инструмент `go mod init` по умолчанию ориентирует новые модули на предыдущую поддерживаемую версию). citeturn6view1turn7search0  
  - `toolchain go1.26.0` (или `go1.26.x`) как «рекомендуемый минимум toolchain» для единообразия CI/локальной разработки. citeturn7search1turn7search0  
- Принять как правило: «обновление go/toolchain — через отдельный PR с прогоном CI + govulncheck». citeturn4search0turn4search1turn6view2

**Структура репозитория и модулей**
- Следовать официальной рекомендации по организации Go-модуля: отдельные директории для программ и общий `internal/` для пакетной логики, переиспользуемой командами. citeturn2search1  
- Практический layout (boring, распространённый):
  - `cmd/<service>/main.go` — тонкий composition root (инициализация зависимостей, wiring).
  - `internal/app/` — сборка приложения (dependencies, lifecycle).
  - `internal/http/` и/или `internal/grpc/` — транспортный слой.
  - `internal/domain/` (или `internal/core/`) — бизнес-логика без инфраструктурных деталей.
  - `internal/storage/` — DB/Cache клиенты + репозитории.
  - `internal/observability/` — трассировка/метрики/логгинг wiring.

**Форматирование, стиль, документация**
- Код форматируется только `gofmt`/`go fmt` (в PR — обязательная проверка). citeturn14search0turn14search15turn14search13  
- Канонические источники стиля:
  - `Effective Go` как базовая идиоматика. citeturn0search4  
  - Go Wiki “Code Review Comments” как практические «линейки» для ревью. citeturn0search0  
  - (Опционально как «второй уровень строгости») гайд по стилю от Google — полезен, когда нужна минимизация трактовок и «угадываний». citeturn0search12  
- Док-комментарии и публичные API пакетов оформлять так, чтобы `go doc`/pkg.go.dev корректно извлекали документацию. citeturn13search15

**Тесты, статанализ, автоматические фиксы**
- Минимальный обязательный набор в CI:
  - `go test ./...` (с покрытием ключевых пакетов). citeturn14search6turn14search10  
  - `go vet ./...` (как низкошумный поиск подозрительных конструкций). citeturn14search1  
  - `govulncheck ./...` как официальный low-noise анализ уязвимостей, зависящий от реально вызываемых функций/методов. citeturn4search0turn4search1turn4search5  
- Для Go 1.26: зафиксировать практику периодического `go fix` (ручной запуск или отдельный PR), поскольку `go fix` переехал на тот же analysis framework, чтоRut (как и `go vet`) и стал инструментом «модернизации». citeturn7search2turn6view1  
- Для тестов принять правила из “Go Test Comments” как «общий язык» замечаний. citeturn0search20

**HTTP runtime baseline**
- Сервер на стандартном `net/http` (внутри — или `ServeMux`, или лёгкий роутер; выбор отражён в matrix). Критично не «framework», а **явные лимиты и shutdown**:
  - Таймауты сервера: `ReadHeaderTimeout`, `WriteTimeout`, `IdleTimeout` (нулевые значения означают «нет таймаута»). citeturn10view2turn17view1  
  - `MaxHeaderBytes` для ограничения размера заголовков. citeturn10view3  
  - Ограничение body через `http.MaxBytesReader`, чтобы избежать случайных/злонамеренных гигабайтных тел. citeturn10view4  
  - Graceful shutdown через `(*http.Server).Shutdown(ctx)`; процесс обязан дождаться его завершения. citeturn10view5  
- Контекст: в handlers использовать `Request.Context()` и **не** использовать `CloseNotifier` (он deprecated; новый код должен опираться на контексты). citeturn3search1turn18view0  
- Исходящие HTTP вызовы:
  - Использовать переиспользуемый `http.Client` (не создавать новый на каждый запрос).
  - Всегда задавать `Client.Timeout` (0 — «нет таймаута») и/или контекст запроса. citeturn17view1turn18view0

**Логи, наблюдаемость, метрики**
- Логирование: стандартный `log/slog` в JSON-формате, структурно (key/value). citeturn3search0turn3search4  
- Модель безопасности по логам: не логировать секреты/PII без явной необходимости; применять “security logging” подход и vocabulary, чтобы мониторинг был автоматизируемым. citeturn8search0turn8search2turn8search11  
- Трассировка/метрики: базовый стек на entity["organization","OpenTelemetry","cncf observability project"]:
  - включить context propagation (по умолчанию W3C Trace Context) и соблюдать propagation на входе/выходе сервисов. citeturn1search5turn1search2  
  - при метриках использовать Semantic Conventions, чтобы атрибуты были совместимы между сервисами. citeturn16search1  
- Если экспонируются метрики в формате entity["organization","Prometheus","monitoring system"]: соблюдать best-practices по naming и **избегать high-cardinality labels** (user_id, email, request_id и т.п.). citeturn16search0

**Security baseline (что считать “production-ready”)**
- Использовать entity["organization","OWASP","web application security project"] ASVS как «каталог требований», а TOP10 для API — как «каталог рисков», чтобы в документах не было произвольных вкусовых решений. citeturn0search2turn8search3turn8search6  
- Минимальная безопасность для шаблона:
  - входная валидация и отсутствие SQL injection: параметризованные запросы/подготовленные выражения, запрет string-concat SQL. citeturn8search1turn8search5turn16search2  
  - ошибки: клиенту — безопасное сообщение, детали — только в логах. citeturn8search7turn8search0  
  - secrets management: секреты не хардкодить и не коммитить; описать требования к хранению/ротации. citeturn8search2turn2search2  
  - управление зависимостями: регулярный `govulncheck`. citeturn4search0turn4search1

**Health checks**
- HTTP: `/livez` и `/readyz` (или аналог) с семантикой, совместимой с probes. citeturn4search2turn4search6  
- gRPC: реализовать стандартный protocol health checking. citeturn4search3

**Кэш (Redis) как стандартный backing service для template**
- Принять кэш как управляемый ресурс с фиксированными лимитами памяти (`maxmemory`) и выбранной политикой eviction; по умолчанию — `allkeys-lru` как «часто хороший дефолт» при отсутствии более точных знаний об access pattern. citeturn20view2turn20view1  
- Запретить `KEYS` в регулярном коде; для итераций — `SCAN`. citeturn16search7turn5search0turn5search3  
- Стандартизировать key schema (namespacing, tenant, versioning) и хранение значений; подробный implementation guide — в последнем разделе. citeturn5search4turn19view2turn12view4

## Decision matrix / trade-offs

Ниже — «матрица решений» для шаблона. Внутри repo это лучше оформить как ADR(ы): *default + когда отступать + какие риски*. (Важное: в шаблоне должно быть **одно** дефолтное решение; альтернативы — описаны, но не включены «по умолчанию», чтобы LLM не смешивала стили.)

| Область | Default для template | Альтернатива | Когда выбирать альтернативу | Риски/стоимость |
|---|---|---|---|---|
| API transport | HTTP/JSON + OpenAPI контракт | gRPC + Protobuf (или gRPC + JSON transcoding) | gRPC — если много межсервисных вызовов, нужна строгая схема; JSON transcoding — если надо обслужить HTTP клиентов без ручного REST слоя | gRPC требует генерации, дисциплины версионирования схем; OpenAPI проще для внешних клиентов. citeturn2search7turn9search2turn4search3turn9search3 |
| Serialization | JSON для внешнего HTTP | Protobuf для внутренних вызовов/кэша | Protobuf — когда критичны размер/скорость и нужна схема; JSON — когда важна читаемость/дебаг | JSON может интерпретироваться парсерами по-разному; в Go `encoding/json` отдельно описывает security considerations. citeturn15view1turn9search2turn9search10 |
| Logging | `log/slog` structured JSON | Zap/zerolog | Если нужны экстремальные требования к производительности логирования или уже стандартизован корпоративный стек | Больше зависимостей и конфигураций; `slog` — стандартная библиотека и проще для шаблона. citeturn3search0turn3search4 |
| Observability | OpenTelemetry SDK + W3C propagation | Prometheus-only | Если инфраструктура уже полностью Prometheus и не нужны traces | Смешивание подходов ведёт к несогласованным атрибутам/лейблам; OTel semconv даёт единый язык. citeturn1search5turn1search2turn16search1turn16search0 |
| HTTP server | `net/http` + явные таймауты/лимиты | framework/router (chi/gin/echo) | Если нужна сложная маршрутизация/мидлвари, и команда это уже обязала | Framework сам по себе не делает сервис «production-ready», если не выставлены лимиты, shutdown, контексты. citeturn10view2turn10view4turn10view5turn3search1 |
| Config | env vars + defaults (12-factor) | конфиг-файлы | Если требуется локальная/air-gapped доставка конфигов файлами | Env уменьшает риск случайного коммита секретов; соответствует 12-factor. citeturn2search2turn2search13 |
| Error handling | типизированные ошибки + wrapping (`%w`) | “errors как строки” | Почти никогда (кроме очень small scripts) | Go 1.13 ввёл стандартный подход к wrapping/unwrapping; строки ломают `errors.Is/As`. citeturn3search7 |
| Redis eviction | `maxmemory` + `allkeys-lru` | `volatile-ttl`, `noeviction`, LFU | `volatile-ttl` — если TTL используется как «подсказка»; `noeviction` — если Redis не кэш, а data store, и вы готовы получать ошибки записи при переполнении | Неправильная политика может вызвать либо лавинообразные eviction и miss rate, либо ошибки при записи и деградацию. citeturn20view2turn20view1turn20view4 |
| Redis key iteration | `SCAN` | `KEYS` | `KEYS` — только для дебага/одноразовых операций (миграции) | `KEYS` — O(N) и «don’t use in regular application code»; `SCAN` инкрементальный и может использоваться в production. citeturn16search7turn5search0turn5search3 |
| Метрики labels | низкая кардинальность | “всё в labels” | Никогда для user_id/request_id и прочих unbounded измерений | Каждая уникальная комбинация labels — новый time series; Prometheus явно предупреждает о high-cardinality. citeturn16search0 |

## Набор правил в формате MUST / SHOULD / NEVER для LLM

Ниже — заготовка, которую имеет смысл почти напрямую положить в `docs/llm/INSTRUCTIONS.md` и (частично) в общий «prompt prefix». Формулировки специально операциональны: чтобы LLM могла следовать им без догадок.

**MUST**
- Генерировать, изменять и форматировать Go-код так, чтобы он проходил `gofmt`/`go fmt`. citeturn14search0turn14search15  
- Любое изменение кода должно сохранять сборку и тесты: минимум `go test ./...` и `go vet ./...` на изменённых пакетах. citeturn14search6turn14search1  
- Для изменений зависимостей и security-sensitive кода — обязательно прогонять `govulncheck`. citeturn4search0turn4search1  
- Всегда использовать `context.Context` по правилам стандартной библиотеки:  
  - `ctx` — первый параметр;  
  - не хранить `Context` внутри структур;  
  - не передавать `nil context`;  
  - `WithCancel/WithTimeout/WithDeadline` — всегда с `defer cancel()`; иначе утечки (и `go vet` это проверяет). citeturn18view0turn14search1  
- В HTTP handlers: всегда использовать `r.Context()`; не использовать deprecated `CloseNotifier` и не эмулировать cancel вручную. citeturn3search1turn18view0  
- На входе HTTP: всегда ставить ограничения размера заголовков/тела запроса (как минимум через `MaxHeaderBytes` и `MaxBytesReader` или эквиваленты). citeturn10view3turn10view4  
- На сервере: выставлять не‑нулевые timeouts (`ReadHeaderTimeout` и др.), т.к. нулевые значения означают отсутствие таймаутов. citeturn10view2  
- Для shutdown: использовать `Server.Shutdown(ctx)` и ждать результата. citeturn10view5  
- Для исходящих HTTP вызовов: использовать переиспользуемый `http.Client` и выставлять `Client.Timeout` (0 — «нет таймаута»). citeturn17view1  
- Ошибки оформлять идиоматично: возвращать `error`, использовать wrapping, чтобы работали `errors.Is/As` (Go 1.13 подход). citeturn3search7  
- Для SQL: использовать параметризацию/подготовленные выражения; запретить конкатенацию пользовательских значений в SQL строку. citeturn8search1turn8search5turn16search2  
- Ошибки наружу (HTTP/gRPC) — безопасные и не раскрывающие внутренности; детали — в логах. citeturn8search7turn8search0  
- Логи писать структурно через `log/slog`, без секретов. citeturn3search0turn8search2turn8search0  
- Для метрик: не добавлять labels с unbounded множествами значений (user_id, email, request_id). citeturn16search0  
- Для Redis-кода:  
  - не использовать `KEYS` в регулярной логике;  
  - итерации — через `SCAN`;  
  - ключи проектировать по единой схеме;  
  - для кэшируемых данных задавать TTL (или иметь явное обоснование «без TTL»), и конфигурировать `maxmemory` + eviction policy. citeturn16search7turn5search0turn19view1turn20view1  

**SHOULD**
- Добавлять/обновлять tests и следовать тестовым правилам Go Wiki (табличные тесты, читаемые assert-сообщения, избегать flaky). citeturn0search20turn14search6  
- Соблюдать Go Code Review Comments и Effective Go как базовый стандарт читаемости. citeturn0search0turn0search4  
- Экспортируемые сущности (пакеты/типы/функции) снабжать doc-comments в стиле `go doc`. citeturn13search15  
- Инструментировать сервис телеметрией через OpenTelemetry и не отходить от семантических конвенций без ADR. citeturn1search2turn16search1  
- При работе с кэшем использовать защиту от stampede (например, singleflight) и jitter для TTL. citeturn11search1  

**NEVER**
- Никогда не «придумывать» несуществующие пакеты/функции или API. Если кода/интерфейса нет в репозитории — сначала создать его как часть изменения, с тестами и docs.  
- Никогда не создавать новый `http.Client` на каждый запрос и не оставлять исходящие вызовы без таймаутов. citeturn17view1  
- Никогда не передавать `nil context` и не оставлять `cancel()` не вызванным. citeturn18view0  
- Никогда не использовать `KEYS` как часть runtime-логики. citeturn16search7turn5search3  
- Никогда не класть в Prometheus labels/OTel attributes данные, дающие высокую кардинальность (идентификаторы пользователей, UUID запросов). citeturn16search0  
- Никогда не логировать секреты/токены/пароли и не возвращать пользователю stack trace/внутренние детали ошибок. citeturn8search2turn8search7turn8search0  

## Concrete good / bad examples

Ниже примеры, которые стоит положить в `docs/examples/` и использовать как «канонические паттерны» для LLM (модель будет копировать стиль и не изобретать).

### Good: HTTP handler с лимитом body, строгим JSON и уважением контекста

```go
// PUT /v1/widgets/{id}
func (h *Handler) UpdateWidget(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1) Ограничить размер тела запроса.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20 /* 1 MiB */)
	defer r.Body.Close()

	// 2) Декодировать JSON из потока.
	var req UpdateWidgetRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}

	// 3) Дать I/O-операциям бюджет времени.
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := h.svc.UpdateWidget(ctx, req); err != nil {
		writeDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

Почему это “good”: `MaxBytesReader` предназначен для ограничения входных тел и прямо описан как защита от больших запросов, тратящих ресурсы сервера. Возможность `DisallowUnknownFields` встроена в `encoding/json.Decoder`. Правила контекстов требуют вызывать `cancel()` и не хранить контексты в структурах. citeturn10view4turn15view4turn18view0

### Bad: бесконтрольный read-all и отсутствие контекстного бюджета

```go
func (h *Handler) BadUpdateWidget(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body) // игнор ошибок + безлимитно
	var req UpdateWidgetRequest
	_ = json.Unmarshal(body, &req) // игнор ошибок

	_ = h.svc.UpdateWidget(context.Background(), req) // игнор отмены клиента/таймаута
	w.WriteHeader(204)
}
```

Почему это “bad”: `context.Background()` вместо `r.Context()` игнорирует отмену и дедлайны; в документации `context` это противоречит идее «цепочки вызовов должна пропагировать Context». Кроме того, лимит тела запроса в HTTP должен быть явным, иначе вы не контролируете потребление памяти/CPU. citeturn18view0turn10view4

### Good: read-through cache с singleflight, versioned key и TTL

```go
type WidgetCache struct {
	redis RedisClient
	sf    singleflight.Group
}

func widgetKey(tenantID string, widgetID int64) string {
	// v3 — версия семантики/схемы этого объекта в кеше.
	return fmt.Sprintf("svc:widgets:v3:tenant:%s:id:%d", tenantID, widgetID)
}

func (c *WidgetCache) GetWidget(ctx context.Context, tenantID string, widgetID int64) (Widget, bool, error) {
	key := widgetKey(tenantID, widgetID)

	// 1) Fast path: cache hit.
	if b, ok, err := c.redis.GetBytes(ctx, key); err != nil {
		return Widget{}, false, err
	} else if ok {
		var w Widget
		if err := json.Unmarshal(b, &w); err != nil {
			// При повреждённом значении можно удалить ключ и считать как miss.
			_ = c.redis.Del(ctx, key)
			return Widget{}, false, nil
		}
		return w, true, nil
	}

	// 2) Stampede protection: один fetch на ключ.
	v, err, _ := c.sf.Do(key, func() (any, error) {
		w, err := c.loadFromDB(ctx, tenantID, widgetID)
		if err != nil {
			return Widget{}, err
		}

		// TTL + jitter (пример: 5m ± 30s).
		ttl := 5*time.Minute + time.Duration(rand.Int63n(int64(30*time.Second)))

		b, err := json.Marshal(w)
		if err != nil {
			return Widget{}, err
		}
		if err := c.redis.SetBytes(ctx, key, b, ttl); err != nil {
			// Ошибка кэша не должна ломать основной путь (в большинстве случаев).
			// Решение спорное; фиксируется через ADR.
		}
		return w, nil
	})
	if err != nil {
		return Widget{}, false, err
	}
	return v.(Widget), false, nil
}
```

Почему это “good”: singleflight задуман как механизм «duplicate suppression» для одинаковой работы, что прямо подходит для защиты от cache stampede. Redis key schema «типа object-type:id» и совет «придерживаться схемы» — отдельная рекомендация в документации Redis про ключи. citeturn11search1turn19view1

### Bad: KEYS в runtime, отсутствие TTL и tenant safety

```go
func (c *WidgetCache) BadInvalidateAll(ctx context.Context) error {
	// KEYS в приложении: плохо.
	keys, _ := c.redis.Keys(ctx, "widgets:*")
	for _, k := range keys {
		_ = c.redis.Del(ctx, k)
	}
	return nil
}

func widgetKeyBad(widgetID int64) string {
	// Нет tenant. Нет version.
	return fmt.Sprintf("widgets:%d", widgetID)
}
```

Почему это “bad”: Redis явно предупреждает, что `KEYS` — команда для дебага/специальных операций и «не используйте в regular application code», а для production-итераций предлагается `SCAN`. Также Redis документация рекомендует придерживаться схемы ключей и предупреждает о проблемах слишком длинных ключей и бессистемности. citeturn16search7turn5search0turn19view2

## Anti-patterns и типичные ошибки/hallucinations LLM

**Контексты и время**
- LLM часто вставляет `context.Background()` в глубине вызовов «для удобства», ломая отмену, дедлайны и трассировку. Это прямо противоречит правилам `context`: контекст должен передаваться явным параметром и быть первым аргументом. citeturn18view0  
- LLM может забыть `defer cancel()` после `WithTimeout/WithCancel`, что документировано как источник утечек дочерних контекстов (и отмечается, что `go vet` проверяет использование cancel-функций). citeturn18view0turn14search1  

**HTTP**
- «Нулевые таймауты по умолчанию норм» — нет: `net/http` чётко описывает, что ноль/отрицательное значение для таймаутов означает отсутствие таймаута. Если шаблон не задаёт явно, LLM часто оставит поля пустыми. citeturn10view2  
- Чтение тела запроса через `io.ReadAll` без лимитов — классическая ошибка; стандартная библиотека предлагает `http.MaxBytesReader` именно для защиты ресурса. citeturn10view4  
- Создание `http.Client{}` per-request: документация подчёркивает, что transport имеет внутреннее состояние (кэш соединений), поэтому clients следует переиспользовать; также `Timeout=0` — без таймаута. citeturn17view1  

**Ошибки и безопасность**
- “Склей SQL строку через fmt.Sprintf” — типичная hallucination. OWASP прямо рассматривает параметризованные запросы как основной способ предотвращения SQL injection; PostgreSQL описывает параметры `$1`, `$2` в prepared statements. citeturn8search1turn8search5turn16search2  
- Возврат внутренних ошибок/stack trace клиенту «чтобы было удобнее дебажить»: OWASP Error Handling Cheat Sheet рекомендует возвращать общий ответ, логируя детали на сервере. citeturn8search7turn8search0  
- Логирование токенов/ключей/паролей: OWASP cheatsheets по Secrets Management и Logging дают противоположный вектор (централизация, контроль доступа, отсутствие утечек через логи). citeturn8search2turn8search0  

**Метрики**
- Высококардинальные labels «для удобства фильтрации»: официальные практики Prometheus прямо предупреждают не хранить high-cardinality измерения (user IDs и т.п.). citeturn16search0  

**Redis / cache-backed логика**
- Использование `KEYS` для invalidation: Redis документация и отдельные материалы Redis про анти‑паттерны объясняют риск блокировки и советуют `SCAN` в production. citeturn16search7turn5search0turn5search3  
- Отсутствие versioning в ключах/значениях: при изменении схемы объекта LLM часто “просто меняет struct”, что ломает backward-compatibility кэша; ключи должны иметь версию схемы (см. recommended key schema ниже). citeturn19view1  
- Бесконтрольная кардинальность ключей: LLM может включить в ключ «всё подряд» (query string, массивы параметров, сортировки) и создать миллион уникальных ключей. Redis напоминает о стоимости длинных ключей и прямо рекомендует для больших значимых частей ключа использовать хеширование. citeturn19view2  
- Непонимание памяти Redis: удаление ключей не гарантирует возврат RSS в ОС; Redis описывает это поведение malloc/allocator и необходимость планировать по peak. citeturn12view4  
- Запись больших payload без лимитов: в Redis есть большие верхние пределы (ключ до 512MB; bulk string по умолчанию ограничен 512MB), но это не означает «можно так делать». В шаблоне должны быть меньшие guardrails. citeturn19view2turn11search3  

## Review checklist для PR / code review

**Сборка, стиль, тесты**
- Код отформатирован `gofmt`/`go fmt`; нет ручного форматирования. citeturn14search0turn14search15  
- `go test ./...` зелёный; новые ветки логики покрыты тестами (минимум table-driven для бизнес-логики). citeturn14search6turn0search20  
- `go vet ./...` зелёный; нет предупреждений (особенно вокруг cancel-функций для контекстов). citeturn14search1turn18view0  

**Security**
- Нет конкатенации SQL со входными данными; используются параметры. citeturn8search1turn16search2  
- Ошибки наружу безопасные; детали и корреляционные поля — только в логах. citeturn8search7turn8search0  
- Секреты не попали в код/логи/fixtures; политика secrets management соблюдена. citeturn8search2  
- Прогнан `govulncheck` (или CI это делает) после изменений зависимостей/критичной логики. citeturn4search0turn4search1  

**HTTP / runtime**
- На входе ограничены headers/body; нет неограниченного `ReadAll`. citeturn10view3turn10view4  
- На сервере выставлены явные timeouts (`ReadHeaderTimeout` минимум), и есть graceful shutdown через `Server.Shutdown`. citeturn10view2turn10view5  
- Исходящие вызовы используют переиспользуемый `http.Client` и не остаются без таймаутов. citeturn17view1  

**Observability**
- Логи структурированы (`slog`), поля стабильны, не содержат секретов. citeturn3search0turn8search0  
- Метрики не содержат high-cardinality labels; naming соответствует практикам (суффиксы, единицы, bounded labels). citeturn16search0  
- Если включён OpenTelemetry: контекст пропагируется, атрибуты следуют semantic conventions. citeturn1search5turn16search1  

**Caching / Redis**
- Ключи строятся через единый key-builder, содержат namespace + tenant + version. citeturn5search4turn19view1  
- Нет `KEYS` в runtime; итерации (если есть) — `SCAN`. citeturn16search7turn5search0  
- Для кэшируемых значений задан TTL, и поведение при ошибке кэша явно описано (fail-open/fail-closed). citeturn19view1turn20view1  
- Для hot keys учтён stampede (singleflight/locking) и есть лимиты размеров payload. citeturn11search1turn11search3  

## Что из результата нужно оформить отдельными файлами в template repo

Ниже — «карта» файлов, которые делают шаблон пригодным для работы с LLM: модель получает источник правды и перестаёт угадывать.

**Документы-стандарты**
- `docs/engineering/standard.md` — то, что вы читаете сейчас (адаптировать под ваш контекст), как норматив. citeturn0search0turn0search4turn18view0turn20view2turn8search7  
- `docs/engineering/decisions.md` или `docs/adr/` — ADR-ы: toolchain policy, transport choice, observability, cache policy. (Рекомендация опирается на необходимость фиксировать спорные решения; сами механизмы опираются на источники ниже.) citeturn7search0turn16search1turn20view2  
- `docs/security/baseline.md` — маппинг на ASVS + применимые OWASP Cheat Sheets (логирование, error handling, secrets). citeturn0search2turn8search0turn8search7turn8search2  
- `docs/observability.md` — OpenTelemetry setup, propagation, семантические конвенции, правила метрик/лейблов. citeturn1search5turn1search2turn16search1turn16search0  
- `docs/cache.md` — **implementation guide по ключам и значениям кэша** (см. следующий раздел). citeturn19view2turn20view2turn12view4  
- `docs/api.md` — OpenAPI (если HTTP) и/или gRPC health checking, правила версионирования API. citeturn2search7turn4search3  
- `docs/llm/INSTRUCTIONS.md` — MUST/SHOULD/NEVER, «как работать модели в этом репо». citeturn18view0turn14search0turn16search7turn8search2  
- `docs/llm/PROMPT_PREFIX.md` — короткий копипаст‑префикс для ChatGPT/Codex/Claude Code, включающий ключевые ограничения (timeouts, context, SQL parameterization, cache key schema). citeturn18view0turn10view4turn8search5turn19view2  

**Repo conventions / tooling**
- `go.mod` с `go` и `toolchain` согласно policy + отдельный ADR про обновления. citeturn7search0turn7search1turn6view2  
- `.golangci.yml` (если выбираете golangci-lint как агрегатор) + описание, почему и какие линтеры. citeturn14search3turn14search7  
- `.github/workflows/ci.yml` (или аналог) с этапами: gofmt-check, go test, go vet, govulncheck. citeturn14search15turn14search1turn4search0  
- `Dockerfile`, `docker-compose.yml` (для локального Postgres/Redis/OTel collector) и `deploy/` (k8s manifests/helm) с probes. citeturn4search2turn20view1  

## Исследование подтемы: key design, serialization, memory discipline

Этот раздел — то, что вы просили как «добавку к общему префиксу» и как практический implementation guide для template, особенно для cache-backed data access logic.

### Cache key design и данные в кеше

**Цели дизайна ключей**
- уникальность и отсутствие коллизий между сервисами/окружениями/тенантами;
- управляемость: возможность выборочной инвалидации «по префиксу» (через `SCAN MATCH`), без `KEYS`;
- безопасность multi-tenant: исключить чтение данных чужого тенанта из-за случайного совпадения ключа;
- контроль кардинальности ключей (не создать миллионы уникальных ключей из-за параметров запросов);
- возможность безопасной эволюции схемы (versioning). citeturn19view1turn5search0turn16search7  

**Рекомендованный key schema (template default)**
Формат (ASCII, lower-case, `:` как разделитель):

`{svc}:{env}:{dataset}:v{keyver}:tenant:{tenantID}:{entity}:{id}:{qualifiers...}`

Примеры:
- `billing:prod:widgets:v3:tenant:t42:widget:123`
- `billing:prod:widgets:v3:tenant:t42:widget:123:lang:ru`
- `billing:prod:search:v2:tenant:t42:q:sha1:<hash>` (когда часть ключа потенциально длинная)

Почему так:
- Redis прямо рекомендует «придерживаться схемы» и приводит `object-type:id` как хорошую идею, а также описывает баланс читаемости и длины ключа. citeturn19view1turn5search4  
- Redis предупреждает, что **очень длинные ключи — плохая идея** и что большие смысловые части лучше хешировать (например SHA1) из соображений памяти/полосы и стоимости сравнений ключей. citeturn19view2  

**Namespacing**
- MUST: первые сегменты — `svc` и `env`, чтобы исключить «перетекание» между окружениями и упростить операционные процедуры (миграции, чистки, анализ). Практика namespacing с `:` прямо упоминается как соглашение в материалах Redis. citeturn5search4  
- SHOULD: `dataset` (или bounded domain) — фиксированное имя набора ключей (например `sessions`, `widgets`, `ratelimit`), чтобы `SCAN MATCH svc:env:dataset:*` был безопасным и предсказуемым. citeturn5search0turn19view1  

**Tenant safety**
- MUST: включать tenant идентификатор в ключ (или иметь жёсткое архитектурное решение «тенанты физически разделены»). Поскольку Redis оперирует ключами как строками (key/value pairs), отсутствие tenant в ключе — прямой путь к случайным коллизиям на уровне приложения. citeturn16search3  
- MUST: tenantID должен быть **нормализован** (ограниченный алфавит/длина) или hashed, если он может быть произвольной строкой. Обоснование — те же причины, что и для длинных ключей: стоимость сравнения, память, переносимость. citeturn19view2  

**Versioning**
Два уровня версионирования, которые стоит различать в документах:
- `keyver`: версия **семантики ключа** (как мы адресуем объект). Меняется, если меняется разбиение на сегменты или смысл, влияющий на уникальность.
- `valver`: версия **схемы значения** (payload). Может быть включена в ключ или в сам payload (например, поле `schema_version`).  

Практический default:
- включать `v{keyver}` в ключ всегда;
- `valver` — включать в ключ, если сериализация не self-describing или миграция сложная; иначе хранить внутри значения.

Эта дисциплина нужна из-за того, что Redis рекомендует «stick with a schema», а без версий любая эволюция схемы превращается в silent data corruption (LLM особенно часто «просто меняет struct»). citeturn19view1  

**Ограничения по длине и размеру (guardrails для template)**
Redis допускает очень большие верхние границы (максимальный размер ключа 512MB; bulk string по умолчанию ограничен 512MB). Это технические лимиты, а не рекомендация. citeturn19view2turn11search3  

Boring defaults для шаблона (как **защитные ограничения**, менять только осознанно через конфиг + ADR):
- лимит длины ключа: **≤ 256 байт** (после форматирования всех сегментов). При превышении — хешировать длинные сегменты (обычно query/filters). Обоснование: Redis отдельно предупреждает про «very long keys» и стоимость. citeturn19view2  
- лимит размера значения: **≤ 1 MiB** (после сериализации; до компрессии). Больше — почти всегда признак неправильного дизайна (лучше кэшировать срез/ID и догружать). Технический верхний предел Redis намного выше, но operationally это защита от давления на память и эвикшена. citeturn11search3turn12view4  
- лимит количества ключей в «одном классе» (dataset): задаётся через бюджет памяти и expected cardinality; при наличии риска неконтролируемой кардинальности — вводить дополнительную агрегацию (см. ниже «hash packing»). citeturn12view4turn20view1  

**TTL и политика eviction**
- Redis описывает key expiration (TTL) как базовую возможность и отмечает, что TTL реплицируется/персистится и что есть команды для задания TTL. citeturn19view1  
- Redis eviction политика выбирается через `maxmemory-policy`; Redis перечисляет доступные политики и даёт rule-of-thumb: `allkeys-lru` — хороший дефолт, если нет причин предпочесть другое, а `volatile-ttl` полезен, если TTL используется как «подсказка» для eviction. citeturn20view1turn20view2  

Template policy:
- MUST: для **кэш-данных** ставить TTL всегда (исключение возможно только при явном ADR «это не кэш, это data store»). citeturn19view1turn20view1  
- MUST: выбрать и задокументировать `maxmemory-policy` на окружение. Default: `allkeys-lru`. citeturn20view2  
- SHOULD: использовать TTL jitter (случайная добавка), чтобы избежать массового истечения множества ключей в одну секунду (эффект stampede). Это не «стандарт Redis», но практический default для микросервисного кэша; для LLM важно иметь явный паттерн.  
- MUST (LLM-ошибка по умолчанию): не assume, что ключи «исчезают ровно по TTL». Redis имеет алгоритм истечения сроков (expiration algorithm), который пытается делать это наименее дорогим способом; приложения не должны строить корректность на точном времени удаления. citeturn5search14turn19view1  

**Eviction, expiration и события**
- Если вам нужно реагировать на `expired`/`evicted` (например, для метрик или внешней синхронизации), Redis keyspace notifications умеют генерировать события `expired` и `evicted`. В шаблоне это должно быть **выключено по умолчанию** и включаться отдельно, т.к. добавляет сложность эксплуатации. citeturn5search2turn20view1  

**Cardinality control и “packing”**
Когда однотипных маленьких ключей становится очень много, память тратится не только на значения, но и на overhead ключей/объектов. Redis официально предлагает при возможности **использовать hashes**, т.к. «small hashes encoded in a very small space» и показывает приёмы упаковки keyspace в hash для memory efficiency. citeturn12view4  

Template рекомендации:
- SHOULD: для «миллионов маленьких объектов» рассмотреть стратегию packing в `HASH` (например, один hash на префикс/шард и поля внутри), если:
  - TTL применяется одинаково ко всему набору (потому что TTL на уровне hash-поля отсутствует; TTL только на ключе). citeturn12view4  
  - операциям нужен быстрый доступ по id, а не сложные выборки.  
- MUST: если используете packing, документировать:  
  - как шардируется keyspace;  
  - какие ключи hash’ей и какие поля;  
  - как управлять TTL и инвалидацией. citeturn12view4turn19view1  

**Запреты на iteration**
- MUST: не использовать `KEYS` в runtime. Redis явно говорит «don’t use KEYS in your regular application code». citeturn16search7  
- MUST: если нужны операции «по префиксу», использовать `SCAN` из-за инкрементальности и меньшего риска блокировок; Redis прямо отмечает, что SCAN «can be used in production without downside of commands like KEYS». citeturn5search0  

### Value serialization format и совместимость

**JSON (default для внешнего API)**
- `encoding/json` описывает security considerations: разные парсеры могут интерпретировать один и тот же JSON по-разному; это важно для межсистемных контрактов. citeturn15view1  
- `Unmarshal` описывает обработку дублирующихся ключей (поздние могут заменять ранние), что может быть неожиданно при проверках подписи/канонизации. citeturn15view2  
- `Decoder.DisallowUnknownFields` помогает делать strict decoding на входе API. citeturn15view4  

Template policy:
- MUST (HTTP вход): strict decode (DisallowUnknownFields) для команд/мутаций, чтобы отсекать опечатки и неожиданные поля. citeturn15view4  
- SHOULD (кэш/межсервис): допускается более мягкая политика (игнор неизвестных полей) для облегчения rollouts. Это спорно и должно фиксироваться ADR: strictness повышает качество контрактов, но усложняет backward compatibility.

**Protobuf (рекомендуемый default для внутренних интерфейсов, если выбран gRPC)**
- Proto3 guide задаёт правила схем и генерации. citeturn9search2  
- Для взаимодействия с JSON-миром есть canonical ProtoJSON mapping. citeturn9search10  
- grpc-gateway может генерировать reverse-proxy HTTP→gRPC, но сам проект предупреждает о накладных расходах парсинга JSON↔protobuf. citeturn9search3turn9search17  

Template policy:
- SHOULD: если сервис internal-only и latency/CPU существенны — предпочесть gRPC+Protobuf, а OpenAPI/JSON — через generated слой только при необходимости. citeturn9search3turn9search17  
- MUST: любые изменения `.proto`/JSON контрактов сопровождать versioning правилами (в docs) и тестами совместимости.

### Compression trade-offs

Redis хранит значения как bytes (bulk string) с большим техническим лимитом (до сотен мегабайт по умолчанию), но это не «норма» для микросервиса. citeturn11search3  

Template policy (boring defaults):
- SHOULD: включать компрессию только после измерений и только для значений выше заданного порога (например, 4–16 KiB), т.к. на маленьких payload overhead и CPU часто перекрывают выгоду.  
- MUST: если компрессия используется, формат должен быть self-describing (например, префикс/магическое число/версия), чтобы можно было безопасно менять алгоритм и не ломать decode.  
- MUST: лимитировать decompressed size при распаковке (защита от zip-bomb логики), аналогично `MaxBytesReader` для HTTP. citeturn10view4  

### Memory discipline: Go runtime + Redis memory behavior

**Go: профилирование и дисциплина аллокаций**
- Go предоставляет диагностику через профили `cpu`, `heap`, `goroutine` и др.; heap profile предназначен для мониторинга аллокаций и утечек. citeturn9search0  
- Официальный блог по pprof описывает использование `go tool pprof` и базовые команды анализа. citeturn9search1  
- GC guide объясняет, как понимать стоимость GC и как улучшать ресурсную эффективность (ключевой практический вывод: снижать давление на heap/аллокации). citeturn13search7  
- `sync.Pool` предназначен для кэширования временных объектов и может очищаться GC «в любой момент», на него нельзя полагаться как на постоянное хранилище. citeturn13search3  
- `bytes.Buffer.Reset()` сохраняет underlying storage для переиспользования; при этом `Bytes()` возвращает slice, который алиасит буфер до следующей модификации (важно для avoiding data races и неожиданных мутаций). citeturn13search1turn13search9  

Template policy:
- MUST: не вводить `sync.Pool` без измерений (pprof/bench) и без документации, что пул — для краткоживущих объектов, и что объект надо reset’ить перед return/reuse. citeturn13search3turn9search0  
- SHOULD: для сериализации/копирования данных выбирать streaming подходы (Encoder/Decoder), и избегать лишних `[]byte↔string` конверсий и `ReadAll` там, где можно. citeturn15view3turn10view4  
- MUST: в шаблоне иметь «профилирование включаемое конфигом» (например, отдельный порт/эндпоинт в dev/stage), и документацию как снять CPU/heap профили. citeturn9search0turn9search1  

**Redis: память, фрагментация, RSS**
- Redis документирует, что удаление ключей не гарантирует возврат памяти ОС: allocator может не уметь «легко» вернуть страницы, поэтому RSS может оставаться на пиковом уровне; планировать надо по peak usage. citeturn12view4  
- Redis документирует memory optimization стратегии, включая special encodings и использование hashes для экономии памяти; это прямо связано с дизайном ключей/значений и кардинальностью. citeturn12view4  
- Eviction политика и `maxmemory` — часть управления памятью; Redis описывает механизмы и перечисляет политики. citeturn20view1turn20view2  

Template policy:
- MUST: считать кэш «утилитарным»: корректность бизнес-логики не должна зависеть от того, что ключ обязательно в кеше (кроме специальных случаев типа rate-limit/locks, которые требуют отдельного ADR). citeturn20view1  
- MUST: иметь метрики/логи cache hit/miss/error и алерты на eviction/rejected writes (особенно для `noeviction`/volatile policies). Redis рекомендует смотреть `INFO`/commandstats для анализа поведения при maxmemory. citeturn20view1  
- SHOULD: при подозрении на memory bloat использовать подходы Redis к memory optimization (hashes и т.п.) и пересматривать key/value дизайн, а не «просто увеличить память». citeturn12view4  

### Добавка к общему префиксу LLM: кэш и ключи

Текст ниже — копипаст‑заготовка в `docs/llm/PROMPT_PREFIX.md`, чтобы любая модель генерировала cache-backed логику предсказуемо:

```text
CACHE RULES (MUST):
- Use Redis keys with schema: {svc}:{env}:{dataset}:v{keyver}:tenant:{tenantID}:{entity}:{id}:{...}.
- Include tenant safety (tenantID) and bump v{keyver} on any key semantics change.
- Do not use Redis KEYS in runtime code; use SCAN for iteration.
- All cache entries MUST have explicit TTL, unless an ADR says this is not a cache.
- Enforce guardrails: key length <= 256 bytes; value size <= 1 MiB (serialized). Hash long segments (e.g., query) instead of embedding them.
- Prevent cache stampede on hot keys using singleflight (or equivalent).
- Cache failures should not crash the request path unless explicitly required (document fail-open vs fail-closed).
- Never cache secrets; avoid caching PII unless explicitly allowed and documented.
```

Эти правила опираются на: запрет/опасность `KEYS` в регулярном коде и предпочтение `SCAN`; рекомендации Redis придерживаться схемы ключей и предупреждения про длинные ключи; эксплуатационную модель `maxmemory`+eviction; и наличие singleflight как механизма suppress duplicate work. citeturn16search7turn5search0turn19view2turn20view2turn11search1