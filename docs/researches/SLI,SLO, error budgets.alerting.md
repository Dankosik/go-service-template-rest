# Engineering standard для SLI/SLO, error budgets и alerting в template Go-микросервисе

## Scope: когда этот подход применять, а когда нет

Этот стандарт имеет смысл применять, когда сервис **реально будет жить в production** и у него будет **эксплуатационная ответственность**: дежурства (on-call), инциденты, приоритизация reliability work vs feature work, и вам нужна формализуемая «граница нормальности» через SLO и error budget. В подходе SRE ключевая идея — не «максимизировать аптайм», а выбрать **достаточную** надежность и управлять риском через измеримые цели и бюджет ошибок. citeturn5view1turn5view0turn18view0

Подход особенно хорошо подходит для типового микросервиса (API/worker/pipeline), потому что availability часто измеряется **как доля успешных запросов/юнитов работы**, а не как «время аптайма», что практичнее для распределённых систем и сервисов, которые частично работают даже при деградациях. citeturn5view1

Не стоит применять «как есть» (или стоит адаптировать) в следующих случаях:

- **Очень низкий трафик / редкие события**, когда одна случайная ошибка может «съесть» существенную долю error budget и породить бессмысленный page. Google отдельно отмечает, что multiwindow/multi-burn-rate хорошо работает при достаточно высоком входящем потоке, но на low-traffic сервисах требуется менять подход (искусственный трафик, агрегация сервисов для мониторинга, изменение продукта/семантики «единицы ошибки»). citeturn26view0
- **R&D/прототипы**, где target и интерфейсы меняются быстрее, чем вы успеваете стабилизировать SLI и эксплуатационные контуры. Даже в зрелых организациях мониторинг/алертинг — отдельная существенная инженерная работа, и ожидать «идеально и сразу» не стоит. citeturn25view2
- **Библиотеки/SDK**, которые не экспонируют «пользовательский опыт» как сервис (лучше тесты, бенчмарки, контрактные проверки). SLO имеет смысл там, где есть наблюдаемая услуга и последствия деградации.

## Recommended defaults для greenfield template

Ниже — дефолты уровня «boring, battle-tested» для репозитория-шаблона. Они должны трактоваться как **стартовые пресеты**, а не как “one true SLO”: выбранные цифры обязаны быть подтверждены стейкхолдерами и закреплены в error budget policy, иначе SLO превращается в отчёт без рычагов управления. citeturn18view0turn3view1

### Дефолтная модель измерений и окна

**Окно измерения (compliance period):** 28 дней rolling (4 недели). Это соответствует примеру SLO-документа в SRE Workbook и удобно для регулярной оценки. citeturn13view1

**SLI по умолчанию формулируется как ratio:** `good_events / total_events` (иногда — `bad_events / total_events`, но документ должен это явно зафиксировать). citeturn13view0turn21search0turn21search8

**Почему ratio-формат — дефолт:** он напрямую приводит к error budget и к burn rate alerting, и его проще агрегировать и интерпретировать. citeturn18view0turn25view0turn22view0

### Стартовые SLI/SLO пресеты по типам сервисов

#### API service (sync request/response)

**Цели SLI (что измеряем):** ориентируемся на «золотые сигналы» (latency, traffic, errors, saturation), но в SLO по умолчанию закрепляем **availability** и **latency** как наиболее близкие к пользовательскому опыту; traffic и saturation — как эксплуатационные сигналы и превентивные алерты (обычно не paging). citeturn8view2turn17view0

**Availability SLI (default):**
- *Event:* HTTP запрос к публичным/критичным endpoint’ам (исключая `/metrics`, `/healthz`, `/readyz`).
- *Good:* ответ **не 5xx**.
- *Total:* все валидные запросы.
- *Почему так:* пример SLO-документа в Workbook считает 5xx «плохими», а остальные — успешными; это снижает шум от клиентских ошибок и фокусирует SLO на сервисной ответственности. citeturn13view1turn13view0

**Latency SLI (default):**
- *Event:* HTTP запрос.
- *Good:* запрос завершился ≤ T секунд (порог).
- *Рекомендуемый формат:* «X% запросов быстрее порога», а не «средняя/медианная задержка», потому что хвостовые задержки реально определяют UX, и SRE отдельно предупреждает о ловушке средних значений. citeturn8view0turn22view1

