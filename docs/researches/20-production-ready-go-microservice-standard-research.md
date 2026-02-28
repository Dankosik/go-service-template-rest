# Production-ready Go микросервис: engineering standard и LLM-инструкции для template-repo

## Scope

Этот стандарт и template-repo подходят, когда вы строите **greenfield** микросервис на Go, который должен быть «сразу production-ready» после clone: предсказуемая структура проекта, безопасные дефолты, наблюдаемость, ясные границы транзакций и формализованные правила для генерации кода LLM (чтобы минимизировать догадки модели). Подход особенно полезен в среде контейнерной оркестрации вроде entity["organization","Kubernetes","container orchestration project"], где важны пробки (liveness/readiness/startup), корректное завершение (SIGTERM → graceful shutdown), внешняя конфигурация и однотипные практики между сервисами. citeturn7search2turn7search20turn7search8turn6search34

Подход рассчитан на «boring, battle‑tested defaults»: официальные инструменты Go, зависимостная гигиена (Go modules + checksum DB), базовые security‑практики по entity["organization","OWASP","security nonprofit"] (ASVS/cheat sheets), стандартная observability‑интеграция и устойчивое поведение при сбоях (timeouts/retries/idempotency), как рекомендуют крупные практики вроде entity["company","Amazon Web Services","cloud provider"] и entity["company","Google","technology company"] (Builders’ Library / SRE‑guidance). citeturn10search9turn10search5turn10search0turn0search2turn8search6turn8search2turn8search3

Не применяйте этот стандарт «как есть», если:

- Вам нужен **монолит** или библиотека (а не самостоятельный сервис), либо сервис не предполагает типичные production‑аспекты (health endpoints, метрики, трассировка, деплой в k8s) — часть инфраструктурных требований будет избыточной. citeturn7search2turn7search20  
- У вас есть требования к **строгой кросс‑сервисной атомарности** и вы сознательно внедряете распределённые транзакции/2PC (XA/координатор). Этот стандарт, наоборот, предполагает, что ACID «через несколько сервисов» обычно не является рабочим базовым допущением в микросервисах, и ориентирует на saga/outbox/eventual consistency. citeturn8search20turn2view0turn8search4  
- У вас уже есть платформа/платформенная команда со своим opinionated stack (service mesh, стандартизированная authN/authZ, централизованные библиотеки, корпоративная схема событий, единый workflow engine). Тогда используйте этот документ как шаблон для адаптации, а не как истину. citeturn5search0turn5search1turn11search31

## Recommended defaults для greenfield template

Ниже — набор дефолтов, который можно почти напрямую оформить как `docs/engineering-standard.md` + `docs/llm-instructions.md` + repo conventions (CI, линтеры, каталоги, шаблоны PR). Все дефолты сделаны так, чтобы LLM могла генерировать код **идиоматично и безопасно**, опираясь на явные решения и встроенные «rails».

**Версия Go и политика обновлений**

- Дефолт для новых сервисов: **Go 1.26** (актуальный релиз на февраль 2026). citeturn6search0turn6search3  
- Закрепляйте версию в `go.mod` (директива `go 1.26`) и документируйте поддержку: Go поддерживает релиз, пока есть **две более новые мажорные версии**; исправления безопасности бекпортятся в поддерживаемые ветки. citeturn6search1turn6search14  
- Обновления Go: планируйте согласно циклу релизов (примерно раз в 6 месяцев). citeturn6search4  

**Структура репозитория и packaging**

Дефолтная структура должна следовать официальной логике модулей и пакетов: каталог = пакет, `internal/` используется для приватных пакетов и явно поддержан официальной документацией по layout. citeturn0search1turn0search26turn0search38

Рекомендуемая структура template-repo (минимально opinionated, но с сильными границами):

