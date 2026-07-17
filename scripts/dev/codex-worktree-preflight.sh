#!/usr/bin/env bash
set -euo pipefail

# Conservative compatibility guards, not an App product contract.
TRACKED_PATCH_LIMIT_BYTES=$((32 * 1024 * 1024))
UNTRACKED_TOTAL_LIMIT_BYTES=$((64 * 1024 * 1024))

block() {
  printf 'BLOCKED: %s\n' "$*" >&2
  exit 1
}

usage() {
  echo "usage: $0 <selected-git-top-level>" >&2
  exit 2
}

file_size() {
  local path="$1" bytes
  if bytes="$(stat -f '%z' "${path}" 2>/dev/null)" && [[ "${bytes}" =~ ^[0-9]+$ ]]; then
    printf '%s\n' "${bytes}"
    return 0
  fi
  if bytes="$(stat -c '%s' "${path}" 2>/dev/null)" && [[ "${bytes}" =~ ^[0-9]+$ ]]; then
    printf '%s\n' "${bytes}"
    return 0
  fi
  return 1
}

report_tracked_inputs() {
  while IFS= read -r -d '' path; do
    printf '  tracked input: %q\n' "${path}" >&2
  done < <(git -C "${git_root}" diff --no-ext-diff --name-only -z HEAD --)
}

[[ $# -eq 1 ]] || usage
selected_root="$(cd -P "$1" 2>/dev/null && pwd -P)" \
  || block "unavailable proof: selected path is not accessible: $1"
git_root="$(git -C "${selected_root}" rev-parse --show-toplevel 2>/dev/null)" \
  || block "unavailable proof: selected path is not a Git worktree: ${selected_root}"
git_root="$(cd -P "${git_root}" && pwd -P)"
[[ "${selected_root}" == "${git_root}" ]] \
  || block "unavailable proof: selected path is not its Git top level: selected=${selected_root} git-top-level=${git_root}"
git -C "${git_root}" rev-parse --verify HEAD^{commit} >/dev/null 2>&1 \
  || block "unavailable proof: selected Git top level has no HEAD commit: ${git_root}"

tracked_patch_bytes="$(git -C "${git_root}" diff --no-ext-diff --binary HEAD -- | wc -c | tr -d '[:space:]')" \
  || block "unavailable proof: could not measure tracked patch at ${git_root}"
if (( tracked_patch_bytes >= TRACKED_PATCH_LIMIT_BYTES )); then
  printf 'BLOCKED: tracked patch is %s bytes (guard: %s bytes)\n' \
    "${tracked_patch_bytes}" "${TRACKED_PATCH_LIMIT_BYTES}" >&2
  report_tracked_inputs
  exit 1
fi

untracked_total_bytes=0
untracked_inputs=()
while IFS= read -r -d '' path; do
  bytes="$(file_size "${git_root}/${path}")" \
    || block "unavailable proof: could not measure nonignored untracked input: ${path}"
  untracked_total_bytes=$((untracked_total_bytes + bytes))
  untracked_inputs+=("${path}" "${bytes}")
  transfer_total_bytes=$((tracked_patch_bytes + untracked_total_bytes))
  if (( transfer_total_bytes >= UNTRACKED_TOTAL_LIMIT_BYTES )); then
    printf 'BLOCKED: working-tree transfer inputs total at least %s bytes (tracked patch %s bytes; nonignored untracked inputs %s bytes; guard: %s bytes)\n' \
      "${transfer_total_bytes}" "${tracked_patch_bytes}" "${untracked_total_bytes}" "${UNTRACKED_TOTAL_LIMIT_BYTES}" >&2
    report_tracked_inputs
    for ((i = 0; i < ${#untracked_inputs[@]}; i += 2)); do
      printf '  untracked input: %q (%s bytes)\n' \
        "${untracked_inputs[i]}" "${untracked_inputs[i + 1]}" >&2
    done
    exit 1
  fi
done < <(git -C "${git_root}" ls-files --others --exclude-standard -z)

printf 'PASS: selected Git top level %s; tracked patch %s bytes; nonignored untracked inputs %s bytes; total transfer inputs %s bytes\n' \
  "${git_root}" "${tracked_patch_bytes}" "${untracked_total_bytes}" "$((tracked_patch_bytes + untracked_total_bytes))"
