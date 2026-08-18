#!/usr/bin/env bash
set -euo pipefail

TEMPLATE_MODULE="github.com/example/go-service-template-rest"
TEMPLATE_SOURCE="github.com/Dankosik/go-service-template-rest"
TEMPLATE_OWNER="@Dankosik"
TEMPLATE_API_TITLE="go-service-template-rest"

usage() {
	echo "usage: CODEOWNER=@user-or-org/team DATABASE=none|postgres HTTP_IDEMPOTENCY=none|postgres JOBS=none|postgres WEBHOOKS=none|durable OUTBOX=none|postgres GRPC=none|enabled AUTHN=none|oidc-jwt OUTBOUND_HTTP=none|bounded OBJECT_STORAGE=none|s3 OUTBOUND_AUTH=none|oauth2-client-credentials MESSAGING=none|nats-jetstream REFERENCE_EXAMPLE=remove|keep $0 [module-path]"
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

	if ! grep -qF -- "${old}" "${file}"; then
		printf 'init-module: %s no longer contains "%s"; that rewrite would be a silent no-op\n' \
			"${file}" "${old}" >&2
		return 1
	fi
	replace_literal "${file}" "${old}" "${new}"
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

	temporary="$(mktemp)"
	awk -v key="\"${key}\":" -v new="\"${new}\"" '
		!replaced && index($0, key) {
			cut = index($0, key) + length(key)
			head = substr($0, 1, cut - 1)
			tail = substr($0, cut)
			sub(/"[^"]*"/, new, tail)
			$0 = head tail
			replaced = 1
		}
		{ print }
	' "${file}" >"${temporary}"
	cat "${temporary}" >"${file}"
	rm -f "${temporary}"
}

