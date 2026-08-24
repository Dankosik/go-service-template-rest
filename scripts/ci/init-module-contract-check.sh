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

verify_authn_profile() {
	local authn="$1"
	local profile_checkout="${tmp}/service-${authn}"
	git clone -q "${checkout}" "${profile_checkout}"
	git -C "${profile_checkout}" remote set-url origin git@github.com:acme/service.git

	(
		cd "${profile_checkout}"
		CODEOWNER=@acme/platform DATABASE=none GRPC=none AUTHN="${authn}" bash ./scripts/init-module.sh
		grep -Fxq 'module github.com/acme/service' go.mod
		grep -Fxq 'database = "none"' template.lock
		grep -Fxq 'grpc = "none"' template.lock
		grep -Fxq "authn = \"${authn}\"" template.lock
		grep -Fxq 'agent_harness = "core"' template.lock
		test ! -e scripts/profiles
		test ! -d .claude
		if grep -R -E --exclude=init-module-contract-check.sh 'profile:[a-z0-9-]+:(start|end)' README.md Makefile api build cmd docs env internal .github scripts/ci scripts/dev 2>/dev/null; then
			echo "initializer contract: unresolved profile marker for AUTHN=${authn}" >&2
			exit 1
		fi

		dependencies="$(go list -deps ./cmd/service)"
		case "${authn}" in
		none)
			test ! -d internal/infra/bearerauthn
			test ! -d internal/infra/oidcjwt
			test ! -d internal/infra/oauthintrospection
			if grep -E 'internal/infra/(bearerauthn|oidcjwt|oauthintrospection)|MicahParks/(keyfunc|jwkset)|golang-jwt/jwt' <<<"${dependencies}"; then
				echo "AUTHN=none retained authentication dependencies" >&2
				exit 1
			fi
			;;
		oidc-jwt)
			test -d internal/infra/bearerauthn
			test -d internal/infra/oidcjwt
			test ! -d internal/infra/oauthintrospection
			grep -Fq 'internal/infra/bearerauthn' <<<"${dependencies}"
			grep -Fq 'internal/infra/oidcjwt' <<<"${dependencies}"
			;;
		oidc-introspection)
			test -d internal/infra/bearerauthn
			test -d internal/infra/oauthintrospection
			test ! -d internal/infra/oidcjwt
			grep -Fq 'internal/infra/bearerauthn' <<<"${dependencies}"
			grep -Fq 'internal/infra/oauthintrospection' <<<"${dependencies}"
			if grep -E 'MicahParks/(keyfunc|jwkset)|golang-jwt/jwt' <<<"${dependencies}"; then
				echo "AUTHN=oidc-introspection retained JWT dependencies" >&2
				exit 1
			fi
			;;
		esac

		go test -vet=off -run '^$' ./...
		GOFLAGS='' go mod tidy -diff
		GOFLAGS='' go -C tools mod tidy -diff
		git add -A
		git -c user.name=init-check -c user.email=init-check@example.invalid commit -qm generated
		make openapi-drift-check sqlc-check
		CODEOWNER=@acme/platform DATABASE=none GRPC=none AUTHN="${authn}" bash ./scripts/init-module.sh >/dev/null
		git diff --quiet
	)
}

verify_outbox_profile() {
	local profile_checkout="${tmp}/service-outbox"
	git clone -q "${checkout}" "${profile_checkout}"
	git -C "${profile_checkout}" remote set-url origin git@github.com:acme/service.git

	(
		cd "${profile_checkout}"
		CODEOWNER=@acme/platform DATABASE=postgres OUTBOX=postgres MESSAGING=nats-jetstream \
			bash ./scripts/init-module.sh
		test ! -e migrations/000001_postgres_outbox.sql
		test -e migrations/000008_river.sql
		grep -Fq 'outbox = "postgres"' template.lock
		grep -Fq 'messaging = "nats-jetstream"' template.lock
		if grep -R -E 'type Outbox(Event|CommitReceipt|OrderingHead|Redrife)' internal/infra/postgres/sqlcgen 2>/dev/null; then
			echo "initializer contract: fresh outbox retained legacy generated models" >&2
			exit 1
		fi
		go test -vet=off -run '^$' ./cmd/outbox-relay/... ./internal/infra/postgresoutbox ./internal/infra/natsjs
	)
}

verify_authn_profile none
verify_authn_profile oidc-jwt
verify_authn_profile oidc-introspection
verify_outbox_profile

echo "initializer contract passed"