- `cmd/<service>/main.go` — точка входа, сборка wiring’а (config, logger, db, http server, background workers). citeturn0search1  
- `internal/app/` — use-cases (бизнес‑операции), orchestration, транзакционные границы. citeturn8search4turn8search20  
- `internal/httpapi/` — HTTP handlers, DTO, валидация, middleware, error mapping. citeturn5search1turn12search3  
- `internal/store/` — репозитории, транзакции, миграции. (DB конкретика вынесена за интерфейсы, чтобы LLM не «изобретала» доступ к данным в handler’ах.) citeturn8search4  
- `internal/observability/` — трассировка/метрики/лог‑корреляция, семантические атрибуты. citeturn11search31turn11search3turn11search13  
- `internal/outbox/` и `internal/inbox/` — опционально, но **включено по умолчанию** как «правильный путь» для событий/команд между сервисами. citeturn1search1turn8search13turn8search1turn13search0  
- `migrations/` — SQL миграции (инструмент фиксируется и описывается в стандарте; LLM не должна выбирать его сама). citeturn8search4  
- `docs/` — архитектурные решения (ADR), LLM‑инструкции, чек‑листы, дизайн multi-step workflows. Требование наличия чек‑листов/гайдов для разработчиков хорошо согласуется с идеей ASVS о доступных secure coding требованиях/политиках. citeturn0search8turn0search2  

**Logging**

- Дефолт: структурные логи через стандартный `log/slog` (встроен в Go 1.21+), формат — JSON в stdout (для контейнеров). citeturn6search2  
- Логи должны быть пригодны для security‑аудита и расследований; используйте практики безопасного логирования (контекст, события безопасности, отсутствие секретов). citeturn5search3turn5search9turn12search1  

**Tracing / context propagation / стандартизация атрибутов**

- Дефолт: distributed tracing на базе entity["organization","Cloud Native Computing Foundation","linux foundation project"] / entity["organization","OpenTelemetry","cncf observability project"]; включить базовую ручную инструментализацию + middleware-инstrumentation. citeturn7search3turn11search1turn7search13  
- Пропагация контекста: используйте стандарт entity["organization","W3C","world wide web consortium"] Trace Context (`traceparent`/`tracestate`) и W3C Baggage через propagators API; это снижает «зоопарк» форматов между языками и вендорами. citeturn11search0turn11search13turn11search5  
- Соблюдайте OpenTelemetry Semantic Conventions (в частности для HTTP spans), чтобы атрибуты были единообразны и пригодны для готовых дашбордов/алёртов. citeturn11search3turn11search31  
- В Go коде это должно опираться на `context.Context` и его корректную передачу через границы API; это прямо предписано документацией `context`. citeturn4search1turn4search8turn4search2  

**Metrics**

- Дефолт: метрики в стиле entity["organization","Prometheus","monitoring system"] (или совместимые) + строгий контроль label-cardinality. citeturn11search2turn11search6  
- Запрещены метки с высокой кардинальностью (user_id, request_id и т.п.) из‑за взрывного роста time series и затрат. citeturn11search6turn11search2  

**HTTP/API security и ограничения ресурсов**

- REST/HTTP API должны обслуживаться только через HTTPS (на уровне ingress/mesh или самим сервисом — но контракт «только HTTPS» фиксируется). citeturn5search1  
- Обязательны лимиты на ресурсы: rate limiting/quotas, ограничения размеров, таймауты, чтобы закрывать класс API4:2023 (Unrestricted Resource Consumption). citeturn12search3turn12search7  
- Таймауты HTTP сервера задавайте явно (ReadHeaderTimeout/ReadTimeout/WriteTimeout/IdleTimeout). Значения зависят от сценариев (маленькие JSON vs стриминг), поэтому дефолты должны быть конфигурируемы; но отсутствие таймаутов — плохой дефолт для «интернета». citeturn12search35turn12search2  

**Configuration & secrets**

- Конфигурация — через env vars согласно 12‑factor «config in the environment». citeturn12search0  
- Секреты — через механизмы оркестратора (Kubernetes Secrets), не в коде и не в образе. citeturn12search1  

**Graceful shutdown и k8s probes**

- Реализуйте graceful shutdown через `http.Server.Shutdown(ctx)`, учитывая дедлайны контекста. citeturn6search34  
- Добавьте endpoints для readiness/liveness/startup и правильно их используйте (liveness может ловить deadlock; readiness управляет приёмом трафика). citeturn7search20turn7search2  
- Если используется `PreStop`, помните: hook не async; он должен завершиться до SIGTERM, и входит в общий `terminationGracePeriodSeconds`. Это влияет на дизайн shutdown (сначала «снять readiness», потом дожать in-flight, потом остановить воркеры). citeturn7search8  

**Dependency & supply chain security**

- Используйте Go modules, `go.sum`, checksum database и proxy по умолчанию; это часть модели безопасности экосистемы. citeturn10search9turn10search1turn10search5  
- В CI включите `govulncheck` как «low-noise» сканер уязвимостей, который репортит только реально достижимые вызовы (call graph), и используйте рекомендации Go security team. citeturn10search0turn10search8turn10search26  

