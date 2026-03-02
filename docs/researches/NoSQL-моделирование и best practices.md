# NoSQL‑моделирование и best practices для микросервисов на Go

## Scope

Этот стандарт применяйте, когда микросервис **владеет своими данными** (database-per-service) и границы домена/контекста достаточно ясны: тогда модель данных можно оптимизировать под ограниченный набор сценариев чтения/записи внутри сервиса, не пытаясь «универсально» закрыть все запросы организации. Такой подход прямо рекомендуется в руководствах по data persistence для микросервисов: отдельное хранилище на сервис упрощает эволюцию схемы и независимое масштабирование, но усложняет кросс-сервисные транзакции и запросы. citeturn10search6turn10search26turn10search8

Этот стандарт **особенно уместен** для NoSQL, потому что во многих NoSQL-системах модель строится «от запросов/доступа» (access-pattern-first): данные, которые читаются вместе, разумно хранить вместе (документные БД), или проектировать таблицы/ключи так, чтобы обслужить запросы без джойнов и без дорогих сканов (key-value/wide-column). Это сформулировано как базовый принцип в руководствах по моделированию для документных и key-value систем. citeturn12search27turn11search7turn0search2turn9search1

Не применяйте (или применяйте с сильными оговорками), если:
- доменная модель **по природе реляционная** и требует частых *ad-hoc* связок/джойнов/сложных агрегатов, а набор запросов быстро меняется (NoSQL часто компенсирует это денормализацией, материализацией представлений и отдельными индексами → растёт сложность и риск несоответствий); citeturn11search7turn11search16turn0search2  
- нужны **распределённые транзакции** между сервисами с ACID‑гарантиями: даже при database-per-service это решают паттернами согласованности (Saga/compensations), а не «магическими» кросс-сервисными транзакциями; citeturn10search8turn10search27turn10search6  
- проект на ранней стадии, и команда ожидает частую переработку ключевых access patterns: даже в «гибкой» документной модели поздние изменения на проде могут быть труднообратимы/дороги, поэтому начинать стоит с простого дизайна и эволюционировать осознанно. citeturn12search3turn12search27  

## Recommended defaults для greenfield template

Базовая политика template: **SQL по умолчанию, NoSQL — осознанный выбор по чётким критериям**. Обоснование прагматичное: многие NoSQL движки не дают джойны и поощряют денормализацию/предварительную материализацию, что требует заранее понятых запросов и дисциплины сопровождения. citeturn11search7turn11search16turn0search2

Если NoSQL всё-таки выбирается, «боевые» дефолты для greenfield‑шаблона должны фиксировать не конкретный продукт, а **инварианты проектирования**, которые LLM не должен угадывать:
- **Access-pattern-first**: сервис обязан иметь артефакт (doc/ADR), где перечислены все чтения/записи с параметрами (ключи, фильтры, сортировки, лимиты, ожидаемая кардинальность и latency/SLO). Это соответствует рекомендациям по моделированию, где схема/таблица проектируется под то, как приложение обращается к данным. citeturn12search27turn11search7turn9search1turn0search2  
- **No unbounded scans / table walks by default**: любые операции наподобие full scan должны быть явным исключением, объяснённым в ADR (например, оффлайн бэкфилл/экспорт). Это прямо отмечено в best practices: Scan по мере роста таблицы замедляется и может «съесть» throughput; в Redis команда `KEYS` блокирующая/опасная для прод‑нагрузки. citeturn11search2turn2search4turn2search16  
- **TTL только как «storage hygiene», не как механизм точной бизнес‑логики**: TTL‑удаление в некоторых системах best-effort и может задерживаться; требуемая «минутная точность» — ответственность приложения, а не TTL. Для DynamoDB это явно зафиксировано в API/гайдах: удаление best‑effort, типично в пределах ~двух дней, а «просроченные» элементы могут ещё попадать в чтения/сканы до фактического удаления. citeturn2search9turn2search0  
- **Schema evolution по умолчанию — через версионирование и совместимость**: документные БД позволяют полиморфизм, но template должен требовать версии схемы/документа и стратегию миграции/совместимости. Для MongoDB есть отдельный паттерн schema versioning; также есть встроенная schema validation через `$jsonSchema`. citeturn7search14turn12search7turn7search1turn12search27  

