SERVICE_NAME := service
BINARY := bin/$(SERVICE_NAME)
OPENAPI_FILE := api/openapi/service.yaml
GO_FILES := $(shell git ls-files --cached --others --exclude-standard -- '*.go' 2>/dev/null | awk '!/^(\.agents|\.cache|vendor)\//' | while IFS= read -r file; do [ -f "$$file" ] && printf '%s\n' "$$file"; done)
GOFUMPT_FILES := $(filter-out internal/api/openapi.gen.go internal/infra/postgres/sqlcgen/%,$(GO_FILES))
REDOCLY_CLI_VERSION := 2.20.3
GO_REQUIRED_VERSION := $(shell awk '/^go / {print $$2; exit}' go.mod)
TEST_REPORT_DIR := .artifacts/test
TEST_JUNIT_FILE := $(TEST_REPORT_DIR)/junit.xml
TEST_JSON_FILE := $(TEST_REPORT_DIR)/test2json.json
COVERAGE_MIN ?= 80.0
COVERAGE_GOTOOLCHAIN ?= go$(GO_REQUIRED_VERSION)
COVERAGE_EXCLUDE_REGEX ?= (^|/)internal/api/openapi\.gen\.go:|(^|/)internal/infra/postgres/sqlcgen/|(^|/)cmd/service/main\.go:|(^|/)cmd/migrate/main\.go:
FUZZ_TIME ?= 45s
LINT_BASE_REF ?= origin/main
LINT_CONCURRENCY ?= 4

BENCH_PACKAGE ?= ./...
BENCH_PATTERN ?= .
BENCH_COUNT ?= 10
BENCH_TIME ?= 1s
BENCH_TAGS ?=
BENCH_OUTPUT ?= .artifacts/bench/current.txt
BENCH_BASELINE ?= .artifacts/bench/baseline.txt
BENCH_CURRENT ?= .artifacts/bench/current.txt
BENCH_COMPARE_OUTPUT ?= .artifacts/bench/comparison.txt
BENCH_PROFILE ?= cpu
BENCH_PROFILE_DIR ?= .artifacts/bench/profiles
BENCH_WORKLOAD_ID ?= go-benchmark
BENCH_DB_PACKAGE ?= ./test/...
BENCH_DB_PATTERN ?= .
BENCH_DB_OUTPUT ?= .artifacts/bench/db/current.txt
BENCH_DB_BASELINE ?= .artifacts/bench/db/baseline.txt
BENCH_DB_CURRENT ?= .artifacts/bench/db/current.txt
BENCH_DB_COMPARE_OUTPUT ?= .artifacts/bench/db/comparison.txt
BENCH_DB_WORKLOAD_ID ?=
POSTGRES_TEST_IMAGE := $(shell sed -n 's/^const postgresTestImage = "\(.*\)"$$/\1/p' test/postgres_integration_test.go)
HTTP_BENCH_SCRIPT ?= test/performance/http/single-flow.js
HTTP_BENCH_ARTIFACT_DIR ?= .artifacts/bench/http
HTTP_BENCH_ENV_FILE ?= .env.bench
HTTP_BENCH_DOCKER_NETWORK ?=
HTTP_BENCH_RAW_SAMPLES ?= 0

TRIVY_IMAGE ?= aquasec/trivy:0.69.2@sha256:3d1f862cb6c4fe13c1506f96f816096030d8d5ccdb2380a3069f7bf07daa86aa
GENERATED_DRIFT_CHECK_SCRIPT := bash ./scripts/ci/generated-drift-check.sh
BENCHMARK_SCRIPT := bash ./scripts/dev/benchmark.sh
BENCHMARK_REMOTE_SCRIPT := bash ./scripts/dev/benchmark-remote.sh

.DEFAULT_GOAL := help

.PHONY: help template-init template-init-check check check-full pr-check \
	tidy fmt mod-check fmt-check test test-summary test-watch test-race test-cover test-report coverage-effective-total coverage-summary coverage-check test-fuzz-smoke test-flake-smoke test-integration \
	bench bench-baseline bench-compare bench-profile bench-db bench-db-baseline bench-db-compare bench-http bench-http-inspect benchmark-infra-check benchmark-remote-check benchmark-remote-image \
	lint lint-fast deadcode nilaway modernize-check test-parallelism-check govulncheck gosec go-security secret-scan ci-local \
	sqlc-generate sqlc-check openapi-generate openapi-drift-check openapi-runtime-contract-check openapi-lint openapi-validate openapi-breaking openapi-check \
	migration-validate container-security run build docker-build docker-run compose-up compose-down vendor claude-skills-sync

