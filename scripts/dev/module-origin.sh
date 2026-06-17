#!/usr/bin/env bash

detect_module_from_origin() {
	local remote_url host path without_scheme

	remote_url="$(git config --get remote.origin.url 2>/dev/null || true)"
	if [[ -z "${remote_url}" ]]; then
		return 1
	fi

	case "${remote_url}" in
	git@*:* )
		host="${remote_url#git@}"
		host="${host%%:*}"
		path="${remote_url#*:}"
		;;
	ssh://git@*/*)
		without_scheme="${remote_url#ssh://git@}"
		host="${without_scheme%%/*}"
		path="${without_scheme#*/}"
		;;
	http://*|https://*)
		without_scheme="${remote_url#*://}"
		host="${without_scheme%%/*}"
		path="${without_scheme#*/}"
		;;
	*)
		return 1
		;;
	esac

	path="${path%.git}"
	path="${path#/}"
	path="${path%/}"
	host="${host%/}"
	host="${host%%:*}"

	if [[ -z "${host}" || -z "${path}" || "${path}" == "${remote_url}" ]]; then
		return 1
	fi

	printf '%s/%s\n' "${host}" "${path}"
}
