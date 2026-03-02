# Performance: DB, cache и I/O слой в production-ready Go микросервисе

## Scope

Этот стандарт применяйте для greenfield Go‑микросервисов, у которых есть “горячий” путь обработки запросов с удалёнными зависимостями: реляционная/NoSQL БД, распределённый кэш (например, Redis), объектное хранилище, внешние HTTP‑API. Цель — предсказуемая производительность (особенно tail latency), управляемая нагрузка на зависимости и отсутствие “скрытых” деградаций из‑за неправильного управления соединениями/ресурсами. Это особенно критично в архитектурах с fan‑out (много параллельных обращений к зависимостям на один входящий запрос): даже если средняя задержка мала, 99‑й перцентиль быстро ухудшается с ростом fan‑out. citeturn17search0

Подход **не** предназначен как “универсальный тюнинг на глаз”. Он задаёт boring, battle‑tested дефолты и рамки проектирования data access / cache / I/O слоя, чтобы LLM (ChatGPT/Codex/Claude Code и т.п.) могла генерировать код без догадок: с явными лимитами, правильным управлением соединениями и корректной семантикой кэширования/инвалидации. citeturn9view0turn9view1turn1search0turn17search15turn18view0

Когда **не применять** (или применять с жёсткими оговорками):
- Длинные batch/ETL/аналитические задачи, тяжёлые отчёты, “длинные” транзакции/сессии: тут часто нужны другие приоритеты (throughput вместо latency), другие окна таймаутов, иной подход к connection pooling (иногда — вообще без pooler‑прокси). Например, PgBouncer в transaction pooling хорошо подходит для OLTP, но ограничивает фичи, требующие устойчивых сессий, и не идеален для длительных операций. citeturn21view0
- Среды, где соединения часто обрываются/пересоздаются (часть serverless‑сценариев, aggressive autoscaling) — потребуется внешний пулер/прокси и отдельные правила “connection budget” на инстанс. citeturn12search2turn12search36

В источниках ниже приоритет отдан primary/authoritative guidance: Go docs, PostgreSQL/MySQL/Redis docs, entity["company","Amazon Web Services","cloud provider"], entity["company","Google","technology company"], entity["company","Microsoft","software company"], entity["organization","OWASP","web security nonprofit"], entity["organization","OpenTelemetry","observability project"], entity["organization","PostgreSQL Global Development Group","postgresql project"]. citeturn8view0turn9view0turn11view0turn15view2turn18view0turn20view2

## Recommended defaults для greenfield template

### DB: соединения, запросы, батчи

Базовый слой доступа к SQL‑БД в Go‑шаблоне должен строиться вокруг `database/sql`, где `*sql.DB` — это **пул соединений**, безопасный для конкурентного использования, а не “одно соединение”. Каждая операция (`Query/Exec`) берёт соединение из пула или создаёт новое при необходимости; соединение возвращается в пул, когда больше не нужно. citeturn9view0turn8view0

Дефолты для пула (обязательные env‑параметры шаблона):
- `DB_MAX_OPEN_CONNS`: **обязателен** и должен быть >0. По умолчанию `database/sql` не ограничивает число открытых соединений (`SetMaxOpenConns(n<=0)` ⇒ “no limit”), что в проде часто приводит к исчерпанию `max_connections` на стороне БД и лавинообразной деградации. citeturn5view0turn9view0turn3search6  
- `DB_MAX_IDLE_CONNS`: задавайте явно; помните, что дефолт — “держать 2 idle connections”, и это может быть недостаточно при параллелизме (лишние reconnection‑затраты), либо наоборот — избыточно, если `MAX_OPEN` мал и idle не нужен. citeturn9view0turn5view0  
- `DB_CONN_MAX_IDLE_TIME` и `DB_CONN_MAX_LIFETIME`: задавайте явно для “гигиены” и для сред с балансировкой/проксированием на уровне БД, где “вечные” соединения — плохая идея. Go docs прямо отмечают, что без `SetConnMaxLifetime` соединение может переиспользоваться неопределённо долго и что lifetime может быть полезен в системах с load‑balanced database server. citeturn9view0turn5view0

Стартовые “boring” значения (как дефолт шаблона, **с обязательной возможностью переопределения**):
- `DB_MAX_OPEN_CONNS`: 10–30 на инстанс для OLTP‑сервисов как initial guess; далее тюнинг по метрикам `DB.Stats()` и бюджету соединений на БД. (Это эвристика, не “стандарт из документации”; документируйте, что итог зависит от `max_connections`, числа реплик сервиса, latency запросов и think time.) citeturn5view0turn3search6turn12search0  
- `DB_MAX_IDLE_CONNS`: для MySQL драйвера `go-sql-driver/mysql` рекомендовано ставить `SetMaxIdleConns()` равным `SetMaxOpenConns()`, иначе соединения могут открываться/закрываться гораздо чаще, чем ожидается (connection churn). citeturn14view0  
- `DB_CONN_MAX_LIFETIME`: 30–60 минут; `DB_CONN_MAX_IDLE_TIME`: 5–15 минут (типичные безопасные значения для ротации idle‑соединений; при serverless/частых рестартах — может потребоваться меньше). Go docs объясняют семантику idle time/lifetime и закрытие “просроченных” коннектов. citeturn9view0turn5view0