help:
	@echo "Setup and everyday development:"
	@echo "  make template-init MODULE=github.com/acme/service CODEOWNER=@acme/team"
	@echo "  make check              # formatting, lint, and unit tests"
	@echo "  make ci-local           # deterministic native CI aggregate"
	@echo "  make check-full         # native aggregate plus Docker-backed gates"
	@echo "  make pr-check BASE_REF=origin/main"
	@echo "  make run"
	@echo ""
	@echo "Focused validation:"
	@echo "  make test | test-race | test-report | test-integration"
	@echo "  make lint | lint-fast | go-security | secret-scan"
	@echo "  make openapi-check | sqlc-check | migration-validate"
	@echo "  make docker-build | container-security"
	@echo ""
	@echo "Benchmarking:"
	@echo "  make bench | bench-baseline | bench-compare | bench-profile"
	@echo "  make bench-db BENCH_DB_WORKLOAD_ID=<fixture-state> | bench-db-baseline | bench-db-compare"
	@echo "  make bench-http | bench-http-inspect | benchmark-infra-check | benchmark-remote-check | benchmark-remote-image"
	@echo ""
	@echo "Reference: docs/build-test-and-development-commands.md"

template-init:
	@if [ -n "$(MODULE)" ]; then \
		CODEOWNER="$(CODEOWNER)" bash ./scripts/init-module.sh "$(MODULE)"; \
	else \
		CODEOWNER="$(CODEOWNER)" bash ./scripts/init-module.sh; \
	fi

template-init-check:
	bash ./scripts/ci/template-init-check.sh

