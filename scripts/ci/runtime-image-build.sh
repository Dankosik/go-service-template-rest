#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
IMAGE="${1:-service:ci}"
FIXTURE="${2:-health-only}"

if [[ "$#" -gt 2 ]]; then
	echo "usage: $0 [image] [health-only|postgres-http-idempotency-active]" >&2
	exit 2
fi

case "${FIXTURE}" in
	health-only | postgres-http-idempotency-active) ;;
	*)
		echo "unknown runtime image fixture: ${FIXTURE}" >&2
		exit 2
		;;
esac

build_image() {
	local context="$1"
	local worktree_fingerprint
	worktree_fingerprint="$({
		git -C "${context}" rev-parse HEAD
		git -C "${context}" --no-pager diff --no-ext-diff --binary HEAD --
		while IFS= read -r -d '' file; do
			if [[ -f "${context}/${file}" ]]; then
				file_digest="$(shasum -a 256 "${context}/${file}" | awk '{print $1}')"
				printf '%s  %s\n' "${file_digest}" "${file}"
			fi
		done < <(git -C "${context}" ls-files -z --others --exclude-standard)
	} | shasum -a 256 | awk '{print $1}')"

	docker build \
		--build-arg "APP_VERSION=${APP_VERSION:-dev}" \
		--build-arg "VCS_REF=${VCS_REF:-unknown}" \
		--build-arg "SOURCE_URL=${SOURCE_URL:-}" \
		--build-arg "WORKTREE_FINGERPRINT=${worktree_fingerprint}" \
		-f "${context}/build/docker/Dockerfile" \
		-t "${IMAGE}" \
		"${context}"
}

# profile:messaging-nats-jetstream:start
verify_worker_image() {
	local output
	if output="$(docker run --rm --read-only --network none --env APP__MESSAGING__ENABLED=false --entrypoint /worker "${IMAGE}" 2>&1)"; then
		echo "runtime image worker exited successfully under its fail-closed smoke configuration" >&2
		exit 1
	fi
	# The source template has no handler builder. A derived service with a real
	# builder crosses that boundary and rejects the explicitly disabled transport.
	# Both outcomes prove the image contains an executable worker without network I/O.
	if ! grep -Fq 'worker feature handler builder is not registered' <<<"${output}" &&
		! grep -Fq 'messaging must be enabled for worker' <<<"${output}"; then
		echo "runtime image worker did not execute the expected fail-closed binary" >&2
		echo "${output}" >&2
		exit 1
	fi
}
# profile:messaging-nats-jetstream:end

# profile:outbox-postgres:start
verify_outbox_relay_image() {
	local output
	if output="$(docker run --rm --read-only --network none --entrypoint /outbox-relay "${IMAGE}" 2>&1)"; then
		echo "runtime image outbox relay exited successfully without PostgreSQL" >&2
		exit 1
	fi
	if ! grep -Eq '^(postgres outbox config: postgres must be enabled for outbox relay|load outbox relay config: config validate: )' <<<"${output}"; then
		echo "runtime image outbox relay did not execute the expected fail-closed binary" >&2
		echo "${output}" >&2
		exit 1
	fi
}
# profile:outbox-postgres:end

# profile:jobs-postgres:start
verify_jobs_worker_image() {
	local output
	if output="$(docker run --rm --read-only --network none --entrypoint /jobs-worker "${IMAGE}" 2>&1)"; then
		echo "runtime image jobs worker exited successfully without a builder" >&2
		exit 1
	fi
	if ! grep -Eq 'jobs worker builder is not registered|jobs and postgres must be enabled for jobs-worker|webhooks must be enabled for the template jobs worker|config validate: jobs worker requires postgres.enabled' <<<"${output}"; then
		echo "runtime image jobs worker did not execute the expected fail-closed binary" >&2
		echo "${output}" >&2
		exit 1
	fi
}
# profile:jobs-postgres:end

# Generated services no longer own profile sources, so their checkout is
# already the exact production source and needs no fixture.
if [[ ! -d "${ROOT_DIR}/scripts/profiles" ]]; then
	build_image "${ROOT_DIR}"
	# profile:messaging-nats-jetstream:start
	verify_worker_image
	# profile:messaging-nats-jetstream:end
	# profile:outbox-postgres:start
	verify_outbox_relay_image
	# profile:outbox-postgres:end
	# profile:jobs-postgres:start
	verify_jobs_worker_image
	# profile:jobs-postgres:end
	exit 0
fi