remove_profile_blocks() {
	local file="$1"
	local profile="$2"
	local temporary

	[[ -f "${file}" ]] || return 0
	temporary="$(mktemp)"
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
	temporary="$(mktemp)"
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
	local stripped_go_files=()

	while IFS= read -r profile_file; do
		[[ -n "${profile_file}" ]] || continue
		if [[ "${mode}" == "remove" ]]; then
			remove_profile_blocks "${profile_file}" "${profile}"
		else
			remove_profile_markers "${profile_file}" "${profile}"
		fi
		if [[ "${profile_file}" == *.go ]]; then
			stripped_go_files+=("${profile_file}")
		fi
	done < <(grep -rl "profile:${profile}:start" \
		README.md Makefile railway.toml .gitleaks.toml .golangci.yml api build cmd docs env internal test .github scripts/dev scripts/ci 2>/dev/null || true)

	if ((${#stripped_go_files[@]} > 0)); then
		gofmt -w "${stripped_go_files[@]}"
	fi
}

# The idempotency profile adds the one branch that needs this source-only
# directive; the profile-off output is below the cyclomatic ceiling.
remove_http_idempotency_profile_lint() {
	local temporary
	temporary="$(mktemp)"
	awk '/nolint:cyclop/ { next } { print }' cmd/service/internal/bootstrap/run.go >"${temporary}"
	mv "${temporary}" cmd/service/internal/bootstrap/run.go
}

# write_template_lock records where this service came from. Initialization
# rewrites the module path, deletes packages, and resolves profiles in place, so
# without this there is no way to tell which upstream commit a service forked
# from — and therefore no way to review or pull a later upstream fix.
write_template_lock() {
	local database="$1"
	local http_idempotency="$2"
	local outbox="$3"
	local grpc="$4"
	local authn="$5"
	local outbound_http="$6"
	local outbound_auth="$7"
	local messaging="$8"
	local reference_example="$9"
	local object_storage="${10}"
	local jobs="${11}"
	local webhooks="${12}"
	local source_revision

	source_revision="$(git rev-parse HEAD 2>/dev/null || true)"
	[[ -n "${source_revision}" ]] || source_revision="unknown"

	cat >template.lock <<EOF
# Generated by scripts/init-module.sh. Records the template revision this
# service was derived from, so upstream fixes can be reviewed against it:
#
#   git remote add template https://github.com/Dankosik/go-service-template-rest.git
#   git fetch template
#   git log ${source_revision}..template/main -- cmd internal build Makefile
source = "${TEMPLATE_SOURCE}"
source_revision = "${source_revision}"
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
EOF
}

# rebase_coverage_floor turns the template's own coverage gate into a starting
# ratchet the new service can actually move.
#
# The gate measures effective coverage across the whole module, and on day one
# that denominator is almost entirely template code sitting near 82%. Against an
# 80.0 floor a team can add only about forty uncovered statements — one handler
# and a repository method — before CI goes red for a structural reason they did
# not create, in the week they are least equipped to debug it. The usual reaction
# is to raise the floor or switch it off, which is the wrong lesson to learn on
# day three.
#
# 70.0 leaves roughly two hundred and forty statements of runway while still
# failing on wholesale untested code. Raise it as the service's own tests land.
#
# The Makefile declares the floor and the workflows read it from there, so this
# one substitution moves the gate everywhere. An `if` rather than a trailing
# `&&`: this is the function's last statement, and under `set -e` a missing
# Makefile would abort initialization instead of skipping the rebase.
rebase_coverage_floor() {
	if [[ -f Makefile ]]; then
		replace_literal Makefile "COVERAGE_MIN ?= 80.0" "COVERAGE_MIN ?= 70.0"
	fi
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

<!-- profile:authn-oidc-jwt:start -->
Authentication uses strict OIDC discovery and signed JWT access tokens. Configure it
using \`docs/authentication.md\` before starting the service.
<!-- profile:authn-oidc-jwt:end -->
<!-- profile:messaging-nats-jetstream:start -->
This service includes the direct NATS JetStream messaging pack. Configure streams,
publisher limits, and the separate consumer worker using
\`docs/durable-messaging.md\` before enabling it.
<!-- profile:messaging-nats-jetstream:end -->
<!-- profile:outbox-postgres:start -->
This service includes the PostgreSQL transactional outbox and its separately
deployed bounded relay. Publication is at-least-once and consumers must tolerate
duplicate event IDs; see \`docs/postgres-transactional-outbox.md\`.
<!-- profile:outbox-postgres:end -->
<!-- profile:webhooks-durable:start -->
This service includes durable PostgreSQL-backed outbound webhook delivery and
its separately deployed worker; see \`docs/outbound-webhook-delivery.md\`.
<!-- profile:webhooks-durable:end -->
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
if [[ "${webhooks}" == "durable" && "${database}" != "postgres" ]]; then
	echo "WEBHOOKS=durable requires DATABASE=postgres"
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
	echo "AUTHN must be one of: none, oidc-jwt"
	exit 1
fi
authn="${AUTHN:-none}"
case "${authn}" in
none | oidc-jwt) ;;
*)
	echo "AUTHN must be one of: none, oidc-jwt"
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
if [[ "${outbound_auth}" == "oauth2-client-credentials" && "${outbound_http}" != "bounded" && "${grpc}" != "enabled" ]]; then
	echo "OUTBOUND_AUTH=oauth2-client-credentials requires OUTBOUND_HTTP=bounded or GRPC=enabled"
	exit 1
fi

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

# The reference example is upstream teaching material. Keeping it would make a
# generated service own five extra packages, a second OpenAPI contract, and a
# second main() that it must lint, test, and regenerate forever.
reference_example="${REFERENCE_EXAMPLE:-remove}"
case "${reference_example}" in
remove | keep) ;;
*)
	echo "REFERENCE_EXAMPLE must be one of: remove, keep"
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

if [[ -f template.lock ]]; then
	if [[ -d scripts/profiles ]]; then
		echo "template.lock exists while unresolved profile sources remain"
		exit 1
	fi
	for expected in \
		"database = \"${database}\"" \
		"http_idempotency = \"${http_idempotency}\"" \
		"jobs = \"${jobs}\"" \
		"webhooks = \"${webhooks}\"" \
		"outbox = \"${outbox}\"" \
		"grpc = \"${grpc}\"" \
		"authn = \"${authn}\"" \
		"outbound_http = \"${outbound_http}\"" \
		"outbound_auth = \"${outbound_auth}\"" \
		"messaging = \"${messaging}\"" \
		"reference_example = \"${reference_example}\"" \
		"object_storage = \"${object_storage}\""; do
		grep -Fqx "${expected}" template.lock || {
			echo "repository is already initialized with different profile choices"
			exit 1
		}
	done
	echo "template initialization already complete"
	echo "  module: ${new_module}"
	echo "  database: ${database}"
	echo "  jobs: ${jobs}"
	echo "  webhooks: ${webhooks}"
	echo "  outbox: ${outbox}"
	echo "  gRPC: ${grpc}"
	echo "  authentication: ${authn}"
	echo "  outbound HTTP: ${outbound_http}"
	echo "  object storage: ${object_storage}"
	echo "  outbound authentication: ${outbound_auth}"
	echo "  messaging: ${messaging}"
	echo "  reference example: ${reference_example}"
	exit 0
fi

if [[ "${new_module}" != "${current_module}" ]]; then
	go mod edit -module="${new_module}"
	go -C tools mod edit -module="${new_module}/tools"

	while IFS= read -r file; do
		[[ -f "${file}" ]] || continue
		# Generated protobuf descriptors embed the go_package value as encoded
		# bytes. A textual replacement can change its length and corrupt the
		# descriptor, so rewrite the schema and regenerate these files below.
		if [[ "${file}" == internal/gen/proto/* || "${file}" == */internal/gen/proto/* ]]; then
			continue
		fi
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
		replace_required_literal Makefile "SERVICE_NAME := service" "SERVICE_NAME := ${service_name}"
	fi
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
	rebase_coverage_floor

	if [[ "${outbox}" == "none" ]]; then
		rm -rf -- cmd/outbox-relay internal/infra/postgresoutbox
		rm -f -- \
			examples/reference-service/postgres_outbox_reconciliation_integration_test.go \
			internal/config/outbox_config.go \
			internal/config/outbox_config_test.go \
			internal/infra/natsjs/outbox_publisher.go \
			internal/infra/natsjs/outbox_publisher_test.go \
			internal/infra/postgres/queries/postgres_outbox.sql \
			internal/infra/postgres/sqlcgen/postgres_outbox.sql.go \
			test/postgres_outbox_*_test.go \
			docs/postgres-transactional-outbox.md
		remove_outbox_migrations
		strip_profile outbox-postgres remove
	else
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
			migrations/000008_river_jobs.sql \
			test/postgres_jobs_*_test.go \
			docs/postgres-durable-background-jobs.md
		strip_profile jobs-postgres remove
	else
		strip_profile jobs-postgres keep
		fi

		if [[ "${webhooks}" == "none" ]]; then
			rm -rf -- cmd/webhook-worker internal/infra/postgreswebhook
			rm -f -- \
				docs/outbound-webhook-delivery.md \
				internal/config/webhooks_config.go \
				internal/config/webhooks_config_test.go \
				migrations/000005_postgres_webhooks.sql \
				migrations/000006_postgres_webhook_reference_repairs.sql \
				internal/infra/postgres/queries/postgres_webhooks.sql \
				internal/infra/postgres/sqlcgen/postgres_webhooks.sql.go \
				test/postgres_webhook_*_test.go \
				test/webhook_network_integration_test.go \
				test/webhook_process_integration_test.go
			strip_profile webhooks-durable remove
		else
			strip_profile webhooks-durable keep
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
			scripts/ci/migration-source-check.sh \
			scripts/ci/migration-history-check.sh \
			scripts/ci/migration-check-self-test.sh \
			scripts/ci/migration-image-history-check.sh \
			scripts/ci/migration-publication-check.sh \
			env/docker-compose.yml
		cp \
			scripts/profiles/database-none/startup_dependencies.go.tmpl \
			cmd/service/internal/bootstrap/startup_dependencies.go
		replace_literal \
			cmd/service/internal/bootstrap/startup_dependencies.go \
			"${current_module}" \
			"${new_module}"
		strip_profile database-postgres remove
		go -C tools mod edit -droptool=github.com/sqlc-dev/sqlc/cmd/sqlc
		go -C tools mod edit -droptool=github.com/pressly/goose/v3/cmd/goose
		go mod tidy
		go -C tools mod tidy
	else
		strip_profile database-postgres keep
	fi

	if [[ "${authn}" == "none" ]]; then
		# authntrust exists only to be shared by the verifier and internal/config's
		# authn validation. With the profile off it has no caller at all, so it
		# leaves with them rather than becoming an unreferenced leaf.
		rm -rf -- internal/infra/oidcjwt internal/authntrust
		rm -f -- \
			cmd/service/internal/bootstrap/authn_bootstrap_test.go \
			cmd/service/internal/bootstrap/authn_readiness_test.go \
			cmd/service/internal/bootstrap/startup_authn.go \
			internal/config/authn_config.go \
			internal/config/authn_config_test.go \
			internal/infra/grpc/authn_health_test.go \
			internal/infra/http/authn_router_test.go \
			internal/infra/httpclient/authn_policy_test.go \
			docs/authentication.md
		replace_literal api/openapi/service.yaml \
			'security: [{bearerAuth: []}]' \
			'security: []'
		strip_profile authn-oidc-jwt remove
	else
		strip_profile authn-oidc-jwt keep
	fi

	if [[ "${outbound_auth}" == "none" ]]; then
		rm -rf -- internal/infra/oauth2clientcredentials
		rm -f -- \
			cmd/service/internal/bootstrap/startup_outbound_auth.go \
			cmd/service/internal/bootstrap/startup_outbound_auth_test.go \
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
			scripts/ci/s3-source-receipt.sh \
			test/s3conformance/conformance_test.go \
			docs/s3-compatible-object-storage.md
		strip_profile object-storage remove
	else
		strip_profile object-storage keep
	fi

	if [[ "${authn}" == "none" && "${outbound_auth}" == "none" && "${object_storage}" == "none" ]]; then
		strip_profile bootstrap-config remove
	else
		strip_profile bootstrap-config keep
	fi

if [[ "${outbound_auth}" == "oauth2-client-credentials" && "${outbound_http}" == "bounded" ]]; then
	strip_profile outbound-auth-http keep
else
	rm -f -- \
		internal/infra/httpclient/attempt_authorization.go \
		internal/infra/httpclient/attempt_authorization_test.go \
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

	if [[ "${authn}" == "none" && "${outbound_auth}" == "none" && "${object_storage}" == "none" ]]; then
		rm -f -- internal/infra/httpclient/credential_provider_contract_test.go
		strip_profile credential-provider-http remove
	else
		strip_profile credential-provider-http keep
	fi

	if [[ "${messaging}" == "none" ]]; then
		rm -rf -- cmd/worker internal/infra/natsjs
		rm -f -- \
			cmd/internal/runtimeopts/messaging.go \
			cmd/outbox-relay/internal/bootstrap/natsjs_publisher.go \
			cmd/outbox-relay/internal/bootstrap/natsjs_publisher_test.go \
			cmd/service/internal/bootstrap/startup_messaging.go \
			cmd/service/internal/bootstrap/startup_messaging_test.go \
			docs/durable-messaging.md \
			internal/config/messaging_config.go \
			internal/config/messaging_config_test.go \
			internal/config/configtest/messaging.go \
			test/nats_messaging_*_test.go \
			test/postgres_outbox_natsjs_integration_test.go
		strip_profile messaging-nats-jetstream remove
		go mod tidy
	else
		strip_profile messaging-nats-jetstream keep
	fi

	if [[ "${grpc}" == "none" ]]; then
		rm -rf -- \
			internal/gen/proto \
			internal/grpclimits \
			internal/infra/grpc \
			internal/infra/grpcclient \
			examples/grpc-reference-service
		rm -f -- \
			buf.yaml \
			buf.gen.yaml \
			examples/reference-service/grpc_failure_mapping_contract_test.go \
			cmd/service/internal/bootstrap/authn_readiness_test.go \
			cmd/service/internal/bootstrap/startup_grpc.go \
			cmd/service/internal/bootstrap/startup_grpc_test.go \
			cmd/service/internal/bootstrap/startup_grpc_tls.go \
			cmd/service/internal/bootstrap/startup_grpc_tls_test.go \
			docs/grpc.md \
			internal/config/grpc_config.go \
			internal/config/grpc_config_test.go \
			internal/infra/oidcjwt/grpc.go \
			internal/infra/oidcjwt/grpc_test.go \
			internal/infra/oidcjwt/grpc_tls_contract_test.go \
			test/grpc_process_integration_test.go \
			scripts/proto.sh \
			scripts/run-buf.sh \
			scripts/ci/proto-check.sh
		strip_profile grpc remove
		go -C tools mod edit -droptool=google.golang.org/protobuf/cmd/protoc-gen-go
		go -C tools mod edit -droptool=google.golang.org/grpc/cmd/protoc-gen-go-grpc
	else
		strip_profile grpc keep
		bash ./scripts/proto.sh generate
	fi

	# The end-to-end gRPC benchmark imports the removable reference schema and
	# command. Keep its shared runner/Make/docs surface only when both owners are
	# retained; the in-package grpcx microbenchmarks remain gRPC-runtime-owned.
	if [[ "${grpc}" == "enabled" && "${reference_example}" == "keep" ]]; then
		strip_profile grpc-reference-benchmark keep
	else
		rm -rf -- test/performance/grpc
		rm -f -- scripts/dev/benchmark-grpc-check.sh
		strip_profile grpc-reference-benchmark remove
	fi

	# internal/config/configtest exists for parity tests that hold runtime owners
	# and internal/config to one answer. Retained Postgres, jobs, outbox, and
	# outbound-auth tests also use it, so it leaves only after every current importer.
	if [[ "${database}" == "none" && "${jobs}" == "none" && "${outbox}" == "none" && "${grpc}" == "none" && "${authn}" == "none" && "${outbound_auth}" == "none" && "${messaging}" == "none" ]]; then
		rm -rf -- internal/config/configtest
	fi

	# internal/packagetest owns the source walk behind the comment and doc proofs
	# in retained PostgreSQL outbox code as well as grpc, authn, outbound auth,
	# and natsjs. It leaves only after every current importer.
	if [[ "${http_idempotency}" == "none" && "${outbox}" == "none" && "${grpc}" == "none" && "${authn}" == "none" && "${outbound_auth}" == "none" && "${messaging}" == "none" ]]; then
		rm -rf -- internal/packagetest
	fi

	# internal/waittest is deliberately not removed by any profile. It once left
	# with gRPC and messaging together, because every importer was one of their
	# integration tests; cmd/internal/runtimeopts' diagnostics test now reserves
	# its listen address through the same package, and that binary ships in every
	# profile. A package with an unconditional importer is not an unreferenced
	# leaf, so removing it would break the minimal service rather than trim it.

	# Profile sources are the generator's own inputs. Every profile has consumed
	# what it needs by now, so no generated service keeps them.
	rm -rf -- scripts/profiles

	# The template's own closed spec bundles are decisions about developing the
	# template, not about this service. specs/README.md already says a completed
	# bundle is deleted rather than kept as an example, and AGENTS.md tells every
	# harness that task-local artifacts own accepted task decisions — so shipping
	# them would hand a new service authoritative-looking records that describe a
	# repository it does not have. They stay readable upstream.
	rm -rf -- specs

	# Upstream project furniture. The hero image is referenced only by the README
	# initialization overwrites, and the issue forms route bug reports for the
	# template rather than for this service.
	rm -rf -- .github/assets .github/ISSUE_TEMPLATE

		if [[ "${outbound_http}" == "none" && "${authn}" == "none" && "${outbound_auth}" == "none" && "${object_storage}" == "none" && "${webhooks}" == "none" ]]; then
			rm -rf -- internal/infra/httpclient
		fi
	if [[ "${outbound_http}" == "none" && "${authn}" == "none" && "${outbound_auth}" == "none" && "${object_storage}" == "none" && "${webhooks}" == "none" ]]; then
		rm -rf -- internal/outboundtrust
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

	write_template_lock "${database}" "${http_idempotency}" "${outbox}" "${grpc}" "${authn}" "${outbound_http}" "${outbound_auth}" "${messaging}" "${reference_example}" "${object_storage}" "${jobs}" "${webhooks}"

fi

go mod tidy
go -C tools mod tidy

if [[ ! -f .env ]]; then
	cp env/.env.example .env
	echo "created .env from env/.env.example"
fi

echo "template initialization complete"
echo "  module: ${new_module}"
echo "  database: ${database}"
echo "  HTTP idempotency: ${http_idempotency}"
echo "  jobs: ${jobs}"
echo "  webhooks: ${webhooks}"
echo "  outbox: ${outbox}"
echo "  gRPC: ${grpc}"
echo "  authentication: ${authn}"
echo "  outbound HTTP: ${outbound_http}"
echo "  object storage: ${object_storage}"
echo "  outbound authentication: ${outbound_auth}"
echo "  messaging: ${messaging}"
echo "  reference example: ${reference_example}"
if [[ -n "${codeowner}" ]]; then
	echo "  codeowner: ${codeowner}"
fi
