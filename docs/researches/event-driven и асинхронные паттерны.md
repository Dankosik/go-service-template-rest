# Production-ready Go microservice template: engineering standard + LLM-instruction docs с фокусом на event-driven

## Scope

Этот стандарт и шаблонный репозиторий предназначены для **greenfield микросервиса на Go**, который разрабатывается и эксплуатируется в cloud-native среде (часто — в entity["organization","Kubernetes","container orchestration"]), и где критичны: воспроизводимая сборка, управляемая сложность, устойчивое поведение при отказах, наблюдаемость и предсказуемая работа асинхронных интеграций. Определение микросервисной архитектуры и event-driven подхода полезно закреплять через глоссарий entity["organization","Cloud Native Computing Foundation (CNCF)","cloud native foundation"] (для единообразия терминов внутри инженерной организации). citeturn21search2turn26search2

Подход **нужно применять**, если у вас:
- один сервис = один ownership, один цикл релизов, отдельная эксплуатация и независимое масштабирование (идея “independent microservices”). citeturn21search2  
- есть внешние зависимости и “реальный прод”: timeouts, graceful shutdown, метрики/трейсы/логи, безопасная конфигурация, supply chain hygiene. citeturn22search0turn22search2turn21search13turn16search3  
- ожидается **event-driven интеграция** и вы хотите заранее «зафиксировать реальную математику гарантий» (at-least-once, дедуп, ordering, DLQ), вместо мифов про «exactly-once везде». citeturn13search11turn18search0turn14search17  

Подход **не нужно применять** (или применять частично), если:
- это не сервис, а библиотека/SDK/CLI/инфраструктурный агент: структура, правила run-time конфигурации и SLA будут другими (ориентируйтесь на официальное руководство по структуре модулей/проектов в Go и выбирайте минимум лишних “микросервисных” слоев). citeturn23search3  
- у вас уже есть зрелая платформа/standard library внутри компании (общие SDK, внутренние middleware, единый брокер/формат событий): тогда ваш template должен быть “адаптером” к внутренним стандартам, а не «универсальным». citeturn26search0turn26search4  
- низколатентный/высокочастотный контур, где допустимы специализированные решения и невозможна «boring defaults» эксплуатация (часть практик останется верной, но конкретные дефолты по timeouts/логированию/инструментации будут иными). citeturn2view0turn15search2  

## Recommended defaults для greenfield template

Ниже — дефолты, которые стоит **жестко зафиксировать в template**, чтобы LLM не “догадывалась”, а воспроизводила одинаковый, идиоматичный и безопасный код.

### Toolchain и репозиторий как “источник истины”
- **Go version:** фиксируйте текущий “latest release family” как минимум в `go.mod` (на дату 2026‑02‑28 это Go 1.26). citeturn25search0turn25search5turn5search0  
- Поддерживайте управляемый выбор toolchain (например, через `GOTOOLCHAIN`/toolchain-mgmt рекомендации). Цель — одинаковые версии компилятора/линкера в CI и у разработчиков. citeturn3search19  
- Структура проекта — по официальному гайду Go team (минимализм + `internal/` для поддерживающих пакетов сервиса). Это снимает «споры про layout» и снижает шанс, что LLM “изобретет архитектуру папок”. citeturn23search3  

### Форматирование, статанализ, тесты как обязательный контракт
- Форматирование только gofmt (и/или `go fmt`, который запускает gofmt). citeturn23search0turn23search8turn23search4  
- Минимально обязательный статанализ: `go vet` (как часть стандартного toolchain). citeturn23search2turn23search10  
- Race detector в CI на тестах “где возможно” (особенно на consumers/producers, pools, кешах). citeturn6search3turn6search2  
- Фаззинг — включать для критичных парсеров, boundary cases и security-sensitive логики: fuzzing встроен в Go toolchain с Go 1.18. citeturn23search1turn23search5  