Дефолтные ограничения (фиксируются в стандарте, чтобы LLM не «забывал»):
- DynamoDB: максимальный размер item — 400 KB (включая имена атрибутов), TTL best-effort/не мгновенный. citeturn8search6turn2search9  
- MongoDB: максимальный BSON‑документ — 16 MiB; избегать неограниченно растущих массивов через subset/bucket/outlier паттерны. citeturn7search4turn7search3turn7search2turn7search30  
- Cassandra/Keyspaces‑класс wide-column: «таблица под запрос», избегать `ALLOW FILTERING` и проектировать партиции так, чтобы не было «горячих»/сверхкрупных партиций. citeturn0search2turn2search11turn6search5turn6search1  
- Redis‑класс key-value: сканирование ключевого пространства — через `SCAN`, а не `KEYS`; кластерные multi-key операции требуют keys-in-same-slot (hash tags). citeturn2search16turn2search4turn5search7turn5search19  

## Decision matrix / trade-offs

Ниже — «матрица выбора» как правило для template: не «какую БД любят», а **какой класс NoSQL оправдан** и что вы обязаны принять как цену.

| Класс | Когда выбирать вместо SQL | Цена/ограничения | Типовые компенсирующие паттерны |
|---|---|---|---|
| Document store | Иерархические сущности, «данные читаются вместе», нужен гибкий документ/полиморфизм | Лимит размера документа, риск unbounded arrays, join‑подобные операции (`$lookup`) могут быть дорогими | Embedding, subset/bucket/outlier, schema validation + versioning |
| Key-value | Кеши/сессии/идемпотентность, быстрый доступ по ключу, простые структуры данных | Нету естественных запросов «по атрибутам» без доп. индексации ключами; durability/replication trade-offs; ключевое пространство нельзя «просматривать» блокирующе | Доп. ключи-индексы, TTL + jitter, SCAN, hash tags в кластере |
| Wide-column | Очень высокие скорости записи, предсказуемые запросы по партиции + диапазону, масштабирование по партициям | Таблицы проектируются под запросы; слабые ad-hoc запросы; вторичные индексы ограниченно полезны; опасны `ALLOW FILTERING` и большие партиции | Table-per-query, денормализация, bucketing, materialized views/индексы только по правилам |
| Time-series | Данные «время + метрики/измерения», нужны range queries по времени и downsampling/retention | Высокая кардинальность (теги/лейблы) быстро рушит производительность/стоимость; необходимость ретеншна и rollups | Правильное разделение tags/fields, ограничения на labels, retention/downsampling, pre-aggregation |

Ключевой trade-off: NoSQL часто выигрывает в предсказуемости и масштабировании **при заранее спроектированных access patterns**, но проигрывает в «свободе запросов». Это напрямую видно в рекомендациях по DynamoDB (нет JOIN → денормализация; агрегаты `SUM/COUNT` не нативны → materialized aggregation) и в best practices wide-column (денормализация/таблицы под запрос, избегать запросов, которые заставляют фильтровать без ключа). citeturn11search7turn11search16turn0search2turn2search11turn11search2

Отдельно фиксируйте trade-offs согласованности/транзакций:
- DynamoDB поддерживает strong read consistency и ACID transactions, но strong reads не поддерживаются на GSI, а TTL — best-effort. citeturn3search0turn2search9turn11search7turn2search2  
- Cassandra‑класс — «настраиваемая согласованность» (не одна «магическая»), а “lightweight transactions” реализуются Paxos/CAS и по смыслу не равны «реляционным транзакциям на всё». citeturn6search19turn8search3  
- MongoDB даёт readConcern/writeConcern для контроля гарантий чтения/записи, и поддерживает multi-document transactions, но в проде есть отдельные production considerations (доступность/лимиты/совместимость/влияние на cache/oplog). citeturn3search4turn3search0turn3search1turn3search2  

## Практические best practices по типам NoSQL

Ниже — «decision framework» и требования к моделированию **по классам**. В каждой подсекции структура одинаковая: access patterns → ключи/партиции → индексы → согласованность/транзакции → TTL → эволюция схемы → агрегаты → операционные caveats.

**Document stores (пример: MongoDB)**  
Основа моделирования: «данные, которые читаются вместе, должны храниться вместе», т.е. embedding обычно предпочтительнее «склеивания на чтении». Это сформулировано как core principle для MongoDB и повторяется в официальных материалах по data modeling. citeturn12search27turn0search0  

