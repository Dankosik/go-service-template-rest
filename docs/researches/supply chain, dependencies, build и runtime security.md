# Security baseline для production-ready Go-микросервиса: supply chain, зависимости, build и runtime hardening

## Scope

Этот стандарт применим, когда вы делаете greenfield-template микросервиса на Go, который собирается в CI/CD, поставляется как контейнерный образ и запускается в оркестраторе (в первую очередь в entity["organization","Kubernetes","container orchestration project"]). Он особенно полезен, если цель — чтобы разработчик мог «склонировал → запустил CI → деплоит в dev/stage/prod» без локальных шаманств, а LLM-инструменты генерировали код и инфраструктурные фрагменты без опасных догадок при ограниченном контексте. citeturn3view2turn5view0turn3view1

Этот стандарт НЕ закрывает все аспекты AppSec (аутентификация/авторизация, криптография протоколов, безопасная обработка входных данных и т.д.) и не заменяет threat modeling на уровень продукта/домена. Он также не является «сертификацией» — это baseline, оптимизированный под boring и battle-tested defaults, который можно ужесточать по мере роста рисков и требований. citeturn1search0turn8search0

Стандарт может оказаться избыточным или неподходящим, если:  
- сервис не контейнеризуется (например, embedded/edge) и не использует общий CI/CD pipeline;  
- среда полностью air‑gapped и запрещает любые внешние сети (тогда часть механизмов Go Modules/proxy/sumdb и поставки артефактов нужно адаптировать через внутренние прокси/репозитории);  
- у вас уже есть корпоративная платформа с жестко заданными политиками (например, централизованная сборка/подпись/аттестации) — здесь следует встроиться в неё, а не вводить «второй стандарт». citeturn3view1turn12view0turn8search3

## Recommended defaults для greenfield template

Ниже — «канонический baseline» для репозитория шаблона. Формулировки сознательно нормативные и ориентированы на автоматизацию.

### Go modules integrity и политика зависимости

1) **Всегда используйте Go Modules, фиксируйте `go.mod` и `go.sum` в репозитории.** Go toolchain проверяет криптографические хэши загруженных модулей по `go.sum`, а при отсутствии хэша в `go.sum` может сверяться с checksum database (если модуль не приватный и sumdb не отключена). citeturn3view1turn8search2turn3view0  

2) **Не отключайте checksum database по умолчанию.** Установка `GOSUMDB=off` отключает обращения к checksum database и лишает гарантии «проверяемых повторяемых скачиваний» для модулей, отсутствующих в `go.sum`; в документации прямо отмечено, что это делается ценой потери security guarantee. citeturn3view1  

3) **Приватные модули: используйте `GOPRIVATE` (и при необходимости `GONOPROXY`/`GONOSUMDB`) точечно по префиксам.** `GOPRIVATE` действует как default для `GONOSUMDB` и `GONOPROXY`, поэтому обычно достаточно задать только `GOPRIVATE`, а более тонкую настройку делать при реальной необходимости. citeturn3view1turn2search1  

4) **CI верифицирует зависимости и запрещает «неявные» изменения модулей.**  
- `GOFLAGS=-mod=readonly` в CI, чтобы сборка не меняла `go.mod`/`go.sum` и не «подтягивала» зависимости скрытно (вендоринг можно включать отдельно). citeturn4view0  
- `go mod tidy -diff` как обязательная проверка чистоты зависимостей (ошибка, если есть diff). citeturn4view0  
- `go mod verify` как проверка, что содержимое модулей в module cache не было изменено после скачивания. citeturn8search2turn4view0  

5) **Vendoring** по умолчанию выключен, но допустим как опция для некоторых корпоративных/air‑gapped сред. Go tooling явно поддерживает `-mod=vendor` (и возможность отключить vendoring через `-mod=readonly` или `-mod=mod`). citeturn4view0  

### Обязательные проверки в CI/CD (build-time security gates)

**Минимальный обязательный набор security gates**, который должен быть greenfield-дефолтом:

