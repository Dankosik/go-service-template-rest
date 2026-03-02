# Engineering standard для синхронных коммуникаций между микросервисами на Go

Этот материал предназначен для превращения в **внутренний engineering standard** и **LLM-instruction docs** внутри template-репозитория: чтобы разработчик мог клонировать репо и сразу делать production-ready сервис, а LLM (ChatGPT/Codex/Claude Code и т.п.) генерировала идиоматичный, безопасный и поддерживаемый Go-код без «догадок» про протоколы, таймауты, ретраи, модели ошибок и совместимость контрактов. Нормативность формулировок (MUST/SHOULD) согласуется с общепринятыми уровнями требований (RFC 2119-стиль), что явно используется в зрелых API-гайдах. citeturn13search7turn14search4

## Scope

**Когда применять (в рамках синхронного взаимодействия):**  
Подход рассчитан на микросервисную архитектуру, где сервисы вызывают друг друга по request-reply (RPC/HTTP), и вам нужны: (a) формальные контракты (схемы), (b) однозначная модель ошибок, (c) строгие дедлайны/таймауты, (d) предсказуемая совместимость при эволюции API. Это особенно актуально, когда вы хотите опираться на генерацию клиентов/серверов и механические проверки обратной совместимости (например, схемы Protobuf) и превращать большинство решений в «boring defaults». citeturn1search4turn0search1turn10view1turn13search0

**Когда не применять (или применять частично):**  
Если операции часто занимают непредсказуемо долго, требуют сложных саг/оркестрации, или естественно выражаются через события и асинхронные потоки, то принуждение всего домена к синхронным цепочкам вызовов повышает риск каскадных отказов и «запирания» ресурсов. В таких случаях используйте паттерны асинхронной обработки (например, async request-reply/long-running operations) вместо удержания соединения «до победы». citeturn8search8turn2search1turn2search5

**Границы темы:**  
Документ фокусируется на синхронных коммуникациях (HTTP/JSON, gRPC/Protobuf, Connect), а также на обязательных сопутствующих аспектах, без которых синхронная интеграция «ломается» в проде: service discovery, дедлайны/таймауты, retry-политики, идемпотентность, модель ошибок, пагинация, long-running operations, backward compatibility и versioning контрактов. citeturn1search2turn0search1turn2search2turn0search0turn2search0turn2search1turn10view1

## Recommended defaults для greenfield template

Ниже — рекомендуемые **boring, battle-tested defaults**, которые удобно «вшить» в шаблон Go-микросервиса, чтобы LLM не гадала, а следовала стандарту.

**Дефолтный выбор транспорта:**
- **Внутренние (service-to-service) синхронные вызовы: gRPC + Protobuf (proto3)** как основной стандарт. Это упрощает типизацию, генерацию клиентов/серверов и дисциплину обратной совместимости через правила эволюции схем. citeturn10view0turn11search19turn1search4  
- **Внешние API (для браузеров/партнёров/публичных клиентов): HTTP/JSON + OpenAPI 3.1**, публикуемые через **API Gateway** или **BFF**, а не напрямую из каждого внутреннего сервиса. OpenAPI даёт стандартный контракт для HTTP API. citeturn1search5turn8search1turn8search0  
- **Connect-подход (опционально):** если вы хотите **единый Protobuf-контракт** и при этом «человеко-дебажные» HTTP-эндпоинты (включая JSON) и совместимость с gRPC/gRPC-Web — можно использовать Connect поверх Protobuf. При этом фиксируйте правила таймаутов и error model именно по Connect Protocol (включая `connect-timeout-ms` и формат ошибок). citeturn1search7turn12view0turn1search11

**Контракты и их хранение в репозитории (API contracts as code):**
- **Protobuf (proto3) как source of truth** для внутренних RPC; контракты версионируются **мажорной версией в конце protobuf package** (например, `company.product.orders.v1`) — это зрелая и широко тиражируемая практика в гайдлайнах API-дизайна. citeturn14search0turn14search1turn10view0  
- **Никогда не переиспользуйте номера полей и не «перенумеровывайте ради красоты»**; повторное использование номеров делает декодирование неоднозначным и может приводить к ошибкам парсинга, corruption и даже утечкам данных — это прямо отмечено в официальной документации Protobuf. citeturn10view1turn10view0  
- В template по умолчанию включите механические проверки схем: **lint** и **breaking-change detection** (например, через Buf). Это снижает зависимость от «внимательности ревьюера» и уменьшает вероятность того, что LLM случайно внесёт breaking change в `.proto`. citeturn13search5turn13search12turn13search0

