# Introduction
An operator to deploy confidential containers runtime (and required configs) on a Kubernetes cluster

## Installation

Ensure KUBECONFIG points to target Kubernetes cluster
```
kubectl apply -f https://raw.githubusercontent.com/confidential-containers/operator/main/deploy/deploy.yaml
```

## Create Custom Resource (CR)
```
kubectl apply  -f https://raw.githubusercontent.com/confidential-containers/operator/main/config/samples/ccruntime.yaml
```

## Changing Runtime bundle

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

## Uninstallation

Delete the CR
```
kubectl delete  -f https://raw.githubusercontent.com/confidential-containers/operator/main/config/samples/ccruntime.yaml
```

Delete the Operator
```
kubectl delete -f https://raw.githubusercontent.com/confidential-containers/operator/main/deploy/deploy.yaml
```
