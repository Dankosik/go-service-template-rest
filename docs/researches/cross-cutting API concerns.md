# Production-ready Go микросервис template: engineering standard и LLM-instructions для репозитория

## Scope

Этот подход применим, когда вы создаёте **greenfield** сервис, который должен быстро стать “нормальным продакшеном”: предсказуемые контракты (HTTP и/или gRPC), наблюдаемость, безопасные дефолты, CI-гейты, и единая архитектура, которую удобно расширять с помощью LLM-инструментов. Цель — чтобы клон репозитория давал “рабочее место инженера”: каждый новый эндпоинт автоматически получает базовые гарантии (timeout’ы, лимиты, корреляцию, структурные логи, метрики/трейсы, единый error model), а LLM не пришлось “угадывать” архитектуру и договорённости. citeturn22view0turn16view0turn5search1turn4search3

Подход особенно подходит для сервисов, которые:
- деплоятся в контейнерах и оркестраторах (Kubernetes-подобные среды), где важны readiness/liveness/startup сигналы и graceful shutdown. citeturn2search2turn2search6turn16view2  
- используют политику “boring defaults”: меньше фреймворков, больше стандартных механизмов языка/экосистемы, понятных как людям, так и LLM. citeturn14search0turn4search0turn0search1  
- должны быть защищены от типовых API-рисков: broken auth, BOLA, misconfiguration, отсутствие лимитов и валидации, нестрогость error model. citeturn2search0turn13search0turn13search2

Не применяйте этот подход “как есть”, если:
- вы пишете **библиотеку** (не сервис): многие правила (health endpoints, middleware, observability bootstrap, контейнеризация) будут лишними и могут создавать шум. citeturn22view0  
- ваш сервис — это “edge gateway”/API gateway с сложными policy-пайплайнами, где routing/ratelimiting/auth сильно отличаются и часто нужны специализированные решения (Envoy, API gateways и т.п.); этот template всё ещё полезен как стиль и гигиена, но не как финальная архитектура. citeturn1search6turn1search0  
- у вас есть жёстко заданный корпоративный стек (например, обязательный фреймворк, централизованный auth SDK, готовая платформа), и отступления запрещены; тогда этот документ нужно “мэппить” на корпоративные аналоги, а не копировать. citeturn2search1turn0search9

## Recommended defaults для greenfield template

Ниже — дефолты, ориентированные на Go 1.26 и “battle-tested” практики. Сам принцип “фиксируем версию и правила” снижает пространство догадок для LLM и уменьшает рассинхрон между локальной разработкой и CI. citeturn0search7turn0search3turn24view1

**Версия Go и toolchain**
- Зафиксируйте минимум: `go 1.26` в `go.mod`. Начиная с Go 1.21 `go` directive становится обязательным требованием (toolchain откажется работать с модулем, требующим более новую версию). citeturn24view1turn0search3turn0search7  
- Рекомендуется добавить `toolchain go1.26.0` (или актуальный patch) для воспроизводимости и предсказуемого поведения сборки, особенно в CI. citeturn11search0turn11search7turn24view2  

**Модуль и layout репозитория**
- Один модуль на репозиторий — дефолт. Мульти-модульный репозиторий допустим, но обычно усложняет жизнь и требует отдельной дисциплины. citeturn21search2turn4search1  
- Используйте официальный рекомендуемый layout для server project: `cmd/<service>/main.go` + большинство логики в `internal/...`, чтобы облегчить рефакторинг и не поддерживать “публичный API” случайно. citeturn22view0  

**HTTP стек (boring default)**
- Для HTTP routing используйте стандартный `net/http` и обновлённый `http.ServeMux` (Go 1.22+): method-matching и wildcards уменьшают потребность в сторонних роутерах и делают шаблон более “stdlib-first”. citeturn14search0turn14search1turn15view0  
- Явно настройте `http.Server` таймауты и лимиты заголовков:
  - `ReadHeaderTimeout` — как базовый защиты от медленного чтения заголовков; `ReadTimeout` можно использовать дополнительно, но документация прямо отмечает, что многие предпочитают `ReadHeaderTimeout`, потому что handler может жить с собственными дедлайнами на body/upload. citeturn16view0turn16view1  
  - `IdleTimeout`, `WriteTimeout`, `MaxHeaderBytes` (или оставить дефолт и документировать). citeturn16view0turn15view1  
