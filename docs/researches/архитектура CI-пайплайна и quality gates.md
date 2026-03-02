# Стандарты container image, Dockerfile и CI quality gates для production-ready Go microservice template

## Scope

Этот стандарт применим, когда вы делаете **greenfield микросервис на Go**, который будет запускаться в контейнере (entity["company","Docker","docker inc"]) и/или оркестрироваться в entity["organization","Kubernetes","container orchestration"], и хотите, чтобы разработчик мог **склонировать репозиторий и сразу писать production-код**, а LLM-инструменты генерировали **идиоматичный, безопасный, поддерживаемый и предсказуемый** Go-код и инфраструктурные изменения без «догадок». citeturn13search0turn12view0turn15view0

Стандарт особенно полезен, если:
- вы хотите **boring defaults**: многократно проверенные практики (multi-stage, minimal runtime, non-root, понятные CI gates), а не «оптимизации ради оптимизаций»; citeturn13search0turn13search4turn33view0
- вы планируете целиться в **Pod Security Standards Restricted** и подобные политики: `runAsNonRoot`, `allowPrivilegeEscalation: false`, `seccompProfile: RuntimeDefault`, drop capabilities; citeturn16view0turn16view3
- вы хотите, чтобы контейнерный runtime и CI pipeline **минимизировали двусмысленность** для LLM (какой базовый образ, какие флаги сборки, какие проверки обязательны). citeturn12view0turn17view0turn21view3

Не применять «как есть», если:
- сервис **не контейнеризуется** (pure VM/bare-metal без OCI-образов) — часть требований (Dockerfile, образ, сканирование образа) будет лишней; citeturn13search0turn13search4
- сервис **существенно зависит от CGO**/системных библиотек, OpenSSL legacy-провайдеров, специфических драйверов, или требует «толстого» runtime (командные утилиты, shell-скрипты как часть продукта): стандарт можно адаптировать, но default `distroless/static + CGO_ENABLED=0` будет неприменим; citeturn33view0turn12view0
- вы делаете **CLI/утилиту**, где требования к сигналам, runtime, размеру образа и probes отличаются (хотя часть практик всё равно полезна). citeturn30view0turn13search0

## Recommended defaults для greenfield template

Ниже — **нормативный baseline** для template repo (containerization + CI quality gates). Это можно почти напрямую выносить в `docs/` и «repo conventions».

### Default runtime образ и базовые принципы

**Default**: `gcr.io/distroless/static-debian12:nonroot` как runtime-стадия (для статически собранного Go binary без libc). Причины:

- distroless-образы **не содержат package manager и shell**, что уменьшает поверхность атаки и заставляет не полагаться на «внутриконтейнерную интерактивность»; citeturn12view0
- `distroless/static` по документации содержит **CA certificates, tzdata, /tmp, /etc/passwd entry**, что закрывает типовые проблемы «scratch» (TLS, time zones, временные файлы, user resolution); citeturn33view0
- есть официальные теги `nonroot` / `debug` / `debug-nonroot`; при необходимости отладки можно собирать debug-вариант, не размазывая инструменты в production runtime; citeturn12view0
- distroless рекомендует **явно указывать Debian suffix** (`-debian12`), потому что «без суффикса» сейчас ведёт на debian12, но это будет изменено на более новую версию Debian в будущем (риск «ломающих» апдейтов); citeturn12view0

**Linking default**: `CGO_ENABLED=0` (статическая сборка), потому что `distroless/static` предназначен для приложений, **не требующих libc/cgo**. Для приложений, требующих libc/cgo — переход на `distroless/base(-nossl)` согласно их описанию. citeturn33view0turn25search1

### Default Dockerfile (production)

Ниже — стандартный Dockerfile, который:
- использует multi-stage сборку; citeturn13search4turn13search0  
- делает reproducible-ish сборку (`-trimpath`, контроль `-buildvcs`, `-mod`); citeturn4view1turn4view0turn5view3  
- использует BuildKit cache mounts (ускорение, но не влияет на корректность); citeturn18view0  
- гарантирует non-root runtime; citeturn12view0turn11view1turn16view0  
- использует exec form entrypoint, что важно и для distroless (нет shell), и для корректной доставки сигналов; citeturn12view0turn17view0  

