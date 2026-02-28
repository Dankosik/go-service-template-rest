# Методология performance engineering для production-ready Go-микросервиса

## Scope

Эта методология предназначена для greenfield Go‑микросервиса, который должен быть **измеримо** быстрым и предсказуемым под нагрузкой, а не «оптимизированным на глаз». Она особенно уместна, когда вы можете (а) зафиксировать цели по пропускной способности и хвостовой задержке, (б) собирать метрики/трейсы/профили и (в) воспроизводимо прогонять репрезентативную нагрузку и сравнивать результаты. В терминах наблюдаемости она опирается на «golden signals» (latency/traffic/errors/saturation) и практику SLO/SLI как способ формализовать «что считать достаточно быстрым». citeturn19search3turn19search2turn13search1

Подход **не подходит как “default”** для:
- одноразовых CLI/скриптов и мелких утилит, где стоимость построения инфраструктуры измерений превышает пользу (хотя локальные бенчмарки и профили всё равно полезны); citeturn14view0turn20view0
- систем, где performance‑цели не выражены в SLO/продуктовых требованиях, и команда не готова поддерживать «эталонные» сценарии/датасеты (в этом случае оптимизация почти неизбежно скатывается в несистемные правки); citeturn19search2turn13search1
- кейсов, где «узкое место» почти целиком вне сервиса (например, вы ограничены внешним API/хранилищем), и без контроля над внешними зависимостями вы не сможете стабильно измерять end‑to‑end; тут всё равно нужно начинать с наблюдаемости и SLO, но ожидать больших выигрышей от микроскопических оптимизаций в коде — ошибка методологии; citeturn19search3turn9view2
- требований уровня «ультранизкая задержка любой ценой» (HFT/RTOS), где возможны другие компромиссы, другие инструменты и иной профиль рисков.

В template‑репозитории эту тему следует позиционировать как **строгую процедуру**: «сначала цель → потом измерение → затем локализация bottleneck → потом изменения → затем статистически корректная проверка и регрессионный контроль». Стандартная диагностика Go даёт встроенные опоры: benchmark framework, pprof (CPU/heap/… профили), execution traces, и инструменты визуализации/сравнения. citeturn14view0turn17view0turn20view1turn1search1turn18view0

## Recommended defaults для greenfield template

Ниже — «boring, battle‑tested defaults», которые нужно закодировать в **репозиторий** (Makefile/скрипты/CI) и в **LLM‑инструкции**, чтобы модель не «фантазировала оптимизации», а действовала по методологии.

**Базовая платформа**
- Минимальная версия Go для template: **Go 1.26** (актуальный релиз на февраль 2026) с фиксацией в `go.mod` и в документации репозитория. citeturn15search0turn15search5
- Внутренний SLA на производительность формулировать через SLO/SLI (например, “99% запросов < 250ms при 2k RPS, error rate < 0.1%”). Это ровно тот тип измеримых индикаторов, на котором строится SRE‑подход. citeturn19search2turn13search1turn19search3

**Обязательная наблюдаемость для performance**
- **RED + saturation** как минимальный набор:  
  - RED: Rate / Errors / Duration, как практичный минимум для request‑driven микросервисов (исходная формулировка популяризирована entity["people","Tom Wilkie","observability engineer"]). citeturn8search0turn19search3  
  - Saturation/Utilization/Errors по USE‑методологии (entity["people","Brendan Gregg","performance engineer"]) — чтобы видеть «упёрлись в ресурс» и не путать это с «код медленный». citeturn8search1turn19search3
- **Латентность измерять распределениями, а не средним**: для микросервисов критичен хвост (p95/p99), особенно под высокой утилизацией; tail latency — системный эффект, а не “шум”. citeturn3search3turn19search3

**Метрики задержки: Prometheus‑style гистограммы**
- Для серверной задержки по умолчанию использовать **гистограммы**, потому что их можно агрегировать между инстансами и вычислять квантили на стороне сервера, тогда как агрегировать client‑side quantiles из summary обычно статистически бессмысленно. citeturn9view2turn2search12
- Если вы используете OpenTelemetry‑семантику, базовый ориентир по бакетам (ExplicitBucketBoundaries advisory) для `http.server.request.duration` уже задан спецификацией. citeturn9view3  
  Практический стандарт для template: «бакеты привязаны к SLO (вокруг порогов), но стартуем с рекомендованных и уточняем по данным».

