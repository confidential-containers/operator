# Introduction
These instructions should help you to build a custom version of the operator with your
changes

## Prerequisites
- Golang (1.18.x)
- Operator SDK version (1.23.x+)
- podman and podman-docker or docker
- Access to Kubernetes cluster (1.24+)
- Container registry to store images


## Set Environment Variables
```
export QUAY_USER=<userid>
export IMG=quay.io/${QUAY_USER}/cc-operator
```

## Viewing available Make targets
```
make help
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

## Create Custom Resource (CR)
```
kubectl create -k config/samples/ccruntime/default
```

## Uninstalling Operator

Ensure KUBECONFIG points to target Kubernetes cluster
```
make uninstall && make undeploy
```

## Using Kind Kubernetes cluster

You can use a [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/) cluster running on non-TEE hardware 
for development purposes.

Kind version `v0.16.0` have been successfully tested on the following Linux distros.
- `CentOS Stream 8`
- `RHEL9`
- `Ubuntu 20.04`
- `Ubuntu 22.04`

>**Note**: Only `kata-clh` runtimeclass works with Kind cluster.
