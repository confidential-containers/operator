#!/bin/bash
#
# Copyright 2022 Red Hat
#
# SPDX-License-Identifier: Apache-2.0
#
set -o errexit
set -o nounset
set -o pipefail

script_dir="$(dirname "$(readlink -f "$0")")"

usage() {
	cat <<-EOF
	$0: run the end-to-end tests on local host.

	It requires Ansible to run.
	Important: it will change the system so ensure it is executed in a development
	environment.
	EOF
}

parse_args() {
	while getopts "h" opt; do
		case $opt in
			h) usage && exit 0;;
			*) usage && exit 1;;
		esac
	done
}

main() {
	parse_args $@

	# Check Ansible is installed.
	if ! command -v ansible-playbook >/dev/null; then
		echo "ERROR: ansible-playbook is required to run this script."
		exit 1
	fi

	pushd "$script_dir" >/dev/null
	echo "INFO: Bootstrap the local machine"
	ansible-playbook -i localhost, -c local ansible/main.yml

	echo "INFO: Bring up the test cluster"
	sudo ./cluster/up.sh
	export KUBECONFIG=/etc/kubernetes/admin.conf

	echo "INFO: Build and install the operator"
	sudo -E PATH="$PATH" ./operator.sh

	echo "INFO: Run tests"
	sudo -E PATH="$PATH" ./tests_runner.sh
	popd >/dev/null
}

main "$@"