**Data consistency дефолты (ключевая часть template)**

1) **Транзакционная граница — локальная**: один сервис, одна база, одна ACID транзакция внутри конкретного сервиса. Это упрощает проектирование и масштабирование, но означает, что кросс‑сервисная консистентность должна проектироваться осознанно (eventual consistency). citeturn8search4turn8search20  

2) Для публикации событий/команд из write‑потока используйте **Transactional Outbox**: записывайте событие в outbox‑таблицу *в той же транзакции*, что и изменение бизнес‑данных, а отправку в брокер делайте отдельным relay‑процессом/воркером (polling или CDC). Это закрывает «dual write» проблему и снижает шанс рассинхронизации между DB и message bus. citeturn1search1turn8search1turn8search13turn8search8  

3) Для обработки входящих событий/команд проектируйте **идемпотентных потребителей** (Idempotent Consumer/Inbox): большинство брокеров дают at‑least‑once доставку, значит дубль сообщения — нормальный режим, а не исключение. citeturn13search0turn13search1turn13search6  

4) Multi-step бизнес‑флоу между сервисами моделируйте через **Saga**: последовательность локальных транзакций + сообщения/события + compensation. При проектировании выделяйте pivot step («point of no return») и делайте retryable steps идемпотентными, чтобы saga могла «дотолкаться» до финального согласованного состояния. citeturn8search20turn2view1  

5) **2PC по умолчанию избегайте**: даже на уровне одной СУБД (пример: `PREPARE TRANSACTION` в PostgreSQL) документация прямо говорит, что это механизм для внешнего transaction manager и что оставлять prepared transactions опасно (holding locks, проблемы с VACUUM и вплоть до остановки из‑за transaction ID wraparound); если нет transaction manager — лучше выключать `max_prepared_transactions`. citeturn2view0  

6) Для внешних «мутаций» (обычно POST/PATCH) используйте **idempotency keys**: семантика идемпотентности в HTTP отдельно описана в RFC 9110, а для `Idempotency-Key` существует draft стандарта entity["organization","IETF","internet standards body"] (пока не RFC), который отлично подходит как межкомандный контракт. citeturn3view0turn1search12  

7) Если чтения усложняются (отчёты, поисковые запросы, read‑heavy): применяйте **CQRS/read models** — отделяйте модель записи от модели чтения и обновляйте read модель асинхронно через события/outbox. Но включайте CQRS только при реальной потребности: это осознанное усложнение ради производительности/масштабируемости/безопасности. citeturn7search0turn8search4  

image_group{"layout":"carousel","aspect_ratio":"16:9","query":["transactional outbox pattern diagram","saga orchestration vs choreography diagram","CQRS read model diagram","idempotency key request flow diagram"],"num_per_query":1}

## Decision matrix / trade-offs

Ниже — матрицы решений, которые стоит закрепить в template как «стандартные выборы» + «когда отклоняться», чтобы LLM не спорила с платформой, а следовала правилам.