- **`go vet ./...`** как базовый статический анализ на подозрительные конструкции. Важно помнить, что `vet` использует эвристики и не гарантирует, что каждое сообщение — реальная проблема; но как «гейт качества» для шаблона это разумный минимум. citeturn9search0turn9search4  

- **`govulncheck ./...`** как первичный механизм выявления известных уязвимостей в зависимостях с «низким шумом»: инструмент анализирует зависимости и определяет, есть ли реальные вызовы уязвимых функций из вашего кода (direct/indirect calls). Это снижает false positives по сравнению с чисто «по наличию пакета». citeturn0search1turn0search21turn0search37  

  Критический нюанс для CI: `govulncheck` **возвращает ненулевой код выхода при найденных уязвимостях**, но если вы запускаете его с `-json`, `-format sarif` или `-format openvex`, то он **всегда выходит с кодом 0 независимо от количества найденных уязвимостей** — это легко превращается в ложный green build. citeturn8search9turn0search9  

- **Проверки целостности модулей**: `go mod tidy -diff` + `go mod verify` + сборка с `-mod=readonly`. citeturn4view0turn8search2  

**Рекомендуемые (SHOULD) проверки**, которые уместны как «включено по умолчанию, но допускает отключение по обоснованию»:  
- SAST уровня репозитория через CodeQL (особенно если репозиторий в entity["company","GitHub","code hosting company"]): GitHub документирует готовые query suites для Go (`default`/`security-extended`). citeturn9search2turn9search10  
- Go‑specific security linter, например gosec (сканирует AST/SSA на типовые security issues). Это не «официальный» инструмент Go, поэтому его стоит держать как SHOULD, а не MUST, и фиксировать правила/исключения в репозитории. citeturn9search1  

### SBOM, provenance, SLSA и подпись артефактов как baseline поставки

1) **Сервис должен выпускать SBOM для релизного артефакта (как минимум для контейнерного образа).**  
- В терминах SPDX SBOM — это набор элементов, описывающих состав пакета, включая сведения о составе, лицензировании, provenance и т.п. citeturn7search0turn7search12  
- SPDX — международный стандарт (ISO/IEC 5962:2021). citeturn7search4turn7search8  
- CycloneDX — BOM/SBOM стандарт, развиваемый OWASP и изданный как ECMA‑424. citeturn7search13turn7search1  

   **Default выбора** (boring): генерировать **CycloneDX JSON** (операционно удобно для supply-chain use cases) и/или **SPDX JSON** (как ISO-ориентированный обменный формат). citeturn7search13turn7search4  

2) **Provenance/attestations должны быть доступны для релизных артефактов.** В SLSA «provenance» — это проверяемая информация, позволяющая отследить артефакт назад по цепочке поставки: где/когда/как он был произведён. citeturn1search20turn1search0  

3) **Сборка должна двигаться к SLSA Build Track минимум Level 2 как практический baseline.** SLSA формализует требования по уровням (source/build/provenance требования и т.д.); для template разумно зафиксировать: автоматизированная сборка + генерация provenance + аутентификация/подпись provenance/артефактов. citeturn1search0turn7search6  

4) **Контейнерные образы должны быть подписаны.** В entity["organization","Sigstore","software signing project"] Cosign поддерживает keyless‑подпись контейнеров через OIDC (эпhemeral keys), с командой уровня `cosign sign $IMAGE`. citeturn1search1turn1search25  

5) **Если используете BuildKit/современную Docker-сборку**, включайте provenance attestations: документация entity["company","Docker","container tooling company"] отмечает, что provenance attestations по умолчанию следуют SLSA provenance schema (по умолчанию v0.2) и могут быть переключены на v1. citeturn7search22turn7search14  

### Container image hardening как baseline

1) **Минимизируйте surface area образа.** Distroless‑подход: образы содержат только приложение и его runtime‑зависимости и не включают package managers, shells и прочие утилиты «обычного» дистрибутива. Это уменьшает площадь атаки и упрощает контроль состава. citeturn6search1turn6search9  

2) **Не воспринимайте “distroless” как серебряную пулю.** Известный практический контраргумент: «distroless» не «убирает ОС», а только сильно урезает user space; безопасность зависит от процессов обновления, сканирования, подписей и политик запуска. citeturn6search21turn12view0turn14view0  