### Runtime: HTTP сервер, timeouts, graceful shutdown, конфиг
- `net/http` сервер должен иметь **явные timeouts** (ReadHeaderTimeout/WriteTimeout/IdleTimeout) и осознанный подход к request-body таймингам: docs прямо объясняют, почему ReadHeaderTimeout часто предпочтительнее ReadTimeout на уровне всего body. citeturn2view0  
- Graceful shutdown через `http.Server.Shutdown(ctx)` (он закрывает listeners, затем idle conns, и уважает deadline/timeout контекста). citeturn22search0turn22search14  
- Для работы в Kubernetes учитывайте порядок/семантику завершения Pod: preStop и общий grace period, дефолт `terminationGracePeriodSeconds=30s`. Ключевое: preStop **не асинхронен** относительно shutdown сигнала контейнеру, и может “съесть” ваш grace period, если зависнет. citeturn22search2turn22search3turn0search10  
- Конфигурация “наружу”: environment variables как базовый контракт (12-factor), а в Kubernetes — через ConfigMap/Secret как поставщик env/файлов. citeturn21search1turn21search0turn21search17  
- Secrets: документируйте и соблюдайте практики хранения/шифрования; Kubernetes отдельно подчеркивает необходимость encryption-at-rest в etcd, а entity["organization","OWASP","web app security"] дает общие принципы управления секретами и безопасного логирования. citeturn21search13turn16search2turn16search6  

### Observability “по умолчанию”
- Трейсинг/метрики: используйте entity["organization","OpenTelemetry","observability framework"] SDK и semantic conventions для единообразных имен атрибутов/метрик/спанов (иначе в распределенной системе сравнимость ломается). citeturn15search1turn15search0turn15search12  
- Логи: используйте `log/slog` как структурированный логгер стандартной библиотеки Go (ключ‑значение, обработчики, производительность). citeturn1search0turn1search4turn1search12  
- Метрики Prometheus: следуйте правилам имен/лейблов и **избегайте высокой кардинальности** (user_id/email и т.п. в labels запрещайте на уровне стандартов). citeturn15search2  
- Ресурсы: фиксируйте requests/limits для CPU/memory и документируйте, что request — гарантируемый минимум (Kubernetes описывает units и гарантии). citeturn15search3turn15search7  

### Event-driven и асинхронные паттерны как часть template (подтема c)

image_group{"layout":"carousel","aspect_ratio":"16:9","query":["transactional outbox pattern diagram","kafka partition consumer group diagram","rabbitmq dead letter exchange diagram","nats jetstream consumer ack flow diagram"],"num_per_query":1}

**Базовый принцип template:** “at-least-once delivery — норма, значит consumer обязан быть идемпотентным”. Это прямо формулируется в pattern guidance: при at-least-once брокер может доставлять дубликаты, поэтому результат обработки одного и того же сообщения многократно должен быть эквивалентен обработке один раз. citeturn18search0turn14search17turn13search11  

Фиксируем “boring defaults”:

- **Events vs commands (практическая граница):**  
  - Event = факт/изменение состояния, не адресован конкретному получателю; он «произошел» и публикуется для заинтересованных. citeturn26search2turn32view0  
  - Command/task в choreography модели выступает как «работа, которую нужно выполнить», но choreography определяется как система без центрального оркестратора, где компоненты реагируют на входящий task и могут эмитить следующий task (очень легко перепутать доменный event с “RPC по брокеру”). citeturn26search0turn26search3  

- **Выбор “контура” для событий (envelope + контракт):**  
  - Envelope: используйте entity["organization","CloudEvents","event format spec"] JSON как минимально переносимый формат: spec требует поддержку JSON format всеми реализациями. citeturn32view0  
  - Обязательные атрибуты CloudEvents: `id`, `source`, `specversion`, `type`. Важно: `source+id` должны быть уникальны; consumer может считать одинаковые `source` и `id` дубликатом. citeturn32view0  
  - Документация async API: храните AsyncAPI документ в репозитории как контракт по каналам/топикам/схемам (машиночитаемо, protocol-agnostic). entity["organization","AsyncAPI Initiative","async api spec"] описывает AsyncAPI как спецификацию асинхронных API (message-driven). citeturn19search9turn19search5  