- Ограничьте размер body через `http.MaxBytesReader` (и фиксируйте, как именно сервис отвечает на превышение лимита). Это прямой механизм `net/http` для защиты ресурса сервера от слишком больших тел запросов. citeturn16view3  
- Реализуйте graceful shutdown через `Server.Shutdown(ctx)` и **дожидайтесь** завершения shutdown перед выходом процесса; `Shutdown` не закрывает hijacked соединения (например WebSocket), это нужно учитывать. citeturn16view2  

**gRPC дефолты**
- Если сервис предоставляет/потребляет gRPC: сделайте обязательными дедлайны на клиентах и корректную propagation. gRPC гайд подчёркивает, что дедлайны конвертируются в timeout, чтобы нивелировать clock skew. citeturn3search3  
- Все cross-cutting concerns делайте через interceptors (серверные/клиентские). Это штатный механизм gRPC. citeturn8search1  
- Используйте стандартный gRPC health checking protocol (`grpc.health.v1`). citeturn5search0turn5search12  
- Для передачи auth/trace/корреляции используйте gRPC metadata как “side channel”. citeturn8search10turn8search6  

**Observability дефолты**
- Логи: `log/slog` как стандартная структурная база (key-value). Это снижает необходимость “выбирать zelolog/zap/logrus” в шаблоне и даёт единый стиль. citeturn4search3turn4search7  
- Трейсы/контекст: ориентируйтесь на entity["organization","OpenTelemetry","observability spec project"]. Документация подчёркивает, что дефолтный propagator использует заголовки стандарта W3C TraceContext. citeturn2search3turn8search0turn5search1  
- Метрики: базовый язык — Prometheus exposition, с соблюдением naming/label-guidelines (bounded cardinality, не “кодировать label в имя метрики”). citeturn5search2turn8search3turn8search7  

**Безопасность и стандартные риски API**
- За основу “что может пойти не так” используйте entity["organization","OWASP","web security org"] API Security Top 10 (2023) и ASVS как контрольную рамку для требований. citeturn2search0turn2search1turn2search9  
- Валидация входа должна происходить как можно раньше по потоку данных (идеально — сразу на границе API). Это прямое правило из OWASP Input Validation guidance. citeturn13search0  
- Для auth на HTTP — Bearer tokens по OAuth 2.0 (RFC 6750), а идентификацию в enterprise-SSO обычно строят через OpenID Connect. citeturn3search0turn3search2turn13search7  
- JWT — формат, определённый RFC 7519; при использовании JWT делайте отдельный раздел “security considerations”, потому что ошибки интеграции JWT типовые и дорогостоящие. citeturn3search1turn13search2  

**Error model и контракты**
- Для HTTP ошибок используйте стандарт Problem Details (RFC 9457, `application/problem+json`) как единый формат. RFC 9457 прямо говорит, что документ определяет problem details и обсо́летит RFC 7807. citeturn1search0turn1search12  
- Для gRPC ошибок используйте canonical gRPC status codes, и при необходимости rich details — нормируйте через `google.rpc.Status` (Google AIP-193 делает это MUST для API errors в Google-style). citeturn17search3turn6search3turn6search18  

**Contract toolchain для proto/OpenAPI**
- HTTP: OpenAPI 3.1 как главный источник спецификации для внешнего HTTP-контракта (OAS определяет стандартное описания HTTP API). citeturn6search2  
- Protobuf/gRPC: для формата/линтинга/брейкинг-чеков используйте entity["company","Buf","protobuf tooling vendor"] CLI: он включает formatter/linter/breaking change detector и помогает сделать API-эволюцию механически проверяемой. citeturn12search23turn12search15  
- Валидация protobuf: `protoc-gen-validate` в maintenance mode, а для новых проектов рекомендуют перейти на `protovalidate` (оф. позиция проекта). citeturn6search8turn6search5turn6search0  

**Tooling в CI (минимальный обязательный набор)**
- `gofmt` обязателен как единый форматтер исходников. citeturn10search0  
- `go vet` как базовый статический анализ (подозрительные конструкции, printf-mismatch и т.п.). citeturn10search1  
- `go test` как базовый уровень покрытия корректности. citeturn10search2  
- `govulncheck` как официальный инструмент Go vulnerability management: он умеет снижать шум, сопоставляя уязвимости с реально вызываемыми функциями. citeturn0search17turn0search9turn0search6  

