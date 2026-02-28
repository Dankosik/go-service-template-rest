# Secure coding standard для production-ready Go-микросервиса и LLM-инструкций

## Область применения и границы подхода

Этот стандарт предназначен для greenfield Go-микросервиса, который принимает запросы по HTTP (чаще всего JSON API), обрабатывает недоверенный ввод, обращается к БД/кэшу/внешним HTTP-сервисам и должен быть «secure-by-default» при генерации кода LLM’ом. В качестве ориентиров по классам рисков используется entity["organization","OWASP","appsec nonprofit"] (в частности API Security Top 10 и Cheat Sheet Series), а по протоколу HTTP — нормы entity["organization","IETF","internet standards body"] (RFC). citeturn14search1turn14search0turn14search5turn1search4turn10search0

Особенно хорошо подход применим, когда:
- сервис имеет внешние входы (публичный API или internal API в «не-идеально доверенной» сети), где ошибки Broken Authentication / Broken Authorization, инъекции и DoS-эффекты встречаются системно; citeturn14search0turn14search1turn14search5
- сервис выполняет исходящие сетевые запросы (риск SSRF, утечек через редиректы, зависаний без timeout); citeturn0search1turn5view2
- сервис работает с файлами/архивами/загрузками (типичный источник path traversal, unsafe file handling, resource exhaustion); citeturn12view0turn13view0turn7search1

Подход не является заменой:
- threat modeling и дизайн-решений по сети/идентификации/сегментации (напр. SSRF нельзя «вылечить только кодом» — часто нужны egress-политики и контроль маршрутов); citeturn0search1
- полноценного процесса security review/pen-test и оперативного управления уязвимостями зависимостей (для этого отдельно используются инструменты проверки уязвимостей и политики обновлений). citeturn7search2turn7search9

## Рекомендуемые безопасные defaults для greenfield Go-микросервиса

Ниже — «boring, battle-tested defaults», которые шаблон должен включать сразу, чтобы LLM не «додумывала» критичные детали и не генерировала небезопасные варианты.

По умолчанию ориентируйтесь на актуальный стабильный Go релиз: на февраль 2026 в официальных release notes указан Go 1.26. citeturn22search0turn22search3

**Сетевой периметр и HTTP-сервер**
- Всегда создавайте `http.Server` явно и выставляйте таймауты/лимиты: `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`, `MaxHeaderBytes`. Нулевые/отрицательные значения — «нет таймаута», что опасно как дефолт для production. citeturn6view1turn6view2
- Для ограничения тела запроса применяйте `http.MaxBytesReader` на входе в обработчик или middleware, чтобы не дать клиенту «выжечь» память/CPU чрезмерным телом. citeturn4view0turn3search0
- Для дорогостоящих handler’ов используйте строгую модель «deadline сверху»: контекст запроса (`r.Context()`) + derive `context.WithTimeout` для внутренних операций, и обязателен корректный stop work при `ctx.Done()`; это соответствует модели отмены context в Go и интеграции с HTTP. citeturn23search2turn23search8

**JSON-интерфейсы: строгий парсинг и защита от “mass assignment”**
- Декодируйте JSON через `json.Decoder`, а не `json.Unmarshal` на неограниченном `io.ReadAll`, и включайте `Decoder.DisallowUnknownFields()` для DTO по умолчанию. По стандарту `encoding/json` неизвестные поля при декодировании в struct игнорируются, если явно не включить `DisallowUnknownFields`; это повышает риск “массового присваивания”/скрытого влияния на доменную модель и усложняет контроль схемы. citeturn8view0
- Учитывайте documented “Security Considerations” `encoding/json`: дубликаты ключей, case-insensitive сопоставление ключей в struct, игнор неизвестных ключей по умолчанию — всё это может иметь security-эффект при проверках/подписях/ACL, если разные компоненты парсят JSON по-разному. citeturn8view0
- Вывод JSON всегда через encoder (`json.NewEncoder(w).Encode(...)`) и с корректным `Content-Type`. Некорректный Content-Type может привести к неправильной интерпретации контента на клиенте и расширить поверхность XSS/MIME confusion. citeturn18view0

