# Engineering standard для template Go-микросервиса: Authentication, Authorization и Service Identity

## Scope

Этот стандарт предназначен для **greenfield production-ready микросервиса**, который встраивается в микросервисную архитектуру и должен безопасно работать при допущении **“внутренняя сеть небезопасна”** (zero trust) и наличия нескольких типов идентичностей (как минимум: **workload/service identity** и **end-user identity**). citeturn8view0turn8view1turn6view0

Подход применять, когда:
- Сервис — **resource server** (API) и не хочет/не должен “изобретать OAuth/OIDC”, а валидирует и потребляет токены/идентичности, выпущенные внешним IdP / STS. citeturn11search0turn19view0turn24view4
- Архитектура ожидает **service-to-service вызовы** и требуется сильная **идентичность сервисов** (workload identity), предпочтительно через **mTLS** и автоматическую ротацию краткоживущих сертификатов/ключей. citeturn10view2turn8view1turn3search1turn0search3
- Требуется системная защита от типовых API-уязвимостей: “сломанная аутентификация”, ошибки object-level authorization (BOLA/IDOR), слабая авторизация и отсутствие “deny-by-default”. citeturn2search2turn14search0turn4search3turn2search7
- Команда хочет, чтобы LLM генерировала **идиоматичный и безопасный код** без догадок: нужны жёсткие defaults, явные интерфейсы и “зоны ответственности” (edge vs service, transport vs app-level). citeturn8view0turn8view2turn24view4

Подход не применять “как есть”, когда:
- Это **монолит** или “один сервис без внутренних вызовов”; service identity и сложная прокси/mesh-инфраструктура могут быть избыточны (но правила JWT-валидации и object-level authorization всё равно актуальны). citeturn2search12turn14search0turn16view2
- Продукт — **BFF/монолитный web-app**, где первична cookie-based сессия и CSRF; здесь JWT-бейреры не обязаны быть основой, а требования к cookie и сессиям выходят на первый план. citeturn14search2turn14search22turn14search7
- Система требует **высоких гарантий немедленной ревокации** на каждый запрос и допускает дополнительную латентность — тогда “offline JWT validation” может быть недостаточно, и придётся проектировать introspection/онлайн-проверки. citeturn15view3turn24view4turn22view4

## Recommended defaults для greenfield template

Ниже — “boring, battle-tested defaults”, которые минимизируют допущения для LLM и позволяют сервису безопасно стартовать в разных окружениях.

### Базовая модель идентичности: разделяйте end-user и workload identity

1) **Workload (service) identity**: идентичность самого сервиса/инстанса для service-to-service доверия и политик. citeturn8view0turn8view1turn6view4  
2) **End-user identity**: идентичность пользователя/клиента, для которого выполняется запрос (если применимо). citeturn8view0turn8view2

В template это должно быть отражено в коде и интерфейсах, иначе LLM будет смешивать уровни (частая причина уязвимостей “authenticated-but-not-authorized”). citeturn14search0turn2search2turn4search20

### Default для end-user AuthN: OAuth2/OIDC + access token как JWT (профиль RFC 9068), сервис — resource server

**Default**: сервис принимает **Bearer access token** в `Authorization: Bearer …` и валидирует его как **JWT access token** по **RFC 9068** (если ваша экосистема позволяет). citeturn24view4turn15view2turn11search0  
Ключевые нормативные требования профиля (resource server):
- JWT access token **MUST быть подписан** и **MUST NOT** использовать `alg: "none"`. citeturn24view4turn16view0  
- Resource server **MUST** проверять `iss`, `aud`, подпись и запрещать `alg="none"`. citeturn24view4turn16view3  
- Resource server **MUST** проверять тип токена через `typ` = `at+jwt` или `application/at+jwt` и отклонять другое (для предотвращения token substitution / ID token confusion). citeturn24view3turn24view4turn16view2  

**Транспорт**: TLS обязателен для Bearer tokens по RFC 6750 (иначе токен легко перехватывается и переиспользуется). citeturn15view2turn11search9  

**Ключи/метаданные**: привязка `iss → jwks_uri` через discovery/metadata и строгая проверка, что ключи принадлежат этому issuer — обязательна для защиты от подмены ключей/issuer. citeturn16view3turn12search1turn12search2turn13search1  