3) **Не включайте SSH/remote shell tooling внутрь контейнеров.** entity["organization","NIST","us standards agency"] в SP 800‑190 явно указывает, что SSH и другие remote administration tools, дающие remote shells, не должны быть включены в контейнеры; контейнеры должны запускаться иммутабельно, а администрирование — через runtime APIs/оркестратор. citeturn14view1turn13view0  

4) **Валидация образов и “quality gates”.** NIST отдельно рекомендует использовать container‑specific vulnerability management tools и возможность предотвращать запуск non‑compliant images. citeturn12view0turn14view0  

Практический default для template:  
- выпускать минимальный runtime‑образ (distroless/scratch‑family) из multi-stage Dockerfile;  
- подписывать образ;  
- публиковать SBOM и provenance рядом с образом как часть релиза;  
- в CI сканировать образ (или SBOM) на CVE и конфигурационные дефекты (как SHOULD→MUST при повышении уровня зрелости). citeturn14view0turn15search0turn15search3  

### Runtime hardening: least privilege, seccomp/AppArmor, network policies, admission controls

1) **Target runtime baseline = Kubernetes Pod Security Standard “Restricted”** (как целевое состояние). Этот профиль прямо ориентирован на «current pod hardening best practices», хотя и ценой совместимости. citeturn1search2turn5view1  

2) **Enforcement через Pod Security Admission.** Kubernetes предоставляет встроенный admission controller, применяющий Pod Security Standards на уровне namespace (режимы enforce/audit/warn). citeturn3view2  

3) **Нормы Restricted, которые важно “вшить” в шаблон деплоя:**
- `allowPrivilegeEscalation: false` (не разрешать privilege escalation). citeturn16view2turn5view0  
- `runAsNonRoot: true` и запрет `runAsUser: 0`. citeturn16view2turn5view0  
- `seccompProfile.type` не должен быть `Unconfined`; разрешены `RuntimeDefault` или `Localhost` (а для некоторых версий/режимов “Restricted” требуется явная установка профиля). citeturn16view3turn3view3turn5view0  
- Linux capabilities: контейнеры должны drop’ать `ALL` и могут добавлять обратно только `NET_BIND_SERVICE` (Linux-only policy для определённых версий). citeturn16view0turn16view1  

4) **AppArmor как дополнительный слой.** Kubernetes документирует, как загружать AppArmor profiles на ноды и применять их к Pod’ам; фича отмечена как stable (v1.31, enabled by default). citeturn2search3turn5view0  

5) **Network policies: default deny.** Kubernetes показывает шаблоны “default deny all ingress and all egress” на namespace, позволяя затем явно открыть необходимые направления. citeturn1search3  

6) **Admission controls для образов и supply chain.** Kubernetes описывает admission controllers и, в частности, ImagePolicyWebhook как validating admission controller (отключён по умолчанию), что даёт точку интеграции для политик “что можно запускать”. citeturn8search3  

Практическая цель template: «не допускать в prod неподписанные/несоответствующие политики образы и workload’ы, которые не соответствуют Restricted». NIST отдельно подчеркивает ценность предотвращения запуска non‑compliant images и необходимость container‑specific vulnerability management. citeturn14view0turn12view0turn3view2  

## Decision matrix / trade-offs

Ниже — матрица решений, которые чаще всего вызывают споры. Default выбран как boring baseline; альтернативы описаны с условиями применимости и рисками.

image_group{"layout":"carousel","aspect_ratio":"16:9","query":["SLSA provenance levels diagram","Sigstore cosign keyless signing diagram","SPDX SBOM example JSON","Kubernetes Pod Security Standards restricted diagram"],"num_per_query":1}

### Go dependencies: публичный proxy/sumdb vs приватная инфраструктура

**Default:** использовать стандартную модель Go Modules: proxy + checksum database, не отключая sumdb. Go toolchain по умолчанию может скачивать модули через proxy и аутентифицировать их через checksum database; поведение управляется `GOPROXY`, `GOSUMDB`, `GOPRIVATE` и др. citeturn3view0turn3view1turn2search1  

