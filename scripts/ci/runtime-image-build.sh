#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
IMAGE="${1:-service:ci}"
FIXTURE="${2:-}"
if [[ -z "${FIXTURE}" ]]; then
	if [[ "${IMAGE}" == *oidc-introspection* ]]; then
		FIXTURE="oidc-introspection"
	else
		FIXTURE="health-only"
	fi
fi

if [[ "$#" -gt 2 ]]; then
	echo "usage: $0 [image] [health-only|postgres-http-idempotency-active|oidc-introspection]" >&2
	exit 2
fi

case "${FIXTURE}" in
	health-only | postgres-http-idempotency-active | oidc-introspection) ;;
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
	if output="$(docker run --rm --read-only --network none --entrypoint /worker "${IMAGE}" 2>&1)"; then
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

install_introspection_protected_probe() {
	python3 - <<'PY'
from pathlib import Path
path = Path("api/openapi/service.yaml")
text = path.read_text()
needle = "paths:\n  /health/live:"
insert = """paths:
  /secure-probe:
    get:
      operationId: secureProbe
      x-security-decision:
        exposure: protected
        rationale: disposable runtime-image introspection fixture
      responses:
        "204":
          description: accepted
        "400":
          $ref: "#/components/responses/BadRequest"
        "401":
          description: unauthorized
        "403":
          description: forbidden
        "413":
          $ref: "#/components/responses/RequestEntityTooLarge"
        "431":
          description: request header fields too large
        "500":
          $ref: "#/components/responses/InternalServerError"
        "503":
          description: service unavailable
        "504":
          description: gateway timeout
  /health/live:"""
if needle not in text:
    raise SystemExit("openapi paths block not found")
path.write_text(text.replace(needle, insert, 1))
PY
	cat >internal/infra/http/secure_probe_fixture.go <<'EOF'
package httpx

import (
	"context"

	"github.com/acme/runtime-image-proof/internal/openapi"
)

type introspectionFixtureAPI struct{}

func (introspectionFixtureAPI) HealthLive(context.Context, openapi.HealthLiveRequestObject) (openapi.HealthLiveResponseObject, error) {
	return openapi.HealthLive200TextResponse("ok"), nil
}

func (introspectionFixtureAPI) HealthReady(context.Context, openapi.HealthReadyRequestObject) (openapi.HealthReadyResponseObject, error) {
	return openapi.HealthReady200TextResponse("ok"), nil
}

func (introspectionFixtureAPI) SecureProbe(context.Context, openapi.SecureProbeRequestObject) (openapi.SecureProbeResponseObject, error) {
	return openapi.SecureProbe204Response{}, nil
}
EOF
	python3 - <<'PY'
from pathlib import Path
path = Path("cmd/service/internal/bootstrap/run.go")
text = path.read_text()
old = """			Handlers: httpx.Handlers{
				Health: healthSvc,"""
new = """			Handlers: httpx.Handlers{
				API:    httpx.IntrospectionFixtureAPI(),
				Health: healthSvc,"""
if old not in text:
    raise SystemExit("http handlers block not found")
path.write_text(text.replace(old, new, 1))
PY
	cat >>internal/infra/http/secure_probe_fixture.go <<'EOF'

func IntrospectionFixtureAPI() openapi.StrictServerInterface {
	return introspectionFixtureAPI{}
}
EOF
}