| Тема | Boring default | Когда менять | Основные trade-offs / риски |
|---|---|---|---|
| Cross-service consistency | Saga + local tx + eventual consistency | Только при редкой необходимости строгой атомарности и наличии transaction manager | Saga требует явного проектирования компенсаций и диагностики, но масштабируется лучше, чем попытки ACID через сервисы. 2PC/Prepared tx опасны при долгом удержании и требуют внешнего менеджера. citeturn8search20turn2view0turn8search4 |
| Saga coordination | Orchestration по умолчанию для сложных флоу | Choreography — если поток простой, команды автономны и рост участников ожидается | Orchestration упрощает наблюдаемость/таймауты/ретраи, но добавляет центральный компонент; choreography повышает автономность, но сложнее дебажить каскады событий. citeturn1search13turn2view2turn8search20 |
| Emitting events | Transactional Outbox | CDC‑relay (Debezium) vs polling‑relay зависит от инфраструктуры | Outbox устраняет dual writes. CDC снижает нагрузку polling’а и улучшает latency, но добавляет инфраструктуру (connectors) и операционные процессы. citeturn1search1turn8search1turn8search13 |
| Consuming events | Idempotent Consumer + inbox/dedup | Если операция «естественно идемпотентна» (set‑value), можно без inbox, но это должно быть доказуемо | At‑least‑once доставка означает дубли. Azure/AWS прямо рекомендуют идемпотентную обработку. Inbox добавляет хранение processed IDs и чистку. citeturn13search0turn13search1turn13search6 |
| API idempotency | Idempotency-Key для POST/PATCH | Если мутация реально идемпотентна по дизайну (PUT как set), можно без ключа | RFC 9110 объясняет, почему idempotent методы можно безопасно ретраить; draft Idempotency-Key делает не‑идемпотентные операции fault‑tolerant, но требует серверного хранилища результатов/статуса. citeturn3view0turn1search12 |
| Distributed locking | Избегать для корректности; предпочитать DB constraints/optimistic concurrency | Использовать только для efficiency‑координации, либо с fencing tokens и пониманием модели | Распределённые lock’и сложны: Redis Redlock — спорная тема (есть публичная критика), Redis просит анализ/feedback, etcd предупреждает о «неочевидных свойствах». Advisory locks в PostgreSQL «advisory» и требуют дисциплины приложения. citeturn9search0turn9search1turn9search6turn9search2 |
| Observability | Traces: OpenTelemetry + W3C Trace Context; Metrics: Prometheus‑style + low cardinality; Logs: slog | Если платформа стандартизировала другое (например, OTLP‑only метрики), меняйте пакетно | W3C Trace Context — межвендорный формат. Prometheus предупреждает о label cardinality. OTel семконвенции дают единый словарь атрибутов. citeturn11search0turn11search2turn11search3turn6search2 |
| Go toolchain security | go modules + checksum db + govulncheck | В офлайновых/закрытых сетях — зеркалирование proxy/sumdb | Go proxy/sumdb и прозрачный лог повышают устойчивость supply chain. govulncheck — «низкошумный» и привязан к реально достижимым вызовам. citeturn10search1turn10search5turn10search0turn10search8 |

Отдельно: **timeouts/retries** — это не «мелочь», а основа устойчивости. Ретраи без идемпотентности создают дубли и side‑effects; ретраи без jitter и бюджета усиливают каскадные отказы. citeturn8search2turn8search6turn8search3turn3view0

## Набор правил MUST / SHOULD / NEVER для LLM

Этот раздел предназначен для прямого переноса в LLM‑instruction файл (например, `docs/llm-instructions.md` и/или инструкционные файлы для конкретных агентов). Формулировки сделаны так, чтобы модель действовала как «сильный инженер, но без права на догадки».

### MUST

- MUST следовать официальной структуре Go module/package: не создавать пакеты только «для красоты», помнить что каталог = пакет; приватную реализацию держать в `internal/`. citeturn0search1turn0search26turn0search38  
- MUST форматировать код `gofmt`, не спорить со стилем; при сомнениях ориентироваться на Go Code Review Comments. citeturn0search0turn0search29  
- MUST протаскивать `context.Context` через все I/O‑границы: входящий запрос создаёт/имеет контекст, исходящие вызовы принимают контекст; не терять cancellation и deadlines. citeturn4search1turn4search2turn4search8  
- MUST оборачивать ошибки стандартным способом (`fmt.Errorf(... %w ...)`) и сохранять возможность `errors.Is/As`, а не сравнивать строки. citeturn4search0turn4search4turn4search12  
- MUST реализовать graceful shutdown (`http.Server.Shutdown`) и учитывать дедлайн shutdown‑контекста. citeturn6search34  
- MUST добавить readiness/liveness/startup endpoints и документацию по их смыслу. citeturn7search2turn7search20  
- MUST хранить конфигурацию вне кода (env vars), секреты — через оркестратор. citeturn12search0turn12search1  
- MUST включить supply chain и dependency security: `go.sum` не править вручную; использовать checksum DB/proxy по дефолту; CI должен запускать `govulncheck`. citeturn10search9turn10search1turn10search0  
- MUST обеспечивать консистентность данных по дефолту через local transactions + eventual consistency; межсервисные побочные эффекты (публикация событий/команд) — через Transactional Outbox. citeturn8search4turn1search1turn8search1  
- MUST проектировать обработчики сообщений идемпотентными (at‑least‑once доставка означает дубли), при необходимости — inbox/dedup таблица. citeturn13search0turn13search6turn13search1  
- MUST проектировать multi-step flow как saga: явно описать шаги, события/команды, компенсации, pivot и retryable‑шаги. citeturn8search20turn2view1  
- MUST ограничивать ресурсы API (rate limits, quotas, size/timeouts) против класса API4:2023. citeturn12search3turn12search7  
- MUST избегать метрик с высокой кардинальностью (labels), иначе это создаёт стоимость и риск деградации. citeturn11search2turn11search6  
- MUST использовать W3C Trace Context и OTel propagators/semconv там, где это применимо. citeturn11search0turn11search13turn11search3  

