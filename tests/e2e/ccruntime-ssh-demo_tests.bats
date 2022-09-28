#!/usr/bin/env bats
# Copyright Confidential Containers Contributors
#
# SPDX-License-Identifier: Apache-2.0
#
# Implement tests for the ccruntime-ssh-demo runtime.
#
load "${BATS_TEST_DIRNAME}/lib.sh"
test_tag="[cc][ssh demo]"

RUNTIMECLASS="${RUNTIMECLASS:-kata}"
deployment_file="$BATS_TEST_TMPDIR/ssh-demo.yaml"
ssh_key_file="$BATS_TEST_TMPDIR/ssh-key"

setup() {
	cat <<- EOF > "$ssh_key_file"
	-----BEGIN OPENSSH PRIVATE KEY-----
	b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
	QyNTUxOQAAACAfiGV2X4o+6AgjVBaY/ZR2UvZp84dVYF5bpNZGMLylQwAAAIhawtHJWsLR
	yQAAAAtzc2gtZWQyNTUxOQAAACAfiGV2X4o+6AgjVBaY/ZR2UvZp84dVYF5bpNZGMLylQw
	AAAEAwWYIBvBxQZgk0irFku3Lj1Xbfb8dHtVM/kkz/Uz/l2h+IZXZfij7oCCNUFpj9lHZS
	9mnzh1VgXluk1kYwvKVDAAAAAAECAwQF
	-----END OPENSSH PRIVATE KEY-----
	EOF

	chmod 600 "$ssh_key_file"

	cat <<- EOF > "$deployment_file"
	kind: Service
	apiVersion: v1
	metadata:
	  name: ccv0-ssh
	spec:
	  selector:
	    app: ccv0-ssh
	  ports:
	  - port: 22
	---
	kind: Deployment
	apiVersion: apps/v1
	metadata:
	  name: ccv0-ssh
	spec:
	  selector:
	    matchLabels:
	      app: ccv0-ssh
	  template:
	    metadata:
	      labels:
	        app: ccv0-ssh
	    spec:
	      runtimeClassName: ${RUNTIMECLASS}
	      containers:
	      - name: ccv0-ssh
	        image: docker.io/katadocker/ccv0-ssh
	        imagePullPolicy: Always
	EOF

	# TODO: improve me.
	kubectl delete -f "$deployment_file" || true
	sleep 10
}

@test "$test_tag Can ssh in demo pod" {
	kubectl apply -f "${deployment_file}"
	# TODO: wait it be ready.
	sleep 5
	local pod_ip_address=$(kubectl get service ccv0-ssh \
		-o jsonpath="{.spec.clusterIP}")

	# TODO: wait it be ready.
	sleep 30
	ssh -i "$ssh_key_file" -o StrictHostKeyChecking=accept-new root@${pod_ip_address} \
		grep 'agent.config_file=/etc/agent-config.toml' /proc/cmdline
}

teardown() {
	kubectl delete -f "${deployment_file}" || true
}
