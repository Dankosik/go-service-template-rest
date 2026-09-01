#!/usr/bin/env bash
set -euo pipefail

TEMPLATE_MODULE="github.com/example/go-service-template-rest"
TEMPLATE_SOURCE="github.com/Dankosik/go-service-template-rest"
TEMPLATE_OWNER="@Dankosik"
TEMPLATE_API_TITLE="go-service-template-rest"

PROFILE_ROOTS=(
	README.md Makefile railway.toml .gitleaks.toml .golangci.yml
	api build cmd docs env internal migrations test .github scripts/dev scripts/ci
)

usage() {
	echo "usage: CODEOWNER=@user-or-org/team DATABASE=none|postgres HTTP_IDEMPOTENCY=none|postgres JOBS=none|postgres WEBHOOKS=none|durable INBOUND_WEBHOOKS=none|standard-webhooks OUTBOX=none|postgres GRPC=none|enabled AUTHN=none|oidc-jwt|oidc-introspection OUTBOUND_HTTP=none|bounded OBJECT_STORAGE=none|s3 OUTBOUND_AUTH=none|oauth2-client-credentials MESSAGING=none|nats-jetstream REFERENCE_EXAMPLE=remove|keep AGENT_HARNESS=core|cursor|claude|qwen|grok|opencode|codex|all $0 [module-path]"
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

rewrite_temp_for() {
	local file="$1"
	local temporary

	temporary="$(mktemp "${file}.init-module.XXXXXX")"
	cp -p "${file}" "${temporary}"
	printf '%s\n' "${temporary}"
}

service_name_from_module() {
	local module="$1"
	local service_name="${module##*/}"

	if [[ "${service_name}" =~ ^v([2-9]|[1-9][0-9]+)$ ]]; then
		module="${module%/*}"
		service_name="${module##*/}"
	fi
	[[ -n "${service_name}" ]] || return 1
	printf '%s\n' "${service_name}"
}

replace_literal() {
	local file="$1"
	local old="$2"
	local new="$3"
	local temporary

	[[ "${old}" != "${new}" ]] || return 0

	temporary="$(rewrite_temp_for "${file}")"
	awk -v old="${old}" -v new="${new}" '{
		line = $0
		while ((index_at = index(line, old)) != 0) {
			line = substr(line, 1, index_at - 1) new substr(line, index_at + length(old))
		}
		print line
	}' "${file}" >"${temporary}"
	mv "${temporary}" "${file}"
}

# replace_required_literal is replace_literal for a rewrite whose absence is a
# defect rather than an ordinary miss. The module-path pass legitimately visits
# files that never mention the old value, so replace_literal cannot fail on a
# miss; the call sites below rewrite the template's own identity, where a miss
# means the derived service silently keeps it. awk reports nothing either way,
# so the check has to live here.
replace_required_literal() {
	local file="$1"
	local old="$2"
	local new="$3"

	if grep -qF -- "${old}" "${file}"; then
		replace_literal "${file}" "${old}" "${new}"
		return
	fi
	if ! grep -qF -- "${new}" "${file}"; then
		printf 'init-module: %s no longer contains "%s"; that rewrite would be a silent no-op\n' \
			"${file}" "${old}" >&2
		return 1
	fi
}

# replace_go_map_value rewrites one key's value in a Go map literal without
# encoding the gofmt column between them. That column widens whenever a longer
# key joins the block, which is exactly how the service_name rewrite became a
# silent no-op in 6bf6ac52. strip_profile's gofmt pass restores alignment after
# the new value changes the width.
replace_go_map_value() {
	local file="$1"
	local key="$2"
	local new="$3"
	local temporary

	if ! grep -qF -- "\"${key}\":" "${file}"; then
		printf 'init-module: %s no longer declares "%s"; that rewrite would be a silent no-op\n' \
			"${file}" "${key}" >&2
		return 1
	fi

	temporary="$(rewrite_temp_for "${file}")"
	awk -v key="\"${key}\":" -v new="\"${new}\"" '
		!replaced && index($0, key) {
			cut = index($0, key) + length(key)
			head = substr($0, 1, cut - 1)
			tail = substr($0, cut)
			if (!sub(/"[^"]*"/, new, tail)) exit 2
			$0 = head tail
			replaced = 1
		}
		{ print }
		END { if (!replaced) exit 2 }
	' "${file}" >"${temporary}" || {
		rm -f "${temporary}"
		printf 'init-module: %s no longer has a quoted value for "%s"\n' "${file}" "${key}" >&2
		return 1
	}
	mv "${temporary}" "${file}"
}

validate_profile_markers() {
	local file="$1"
	local profile="$2"

	awk -v start="profile:${profile}:start" -v finish="profile:${profile}:end" '
		index($0, start) {
			if (skip) exit 2
			skip = 1
			next
		}
		index($0, finish) {
			if (!skip) exit 2
			skip = 0
		}
		END { if (skip) exit 2 }
	' "${file}"
}

