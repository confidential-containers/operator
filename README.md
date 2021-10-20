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

Ensure KUBECONFIG points to target Kubernetes cluster and IMG var is set
```
make install && make deploy
```

## Create Custom Resource (CR)
```
kubectl create -f config/samples/ccruntime.yaml
```

## Uninstalling Operator

Ensure KUBECONFIG points to target Kubernetes cluster and IMG var is set
```
make uninstall && make undeploy
```

## Runtime bundle

The operator by default uses the `quay.io/kata-containers/kata-deploy-cc:v0` image
as the payload.
You can change it when creating the CR by changing the `payloadImage` config.
The following yaml shows an example where `v2` version of the image is used
```
apiVersion: confidentialcontainers.org/v1beta1
kind: CcRuntime
metadata:
  name: ccruntime-sample
  namespace: confidential-containers-system
spec:
  # Add fields here
  runtimeName: kata
  config:
    installType: bundle
    payloadImage: quay.io/kata-containers/kata-deploy-cc:v2
```

