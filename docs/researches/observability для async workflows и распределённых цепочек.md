# Стандарт observability для async и event-driven компонентов в Go microservice template

## Scope

Этот стандарт обязателен для компонентов, где бизнес‑операции реализованы асинхронно: event-driven микросервисы, consumers/workers, фоновые jobs, интеграции через очереди/топики, а также любые distributed chains, где «одна логическая операция» проходит через брокер(ы), ретраи и возможные DLQ. Ключевая цель — обеспечить сквозную диагностируемость и управляемость: связать producer ↔ consumer, ретраи и «хвосты» в очередях, не упираясь в догадки и «ручной археологический» поиск по логам. Это прямо следует из того, что без context propagation невозможно связать входящий request с downstream HTTP и с message producer и его consumers. citeturn13view0turn25view3

Подход применим, когда вы хотите придерживаться boring/battle-tested default: стандартизированная телеметрия, переносимая между бэкендами наблюдаемости и совместимая с инфраструктурой (включая managed services). Рекомендованный baseline: начинать с небольшого количества «сигналов», но так, чтобы они коррелировались (trace ↔ logs ↔ metrics) и давали ответ на «что сломалось» и «почему». citeturn13view0turn27view0

Этот стандарт **не** является лучшим выбором, если:
- система строго синхронная (чистый request/response) и асинхронность отсутствует — тогда достаточно базового HTTP/RPC‑стандарта;
- вы физически не можете прокидывать метаданные (например, протокол/шина не позволяет заголовки/атрибуты, либо они обрезаются), и вы сознательно принимаете отсутствие end‑to‑end traces; в таком случае вы всё равно обязаны обеспечить корреляцию через стабильные message/workflow IDs в payload и log‑корреляцию, но это будет компромисс с более высокой стоимостью расследований;
- комплаенс/модель угроз запрещают перенос идентификаторов или контекста между trust‑zones (тогда требуется отдельный security review и политика редактирования/обрезания контекста, особенно baggage). citeturn29view0turn31view1

## Recommended defaults для greenfield template

### Базовый стек и «boring defaults»

В качестве общего стандарта телеметрии используется entity["organization","OpenTelemetry","cncf observability project"]: в приложение встраивается SDK, а экспорт идёт по OTLP в коллектор (или напрямую в бэкенд, если инфраструктура такова, но в template по умолчанию должен быть предусмотрен Collector как точка контроля). Коллектор рассматривается как исполняемый компонент, который принимает телеметрию, обрабатывает и экспортирует её по pipeline’ам. citeturn32view0turn32view2

Критично, чтобы Collector‑конфигурация была «минимально необходимой» (minimize attack surface) и включала шифрование и аутентификацию на канале передачи телеметрии. citeturn32view1

Конфигурация экспортёров через env‑vars допустима, но в стандарте нужно считать это **неполностью переносимым** между языками/SDK и требовать эквивалентной code‑конфигурации (то есть env не должен быть «единственным способом»). citeturn32view2turn32view3

### Контекст и trace propagation через сообщения/события

По умолчанию используется W3C Trace Context (traceparent/tracestate) от entity["organization","World Wide Web Consortium","web standards body"] как универсальный формат переносимого trace context. Это согласуется с тем, что OpenTelemetry использует W3C TraceContext как формат propagation по умолчанию, а сам стандарт определяет заголовки и формат передачи контекста. citeturn30view0turn30view2turn13view0

Механизм переносимости через «carrier» обязан соответствовать Propagators API: Inject/Extract на string key/value носителе; на чтении ошибки парсинга не должны бросать исключения и не должны затирать валидный контекст; для повторно используемых carrier’ов (например, реюз объекта сообщения в ретраях) поля пропагации нужно очищать перед Inject. citeturn30view1

