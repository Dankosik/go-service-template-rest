# Load testing, capacity planning и scaling strategy для production-ready Go-микросервиса

## Scope

Подход из этого документа применим, когда вы делаете stateless (или почти stateless) микросервис с сетевым I/O (HTTP/gRPC), который должен выдерживать тысячи RPS/QPS и при этом сохранять предсказуемую tail latency, а развёртывание предполагается в оркестраторе контейнеров с autoscaling (типично — HPA/VPA + node autoscaling). В таких системах capacity planning опирается на прогноз спроса и **регулярные нагрузочные тесты**, которые связывают «сырьё» (серверы/ядра/память) с реальной «вместимостью сервиса» (запросы/секунда при заданной скорости ответа). citeturn14view0

Подход уместен именно как *engineering standard* для template‑репозитория: он фиксирует «boring defaults», требуемые артефакты и то, какие решения должна уметь обосновывать LLM на основании результатов тестов и профилирования. Это соответствует SRE‑практике: производительность и capacity planning считаются ключевыми обязанностями эксплуатации, а «надежда — не стратегия» (т.е. нельзя “угадать” ёмкость без измерений). citeturn14view0

Подход **не** подходит (или требует серьёзной адаптации), если:
- сервис — тяжёлый batch/ETL, а не latency‑sensitive API (тогда важнее throughput/время джоба, другой профиль нагрузки);
- нагрузка определяется очередями/стримингом и backpressure важнее RPS на HTTP (тогда тесты строятся вокруг глубины очередей/скорости обработки и scaling сигналов для event‑driven);
- в тестовой среде невозможно воспроизвести производственные зависимости/сеть/лимиты ресурсов (результаты будут иметь слабую переносимость);
- есть сильная statefulness (например, едущая рядом база в одном процессе) — тогда capacity planning должен включать модель хранения и деградации состояния, а не только API‑слой.

## Recommended defaults для greenfield template

### Нормативная «performance contract» для каждого сервиса

В template должны существовать явно оформленные performance SLI/SLO и бюджеты, иначе любые “тысячи RPS” остаются лозунгом. В SRE‑терминах: SLI — измеряемая метрика качества сервиса (латентность, error rate, throughput), SLO — целевое значение/диапазон этой метрики. Важно, что latency/error/throughput — типичные SLI для сервисов. citeturn6view4

Критично фиксировать **tail latency**, а не только средние значения: среднее может скрывать «длинный хвост» выбросов, при котором часть пользователей получает очень плохое время ответа; SRE‑best practice — задавать цели по 95‑ и/или 99‑перцентилю и снижать именно tail, а не «среднюю температуру». citeturn11view0

Для template‑сервиса предлагается «по умолчанию» (как пример‑заглушка, которую команда обязана заменить):
- Latency SLO: `p95 < 200ms`, `p99 < 400ms` для ключевых API (или root span для end‑to‑end).
- Error SLO: `rate(5xx + timeouts) < 1%` на окне теста.
- Capacity goal: N RPS при соблюдении SLO, в конфигурации N+headroom (см. ниже).
Эти числа не являются универсальными, но удобны как дефолт, потому что k6 напрямую поддерживает threshold‑критерии вида `p(95)<200` и `rate<0.01`. citeturn6view3

Также в template фиксируется минимальный набор наблюдаемости «четыре золотых сигнала»: latency, traffic, errors, saturation. citeturn2search0  
Эти же сигналы используются как «матрица интерпретации» результатов performance‑тестов.

### Выбор сценариев и модель нагрузки

**Сценарии должны отражать реальную структуру трафика**: k6‑гайд по automated performance testing рекомендует определять типичный профиль трафика (например, из аналитики/мониторинга), затем строить тесты под выбранные сценарии. citeturn15view2  
Template должен требовать, чтобы каждый performance‑suite начинался с маленького набора:
- «1–3 ключевых endpoint/операции» (read/write mix).
- «Сценарий авторизации/кеша/DB» — если это доминирует в реальном пути запроса.
- «Деградационные случаи» (ограничение времени, отказ зависимостей) — если сервис обязан graceful degradation/load shedding (см. ниже). citeturn16view0

