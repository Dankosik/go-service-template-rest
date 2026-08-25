#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if [[ ${1:-} == --select-from-files ]]; then
	engine=false
	selected=' '
	add_selected() {
		case "${selected}" in
		*" $1 "*) return ;;
		esac
		selected+=" $1 "
	}
	while IFS= read -r file; do
		[[ -n ${file} ]] || continue
		case "${file}" in
		scripts/init-module.sh | scripts/ci/init-module-contract-check.sh | scripts/ci/template-init-check.sh | template-owned.paths | template.lock | env/.env.example | env/config/*)
			engine=true
			;;
		scripts/profiles/authn-oidc-introspection* | *oauthintrospection*)
			add_selected oidc-introspection
			;;
		scripts/profiles/authn-oidc-jwt* | *oidcjwt*)
			add_selected oidc-jwt
			;;
		scripts/profiles/database-* | scripts/profiles/*postgres* | *http-idempotency*)
			add_selected postgres
			;;
		scripts/profiles/jobs-* | */postgresjobs/*)
			add_selected jobs
			;;
		scripts/profiles/webhooks-* | */postgreswebhook/*)
			add_selected webhooks
			;;
		scripts/profiles/inbound-* | */inboundwebhook* | */postgresinboundwebhook/*)
			add_selected inbound-webhooks
			;;
		scripts/profiles/outbox-* | */postgresoutbox/*)
			add_selected outbox
			;;
		scripts/profiles/messaging-* | */messagingconfig/* | */natsjs/*)
			add_selected messaging
			;;
		scripts/profiles/object-storage* | */infra/s3/*)
			add_selected object-storage
			;;
		scripts/profiles/outbound-auth* | */oauth2clientcredentials/*)
			add_selected outbound-auth
			;;
		esac
	done
	if [[ ${engine} == true || ${selected} == ' ' ]]; then
		printf '%s\n' minimal oidc-jwt postgres outbound-auth
	else
		# shellcheck disable=SC2086
		printf '%s\n' minimal ${selected} | awk 'NF'
	fi
	exit 0
fi

if [[ ! -d "${ROOT_DIR}/scripts/profiles" ]]; then
	echo "initializer contract is template-only"
	exit 0
fi

tmp="$(mktemp -d -t init-module-check.XXXXXX)"
checkout="${tmp}/service"
trap 'rm -rf -- "${tmp}"' EXIT
mkdir -p "${checkout}"
fixture_index="${tmp}/fixture.index"
GIT_INDEX_FILE="${fixture_index}" git -C "${ROOT_DIR}" read-tree HEAD
GIT_INDEX_FILE="${fixture_index}" git -C "${ROOT_DIR}" add -A
fixture_tree="$(GIT_INDEX_FILE="${fixture_index}" git -C "${ROOT_DIR}" write-tree)"
git -C "${ROOT_DIR}" archive "${fixture_tree}" | tar -xf - -C "${checkout}"
git -C "${checkout}" init -q
git -C "${checkout}" remote add origin git@github.com:acme/service.git

git -C "${checkout}" add -A
git -C "${checkout}" -c user.name=init-check -c user.email=init-check@example.invalid commit -qm baseline

base_profile_env=(
	CODEOWNER=@acme/platform
	DATABASE=none
	HTTP_IDEMPOTENCY=none
	JOBS=none
	WEBHOOKS=none
	INBOUND_WEBHOOKS=none
	OUTBOX=none
	GRPC=none
	AUTHN=none
	OUTBOUND_HTTP=none
	OBJECT_STORAGE=none
	OUTBOUND_AUTH=none
	MESSAGING=none
	REFERENCE_EXAMPLE=remove
	AGENT_HARNESS=core
)

run_init() {
	local root="$1"
	local module="$2"
	shift 2
	local command=(bash ./scripts/init-module.sh)

	[[ -z "${module}" ]] || command+=("${module}")
	(
		cd "${root}"
		env -i \
			HOME="${HOME}" \
			PATH="${PATH}" \
			TMPDIR="${TMPDIR:-/tmp}" \
			"${base_profile_env[@]}" \
			"$@" \
			"${command[@]}"
	)
}

unchanged_failure() {
	local root="$1"
	local module="$2"
	shift 2

	if run_init "${root}" "${module}" "$@" >/dev/null 2>&1; then
		echo "initializer contract: expected unchanged failure: ${module} $*" >&2
		exit 1
	fi
	git -C "${root}" diff --quiet
	[[ -z "$(git -C "${root}" status --porcelain)" ]]
}

assert_no_profile_markers() {
	local root="$1"

	git -C "${root}" add -A
	if git -C "${root}" grep --cached -n -E \
		'profile:[a-z0-9-]+:(start|end)' -- \
		':!scripts/init-module.sh' \
		':!.agents/**'; then
		echo "initializer contract: unresolved profile marker" >&2
		exit 1
	fi
}

assert_identity() {
	local root="$1"
	local module="$2"
	local service_name="$3"
	local harness="$4"

	grep -Fxq "module ${module}" "${root}/go.mod"
	grep -Fxq "module ${module}/tools" "${root}/tools/go.mod"
	grep -Fxq 'state = "complete"' "${root}/template.lock"
	grep -Fxq "module = \"${module}\"" "${root}/template.lock"
	grep -Fxq 'original_module = "github.com/example/go-service-template-rest"' "${root}/template.lock"
	grep -Fxq "service_name = \"${service_name}\"" "${root}/template.lock"
	grep -Fxq 'codeowner = "@acme/platform"' "${root}/template.lock"
	grep -Fxq "agent_harness = \"${harness}\"" "${root}/template.lock"
	grep -Fxq "SERVICE_NAME := ${service_name}" "${root}/Makefile"
	awk -v key='"observability.otel.service_name":' -v value="\"${service_name}\"" '
		index($0, key) && index($0, value) { found = 1 }
		END { exit !found }
	' "${root}/internal/config/observability_config.go"
	grep -Fq "\"service.name\", \"${service_name}\"" "${root}/cmd/service/internal/bootstrap/run.go"
	grep -Fxq "APP__OBSERVABILITY__OTEL__SERVICE_NAME=${service_name}" "${root}/env/.env.example"
	grep -Fq "title: \"${service_name}\"" "${root}/api/openapi/service.yaml"
	grep -Fxq "# ${service_name}" "${root}/README.md"
	grep -Fxq "Module: \`${module}\`" "${root}/README.md"
	awk '
		/^[[:space:]]*#/ { next }
		/@Dankosik/ { stale = 1 }
		/@acme\/platform/ { found = 1 }
		END { exit !(found && !stale) }
	' "${root}/.github/CODEOWNERS"
	test -x "${root}/scripts/ci/init-module-contract-check.sh"
	test ! -e "${root}/scripts/profiles"
	assert_no_profile_markers "${root}"
}

assert_harness() {
	local root="$1"
	local harness="$2"

	case "${harness}" in
	core)
		test ! -e "${root}/.claude"
		test ! -e "${root}/.qwen"
		test ! -e "${root}/.codex"
		test ! -e "${root}/.cursor/agents"
		test -d "${root}/.cursor/rules"
		test ! -e "${root}/.grok"
		test ! -e "${root}/.opencode"
		test ! -e "${root}/opencode.json"
		;;
	all)
		test -d "${root}/.claude/agents"
		test -d "${root}/.qwen/agents"
		test -d "${root}/.codex/agents"
		test -d "${root}/.cursor/agents"
		test -d "${root}/.grok/agents"
		test -d "${root}/.opencode/agents"
		test -f "${root}/opencode.json"
		;;
	esac
}

assert_profile() {
	local root="$1"
	local name="$2"
	local dependencies=""

	case "${name}" in
	oidc-jwt | oidc-introspection)
		dependencies="$(cd "${root}" && go list -deps ./cmd/service)"
		;;
	esac
	case "${name}" in
	minimal)
		test ! -d "${root}/internal/infra/postgres"
		test ! -d "${root}/internal/infra/bearerauthn"
		test ! -d "${root}/internal/infra/natsjs"
		test ! -d "${root}/internal/messagingconfig"
		;;
	oidc-jwt)
		test -d "${root}/internal/infra/oidcjwt"
		test ! -d "${root}/internal/infra/oauthintrospection"
		grep -Fq 'internal/infra/oidcjwt' <<<"${dependencies}"
		grep -Fq 'TokenProfile' "${root}/internal/config/authn_config.go"
		! grep -Fq 'IntrospectionEndpoint' "${root}/internal/config/authn_config.go"
		;;
	oidc-introspection)
		test -d "${root}/internal/infra/oauthintrospection"
		test ! -d "${root}/internal/infra/oidcjwt"
		grep -Fq 'internal/infra/oauthintrospection' <<<"${dependencies}"
		grep -Fq 'IntrospectionEndpoint' "${root}/internal/config/authn_config.go"
		! grep -Fq 'TokenProfile' "${root}/internal/config/authn_config.go"
		! grep -E 'MicahParks/(keyfunc|jwkset)|golang-jwt/jwt' <<<"${dependencies}"
		;;
	postgres)
		test -d "${root}/internal/infra/postgres"
		test ! -e "${root}/migrations/000008_river.sql"
		! find "${root}/migrations" -type f -name '*_postgres_outbox*.sql' -print -quit 2>/dev/null | grep -q .
		;;
	http-idempotency)
		test -d "${root}/internal/infra/postgresidempotency"
		test -e "${root}/migrations/000003_postgres_http_idempotency.sql"
		;;
	jobs)
		test -d "${root}/cmd/jobs-worker"
		test -e "${root}/migrations/000004_postgres_jobs.sql"
		test -e "${root}/migrations/000008_river.sql"
		;;
	webhooks)
		test -d "${root}/internal/infra/postgreswebhook"
		test -d "${root}/cmd/jobs-worker"
		;;
	inbound-webhooks)
		test -d "${root}/internal/inboundwebhook/manifest"
		test -d "${root}/internal/infra/postgresinboundwebhook"
		test -e "${root}/migrations/000010_postgres_inbound_webhooks.sql"
		;;
	outbox)
		test -d "${root}/internal/infra/postgresoutbox"
		test ! -e "${root}/migrations/000001_postgres_outbox.sql"
		test -e "${root}/migrations/000008_river.sql"
		! grep -R -E 'type Outbox(Event|CommitReceipt|OrderingHead|Redrife)' "${root}/internal/infra/postgres/sqlcgen" 2>/dev/null
		;;
	messaging)
		test -d "${root}/internal/messagingconfig"
		test -d "${root}/internal/infra/natsjs"
		test -d "${root}/cmd/worker"
		test ! -d "${root}/internal/infra/postgres"
		;;
	object-storage)
		test -d "${root}/internal/infra/s3"
		;;
	outbound-auth)
		test -d "${root}/internal/infra/oauth2clientcredentials"
		test -f "${root}/internal/infra/oauth2clientcredentials/http.go"
		test -f "${root}/internal/infra/oauth2clientcredentials/grpc.go"
		;;
	esac
}

verify_template_sync() {
	local root="$1"
	local harness="$2"

	bash "${root}/scripts/template-sync.sh" --apply --from "${checkout}" --repo "${root}"
	bash "${root}/scripts/template-sync.sh" --check --from "${checkout}" --repo "${root}"
	assert_harness "${root}" "${harness}"
}

verify_profile() {
	local name="$1"
	local module="$2"
	local service_name="$3"
	local harness="$4"
	shift 4
	local overrides=("")
	local override
	local profile_checkout="${tmp}/service-${name}"
	for override in "$@"; do overrides+=("${override}"); done

	git clone -q "${checkout}" "${profile_checkout}"
	git -C "${profile_checkout}" remote set-url origin git@github.com:acme/service.git
	run_init "${profile_checkout}" "${module}" "${overrides[@]:1}"
	assert_identity "${profile_checkout}" "${module}" "${service_name}" "${harness}"
	assert_profile "${profile_checkout}" "${name}"
	assert_harness "${profile_checkout}" "${harness}"
	(
		cd "${profile_checkout}"
		go test -vet=off -run '^$' ./...
		GOFLAGS='' go mod tidy -diff
		if [[ "${name}" == "minimal" ]]; then
			if [[ ${INIT_MODULE_RESOLVE_TOOLS:-} == 1 ]]; then
				TOOLS_RESOLUTION_ALL=1 make tools-dependencies-check
			else
				make tools-mod-check
			fi
		fi
		make openapi-drift-check sqlc-check
		git add -A
		git -c user.name=init-check -c user.email=init-check@example.invalid commit -qm generated
	)

	if [[ "${name}" == "minimal" || "${harness}" == "all" ]]; then
		verify_template_sync "${profile_checkout}" "${harness}"
	fi
	if [[ "${name}" == "minimal" ]]; then
		run_init "${profile_checkout}" "${module}" "${overrides[@]:1}" >/dev/null
		git -C "${profile_checkout}" diff --quiet
		[[ -z "$(git -C "${profile_checkout}" status --porcelain)" ]]
		unchanged_failure "${profile_checkout}" github.com/acme/other "${overrides[@]:1}"
	fi
}

canonical_checkout="${tmp}/canonical-source"
git clone -q "${checkout}" "${canonical_checkout}"
git -C "${canonical_checkout}" remote set-url origin https://github.com/Dankosik/go-service-template-rest.git
run_init "${canonical_checkout}" "" >/dev/null
git -C "${canonical_checkout}" diff --quiet
[[ -z "$(git -C "${canonical_checkout}" status --porcelain)" ]]
unchanged_failure "${canonical_checkout}" github.com/acme/forbidden

unchanged_failure "${checkout}" "" CODEOWNER=
unchanged_failure "${checkout}" "" GRPC=
unchanged_failure "${checkout}" "" GRPC=custom
unchanged_failure "${checkout}" 'bad module'
unchanged_failure "${checkout}" github.com/example/go-service-template-rest
unchanged_failure "${checkout}" "" HTTP_IDEMPOTENCY=postgres
unchanged_failure "${checkout}" "" JOBS=postgres
unchanged_failure "${checkout}" "" WEBHOOKS=durable
unchanged_failure "${checkout}" "" INBOUND_WEBHOOKS=standard-webhooks
unchanged_failure "${checkout}" "" OUTBOX=postgres MESSAGING=nats-jetstream
unchanged_failure "${checkout}" "" DATABASE=postgres OUTBOX=postgres

partial_checkout="${tmp}/service-partial"
git clone -q "${checkout}" "${partial_checkout}"
git -C "${partial_checkout}" remote set-url origin git@github.com:acme/service.git
fail_bin="${tmp}/fail-bin"
mkdir -p "${fail_bin}"
real_go="$(command -v go)"
cat >"${fail_bin}/go" <<EOF
#!/usr/bin/env bash
if grep -Fxq 'state = "initializing"' template.lock 2>/dev/null; then
	count=0
	[[ ! -f "${tmp}/go-calls" ]] || count=\$(cat "${tmp}/go-calls")
	count=\$((count + 1))
	printf '%s\n' "\${count}" >"${tmp}/go-calls"
	if [[ "\${count}" == 2 ]]; then
		exit 97
	fi
fi
exec "${real_go}" "\$@"
EOF
chmod +x "${fail_bin}/go"
if run_init "${partial_checkout}" github.com/acme/service "PATH=${fail_bin}:${PATH}" >/dev/null 2>&1; then
	echo "initializer contract: injected partial failure unexpectedly succeeded" >&2
	exit 1
fi
grep -Fxq 'state = "initializing"' "${partial_checkout}/template.lock"
test -d "${partial_checkout}/scripts/profiles"
run_init "${partial_checkout}" github.com/acme/service >/dev/null
assert_identity "${partial_checkout}" github.com/acme/service service core

run_selected_profile() {
	local name=$1
	shift
	if [[ -n ${INIT_MODULE_PROFILES:-} ]]; then
		case " ${INIT_MODULE_PROFILES} " in
		*" ${name} "*) ;;
		*) return 0 ;;
		esac
	fi
	verify_profile "${name}" "$@"
}

run_selected_profile minimal github.com/acme/service service core
run_selected_profile oidc-jwt github.com/acme/service service core AUTHN=oidc-jwt GRPC=enabled
run_selected_profile oidc-introspection github.com/acme/service service core AUTHN=oidc-introspection GRPC=enabled
run_selected_profile postgres github.com/acme/service service core DATABASE=postgres
run_selected_profile http-idempotency github.com/acme/service service core DATABASE=postgres HTTP_IDEMPOTENCY=postgres
run_selected_profile jobs github.com/acme/service service core DATABASE=postgres JOBS=postgres
run_selected_profile webhooks github.com/acme/service service core DATABASE=postgres JOBS=postgres WEBHOOKS=durable
run_selected_profile inbound-webhooks github.com/acme/service service core DATABASE=postgres JOBS=postgres INBOUND_WEBHOOKS=standard-webhooks
run_selected_profile outbox github.com/acme/service service core DATABASE=postgres OUTBOX=postgres MESSAGING=nats-jetstream
run_selected_profile messaging github.com/acme/service service core MESSAGING=nats-jetstream
run_selected_profile object-storage github.com/acme/service service core OBJECT_STORAGE=s3
run_selected_profile outbound-auth github.com/acme/service/v2 service all GRPC=enabled OUTBOUND_HTTP=bounded OUTBOUND_AUTH=oauth2-client-credentials AGENT_HARNESS=all

echo "initializer contract passed"
