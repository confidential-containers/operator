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

# Run kubeadm reset to clean up as much as possible the installed cluster.
#
reset_kubeadm() {
	# TODO: make it configurable.
	local cri_runtime_socket="/run/containerd/containerd.sock"

	kubeadm reset -f --cri-socket="${cri_runtime_socket}"

	# kubectl -n kube-system get cm kubeadm-config -o yaml
	rm -rf ~/.kube/ || true

	# TODO: recover iptables
	#
	# The reset process does not reset or clean up iptables rules or IPVS tables.
        # If you wish to reset iptables, you must do so manually by using the "iptables" command.
        #If your cluster was setup to utilize IPVS, run ipvsadm --clear (or similar) to reset your system's IPVS tables.

}

remove_cni() {
	local dev="cni0"

	rm -rf /etc/cni/net.d
	ip link set dev "$dev" down || true
	ip link del "$dev" || true
}

remove_flannel() {
	local dev="flannel.1"

	ip link set dev "$dev" down || true
	ip link del "$dev" || true
}

main() {
	reset_kubeadm
	remove_cni
	remove_flannel
}

main "$@"