**Модель нагрузки по умолчанию для проверки “тысяч RPS” — open model (arrival‑rate)**. Причина: в closed model скорость генерации запросов связана с временем ответа, и при росте латентности реальная подаваемая нагрузка падает; это не подходит, когда цель — гарантировать фиксированный arrival rate/throughput (и в литературе описывается как coordinated omission). citeturn22view3  
Поэтому default:
- Для capacity/throughput подтверждения: `constant-arrival-rate` или `ramping-arrival-rate`. citeturn6view2turn22view3  
- Для user‑journey «как у пользователя»: допускается closed model (`ramping-vus`), но интерпретация должна учитывать риск coordinated omission. citeturn22view3

k6 поддерживает многосценарность: несколько независимых сценариев в одном скрипте с отдельными executor‑профилями и тегами для анализа. citeturn16view4

### Набор обязательных тестов и частота запуска

k6 выделяет типовые виды тестов: smoke, average‑load, stress, spike, soak. В частности, рекомендуется **всегда** иметь average‑load тест для baseline‑сравнений и smoke тест для проверки скриптов до тяжёлых прогонов. citeturn15view2

Template должен содержать минимум пять тестов (как код + описание целей):
- **Smoke**: “скрипт корректен, сервис отвечает, метрики пишутся” (короткий прогон). citeturn15view2  
- **Baseline / Average‑load**: репрезентативная “норма” для сравнения тренда и регрессий. citeturn15view2  
- **Load (target)**: целевой RPS (например 1000/3000 RPS) при SLO‑thresholds.
- **Spike**: резкий скачок (проверка поведения autoscaling/кешей/лимитов).
- **Soak**: длительная нагрузка для поиска утечек, деградации, роста latency во времени. citeturn15view2

Внутренний стандарт также должен требовать, чтобы baseline и target‑load тесты **были автоматизируемыми** с pass/fail критериями (thresholds). k6 явно позиционирует thresholds как механизм pass/fail, часто используемый для кодирования SLO, и подчёркивает их важность для автоматизации. citeturn6view3

### Success criteria, budgets и «готовность к тысячам RPS»

Готовность к N тысячам RPS в template должна проверяться не “одним числом”, а набором критериев:
- **SLO‑соответствие по перцентилям** (p95/p99) и error budget на окне теста. citeturn11view0turn6view3  
- **Отсутствие признаков насыщения** по golden signals: рост latency должен быть объясним saturation‑метриками (CPU throttling, рост in‑flight/очередей, memory pressure), а не «магией». citeturn2search0  
- **Проверка поведения при перегрузе**: service должен “вести себя разумно” под нагрузкой (graceful деградация, отсечение дорогих фич, load shedding), а не уходить в каскадный отказ. citeturn16view0  
- **Повторяемость**: результаты должны быть сравнимы с baseline (тренд, регрессии). citeturn15view2

Для вычислений и sanity‑check’ов capacity планирования template может использовать **закон Литтла** (как инженерную проверку согласованности throughput/latency/in‑flight): \(L = \lambda W\) (среднее число заявок в системе равно arrival rate, умноженному на среднее время в системе). Это полезно для оценки требуемой concurrency (соединения, goroutine‑пулы, лимиты in‑flight). citeturn20view0

### Артефакты после performance testing

Template должен требовать, чтобы каждый значимый прогон создавал минимальный набор файлов‑артефактов:

**Артефакты нагрузочного инструмента (k6 как default)**:
- end‑of‑test summary с p90/p95/p99 и результатом thresholds. citeturn15view1turn6view3  
- machine‑readable summary JSON через `handleSummary()` (например `summary.json`). k6 прямо описывает это как стандартный способ получить структуру агрегированных метрик и сохранить в файл. citeturn15view0  
- time‑series результаты либо в файл (JSON/CSV), либо стрим в внешнее хранилище — k6 описывает `--out` и необходимость granular time‑series для глубокого анализа. citeturn15view1

**Артефакты диагностики сервиса (Go tooling)**:
- CPU profile / heap profile / goroutine profile, собранные через `/debug/pprof/` endpoints (или эквивалентно через встроенные профили). citeturn9view0turn9view3  
- execution trace для анализа scheduler/GC/syscall событий (трейс можно анализировать `go tool trace`; execution trace фиксирует широкую палитру рантайм‑событий). citeturn9view2  
- (опционально) mutex/block профили при подозрении на contention: Go diagnostics указывает, что block и mutex профили не включены по умолчанию и включаются отдельно. citeturn17search27turn17search1

