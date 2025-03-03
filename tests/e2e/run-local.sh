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
timeout="false"

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
	-t: enable default timeout for each operation (useful for CI)
	EOF
}

parse_args() {
	while getopts "hr:ut" opt; do
		case $opt in
			h) usage && exit 0;;
			r) runtimeclass="$OPTARG";;
			u) undo="true";;
			t) timeout="true";;
			*) usage && exit 1;;
		esac
	done
}

run() {
	duration=$1; shift
	if [ "$timeout" == "true" ]; then
		timeout -v -s 9 $duration "$@"
	else
		"$@"
	fi
}

undo_changes() {
	pushd "$script_dir" >/dev/null
	# Do not try to undo steps that did not execute.
	if [ $step_install_operator -eq 1 ]; then
		echo "::info:: Uninstall the operator"
		run 10m sudo -E PATH="$PATH" bash -xc './operator.sh uninstall' || true
	fi

	if [ $step_start_cluster -eq 1 ]; then
		echo "::info:: Shutdown the cluster"
		run 5m sudo -E PATH="$PATH" bash -xc './cluster/down.sh' || true
	fi

	if [ $step_bootstrap_env -eq 1 ]; then
		echo "::info:: Undo the bootstrap"
		run 5m ansible-playbook -i localhost, -c local --tags undo ansible/main.yaml || true
	fi
	popd >/dev/null
}

on_exit() {
	RET="$?"
	if [ "$undo" == "true" ]; then
		[ "$RET" -ne 0 ] && echo && echo "::error:: Testing failed with $RET, starting the cleanup..."
		undo_changes
	fi
	[ "$RET" -ne 0 ] && echo && echo "::error:: Testing failed with $RET" || echo "::info:: Testing passed"
}

trap on_exit EXIT

main() {
	local cmd

	parse_args $@

	# Check Ansible is installed.
	if ! command -v ansible-playbook >/dev/null; then
		echo "::error:: ansible-playbook is required to run this script."
		exit 1
	fi

	export "PATH=$PATH:/usr/local/bin"

	pushd "$script_dir" >/dev/null
	echo "::info:: Bootstrap the local machine"
	step_bootstrap_env=1
	run 10m ansible-playbook -i localhost, -c local --tags untagged ansible/main.yaml

	echo "::info:: Bring up the test cluster"
	step_start_cluster=1
	run 10m sudo -E PATH="$PATH" bash -xc './cluster/up.sh'
	export KUBECONFIG=/etc/kubernetes/admin.conf

	echo "::info:: Build and install the operator"
	step_install_operator=1
	run 20m sudo -E PATH="$PATH" bash -xc './operator.sh'

	echo "::info:: Run tests"
	local cmd="run 20m sudo -E PATH=\"$PATH\" bash -c "
	if [ -z "$runtimeclass" ]; then
		cmd+="'bash -x ./tests_runner.sh'"
	else
		cmd+="'bash -x ./tests_runner.sh -r $runtimeclass'"
	fi
	eval $cmd
	popd >/dev/null
}

main "$@"