Для CloudEvents‑событий используйте Distributed Tracing Extension CloudEvents: `traceparent`/`tracestate` как атрибуты события, при этом для multi‑hop цепочек extension должен хранить именно «starting trace» передачи и **не должен** перезаписываться на каждом hop (иначе вы теряете причинно‑следственную историю передач). citeturn15view0turn14view2

### Tracing‑модель для async workflows и distributed chains

В template по умолчанию следует моделировать messaging‑операции по OpenTelemetry semantic conventions для messaging spans (важно понимать, что статус этих конвенций — Development, и это нужно явно зафиксировать как управляемый риск миграций). citeturn10search11turn25view0

Минимальная структура span’ов для producer/consumer должна позволять:
- коррелировать producer ↔ consumer через message creation context и context propagation (иначе consumer trace будет «оторван»); citeturn25view3turn13view0  
- корректно представлять batch‑получение/обработку и «несколько причин» (multi‑parent) через **Span Links**, потому что span имеет только одного родителя; Links прямо предназначены для batched operations и причинной связи между traces. citeturn25view3turn1search13turn25view2

Операционные типы и kind’ы в messaging‑спанах должны соответствовать таблице OTel:
- `send` → `PRODUCER` (в частном случае может быть `CLIENT`, если send не несёт creation context),
- `receive` → `CLIENT`,
- `process` → `CONSUMER`. citeturn25view0turn24view2

Отдельно: OTel прямо указывает, что если обработка сообщения происходит «в scope другого span» (например, consumer внутри HTTP‑обработчика), то не рекомендуется делать creation context родителем process‑спана по умолчанию; вместо этого нужно сохранять корреляцию через links (и при необходимости линковать ambient context). Это ключевой нюанс для «асинхронности внутри синхронного запроса» и для fan‑in/fan‑out. citeturn25view1turn25view3

### Обязательный telemetry contract для async components

Ниже — «практический норматив», который в template должен быть реализован в виде повторно используемых helper’ов/обёрток. Он специально сформулирован так, чтобы LLM не «придумывала» названия/атрибуты и не ломала корреляцию.

**Обязательные идентификаторы и поля корреляции для сообщений** (хранятся в message headers/attributes, а не в метриках‑лейблах):
- `traceparent`, `tracestate` — W3C trace context (или CloudEvents‑эквивалент), citeturn30view0turn15view0  
- `message_id` — уникальный ID сообщения/события (в OTel это `messaging.message.id`), citeturn24view0turn25view2  
- `correlation_id` (conversation id) — стабильный ID «диалога/цепочки»; в OTel это `messaging.message.conversation_id` (иногда так и называется «Correlation ID»), citeturn24view0turn19view0turn8view0  
- `attempt`/`delivery_count` — номер попытки обработки (где доступно — берётся из брокера; иначе ведётся приложением и сохраняется при ретраях), citeturn20view0turn34search0  
- `first_seen_at`/`enqueued_at` — timestamp первого появления/постановки, если брокер предоставляет (для расчёта end‑to‑end latency). citeturn34search0turn34search1turn4view4  

**Producer MUST: traces**
- Создавать span операции отправки с operation.type=`send` и корректным span kind (обычно `PRODUCER`) и заполнять messaging attributes: `messaging.system`, `messaging.destination.name` (при необходимости `messaging.destination.template` для низкой кардинальности), `messaging.message.id` и (если есть) `messaging.message.conversation_id`. citeturn24view0turn23view0turn25view0turn24view3  
- Инжектить W3C trace context в message headers/attributes через TextMap‑propagator (а не в payload), обеспечивая транспортную независимость. citeturn30view1turn25view3turn35search2  

**Consumer MUST: traces**
- Делать Extract контекста из сообщения, и создавать `process` span kind=`CONSUMER` для выполнения handler’а; при pull‑модели (poll/receive) отдельный `receive` span kind=`CLIENT` допустим и полезен (особенно для диагностики «polling/empty receives»), но `process` span является обязательным как «истинная» обработка. citeturn25view0turn25view3turn23view0  
- Для batch обработки: один processing span обязан содержать links на каждый message creation context, чтобы одна batch‑операция не «потеряла» причинность. citeturn25view2turn1search13  
- Ошибки должны отражаться в span статусе/ошибках, а `error.type` должен быть предсказуемым и с низкой кардинальностью. Это важно одновременно для trace‑аналитики и метрик, где `error.type` используется как условный атрибут. citeturn23view0