Важно: инструменты диагностики могут влиять друг на друга; официальная Go‑документация предупреждает, что некоторые tools интерферируют (например, точное memory profiling и block profiling могут искажать другие измерения), поэтому template должен требовать «снимать профили в изоляции», если нужна точность. citeturn22view4

### Default observability для анализа bottlenecks

Для анализа latency в распределении часто применяются Prometheus histograms и расчёт квантилей. Однако Prometheus подчёркивает, что histogram‑квантиль — оценка, зависящая от bucket boundaries; “вычисленный” p95 может выглядеть существенно хуже реального при неудачных границах, хотя гистограмма при этом всё равно способна корректно показать “внутри/вне SLO” при подходящих bucket’ах. citeturn22view0  
Отсюда стандарт template:
- latency SLO должны быть выражены через понятные bucket boundaries (например вокруг целевых 200/400ms), иначе перцентили могут “врать” в сторону паники или ложного спокойствия. citeturn22view0

### Scaling strategy: requests/limits, autoscaling signals, ceilings

**requests/limits как обязательная часть производительности**. Kubernetes явно описывает:
- scheduler размещает Pod так, чтобы сумма **requests** не превышала capacity узла; это защищает от будущего роста нагрузки даже если текущая утилизация мала. citeturn8view0  
- CPU limit — «жёсткий потолок», enforced через throttling; memory limit — enforced через OOM kill. citeturn8view2  
- CPU request в Linux обычно работает как “вес” в условиях contention, а CPU limit — как жёстный cgroup‑предел. citeturn8view0

Для Go‑сервисов под нагрузкой важно понимать взаимодействие CPU limits и tail latency: Go‑команда в блоге про container‑aware GOMAXPROCS отмечает, что kernel throttling при CPU limits — «грубый механизм» с потенциалом существенного влияния на tail latency (типичный период throttling ~100ms), а spikes рантайма (например GC) тоже могут приводить к throttling. citeturn9view4  
С Go 1.25 дефолт `GOMAXPROCS` стал учитывать CPU limit контейнера и динамически подстраиваться при изменении лимита. citeturn9view4  
Следствие для template: нельзя обсуждать pod sizing и capacity без явной политики CPU requests/limits и версии Go runtime.

**HPA signals**. Kubernetes HPA:
- масштабирует число Pod‑реплик по наблюдаемым метрикам (CPU/memory/custom/external). citeturn23view0  
- по CPU метрике использует CPU utilization как процент от resource request; если request не задан, метрика для Pod будет undefined и autoscaler не сможет действовать по этой метрике. citeturn23view0  
- может масштабировать по нескольким метрикам, выбирая максимальный рекомендованный размер среди них. Это позволяет комбинировать «user‑impact metric» (например RPS/queue depth) с «safety metric» (CPU). citeturn23view1  
- поддерживает настройку поведения scaling (`behavior`): rate policies, stabilization window против flapping, tolerance. citeturn23view2

**Источники метрик для HPA**:
- HPA часто читает `metrics.k8s.io`, `custom.metrics.k8s.io`, `external.metrics.k8s.io`, и `metrics.k8s.io` обычно предоставляется Metrics Server (который нужно установить отдельно). citeturn23view0  
- Metrics Server прямо позиционируется как источник метрик **для autoscaling**, и предупреждает не использовать его как замену полноценного мониторинга. citeturn22view1  
- Kubernetes docs по resource metrics pipeline подчёркивают, что роль Metrics API — «кормить autoscaler компоненты», и что metrics‑server (или эквивалент) нужен для доступа к metrics.k8s.io. citeturn22view2

**Node autoscaling** (когда pods больше, чем помещается на узлах) — отдельный слой, и Kubernetes описывает, что node autoscalers provision nodes для unschedulable Pods и могут сочетаться с workload autoscaling (HPA). citeturn16view3

**VPA** в Kubernetes корректирует requests/limits на основании исторического использования и событий типа OOM; это отдельный компонент, который устанавливается как add‑on и работает через CRD. citeturn16view2  
Для стандартного greenfield template VPA чаще используется как recommendation engine (пока команда не уверена в автоматических eviction) — этот момент спорный и должен фиксироваться как trade‑off (см. ниже). Факт механики “adjusting requests/limits to match actual usage” — из официального описания. citeturn16view2