```Dockerfile
# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26.0

############################
# Build stage
############################
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build
WORKDIR /src

# 1) Dependencies first for better layer caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# 2) Copy the rest
COPY . .

# Build args for cross-build with buildx
ARG TARGETOS
ARG TARGETARCH

# Default: static binary (no libc/cgo) for distroless/static
ENV CGO_ENABLED=0

# Reproducibility-oriented flags:
# -trimpath: remove local paths
# -buildvcs=false: avoid embedding VCS info (optional policy)
# -mod=readonly: forbid implicit go.mod/go.sum changes
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build \
      -trimpath \
      -buildvcs=false \
      -mod=readonly \
      -o /out/service \
      ./cmd/service

############################
# Runtime stage
############################
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

# Optional: be explicit about workdir (avoid surprises from base image defaults)
WORKDIR /

# Copy binary
COPY --from=build /out/service /service

# Nonroot user in distroless (documented constant)
USER 65532:65532

# Port is optional metadata; Kubernetes uses containerPort in manifests
EXPOSE 8080

# Distroless has no shell: must be exec/vector form
ENTRYPOINT ["/service"]
```

Обоснования ключевых строк:
- multi-stage — официальная рекомендация для минимизации финального образа и разделения build/runtime; citeturn13search4turn13search0
- `RUN --mount=type=cache` — Dockerfile reference: кэшировать директории для компиляторов/пакетных менеджеров; кэш **только для производительности**, сборка обязана быть корректной независимо от содержимого кэша; citeturn18view0
- `-trimpath` уменьшает зависимость бинарника от путей на build host; citeturn4view1
- `-buildvcs` управляет встраиванием VCS-информации (по умолчанию `auto`), но для шаблона часто лучше выключить и явно управлять версионированием в release job; citeturn4view0
- `-mod` по умолчанию ведёт себя как `readonly` (или `vendor` при наличии vendor dir для go>=1.14), но для шаблона лучше фиксировать намерение явно; citeturn5view3
- distroless требует exec/vector form entrypoint и не содержит shell; citeturn12view0
- UID `65532` как `NONROOT` определён в distroless sources; citeturn11view1
- `distroless/static` содержит CA certs и tzdata — критично для HTTPS и time zones; citeturn33view0turn25search0

### Default .dockerignore (обязателен)

**Стандарт**: `.dockerignore` MUST существовать и MUST исключать секреты/мусор/артефакты сборки. Docker прямо рекомендует исключать ненужное через `.dockerignore`, а build context в целом описывает как важную часть процесса. citeturn13search0turn13search1

Минимальный baseline:

```gitignore
# VCS
.git
.gitignore

# Local dev / IDE
.idea
.vscode

# Build output
bin/
dist/
out/
*.test

# OS junk
.DS_Store

# Secrets / env
.env
**/*.pem
**/*.key
**/*secret*
```

### Kubernetes runtime defaults, которые Dockerfile «предполагает»

Этот стандарт предполагает, что runtime policy стремится к **PSS Restricted**. Минимальный набор (pod/container securityContext):

- `runAsNonRoot: true` (обязательное требование restricted); citeturn16view0  
- `runAsUser: 65532` (любой non-zero допустим; 65532 соответствует distroless `NONROOT`); citeturn16view0turn11view1  
- `allowPrivilegeEscalation: false`; citeturn16view0  
- `seccompProfile.type: RuntimeDefault`; citeturn16view0  
- `capabilities.drop: ["ALL"]` (и только при необходимости `NET_BIND_SERVICE`, но для Go-сервисов на портах >1024 обычно не нужно); citeturn16view3turn33view0  

Также стандарт предполагает **graceful shutdown** на SIGTERM: Kubernetes при удалении Pod даёт grace period (по умолчанию 30 секунд) и затем принудительно завершает. citeturn0search8turn31view1

## Decision matrix / trade-offs

Ниже — ключевые развилки, которые должны быть явно «закодированы» в стандартах и LLM-инструкциях, чтобы не было скрытых предположений.

### Distroless vs scratch vs Alpine/Debian

**Distroless (рекомендуется по умолчанию)**
- Плюсы: минимальный runtime без shell/package manager; есть `debug` варианты; `static` включает CA/tzdata/etc; хорошие размеры; citeturn12view0turn33view0
- Минусы: отладка «через exec внутрь» затруднена (нужны debug images или внешние методы); нельзя «доустановить пакет» в рантайме (и это хорошо как policy, но требует дисциплины). citeturn12view0

**scratch (не default)**
- Плюсы: максимально минимально по размеру.
- Минусы: легко сломать TLS (нет CA), time zones, `/tmp`, user resolution; вы берёте на себя ручное наполнение; distroless прямо перечисляет эти «базовые» вещи как содержимое `static`. citeturn33view0turn28view0turn25search0