Access-pattern-first: документная модель хороша, когда ваш read-path действительно «схлопывается» до одного документа или ограниченного числа документов по индексу. Если вынужденно делаете join‑подобные вещи (`$lookup`) — фиксируйте это как сознательный компромисс: Atlas schema advisor прямо отмечает, что `$lookup` может быть непроизводительным, но иногда это оправдано ради контроля размера/структуры. citeturn7search7turn11search6  

Ограничения и «боль»: максимальный BSON документ — 16 MiB; неограниченно растущие массивы приводят к непредсказуемому росту документа и деградации чтений/индексов. Официальные anti-patterns рекомендуют избегать unbounded arrays и применять subset pattern, references или bucket pattern (для серийных/таймсерийных данных). citeturn7search4turn7search3turn7search2  

Согласованность/транзакции: используйте атомарность документа как «первый инструмент»; multi-document transactions включайте только когда нужна атомарность на нескольких документах/коллекциях и нельзя разумно денормализовать/перестроить документ. MongoDB поддерживает транзакции, но официально выносит отдельные production considerations (availability/limits/oplog/cache/locks и пр.), поэтому template должен требовать ADR, объясняющий почему транзакции нужны и как ограничиваются (короткие по времени, маленький write set, явные timeouts). citeturn3search2turn3search1turn1search9turn1search5  

Schema evolution: не подменяйте «гибкую схему» отсутствием дисциплины. В MongoDB есть schema validation (в т.ч. через `$jsonSchema`) и официальные рекомендации/паттерн schema versioning (несколько форм документов в одной коллекции + миграция на чтении/записи). Это особенно важно для LLM‑генерации: модель не должна «расширять документ» произвольно без версии и валидатора. citeturn12search7turn7search1turn7search14  

Агрегации: агрегатный pipeline может использовать индексы на входной коллекции, что нужно учитывать при дизайне `$match/$sort/$group`. В production template правило простое: любой pipeline должен иметь явный индекс‑план (или объяснение почему индекс невозможен) и нагрузочный тест как часть acceptance criteria. citeturn11search3turn11search10  

**Key-value (пример: Redis и DynamoDB как key-value/document)**  
Ключевой принцип: если хранилище «по ключу», то модель данных — это **дизайн ключей**, а не «структура таблиц». Для DynamoDB первичный ключ задаёт физическую раскладку: partition key проходит через внутренний hash и определяет физическую партицию хранения; sort key группирует элементы внутри partition key и позволяет диапазонные запросы. citeturn9search3turn9search1turn0search1  

Для DynamoDB:  
- Денормализация — не «грех», а рекомендуемый путь, потому что JOIN не поддерживается; это прямо написано в введении: «нет JOIN → рекомендуем denormalize чтобы снизить round-trips». citeturn11search7  
- Scan должен быть исключением: best practices рекомендуют проектировать таблицы/индексы так, чтобы использовать Query вместо Scan, иначе Scan с ростом таблицы замедляется и может «съесть» throughput. citeturn11search2turn11search13  
- Агрегации `SUM/COUNT` не нативны: есть официальный паттерн materialized aggregation (предвычислять и хранить как обычные items, часто через GSI). citeturn11search16  
- Ограничения: item ≤ 400 KB; TTL best-effort, типично удаляет в пределах ~двух дней, а просроченные items могут появляться в чтениях/запросах до удаления. citeturn8search6turn2search9  
- Индексы: проектируйте GSIs/LSIs осознанно (они копируют/проецируют данные и меняют стоимость записи); также сильная согласованность не поддерживается на GSI. citeturn0search8turn3search0turn12search25  
- Concurrency control: используйте условные записи и optimistic locking; в официальном гайде AWS это описано как стратегия с version attribute и conditional writes. Для global tables отдельно отмечено: reconciliation “last writer wins” ломает ожидания «версионирования» между регионами, значит конфликт-резолвинг должен быть на уровне приложения. citeturn12search2turn12search6  
- Эволюция ключей: вы не можете обновлять primary key атрибуты через UpdateItem — только delete+put (это важно для дизайна ключа: «ошибка ключа» = миграция данных, а не ALTER). citeturn12search4turn12search12  

