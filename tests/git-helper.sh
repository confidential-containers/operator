#!/usr/bin/env bash
#
# Copyright Confidential Containers Contributors
#
# SPDX-License-Identifier: Apache-2.0
#

set -o errexit
set -o nounset
set -o pipefail

function install_jq() {
	if [ -n "$(command -v apt-get)" ]; then
		sudo apt-get update
		sudo apt-get install -y jq
	elif [ -n "$(command -v yum)" ]; then
		sudo yum install -y epel-release
		sudo yum install -y jq
	else
		>&2 echo "No supported package manager found"
		exit 1
	fi
}

function configure_github() {
	if [ ! command -v jq &> /dev/null ]; then
		echo "jq is not installed, installing it"
		install_jq
	fi
	USERNAME=$(jq -r '.pull_request.user.login' "$GITHUB_EVENT_PATH")
	EMAIL=$(jq -r '.pull_request.user.email' "$GITHUB_EVENT_PATH")
	# if the email is null, stuff with a dummy email
	if [ "${EMAIL}" == "null" ]; then
		EMAIL="dummy@email.address"
	fi
	echo "Adding user name ${USERNAME} and email ${EMAIL} to the local git repo"
	git config user.name "${USERNAME}"
	git config user.email "${EMAIL}"
}

function rebase_atop_of_the_latest_target_branch() {
	if [ -n "${TARGET_BRANCH}" ]; then
		configure_github
		echo "Rebasing atop of the latest ${TARGET_BRANCH}"
		# Recover from any previous rebase left halfway
		git rebase --abort 2> /dev/null || true
		if ! git rebase origin/${TARGET_BRANCH}; then
			>&2 echo "Rebase failed, exiting"
			exit 1
		fi
	fi
}

function main() {
	action="${1:-}"

	case "${action}" in
	rebase-atop-of-the-latest-target-branch) rebase_atop_of_the_latest_target_branch;;
	*) >&2 echo "Invalid argument"; exit 2 ;;
	esac
}

main "$@"