### SHOULD

- SHOULD предпочитать стандартную библиотеку и минимальный набор внешних зависимостей; каждую dependency фиксировать как decision в docs (чтобы LLM не «выбирала библиотеку»). citeturn6search1turn10search9  
- SHOULD добавлять тесты в идиоматичном стиле (table-driven; subtests), особенно для бизнес‑логики и edge cases. citeturn4search3turn4search15  
- SHOULD делать операции идемпотентными «по смыслу» (set‑семантика, уникальные ключи, upsert), чтобы ретраи были безопасны. citeturn8search2turn8search18turn3view0  
- SHOULD использовать bounded retries с backoff+jitter и retry budgets, иначе ретраи могут усилить каскадные отказы. citeturn8search6turn8search3  
- SHOULD документировать наблюдаемость: какие метрики/трейсы/логи ожидаются и какие алёрты подразумеваются. citeturn11search31turn5search3  
- SHOULD проектировать read models/CQRS только при наличии реальной причины (read-heavy, сложные запросы), иначе избегать «архитектурной моды». citeturn7search0turn8search4  

### NEVER

- NEVER делать dual writes (пишем в DB и «сразу» публикуем/зовём внешний сервис без атомарной связки). Вместо этого — outbox/CDC. citeturn1search1turn8search13turn8search4  
- NEVER использовать 2PC/`PREPARE TRANSACTION` в прикладном коде без внешнего transaction manager; не оставлять prepared transactions «на потом». citeturn2view0  
- NEVER пытаться достигнуть «exactly once» магией. Считайте, что дубли/повторы возможны: проектируйте идемпотентность и дедуп на приёмнике. citeturn13search0turn13search6turn8search18  
- NEVER использовать распределённые locks как основу корректности бизнес‑инвариантов. Если всё же нужен lock — пояснить модель и риски (фактически ADR), иначе запрет. citeturn9search0turn9search6turn9search2  
- NEVER логировать секреты/PII и никогда не «дампить env» в логах в проде. citeturn5search3turn12search1  
- NEVER добавлять Prometheus labels с user_id/request_id и прочими unbounded значениями. citeturn11search6turn11search2  

## Concrete good / bad examples, где уместно — на Go

Примеры ниже рассчитаны на прямое включение в docs как «эталон». (Код — иллюстративный; конкретные пакеты/имена должны совпадать с вашим template‑repo.)

### Transactional Outbox: запись бизнес-данных + события в одной транзакции

**GOOD: одна локальная транзакция → изменение агрегата + запись в outbox**

```go
func (s *Service) CreateOrder(ctx context.Context, req CreateOrderRequest) (OrderID, error) {
	id := NewOrderID()

	event := OutboxMessage{
		ID:        NewMessageID(),
		Topic:     "orders.v1",
		Key:       id.String(),
		Type:      "OrderCreated",
		Payload:   mustJSON(OrderCreated{OrderID: id}),
		CreatedAt: time.Now().UTC(),
	}

	err := s.store.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		if err := s.store.Orders(tx).Insert(ctx, Order{ID: id, Status: "PENDING"}); err != nil {
			return fmt.Errorf("insert order: %w", err)
		}
		if err := s.store.Outbox(tx).Insert(ctx, event); err != nil {
			return fmt.Errorf("insert outbox: %w", err)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return id, nil
}
```

Почему это good: outbox паттерн специально создан, чтобы избегать рассинхронизации между «commit в DB» и «сообщение в брокер», а публикацию вынести в отдельный процесс/relay. citeturn1search1turn8search1turn8search13turn8search8

**BAD: dual write (DB commit отдельно, publish отдельно)**

```go
func (s *Service) CreateOrder(ctx context.Context, req CreateOrderRequest) (OrderID, error) {
	id := NewOrderID()

	if err := s.store.Orders(nil).Insert(ctx, Order{ID: id, Status: "PENDING"}); err != nil {
		return "", err
	}

	// ❌ publish может упасть, order уже создан -> рассинхронизация
	if err := s.bus.Publish(ctx, "orders.v1", OrderCreated{OrderID: id}); err != nil {
		return "", err
	}

	return id, nil
}
```