**Alpine/Debian slim (не default, но иногда оправдано)**
- Плюсы: проще интерактивная диагностика, проще ставить утилиты, привычнее многим опсам.
- Минусы: больше пакетов → больше CVE и шум сканеров; выше риск «в контейнере есть то, что не нужно». distroless подчёркивает отсутствие package manager/shell как принцип. citeturn12view0turn13search0

**Практический выбор**:
- если у сервиса нет CGO — `distroless/static-debian12:nonroot`; citeturn33view0turn12view0  
- если нужен CGO/libc — `distroless/base(-nossl)-debian12:nonroot` (или `cc-...`, если нужны дополнительные runtime libs); citeturn33view0turn12view0  

### Static vs dynamic linking (CGO)

**Static (`CGO_ENABLED=0`)**
- Работает с `distroless/static`; уменьшает зависимость от libc/cgo; citeturn33view0
- Может менять поведение DNS resolver’а: `netgo` build tag полностью отключает cgo resolver; в целом Go различает go/cgo резолверы и их предпочтения. Это важно, если у вас специфические требования к resolv.conf, NSS, search domains. citeturn25search1turn25search5

**Dynamic (`CGO_ENABLED=1`)**
- Нужно, если используете драйверы/библиотеки, требующие libc/cgo.
- Тогда runtime base MUST содержать необходимые shared libraries (distroless/base или cc), и это должно быть отражено в Dockerfile standard и decision matrix. citeturn33view0turn12view0

### CA certs и пользовательские trust stores

Базовый принцип: **полагаться на system roots**, которые есть в distroless/static, и добавлять кастомные CA только декларативно.

- Go использует system cert pool; его можно переопределять `SSL_CERT_FILE`/`SSL_CERT_DIR` (Unix кроме macOS), а также предусмотрен механизм fallback roots для сред без root bundle (например, контейнеры без CA). citeturn28view0turn28view3  
- В production шаблоне **НЕ рекомендуется** «встраивать» произвольные fallback roots без governance: это превращается в скрытую supply-chain зависимость. Если уж нужно — делайте это как отдельное архитектурное решение с ревью. citeturn28view3turn6search3

### tzdata: OS vs embed в бинарник

- Distroless/static включает tzdata; citeturn33view0  
- Для extreme-minimal образов или нестандартных окружений возможно встраивать tzdata в бинарник через `time/tzdata` (примерно +450 KB) или build tag `timetzdata`. citeturn25search0turn25search2  
**Default**: не импортировать `time/tzdata` в шаблоне, если runtime образ гарантированно включает tzdata. Включать только если вы сознательно хотите независимость от OS tzdata.

### Image size vs operability

- distroless debug images дают busybox shell и явно рекомендуются как способ диагностики (без включения shell в production image); citeturn12view0  
**Default policy**: production image — минимальный, debug — отдельная сборка/тег/target.

### CI quality gates: строгие vs «шумные»

- «Слишком много линтеров» даёт шум и демотивирует; но базовые gates (format, test, vet, vulncheck, drift) должны быть железными.
- Go tooling даёт хорошие первичные гарантии: `go fmt` = `gofmt -l -w`; `go test` прогоняет high-confidence subset `go vet` и при ошибках vet не запускает тест-бинарник; это сильный baseline. citeturn22view0turn21view3  

## Набор правил в формате MUST / SHOULD / NEVER для LLM

Ниже — правила, которые должны попасть в LLM-instruction docs шаблона (например, `docs/llm/…`). Они формулируются так, чтобы модель **не “изобретала” инфраструктуру**, а следовала стандарту.

### MUST

1) **MUST использовать multi-stage Docker build**: build-стадия и runtime-стадия разделены; финальный образ содержит только runtime артефакты. citeturn13search4turn13search0

2) **MUST использовать distroless runtime образ по умолчанию**:
- `gcr.io/distroless/static-debian12:nonroot` для `CGO_ENABLED=0`;
- если CGO/libc обязателен — переключать на `gcr.io/distroless/base(-nossl)-debian12:nonroot` (и документировать почему). citeturn33view0turn12view0

3) **MUST задавать ENTRYPOINT/CMD в exec (vector) form**, особенно для distroless (нет shell). citeturn12view0turn17view0

