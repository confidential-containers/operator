## Introduction
These instructions outline how to build and install the operator with the peerpods
controllers. Please refer to the DEVELOPMENT.md and INSTALL.md guides for additional
instructions and prerequisites.

Please visit (https://github.com/confidential-containers/cloud-api-adaptor) for more
information on peerpods, peerpodconfig-ctrl, and peerpod-ctrl.

## Note
Currently only the libvirt provider is supported.

## Set Environment Variables
```
export QUAY_USER=<userid>
export IMG=quay.io/${QUAY_USER}/cc-operator
export PEERPODS=1
```

## Building Operator image
```
make docker-build
make docker-push
```

## Deploying Operator
Ensure KUBECONFIG points to target Kubernetes cluster
```
make install && make deploy
```

## Apply the ssh secret needed for the libvirt connection.
```
kubectl create secret generic ssh-key-secret -n confidential-containers-system --from-file=id_rsa.pub=./id_rsa.pub --from-file=id_rsa=./id_rsa
```

## Apply the peerpod secret and config map
Following is an example for libvirt. Change it if you are using AWS/Azure/IBMCloud etc...

```
kubectl apply -f peer-pods-cm.yaml
```
peer-pods-cm.yaml example
```
apiVersion: v1
kind: ConfigMap
metadata:
  name: peer-pods-cm
  namespace: confidential-containers-system
data:
  CLOUD_PROVIDER: "libvirt"
  PROXY_TIMEOUT: 30m
```

```
kubectl apply -f peer-pods-secret.yaml
```
peer-pods-secret.yaml example
```
apiVersion: v1
kind: Secret
metadata:
  name: peer-pods-secret
  namespace: confidential-containers-system
type: Opaque
stringData:
  CLOUD_PROVIDER: "libvirt"
  VXLAN_PORT: "9000"
  LIBVIRT_URI: "qemu+ssh://root@192.168.122.1/system?no_verify=1"
  LIBVIRT_NET: "default"
  LIBVIRT_POOL: "default"
```

## Create Custom Resource (CR)
```
kubectl create -k config/samples/ccruntime/peerpods
```

## Uninstalling Operator

Ensure KUBECONFIG points to target Kubernetes cluster
```
make uninstall && make undeploy
```