Для Redis‑класса key-value (часто cache/session/idempotency store):  
- Никогда не используйте `KEYS` на проде: документация прямо предупреждает, что это «O(N)» и может надолго блокировать сервер; использовать `SCAN`. citeturn2search4turn2search16  
- TTL/expiration: expiry происходит пассивно (при доступе) и активно (периодическая выборка ключей), т.е. TTL не является «точным таймером» как cron-job. citeturn5search1turn5search5  
- Eviction: при `maxmemory` вы обязаны выбрать политику вытеснения и понимать её последствия (какие ключи могут исчезнуть). Документация описывает механизм eviction policy и связь с memory limit. citeturn5search0turn5search16  
- Кластер и multi-key операции: в Redis Cluster многие multi-key команды/транзакции работают только если ключи в одном hash slot; для этого есть hash tags. Документация это фиксирует, и template должен требовать «key naming spec» с hash tag правилами (если multi-key нужен). citeturn5search19turn5search7turn5search3  
- Репликация/консистентность: replication асинхронная, есть окно возможной потери подтверждённых записей при failover; официальные материалы прямо говорят о data loss window и рекомендуют `WAIT`, если нужно дождаться реплик (понимая, что это всё ещё не превращает систему в синхронно‑реплицируемую БД). citeturn8search0turn8search4turn8search32  
- Транзакции: Redis transactions (`MULTI/EXEC/WATCH`) дают изоляцию на уровне выполнения (команды выполняются как одно «последовательное» действие), но это не «SQL‑транзакции с rollback на любую ошибку»; template должен запрещать LLM обещать «полный rollback всех команд» без уточнения семантики Redis. citeturn5search2turn5search10turn5search3  

**Wide-column (пример: Cassandra/Keyspaces‑класс)**  
Фундамент: моделирование = «таблица под запрос» и денормализация. В best practices подчёркивается, что в wide-column «нет JOIN», поэтому данные, нужные для запроса, обычно должны быть в одной таблице; модель строят вокруг ключей и запросов. citeturn0search2turn2search11  

Partition key и hot partitions: качество распределения нагрузки почти полностью определяется partition key. Практически важно избегать «очень больших» и/или «горячих» партиций: официальные/вендорские рекомендации для Cassandra‑совместимых систем включают правила про размер партиции и необходимость дробления (bucketing) по времени/шардам (например, «по дням» или «N‑шардов на tenant»). citeturn6search5turn6search1  

Secondary indexes: используйте крайне осторожно. В официальной документации Cassandra есть отдельное руководство «When to use 2i», подчёркивающее, что 2i подходит не для всех случаев и требует понимания распределения данных и запросов. В template правило: «если индекс — то ADR с обоснованием кардинальности, селективности и нагрузочного профиля». citeturn6search2  

Запросы и `ALLOW FILTERING`: это красная зона. Документация по CQL подчёркивает, что `ALLOW FILTERING` может привести к непредсказуемой производительности, поскольку разрешает выполнение «дорогих» запросов. Для production template: `ALLOW FILTERING` **запрещён**, кроме разовых админских операций вне SLO. citeturn2search11turn6search1  

Consistency model: Cassandra‑класс предоставляет настраиваемые уровни согласованности, и «правильность» часто зависит от того, как вы выбираете read/write consistency относительно replication factor (классическое правило пересечения `W + R > RF` для гарантии пересечения кворумов описано в документации). Template должен требовать явного выбора consistency levels для критичных операций и тестов на read-your-writes/monotonicity, а не «молчаливых ожиданий». citeturn6search19turn6search9  

TTL и tombstones: TTL — это не «бесплатное удаление». Для Cassandra TTL создаёт tombstones; официальные материалы по tombstones и deletion подчёркивают их влияние и необходимость правильной настройки/понимания. В template фиксируйте правило: если используете TTL массово, обязателен план контроля tombstone‑нагрузки (метрики, compaction strategy, запросы без range‑сканов по широким партициям). citeturn1search7turn3search3turn6search5  

Транзакции: «lightweight transactions» реализуются через Paxos и дают линераризуемую CAS‑семантику для конкурентных операций, но это специализированный инструмент, а не «сделаем ACID как в Postgres». Template должен ограничивать применение LWT сценариями CAS/уникальности/защиты от гонок и требовать ADR по стоимости/латентности. citeturn8search3turn8search11  

**Time-series (пример: InfluxDB/Timestream/Prometheus‑класс хранения)**  
Ключ к успешной модели time-series — управление кардинальностью и ретеншном. В InfluxDB серии определяются комбинацией measurement + tag set (+ дополнительные элементы, зависящие от версии/движка), а high cardinality — один из основных источников проблем производительности/стоимости. Официальные документы объясняют устройство series key и дают guidance по schema design (tag vs field). citeturn5search12turn5search18turn5search2  