# The upstream template intentionally contains every mutually exclusive pack.
# It is not itself a runnable initialized service: an enabled OIDC pack must
# reject absent trust configuration. Build one deterministic production-shaped
# profile for image lifecycle proof instead of weakening that fail-closed
# contract or inventing a development bypass.
TEMP_ROOT="$(mktemp -d -t runtime-image-build.XXXXXX)"
CHECKOUT="${TEMP_ROOT}/service"
cleanup() {
	rm -rf -- "${TEMP_ROOT}"
}
trap cleanup EXIT INT TERM

mkdir -p "${CHECKOUT}"
while IFS= read -r -d '' file; do
	if [[ ! -f "${ROOT_DIR}/${file}" && ! -L "${ROOT_DIR}/${file}" ]]; then
		continue
	fi
	mkdir -p "${CHECKOUT}/$(dirname "${file}")"
	cp -P "${ROOT_DIR}/${file}" "${CHECKOUT}/${file}"
done < <(git -C "${ROOT_DIR}" ls-files -z --cached --others --exclude-standard)

git -C "${CHECKOUT}" init -q
git -C "${CHECKOUT}" \
	-c user.name=runtime-image-build \
	-c user.email=runtime-image-build@invalid \
	commit --allow-empty -qm "runtime image fixture"

profile_environment=(
	"CODEOWNER=@acme/platform"
	"DATABASE=postgres"
	"HTTP_IDEMPOTENCY=none"
	"JOBS=none"
	"GRPC=enabled"
	"AUTHN=none"
	"OUTBOUND_HTTP=bounded"
	"OBJECT_STORAGE=none"
	# profile:outbound-auth-oauth2-client-credentials:start
	"OUTBOUND_AUTH=none"
	# profile:outbound-auth-oauth2-client-credentials:end
	# profile:outbox-postgres:start
	"OUTBOX=postgres"
	# profile:outbox-postgres:end
	# profile:jobs-postgres:start
	"JOBS=postgres"
	# profile:jobs-postgres:end
	# profile:webhooks-durable:start
	"WEBHOOKS=durable"
	# profile:webhooks-durable:end
	"REFERENCE_EXAMPLE=remove"
)
# profile:messaging-nats-jetstream:start
profile_environment+=("MESSAGING=nats-jetstream")
# profile:messaging-nats-jetstream:end
if [[ "${FIXTURE}" == "postgres-http-idempotency-active" ]]; then
	profile_environment+=("AUTHN=oidc-jwt" "HTTP_IDEMPOTENCY=postgres")
fi
(
	cd "${CHECKOUT}"
	env "${profile_environment[@]}" bash ./scripts/init-module.sh github.com/acme/runtime-image-proof
	if [[ "${FIXTURE}" == "postgres-http-idempotency-active" ]]; then
		git apply --recount "${ROOT_DIR}/scripts/ci/fixtures/postgres-http-idempotency-active.patch"
		make openapi-generate
		go mod tidy
	fi
)

grep -Fqx 'database = "postgres"' "${CHECKOUT}/template.lock"
grep -Fqx 'grpc = "enabled"' "${CHECKOUT}/template.lock"
if [[ "${FIXTURE}" == "postgres-http-idempotency-active" ]]; then
	grep -Fqx 'authn = "oidc-jwt"' "${CHECKOUT}/template.lock"
else
	grep -Fqx 'authn = "none"' "${CHECKOUT}/template.lock"
fi
grep -Fqx 'outbound_http = "bounded"' "${CHECKOUT}/template.lock"
# profile:outbox-postgres:start
grep -Fqx 'outbox = "postgres"' "${CHECKOUT}/template.lock"
# profile:outbox-postgres:end
# profile:jobs-postgres:start
grep -Fqx 'jobs = "postgres"' "${CHECKOUT}/template.lock"
# profile:jobs-postgres:end
# profile:webhooks-durable:start
grep -Fqx 'webhooks = "durable"' "${CHECKOUT}/template.lock"
# profile:webhooks-durable:end
# profile:messaging-nats-jetstream:start
grep -Fqx 'messaging = "nats-jetstream"' "${CHECKOUT}/template.lock"
# profile:messaging-nats-jetstream:end
if [[ "${FIXTURE}" == "health-only" ]]; then
	[[ ! -d "${CHECKOUT}/internal/infra/oidcjwt" ]]
fi

build_image "${CHECKOUT}"
# profile:messaging-nats-jetstream:start
verify_worker_image
# profile:messaging-nats-jetstream:end
# profile:outbox-postgres:start
verify_outbox_relay_image
# profile:outbox-postgres:end
# profile:jobs-postgres:start
verify_jobs_worker_image
# profile:jobs-postgres:end