Жёсткое правило шаблона: **любая операция к БД должна иметь deadline** через `context.Context`. Иначе “подвисший” запрос держит соединение и ухудшает throughput пула; таймауты в `database/sql` работают корректно только если драйвер поддерживает cancellation. citeturn7view1turn9view0

Prepared statements:
- Для безопасности и предсказуемости **все параметры в SQL должны быть привязаны как параметры**, а не конкатенироваться строками; это также базовая защита от SQL injection. citeturn2search2  
- В PostgreSQL `PREPARE` действительно может дать выигрыш, уменьшая повторяющуюся работу парсинга/анализа, но prepared statements живут только в рамках **текущей сессии** и не шарятся между “клиентами” (соединениями). Это напрямую влияет на дизайн при использовании pooler’ов и при большом количестве соединений. citeturn15view2turn12search0  
- В Go закрывайте `Stmt` после использования: документация `database/sql` прямо предупреждает, что prepared statements занимают ресурсы на сервере и должны закрываться после использования. citeturn7view1  
- В PostgreSQL prepared statements могут использовать generic или custom plans; “универсальный” prepared statement может внезапно стать медленнее на некоторых распределениях данных, поэтому включайте в стандарт правило: “Измеряй, смотри EXPLAIN, не предполагай”. citeturn15view2turn15view3

Batching / bulk writes:
- Для PostgreSQL при массовой загрузке данных предпочитайте `COPY` как более эффективный механизм, чем `INSERT` (официальная документация отмечает это как рекомендацию). citeturn15view0turn15view1  
- Для MySQL официально рекомендованы multi‑row `INSERT ... VALUES (...), (...), ...` как “значительно быстрее” серии single‑row INSERT; а `LOAD DATA` обычно ещё быстрее при загрузке из файла. citeturn14view1turn14view3  
- Для InnoDB MySQL полезно учитывать физику clustered index: bulk‑вставки быстрее при вставке в порядке `PRIMARY KEY` и multi‑row INSERT снижает communication overhead. citeturn14view2

Query optimization и индексы:
- Индексы ускоряют чтение, но “неправильные” индексы могут ухудшить производительность (стоимость записи/хранения/планирования). Это прямо сказано в документации по `CREATE INDEX`. citeturn15view4  
- Обязательный инструмент в стандарт: `EXPLAIN` для анализа планов и фактической стратегии доступа (seq scan vs index scan и т.п.) — официальный путь проверки “почему медленно”. citeturn15view3turn1search15

Connection budget (принцип, который должен быть написан в шаблоне):
- PostgreSQL использует модель “process per user”: на каждое клиентское подключение создаётся отдельный backend process. Это делает большое число соединений дорогостоящим по ресурсам и усиливает деградации при “переподключениях” и oversubscription. citeturn12search0  
- У PostgreSQL параметр `max_connections` ограничивает максимум конкурентных соединений (типично 100 по умолчанию, зависит от окружения). citeturn3search6  
- В managed‑средах может быть дополнительный резерв/вычет из `max_connections` (например, Azure описывает формулу доступных user connections как `max_connections - (reserved + superuser_reserved)`), поэтому “бюджет соединений” должен считаться по документации конкретного провайдера. citeturn3search30turn3search6  
- В Cloud Run/Cloud SQL есть лимиты “соединений на инстанс” (пример: 100 при использовании built‑in Cloud SQL connection), и при масштабировании числа инстансов общее число соединений растёт. Это должно быть явно отражено в правилах выбора `DB_MAX_OPEN_CONNS`. citeturn12search2turn12search16

Если ожидается большое число краткоживущих соединений (или serverless‑пики), включайте в шаблон “план Б”: внешний пулер/прокси (PgBouncer/RDS Proxy). Для PostgreSQL PgBouncer часто используют в transaction pooling (хорош для OLTP), но он ограничивает features, требующие persistent sessions, включая prepared statements, сохраняющиеся между транзакциями. citeturn21view0turn19view0  
Для entity["company","Amazon Web Services","cloud provider"] RDS Proxy описывает connection pooling параметры (idle timeouts, max connections percent и т.п.) и возвращение underlying DB‑соединений в пул при простое client connection. citeturn12search36