**SLO (starter preset, требует калибровки под продукт):**
- Availability: **99.9%** за 28 дней (user-facing); **99.5%** за 28 дней (internal best-effort).  
  Выбор числа «девяток» должен следовать ожиданиям пользователей, цене простоя и позиционированию сервиса; в SRE это прямо выделяется как управляемый бизнесом риск. citeturn5view1turn1search0
- Latency: **95% ≤ 300ms** и **99% ≤ 1s** (для типового API).  
  Пример из Prometheus практик показывает, что SLO вида “95% ≤ 300ms” естественно реализуется через histogram bucket и удобен для алертинга. citeturn22view1

**Как LLM должна «материализовать» это в метриках:** обязательно иметь histogram, где bucket boundary включает SLO-порог (например, `le="0.3"` для 300ms), чтобы SLI вычислялся напрямую как доля попаданий в bucket. citeturn22view1

#### Worker service (background jobs, queue consumer)

**Фундаментальная оговорка:** «availability» для worker — это чаще **успешность unit-of-work**, а «latency» — это либо время обработки, либо (важнее) **end-to-end** время от появления задания до успешного завершения. SRE подчёркивает применимость «request success rate» к non-serving системам как к “units of work”. citeturn5view1

**Default SLI для worker:**
- **Job success rate:** `jobs_success_total / jobs_started_total` (или / jobs_finished_total, если ретраи учитываются отдельно).  
- **Processing latency (in-worker):** “X% job durations ≤ T”.
- **Queueing delay / age (end-to-end latency proxy):** “X% job age at start ≤ T” (если событие несёт timestamp эмиссии/публикации).

**SLO (starter presets):**
- Success rate: **99.9%** успешных job’ов за 28 дней для критичного worker; **99.0–99.5%** для best-effort/пакетных задач (зависит от retry-механики и бизнес-цены потери). citeturn5view1turn18view0
- End-to-end latency: два порога (примерная структура как в SRE Workbook для latency SLO через thresholds):  
  - near-real-time: **90% ≤ 30s**, **99% ≤ 5m**  
  - batch: **90% ≤ 15m**, **99% ≤ 2h**  
  Подбор порогов должен исходить из user expectation и того, что считается «неприемлемо плохо». citeturn25view3turn13view1

#### Async processing / pipeline (materialized views, ETL-like, async API)

Здесь ключевое отличие: пользователь часто ощущает проблему как **stale data**, а не как “ошибка запроса”. Поэтому дефолтный набор должен включать **freshness**, а иногда — coverage/completeness и correctness.

**Freshness SLI (default):**
- *Event:* “data read” в точке, где пользователь/клиент реально потребляет результат (read path).
- *Good:* freshness(age) ≤ threshold.
- *Референс:* пример SLO-документа в SRE Workbook задаёт freshness как долю чтений, использующих данные «не старше N минут», с двумя порогами; и рекомендует вариант, где клиенты проверяют watermark и инкрементируют метрики — как более близкий к пользовательскому опыту. citeturn13view1turn13view0

**Coverage/Completeness SLI (default, если применимо):**
- *Event:* pipeline run (или time-slice).
- *Good:* обработано 100% ожидаемых данных в этом run’е.
- *Референс:* example SLO doc задаёт completeness как долю часов/запусков, где покрыто 100% данных. citeturn13view1

**Correctness SLI (optional default):**
- Для большинства микросервисов correctness обеспечивается тестированием, но если есть независимая валидация (synthetic golden data), можно завести correctness-prober, как в примере SLO doc. citeturn13view1turn13view0

**SLO (starter preset):**
- Freshness: **90% ≤ 1m**, **99% ≤ 10m** (калибровать под продукт). Эта структура буквально встречается в example SLO doc. citeturn13view1
- Completeness: **99%** запусков без пропусков (если пропуски реально user-impacting). citeturn13view1

### Error budgets (policy defaults) и связь с релизами

