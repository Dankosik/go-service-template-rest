#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "${ROOT_DIR}"

requested_image=${1:-}
expected_version=${2:-}
service_name=${3:-service}
go_command=${GO:-go}
migration_dir=${MIGRATION_DIR:-migrations}
migration_engine=${MIGRATION_ENGINE:-goose}
project="service-migration-$(date +%s)-$$"

case "${migration_engine}" in
	goose|golang-migrate) ;;
	*) echo "unsupported migration engine: ${migration_engine}" >&2; exit 2 ;;
esac

if [[ ! -d "${migration_dir}" ]]; then
	echo "configured migration directory does not exist: ${migration_dir}" >&2
	exit 2
fi

compose() {
	POSTGRES_PORT=0 docker compose -p "${project}" -f env/docker-compose.yml "$@"
}

cleanup() {
	compose down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

compose up -d --wait postgres
address=$(compose port postgres 5432)
port=${address##*:}
[[ -n ${port} ]] || {
	echo "failed to resolve rehearsal Postgres port" >&2
	exit 1
}
dsn="postgres://app:app@localhost:${port}/app?sslmode=disable"

case "${migration_engine}" in
	goose) "${go_command}" tool -modfile=tools/go.mod goose -dir "${migration_dir}" validate ;;
	golang-migrate)
		"${go_command}" run -mod=readonly -tags=postgres github.com/golang-migrate/migrate/v4/cmd/migrate \
			-path "${migration_dir}" -database "${dsn}" up
		"${go_command}" run -mod=readonly -tags=postgres github.com/golang-migrate/migrate/v4/cmd/migrate \
			-path "${migration_dir}" -database "${dsn}" down 1
		"${go_command}" run -mod=readonly -tags=postgres github.com/golang-migrate/migrate/v4/cmd/migrate \
			-path "${migration_dir}" -database "${dsn}" up 1
		;;
esac

rehearsal=$(mktemp)
trap 'rm -f "${rehearsal}"; cleanup' EXIT INT TERM
PGTEST_POSTGRES_DSN="${dsn}" REQUIRE_DOCKER=1 "${go_command}" test -vet=off -count=1 -tags=integration ./test \
	-run '^TestPostgresMigrateUpAppliesAndReplaysMigrations$' -json | tee "${rehearsal}"
grep -Eq '"Action":"pass".*"Test":"TestPostgresMigrateUpAppliesAndReplaysMigrations"' "${rehearsal}"

image=${requested_image:-${service_name}:migration}
if [[ -z ${requested_image} ]]; then
	make runtime-image-build RUNTIME_IMAGE="${image}"
fi

docker run --rm --network "${project}_default" \
	-e APP__POSTGRES__ENABLED=true \
	-e APP__POSTGRES__DSN="postgres://app:app@postgres:5432/app?sslmode=disable" \
	--entrypoint /migrate "${image}"

RUNTIME_IMAGE_NETWORK="${project}_default" \
	RUNTIME_IMAGE_POSTGRES_DSN="postgres://app:app@postgres:5432/app?sslmode=disable" \
	bash ./scripts/ci/runtime-image-check.sh "${image}" "${expected_version}"