4) **MUST обеспечивать non-root runtime** (в Dockerfile и Kubernetes manifests):
- Dockerfile: `USER 65532:65532` (или эквивалент); UID `65532` соответствует distroless `NONROOT`; citeturn11view1turn16view0
- Kubernetes: `runAsNonRoot: true`, `allowPrivilegeEscalation: false`, `seccompProfile: RuntimeDefault`, `capabilities.drop: ["ALL"]`. citeturn16view0turn16view3

5) **MUST включать `.dockerignore` и не добавлять секреты в build context**. citeturn13search0turn13search1

6) **MUST обеспечивать reproducibility-oriented сборку** в container build:
- `go build -trimpath`; citeturn4view1
- контролировать `-buildvcs` (default `auto` → политика должна быть явной); citeturn4view0
- `-mod=readonly` (или иное явно принятое решение). citeturn5view3

7) **MUST учитывать Kubernetes termination semantics**: сервис обязан корректно завершаться по SIGTERM в рамках grace period (по умолчанию 30s) и закрывать HTTP server gracefully. citeturn0search8turn31view1turn31view2

8) **MUST учитывать CI gates**: любой PR, который добавляет код/файлы, должен быть совместим с обязательными проверками:
- форматирование `go fmt` (gofmt); citeturn22view0
- тесты `go test ./...` (желательно `-count=1` в CI), включая встроенный vet subset; citeturn21view3turn21view2
- генерация (если в репо принято): `go generate` запускается *только явно*, значит CI должен запускать и проверять отсутствие drift. citeturn22view0turn5view3
- vulnerability scanning на Go deps: `govulncheck`. citeturn5search0turn5search2

### SHOULD

1) **SHOULD использовать BuildKit cache mounts** для ускорения `go mod download` и `go build`, но не «завязывать» корректность сборки на кэш. citeturn18view0

2) **SHOULD разделять build targets**:
- `runtime` (production),
- `runtime-debug` (debug image на базе `:debug-nonroot`). citeturn12view0

3) **SHOULD делать race checks** хотя бы на одной платформе в CI (`go test -race ./...`), учитывая ограничения поддержки `-race`. citeturn4view2turn21view3

4) **SHOULD фиксировать версию Go в template** на актуальную stable (на дату 2026‑03‑02 это Go 1.26.0, опубликован 10 Feb 2026) и следовать policy поддержки релизов. citeturn3search1turn3search0

5) **SHOULD включать supply-chain практики** в release pipeline (на main/release):
- SBOM/provenance/attestations по SLSA/in-toto; citeturn24search0turn24search35
- подпись контейнерных образов (например, keyless signing). citeturn24search3turn12view0  
Это напрямую связано с рисками цепочки поставки, которые отражаются и в современных security guidance. citeturn6search3turn6search7

### NEVER

1) **NEVER использовать shell form ENTRYPOINT/CMD** в distroless (не будет работать) и вообще избегать shell form там, где нужны корректные сигналы и прозрачность. citeturn12view0turn17view0

2) **NEVER добавлять “удобства” (curl, bash, apt/apk) в production runtime образ**. Если нужно — используйте debug image или отдельный tooling контейнер. citeturn12view0

3) **NEVER хранить или прокидывать секреты через `ARG`/Docker build args** (они видны и в history, и в provenance при определённых режимах). Использовать секреты BuildKit (`RUN --mount=type=secret`) либо секреты CI. citeturn17view0turn18view0

4) **NEVER полагаться на то, что `go generate` «само где-то запустится»** — оно *никогда* не запускается автоматически `go build/test`. citeturn22view0

5) **NEVER отключать обязательные CI gates “временно”** без явного исключения и документации (иначе шаблон деградирует и LLM будет воспроизводить деградацию). Логика: CI gates — часть спецификации системы, не “опция”. citeturn13search0turn6search3

## Concrete good / bad examples

### Good: graceful shutdown для HTTP сервера (SIGTERM → Shutdown)

```go
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Parent context cancels on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Run server in background.
	go func() {
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()

	// Kubernetes default grace is finite; use bounded shutdown ctx.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
```

Почему это «правильно» для контейнера/Кубера:
- по умолчанию SIGTERM завершает Go-процесс; через `NotifyContext` мы перехватываем сигнал и делаем controlled shutdown; citeturn31view3turn31view2
- `http.Server.Shutdown` выполняет graceful shutdown: закрывает listeners, закрывает idle conns и ждёт активные до idle, с учётом контекста; citeturn31view1
- Kubernetes сначала посылает SIGTERM и даёт grace period (по умолчанию 30s), затем может принудительно завершить — значит таймаут shutdown должен быть < grace period. citeturn0search8

### Bad: «неуправляемое» завершение

