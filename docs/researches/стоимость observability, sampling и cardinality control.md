# Cost-aware observability для production Go микросервиса: sampling, cardinality и контроль стоимости

## Scope: когда этот подход применять, а когда нет

Этот стандарт предназначен для greenfield микросервиса на Go, который должен быть «production-ready» сразу после клонирования репозитория и при этом активно разрабатываться с помощью LLM-инструментов. Основная цель — заранее зафиксировать ограничения и «boring defaults», чтобы telemetry (метрики/логи/трейсы) не превращалась в неконтролируемую статью расходов и не деградировала производительность observability backend из‑за cardinality/volume explosion. Это особенно важно, потому что даже «невинные» изменения в instrumentation (ещё один label, перенос значения из body в атрибут, «удобный» span name из URL) могут умножить число time series и стоимость хранения/запросов. citeturn5view1turn11view0turn5view2

Подход рекомендуется применять, когда вы используете **entity["organization","OpenTelemetry","cncf observability project"]** как стандартную библиотеку instrumentation и/или передаёте сигналы в TSDB/лог-хранилище, чувствительные к cardinality и объёму (как минимум, **entity["organization","Prometheus","cncf monitoring system"]**-совместимая метрика-модель и labelsets). У **Prometheus** каждый уникальный labelset создаёт отдельный time series с затратами по RAM/CPU/disk/network; а гистограммы дополнительно создают «множество» рядов на один инструмент. citeturn5view1turn11view0

Подход особенно критичен для:
- внешних HTTP API (атрибуты из заголовков/URL могут быть под контролем атакующего и провоцировать cardinality-limit атаки на метрики), citeturn12view3  
- central logging, где индекс строится по labels и высокая cardinality ухудшает performance/cost (например, **entity["organization","Grafana Loki","log aggregation project"]** прямо предупреждает, что high-cardinality labels приводят к огромному индексу и множеству мелких chunk’ов, резко снижая cost‑effectiveness), citeturn5view2  
- multi-tenant архитектур, где «tenant_id как label» быстро превращается в миллионы комбинаций, citeturn5view1turn2search23turn2search7  
- сред с сильными privacy/PII требованиями: telemetry может «случайно» начать содержать персональные данные/секреты, и тогда стоимость — это уже не только деньги, но и риск комплаенса/инцидентов. citeturn5view4turn16view1turn5view5

Не стоит применять документ «как есть» (без адаптаций), если:
- ваш продукт/организация осознанно выбирает high-cardinality observability (например, полноценный событийный анализ «по каждому пользователю») и готова платить за это инфраструктурой/лицензиями и обеспечить строгие guardrails на ingestion/query; базовые правила ниже всё равно полезны, но defaults по sampling/retention, вероятно, будут другими. citeturn5view1turn5view2  
- сервис — краткоживущий прототип/скрипт, где трассировка 100% и «логируем всё» оправданы временем разработки; но даже тогда действуют ограничения по PII и секретам. citeturn5view4turn16view0turn16view1

## Recommended defaults для greenfield template

Ниже — практичные дефолты, которые следует «зашить» в шаблон: часть — как код/константы, часть — как конфиги pipeline (Collector/agents/backends), часть — как LLM‑правила.

### Telemetry budget и «глобальные ограничения» как default contract

1) **Cardinality budget для метрик**: ориентир — держать cardinality метрик «около 10» и избегать роста >100 без отдельного дизайна/обоснования. Это не «красивое число», а практический safety‑rail: у **Prometheus** каждый labelset добавляет стоимость, и рекомендации прямо предлагают начинать с отсутствия labels и добавлять их только по мере появления конкретных use case. citeturn5view1turn1search9

2) **Span/Log attribute limits** включать и задавать явно через env (и документировать как контракт шаблона). В спецификации SDK env vars у **OpenTelemetry** есть лимиты на количество атрибутов и длину значений, отдельно для Span и LogRecord; по умолчанию для количества атрибутов часто стоит 128, а длина значений — без лимита (что опасно для стоимости/PII). citeturn13search3turn0search2  
Практический default для template:  
- ограничить **количество** атрибутов управляемыми лимитами (оставить 128 как старт, но «включить контроль» и мониторить drops),  
- ограничить **длину строковых значений** (опционально, но крайне полезно против «случайного» логирования JSON/HTML/stacktrace в атрибут). citeturn13search3turn13search27