Это плохо ровно потому, что микросервисы требуют осознанного распространения обновлений между сервисами; «одно место истины» нарушается, и без outbox вы получаете eventual consistency без механизма гарантированной доставки. citeturn8search4turn1search1turn8search1

### Idempotency key для POST: хранение результата и повторяемость

**GOOD: сервер принимает Idempotency-Key и возвращает тот же результат при повторе**

```go
func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		http.Error(w, "missing Idempotency-Key", http.StatusBadRequest)
		return
	}

	// 1) try load cached response by key (within TTL policy)
	if cached, ok := h.idemStore.Get(ctx, key); ok {
		writeJSON(w, cached.StatusCode, cached.Body)
		return
	}

	// 2) execute business op
	resp, err := h.svc.CreatePayment(ctx, parseReq(r))
	if err != nil {
		// map errors -> proper HTTP code
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 3) store result for this key
	h.idemStore.Put(ctx, key, CachedResponse{StatusCode: http.StatusCreated, Body: mustJSON(resp)})

	writeJSON(w, http.StatusCreated, resp)
}
```

Почему это good: HTTP семантика различает идемпотентные методы (PUT/DELETE/SAFE) и предупреждает об автоповторах; для POST/PATCH нужна явная договорённость. Draft `Idempotency-Key` описывает именно этот механизм «сделать не‑идемпотентные методы fault‑tolerant», но это требует серверной памяти/хранилища и политики TTL. citeturn3view0turn1search12turn8search18

**BAD: «просто ретраим POST» без идемпотентности**

```go
// ❌ при сетевых сбоях клиент ретраит, создаются дубликаты платежей/заказов
resp, err := http.Post(url, "application/json", body)
```

Плохо, потому что ретраи — нормальная стратегия против transient failures, но без идемпотентности они превращаются в генератор дублей и side-effects. AWS прямо описывает, что ретраи безопасны, когда операции идемпотентны. citeturn8search2turn8search6

### Context propagation и отмена

**GOOD: use request context, propagate, respect deadlines**

```go
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, err := h.repo.GetUser(ctx, userIDFromURL(r))
	if err != nil {
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, user)
}
```

В `context` документации прямо сказано: входящие запросы должны создавать/нести контекст, исходящие вызовы должны принимать контекст; цепочка должна его протаскивать. citeturn4search1turn4search2

**BAD: handler создаёт Background(), убивая cancellation**

```go
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background() // ❌ игнорирует отмену клиента, дедлайны, трассировку
	user, _ := h.repo.GetUser(ctx, userIDFromURL(r))
	writeJSON(w, http.StatusOK, user)
}
```

Это нарушает смысл `context` и ломает cancellation/tracing/timeout semantics по всей цепочке. citeturn4search1turn4search8

### Ошибки: wrap с %w

**GOOD**

```go
if err := do(); err != nil {
	return fmt.Errorf("do something: %w", err)
}
```

%w — стандартный механизм wrapping, чтобы `errors.Is/As` продолжали работать. citeturn4search0turn4search4

**BAD**

```go
if err := do(); err != nil {
	return fmt.Errorf("do something: %v", err) // ❌ теряется unwrap-цепочка
}
```

Go ErrorValueFAQ объясняет, почему `%v` ломает идентичность/unwrap и почему `%w` предпочтителен для добавления контекста без разрушения обработки ошибок. citeturn4search12turn4search0

## Anti-patterns и типичные ошибки/hallucinations LLM

Ниже — «красные флаги», которые особенно часто возникают при LLM‑генерации. Их стоит зафиксировать как отдельный раздел docs и как автоматические проверки (линтеры/ревью).

**Dual writes и «синхронная атомарность на глаз»**  
LLM часто генерирует схемы «записали в свою БД → вызвали другой сервис → всё ок». В микросервисной архитектуре это приводит к рассинхронизации и сложной восстановляемости; вместо этого используйте outbox и обязательно проектируйте обработку дублей на стороне потребителя. citeturn8search4turn1search1turn13search0turn8search1  

**Псевдо‑exactly-once и игнорирование дублей**  
Модель может «обещать» exactly-once, не вводя дедуп/идемпотентность. Документы AWS/Azure прямо рекомендуют проектировать идемпотентную обработку, потому что at‑least‑once доставку и повторную обработку нужно считать ожидаемыми. citeturn13search6turn13search1turn8search18  

