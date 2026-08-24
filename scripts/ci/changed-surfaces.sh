#!/usr/bin/env bash
set -euo pipefail

names=(
	go_source go_root_dependencies go_tool_dependencies go_lint_config
	openapi protobuf sqlc module_initializer integration_initializer
	agent_instructions shell github_workflows dependency_automation
	db_integration messaging_integration process_integration integration_race
	migrations runtime_image image_security documentation
)

reset() {
	local name
	for name in "${names[@]}"; do
		printf -v "${name}" '%s' false
	done
}

mark() {
	local name
	for name in "$@"; do
		printf -v "${name}" '%s' true
	done
}

emit() {
	local name
	for name in "${names[@]}"; do
		printf '%s=%s\n' "${name}" "${!name}"
	done
}

classify() {
	local file
	reset
	while IFS= read -r file; do
		[[ -n "${file}" ]] || continue

		case "${file}" in
			*.go) mark go_source ;;
		esac
		case "${file}" in
			go.mod|go.sum|examples/*/go.mod|examples/*/go.sum) mark go_root_dependencies ;;
			tools/go.mod|tools/go.sum|scripts/ci/tools-smoke.sh) mark go_tool_dependencies ;;
		esac
		case "${file}" in
			.golangci.yml) mark go_lint_config ;;
		esac
		case "${file}" in
			.redocly.yaml|api/openapi/*|api/external/*/openapi.yaml|examples/reference-service/api/openapi.yaml|examples/reference-service/internal/openapi/*|internal/openapi/*|internal/infra/*/internal/openapi/*)
				mark openapi
				;;
		esac
		case "${file}" in
			buf.yaml|buf.gen.yaml|buf.lock|api/proto/*|examples/grpc-reference-service/buf.yaml|examples/grpc-reference-service/buf.gen.yaml|examples/grpc-reference-service/buf.lock|examples/grpc-reference-service/api/proto/*|examples/grpc-reference-service/internal/gen/proto/*|internal/gen/proto/*)
				mark protobuf
				;;
		esac
		case "${file}" in
			internal/infra/postgres/queries/*|internal/infra/postgres/sqlc.yaml|internal/infra/postgres/sqlcgen/*)
				mark sqlc db_integration
				;;
		esac
		case "${file}" in
			scripts/init-module.sh|scripts/ci/init-module-contract-check.sh|scripts/ci/template-init-check.sh|scripts/profiles/*|template-owned.paths|template.lock)
				mark module_initializer
				;;
		esac
		case "${file}" in
			scripts/integration-init.sh|scripts/ci/integration-init-check.sh)
				mark integration_initializer
				;;
		esac
		case "${file}" in
			*.sh) mark shell ;;
		esac
		case "${file}" in
			.github/workflows/*|.github/actions/*) mark github_workflows ;;
			.github/dependabot.yml) mark dependency_automation ;;
		esac
		case "${file}" in
			AGENTS.md|CLAUDE.md|Grok.md|QWEN.md|.agents/*|.claude/*|.codex/*|.cursor/*|.grok/*|.opencode/*|.qwen/*|docs/agent-harness/*|docs/spec-first-workflow/*|docs/prompt-*|docs/skill-authoring.md|docs/validation/*|scripts/agent-roles-sync.sh|scripts/harness-skills-sync.sh|scripts/codex-agents-sync.sh|scripts/template-sync.sh|scripts/ci/template-owned-purity-check.sh|scripts/ci/template-sync-behavior-check.sh)
				mark agent_instructions
				;;
		esac
		case "${file}" in
			cmd/migrate/*|internal/infra/postgres/*|internal/infra/postgresidempotency/*|internal/infra/postgresjobs/*|internal/infra/postgresmigrate/*|internal/infra/postgreswebhook/*|internal/inboundwebhook/*|migrations/*|test/postgres*|test/inbound_webhook*|test/webhook*)
				mark db_integration
				;;
		esac
		case "${file}" in
			cmd/worker/*|cmd/outbox-relay/*|internal/domainevent/*|internal/infra/natsjs/*|internal/infra/postgresoutbox/*|test/nats*|test/postgres_outbox_natsjs*)
				mark messaging_integration
				;;
		esac
		case "${file}" in
			cmd/*/main.go|cmd/*/internal/bootstrap/*|cmd/internal/runtimeopts/*|test/*process*|test/current_repository_fixture*)
				mark process_integration
				;;
		esac
		case "${file}" in
			internal/background/*|internal/infra/natsjs/*|internal/infra/postgresoutbox/*|internal/infra/postgreswebhook/*|cmd/worker/*|cmd/outbox-relay/*|test/nats_messaging_delivery*|test/nats_messaging_drain*|test/postgres_outbox*|test/postgres_webhook*|test/webhook_network*)
				mark integration_race
				;;
		esac
		case "${file}" in
			cmd/migrate/*|internal/infra/postgresmigrate/*|migrations/*|scripts/ci/migration-*)
				mark migrations db_integration
				;;
		esac
		case "${file}" in
			.dockerignore|build/docker/*|env/docker-compose.yml|scripts/ci/runtime-image-*.sh|.github/actions/publish-image/*)
				mark runtime_image image_security
				;;
		esac
		case "${file}" in
			*.md|docs/*) mark documentation ;;
		esac

		# The root file still composes every public target. Domain-owned mk files can
		# replace this conservative rule only after the profile initializer owns them.
		case "${file}" in
			Makefile|scripts/ci/changed-surfaces.sh) mark "${names[@]}" ;;
		esac
	done
	emit
}

assert_case() {
	local file=$1 true_names=$2 false_names=$3 output name
	output="$(printf '%s\n' "${file}" | classify)"
	for name in ${true_names}; do
		grep -qx "${name}=true" <<<"${output}" || {
			printf '%s: expected %s=true\n%s\n' "${file}" "${name}" "${output}" >&2
			return 1
		}
	done
	for name in ${false_names}; do
		grep -qx "${name}=false" <<<"${output}" || {
			printf '%s: expected %s=false\n%s\n' "${file}" "${name}" "${output}" >&2
			return 1
		}
	done
}

self_test() {
	assert_case README.md \
		"documentation" \
		"go_source go_root_dependencies go_tool_dependencies go_lint_config db_integration messaging_integration process_integration integration_race runtime_image image_security"
	assert_case tools/go.mod \
		"go_tool_dependencies" \
		"go_source go_root_dependencies go_lint_config db_integration messaging_integration process_integration integration_race runtime_image image_security"
	assert_case scripts/ci/tools-smoke.sh \
		"go_tool_dependencies shell" \
		"go_source go_root_dependencies go_lint_config db_integration messaging_integration process_integration integration_race runtime_image image_security"
	assert_case .golangci.yml \
		"go_lint_config" \
		"go_source go_root_dependencies go_tool_dependencies db_integration messaging_integration process_integration integration_race runtime_image image_security"
	assert_case build/docker/Dockerfile \
		"runtime_image image_security" \
		"go_source go_root_dependencies go_tool_dependencies go_lint_config db_integration messaging_integration process_integration integration_race"
	assert_case scripts/ci/runtime-image-check.sh \
		"shell runtime_image image_security" \
		"go_source go_root_dependencies go_tool_dependencies go_lint_config db_integration messaging_integration process_integration integration_race"
	assert_case examples/grpc-reference-service/buf.lock \
		"protobuf" \
		"go_source go_root_dependencies go_tool_dependencies go_lint_config db_integration messaging_integration process_integration integration_race runtime_image image_security"
	assert_case scripts/integration-init.sh \
		"integration_initializer shell" \
		"module_initializer go_source db_integration messaging_integration process_integration integration_race runtime_image image_security"
	assert_case .github/dependabot.yml \
		"dependency_automation" \
		"github_workflows go_source go_root_dependencies go_tool_dependencies runtime_image image_security"
	assert_case .github/workflows/ci.yml \
		"github_workflows" \
		"dependency_automation go_source go_root_dependencies go_tool_dependencies runtime_image image_security"
	assert_case internal/infra/postgres/queries/widgets.sql \
		"sqlc db_integration" \
		"go_source messaging_integration process_integration integration_race runtime_image image_security"
	assert_case internal/infra/natsjs/client.go \
		"go_source messaging_integration integration_race" \
		"db_integration migrations runtime_image image_security"

	local output name
	output="$(printf '%s\n' scripts/ci/changed-surfaces.sh | classify)"
	for name in "${names[@]}"; do
		grep -qx "${name}=true" <<<"${output}"
	done
	output="$(printf '%s\n' scripts/ci/template-sync-behavior-check.sh | classify)"
	grep -qx 'agent_instructions=true' <<<"${output}"
	grep -qx 'shell=true' <<<"${output}"
	output="$(bash "$0" --release)"
	for name in "${names[@]}"; do
		grep -qx "${name}=true" <<<"${output}"
	done
}

case "${1:-}" in
	--all|--release)
		reset
		mark "${names[@]}"
		emit
		;;
	--self-test)
		self_test
		;;
	"")
		classify
		;;
	*)
		echo "usage: $0 [--all|--release|--self-test]" >&2
		exit 2
		;;
esac