**Code style и review-нормы**
- Базовые правила идиоматики: Effective Go + Go Code Review Comments. Это первичные источники, которым можно ссылаться как на “норматив”. citeturn4search0turn0search1turn4search8  
- Если вам нужен более подробный договор по стилю: Google Go Style Guide помечен как “normative and canonical” внутри Google. citeturn10search7turn10search3  

## Decision matrix и trade-offs

Матрица ниже нужна не ради “прекрасных альтернатив”, а чтобы зафиксировать, **что является дефолтом**, и что считается допустимым отклонением — с явными последствиями для поддержки LLM и людей.

| Область | Дефолт | Альтернатива | Trade-off / когда менять |
|---|---|---|---|
| Роутинг HTTP | `net/http` `ServeMux` (Go 1.22+ patterns) | Сторонний router | Stdlib снижает зависимость и пространство догадок для LLM; сторонний router может дать больше middleware-экосистемы и привычный DX, но увеличивает entropy шаблона. citeturn14search0turn15view0turn15view1 |
| Ошибки HTTP | RFC 9457 Problem Details | Custom JSON error | RFC-формат повышает интероперабельность и консистентность; custom формат легко “расползается” по сервису и клиентам. citeturn1search0turn1search12 |
| Ошибки gRPC | canonical status codes + (опц.) `google.rpc.Status` | “любой текст в message” | Canonical codes позволяют клиентам иметь предсказуемые стратегии; AIP-193 формализует richer error model. citeturn17search3turn6search3turn6search18 |
| Protobuf валидация | protovalidate | protoc-gen-validate | PGV поддерживается, но для новых/существующих проектов рекомендуют переходить на protovalidate (официально). citeturn6search8turn6search5turn6search0 |
| Retry и идемпотентность | Явная классификация retry-safe + idempotency keys для unsafe операций | “клиент сам разберётся” | Без классификации ретраи ломают деньги/платежи/дубли; AIP-194 и AIP-155 дают практическую норму, особенно для RPC. citeturn12search1turn12search5turn1search3 |
| Rate limiting семантика | 429 + Retry-After; документированный quota policy | Самодельные X-RateLimit-* без spec | 429 стандартизирован (RFC 6585), Retry-After описан в HTTP semantics; IETF draft по RateLimit headers полезен, но это draft — помечайте как “не RFC”. citeturn1search6turn20view1turn1search5turn9search7 |
| Async API | 202 Accepted + operation resource / LRO pattern | “долго висим синхронно” | 202 прямо описывает асинхронность и ограничение HTTP (невозможно “дослать статус”); AIP-151 даёт дизайн LRO. citeturn23view0turn12search0 |
| Observability | OpenTelemetry + W3C TraceContext | “только логи” | Без трейсинга сложно чинить распределённые задержки; OTEL по умолчанию использует W3C TraceContext. citeturn2search3turn8search0turn5search1 |

## Набор правил MUST / SHOULD / NEVER для LLM

Ниже — формулировки, которые можно почти напрямую положить в `docs/llm-instructions.md` и использовать как “system prompt / repo rules”. Они специально написаны так, чтобы модель не “догадывалась”, а действовала в рамках репозитория.

### MUST

1) **Сначала читать репозиторий, потом писать код**
- MUST прочитать: `README`, `docs/`, существующие package API, `go.mod`, существующие middleware/interceptors, договор по error model и API contracts. citeturn24view0turn22view0turn4search1  
- MUST использовать существующие типы ошибок/логгер/метрики и не изобретать параллельные. citeturn4search3turn1search0turn17search3  

2) **Идиоматика Go как обязательный baseline**
- MUST соответствовать Effective Go и Go Code Review Comments (именование, ошибки, комментарии, минимизация интерфейсов, простота). citeturn4search0turn0search1  
- MUST форматировать код `gofmt` и не “ручной стиль”. citeturn10search0  

3) **Контекст, дедлайны, отмена**
- MUST принимать `context.Context` первым параметром в публичных API (внутри сервиса) и прокидывать его во все IO-вызовы (БД, HTTP, gRPC), чтобы отмена освобождала ресурсы. citeturn7search1turn7search18turn3search3  
- MUST не создавать `context.Background()` внутри request-handling, кроме как в bootstrap (main) или в “fire-and-forget” с явным ADR/комментарием. citeturn7search1turn16view2  

