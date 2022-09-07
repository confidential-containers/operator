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

runtimeclass=""
undo="false"

usage() {
	cat <<-EOF
	Prepare the local host and run end-to-end tests.
	It requires Ansible to run.
	Important: it will change the system so ensure it is executed in a development
	environment.

	Use: $0 [-h|--help] [-r RUNTIMECLASS] [-u], where:
	-h | --help : show this usage
	-r RUNTIMECLASS: configure to use the RUNTIMECLASS (e.g. kata-clh) on
                         the tests. Defaults to "kata-qemu".
	-u: undo the installation and configuration before exiting. Useful for
	    baremetal machine were it needs to do clean up for the next tests.
	EOF
}

parse_args() {
	while getopts "hr:u" opt; do
		case $opt in
			h) usage && exit 0;;
			r) runtimeclass="$OPTARG";;
			u) undo="true";;
			*) usage && exit 1;;
		esac
	done
}

undo_changes() {
	# TODO: in case the script failed, we should undo only the steps
	# executed.
	pushd "$script_dir" >/dev/null
	sudo -E PATH="$PATH" ./operator.sh uninstall || true
	sudo -E PATH="$PATH" ./cluster/down.sh || true
	ansible-playbook -i localhost, -c local --tags undo ansible/main.yml || true
	popd
}

on_exit() {
	if [ "$undo" == "true" ]; then
		undo_changes
	fi
}

trap on_exit EXIT

main() {
	local cmd

	parse_args $@

	# Check Ansible is installed.
	if ! command -v ansible-playbook >/dev/null; then
		echo "ERROR: ansible-playbook is required to run this script."
		exit 1
	fi

	pushd "$script_dir" >/dev/null
	echo "INFO: Bootstrap the local machine"
	ansible-playbook -i localhost, -c local --tags untagged ansible/main.yml

	echo "INFO: Bring up the test cluster"
	sudo -E PATH="$PATH" ./cluster/up.sh
	export KUBECONFIG=/etc/kubernetes/admin.conf

	echo "INFO: Build and install the operator"
	sudo -E PATH="$PATH" ./operator.sh

	echo "INFO: Run tests"
	cmd="sudo -E PATH=\"$PATH\" ./tests_runner.sh"
	[ -z $runtimeclass ] || cmd+=" -r $runtimeclass"
	eval $cmd
	popd >/dev/null
}

main "$@"