**Когда нужна альтернатива:**  
- privacy/compliance требует не раскрывать module paths внешним сервисам; документация прямо говорит, что checksum database получает полный module path, и что для приватных модулей используются `GOPRIVATE`/`GONOSUMDB`. citeturn3view1  
- air‑gapped: нужен внутренний module proxy; Go допускает, что proxy может mirror’ить checksum database, чтобы клиент не ходил к sumdb напрямую. citeturn3view1  

**Риск/трейд‑офф:** чрезмерно широкий `GOPRIVATE` отключает для соответствующих префиксов и proxy, и sumdb‑валидацию; это повышает риск supply-chain tampering для “вдруг публичных” зависимостей, попавших под паттерн. citeturn3view1  

### Vendoring зависимостей

**Default:** не вендорить.  

**Альтернатива:** vendoring включать, если нужен полностью оффлайн build или репликация зависимостей как артефакта. Go явно поддерживает `-mod=vendor`. citeturn4view0  

**Риск/трейд‑офф:** vendor увеличивает репозиторий, усложняет обновления и диффы; без строгой дисциплины это может привести к “дрейфу” и несоответствию vendor ↔ go.mod. В шаблоне это компенсируется обязательным `go mod tidy -diff` и отдельным idiomatic процессом обновления. citeturn4view0turn8search10  

### SBOM формат: SPDX vs CycloneDX

**Default:** CycloneDX JSON как основной, SPDX JSON как опциональный “compat export”. CycloneDX — BOM/SBOM стандарт OWASP (ECMA‑424), SPDX — ISO стандарт. citeturn7search13turn7search4  

**Трейд‑офф:**  
- SPDX часто проще ложится в compliance‑контуры из‑за статуса ISO/IEC 5962:2021. citeturn7search4turn7search8  
- CycloneDX часто удобнее в AppSec/Supply Chain tooling и экосистеме OWASP. citeturn7search13turn7search9  

### Подпись: keyless (OIDC) vs long‑lived keys

**Default:** keyless‑подпись через Sigstore/Cosign (OIDC). Документация Sigstore описывает keyless flow и простую команду `cosign sign`. citeturn1search1turn1search25  

**Альтернатива:** ключи в HSM/KMS или классические ключи (GPG/PKI) — чаще применимо в regulated industries.  

**Трейд‑офф:** keyless снижает операционную стоимость управления ключами, но требует доверия к OIDC‑идентичности и прозрачному логу/экосистеме. При строгом compliance часто нужно “закрепить” trust policy и процессы верификации на стороне admission. citeturn1search1turn8search3  

### Runtime hardening: Restricted vs Baseline PSS

**Default target:** Restricted как “конечная” цель, но внедрять поэтапно через warn/audit → enforce; Kubernetes описывает режимы enforce/audit/warn. citeturn3view2turn5view1  

**Трейд‑офф:** Restricted может ломать workloads (например, требование seccomp, drop ALL capabilities и т.п.), поэтому в greenfield template разумно сразу сделать манифесты соответствующими Restricted, а в кластере — включать enforce тогда, когда инфраструктура готова. citeturn16view3turn16view0  

## Набор правил в формате MUST / SHOULD / NEVER для LLM

Далее — правила для LLM-инструкций в `docs/llm/` и как «policy» для автогенерации кода/CI/манифестов.

### MUST

