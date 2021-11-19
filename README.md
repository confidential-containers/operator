# Confidential Containers Operator

[![Build](https://github.com/confidential-containers/operator/actions/workflows/makefile.yml/badge.svg)](https://github.com/confidential-containers/operator/actions/workflows/makefile.yml)
[![Container Image](https://github.com/confidential-containers/operator/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/confidential-containers/operator/actions/workflows/docker-publish.yml)

This Confidential Containers Operator provides a means to deploy and manage Confidential Containers Runtime on Kubernetes clusters. 
The primary resource is `CcRuntime` which describes runtime details like installation type, source, nodes to deploy etc.

## Installation

Ensure KUBECONFIG points to the target Kubernetes cluster
```
kubectl apply -f https://raw.githubusercontent.com/confidential-containers/operator/main/deploy/deploy.yaml
```

## Create Custom Resource (CR)
```
kubectl apply  -f https://raw.githubusercontent.com/confidential-containers/operator/main/config/samples/ccruntime.yaml
```

## Changing Runtime bundle

The operator by default uses the `quay.io/confidential-contianers/runtime-payload:v0` image
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
    payloadImage: quay.io/confidential-contianers/runtime-payload:v2
```

## Uninstallation

Delete the CR
```
kubectl delete  -f https://raw.githubusercontent.com/confidential-containers/operator/main/config/samples/ccruntime.yaml
```

Delete the Operator
```
kubectl delete -f https://raw.githubusercontent.com/confidential-containers/operator/main/deploy/deploy.yaml
```
