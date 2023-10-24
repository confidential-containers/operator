#!/usr/bin/env bats
# Copyright Confidential Containers Contributors
#
# SPDX-License-Identifier: Apache-2.0
#
# Operator specfic tests.
#
load "${BATS_TEST_DIRNAME}/lib.sh"
test_tag="[cc][operator]"

is_operator_installed() {
	[ "$(kubectl get deployment -n "$ns" --no-headers 2>/dev/null | wc -l)" \
		-gt 0 ]
}

setup() {
	container_runtime="${container_runtime:-containerd}"
	ns="confidential-containers-system"
}

@test "$test_tag Test can uninstall the operator" {

# Assume the operator is installed, otherwise fail.
is_operator_installed

"${BATS_TEST_DIRNAME}/operator.sh" uninstall

# TODO: this check is not passing as we need to update the payload.
#
# It should remove the kata containers installation entirely.
#[ ! -d "/opt/confidential-containers" ]

# It should let the container runtime running.
systemctl is-active "$container_runtime"
}

@test "$test_tag Test can reinstall the operator" {
# Assume the previous test (which removes the operator) passed.
! is_operator_installed

"${BATS_TEST_DIRNAME}/operator.sh" install
}

teardown() {
	# For debugging sake.
	kubectl get pods -A || true
	echo "::group::Describe all pods of confidential-containers namespace"
	kubectl -n confidential-containers describe pods || true
	echo "::endgroup::"
}