3) **Fail-closed redaction на pipeline уровне** (не в коде сервиса) как «страховка от LLM и человеческих ошибок». В contrib redaction processor:  
- допускается строгий allowlist ключей (если empty — удаляется всё),  
- есть шаблоны blocked_key_patterns/blocked_values,  
- есть url_sanitizer для снижения cardinality (санитайзинг URL в атрибутах и span name),  
- есть db_sanitizer для санитайзинга db.statement/команд и т.п. citeturn8view0turn5view4

### Метрики: low-cardinality by design

1) **HTTP server метрики**: в template зафиксировать семантические метрики и bucket layout, чтобы LLM не «придумывала» их. В HTTP metrics semantic conventions указано, что `http.server.request.duration` — required и для неё даны рекомендуемые explicit bucket boundaries (совет параметра границ бакетов): `[0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10]` секунд. citeturn12view3  
Практический default: использовать ровно этот набор бакетов для latency‑гистограмм HTTP (и по возможности переиспользовать его для большинства «duration» гистограмм сервиса, если нет сильных причин иначе).

2) **`http.route` только как route template**: `http.route` в semconv прямо требует low-cardinality и placeholder для динамических сегментов. Более того, запрещено подставлять URI path вместо `http.route`, если framework не умеет route template. citeturn12view0turn12view3  
Практический default: в вашем HTTP router слое обязателен механизм получения route template (например, `/users/{id}`), иначе `http.route` не ставить вообще, но **никогда** не ставить туда `r.URL.Path`.

3) **Не включать «опасные» dimensions из заголовков по умолчанию**: в HTTP metrics semconv есть предупреждения, что некоторые атрибуты основаны на HTTP headers, и opt‑in к ним может позволить атакующему вызвать cardinality‑деградацию полезности метрик. citeturn12view3  
Практический default: не добавлять в metric‑attributes значения, которые могут прямо следовать из невалидированных/неограниченных headers (в частности Host/порт), пока не определён жёсткий allowlist/нормализация.

4) **`error.type` как единственный «ошибочный» dimension и только низкой кардинальности**: “Recording errors” рекомендует при ошибке выставлять `error.type` на span и метрики длительности; при успехе `error.type` не включать, чтобы можно было фильтровать ошибки. Также рекомендуется репортить одну метрику, включающую success+failure, а не отдельные серии. citeturn18view0  
Практический default: `error.type` в вашем сервисе должен быть **перечислением/классификатором** (например, `timeout`, `canceled`, `validation`, `dependency_unavailable`, `internal`) и никогда не должен быть сырым `err.Error()`.

### Гистограммы: bucket design как часть API‑контракта

Классические гистограммы в **Prometheus** дорогие по времени series: документ прямо подчёркивает, что один histogram создаёт множество time series; в сравнительной таблице — «один ряд на bucket» (в дополнение к `_sum` и `_count`). citeturn11view0turn5view1  
Практический default:
- на гистограммах **заранее** фиксировать bucket boundaries (не «подбирать на глаз» LLM),  
- держать число бакетов умеренным (типично 10–15) и помнить, что увеличение бакетов мультипликативно влияет на общее число рядов, особенно при наличии labels. citeturn11view0turn5view1  
- добавлять границы вокруг SLO‑точек: **Prometheus** прямо показывает пример: если SLO 95% ≤ 300ms, нужна граница bucket `le="0.3"`, чтобы легко считать долю запросов ≤ SLO. citeturn11view0

Дополнительно (не как дефолт, а как осознанный выбор): **Prometheus** отмечает, что документ по histograms/summaries предшествует native histograms и что native histograms стали стабильными в v3.8 после экспериментальной фазы (в v2.40). Это потенциально важно для стоимости/series count, но экосистема поддержки и совместимость могут отличаться. citeturn11view0turn0search13

### Трейсы: sampling как механизм контроля стоимости

1) **Head sampling (в SDK) как минимальный boring default**. В Go документации **OpenTelemetry** sampling описан как ограничение числа создаваемых span; рекомендуют принимать решение в начале trace и распространять его между сервисами. Для production предлагается комбинация `ParentBased` + `TraceIDRatioBased`. citeturn9view2turn4search8  
Практический default для template:
- dev/local: AlwaysSample (чтобы дебажить без сюрпризов), citeturn9view2  
- prod: env‑конфиг `OTEL_TRACES_SAMPLER=parentbased_traceidratio`, `OTEL_TRACES_SAMPLER_ARG=<p>` (вероятность p в [0..1]), citeturn4search8turn9view2  
- “p” в шаблоне как стартовое мнение: 0.01 (1%) для high‑traffic публичных сервисов; документировать, что это **opinionated default** и должно калиброваться на реальном трафике/стоимости.