**Злоупотребление 2PC / PREPARE TRANSACTION**  
LLM может предложить 2PC как «простое решение консистентности». PostgreSQL документация предупреждает, что `PREPARE TRANSACTION` не для приложений; это инструмент внешнего transaction manager, и «prepared» состояния держат locks и мешают VACUUM, создавая операционный риск. citeturn2view0turn8search20  

**Distributed locks как «универсальная таблетка»**  
LLM часто предлагает Redis locks для борьбы с гонками, превращая локальную проблему в распределённую. Зафиксируйте правило: locks — не для корректности бизнес‑инвариантов. В экосистеме Redis есть публичный спор о безопасности Redlock, Redis сам просит комьюнити анализировать алгоритм; etcd прямо предупреждает о неочевидных свойствах distributed locks; PostgreSQL подчёркивает advisory‑характер advisory locks (приложение должно быть дисциплинированным). citeturn9search0turn9search1turn9search6turn9search2  

**Потеря context и «вечные горутины»**  
Паттерн: `context.Background()` внутри handler’ов/воркеров + goroutine без остановки. Это ломает cancellation, дедлайны и трассировку. Документация `context` требует протаскивания контекста через API границы. citeturn4search1turn4search8turn4search2  

**Непредсказуемые retries**  
LLM может вставить бесконечные ретраи или ретраи без jitter. Практики надёжности рекомендуют bounded retries, randomized backoff и retry budget, иначе ретраи усиливают каскадные отказы. citeturn8search6turn8search3  

**High-cardinality metrics**  
LLM любит добавлять label’ы «для удобства дебага» (user_id, order_id). Prometheus прямо предупреждает, что это резко увеличивает стоимость time series; такие данные должны быть в логах/трейсах, а не labels. citeturn11search2turn11search6  

**Секреты в логах и конфиге**  
LLM может добавить логирование всех env vars «для диагностики». В Kubernetes секреты часто доставляются через Secret объекты; логирование окружения может раскрыть чувствительные данные. Логи должны следовать security‑гайдам, а не быть дампом окружения. citeturn12search1turn5search3  

**«Неявные» решения по библиотекам/инфре**  
Частая hallucination: модель выбирает брокер, мигратор, роутер, ORM без указаний. Лечится только жёстким стандартом: перечислить и закрепить choices в repo и в инструкциях (LLM не выбирает стек). Это особенно важно для supply chain и повторяемости. citeturn10search9turn10search5turn10search0  

## Review checklist для PR/code review

Этот чек‑лист предназначен для `docs/review-checklist.md` и для PR template. Он написан так, чтобы reviewer мог быстро отлавливать ошибки, включая «LLM‑побочки».

**Build / test / style**

- Код отформатирован (`gofmt`), структура пакетов не «искусственная», `internal/` используется корректно. citeturn0search0turn0search1  
- Ошибки оборачиваются через `%w`, нет сравнения ошибок по строкам. citeturn4search0turn4search12  
- Есть тесты на бизнес‑логику и tricky cases (table-driven, subtests). citeturn4search3turn4search15  

**Runtime correctness**

- Все I/O функции принимают `context.Context`; handler использует `r.Context()`. citeturn4search1turn4search2  
- Реализован graceful shutdown (`Server.Shutdown`), и он учитывает поведение Kubernetes termination (preStop/terminationGracePeriod). citeturn6search34turn7search8  
- Прописаны HTTP server timeouts (как минимум ReadHeaderTimeout/IdleTimeout) или документировано, почему иначе. citeturn12search35turn12search2  

**Security**

- Нет утечек секретов в логах; конфигурация через env; секреты через Secrets. citeturn12search0turn12search1turn5search3  
- Учтены рекомендации OWASP по REST и ограничению ресурсов (rate limiting/quotas). citeturn5search1turn12search3turn5search2  
- В CI/локально прогнан `govulncheck` и зафиксированы действия по уязвимостям. citeturn10search0turn10search26  

**Observability**

- Трассировка не ломается: контекст не теряется; используется W3C Trace Context; следование semantic conventions для HTTP spans. citeturn11search0turn11search13turn11search3  
- Метрики не имеют high-cardinality labels; есть базовые RED/USE метрики (ошибки/латентность/throughput/ресурсы) и понятные имена. citeturn11search2turn11search6  
- Логи структурированы (slog), есть корреляция trace_id/request_id без раскрытия секретов. citeturn6search2turn5search3  

