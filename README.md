# Introduction
An operator to deploy confidential containers runtime (and required configs) on a Kubernetes cluster

## Prerequisites
- Golang (1.16.x)
- Operator SDK version (1.11.x+)
- podman, podman-docker or docker
- Access to Kubernetes cluster (1.21+)
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
make install && make deploy IMG=quay.io/user/cc-operator
```

## Create Custome Resource (CR)
```
kubectl create -f config/samples/_v1beta1_confidentialcontainersruntime.yaml
```

## Uninstalling Operator

Ensure KUBECONFIG points to target Kubernetes cluster
```
make uninstall && make undeploy IMG=quay.io/user/cc-operator
```