Prometheus‑класс (часто именно для метрик, но принцип применим шире): официальные best practices и naming guidance предупреждают про высокую кардинальность лейблов и необходимость осторожного выбора labels. В template закрепите: **нельзя** использовать пользовательские идентификаторы (user_id/session_id/request_id) как label/tag в метриках/таймсериях без агрегации/квантования. citeturn5search0turn5search1  

Timestream‑класс (managed time-series): схема строится вокруг measures и dimensions; в документации есть разделы best practices/schema design и отдельные best practices для partition keys. Template должен требовать: (1) явный выбор dimension‑ов, (2) стратегию партиционирования и (3) ретеншн/агрегации (downsampling), иначе time-series быстро «взрывается» по стоимости. citeturn5search3turn5search6turn5search17  

## Набор правил в формате MUST / SHOULD / NEVER для LLM

Ниже — правила, которые стоит почти дословно перенести в LLM‑instruction doc (для ChatGPT/Codex/Claude Code и т.п.). Формулировки специально «жёсткие», чтобы снижать галлюцинации.

**MUST**
- MUST начинать любое NoSQL‑проектирование с явного списка access patterns (CRUD/reads/aggregations), включая: параметры запроса, ожидаемые объёмы (кардинальность), сортировки/пагинацию, SLO/latency и требования к консистентности. Без этого нельзя корректно спроектировать ключи/индексы; это согласуется с официальными принципами моделирования (хранить вместе то, что читается вместе; проектировать под запросы). citeturn12search27turn9search1turn0search2turn11search7  
- MUST для DynamoDB‑класса: показывать, как каждый access pattern реализуется через `Query`/`GetItem`/индекс, и объяснять, почему `Scan` допустим или недопустим; по best practices Scan на больших таблицах нежелателен. citeturn11search2turn11search13turn9search3  
- MUST для wide-column: проектировать primary key так, чтобы запросы не требовали `ALLOW FILTERING`, и явно ограничивать размер/ширину партиций (bucketing/шардинг). citeturn2search11turn6search5turn6search1  
- MUST для document store: учитывать лимит размера документа и избегать unbounded arrays, предлагая один из официальных паттернов (subset/bucket/outlier/references). citeturn7search4turn7search3turn7search2turn7search30  
- MUST описывать consistency model выбранного хранилища и как приложение с ним живёт (read/write concerns, кворумы, eventual consistency, last-writer-wins). citeturn3search4turn3search0turn6search19turn12search2  
- MUST документировать TTL семантику и не обещать «точное удаление по расписанию», если хранилище делает best-effort (например, DynamoDB). citeturn2search9turn2search0  
- MUST применять безопасные API/валидацию ввода, чтобы предотвратить NoSQL injection: особенно запрещать «сырые JSON-фрагменты» от клиента и клиент-контролируемые операторы (`$where`, `$regex`, `$expr` и т.п.), согласно рекомендациям entity["organization","OWASP","appsec nonprofit"] по NoSQL Security. citeturn0search1turn0search4turn0search2  

**SHOULD**
- SHOULD предлагать boring defaults: денормализовать только то, что нужно для ключевых access patterns; агрегации делать materialized (для DynamoDB‑класса) или поддерживать pre-aggregation/rollups (для time-series). citeturn11search16turn5search6turn5search12  
- SHOULD для Redis‑класса: использовать `SCAN` вместо `KEYS`, задавать `maxmemory`/eviction policy и объяснять последствия (вплоть до потери ключей). citeturn2search16turn2search4turn5search0  
- SHOULD для Redis‑кластера: использовать hash tags только если реально нужны multi-key операции; иначе по умолчанию распределять ключи равномерно. citeturn5search11turn5search7  
- SHOULD включать «миграционный/эволюционный план»: версионирование документов (document stores), новая таблица/новый индекс при эволюции ключа/проекции (DynamoDB‑класс), обратная совместимость при чтении. citeturn7search14turn12search25turn12search4  
- SHOULD ограничивать использование транзакций: MongoDB transactions — только при необходимости атомарности на нескольких документах и с учётом production considerations; Cassandra LWT — только CAS/уникальность; Redis transactions — в рамках их гарантий и ограничений. citeturn3search1turn3search2turn8search3turn5search2  

