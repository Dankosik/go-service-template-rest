# go-service-template-rest

Production-ready шаблон REST-микросервиса на Go с упором на понятную структуру, надежный запуск и базовую наблюдаемость.

## Что внутри

- `cmd/service` как тонкая точка входа
- `internal` для приватной логики (`app/domain/infra/config`)
- конфиг через env-переменные
- структурные JSON-логи через `log/slog`
- `GET /health/live`, `GET /health/ready`, `GET /api/v1/ping`, `GET /metrics`
- базовые HTTP timeout-ы и graceful shutdown
- optional Postgres readiness probe (через `POSTGRES_DSN`)
- OpenAPI workflow: codegen (`oapi-codegen`) + lint + validate + breaking-check
- Docker multi-stage + distroless runtime
- CI: unit tests + race detector + coverage artifact + integration tests + lint + OpenAPI gates + security gates
- Dependabot для `gomod` и GitHub Actions

## Структура

```text
.
├── api/
├── build/
├── cmd/
├── internal/
├── env/
├── scripts/
├── test/
├── .github/workflows/
├── Makefile
├── go.mod
└── README.md
```

## Быстрый старт

1. Подними локальный Postgres (опционально):

```bash
make compose-up
```

2. Скопируй env-шаблон и при необходимости поправь значения:

```bash
cp env/.env.example .env
```

3. Запусти сервис:

```bash
set -a
source .env
set +a
go run ./cmd/service
```

Если Go не установлен локально, можно собрать контейнер:

```bash
make docker-build
```

## Endpoints

- `GET /api/v1/ping` -> `pong`
- `GET /health/live` -> `ok`
- `GET /health/ready` -> `ok` или `503 not ready`
- `GET /metrics` -> Prometheus metrics

## Основные команды

```bash
make fmt
make test
make test-race
make test-cover
make test-integration
make lint
make openapi-generate
make openapi-lint
make openapi-validate
make build
make run
make docker-build
```

Проверка breaking-изменений OpenAPI локально:

```bash
BASE_OPENAPI=/path/to/base-service.yaml make openapi-breaking
```

`make test-integration` запускает тесты с тегом `integration` и требует Docker.

## Конфигурация

Смотри `env/.env.example`:

- `APP_ENV`
- `HTTP_ADDR`
- `HTTP_SHUTDOWN_TIMEOUT`
- `HTTP_READ_HEADER_TIMEOUT`
- `HTTP_READ_TIMEOUT`
- `HTTP_WRITE_TIMEOUT`
- `HTTP_IDLE_TIMEOUT`
- `HTTP_MAX_HEADER_BYTES`
- `LOG_LEVEL`
- `POSTGRES_DSN`

## Контракты API

- OpenAPI: `api/openapi/service.yaml`
- Protobuf (опционально): `api/proto/service/v1/service.proto`

### OpenAPI codegen

Go bindings генерируются через `oapi-codegen`:

```bash
make openapi-generate
```

Точка генерации находится в `internal/api/doc.go` (`go:generate`), конфиг — `internal/api/oapi-codegen.yaml`.

## Миграции

SQL-миграции хранятся в `env/migrations`.
Инструмент миграций (migrate/goose/atlas/tern) выбирается на уровне команды.

## CI quality gates

Workflow `.github/workflows/ci.yml` включает:

- `test`: `go test ./...`
- `test-race`: `go test -race ./...`
- `test-coverage`: `go test -covermode=atomic -coverprofile=coverage.out ./...` + публикация `coverage.out` как артефакта
- `test-integration`: `go test -tags=integration ./test/...`
- `lint`: `golangci-lint`
- `openapi-contract`: generate + validate + lint OpenAPI
- `openapi-breaking` (PR): проверка breaking изменений между base и текущей OpenAPI-спекой
- `go-security`: `govulncheck` и `gosec`
- `container-security`: Trivy scan для Docker image