**Internal vs external APIs: границы и публикация:**
- **Internal API**: допускает более быструю эволюцию, но всё равно требует дисциплины совместимости (wire/source compatibility), потому что клиенты и серверы обновляются не атомарно. citeturn14search1turn11search19turn10view1  
- **External API**: публикуется через **Gateway** (маршрутизация, политика безопасности, rate limiting, TLS, WAF и т.п.) и/или **BFF** (адаптация под конкретный клиент/канал). Это уменьшает разрастание клиент-специфической логики внутри доменных микросервисов. citeturn8search1turn8search0turn8search16

**Service discovery (boring default):**
- Если деплой в Kubernetes, **используйте Service DNS** (а не IP/конфиг-файлы с адресами): Kubernetes создаёт DNS-записи для сервисов и pod’ов, позволяя обращаться по стабильным именам. citeturn1search2turn1search6  
- Для gRPC клиентов базовая схема: name resolver отдаёт список адресов, а LB policy распределяет вызовы; DNS — распространённый резолвер (в том числе в grpc-go примерах). citeturn9search1turn9search4turn9search5

**Дедлайны и таймауты (обязательная часть контракта выполнения):**
- **gRPC:** дедлайн должен быть установлен на стороне клиента; дедлайны помогают ограничить время выполнения и освобождать ресурсы; сервер отменяет вызов при истечении дедлайна (CANCELLED/DEADLINE_EXCEEDED сценарии) — это ключевой механизм надёжности. citeturn0search9turn0search1turn0search13  
- **Connect:** если заголовок таймаута отсутствует, сервер **должен считать таймаут бесконечным**; значит, в template нужно сделать правило «таймаут обязателен», иначе вы по умолчанию получите потенциально «вечные» запросы. citeturn12view0  
- **HTTP (Go net/http):** нулевые значения таймаутов означают «без таймаута» как на клиенте (`http.Client.Timeout`), так и на сервере (`ReadTimeout`, `WriteTimeout` и др.). Поэтому template должен явно задавать таймауты сервера и клиента. citeturn6view0turn5view3  
- **Контекст в Go:** отмена контекста освобождает связанные ресурсы; `WithTimeout/WithDeadline` требует `cancel()` для корректного высвобождения ресурсов. Это не «стилистика», а практическое предотвращение утечек и зависаний. citeturn4view1

**Retry policy (по умолчанию — крайне консервативно):**
- **gRPC:** ретраи включены «прозрачно» (transparent retry) только в узких безопасных случаях, но **дефолтной retry policy нет**, т.е. «само не заретраится правильно». Настраивать retry policy нужно осознанно через Service Config (и желательно с throttling), иначе можно усилить нагрузку и усугубить деградацию. citeturn2search2turn2search6turn9search5  
- На уровне инфраструктуры/edge proxy (например, Envoy/Gateway) ретраи и circuit breaking являются отдельными правилами и должны иметь лимиты, чтобы не превращаться в «retry storm». citeturn8search2turn8search6

**Идемпотентность как условие для ретраев:**
- В HTTP семантике часть методов по определению идемпотентны (например, PUT/DELETE и safe methods), и это прямо фиксируется стандартом HTTP. citeturn0search0turn0search4  
- Для «неидемпотентных» HTTP операций (POST/PATCH) всё чаще применяется `Idempotency-Key`, но на уровне IETF это пока draft (не финальный RFC), поэтому в стандарте нужно явно обозначить статус и политику внедрения. citeturn7search0turn7search20  
- В Connect (и шире в RPC/Protobuf экосистеме) можно явно помечать методы как `NO_SIDE_EFFECTS` через `idempotency_level`, что позволяет, например, безопасное использование GET в unary Connect для операций без побочных эффектов. citeturn12view0

