# Installation

## Prerequisites
- Ensure a minimum of 8GB RAM and 2 vCPU for the Kubernetes cluster node
- Only containerd runtime based Kubernetes clusters are supported with the current Confidential Containers (CoCo) release
- The minimum Kubernetes version should be 1.24.
- Ensure KUBECONFIG points to the target Kubernetes cluster.
- Ensure at least one Kubernetes node in the cluster is having the label `node.kubernetes.io/worker=`
  ```
  kubectl label node $NODENAME node.kubernetes.io/worker=
  ```

## Deploy the Operator

Deploy the operator by running the following command where `<RELEASE_VERSION>` needs to be substituted with the desired [release tag](https://github.com/confidential-containers/operator/tags). For example, to deploy the `v0.2.0` release run: `export RELEASE_VERSION="v0.2.0"`.

```
export RELEASE_VERSION=<RELEASE_VERSION>
kubectl apply -k "github.com/confidential-containers/operator/config/release?ref=${RELEASE_VERSION}"
```

The operator deploys all resources under `confidential-containers-system` namespace.

Wait until each pod has the STATUS of Running.

```
kubectl get pods -n confidential-containers-system --watch
```

### Custom Resource Definition (CRD)

The operator is responsible for creating the custom resource definition (CRD) which is
then used for creating a custom resource (CR).

The operator creates the `ccruntime` CRD as can be observed in the following command:

```
kubectl get crd | grep ccruntime

ccruntimes.confidentialcontainers.org   2022-09-08T06:10:37Z
```

Execute the following command to get details on the `ccruntime` CRD:

```
kubectl explain ccruntimes.confidentialcontainers.org
```

Output:

```
KIND:     CcRuntime
VERSION:  confidentialcontainers.org/v1beta1

DESCRIPTION:
     CcRuntime is the Schema for the ccruntimes API

FIELDS:
   apiVersion	<string>
     APIVersion defines the versioned schema of this representation of an
     object. Servers should convert recognized schemas to the latest internal
     value, and may reject unrecognized values. More info:
     https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources

   kind	<string>
     Kind is a string value representing the REST resource this object
     represents. Servers may infer this from the endpoint the client submits
     requests to. Cannot be updated. In CamelCase. More info:
     https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds

   metadata	<Object>
     Standard object's metadata. More info:
     https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata

   spec	<Object>
     CcRuntimeSpec defines the desired state of CcRuntime

   status	<Object>
     CcRuntimeStatus defines the observed state of CcRuntime
```

The complete CRD can be seen by running the following command:
```
kubectl explain --recursive=true ccruntimes.confidentialcontainers.org
```
You can also see the details of the `ccruntime` CRD in the following [file](https://github.com/confidential-containers/operator/blob/main/api/v1beta1/ccruntime_types.go#L90).

## Create Custom Resource (CR)

Creating a custom resource installs the required CC runtime pieces into the cluster node and creates the RuntimeClasses.

The default CR can be created as shown below where `<RELEASE_VERSION>` needs to be substituted with the
desired [release tag](https://github.com/confidential-containers/operator/tags):

```
kubectl apply -k "github.com/confidential-containers/operator/config/samples/ccruntime/default?ref=${RELEASE_VERSION}"
```

Wait until each pod has the `STATUS` as `Running`:

```
kubectl get pods -n confidential-containers-system --watch
```

## Verify

- Check the status of the operator PODs.

```
kubectl get pods -n confidential-containers-system
```

A successful install should show `STATUS` field of all pods as `Running`.

```
NAME                                              READY   STATUS    RESTARTS   AGE
cc-operator-controller-manager-5df7584679-kffzf   2/2     Running   0          21m
cc-operator-daemon-install-xz697                  1/1     Running   0          6m45s
cc-operator-pre-install-daemon-rtdls              1/1     Running   0          7m2s
```

- Check `RuntimeClasses`

```
kubectl get runtimeclass
```

A successful install should show the following `RuntimeClasses`.

```
NAME            HANDLER         AGE
kata            kata            9m55s
kata-clh        kata-clh        9m55s
kata-qemu       kata-qemu       9m55s
kata-qemu-tdx   kata-qemu-tdx   9m55s
kata-qemu-sev   kata-qemu-sev   9m55s
kata-qemu-snp   kata-qemu-snp   9m55s
```

## Changing Runtime bundle

You can change the runtime payload when creating the CR by creating a new [kustomize](https://kustomize.io) overlay
as shown below, where `<MY_CUSTOM_CR>`, `<MY_PAYLOAD>`, and `<TAG>` needs to be changed according to your payload.

```
make kustomize
cp -r config/samples/ccruntime/default config/samples/ccruntime/<MY_CUSTOM_CR>
cd config/samples/ccruntime/<MY_CUSTOM_CR>
../../../../bin/kustomize edit set image quay.io/kata-containers/kata-deploy=<MY_PAYLOAD>:<TAG>
```

Then install the new CR as:

```
kubectl apply -k config/samples/ccruntime/<MY_CUSTOM_CR>
```

## Uninstallation

### Delete the CR
```
kubectl delete -k "github.com/confidential-containers/operator/config/samples/ccruntime/default?ref=${RELEASE_VERSION}"
```

### Delete the Operator

```
kubectl delete -k "github.com/confidential-containers/operator/config/release?ref=${RELEASE_VERSION}"
```
