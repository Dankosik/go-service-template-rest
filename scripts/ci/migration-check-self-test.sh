#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CHECK="${ROOT_DIR}/scripts/ci/migration-history-check.sh"
SOURCE_CHECK="${ROOT_DIR}/scripts/ci/migration-source-check.sh"
fixture="$(mktemp -d)"
trap 'rm -rf "${fixture}"' EXIT INT TERM

git -C "${fixture}" init -q
git -C "${fixture}" config user.email migration-check@example.invalid
git -C "${fixture}" config user.name migration-check
mkdir -p "${fixture}/migrations"
printf '%s\n' '-- +goose Up' 'SELECT 1;' '-- +goose Down' 'SELECT 1;' \
	>"${fixture}/migrations/000001_create.sql"
git -C "${fixture}" add migrations
git -C "${fixture}" commit -qm base
base="$(git -C "${fixture}" rev-parse HEAD)"

printf '%s\n' '-- +goose Up' 'SELECT 2;' '-- +goose Down' 'SELECT 1;' \
	>"${fixture}/migrations/000001_create.sql"
if MIGRATION_REPO_ROOT="${fixture}" MIGRATION_HISTORY_MODE=worktree bash "${CHECK}" >/dev/null 2>&1; then
	echo "migration history self-test: modified migration passed" >&2
	exit 1
fi
git -C "${fixture}" restore migrations/000001_create.sql

printf '%s\n' '-- +goose Up' 'SELECT 2;' '-- +goose Down' 'SELECT 2;' \
	>"${fixture}/migrations/000002_add.sql"
MIGRATION_REPO_ROOT="${fixture}" MIGRATION_HISTORY_MODE=worktree bash "${CHECK}" >/dev/null
git -C "${fixture}" add migrations
git -C "${fixture}" commit -qm addition
head="$(git -C "${fixture}" rev-parse HEAD)"
MIGRATION_REPO_ROOT="${fixture}" MIGRATION_HISTORY_MODE=exact-base \
	BASE_REF="${base}" HEAD_REF="${head}" bash "${CHECK}" >/dev/null

git -C "${fixture}" mv migrations/000001_create.sql migrations/000003_rewritten.sql
git -C "${fixture}" commit -qam rewrite
rewritten="$(git -C "${fixture}" rev-parse HEAD)"
if MIGRATION_REPO_ROOT="${fixture}" MIGRATION_HISTORY_MODE=exact-base \
	BASE_REF="${head}" HEAD_REF="${rewritten}" bash "${CHECK}" >/dev/null 2>&1; then
	echo "migration history self-test: renamed migration passed" >&2
	exit 1
fi

MIGRATION_REPO_ROOT="${fixture}" MIGRATION_HISTORY_MODE=exact-base \
	BASE_REF=0000000000000000000000000000000000000000 \
	HEAD_REF="${base}" bash "${CHECK}" >/dev/null

source_fixture="${fixture}/source-fixture"
mkdir -p "${source_fixture}/migrations"
printf '%s\n' '-- +goose Down' 'SELECT 1;' '-- +goose Up' 'SELECT 1;' \
	>"${source_fixture}/migrations/000001_misordered.sql"
if MIGRATION_REPO_ROOT="${source_fixture}" bash "${SOURCE_CHECK}" >/dev/null 2>&1; then
	echo "migration source self-test: misordered directions passed" >&2
	exit 1
fi
printf '%s\n' '-- +goose NO TRANSACTION' '-- +goose Up' 'SELECT 1;' '-- +goose Down' 'SELECT 1;' \
	>"${source_fixture}/migrations/000001_misordered.sql"
if MIGRATION_REPO_ROOT="${source_fixture}" bash "${SOURCE_CHECK}" >/dev/null 2>&1; then
	echo "migration source self-test: non-transactional migration passed" >&2
	exit 1
fi
rm "${source_fixture}/migrations/000001_misordered.sql"
ln -s ../outside.sql "${source_fixture}/migrations/000001_symlink.sql"
if MIGRATION_REPO_ROOT="${source_fixture}" bash "${SOURCE_CHECK}" >/dev/null 2>&1; then
	echo "migration source self-test: symbolic link passed" >&2
	exit 1
fi

echo "migration history self-test passed"