**Error model (единые правила интерпретации ошибок):**
- **gRPC:** используйте только стандартные gRPC status codes; статус — это код + сообщение, и это базовая часть протокола. citeturn1search4turn1search24  
- Для структурированных деталей ошибки в RPC ориентируйтесь на `google.rpc.Status` (код/сообщение/details); это считается канонической моделью ошибок в gRPC-экосистеме и отражено в зрелых API-гайдах. citeturn1search0turn13search3  
- **HTTP:** для JSON-ошибок используйте стандарт **Problem Details** (RFC 9457, obsoletes RFC 7807) с `application/problem+json`. citeturn0search3turn0search23  
- **Connect:** имеет формально описанную модель ошибок и mapping HTTP status → Connect code; это нужно учитывать, чтобы не «потерять» смысл ошибок при проксировании/интеграциях. citeturn12view0

**Pagination и Long-running operations (LRO):**
- Для списков по умолчанию используйте **page token** (cursor-based) паттерн: `page_size`, `page_token`, `next_page_token` и правило «нельзя менять другие параметры между страницами, иначе INVALID_ARGUMENT». Это снижает риск ошибок и делает пагинацию стабильной при изменениях данных. citeturn2search0turn2search8  
- Если операция потенциально долгая, используйте **long-running operations** (токен/operation resource + polling), а не «держите соединение минутами». Это описано как отдельный паттерн. citeturn2search1turn2search5turn8search8

**Backward compatibility и versioning (строгая дисциплина):**
- Protobuf: номера полей «священны», их нельзя менять/переиспользовать; это фундамент wire compatibility. citeturn10view1turn10view0  
- Версионирование интерфейсов через мажорную версию в package/URI — практический «дефолт» в зрелых гайдлайнах; обратную совместимость описывайте через понятия source/wire compatibility и связывайте с уровнем стабильности поверхности. citeturn14search0turn14search1turn11search0  
- Для управления ожиданиями версий артефактов (SDK, libs) используйте SemVer как минимальный общий язык, но не смешивайте «SemVer пакета/библиотеки» и «SemVer публичного API» без явного ADR — это разные объекты с разными политиками. citeturn11search1turn14search0

## Decision matrix и trade-offs

Ниже — практическая матрица выбора, которую LLM должна использовать, а человек — применять как основу для ADR. Указаны **trade-offs**, а где выбор спорный — отмечено явно.

image_group{"layout":"carousel","aspect_ratio":"16:9","query":["microservices api gateway diagram","backend for frontend pattern diagram","gRPC vs REST diagram microservices","Kubernetes service discovery DNS diagram"],"num_per_query":1}

**gRPC/Protobuf (внутренний стандарт по умолчанию):**
- Сильные стороны: строгая типизация и генерация кода из `.proto`, единый набор status codes, стандартные протоколы health checking/reflection, понятная дисциплина совместимости при эволюции схем. citeturn10view0turn1search4turn7search2turn7search7  
- Слабые стороны: сложнее «ручной дебаг» без tooling; браузерные клиенты требуют дополнительных адаптеров (gRPC-Web/прокси) или alternative transports; при неправильных таймаутах/ретраях легко получить каскадные отказы. citeturn0search9turn2search2turn7search7turn12view0  
- Когда выбирать: **service-to-service** внутри доверенной сети/кластера, высокая интенсивность вызовов, много внутренних клиентов, нужен бинарный эффективный формат и унификация error model. citeturn1search4turn9search1turn9search5

**HTTP/JSON + OpenAPI (внешний стандарт по умолчанию):**
- Сильные стороны: максимальная совместимость клиентов/инфраструктуры, стандартное описание API через OpenAPI, простота интеграций и «человеческий» дебаг; стандартные форматы ошибок через RFC 9457. citeturn1search5turn0search3  
- Слабые стороны: «строгость» типов слабее, чем у protobuf; нужно жёстко стандартизировать error schema и pagination, иначе API быстро деградирует в «stringly-typed JSON». citeturn0search3turn2search0turn1search5  
- Когда выбирать: публичные/партнёрские API, BFF для фронтендов, интеграции через API gateway/ingress, сценарии где важны кеширование, наблюдаемость и зрелые HTTP инструменты. citeturn8search0turn8search1turn1search5