**Capacity headroom и отказоустойчивость**. entity["company","Google","tech company"] в SRE best practices рекомендует capacity planning с запасом на одновременный planned + unplanned outage (подход “N + 2”), а ресурс‑к‑вместимости ratio устанавливать **load testing‑ом, а не традицией**. citeturn16view0  
Также подчёркивается, что сервисы должны уметь деградировать под overload и практиковать load shedding; в SRE примерах даже тестируют кластера «за пределами rated capacity», чтобы убедиться в приемлемом поведении. citeturn16view0

**Практическая связка для template (“как выйти на тысячи RPS”)**:
1) Baseline test → измерить latency/ошибки/сaturation на “норме”. citeturn15view2turn2search0  
2) Ramping arrival rate → найти “knee” кривой (рост p99 и ошибок при росте saturation). Open model по умолчанию, чтобы не скрыть деградацию. citeturn22view3turn6view2  
3) Plateau на target RPS (например 1000, 3000, 10000) → собрать pprof/trace + метрики. citeturn9view0turn9view2turn15view1  
4) Spike/Soak → проверить autoscaling и деградацию во времени. citeturn15view2turn23view2turn16view0  
5) Задокументировать ceilings: maxReplicas, max nodes, лимиты зависимостей (DB connections, upstream quotas). Для HPA maxReplicas — часть спецификации, а multi‑metric scaling берёт максимум рекомендаций, что важно учитывать как “ceiling pressure”. citeturn23view1

### Где запускать load tests: локально vs distributed

Для тысяч RPS часто требуется распределённая генерация нагрузки. entity["company","Grafana Labs","observability company"] предоставляет k6 Operator для запуска distributed k6 тестов прямо в кластере; официальная документация описывает установку оператора и необходимость использовать tagged releases для стабильности. citeturn26search0turn26search1  
Документация по distributed tests прямо говорит, что первым шагом является установка оператора. citeturn26search2

**Стандарт template**: локальный запуск допустим для smoke/baseline, но для «тысячи+ RPS» должен существовать documented путь распределённого прогона (например в отдельном performance‑кластере) с воспроизводимым описанием окружения.

image_group{"layout":"carousel","aspect_ratio":"16:9","query":["k6 load testing results output console p95 p99","k6 operator kubernetes distributed load testing architecture","Go pprof flame graph example","Horizontal Pod Autoscaler behavior stabilizationWindowSeconds diagram"],"num_per_query":1}

## Decision matrix / trade-offs

### Инструмент для load testing

**k6 (default в template)**  
Плюсы: зрелая модель сценариев/экзекьюторов, чёткая документация про open vs closed модель и coordinated omission, встроенные thresholds (pass/fail) и механика export результатов (summary, time‑series, кастомный `handleSummary`). citeturn22view3turn6view3turn15view1turn15view0  
Минусы: scripting на JS; при очень высокой нагрузке часто нужен distributed execution (решается operator’ом, но это дополнительная инфраструктура). citeturn26search1turn26search2  
Когда выбирать: greenfield template, где важны стандарты, воспроизводимость, thresholds и матрица сценариев “smoke/baseline/load/spike/soak”. citeturn15view2turn6view3

**Vegeta**  
Позиционируется как HTTP load testing tool, созданный из потребности «бурить HTTP сервисы постоянной скоростью запросов». citeturn24search0  
Плюсы: компактность, CLI + библиотека, хорошо для “constant rate” drilling. citeturn24search0turn24search28  
Минусы: меньше встроенных high‑level концепций сценариев/отчётности в стиле k6; часто нужен собственный стандарт артефактов.

**wrk2**  
Описывается как инструмент HTTP benchmarking, “constant throughput, correct latency recording variant” wrk. citeturn24search1  
Плюсы: очень высокая производительность генератора, полезен для чистого HTTP‑throughput и latency distribution. citeturn24search1  
Минусы: скорее benchmarking инструмент (микро‑нагрузка), больше риска “непохожести” на реальные пользовательские сценарии; Lua‑скрипты ограничены по выразительности. citeturn24search1

**hey**  
Минималистичный генератор нагрузки для web‑приложений (ab‑replacement). citeturn24search2  
Плюсы: простота, быстрый sanity‑check. citeturn24search2  
Минусы: слабее сценарность/артефакты/стандарты; для серьёзного capacity planning template‑уровня обычно недостаточен.

**Locust**  
Документация описывает тест как Python‑программу, что удобно для сложных user flows. citeturn24search35turn24search3  
Плюсы: выразительность сценариев, распределённый режим. citeturn24search3  
Минусы: Python‑стек в Go‑репо может быть “template friction”; нужен свой стандарт thresholds/артефактов.