2) **Tail sampling (в Collector) как «умный» sampling** для сохранения ошибок/латентных outliers. В концептах **OpenTelemetry** приводят примеры tail sampling: всегда сохранять трейсы с ошибками; сохранять по общей latency; по значениям атрибутов (например, больше сохранять для новой версии). citeturn0search3  
Практический default для template: включать tail sampling только если есть инфраструктура для этого (Collector/Alloy) и вы готовы поддерживать policies.

3) **Операционные ограничения tail sampling** обязательно фиксировать в шаблонных документах: tail sampling processor требует, чтобы все spans одного trace попадали в один и тот же экземпляр Collector (иначе решение «по trace целиком» неисполнимо). Processor поддерживает политики `status_code`, `latency`, `probabilistic`, `rate_limiting`, `bytes_limiting`, может ограничивать `maximum_trace_size_bytes` для защиты памяти и имеет параметры ожидания решения/числа traces в памяти. citeturn19view0  
Практический default: если включаете tail sampling — сразу задайте защитные лимиты (`maximum_trace_size_bytes`, rate/bytes limits) и документируйте требования к routing/конфигурации. citeturn19view0

4) **Custom sampler в приложении — не дефолт**. В Go docs подчёркивают: `ShouldSample` вызывается синхронно при создании каждого span; тяжёлые вычисления в нём вредны. При написании кастомного sampler критично сохранять `tracestate`, иначе ломается контекст‑пропагация. citeturn9view2  
Практический default: в template разрешить custom sampler только по explicit decision record, иначе запрещать.

### Логи: контроль объёма и индекса

1) **Структурированные логи и “log only what you need”**. **OWASP** подчёркивает, что важно не логировать «слишком много или слишком мало»; объём/детали должны определяться целями логирования (security/ops) и контролироваться. citeturn16view0turn16view1

2) **Labels в Loki — только low-cardinality**. **Grafana Loki** объясняет, что содержимое log line не индексируется; индекс строится по streams, определяемым labels. High cardinality приводит к огромному индексу и множеству мелких chunk’ов, ухудшая производительность и стоимость. citeturn4search3turn5view2  
Практический default для template‑доков:
- labels описывают источник (namespace/cluster/region/service), значения должны быть bounded, citeturn5view2  
- request‑specific поля (request_id, user_id, ip, pod UID) — **не labels**. citeturn5view2turn1search9

3) **Structured metadata вместо labels для «нужных, но дорогих» полей**. В Loki structured metadata позволяет прикреплять метаданные без индексации; прямо приводятся примеры высококардинальных полей вроде pod name или PID и отмечается, что это удобно для полей, часто используемых в запросах, но слишком дорогих как labels. citeturn10search3turn10search7  
Практический default: если в вашей организации стандарт — Loki, в template‑docs фиксировать: high-cardinality идентификаторы кладём в structured metadata либо внутрь JSON log line (и извлекаем при поиске), но не в labels. citeturn10search3turn5view2

### Exemplars: корреляция метрик и трейсов без «телеметрического шторма»

Exemplar в спецификации **OpenTelemetry** описан как записанное значение, связывающее OpenTelemetry context с metric event; один из use case — связывать traces и metrics. citeturn3search5turn3search24  
Практический default: использовать exemplars **только** на ключевых latency‑гистограммах (HTTP server duration, критические dependency duration) и только при соблюдении двух условий:
- метрика имеет строго низкую cardinality по labels (иначе exemplar blob тоже размножается по сериям), citeturn5view1turn11view0  
- sampling в traces настроен так, что «попасть» в trace по exemplar имеет смысл (иначе будет много «traceID без следа»). citeturn9view2turn3search5  

Важно понимать memory trade-off на стороне **Prometheus**: exemplar storage реализован как фиксированный in‑memory circular buffer на все series; в документации feature flags указано, что exemplar с одним `trace_id` занимает примерно 100 байт памяти в in-memory exemplar storage. citeturn3search0  
Практический default: exemplars — opt‑in feature флагом/конфигом, с заранее установленным budget на storage/exemplars.

### Retention: по умолчанию «не бесконечно»

Retention — ключевой рычаг стоимости хранения.

- **Prometheus**: в storage docs указано, что если retention не задан флагом/конфигом, retention time по умолчанию 15d. citeturn10search0  
- **Grafana Loki**: в документации по retention сказано, что по умолчанию `compactor.retention-enabled` не установлен, поэтому логи «живут вечно». citeturn2search1  
- **Grafana Mimir**: в docs указано, что по умолчанию метрики в object storage никогда не удаляются и потребление хранилища будет расти; retention нужно настроить. citeturn10search13  
- **Grafana Tempo** (как пример trace backend): в конфигурации `block_retention` по умолчанию 336h (14 дней), и для оценки storage прямо приводится грубая формула: ingested bytes per day × retention days = stored bytes. citeturn15view0turn15view1  