- **Outbox как дефолт для публикации событий из транзакционного сервиса:**  
  - Проблема dual-write (БД + брокер) — фундаментальная. Transactional outbox фиксирует событие в БД в той же транзакции, что и бизнес-изменение, а отдельный процесс публикует его в брокер. citeturn0search7turn18search1  
  - Даже с outbox возможны дубликаты на стороне “процессора событий/ретраев”, поэтому consuming service должна быть идемпотентной/с дедуп-трекингом. citeturn18search10turn18search0  
  - Для entity["organization","PostgreSQL","database system"] типичный boring механизм конкурентной обработки outbox-строк: `SELECT … FOR UPDATE SKIP LOCKED`/CTE‑паттерн, чтобы несколько воркеров не брали одну и ту же строку. Это прямо поддерживается и документируется в PostgreSQL. citeturn20search0turn20search8  

- **Consumer groups и параллелизм:**  
  - В log-based брокерах (пример: entity["organization","Apache Kafka","streaming platform"]) топик разбит на partitions, каждая partition — упорядоченный “commit log”. Семантическое partitioning используется, чтобы “делить обработку между consumer процессами” и **сохранять порядок внутри partition**. citeturn12view0  
  - Kafka прямо описывает семантику at-least-once на стороне consumer при падении до фиксации позиции: если consumer обработал сообщения, но не сохранил позицию, новый процесс может получить и обработать их повторно — это и есть at-least-once при сбое. citeturn13search10  

- **Delivery semantics по умолчанию и “мифы”:**  
  - Kafka дизайн-док говорит: Kafka “guarantees at-least-once delivery by default” и позволяет реализовать at-most-once (отключая retries у producer и коммитя offsets до обработки). Это удобное место, чтобы закрепить “что реально возможно” и где лежит переключатель. citeturn13search11  
  - entity["organization","RabbitMQ","message broker"] четко связывает ack с guarantees: acknowledgements дают at-least-once; без ack возможны потери и гарантируется только at-most-once. citeturn14search17  
  - entity["organization","NATS","messaging system"] в core режиме — at-most-once; JetStream добавляет хранение и возможность replay, а также at-least-once/“exactly-once quality” через publication-id/dedup. citeturn29search4turn29search2turn30search14turn30search0  
  - Следствие для шаблона: “exactly-once end-to-end” почти всегда превращается в комбинацию outbox + идемпотентный consumer + дедуп/уникальные ограничения в целевом datastore, а «exactly-once» в документации брокера обычно ограничено рамками одного брокера/транзакционных границ и не покрывает побочные эффекты во внешней БД. Этот вывод следует прямо из описаний at-least-once и необходимости idempotent consumer. citeturn18search0turn13search11turn0search7  

- **Дедупликация и идемпотентность:**
  - На уровне базы: используйте unique constraints и `INSERT … ON CONFLICT` как базовый building block для dedup/idempotency. PostgreSQL явно описывает `ON CONFLICT` как механизм альтернативы ошибке уникального ограничения. citeturn20search2  
  - В JetStream: dedup на публикации через `Nats-Msg-Id` и Duplicate Window (по умолчанию 2 минуты; docs предупреждают про слишком большие окна). citeturn30search0turn30search2turn30search1  

- **Ordering:**
  - Kafka: порядок гарантирован внутри одного TCP connection (порядок запросов/ответов) и, на уровне данных, partition — ordered commit log; глобального порядка “между partitions” нет по определению sharding. citeturn12view0  
  - RabbitMQ: ordering может изменяться из-за priorities и requeue; при multiple consumers доставка идет round-robin. Для “bullet proof FIFO включая redelivery” официально рекомендуют Single Active Consumer + prefetch=1. citeturn28search0turn28search3turn28search9  