LLM MUST:
- **Сохранять и уважать go.mod/go.sum как источник истины для зависимостей**, не предлагать “просто скачай latest”, не менять зависимости “для исправления ошибок” без указания причины и без соответствующих команд Go. citeturn8search10turn3view1  
- **В CI предлагать `GOFLAGS=-mod=readonly` и `go mod tidy -diff` как обязательные проверки**, чтобы сборка не модифицировала модульные файлы неявно. citeturn4view0  
- **Всегда добавлять `govulncheck` как обязательный шаг security**, и настраивать его так, чтобы pipeline падал при найденных уязвимостях (не использовать `-json/-format sarif/-format openvex` как единственный режим гейтинга без дополнительной логики). citeturn0search1turn8search9  
- **Встраивать в Kubernetes-манифесты securityContext под Restricted**: `allowPrivilegeEscalation: false`, `runAsNonRoot: true`, `seccompProfile: RuntimeDefault`, drop ALL capabilities (и добавлять только `NET_BIND_SERVICE`, если нужен порт <1024). citeturn16view2turn16view3turn16view0turn5view0  
- **Добавлять default deny NetworkPolicy** (ingress+egress) и затем явно разрешать только нужные направления. citeturn1search3  
- **Не предлагать SSH/remote shell внутри контейнеров**, и не включать соответствующие пакеты/демоны в образ, так как это противоречит иммутабельной модели контейнеров. citeturn14view1turn13view0  
- **Предусматривать процесс подписи релизных образов (Cosign) и публикации provenance/SBOM** как часть release pipeline. citeturn1search25turn7search22turn7search13  

### SHOULD

LLM SHOULD:
- **По умолчанию не отключать checksum database**, а для приватных модулей использовать `GOPRIVATE`/`GONOSUMDB` точечно по префиксам. citeturn3view1  
- **Добавлять `go mod verify`** как часть CI, чтобы фиксировать вмешательства в module cache. citeturn8search2  
- **Использовать минимальные/“distroless” runtime-образы** для уменьшения attack surface, но фиксировать, что это не заменяет сканирование, обновления и политики запуска. citeturn6search1turn6search21turn14view0  
- **Включать Pod Security Admission как механизм enforcement** (по крайней мере audit/warn в dev/stage) и указывать, что PSS применяется на уровне namespace. citeturn3view2  
- **Добавлять SAST в стиле CodeQL или аналогичный**, если репозиторий в GitHub или есть стандарт на code scanning; для Go GitHub документирует встроенные queries и suites. citeturn9search2turn9search10  
- **Использовать дополнительные линтеры безопасности (например, gosec) как “второй слой”**, но фиксировать исключения и не превращать baseline в “шумогенератор”. citeturn9search1turn0search21  
- **Пояснять трейд‑оффы** (например, Restricted vs Baseline, distroless vs debug‑friendly base), вместо того чтобы “тихо выбрать” потенциально несовместимый вариант. citeturn5view1turn6search21  

### NEVER

LLM NEVER:
- **НЕ предлагать `GOSUMDB=off` как “ускорение” или “фикс скачивания”** без явного признания, что это снижает security guarantee, и без альтернатив (например, корректная настройка приватных модулей). citeturn3view1  
- **НЕ считать, что `govulncheck -json` провалит CI при уязвимостях** (он вернёт 0). citeturn8search9  
- **НЕ генерировать Kubernetes-манифесты, которые требуют privileged, hostNetwork/hostPID/hostPath и т.п. как “дефолт”**, если это не обосновано; baseline должен соответствовать Restricted. citeturn5view1turn16view0  
- **НЕ включать shell/SSH/пакетный менеджер в runtime-образ “ради удобства дебага”**; для дебага используйте отдельный debug‑вариант образа/эпемерные инструменты, но не production runtime. citeturn6search1turn14view1  
- **НЕ встраивать секреты в образ или репозиторий** (даже “для теста”); NIST отдельно отмечает риск embedded clear text secrets и то, что secrets должны храниться вне образов и внедряться динамически в runtime. citeturn14view1turn13view0  

## Concrete good / bad examples

### Good: корректный CI-гейт для `govulncheck` (падает при уязвимостях)

```bash
# ✅ Good: default mode — CI упадёт, если найдены уязвимости
govulncheck ./...
```

Обоснование: `govulncheck` в обычном режиме возвращает non‑zero при найденных уязвимостях. citeturn8search9turn0search1  

### Bad: “красивый SARIF”, но нулевой exit code → ложный green build

```bash
# ❌ Bad: exit code будет 0 даже при найденных уязвимостях
govulncheck -format sarif ./... > govulncheck.sarif
```

Почему плохо: документация явно говорит, что при `-format sarif` (а также `-json`/`-format openvex`) govulncheck “exits successfully regardless of the number of detected vulnerabilities”. citeturn8search9  

