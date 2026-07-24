#!/usr/bin/env bash
set -euo pipefail

TEMPLATE_MODULE="github.com/example/go-service-template-rest"
TEMPLATE_SOURCE="github.com/Dankosik/go-service-template-rest"
TEMPLATE_OWNER="@Dankosik"
TEMPLATE_API_TITLE="go-service-template-rest"

usage() {
	echo "usage: CODEOWNER=@user-or-org/team DATABASE=none|postgres OUTBOUND_HTTP=none|bounded AGENT_WORKFLOW=none|full $0 [module-path]"
	echo "module-path is derived from git remote origin when omitted"
}

detect_module_from_origin() {
	local remote_url host path remainder

	remote_url="$(git config --get remote.origin.url 2>/dev/null || true)"
	[[ -n "${remote_url}" ]] || return 1

	case "${remote_url}" in
	git@*:*)
		host="${remote_url#git@}"
		host="${host%%:*}"
		path="${remote_url#*:}"
		;;
	ssh://git@*/*)
		remainder="${remote_url#ssh://git@}"
		host="${remainder%%/*}"
		path="${remainder#*/}"
		;;
	http://* | https://*)
		remainder="${remote_url#*://}"
		host="${remainder%%/*}"
		path="${remainder#*/}"
		;;
	*)
		return 1
		;;
	esac

	host="${host%%:*}"
	path="${path#/}"
	path="${path%/}"
	path="${path%.git}"
	[[ -n "${host}" && -n "${path}" ]] || return 1
	printf '%s/%s\n' "${host}" "${path}"
}

replace_literal() {
	local file="$1"
	local old="$2"
	local new="$3"
	local temporary

	temporary="$(mktemp)"
	awk -v old="${old}" -v new="${new}" '{
		line = $0
		while ((index_at = index(line, old)) != 0) {
			line = substr(line, 1, index_at - 1) new substr(line, index_at + length(old))
		}
		print line
	}' "${file}" >"${temporary}"
	cat "${temporary}" >"${file}"
	rm -f "${temporary}"
}

remove_profile_blocks() {
	local file="$1"
	local profile="$2"
	local temporary

	[[ -f "${file}" ]] || return 0
	temporary="$(mktemp)"
	if ! awk -v start="# profile:${profile}:start" -v finish="# profile:${profile}:end" '
		index($0, start) {
			if (skip) exit 2
			skip = 1
			next
		}
		index($0, finish) {
			if (!skip) exit 2
			skip = 0
			next
		}
		!skip { print }
		END {
			if (skip) exit 2
		}
	' "${file}" >"${temporary}"; then
		rm -f "${temporary}"
		echo "invalid ${profile} profile markers in ${file}"
		exit 1
	fi
	mv "${temporary}" "${file}"
}

replace_codeowner_rules() {
	local owner="$1"
	local temporary

	temporary="$(mktemp)"
	awk -v old="${TEMPLATE_OWNER}" -v new="${owner}" '
		/^[[:space:]]*#/ { print; next }
		{ gsub(old, new); print }
	' .github/CODEOWNERS >"${temporary}"
	cat "${temporary}" >.github/CODEOWNERS
	rm -f "${temporary}"
}