**Connect (как «мост» между RPC и HTTP):**
- Сильные стороны: один Protobuf контракт; HTTP-совместимость (включая HTTP/1.1 для большинства unary/стриминг-типов, кроме bidi), JSON или Protobuf payload, отсутствие трейлеров в протоколе ⇒ легче проходит через инфраструктуру; понятные заголовки, включая `connect-timeout-ms`. citeturn12view0turn1search11  
- Слабые стороны: экосистема менее «встроена» в платформы, чем чистый HTTP/JSON; нужно принять специфику Connect error model и заголовков; риск «двойного стандарта», если параллельно поддерживать и REST, и Connect без чётких границ. citeturn12view0turn1search7  
- Когда выбирать: когда вы хотите **единый контракт** и одновременно: (a) CLI/curl-дебаг бинарных/JSON RPC, (b) поддержку браузера через gRPC-Web совместимость, (c) постепенную миграцию. citeturn1search7turn12view0

**Internal vs external API: прямой доступ vs gateway/BFF:**
- Direct-to-service (внешние клиенты ходят прямо в микросервисы) часто упирается в безопасность/эволюцию/клиент-специфичность; поэтому default — использовать Gateway API/Ingress/Gateway и при необходимости BFF. citeturn8search1turn8search0turn8search16  
- BFF оправдан, когда разные клиенты (web/mobile/партнёры) требуют разных API-форматов или агрегации данных: BFF изолирует фронтенды от внутренних изменений. citeturn8search0turn8search16

**Retries / timeouts: библиотека vs прокси:**
- Прокси/mesh/gateway может централизованно задавать ретраи/таймауты/circuit breakers, но это не отменяет необходимости дедлайнов на уровне приложения (клиент должен ограничивать ожидание). citeturn0search9turn8search6turn8search2  
- В gRPC retry policy задаётся через service config; без явной политики «по умолчанию» ретраи ограничены прозрачными случаями. Это аргумент в пользу: «ретраи включать только через стандартную политику + бюджеты». citeturn2search2turn2search6turn9search5

## Набор правил для LLM в формате MUST / SHOULD / NEVER

Правила ниже предназначены для помещения в LLM-instruction docs. Они сформулированы так, чтобы модель могла принимать решения автоматически и не «изобретать» транспорт, таймауты, контракт или error model.

**MUST (обязательно):**
- MUST выбирать transport по умолчанию так: **внутри кластера service-to-service → gRPC/Protobuf**, с явными дедлайнами; **внешний край → HTTP/JSON с OpenAPI**, публикуемый через gateway/BFF. citeturn1search5turn0search9turn8search1turn8search0  
- MUST задавать дедлайн/таймаут на каждый исходящий вызов:  
  - gRPC: deadline в `context` на клиенте; считать отсутствие дедлайна ошибкой стандартов. citeturn0search9turn0search1turn4view1  
  - Connect: всегда выставлять `connect-timeout-ms`; отсутствие заголовка трактуется как бесконечный таймаут. citeturn12view0  
  - HTTP: не использовать `http.DefaultClient` без таймаута; `http.Client.Timeout == 0` означает «нет таймаута». citeturn6view0turn5view2  
- MUST на HTTP-сервере (если сервис обслуживает HTTP вход) задавать timeouts (`ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`) — нули/отрицательные значения означают отсутствие таймаута. citeturn5view3  
- MUST пропагировать `context.Context` через весь call chain и уважать отмену: не создавать `context.Background()` внутри request scope; всегда вызывать `cancel()` у `WithTimeout/WithDeadline`. citeturn4view1turn0search9  
- MUST применять единый error model:  
  - gRPC: возвращать только стандартные gRPC status codes (не «ошибку в payload при OK») и при необходимости использовать структурированные детали (например, `google.rpc.Status`). citeturn1search4turn13search3turn1search0  
  - HTTP: ошибки отдавать в формате RFC 9457 Problem Details (`application/problem+json`) плюс корректный HTTP status. citeturn0search3  
  - Connect: следовать Connect protocol error model и mapping. citeturn12view0  
