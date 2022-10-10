#!/bin/bash
#
# Copyright Confidential Containers Contributors
#
# SPDX-License-Identifier: Apache-2.0
#
set -o errexit
set -o nounset
set -o pipefail

script_dir="$(dirname "$(readlink -f "$0")")"
# Track the executed steps for the purpose of undoing only those needed.
step_bootstrap_env=0
step_start_cluster=0
step_install_operator=0
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
	pushd "$script_dir" >/dev/null
	# Do not try to undo steps that did not execute.
	if [ $step_install_operator -eq 1 ]; then
		echo "INFO: Uninstall the operator"
		sudo -E PATH="$PATH" bash -c './operator.sh uninstall' || true
	fi

	if [ $step_start_cluster -eq 1 ]; then
		echo "INFO: Shutdown the cluster"
		sudo -E PATH="$PATH" bash -c './cluster/down.sh' || true
	fi

	if [ $step_bootstrap_env -eq 1 ]; then
		echo "INFO: Undo the bootstrap"
		ansible-playbook -i localhost, -c local --tags undo ansible/main.yml || true
	fi
	popd >/dev/null
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
	step_bootstrap_env=1
	ansible-playbook -i localhost, -c local --tags untagged ansible/main.yml

	echo "INFO: Bring up the test cluster"
	step_start_cluster=1
	sudo -E PATH="$PATH" bash -c './cluster/up.sh'
	export KUBECONFIG=/etc/kubernetes/admin.conf

	echo "INFO: Build and install the operator"
	step_install_operator=1
	sudo -E PATH="$PATH" bash -c './operator.sh'

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