### Cache: стратегия, TTL, защита от stampede

Базовый дефолт шаблона: **cache‑aside (lazy loading)**, потому что он прост, прозрачен, и именно приложение контролирует консистентность и инвалидацию. citeturn17search15turn18view1

image_group{"layout":"carousel","aspect_ratio":"16:9","query":["cache-aside pattern diagram","redis client-side caching tracking diagram","redis pipelining diagram","database connection pool diagram"],"num_per_query":1}

Ключевые дефолты:
- Все cache keys должны иметь TTL (исключения — редкие случаи write‑through, когда ключ гарантированно обновляется на запись). AWS отдельно выделяет правило “всегда ставь TTL”, чтобы баги инвалидации не превращались в вечный stale cache. citeturn18view0  
- TTL нельзя делать “слишком коротким” без необходимости: в cache‑aside это увеличивает число cache misses и нагрузку на origin/data store. Это отмечено в guidance по cache‑aside у entity["company","Microsoft","software company"]. citeturn1search1turn17search15  
- Stampede (thundering herd / dog‑pile): AWS описывает эффект как ситуацию, когда множество процессов одновременно получают cache miss и бьют одинаковый запрос в БД; TTL‑массовые истечения могут усугублять это. AWS предлагает mitigation через prewarm и через добавление randomness к TTL, чтобы ключи не истекали синхронно. citeturn18view0  
- На уровне одного инстанса Go‑сервиса стандартный инструмент “request coalescing” — `singleflight.Group` (duplicate suppression в рамках ключа). Это должно быть частью шаблона для горячих ключей, чтобы при одном инстансе не было N одинаковых запросов в БД. citeturn1search0  
- Для распределённой защиты (несколько реплик сервиса) допускается использовать распределённые locks в Redis (`SET ... NX EX`), но важно честно зафиксировать trade‑off: документация Redis прямо говорит, что “простой” lock‑паттерн есть, но он *discouraged* по сравнению с Redlock (лучшие гарантии и fault tolerance). То есть шаблон должен описывать **когда простая блокировка приемлема**, а когда нужно более строго. citeturn3search0turn9view2

Local vs distributed cache:
- Local in‑memory cache даёт минимальную latency, но создаёт проблему “частных кэшей”: разные инстансы имеют разные копии и они быстро становятся неконсистентными; это explicitly отмечено в Azure cache‑aside guidance. citeturn1search1  
- Для Redis возможна “near cache” / client‑side caching (server assisted) через `CLIENT TRACKING`, включая opt‑out режим. Это снижает network round trips и нагрузку на Redis, но добавляет сложность: invalidation сообщения, режимы tracking, префиксы и т.д. citeturn13view2turn3search9

Redis‑специфичные boring defaults (если выбран Redis как distributed cache):
- Клиент обязан использовать **connection pooling** — Redis docs подчёркивают, что постоянное создание/пересоздание соединений создаёт ненужную нагрузку, а pooling заметно влияет на производительность. citeturn13view1  
- Для высокой пропускной способности используйте **pipelining** (batch Redis commands без ожидания ответа на каждый), чтобы уменьшить RTT‑стоимость; Redis docs и tutorial по anti‑patterns дают оценку, что выигрыш может быть кратным (особенно при ненулевой сетевой задержке). citeturn0search3turn13view0turn13view1  
- Anti‑pattern: “cache keys без TTL” и “сериализация множества операций вместо pipelining” — официально перечислены как ошибки, ведущие к росту памяти/эвикшенам и лишним накладным расходам. citeturn13view0

### I/O: HTTP, большие payload, object storage offloading

HTTP‑клиент:
- `http.Client` должен **переиспользоваться**, потому что `Transport` держит внутреннее состояние и кэш TCP‑соединений; стандартная документация прямо говорит “Clients should be reused” и что они безопасны для concurrency. citeturn9view1turn1search2  
- При `Client.Do` вызывающий обязан закрывать `Response.Body` и (для connection reuse) читать body до EOF. Иначе `Transport` может не переиспользовать keep‑alive соединение, что увеличит churn и снизит throughput. citeturn20view2  
- По умолчанию `Client.Timeout == 0` означает “нет таймаута”; в шаблоне должен быть дефолтный timeout и правила пер‑операционного deadline через context. citeturn20view1turn20view2

HTTP‑сервер:
- В `http.Server` должны быть настроены таймауты: `ReadHeaderTimeout`, `IdleTimeout`, `WriteTimeout` и (опционально) `ReadTimeout`. Go docs объясняют семантику и отдельно подчёркивают, что `ReadTimeout` не даёт handler’ам принимать решения per‑request и что многие предпочитают `ReadHeaderTimeout`. citeturn5view1

