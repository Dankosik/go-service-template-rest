SERVICE_NAME := service
BINARY := bin/$(SERVICE_NAME)
OPENAPI_FILE := api/openapi/service.yaml
REDOCLY_CLI_VERSION := 2.20.0
KIN_OPENAPI_VALIDATE_VERSION := v0.133.0
OASDIFF_VERSION := v1.11.10
MIGRATE_VERSION := v4.19.0
GOLANGCI_LINT_VERSION := v2.10.1
DOCS_DRIFT_SCRIPT := scripts/ci/docs-drift-check.sh
GUARDRAILS_CHECK_SCRIPT := scripts/ci/required-guardrails-check.sh

.PHONY: setup doctor init-module tidy fmt test test-race test-cover test-integration lint run build docker-build compose-up compose-down vendor \
	openapi-generate openapi-drift-check openapi-runtime-contract-check openapi-lint openapi-validate openapi-breaking openapi-check \
	mod-check fmt-check docs-drift-check guardrails-check migration-validate

setup:
	./scripts/dev/setup.sh "$(GOLANGCI_LINT_VERSION)"

doctor:
	./scripts/dev/doctor.sh

init-module:
	@test -n "$(MODULE)" || (echo "MODULE is required, example: MODULE=github.com/acme/my-service"; exit 1)
	./scripts/init-module.sh "$(MODULE)"

tidy:
	go mod tidy

fmt:
	gofmt -w $(shell find . -type f -name '*.go' -not -path './vendor/*')

mod-check:
	GOFLAGS= go mod tidy -diff
	go mod verify
	git diff --exit-code -- go.mod go.sum

fmt-check:
	$(MAKE) fmt
	git diff --exit-code

test:
	go test ./...

test-race:
	go test -race ./...

test-cover:
	go test -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

test-integration:
	go test -tags=integration ./test/...

lint:
	golangci-lint run

openapi-generate:
	go generate ./internal/api

openapi-drift-check:
	@git diff --quiet -- internal/api || (echo "tracked openapi codegen drift detected in internal/api"; git diff -- internal/api; exit 1)
	@untracked="$$(git ls-files --others --exclude-standard -- internal/api)"; \
	if [ -n "$$untracked" ]; then \
		echo "untracked openapi artifacts detected in internal/api"; \
		echo "$$untracked"; \
		echo "run 'make openapi-generate' and commit updated generated files"; \
		exit 1; \
	fi

openapi-runtime-contract-check:
	go test ./internal/infra/http -run '^TestOpenAPIRuntimeContract' -count=1

openapi-lint:
	npx @redocly/cli@$(REDOCLY_CLI_VERSION) lint $(OPENAPI_FILE)

openapi-validate:
	go run github.com/getkin/kin-openapi/cmd/validate@$(KIN_OPENAPI_VALIDATE_VERSION) -- $(OPENAPI_FILE)

openapi-breaking:
	@test -n "$(BASE_OPENAPI)" || (echo "BASE_OPENAPI is required"; exit 1)
	go run github.com/oasdiff/oasdiff@$(OASDIFF_VERSION) breaking --fail-on ERR $(BASE_OPENAPI) $(OPENAPI_FILE)

openapi-check: openapi-generate openapi-drift-check openapi-runtime-contract-check openapi-lint openapi-validate

docs-drift-check:
	@test -n "$(BASE_REF)" || (echo "BASE_REF is required"; exit 1)
	@test -n "$(HEAD_REF)" || (echo "HEAD_REF is required"; exit 1)
	$(DOCS_DRIFT_SCRIPT) "$(BASE_REF)" "$(HEAD_REF)"

guardrails-check:
	$(GUARDRAILS_CHECK_SCRIPT)

migration-validate:
	@test -n "$(MIGRATION_DSN)" || (echo "MIGRATION_DSN is required"; exit 1)
	go run github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) -path env/migrations -database "$(MIGRATION_DSN)" up
	go run github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) -path env/migrations -database "$(MIGRATION_DSN)" down 1
	go run github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) -path env/migrations -database "$(MIGRATION_DSN)" up 1

run:
	go run ./cmd/$(SERVICE_NAME)

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o $(BINARY) ./cmd/$(SERVICE_NAME)

docker-build:
	docker build -f build/docker/Dockerfile -t $(SERVICE_NAME):local .

compose-up:
	docker compose -f env/docker-compose.yml up -d

compose-down:
	docker compose -f env/docker-compose.yml down -v

vendor:
	go mod vendor
