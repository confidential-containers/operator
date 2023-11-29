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

# The container runtime class (e.g. kata, kata-qemu, kata-clh) test pods should
# be created with.
runtimeclass="kata-qemu"

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

main() {
	parse_args $@

	# This will make the pods created by the tests to use the $runtimeclass.
	export RUNTIMECLASS="${runtimeclass}"

	# Run tests.
	case $runtimeclass in
		kata-qemu|kata-clh|kata-clh-tdx|kata-qemu-se|kata-qemu-tdx|kata-qemu-sev|kata-qemu-snp)
			echo "INFO: Running operator tests for $runtimeclass"
			bats "${script_dir}/operator_tests.bats"
			;;
		*)
			echo "ERROR: no known tests for runtime class ${runtimeclass} "
			exit 1
			;;
	esac
}

main "$@"
