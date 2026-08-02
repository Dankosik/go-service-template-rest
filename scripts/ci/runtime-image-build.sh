#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
IMAGE="${1:-service:ci}"

if [[ "$#" -gt 1 ]]; then
	echo "usage: $0 [image]" >&2
	exit 2
fi

build_image() {
	local context="$1"

	docker build \
		--build-arg "APP_VERSION=${APP_VERSION:-dev}" \
		--build-arg "VCS_REF=${VCS_REF:-unknown}" \
		--build-arg "SOURCE_URL=${SOURCE_URL:-}" \
		-f "${context}/build/docker/Dockerfile" \
		-t "${IMAGE}" \
		"${context}"
}

# profile:messaging-nats-jetstream:start
verify_worker_image() {
	local output
	if output="$(docker run --rm --read-only --network none --entrypoint /worker "${IMAGE}" 2>&1)"; then
		echo "runtime image worker exited successfully without a registered feature handler" >&2
		exit 1
	fi
	if ! grep -Fq 'worker feature handler builder is not registered' <<<"${output}"; then
		echo "runtime image worker did not execute the expected fail-closed binary" >&2
		echo "${output}" >&2
		exit 1
	fi
}
# profile:messaging-nats-jetstream:end

# Generated services no longer own profile sources, so their checkout is
# already the exact production source and needs no fixture.
if [[ ! -d "${ROOT_DIR}/scripts/profiles" ]]; then
	build_image "${ROOT_DIR}"
	# profile:messaging-nats-jetstream:start
	verify_worker_image
	# profile:messaging-nats-jetstream:end
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
	"GRPC=enabled"
	"AUTHN=none"
	"OUTBOUND_HTTP=bounded"
	"REFERENCE_EXAMPLE=remove"
)
# profile:messaging-nats-jetstream:start
profile_environment+=("MESSAGING=nats-jetstream")
# profile:messaging-nats-jetstream:end
(
	cd "${CHECKOUT}"
	env "${profile_environment[@]}" bash ./scripts/init-module.sh github.com/acme/runtime-image-proof
)

grep -Fqx 'database = "postgres"' "${CHECKOUT}/template.lock"
grep -Fqx 'grpc = "enabled"' "${CHECKOUT}/template.lock"
grep -Fqx 'authn = "none"' "${CHECKOUT}/template.lock"
grep -Fqx 'outbound_http = "bounded"' "${CHECKOUT}/template.lock"
# profile:messaging-nats-jetstream:start
grep -Fqx 'messaging = "nats-jetstream"' "${CHECKOUT}/template.lock"
# profile:messaging-nats-jetstream:end
[[ ! -d "${CHECKOUT}/internal/infra/oidcjwt" ]]

build_image "${CHECKOUT}"
# profile:messaging-nats-jetstream:start
verify_worker_image
# profile:messaging-nats-jetstream:end