```go
func main() {
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Проблемы:
- нет контроля остановки; в Kubernetes shutdown может оборвать активные запросы;
- LLM часто генерирует это как «самый простой старт», но для production template это анти-стандарт. citeturn31view1turn0search8

### Good: Dockerfile entrypoint в vector form (distroless)

```Dockerfile
FROM gcr.io/distroless/static-debian12:nonroot
COPY myservice /myservice
ENTRYPOINT ["/myservice"]
```

Distroless прямо требует vector form, потому что shell отсутствует. citeturn12view0

### Bad: Dockerfile shell form (сломается на distroless)

```Dockerfile
FROM gcr.io/distroless/static-debian12:nonroot
COPY myservice /myservice
ENTRYPOINT "/myservice"
```

Distroless README: «must be specified in vector form». citeturn12view0

### Good: CI gate «format + tests + generated drift»

Провайдер-агностичное описание job (в виде шагов):

1) `go fmt ./...` (или `gofmt -w` по файлам). `go fmt` запускает `gofmt -l -w` и печатает изменённые файлы. citeturn22view0  
2) `go test -count=1 ./...` (в CI). `-count=1` — идиоматичный способ отключить caching. citeturn21view2turn21view3  
3) `go test -race ./...` (на поддерживаемой платформе, хотя бы nightly или на main). citeturn4view2  
4) `govulncheck ./...` (блокирующий gate). citeturn5search2turn5search0  
5) `go generate ./...` + `git diff --exit-code` (если в репо принято). Go generate не запускается автоматически, значит drift надо ловить явно. citeturn22view0  

## Anti-patterns и типичные ошибки/hallucinations LLM

### Контейнеризация

1) **“Alpine по умолчанию” без явного решения**  
LLM часто выбирает `alpine` как «маленький» образ и добавляет `apk add ca-certificates`, но это вводит distro-специфичность и CVE-шум (а distroless закрывает основную мотивацию: минимальный runtime). citeturn12view0turn33view0

2) **Shell form ENTRYPOINT/CMD в distroless**  
Это частая ошибка: модель «видит» ENTRYPOINT строкой и не учитывает, что distroless без shell. Distroless явно предупреждает. citeturn12view0

3) **“Доустановить пакеты в runtime”**  
LLM может предложить `RUN apt-get ...` в финальной стадии — на distroless это невозможно как минимум из-за отсутствия package manager/shell. Правильный подход: всё готовится в build stage. citeturn12view0turn13search4

4) **Секреты через `ARG`/ENV в Dockerfile**  
Dockerfile reference предупреждает, что build args не должны использоваться для секретов; вместо этого — `RUN --mount=type=secret`. citeturn17view0turn18view0

5) **Неверная уверенность в кэше BuildKit**  
LLM иногда делает сборку «зависящей» от кэша. Docker прямо говорит: cache mounts только для performance; корректность сборки не должна зависеть от содержимого кэша. citeturn18view0

6) **Сборка с CGO по умолчанию и запуск на `distroless/static`**  
Если CGO требуется, `distroless/static` неверен. Distroless base README даёт прямой выбор: `static` без libc, `base(-nossl)` с libc. citeturn33view0

7) **Отсутствие CA/tzdata в extreme-minimal образах**  
Для HTTPS Go использует system cert pool, который может отсутствовать; Go также предоставляет fallback mechanisms, но стандартный production путь — иметь CA bundle в образе (distroless/static содержит ca-certificates). citeturn33view0turn28view0  
Для time zones — либо tzdata в образе, либо явное встраивание `time/tzdata` (+~450KB). citeturn25search0turn33view0

### CI/CD / release engineering

1) **“go test достаточно, go vet не нужен” (в неверном смысле)**  
`go test` действительно запускает subset `go vet` и не запускает тестовый бинарь при ошибках vet; но при добавлении собственных статических анализаторов это надо документировать как отдельные gates. citeturn21view3

2) **“go generate запускается в go test/build”**  
Нет: `go generate` никогда не запускается автоматически. Если генерация важна — CI должен запускать и проверять drift. citeturn22view0

3) **Непонимание test caching**  
LLM может не учитывать, что `go test ./...` в package-list mode кэширует успешные результаты. В CI часто нужно `-count=1`, чтобы исключить «ложную зелень». citeturn21view3turn21view2

4) **Модульные изменения “втихаря”**  
Если шаблон требует `-mod=readonly` в сборке, то любые изменения зависимостей должны явно отражаться через обновление `go.mod/go.sum` (и хорошо иметь `go mod tidy -diff` gate). citeturn5view3

## Review checklist для PR / code review

Этот чеклист рассчитан на PR, который меняет код сервиса, Dockerfile, Kubernetes манифесты или CI.

**Контейнер / Dockerfile**
- Dockerfile multi-stage, runtime stage не содержит toolchain. citeturn13search4turn13search0  
- Финальный образ по умолчанию `distroless/static-debian12:nonroot` (или обоснованный switch на `base(-nossl)` из-за CGO). citeturn33view0turn12view0  
- ENTRYPOINT/CMD в exec form; для distroless это обязательное условие. citeturn12view0turn17view0  
- Non-root: Dockerfile `USER` и (если манифесты есть) `runAsNonRoot/runAsUser`. citeturn11view1turn16view0  
- `.dockerignore` присутствует и исключает секреты/артефакты. citeturn13search0turn13search1  
- Сборка использует `-trimpath`, явную политику `-buildvcs`, и `-mod=readonly`. citeturn4view1turn4view0turn5view3  
- Если используются BuildKit cache mounts — они не влияют на корректность сборки. citeturn18view0  

**Kubernetes runtime**
- SecurityContext соответствует Restricted intent: `allowPrivilegeEscalation: false`, `runAsNonRoot: true`, seccomp RuntimeDefault, drop ALL caps. citeturn16view0turn16view3  
- Приложение корректно обрабатывает SIGTERM и завершает работу через graceful shutdown в пределах grace period (дефолт 30s). citeturn0search8turn31view1turn31view2  

**CI gates**
- `go fmt`/gofmt соблюдён. citeturn22view0  
- `go test -count=1 ./...` зелёный; vet-ошибок нет. citeturn21view3turn21view2  
- Race-check (если включён в текущем gate наборе) зелёный и запускается на поддерживаемой платформе. citeturn4view2  
- `govulncheck` зелёный или есть документированное исключение/обоснование. citeturn5search2turn5search0  
- Если в репо есть генерация — `go generate` прогнан и нет `git diff`. citeturn22view0  
- Docker image build проходит (как минимум `docker build`). citeturn13search0  

## Что из результата нужно оформить отдельными файлами в template repo

Ниже — «разбиение» на файлы, чтобы это стало внутренним стандартом и работало как repo conventions + LLM instructions.

- `Dockerfile` — default multi-stage build (production). Основан на стандарте выше. citeturn13search4turn12view0turn33view0  
- `Dockerfile.debug` **или** build target в одном Dockerfile (`runtime-debug`), чтобы собирать `:debug-nonroot` для диагностики. citeturn12view0  
- `.dockerignore` — обязательный baseline (секреты, артефакты, VCS). citeturn13search0turn13search1  
- `docs/standards/container-images.md` — нормативный документ «Container Image Standard»: базовые образы, non-root, exec-form, CA/tzdata, CGO decision, debug images, anti-patterns. citeturn33view0turn12view0turn28view0turn25search0  
- `docs/standards/ci-quality-gates.md` — «CI Pipeline Standard»: обязательные stages, блокирующие проверки, async проверки, команды. Основано на go toolchain semantics (`go fmt`, `go test` caching+vet subset, `go generate`), плюс `govulncheck`. citeturn22view0turn21view3turn21view2turn5search2  
- `docs/llm/instructions.md` — короткий «контракт для LLM»: MUST/SHOULD/NEVER правила из этого отчёта (с акцентом на: не менять образ без причины, не ломать CI gates, не использовать shell form, не добавлять секреты в Dockerfile). citeturn12view0turn17view0turn18view0  
- `Makefile` (или `mage`, но Make проще для шаблона) — единая точка входа для CI и LLM:
  - `make fmt` → `go fmt ./...` citeturn22view0  
  - `make test` → `go test -count=1 ./...` citeturn21view2turn21view3  
  - `make test-race` → `go test -race ./...` citeturn4view2  
  - `make vuln` → `govulncheck ./...` citeturn5search2  
  - `make generate` → `go generate ./...` (если принято) citeturn22view0  
  - `make docker-build` → `docker build …` citeturn13search0  
- `ci/` (или `.github/workflows/ci.yml` если вы выбираете GitHub как дефолт) — pipeline с quality gates:
  - **Blocking на PR**: fmt, unit tests, vulncheck, generate drift, go mod tidy drift, docker build.
  - **Async/ночные**: race (если долго), container image vulnerability scan, SBOM/provenance/signing. citeturn21view2turn6search3turn24search0turn24search3