**Jobs / batch / reconcilers MUST: traces**
- Каждое выполнение job должно иметь root span (новый trace), с явными атрибутами «что это за job» и «какой run». Если job обрабатывает сообщения пачкой/из разных «родителей», то job‑span обязан линковаться (Span Links) к upstream trace context каждого исходного сообщения/триггера. citeturn1search13turn25view2turn15view0  

### Обязательные метрики для async observability

В стандарте должны быть **две группы метрик**: (1) app‑level метрики обработки и стабильность handler’ов; (2) broker/platform‑level метрики очередей/лагов.

**App‑level метрики MUST** (по OTel messaging metrics; статус Development — закрепить в doc и в wrapper‑слое):  
- `messaging.client.operation.duration` (Histogram, unit seconds) — длительность messaging операций send/receive/ack; требуемые атрибуты включают `messaging.operation.name`, `messaging.system` и др. citeturn23view0turn24view0  
- `messaging.client.sent.messages` — счётчик отправленных сообщений producer’ом. citeturn23view0  
- `messaging.client.consumed.messages` — счётчик потреблённых сообщений consumer’ом. citeturn23view0  
- `messaging.process.duration` (Histogram, unit seconds) — длительность обработки (handler time) и обязательная для операций `process`. citeturn23view0  

**Дополнительные app‑level метрики MUST** (это «template‑standard», чтобы ловить типовые EDA‑паттерны отказов; имена фиксируются в repo conventions):
- `async_handler_outcome_total{outcome="success|retryable_error|non_retryable_error|poison"}` — счётчик исходов обработки (лейблы строго bounded). Обоснование: SRE практики требуют измерять errors и saturation; для async это не только «ошибка», но и её класс, влияющий на ретраи и DLQ. citeturn27view0turn23view0  
- `async_retry_scheduled_total{reason="..."}` и `async_dlq_published_total{reason="..."}` — видимость ретраев и DLQ. Причины должны быть низкой кардинальности (категории). citeturn23view0turn18view0turn1search3  
- `async_idempotency_decision_total{decision="processed|duplicate_ignored|conflict"}` — наблюдаемость идемпотентности (иначе «дубликаты» маскируют деградации и приводят к неочевидным финансовым/состоянийным багам). Требование идемпотентности особенно обязательно для at‑least‑once очередей. citeturn4view5turn3search3turn28view1  

**Правило кардинальности метрик MUST**: запрещено использовать в label’ах неограниченные значения (message_id, correlation_id, user_id, request_id, routing_key с высокой вариативностью и т.п.). Это прямо запрещено best practices Prometheus: каждый уникальный набор label’ов создаёт новый time series и может взорвать стоимость/производительность. citeturn33view0  

### Broker/platform‑level метрики MUST: backlog, lag, DLQ visibility

**SQS MUST**: мониторить рост очереди и возраст сообщений через CloudWatch метрики `ApproximateNumberOfMessagesVisible`, `ApproximateNumberOfMessagesNotVisible`, `ApproximateAgeOfOldestMessage`. Для DLQ нельзя полагаться на `NumberOfMessagesSent` (оно не учитывает auto‑move в DLQ) — рекомендованная метрика состояния DLQ: `ApproximateNumberOfMessagesVisible`. citeturn4view4

**Kafka MUST**: мониторить consumer lag и способность «успевать» за продьюсером. Apache Kafka рекомендует для consumer’ов следить за max lag и min fetch rate; также в monitoring‑метриках присутствует `records-lag-max` как максимальный лаг по партициям. citeturn21search1turn21search0  
Если Kafka — managed (например, AWS), мониторинг consumer lag поддерживается через CloudWatch/Prometheus, и цель — выявлять slow/stuck consumers и предпринимать ремедиации (масштабирование/перезапуск). citeturn21search18