remove_profile_blocks() {
	local file="$1"
	local profile="$2"
	local temporary

	[[ -f "${file}" ]] || return 0
	temporary="$(rewrite_temp_for "${file}")"
	# The sentinel carries no comment prefix so the same markers work in shell,
	# YAML, Make, Dockerfile, and Go sources.
	if ! awk -v start="profile:${profile}:start" -v finish="profile:${profile}:end" '
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

# remove_profile_markers keeps a retained profile's content and deletes only its
# sentinel lines. The markers are the generator's own inputs; once a profile has
# been resolved they are inert comments that a service owner has to read, decide
# about, and carry forever.
remove_profile_markers() {
	local file="$1"
	local profile="$2"
	local temporary

	[[ -f "${file}" ]] || return 0
	temporary="$(rewrite_temp_for "${file}")"
	awk -v start="profile:${profile}:start" -v finish="profile:${profile}:end" '
		index($0, start) { next }
		index($0, finish) { next }
		{ print }
	' "${file}" >"${temporary}"
	mv "${temporary}" "${file}"
}

# remove_postgres_integration_tests deletes the PostgreSQL integration proofs.
# nullglob is scoped to this function so a pattern that matches nothing expands to
# nothing instead of to itself.
remove_postgres_integration_tests() {
	local previous_nullglob
	previous_nullglob="$(shopt -p nullglob || true)"
	shopt -s nullglob
	local file
	for file in test/postgres_*_test.go examples/*/postgres_*_test.go; do
		rm -f -- "${file}"
	done
	rm -f -- test/grpc_process_integration_test.go
	eval "${previous_nullglob}"
}

# remove_outbox_migrations deletes every migration that owns outbox schema,
# discovering them by name rather than listing them. A listed inventory keeps a
# later addition silently, and the generated service then fails only once its
# own migration run reaches a file whose tables were never created.
remove_outbox_migrations() {
	local previous_nullglob
	previous_nullglob="$(shopt -p nullglob || true)"
	shopt -s nullglob
	local file
	for file in migrations/[0-9]*_postgres_outbox*.sql; do
		rm -f -- "${file}"
	done
	eval "${previous_nullglob}"
}

# strip_profile applies one profile decision across every file that carries its
# markers, discovering them instead of listing them. A file missing from a
# hand-written list fails silently: the generated service keeps configuration it
# has no dependency for, or stops compiling because a marked type is gone but its
# users are not. Reformats the Go files it touched, because removing a block can
# leave a doubled or trailing blank line that `make fmt-check` rejects.
strip_profile() {
	local profile="$1"
	local mode="$2"
	local profile_file
	local stripped_go_files=("")
	local profile_files=("")

	while IFS= read -r profile_file; do
		[[ -n "${profile_file}" ]] || continue
		profile_files+=("${profile_file}")
	done < <(grep -rl \
		-e "profile:${profile}:start" \
		-e "profile:${profile}:end" \
		"${PROFILE_ROOTS[@]}" 2>/dev/null || true)

	for profile_file in "${profile_files[@]}"; do
		[[ -n "${profile_file}" ]] || continue
		if ! validate_profile_markers "${profile_file}" "${profile}"; then
			echo "invalid ${profile} profile markers in ${profile_file}"
			exit 1
		fi
	done

	for profile_file in "${profile_files[@]}"; do
		[[ -n "${profile_file}" ]] || continue
		if [[ "${mode}" == "remove" ]]; then
			remove_profile_blocks "${profile_file}" "${profile}"
		else
			remove_profile_markers "${profile_file}" "${profile}"
		fi
		if [[ "${profile_file}" == *.go ]]; then
			stripped_go_files+=("${profile_file}")
		fi
	done

	if ((${#stripped_go_files[@]} > 1)); then
		gofmt -w "${stripped_go_files[@]:1}"
	fi
}

# The idempotency profile adds the one branch that needs this source-only
# directive; the profile-off output is below the cyclomatic ceiling.
remove_http_idempotency_profile_lint() {
	local temporary
	temporary="$(rewrite_temp_for cmd/service/internal/bootstrap/run.go)"
	awk '/nolint:cyclop/ { next } { print }' cmd/service/internal/bootstrap/run.go >"${temporary}"
	mv "${temporary}" cmd/service/internal/bootstrap/run.go
}

lock_value() {
	local key="$1"
	local line

	line="$(grep -E "^${key} = \"" template.lock | head -n1 || true)"
	[[ -n "${line}" ]] || return 1
	line="${line#*\"}"
	printf '%s\n' "${line%%\"*}"
}

# template.lock is a resumable state journal. checkout_revision is deliberately
# the derived checkout's revision, not an unverified upstream template commit.
write_template_lock() {
	local state="$1"
	local checkout_revision temporary

	checkout_revision="$(git rev-parse HEAD 2>/dev/null || true)"
	[[ -n "${checkout_revision}" ]] || checkout_revision="unknown"
	if [[ -f template.lock ]]; then
		temporary="$(rewrite_temp_for template.lock)"
	else
		temporary="$(mktemp template.lock.init-module.XXXXXX)"
	fi

	cat >"${temporary}" <<EOF
# Generated by scripts/init-module.sh. checkout_revision identifies the local
# repository state that was initialized; template sync verifies its upstream
# revision independently.
state = "${state}"
source = "${TEMPLATE_SOURCE}"
checkout_revision = "${checkout_revision}"
module = "${new_module}"
original_module = "${initial_module}"
service_name = "${service_name}"
codeowner = "${codeowner}"
database = "${database}"
http_idempotency = "${http_idempotency}"
outbox = "${outbox}"
grpc = "${grpc}"
authn = "${authn}"
outbound_http = "${outbound_http}"
outbound_auth = "${outbound_auth}"
messaging = "${messaging}"
reference_example = "${reference_example}"
object_storage = "${object_storage}"
jobs = "${jobs}"
webhooks = "${webhooks}"
inbound_webhooks = "${inbound_webhooks}"
agent_harness = "${agent_harness}"
EOF
	mv "${temporary}" template.lock
}

validate_lock_selection() {
	local expected

	for expected in \
		"module = \"${new_module}\"" \
		"service_name = \"${service_name}\"" \
		"codeowner = \"${codeowner}\"" \
		"database = \"${database}\"" \
		"http_idempotency = \"${http_idempotency}\"" \
		"jobs = \"${jobs}\"" \
		"webhooks = \"${webhooks}\"" \
		"inbound_webhooks = \"${inbound_webhooks}\"" \
		"outbox = \"${outbox}\"" \
		"grpc = \"${grpc}\"" \
		"authn = \"${authn}\"" \
		"outbound_http = \"${outbound_http}\"" \
		"outbound_auth = \"${outbound_auth}\"" \
		"messaging = \"${messaging}\"" \
		"reference_example = \"${reference_example}\"" \
		"object_storage = \"${object_storage}\"" \
		"agent_harness = \"${agent_harness}\""; do
		grep -Fqx "${expected}" template.lock || {
			echo "repository is already initialized with different identity or profile choices"
			return 1
		}
	done
}

assert_no_profile_markers() {
	if grep -R -E 'profile:[a-z0-9-]+:(start|end)' "${PROFILE_ROOTS[@]}" 2>/dev/null; then
		echo "init-module: unresolved profile marker"
		return 1
	fi
}

verify_identity_postconditions() {
	grep -Fxq "module ${new_module}" go.mod
	grep -Fxq "module go-service-template-tools" tools/go.mod
	awk -v owner="${codeowner}" -v old="${TEMPLATE_OWNER}" '
		/^[[:space:]]*#/ { next }
		index($0, owner) { found = 1 }
		old != owner && index($0, old) { stale = 1 }
		END { exit !(found && !stale) }
	' .github/CODEOWNERS
	awk -v key='"observability.otel.service_name":' -v value="\"${service_name}\"" '
		index($0, key) && index($0, value) { found = 1 }
		END { exit !found }
	' internal/config/observability_config.go
	grep -Fq "\"service.name\", \"${service_name}\"" cmd/service/internal/bootstrap/run.go
	grep -Fxq "APP__OBSERVABILITY__OTEL__SERVICE_NAME=${service_name}" env/.env.example
	grep -Fq "title: \"${service_name}\"" api/openapi/service.yaml
	grep -Fxq "# ${service_name}" README.md
	grep -Fxq "Module: \`${new_module}\`" README.md
	assert_no_profile_markers
}

print_summary() {
	local message="$1"

	echo "${message}"
	echo "  module: ${new_module}"
	echo "  database: ${database}"
	echo "  HTTP idempotency: ${http_idempotency}"
	echo "  jobs: ${jobs}"
	echo "  webhooks: ${webhooks}"
	echo "  inbound webhooks: ${inbound_webhooks}"
	echo "  outbox: ${outbox}"
	echo "  gRPC: ${grpc}"
	echo "  authentication: ${authn}"
	echo "  outbound HTTP: ${outbound_http}"
	echo "  object storage: ${object_storage}"
	echo "  outbound authentication: ${outbound_auth}"
	echo "  messaging: ${messaging}"
	echo "  reference example: ${reference_example}"
	echo "  agent harness: ${agent_harness}"
	echo "  codeowner: ${codeowner}"
}

# strip_unselected_harness removes unused harness adapters from a derived service.
strip_unselected_harness() {
	local harness="$1"

	rm -f -- \
		scripts/ci/template-owned-purity-check.sh \
		scripts/ci/template-sync-behavior-check.sh

	if [[ "${harness}" == "all" ]]; then
		return 0
	fi

	if [[ "${harness}" != "claude" ]]; then
		rm -rf -- .claude
	fi
	if [[ "${harness}" != "qwen" ]]; then
		rm -rf -- .qwen
	fi
	if [[ "${harness}" != "codex" ]]; then
		rm -rf -- .codex
	fi
	if [[ "${harness}" != "cursor" ]]; then
		rm -rf -- .cursor/agents
		if [[ "${harness}" != "core" ]]; then
			rm -rf -- .cursor
		fi
	fi
	if [[ "${harness}" != "grok" ]]; then
		rm -rf -- .grok
	fi
	if [[ "${harness}" != "opencode" ]]; then
		rm -rf -- .opencode
		rm -f -- opencode.json
	fi
}

replace_codeowner_rules() {
	local owner="$1"
	local temporary

	if awk -v owner="${owner}" -v old="${TEMPLATE_OWNER}" '
		/^[[:space:]]*#/ { next }
		index($0, owner) { found = 1 }
		old != owner && index($0, old) { stale = 1 }
		END { exit !(found && !stale) }
	' .github/CODEOWNERS; then
		return
	fi

	temporary="$(rewrite_temp_for .github/CODEOWNERS)"
	awk -v old="${TEMPLATE_OWNER}" -v new="${owner}" '
		/^[[:space:]]*#/ { print; next }
		{ replaced += gsub(old, new); print }
		END { if (!replaced) exit 2 }
	' .github/CODEOWNERS >"${temporary}"
	mv "${temporary}" .github/CODEOWNERS
}

write_derived_readme() {
	local service_name="$1"
	local module="$2"
	local temporary

	temporary="$(rewrite_temp_for README.md)"
	cat >"${temporary}" <<EOF
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

<!-- profile:authn-oidc-jwt:start -->
Authentication uses OIDC discovery and signed JWT access tokens. Configure it
using \`docs/authentication.md\` before starting the service.
<!-- profile:authn-oidc-jwt:end -->
<!-- profile:authn-oidc-introspection:start -->
Authentication uses uncached RFC 7662 token introspection. Configure it using
\`docs/authentication.md\` before starting the service.
<!-- profile:authn-oidc-introspection:end -->
<!-- profile:messaging-nats-jetstream:start -->
This service includes typed events over NATS JetStream. Configure operator-owned
streams, composition-owned routes, and the separate consumer worker using
\`docs/durable-messaging.md\` before enabling it.
<!-- profile:messaging-nats-jetstream:end -->
<!-- profile:outbox-postgres:start -->
This service includes a River-backed PostgreSQL transactional outbox and its
separate NATS publication worker. Publication is at-least-once and consumers
must tolerate duplicate event IDs; see \`docs/postgres-transactional-outbox.md\`.
<!-- profile:outbox-postgres:end -->
<!-- profile:webhooks-durable:start -->
This service stages one durable job per webhook receiver and delivers it from
the shared jobs worker; see \`docs/outbound-webhook-delivery.md\`.
<!-- profile:webhooks-durable:end -->
EOF
	mv "${temporary}" README.md
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

if [[ "${OUTBOX+x}" == "x" && -z "${OUTBOX-}" ]]; then
	echo "OUTBOX must be one of: none, postgres"
	exit 1
fi
outbox="${OUTBOX:-none}"
case "${outbox}" in
none | postgres) ;;
*)
	echo "OUTBOX must be one of: none, postgres"
	exit 1
	;;
esac
if [[ "${outbox}" == "postgres" && "${database}" != "postgres" ]]; then
	echo "OUTBOX=postgres requires DATABASE=postgres"
	exit 1
fi

if [[ "${INBOX+x}" == "x" ]]; then
	echo "INBOX is no longer supported"
	exit 1
fi

if [[ "${HTTP_IDEMPOTENCY+x}" == "x" && -z "${HTTP_IDEMPOTENCY-}" ]]; then
	echo "HTTP_IDEMPOTENCY must be one of: none, postgres"
	exit 1
fi
http_idempotency="${HTTP_IDEMPOTENCY:-none}"
case "${http_idempotency}" in
none | postgres) ;;
*)
	echo "HTTP_IDEMPOTENCY must be one of: none, postgres"
	exit 1
	;;
esac
if [[ "${http_idempotency}" == "postgres" && "${database}" != "postgres" ]]; then
	echo "HTTP_IDEMPOTENCY=postgres requires DATABASE=postgres"
	exit 1
fi

if [[ "${JOBS+x}" == "x" && -z "${JOBS-}" ]]; then
	echo "JOBS must be one of: none, postgres"
	exit 1
fi
jobs="${JOBS:-none}"
case "${jobs}" in
none | postgres) ;;
*)
	echo "JOBS must be one of: none, postgres"
	exit 1
	;;
esac
if [[ "${jobs}" == "postgres" && "${database}" != "postgres" ]]; then
	echo "JOBS=postgres requires DATABASE=postgres"
	exit 1
fi

if [[ "${WEBHOOKS+x}" == "x" && -z "${WEBHOOKS-}" ]]; then
	echo "WEBHOOKS must be one of: none, durable"
	exit 1
fi
webhooks="${WEBHOOKS:-none}"
case "${webhooks}" in
none | durable) ;;
*)
	echo "WEBHOOKS must be one of: none, durable"
	exit 1
	;;
esac
if [[ "${webhooks}" == "durable" && ( "${database}" != "postgres" || "${jobs}" != "postgres" ) ]]; then
	echo "WEBHOOKS=durable requires DATABASE=postgres JOBS=postgres"
	exit 1
fi

if [[ "${INBOUND_WEBHOOKS+x}" == "x" && -z "${INBOUND_WEBHOOKS-}" ]]; then
	echo "INBOUND_WEBHOOKS must be one of: none, standard-webhooks"
	exit 1
fi
inbound_webhooks="${INBOUND_WEBHOOKS:-none}"
case "${inbound_webhooks}" in
none | standard-webhooks) ;;
*)
	echo "INBOUND_WEBHOOKS must be one of: none, standard-webhooks"
	exit 1
	;;
esac
if [[ "${inbound_webhooks}" == "standard-webhooks" && ( "${database}" != "postgres" || "${jobs}" != "postgres" ) ]]; then
	echo "INBOUND_WEBHOOKS=standard-webhooks requires DATABASE=postgres JOBS=postgres"
	exit 1
fi

if [[ "${GRPC+x}" == "x" && -z "${GRPC-}" ]]; then
	echo "GRPC must be one of: none, enabled"
	exit 1
fi
grpc="${GRPC:-none}"
case "${grpc}" in
none | enabled) ;;
*)
	echo "GRPC must be one of: none, enabled"
	exit 1
	;;
esac

if [[ "${AUTHN+x}" == "x" && -z "${AUTHN-}" ]]; then
	echo "AUTHN must be one of: none, oidc-jwt, oidc-introspection"
	exit 1
fi
authn="${AUTHN:-none}"
case "${authn}" in
none | oidc-jwt | oidc-introspection) ;;
*)
	echo "AUTHN must be one of: none, oidc-jwt, oidc-introspection"
	exit 1
	;;
esac

if [[ "${OUTBOUND_HTTP+x}" == "x" && -z "${OUTBOUND_HTTP-}" ]]; then
	echo "OUTBOUND_HTTP must be one of: none, bounded"
	exit 1
fi
outbound_http="${OUTBOUND_HTTP:-none}"
case "${outbound_http}" in
none | bounded) ;;
*)
	echo "OUTBOUND_HTTP must be one of: none, bounded"
	exit 1
	;;
esac

if [[ "${OBJECT_STORAGE+x}" == "x" && -z "${OBJECT_STORAGE-}" ]]; then
	echo "OBJECT_STORAGE must be one of: none, s3"
	exit 1
fi
object_storage="${OBJECT_STORAGE:-none}"
case "${object_storage}" in
none | s3) ;;
*)
	echo "OBJECT_STORAGE must be one of: none, s3"
	exit 1
	;;
esac

if [[ "${OUTBOUND_AUTH+x}" == "x" && -z "${OUTBOUND_AUTH-}" ]]; then
	echo "OUTBOUND_AUTH must be one of: none, oauth2-client-credentials"
	exit 1
fi
outbound_auth="${OUTBOUND_AUTH:-none}"
case "${outbound_auth}" in
none | oauth2-client-credentials) ;;
*)
	echo "OUTBOUND_AUTH must be one of: none, oauth2-client-credentials"
	exit 1
	;;
esac
if [[ "${MESSAGING+x}" == "x" && -z "${MESSAGING-}" ]]; then
	echo "MESSAGING must be one of: none, nats-jetstream"
	exit 1
fi
messaging="${MESSAGING:-none}"
case "${messaging}" in
none | nats-jetstream) ;;
*)
	echo "MESSAGING must be one of: none, nats-jetstream"
	exit 1
	;;
esac
if [[ "${outbox}" == "postgres" && "${messaging}" != "nats-jetstream" ]]; then
	echo "OUTBOX=postgres requires MESSAGING=nats-jetstream"
	exit 1
fi

# The reference example is upstream teaching material. Keeping it would make a
# generated service own extra packages and a second OpenAPI contract that it
# must lint, test, and regenerate forever.
reference_example="${REFERENCE_EXAMPLE:-remove}"
case "${reference_example}" in
remove | keep) ;;
*)
	echo "REFERENCE_EXAMPLE must be one of: remove, keep"
	exit 1
	;;
esac

if [[ "${AGENT_HARNESS+x}" == "x" && -z "${AGENT_HARNESS-}" ]]; then
	echo "AGENT_HARNESS must be one of: core, cursor, claude, qwen, grok, opencode, codex, all"
	exit 1
fi
agent_harness="${AGENT_HARNESS:-core}"
case "${agent_harness}" in
core | cursor | claude | qwen | grok | opencode | codex | all) ;;
*)
	echo "AGENT_HARNESS must be one of: core, cursor, claude, qwen, grok, opencode, codex, all"
	exit 1
	;;
esac

for required_file in go.mod tools/go.mod Makefile make/template.mk env/.env.example .github/CODEOWNERS .golangci.yml api/openapi/service.yaml; do
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

if [[ "${detected_module}" == "${TEMPLATE_SOURCE}" && "${current_module}" == "${TEMPLATE_MODULE}" ]]; then
	if [[ -n "${new_module}" ]]; then
		echo "refusing to initialize the canonical template source checkout"
		exit 1
	fi
	echo "template source checkout; initialization not required"
	exit 0
fi

if [[ -z "${new_module}" ]]; then
	[[ -n "${detected_module}" ]] || {
		echo "module path is required when git remote origin cannot be derived"
		usage
		exit 1
	}
	new_module="${detected_module}"
fi

if [[ "${new_module}" == "${TEMPLATE_MODULE}" ]]; then
	echo "derived repositories must replace the template module path"
	exit 1
fi

validation_mod="$(mktemp)"
cp go.mod "${validation_mod}"
trap 'rm -f "${validation_mod}"' EXIT
if ! go mod edit -module="${new_module}" "${validation_mod}" >/dev/null 2>&1; then
	echo "invalid Go module path: ${new_module}"
	exit 1
fi

codeowner="${CODEOWNER:-}"
if [[ -z "${codeowner}" && -f template.lock ]]; then
	codeowner="$(lock_value codeowner || true)"
fi
[[ -n "${codeowner}" ]] || {
	echo "CODEOWNER is required when initializing a repository derived from the template"
	exit 1
}
if [[ ! "${codeowner}" =~ ^@[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?(/[A-Za-z0-9]([A-Za-z0-9_-]*[A-Za-z0-9])?)?$ ]]; then
	echo "CODEOWNER must be one @username or @org/team-name token"
	exit 1
fi

service_name="$(service_name_from_module "${new_module}")" || {
	echo "failed to derive service name from module path: ${new_module}"
	exit 1
}

initial_module="${current_module}"
if [[ -f template.lock ]]; then
	initial_module="$(lock_value original_module || true)"
	[[ -n "${initial_module}" ]] || {
		echo "template.lock has no original module for safe resume"
		exit 1
	}
fi

if [[ -f template.lock ]]; then
	lock_state="$(lock_value state || true)"
	case "${lock_state}" in
	initializing | complete) ;;
	*)
		echo "template.lock has no supported initialization state"
		exit 1
		;;
	esac
	validate_lock_selection
	if [[ "${lock_state}" == "complete" ]]; then
		rm -rf -- scripts/profiles
		verify_identity_postconditions
		print_summary "template initialization already complete"
		exit 0
	fi
else
	write_template_lock initializing
fi

if [[ "${new_module}" != "${current_module}" ]]; then
	go mod edit -module="${new_module}"
	current_module="${new_module}"
fi
if [[ "${initial_module}" != "${new_module}" ]]; then
	while IFS= read -r file; do
		[[ -f "${file}" ]] || continue
		# Generated protobuf descriptors embed the go_package value as encoded
		# bytes. A textual replacement can change its length and corrupt the
		# descriptor, so rewrite the schema and regenerate these files below.
		if [[ "${file}" == internal/gen/proto/* || "${file}" == */internal/gen/proto/* ]]; then
			continue
		fi
		if grep -Fq "${initial_module}" "${file}"; then
			replace_literal "${file}" "${initial_module}" "${new_module}"
		fi
	done < <(git ls-files --cached --others --exclude-standard -- '*.go' '*.proto')

fi

if [[ "${source_checkout}" != true ]]; then
	replace_codeowner_rules "${codeowner}"
	if [[ -f internal/config/observability_config.go ]]; then
		replace_go_map_value \
			internal/config/observability_config.go \
			"observability.otel.service_name" \
			"${service_name}"
	fi
	if [[ -f cmd/service/internal/bootstrap/run.go ]]; then
		replace_required_literal \
			cmd/service/internal/bootstrap/run.go \
			"\"service.name\", \"service\"" \
			"\"service.name\", \"${service_name}\""
	fi
	replace_required_literal \
		env/.env.example \
		"APP__OBSERVABILITY__OTEL__SERVICE_NAME=service" \
		"APP__OBSERVABILITY__OTEL__SERVICE_NAME=${service_name}"
	replace_required_literal \
		api/openapi/service.yaml \
		"title: \"${TEMPLATE_API_TITLE}\"" \
		"title: \"${service_name}\""
	write_derived_readme "${service_name}" "${new_module}"
	if [[ "${outbox}" == "none" ]]; then
		rm -rf -- cmd/outbox-relay internal/infra/postgresoutbox
		rm -f -- \
			internal/infra/natsjs/outbox.go \
			internal/infra/natsjs/outbox_test.go \
			test/postgres_outbox_*_test.go \
			docs/postgres-transactional-outbox.md
		remove_outbox_migrations
		strip_profile outbox-postgres remove
	else
		# Existing adopters retain 000001 for rollback; a new service starts on River.
		rm -f -- migrations/000001_postgres_outbox.sql
		strip_profile outbox-postgres keep
	fi

	if [[ "${http_idempotency}" == "none" ]]; then
		rm -rf -- internal/httpidempotency internal/infra/postgresidempotency
		rm -f -- \
			cmd/service/internal/bootstrap/startup_idempotency.go \
			cmd/service/internal/bootstrap/startup_idempotency_test.go \
			internal/config/http_idempotency_config.go \
			internal/config/http_idempotency_config_test.go \
			internal/infra/http/idempotency.go \
			internal/infra/http/idempotency_test.go \
			internal/problem/idempotency_problem_test.go \
			migrations/000003_postgres_http_idempotency.sql \
			migrations/000009_postgres_http_idempotency_simplify.sql \
			internal/infra/postgres/queries/postgres_http_idempotency.sql \
			internal/infra/postgres/sqlcgen/postgres_http_idempotency.sql.go \
			test/postgres_http_idempotency_http_integration_test.go \
			test/postgres_http_idempotency_integration_test.go \
			docs/postgres-http-idempotency.md
		strip_profile http-idempotency-postgres remove
		remove_http_idempotency_profile_lint
	else
		strip_profile http-idempotency-postgres keep
	fi

		if [[ "${jobs}" == "none" ]]; then
		rm -rf -- cmd/jobs-worker
			rm -f -- \
			internal/config/jobs_config.go \
			internal/config/jobs_config_test.go \
			internal/config/jobs_worker_config.go \
			internal/config/jobs_worker_config_test.go \
			migrations/000004_postgres_jobs.sql \
			test/postgres_jobs_*_test.go \
			docs/postgres-durable-background-jobs.md
		strip_profile jobs-postgres remove
	else
		rm -f -- migrations/000004_postgres_jobs.sql
		strip_profile jobs-postgres keep
		fi

		if [[ "${webhooks}" == "none" ]]; then
			rm -rf -- internal/infra/postgreswebhook
			rm -f -- \
				docs/outbound-webhook-delivery.md \
				internal/config/webhooks_config.go \
				internal/config/webhooks_config_test.go \
				migrations/000005_postgres_webhooks.sql \
				migrations/000006_postgres_webhook_reference_repairs.sql \
				migrations/000007_postgres_webhooks_retire.sql \
				test/postgres_webhook_*_test.go
			if [[ "${inbound_webhooks}" == "none" ]]; then
				rm -f -- \
					cmd/jobs-worker/builder_webhooks.go \
					cmd/jobs-worker/builder_webhooks_test.go
			else
				rm -f -- cmd/jobs-worker/builder_webhooks_test.go
			fi
			strip_profile webhooks-durable remove
		else
			rm -f -- \
				migrations/000005_postgres_webhooks.sql \
				migrations/000006_postgres_webhook_reference_repairs.sql \
				migrations/000007_postgres_webhooks_retire.sql
			strip_profile webhooks-durable keep
		fi

		if [[ "${inbound_webhooks}" == "none" ]]; then
			rm -rf -- internal/inboundwebhook internal/infra/postgresinboundwebhook
			rm -f -- \
				cmd/jobs-worker/inbound_webhook_bindings.go \
				cmd/jobs-worker/builder_inbound_testworker.go \
				cmd/jobs-worker/builder_webhooks_inbound_test.go \
				cmd/jobs-worker/internal/bootstrap/run_inbound_test.go \
				cmd/service/internal/bootstrap/startup_inbound_webhooks.go \
				cmd/service/internal/bootstrap/startup_inbound_webhooks_test.go \
				cmd/service/internal/bootstrap/service_api.go \
				internal/config/inbound_webhooks_config.go \
				internal/config/inbound_webhooks_config_test.go \
				internal/infra/http/inbound_webhook.go \
				internal/infra/http/inbound_webhook_test.go \
				migrations/000010_postgres_inbound_webhooks.sql \
				internal/infra/postgres/queries/postgres_inbound_webhooks.sql \
				internal/infra/postgres/sqlcgen/postgres_inbound_webhooks.sql.go \
				docs/inbound-webhook-receipt.md \
				test/postgres_inbound_webhook_integration_test.go \
				test/inbound_webhook_process_integration_test.go
			if [[ "${webhooks}" == "none" ]]; then
				rm -f -- \
					cmd/jobs-worker/builder_webhooks.go \
					cmd/jobs-worker/builder_webhooks_test.go
			fi
			strip_profile inbound-webhooks-standard remove
			if [[ -f cmd/jobs-worker/builder_webhooks.go ]]; then
				temporary="$(rewrite_temp_for cmd/jobs-worker/builder_webhooks.go)"
				sed 's/ && !inbound_webhook_test_worker//' cmd/jobs-worker/builder_webhooks.go >"${temporary}"
				mv "${temporary}" cmd/jobs-worker/builder_webhooks.go
			fi
		else
			strip_profile inbound-webhooks-standard keep
		fi

	if [[ "${jobs}" == "none" && "${outbox}" == "none" && "${webhooks}" == "none" && "${inbound_webhooks}" == "none" ]]; then
		rm -f -- migrations/000008_river.sql
	fi

	# All PostgreSQL capability profiles derive from the same SQLC package. Resolve
	# their source sets before one regeneration so any sibling can remain alone.
	if ! find internal/infra/postgres/queries -type f -name '*.sql' -print -quit 2>/dev/null | grep -q .; then
		rm -rf -- internal/infra/postgres/queries internal/infra/postgres/sqlcgen
	else
		make sqlc-generate
	fi
	rmdir migrations 2>/dev/null || true

	if [[ "${database}" == "none" ]]; then
		# The migration runner and directory belong to the PostgreSQL profile.
		rm -rf -- cmd/migrate internal/infra/postgres internal/infra/postgresmigrate migrations
		# Matched rather than listed. Every test/postgres_*_test.go is PostgreSQL
		# integration proof by construction, and a new one that a hand-written list
		# missed leaves an import of a package this profile just deleted — which
		# surfaces as Go trying to resolve the generated module from the network.
		remove_postgres_integration_tests
		rm -f -- \
			cmd/internal/runtimeopts/postgres.go \
			cmd/service/internal/bootstrap/startup_dependencies.go \
			cmd/service/internal/bootstrap/startup_dependencies_test.go \
			cmd/service/internal/bootstrap/startup_rejections_test.go \
			internal/config/postgres_config.go \
			internal/config/postgres_config_test.go \
			internal/infra/telemetry/telemetrytest/metrics.go \
				env/docker-compose.yml
		cp \
			scripts/profiles/database-none/startup_dependencies.go.tmpl \
			cmd/service/internal/bootstrap/startup_dependencies.go
		replace_literal \
			cmd/service/internal/bootstrap/startup_dependencies.go \
			"${TEMPLATE_MODULE}" \
			"${new_module}"
		strip_profile database-postgres remove
		else
		strip_profile database-postgres keep
	fi

	if [[ "${authn}" == "none" ]]; then
		# authntrust exists only to be shared by the verifier and internal/config's
		# authn validation. With the profile off it has no caller at all, so it
		# leaves with them rather than becoming an unreferenced leaf.
		rm -rf -- internal/infra/bearerauthn internal/infra/oidcjwt internal/infra/oauthintrospection internal/authntrust
		rm -f -- \
			cmd/service/internal/bootstrap/authn_bootstrap_test.go \
			cmd/service/internal/bootstrap/startup_authn.go \
			cmd/service/internal/bootstrap/startup_authn_profile.go \
			internal/config/authn_config.go \
			internal/config/authn_config_test.go \
			internal/infra/http/authn_router_test.go \
			internal/infra/http/introspection_disclosure_test.go \
			internal/infra/httpclient/authn_policy_test.go \
			docs/authentication.md
		replace_literal api/openapi/service.yaml \
			'security: [{bearerAuth: []}]' \
			'security: []'
		strip_profile authn-bearer remove
		strip_profile authn-oidc-jwt remove
		strip_profile authn-oidc-introspection remove
	elif [[ "${authn}" == "oidc-introspection" ]]; then
		cp \
			scripts/profiles/authn-oidc-introspection/startup_authn_profile.go.tmpl \
			cmd/service/internal/bootstrap/startup_authn_profile.go
		replace_literal cmd/service/internal/bootstrap/startup_authn_profile.go \
			"${TEMPLATE_MODULE}" \
			"${new_module}"
		rm -rf -- internal/infra/oidcjwt
		rm -f -- \
			internal/authntrust/token_profile.go \
			internal/authntrust/token_profile_test.go
		strip_profile authn-bearer keep
		strip_profile authn-oidc-jwt remove
		strip_profile authn-oidc-introspection keep
	else
		rm -rf -- internal/infra/oauthintrospection
		rm -f -- \
			internal/authntrust/introspection.go \
			internal/authntrust/introspection_test.go
		strip_profile authn-bearer keep
		strip_profile authn-oidc-jwt keep
		strip_profile authn-oidc-introspection remove
	fi

	if [[ "${outbound_auth}" == "none" ]]; then
		rm -rf -- internal/infra/oauth2clientcredentials
		rm -f -- \
			internal/config/outbound_auth_config.go \
			internal/config/outbound_auth_config_test.go \
			docs/outbound-machine-authentication.md
		strip_profile outbound-auth-oauth2-client-credentials remove
	else
		strip_profile outbound-auth-oauth2-client-credentials keep
	fi

	if [[ "${object_storage}" == "none" ]]; then
		rm -rf -- internal/objectstorage internal/infra/s3
		rm -f -- \
			cmd/service/internal/bootstrap/startup_object_storage.go \
			cmd/service/internal/bootstrap/startup_object_storage_test.go \
			internal/config/object_storage_config.go \
			internal/config/object_storage_config_test.go \
			test/s3conformance/conformance_test.go \
			docs/s3-compatible-object-storage.md
		strip_profile object-storage remove
	else
		strip_profile object-storage keep
	fi

	if [[ "${authn}" == "none" && "${object_storage}" == "none" ]]; then
		strip_profile bootstrap-config remove
	else
		strip_profile bootstrap-config keep
	fi

if [[ "${outbound_auth}" == "oauth2-client-credentials" ]]; then
	strip_profile outbound-auth-http keep
else
	rm -f -- \
		internal/infra/oauth2clientcredentials/http.go \
		internal/infra/oauth2clientcredentials/http_test.go
	strip_profile outbound-auth-http remove
fi

if [[ "${outbound_auth}" == "oauth2-client-credentials" && "${grpc}" == "enabled" ]]; then
	strip_profile outbound-auth-grpc keep
else
	rm -f -- \
		internal/infra/oauth2clientcredentials/grpc.go \
		internal/infra/oauth2clientcredentials/grpc_test.go
	strip_profile outbound-auth-grpc remove
fi

	if [[ "${messaging}" == "none" ]]; then
		rm -rf -- cmd/worker internal/domainevent internal/infra/natsjs internal/messagingconfig
		rm -f -- \
			cmd/internal/runtimeopts/messaging.go \
			cmd/service/internal/bootstrap/startup_messaging.go \
			cmd/service/internal/bootstrap/startup_messaging_test.go \
			docs/durable-messaging.md \
			internal/config/messaging_config.go \
			internal/config/messaging_config_test.go \
			internal/config/configtest/messaging.go \
			test/nats_messaging_*_test.go \
			test/postgres_outbox_natsjs_integration_test.go
		strip_profile messaging-nats-jetstream remove
	else
		strip_profile messaging-nats-jetstream keep
	fi

	if [[ "${grpc}" == "none" ]]; then
		rm -rf -- \
			internal/gen/proto \
			internal/infra/grpc \
			internal/infra/grpcclient \
			examples/grpc-reference-service \
			docs/grpc
			rm -f -- \
				cmd/service/internal/bootstrap/startup_grpc.go \
			cmd/service/internal/bootstrap/startup_grpc_test.go \
			cmd/service/internal/bootstrap/startup_grpc_tls.go \
			cmd/service/internal/bootstrap/startup_grpc_tls_test.go \
			cmd/service/internal/bootstrap/startup_readiness_test.go \
			docs/grpc.md \
			internal/config/grpc_config.go \
			internal/config/grpc_config_test.go \
			internal/infra/bearerauthn/grpc.go \
			internal/infra/bearerauthn/grpc_test.go \
			internal/infra/bearerauthn/grpc_tls_contract_test.go \
			test/grpc_process_integration_test.go
		strip_profile grpc remove
		else
		strip_profile grpc keep
		make proto-generate
	fi

	# configtest remains in every profile because internal/config's test helpers
	# delegate environment isolation to it. Its messaging corpus still leaves
	# with the messaging profile above.

	# internal/waittest is deliberately not removed by any profile. It once left
	# with gRPC and messaging together, because every importer was one of their
	# integration tests; cmd/internal/runtimeopts' diagnostics test now reserves
	# its listen address through the same package, and that binary ships in every
	# profile. A package with an unconditional importer is not an unreferenced
	# leaf, so removing it would break the minimal service rather than trim it.

	# The template's own closed spec bundles are decisions about developing the
	# template, not about this service. specs/README.md already says a completed
	# bundle is deleted rather than kept as an example, and AGENTS.md tells every
	# harness that task-local artifacts own accepted task decisions — so shipping
	# them would hand a new service authoritative-looking records that describe a
	# repository it does not have. They stay readable upstream.
	rm -rf -- specs

	strip_unselected_harness "${agent_harness}"

	# Upstream project furniture. The hero image is referenced only by the README
	# initialization overwrites, and the issue forms route bug reports for the
	# template rather than for this service.
	rm -rf -- .github/assets .github/ISSUE_TEMPLATE

	if [[ "${outbound_http}" == "none" && "${authn}" == "none" && "${outbound_auth}" == "none" ]]; then
		rm -rf -- internal/infra/httpclient
	fi

	if [[ "${reference_example}" == "remove" ]]; then
		rm -rf -- examples
	fi

	# Authentication selection changes canonical OpenAPI security and removes
	# generator-only marker comments. Regenerate after the profile is resolved so
	# the initialized service starts with byte-current embedded contract output.
	if [[ -d internal/openapi ]]; then
		go generate ./internal/openapi
	fi

fi

go mod tidy
go -C tools mod tidy

if [[ ! -f .env ]]; then
	cp env/.env.example .env
	echo "created .env from env/.env.example"
fi

verify_identity_postconditions
write_template_lock complete

# This is the only fallible cleanup after the complete journal write. A rerun
# retries it before reporting success if the process stopped here.
rm -rf -- scripts/profiles
print_summary "template initialization complete"