**Трейсинг**
- Для distributed tracing в template закладывать OpenTelemetry (как vendor‑neutral стандарт в экосистеме и часть entity["organization","CNCF","cloud native foundation"]).  
  Ключевой default: **sampling обязателен** (иначе трейсинг под высокой нагрузкой становится дорогим), причём политика должна быть явно описана в docs и конфигурируема. citeturn2search6turn20view0
- Минимум: трассировать **входящий запрос end‑to‑end** и ключевые внешние вызовы (БД/кэш/HTTP‑клиенты), чтобы отличать «наш код» от «ждём сеть/БД». Это прямо соответствует назначению distributed tracing как инструмента анализа latency цепочек. citeturn20view0

**Профилирование**
- Встроенный стандарт: `pprof` как опорный механизм профилирования. Go‑документация прямо описывает сбор профилей через `go test` и через HTTP endpoints `net/http/pprof`, а также рекомендует практики вроде периодического профилирования случайной реплики, и предупреждает, что сбор профилей может мешать друг другу (поэтому — один профиль за раз). citeturn20view0turn20view1
- В template профилирование размещать на **отдельном debug‑listener/порту** и/или отдельном mux, а не смешивать с публичным API; стандартная документация Go показывает, что handlers можно регистрировать на другом пути/порту, не только на default mux. citeturn20view0turn20view1
- Для визуализации и анализа считать обязательными: `go tool pprof` (включая web UI) и режим `-http` у pprof для интерактивного просмотра. citeturn17view0turn12view0turn20view0

**Execution traces (Go runtime tracing)**
- В template нужно иметь «быстрый путь» до `go tool trace`: трассы могут показать то, чего не видно в CPU профиле (например, блокировки/простои/планировщик), и Go прямо развивает этот инструментарий как диагностику исполнения горутин. citeturn18view0turn1search1turn3search2
- Для production‑сценариев под высокой нагрузкой полезно знать про flight recorder (в Go 1.25+): он решает типичную проблему «мы не успели включить трассировку до инцидента», буферизуя последние секунды и позволяя снять snapshot при триггере (например, медленный запрос). citeturn18view2turn18view1

**Бенчмаркинг и сравнение результатов**
- Для микробенчмарков: использовать фреймворк `testing` и его правила, включая современный `B.Loop()` (он аккуратно управляет таймером и снижает риск оптимизаций компилятора “в ноль”), плюс измерение аллокаций через `ReportAllocs`/`-benchmem`. citeturn14view0turn7view0
- Для статистически корректного A/B‑сравнения бенчмарков: `benchstat` из `golang.org/x/perf/cmd`. Это прямо рекомендовано документацией Go как инструмент “statistically robust A/B comparisons”. citeturn14view0turn0search7  

image_group{"layout":"carousel","aspect_ratio":"16:9","query":["go tool pprof flame graph web ui","go tool trace viewer screenshot","prometheus histogram_quantile latency dashboard","opentelemetry trace waterfall microservice"],"num_per_query":1}

## Decision matrix / trade-offs

**Микробенчмарки vs нагрузочное тестирование**
- Микробенчмарки (`go test -bench`) хороши для: сравнения реализаций алгоритма, поиска аллокаций, regression‑контроля на уровне функции/пакета. Но они по определению не моделируют сеть/пулы соединений/GC под реальной смесью запросов. Фреймворк бенчмарков специально подбирает число итераций для надёжного замера и требует аккуратного setup/тайминга. citeturn14view0turn7view0  
- Нагрузочные тесты нужны, чтобы зафиксировать «ресурс‑к‑ёмкости» и проверить хвостовую задержку под целевой утилизацией, как рекомендует SRE‑практика capacity planning и service best practices. citeturn13search8turn13search4turn19search20

**pprof профили vs execution traces**
- CPU/heap профили (pprof) показывают “где тратим CPU/память” и основаны на сэмплинге; в Go CPU‑профилирование по описанию снимает порядка 100 выборок в секунду, поэтому слишком короткие замеры дают мало данных. citeturn17view0turn20view1  
- Execution trace показывает «когда горутины не исполняются» (блокировки, планировщик, паузы), что может быть невидимо в CPU‑профиле. Go прямо подчёркивает эту разницу и улучшал overhead/масштабируемость трасс. citeturn18view0turn1search1

