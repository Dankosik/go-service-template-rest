#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if [[ ! -d "${ROOT_DIR}/scripts/profiles" ]]; then
	echo "initializer contract is template-only"
	exit 0
fi

tmp="$(mktemp -d -t init-module-check.XXXXXX)"
checkout="${tmp}/service"
trap 'rm -rf -- "${tmp}"' EXIT
mkdir -p "${checkout}"
git -C "${ROOT_DIR}" archive HEAD | tar -xf - -C "${checkout}"
git -C "${checkout}" init -q
git -C "${checkout}" remote add origin git@github.com:acme/service.git

patch="${tmp}/automation.patch"
git -C "${ROOT_DIR}" diff --binary HEAD -- \
	Makefile README.md CONTRIBUTING.md .github .gitleaks.toml scripts tools/go.mod tools/go.sum \
	buf.yaml buf.gen.yaml examples/grpc-reference-service/buf.gen.yaml \
	internal/openapi/doc.go examples/reference-service/internal/openapi/doc.go \
	docs/benchmarking.md docs/benchmarking docs/build-test-and-development-commands.md \
	docs/ci-cd-production-ready.md docs/template-sync.md docs/validation-routing.md \
	docs/validation template-owned.paths >"${patch}"
[[ ! -s "${patch}" ]] || git -C "${checkout}" apply "${patch}"
mkdir -p "${checkout}/scripts/ci" "${checkout}/.github/workflows"
cp "${ROOT_DIR}/scripts/ci/init-module-contract-check.sh" "${checkout}/scripts/ci/"
cp "${ROOT_DIR}/.github/workflows/integration.yml" "${checkout}/.github/workflows/"

# Keep unrelated profile candidates outside this automation oracle until their
# initializer selector is present in the accepted owner.
if ! grep -Fq 'INBOUND_WEBHOOKS' "${checkout}/scripts/init-module.sh"; then
	awk '
		/profile:inbound-webhooks-standard:start/ { skip = 1; next }
		/profile:inbound-webhooks-standard:end/ { skip = 0; next }
		!skip { print }
	' "${checkout}/Makefile" >"${checkout}/Makefile.next"
	mv "${checkout}/Makefile.next" "${checkout}/Makefile"
fi

git -C "${checkout}" add -A
git -C "${checkout}" -c user.name=init-check -c user.email=init-check@example.invalid commit -qm baseline

unchanged_failure() {
	if (cd "${checkout}" && "$@") >/dev/null 2>&1; then
		echo "initializer contract: expected failure: $*" >&2
		exit 1
	fi
	git -C "${checkout}" diff --quiet
	[[ -z "$(git -C "${checkout}" status --porcelain)" ]]
}

unchanged_failure env -u CODEOWNER bash ./scripts/init-module.sh
unchanged_failure env CODEOWNER=@acme/platform GRPC=custom bash ./scripts/init-module.sh
unchanged_failure env CODEOWNER=@acme/platform bash ./scripts/init-module.sh 'bad module'

(
	cd "${checkout}"
	CODEOWNER=@acme/platform DATABASE=none bash ./scripts/init-module.sh
	grep -Fxq 'module github.com/acme/service' go.mod
	grep -Fxq 'database = "none"' template.lock
	grep -Fxq 'agent_harness = "core"' template.lock
	test ! -e scripts/profiles
	test ! -e evals
	test ! -e scripts/ci/instruction-evals-check.sh
	test ! -d .claude
	if grep -R -E --exclude=init-module-contract-check.sh 'profile:[a-z0-9-]+:(start|end)' README.md Makefile api build cmd docs env internal .github scripts/ci scripts/dev 2>/dev/null; then
		echo "initializer contract: unresolved profile marker" >&2
		exit 1
	fi
	go test -vet=off -run '^$' ./...
	GOFLAGS='' go mod tidy -diff
	GOFLAGS='' go -C tools mod tidy -diff
	git add -A
	git -c user.name=init-check -c user.email=init-check@example.invalid commit -qm generated
	make openapi-drift-check sqlc-check
	CODEOWNER=@acme/platform DATABASE=none bash ./scripts/init-module.sh >/dev/null
	git diff --quiet
)

echo "initializer contract passed"