- MUST для list/search API использовать pagination с `page_token` и `next_page_token` (cursor/token), а не «голый offset» как дефолт; соблюдать правило неизменности параметров между страницами. citeturn2search0turn2search8  
- MUST для потенциально долгих операций выбирать LRO/async request-reply вместо удержания соединения сверх разумного дедлайна. citeturn2search1turn8search8  
- MUST обеспечивать backward compatibility контрактов:  
  - Protobuf: не менять номера полей, не переиспользовать их, не «перенумеровывать». citeturn10view1turn10view0  
  - Версионирование: мажорная версия в конце protobuf package; breaking change → новый major. citeturn14search0turn11search19  
- MUST документировать proto-элементы комментариями (service/method/message/field/enum): это важно для tooling и читаемости контрактов. citeturn14search2turn14search19

**SHOULD (рекомендуется):**
- SHOULD использовать механические проверки контрактов: lint + breaking checks в CI (например, Buf) и/или специализированные линтеры для AIP-подобных правил. citeturn13search5turn13search12turn14search5  
- SHOULD стандартизировать health checking для gRPC через протокол `grpc.health.v1` и обновлять статусы корректно. citeturn7search2turn7search6  
- SHOULD включать reflection только для dev/debug сценариев и явно управлять экспозицией (reflection — опциональное расширение). citeturn7search7turn7search3  
- SHOULD ограничивать ретраи: включать retry policy только для явно идемпотентных операций и с ограничениями (throttling/бюджеты); по умолчанию ретраи должны быть минимальными. citeturn2search2turn2search6turn0search0  
- SHOULD рассматривать `Idempotency-Key` для POST/PATCH, но отмечать, что это IETF draft (стандарт может эволюционировать); поведение сервера (хранилище ключей, TTL, дедупликация) должно быть явно специфицировано. citeturn7search0turn7search20  
- SHOULD, если сервис использует API Gateway/mesh, согласовывать таймауты/ретраи/circuit breakers с инфраструктурой, чтобы не получить «два независимых набора ретраев». citeturn8search2turn8search6  
- SHOULD проектировать внешние API с учётом типовых API security рисков: например, объектный уровень авторизации — частая причина инцидентов; проверки должны быть явными в каждом handler’e. citeturn8search3turn8search7

**NEVER (запрещено):**
- NEVER делать исходящие вызовы без таймаута/дедлайна (включая использование `http.DefaultClient`/`DefaultTransport` без настроек таймаутов на верхнем уровне политики клиента). citeturn6view0turn0search9  
- NEVER писать «вечные ретраи» или ретраи без backoff/лимитов, особенно для неидемпотентных операций; это ускоряет деградацию при инцидентах. citeturn2search2turn8search2  
- NEVER возвращать ошибки «в теле» при HTTP 200 / gRPC OK; это ломает контракт и клиентскую обработку ошибок. citeturn1search4turn0search3  
- NEVER переиспользовать номера полей Protobuf или менять их «безболезненно» — это breaking change на wire-уровне. citeturn10view1turn10view0  
- NEVER экспонировать внутренние микросервисы напрямую как внешний контракт без gateway/BFF политики, если API предназначено для внешних клиентов (безопасность/эволюция/версии). citeturn8search1turn8search0  
- NEVER вводить offset-пагинацию как дефолт для «больших» коллекций; используйте token-based подход. citeturn2search0turn2search8

## Concrete good / bad examples

Ниже — примеры, которые можно почти напрямую перенести в docs как «канонические». Код — иллюстративный, но ориентирован на идиоматичный Go и описанные стандарты.

### Good: HTTP client с таймаутом и request-scoped context
```go
// Good: общий HTTP-клиент создаётся один раз при старте приложения,
// имеет верхнеуровневый Timeout (иначе Timeout=0 означает "без таймаута").
// На каждый запрос также прокидывается request-scoped context.
type DownstreamClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewDownstreamClient(baseURL string, timeout time.Duration) *DownstreamClient {
	return &DownstreamClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *DownstreamClient) GetThing(ctx context.Context, id string) (*Thing, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/things/"+id, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// ... decode, handle status codes ...
	return &Thing{}, nil
}
```
Почему это good: `http.Client.Timeout` задаёт верхнюю границу времени и при `0` таймаута не будет; отмена/дедлайн также прокидывается через `Request.Context`. citeturn6view0turn4view1