Практический default для template:
- в docs/infra‑гайдах требовать явной настройки retention для logs/metrics/traces в выбранном стеке; иначе «непредсказуемая стоимость» — вопрос времени, а не вероятности. citeturn2search1turn10search13turn10search0turn15view1  
- принять стартовые окна retention, согласованные с дефолтами популярных OSS‑бекендов (пример: метрики 15 дней, трейсы 14 дней), а далее менять только через решение с оценкой стоимости. citeturn10search0turn15view0

### Multi-tenant telemetry: изоляция без «tenant_id как label»

Если ваш template рассчитан на multi-tenant:
- Loki multi-tenancy использует заголовок `X-Scope-OrgID` и рекомендует разумно ограничивать размер tenant ID (примерно 20 байт «обычно достаточно»), citeturn2search23  
- Mimir также multi-tenant и берёт tenant ID из `X-Scope-OrgID` (через прокси), citeturn2search7turn2search31  
- Tempo в конфиге прямо описывает, что multitenancy требует `X-Scope-OrgID` на всех запросах (при включении `multitenancy_enabled`). citeturn14view0turn15view0  

Практический default:
- tenant isolation реализуется на ingestion/query уровне через tenant header/проект, а не путём добавления `tenant_id` в labels метрик (это почти всегда взрывает cardinality); citeturn5view1turn2search7turn2search23  
- если бизнес требует per-tenant метрик — тогда это отдельный, осознанный набор агрегированных метрик с очень ограниченной размерностью и явными квотами. citeturn5view1turn1search9

### Privacy/PII: “минимизируй сбор” + централизованная редакция

**OpenTelemetry** формулирует принцип data minimization: собирать только то, что служит наблюдаемости, избегать персональных данных без необходимости, рассматривать агрегирование/анонимизацию и регулярно пересматривать атрибуты. Также описаны Collector processors для удаления/модификации/фильтрации/редакции и приводятся примеры хэширования/удаления user‑полей и предупреждение о рисках «анонимизации хешами» при низкой энтропии. citeturn5view4

Для URL semconv: `url.full` **не должен** содержать credentials вида `https://user:pass@…`; при наличии credentials их следует редактировать. Также указаны query parameter keys, которые следует редактировать по умолчанию (например, `AWSAccessKeyId`, `Signature`, `sig`, `X-Goog-Signature`), и требование сохранять ключ параметра при редактировании значения. citeturn5view5  
Практический default: не писать `url.full/url.query/url.original` в метрики и любые индексы без явной санитизации/редакции; в traces — только если включена централизованная санитизация. citeturn5view5turn8view0

Для security logging: **OWASP** предупреждает о риске утечки sensitive info в логах (PII/PHI) и о необходимости корректной encoding, чтобы избежать атак на logging/monitoring системы. citeturn16view1  
Практический default: policy «PII/секреты не попадают ни в логи, ни в трейсы, ни в метрики» + central redaction/filter на pipeline как mandatory барьер. citeturn16view1turn5view4turn8view0

## Decision matrix / trade-offs

Ниже — ключевые решения, которые должны быть формализованы в template‑docs как «по умолчанию», с явными условиями, когда их менять.

**Labels/attributes: добавить размерность vs стоимость**
- Больше labels = больше time series и выше RAM/CPU/disk/network стоимость (metrics) и больше индекс/streams (logs); поэтому default — минимальная размерность. citeturn5view1turn5view2  
- Если нужен drill-down, предпочтительнее: (а) отдельная агрегированная метрика для конкретного use case, либо (б) аналитика вне TSDB, чем добавление unbounded label. citeturn5view1turn1search9

**Гистограммы: точность/SLO‑полезность vs series count**
- Гистограммы позволяют агрегировать и вычислять percentiles на сервере, но создают множество time series; один bucket = один series помимо `_sum/_count`. citeturn11view0  
- Summary дешевле на сервере, но не агрегируется корректно между инстансами; гистограммы обычно предпочтительнее в микросервисах с репликацией. citeturn11view0  
- Native histograms потенциально меняют экономику, но требуют проверки зрелости/поддержки в вашем стеке; для «boring defaults» лучше классические гистограммы с фиксированными buckets. citeturn11view0turn0search13

