# Аудит Go tooling в репозитории

Дата аудита: 3 марта 2026  
Проверено по: `Makefile`, `.golangci.yml`, `go.mod`, `scripts/dev/docker-tooling.sh`, `.github/workflows/ci.yml`, `.github/workflows/nightly.yml`, `.github/workflows/cd.yml`.

Критерии статуса:
- `Полностью`: есть рабочая команда/конфиг и это запускается в CI.
- `Частично`: есть только часть практики (например, косвенно через другой инструмент, локально без CI, или урезанный охват).
- `Нет`: не найдено команды/конфига/CI-интеграции.

## Полностью настроено

- `goimports` (форматирование и импорты): `make fmt`, `make fmt-check`, CI `repo-integrity`.
- `golangci-lint`: `make lint`, CI job `lint`.
- `staticcheck` (через `golangci-lint`): включён в `.golangci.yml`, запускается в CI job `lint`.
- `errcheck` (через `golangci-lint`): включён в `.golangci.yml`, запускается в CI job `lint`.
- `govulncheck`: `make go-security`, CI job `go-security`.
- `gosec`: `make go-security`, CI job `go-security`.
- `gitleaks`: `make secrets-scan`, CI job `secret-scan`.
- `Trivy` (container scan): CI jobs `container-security`, nightly, cd.
- `go test` базовый цикл: `make test`, `make test-race`, `make test-cover`, `make test-integration` + отдельные CI jobs.
- `go tool cover`: используется в `make test-cover` и `make test-cover-local`.
- `gotestsum` + JUnit/JSON отчёты: `make test-report`, CI job `test-coverage` публикует `.artifacts/test/junit.xml` и `.artifacts/test/test2json.json`.
- `go generate` для OpenAPI: `make openapi-generate`, CI `openapi-contract`.
- `Go modules` как dependency management: `go.mod/go.sum`, `make mod-check` (`go mod tidy -diff`, `go mod verify`) в CI.

## Частично настроено

- `gofmt`: отдельного гейта `gofmt -l` нет; форматирование стандартизовано через `goimports`.
- `go vet`: отдельно не запускается (`go vet ./...` нет), но `govet` включён внутри `golangci-lint`.
- Профилирование (`pprof`): флаги/инструменты toolchain доступны, но нет репозиторного make-target/CI-гейта под `-cpuprofile/-memprofile`.
- `go tool trace`: нет отдельных команд/CI-проверок.
- `fuzz` (`go test -fuzz`): есть `make test-fuzz-smoke` и nightly step, но нет обязательного PR gate.
- Закрепление dev-tools: начат переход на `tool`-директивы в `go.mod` (`gotest.tools/gotestsum`), остальной tooling пока пинится через `Makefile` и Docker tooling-образы.
- Генерация кода: автоматизирован только OpenAPI codegen; остальные популярные генераторы не подключены.

## Не настроено

- `gofumpt`.
- `gci`.
- `revive`.
- `covdata` / `GOCOVERDIR` workflow для агрегации multi-run coverage.
- `benchstat`.
- `stringer`.
- `mockgen` / `go.uber.org/mock`.
- `sqlc`.
- `wire` (или его форк).
- `GoReleaser` (нет `.goreleaser.yml` и release job на него).
- `toolchain`-директива в `go.mod`.
- Автоматизированный `go fix` шаг.
- `Ginkgo` как тестовый фреймворк.
- Явная интеграция `testify` как выбранного стандарта (зависимость присутствует косвенно, но в коде проекта использования не найдено).

## Короткий вывод

Для шаблонного Go-сервиса у вас уже сильный baseline: форматирование/линт, тесты (включая `-race`, coverage threshold и JUnit/JSON отчётность), security checks (`govulncheck`, `gosec`, `gitleaks`) и container scan (Trivy) полностью заведены и в CI, и в локальные команды. Основные пробелы относительно полного списка: обязательный fuzz gate на PR, trace/profile/bench pipeline, расширение `tool`-директив на весь dev-tooling, и release automation через `GoReleaser`.