Ограничение входящих payload:
- Используйте `http.MaxBytesReader` для лимитирования размера входящих request bodies: это механизм в stdlib, предназначенный для предотвращения waste ресурсов и (по возможности) закрывающий соединение после превышения лимита. citeturn20view0  
- В контексте multipart/form‑data: официальная advisory‑информация указывает, что некоторые пути обработки form data не лимитируют потребление диска временными файлами, и что вызывающие могут ограничивать размер form data через `http.MaxBytesReader`. citeturn10search12

Большие бинарные объекты и object storage:
- Шаблон должен продвигать паттерн “blob в object storage, ссылка/метаданные в БД”, особенно если payload плохо подходит для SQL‑запросов. Как минимум, entity["company","Amazon Web Services","cloud provider"] рекомендует хранить “слишком большие” атрибуты как объекты в S3 и хранить идентификатор объекта в записи БД; при этом подчёркивает, что транзакции “между S3 и БД” не поддерживаются и приложение должно обрабатывать сбои и чистить orphaned objects. Это удобная, практическая формулировка trade‑off, которую можно перенести и на RDBMS‑сервисы. citeturn16view1  
- Для высоконагруженного доступа к объектам в S3 есть отдельные performance design patterns: кэширование горячего набора, retries/backoff для 503 Slow Down, горизонтальное масштабирование параллельных запросов и явная рекомендация использовать пул HTTP‑соединений и keep‑alive (не создавать соединение на каждый запрос). citeturn16view0

## Decision matrix / trade-offs

Ниже — “decision framework”, который LLM должна применять при проектировании DAL/cache/I/O. Он намеренно ориентирован на измеримость (планы запросов, метрики пула, hit ratio), а не на “красивую архитектуру”.

| Домен решения | Boring default | Когда менять | Основные trade-offs |
|---|---|---|---|
| DB соединения | Явно задать `SetMaxOpenConns/SetMaxIdleConns/SetConnMaxIdleTime/SetConnMaxLifetime`; все запросы — с context deadline | Serverless/всплески соединений ⇒ внешний пулер/прокси (PgBouncer/RDS Proxy) | Unlimited connections по умолчанию опасны; лимит может вести к ожиданиям/“lock‑like” поведению и даже к deadlock‑сценариям при неправильном использовании транзакций/горутин citeturn9view0turn5view0turn12search36turn21view0 |
| Prepared statements | Параметризация всегда; explicit prepare — только при повторяющихся/сложных запросах и измеряемом выигрыше | PgBouncer transaction pooling / частая смена соединений ⇒ server‑side PREPARE может потерять смысл или быть ограничен | В PostgreSQL PREPARE — per session; есть выбор generic/custom plan (может неожиданно влиять на перф) citeturn15view2turn21view0turn2search2 |
| Bulk writes | Postgres: `COPY` для массовых загрузок; MySQL: multi‑row INSERT или `LOAD DATA` | Если нужны строгие инварианты на каждую строку/триггеры/сложная логика, может потребоваться другой путь | COPY быстрее, но есть тонкости (частично вставленные строки при ошибке, необходимость VACUUM для reclaim) citeturn15view0turn15view1turn14view1turn14view3 |
| Индексы/планы | Любой “важный” запрос должен иметь целевой индекс или осознанный seq scan; проверка через EXPLAIN | При write‑heavy нагрузке индексы могут сильнее вредить; нужен workload‑driven набор | Indexes могут ухудшить производительность при неправильном применении; EXPLAIN — основной инструмент анализа citeturn15view4turn15view3 |
| Pagination | Limit/Offset допустимы только для “мелкой глубины”; для глубокой прокрутки — cursor/keyset | Если нужен “jump to page N” и глубина мала — offset проще | PostgreSQL прямо предупреждает: большой OFFSET может быть неэффективен, skipped rows всё равно вычисляются citeturn11view0 |
| Кэширование | Cache‑aside + TTL; singleflight для горячих ключей; TTL randomness для уменьшения синхронных истечений | Сильные требования к консистентности ⇒ минимизировать кэш или применять event‑driven invalidation | Cache‑aside не гарантирует консистентность; thundering herd усиливается TTL; mitigation через prewarm/jitter/rand TTL citeturn17search15turn18view0turn1search0turn1search1 |
| Redis взаимодействие | Клиент с connection pool и pipelining; TTL на ключи | Сверх‑горячие ключи/cluster bottlenecks ⇒ redesign keyspace/sharding | Redis официально выделяет anti‑patterns: serial operations вместо pipelining, отсутствие TTL, hot keys citeturn13view1turn13view0 |
| HTTP I/O | Один `http.Client` на сервис, timeouts включены; `resp.Body` читать+закрывать; входящие тела ограничивать `MaxBytesReader` | Особые потоки/стриминг ⇒ внимательно к server WriteTimeout и per‑request deadlines | Без закрытия body теряется connection reuse; Timeout=0 означает “no timeout”; MaxBytesReader предназначен против resource exhaustion citeturn20view2turn20view1turn20view0turn5view1 |
| Object storage | Большие бинарные payload хранятся в S3/аналогах, в БД — ссылка и метаданные | Если требуется транзакционность “вместе с записью в БД” — придётся строить компенсирующие механизмы | Нет кросс‑транзакций между S3 и БД; нужны cleanup orphan objects; есть отдельные performance patterns и retry/backoff citeturn16view1turn16view0 |