4) **HTTP безопасность по умолчанию**
- MUST включать server timeouts (`ReadHeaderTimeout` как минимум) и лимиты заголовков. citeturn16view0turn15view1  
- MUST ограничивать размер request body через `http.MaxBytesReader` и возвращать корректный статус (обычно 413). citeturn16view3turn20view1  

5) **Валидация и auth на границе**
- MUST валидировать входные данные максимально рано (в handler/middleware или interceptor) и не допускать “полуі-валидных” объектов в бизнес-слой. citeturn13search0  
- MUST явно формировать auth context (principal, scopes/roles/tenant, correlation ids) и использовать его для авторизации, включая object-level checks (BOLA — топ-рисковый класс). citeturn2search0turn13search12turn8search10  

6) **Единый error model**
- MUST для HTTP использовать RFC 9457 problem details (единый контейнер ошибок + расширения), не “разные JSON-форматы на каждом endpoint”. citeturn1search0turn1search12  
- MUST для gRPC возвращать корректные status codes, избегая `UNKNOWN/INTERNAL`, когда существует более точная классификация. citeturn17search3  

7) **Observability обязательна**
- MUST логировать структурно через `log/slog`. citeturn4search3turn4search7  
- MUST сохранять trace context и поддерживать propagation по W3C TraceContext (HTTP) и эквивалентный контекст в gRPC metadata. citeturn8search0turn2search3turn8search10  
- MUST придерживаться правил Prometheus по метрикам/лейблам (bounded labels, не генерировать куски имени процедурно). citeturn8search7turn8search3turn5search2  

8) **Security hygiene**
- MUST не логировать секреты/токены/PII; следовать security-logging guidance. citeturn13search1turn3search0  
- MUST использовать официальные стандарты для токенов/протоколов (RFC 6750 для Bearer, RFC 7519 для JWT, OIDC Core для аутентификации), а не “самодельные схемы”. citeturn3search0turn3search1turn3search2  
- MUST запускать `go vet` и `govulncheck` (или гарантировать, что CI их запускает) при внесении изменений зависимостей/критичных участков. citeturn10search1turn0search17turn0search9  

### SHOULD

- SHOULD минимизировать новые зависимости; если зависимость добавляется, то нужны: мотивация, альтернатива (stdlib), лицензия/поддержка, план обновлений, и влияние на LLM-ergonomics (чёткие примеры/README). citeturn4search1turn11search2  
- SHOULD фиксировать контрактные решения (error model, idempotency semantics, лимиты) в OpenAPI/proto и в `docs/api-contracts.md`, чтобы контракт был единственным источником правды. citeturn6search2turn12search23turn1search0  
- SHOULD для gRPC/proto использовать Buf lint + breaking change detection как “механический” контроль API-эволюции. citeturn12search15turn12search3turn12search7  
- SHOULD держать конфиг в environment (12-factor). citeturn5search3  

### NEVER

- NEVER “тихо” игнорировать ошибки, особенно от IO, parsing, crypto, auth, БД. citeturn4search0turn0search1  
- NEVER писать новый формат ошибок для HTTP вместо RFC 9457 (кроме случаев, когда есть формализованное исключение/ADR). citeturn1search0  
- NEVER зависеть от нефиксированных версий инструментов, если репозиторий уже фиксирует `go`/`toolchain`; не “обновлять всё подряд” без причины. citeturn24view2turn11search0  
- NEVER делать валидацию “где-то в глубине бизнес-слоя”, позволяя грязным данным гулять по domain. citeturn13search0  
- NEVER вводить высококардинальные лейблы (например `user_id`) в Prometheus-метрики. citeturn8search7turn8search3  

### Concrete good / bad examples (Go)

**Good: HTTP handler с ограничением размера, строгим JSON, валидацией, и единым error model**