### Default для service-to-service AuthN: mTLS с workload identity, предпочтительно SPIFFE/SPIRE

**Default** для внутреннего трафика — **mTLS и политика, требующая mTLS для service calls** (service-level authentication). citeturn8view1turn6view0turn10view1  
NIST прямо исходит из предпосылки, что “вся сеть и все микросервисы недоверенные”, и требует mutual authentication и защищённые каналы (mTLS). citeturn6view0turn10view1  

Если есть возможность внедрить стандартизированную workload identity:
- Используйте **SPIFFE IDs** (формат `spiffe://trust-domain/workload-identifier`) и **SVIDs (X.509-SVID / JWT-SVID)**, получаемые через Workload API у провайдера вроде SPIRE. citeturn3search0turn3search1turn3search12  
- SPIRE ориентирован на выпуск **короткоживущих автоматически ротируемых ключей и сертификатов** для массового mTLS между ворклоадами. citeturn0search3turn3search12  
- SPIFFE/SPIRE — зрелые CNCF-проекты (Graduated), что делает их “boring default” для workload identity в cloud-native сценариях. citeturn4search0turn4search8turn4search4  

NIST рекомендует практики для сертификатов/идентичностей в mesh-сценариях:  
- lifetime identity-сертификата “предпочтительно порядка часов” и ротация для ограничения окна атаки. citeturn10view2  
- при ротации прокси/клиенту стоит перевыпускать соединения, т.к. сертификаты проверяются в mTLS handshake. citeturn10view2  

### Default для AuthZ: “Deny by default”, RBAC как старт, ABAC как эволюция

- **Deny by default** на уровне всего сервиса и отдельных endpoint’ов. Требование “access controls fail securely” и запрет на клиентское влияние на атрибуты/политики — базовые ожидания. citeturn2search7turn4search14turn4search20  
- Для старта (template) выбирайте **RBAC** как более простой и объяснимый подход; в Kubernetes это основной механизм авторизации и least privilege для service accounts. citeturn4search1turn4search5turn4search7  
- Для сложных доменов (multi-tenant B2B, fine-grained entitlements, контекст/атрибуты) эволюционируйте к **ABAC**, где решение принимается по атрибутам субъекта, ресурса и окружения. citeturn2search5turn6view3  
- Для внешнего/централизованного policy engine рассмотрите **entity["organization","Open Policy Agent","policy engine cncf"]**: он CNCF Graduated и типично используется как PDP/PEP-паттерн в распределённой авторизации. citeturn4search2turn4search6turn6view0  

### Default для multi-tenancy и tenant isolation

- В каждом запросе должен существовать **tenant context** (claim/атрибут), и **object-level authorization** обязана учитывать tenant boundary; иначе получите API1 BOLA/IDOR как классическую уязвимость. citeturn14search0turn14search1  
- Multi-tenant дизайн должен специально предотвращать cross-tenant атаки (изоляция данных/кэша/индексов/контекстов). citeturn14search1turn14search0  

### Identity propagation по умолчанию: “не придумывайте магию”, фиксируйте выбранный режим

NIST описывает распространённый паттерн: на входе часто **обменивают внешний credential** (например OAuth bearer token) на **внутренний credential** (часто JWT) для внутреннего периметра. citeturn8view2  
В стандарте template нужно явно задать (и закодировать):
- Прокидываем ли мы **оригинальный end-user access token** дальше?
- Или делаем **token exchange** / internal token mint?
- Или вообще не прокидываем токен, а прокидываем *минимальные атрибуты* (но тогда нужна криптографическая защищённость и строгая доверенная граница). citeturn1search0turn24view4turn16view2  

image_group{"layout":"carousel","aspect_ratio":"16:9","query":["OAuth 2.0 authorization code flow diagram","service mesh mTLS architecture diagram","SPIFFE SPIRE workload identity diagram"],"num_per_query":1}

## Decision matrix и trade-offs

Ниже — практический decision framework: какую схему выбирать под тип системы и какие компромиссы принять. (Это именно то, что нужно зафиксировать в docs/ и дать LLM как “решающий алгоритм”.)