**Data consistency / workflow correctness (самое важное для микросервисов)**

- Границы транзакций локальные, нет попыток «сделать атомарно» через несколько сервисов прямыми вызовами. citeturn8search20turn8search4  
- Если сервис публикует события/команды: используется outbox (или документирован CDC‑вариант), нет dual write. citeturn1search1turn8search1turn8search13  
- Потребление сообщений идемпотентно (дедуп или естественная идемпотентность доказуема), обработка дублей продумана. citeturn13search0turn13search6  
- Multi-step flows оформлены как saga: определены шаги, компенсации, pivot, retryable этапы, timeouts, ретраи, восстановление. citeturn8search20turn2view1  
- Нет 2PC / prepared transactions в прикладном коде. citeturn2view0  
- Нет distributed locks «для корректности»; если lock есть — есть ADR с моделью, fencing tokens/ограничения и доказательство необходимости. citeturn9search0turn9search6  

## Что из результата нужно оформить отдельными файлами в template repo

Ниже — конкретный «file plan», который превращает этот report в набор документов и репо‑конвенций. Идея: LLM видит эти файлы и перестаёт «угадывать», потому что решения уже приняты и описаны.

**Документы в `docs/`**

- `docs/engineering-standard.md` — полный стандарт: структура репо, правила Go-кода (context, errors, shutdown), observability, security, CI. Основание: Go Code Review Comments, go.dev security/vuln, и т.д. citeturn0search0turn10search26turn10search8  
- `docs/llm-instructions.md` — MUST/SHOULD/NEVER правила из этого отчёта + «как работать с репо»: какие пакеты использовать, как добавлять эндпоинт, как делать миграцию, как писать тест, как делать outbox/inbox, какие запреты на догадки. (Этот файл — главный «анти‑hallucination rail».) citeturn0search1turn8search4turn13search0  
- `docs/architecture/data-consistency.md` — отдельный документ про local tx, outbox, saga (orchestration/choreography), компенсации, идемпотентность, CQRS/read models, reconciliation jobs, distributed locks и почему избегаем 2PC. citeturn8search4turn8search20turn1search1turn13search0turn2view0  
- `docs/architecture/observability.md` — стандарты трассировки/метрик/логов: W3C Trace Context, OTel semantic conventions, cardinality правила. citeturn11search0turn11search3turn11search2turn6search2  
- `docs/security.md` — минимальный security baseline: OWASP REST Security, OWASP API Top 10 (особенно API4), logging guidance, secrets handling, dependency scanning (govulncheck). citeturn5search1turn5search2turn12search3turn5search3turn10search0  
- `docs/pr-review-checklist.md` — чек-лист из раздела Review (можно дублировать в PR template). citeturn0search0turn10search0turn13search0  

**Repo conventions / meta**

- `.github/pull_request_template.md` — короткий чек-лист: tests, migrations, observability, data consistency, security. citeturn10search26turn13search0  
- `.github/workflows/ci.yml` — сборка, тесты, `go test`, `go vet`, линт (закреплённый), `govulncheck`. (Набор инструментов фиксируется, чтобы LLM не «пересобирала CI по настроению».) citeturn10search0turn0search0turn0search29  
- `go.mod` / `go.sum` — политика версий и зависимостей, checksum db по умолчанию. citeturn10search9turn10search5turn10search1  
- `deploy/` (или `k8s/`): пример Deployment/Service с probes и terminationGracePeriodSeconds + пояснения. citeturn7search2turn7search8turn7search20  
- `internal/outbox/README.md` и `internal/inbox/README.md` — краткие «как использовать» (для LLM и инженеров), со ссылкой на основной `docs/architecture/data-consistency.md`. citeturn1search1turn13search0turn8search1  

**Встроенные decisions (чтобы LLM не выбирала вместо вас)**  
В template обязательно храните «закреплённые выборы» (ADR или `docs/decisions/`): какие библиотеки для роутинга/валидации/миграций/DB driver/брокера (или интерфейса брокера), какую схему outbox/inbox применяем, какие стандарты событий и ключей идемпотентности. Это согласуется с практикой «дать разработчикам чек‑лист/гайд», которую ASVS рассматривает как важный контроль. citeturn0search8turn8search4turn1search1