**Trace sampling: head vs tail**
- Head sampling дешевле и проще (решение в начале trace, можно распространять между сервисами); для production рекомендуют ParentBased+TraceIDRatioBased как базовый вариант. citeturn9view2turn4search8  
- Tail sampling лучше сохраняет ошибки/slow traces, но требует, чтобы все spans trace попали в один Collector, хранит traces в памяти до решения и нуждается в ограничителях (`num_traces`, `maximum_trace_size_bytes`, rate/bytes limiting). citeturn19view0turn0search3

**Логи: индексация/labels vs поисковые сценарии**
- В Loki индексируется только labels, а не текст log line; high-cardinality labels приводят к большой цене индекса/чанков. Поэтому default — labels только для «источника», а request‑специфичные данные — в structured metadata или внутри log line. citeturn4search3turn5view2turn10search3

**Retention: диагностика «давних» инцидентов vs стоимость**
- Без явной retention некоторые системы фактически становятся «бесконечными» по хранению (Loki logs, Mimir object storage), что гарантирует рост стоимости. citeturn2search1turn10search13  
- В Tempo приводится простой способ прикинуть storage: ingested bytes/day × retention days. Это good enough для первичной оценки trade-off. citeturn15view1  
- Prometheus имеет разумный default 15d, но это не значит «подходит всем»; менять retention следует только вместе с планом по capacity и потребностями расследований. citeturn10search0

**Multi-tenancy: изоляция vs cardinality**
- Multi-tenant лучше решать на уровне tenancy header/проектов и per-tenant retention/quotas, а не через `tenant_id` label повсюду. Loki/Mimir/Tempo документируют модель с `X-Scope-OrgID`. citeturn2search23turn2search7turn14view0  
- Если per-tenant breakdown критичен, это отдельный продуктовый/финансовый выбор: ограниченные метрики + квоты + строгая размерность. citeturn5view1turn1search9

## Набор правил в формате MUST / SHOULD / NEVER для LLM

Эти правила должны быть помещены в LLM-instruction docs репозитория и считаться частью engineering standard.

### MUST

- LLM MUST проектировать метрики так, чтобы cardinality была bounded: каждый новый labelset — новый time series со стоимостью, поэтому labels должны быть малочисленными и низкокардинальными. citeturn5view1turn1search9  
- LLM MUST использовать `http.route` только как route template с placeholders; запрещено подставлять `url.path`/`r.URL.Path` вместо route template. citeturn12view0turn12view3  
- LLM MUST применять фиксированный bucket layout для HTTP latency (как в semconv) и не менять его без отдельного решения; bucket boundaries: `[0.005 … 10]` секунд. citeturn12view3  
- LLM MUST помнить, что histogram создаёт множество time series и «один ряд на bucket» (плюс `_sum/_count`), поэтому нельзя добавлять гистограммы «везде» и нельзя делать десятки/сотни buckets. citeturn11view0turn5view1  
- LLM MUST конфигурировать trace sampling через стандартные механизмы SDK: для production — `ParentBased` + `TraceIDRatioBased`, решение в начале trace и пропагация между сервисами. citeturn9view2turn4search8  
- LLM MUST соблюдать требования к custom sampler: сохранять tracestate и не делать тяжёлых операций в `ShouldSample` (оно вызывается синхронно при создании каждого span). citeturn9view2  
- LLM MUST при ошибках применять стандарт “Recording errors”: при ошибке — выставлять span status `Error`, задавать `error.type`, на метриках включать `error.type` только для неуспешных операций; successes — без `error.type`. citeturn18view0turn12view3  
- LLM MUST не помещать чувствительные данные в telemetry и следовать data minimization; если сбор потенциально чувствительных атрибутов неизбежен — MUST предусматривать централизованную редакцию/удаление на Collector pipeline. citeturn5view4turn16view1turn8view0turn5view5  
- LLM MUST держать включёнными и документированными SDK limits для атрибутов (span/logrecord count/length) и не добавлять код, который создаёт «лог-шторм» при усечении/дропе атрибутов. citeturn13search3turn13search27  

### SHOULD

- LLM SHOULD использовать семантические конвенции и стабильные ключи атрибутов; для HTTP метрик — атрибуты из semconv и избегать opt‑in атрибутов из headers по умолчанию. citeturn12view3turn9view1  
- LLM SHOULD выносить instrumentation в центральные helper’ы/обёртки шаблона (единый пакет metrics/tracing/logging), чтобы изменения labels/атрибутов были «в одном месте» и проходили review по checklist. Обоснование — риск и стоимость от каждого нового labelset. citeturn5view1turn5view2  
- LLM SHOULD включать exemplars только для ключевых latency‑гистограмм и только с явным budget/флагами на стороне backend (exemplar storage). citeturn3search5turn3search0  
- LLM SHOULD проектировать multi-tenancy через tenancy‑модель backends (`X-Scope-OrgID`), а не через «tenant_id в labels». citeturn2search23turn2search7turn14view0turn5view1