**Базовые принципы:**
- SLO не должен быть 100%: требовать 100% ведёт к чрезмерно дорогим/консервативным решениям и снижению скорости изменений; вместо этого вводится error budget и его регулярный трекинг. citeturn5view0turn5view1
- Error budget = `1 - SLO target` (в доле), и может выражаться в количестве ошибок на объём событий. citeturn3view1turn13view1

**Дефолтная “error budget policy” для template repo (минимально жизнеспособная):**
- Если сервис **в пределах SLO** — релизы по обычному процессу.
- Если сервис **превысил error budget за последние 4 недели** — freeze всех изменений, кроме P0/security fixes, до возврата в бюджет. citeturn3view1
- Если единичный инцидент “съел” **>20% бюджета за 4 недели** — обязателен постмортем и минимум один action item самого высокого приоритета на устранение root cause. citeturn3view1
- Политика должна быть письменно согласована ключевыми стейкхолдерами (PM/dev/SRE/ops). Сам факт согласования — тест “fit for purpose” SLO. citeturn18view0

### Alerting defaults: burn rate, paging vs ticket, anti-noise

**Почему алертить по SLO/burn rate — дефолт:** SRE workbook позиционирует SLO-based alerts как наиболее качественный сигнал того, что on-call должен реагировать, потому что SLO измеряет reliability как её видит пользователь. citeturn20search0turn17view0

**Дефолтная схема (multiwindow, multi-burn-rate):**
- Для SLO 99.9% workbook рекомендует как стартовые параметры:  
  - **Page:** long=1h, short=5m, burn=14.4 (≈2% бюджета)  
  - **Page:** long=6h, short=30m, burn=6 (≈5% бюджета)  
  - **Ticket:** long=3d, short=6h, burn=1 (≈10% бюджета) citeturn26view0turn25view0
- Идея multiwindow — снижать ложные срабатывания и уменьшать “reset time”, проверяя, что burn продолжается и в коротком окне. citeturn4view1turn26view0
- Workbook прямо формулирует эвристику “page vs ticket”: если проблема может выжечь бюджет за часы/пару дней — нужна активная нотификация; если у вас есть запас времени — тикет на следующий рабочий день. citeturn25view1

**Low-traffic адаптация (policy):**
- Если трафик низкий, один флап может давать огромный burn rate (пример: 10 req/hour и 1 ошибка → 1000× burn rate для SLO 99.9%). citeturn26view0  
  В дефолтный стандарт для template repo нужно включить правило: **SLO-alerts MUST иметь guardrail по минимальному объёму событий** (например, “не алертить по burn rate, если total events < N за окно”), либо использовать подходы из Workbook: искусственный трафик, укрупнение сервиса для мониторинга, или изменение продукта/семантики failure. citeturn26view0

**Превентивные алерты по saturation (не paging по умолчанию):**
- Хороший алертинг должен быть symptom-based и actionable; SRE допускает некоторые preventive alerts по внутренним метрикам для предотвращения “резкого падения в 100% failure” при достижении жёстких квот. Но общий принцип — избегать алертов на internal behavior, потому что они плохо отражают user impact и хрупки при изменении реализации. citeturn17view0turn8view2

### Routing, runbooks, dashboard hierarchy (default)

**Routing/группировка/дедупликация:** использовать **entity["organization","Alertmanager","prometheus alert router"]** как дефолтный агрегатор — он умеет grouping, deduplication, routing, silencing, inhibition. citeturn16view0  
Дефолтный repo convention:
- Все алерты должны иметь label’ы `service`, `severity`, `team` (или `owner`) и `runbook`.
- Grouping на уровне route должен группировать по bounded label’ам вроде `alertname`/`service`/`cluster`, чтобы при массовом фейле приходил один page вместо сотен. citeturn16view0turn16view1
- Использовать inhibition, чтобы, например, “cluster down” глушил downstream алерты этого кластера. citeturn16view0turn16view1

**Runbooks:** в on-call главе прямо сказано: новые алерты должны тщательно ревьюиться, и **каждый алерт должен иметь playbook entry**; также рекомендуется “прогонять” алерты в test mode, чтобы отловить false positives, прежде чем повышать до paging. citeturn8view3