Если нужен SARIF для UI, делайте **двойной запуск**: первый — “гейт” (default), второй — “репорт” (SARIF). citeturn8search9turn0search37  

### Good: верификация модулей и запрет дрейфа модульных файлов в CI

```bash
# ✅ Good: зависимостям нельзя "дрейфовать"
export GOFLAGS="-mod=readonly"

go mod tidy -diff
go mod verify
go test ./...
go vet ./...
govulncheck ./...
```

- `-mod=readonly` запрещает неявный апдейт модульных файлов. citeturn4view0  
- `go mod tidy -diff` даёт воспроизводимую проверку соответствия кода и go.mod/go.sum. citeturn4view0  
- `go mod verify` проверяет, что зависимости в module cache не модифицированы. citeturn8search2  

### Bad: “починим приватные модули” через отключение sumdb для всех

```bash
# ❌ Bad: ломает гарантию аутентификации модулей для всего проекта
export GOSUMDB=off
```

Даже документация Go отмечает, что `GOSUMDB=off` отключает обращения к checksum database и тем самым снижает security guarantee. Правильнее — использовать `GOPRIVATE`/`GONOSUMDB` по префиксам приватных модулей. citeturn3view1  

### Good: Kubernetes securityContext под “Restricted” (минимально совместимый пример)

```yaml
# ✅ Good: соответствует базовым ожиданиям Restricted
securityContext:
  allowPrivilegeEscalation: false
  runAsNonRoot: true
  seccompProfile:
    type: RuntimeDefault
  capabilities:
    drop: ["ALL"]
    # add: ["NET_BIND_SERVICE"]  # только если действительно нужно слушать <1024
```

Эти требования прямо отражены в Kubernetes Pod Security Standards (Restricted): запрет privilege escalation, требование runAsNonRoot, seccomp RuntimeDefault/Localhost (без Unconfined), drop ALL capabilities и разрешение добавлять обратно только NET_BIND_SERVICE. citeturn16view2turn16view3turn16view1  

### Bad: типовые “LLM‑шаблоны” для манифестов, которые ломают baseline

```yaml
# ❌ Bad: фактически выключает hardening
securityContext:
  privileged: true
  allowPrivilegeEscalation: true
  seccompProfile:
    type: Unconfined
  capabilities:
    add: ["NET_ADMIN"]
```

- “Privileged” запрещён в Restricted. citeturn16view0  
- `allowPrivilegeEscalation` должен быть `false`. citeturn16view2  
- `seccompProfile` не должен быть `Unconfined`. citeturn16view3  
- Добавление capabilities сверх разрешённых в Restricted должно быть запрещено. citeturn16view0turn16view1  

### Good: запрет SSH/remote shells в контейнере — как правило для образа и Dockerfile

```dockerfile
# ✅ Good: runtime stage не содержит shell/ssh и использует минимальный базовый образ
# (конкретный base выбирайте по вашей платформе/регистри)
FROM gcr.io/distroless/static:nonroot
COPY service /service
USER nonroot:nonroot
ENTRYPOINT ["/service"]
```

Distroless подход: нет package managers/shells и т.п. citeturn6search1turn6search9  
NIST: SSH и remote shells внутри контейнеров не должны быть включены. citeturn14view1  

## Anti-patterns и типичные ошибки/hallucinations LLM

### Ошибки вокруг Go modules / supply chain

1) **Hallucination: “отключим проверки sumdb, чтобы стало стабильнее/быстрее”.** На самом деле `GOSUMDB=off` — это сознательный отказ от части гарантий аутентификации модулей. Для приватных модулей корректнее настроить `GOPRIVATE`/`GONOSUMDB`, а не отключать все проверки. citeturn3view1  

2) **Hallucination: “govulncheck в SARIF режиме провалит пайплайн”.** Он вернёт 0 независимо от уязвимостей; это documented behavior. citeturn8search9  

3) **LLM‑ошибка: игнорировать `go mod tidy -diff` и принимать PR, где go.mod/go.sum изменены вручную.** Go документация подчёркивает, что использование go command для управления зависимостями помогает сохранять консистентность требований и валидность go.mod. citeturn8search10  