**Distributed tracing vs runtime execution trace**
- Distributed tracing отвечает на вопрос “где задержка по цепочке RPC между сервисами”, особенно когда bottleneck проявляется только в проде и не локализуется профайлером одного процесса. citeturn20view0  
- Runtime execution trace отвечает на вопрос “что происходило внутри процесса и рантайма” (scheduler, горутины, ожидания), и полезен при contention/GC/блокировках. citeturn18view0turn18view2  
Практический вывод для template: **оба нужны**, но включаются и используются по разным триггерам.

**Prometheus histogram vs summary для latency percentiles**
- Summary считает φ‑квантили на клиенте и в общем случае **не агрегируется** между инстансами; histogram агрегируется и даёт возможность вычислять квантили функцией `histogram_quantile`, причём Prometheus документация прямо показывает пример “avg(…quantile…) // BAD” и “histogram_quantile(…) // GOOD”. citeturn9view2turn2search12  
Trade‑off: histogram требует выбора бакетов (точность vs стоимость/кардинальность), summary может быть точнее локально, но обычно проигрывает в распределённой агрегации. citeturn9view2turn2search12

**Sampling в трейсинге**
- Полные трейс‑потоки на высокой нагрузке дороги; OTel прямо перечисляет ситуации, когда sampling нужен (например, высокий TPS трасс), и рекомендует правила отбора (ошибки/высокая задержка/доменные критерии). citeturn2search6turn20view0  
Trade‑off: агрессивный sampling снижает стоимость, но ухудшает diagnosability “редких” путей; поэтому template должен фиксировать политику и иметь “escape hatch” для увеличения уровня при расследовании.

**Включать ли профилирование в проде**
- Go‑диагностика допускает и даже рекомендует практику периодического профилирования прод‑реплик, но подчёркивает, что сбор профилей может мешать друг другу, и это надо контролировать. citeturn20view0  
- С другой стороны, `net/http/pprof` исторически обсуждался как рискованный по умолчанию из‑за “слишком легко случайно выставить наружу”, и существуют отдельные issue по security implications. citeturn16search4turn16search1  
Trade‑off для template: **по умолчанию debug endpoints выключены/закрыты**, включаем только в доверенной сети/под аутентификацией/через отдельный порт.

## Набор правил MUST / SHOULD / NEVER для LLM

Ниже — формулировки, которые стоит почти напрямую копировать в `docs/llm/performance.md` и в «system prompt» для кодогенерации.

**MUST**
- MUST начинать работу с performance‑изменением с фиксации цели: какой SLI/SLO затрагивается (latency percentile / throughput / saturation / error rate) и каким тестом/замером это будет подтверждено. SLO/SLI должны быть количественными и проверяемыми. citeturn19search2turn13search1turn19search3  
- MUST добавлять или обновлять измерения вместе с изменением:  
  - для локальной компонентной оптимизации — benchmark (`go test -bench`) и сравнение результатов через `benchstat`; citeturn14view0turn0search7  
  - для end‑to‑end — reproducible load‑сценарий и проверка хвостовой задержки (p95/p99), а не только среднего. citeturn3search3turn13search8turn13search3
- MUST писать новые микробенчмарки в современном стиле `for b.Loop() { ... }`, а если используется b.N‑стиль — корректно отделять setup через `b.ResetTimer()`. citeturn14view0  
- MUST измерять аллокации при оптимизациях “под нагрузку”: применять `ReportAllocs`/`-benchmem` и трактовать изменения в allocs/op и B/op как первые сигналы риска для GC/latency tail. citeturn14view0turn7view0  
- MUST использовать профили/трейсы для локализации bottleneck до масштабных рефакторингов:  
  - CPU/heap профили анализировать через `go tool pprof` (включая web UI); citeturn17view0turn12view0turn20view0  
  - execution trace анализировать через `go tool trace`, особенно при подозрении на блокировки/контеншн. citeturn1search1turn18view0turn20view1
