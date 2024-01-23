# Confidential Containers with IBM Secure Execution

This document explains how to install and run a confidential container on an IBM Secure 
Execution-enabled Z machine. A secure image is an encrypted Linux image comprising a kernel image,
an initial RAM file system (initrd) image, and a file specifying kernel parameters (parmfile).
It is an essential component for running a confidential container. The public key used for
encryption is associated with a private key managed by a trusted firmware called
[ultravisor](https://www.ibm.com/docs/en/linux-on-systems?topic=execution-components).

This means that a secure image is machine-specific, resulting in its absence from a released
payload image in `ccruntime`. To use it, you need to build a secure image with your own public
key and create a payload image bundled with it. The following sections elaborate on how to
accomplish this step-by-step.

## Prerequisites

Kindly review the [section](https://github.com/confidential-containers/operator/blob/main/docs/INSTALL.md#prerequisites) titled identically in the `INSTALL.md` document.

- `kustomize`: Kubernetes native configuration management tool which can be installed simply by:

```
$ mkdir -p $GOPATH/src/github.com/confidential-containers
$ cd $GOPATH/src/github.com/confidential-containers
$ git clone https://github.com/confidential-containers/operator.git && cd operator
$ make kustomize
$ export PATH=$PATH:$(pwd)/bin
```

## Build a Payload Image via kata-deploy

If you have a local container registry running at `localhost:5000`, refer to the
[document](https://github.com/kata-containers/kata-containers/blob/main/docs/how-to/how-to-run-kata-containers-with-SE-VMs.md#using-kata-deploy-with-confidential-containers-operator)
on Kata Containers for details on building a payload image.

## Install Operator

Let us install an operator with:

```
$ cd $GOPATH/src/github.com/confidential-containers/operator
$ export IMG=localhost:5000/cc-operator
$ make docker-build && make docker-push
$ make install && make deploy
namespace/confidential-containers-system created
customresourcedefinition.apiextensions.k8s.io/ccruntimes.confidentialcontainers.org created
serviceaccount/cc-operator-controller-manager created
role.rbac.authorization.k8s.io/cc-operator-leader-election-role created
clusterrole.rbac.authorization.k8s.io/cc-operator-manager-role created
clusterrole.rbac.authorization.k8s.io/cc-operator-metrics-reader created
clusterrole.rbac.authorization.k8s.io/cc-operator-proxy-role created
rolebinding.rbac.authorization.k8s.io/cc-operator-leader-election-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/cc-operator-manager-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/cc-operator-proxy-rolebinding created
configmap/cc-operator-manager-config created
service/cc-operator-controller-manager-metrics-service created
deployment.apps/cc-operator-controller-manager created
$ kubectl get pods -n confidential-containers-system
NAME                                              READY   STATUS    RESTARTS   AGE
cc-operator-controller-manager-64fcc48b74-xd9sq   2/2     Running   0          74s
```

## Install Custom Resource (CR)

You can install `ccruntime` with an existing overlay directory named
[`s390x`](config/samples/ccruntime/s390x), by replacing the image name and tag
for a payload image with the ones you pushed to the local registry
(e.g. `localhost:5000/build-kata-deploy:latest`):

```
$ cd config/samples/ccruntime/s390x
$ kustomize edit set image quay.io/kata-containers/kata-deploy=localhost:5000/build-kata-deploy:latest
$ kubectl create -k .
ccruntime.confidentialcontainers.org/ccruntime-sample-s390x created
$ kubectl get pods -n confidential-containers-system
NAME                                              READY   STATUS    RESTARTS   AGE
cc-operator-controller-manager-64fcc48b74-xd9sq   2/2     Running   0          3m
cc-operator-daemon-install-9t4qd                  1/1     Running   0          2m
cc-operator-pre-install-daemon-gsnj7              1/1     Running   0          2m
$ # To verify if a payload image is pulled from the updated location
$ kubectl get pods -oyaml -n confidential-containers-system cc-operator-daemon-install-9t4qd | grep image:
    image: localhost:5000/build-kata-deploy:test
    image: localhost:5000/build-kata-deploy:test
```

You have to wait until a set of runtime classes is deployed like:

```
$ kubectl get runtimeclass
NAME           HANDLER        AGE
kata           kata-qemu      60s
kata-qemu      kata-qemu      61s
kata-qemu-se   kata-qemu-se   61s
```

## Verify the Installation

To verify the installation, use the following runtime class: `kata-qemu-se`:

```
$ cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: nginx-kata
spec:
  runtimeClassName: kata-qemu-se
  containers:
  - name: nginx
    image: nginx
EOF
pod/nginx-kata created
$ kubectl get pods
NAME         READY   STATUS    RESTARTS   AGE
nginx-kata   1/1     Running   0          15s
```

## Uninstall Resources

You can uninstall confidential containers by removing the resources in reverse order:

```
$ cd $GOPATH/src/github.com/confidential-containers/operator
$ kubectl delete -k config/samples/ccruntime/s390x
$ make undeploy
```