### Матрица выбора AuthN/AuthZ/Identity

| Сценарий | End-user AuthN | Service-to-service AuthN (workload identity) | AuthZ модель | Identity propagation | Почему / trade-offs |
|---|---|---|---|---|---|
| Публичный API (mobile/web), один сервис или небольшой набор | OIDC/OAuth2, Bearer access token; ресурс-сервер валидирует JWT по RFC 9068 | mTLS желательно, минимум TLS внутри; в идеале mTLS в mesh, но можно начать без | RBAC + object-level checks | Forward access token (если `aud` корректен) или gateway/internal token | Bearer токены требуют TLS. Типовые проблемы — token leakage и неверная валидация JWT. citeturn15view2turn24view4turn2search2turn14search0 |
| Публичный API + много микросервисов, высокий риск lateral movement | OIDC/OAuth2; prefer per-service audience | **mTLS MUST**; предпочтительно SPIFFE/SPIRE | ABAC (атрибуты + политики) или RBAC+OPA | Token exchange (RFC 8693) или downscoped tokens | Per-service `aud` снижает риск “token usable everywhere”. Token exchange сложнее, но даёт least privilege по hop’ам. citeturn24view4turn1search0turn10view2turn6view3 |
| Internal-only сервисы (без end-user), machine-to-machine | OAuth2 Client Credentials (JWT) **или** чистый mTLS | mTLS + workload identity (SPIFFE предпочтительно) | RBAC (service principals) или ABAC | Обычно не требуется user propagation | Для M2M важно отличать “кто сервис” и разрешения сервису. Bearer сервис-токены лучше sender-constrain via mTLS/DPoP если риск высокий. citeturn22view3turn1search3turn13search0turn6view0 |
| Кросс-организационный / B2B интеграции | OIDC/OAuth2 строго по профилю, можно opaque+introspection | mTLS для B2B каналов, отдельные trust bundles | ABAC (контракты, entitlements) | Обычно token exchange/STS | Чаще нужен контроль ревокации и контрактов; introspection даёт “active state”, но добавляет зависимость/латентность. citeturn15view3turn1search2turn24view4turn2search1 |
| Browser-first приложения (cookie session) | Cookie session + CSRF defense | mTLS опционально внутри | RBAC/ABAC | Обычно не прокидывают cookie между сервисами; используют internal tokens | Cookies — stateful sessions; нужны Secure/HttpOnly/SameSite и CSRF меры. Это другой профиль угроз. citeturn14search2turn14search22turn14search7 |

### Ключевые развилки и “boring defaults” по каждой

**JWT vs opaque tokens**
- JWT (offline validation): меньше латентности и зависимости от IdP на каждый запрос, но сложнее история с ревокацией и key rotation; нужны строгие проверки `iss/aud/typ/alg`, иначе риск token substitution и алгоритмических атак. citeturn24view4turn16view0turn16view2  
- Opaque + introspection: даёт “active/revoked state” через сервер авторизации, но увеличивает coupling и нагрузку; introspection endpoint должен быть защищён от token-scanning и требует авторизации доступа. citeturn15view3turn1search2  

**Bearer tokens vs sender-constrained tokens (mTLS / DPoP)**
- Bearer: “кто владеет — тот и использует”, поэтому критичны защита в транспорте/хранении. citeturn11search9turn15view2  
- Sender-constrained: снижает ущерб от утечки токена (нельзя просто переиспользовать), но добавляет сложность: mTLS-binding (RFC 8705) или DPoP (RFC 9449), плюс операционные детали. citeturn1search3turn13search0turn22view4  

**RBAC vs ABAC**
- RBAC: проще, предсказуемее, легче review; часто достаточно для early-stage. citeturn4search1turn4search3  
- ABAC: гибкость и точность (атрибуты субъект/ресурс/окружение), полезно в multi-tenant и комплексных доменах; дороже в проектировании и тестировании. citeturn2search5turn6view3turn14search1  