**Рекомендация template**: k6 как дефолт; Vegeta/wrk2/hey как дополнительные инструменты для узких задач (быстрый drilling, чистый HTTP benchmarking, sanity checks), но без замены основного performance‑suite. Обоснование: k6 обеспечивает нормативные тест‑типы, thresholds и экспорт результатов. citeturn15view2turn6view3turn15view1

### Open vs closed model

- Closed model удобен для “как пользователь” (итерация стартует после завершения предыдущей), но деградация SUT снижает нагрузку и может скрывать tail latency (coordinated omission). citeturn22view3  
- Open model отделяет arrival rate от времени ответа и подходит для проверки “можем ли выдержать X RPS”. citeturn22view3turn6view2  
**Template default**: open model для capacity/target RPS; closed model допускается только при явной цели «user journey simulation» и отдельной интерпретации.

### CPU limits для Go‑сервиса в контейнере

Это спорная зона, поэтому в template должна быть явная политика и trade‑offs.

Факты:
- Kubernetes CPU limits enforced throttling, что может влиять на latencies. citeturn8view2  
- Go‑команда описывает throttling как механизм с потенциально существенным tail‑impact, и отмечает, что до Go 1.25 `GOMAXPROCS` мог быть сильно выше CPU limit, что делало ситуацию хуже; в Go 1.25 дефолт стал container‑aware. citeturn9view4

Trade‑off:
- **С CPU limit**: проще получить предсказуемую верхнюю границу CPU и корректный `GOMAXPROCS` по умолчанию (Go 1.25), но вы принимаете риск дополнительных tail‑latency эффектов из‑за throttling при spikes. citeturn9view4turn8view2  
- **Без CPU limit (только request)**: возможна лучшая утилизация “idle CPU” при отсутствии contention, но предсказуемость меньше, а поведение при конкурентной нагрузке зависит от планировщика/“весов” requests. citeturn9view4turn8view0

**Boring default для template**:  
- requests для CPU/памяти — обязательны; память должна иметь разумный limit (иначе pod может потреблять всю память узла). citeturn8view3  
- наличие/отсутствие CPU limit делается переключаемым профилем (например Helm/Kustomize value), а в docs прописывается обязательная проверка tail latency под throttling (если лимит включён). citeturn9view4turn8view2

### Autoscaling signals

- CPU/memory scaling (resource metrics) требует корректных requests; иначе HPA не определит utilization. citeturn23view0  
- Custom/external metrics поддерживаются (autoscaling/v2) и читаются через APIs (`custom.metrics.k8s.io`, `external.metrics.k8s.io`). citeturn23view1  
- При нескольких метриках HPA берёт максимум рекомендованного размера, что позволяет “primary metric + safety net”. citeturn23view1  
- Flapping управляется stabilization window и scaling policies в `behavior`. citeturn23view2

**Default**: CPU как safety net + (если доступно) RPS/queue depth как primary metric, потому что они ближе к “user‑visible load”; это должно быть оформлено как опция, так как требует метрик‑адаптера. citeturn23view1turn23view0

## Набор правил в формате MUST / SHOULD / NEVER для LLM

### MUST

- MUST начинать performance‑план с явного описания SLI/SLO и окна измерения (latency/error/throughput), используя перцентили, а не средние значения. citeturn6view4turn11view0  
- MUST использовать thresholds как формальные pass/fail критерии (и генерировать скрипты/конфиги так, чтобы CI мог провалить прогон при нарушении SLO). citeturn6view3  
- MUST включать как минимум smoke + average‑load baseline тесты до stress/spike/soak, и явно использовать baseline для сравнений. citeturn15view2  
- MUST выбирать open model (arrival‑rate) для целей “гарантировать X RPS”, иначе объяснить риск coordinated omission при closed model. citeturn22view3turn6view2  
- MUST сохранять machine‑readable артефакты: `summary.json` (через `handleSummary()`), и при необходимости time‑series output (json/csv или external output). citeturn15view0turn15view1  
- MUST при анализе результатов проверять “четыре золотых сигнала” (latency/traffic/errors/saturation) и явно указывать, какой сигнал подтверждает гипотезу bottleneck’а. citeturn2search0  
- MUST при подозрении на bottleneck предлагать сбор pprof/trace и перечислять конкретные профили (CPU/heap/goroutine/trace; mutex/block — при contention). citeturn9view0turn9view2turn17search27turn17search1  
- MUST помнить, что diagnostic tools могут интерферировать; для точности профили нужно снимать в изоляции и фиксировать методику. citeturn22view4  
- MUST для autoscaling учитывать, что HPA по CPU использует utilization как процент от request; если request не задан, по этой метрике HPA не сможет действовать. citeturn23view0  
- MUST при обсуждении CPU limits и Go runtime учитывать контейнерную семантику: CPU limits → throttling; Go 1.25 → container‑aware GOMAXPROCS по CPU limit. citeturn8view2turn9view4  
- MUST пояснять, что Metrics Server предназначён для autoscaling и не является полноценной системой мониторинга. citeturn22view1