**NEVER**
- NEVER генерировать «SQL‑мышление» поверх NoSQL: JOIN/foreign keys как обязательную норму в DynamoDB‑классе (указано: JOIN не поддерживается) или в wide-column (денормализация и таблицы под запрос). citeturn11search7turn0search2  
- NEVER предлагать `Scan` как «нормальный способ фильтрации» больших таблиц в DynamoDB‑классе без явного ADR и нагрузочного обоснования. citeturn11search2turn11search13  
- NEVER использовать `KEYS` в Redis для прод‑кода. citeturn2search4  
- NEVER обещать «TTL удаляет ровно в момент истечения» (особенно для DynamoDB), и NEVER полагаться на TTL для критической корректности. citeturn2search9turn5search1  
- NEVER расширять схему «тихо»: новые поля без версии/валидатора/контракта, потому что это ломает совместимость и эксплуатацию; MongoDB даёт механизмы schema validation, и их нужно использовать в production‑контексте, если данные критичны. citeturn12search7turn7search9turn7search14  

## Concrete good / bad examples

### DynamoDB: плохой `Scan` vs хороший `Query` (Go, AWS SDK v2)

Bad (типичная LLM‑ошибка: «фильтруем по неключевому атрибуту через Scan»):

```go
out, err := db.Scan(ctx, &dynamodb.ScanInput{
    TableName: aws.String("Orders"),
    FilterExpression: aws.String("customer_id = :cid"),
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":cid": &types.AttributeValueMemberS{Value: customerID},
    },
})
```

Почему плохо: best practices рекомендуют избегать Scan на больших таблицах и проектировать ключи/индексы под Query. citeturn11search2turn11search13

Good (Query по partition key + диапазон по sort key):

```go
out, err := db.Query(ctx, &dynamodb.QueryInput{
    TableName: aws.String("Orders"),
    KeyConditionExpression: aws.String("pk = :pk AND begins_with(sk, :prefix)"),
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":pk":     &types.AttributeValueMemberS{Value: "CUSTOMER#" + customerID},
        ":prefix": &types.AttributeValueMemberS{Value: "ORDER#"},
    },
    Limit: aws.Int32(50),
})
```

Почему хорошо: соответствует принципам composite keys и «организации данных sort key‑ами», чтобы эффективно доставать группы связанных items. citeturn9search1turn9search3turn11search7  

### Redis: плохой `KEYS` vs хороший `SCAN` (Go)

Bad:

```go
keys, _ := rdb.Keys(ctx, "session:*").Result()
```

Почему плохо: `KEYS` — O(N), блокирует и «может надолго остановить» прод‑инстанс при большом keyspace; документация предупреждает об этом. citeturn2search4  

Good:

```go
var cursor uint64
for {
    var keys []string
    var err error
    keys, cursor, err = rdb.Scan(ctx, cursor, "session:*", 100).Result()
    if err != nil { return err }
    // обработка keys
    if cursor == 0 { break }
}
```

Почему хорошо: `SCAN` — инкрементальный итератор и рекомендуемая альтернатива для прод‑кейсов. citeturn2search16  

### Cassandra: `ALLOW FILTERING` как анти‑паттерн

Bad:

```sql
SELECT * FROM events WHERE tenant_id = ? AND event_type = ? ALLOW FILTERING;
```

Почему плохо: документация CQL подчёркивает, что `ALLOW FILTERING` может привести к непредсказуемым задержкам/нагрузке. citeturn2search11turn6search1  

Good (таблица/primary key под запрос):

```sql
-- expected query: tenant_id + event_type + time range
CREATE TABLE events_by_type (
  tenant_id text,
  event_type text,
  ts timeuuid,
  payload text,
  PRIMARY KEY ((tenant_id, event_type), ts)
) WITH CLUSTERING ORDER BY (ts DESC);
```

Почему хорошо: запрос обслуживается по partition key + range, что соответствует «таблица под запрос» и снижает риск фильтрации вне ключа. citeturn0search2turn6search5turn2search11  

## Anti-patterns и типичные ошибки/hallucinations LLM

LLM‑ошибки почти всегда сводятся к «подмене физических ограничений БД желаемой семантикой» или к «SQL‑переносу мышления». Ниже — список «красных флагов», которые template должен проверять автоматически (линтером/ревью/генераторами).

