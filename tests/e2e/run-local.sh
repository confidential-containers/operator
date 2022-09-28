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

readonly default_runtime_payload="ccruntime"
runtimeclass=""
runtime_payload=""
undo="false"

usage() {
	cat <<-EOF
	Prepare the local host and run end-to-end tests.
	It requires Ansible to run.
	Important: it will change the system so ensure it is executed in a development
	environment.

	Use: $0 [-h|--help] [-p RUNTIME_PAYLOAD] [-r RUNTIMECLASS] [-u], where:
	-h | --help : show this usage
	-p RUNTIME_PAYLOAD: install an alternative runtime payload (e.g.
	                    cc-demo-runtime). Defaults to "ccruntime".
	-r RUNTIMECLASS: configure to use the RUNTIMECLASS (e.g. kata-clh) on
	                 the tests. Defaults to "kata-qemu".
	-u: undo the installation and configuration before exiting. Useful for
	    baremetal machine were it needs to do clean up for the next tests.
	EOF
}

parse_args() {
	while getopts "hp:r:u" opt; do
		case $opt in
			h) usage && exit 0;;
			p) runtime_payload="$OPTARG";;
			r) runtimeclass="$OPTARG";;
			u) undo="true";;
			*) usage && exit 1;;
		esac
	done

	# Use the default payload if none is passed.
	[ -n "$runtime_payload" ] || runtime_payload="$default_runtime_payload"
}

undo_changes() {
	# TODO: in case the script failed, we should undo only the steps
	# executed.
	pushd "$script_dir" >/dev/null
	sudo -E PATH="$PATH" bash -c "./operator.sh uninstall $runtime_payload" || true
	sudo -E PATH="$PATH" bash -c './cluster/down.sh' || true
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

	export "PATH=$PATH:/usr/local/bin"

	pushd "$script_dir" >/dev/null
	echo "INFO: Bootstrap the local machine"
	ansible-playbook -i localhost, -c local --tags untagged ansible/main.yml

	echo "INFO: Bring up the test cluster"
	sudo -E PATH="$PATH" bash -c './cluster/up.sh'
	export KUBECONFIG=/etc/kubernetes/admin.conf

	echo "INFO: Build the operator"
	sudo -E PATH="$PATH" bash -c './operator.sh build'
	echo "INFO: Install the operator"
	sudo -E PATH="$PATH" bash -c "./operator.sh install $runtime_payload"

	echo "INFO: Run tests"
	cmd="sudo -E PATH=\"$PATH\" bash -c "
	if [ -z "$runtimeclass" ]; then
		cmd+="'./tests_runner.sh'"
	else
		cmd+="'./tests_runner.sh -r $runtimeclass'"
	fi
	eval $cmd
	popd >/dev/null
}

main "$@"