```go
func (h *Handler) CreateWidget(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1) Limits: protect server resources.
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxRequestBodyBytes)
	defer r.Body.Close()

	// 2) Parse строго: неизвестные поля запрещены (контракт не должен молча расширяться).
	var req CreateWidgetRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeProblem(w, problemInvalidJSON(err))
		return
	}

	// 3) Валидация на границе.
	if err := req.Validate(); err != nil {
		writeProblem(w, problemValidation(err))
		return
	}

	// 4) Бизнес-операция с ctx (timeouts/cancel propagation).
	widget, err := h.svc.CreateWidget(ctx, req)
	if err != nil {
		writeProblem(w, mapDomainError(err))
		return
	}

	writeJSON(w, http.StatusCreated, widget)
}
```

**Bad: “типичная LLM-галлюцинация”**
- нет лимита body,
- `context.Background()` внутри запроса,
- игнор ошибок декодирования,
- возврат произвольного JSON-ошибочного объекта без стандарта.

```go
func CreateWidget(w http.ResponseWriter, r *http.Request) {
	var req CreateWidgetRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // игнор ошибок

	ctx := context.Background() // теряем cancel/deadline
	widget, _ := service.CreateWidget(ctx, req) // игнор ошибок

	w.WriteHeader(500) // бессмысленный статус
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": "something went wrong",
	})
}
```

## Anti-patterns и типичные ошибки/hallucinations LLM

Ниже — список ошибок, которые чаще всего появляются при LLM-кодогенерации в Go-шаблонах. Это полезно держать отдельным разделом в `docs/llm-hallucinations.md`, потому что “память модели” обычно обобщает по разным репозиториям и тащит неподходящие паттерны.

**Отсутствие лимитов и таймаутов**
- HTTP server без `ReadHeaderTimeout` и `MaxHeaderBytes`, handler читает body без `MaxBytesReader`. citeturn16view0turn16view3turn20view1  
- “длинные” операции без дедлайнов в gRPC: клиент не выставляет deadline, сервер не уважает cancellation. citeturn3search3turn7search1  

**Смешивание слоёв и неявные зависимости**
- LLM создаёт “глобальный singleton” для БД/клиентов, но не фиксирует жизненный цикл и shutdown; это ломает graceful shutdown и тестируемость. citeturn16view2turn22view0  
- LLM добавляет новый логгер/метрики, игнорируя `slog` и Prometheus naming. citeturn4search3turn8search3  

**Ошибка в contract semantics**
- Возвращает разные форматы ошибок на разных endpoint’ах (вместо RFC 9457). citeturn1search0  
- Использует 200 для операций, которые реально асинхронны, или “висит” слишком долго вместо 202 + async контракт. citeturn23view0  
- Не различает retry-safe и retry-unsafe операции; у клиента появляются повторы и дубли. citeturn12search1turn12search5  

**Security ошибки**
- Логи содержат bearer token/JWT/секреты или чувствительные поля. citeturn13search1turn3search0  
- Проверка только “аутентификации”, но не object-level authorization (BOLA), хотя это №1 риск в OWASP API Top 10 (2023). citeturn2search0turn13search12  

**Proto/API evolution ошибки**
- LLM меняет proto/OpenAPI и забывает про breaking change detection/миграционные заметки. Ровно для этого нужен механический контроль (Buf breaking). citeturn12search15turn12search3  

## Исследование подтемы: cross-cutting API concerns

Ниже — итог как **единые repo-wide правила**. Их смысл: каждый endpoint — это не только “handler”, а контракт + стандартный set семантик (валидация, auth context, retries, лимиты, errors). Эти правила должны быть отражены в:
- HTTP/gRPC контрактах (OpenAPI/proto),  
- middleware/interceptors,  
- документации (единый раздел `docs/api-contracts.md`),  
- generated examples (пример запроса/ответа в docs/ и в `examples/`). citeturn6search2turn8search1turn1search0  

### Request validation

**Repo-wide правило**
- MUST: валидация происходит на границе API, до вызова бизнес-логики. Это соответствует рекомендациям OWASP: валидировать как можно раньше по потоку данных. citeturn13search0  
- MUST: контрактно фиксировать ограничения (форматы, min/max, enum, required), а не только “в коде”. citeturn6search2turn6search0  

**HTTP контракт (OpenAPI)**
- MUST: схемы request/response валидируются и отражают ограничения (JSON Schema в рамках OAS 3.1). citeturn6search2turn6search9  
- SHOULD: запрет “неизвестных полей” (или явно принимать их) — это часть совместимости. В шаблоне лучше default “строго”, чтобы расширение контракта происходило осознанно.