write_derived_readme() {
	local service_name="$1"
	local module="$2"

	cat >README.md <<EOF
# ${service_name}

Go HTTP service initialized from
[go-service-template-rest](https://github.com/Dankosik/go-service-template-rest).

Module: \`${module}\`

## Local development

\`\`\`bash
cp env/.env.example .env
make run
make check
\`\`\`

The client API contract is \`api/openapi/service.yaml\`. Start with
\`docs/first-production-feature.md\` for the first vertical slice and
\`docs/repo-architecture.md\` for ownership boundaries.
EOF
}

write_minimal_agents_contract() {
	cat >AGENTS.md <<'EOF'
# AGENTS.md

- Keep production Go under `internal/` and composition under `cmd/`.
- Treat `api/openapi/service.yaml` as canonical; never hand-edit generated Go.
- Prefer the standard library and existing dependencies before adding code or
  tools.
- Preserve context, errors, resource cleanup, readiness, and bounded shutdown.
- Run the narrowest focused test while iterating and `make check` before
  publishing.
EOF
}

if [[ $# -gt 1 ]]; then
	usage
	exit 1
fi

database="${DATABASE:-none}"
case "${database}" in
none | postgres) ;;
*)
	echo "DATABASE must be one of: none, postgres"
	exit 1
	;;
esac

outbound_http="${OUTBOUND_HTTP:-none}"
case "${outbound_http}" in
none | bounded) ;;
*)
	echo "OUTBOUND_HTTP must be one of: none, bounded"
	exit 1
	;;
esac

agent_workflow="${AGENT_WORKFLOW:-none}"
case "${agent_workflow}" in
none | full) ;;
*)
	echo "AGENT_WORKFLOW must be one of: none, full"
	exit 1
	;;
esac

for required_file in go.mod tools/go.mod env/.env.example .github/CODEOWNERS .golangci.yml api/openapi/service.yaml; do
	[[ -f "${required_file}" ]] || {
		echo "required template file not found: ${required_file}"
		exit 1
	}
done

current_module="$(awk '/^module / { print $2; exit }' go.mod)"
[[ -n "${current_module}" ]] || {
	echo "failed to read module path from go.mod"
	exit 1
}

detected_module="$(detect_module_from_origin || true)"
new_module="${1:-}"
source_checkout=false

if [[ -z "${new_module}" ]]; then
	[[ -n "${detected_module}" ]] || {
		echo "module path is required when git remote origin cannot be derived"
		usage
		exit 1
	}
	if [[ "${detected_module}" == "${TEMPLATE_SOURCE}" && "${current_module}" == "${TEMPLATE_MODULE}" ]]; then
		new_module="${current_module}"
		source_checkout=true
	else
		new_module="${detected_module}"
	fi
fi

validation_mod="$(mktemp)"
cp go.mod "${validation_mod}"
trap 'rm -f "${validation_mod}"' EXIT
if ! go mod edit -module="${new_module}" "${validation_mod}" >/dev/null 2>&1; then
	echo "invalid Go module path: ${new_module}"
	exit 1
fi

codeowner="${CODEOWNER:-}"
if [[ "${source_checkout}" != true ]]; then
	[[ -n "${codeowner}" ]] || {
		echo "CODEOWNER is required when initializing a repository derived from the template"
		exit 1
	}
	if [[ ! "${codeowner}" =~ ^@[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?(/[A-Za-z0-9]([A-Za-z0-9_-]*[A-Za-z0-9])?)?$ ]]; then
		echo "CODEOWNER must be one @username or @org/team-name token"
		exit 1
	fi
fi

if [[ "${new_module}" != "${current_module}" ]] && ! grep -Fq "${current_module}" .golangci.yml; then
	echo ".golangci.yml does not contain the current module path; refusing to disable depguard during initialization"
	exit 1
fi

if [[ "${new_module}" != "${current_module}" ]]; then
	go mod edit -module="${new_module}"
	go -C tools mod edit -module="${new_module}/tools"

	while IFS= read -r file; do
		[[ -f "${file}" ]] || continue
		if grep -Fq "${current_module}" "${file}"; then
			replace_literal "${file}" "${current_module}" "${new_module}"
		fi
	done < <(git ls-files --cached --others --exclude-standard -- '*.go' '*.proto')

	replace_literal .golangci.yml "${current_module}" "${new_module}"
fi

if [[ "${source_checkout}" != true ]]; then
	replace_codeowner_rules "${codeowner}"
	service_name="${new_module##*/}"
	if [[ -f Makefile ]]; then
		replace_literal Makefile "SERVICE_NAME := service" "SERVICE_NAME := ${service_name}"
	fi
	if [[ -f internal/config/defaults.go ]]; then
		replace_literal \
			internal/config/defaults.go \
			"\"observability.otel.service_name\":           \"service\"" \
			"\"observability.otel.service_name\":           \"${service_name}\""
	fi
	if [[ -f cmd/service/internal/bootstrap/run.go ]]; then
		replace_literal \
			cmd/service/internal/bootstrap/run.go \
			"\"service.name\", \"service\"" \
			"\"service.name\", \"${service_name}\""
	fi
	replace_literal \
		env/.env.example \
		"APP__OBSERVABILITY__OTEL__SERVICE_NAME=service" \
		"APP__OBSERVABILITY__OTEL__SERVICE_NAME=${service_name}"
	replace_literal \
		api/openapi/service.yaml \
		"title: \"${TEMPLATE_API_TITLE}\"" \
		"title: \"${service_name}\""
	write_derived_readme "${service_name}" "${new_module}"

	if [[ "${database}" == "none" ]]; then
		rm -rf -- cmd/migrate internal/infra/postgres internal/infra/postgresmigrate
		rm -f -- \
			test/postgres_integration_test.go \
			test/postgres_migrate_runner_integration_test.go \
			cmd/service/internal/bootstrap/startup_dependencies.go \
			cmd/service/internal/bootstrap/startup_dependencies_address_test.go \
			cmd/service/internal/bootstrap/startup_dependencies_test.go \
			cmd/service/internal/bootstrap/startup_rejections_test.go \
			cmd/service/internal/bootstrap/startup_retry_test.go \
			env/docker-compose.yml
		cp \
			scripts/profiles/database-none/startup_dependencies.go.tmpl \
			cmd/service/internal/bootstrap/startup_dependencies.go
		replace_literal \
			cmd/service/internal/bootstrap/startup_dependencies.go \
			"${current_module}" \
			"${new_module}"
		for profile_file in \
			Makefile \
			build/docker/Dockerfile \
			railway.toml \
			.github/workflows/ci.yml \
			.github/workflows/cd.yml \
			.github/dependabot.yml \
			env/.env.example \
			env/config/local.yaml; do
			remove_profile_blocks "${profile_file}" "database-postgres"
		done
		go -C tools mod edit -droptool=github.com/sqlc-dev/sqlc/cmd/sqlc
		go mod tidy
		go -C tools mod tidy
		rm -rf -- scripts/profiles/database-none
	fi

	if [[ "${outbound_http}" == "none" ]]; then
		rm -rf -- internal/infra/httpclient
	fi

	if [[ "${agent_workflow}" == "none" ]]; then
		remove_profile_blocks Makefile "agent-workflow-full"
		rm -rf -- .agents .codex .claude .qwen specs
		rm -f -- CLAUDE.md QWEN.md
		write_minimal_agents_contract
	fi
fi

go mod tidy

if [[ ! -f .env ]]; then
	cp env/.env.example .env
	echo "created .env from env/.env.example"
fi

echo "template initialization complete"
echo "  module: ${new_module}"
echo "  database: ${database}"
echo "  outbound HTTP: ${outbound_http}"
echo "  agent workflow: ${agent_workflow}"
if [[ -n "${codeowner}" ]]; then
	echo "  codeowner: ${codeowner}"
fi