**Где делать проверку end-user токена**
- На edge (gateway/ingress) + внутри сервиса (defense-in-depth): снижает риск обхода gateway и “скрытых” внутренних входов; соответствует zero-trust предпосылкам. citeturn6view0turn9view3turn24view4  
- Только на edge: проще и быстрее, но опасно при наличии обходных путей/внутренних соединений (часто реальность). citeturn6view0turn2search12  

## Набор правил MUST / SHOULD / NEVER для LLM

Цель этого раздела — буквально лечь в `docs/llm/security-authn-authz.md` и использоваться как “system prompt” для генерации кода в template.

### Модель и границы доверия

- **MUST** считать внутренний трафик недоверенным и проектировать service-to-service аутентификацию (минимум: mTLS в идеале). citeturn6view0turn8view1turn10view1  
- **MUST** различать “кто вызывает” (workload identity) и “для кого действует” (end-user identity) и хранить их раздельно в контексте запроса. citeturn8view0turn8view1  
- **NEVER** использовать “доверенные заголовки” (`X-User-Id`, `X-Tenant`) как источник истины без криптографической защиты и явно ограниченной доверенной границы. (Иначе это подмена identity.) citeturn16view2turn14search0turn2search7  

### JWT / OAuth2 / OIDC валидация

- **MUST** валидировать JWT полностью: подпись, `iss`, `aud`, допустимый `alg`, и отклонять токены при любой ошибке криптопроверки. citeturn16view0turn16view3turn24view4  
- **MUST** whitelist’ить алгоритмы и запрещать “алгоритмическую подмену” (`RS256 → HS256`, `alg=none`) и любые алгоритмы вне набора. citeturn16view0turn16view1  
- **MUST** проверять `aud` и отклонять токены без корректной аудитории, если issuer выпускает токены для нескольких relying parties/resource servers. citeturn16view2turn24view4  
- **MUST** проверять, что ключи верификации действительно принадлежат issuer (binding `iss → jwks_uri`, issuer metadata). citeturn16view3turn12search1turn12search2  
- **MUST** (если используется профиль RFC 9068) проверять `typ` = `at+jwt`/`application/at+jwt` и отклонять иные значения, чтобы избежать принятия ID Token за Access Token. citeturn24view3turn24view4  
- **NEVER** “оптимизировать” безопасность выключением проверок (`SkipClientIDCheck`, пропуск `aud/iss`, “только декодируем payload”). Это прямой путь к подделке токенов. citeturn24view4turn21view0turn16view2  

### OAuth security best practice (важно для подсказок и code review)

- **SHOULD** избегать implicit grant (`response_type=token`) и любых режимов, где access token выдаётся в authorization response, из‑за рисков утечки/реигрыша токена. citeturn22view0turn20view4  
- **NEVER** использовать Resource Owner Password Credentials grant (пароли через клиент): RFC 9700 прямо запрещает. citeturn22view2  
- **SHOULD** (на стороне клиентов/IdP-конфигурации) использовать PKCE и предотвращать downgrade; даже если сервис не реализует OAuth client flow, LLM не должна предлагать небезопасные схемы “вокруг” сервиса. citeturn22view1turn12search3  
- **MUST** помнить: refresh tokens — очень привлекательная цель; при их использовании нужны меры против replay (rotation, sender-constraining и т.п.). citeturn22view4  

### Service-to-service identity и mTLS

- **MUST** при mTLS привязывать разрешения/политику к **service identity**, а не к “IP/hostname”, и учитывать, что сервисы перемещаются/масштабируются. citeturn8view0turn6view4turn10view2  
- **SHOULD** использовать workload identity, совместимую со SPIFFE (SPIFFE ID + SVID через Workload API), если доступно. citeturn3search0turn3search1turn4search8  
- **SHOULD** обеспечивать короткие TTL для identity‑сертификатов и их автоматическую ротацию на уровне инфраструктуры. citeturn10view2turn7view1  
- **NEVER** использовать `InsecureSkipVerify` или отключать проверку цепочки доверия/имён ради “быстрого старта”. Это ломает сам смысл mTLS. citeturn10view1turn6view0  

### Authorization (внутри сервиса)