## Набор правил MUST / SHOULD / NEVER для LLM

Ниже — текст, который можно почти напрямую переносить в LLM‑instruction docs (например, `docs/llm/performance-db-cache-io.md`). Формулировки сознательно “нормативные”.

**MUST**
- MUST: Инициализировать DB pool один раз на процесс и трактовать `*sql.DB` как пул. Запросы/команды должны выполняться через эту общую сущность, а не через “открытие соединения на каждый запрос”. citeturn8view0turn9view0  
- MUST: Явно задавать `SetMaxOpenConns` (>0) и документировать “connection budget” на уровне сервиса (пер‑инстанс * число инстансов ≤ доступные соединения БД). citeturn5view0turn3search6turn12search2  
- MUST: Оборачивать каждый DB/cache/HTTP вызов в `context` с deadline и корректно пробрасывать cancellation вниз. citeturn9view0turn20view1  
- MUST: Закрывать ресурсы: `rows.Close()`, `stmt.Close()`, `resp.Body.Close()`; для HTTP ещё и читать body до EOF для connection reuse. citeturn8view0turn7view1turn20view2  
- MUST: Использовать параметризацию SQL (никакой конкатенации пользовательских значений). citeturn2search2  
- MUST: Для Redis использовать client‑side pooling и (при батч‑паттернах) pipelining. citeturn13view1turn13view0  
- MUST: Ограничивать размер входящих HTTP тел `http.MaxBytesReader` на endpoints, принимающих payload (особенно upload/JSON), и иметь явные лимиты в конфиге. citeturn20view0turn10search12  
- MUST: Для PostgreSQL избегать глубокой пагинации через большой OFFSET в “горячих” путях; использовать keyset/cursor pagination. citeturn11view0

**SHOULD**
- SHOULD: Настраивать `SetConnMaxLifetime` и `SetConnMaxIdleTime` как “hygiene settings”, особенно при балансировке/проксировании на уровне БД. citeturn9view0turn5view0  
- SHOULD: Использовать batching: Postgres — `COPY` для bulk ingest; MySQL — multi‑row INSERT или `LOAD DATA` (если модель позволяет). citeturn15view0turn14view1turn14view3  
- SHOULD: Вводить кэширование через cache‑aside для горячих read‑path’ов и явно описывать TTL/инвалидацию; использовать `singleflight` или аналог request collapsing для горячих ключей (anti‑stampede). citeturn17search15turn1search0turn18view0  
- SHOULD: Добавлять randomness к TTL для уменьшения синхронных истечений ключей при большом масштабе. citeturn18view0  
- SHOULD: Включать правило “любой новый важный запрос сопровождается проверкой/обоснованием через EXPLAIN и (при необходимости) индексом”, понимая, что лишний индекс может вредить. citeturn15view3turn15view4  
- SHOULD: Для больших бинарных данных использовать object storage и хранить ссылку в БД, документируя отсутствие кросс‑транзакций и необходимость cleanup orphan objects. citeturn16view1turn16view0  
- SHOULD: Делать `http.Client` singleton’ом (или зависимостью уровня приложения) и задавать timeouts. citeturn9view1turn20view1  
- SHOULD: В `http.Server` настраивать `ReadHeaderTimeout/IdleTimeout/WriteTimeout` как baseline защиты и предсказуемости. citeturn5view1

**NEVER**
- NEVER: Не оставлять `SetMaxOpenConns` по умолчанию (unlimited) и не ставить `SetMaxOpenConns(0)` “чтобы выключить” — это означает “без лимита”. citeturn5view0turn9view0  
- NEVER: Не создавать новый `http.Client` на каждый запрос и не игнорировать закрытие `Response.Body`. citeturn9view1turn20view2  
- NEVER: Не кэшировать ключи “навсегда” без TTL в Redis; это официальный anti‑pattern. citeturn13view0turn18view0  
- NEVER: Не реализовывать распределённую блокировку в Redis как “абсолютно надёжную” без описания ограничений; Redis docs прямо помечают простой паттерн как discouraged в пользу Redlock для лучших гарантий. citeturn3search0turn9view2  
- NEVER: Не использовать большой OFFSET для глубокой пагинации в hot path. citeturn11view0