- MUST встраивать метрики latency как histogram (агрегируемые) и не использовать агрегирование quantiles из summaries между инстансами. citeturn9view2turn2search12  
- MUST соблюдать низкую кардинальность меток/атрибутов: метрики и OTel‑атрибуты не должны размножаться по user input (ID, email, raw URL), иначе наблюдаемость сама становится причиной деградации. (В OTel семантике прямо подразумевается контролируемое количество атрибутов для “finely tuned filtering”, а в Prometheus‑подходе стоимость временных рядов делает этот риск практическим.) citeturn9view3turn9view2
- MUST документировать, как включать pprof/trace endpoints безопасно (отдельный порт/путь, ограничение доступа), и уметь переназначать handlers на кастомный mux/порт. citeturn20view0turn20view1turn16search4

**SHOULD**
- SHOULD формулировать performance‑гипотезы как «если X — то мы увидим Y в профиле/метриках» и проверять их одним изменением за раз (избегать “больших пакетов оптимизаций” без измерения). Основание: инструментарий pprof/trace создан именно для причинно‑следственного анализа. citeturn20view0turn18view0turn17view0  
- SHOULD использовать SRE‑ориентированный минимум сигналов: golden signals + RED + saturation/USE, чтобы не тонуть в случайных метриках. citeturn19search3turn8search0turn8search1  
- SHOULD применять k6 (или эквивалент) для автоматизации SLO‑подобных thresholds (p95/p99 и error rate) в нагрузочном тестировании. citeturn13search3  
- SHOULD на Go 1.25+ рассмотреть flight recorder как “production‑safe” способ поймать след медленных запросов без постоянного многосекундного tracing‑дампа. citeturn18view2turn18view1  
- SHOULD рассматривать PGO (profile‑guided optimization) как «второй этап» после того, как измерения и профили стабилизированы, и есть уверенность в representative workload для профиля. citeturn2search19

**NEVER**
- NEVER делать “оптимизации” без воспроизводимого измерения (benchmark/profile/trace/load test). Любая «оценка» без замера должна быть явно оформлена как гипотеза + план проверки. citeturn20view0turn14view0  
- NEVER заменять histogram‑подход на summary только ради “удобных квантилей”, если метрика должна агрегироваться между инстансами. citeturn9view2turn2search12  
- NEVER включать `net/http/pprof` на публичном интерфейсе сервиса “как есть”; только через отдельный listener/порт и контроль доступа. citeturn16search4turn16search1turn20view0  
- NEVER путать pprof‑профили и execution trace: `/debug/pprof/trace` открывается `go tool trace`, а не `go tool pprof`. citeturn20view1turn1search1turn18view1  
- NEVER “подкручивать” параллелизм тестов/фаззинга выше `GOMAXPROCS` без нужды: документация `go test` прямо предупреждает про риск деградации из‑за CPU contention. citeturn7view0

## Concrete good / bad examples

**Good: микробенчмарк с отделением setup, измерением аллокаций и реалистичной нагрузкой на данные**

```go
package mypkg

import (
	"bytes"
	"testing"
)

func BenchmarkNormalizeJSON(b *testing.B) {
	// Setup: реалистичный payload (лучше — из testdata/).
	payload := bytes.Repeat([]byte(`{"a":"b","n":123,"arr":[1,2,3]}`), 32)

	b.ReportAllocs()

	// Современный стиль: b.Loop сам управляет таймером и мешает компилятору выкинуть тело.
	for b.Loop() {
		_, _ = NormalizeJSON(payload)
	}
}
```

Этот шаблон следует правилам `testing` для бенчмарков (включая `B.Loop` и `ReportAllocs`) и делает результат полезным для GC/latency‑рисков через allocs/op. citeturn14view0

**Bad: микробенчмарк “меряет таймер и аллокации бенчмарка”, а не функцию**

```go
func BenchmarkBad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Плохая идея: time.Now() и форматирование часто доминируют над полезной работой.
		NormalizeJSON([]byte(`{"a":"b"}`))
	}
}
```

Проблема не в b.N‑стиле как таковом, а в отсутствии отделения setup и в том, что замер легко “съезжает” на посторонние расходы; документация `testing` прямо объясняет, что b.N‑бенчмарк может вызываться многократно с разными N и что setup до цикла может выполняться несколько раз, поэтому таймер нужно сбрасывать, если setup дорогой. citeturn14view0

**Good: статистически корректное сравнение (A/B) через benchstat**