- **MUST** выполнять **object-level authorization** (владелец/тенант/ACL) для каждого доступа по id/ключу; аутентификация без объектной проверки = OWASP API1 (BOLA). citeturn14search0turn14search4  
- **MUST** “deny by default”: отсутствие правила/роли/условия должно означать отказ. citeturn2search7turn4search20  
- **SHOULD** комбинировать claims (`scope/roles/entitlements`) с контекстом запроса и правилами домена; scopes сами по себе не доказывают право на конкретный объект. citeturn24view4turn4search3turn14search0  
- **NEVER** позволять клиенту управлять атрибутами/политикой (например, принимать `role=admin` из тела запроса). citeturn2search7turn4search14  

### Логи, ошибки и приватность

- **NEVER** логировать токены/refresh tokens/authorization codes целиком; считать их секретами. Bearer token, попавший в лог, становится доступом. citeturn11search9turn15view2turn22view4  
- **SHOULD** возвращать ошибки валидатора токена как `invalid_token` (когда применим профиль bearer), не раскрывая детали, и отделять “401 unauthenticated” от “403 forbidden”. citeturn24view4turn15view2  
- **SHOULD** вычищать/маскировать персональные данные в логах и трейсах, особенно в multi-tenant сценариях. citeturn14search1turn2search3  

## Concrete good / bad examples на Go

Ниже — примеры, которые можно почти дословно переносить в template. Они демонстрируют: строгую JWT‑валидацию, разделение субъектов, и “fail closed”.

### Good: HTTP middleware для JWT access tokens (RFC 9068‑ориентированный)

Этот пример следует требованиям: проверка `iss/aud/typ/alg`, запрет `none`, и общая логика resource server validation. citeturn24view4turn16view0turn16view2turn15view2

```go
package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

// Principal — минимальный набор, который нужен сервису для AuthZ.
// Важно: не хранить сырой токен в контексте.
type Principal struct {
	Subject   string            // end-user или client_id (зависит от grant)
	TenantID  string            // multi-tenant boundary
	Scopes    map[string]bool   // для быстрых проверок
	RawClaims map[string]any    // опционально, для ABAC (осторожно с PII)
}

type ctxKey struct{}

var principalKey ctxKey

func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	v := ctx.Value(principalKey)
	p, ok := v.(*Principal)
	return p, ok
}

// JWTVerifier инкапсулирует OIDC discovery + JWKS.
type JWTVerifier struct {
	verifier *oidc.IDTokenVerifier
	issuer   string
	audience string
	typAllow map[string]bool // e.g., {"at+jwt":true, "application/at+jwt":true}
	clockSkew time.Duration  // допустимый сдвиг часов
}

type JWTVerifierConfig struct {
	IssuerURL   string        // iss
	Audience    string        // aud (resource indicator value)
	ClockSkew   time.Duration // например 30s
}

func NewJWTVerifier(ctx context.Context, cfg JWTVerifierConfig) (*JWTVerifier, error) {
	if cfg.IssuerURL == "" || cfg.Audience == "" {
		return nil, errors.New("issuer and audience are required")
	}

	// Важно: provider должен жить долго; не создавать на каждый запрос.
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, err
	}

	// IDTokenVerifier проверяет подпись + iss + aud + exp.
	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.Audience, // для access tokens часто соответствует resource indicator / aud
		// НЕ включать SkipClientIDCheck в template defaults.
	})

	typAllow := map[string]bool{
		"at+jwt":              true,
		"application/at+jwt":  true,
		"at+JWT":              true, // встречается в примерах; безопаснее нормализовать ниже
		"application/at+JWT":  true,
	}

	return &JWTVerifier{
		verifier: verifier,
		issuer: cfg.IssuerURL,
		audience: cfg.Audience,
		typAllow: typAllow,
		clockSkew: cfg.ClockSkew,
	}, nil
}

func (v *JWTVerifier) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("Authorization")
		if raw == "" {
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(raw, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "invalid authorization scheme", http.StatusUnauthorized)
			return
		}

		token := strings.TrimSpace(parts[1])
		if token == "" {
			http.Error(w, "empty token", http.StatusUnauthorized)
			return
		}

		idToken, err := v.verifier.Verify(r.Context(), token)
		if err != nil {
			http.Error(w, "invalid_token", http.StatusUnauthorized)
			return
		}

		// Защита от token substitution: enforce typ, если используете RFC 9068.
		// go-oidc не делает этого автоматически.
		typ := strings.ToLower(strings.TrimSpace(idToken.Header["typ"].(string)))
		if typ != "" && !v.typAllow[typ] {
			http.Error(w, "invalid_token", http.StatusUnauthorized)
			return
		}

		// Дополнительная защита от clock skew (если нужно).
		now := time.Now()
		if now.Add(v.clockSkew).After(idToken.Expiry) {
			http.Error(w, "invalid_token", http.StatusUnauthorized)
			return
		}

		claims := map[string]any{}
		if err := idToken.Claims(&claims); err != nil {
			http.Error(w, "invalid_token", http.StatusUnauthorized)
			return
		}

		p, err := principalFromClaims(claims)
		if err != nil {
			http.Error(w, "invalid_token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), principalKey, p)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Пример очень простой маппинга.
// В template это MUST быть документировано как "адаптер под ваш IdP".
func principalFromClaims(claims map[string]any) (*Principal, error) {
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, errors.New("missing sub")
	}

	tenant, _ := claims["tenant_id"].(string) // пример; зависит от IdP/домена

	scopeStr, _ := claims["scope"].(string) // RFC 9068 рекомендует scope claim
	scopes := map[string]bool{}
	for _, s := range strings.Fields(scopeStr) {
		scopes[s] = true
	}

	return &Principal{
		Subject:   sub,
		TenantID:  tenant,
		Scopes:    scopes,
		RawClaims: claims,
	}, nil
}
```