**Аутентификация и авторизация**
- Для каждого endpoint’а, который принимает объектный идентификатор (path/query/body) и выполняет действие над объектом, обязательна object-level authorization (типичный #1 риск для API). Это должно быть правилом для «каждого endpoint’а с ID». citeturn14search1
- Старайтесь проектировать DTO так, чтобы исключить неявное обновление “чувствительных полей” (property-level authorization / mass assignment), и вводить явные allowlist’ы записываемых полей. citeturn14search2turn14search4
- Authentication boundary должна быть очевидна (token/cookie/mTLS), иначе ошибки “Broken Authentication” становятся системными. citeturn14search0

**SQL/NoSQL доступ**
- Для SQL используйте parameterized queries и prepared statements; для `database/sql` аргументы `Exec/Query` предназначены для placeholder’ов. citeturn9search1turn9search4turn9search13
- В официальной документации по работе с БД для Go отдельно подчеркнуто: не собирайте SQL через `fmt.Sprintf` — это прямой путь к SQL injection. citeturn9search13
- Для NoSQL: не принимайте «сырые JSON-фрагменты» от клиента в качестве query/filter, запрещайте клиент-контролируемые операторы (например, `$where`, `$regex`, …) без строгой необходимости и валидации; применяйте allowlist ключей/операторов. citeturn9search5turn9search2

**Исходящие HTTP-запросы (SSRF и зависания)**
- Никогда не используйте “голый” `http.Client` без `Timeout`: `Timeout == 0` означает «нет таймаута». citeturn5view2
- Если сервис делает запросы по URL, составленному из недоверенного ввода, включайте SSRF-дефенсы: allowlist схем/хостов, блокировка private/loopback/link-local диапазонов, контроль редиректов, ограничения портов и egress policy на уровне инфраструктуры. citeturn0search1

**Файловая система, path traversal, загрузки**
- Для операций с потенциально attacker-controlled путями используйте `os.Root` / `os.OpenInRoot` (Go 1.24+) как более строгую и robust защиту от path traversal, включая попытки «выйти» через `..` и через symlink’и. citeturn12view0turn22search1
- Если модель угроз ограничена и attacker не контролирует локальную ФС, всё равно предпочтительны `filepath.IsLocal`/`filepath.Localize` для проверки «локальности» пути, а также `io/fs.ValidPath` для slash-separated путей при работе с `fs.FS`. citeturn12view0turn11search0
- Для загрузок файлов используйте defense-in-depth: allowlist расширений, не доверяйте `Content-Type`, генерируйте filename на стороне приложения, задавайте лимиты размера, храните вне webroot, и по возможности сканируйте/песочьте. citeturn13view0
- Будьте осторожны с multipart parsing: известны случаи чрезмерного потребления ресурсов при `mime/multipart.Reader.ReadForm` и связанных методах (`Request.ParseMultipartForm`, `FormFile`, …) при неудачной конфигурации лимитов. citeturn7search1turn3search4

**HTTP request smuggling**
- Наличие одновременно `Transfer-Encoding` и `Content-Length` запрещено нормой HTTP/1.1: sender MUST NOT посылать `Content-Length` при наличии `Transfer-Encoding`. В такой ситуации серверу разумно отвечать ошибкой и закрывать соединение, т.к. это также индикатор попытки smuggling/desync в цепочке proxy↔backend. citeturn10search0turn10search1

**TLS и криптография**
- По умолчанию минимальная версия TLS у серверов `crypto/tls` начиная с Go 1.22 — TLS 1.2 (если не задано явно), с возможностью отката через GODEBUG; в современных шаблонах это хороший дефолт. citeturn19search3turn19search11turn21view1
- `tls.Config.InsecureSkipVerify` делает TLS уязвимым к MITM, т.к. отключает проверку цепочки и hostname; допустим только для тестов или при корректной замене проверок через `VerifyConnection`/`VerifyPeerCertificate`. citeturn21view0
- Для криптографически стойких токенов/nonce используйте `crypto/rand`, а не `math/rand` (последний прямо помечен как непригодный для security-sensitive work). citeturn24search0turn24search1

**Безопасные HTTP-заголовки и неразглашение ошибок**
- Для API-варианта hardening: корректный `Content-Type` + `X-Content-Type-Options: nosniff`, минимизация disclosure заголовков (`Server`, `X-Powered-By`), и осознанное управление CORS. citeturn18view0
- По ошибкам: клиенту — generic response, детали — только в server-side логах; это базовый паттерн безопасной обработки ошибок. citeturn1search3turn1search18
- HSTS задаётся через `Strict-Transport-Security` и нормирован RFC; на практике его обычно выставляют на edge (ingress/gateway), но стандарт должен регламентировать ответственность и безопасную конфигурацию. citeturn19search1turn18view0

**Tooling как “security guardrails” в шаблоне**
- В CI обязательно: `govulncheck` как low-noise инструмент поиска известных уязвимостей зависимостей на основе анализа реальных вызовов в коде. citeturn7search2turn7search6turn7search9
- В CI/локально: `go vet` как часть стандартного инструментария (набор проверок “suspicious constructs”). citeturn15search1
- Рекомендуемо: `staticcheck` (находит баги/проблемы производительности/качества). citeturn15search19
- Дополнительно (trade-off: возможны FP): `gosec` как AST/SSA security-linter. citeturn15search2
- Для конкурентных дефектов: запуск тестов с race detector и соблюдение memory model; data race определён формально, и go tooling предоставляет детектор. citeturn16search0turn16search1turn16search3

## Матрица решений и trade-offs

Ниже — типовые спорные точки для шаблона (и что LLM должна уметь выбирать осознанно).

**Строгий JSON (DisallowUnknownFields) vs forward compatibility**
- За “строго”: снижает риск скрытых полей, неожиданных ключей, упрощает контроль DTO и audit. citeturn8view0turn14search4
- Против: ломает “мягкую эволюцию” контрактов (старые клиенты отправляют новые поля — сервер начнёт 400). Компромисс: включать строгость по умолчанию, а для «расширяемых» endpoint’ов — явно документировать и кодировать “allow unknown” или `map[string]json.RawMessage` под строгим контролем. citeturn8view0

**ReadTimeout/WriteTimeout на сервере vs streaming/long-poll**
- За: нулевые таймауты означают “no timeout”, что увеличивает риск slow HTTP/DoS. citeturn6view2turn23search3
- Против: для streaming (SSE, download большого файла) агрессивные `WriteTimeout` могут «убивать» легитимные ответы. Компромисс: для базового шаблона (JSON API) — ставить разумные таймауты, а для streaming endpoint’ов — выделять отдельный Server/route group с другими timeout’ами и это фиксировать в решении. citeturn6view1

**Reject TE+CL (request smuggling hardening) vs совместимость с “кривыми” клиентами**
- За: RFC запрещает сочетание, а OWASP WSTG описывает smuggling как класс проблем из-за разночтений в цепочке компонентов. citeturn10search0turn10search1
- Против: теоретически может сломать редких клиентов/посредников. Компромисс: в public edge — reject+close; во внутренних сетях — хотя бы detect+metric+alert, затем включать reject по мере готовности. citeturn10search1turn10search0

**os.Root/OpenInRoot (Go 1.24+) vs более старая версия Go**
- За: `os.Root` — специально созданный механизм против path traversal, включая symlink escape, и документирован как robust defense. citeturn12view0turn22search1
- Против: требует Go 1.24+; если по орг-причинам pinned старее, используйте `filepath.IsLocal/Localize` (для ограниченной threat model) и тщательно документируйте, что TOCTOU/symlink threats остаются. citeturn12view0

**gosec / security linters vs шум и ложные срабатывания**
- За: ловит распространённые security smells на уровне кода. citeturn15search2
- Против: возможны false positives и “lint fatigue”. Компромисс: keep `govulncheck` как MUST (официальный, low-noise), `gosec` как SHOULD с фиксированным набором правил и процессом suppressions (с обоснованием). citeturn7search6turn15search2

## Нормативные правила для LLM MUST / SHOULD / NEVER

Ниже — формулировки в стиле внутреннего LLM-instruction. Их цель: по умолчанию предотвращать ключевые классы уязвимостей (инъекции, SSRF, traversal, smuggling, auth bypass, resource exhaustion, disclosure), использовать безопасные API Go, и явно отмечать trade-offs там, где они есть.

**MUST**
- MUST считать любой вход из HTTP (path/query/header/body), а также данные из очередей/внешних сервисов недоверенными до валидации. citeturn1search0turn14search0
- MUST ограничивать ресурсы на входе: лимит тела запроса через `http.MaxBytesReader`, лимит заголовков через `Server.MaxHeaderBytes`, таймауты `ReadHeaderTimeout/ReadTimeout/WriteTimeout/IdleTimeout`. citeturn3search0turn6view1turn6view2
- MUST использовать контексты во всех I/O операциях (DB/HTTP/FS) и корректно прекращать работу по `ctx.Done()`. citeturn23search2turn23search8turn9search1
- MUST декодировать JSON по умолчанию строго: `json.Decoder` + `DisallowUnknownFields`, с явной обработкой ошибок и ограничением размера body до декодирования. citeturn8view0turn3search0
- MUST применять parameterized queries для SQL (`database/sql` args для placeholder’ов) и MUST NOT собирать SQL строковой конкатенацией/форматированием. citeturn9search1turn9search13turn1search1
- MUST для NoSQL запрещать выполнение “сырых” запросов, собранных из client-provided JSON, и MUST применять allowlist полей/операторов (отдельно защищаться от operator injection). citeturn9search5turn9search2
- MUST предотвращать SSRF при любых исходящих запросах по URL из недоверенного источника: allowlist схем/хостов, запрет private/loopback/link-local, контроль редиректов, обязательные таймауты на клиенте. citeturn0search1turn5view2
- MUST предотвращать path traversal при работе с attacker-controlled путями: использовать `os.OpenInRoot`/`os.Root` (если доступно), иначе `filepath.IsLocal/Localize` и строгую нормализацию/валидацию. citeturn12view0turn22search1
- MUST для загрузок файлов реализовывать defense-in-depth: allowlist расширений, не доверять `Content-Type`, генерировать filename, лимиты размера, хранение вне webroot. citeturn13view0turn23search3
- MUST не раскрывать внутренние ошибки клиенту: возвращать generic сообщение + корректный HTTP статус, а детали логировать на сервере. citeturn1search3turn1search18
- MUST выполнять object-level authorization на каждом endpoint’е с object ID, а также проектировать DTO так, чтобы избежать property-level authorization bypass/mass assignment. citeturn14search1turn14search2turn14search4
- MUST обрабатывать подозрительные HTTP-запросы, связанные с request smuggling, как ошибку и закрывать соединение (например, `Transfer-Encoding` вместе с `Content-Length`). citeturn10search0turn10search1
- MUST проверять зависимости через `govulncheck` в CI и фиксировать уязвимости согласно policy. citeturn7search6turn7search9
- MUST прогонять тесты с race detector хотя бы на CI для пакетов с конкуррентностью и следовать memory model (data races недопустимы). citeturn16search0turn16search1turn16search3

**SHOULD**
- SHOULD использовать `http.Server` вместо `http.ListenAndServe` «в одну строку», чтобы не забыть таймауты/лимиты. Логика “нулевые таймауты = нет таймаута” делает “быстрые демки” плохим шаблоном для production. citeturn6view2turn6view1
- SHOULD выставлять базовые security response headers для API (по применимости): `X-Content-Type-Options: nosniff`, корректный `Content-Type`, минимизация `Server`/`X-Powered-By`, аккуратный CORS. citeturn18view0
- SHOULD явно документировать ограничения и лимиты (max body, max upload, max page size, timeouts), т.к. отсутствие limit/quotas является типовым API-риском (resource consumption). citeturn14search5turn23search7
- SHOULD использовать `crypto/rand` для токенов/ID, влияющих на безопасность, и избегать `math/rand` для security-sensitive. citeturn24search0turn24search1
- SHOULD применять `go vet` и `staticcheck` в CI как “quality gates”, а `gosec` — как security gate при настроенном шуме. citeturn15search1turn15search19turn15search2
- SHOULD для HTML-рендеринга использовать `html/template` и избегать bypass-типов (`template.HTML`, `template.JS`, …) без доказуемой санации/контроля. citeturn17view0
- SHOULD использовать `tls.Config` с дефолтами современного Go (минимум TLS 1.2 по умолчанию) и избегать ручной криптографии без необходимости. citeturn21view1turn19search3

**NEVER**
- NEVER использовать `tls.Config.InsecureSkipVerify=true` в production «просто чтобы починилось». Это отключает проверку сертификата/hostname и делает TLS уязвимым к MITM. citeturn21view0
- NEVER собирать SQL через `fmt.Sprintf`/конкатенацию строк. citeturn9search13
- NEVER принимать “сырой” JSON от клиента как фильтр/запрос к NoSQL без строгого allowlist’а операторов/полей. citeturn9search5
- NEVER выполнять shell-команды через `sh -c`, `bash -c` или аналог, собирая строку из пользовательского ввода; по умолчанию избегать вызова ОС-команд как класса. citeturn2search1turn3search1
- NEVER возвращать клиенту stack trace/сырые `err.Error()` из внутренних зависимостей (DB/HTTP clients), если ошибка потенциально содержит детали инфраструктуры или секреты. citeturn1search3turn1search18
- NEVER делать исходящие HTTP-запросы через `http.Get`/`DefaultClient` в production-коде без явного `Timeout` и политики SSRF. citeturn5view2turn0search1
- NEVER использовать attacker-controlled путь в `filepath.Join(baseDir, userPath)` + `os.Open` как “защиту”; вместо этого применять `os.OpenInRoot`/`os.Root` или строгие проверки local-path. citeturn12view0
- NEVER использовать `text/template` для генерации HTML (только `html/template`). citeturn17view0
- NEVER игнорировать подозрительные комбинации `Transfer-Encoding`/`Content-Length` и другие признаки request smuggling в edge-facing сервисах. citeturn10search0turn10search1
- NEVER добавлять `unsafe` ради «микро-оптимизации» в шаблоне; `unsafe` предназначен для обхода type-safety и требует крайне аккуратного применения. citeturn3search2

## Примеры good / bad на Go по классам уязвимостей

### Строгий JSON parsing + input validation + ограничение ресурсов

**Bad: неограниченный body, silent ignore неизвестных полей, слабая обработка ошибок**
```go
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // игнорируем err, неизвестные поля silently ignored
	// ... дальше используем req как есть
}
```

**Good: MaxBytesReader + DisallowUnknownFields + явная валидация + корректные ошибки**
```go
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req CreateUserRequest
	if err := dec.Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}
	// Запрещаем JSON с "хвостом"
	if dec.More() {
		writeProblem(w, http.StatusBadRequest, "invalid_json", "Trailing data")
		return
	}

	if err := validateCreateUser(req); err != nil {
		writeProblem(w, http.StatusBadRequest, "validation_failed", "Invalid input")
		return
	}

	// ... бизнес-логика с r.Context()
}
```

Почему так: лимит тела предотвращает ресурсное истощение (`http.MaxBytesReader` явно предназначен для защиты от “accidentally or maliciously sending a large request”), а `DisallowUnknownFields` выключает default-поведение `encoding/json` “unknown keys ignored” и снижает риск неожиданных полей/массового присваивания. citeturn4view0turn3search0turn8view0turn1search0

### Error disclosure и безопасная обработка ошибок

**Bad: утечка деталей внутренностей**
```go
if err != nil {
	http.Error(w, err.Error(), http.StatusInternalServerError) // может раскрыть SQL, DSN, детали сети
	return
}
```

**Good: клиенту — минимум, серверу — детали**
```go
if err != nil {
	h.log.Error("create user failed", "err", err, "request_id", requestIDFrom(r.Context()))
	writeProblem(w, http.StatusInternalServerError, "internal_error", "Something went wrong")
	return
}
```

Паттерн “generic response + server-side logging” — целевое поведение безопасной обработки ошибок. citeturn1search3turn1search18

### SQL injection и безопасная параметризация, включая “опасные места” (ORDER BY)

**Bad: SQL через форматирование**
```go
q := fmt.Sprintf("SELECT id, email FROM users WHERE email = '%s'", email)
row := db.QueryRowContext(ctx, q)
```

**Good: parameterized query**
```go
row := db.QueryRowContext(ctx, "SELECT id, email FROM users WHERE email = $1", email)
```

**Bad: динамический ORDER BY из query-параметра**
```go
order := r.URL.Query().Get("order") // "email desc; drop table users; --"
q := "SELECT id, email FROM users ORDER BY " + order
rows, _ := db.QueryContext(ctx, q)
```

**Good: allowlist для “непараметризуемых” фрагментов**
```go
order := r.URL.Query().Get("order")
col := "id"
switch order {
case "id", "":
	col = "id"
case "email":
	col = "email"
default:
	writeProblem(w, http.StatusBadRequest, "validation_failed", "Invalid order")
	return
}

rows, err := db.QueryContext(ctx,
	"SELECT id, email FROM users ORDER BY "+col+" LIMIT $1",
	100,
)
if err != nil { /* ... */ }
```

Официальная Go-документация прямо предупреждает не использовать string formatting (`fmt.Sprintf`) для сборки SQL из-за риска SQL injection; а `database/sql` описывает `args` как параметры для placeholder’ов. citeturn9search13turn9search1turn1search1

### NoSQL injection (operator injection) — общий шаблон защиты

**Bad: принимаем filter как `map[string]any` “как есть”**
```go
var filter map[string]any
_ = json.NewDecoder(r.Body).Decode(&filter)
// затем передаём filter в драйвер как query
```

**Good: строго типизированный DTO + запрет операторов/лишних ключей**
```go
type UserSearch struct {
	Email string `json:"email"`
}

dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
dec.DisallowUnknownFields()

var req UserSearch
if err := dec.Decode(&req); err != nil { /* ... */ }

if !isValidEmail(req.Email) { /* ... */ }

// query строится из контролируемых полей, без клиентских операторов
filter := map[string]any{"email": req.Email}
```

Идея: “Do not accept raw JSON fragments from the client to execute as queries” и “Disallow client-controlled query operators … unless strictly required and validated.” citeturn9search5turn1search0

### SSRF: безопасный outbound HTTP client + allowlist + контроль редиректов

**Bad: SSRF через прямой fetch**
```go
u := r.URL.Query().Get("url")
resp, err := http.Get(u) // и без таймаута
```

**Good: выделенный http.Client с Timeout + запрет редиректов + allowlist хостов**
```go
var httpClient = &http.Client{
	Timeout: 5 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse // не следуем редиректам автоматически
	},
}

func (h *Handler) Fetch(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("url")

	uu, err := url.Parse(raw)
	if err != nil || (uu.Scheme != "https" && uu.Scheme != "http") {
		writeProblem(w, http.StatusBadRequest, "validation_failed", "Invalid URL")
		return
	}

	host := strings.ToLower(uu.Hostname())
	if !allowedHost(host) {
		writeProblem(w, http.StatusBadRequest, "validation_failed", "Host not allowed")
		return
	}

	req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, uu.String(), nil)
	resp, err := httpClient.Do(req)
	if err != nil { /* ... */ }
	defer resp.Body.Close()
	// ...
}
```

Почему так: OWASP подчёркивает необходимость защит от SSRF, а `net/http` документирует, что `Client.Timeout == 0` означает “no timeout”, что опасно в production. citeturn0search1turn5view2

### Path traversal: `os.OpenInRoot` / `os.Root` вместо `filepath.Join(base, userPath)`

**Bad: классический traversal через Join**
```go
f, err := os.Open(filepath.Join(baseDir, userPath))
```

**Good: directory-limited API**
```go
f, err := os.OpenInRoot(baseDir, userPath)
if err != nil { /* ... */ }
defer f.Close()
```

Обоснование: Go security guidance по `os.Root` прямо описывает паттерн “filepath.Join(fixedDir, externally-provided filename)” как подозрительный, и показывает `OpenInRoot` как корректную защиту от выхода из каталога через `..` и symlink. citeturn12view0turn22search1

### File upload: лимиты, безопасные имена, хранение, DoS защита

**Bad: “просто сохраним файл как пришёл”**
```go
_ = r.ParseMultipartForm(32 << 20)
file, header, _ := r.FormFile("file")
defer file.Close()

dst, _ := os.Create(filepath.Join(uploadDir, header.Filename)) // traversal, спец-имена, overwrite
defer dst.Close()
io.Copy(dst, file) // без лимитов
```

**Good: лимит на HTTP body + безопасное имя + запись с ограничением**
```go
const maxUpload = 10 << 20 // 10 MiB
r.Body = http.MaxBytesReader(w, r.Body, maxUpload)

mr, err := r.MultipartReader()
if err != nil {
	writeProblem(w, http.StatusBadRequest, "invalid_multipart", "Invalid multipart data")
	return
}

for {
	part, err := mr.NextPart()
	if err == io.EOF {
		break
	}
	if err != nil { /* ... */ }

	if part.FormName() != "file" {
		continue
	}

	// Генерируем имя сами, original name не используем как путь/имя файла
	safeName := uuid.NewString() + ".bin" // расширение — по allowlist/контенту

	// Храним вне webroot; если нужно — используем OpenInRoot для гарантии
	dst, err := os.OpenInRoot(uploadDir, safeName) // или root.Create(...)
	if err != nil { /* ... */ }
	defer dst.Close()

	// Пишем ограниченно (доп. страховка к MaxBytesReader)
	if _, err := io.Copy(dst, io.LimitReader(part, maxUpload)); err != nil { /* ... */ }
}
```

OWASP File Upload Cheat Sheet рекомендует allowlist расширений, не доверять `Content-Type`, генерировать имена на стороне приложения, задавать лимиты размера, хранить вне webroot и использовать defense-in-depth. Дополнительно, в экосистеме Go документированы DoS-риски чрезмерного потребления ресурсов при multipart parsing при неправильной конфигурации. citeturn13view0turn4view0turn7search1turn12view0

### Request smuggling: reject+close для TE+CL

```go
func rejectSmuggling(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		te := r.Header.Get("Transfer-Encoding")
		if te != "" && r.Header.Get("Content-Length") != "" {
			w.Header().Set("Connection", "close")
			writeProblem(w, http.StatusBadRequest, "bad_request", "Invalid framing")
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

Норма HTTP/1.1 (RFC 9112) запрещает отправителю сочетать `Transfer-Encoding` и `Content-Length`, а OWASP WSTG описывает request smuggling как класс проблем из-за разночтений во frontend/backend parsing. citeturn10search0turn10search1

### Command injection: избегать OS-команд; если неизбежно — без shell и с allowlist

**Bad: shell-команда из пользовательского ввода**
```go
cmd := exec.Command("sh", "-c", "ping -c 1 "+host)
out, _ := cmd.CombinedOutput()
```

**Good: минимизация поверхности + не использовать shell**
```go
if !isAllowedHostForPing(host) {
	writeProblem(w, http.StatusBadRequest, "validation_failed", "Invalid host")
	return
}

ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
defer cancel()

cmd := exec.CommandContext(ctx, "ping", "-c", "1", host)
out, err := cmd.CombinedOutput()
if err != nil { /* ... */ }
```

OWASP рекомендует “avoid calling OS commands directly” как primary defense; `os/exec` — API выполнения внешних команд и требует осознанного безопасного использования. citeturn2search1turn3search1

### Template escaping и output encoding

**Bad: рендер HTML через `text/template` или принудительный bypass escaping**
```go
t := texttemplate.Must(texttemplate.New("x").Parse(`<div>{{.}}</div>`))
_ = t.Execute(w, userInput) // потенциальный XSS
```

**Good: `html/template` авто-экранирует в контексте**
```go
t := template.Must(template.New("x").Parse(`<div>{{.}}</div>`))
_ = t.Execute(w, userInput)
```

`html/template` описан как пакет для генерации HTML “safe against code injection”, делает контекстный escaping, а bypass-типы (`template.HTML`, `template.JS`, …) исключаются из escaping и требуют доверенного источника/санации. citeturn17view0turn2search0

### Secure HTTP defaults: пример server-конфигурации “по шаблону”

```go
srv := &http.Server{
	Addr:              ":8080",
	Handler:           handler,
	ReadHeaderTimeout: 5 * time.Second,
	ReadTimeout:       15 * time.Second,
	WriteTimeout:      15 * time.Second,
	IdleTimeout:       60 * time.Second,
	MaxHeaderBytes:    1 << 20, // 1 MiB
}
log.Fatal(srv.ListenAndServe())
```

`net/http` документирует семантику timeout’ов как “0/negative => no timeout”, и разделяет `ReadHeaderTimeout`/`ReadTimeout` как разные инструменты контроля медленных/больших запросов. citeturn6view1turn6view2

## Anti-patterns и типичные ошибки или hallucinations LLM

Ниже — набор “часто встречаемых” способностей LLM ошибаться при генерации Go-кода для API; их стоит прямо закрепить как запреты/алерты в LLM-instruction и включить в ревью-лист.

LLM часто:
- **забывает лимиты и таймауты**: генерирует `http.ListenAndServe(...)`, `http.Get(...)` без таймаутов и лимитов тела. В Go `Client.Timeout == 0` означает no-timeout, а у `http.Server` 0/negative также no-timeout — это «production footgun». citeturn5view2turn6view2
- **делает SQL через конкатенацию/форматирование**, включая “сложные кейсы” вроде ORDER BY / IN / LIKE. Это прямо запрещено рекомендациями по Go database docs из-за SQL injection. citeturn9search13turn1search1
- **декодирует JSON “мягко”** и пропускает validation: `json.NewDecoder(r.Body).Decode(&dto)` без `DisallowUnknownFields`, без лимита body. По `encoding/json` это означает игнор unknown keys по умолчанию. citeturn8view0turn4view0
- **подменяет object-level auth на “один раз проверили токен”**: пропускает проверку прав на конкретный объект, хотя OWASP подчёркивает необходимость object-level authorization для каждого endpoint’а с ID. citeturn14search1
- **делает SSRF-дырки**: скачивает URL из query/body без allowlist/запрета внутреннего адресного пространства и без политики редиректов, хотя OWASP SSRF guidance описывает конкретные меры. citeturn0search1
- **делает path traversal**: использует `filepath.Join(baseDir, userPath)` как “магическую защиту”, хотя path traversal и TOCTOU/symlink риски требуют более строгих механизмов; Go рекомендует `os.OpenInRoot/os.Root` как robust вариант. citeturn12view0turn22search1
- **допускает опасные TLS “чтобы заработало”**: `InsecureSkipVerify: true` как «быстрое решение». Документация `crypto/tls` прямо предупреждает о MITM и ограничивает применение тестами/кастомной проверкой. citeturn21view0
- **маскирует проблему request smuggling**: игнорирует конфликтующие заголовки TE+CL. RFC запрещает такой фрейминг, OWASP WSTG описывает риск десинхронизации между прокси и бэкендом. citeturn10search0turn10search1
- **использует `math/rand` для токенов/секретов**, хотя пакет прямо помечен как непригодный для security-sensitive work; нужен `crypto/rand`. citeturn24search1turn24search0
- **создаёт XSS через шаблоны**: `text/template` для HTML или использование bypass-типов `html/template` без санации. citeturn17view0

## Review checklist и разбиение на файлы в template repo

### Review checklist для PR / code review

Checklist сформулирован так, чтобы его можно было использовать как `docs/review/security-checklist.md` и как шаблон секции в PR template.

**Входные данные и границы доверия**
- Все входы (path/query/header/body) валидированы allowlist’ами, типами, диапазонами, длинами; нет “blacklist-only” фильтров. citeturn1search0
- Для JSON: есть лимит body (`MaxBytesReader`), `Decoder.DisallowUnknownFields` (или документированное исключение), корректная обработка decode-ошибок и trailing данных. citeturn4view0turn8view0

**HTTP server hardening**
- `http.Server` создан явно, выставлены `ReadHeaderTimeout/ReadTimeout/WriteTimeout/IdleTimeout/MaxHeaderBytes` и они соответствуют классу endpoint’ов (не ломают streaming — если есть). citeturn6view1turn6view2
- Есть защита от resource exhaustion (лимиты размеров, timeouts, ограничения дорогих операций); отсутствие лимитов — известная категория API риска. citeturn14search5turn23search3

**AuthN/AuthZ**
- Authentication проверен корректно и последовательно (нет “optional auth” без причины). citeturn14search0
- На каждом endpoint’е с object ID выполнен object-level authorization. citeturn14search1
- Нет property-level bypass: DTO/patch/update реализованы через allowlist обновляемых полей. citeturn14search2turn14search4

**Инъекции**
- SQL: только параметризованные запросы, нет fmt.Sprintf/конкатенации SQL. citeturn9search13turn9search1
- NoSQL: нет выполнения client-provided JSON как query, запрещены (или жёстко валидированы) операторы. citeturn9search5
- OS commands: по умолчанию отсутствуют; если есть — нет shell, есть allowlist, timeouts, контекст. citeturn2search1turn3search1

**SSRF и исходящие запросы**
- Исходящие HTTP: нет `http.Get`/DefaultClient без timeout; на клиенте выставлен `Timeout`, редиректы контролируются, есть SSRF-политика. citeturn5view2turn0search1

**Файлы и пути**
- Path traversal: attacker-controlled пути обрабатываются через `os.OpenInRoot/os.Root` или строгую валидацию local-path; нет “Join(base, userPath) и надеемся”. citeturn12view0turn22search1
- Upload: соблюдены базовые рекомендации OWASP (allowlist расширений, не доверять Content-Type, safe filename, size limits, storage вне webroot, авторизация на upload). citeturn13view0
- Multipart parsing/архивы: лимиты выставлены осознанно; нет пути к “unbounded resource consumption”. citeturn7search1turn14search5

**Протокол и заголовки**
- Есть минимальная защита от request smuggling (reject TE+CL + close) в edge-facing сервисе или документированная точка ответственности на gateway. citeturn10search0turn10search1
- Response headers: корректный Content-Type, `X-Content-Type-Options: nosniff`, минимизация информационных заголовков; CORS настроен намеренно. citeturn18view0

**Ошибки и логирование**
- Клиенту не возвращаются внутренние детали; логи содержат детали и корреляцию. citeturn1search3turn1search18

**Инструменты и CI**
- PR проходит `govulncheck` (обяз.), `go vet` (обяз.), тесты с `-race` (обяз. там, где есть конкуррентность), плюс линтеры по политике (`staticcheck`, возможно `gosec`). citeturn7search6turn15search1turn16search0turn15search19turn15search2

### Что вынести в отдельные файлы в template repo

Ниже — разбиение “почти готовое к docs/ и repo conventions”, с акцентом на безопасность и LLM-guardrails.

- `docs/security/secure-coding-standard.md`  
  Нормативный стандарт (по сути разделы MUST/SHOULD/NEVER + решения по умолчанию + исключения и процесс их согласования). Основание: OWASP cheat sheets, Go stdlib docs, RFC 9112, Go release notes. citeturn1search4turn6view1turn10search0turn22search0

- `docs/security/threat-model-assumptions.md`  
  Явные допущения: внешний ввод недоверенный, обязательность object-level auth, запрет сырых NoSQL queries, SSRF-модель, правила для файлов/путей. citeturn14search1turn9search5turn0search1turn12view0

- `docs/llm/secure-coding-instructions.md`  
  Компактная версия MUST/SHOULD/NEVER для вставки в system prompt / repo-level LLM инструкции. Включить “stop conditions”: если требуется нарушить правило, LLM обязана явно пометить риск и предложить безопасную альтернативу. citeturn21view0turn9search13turn5view2

- `docs/llm/codegen-checklist-security.md`  
  “Перед тем как сгенерировать PR”: короткий чеклист (timeouts/limits, DisallowUnknownFields, parameterized SQL, SSRF policy, traversal-safe FS, error disclosure). citeturn6view2turn8view0turn0search1turn12view0turn1search3

- `internal/httpx/` (или аналог)  
  Пакет-обвязка для безопасных defaults:
  - создание `http.Server` с timeouts/MaxHeaderBytes; citeturn6view1turn6view2  
  - middleware: `MaxBytesReader`, security headers, request smuggling guard (TE+CL), correlation id; citeturn4view0turn10search0turn18view0  
  - helpers для JSON decode/encode + problem response. citeturn8view0turn18view0

- `internal/dbx/`  
  Обёртки/пример репозитория с parameterized queries, запретом fmt.Sprintf и шаблонами allowlist для динамических частей. citeturn9search13turn9search1turn9search4

- `internal/ssrf/`  
  Библиотечка политик SSRF (allowlist hosts/schemes, redirect policy, IP blocking), чтобы LLM не изобретала «каждый раз по-разному». citeturn0search1turn5view2

- `internal/fsx/`  
  Пакет для безопасной работы с путями: `OpenInRoot`/`os.Root` и helpers `filepath.IsLocal/Localize` + документирование threat model. citeturn12view0turn22search1

- `docs/security/file-upload.md`  
  Правила upload/download: allowlist расширений, сигнатуры, переименование, лимиты, хранение, сканирование. citeturn13view0turn7search1

- `docs/security/http-hardening.md`  
  Таймауты, лимиты, request smuggling, security headers, HSTS responsibility (обычно edge), error handling. citeturn6view1turn10search0turn18view0turn19search1turn1search3

- `Makefile` / `Taskfile` / `scripts/`  
  Команды: `govulncheck ./...`, `go vet ./...`, `staticcheck ./...`, `gosec ./...`, `go test -race ./...` (с оговорками по платформам). citeturn7search6turn15search1turn15search19turn15search2turn16search0

- CI workflow (например, `.github/workflows/ci.yml`, если платформа — GitHub)  
  Gates: тесты, race, govulncheck, vet, линтеры; политика “fail on findings” минимум для govulncheck/vet. citeturn7search9turn15search1