- **Retries / backoff / DLQ / poison messages:**
  - Retry: используйте exponential backoff и ограничивайте попытки; gRPC guidance прямо говорит, что приложения должны понимать, что ретраить, определить backoff параметры и мониторить retry-метрики. citeturn33search2turn33search0  
  - DLQ: используйте Dead Letter Channel как стандартный паттерн (сообщения, которые нельзя/не нужно доставить/обработать, перемещаются в отдельный канал). citeturn34search0  
  - RabbitMQ: Dead Letter Exchanges — встроенный механизм “dead-lettering”; quorum queues имеют отдельную “poison message handling” с `x-delivery-count` и лимитом повторных доставок, после которого сообщение drop или dead-lettered. citeturn14search2turn34search3turn27view0  
  - Replay: JetStream предназначен для хранения и replay; replay policy (`ReplayInstant` vs `ReplayOriginal`) описан в docs. RabbitMQ streams официально описывает replay/time-traveling и возможность читать “с offset/времени”. citeturn29search2turn29search1turn28search6  

- **Schema evolution и versioning событий:**
  - Если вы используете Avro — спецификация позиционируется как authoritative и изначально рассчитана на schema-based хранение/чтение. citeturn19search1turn19search7  
  - Если используете Schema Registry (vendor-neutral по идее, но часто через Confluent экосистему) — фиксируйте compatibility режимы и понимайте transitive vs latest проверку совместимости. citeturn19search3turn19search0  
  - Protobuf: следуйте best practices (не менять тип поля, не переиспользовать tag numbers; изменения ломают десериализацию). citeturn17search0  
  - CloudEvents: версионируйте тип события (рекомендация включать версию в `type`/семантику типа и придерживаться reverse-DNS префикса). citeturn32view0  

## Decision matrix / trade-offs

Эта матрица — то, что в template стоит оформить как ADR-шаблоны и “decision log”: LLM должна ссылаться на нее, а не «выбирать по вкусу».

### Синхронно vs асинхронно
Асинхронность оправдана, когда вам нужно: изоляция по отказам/пиковым нагрузкам, буферизация, fan-out, ослабление связанности и независимый throughput обработчиков (идея choreography: сервис работает “когда получает работу”, на своем темпе). Но это платится сложностью: delivery semantics, дедуп, ordering, troubleshooting. citeturn26search0turn18search0turn14search17  

Практическая политика:
- default: **синхронный API** (HTTP/gRPC) для user-facing и критичных по latency путей, где требуется быстрый ответ и понятная семантика ошибок. Гарантии timeouts/deadlines — обязательны (gRPC рекомендует всегда ставить deadline; сервер отменяет вызов после истечения). citeturn17search14turn17search2turn1search1  
- асинхронно: интеграционные/интеграционно-аналитические события, фоновые процессы, fan-out, “eventual consistency” (включая saga). citeturn33search3turn0search7turn26search2  

### Queue vs log-based broker (и что это значит для кода)
- Log-based (Kafka): сильная модель partitioning как “semantic partitioning”, где порядок сохраняется внутри partition, а consumer groups дают параллелизм = number of partitions. Это удобно для replay и масштабирования, но требует дисциплины по keys/partitioning и понимания offsets. citeturn12view0turn10search3turn13search6  
- Queue (RabbitMQ classic/quorum): проще для task-очередей и routing; при нескольких consumers — round-robin, ordering может “поплыть” из-за requeue/priority; для HA используйте quorum queues как современный replicated default. citeturn28search3turn28search0turn27view1  
- NATS Core vs JetStream: core — at-most-once (нужна активная подписка), JetStream — хранение, replay, at-least-once и dedup на публикации. citeturn29search4turn29search2turn30search0  

### Outbox vs “напрямую publish в брокер”
- Прямой publish в брокер “из transaction handler” дает dual-write проблему: не существует атомарности между БД и брокером без дополнительных механизмов. Transactional outbox — стандартный ответ. citeturn0search7turn18search1  
- Минусы outbox: дополнительные таблицы/воркеры/мониторинг, возможные дубликаты при ретраях delivery. Плюсы: консистентность “read your own writes” и предсказуемая доставка. citeturn18search15turn18search10  