**RabbitMQ MUST**: метрики depth/застревания очереди должны различать «готовые к доставке» и «выданные, но не ack’нутые» сообщения. Через HTTP API доступно `messages`, `messages_ready`, `messages_unacknowledged`; также официально рекомендованы monitoring/Prometheus плагины (низкий overhead, highly recommended для production). citeturn26search2turn26search0turn19view0  

### Наблюдаемость ретраев, DLQ и «retry storms»

**Retry storms MUST быть видимы как отдельный класс деградации**, иначе система будет «само‑DDoS’иться» при частичных отказах. Amazon прямо указывает: ретраи «selfish», при overload они ухудшают ситуацию и могут задержать recovery; базовый паттерн — exponential backoff, capped, с ограничением числа ретраев, и обязательно jitter, чтобы убрать коррелированные пики повторов. citeturn28view1turn28view0  
Это относится не только к ретраям, но и к периодическим задачам: рекомендуется добавлять jitter к таймерам/periodic jobs, иначе вы получаете синхронные пики нагрузки, которые агрегированные метрики могут скрывать. citeturn28view1  
Для SRE‑мониторинга это напрямую связано с предотвращением cascading failures (положительная обратная связь, когда перегруз одной части системы повышает вероятность отказа остальных). citeturn27view1  

**DLQ visibility MUST** включать:
- метрику глубины DLQ и возраста сообщений в DLQ,
- топ причин (bounded categories),
- возможность из DLQ‑сообщения восстановить «почему» и «сколько попыток было» (delivery_count / receive_count / x‑death и т.п.),
- ссылку на исходный trace/correlation.  

Для RabbitMQ DLX стандарт должен учитывать, что dead-lettering модифицирует headers и ведёт историю dead-letter событий (x‑death) и что возможны циклы dead-lettering (RabbitMQ может детектить цикл и drop’нуть сообщение), а при отсутствии DLX‑цели сообщения могут «тихо» теряться. Это должно быть отражено в SRE‑алертах как «конфигурационная авария». citeturn18view0  
Для SQS DLQ стандарт должен учитывать maxReceiveCount как условие перемещения в DLQ и необходимость выставлять его достаточно большим для устойчивости. citeturn34search2turn1search3  

## Decision matrix / trade-offs