## Concrete good / bad examples

### Good: инициализация DB pool с явными лимитами и “гигиеной”

Следующий стиль соответствует тому, что Go docs называют “database handle (pool)” и тому, как `database/sql` ожидает управлять соединениями. citeturn8view0turn9view0turn5view0

```go
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type DBConfig struct {
	DSN             string
	MaxOpenConns    int           // MUST be > 0
	MaxIdleConns    int           // usually <= MaxOpenConns
	ConnMaxIdleTime time.Duration // e.g. 10m
	ConnMaxLifetime time.Duration // e.g. 1h
	PingTimeout     time.Duration // e.g. 2s
}

func OpenDB(ctx context.Context, driverName string, cfg DBConfig) (*sql.DB, error) {
	if cfg.MaxOpenConns <= 0 {
		return nil, fmt.Errorf("MaxOpenConns must be > 0 (0 means unlimited in database/sql)")
	}

	db, err := sql.Open(driverName, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	pctx, cancel := context.WithTimeout(ctx, cfg.PingTimeout)
	defer cancel()
	if err := db.PingContext(pctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("db.PingContext: %w", err)
	}

	return db, nil
}
```

### Bad: неограниченные соединения и “на каждый запрос новый handle”

Это приводит к неконтролируемому росту числа соединений (по умолчанию unlimited) и легко выбивает `max_connections` в PostgreSQL, где каждое соединение — отдельный backend process. citeturn5view0turn3search6turn12search0

```go
// ❌ Плохо: sql.Open в hot path, нет SetMaxOpenConns, нет timeouts.
func Handle(w http.ResponseWriter, r *http.Request) {
	db, _ := sql.Open("postgres", os.Getenv("DSN"))
	rows, _ := db.Query("SELECT * FROM users") // no ctx, no params, SELECT *
	// rows.Close() забыли, db.Close() тоже
	_ = rows
}
```

### Good: cache-aside + singleflight (защита от stampede внутри инстанса)

`singleflight.Group` предоставляет duplicate suppression; AWS описывает thundering herd, а Go‑пакет `singleflight` — стандартный building block request collapsing. citeturn1search0turn18view0

```go
package readmodel

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"golang.org/x/sync/singleflight"
)

type Cache interface {
	Get(ctx context.Context, key string) (val []byte, ok bool, err error)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Repo struct {
	db    *sql.DB
	cache Cache
	sf    singleflight.Group
}

func ttlWithJitter(base time.Duration) time.Duration {
	// Simple TTL jitter to reduce synchronized expirations.
	// Keep jitter small relative to base.
	j := time.Duration(rand.Int63n(int64(base / 20))) // up to 5%
	return base + j
}

func (r *Repo) GetUser(ctx context.Context, id int64) (User, error) {
	key := fmt.Sprintf("user:%d", id)

	if b, ok, err := r.cache.Get(ctx, key); err == nil && ok {
		var u User
		if err := json.Unmarshal(b, &u); err == nil {
			return u, nil
		}
		// If cache corrupted, fall through to reload.
	}

	v, err, _ := r.sf.Do(key, func() (any, error) {
		// Double-check after we become the "leader"
		if b, ok, err := r.cache.Get(ctx, key); err == nil && ok {
			var u User
			if err := json.Unmarshal(b, &u); err == nil {
				return u, nil
			}
		}

		var u User
		q := `SELECT id, name FROM users WHERE id = $1`
		if err := r.db.QueryRowContext(ctx, q, id).Scan(&u.ID, &u.Name); err != nil {
			return User{}, err
		}

		b, _ := json.Marshal(u)
		_ = r.cache.Set(ctx, key, b, ttlWithJitter(10*time.Minute))
		return u, nil
	})

	if err != nil {
		return User{}, err
	}
	return v.(User), nil
}
```

### Bad: кэш без TTL и без защиты от stampede

Redis официально называет “кэш ключей без TTL” anti‑pattern, а AWS отдельно описывает thundering herd и предлагает jitter/рандомизацию TTL. citeturn13view0turn18view0

```go
// ❌ Плохо: нет TTL, нет singleflight/locks, при истечении/инвалидации возможен stampede.
cache.Set(ctx, key, value, 0) // "навсегда"
```

### Good: HTTP client reuse + закрытие/дочитывание body

Стандартная документация `net/http` требует закрывать `Response.Body`, и предупреждает: если body не прочитан до EOF и не закрыт, Transport может не переиспользовать keep‑alive соединение. citeturn9view1turn20view2