### Bad: HTTP вызов без таймаута и без контекста
```go
// Bad: DefaultClient с Timeout=0 => запрос может зависнуть навсегда.
resp, err := http.Get(url) // не контролируется request-scope timeout/cancel
```
Почему это bad: нулевой Timeout означает «нет таймаута», что в проде превращается в висящие goroutine/коннекты при проблемах сети/даунстрима. citeturn6view0turn4view1

### Good: net/http server с явно заданными таймаутами
```go
srv := &http.Server{
	Addr:              ":8080",
	Handler:           handler,
	ReadHeaderTimeout: 5 * time.Second,
	ReadTimeout:       30 * time.Second,
	WriteTimeout:      30 * time.Second,
	IdleTimeout:       60 * time.Second,
}
```
Почему это good: `ReadHeaderTimeout/ReadTimeout/WriteTimeout/IdleTimeout` имеют документированные семантики; нулевые/отрицательные значения означают отсутствие таймаута, поэтому настройки должны быть явными. citeturn5view3

### Good: HTTP error response в стиле RFC 9457 (Problem Details)
```go
type ProblemDetails struct {
	Type     string `json:"type,omitempty"`
	Title    string `json:"title,omitempty"`
	Status   int    `json:"status,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

func writeProblem(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ProblemDetails{
		Title:  title,
		Status: status,
		Detail: detail,
	})
}
```
Почему это good: RFC 9457 стандартизует machine-readable формат ошибок для HTTP API и заменяет RFC 7807, что уменьшает «зоопарк» кастомных error payloads. citeturn0search3turn0search23

### Bad: «Ошибка в теле при 200 OK»
```go
w.WriteHeader(http.StatusOK)
_ = json.NewEncoder(w).Encode(map[string]any{
	"error": "permission denied",
})
```
Почему это bad: ломает контракт и стандартную обработку ошибок на клиентах; для ошибок должны быть корректные HTTP status + стандартный формат, а не «OK + error field». citeturn0search3turn1search5

### Good: Pagination по page_token (AIP-158 style)
```proto
message ListWidgetsRequest {
  int32 page_size = 1;
  string page_token = 2;
}