**gRPC контракт**
- MUST: использовать protovalidate annotations для базовой семантической валидации; PGV в maintenance, рекомендуется переход. citeturn6search8turn6search0turn6search5  
- MUST: подключить validation interceptor и выдать единообразный mapping ошибок в `INVALID_ARGUMENT`. citeturn8search1turn17search3  

### Authentication/authorization context

**Repo-wide правило**
- MUST: определить единый “principal context” (например `AuthContext`), который включает: subject, tenant/org, scopes/roles, session id, request id/correlation id, и (если нужно) device/client metadata.  
- MUST: object-level authorization обязателен для операций по ID/ownership (OWASP API1:2023 подчёркивает, что ID в API создают большую поверхность BOLA). citeturn2search0  

**HTTP**
- MUST: использовать Bearer scheme по RFC 6750 (Authorization header), не query param. citeturn3search0  
- SHOULD: если используется OIDC для аутентификации, отделять “authentication/SSO” от “authorization to APIs” (соответствует OWASP Authentication guidance и OIDC spec). citeturn13search7turn3search2  

**gRPC**
- MUST: принимать credential/trace/correlation из metadata; gRPC metadata по определению используется как side channel для таких данных. citeturn8search10turn8search6  

### Idempotency keys и retry-safe endpoints

**База семантики**
- HTTP определяет свойства методов (safe/idempotent) и статус-коды, но **idempotency keys** как механизм поверх POST не стандартизированы RFC напрямую; поэтому важно нормировать правила внутри репозитория и явно документировать. citeturn1search3turn12search5  

**Repo-wide правило**
- MUST: классифицировать endpoints на:
  - retry-safe “по природе” (GET/HEAD и др. safe/idempotent по HTTP semantics), citeturn1search3turn9search13  
  - retry-safe “по контракту” (PUT/DELETE где повтор не создаёт доп. эффект), citeturn1search3turn9search13  
  - retry-unsafe (обычно POST, финтех/платежи/побочные эффекты), где требуется idempotency key. citeturn12search1turn12search5  
- MUST: для retry-unsafe операций принять единый механизм request identification / idempotency key. В RPC-стиле это формализовано как request IDs (AIP-155 подчёркивает, что ключевая цель request IDs — идемпотентность при повторной отправке). citeturn12search5  

**HTTP контракт**
- MUST: поддерживать заголовок `Idempotency-Key` для операций, которые могут быть ретраены клиентом (де-факто индустриальный паттерн; например entity["company","Stripe","payments company"] подробно описывает TTL, сравнение параметров и повторное использование ключей). citeturn9search2turn9search19  
- MUST: документировать TTL хранения ключей, область уникальности (per user/tenant), и поведение при конфликте параметров. citeturn9search2turn12search5  

**gRPC контракт**
- MUST: иметь поле `request_id`/`idempotency_key` в request message (или отдельный wrapper), и сервер обязан дедуплицировать. citeturn12search5turn12search1  

### Rate limiting semantics

**Repo-wide правило**
- MUST: при превышении лимита использовать 429 (RFC 6585) и возвращать объяснение условия; MAY включать `Retry-After`, RFC 6585 явно допускает. citeturn1search6turn1search2turn9search7  
- SHOULD: если лимит временный, HTTP semantics рекомендует указывать `Retry-After` (в RFC 9110 также это встречается для 413/503). citeturn20view1turn9search7  
- SHOULD (осторожно): поддержать RateLimit headers из IETF draft (RateLimit-Policy / RateLimit). Важно пометить в документации, что это draft (не RFC), и обеспечить fallback для клиентов. citeturn1search5turn1search1  

**Контрактное отражение**
- OpenAPI: описать 429 response, headers `Retry-After` и (опционально) RateLimit-*. citeturn1search6turn9search16turn1search5  
- gRPC: вернуть `RESOURCE_EXHAUSTED` как canonical код для лимитов. citeturn17search3  

### Request size limits

**Repo-wide правило**
- MUST: лимитировать размер:
  - headers (`Server.MaxHeaderBytes` или документированный дефолт), citeturn16view0turn15view1  
  - body (`http.MaxBytesReader`), citeturn16view3  
  - и (если применимо) URL/URI длину (на уровне ingress/gateway и/или сервера). citeturn20view1  