```go
var httpClient = &http.Client{
	Timeout: 2 * time.Second, // 0 means no timeout
}

func FetchJSON(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Optional: drain body if you might not fully decode it,
	// to maximize connection reuse.
	dec := json.NewDecoder(resp.Body)
	return dec.Decode(dst)
}
```

### Good: лимит request body через MaxBytesReader

`http.MaxBytesReader` предназначен для лимитирования входящих request bodies и предотвращения waste ресурсов; он также, если возможно, сигнализирует серверу закрыть connection после превышения лимита. citeturn20view0turn10search12

```go
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	const maxSize = 10 << 20 // 10 MiB
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)
	defer r.Body.Close()

	// дальше — streaming/decoder без ReadAll
	// ...
}
```

### Good: pagination keyset вместо большого OFFSET

PostgreSQL прямо предупреждает, что большой OFFSET неэффективен (skipped rows всё равно вычисляются). citeturn11view0

```go
// Keyset pagination: стабильный порядок + "seek method".
const q = `
SELECT id, created_at, payload
FROM events
WHERE (created_at, id) < ($1, $2)
ORDER BY created_at DESC, id DESC
LIMIT $3;
`
```

## Anti-patterns и типичные ошибки/hallucinations LLM

Ниже — список вещей, которые чаще всего “убивают” latency/throughput в DAL/cache/I/O. Формулируйте их как lint‑правила для LLM и как пункты code review.

**DB**
- Unbounded connections: LLM часто “забывает” `SetMaxOpenConns` или ставит `0`, думая что это “запретить соединения”. В `database/sql` это означает “no limit”, что быстро выбивает лимиты PostgreSQL/MySQL. citeturn5view0turn3search6  
- Connection leaks: забытый `rows.Close()`/непрочитанные rows держат ресурсы (Go docs подчёркивают необходимость освобождать ресурсы `sql.Rows`), что уменьшает доступность пула и увеличивает ожидания. citeturn8view0turn7view1  
- “Думать, что prepared statements автоматически ускоряют всё”: в PostgreSQL выигрыш максимален при большом числе похожих запросов в одной сессии; при простых запросах или при доминировании execution cost выигрыш мало заметен. Более того, generic vs custom plans могут вести к неожиданным эффектам — это нужно измерять через EXPLAIN. citeturn15view2turn15view3  
- Чаттинг (chatty) к БД: N+1 запрос в цикле, отсутствие batching, отсутствие multi‑row insert/COPY/LOAD DATA там, где это естественно. Это напрямую усиливает tail latency при fan‑out (см. “The Tail at Scale”). citeturn17search0turn14view1turn15view0  
- “Индексы как серебряная пуля”: LLM может предлагать индексы “на всё”. Документация PostgreSQL прямо говорит, что неуместные индексы могут замедлять систему. citeturn15view4  

**Cache**
- Кэш без TTL: Redis официально называет это anti‑pattern (unbounded memory growth/eviction pressure). citeturn13view0  
- TTL “везде одинаковый” без jitter на большом масштабе: приводит к синхронным истечениям и шипам нагрузки; AWS явно предлагает добавлять randomness к TTL. citeturn18view0  
- Нет защиты от stampede: при cache miss все реплики идут в БД одинаково; AWS описывает thundering herd и варианты mitigation (prewarm, jitter). Для in‑process — `singleflight`. citeturn18view0turn1search0  
- “Локальный кэш как будто он разделяемый”: LLM иногда проектирует local in‑memory cache как источник консистентных данных “для всего кластера”. Azure guidance предупреждает, что private/local caches быстро становятся неконсистентными между инстансами. citeturn1search1  
- Неправильная уверенность в distributed locks: Redis docs помечают простой `SET ... NX EX` lock‑паттерн как discouraged в пользу Redlock; это должно быть отражено в документах как trade‑off, а не как “надёжная блокировка”. citeturn3search0turn9view2

**I/O**
- Новый `http.Client` на каждый запрос или отсутствие timeout: документация `net/http` говорит, что Client должен переиспользоваться, и что `Timeout==0` означает “нет таймаута”. citeturn9view1turn20view1  
- Не закрывать/не дочитывать `resp.Body`: это ломает connection reuse и увеличивает connection churn. citeturn20view2  
- Читать большие тела целиком в память (`ReadAll`) вместо streaming + лимитов; также отсутствие `MaxBytesReader` на upload endpoints. `MaxBytesReader` — стандартный механизм против resource exhaustion. citeturn20view0turn10search12  
- Большие бинарные payload хранить “как есть” в БД без осознанной стратегии: часто это приводит к росту I/O и усложняет масштабирование; при этом даже в AWS guidance для DynamoDB отдельно рекомендуют переносить большие объекты в S3 и хранить ссылку, подчёркивая trade‑off отсутствия cross‑transactions. citeturn16view1turn16view0