### NEVER

- LLM MUST NEVER использовать в labels/metric-attributes значения с высокой cardinality или unbounded множеством: user IDs, email, request_id, UUID, IP, произвольные query params, timestamps. citeturn1search9turn5view1turn5view2  
- LLM MUST NEVER добавлять в метрики/индексируемые поля значения из HTTP headers без строгой нормализации/allowlist (это может позволить cardinality-атаки и деградацию метрик). citeturn12view3turn5view2  
- LLM MUST NEVER писать credentials в `url.full` и не должна тащить query string «как есть»; если URL нужно сохранять — требуется санитизация/редакция. citeturn5view5turn8view0  
- LLM MUST NEVER записывать “сырые” SQL/команды с параметрами/PII в telemetry без включённого db_sanitizer/редакции на pipeline. citeturn8view0turn5view4  
- LLM MUST NEVER дублировать exception/error в нескольких местах (например, и как event, и как log, и как status description), особенно для «handled» ошибок; это увеличивает объём и ухудшает сигнал/шум. citeturn18view0turn17search3

## Concrete good / bad examples, где уместно — на Go

### Good: HTTP latency histogram с низкой cardinality и route template

```go
package observability

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.Meter("service/observability")
	// Bucket boundaries: fixed contract for http.server.request.duration (seconds).
	httpServerDuration metric.Float64Histogram
)

func init() {
	var err error
	httpServerDuration, err = meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10,
		),
	)
	if err != nil {
		panic(err)
	}
}

// RecordHTTPServerDuration records one observation with bounded attributes.
// routeTemplate MUST be a low-cardinality template like "/users/{id}".
func RecordHTTPServerDuration(ctx context.Context, r *http.Request, routeTemplate string, statusCode int, start time.Time, errType string) {
	attrs := []attribute.KeyValue{
		attribute.String("http.request.method", r.Method),
		attribute.String("http.route", routeTemplate),
		attribute.Int("http.response.status_code", statusCode),
	}

	// Only attach error.type when the operation actually failed.
	if errType != "" {
		attrs = append(attrs, attribute.String("error.type", errType))
	}

	httpServerDuration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(attrs...))
}
```

Почему это «good»:
- фиксированные bucket boundaries совпадают с рекомендацией semconv для `http.server.request.duration`; это снижает вероятность «самодельных бакетов» от LLM и упрощает эксплуатацию. citeturn12view3  
- `http.route` — только шаблон, низкая cardinality; это прямое требование semconv. citeturn12view0turn12view3  
- `error.type` добавляется только при ошибке — так рекомендует “Recording errors” для метрик (успехи без `error.type`). citeturn18view0turn12view3  
- атрибуты ограничены небольшим набором, что соответствует идее «не переиспользовать labels» и удерживать cardinality. citeturn5view1  

### Bad: cardinality explosion через user_id, raw path и headers

```go
// BAD: unbounded labels/attributes -> telemetry explosion.
attrs := []attribute.KeyValue{
	attribute.String("user_id", userID),                 // unbounded
	attribute.String("http.route", r.URL.Path),          // raw path, not a template
	attribute.String("host", r.Host),                    // header-derived
	attribute.String("request_id", r.Header.Get("X-Request-ID")), // unbounded
}
hist.Record(ctx, latencySeconds, metric.WithAttributes(attrs...))
```

Почему это «bad»:
- user_id/request_id/IP/UUID‑подобные значения прямо запрещены как labels/размерности метрик из‑за высокой cardinality и взрывного роста time series. citeturn1search9turn5view1  
- `http.route` не может быть заменён на URI path (это прямо запрещено), и должен быть low-cardinality шаблоном. citeturn12view0turn12view3  
- атрибуты из headers могут позволить атакующему спровоцировать cardinality‑лимит и деградацию метрик (semconv предупреждает об этом). citeturn12view3  

### Good: pipeline-level fail-closed redaction + URL/DB sanitization

```yaml
processors:
  redaction:
    allow_all_keys: false
    allowed_keys:
      - http.request.method
      - http.route
      - http.response.status_code
      - error.type
      - service.name
      - service.version
    blocked_key_patterns:
      - ".*token.*"
      - ".*api_key.*"
    url_sanitizer:
      enabled: true
      attributes: ["http.url", "url.full"]
      sanitize_span_name: true
    db_sanitizer:
      sanitize_span_name: true
      sql:
        enabled: true
        attributes: ["db.statement", "db.query"]
```

