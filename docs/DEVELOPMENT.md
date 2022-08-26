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
export IMG=quay.io/user/cc-operator
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
kubectl create -f config/samples/ccruntime.yaml
```

## Uninstalling Operator

Ensure KUBECONFIG points to target Kubernetes cluster
```
make uninstall && make undeploy
```