### SHOULD

- SHOULD рекомендовать open model executors (`constant-arrival-rate` / `ramping-arrival-rate`) для проверки capacity и выявления “knee point” на кривой. citeturn6view2turn22view3  
- SHOULD комбинировать метрики HPA (например custom + CPU safety) и объяснять, что HPA берёт максимум рекомендуемого размера между метриками. citeturn23view1  
- SHOULD включать стабилизацию и ограничение скорости scaling через `behavior` (stabilization window, scaling policies), чтобы снизить flapping. citeturn23view2  
- SHOULD при оценке “тысячи RPS” учитывать закон Литтла как sanity‑check согласованности latency/throughput/in‑flight. citeturn20view0  
- SHOULD предупреждать о точности квантилей из гистограмм и требовать корректные bucket boundaries около SLO. citeturn22view0  
- SHOULD при необходимости предлагать distributed load execution через k6 Operator и ссылаться на tagged releases как более стабильный способ установки. citeturn26search0turn26search1

### NEVER

- NEVER утверждать “сервис выдерживает N RPS”, если тест был в нерепрезентативной среде (другие requests/limits, другая сеть, другие зависимости) без явного дисклеймера и артефактов окружения. (Нормативный запрет: результаты без условий воспроизведения не являются стандартом доказательства.) citeturn14view0  
- NEVER опираться только на average latency как критерий качества; average скрывает long tail и вводит в заблуждение. citeturn11view0  
- NEVER строить target‑throughput выводы на closed model, не упомянув coordinated omission/влияние latency на подаваемую нагрузку. citeturn22view3  
- NEVER включать несколько “тяжёлых” профилировщиков одновременно и делать далеко идущие выводы без упоминания возможной интерференции. citeturn22view4  
- NEVER рекомендовать “просто поднять CPU limit” или “убрать CPU limit” как универсальный фикс latency без анализа throttling/Go runtime семантики и метрик saturation. citeturn8view2turn9view4  
- NEVER использовать Metrics Server как основу мониторинга/alerting вместо полноценной метрики/логов/трейсов системы. citeturn22view1

## Concrete good / bad examples

### Good: k6 target‑RPS тест (open model) с thresholds и `summary.json`

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    target_rps: {
      executor: 'constant-arrival-rate',
      rate: 1000,          // целевой arrival rate (итераций/сек)
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 200,
      maxVUs: 2000,
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],          // error budget
    http_req_duration: ['p(95)<200', 'p(99)<400'], // tail latency goals
  },
};

export default function () {
  const res = http.get(`${__ENV.BASE_URL}/healthz`);
  check(res, { 'status is 200': (r) => r.status === 200 });
  sleep(0.001);
}

