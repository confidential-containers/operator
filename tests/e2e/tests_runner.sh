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
project_dir="$(readlink -f ${script_dir}/../..)"

export GOPATH="$(mktemp -d)"
tests_repo_dir="$GOPATH/src/github.com/kata-containers/tests"
export CI=true

# TODO: Debug should be enabled in order for some tests to enable the console
# debug and search for patterns on agent logs. Probably console debug should
# be enabled always.
export DEBUG=true

# The container runtime class (e.g. kata, kata-qemu, kata-clh) test pods should
# be created with.
runtimeclass="kata-qemu"

# Tests will attempt to re-configure containerd. In our case it is already
# proper set by the operator, so let's skip that step.
export TESTS_CONFIGURE_CC_CONTAINERD=no

clone_kata_tests() {
	local cc_branch="topic/CC-set-io.containerd.cri.runtime.handler-annotation"

	# TODO: checkout on the exact sha1 where the kata-deploy was created
	# so that we ensure the same tests are used here.
	git clone --branch="$cc_branch" \
		https://github.com/fidencio/kata-tests "$tests_repo_dir"
}

cleanup() {
	[ ! -s "/usr/local/bin/kata-runtime" ] || \
		unlink /usr/local/bin/kata-runtime
	rm -rf "$tests_repo_dir" || true
}

trap cleanup EXIT

parse_args() {
	while getopts "hr:" opt; do
		case $opt in
			h) usage && exit 0;;
			r) runtimeclass="$OPTARG";;
			*) usage && exit 1;;
		esac
	done
}

usage() {
	cat <<-EOF
	Utility to run the tests.

	Use: $0 [-h|--help] [-r RUNTIMECLASS], where:
	-h | --help : show this usage
	-r RUNTIMECLASS: run tests for RUNTIMECLASS (e.g. kata-clh).
	                 Defaults to "kata-qemu".
	EOF
}

# tests for CC without specific hardware support
run_non_tee_tests() {
	local runtimeclass="$1"
	local aa_kbc="${2:-"offline_fs_kbc"}"

	# This will be extended further to export differently based on a type of runtimeclass.
	# At the time of writing, it is assumed that all non-tee tests use offline_fs_kbc.
	# Discussion: https://github.com/confidential-containers/operator/pull/142#issuecomment-1359349595
	export AA_KBC="${aa_kbc}"

	# TODO: this is a workaround for the tests that rely on `kata-runtime kata-env`
	# to get the path to kata's configuration.toml and image files. Without this
	# they will change the default configuration.toml, which doesn't correspond to
	# the one used by the runtimeclass, therefore, performing bogus changes.
	local runtime_config_file="/opt/kata/share/defaults/kata-containers/"
	runtime_config_file+="configuration-${runtimeclass/kata-/}.toml"
	sed -i "s#kata-runtime kata-env#kata-runtime --config $runtime_config_file kata-env#g" \
		../../../lib/common.bash

	bats \
		"agent_image.bats" \
		"agent_image_encrypted.bats" \
		"${script_dir}/operator_tests.bats"

}

# Tests for CC with QEMU on SEV HW
run_kata_qemu_sev_tests() {
	bats "sev.bats"

}

# Tests for CC with QEMU on SNP HW
run_kata_qemu_snp_tests() {
	bats "snp.bats"
}

main() {

	parse_args $@

	clone_kata_tests

	# This will make the pods created by the tests to use the $runtimeclass.
	export RUNTIMECLASS="${runtimeclass}"

	cd "${tests_repo_dir}/integration/kubernetes/confidential"

	# Test scripts rely on kata-runtime so it should be reacheable on PATH.
	# Re-export PATH is error prone as some calls to kata-runtime use sudo,
	# so let's create a symlink.
	ln -sf /opt/kata/bin/kata-runtime \
		/usr/local/bin/kata-runtime

	# Run tests.
	case $runtimeclass in
		kata-qemu|kata-clh|kata-clh-tdx)
			echo "INFO: Running non-TEE tests for $runtimeclass using OfflineFS KBC"
			run_non_tee_tests "$runtimeclass"
			;;
		kata-qemu-se)
			echo "INFO: Running TEE (IBM SE) tests for $runtimeclass using OfflineFS KBC"
			run_non_tee_tests "$runtimeclass"
			;;
		kata-qemu-tdx)
			echo "INFO: Running non-TEE tests for $runtimeclass using CC KBC"
			run_non_tee_tests "$runtimeclass" "cc_kbc"
			;;
		kata-qemu-sev)
			echo "INFO: Running kata-qemu-sev tests for $runtimeclass"
			run_kata_qemu_sev_tests
			;;
		kata-qemu-snp)
			echo "INFO: Running kata-qemu-snp tests for $runtimeclass"
			run_kata_qemu_snp_tests
			;;
		*)
			echo "ERROR: no known tests for runtime class ${runtimeclass} "
			exit 1
			;;
	esac
}

main "$@"