**Dashboard hierarchy:** SRE Workbook рекомендует, чтобы SLI-метрики были первыми, которые видит инженер при SLO-alert, и располагались заметно (landing page). Для расследования причин нужны также метрики “intended changes” (version/flags/config) и метрики зависимостей. citeturn9view0  
В SRE book дополнительно подчёркивается: paging должен ловить симптомы; а детали и субкритические проблемы — жить на дашбордах, а не в “email-alert noise”. citeturn8view2turn8view0

## Decision matrix / trade-offs

Ниже — «матрица решений» в виде коротких, но нормативных trade-offs, которые LLM не должна “догадывать”, а должна выбирать по зафиксированным правилам репозитория.

**Uptime vs request-based availability**
- **Default:** request/unit-of-work success rate, потому что в распределённых системах time-based availability часто не отражает реальную доступность. citeturn5view1turn3view1
- **Trade-off:** для систем с очень малым числом событий “per-request” метрика может быть шумной; тогда нужно либо синтетическая нагрузка, либо иная семантика “события”. citeturn26view0

**Что считать “ошибкой” для availability**
- **Default:** считать “bad” только **5xx/timeout** на серверной стороне (пример Workbook). citeturn13view1turn13view0
- **Trade-off:** если 4xx отражают ошибку сервиса (например, неверная валидация/контракт), можно завести отдельный SLI “correctness/quality” или отдельный SLO на конкретный user journey. Общий принцип — метрика должна соответствовать пользовательскому опыту, а не внутренней интерпретации. citeturn17view0turn13view0

**Latency: average vs percentiles / thresholds**
- **NEVER:** опираться на среднюю задержку как главный индикатор — tail может быть катастрофическим при «хорошем среднем». citeturn8view0
- **Default:** thresholds (например, 95% ≤ 300ms) и histogram-based вычисление, потому что это прямо поддерживается Prometheus практиками и удобно алертится. citeturn22view1

**Histograms vs summaries**
- **Default:** histograms для latency/freshness, потому что они агрегируются между инстансами; Prometheus явно показывает, что “avg(quantile)” — статистически бессмысленно, а histogram_quantile — корректный путь. citeturn22view1
- **Trade-off:** histograms требуют выбора buckets; но практический совет — ставить buckets вокруг SLO-порогов, чтобы корректно отслеживать “внутри/снаружи” цели. citeturn22view1turn8view0

**Burn-rate alerting vs simple error-rate threshold**
- **Default:** burn rate, потому что он привязан к budget consumption и позволяет различать быстрый и медленный выжиг бюджета. citeturn25view0turn26view0
- **Trade-off:** больше параметров и больше recording rules; это управляется conventions и тестированием правил. citeturn26view0turn22view0turn20search5

**Paging vs ticket**
- **Default:** multi-burn-rate с разными severity (page/ticket), как в SRE Workbook: 2%/1h и 5%/6h — page; 10%/3d — ticket. citeturn25view0turn25view1turn26view0
- **Trade-off:** для “очень busy” сервисов часть 6h может уходить в ticket, если page load иначе будет чрезмерным. Workbook прямо оговаривает зависимость от baseline page load и источников шума (выходные/праздники). citeturn25view0

## Набор правил в формате MUST / SHOULD / NEVER для LLM

Это — прямой материал для `docs/llm/observability_slo_alerting.md` (или аналогичного файла), чтобы модель генерировала измеримое и совместимое с repo. Каждое правило подразумевает, что соответствующая конфигурация/схема уже лежит в репозитории.

### MUST