4) **Случайная деградация трассируемости артефакта.** Убирание build metadata без причины затрудняет supply-chain аудит. Go tooling поддерживает `-buildvcs` для простановки информации VCS в бинарь (и управление режимом auto/true/false). citeturn4view3  

### Ошибки вокруг контейнеров и runtime hardening

1) **“Поставим SSH, чтобы проще дебажить в проде”.** NIST прямо рекомендует не включать SSH и remote shell tooling внутрь контейнеров и управлять ими через APIs оркестратора/рантайма. citeturn14view1turn13view0  

2) **“Сделаем образ на full OS (apt/yum) и будем патчить внутри контейнера”.** NIST подчёркивает иммутабельность контейнеров как ключевой принцип; обновления должны происходить “upstream в images” с последующим redeploy, а не “в поле”. citeturn13view0turn14view1  

3) **“Запустим как root, потом ограничим”.** Restricted PSS требует runAsNonRoot и запрет runAsUser=0; privilege escalation запрещён; capabilities должны быть drop ALL. Это не “nice to have”, а baseline enforcement‑модель. citeturn16view2turn16view0turn16view1  

4) **“seccomp = Unconfined, иначе может сломаться”.** Kubernetes PSS Restricted запрещает явный Unconfined; а Kubernetes документация по seccomp описывает RuntimeDefault как сильные defaults, сохраняющие функциональность, хотя и зависящие от runtime. citeturn16view3turn3view3turn2search6  

5) **Отсутствие network segmentation по умолчанию.** Без NetworkPolicy в кластере часто получается “flat network”. Kubernetes документирует подход default deny ingress+egress как базовый шаблон для последующего явного открытия нужного. citeturn1search3  

### Ошибки вокруг enforcement/admission

1) **“Мы подписываем образы, значит всё ок”.** Подпись без enforcement = декларация без контроля. Kubernetes предоставляет точки расширения через admission controllers; Pod Security Admission реализует enforcement PSS, а image policies требуют отдельной политики/интеграции (например, через webhook‑подход, где ImagePolicyWebhook — один из механизмов). citeturn3view2turn8search3turn1search25  

2) **“Нужно сразу enforce restricted на весь кластер”.** В greenfield template правильнее обеспечить manifests, совместимые с Restricted, но rollout enforcement делать поэтапно (warn/audit → enforce), иначе рискуете сломать системные компоненты/легаси workloads. Kubernetes описывает режимы enforce/audit/warn на namespace. citeturn3view2turn5view1  

## Review checklist для PR/code review

Этот чеклист можно почти напрямую перенести в `docs/review/security.md` и использовать как шаблон PR.

### Supply chain / dependencies

- PR **не меняет `go.mod`/`go.sum` вручную**; изменения объяснены (почему, какой риск/патч), и есть подтверждение, что `go mod tidy -diff` чист. citeturn4view0turn8search10  
- В CI включён `GOFLAGS=-mod=readonly`. citeturn4view0  
- Есть шаг `go mod verify`. citeturn8search2  
- Нет необоснованного `GOSUMDB=off`; приватные зависимости настраиваются через `GOPRIVATE`/`GONOSUMDB` по префиксам. citeturn3view1  
- `govulncheck` запускается как гейт (mode, который фейлит билд при уязвимостях), и нет “фиктивного зелёного” в SARIF/JSON‑only режиме. citeturn8search9turn0search37  

### Build/release artifacts

- Релизный pipeline публикует SBOM (SPDX и/или CycloneDX), формат документирован. citeturn7search4turn7search13  
- Публикуется provenance/attestation для артефакта, и есть понимание уровня SLSA, к которому вы стремитесь (как минимум: автоматизированная сборка + provenance). citeturn1search0turn1search20  
- Образ подписан (Cosign keyless или другой механизм), а политика потребления/проверки подписи прописана. citeturn1search25turn1search1  

### Container image hardening