### Bad: “просто декодируем JWT payload и верим”

Ниже пример того, что LLM часто “галлюцинирует”: отсутствует проверка подписи, `iss`, `aud`, `alg` и тип токена. Это нарушает BCP по JWT, профиль JWT access tokens и позволяет подделку токена/подмену контекста. citeturn16view0turn16view2turn24view4

```go
// ❌ НЕ ДЕЛАТЬ: это не аутентификация, а парсинг JSON.
func InsecureReadUserID(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var m map[string]any
	_ = json.Unmarshal(payload, &m)
	sub, _ := m["sub"].(string)
	return sub // ❌ attacker-controlled
}
```

### Good: SPIFFE mTLS для gRPC/HTTP client (workload identity)

В Go экосистеме SPIFFE предоставляет библиотеку, которая упрощает получение SVID и настройку mTLS поверх Workload API. citeturn3search2turn3search17turn3search9

```go
package spiffemtls

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

// HTTPClientWithSPIFFE создает http.Client, который:
// - предъявляет собственный X.509-SVID,
// - проверяет peer по SPIFFE ID и trust bundle.
func HTTPClientWithSPIFFE(ctx context.Context, serverID spiffeid.ID) (*http.Client, func(), error) {
	// X509Source автоматически обновляется при ротации SVID.
	x509Src, err := workloadapi.NewX509Source(ctx)
	if err != nil {
		return nil, nil, err
	}

	tlsConf := tlsconfig.MTLSClientConfig(x509Src, x509Src, tlsconfig.AuthorizeID(serverID))

	transport := &http.Transport{
		TLSClientConfig: tlsConf,
		// В template здесь также должны быть timeouts/keep-alive настройки как часть boring defaults.
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	cleanup := func() { _ = x509Src.Close() }
	return client, cleanup, nil
}
```

## Anti-patterns и типичные ошибки / hallucinations LLM

Ниже — список “часто приводит к взлому”, который должен быть отдельным разделом в LLM‑instructions и в code review checklist.

### Ошибки JWT/OIDC

