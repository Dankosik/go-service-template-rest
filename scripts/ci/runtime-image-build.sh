#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
IMAGE="${1:-service:ci}"
context="${ROOT_DIR}"
tmp=""

cleanup() {
	[[ -z "${tmp}" ]] || rm -rf -- "${tmp}"
}
trap cleanup EXIT

if [[ -d "${ROOT_DIR}/scripts/profiles" ]]; then
	tmp="$(mktemp -d -t runtime-image.XXXXXX)"
	context="${tmp}/service"
	mkdir -p "${context}"
	(
		cd "${ROOT_DIR}"
		git ls-files --cached --others --exclude-standard |
			awk '!/^(\.artifacts|\.cache|bin)\// && $0 != "jobs-worker" && $0 != "s3.test"' |
			while IFS= read -r file; do
				[[ -f "${file}" || -L "${file}" ]] && printf '%s\n' "${file}"
			done |
			tar -cf - -T -
	) | tar -xf - -C "${context}"
	git -C "${context}" init -q
	git -C "${context}" remote add origin git@github.com:acme/runtime-image.git
	git -C "${context}" add -A
	git -C "${context}" -c user.name=runtime-image -c user.email=runtime-image@example.invalid commit -qm baseline
	(cd "${context}" && CODEOWNER=@acme/platform DATABASE=postgres bash ./scripts/init-module.sh)
fi

fingerprint="$({
	cd "${context}"
	find . -type f -not -path './.git/*' -exec shasum -a 256 {} +
} | LC_ALL=C sort | shasum -a 256 | awk '{print $1}')"

docker build \
	--build-arg "APP_VERSION=${APP_VERSION:-dev}" \
	--build-arg "VCS_REF=${VCS_REF:-unknown}" \
	--build-arg "SOURCE_URL=${SOURCE_URL:-}" \
	--build-arg "WORKTREE_FINGERPRINT=${fingerprint}" \
	-f "${context}/build/docker/Dockerfile" \
	-t "${IMAGE}" \
	"${context}"