verify_introspection_runtime() {
	local cid port logs secret
	secret="introspection-image-secret-canary"
	port="$(python3 -c 'import socket; s=socket.socket(); s.bind(("", 0)); print(s.getsockname()[1]); s.close()')"
	cid="$(docker run -d --rm \
		-p "127.0.0.1:${port}:8080" \
		-e APP__APP__ENV=local \
		-e APP__HTTP__ADDR=:8080 \
		-e APP__AUTHN__ISSUER=https://issuer.example.invalid \
		-e APP__AUTHN__AUDIENCE=https://api.example.invalid \
		-e APP__AUTHN__INTROSPECTION_ENDPOINT=https://idp.example.invalid/oauth/introspect \
		-e APP__AUTHN__INTROSPECTION_TARGET_CLASS=external-https \
		-e APP__AUTHN__INTROSPECTION_CLIENT_ID=runtime-image-client \
		-e APP__AUTHN__INTROSPECTION_CLIENT_SECRET="${secret}" \
		"${IMAGE}")"
	cleanup_cid() {
		docker logs "${cid}" >"${TEMP_ROOT}/introspection.log" 2>&1 || true
		docker stop "${cid}" >/dev/null 2>&1 || true
	}
	trap 'cleanup_cid; cleanup' EXIT INT TERM
	local ready=0
	local i
	for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do
		if curl -fsS "http://127.0.0.1:${port}/health/live" >/dev/null && curl -fsS "http://127.0.0.1:${port}/health/ready" >/dev/null; then
			ready=1
			break
		fi
		sleep 1
	done
	if [[ "${ready}" -ne 1 ]]; then
		echo "introspection runtime did not become live/ready" >&2
		docker logs "${cid}" >&2 || true
		exit 1
	fi
	local protected
	protected="$(curl -sS -o "${TEMP_ROOT}/protected.body" -w '%{http_code}' \
		-H 'Authorization: Bearer runtime-image-token-canary' \
		"http://127.0.0.1:${port}/secure-probe")"
	if [[ "${protected}" != "503" ]]; then
		echo "protected request status = ${protected}, want 503" >&2
		cat "${TEMP_ROOT}/protected.body" >&2 || true
		exit 1
	fi
	if grep -F -e "${secret}" -e 'runtime-image-token-canary' -e 'idp.example.invalid' "${TEMP_ROOT}/protected.body"; then
		echo "protected response disclosed a canary" >&2
		exit 1
	fi
	if ! curl -fsS "http://127.0.0.1:${port}/health/ready" >/dev/null; then
		echo "readiness flipped after provider failure" >&2
		exit 1
	fi
	cleanup_cid
	trap cleanup EXIT INT TERM
	if grep -F -e "${secret}" -e 'runtime-image-token-canary' "${TEMP_ROOT}/introspection.log"; then
		echo "runtime logs disclosed a canary" >&2
		exit 1
	fi
}

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
	# profile:authn-oidc-introspection:start
	# The oidc-introspection fixture overrides this after the array is built.
	# profile:authn-oidc-introspection:end
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
if [[ "${FIXTURE}" == "oidc-introspection" ]]; then
	profile_environment=("CODEOWNER=@acme/platform" "DATABASE=none" "HTTP_IDEMPOTENCY=none" "JOBS=none" "GRPC=none" "AUTHN=oidc-introspection" "OBJECT_STORAGE=none" "REFERENCE_EXAMPLE=remove")
fi
(
	cd "${CHECKOUT}"
	env "${profile_environment[@]}" bash ./scripts/init-module.sh github.com/acme/runtime-image-proof
	if [[ "${FIXTURE}" == "postgres-http-idempotency-active" ]]; then
		git apply --recount "${ROOT_DIR}/scripts/ci/fixtures/postgres-http-idempotency-active.patch"
		make openapi-generate
		go mod tidy
	fi
	if [[ "${FIXTURE}" == "oidc-introspection" ]]; then
		install_introspection_protected_probe
		make openapi-generate
		go mod tidy
	fi
)

if [[ "${FIXTURE}" == "oidc-introspection" ]]; then
	grep -Fqx 'authn = "oidc-introspection"' "${CHECKOUT}/template.lock"
	[[ -d "${CHECKOUT}/internal/infra/bearerauthn" ]]
	[[ -d "${CHECKOUT}/internal/infra/oauthintrospection" ]]
	[[ ! -d "${CHECKOUT}/internal/infra/oidcjwt" ]]
	if grep -R -E -e 'profile:authn-(oidc-jwt):' "${CHECKOUT}/internal" "${CHECKOUT}/docs" "${CHECKOUT}/env" >/dev/null 2>&1; then
		echo "introspection fixture retained JWT profile text" >&2
		exit 1
	fi
else
	grep -Fqx 'database = "postgres"' "${CHECKOUT}/template.lock"
	grep -Fqx 'grpc = "enabled"' "${CHECKOUT}/template.lock"
	if [[ "${FIXTURE}" == "postgres-http-idempotency-active" ]]; then
		grep -Fqx 'authn = "oidc-jwt"' "${CHECKOUT}/template.lock"
	else
		grep -Fqx 'authn = "none"' "${CHECKOUT}/template.lock"
	fi
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
		[[ ! -d "${CHECKOUT}/internal/infra/bearerauthn" ]]
		[[ ! -d "${CHECKOUT}/internal/infra/oidcjwt" ]]
		[[ ! -d "${CHECKOUT}/internal/infra/oauthintrospection" ]]
	fi
fi

build_image "${CHECKOUT}"
if [[ "${FIXTURE}" == "oidc-introspection" ]]; then
	verify_introspection_runtime
else
	# profile:messaging-nats-jetstream:start
	verify_worker_image
	# profile:messaging-nats-jetstream:end
	# profile:outbox-postgres:start
	verify_outbox_relay_image
	# profile:outbox-postgres:end
	# profile:jobs-postgres:start
	verify_jobs_worker_image
	# profile:jobs-postgres:end
fi