```bash
go test -run=^$ -bench=. -benchmem -count=10 ./... > old.txt
# применили изменения
go test -run=^$ -bench=. -benchmem -count=10 ./... > new.txt

benchstat old.txt new.txt
```

`benchstat` предназначен для статистических сводок и A/B‑сравнений результатов бенчмарков. citeturn0search7turn14view0

**Good: сбор профиля и просмотр trace правильным инструментом**

```bash
# CPU профиль на 30 секунд
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Execution trace на 5 секунд (важно: смотреть через go tool trace)
curl -o trace.out "http://localhost:6060/debug/pprof/trace?seconds=5"
go tool trace trace.out
```

Эти команды соответствуют официальным примерам `net/http/pprof` и `cmd/trace`. citeturn20view1turn1search1

## Anti-patterns и типичные ошибки/hallucinations LLM

Ошибки ниже стоит включить в “Known LLM failure modes” в репозитории, чтобы ревьюер мог быстро распознавать опасные автогенерации.

- **Premature optimization вместо измерения**: модель переписывает код “на более быстрый” (sync.Pool, unsafe‑конверсии, ручные буферы) без бенчмарка, профиля и цели по SLO. Это ломает поддерживаемость и часто ухудшает хвостовую задержку из‑за GC/контеншна. Методологически это противоречит подходу “диагностировать по профилям/трейсам” и SLO‑ориентированному процессу. citeturn20view0turn14view0turn19search2
- **Подмена хвостовой задержки средними**: модель добавляет только `avg latency` или summary‑quantile и потом агрегирует его между инстансами. Prometheus документация прямо отмечает, что такое агрегирование quantiles обычно “nonsensical”, и показывает BAD/GOOD паттерны. citeturn9view2turn2search12
- **Нереалистичный workload**: нагрузочный тест гоняет один endpoint синтетическим payload’ом без данных, похожих на прод; затем по результатам “оптимизируют” код. SRE‑практика подчёркивает необходимость load testing для установления “ресурс‑к‑ёмкости”, а paper про tail latency объясняет, почему под нагрузкой меняется распределение задержек и хвост критичен. citeturn13search8turn3search3
- **Путаница инструментов: trace vs pprof**: LLM часто предлагает смотреть `/debug/pprof/trace` через `go tool pprof`. В реальности `net/http/pprof` документирует, что trace собирается в файл и анализируется `go tool trace`. citeturn20view1turn1search1
- **Экспозиция debug endpoints наружу**: LLM «для удобства» добавляет `_ "net/http/pprof"` в основной HTTP‑сервер без отдельного порта/доступа. По Go‑issue обсуждаются security implications таких endpoints, а официальная диагностика Go предлагает возможность вынести handlers на другой порт/путь. citeturn16search4turn20view0turn20view1
- **Микробенч ловит шум CI и объявляет “регрессию”**: сравнение одного прогона `go test -bench` в shared CI без -count и без benchstat. Документация Go подчёркивает, что `benchstat` делает статистически робастные сравнения, а сами бенчмарки запускаются многократно и чувствительны к среде. citeturn14view0turn0search7
- **“Увеличьте parallelism выше GOMAXPROCS” как универсальный совет**: документация `go test` предупреждает, что это может ухудшить производительность из‑за CPU contention (особенно в fuzzing). citeturn7view0
- **Снятие нескольких профилей одновременно**: LLM инициирует параллельный сбор CPU/heap/trace одновременно “для полноты”. Go диагностика прямо предупреждает, что сбор профилей может мешать друг другу, и рекомендует собирать по одному за раз. citeturn20view0turn20view1

## Review checklist для PR/code review

Этот checklist стоит внедрить как `docs/review/performance.md` и как секцию PR template.

- Изменение содержит явную цель: какой SLI/SLO (latency percentile / throughput / saturation / error rate) улучшает или защищает. citeturn19search2turn13search1  
- Есть воспроизводимый метод измерения: benchmark/profile/trace и/или load test. Если заявлено ускорение — приложены результаты и способ их получить. citeturn14view0turn20view0turn18view0  
- Микробенчмарки:
  - используют `b.Loop()` или корректный b.N‑стиль с `ResetTimer` при дорогом setup; citeturn14view0  
  - измеряют аллокации (`-benchmem`/`ReportAllocs`) при performance‑изменениях; citeturn14view0turn7view0  
  - A/B сравнение сделано через `benchstat`, а не “на глаз” по одному прогона. citeturn0search7turn14view0