- MUST формулировать любой SLI как **ratio good/total** (или bad/total), и в тексте/коде явно фиксировать, что входит в numerator/denominator. citeturn13view0turn21search0
- MUST реализовывать latency/freshness SLIs через **histogram buckets**, которые включают SLO thresholds, чтобы можно было выражать “X% ≤ T” через bucket/count. citeturn22view1
- MUST измерять и алертить преимущественно **симптомы**, а не “внутреннее стало странно”; внутренние preventive alerts допустимы только там, где иначе возможен мгновенный переход к 100% failure (жёсткая квота/лимит). citeturn17view0turn8view2
- MUST использовать multiwindow/multi-burn-rate конфигурацию для SLO-alerts как дефолт и начинать с параметров из SRE Workbook (14.4/6/1 и окна 1h/5m, 6h/30m, 3d/6h). citeturn26view0turn25view0
- MUST добавлять **guardrails для low-traffic** (минимальное число событий в окне или адаптированный подход), иначе burn-rate алерты будут бессмысленно шуметь. citeturn26view0
- MUST соблюдать правила именования метрик и единиц (base units, `_total` для counters, `_seconds`/`_bytes` для единиц), иначе downstream YAML/PromQL становится нечитаемым и хрупким. citeturn23view0
- MUST избегать unbounded/high-cardinality label values (user_id, request_id, raw URL path и т.п.). citeturn23view0
- MUST при построении recording rules агрегировать numerator и denominator **раздельно**, а потом делить (не “среднее от средних”). citeturn22view0turn22view1
- MUST обеспечивать, что каждый paging alert имеет **runbook/playbook** (ссылка должна быть в аннотациях/labels алерта) и проходит ревью/обкатку до повышения до paging. citeturn8view3turn17view0
- MUST держать SLI-метрики на landing dashboard и иметь drill-down до golden signals и “intended changes” (version/flags/config). citeturn9view0turn8view2
- MUST иметь written error budget policy, иначе бюджет ошибок не будет “рычагом” (в т.ч. релизный freeze/приоритизация). citeturn18view0turn3view1

### SHOULD

- SHOULD различать latency успешных и неуспешных запросов, чтобы не получать “ложно хорошие” latency из-за быстрых ошибок. citeturn8view2
- SHOULD использовать grouping/dedup/inhibition в Alertmanager, чтобы массовый инцидент не превращался в сотни страниц. citeturn16view0turn16view1
- SHOULD добавлять метрики “build/version/config” (например, build_info) и отображать их на дашбордах, чтобы проще коррелировать инциденты с релизами и конфиг-изменениями. citeturn9view0turn23view0
- SHOULD документировать rationale выбора чисел (даже если они ad hoc) и явно помечать, evidence-based они или нет. citeturn18view0turn13view1
- SHOULD выбирать histograms вместо client-side summaries, если нужна агрегация по инстансам/репликам. citeturn22view1

### NEVER

- NEVER задавать SLO = 100% “потому что так правильно”; это противоречит SRE мотивации error budget и ведёт к потере velocity/чрезмерной стоимости. citeturn5view0turn5view1
- NEVER строить paging alerts на метрике, которую нельзя **действительно** и быстро отработать (не actionable) или которая не требует срочности. citeturn8view1turn17view0
- NEVER алертить на “среднюю задержку” как главный сигнал. citeturn8view0
- NEVER делать `avg(<summary_quantile>)` по репликам для percentiles — Prometheus прямо помечает это как BAD. citeturn22view1
- NEVER добавлять label’ы с unbounded cardinality (raw path, user_id, email, uuid и т.п.). citeturn23view0
- NEVER выпускать новые paging alerts без периода тестового прогона/обкатки и командного ревью (как код). citeturn8view3

## Concrete good / bad examples

### Пример SLI/SLO через Prometheus histogram: GOOD

Цель: latency SLI “95% запросов ≤ 300ms” реализуется как `bucket(le="0.3") / count`.

```promql
sum(rate(http_server_request_duration_seconds_bucket{le="0.3"}[5m])) by (service)
/
sum(rate(http_server_request_duration_seconds_count[5m])) by (service)
```

Это ровно тот паттерн, который Prometheus рекомендует для SLO вида “95% within 300ms”. citeturn22view1

### Пример percentiles: GOOD vs BAD

BAD (неагрегируемо/статистически бессмысленно при нескольких инстансах):

```promql
avg(http_request_duration_seconds{quantile="0.95"})
```

GOOD (агрегация histograms и вычисление квантиля сервер-сайд):

```promql
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))
```

Prometheus документация явно показывает этот контраст. citeturn22view1

### Пример recording rule для ratio: GOOD

Правило: numerator и denominator агрегируются раздельно, потом деление. Это прямо закреплено в best practices по recording rules. citeturn22view0

