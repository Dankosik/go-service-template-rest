#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "usage: $0 metadata <mode> <candidate-sha> <repository> <default-branch> <ref-name> <ref> <run-id> <run-attempt> | promote <repo> <candidate-image> <digest> <tags> <summary> | self-test" >&2
	exit 2
}

promote() {
	local image_repo="$1" primary_image="$2" digest="$3" promote_tags="$4" summary="$5" tag
	local -a tags promoted

	read -ra tags <<<"${promote_tags}"
	[[ ${#tags[@]} -gt 0 ]] || { echo "publish-image: at least one promotion tag is required" >&2; return 1; }
	for tag in "${tags[@]}"; do
		docker tag "${primary_image}" "${image_repo}:${tag}"
		if ! docker push "${image_repo}:${tag}" ||
			[[ "$(docker buildx imagetools inspect --format '{{.Manifest.Digest}}' "${image_repo}:${tag}")" != "${digest}" ]]; then
			{
				echo "### Partial image promotion"
				echo
				echo "Digest: ${image_repo}@${digest}"
				echo "Migration history was already advanced."
				echo "Promoted tags before failure: ${promoted[*]:-none}"
				echo "Failed tag: ${tag}"
			} >>"${summary}" || true
			return 1
		fi
		promoted+=("${tag}")
	done
}

metadata() {
	local mode="$1" candidate_sha="$2" repository="$3" default_branch="$4"
	local ref_name="$5" ref="$6" run_id="$7" run_attempt="$8"
	local short_sha version promote_tags artifact_name identity image_repo

	[[ "${candidate_sha}" =~ ^[0-9a-f]{40}$ ]] || { echo "publish-image: candidate SHA must be 40 lowercase hex characters" >&2; return 1; }
	[[ "${repository}" == */* && -n "${default_branch}" ]] || { echo "publish-image: repository and default branch are required" >&2; return 1; }
	[[ "${run_id}" =~ ^[1-9][0-9]*$ && "${run_attempt}" =~ ^[1-9][0-9]*$ ]] || { echo "publish-image: run identity must be positive integers" >&2; return 1; }

	short_sha="${candidate_sha:0:12}"
	image_repo="ghcr.io/$(printf '%s' "${repository}" | tr '[:upper:]' '[:lower:]')"
	case "${mode}" in
	main)
		version="sha-${short_sha}"
		promote_tags="sha-${short_sha} main"
		artifact_name="sbom-main-${short_sha}-${run_id}-${run_attempt}"
		identity="https://github.com/${repository}/.github/workflows/cd.yml@refs/heads/${default_branch}"
		;;
	release)
		[[ "${ref}" == refs/tags/v* && "${ref_name}" =~ ^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$ ]] || {
			echo "publish-image: release ref must be a Docker-compatible v* tag" >&2
			return 1
		}
		version="${ref_name}"
		promote_tags="${ref_name} latest"
		artifact_name="sbom-${ref_name}-${run_id}-${run_attempt}"
		identity="https://github.com/${repository}/.github/workflows/cd.yml@${ref}"
		;;
	*)
		echo "publish-image: unsupported mode '${mode}'" >&2
		return 1
		;;
	esac

	printf 'image_repo=%s\n' "${image_repo}"
	printf 'primary_image=%s:build-%s-%s\n' "${image_repo}" "${run_id}" "${run_attempt}"
	printf 'version=%s\n' "${version}"
	printf 'promote_tags=%s\n' "${promote_tags}"
	printf 'artifact_name=%s\n' "${artifact_name}"
	printf 'identity=%s\n' "${identity}"
}

self_test() {
	local sha="0123456789abcdef0123456789abcdef01234567"
	local digest="sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	local got expected summary

	got="$(metadata main "${sha}" Owner/Service main '' '' 42 3)"
	expected=$'image_repo=ghcr.io/owner/service\nprimary_image=ghcr.io/owner/service:build-42-3\nversion=sha-0123456789ab\npromote_tags=sha-0123456789ab main\nartifact_name=sbom-main-0123456789ab-42-3\nidentity=https://github.com/Owner/Service/.github/workflows/cd.yml@refs/heads/main'
	[[ "${got}" == "${expected}" ]] || { echo "publish-image metadata self-test: main mismatch" >&2; return 1; }

	got="$(metadata release "${sha}" Owner/Service main v1.2.3 refs/tags/v1.2.3 42 4)"
	expected=$'image_repo=ghcr.io/owner/service\nprimary_image=ghcr.io/owner/service:build-42-4\nversion=v1.2.3\npromote_tags=v1.2.3 latest\nartifact_name=sbom-v1.2.3-42-4\nidentity=https://github.com/Owner/Service/.github/workflows/cd.yml@refs/tags/v1.2.3'
	[[ "${got}" == "${expected}" ]] || { echo "publish-image metadata self-test: release mismatch" >&2; return 1; }

	if metadata main invalid Owner/Service main '' '' 42 1 >/dev/null 2>&1; then
		echo "publish-image metadata self-test: invalid SHA passed" >&2
		return 1
	fi
	if metadata release "${sha}" Owner/Service main v1/bad refs/tags/v1/bad 42 1 >/dev/null 2>&1; then
		echo "publish-image metadata self-test: invalid release tag passed" >&2
		return 1
	fi

	summary="$(mktemp -t publish-image-summary.XXXXXX)"
	PUBLISH_IMAGE_TEST_SUMMARY="${summary}"
	trap 'rm -f -- "${PUBLISH_IMAGE_TEST_SUMMARY}"' EXIT
	PUBLISH_IMAGE_TEST_DIGEST="${digest}"
	docker() {
		case "$1" in
		tag) return 0 ;;
		push) [[ "$2" != "${PUBLISH_IMAGE_TEST_FAIL_TAG:-}" ]] ;;
		buildx) printf '%s\n' "${PUBLISH_IMAGE_TEST_DIGEST}" ;;
		*) return 1 ;;
		esac
	}
	promote ghcr.io/owner/service ghcr.io/owner/service:build-42-1 "${digest}" 'sha-0123456789ab main' "${summary}"
	PUBLISH_IMAGE_TEST_FAIL_TAG=ghcr.io/owner/service:main
	if promote ghcr.io/owner/service ghcr.io/owner/service:build-42-1 "${digest}" 'sha-0123456789ab main' "${summary}"; then
		echo "publish-image metadata self-test: partial promotion passed" >&2
		return 1
	fi
	grep -Fq 'Promoted tags before failure: sha-0123456789ab' "${summary}" || {
		echo "publish-image metadata self-test: partial receipt is incomplete" >&2
		return 1
	}
	: >"${summary}"
	PUBLISH_IMAGE_TEST_FAIL_TAG=ghcr.io/owner/service:sha-0123456789ab
	if promote ghcr.io/owner/service ghcr.io/owner/service:build-42-1 "${digest}" 'sha-0123456789ab main' "${summary}"; then
		echo "publish-image metadata self-test: first-tag failure passed" >&2
		return 1
	fi
	grep -Fq 'Promoted tags before failure: none' "${summary}" || {
		echo "publish-image metadata self-test: empty partial receipt is incomplete" >&2
		return 1
	}
	echo "publish-image metadata self-test passed"
}

case "${1:-}" in
metadata)
	[[ $# -eq 9 ]] || usage
	metadata "$2" "$3" "$4" "$5" "$6" "$7" "$8" "$9"
	;;
promote)
	[[ $# -eq 6 ]] || usage
	promote "$2" "$3" "$4" "$5" "$6"
	;;
self-test)
	[[ $# -eq 1 ]] || usage
	self_test
	;;
*) usage ;;
esac
