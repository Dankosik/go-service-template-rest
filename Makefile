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
BRANCH_PROTECTION_SCRIPT := scripts/dev/configure-branch-protection.sh
DOCKER_TOOLING_SCRIPT := scripts/dev/docker-tooling.sh
SKILLS_SYNC_SCRIPT := scripts/dev/sync-skills.sh

.PHONY: setup doctor init-module tidy fmt test test-race test-cover test-integration lint run build docker-build docker-run compose-up compose-down vendor \
	openapi-generate openapi-drift-check openapi-runtime-contract-check openapi-lint openapi-validate openapi-breaking openapi-check \
	mod-check fmt-check docs-drift-check guardrails-check migration-validate gh-protect skills-sync skills-check \
	setup-native setup-docker doctor-native doctor-docker docker-pull-tools docker-init-module docker-mod-check docker-fmt docker-fmt-check \
	docker-test docker-test-race docker-test-cover docker-test-integration docker-lint docker-openapi-check docker-go-security docker-ci \
	docker-guardrails-check docker-skills-check docker-docs-drift-check docker-migration-validate docker-container-security

setup:
	./scripts/dev/setup.sh

setup-native:
	./scripts/dev/setup.sh --native

setup-docker:
	./scripts/dev/setup.sh --docker

doctor:
	./scripts/dev/doctor.sh --mode auto

doctor-native:
	./scripts/dev/doctor.sh --mode native

doctor-docker:
	./scripts/dev/doctor.sh --mode docker

init-module:
	@test -n "$(MODULE)" || (echo "MODULE is required, example: MODULE=github.com/acme/my-service (optional CODEOWNER=@your-org/your-team)"; exit 1)
	./scripts/init-module.sh "$(MODULE)"

docker-pull-tools:
	$(DOCKER_TOOLING_SCRIPT) pull-images

docker-init-module:
	@test -n "$(MODULE)" || (echo "MODULE is required, example: MODULE=github.com/acme/my-service (optional CODEOWNER=@your-org/your-team)"; exit 1)
	CODEOWNER="$(CODEOWNER)" $(DOCKER_TOOLING_SCRIPT) init-module "$(MODULE)"

tidy:
	go mod tidy

fmt:
	gofmt -w $(shell find . -type f -name '*.go' -not -path './vendor/*')

mod-check:
	GOFLAGS= go mod tidy -diff
	go mod verify
	git diff --exit-code -- go.mod go.sum

docker-mod-check:
	$(DOCKER_TOOLING_SCRIPT) mod-check

fmt-check:
	$(MAKE) fmt
	git diff --exit-code

docker-fmt:
	$(DOCKER_TOOLING_SCRIPT) fmt

docker-fmt-check:
	$(DOCKER_TOOLING_SCRIPT) fmt-check

test:
	go test ./...

test-race:
	go test -race ./...

test-cover:
	go test -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

docker-test:
	$(DOCKER_TOOLING_SCRIPT) test

docker-test-race:
	$(DOCKER_TOOLING_SCRIPT) test-race

docker-test-cover:
	$(DOCKER_TOOLING_SCRIPT) test-cover

test-integration:
	go test -tags=integration ./test/...

docker-test-integration:
	$(DOCKER_TOOLING_SCRIPT) test-integration

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run --timeout=3m

docker-lint:
	$(DOCKER_TOOLING_SCRIPT) lint

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
	npx @redocly/cli@$(REDOCLY_CLI_VERSION) lint --config .redocly.yaml $(OPENAPI_FILE)

openapi-validate:
	go run github.com/getkin/kin-openapi/cmd/validate@$(KIN_OPENAPI_VALIDATE_VERSION) -- $(OPENAPI_FILE)

openapi-breaking:
	@test -n "$(BASE_OPENAPI)" || (echo "BASE_OPENAPI is required"; exit 1)
	go run github.com/oasdiff/oasdiff@$(OASDIFF_VERSION) breaking --fail-on ERR $(BASE_OPENAPI) $(OPENAPI_FILE)

openapi-check: openapi-generate openapi-drift-check
	go test ./internal/api
	$(MAKE) openapi-runtime-contract-check openapi-lint openapi-validate

docker-openapi-check:
	$(DOCKER_TOOLING_SCRIPT) openapi-check

docs-drift-check:
	@test -n "$(BASE_REF)" || (echo "BASE_REF is required"; exit 1)
	@test -n "$(HEAD_REF)" || (echo "HEAD_REF is required"; exit 1)
	$(DOCS_DRIFT_SCRIPT) "$(BASE_REF)" "$(HEAD_REF)"

guardrails-check:
	$(GUARDRAILS_CHECK_SCRIPT)

skills-sync:
	$(SKILLS_SYNC_SCRIPT)

skills-check:
	$(SKILLS_SYNC_SCRIPT) --check

docker-go-security:
	$(DOCKER_TOOLING_SCRIPT) go-security

docker-guardrails-check:
	$(DOCKER_TOOLING_SCRIPT) guardrails-check

docker-skills-check:
	$(DOCKER_TOOLING_SCRIPT) skills-check

docker-docs-drift-check:
	@test -n "$(BASE_REF)" || (echo "BASE_REF is required"; exit 1)
	@test -n "$(HEAD_REF)" || (echo "HEAD_REF is required"; exit 1)
	$(DOCKER_TOOLING_SCRIPT) docs-drift-check "$(BASE_REF)" "$(HEAD_REF)"

docker-migration-validate:
	$(DOCKER_TOOLING_SCRIPT) migration-validate

docker-container-security:
	$(DOCKER_TOOLING_SCRIPT) container-security

docker-ci:
	$(DOCKER_TOOLING_SCRIPT) ci

migration-validate:
	@test -n "$(MIGRATION_DSN)" || (echo "MIGRATION_DSN is required"; exit 1)
	go run github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) -path env/migrations -database "$(MIGRATION_DSN)" up
	go run github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) -path env/migrations -database "$(MIGRATION_DSN)" down 1
	go run github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) -path env/migrations -database "$(MIGRATION_DSN)" up 1

gh-protect:
	$(BRANCH_PROTECTION_SCRIPT) "$${BRANCH:-main}"

run:
	@set -a; \
	if [ -f .env ]; then . ./.env; fi; \
	set +a; \
	go run ./cmd/$(SERVICE_NAME)

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o $(BINARY) ./cmd/$(SERVICE_NAME)

docker-build:
	docker build -f build/docker/Dockerfile -t $(SERVICE_NAME):local .

docker-run:
	docker run --rm -p 8080:8080 --env-file .env $(SERVICE_NAME):local

compose-up:
	docker compose -f env/docker-compose.yml up -d

compose-down:
	docker compose -f env/docker-compose.yml down -v

vendor:
	go mod vendor