### “Exactly-once” vs at-least-once + идемпотентность
- Kafka официально описывает at-least-once by default и путь к at-most-once; при этом на практике “end-to-end exactly once” требует либо транзакционных границ внутри Kafka, либо идемпотентного sink во внешнюю БД. citeturn13search11turn18search0  
- RabbitMQ: acknowledgements => at-least-once. Следовательно, “no duplicates” — это не гарантия брокера, а свойство consumer обработки. citeturn14search17turn18search0  
- JetStream dedup обеспечивает “exactly-once publication” в рамках окна/механизма dedup, но обработка у consumer все равно должна быть идемпотентной при redelivery. citeturn30search0turn29search1  

## Набор правил MUST / SHOULD / NEVER для LLM

Ниже — набор правил, который стоит вынести в отдельный LLM-guideline файл и применять как “policy layer”. Формулировки сознательно **нормативные**.

### Архитектура и структура кода
- MUST следовать официальному layout для Go module/command + `internal/`, не изобретать “стандартный layout из интернет-репо” без ADR. citeturn23search3turn0search5  
- MUST держать public API (HTTP/gRPC/events) как “single source of truth” в `docs/api/` (OpenAPI/AsyncAPI/proto) и обновлять реализацию вместе с контрактом. OpenAPI определяет цель: machine-readable описание HTTP API. citeturn17search11turn19search9  
- SHOULD минимизировать внешние зависимости; если dependency добавляется — объяснить зачем (ADR) и обновить `go.mod/go.sum` и CI. Go toolchain проверяет hashes через `go.sum`. citeturn5search0turn4search8  
- NEVER добавлять “магические” архитектурные слои (clean/hexagonal) без явной необходимости и без того, чтобы они реально снижали сложность. Официальный гайд Go по структуре проектов намеренно оставляет свободу выбора, но не требует “слоистой архитектуры”. citeturn23search3turn18search18  

### Go-идиомы, ошибки, контекст, конкуррентность
- MUST использовать `context.Context` для отмены/timeout в I/O и длительных операциях; передавать ctx вниз по стеку. Context может отменяться по дедлайну и каскадно отменяет derived contexts. citeturn1search1turn1search5  
- NEVER использовать `context.WithValue` как способ передать “параметры функции” или зависимости; документация context прямо запрещает это (только request-scoped data “that transits processes and APIs”). citeturn24search10  
- MUST оборачивать ошибки с контекстом (`%w`) и использовать `errors.Is/As` для ветвления по причинам. citeturn3search0turn3search1turn1search2  
- SHOULD запускать `-race` для тестов, затрагивающих concurrency; data races приводят к memory corruption и крайне сложно дебажатся. citeturn6search3turn6search2  

### HTTP/gRPC и жизненный цикл сервиса
- MUST выставлять timeouts у HTTP сервера и документировать их значения; обоснование timeouts (и почему ReadHeaderTimeout часто важнее) дано в docs `net/http`. citeturn2view0  
- MUST корректно завершать сервис: `Server.Shutdown(ctx)` + обработка SIGTERM, учитывая Kubernetes termination semantics и дефолтный grace period. citeturn22search0turn22search2turn22search3  
- SHOULD иметь liveness/readiness endpoints и корректно переключать readiness в “not ready” при начале завершения или деградации зависимостей. Kubernetes описывает probes и их работу. citeturn0search6turn22search2  
- SHOULD в gRPC всегда использовать deadlines (best practice: “Always set a deadline”), а на сервере уважать отмену. citeturn17search14turn17search2  

### Observability и безопасность
- MUST логировать структурированно (slog) и избегать утечки секретов/PII. citeturn1search0turn16search6turn16search2  
- MUST следовать Prometheus правилам лейблов: никогда не использовать high-cardinality labels (user_id/email). citeturn15search2  
- SHOULD использовать OpenTelemetry semantic conventions и единые имена атрибутов/метрик/спанов. citeturn15search0turn15search4  
- MUST прогонять `govulncheck` в CI и вносить фиксы зависимостей при обнаружении уязвимостей; govulncheck — “low-noise” и встроен в экосистему управления уязвимостями Go. citeturn4search1turn4search10turn4search4  
- SHOULD ориентироваться на OWASP Top 10 и ASVS как на чеклист требований (особенно для authn/authz/secret handling/logging). citeturn16search1turn16search8turn16search0  