Почему это «good»:
- redaction processor предназначен для удаления атрибутов, не входящих в allowlist (fail‑closed), маскирования значений по шаблонам и имеет встроенный URL sanitization для снижения cardinality, а также DB sanitization для удаления чувствительных/переменных частей запросов. citeturn8view0turn5view4  
- это соответствует рекомендациям OpenTelemetry по data minimization и «централизованным» механизмам защиты через Collector processors. citeturn5view4turn8view0  

## Anti-patterns и типичные ошибки/hallucinations LLM

**Label/attribute как «контейнер данных»**. LLM часто превращает labels в место хранения «данных запроса»: user_id, request_id, IP, email, полный URL, user-agent. Это приводит к росту time series/streams и деградации backend. В **Prometheus** прямо запрещают high-cardinality labels (user IDs/emails), а **Loki** прямо описывает, что high-cardinality labels создают огромный индекс и множество мелких chunk’ов. citeturn1search9turn5view1turn5view2

**Подмена route template на URL path**. Частая «галлюцинация»: `http.route = r.URL.Path`. Semconv прямо запрещает подставлять URI path вместо route template; `http.route` должен быть low‑cardinality. citeturn12view0turn12view3

**Неконтролируемые histogram buckets**. LLM может:
- выбрать buckets «в миллисекундах» без единиц,  
- сделать 50–200 buckets «для точности»,  
- менять buckets между сервисами.  
Но **Prometheus** подчёркивает, что один histogram создаёт множество time series (bucket‑ряды + `_sum/_count`), и стоимость растёт мультипликативно. citeturn11view0turn5view1  
Шаблон должен фиксировать bucket layout (например, как в semconv для HTTP). citeturn12view3

**Error как строка**. LLM склонна делать `error.type = err.Error()` или записывать «уникальные» stacktrace/message в атрибуты, превращая `error.type` в high-cardinality dimension. “Recording errors” требует `error.type` как классификатор и предупреждает о status description: её следует ставить только если не ожидаются sensitive details; также не рекомендуется дублировать error.type/status code в description и не стоит записывать handled exceptions как ошибки. citeturn18view0

**PII/секреты в URL и атрибутах**. Типичная ошибка — сохранять `url.full`/`url.query` «как есть», включая подписи/токены, или писать credentials в URL. Semconv для URL запрещает credentials в `url.full` и задаёт правила редактирования чувствительных query params; OpenTelemetry security guidance требует data minimization и предлагает processors для удаления/хэширования/редакции. citeturn5view5turn5view4turn8view0

**Tail sampling “в коде приложения” или без учёта routing/лимитов**. tail sampling должен происходить там, где есть весь trace. У tail sampling processor есть жёсткое требование: все spans одного trace должны попасть в один Collector instance; также есть memory/size protections (`num_traces`, `maximum_trace_size_bytes`, bytes_limiting). Игнорирование этого приводит к «дорогой» и малоэффективной схеме. citeturn19view0turn0search3

**“Map all resource attributes to Prometheus labels”**. На практике LLM может предложить автоматически выводить все resource attributes как labels. В материалах CNCF отдельно отмечают, что маппинг всех resource attributes в labels создаёт проблемы cardinality explosion; требуется selective promotion. citeturn13search10turn5view1

**Лог‑шторм из‑за guardrails**. Ещё один класс ошибок — логировать каждое усечение/дроп атрибутов, что создаёт вторичный шторм. В common spec прямо сказано, что SDK может логировать факт truncation/discard, но чтобы избежать excessive logging, такой лог не должен эмититься более одного раза на record. citeturn13search27

## Review checklist для PR/code review

Этот checklist должен применяться к любым изменениям instrumentation, логирования и pipeline конфигов (включая изменения, сгенерированные LLM).