- Принятие `alg=none` или отсутствие whitelist алгоритмов → позволяет обойти подпись или провести algorithm confusion (`RS256 → HS256` и т.п.). citeturn16view0turn16view1turn24view4  
- Отсутствие проверки `aud` или использование “общего audience” без понимания → токен может быть подставлен другому сервису (cross-JWT confusion). citeturn16view2turn24view4turn24view1  
- Не проверять связь `iss → ключи` (неправильный JWKS, подмена issuer metadata) → сервис доверяет чужим ключам. citeturn16view3turn12search1turn13search1  
- Принимать **ID Token как access token** (особенно если `typ` не проверяется) → подмена контекста/аудитории. Профиль RFC 9068 требует явного типирования как защиты. citeturn24view3turn24view4  
- “Оптимизация”: создавать discovery/JWKS клиент на каждый запрос или отменять контексты неправильно → нестабильность, гонки, DoS на IdP. (Это не уязвимость напрямую, но ломает production.) Для go-oidc рекомендуемый паттерн — long-lived provider/verifier. citeturn23search5turn23search1  

### Ошибки OAuth flow вокруг сервиса (LLM любит предлагать)

- Предлагать implicit grant или хранение токенов в URL fragment как “норму” → RFC 9700 говорит избегать implicit из‑за утечек/реигрыша. citeturn22view0turn20view4  
- Предлагать password grant (ROPC) → запрещено современными best practices. citeturn22view2  
- Игнорировать PKCE / downgrade‑риски → RFC 9700 задаёт строгие требования к PKCE и его обнаружению через metadata. citeturn22view1turn13search1  

### Ошибки service-to-service и mTLS

- Отключать проверку TLS (например, `InsecureSkipVerify`) ради “self-signed” → компрометирует аутентификацию сервисов. citeturn10view1turn6view0  
- Считать, что “mTLS решает всё” и не делать end-user AuthZ → mTLS даёт идентичность **сервиса**, а не пользователя; NIST явно разделяет эти роли. citeturn8view0turn8view2turn6view4  

### Ошибки авторизации и multi-tenancy

- “Пользователь аутентифицирован ⇒ может читать объект” без проверки ownership/tenant → это OWASP API1 BOLA (часто самый критичный риск). citeturn14search0turn14search4  
- Не фиксировать tenant context в каждом слое (handlers → service → repository), смешивать tenant в кеше/индексах → cross-tenant data leak. citeturn14search1turn14search0  
- Разрешения “по умолчанию” слишком широкие (нет least privilege) → потом тяжело “отобрать” доступ и это приводит к инцидентам. citeturn4search3turn4search7  

### Ошибки сессий и токенов (когда сервис — BFF)

- Класть bearer/JWT в cookie без понимания CSRF и cookie‑атрибутов → риск CSRF и token leakage; cookie storage требует Secure/HttpOnly/правильной политики и защиты от CSRF. citeturn14search2turn14search22turn14search7  

## Review checklist для PR / code review

Этот чеклист стоит положить в repo как `docs/review/security-auth-checklist.md` и использовать в PR‑шаблоне.

**Аутентификация (end-user / tokens)**
- Проверки JWT выполняются полностью: подпись + `iss` + `aud` + допустимый `alg`; `alg=none` отвергается. citeturn16view0turn16view3turn24view4  
- Если используется RFC 9068, проверяется `typ=at+jwt`/`application/at+jwt` и токены других типов отвергаются. citeturn24view3turn24view4  
- Ошибки аутентификации приводят к fail-closed (401) без утечки деталей; bearer‑совместимые ответы используют `invalid_token` где уместно. citeturn24view4turn15view2  
- Токены/секреты не логируются; присутствует редактирование/маскирование. citeturn11search9turn22view4  

**Авторизация**
- Для каждого endpoint, который читает/пишет объект по идентификатору, есть object-level check (ownership/ACL/tenant). citeturn14search0turn14search4  
- “Deny by default” реализован явно: отсутствие правил = запрет. citeturn2search7turn4search20  
- Claims из токена не принимаются “на веру” в смысле доменных прав; используется минимум: scopes/roles + доменные проверки. citeturn24view4turn4search3turn14search0  

**Service identity / mTLS**
- Внутренние вызовы по возможности требуют mTLS; нет отключения TLS‑проверок. citeturn6view0turn10view1  
- Если используется SPIFFE/SPIRE: peer authorizer проверяет ожидаемый SPIFFE ID (или набор), а не “любого валидного сертификата”. citeturn3search0turn3search2turn3search1  
- Ротация identity‑сертификатов предусмотрена инфраструктурно (короткий TTL, обновление соединений). citeturn10view2turn10view1  