| Решение | Вариант | Плюсы | Минусы/риски | Recommended default |
|---|---|---|---|---|
| Формат trace propagation | W3C tracecontext (traceparent/tracestate) | Совместимость и стандартизация; поддерживается OTel по умолчанию; проще сквозная корреляция citeturn30view0turn30view2turn13view0 | Требует дисциплины прокидывания через broker headers/attributes; некоторые managed‑слои могут терять метадату | **Да** |
| Где хранить trace context | Headers/attributes (TextMap) | Соответствует Propagators API и OTel context propagation; не загрязняет payload citeturn30view1turn25view3turn35search2 | Для некоторых брокеров есть лимиты на headers/attributes; нужно следить за очисткой при ретраях/reuse carrier citeturn30view1 | **Да** |
| Топология trace для async | Parent-child везде | Простой mental model | Во многих async сценариях parent-child «ломается» (batch, multi-hop, потребление в scope другого контекста); теряется смысл | **Нет** |
| Топология trace для async | Links (span links) на upstream contexts | Единственный универсальный способ для batch/multi-parent; рекомендовано OTel для batched операций citeturn1search13turn25view2turn25view3 | Не все бэкенды одинаково хорошо показывают links; требует дисциплины в коде | **Да** |
| Семантические конвенции messaging | Использовать OTel semconv сейчас | Единый словарь атрибутов/метрик; переносимость дашбордов и анализов citeturn24view0turn23view0 | Статус Development; возможны изменения и миграции; нужен wrapper и стратегия opt‑in `OTEL_SEMCONV_STABILITY_OPT_IN` citeturn23view0turn25view0 | **Да, но через wrapper** |
| Корреляция ретраев | Только trace_id | Удобно в traces | При sampling часть цепочки исчезает; ретраи/DLQ могут уйти в другой trace; нужна независимая «нить» workflow | **Нет** |
| Корреляция ретраев | correlation_id + message_id + attempt (+ trace links) | Устойчиво к sampling; DLQ/ретраи видимы как отдельные события; легче делать reconciliation | Требует стандартных полей, строгой low-cardinality политики для метрик | **Да** |
| Baggage | Включить и тащить бизнес‑ID | Удобно для downstream анализа citeturn29view0 | Риск утечки чувствительных данных: baggage автоматически уезжает в headers; нет встроенных integrity checks; может выйти за пределы сети citeturn29view0 | **По умолчанию: выключено/минимум** |
| Метрики backlog | Только app‑metrics | Простота | Не видите queue depth/lag; теряете saturation‑сигнал async‑системы | **Нет** |
| Метрики backlog | Broker/platform metrics + app metrics | Видите лаг, глубину, возраст; можно алертить на backlog и SLA обработки citeturn4view4turn21search0turn26search2 | Требует интеграции с платформой (CloudWatch/JMX/Prometheus exporters) | **Да** |
| Retry policy | Без jitter | Проще | Коррелированные ретраи → шторма, продление восстановления citeturn28view0turn28view1 | **Никогда** |
| Retry policy | Capped backoff + jitter + лимит ретраев | Наиболее устойчивый массовый дефолт; снижает корреляцию и нагрузку citeturn28view1turn28view0 | Нужны метрики ретраев и причины; непредсказуемость времени повторов для бизнес‑потоков | **Да** |

## Набор правил MUST / SHOULD / NEVER для LLM

Ниже — правила, которые нужно положить в LLM‑instruction docs для template, чтобы модель не «домысливала» и не портила observability.

**MUST**
- Всегда использовать W3C tracecontext и OpenTelemetry propagators для прокидывания контекста через сообщения по key/value carrier’у; Inject/Extract должны идти через TextMap‑модель. citeturn30view0turn30view1turn35search2  
- Для каждой отправки сообщения создавать tracing span по messaging conventions (operation.type=`send`, корректный span kind) и инжектить trace context в headers/attributes сообщения. citeturn25view0turn25view3turn24view0  
- Для каждого обработчика сообщения создавать `process` span kind=`CONSUMER`; при batch обработке добавлять span links на все upstream message contexts. citeturn25view0turn25view2turn1search13  
- Заполнять `messaging.message.id` и `messaging.message.conversation_id` (если есть) в span attributes; различать Kafka message key и message.id (key не уникален). citeturn24view0turn24view2  
- Писать структурные логи, коррелируемые с trace/span (TraceId/SpanId в LogRecord или эквивалентные поля в structured logs/pipeline), и включать message_id/correlation_id/attempt как поля лог‑события (без использования их как metric label). citeturn31view0turn33view0  
- Добавлять и поддерживать обязательные app‑метрики: `messaging.process.duration`, `messaging.client.consumed.messages`, `messaging.client.sent.messages`, `messaging.client.operation.duration`, плюс стандартные outcome/retry/dlq/idempotency метрики template. citeturn23view0turn24view0  
- Все retry политики должны быть capped exponential backoff + jitter + лимит попыток; ретраи должны быть видимы метриками и логами. citeturn28view1turn28view0  
- Для SQS использовать `ApproximateReceiveCount`/timestamps при их наличии для attempt/end‑to‑end latency и корректно учитывать visibility timeout. citeturn34search0turn34search13  
- Для RabbitMQ использовать delivery metadata (`redelivered`) и message properties (Message ID/Correlation ID/Headers); для DLX учитывать `x-death`/delivery limits (особенно quorum queues). citeturn19view0turn18view0turn20view0  