- **Придуманные JOIN/foreign keys** для DynamoDB‑класса или wide-column: на практике JOIN не поддерживается, и рекомендуется денормализовать/материализовать нужные представления. citeturn11search7turn0search2turn11search16  
- **Рекомендация Scan “как основной способ запросов”** без проектирования ключей/индексов: это противоречит best practices (Scan замедляется с ростом таблицы, «сканирует всё», может сжечь throughput). citeturn11search2turn11search13  
- **Ожидание точного TTL** (например, «удалится через 5 минут») в системах, где TTL best-effort: DynamoDB прямо говорит, что удаляет best-effort и может удалять в пределах ~двух дней, причём expire items могут ещё появляться в Query/Scan. citeturn2search9turn2search0  
- **Неучёт лимитов размера** (400 KB item в DynamoDB; 16 MiB документ в MongoDB) → LLM предлагает хранить «весь отчёт/все логи/все события пользователя» в одном item/document. citeturn8search6turn7search4  
- **Unbounded arrays в документной модели** (например, `user.events[]` бесконечно растёт): официальные anti-patterns описывают деградацию чтения/индексации и рекомендуют subset/references/bucket. citeturn7search3turn7search2  
- **Непонимание консистентности**: LLM часто пишет «система строго консистентна», игнорируя read/write concerns (MongoDB), tunable consistency (Cassandra), ограничения strong reads на GSI (DynamoDB) или last-writer-wins в multi-region режиме (DynamoDB global tables). citeturn3search4turn6search19turn3search0turn12search2  
- **Redis как “истина” без оговорок**: игнорирование асинхронной репликации и окна потери writes при failover, и отсутствие явной eviction/persistence стратегии. citeturn8search0turn8search4turn5search0turn8search17  
- **Time-series кардинальность**: LLM добавляет `user_id`/`request_id` в tags/labels, что противоречит best practices Prometheus и принципам schema design time-series (кардинальность). citeturn5search0turn5search12turn5search2  
- **NoSQL injection слепые зоны**: генерация запросов из «сырого JSON от клиента», разрешение `$where/$regex` и т.п. без белых списков и валидации; entity["company","MongoDB","database vendor"] и другие документные движки уязвимы к operator injection при неправильной сборке запроса. citeturn0search1turn0search4turn0search2  

## Review checklist для PR / code review и список файлов для template repo

### Review checklist

**Модель данных и запросы**
- Есть документированный список access patterns (CRUD/reads/aggregations), и каждый имеет сопоставление на конкретные операции/запросы хранилища (Query/GetItem vs Scan; primary key vs ALLOW FILTERING; pipeline stages и индексы). citeturn11search2turn12search27turn0search2  
- Для DynamoDB‑класса: нет неожиданных `Scan`/фильтров «после чтения всего»; если Scan есть — есть ADR и ограничение (параллельный скан, бэкфилл, маленький объём, отдельный job). citeturn11search2turn11search13  
- Для wide-column: нет `ALLOW FILTERING` в прод‑пути; primary key спроектирован под запросы и учитывает размер/горячесть партиций. citeturn2search11turn6search5turn6search1  
- Для document store: нет unbounded arrays; учтён лимит размера документа; применён subset/bucket/outlier где нужно. citeturn7search4turn7search3turn7search2turn7search30  
- Для time-series: не используются high-cardinality labels/tags; есть retention/downsampling стратегия. citeturn5search0turn5search12turn5search6  
- Для Redis‑класса: нет `KEYS`; есть стратегия eviction/persistence; учтены cluster slot правила при multi-key. citeturn2search4turn5search0turn5search19  

**Согласованность, транзакции, конкурентность**
- Явно описаны read/write guarantees и параметры (MongoDB readConcern/writeConcern; Cassandra consistency levels; DynamoDB strong vs eventual и ограничения индексов; global tables conflict model). citeturn3search4turn3search0turn6search19turn3search0turn12search2  
- Транзакции применяются точечно и с обоснованием: MongoDB transactions учитывают production considerations; Cassandra LWT используется как CAS; Redis MULTI/EXEC не интерпретируется как «полный rollback». citeturn3search1turn8search3turn5search2  
- Optimistic locking / conditional writes применены там, где это нужно (особенно в DynamoDB‑классе), и тестами проверены конфликтные сценарии. citeturn12search2turn12search6  

**TTL, удаление, жизненный цикл данных**
- TTL не используется как точный механизм бизнес‑правил, если хранилище удаляет best‑effort (DynamoDB); приложение фильтрует/проверяет expiry на чтении при необходимости. citeturn2search9turn2search0  
- Для Cassandra‑класса есть план контроля tombstones при TTL/удалениях. citeturn1search7turn3search3  
- Для Redis‑класса учтено, что expiration — комбинация пассивного и активного удаления, и это влияет на ожидания по времени. citeturn5search1turn5search5  