- MUST: при превышении размера body отвечать 413 (RFC 9110 описывает “Content Too Large” и допускает `Retry-After` при временной ситуации). citeturn20view1  

### File uploads

**Repo-wide правило**
- MUST: если поддерживаются загрузки файлов, контракт должен явно выбирать механизм:
  - `multipart/form-data` по RFC 7578, citeturn9search8turn9search11  
  - либо “upload session”/pre-signed URL (если файлы большие).  
- MUST: лимитировать размер и типы, избегать чтения всего файла в память; обеспечивать контроль ресурса (streaming + лимиты). Базовый building block — `MaxBytesReader` на HTTP стороне. citeturn16view3turn9search8  
- SHOULD: иметь security-план (например, антивирус/сканер/контент-проверки) на уровне async pipeline, если файлы потом исполняются/парсятся.

### Webhooks и callback patterns

**Repo-wide правило**
- MUST: считать webhooks “at-least-once delivery”: один и тот же event может прийти несколько раз, в другом порядке, с задержкой. Это должно быть прописано в контракте и примерах. (Это индустриальная необходимость; формализуйте её в репозитории как правило.)  
- SHOULD: стандартизировать envelope событий через CloudEvents v1.0 (CNCF позиционирует CloudEvents как спецификацию для описания event data в общем виде; есть и спецификация, и объявление о v1.0). citeturn9search5turn9search18turn9search12  
- MUST: webhooks endpoints должны быть идемпотентны по `event_id` (или `ce-id`) и иметь дедупликацию на стороне consumer. citeturn12search5turn9search12  

### Async API semantics

**Repo-wide правило**
- MUST: если операция не завершается быстро и детерминированно, возвращать 202 Accepted и выдавать клиенту способ проверять статус (operation resource / polling / callback). RFC 9110 описывает, что 202 означает “принято, но обработка не завершена”, и в HTTP нет механизма “дослать” финальный статус тем же запросом. citeturn23view0turn23view1  
- SHOULD: для RPC-стиля использовать long-running operations pattern (AIP-151), где клиент получает token/Operation и проверяет прогресс/результат. citeturn12search0turn12search4  

### Eventual consistency disclosure

**Repo-wide правило**
- MUST: если API не гарантирует read-after-write, либо использует асинхронную репликацию/индексацию, это должно быть явно указано в контракте и документации endpoint’а: какая стейлность возможна, где появляются “окна” неконсистентности.  
- Документируйте терминологию: “eventual consistency” означает, что при отсутствии новых обновлений чтения со временем придут к последнему значению; это определение хорошо формализовано в материалах о consistency. citeturn17search0  
- SHOULD: предпочитать strong consistency там, где возможно, потому что она упрощает прикладной код и повышает доверие (см. аргументацию в материалах по выбору strong consistency). citeturn17search4  

### Error model consistency

**HTTP**
- MUST: единый Problem Details (RFC 9457), фиксированный набор `type` значений (желательно как стабильные URIs) и единый mapping status code → тип ошибки. citeturn1search0turn1search12turn20view1  

**gRPC**
- MUST: единый mapping domain errors → gRPC status codes (NOT_FOUND, INVALID_ARGUMENT, PERMISSION_DENIED, UNAUTHENTICATED, RESOURCE_EXHAUSTED, FAILED_PRECONDITION…). citeturn17search3  
- SHOULD: при необходимости деталей использовать `google.rpc.Status`/details для машинного парсинга, с нормированными error details. citeturn6search3turn6search18turn6search7  

## Review checklist для PR / code review и список файлов для template repo

### PR / code review checklist

**Контракт и совместимость**
- Изменения API отражены в OpenAPI/proto; для proto проходит lint + breaking change detection (или есть явно утверждённое исключение). citeturn6search2turn12search15turn12search23  
- Ошибки соответствуют RFC 9457 (HTTP) и canonical codes (gRPC); нет “рандомного JSON”. citeturn1search0turn17search3  
- Для retry-unsafe операций реализована идемпотентность (request_id / Idempotency-Key), документированы TTL/semantics. citeturn12search5turn9search2  

**Безопасность**
- Валидация на границе API; нет “грязных” данных в домене. citeturn13search0  
- AuthN/AuthZ: корректный bearer/OIDC подход; object-level checks присутствуют там, где есть доступ по идентификаторам. citeturn3search0turn2search0turn13search12  
- Логи не содержат секретов/токенов/PII. citeturn13search1turn3search0  