```yaml
- record: service:http_requests:rate5m
  expr: sum without (instance) (rate(http_requests_total[5m]))

- record: service:http_errors:rate5m
  expr: sum without (instance) (rate(http_requests_total{code=~"5.."}[5m]))

- record: service:http_errors_per_requests:ratio_rate5m
  expr: |
      service:http_errors:rate5m
    /
      service:http_requests:rate5m
```

Конвенция `level:metric:operations` и правило “aggregating up numerator/denominator separately” — из официальной документации Prometheus. citeturn22view0

### Go instrumentation: GOOD vs BAD (cardinality)

GOOD: bounded labels (`method`, `route`, `code`) и один histogram на latency. Важно: `route` — это **шаблон маршрута** (например, `/v1/users/{id}`), а не raw path.

```go
type Metrics struct {
    reqTotal   *prometheus.CounterVec
    reqLatency *prometheus.HistogramVec
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
    m := &Metrics{
        reqTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "http_server_requests_total",
                Help: "Total number of HTTP server requests.",
            },
            []string{"method", "route", "code"},
        ),
        reqLatency: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "http_server_request_duration_seconds",
                Help:    "HTTP server request latency in seconds.",
                Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.45, 0.6, 1.0, 2.5, 5.0},
            },
            []string{"method", "route", "code"},
        ),
    }
    reg.MustRegister(m.reqTotal, m.reqLatency)
    return m
}
```

Ключевые моменты здесь поддерживаются практиками Prometheus: именование с единицами `_seconds`, `_total`, использование base units и требование избегать high-cardinality labels. citeturn23view0

BAD: raw path + request_id → взрыв кардинальности и рост TSDB.

```go
reqTotal := prometheus.NewCounterVec(
    prometheus.CounterOpts{Name: "http_requests_total"},
    []string{"path", "request_id"},
)
```

Prometheus прямо предупреждает, что high-cardinality label values (например, user IDs и вообще unbounded множества значений) нельзя класть в labels. citeturn23view0

## Anti-patterns и типичные ошибки/hallucinations LLM

**“Поставим 100% SLO, ведь прод”**  
Ошибка: SRE подчёркивает, что 100% и нереалистично и вредно; error budget нужен именно для баланса скорости изменений и надежности. citeturn5view0turn5view1

**“Давайте алерт на CPU > 80% как paging”**  
Ошибка: symptom-based правило нарушено; CPU — причина/потенциальная причина, не обязательно user-impact. Допустимы preventive alerts только там, где близость к жёстким лимитам может привести к мгновенному массовому фейлу. citeturn17view0turn8view2

**“Считаем latency по среднему”**  
Ошибка: хвостовые задержки определяют деградацию, а среднее скрывает реальные проблемы; SRE рекомендует histogram-based подход и явную работу с tail. citeturn8view0turn22view1

**“Мы на low-traffic, но всё равно включим burn-rate paging без guardrails”**  
Ошибка: Workbook показывает, что один флап может породить 1000× burn rate и «съесть» значимую долю бюджета. Для low-traffic нужны специальные меры. citeturn26view0

**“Сделаем summary quantile на каждом инстансе и усредним”**  
Ошибка: Prometheus прямо помечает `avg(…quantile…)` как BAD и объясняет, почему summaries плохо агрегируются. citeturn22view1

**“Добавим label user_id/request_id/полный URL”**  
Ошибка: нарушение cardinality best practices; это ломает масштабируемость метрик и usability мониторинга. citeturn23view0

**“У нас есть SLO, но нет error budget policy”**  
Ошибка: SRE workbook делает commit к использованию error budget формальным и требует письменной политики; иначе SLO не управляет решениями. citeturn18view0turn3view1

## Review checklist для PR/code review

Чеклист предназначен для PR, где меняются endpoint’ы, очереди, метрики, SLO или алерты.