- Проверить, что новые metric attributes/labels **bounded**; нет user_id/request_id/UUID/email/IP/времён/полных URL и прочих unbounded значений. citeturn1search9turn5view1turn5view2  
- Проверить, что `http.route` — route template и не заполняется `url.path`; если framework не поддерживает шаблон — `http.route` не ставится вовсе. citeturn12view0turn12view3  
- Проверить, что для HTTP latency используется фиксированный bucket layout из semconv; изменение bucket boundaries требует отдельного RFC/decision record. citeturn12view3  
- Проверить, что новые гистограммы не создаются «просто так» и имеют минимально разумное число buckets; помнить про “one time series per bucket” + `_sum/_count`. citeturn11view0turn5view1  
- Проверить, что `error.type` — перечислимый классификатор, не `err.Error()`, и применяется согласованно на spans и метриках; successes не включают `error.type`. citeturn18view0  
- Проверить, что span status description не содержит чувствительных данных и не дублирует `error.type`/status code; handled errors не записываются как «ошибки операции». citeturn18view0turn5view4  
- Проверить наличие/неизменность SDK limits для span/logrecord attributes (count/length) и отсутствие кода, генерирующего «log spam» при дропах. citeturn13search3turn13search27  
- Проверить trace sampling: продовый default — ParentBased+TraceIDRatioBased через стандартный конфиг; custom sampler допускается только при явной необходимости и соблюдении требований (preserve tracestate, лёгкий `ShouldSample`). citeturn9view2turn4search8  
- Если включён tail sampling: проверить routing requirement (all spans of a trace to same Collector), защитные лимиты (`maximum_trace_size_bytes`, rate/bytes limiting, `num_traces`) и порядок processors. citeturn19view0  
- Проверить, что логи не содержат PII/секретов и защищены от log injection/некорректного encoding; избегать «logging sensitive info». citeturn16view1turn16view0turn5view4  
- Для Loki‑ориентированного стека: проверить, что labels низкокардинальны, описывают источник, а request‑специфичные поля не стали labels; при необходимости high-cardinality метаданных — использовать structured metadata. citeturn5view2turn10search3turn4search3  
- Проверить, что retention в целевом observability stack задан явно (особенно там, где default «бесконечный»), и что изменения retention сопровождаются оценкой стоимости. citeturn2search1turn10search13turn15view1turn10search0  
- Для multi-tenant: проверить, что изоляция реализуется tenancy header/проектом (например, `X-Scope-OrgID`), а не повсеместным `tenant_id` label. citeturn2search23turn2search7turn14view0turn5view1  
- Проверить наличие pipeline-level redaction/sanitization (allowlist + URL/DB sanitizers) как «страховки» от ошибок кода/LLM. citeturn8view0turn5view4  

## Что из результата нужно оформить отдельными файлами в template repo

Чтобы это стало «нормативным, практическим результатом», в template‑репозитории стоит выделить отдельные документы и артефакты конфигурации, чтобы LLM могла ссылаться на них и не додумывать.

- `docs/observability/cost-control.md` — основной практический гайд: budgets, запрещённые dimensions, примеры “good/bad”, правила изменения buckets/sampling/retention, и объяснение мультипликативной стоимости time series/streams. citeturn5view1turn11view0turn5view2  
- `docs/observability/llm-instructions.md` — MUST/SHOULD/NEVER правила из этого отчёта в виде «контракта для LLM». Основание — требования semconv, sampling и data minimization. citeturn12view0turn9view2turn5view4turn16view1  
- `docs/observability/telemetry-allowlist.md` — явный allowlist attribute keys для traces/logs/metrics (то, что разрешено уходить наружу), плюс denylist patterns (token/api_key/credentials) и правила для URL/query. citeturn8view0turn5view5turn5view4  
- `config/otelcol/otel-collector.yaml` — пример production pipeline: redaction (allowlist + url_sanitizer + db_sanitizer), optional tail sampling policies (errors/latency/rate/bytes limiting), и защитные лимиты. citeturn8view0turn19view0turn0search3  
- `internal/observability/metrics.go` — единые имена метрик и фиксированные bucket boundaries (в частности HTTP duration buckets из semconv), чтобы LLM не создавала новые имена/бакеты. citeturn12view3turn11view0  
- `internal/observability/tracing.go` — инициализация tracer provider и sampling через env (`OTEL_TRACES_SAMPLER`, `OTEL_TRACES_SAMPLER_ARG`), плюс запрет/шаблон для custom sampler (с явным комментарием про `tracestate` и стоимость `ShouldSample`). citeturn9view2turn4search8  
- `internal/logging/logging.go` — structured logging defaults и правила «что не логировать» (PII/секреты), плюс договорённость о том, какие поля могут становиться labels/structured metadata в Loki‑ориентированной схеме. citeturn16view0turn16view1turn5view2turn10search3  
- `docs/observability/retention-and-tenancy.md` — baseline‑параметры retention и multi-tenancy: где default бесконечный, где по умолчанию 14–15 дней, и как считается storage (bytes/day × retention days), плюс модель `X-Scope-OrgID` для multi-tenant backends. citeturn2search1turn10search13turn15view1turn10search0turn2search23turn2search7turn14view0