### Event-driven: producer/consumer правила
- MUST считать at-least-once delivery нормой и проектировать consumers идемпотентными. citeturn18search0turn14search17turn13search11  
- MUST использовать transactional outbox для публикации событий, связанных с изменением в БД. citeturn0search7turn18search1  
- MUST включать в событие уникальный идентификатор (CloudEvents `id` + `source` как ключ дедупликации). citeturn32view0  
- SHOULD хранить и валидировать схемы и compatibility правила (Avro/Protobuf/JSON schema) и документировать порядок rollout при compatibility режимах. citeturn19search1turn17search0turn19search3  
- NEVER “ack/commit before side effects”: подтверждение обработки сообщения должно происходить **после** того, как сделаны все необратимые эффекты (запись в БД, публикация follow-up событий и т.п.), иначе вы получите потери/рассинхрон или дубли. Kafka дизайн и consumer docs описывают риски вокруг offsets/commit. citeturn13search10turn13search1  

## Concrete good / bad examples

Ниже — примеры, которые стоит включить в docs как эталон. Они специально “скучные” и предсказуемые.

### Good: HTTP server с timeouts + graceful shutdown

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
	mux.HandleFunc("/live", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MiB
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		logger.Info("http server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "err", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	logger.Info("http server shutting down")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http shutdown error", "err", err)
	}
}
```

Почему это “good”: `net/http` прямо описывает семантику timeouts и `Server.Shutdown(ctx)` как graceful shutdown; корректный timeout на shutdown важен в Kubernetes, где есть ограничение на termination grace period. citeturn2view0turn22search0turn22search2turn22search3  

### Bad: бесконечный HTTP без timeouts и без Shutdown

```go
http.ListenAndServe(":8080", handler) // no timeouts, no graceful shutdown
```

Почему это “bad”: без timeouts вы повышаете риск зависаний на медленных клиентах/slowloris, а без `Shutdown` сервис не закрывает соединения корректно при SIGTERM/termination, что конфликтует с ожиданиями Kubernetes завершения Pod. citeturn2view0turn22search0turn22search8  

### Good: Outbox worker в PostgreSQL с SKIP LOCKED

```go
// Псевдокод SQL-цикла (идея):
//
// WITH cte AS (
//   SELECT id
//   FROM outbox
//   WHERE status = 'new'
//   ORDER BY id
//   FOR UPDATE SKIP LOCKED
//   LIMIT 50
// )
// UPDATE outbox
// SET status = 'processing', locked_at = now()
// WHERE id IN (SELECT id FROM cte)
// RETURNING id, payload;
```

Почему это “good”: `SKIP LOCKED` предназначен как раз для ситуации, когда несколько обработчиков конкурируют за строки, и позволяет избегать “двойной обработки” одной и той же строки в конкурентных воркерах. citeturn20search0turn20search8  

### Bad: publish event “внутри транзакции” без outbox

```go
tx := db.Begin()
_ = updateBusinessState(tx)
_ = broker.Publish(event) // dual-write risk
_ = tx.Commit()
```

Почему это “bad”: transactional atomicity между БД и брокером здесь отсутствует; transactional outbox — стандартная тактика решения dual-write при микросервисной интеграции. citeturn0search7turn18search1turn18search10  

### Good: Idempotent consumer через unique constraint + ON CONFLICT

```sql
-- таблица дедупликации / inbox
CREATE TABLE inbox_processed (
  consumer_group text NOT NULL,
  event_id text NOT NULL,
  processed_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (consumer_group, event_id)
);