**SHOULD**
- Реализовывать «telemetry contract» через внутренний пакет (например, `internal/observability/async`), чтобы код не разъезжался и semconv‑миграции (Development → Stable) делались централизованно, включая поддержку `OTEL_SEMCONV_STABILITY_OPT_IN`. citeturn23view0turn25view0  
- Для periodic jobs добавлять deterministic jitter, чтобы разнести пики; логировать schedule delay и измерять фактический runtime. citeturn28view1turn27view0  
- Для DLQ/poison messages логировать и метрифицировать «почему» (bounded categories) и обеспечивать «safe re-drive»: redrive только после фикса root cause. Это особенно важно для систем, где DLX может drop’нуть сообщение при цикле или недоступности цели. citeturn18view0turn4view4  
- Использовать baggage только при явном бизнес‑требовании и после security review; минимизировать набор ключей и не доверять входящему baggage. citeturn29view0turn31view1  

**NEVER**
- Никогда не класть message_id/correlation_id/user_id в Prometheus/OTel metric labels (высокая кардинальность). citeturn33view0  
- Никогда не логировать PII/секреты/токены и не писать «сырые headers» без фильтрации; санитизировать вводимые поля против log injection. citeturn31view1turn29view0  
- Никогда не генерировать новый correlation_id на каждый retry как замену «стабильной нити» операции; retry должен сохранять correlation и message_id, а attempt должен увеличиваться. (Рекомендация основана на требованиях идемпотентности/at-least-once и наблюдаемости ретраев.) citeturn4view5turn28view1  
- Никогда не делать «безлимитные ретраи без jitter» — риск retry storms и cascading failures. citeturn28view1turn27view1  
- Для RabbitMQ: никогда не использовать polling (`basic.get`) в production; это явно помечено как strongly recommended against. citeturn19view0  

## Concrete good / bad examples и типичные LLM‑ошибки

### Good: инжект/экстракт W3C tracecontext в сообщение через TextMapCarrier

Пример показывает принцип: carrier — это abstraction над headers/attributes, а Inject/Extract — через глобальный propagator. Интерфейс carrier в Go соответствует `TextMapCarrier` (Get/Set/Keys). citeturn35search2turn30view1

```go
package tracingmsg

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// Generic carrier on top of map[string]string.
// Adapt this to Kafka headers / AMQP headers / SQS message attributes.
type MapCarrier map[string]string

func (c MapCarrier) Get(key string) string { return c[key] }
func (c MapCarrier) Set(key, value string) { c[key] = value }
func (c MapCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

func Inject(ctx context.Context, headers map[string]string) {
	carrier := MapCarrier(headers)
	// If headers map is reused across retries, clear propagation keys first.
	// (Propagators spec: reused, retryable carriers should clear fields.) 
	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

func Extract(ctx context.Context, headers map[string]string) context.Context {
	carrier := MapCarrier(headers)
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// Optional: ensure carrier implements propagation.TextMapCarrier at compile time.
var _ propagation.TextMapCarrier = MapCarrier(nil)
```

Почему это good: соответствует Propagators API и позволяет переносить trace context через «сообщения, которыми обмениваются приложения», а не только HTTP. citeturn30view1turn25view3

### Good: batch processing с Span Links вместо «выбрать одного родителя»

OTel прямо иллюстрирует batch receiving через links: consumer span должен линковаться на каждый producer span. citeturn25view2turn1search13

Псевдокод‑набросок (идея важнее конкретной библиотеки брокера):