message ListWidgetsResponse {
  repeated Widget widgets = 1;
  string next_page_token = 2;
}
```
Почему это good: паттерн page token фиксирует правила взаимодействия клиента и сервера (включая корректность повторных запросов и неизменность параметров), и он рекомендуем в зрелых гайдах пагинации. citeturn2search0turn2search8

### Bad: offset-пагинация «как дефолт»
```proto
message ListWidgetsRequest {
  int32 limit = 1;
  int32 offset = 2; // Bad default for large datasets
}
```
Почему это bad: offset-пагинация часто нестабильна при изменениях набора данных и провоцирует дорогостоящие запросы на больших объёмах; поэтому в стандарте должен быть token-based дефолт. citeturn2search0turn2search8

## Anti-patterns и типичные ошибки/hallucinations LLM

Этот раздел — список того, что LLM часто «галлюцинирует» или делает по привычке, и что нужно запретить/перехватывать правилами и ревью.

**Таймауты и контексты:**
- «Забытый таймаут» (особенно `http.Client.Timeout=0`) и отсутствие `context.WithTimeout` на исходящих вызовах → зависания и постепенное истощение ресурсов. citeturn6view0turn4view1  
- Создание `context.Background()` внутри handler’а или клиента «для удобства» → отмена запроса не останавливает downstream calls. citeturn4view1turn0search9  
- Для Connect: отсутствие `connect-timeout-ms` → «бесконечные» RPC по спецификации. citeturn12view0

**Retries и идемпотентность:**
- Включение ретраев «на всё подряд» или ретраи POST без идемпотентности → двойные списания/двойные сайд-эффекты. HTTP определяет идемпотентность методов, но это не делает произвольный POST идемпотентным. citeturn0search0turn7search0  
- Игнорирование факта, что в gRPC **нет дефолтной retry policy** (есть только transparent retry в узких случаях) → ложное ожидание «оно само заретраится». citeturn2search2turn9search2  
- «Retry storm» при деградации даунстрима, особенно если ретраи настроены и в клиенте, и в gateway/mesh одновременно. citeturn8search2turn8search6

**Error model:**
- Смешивание моделей ошибок: HTTP 200 с error payload; gRPC OK с `error` в response message; произвольные строковые коды ошибок вместо canonical codes. citeturn1search4turn0search3turn13search3  
- Потеря причинно-следственной информации: отсутствие structured error details (в RPC) или неприменение Problem Details (в HTTP). citeturn1search0turn0search3  
- Для Connect: неправильное ожидание трейлеров (в протоколе Connect трейлеры не используются «как в gRPC»), или неправильная интерпретация ошибок в streaming, где HTTP статус может быть 200, а ошибка — в конце body. citeturn12view0

**Contracts / versioning / compatibility:**
- Переиспользование/перенумерация полей Protobuf «потому что поле удалили» → wire ambiguity, parse errors, corruption. citeturn10view1turn10view0  
- Отсутствие комментариев к proto, из-за чего tooling/доки/ревью ухудшаются. citeturn14search2turn14search19  
- «Тихие» breaking changes без новой major версии API (package v2 и т.п.) — даже если это «внутренний» API, клиенты обновляются не синхронно. citeturn14search0turn14search1turn11search19

**Service discovery / адресация:**
- Хардкод IP/портов и игнорирование DNS service discovery в Kubernetes → ломается при рескейле/роллинге. citeturn1search2turn1search6  
- Попытки LLM «реализовать свой service discovery» вместо использования платформенных механизмов (Kubernetes DNS) или стандартных gRPC resolver/LB механизмов. citeturn9search1turn9search4turn1search2

**API boundary ошибки:**
- Экспонирование внутренних API напрямую наружу без gateway/BFF → неконтролируемые клиенты, хрупкая эволюция, повышенная поверхность атаки; BFF описан как способ развязать backend и разные фронтенд-интерфейсы. citeturn8search0turn8search1  
- Игнорирование типовых API security рисков (например, объектного уровня авторизации) в handler’ах. citeturn8search3turn8search7

## Review checklist для PR/code review

Этот чеклист предназначен для PR ревью и для LLM как «самопроверки» перед генерацией/рефакторингом.

**Transport & boundaries**
- Проверено: внутренние вызовы — gRPC/Protobuf (или зафиксирован ADR на иной транспорт); внешняя публикация — через gateway/BFF, а не прямой доступ к доменному сервису. citeturn8search1turn8search0turn11search19  
- Если используется Connect: соответствует спецификации (таймауты/ошибки/streaming semantics). citeturn12view0

**Timeouts, deadlines, context**
- Каждый исходящий вызов имеет дедлайн/таймаут; нет `http.DefaultClient` без `Timeout`; нет «вечных» запросов. citeturn6view0turn0search9  
- Весь request scope использует один `context`, отмена/дедлайн уважены; `cancel()` вызывается. citeturn4view1  
- HTTP server настроен с `ReadHeaderTimeout/ReadTimeout/WriteTimeout/IdleTimeout` (или явно обосновано почему нет). citeturn5view3

**Retries & idempotency**
- Ретраи включены только там, где это безопасно и описано; есть backoff/лимиты/throttling; нет ретраев «на всё». citeturn2search2turn2search6turn8search2  
- Для операций с сайд-эффектами есть явная идемпотентность: HTTP метод идемпотентен по RFC или реализован `Idempotency-Key` (с явно описанным поведением и статусом стандарта). citeturn0search0turn7search0

**Error model**
- gRPC: используется canonical status codes; ошибки не «прячутся» в response payload. citeturn1search4turn1search24  
- RPC structured errors: при необходимости применён `google.rpc.Status`/details. citeturn13search3turn1search0  
- HTTP: ошибки — RFC 9457 Problem Details, корректные status codes. citeturn0search3

**Pagination & LRO**
- List API соответствует page-token паттерну (`page_size`, `page_token`, `next_page_token`), нет дефолта offset. citeturn2search0turn2search8  
- Долгие операции не держат соединение бесконечно: используется LRO/async request-reply. citeturn2search1turn8search8

**Contracts & compatibility**
- Protobuf: нет переиспользования/перенумерации полей; breaking change оформлен как новый major (package v2 и т.д.). citeturn10view1turn14search0  
- Контракты прокомментированы (service/method/message/field/enum). citeturn14search2turn14search19  
- В CI включены lint/breaking checks для контрактов (например, Buf). citeturn13search5turn13search12

**Security**
- В handler’ах есть явные проверки авторизации на уровне объектов/идентификаторов; нет «trust the client». citeturn8search3turn8search7

## Что из результата нужно оформить отдельными файлами в template repo

Ниже — список файлов/разделов, которые следует вынести в template, чтобы стандарт стал исполнимым, а LLM имела «точку истины» внутри репозитория.

**docs/engineering-standards/**
- `sync-communications.md`: данный стандарт (transport selection, deadlines/timeouts, retries, idempotency, error model, pagination, LRO, compatibility, service discovery). citeturn0search9turn12view0turn2search0turn10view1turn1search2  
- `error-model.md`: каноническая модель ошибок: gRPC status codes + `google.rpc.Status`/details, HTTP RFC 9457 Problem Details, Connect mapping. citeturn1search4turn13search3turn0search3turn12view0  
- `timeouts-and-retries.md`: правила дедлайнов, retry budgets/throttling, взаимодействие с gateway/mesh. citeturn2search2turn2search6turn8search2turn6view0  
- `api-boundaries.md`: internal vs external, gateway/BFF, политика публикации API. citeturn8search1turn8search0

**docs/llm-instructions/**
- `llm-sync-communication-rules.md`: MUST/SHOULD/NEVER правила из этого документа в «машиночитаемой» форме, плюс мини-алгоритм выбора транспорта и чеклист «перед генерацией кода». citeturn14search4turn0search9turn12view0turn2search0  
- `llm-contract-editing.md`: отдельные правила для редактирования `.proto` (запрет перенумерации, обязательность комментариев, reserved/deprecated, versioning). citeturn10view1turn14search2turn14search0

**api/** (source of truth для контрактов)
- `api/proto/<org>/<service>/v1/*.proto`: protobuf контракты с мажорной версией в package (v1), с комментариями и правилами совместимости. citeturn14search0turn14search2turn10view1  
- (опционально) `api/openapi/<service>.yaml` или генерируемый OpenAPI (если вы делаете HTTP edge): OpenAPI 3.1 как контракт HTTP API. citeturn1search5turn1search5

**ci/** и конфигурация линтеров/проверок
- `buf.yaml` / `buf.gen.yaml` + CI job: lint + breaking detection, чтобы отлавливать unsafe изменения схем автоматически. citeturn13search17turn13search12turn13search0  
- (если используете grpc-gateway) конфиги генерации OpenAPI из proto аннотаций — иначе LLM начнёт «рисовать OpenAPI руками» и расходиться с реальным контрактом. citeturn13search6turn13search2

**docs/adr/**
- `adr-template.md` + ADR:  
  - «Transport selection: gRPC internal, HTTP external (через gateway/BFF)»  
  - «Retry policy and idempotency policy»  
  - «Error model standardization»  
  Такие ADR опираются на зрелые гайды по versioning/backward-compatibility и формализуют отступления от дефолтов. citeturn14search0turn14search1turn2search2turn0search3

Встроив эти файлы в template, вы обеспечите, что LLM перестанет «угадывать», а будет следовать репозиторно-локальным правилам, основанным на спецификациях HTTP (RFC 9110/9457/8288), gRPC руководствах (deadlines/status/retry/health/reflection), спецификациях Protobuf и зрелых API design практиках. citeturn0search4turn0search3turn7search1turn0search9turn1search4turn2search2turn7search2turn7search7turn10view1turn14search0turn13search5