**Надёжность и ресурсы**
- HTTP: `ReadHeaderTimeout`, лимит body, разумные MaxHeaderBytes; graceful shutdown корректен. citeturn16view0turn16view3turn16view2  
- gRPC: дедлайны и interceptors применены; health checking реализован. citeturn3search3turn8search1turn5search0  
- Никаких неограниченных очередей/бесконечных ретраев без backoff/лимитов; rate limiting семантика consistent (429/Retry-After). citeturn1search6turn9search16turn1search5  

**Observability**
- Логи структурные (`slog`) с стабильными ключами. citeturn4search3turn4search7  
- Метрики следуют правилам Prometheus (bounded labels). citeturn8search7turn8search3  
- Trace context propagation по W3C TraceContext и OpenTelemetry guidance. citeturn8search0turn2search3  

**Tooling / CI**
- `gofmt`, `go vet`, `go test` и `govulncheck` зелёные. citeturn10search0turn10search1turn10search2turn0search17  
- `go.mod` изменения осмыслены: версия Go фиксирована и согласована с toolchain. citeturn24view2turn11search7turn0search3  

### Что оформить отдельными файлами в template repo

Ниже — “минимальный пакет документов”, который уменьшает догадки LLM и людей. Рекомендуемый принцип: каждый файл отвечает на конкретный класс вопросов и содержит нормативные “MUST/SHOULD/NEVER”.

- `docs/engineering-standard.md`  
  Нормы по структуре проекта, слоям, ошибкам, тестам, observability, security. Ссылки на первичные источники: Effective Go, CodeReviewComments, go.dev module layout, Go security/vuln. citeturn4search0turn0search1turn22view0turn0search9  

- `docs/llm-instructions.md`  
  Секция MUST/SHOULD/NEVER (из этого отчёта) + “как работать с репозиторием”: что читать первым, как добавлять endpoint, как обновлять контракт, какие команды запускать. citeturn10search0turn10search1turn0search17  

- `docs/api-contracts.md`  
  Единые cross-cutting rules: validation/auth context/idempotency/rate limiting/retry-safe/request limits/uploads/webhooks/async/error model. Основные ссылки: RFC 9110/9457/6750/7519, OWASP API, IETF RateLimit draft, OpenAPI 3.1, gRPC guides. citeturn20view1turn1search0turn3search0turn3search1turn2search0turn6search2turn8search1turn1search5  

- `docs/observability.md`  
  Как логировать (slog), какие ключи обязательны, как подключать OTEL, какие метрики обязательны и какие labels запрещены. citeturn4search3turn5search1turn8search3turn8search7  

- `docs/security.md`  
  Минимальные требования: input validation, auth/authz, logging, dependency vulns (govulncheck), привязка к OWASP API Top 10 и ASVS. citeturn2search0turn2search1turn13search0turn0search9  

- `docs/runbook.md`  
  Health endpoints, readiness/liveness/startup, shutdown поведение, SLO-метрики (если есть), troubleshooting по логам/метрикам/трейсам. citeturn2search2turn2search6turn16view2turn5search0  

- `CONTRIBUTING.md` + `CODE_REVIEW.md`  
  Review-правила, требуемые проверки, стиль (ссылки на Go Code Review Comments и Google Go Style). citeturn0search1turn10search7turn0search20  

- `SECURITY.md`  
  Политика уязвимостей и минимальный security baseline; интеграция `govulncheck`. citeturn0search9turn0search17  

- `Makefile` или `taskfile` (по выбору), `.github/workflows/ci.yml`  
  Команды: fmt/test/vet/vulncheck, (опц.) buf lint/breaking, генерация контрактов. Конкретные команды должны быть частью “одной кнопки”, чтобы LLM всегда могла сослаться на них, а не “угадывать”. citeturn10search0turn10search1turn0search17turn12search23  

Упоминание “LLM tools” для контекста: если вы разводите документацию под несколько ассистентов, добавьте отдельный файл `docs/llm/prefix.md`, который можно копировать в системный промпт (например, для ChatGPT от entity["company","OpenAI","ai company"] или Claude Code от entity["company","Anthropic","ai company"]), и держите его синхронизированным с текущими правилами репозитория.