**Безопасность**
- Нет динамической сборки NoSQL‑запросов из «сырого ввода»; запрещены/валидированы operator‑ключи (`$where`, `$regex`…), согласно OWASP NoSQL Security. citeturn0search1turn0search4  
- Для MongoDB‑класса включены базовые меры безопасности деплоймента (authz/authn/TLS и т.п.) согласно руководству по security. citeturn0search3  

### Что оформить отдельными файлами в template repo

Набор файлов, который позволяет почти напрямую «перенести» этот стандарт в репозиторий и сделать его LLM‑дружелюбным:

- `docs/engineering/databases/nosql-modeling-standard.md` — этот документ как стандарт (scope, матрица выбора, паттерны по типам, ограничения и caveats). citeturn11search7turn12search27turn0search2turn5search0turn5search12  
- `docs/engineering/databases/nosql-access-patterns-template.md` — шаблон таблицы/формы для описания access patterns (поля: запрос, ключи, индексы, объёмы, SLO, согласованность, TTL). Основание: проектировать схему/ключи под use cases/запросы. citeturn12search3turn9search1turn0search2  
- `docs/llm/nosql-instructions.md` — выдержка MUST/SHOULD/NEVER + «типовые запреты», чтобы LLM не генерировал Scan/KEYS/ALLOW FILTERING/неверные гарантии. citeturn11search2turn2search4turn2search11turn2search9  
- `docs/adr/adr-template-datastore-choice.md` — ADR-шаблон «почему NoSQL, почему этот класс, какие компромиссы» с обязательными секциями: consistency/transactions, TTL semantics, limits, migration/evolution plan. Обоснование: database-per-service имеет преимущества, но усложняет кросс‑сервисные запросы/транзакции и требует компенсирующих паттернов (Saga/CQRS/API composition). citeturn10search6turn10search8turn10search27  
- `.github/pull_request_template.md` — блок «Data model changes»: обновлён access-patterns doc, нет новых full scans, есть тесты на конкуренцию/консистентность, учтены ограничения размеров и TTL. citeturn11search2turn7search4turn2search9turn12search2  
- `docs/security/nosql-security.md` — короткие правила по NoSQL injection и запретам на «сырой JSON в запрос», с ссылкой на OWASP NoSQL Security Cheat Sheet. citeturn0search1turn0search4  
- `docs/engineering/databases/redis-usage-policy.md` — отдельный файл, потому что Redis часто используется как кеш/сессии и имеет специфические риски: асинхронная репликация, eviction, cluster slotting, `KEYS` vs `SCAN`, TTL semantics. citeturn8search0turn5search0turn5search19turn2search4turn5search1  
- `docs/engineering/databases/dynamodb-policy.md` — отдельный файл с «неизбежными» ограничениями: no JOIN, Query vs Scan, materialized aggregation, 400KB item, TTL best-effort, optimistic locking, global tables caveats. citeturn11search7turn11search16turn8search6turn2search9turn12search2  
- `docs/engineering/databases/document-store-policy.md` — отдельный файл для документной БД: лимит 16MiB, unbounded arrays anti-pattern, subset/bucket/outlier, schema validation, schema versioning. citeturn7search4turn7search3turn7search2turn12search7turn7search14  
- `docs/engineering/databases/wide-column-policy.md` — table-per-query, partition key, `ALLOW FILTERING` запрет, secondary index правила, TTL/tombstones, tunable consistency и LWT/CAS. citeturn0search2turn2search11turn6search2turn1search7turn6search19turn8search3  
- `docs/engineering/databases/time-series-policy.md` — правила для time-series: ограничения кардинальности (labels/tags), retention/downsampling, schema design (tags vs fields), плюс отдельные требования для Timestream‑класса (partition keys). citeturn5search0turn5search12turn5search2turn5search6turn5search3  

Для полноты инженерного контекста (и чтобы увязать с «production-ready микросервисом»), в корневом `docs/engineering/architecture/data-persistence.md` стоит одной страницей зафиксировать принцип «хранилище на сервис», а также что cross-service согласованность решается паттернами, а не shared DB/2PC. Ссылаться на guidance от entity["company","Amazon Web Services","cloud provider"] и entity["company","Microsoft","software vendor"] (database-per-service + Saga). citeturn10search6turn10search8turn10search27turn10search26