#!/usr/bin/env bash
set -euo pipefail

names=(
	go go_dependencies openapi protobuf sqlc migrations runtime_image shell
	github_actions agent_instructions template_generator documentation integration
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
			*.go) mark go ;;
		esac
		case "${file}" in
			.golangci.yml) mark go ;;
		esac
		case "${file}" in
			go.mod|go.sum|*/go.mod|*/go.sum) mark go go_dependencies integration ;;
		esac
		case "${file}" in
			.redocly.yaml|api/openapi/*|api/external/*/openapi.yaml|examples/reference-service/api/openapi.yaml|examples/reference-service/internal/openapi/*|internal/openapi/*|internal/infra/*/internal/openapi/*)
				mark openapi
				;;
		esac
		case "${file}" in
			buf.yaml|buf.gen.yaml|api/proto/*|examples/grpc-reference-service/buf.yaml|examples/grpc-reference-service/buf.gen.yaml|examples/grpc-reference-service/api/proto/*|examples/grpc-reference-service/internal/gen/proto/*|internal/gen/proto/*)
				mark protobuf
				;;
		esac
		case "${file}" in
			internal/infra/postgres/queries/*|internal/infra/postgres/sqlc.yaml|internal/infra/postgres/sqlcgen/*)
				mark sqlc integration
				;;
		esac
		case "${file}" in
			cmd/migrate/*|internal/infra/postgresmigrate/*|migrations/*|scripts/ci/migration-*)
				mark migrations integration
				;;
		esac
		case "${file}" in
			.dockerignore|build/docker/*|env/docker-compose.yml|scripts/ci/runtime-image-build.sh|.github/actions/publish-image/*)
				mark go runtime_image integration
				;;
		esac
		case "${file}" in
			*.sh) mark shell ;;
		esac
		case "${file}" in
			.github/workflows/*|.github/actions/*|.github/dependabot.yml) mark github_actions ;;
		esac
		case "${file}" in
			AGENTS.md|CLAUDE.md|Grok.md|QWEN.md|.agents/*|.claude/*|.codex/*|.cursor/*|.grok/*|.opencode/*|.qwen/*|docs/agent-harness/*|docs/spec-first-workflow/*|docs/prompt-*|docs/skill-authoring.md|docs/validation/*|scripts/agent-roles-sync.sh|scripts/harness-skills-sync.sh|scripts/codex-agents-sync.sh|scripts/template-sync.sh|scripts/ci/template-owned-purity-check.sh)
				mark agent_instructions
				;;
		esac
		case "${file}" in
			scripts/init-module.sh|scripts/integration-init.sh|scripts/ci/init-module-contract-check.sh|scripts/ci/integration-init-check.sh|scripts/ci/template-init-check.sh|scripts/profiles/*)
				mark go template_generator integration
				;;
			template-owned.paths|template.lock)
				mark template_generator
				;;
		esac
		case "${file}" in
			*.md|docs/*) mark documentation ;;
		esac
		case "${file}" in
			cmd/jobs-worker/*|cmd/outbox-relay/*|cmd/worker/*|internal/domainevent/*|internal/inboundwebhook/*|internal/infra/natsjs/*|internal/infra/postgres*|test/*)
				mark integration
				;;
		esac

		# These owners compose multiple surfaces; a change to them must exercise the
		# gates they route instead of trusting the classifier that changed.
		case "${file}" in
			Makefile)
				mark go openapi protobuf sqlc migrations runtime_image shell template_generator integration
				;;
			scripts/ci/changed-surfaces.sh)
				mark "${names[@]}"
				;;
		esac
	done
	emit
}

self_test() {
	local output name
	output="$(printf '%s\n' README.md | classify)"
	grep -qx 'documentation=true' <<<"${output}"
	grep -qx 'go=false' <<<"${output}"

	output="$(printf '%s\n' scripts/ci/changed-surfaces.sh | classify)"
	for name in "${names[@]}"; do
		grep -qx "${name}=true" <<<"${output}"
	done

	output="$(printf '%s\n' internal/infra/postgres/queries/widgets.sql | classify)"
	grep -qx 'go=false' <<<"${output}"
	grep -qx 'sqlc=true' <<<"${output}"
	grep -qx 'integration=true' <<<"${output}"

	output="$(bash "$0" --release)"
	grep -qx 'go=true' <<<"${output}"
	grep -qx 'integration=true' <<<"${output}"
	grep -qx 'agent_instructions=false' <<<"${output}"
}

case "${1:-}" in
	--all)
		reset
		mark "${names[@]}"
		emit
		;;
	--release)
		reset
		mark go go_dependencies migrations runtime_image integration
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