```go
// For each message in batch:
// 1) Extract remote context
// 2) Collect SpanContext for links
// 3) Create one "process batch" span with links (one per message)

links := make([]trace.Link, 0, len(msgs))
for _, msg := range msgs {
    ctxRemote := Extract(context.Background(), msg.Headers)
    sc := trace.SpanContextFromContext(ctxRemote)
    if sc.IsValid() {
        links = append(links, trace.Link{SpanContext: sc})
    }
}

ctx, span := tracer.Start(
    context.Background(),
    "consume "+destination,
    trace.WithSpanKind(trace.SpanKindConsumer),
    trace.WithLinks(links...),
)
// ... process batch ...
span.End()
```

Точка стандарта: **не** делайте batch‑обработку «дочерней» только от одного сообщения — это ломает причинность и делает расследования «слепыми». citeturn25view2turn25view3

### Bad: высокая кардинальность метрик и «скрытая катастрофа»

Типичная галлюцинация LLM: «давайте добавим `message_id` лейблом к метрике, чтобы проще искать». Это нарушает best practices Prometheus и может взорвать количество time series. citeturn33view0

```go
// BAD: message_id is unbounded cardinality
msgProcessed.Add(ctx, 1,
    metric.WithAttributes(
        attribute.String("message_id", msgID),
        attribute.String("queue", queueName),
    ),
)
```

Как правильно: `message_id` остаётся в logs/traces, а метрики агрегируются по bounded измерениям (queue, outcome, error.type, consumer_group и т.п.). citeturn31view0turn23view0turn33view0

### Anti-patterns, которые маскируют реальные проблемы в EDA

1) **«DLQ есть, но мы её не наблюдаем»**: в SQS нельзя считать, что рост `NumberOfMessagesSent` отражает auto‑move в DLQ; AWS рекомендует использовать `ApproximateNumberOfMessagesVisible` для контроля состояния DLQ. citeturn4view4

2) **«Инфинитный ретрай‑луп»**: 
- без bounded retries и jitter вы получаете retry storms и усложняете recovery, что прямо описано в guidance Amazon; citeturn28view1turn28view0  
- для RabbitMQ возможны циклы DLX, и брокер может drop’нуть сообщение при обнаружении цикла без rejection в цикле; это превращается в «тихую потерю» без правильных алертов. citeturn18view0  

3) **«Idempotency молчит»**: в at‑least‑once системах вы обязаны проектировать обработку идемпотентной; SQS прямо говорит «design your applications to be idempotent» при at‑least‑once delivery. Если вы не измеряете dedupe/duplicate outcomes, то деградации (например, рост редоставок из‑за visibility timeout или падений consumer) будут выглядеть как «нормальный throughput», пока не проявятся в бизнес‑дублированиях. citeturn4view5turn34search13

4) **«Polling вместо consumers»**: для RabbitMQ pull‑polling (`basic.get`) сильно неэффективен и не рекомендуется в production. Там же указано, что это допустимо в интеграционных тестах, но не как штатная модель. citeturn19view0

5) **«Потеря причины в Kafka»**: путать `message key` и `message id` — ошибка. OTel подчёркивает, что Kafka key не уникален и отличается от `messaging.message.id`. Если LLM начнёт использовать key как уникальный идентификатор либо как метку метрик — вы получите неверные корреляции и/или кардинальность. citeturn24view0turn4view2

6) **«Логи без защиты»**: отсутствие санитизации/защиты логов приводит к log injection и утечкам чувствительных данных; OWASP требует санитизировать данные событий, защищать логи и предотвращать DoS через логирование. citeturn31view1turn33view0

## Review checklist для PR/code review и что вынести в отдельные файлы

### PR / Code Review checklist для async components

Чеклист должен являться «definition of done» для producer/consumer/job‑кода:

