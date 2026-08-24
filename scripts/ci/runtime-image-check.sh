#!/usr/bin/env bash
set -euo pipefail

image=${1:?runtime image is required}
expected_version=${2:-}
container="service-runtime-check-$$"

cleanup() {
	docker rm -f "${container}" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

command -v curl >/dev/null 2>&1 || {
	echo "curl is required for runtime image validation" >&2
	exit 2
}

docker run -d --name "${container}" \
	-p 127.0.0.1::8080 \
	--read-only \
	--cap-drop=ALL \
	--security-opt=no-new-privileges \
	"${image}" >/dev/null

address=$(docker port "${container}" 8080/tcp | head -n 1)
port=${address##*:}
[[ -n "${port}" ]] || {
	echo "failed to resolve runtime service port" >&2
	docker logs "${container}" >&2
	exit 1
}

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