- Проверено, что новый/изменённый SLI остаётся **good/total ratio** (или явно документирован bad/total) и определены numerator/denominator. citeturn13view0
- Если добавлена/изменена latency или freshness метрика, используется **histogram**, а bucket layout покрывает SLO thresholds. citeturn22view1turn8view0
- Любые новые labels bounded; нет user_id/request_id/raw path/unbounded значений. citeturn23view0
- Для SLO-alerts применён multiwindow/multi-burn-rate паттерн; severity соответствует page/ticket, параметры стартуют с рекомендованных (14.4/6/1 и окна). citeturn26view0turn25view1
- Для low-traffic предусмотрены guardrails (min events) или иной метод из рекомендованных (synthetic traffic/aggregation/изменение semantics). citeturn26view0
- Каждый paging alert — urgent/actionable, избегает “непонятной странности”; соответствует философии “every page should be actionable”, “require intelligence”, “symptoms > causes”. citeturn8view1turn8view2turn17view0
- У каждого alert есть ссылка на runbook/playbook, и изменения алертов прошли ревью и/или тестовый прогон перед paging. citeturn8view3
- Recording rules оформлены по `level:metric:operations`; ratios агрегируются корректно (сначала numerator/denominator, потом деление). citeturn22view0
- Landing dashboard содержит SLI-метрики и error budget (или burn-rate) как “первую страницу”; есть drill-down в golden signals и видимость intended changes (version/flags/config) и зависимостей. citeturn9view0turn8view2
- Обновлены документы: SLO doc (rationale, approvers, даты ревью) и/или error budget policy при изменении целей/семантики. citeturn18view0turn13view1

## Что из результата нужно оформить отдельными файлами в template repo

Этот раздел — практически готовый “docs/ + repo conventions” план. Он минимизирует необходимость для LLM «догадываться» и фиксирует контракт.

- `docs/observability/sli_slo_policy.md`  
  Описывает дефолтные SLIs/SLOs для API/worker/pipeline, окна измерения (28d), определение “valid events” и правила исключений (health endpoints, тестовый трафик и т.п.). Основание: SLI как good/total; SLO-документ должен фиксировать детали и rationale. citeturn13view0turn18view0turn13view1

- `docs/observability/error_budget_policy.md`  
  Шаблон политики с условиями freeze/исключениями и постмортем-триггерами (включая >20% бюджета за 4 недели). citeturn3view1turn18view0

- `docs/observability/alerting_policy.md`  
  Норматив: burn rate alerting, разделение page vs ticket, параметры по умолчанию (multiwindow/multi-burn-rate), и отдельная секция “low-traffic guardrails”. citeturn26view0turn25view1

- `docs/observability/metrics_contract.md`  
  Контракт метрик (имена, единицы, labels, cardinality budget), с прямым запретом на unbounded labels и требованием base units. citeturn23view0turn22view1

- `docs/runbooks/README.md` + `docs/runbooks/<service>/...`  
  Минимальный runbook template: “Impact/Severity”, “Immediate mitigation”, “Diagnostics”, “Rollback”, “Escalation”, “Links”. Принцип: каждый алерт имеет playbook entry и проходит ревью. citeturn8view3turn17view0

- `deploy/monitoring/prometheus/recording_rules.yml`  
  Recording rules для вычисления error_ratio по окнам (5m/30m/1h/6h/3d) и burn-rate вспомогательные метрики, оформленные по `level:metric:operations`. citeturn22view0turn26view0

- `deploy/monitoring/prometheus/alerting_rules.yml`  
  Готовые multiwindow/multi-burn-rate правила для page/ticket с шаблонными коэффициентами и местами для подстановки SLO target. citeturn26view0turn25view0

- `deploy/monitoring/alertmanager/alertmanager.yml`  
  Routing tree: group_by, receivers, inhibition для suppress “downstream noise” при крупных авариях, плюс conventions по labels. citeturn16view0turn16view1

- `deploy/monitoring/grafana/dashboards/<service>_overview.json` (или эквивалент)  
  Иерархия: landing (SLI + error budget), затем golden signals, затем dependencies и intended changes (версия/флаги/конфиг). citeturn9view0turn8view2

- `docs/llm/observability_slo_alerting.md`  
  Содержит MUST/SHOULD/NEVER из этого документа, чтобы LLM генерировала совместимые метрики/правила и не ломала контракт (особенно cardinality, histograms, multiwindow burn-rate, runbooks). citeturn23view0turn22view1turn26view0turn8view3