- Trace propagation реализован через Inject/Extract по W3C tracecontext; отсутствует «контекст теряется на async boundary». citeturn30view0turn25view3  
- Producer создаёт span send/create с корректным span kind и messaging attributes; consumer создаёт process span kind CONSUMER; batch использует span links на все upstream contexts. citeturn25view0turn25view2turn24view0  
- Ошибки классифицированы low-cardinality (`error.type`), отражены в span статусе/ошибках и в метриках outcome/retry/dlq. citeturn23view0turn27view0  
- В логах присутствует корреляция TraceId/SpanId + message_id/correlation_id/attempt; отсутствуют secrets/PII; есть санитизация, нет log injection. citeturn31view0turn31view1turn29view0  
- Метрики не содержат high‑cardinality label’ов; соблюдаются правила именования/единиц; есть app‑metrics (process duration, consumed/sent) и платформенные backlog/lag метрики в мониторинге. citeturn23view0turn33view0turn4view4turn21search0turn26search2  
- Retry политика: capped exponential backoff + jitter + лимит ретраев; есть защита от retry multiplication по слоям (ретраи в одной точке стека) и токен‑бакет/локальные лимиты при необходимости. citeturn28view1turn28view0  
- Для DLQ реализованы: метрики глубины/возраста, логирование причины, отсутствие бесконечных циклов, и понятный процесс redrive. Для RabbitMQ проверено, что DLX цель существует и учтено поведение x‑death/циклов; для SQS учтены нюансы метрик DLQ и maxReceiveCount. citeturn18view0turn4view4turn1search3  
- Для SQS учтён visibility timeout как источник редоставок; обработка идемпотентна. citeturn34search13turn4view5  
- Для Kafka есть наблюдаемость lag (`records-lag-max` или эквивалент) и throughput, а алерты/дашборды связывают backlog с ошибками/latency. citeturn21search0turn21search1  
- Для RabbitMQ есть наблюдаемость depth (`messages_ready`, `messages_unacknowledged`) и consumer capacity; polling не используется. citeturn26search2turn19view0turn26search3  

### Что оформить отдельными файлами в template repo

Чтобы этот стандарт стал «почти напрямую» превращаемым в `docs/` и repo conventions, результат стоит разложить на следующие файлы:

- `docs/standards/observability_async.md` — **этот стандарт**: обязательные traces/logs/metrics для producers/consumers/jobs + DLQ/lag/retry patterns, с примерами и анти‑паттернами. Источники должны быть закреплены (OTel specs, SRE book, AWS/RabbitMQ/Kafka docs). citeturn25view0turn27view0turn4view4turn21search0turn18view0  
- `docs/llm/observability_async_instructions.md` — MUST/SHOULD/NEVER правила из этого документа в «promptable» виде, с запретами на high‑cardinality labels и на unsafe logging. citeturn33view0turn31view1turn29view0  
- `internal/observability/async/` — маленький пакет‑обёртка:
  - `propagation.go`: готовые carrier’ы для популярных клиентов (Kafka headers, AMQP headers/table, SQS message attributes) и единый API `Inject/Extract`, основанный на TextMapCarrier. citeturn30view1turn35search2turn5search1turn19view0  
  - `spans.go`: helper’ы для создания send/receive/process span’ов и для batch links (единый стиль span names и атрибутов), а также слой абстракции на случай изменения semconv (Development → Stable). citeturn25view0turn25view2turn23view0  
  - `metrics.go`: создание и запись обязательных метрик (`messaging.*` + async outcome/retry/dlq/idempotency) с жёстким контролем допустимых атрибутов (bounded). citeturn23view0turn33view0turn35search0  
- `docs/runbooks/async_backlog_dlq.md` — операционный runbook: «как читать lag/queue depth/age», «как расследовать DLQ», «как отличить consumer lag от upstream outage», с привязкой к обязательным метрикам из стандарта (SQS/RabbitMQ/Kafka). citeturn4view4turn21search1turn26search2turn28view1  
- `deploy/otel-collector/` (или `infra/otel/`) — минимальная production‑ориентированная конфигурация Collector с security best practices (TLS/auth, минимальный набор компонентов). citeturn32view0turn32view1  
- `.github/pull_request_template.md` — краткая версия PR checklist «Async Observability» (ссылкой на `docs/standards/observability_async.md`). citeturn27view0turn23view0turn33view0