claude-skills-sync:
	@mkdir -p .claude/skills
	@find .claude/skills -maxdepth 1 -type l -delete
	@set -e; for d in .agents/skills/*/; do n=$$(basename "$$d"); ln -s "../../.agents/skills/$$n" ".claude/skills/$$n"; done
	@echo ".claude/skills: $$(ls .claude/skills | wc -l | tr -d ' ') skill links"

check: fmt-check lint test

ci-local:
	$(MAKE) mod-check template-init-check fmt-check lint test-race test-report sqlc-check openapi-check go-security secret-scan

check-full:
	@command -v docker >/dev/null 2>&1 || { echo "Docker is required for make check-full"; exit 1; }
	@docker info >/dev/null 2>&1 || { echo "Docker daemon is not reachable"; exit 1; }
	$(MAKE) ci-local
	REQUIRE_DOCKER=1 $(MAKE) test-integration
	docker build -f build/docker/Dockerfile -t $(SERVICE_NAME):ci .
	$(MAKE) migration-validate RUNTIME_IMAGE=$(SERVICE_NAME):ci
	$(MAKE) container-security CONTAINER_IMAGE=$(SERVICE_NAME):ci

pr-check:
	@test -n "$(BASE_REF)" || { echo "BASE_REF is required, for example BASE_REF=origin/main"; exit 1; }
	$(MAKE) check-full
	@mkdir -p .cache
	@base_openapi="$$(mktemp .cache/openapi-base.XXXXXX)"; \
	trap 'rm -f "$$base_openapi"' EXIT; \
	if git show "$(BASE_REF):$(OPENAPI_FILE)" >"$$base_openapi" 2>/dev/null; then \
		$(MAKE) openapi-breaking BASE_OPENAPI="$$base_openapi"; \
	else \
		echo "No base OpenAPI spec at $(BASE_REF):$(OPENAPI_FILE); breaking check not applicable"; \
	fi

tidy:
	go mod tidy

fmt:
	go tool goimports -w $(GO_FILES)
	go tool gofumpt -w $(GOFUMPT_FILES)

mod-check:
	GOFLAGS= go mod tidy -diff
	go mod verify

fmt-check:
	@unformatted="$$(go tool goimports -l $(GO_FILES))"; \
	if [ -n "$$unformatted" ]; then \
		echo "goimports required for:"; \
		echo "$$unformatted"; \
		echo "run 'make fmt'"; \
		exit 1; \
	fi
	@gofumpt_unformatted="$$(go tool gofumpt -l $(GOFUMPT_FILES))"; \
	if [ -n "$$gofumpt_unformatted" ]; then \
		echo "gofumpt required for:"; \
		echo "$$gofumpt_unformatted"; \
		echo "run 'make fmt'"; \
		exit 1; \
	fi

test:
	go tool gotestsum --format=pkgname-and-test-fails -- ./...

test-summary: test

test-watch:
	go tool gotestsum --watch --format=pkgname-and-test-fails

test-race:
	go test -race ./...

test-cover:
	GOTOOLCHAIN=$(COVERAGE_GOTOOLCHAIN) GOCOVERDIR= go test -covermode=atomic -coverprofile=coverage.out ./...
	$(MAKE) coverage-summary

test-report:
	@mkdir -p $(TEST_REPORT_DIR)
	GOTOOLCHAIN=$(COVERAGE_GOTOOLCHAIN) GOCOVERDIR= go tool gotestsum --format=standard-verbose --junitfile=$(TEST_JUNIT_FILE) --jsonfile=$(TEST_JSON_FILE) -- -covermode=atomic -coverprofile=coverage.out ./...
	$(MAKE) coverage-summary
	$(MAKE) coverage-check COVERAGE_MIN=$(COVERAGE_MIN)

coverage-effective-total:
	@test -f coverage.out || { echo "coverage.out not found; run 'make test-cover' or 'make test-report'"; exit 1; }
	@filtered_cov="$$(mktemp)"; \
	trap 'rm -f "$$filtered_cov"' EXIT; \
	grep -Ev '$(COVERAGE_EXCLUDE_REGEX)' coverage.out >"$$filtered_cov"; \
	total="$$(GOTOOLCHAIN=$(COVERAGE_GOTOOLCHAIN) go tool cover -func="$$filtered_cov" | awk '/^total:/ {gsub(/%/, "", $$3); print $$3}')"; \
	test -n "$$total" || { echo "failed to parse total coverage"; exit 1; }; \
	printf '%s\n' "$$total"

coverage-summary:
	@test -f coverage.out || { echo "coverage.out not found; run 'make test-cover' or 'make test-report'"; exit 1; }
	@raw="$$(GOTOOLCHAIN=$(COVERAGE_GOTOOLCHAIN) go tool cover -func=coverage.out | awk '/^total:/ {print $$3}')"; \
	effective="$$( $(MAKE) --no-print-directory coverage-effective-total )"; \
	test -n "$$raw" && test -n "$$effective" || { echo "failed to parse coverage totals"; exit 1; }; \
	printf 'Raw coverage: %s\nEffective coverage (filtered): %s%%\n' "$$raw" "$$effective"

coverage-check:
	@total="$$( $(MAKE) --no-print-directory coverage-effective-total )"; \
	awk -v total="$$total" -v minimum="$(COVERAGE_MIN)" 'BEGIN { \
		if ((total + 0) < (minimum + 0)) { \
			printf "coverage %.2f%% is below threshold %.2f%%\n", total, minimum; \
			exit 1; \
		} \
		printf "coverage %.2f%% meets threshold %.2f%%\n", total, minimum; \
	}'

test-fuzz-smoke:
	@found=0; \
	pkgs="$$(go list ./...)" || exit $$?; \
	for pkg in $$pkgs; do \
		fuzz_targets="$$(go test "$$pkg" -list '^Fuzz' 2>&1)" || { status=$$?; printf '%s\n' "$$fuzz_targets"; exit $$status; }; \
		if printf '%s\n' "$$fuzz_targets" | grep -q '^Fuzz'; then \
			found=1; \
			go test "$$pkg" -run '^$$' -fuzz=Fuzz -fuzztime=$(FUZZ_TIME) || exit $$?; \
		fi; \
	done; \
	if [ "$$found" -eq 0 ]; then echo "no fuzz targets found; skipping fuzz smoke run"; fi

test-flake-smoke:
	go test -count=5 -shuffle=on ./...

test-integration:
	go test -tags=integration ./test/...

bench:
	BENCH_PACKAGE="$(BENCH_PACKAGE)" BENCH_PATTERN="$(BENCH_PATTERN)" BENCH_COUNT="$(BENCH_COUNT)" BENCH_TIME="$(BENCH_TIME)" BENCH_TAGS="$(BENCH_TAGS)" BENCH_OUTPUT="$(BENCH_OUTPUT)" BENCH_WORKLOAD_ID="$(BENCH_WORKLOAD_ID)" $(BENCHMARK_SCRIPT) run

bench-baseline:
	$(MAKE) bench BENCH_OUTPUT="$(BENCH_BASELINE)"

bench-compare:
	BENCH_BASELINE="$(BENCH_BASELINE)" BENCH_CURRENT="$(BENCH_CURRENT)" BENCH_COMPARE_OUTPUT="$(BENCH_COMPARE_OUTPUT)" $(BENCHMARK_SCRIPT) compare

bench-profile:
	BENCH_PACKAGE="$(BENCH_PACKAGE)" BENCH_PATTERN="$(BENCH_PATTERN)" BENCH_TIME="$(BENCH_TIME)" BENCH_TAGS="$(BENCH_TAGS)" BENCH_PROFILE="$(BENCH_PROFILE)" BENCH_PROFILE_DIR="$(BENCH_PROFILE_DIR)" BENCH_WORKLOAD_ID="$(BENCH_WORKLOAD_ID)" $(BENCHMARK_SCRIPT) profile

bench-db:
	@test -n "$(BENCH_DB_WORKLOAD_ID)" || { echo "BENCH_DB_WORKLOAD_ID is required, for example fixture-10k-warm"; exit 1; }
	REQUIRE_DOCKER=1 BENCH_PACKAGE="$(BENCH_DB_PACKAGE)" BENCH_PATTERN="$(BENCH_DB_PATTERN)" BENCH_COUNT="$(BENCH_COUNT)" BENCH_TIME="$(BENCH_TIME)" BENCH_TAGS=integration BENCH_OUTPUT="$(BENCH_DB_OUTPUT)" BENCH_WORKLOAD_ID="$(BENCH_DB_WORKLOAD_ID)" BENCH_DEPENDENCY_IMAGE="$(POSTGRES_TEST_IMAGE)" BENCH_SCHEMA_PATH=env/migrations $(BENCHMARK_SCRIPT) run

bench-db-baseline:
	$(MAKE) bench-db BENCH_DB_OUTPUT="$(BENCH_DB_BASELINE)"

bench-db-compare:
	BENCH_BASELINE="$(BENCH_DB_BASELINE)" BENCH_CURRENT="$(BENCH_DB_CURRENT)" BENCH_COMPARE_OUTPUT="$(BENCH_DB_COMPARE_OUTPUT)" $(BENCHMARK_SCRIPT) compare

bench-http:
	HTTP_BENCH_SCRIPT="$(HTTP_BENCH_SCRIPT)" HTTP_BENCH_ARTIFACT_DIR="$(HTTP_BENCH_ARTIFACT_DIR)" HTTP_BENCH_ENV_FILE="$(HTTP_BENCH_ENV_FILE)" HTTP_BENCH_DOCKER_NETWORK="$(HTTP_BENCH_DOCKER_NETWORK)" HTTP_BENCH_RAW_SAMPLES="$(HTTP_BENCH_RAW_SAMPLES)" $(BENCHMARK_SCRIPT) http

bench-http-inspect:
	HTTP_BENCH_SCRIPT="$(HTTP_BENCH_SCRIPT)" HTTP_BENCH_ARTIFACT_DIR="$(HTTP_BENCH_ARTIFACT_DIR)" HTTP_BENCH_ENV_FILE="$(HTTP_BENCH_ENV_FILE)" HTTP_BENCH_DOCKER_NETWORK="$(HTTP_BENCH_DOCKER_NETWORK)" HTTP_BENCH_RAW_SAMPLES=0 $(BENCHMARK_SCRIPT) http-inspect

benchmark-infra-check:
	$(BENCHMARK_SCRIPT) check

benchmark-remote-check:
	$(BENCHMARK_REMOTE_SCRIPT) check

benchmark-remote-image:
	$(BENCHMARK_REMOTE_SCRIPT) image-build

lint:
	go tool golangci-lint run --allow-parallel-runners --timeout=3m
	$(MAKE) deadcode
	$(MAKE) nilaway

lint-fast:
	go tool golangci-lint run --fast-only --new-from-rev=$(LINT_BASE_REF) --concurrency=$(LINT_CONCURRENCY) --timeout=3m

deadcode:
	go tool deadcode -test -tags=integration ./...

nilaway:
	@module_path="$$(go list -m)"; \
	printf 'go tool nilaway -include-pkgs=%s -test ./...\n' "$$module_path"; \
	go tool nilaway -include-pkgs="$$module_path" -test ./...

modernize-check:
	go fix -diff ./...

test-parallelism-check:
	go tool golangci-lint run --enable-only=paralleltest,tparallel --timeout=3m --max-issues-per-linter=0 --max-same-issues=0

govulncheck:
	go tool govulncheck ./...

gosec:
	@gosec_cache="$$(mktemp -d)"; \
	trap 'rm -rf "$$gosec_cache"' EXIT; \
	GOCACHE="$$gosec_cache" go tool gosec -exclude-generated -exclude-dir=.agents -exclude-dir=.cache -exclude-dir=.artifacts ./...

go-security: govulncheck gosec

secret-scan:
	go tool gitleaks git --no-banner --redact --exit-code 1 --baseline-path .gitleaks.baseline.json .

sqlc-generate:
	@if [ -z "$$(find internal/infra/postgres/queries -type f -name '*.sql' -print -quit)" ]; then \
		echo "no sqlc query sources; skipping sqlc generation"; \
	else \
		go tool github.com/sqlc-dev/sqlc/cmd/sqlc generate -f internal/infra/postgres/sqlc.yaml; \
	fi

sqlc-check:
	$(GENERATED_DRIFT_CHECK_SCRIPT) sqlc

openapi-generate:
	go generate ./internal/api

openapi-drift-check:
	$(GENERATED_DRIFT_CHECK_SCRIPT) openapi

openapi-runtime-contract-check:
	@tests="$$(go test ./internal/infra/http -list '^TestOpenAPIRuntimeContract')" || { status=$$?; printf '%s\n' "$$tests"; exit $$status; }; \
	printf '%s\n' "$$tests" | grep -q '^TestOpenAPIRuntimeContract' || { echo "no OpenAPI runtime contract tests matched"; exit 1; }
	go test ./internal/infra/http -run '^TestOpenAPIRuntimeContract' -count=1

openapi-lint:
	npx @redocly/cli@$(REDOCLY_CLI_VERSION) lint --config .redocly.yaml $(OPENAPI_FILE)

openapi-validate:
	go tool validate -- $(OPENAPI_FILE)

openapi-breaking:
	@test -n "$(BASE_OPENAPI)" || { echo "BASE_OPENAPI is required"; exit 1; }
	go tool oasdiff breaking --fail-on ERR $(BASE_OPENAPI) $(OPENAPI_FILE)

openapi-check: openapi-drift-check
	go test ./internal/api
	$(MAKE) openapi-runtime-contract-check openapi-lint openapi-validate

migration-validate:
	@if [ -n "$(MIGRATION_DSN)" ]; then \
		APP__POSTGRES__ENABLED=true \
			APP__POSTGRES__DSN="$(MIGRATION_DSN)" \
			MIGRATION_PATH="$${MIGRATION_PATH:-env/migrations}" \
			go run ./cmd/migrate validate; \
	else \
		command -v docker >/dev/null 2>&1 || { echo "MIGRATION_DSN or Docker is required"; exit 1; }; \
		docker info >/dev/null 2>&1 || { echo "Docker daemon is not reachable"; exit 1; }; \
		project="$${MIGRATION_PROJECT:-service-migration-$$(date +%s)-$$$$}"; \
		runtime=""; \
		compose() { POSTGRES_PORT=0 docker compose -p "$$project" -f env/docker-compose.yml "$$@"; }; \
		cleanup() { \
			if [ -n "$$runtime" ]; then docker rm -f "$$runtime" >/dev/null 2>&1 || true; fi; \
			compose down -v --remove-orphans >/dev/null 2>&1 || true; \
		}; \
		trap cleanup EXIT INT TERM; \
		compose up -d --wait postgres; \
		address="$$(compose port postgres 5432)"; \
		port="$${address##*:}"; \
		test -n "$$port" || { echo "failed to resolve rehearsal Postgres port"; exit 1; }; \
		dsn="postgres://app:app@localhost:$$port/app?sslmode=disable"; \
		APP__POSTGRES__ENABLED=true \
			APP__POSTGRES__DSN="$$dsn" \
			MIGRATION_PATH="$${MIGRATION_PATH:-env/migrations}" \
			go run ./cmd/migrate validate; \
		image="$(RUNTIME_IMAGE)"; \
		if [ -z "$$image" ]; then image="$(SERVICE_NAME):migration"; docker build -f build/docker/Dockerfile -t "$$image" .; fi; \
		docker run --rm --network "$${project}_default" \
			-e APP__POSTGRES__ENABLED=true \
			-e APP__POSTGRES__DSN="postgres://app:app@postgres:5432/app?sslmode=disable" \
			--entrypoint /migrate "$$image"; \
		command -v curl >/dev/null 2>&1 || { echo "curl is required for runtime image readiness validation"; exit 1; }; \
		runtime="$${project}-runtime"; \
		docker run -d --name "$$runtime" \
			--network "$${project}_default" \
			-p 127.0.0.1::8080 \
			--read-only \
			--cap-drop=ALL \
			--security-opt=no-new-privileges \
			-e NETWORK_EGRESS_ALLOWLIST=postgres \
			-e APP__POSTGRES__ENABLED=true \
			-e APP__POSTGRES__DSN="postgres://app:app@postgres:5432/app?sslmode=disable" \
			"$$image" >/dev/null; \
		address="$$(docker port "$$runtime" 8080/tcp | head -n 1)"; \
		port="$${address##*:}"; \
		test -n "$$port" || { echo "failed to resolve runtime service port"; docker logs "$$runtime"; exit 1; }; \
		ready=false; \
		attempt=0; \
		while [ "$$attempt" -lt 45 ]; do \
			if curl -fs --max-time 2 "http://127.0.0.1:$$port/health/ready" >/dev/null; then ready=true; break; fi; \
			if [ "$$(docker inspect -f '{{.State.Running}}' "$$runtime")" != "true" ]; then break; fi; \
			attempt=$$((attempt + 1)); \
			sleep 1; \
		done; \
		if [ "$$ready" != "true" ]; then echo "runtime image did not become ready"; docker logs "$$runtime"; exit 1; fi; \
		if [ -n "$(RUNTIME_EXPECTED_VERSION)" ] && ! docker logs "$$runtime" 2>&1 | grep -Fq "\"service.version\":\"$(RUNTIME_EXPECTED_VERSION)\""; then \
			echo "runtime image did not report expected version $(RUNTIME_EXPECTED_VERSION)"; \
			docker logs "$$runtime"; \
			exit 1; \
		fi; \
		docker stop --time 45 "$$runtime" >/dev/null; \
		exit_code="$$(docker inspect -f '{{.State.ExitCode}}' "$$runtime")"; \
		test "$$exit_code" = "0" || { echo "runtime image exited with code $$exit_code after SIGTERM"; docker logs "$$runtime"; exit 1; }; \
	fi

container-security:
	@command -v docker >/dev/null 2>&1 || { echo "Docker is required"; exit 1; }
	@docker info >/dev/null 2>&1 || { echo "Docker daemon is not reachable"; exit 1; }
	@image="$(CONTAINER_IMAGE)"; \
	if [ -z "$$image" ]; then image="$(SERVICE_NAME):ci"; docker build -f build/docker/Dockerfile -t "$$image" .; fi; \
	docker run --rm \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-e DOCKER_HOST=unix:///var/run/docker.sock \
		"$(TRIVY_IMAGE)" image \
		--severity HIGH,CRITICAL \
		--ignore-unfixed \
		--exit-code 1 \
		--format table \
		"$$image"

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
	docker compose -f env/docker-compose.yml up -d --wait

compose-down:
	docker compose -f env/docker-compose.yml down -v

vendor:
	go mod vendor