**Multi-tenancy**
- Tenant context извлекается из проверенного источника (claims) и используется везде: бизнес‑логика, репозитории, кэш‑ключи, очереди/ивенты. citeturn14search1turn14search0  
- Есть тесты/табличные тесты, покрывающие попытки cross-tenant доступа. citeturn14search1turn14search0  

**Identity propagation**
- Выбранный способ propagation документирован и реализован последовательно (forward token vs token exchange vs internal token). citeturn8view2turn1search0turn24view4  
- Нет несанкционированного “повышения привилегий” по мере прохождения через сервисы (например, добавления scopes/roles на ходу). citeturn4search3turn22view4  

## Что из результата оформить отдельными файлами в template repo

Ниже — рекомендуемый набор файлов, которые минимизируют “догадки” для LLM и фиксируют defaults.

- `docs/security/authn-authz-service-identity.md`  
  Краткая “модель мира”: 2 типа identity (workload/end-user), zero trust предпосылки, где стоит gateway, где сервис, и что именно проверяет микросервис. citeturn8view0turn6view0turn9view3  

- `docs/security/oauth-oidc-resource-server.md`  
  Нормативные требования к JWT access tokens: `iss/aud/alg/typ`, запрет `none`, обработка ошибок, требования TLS, ссылки на RFC 6750/9068/8725/9700. citeturn15view2turn24view4turn16view0turn13search7  

- `docs/security/identity-propagation.md`  
  Решающее дерево: forward токена vs token exchange (RFC 8693) vs internal token mint; что выбрать при per-service audiences; риски confused deputy и cross-JWT confusion. citeturn1search0turn24view4turn16view2turn8view2  

- `docs/security/service-mtls-spiffe.md`  
  Default: mTLS для внутренних вызовов; когда нужен SPIFFE/SPIRE; как выглядят SPIFFE ID/SVID; как валидировать peer identity; как это отражать в Go коде (go-spiffe). citeturn3search0turn3search1turn4search8turn10view2  

- `docs/security/multi-tenant-authorization.md`  
  Tenant isolation: какие claims нужны, где хранить tenant context, правила для кэшей/репозиториев, анти‑паттерны, тест‑кейсы против BOLA/IDOR. citeturn14search1turn14search0  

- `docs/llm/security-authn-authz.rules.md`  
  Раздел MUST/SHOULD/NEVER (как в этом отчёте), плюс “код-скелеты” middleware/interceptors и запреты на небезопасные паттерны. citeturn16view0turn22view0turn14search0  

- `docs/review/security-auth-checklist.md` и интеграция в `.github/PULL_REQUEST_TEMPLATE.md`  
  Чеклист из раздела review (минимум 1 блок “AuthN/AuthZ”). citeturn2search2turn24view4turn6view0  

- `internal/auth/` (кодовый модуль template)  
  `verifier.go`, `middleware_http.go`, `interceptor_grpc.go`, `principal.go`, `policy.go` (интерфейс для RBAC/ABAC), `errors.go`. Важно: отделить “проверку токена” от “решения авторизации”. citeturn24view4turn4search20turn6view3  

- `configs/auth.example.yaml` (или env schema)  
  Явные параметры: `ISSUER_URL`, `AUDIENCE`, `CLOCK_SKEW`, `REQUIRED_SCOPES`, `TENANT_CLAIM`, `ENFORCE_TYP`. Смысл — снизить догадки LLM и сделать поведение воспроизводимым. citeturn24view4turn16view2turn14search1  

В тексте всех этих файлов один раз сослаться на ключевые источники: entity["organization","National Institute of Standards and Technology","us standards body"] SP 800‑204A/204B для microservices/service mesh identity и политики, entity["organization","OWASP","app security nonprofit"] для API risks/access control/multi-tenant guidance, и стандарты entity["organization","Internet Engineering Task Force","ietf standards body"] (RFC 6750/8725/9068/9700/8693/7662), плюс проекты entity["organization","Cloud Native Computing Foundation","cloud native foundation"] (SPIFFE/SPIRE/OPA). citeturn10view2turn8view2turn14search1turn24view4turn13search7turn1search0turn4search8turn4search2