-- обработка:
-- 1) попытаться вставить event_id
-- 2) если конфликт — это дубль, выходим без side effects
INSERT INTO inbox_processed (consumer_group, event_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;
```

Почему это “good”: pattern “idempotent consumer” — базовая защита при at-least-once; `ON CONFLICT` — документированный механизм PostgreSQL для обработки уникальных конфликтов без ошибок. citeturn18search0turn20search2turn14search17  

## Anti-patterns и типичные ошибки/hallucinations LLM

Это раздел, который полезно держать рядом с LLM-instructions, чтобы “предупреждать галлюцинации”.

1) **“Exactly-once по умолчанию”**. LLM часто заявляет “мы гарантируем exactly-once”, игнорируя, что брокеры по умолчанию дают at-least-once (Kafka) или ack=>at-least-once (RabbitMQ), следовательно дубликаты возможны, и требуются outbox + idempotent consumer. citeturn13search11turn14search17turn18search0turn0search7  

2) **Ack/commit до выполнения side effects**. Частая ошибка: ack сообщения сразу после “получили”, а запись в БД/вызов API — позже. Kafka docs прямо связывают at-least-once/at-most-once с тем, когда consumer сохраняет позицию относительно обработки; аналогично в RabbitMQ без правильного ack semantics получаются потери/дубли. citeturn13search10turn13search1turn14search17  

3) **Непонимание ordering**:  
   - “Kafka гарантирует глобальный порядок” — неверно: порядок сохраняется внутри partition (commit log), а между partitions — нет. citeturn12view0  
   - “RabbitMQ всегда FIFO для всех случаев” — docs предупреждают, что priorities и requeue меняют наблюдаемый порядок; для строгого FIFO включая redelivery нужна отдельная конфигурация (Single Active Consumer + prefetch=1). citeturn28search0turn28search9  

4) **Использование `context.WithValue` как DI/передачи параметров**. Это противоречит документации context. citeturn24search10  

5) **Перегрузка метрик label’ами высокой кардинальности** (user_id/email/trace_id как label) — очень распространенная “оптимизация” LLM. Prometheus прямо предупреждает “do not use labels with high cardinality”. citeturn15search2  

6) **Секреты в логах/конфиге**. LLM может “для дебага” логировать весь конфиг или request payload. Это нарушает OWASP guidance по secrets management и secure logging. citeturn16search2turn16search6  

7) **Схемы событий “без эволюции”**: LLM часто меняет поля/типы/номера в Protobuf, не понимая wire-compatibility. Protobuf best practices прямо предупреждают: “don’t change field type”, “reusing tag number ломает десериализацию”. citeturn17search0  

8) **Ретраи без ограничений**. LLM любит “retry forever” или “fixed sleep”. Лучше: retry с exponential backoff и лимитами попыток; gRPC guide и AWS pattern фиксируют это как best practice для transient failures. citeturn33search2turn33search0  

## Review checklist для PR/code review

Этот чеклист полезно положить в PR template и в docs/code-review.

- Go toolchain/качество:
  - Код отформатирован gofmt; нет “ручного форматирования”. citeturn23search0turn23search4  
  - `go vet` чистый, нет новых предупреждений. citeturn23search2turn23search10  
  - Тесты покрывают критичные ветки; для concurrency path прогнан `-race`. citeturn6search3turn6search2  

- Контекст/таймауты/жизненный цикл:
  - Все I/O операции принимают `context.Context` и уважают отмену/дедлайн. citeturn1search1turn1search5  
  - HTTP server имеет ReadHeaderTimeout/WriteTimeout/IdleTimeout и понятный shutdown timeout; используется `Server.Shutdown`. citeturn2view0turn22search0  
  - Учтены Kubernetes termination semantics (preStop и grace period). citeturn22search2turn22search3  

- Observability:
  - Логи структурированы на `log/slog`; не логируются секреты/PII. citeturn1search0turn16search6turn16search2  
  - Метрики без high-cardinality labels; имена/лейблы соответствуют best practices. citeturn15search2  
  - Трейсы/атрибуты следуют semantic conventions (если включены). citeturn15search0turn15search1  

- Security и supply chain:
  - Конфиг и секреты вынесены во внешнюю конфигурацию; для Kubernetes используются ConfigMap/Secret; секреты не зашиты в образ/код. citeturn21search0turn21search17turn21search1  
  - `govulncheck` запускается (CI/локально) и результаты учтены. citeturn4search10turn4search1  
  - Changes соответствуют OWASP Top 10/ASVS базовой гигиене (особенно authn/authz/логирование/конфигурация). citeturn16search1turn16search8turn16search0  

- Event-driven (если применимо):
  - События имеют стабильный контракт (CloudEvents/AsyncAPI), уникальный id, понятный source/type и правила versioning. citeturn32view0turn19search9  
  - Producer публикует через outbox; нет dual-write “в транзакции”. citeturn0search7turn18search1  
  - Consumer идемпотентен (dedup keys/таблица/unique constraints), ack/commit делается после side effects. citeturn18search0turn20search2turn13search10turn14search17  
  - Есть стратегия retries/backoff и DLQ/Dead Letter Channel для невалидных/poison сообщений; наблюдаемость ошибок достаточная для прод-диагностики. citeturn33search0turn34search0turn34search3turn14search2  

## Что оформить отдельными файлами в template repo

Ниже — практический “список артефактов”, которые лучше физически положить в репозиторий, чтобы LLM могла ссылаться на них как на источники истины и не “догадываться”.

1) `docs/engineering-standard.md`  
Единый стандарт (основная часть этого отчета) с разделами: toolchain, http runtime, observability, security, eventing. Опирайтесь на официальные источники Go tooling, Kubernetes lifecycle, OWASP, OpenTelemetry, Prometheus. citeturn23search3turn22search2turn16search1turn15search1turn15search2  

2) `docs/llm-instructions.md`  
Нормативные MUST/SHOULD/NEVER правила (раздел выше) + компактный “workflow” для LLM:  
- сначала прочитать `docs/*` и контракт API, затем предложить изменения, затем обновить тесты/линты.  
Смысл: превратить LLM в “исполнителя стандартов”, а не “архитектора по наитию”. citeturn3search14turn3search2  

3) `docs/eventing.md`  
Отдельный документ “Event-driven defaults”: CloudEvents envelope, AsyncAPI, outbox/inbox, dedup, ordering, retries/DLQ, replay, schema evolution/versioning. Он должен фиксировать “реальные гарантии” (Kafka/RabbitMQ/NATS) и запреты на мифы. citeturn32view0turn14search17turn13search11turn29search4turn0search7  

4) `docs/api/openapi.yaml` и/или `api/` (proto/AsyncAPI)  
Контракты как артефакты сборки/ревью. OpenAPI и AsyncAPI — машиночитаемые спецификации, их легко валидировать и генерировать документацию. citeturn17search11turn19search9  

5) `docs/config.md` + `internal/config/`  
Документируйте каждую переменную окружения, тип, дефолт, обязательность, “опасность” (секрет/PII). В Kubernetes отразите маппинг через ConfigMap/Secret. citeturn21search1turn21search0turn21search17turn21search13  

6) `docs/runbook.md`  
Операционный runbook: как читать метрики/логи/трейсы, как делать graceful shutdown, как диагностировать stuck consumers/outbox lag, как работать с DLQ/replay. Основание: Kubernetes termination lifecycle, OTel, Prometheus naming и сообщения брокеров. citeturn22search2turn15search0turn15search2turn28search6turn29search2  

7) `.github/workflows/ci.yml` + `Makefile`/`Taskfile.yml`  
Фиксируйте как минимум: `go test ./...`, `go test -race ./...` (условно), `go vet ./...`, `gofmt -w` check, `govulncheck ./...`. Это превращает стандарт в исполняемый контракт. citeturn6search3turn23search2turn23search0turn4search1  

8) `.github/pull_request_template.md`  
Вставьте краткий checklist (из раздела review) и отдельные чекбоксы для event-driven изменений: schema changes, idempotency, ack/commit semantics. citeturn18search0turn19search3