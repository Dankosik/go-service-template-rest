#!/usr/bin/env bash
set -euo pipefail

image=${1:?runtime image is required}
expected_version=${2:-}
container="service-runtime-check-$$"
project=''

compose() {
	POSTGRES_PORT=0 docker compose -p "${project}" -f env/docker-compose.yml "$@"
}

cleanup() {
	docker rm -f "${container}" >/dev/null 2>&1 || true
	[[ -z ${project} ]] || compose down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

command -v curl >/dev/null 2>&1 || {
	echo "curl is required for runtime image validation" >&2
	exit 2
}

if [[ -z ${RUNTIME_IMAGE_POSTGRES_DSN:-} && -f env/docker-compose.yml ]]; then
	project="service-runtime-check-$(date +%s)-$$"
	compose up -d --wait postgres
	RUNTIME_IMAGE_NETWORK="${project}_default"
	RUNTIME_IMAGE_POSTGRES_DSN='postgres://app:app@postgres:5432/app?sslmode=disable'
fi

docker_args=()
if [[ -n ${RUNTIME_IMAGE_NETWORK:-} ]]; then
	docker_args+=(--network "${RUNTIME_IMAGE_NETWORK}")
fi
if [[ -n ${RUNTIME_IMAGE_POSTGRES_DSN:-} ]]; then
	docker_args+=(
		-e APP__POSTGRES__ENABLED=true
		-e "APP__POSTGRES__DSN=${RUNTIME_IMAGE_POSTGRES_DSN}"
	)
fi
if [[ -n ${RUNTIME_IMAGE_EGRESS_ALLOWLIST:-} ]]; then
	docker_args+=(-e "NETWORK_EGRESS_ALLOWLIST=${RUNTIME_IMAGE_EGRESS_ALLOWLIST}")
fi

docker run -d --name "${container}" \
	-p 127.0.0.1::8080 \
	--read-only \
	--cap-drop=ALL \
	--security-opt=no-new-privileges \
	"${docker_args[@]}" \
	"${image}" >/dev/null

address=$(docker port "${container}" 8080/tcp 2>/dev/null | head -n 1 || true)
port=${address##*:}
if [[ -z ${port} ]]; then
	if [[ $(docker inspect -f '{{.State.Running}}' "${container}") != true ]]; then
		echo "runtime image exited before publishing its service port" >&2
	else
		echo "failed to resolve runtime service port" >&2
	fi
	docker logs "${container}" >&2
	exit 1
fi

ready=false
for _ in {1..45}; do
	if curl -fs --max-time 2 "http://127.0.0.1:${port}/health/ready" >/dev/null; then
		ready=true
		break
	fi
	[[ $(docker inspect -f '{{.State.Running}}' "${container}") == true ]] || break
	sleep 1
done
if [[ ${ready} != true ]]; then
	echo "runtime image did not become ready" >&2
	docker logs "${container}" >&2
	exit 1
fi

logs="$(docker logs "${container}" 2>&1)"
if [[ -n ${expected_version} ]] && ! grep -Fq "\"service.version\":\"${expected_version}\"" <<<"${logs}"; then
	echo "runtime image did not report expected version ${expected_version}" >&2
	printf '%s\n' "${logs}" >&2
	exit 1
fi

docker stop --time 45 "${container}" >/dev/null
exit_code=$(docker inspect -f '{{.State.ExitCode}}' "${container}")
[[ ${exit_code} == 0 ]] || {
	echo "runtime image exited with code ${exit_code} after SIGTERM" >&2
	docker logs "${container}" >&2
	exit 1
}