export function handleSummary(data) {
  return {
    'summary.json': JSON.stringify(data),
  };
}
```

Почему это good:
- `constant-arrival-rate` — open model, decouples arrival rate от latency и подходит для RPS‑проверки. citeturn6view2turn22view3  
- thresholds оформляют SLO как pass/fail. citeturn6view3�
- `handleSummary()` сохраняет агрегированный объект метрик в `summary.json` как артефакт. citeturn15view0

### Bad: попытка “держать 1000 RPS” через closed model VUs

```javascript
export const options = { vus: 200, duration: '5m' };
export default function () { http.get(`${__ENV.BASE_URL}/api`); }
```

Почему это bad:
- closed model связывает throughput с временем ответа: при деградации SUT нагрузка “сама падает”, что может скрыть реальную проблему; k6 прямо описывает этот эффект и указывает coordinated omission. citeturn22view3

### Good: включение pprof endpoints (для тестовой/админской среды)

```go
package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("127.0.0.1:6060", nil))
	}()

	// ... основной сервер ...
}
```

Почему это good:
- `net/http/pprof` регистрирует handlers под `/debug/pprof/` и отдаёт runtime профили в формате pprof. citeturn9view0

### Good: trace для анализа scheduler/GC под нагрузкой

- execution trace фиксирует события goroutine scheduling, syscalls, GC, heap size changes и анализируется `go tool trace`. citeturn9view2turn21search1  
- trace можно скачать через `/debug/pprof/trace`, если импортирован `net/http/pprof`. citeturn9view2turn9view0

### Пример выводов, которые LLM должна уметь сделать по результатам

Если на plateau 3000 RPS:
- p99 вырос, error rate начал расти, а CPU usage приблизился к limit и видны признаки throttling → вероятный bottleneck CPU/throttling; дальше нужно проверить CPU limits/requests и Go runtime семантику (`GOMAXPROCS` vs limit), а также собрать CPU profile. citeturn8view2turn9view4turn9view1  
- p99 вырос, но CPU utilisation низкий, а время в блокировках/lock contention выросло → рассмотреть mutex/block профили и trace, т.к. CPU “не занят” из‑за ожиданий синхронизации. citeturn17search27turn17search1turn9view2  
- p95/p99 на границе SLO по Prometheus histogram_quantile и это “слишком плохо выглядит” → проверить bucket boundaries: Prometheus объясняет, что вычисленный quantile зависит от bucket’ов и может быть сильно искажён при резких пиках распределения. citeturn22view0

## Anti-patterns и типичные ошибки/hallucinations LLM

LLM‑ошибки, которые template должен предотвращать правилами и чеклистом:

- **“1000 RPS” на closed model без оговорок**: модель скрывает деградацию, когда latency растёт (throughput падает), classic coordinated omission. citeturn22view3  
- **Фиксация на average latency** и игнорирование tail: противоречит SRE‑best practice по p95/p99 и может маскировать реальную боль пользователей. citeturn11view0  
- **Непонимание механики HPA CPU utilization**: HPA считает utilization как % от request; при отсутствии request метрика undefined → autoscaling по CPU не работает. LLM часто “забывает” это и предлагает HPA, не задав requests. citeturn23view0  
- **Использование Metrics Server как мониторинга**: Metrics Server заявлен как компонент autoscaling pipeline, с предостережением “не использовать” как мониторинг‑источник. citeturn22view1  
- **Универсальная рекомендация “уберите CPU limit, и latency исправится”** (или наоборот): CPU limits → kernel throttling; Go‑документация объясняет tail‑impact и тонкости; без анализа это «магический совет». citeturn8view2turn9view4  
- **Снятие нескольких диагностических профилей одновременно** и сравнение “как есть”: Go предупреждает об интерференции инструментов. citeturn22view4  
- **Неправильная интерпретация квантилей Prometheus histograms** как “точных” чисел без учёта bucket boundaries: Prometheus прямо объясняет, что true percentile гарантированно внутри bucket interval, а single value — интерполяция, которая может выглядеть сильно хуже/лучше реальности. citeturn22view0  
- **Игнорирование эффекта масштаба на tail latency**: при fan‑out и большом числе компонент даже редкие задержки начинают доминировать на уровне сервиса; в “The Tail at Scale” показано, что при параллельном fan‑out на 100 серверов 99‑перцентиль 1s на одном сервере превращается в массовую долю медленных запросов на уровне сервиса. citeturn13view0  
- **Отсутствие артефактов**: LLM “пишет, что всё ок”, но не создаёт `summary.json`, time‑series output, профили/трейсы. Это ломает воспроизводимость и делает performance testing неаудитируемым. citeturn15view0turn15view1

## Review checklist для PR/code review и что оформить отдельными файлами в template repo

### Review checklist для performance‑готовности

В PR, который заявляет “готов к X тысячам RPS” или меняет performance‑критичный код/конфиг, reviewer должен требовать:

- Есть ли обновлённый performance contract (SLO/thresholds) и почему выбранные p95/p99 цели соответствуют сервису; запрещено принимать изменения, если критерии “размыты”. citeturn6view4turn11view0turn6view3  
- Есть ли baseline и сравнение с ним (trend/regression) — k6 рекомендует average‑load тест как baseline для сравнений. citeturn15view2  
- Доказано ли target RPS в open model (arrival‑rate), либо явно описано, почему выбран closed model и как учтён coordinated omission. citeturn22view3turn6view2  
- Есть ли артефакты теста: `summary.json` (handleSummary), end‑of‑test summary, при необходимости time‑series output. citeturn15view0turn15view1  
- При изменениях autoscaling: HPA использует requests корректно (requests заданы), метрики источников (metrics.k8s.io / custom/external) описаны, `behavior` настроен против flapping, multi‑metric логика учтена (max recommended). citeturn23view0turn23view1turn23view2  
- При изменениях requests/limits: CPU limits/throttling и влияние на tail latency учтены, версия Go runtime/семантика `GOMAXPROCS` указаны. citeturn8view2turn9view4  
- Есть ли результаты профилирования (pprof/trace) при подтверждении bottleneck‑fix; и не нарушено ли правило “tools in isolation” (если нужен точный диагноз). citeturn9view0turn9view2turn22view4  
- Для latency‑метрик через histograms: bucket boundaries адекватны SLO, иначе quantiles могут вводить в заблуждение. citeturn22view0  
- Поведение при overload: есть ли стратегия деградации/load shedding и проверка под перегрузкой (как минимум описательная, а лучше — тестовая). citeturn16view0

### Что вынести в отдельные файлы внутри template repo

Ниже — “почти напрямую” превращаемый список файлов и их назначение (структура может быть адаптирована под ваш репо‑layout, но смысл сохраняется):

- `docs/performance/load-testing.md`  
  Норматив: какие виды тестов обязательны (smoke/baseline/load/spike/soak), почему open vs closed model, как задавать thresholds, какие метрики собирать. Основание: k6 testing guides + thresholds + open/closed models. citeturn15view2turn6view3turn22view3

- `docs/performance/capacity-planning.md`  
  Норматив: как из результатов тестов выводить “RPS per pod”, headroom (N+2), regression baselines, sanity‑check через Little’s Law, ceilings (maxReplicas/узлы/зависимости). Основание: SRE capacity planning best practices + Little’s Law proof. citeturn16view0turn20view0turn14view0

- `docs/performance/scaling-strategy.md`  
  Норматив: HPA signals (CPU vs custom/external), multi‑metric max rule, `behavior` (stabilization/policies/tolerance), связь с metrics‑server, политика requests/limits и Go runtime (GOMAXPROCS). Основание: Kubernetes HPA docs + metrics-server + resource management + Go container-aware GOMAXPROCS. citeturn23view0turn23view1turn23view2turn22view1turn8view2turn9view4

- `docs/llm/performance.instructions.md`  
  MUST/SHOULD/NEVER из этого отчёта в виде “LLM system prompt addendum”: как генерировать тест‑план, как не галлюцинировать выводы, какие артефакты обязательно создавать, как интерпретировать pprof/trace и golden signals. Основание: k6 thresholds/outputs, Go diagnostics caveats, SRE tail latency guidance. citeturn6view3turn15view0turn22view4turn11view0turn2search0

- `perf/k6/`  
  Скрипты (как минимум): `smoke*.js`, `baseline*.js`, `load*.js`, `spike*.js`, `soak*.js` + общий модуль сценариев. Основание: k6 рекомендует переиспользовать логику сценариев и префиксовать тесты по типу нагрузки. citeturn15view2turn16view4

- `perf/results/` (в `.gitignore`, но со структурой и README)  
  Описание того, какие файлы должны появиться: `summary.json`, при необходимости time‑series (`test.json`) и как ими пользоваться. Основание: k6 results output + handleSummary. citeturn15view1turn15view0

- `perf/profiles/` (в `.gitignore` + README)  
  Соглашения по именованию pprof/trace файлов (`cpu.pb.gz`, `heap.pb.gz`, `trace.out`), и когда что снимать. Основание: net/http/pprof + runtime/trace. citeturn9view0turn9view2

- `perf/runbooks/perf-test-runbook.md`  
  Пошаговый “как запустить” (локально и distributed), включая k6 Operator вариант. Основание: k6 operator docs + running distributed tests guide. citeturn26search1turn26search2turn26search0

- (опционально) `deploy/autoscaling/`  
  HPA манифесты (autoscaling/v2) с `behavior`, пример multi‑metrics, и комментарии почему. Основание: Kubernetes HPA docs (behavior, stabilization, multi-metric). citeturn23view1turn23view2