## Review checklist для PR / code review

Этот список лучше хранить как `docs/review/performance-db-cache-io-checklist.md` и использовать как обязательные вопросы ревью.

- Проверка лимитов соединений:
  - Есть ли явные `DB_MAX_OPEN_CONNS`/`DB_MAX_IDLE_CONNS` и соответствующие вызовы `SetMaxOpenConns/SetMaxIdleConns`? citeturn9view0turn5view0  
  - Посчитан ли “connection budget” с учётом `max_connections` и особенностей платформы (например, лимиты Cloud Run/Cloud SQL, reserved connections у managed‑провайдера)? citeturn3search6turn12search2turn3search30  
  - Настроены ли `ConnMaxIdleTime/Lifetime` и объяснены ли значения? citeturn9view0turn5view0

- Корректность освобождения ресурсов:
  - Для каждого `QueryContext` есть гарантированный `rows.Close()` (defer) и обработка `rows.Err()`? citeturn8view0turn6view1  
  - Для `Stmt` есть `Close()` и нет “вечных” prepared statements без необходимости? citeturn7view1  
  - Для внешних HTTP вызовов `resp.Body` читается/закрывается, есть timeout/deadline? citeturn20view2turn20view1

- Семантика запросов и безопасность:
  - Все SQL параметры переданы как параметры (никакой конкатенации)? citeturn2search2  
  - Для PostgreSQL: избегается большой OFFSET в hot path; при необходимости используется keyset/cursor pagination. citeturn11view0  
  - Для “важных” запросов приложен план/обоснование через EXPLAIN и/или индекс‑изменение, понимая цену лишних индексов. citeturn15view3turn15view4

- Batching / bulk:
  - При массовых вставках выбран подходящий механизм (Postgres COPY, MySQL multi‑row INSERT/LOAD DATA) или есть обоснование, почему нельзя. citeturn15view0turn14view1turn14view3  

- Cache:
  - Кэширование оформлено как cache‑aside с понятной инвалидацией/TTL? citeturn17search15turn18view0  
  - Есть ли защита от thundering herd для hot keys (singleflight, prewarm, TTL jitter)? citeturn18view0turn1search0  
  - Для Redis: используются pooling и pipelining там, где есть батчи; нет ключей без TTL (официальный anti‑pattern). citeturn13view1turn13view0  

- Большие payload / object storage:
  - Есть ли лимит на входящие тела (`MaxBytesReader`) и streaming‑подход? citeturn20view0turn10search12  
  - Если данные “большие и бинарные” — рассмотрен ли object storage offload (S3/аналог) и учтён ли trade‑off отсутствия кросс‑транзакций и cleanup orphan objects? citeturn16view1turn16view0

## Что из результата нужно оформить отдельными файлами в template repo

Ниже — рекомендуемая “нарезка” на файлы, чтобы это стало нормативным артефактом репозитория (engineering standard + LLM instructions). Названия можно адаптировать под ваш стиль.

- `docs/engineering/performance-db-cache-io.md`  
  Норматив “для людей”: connection budget, pool sizing принципы, prepared statements trade‑offs, batching/bulk, кэш‑стратегии, pagination, large payload/object storage.

- `docs/llm/performance-db-cache-io.md`  
  Норматив “для LLM”: секции MUST/SHOULD/NEVER, запреты на опасные дефолты (`MaxOpenConns=0`, cache без TTL, OFFSET deep pagination, HTTP client per request), обязательные patterns (singleflight, MaxBytesReader, resp.Body close). citeturn5view0turn13view0turn11view0turn20view2turn20view0

- `docs/review/performance-db-cache-io-checklist.md`  
  PR checklist (раздел из этого отчёта), чтобы ревью было воспроизводимым и быстрым.

- `docs/runbooks/connection-budget.md`  
  Практическая памятка: как считать бюджет соединений (PostgreSQL `max_connections`, вычеты reserved у managed‑провайдера, лимиты Cloud Run/Cloud SQL), как выбирать per‑instance `DB_MAX_OPEN_CONNS`. citeturn3search6turn3search30turn12search2turn12search0

- `internal/config/db.go` + `internal/config/cache.go` + `internal/config/http.go`  
  Единый конфиг‑слой с env‑параметрами для pool/timeouts/limits, чтобы LLM не “придумывала” значения в коде.

- `internal/storage/README.md`  
  Контракт DAL: “все методы принимают context”, “никаких глобальных транзакций”, “батчи/пагинация по правилам”, “EXPLAIN обязателен для критичных запросов”.

- `internal/cache/README.md`  
  Контракт кэша: cache‑aside, TTL required, stampede protection, invalidation semantics, запрет на keys без TTL (кроме явно документированных исключений). citeturn17search15turn18view0turn13view0