- Метрики latency реализованы как histogram и пригодны для агрегации между инстансами; нет “avg quantile” или иных статистически некорректных агрегаций. citeturn9view2turn2search12  
- Метки/атрибуты метрик и трейсинга не имеют высокой кардинальности (нет user_id, raw path, request_id как label). citeturn9view3turn9view2  
- pprof/trace endpoints:
  - вынесены на отдельный listener/порт или защищены;  
  - документирован способ включения и ограничения доступа;  
  - не смешаны с публичным API. citeturn20view0turn20view1turn16search4
- Если изменения затрагивают concurrency/контеншн, приложены либо trace‑аргументы, либо pprof block/mutex профили/обоснование. Go подчёркивает, что execution traces хорошо выявляют concurrency bottlenecks. citeturn18view0turn20view1  
- Изменения не ухудшают tail latency (p95/p99) при целевой нагрузке; нагрузочные thresholds оформлены как “тест‑критерии”, а не ручная интерпретация. citeturn13search3turn3search3

## Что из результата нужно оформить отдельными файлами в template repo

Ниже — конкретный список артефактов для `docs/` и repo‑conventions. Форматируйте так, чтобы LLM могла ссылаться на них как на «единственный источник правды», а ревьюер — быстро находить правила.

- `docs/performance/methodology.md`  
  Канонический документ «когда оптимизировать и как мерить»: golden signals → SLO/SLI → выбор метода (bench/profile/trace/load) → процедура локализации bottleneck → процедура подтверждения улучшения. citeturn19search3turn19search2turn20view0turn14view0turn18view0
- `docs/performance/benchmarking.md`  
  Стандарты написания бенчмарков (B.Loop, ResetTimer, RunParallel, benchmem, -count), стандартные команды запуска и обязательный `benchstat` для сравнений. citeturn14view0turn7view0turn0search7
- `docs/performance/profiling.md`  
  Как включать pprof безопасно, как собирать CPU/heap/block/mutex профили, как пользоваться `go tool pprof` и web UI, почему “один профиль за раз”. Включить примеры вынесения handlers на отдельный порт/путь. citeturn20view0turn20view1turn12view0turn16search4
- `docs/performance/tracing.md`  
  Разделить на:
  - distributed tracing (цели, sampling‑политика, что трассируем); citeturn20view0turn2search6  
  - runtime execution trace (`go tool trace`, сценарии “контеншн/простои/планировщик”); citeturn18view0turn1search1turn20view1  
  - flight recorder как прод‑техника (опционально). citeturn18view2turn18view1
- `docs/performance/metrics.md`  
  RED + saturation/USE, правила гистограмм и percentiles, запреты по кардинальности, рекомендации по бакетам (в том числе из OTel semconv). citeturn8search0turn8search1turn19search3turn9view2turn9view3
- `docs/llm/performance.md`  
  Отдельный LLM‑instruction документ с MUST/SHOULD/NEVER и типичными ошибками; этот файл должен быть “подключаемым префиксом” для ChatGPT/Codex/Claude Code. Основание — правила `testing`, `pprof`, `trace`, `benchstat`, SRE SLO и Prometheus histogram guidance. citeturn14view0turn20view1turn1search1turn0search7turn19search2turn9view2
- `docs/review/performance-checklist.md`  
  PR‑чеклист из раздела выше.

И обязательная «исполняемая» часть template (не только docs):
- `Makefile`/`taskfile` таргеты: `bench`, `bench-cpu`, `bench-mem`, `trace`, `pprof-ui`, `loadtest` (команды и флаги должны ссылаться на docs). Флаги `go test` (включая `-bench`, `-benchtime`, `-count`, `-cpu`, `-trace`, профили) документированы в help `go test/testflag`. citeturn7view0turn14view0turn20view1  
- `internal/debug/` пакет для debug‑listener (pprof/healthz/metrics в доверенной сети) + конфиги включения/ограничения доступа, согласно рекомендациям Go diagnostics о разнесении handlers. citeturn20view0turn20view1  
- `loadtest/` (например, k6‑скрипты) с thresholds по p95/p99/error rate как «go/no-go» критерии. citeturn13search3