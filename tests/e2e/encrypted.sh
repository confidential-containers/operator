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
source "${script_dir}/lib.sh"

export key_file="${key_file:-image_key}"
export key_path="${key_path:-/default/image_key/nginx}"
export trustee_repo="${trustee_repo:-https://github.com/confidential-containers/trustee.git}"
export kbs_namespace="${kbs_namespace:-coco-tenant}"
export aa_kbc="${aa_kbc:-cc_kbc}"
export kbs_svc_name="${kbs_svc_name:-kbs}"
export kbs_ingress_name="${kbs_ingress_name:-kbs}"
export RUNTIMECLASS="${RUNTIMECLASS:-}"
export username="${username:-}"
export encrypted_image="${encrypted_image:-ghcr.io/$username/nginx:encrypted}"

build_encryption_key() {
	head -c 32 /dev/urandom | openssl enc > "${key_file}"
	mkdir output
	docker run -v "$PWD/output:/output" ghcr.io/confidential-containers/staged-images/coco-keyprovider:075b9a9ee77227d9d92b6f3649ef69de5e72d204 /encrypt.sh \
		-k "$(base64 < ${key_file})" \
		-i kbs:///some/key/id \
		-s docker://nginx:stable \
		-d dir:/output

	skopeo copy dir:output "docker://${encrypted_image}"
	skopeo inspect "docker://${encrypted_image}" | jq -r '.LayersData[].MIMEType' | grep encrypted
}

deploy_k8s_kbs() {
	if [ ! -d "${script_dir}/trustee" ]; then
		git clone "${trustee_repo}"
	fi

	pushd "${script_dir}/trustee/kbs/config/kubernetes"
		cat "${script_dir}/${key_file}" > overlays/key.bin
		export DEPLOYMENT_DIR=nodeport
		./deploy-kbs.sh

		kbs_pod=$(kubectl -n "${kbs_namespace}" get pods -o NAME)
		kubectl rollout status -w --timeout=30s deployment/kbs -n "${kbs_namespace}"
	popd
}

delete_k8s_kbs() {
	pushd "${script_dir}/trustee/kbs/config/kubernetes"
		kubectl delete -k overlays/
	popd
	rm -rf "${script_dir}/trustee"
}

provide_image_key() {
	kubectl exec -n "${kbs_namespace}" "${kbs_pod}" -- mkdir -p "/opt/confidential-containers/kbs/repository/$(dirname "$key_path")"
	cat "${script_dir}/${key_file}" | kubectl exec -i -n "${kbs_namespace}" "${kbs_pod}" -- tee "/opt/confidential-containers/kbs/repository/${key_path}" > /dev/null
}

launch_pod() {
	kubectl apply -f "${script_dir}/nginx-encrypted.yaml"
	kubectl rollout status -w --timeout=30s deployment/nginx-encrypted
}

delete_pod() {
	kubectl delete -f "${script_dir}/nginx-encrypted.yaml"
}

check_image_key() {
	kubectl logs -n "${kbs_namespace}" "${kbs_pod}" | grep "${key_path}"
}

set_metadata_annotation() {
	local yaml="${1}"
	local key="${2}"
	local value="${3}"
	local metadata_path="${4:-}"
	local annotation_key=""

 	[ -n "$metadata_path" ] && annotation_key+="${metadata_path}."

 	# yaml annotation key name.
 	annotation_key+="metadata.annotations.\"${key}\""

 	echo "$annotation_key"
	# yq set annotations in yaml. Quoting the key because it can have
	# dots.
	yq -i ".${annotation_key} = \"${value}\"" "${yaml}"
}

set_aa_kbc() {
	local cc_kbs_addr
	export cc_kbs_addr=$(kbs_k8s_svc_http_addr)
	kernel_params_annotation="io.katacontainers.config.hypervisor.kernel_params"
	kernel_params_value="agent.guest_components_rest_api=resource"
 	if [ "${aa_kbc}" = "cc_kbc" ]; then
		kernel_params_value+=" agent.aa_kbc_params=cc_kbc::${cc_kbs_addr}"
	fi
	set_metadata_annotation "${script_dir}/nginx-encrypted.yaml" \
		"${kernel_params_annotation}" \
		"${kernel_params_value}"
}

kbs_k8s_svc_http_addr() {
	local host
	local port

 	host=$(kbs_k8s_svc_host)
	port=$(kbs_k8s_svc_port)

	echo "http://${host}:${port}"
}

kbs_k8s_svc_host() {
	if kubectl get ingress -n "${kbs_namespace}" 2>/dev/null | grep -q kbs; then
		kubectl get ingress "${kbs_ingress_name}" -n "${kbs_namespace}" \
			-o jsonpath='{.spec.rules[0].host}' 2>/dev/null
	elif kubectl get svc "${kbs_svc_name}" -n "${kbs_namespace}" &>/dev/null; then
			local host
			host=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}' -n "${kbs_namespace}")
			echo "$host"
	else
		kubectl get svc "${kbs_svc_name}" -n "${kbs_namespace}" \
			-o jsonpath='{.spec.clusterIP}' 2>/dev/null
	fi
}

kbs_k8s_svc_port() {
	if kubectl get ingress -n "${kbs_namespace}" 2>/dev/null | grep -q kbs; then
		echo "80"
	elif kubectl get svc "${kbs_svc_name}" -n "${kbs_namespace}" &>/dev/null; then
		kubectl get svc "${kbs_svc_name}" -n "${kbs_namespace}" -o jsonpath='{.spec.ports[0].nodePort}'
	else
		kubectl get svc "${kbs_svc_name}" -n "${kbs_namespace}" \
			-o jsonpath='{.spec.ports[0].port}' 2>/dev/null
	fi
}

set_parameters() {
	sudo sed -i "s/runtimeclass/${RUNTIMECLASS}/g" "${script_dir}/nginx-encrypted.yaml"
	sudo sed -i "s/user/${username}/g" "${script_dir}/nginx-encrypted.yaml"
}

main() {
	build_encryption_key
	deploy_k8s_kbs
	set_aa_kbc
 	set_parameters
	provide_image_key
	launch_pod
	check_image_key
	delete_k8s_kbs
 	delete_pod
}

main "$@"
