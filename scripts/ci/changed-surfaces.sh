#!/usr/bin/env bash
set -euo pipefail

names=(
	go_source go_handwritten go_generated go_testdata go_root_dependencies go_tool_dependencies go_lint_config
	openapi protobuf sqlc module_initializer integration_initializer
	agent_instructions shell github_workflows dependency_automation
	db_integration messaging_integration process_integration integration_race
	integration_race_messaging integration_race_outbox integration_race_webhook
	performance_harness migrations compose_environment runtime_image image_security
	publication_metadata secret_scanning documentation integration_records validation_system no_validation_required
)

reset() {
	local name
	unclassified_paths=()
	for name in "${names[@]}"; do
		printf -v "${name}" '%s' false
	done
}

mark() {
	local name
	[[ ${tracking_file:-false} == true ]] && matched=true
	for name in "$@"; do
		printf -v "${name}" '%s' true
	done
}

emit() {
	local name count=0 classified=true unclassified=''
	for name in "${names[@]}"; do
		printf '%s=%s\n' "${name}" "${!name}"
		if [[ ${name} != no_validation_required && ${!name} == true ]]; then ((count += 1)); fi
	done
	if ((${#unclassified_paths[@]})); then
		classified=false
		unclassified=$(IFS=,; echo "${unclassified_paths[*]}")
	fi
	printf 'classified=%s\nsurface_count=%s\nunclassified_files=%s\n' "${classified}" "${count}" "${unclassified}"
}

has_line() {
	[[ $'\n'"$1"$'\n' == *$'\n'"$2"$'\n'* ]]
}

classify() {
	local file matched
	reset
	unclassified_paths=()
	while IFS= read -r file; do
		[[ -n "${file}" ]] || continue
		tracking_file=true
		matched=false

		case "${file}" in
			internal/openapi/*.gen.go|examples/reference-service/internal/openapi/*.gen.go|internal/infra/*/internal/openapi/*.gen.go|internal/infra/postgres/sqlcgen/*.go|internal/gen/proto/*|examples/grpc-reference-service/internal/gen/proto/*)
				mark go_source go_generated
				;;
			*.go) mark go_source go_handwritten ;;
		esac
		case "${file}" in
			go.mod|go.sum|examples/*/go.mod|examples/*/go.sum) mark go_root_dependencies ;;
			tools/go.mod|tools/go.sum|scripts/ci/tools-smoke.sh|scripts/ci/tools-resolution-check.sh|scripts/ci/golangci-lint.sh) mark go_tool_dependencies ;;
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
			api/external/*/openapi.yaml|internal/infra/*/internal/openapi/*)
				mark integration_records
				;;
		esac
		case "${file}" in
			buf.yaml|buf.gen.yaml|buf.lock|api/proto/*|examples/grpc-reference-service/buf.yaml|examples/grpc-reference-service/buf.gen.yaml|examples/grpc-reference-service/buf.lock|examples/grpc-reference-service/api/proto/*|examples/grpc-reference-service/internal/gen/proto/*|internal/gen/proto/*)
				mark protobuf
				;;
		esac
		case "${file}" in
			api/proto/external/*|internal/gen/proto/external/*)
				mark integration_records
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
			scripts/integration-init.sh|scripts/openapi-ref-check.go|scripts/ci/integration-init-check.sh)
				mark integration_initializer
				;;
		esac
		case "${file}" in
			*.sh) mark shell ;;
		esac
		case "${file}" in
			test/performance/*|scripts/dev/benchmark.sh) mark performance_harness ;;
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
			cmd/jobs-worker/*|cmd/migrate/*|examples/reference-service/*|internal/infra/postgres/*|internal/infra/postgresidempotency/*|internal/infra/postgresjobs/*|internal/infra/postgresmigrate/*|internal/infra/postgreswebhook/*|internal/inboundwebhook/*|migrations/*|test/postgres*|test/inbound_webhook*|test/webhook*)
				mark db_integration
				;;
		esac
		case "${file}" in
			cmd/worker/*|cmd/outbox-relay/*|internal/domainevent/*|internal/messagingconfig/*|internal/infra/natsjs/*|internal/infra/postgresoutbox/*|test/nats*|test/postgres_outbox_natsjs*)
				mark messaging_integration
				;;
		esac
		case "${file}" in
			cmd/jobs-worker/*|cmd/*/main.go|cmd/*/internal/bootstrap/*|cmd/internal/runtimeopts/*|test/*process*|test/current_repository_fixture*)
				mark process_integration
				;;
		esac
		case "${file}" in
			internal/background/*|internal/infra/natsjs/*|cmd/worker/*|test/nats_messaging_delivery*|test/nats_messaging_drain*) mark integration_race integration_race_messaging ;;
		esac
		case "${file}" in
			internal/infra/postgresoutbox/*|cmd/outbox-relay/*|test/postgres_outbox*) mark integration_race integration_race_outbox ;;
		esac
		case "${file}" in
			internal/infra/postgreswebhook/*|test/postgres_webhook*|test/webhook_network*) mark integration_race integration_race_webhook ;;
		esac
		case "${file}" in
			cmd/migrate/*|internal/infra/postgresmigrate/*|migrations/*|scripts/ci/migration-*)
				mark migrations db_integration
				;;
		esac
		case "${file}" in
			integrations/*.toml|scripts/ci/integration-record-check.sh|scripts/ci/integration-record-constructor-check.go|scripts/ci/integration-record-bootstrap-check.go|scripts/ci/integration-record-grpc-check.go)
				mark integration_records
				;;
		esac
		case "${file}" in
			env/docker-compose.yml) mark compose_environment ;;
		esac
		case "${file}" in
			.dockerignore|build/docker/*|scripts/ci/runtime-image-*.sh)
				mark runtime_image image_security
				;;
		esac
		case "${file}" in
			.github/actions/publish-image/*|scripts/ci/publish-image-metadata.sh|THIRD_PARTY_NOTICES.md) mark publication_metadata ;;
		esac
		case "${file}" in
			.gitleaks.toml|.gitleaks.baseline.json|scripts/ci/install-gitleaks.sh) mark secret_scanning ;;
		esac
		case "${file}" in
			*/testdata/*) mark go_testdata ;;
		esac
		case "${file}" in
			env/.env.example|env/config/*) mark module_initializer ;;
		esac
		case "${file}" in
			internal/infra/*/client.go|internal/config/*_integration_config.go|cmd/service/internal/bootstrap/startup_*.go|docs/integrations/*)
				mark integration_records
				;;
		esac
		case "${file}" in
			*.md|docs/*) mark documentation ;;
		esac
		case "${file}" in
			.editorconfig|.gitattributes|.gitignore|.nvmrc|LICENSE|railway.toml|.codegraph/.gitignore|.github/CODEOWNERS|.github/assets/*|.github/ISSUE_TEMPLATE/*) mark no_validation_required ;;
			opencode.json) mark agent_instructions ;;
		esac

		case "${file}" in
				Makefile|make/template.mk|make/service.mk|scripts/ci/changed-surfaces.sh|scripts/ci/verify.sh|scripts/ci/validation-lock.sh|scripts/ci/affected-go-packages.sh|scripts/ci/git-changed-paths.sh|scripts/ci/measure.sh)
				mark validation_system
				;;
		esac
		if [[ ${matched} != true ]]; then unclassified_paths+=("${file}"); fi
	done
	tracking_file=false
	emit
	((${#unclassified_paths[@]} == 0))
}

union_classify() {
	local base_ref=$1 tmp files old_files base_script current old current_status old_status name value count=0
	local current_classified current_unclassified line
	tmp=$(mktemp -d)
	trap 'rm -rf -- "${tmp}"' RETURN
	files=${tmp}/files
	old_files=${tmp}/old-files
	base_script=${tmp}/changed-surfaces.sh
	cat >"${files}"
	: >"${old_files}"
	while IFS= read -r file; do
		[[ ${file} != scripts/ci/changed-surfaces.sh ]] || continue
		git cat-file -e "${base_ref}:${file}" 2>/dev/null && printf '%s\n' "${file}" >>"${old_files}"
	done <"${files}"
	git show "${base_ref}:scripts/ci/changed-surfaces.sh" >"${base_script}" || {
		echo "classifier base is unavailable: ${base_ref}" >&2
		return 2
	}
	set +e
	current=$(classify <"${files}")
	current_status=$?
	old=$(bash "${base_script}" <"${old_files}")
	old_status=$?
	set -e
	if ((old_status != 0)); then
		echo "base classifier failed: ${base_ref}" >&2
		return "${old_status}"
	fi
	for name in "${names[@]}"; do
		value=false
		if has_line "${current}" "${name}=true" || has_line "${old}" "${name}=true"; then value=true; fi
		printf '%s=%s\n' "${name}" "${value}"
		if [[ ${name} != no_validation_required && ${value} == true ]]; then ((count += 1)); fi
	done
	while IFS= read -r line; do
		case "${line}" in
		classified=*) current_classified=${line#*=} ;;
		unclassified_files=*) current_unclassified=${line#*=} ;;
		esac
	done <<<"${current}"
	printf 'classified=%s\nsurface_count=%s\nunclassified_files=%s\n' \
		"${current_classified}" \
		"${count}" \
		"${current_unclassified}"
	return "${current_status}"
}

assert_case() {
	local file=$1 true_names=$2 false_names=$3 output name
	output="$(printf '%s\n' "${file}" | classify)"
	for name in ${true_names}; do
		has_line "${output}" "${name}=true" || {
			printf '%s: expected %s=true\n%s\n' "${file}" "${name}" "${output}" >&2
			return 1
		}
	done
	for name in ${false_names}; do
		has_line "${output}" "${name}=false" || {
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
	assert_case scripts/ci/tools-resolution-check.sh \
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
	assert_case internal/messagingconfig/config.go \
		"go_source messaging_integration" \
		"db_integration migrations process_integration integration_race runtime_image image_security"
	assert_case test/performance/http/single-flow.js \
		"performance_harness" \
		"go_source db_integration messaging_integration process_integration integration_race runtime_image image_security"
	assert_case scripts/dev/benchmark.sh \
		"shell performance_harness" \
		"go_source db_integration messaging_integration process_integration integration_race runtime_image image_security"
		assert_case Makefile \
			"validation_system" \
			"go_source go_root_dependencies go_tool_dependencies go_lint_config module_initializer db_integration messaging_integration process_integration integration_race runtime_image image_security github_workflows"
		assert_case make/template.mk \
			"validation_system" \
			"go_source go_root_dependencies go_tool_dependencies go_lint_config module_initializer db_integration messaging_integration process_integration integration_race runtime_image image_security github_workflows"
		assert_case make/service.mk \
			"validation_system" \
			"go_source go_root_dependencies go_tool_dependencies go_lint_config module_initializer db_integration messaging_integration process_integration integration_race runtime_image image_security github_workflows"

	local output name
	output="$(printf '%s\n' scripts/ci/changed-surfaces.sh | classify)"
	has_line "${output}" 'shell=true'
	has_line "${output}" 'validation_system=true'
	output="$(printf '%s\n' scripts/ci/git-changed-paths.sh | classify)"
	has_line "${output}" 'shell=true'
	has_line "${output}" 'validation_system=true'
	output="$(printf '%s\n' scripts/ci/measure.sh | classify)"
	has_line "${output}" 'validation_system=true'
	has_line "${output}" 'classified=true'
	has_line "${output}" 'go_source=false'
	has_line "${output}" 'db_integration=false'

	output="$(printf '%s\n' scripts/ci/template-sync-behavior-check.sh | classify)"
	has_line "${output}" 'agent_instructions=true'
	has_line "${output}" 'shell=true'

	output="$(printf '%s\n' scripts/integration-init.sh | classify)"
	has_line "${output}" 'integration_initializer=true'

	for file in \
		integrations/billing.toml \
		api/external/billing/openapi.yaml \
		internal/infra/billing/internal/openapi/client.gen.go \
		internal/infra/billing/client.go \
		internal/config/billing_integration_config.go \
		cmd/service/internal/bootstrap/startup_billing.go \
		docs/integrations/billing.md \
		api/proto/external/identity/v1/identity.proto \
		internal/gen/proto/external/identity/v1/identity.pb.go; do
		output="$(printf '%s\n' "${file}" | classify)"
		has_line "${output}" 'integration_records=true'
	done
	output="$(bash "$0" --release)"
	for name in "${names[@]}"; do
		has_line "${output}" "${name}=true"
	done
	if output="$(printf '%s\n' unknown/new-owner.xyz | classify 2>&1)"; then
		echo "unknown paths must fail closed" >&2
		return 1
	fi
	has_line "${output}" 'classified=false'
	has_line "${output}" 'unclassified_files=unknown/new-owner.xyz'
	output="$(git ls-files | classify)"
	has_line "${output}" 'classified=true'
	output="$(printf '%s\n' scripts/ci/changed-surfaces.sh | bash "$0" --union HEAD)"
	has_line "${output}" 'shell=true'
	has_line "${output}" 'db_integration=false'
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
	--union)
		[[ -n ${2:-} ]] || { echo "usage: $0 --union BASE_REF" >&2; exit 2; }
		union_classify "$2"
		;;
	"")
		classify
		;;
	*)
		echo "usage: $0 [--all|--release|--self-test|--union BASE_REF]" >&2
		exit 2
		;;
esac