- Runtime‑образ минимален (distroless/minimal) и **не содержит shell/SSH/package manager**; для дебага есть отдельная стратегия (отдельный debug образ или ephemeral tooling), но не production runtime. citeturn6search1turn14view1  
- Нет попыток “администрировать контейнер через SSH”; контейнеры остаются иммутабельными. citeturn14view1turn13view0  

### Runtime hardening / Kubernetes

- Манифесты соответствуют целевому уровню Restricted: `allowPrivilegeEscalation=false`, `runAsNonRoot=true`, seccomp не `Unconfined`, drop `ALL` capabilities (и добавление только `NET_BIND_SERVICE` если требуется). citeturn16view2turn16view3turn16view1  
- Pod Security Admission применяется в нужных namespace (не обязательно enforce сразу везде, но политика управления уровнями documented). citeturn3view2  
- Есть default deny NetworkPolicy и явно описанные allow‑правила. citeturn1search3  
- Рассмотрен AppArmor как дополнительный слой, если платформа поддерживает (документированная процедура). citeturn2search3turn5view0  

## Что из результата нужно оформить отдельными файлами в template repo

Ниже — конкретная раскладка файлов, чтобы результат превращался в «repo conventions» и LLM‑instruction docs без доизобретения.

### Документы стандарта (docs/)

- `docs/security/baseline.md` — этот baseline: обязательные требования, rationale, ссылки на первичные источники. citeturn12view0turn3view1turn5view1  
- `docs/security/supply-chain.md` — отдельно: Go modules integrity (sumdb/proxy/private modules), политика зависимостей, правила обновлений, как трактовать `GOPRIVATE`, как и почему запрещён `GOSUMDB=off`. citeturn3view1turn2search1  
- `docs/security/ci-gates.md` — “CI/CD security gates”: go vet, govulncheck (с нюансом exit codes), go mod tidy -diff, go mod verify, плюс опциональные SAST (CodeQL/gosec). citeturn9search0turn0search37turn8search9turn9search2  
- `docs/security/runtime-hardening-k8s.md` — Kubernetes securityContext под Restricted, seccomp RuntimeDefault, capabilities, AppArmor, network policies, Pod Security Admission rollout. citeturn5view0turn16view3turn1search3turn3view2turn2search3  
- `docs/security/sbom-provenance-signing.md` — SBOM (SPDX/CycloneDX), provenance/SLSA, подпись образов (Cosign), связка “build → attest → sign → enforce”. citeturn7search4turn7search13turn1search20turn1search25  

### LLM‑инструкции (docs/llm/)

- `docs/llm/security.md` — правила MUST/SHOULD/NEVER (прямо как в разделе выше) + “don’t hallucinate” секция о govulncheck exit codes, sumdb и Restricted PSS, чтобы модель не допускала типовых провалов. citeturn8search9turn3view1turn16view0  
- `docs/llm/review-prompt.md` — готовый prompt-шаблон для LLM code review по security checklist, с требованием ссылаться на конкретные файлы/дифф и отмечать нарушения baseline.

### Репозиторные конвенции и CI/CD

- `.github/workflows/security.yml` — обязательные шаги: `go mod tidy -diff`, `go mod verify`, `go vet`, `govulncheck` (в режиме, который фейлит билд), плюс опционально CodeQL. citeturn4view0turn8search2turn8search9turn9search2  
- `.github/workflows/release.yml` — build с `-trimpath` и контролем `-buildvcs`, выпуск SBOM/provenance, подпись образа; хранение артефактов релиза рядом с образом. citeturn4view2turn4view3turn7search22turn1search25  
- `deploy/kubernetes/` (или `charts/`): манифесты по умолчанию соответствуют Restricted PSS + NetworkPolicy default deny; документация по Pod Security Admission rollout. citeturn5view1turn1search3turn3view2  
- `SECURITY.md` — политика disclosure, минимальные ожидания по supply chain и vulnerability handling (даже для внутреннего репозитория это дисциплинирует процесс). (Опирается на подходы зрелых сообществ; конкретные механики в текущей теме не регламентируются первичным источником, поэтому файл рекомендуется как организационный дефолт, но без “фальшивых” нормативных ссылок.)