## Installation

You will need to use either Kubernetes 1.24 or 1.25 versions.
 
Ensure KUBECONFIG points to the target Kubernetes cluster.

Ensure at least one node in the cluster is having the label `node-role.kubernetes.io/worker=`.

```
kubectl label node $NODENAME node-role.kubernetes.io/worker=
```

Deploy the operator by running the following command.
```
kubectl apply -f https://raw.githubusercontent.com/confidential-containers/operator/main/deploy/deploy.yaml
```
The operator deploys all resources under `confidential-containers-system` namespace.

## Create Custom Resource (CR)
```
kubectl apply  -f https://raw.githubusercontent.com/confidential-containers/operator/main/config/samples/ccruntime.yaml
```

## Verify

- Check the status of operator PODs.

```
kubectl get pods -n confidential-containers-system
```
A successful install should show all PODs with "Running" status

```
NAME                                             READY   STATUS        RESTARTS   AGE
cc-operator-controller-manager-dc4846d94-nfnr7   2/2     Running       0          20h
cc-operator-daemon-install-bdp89                 1/1     Running       0          5s
cc-operator-pre-install-daemon-hclk9             1/1     Running       0          9s
```

- Check `RuntimeClasses`

```
kubectl get runtimeclass
```
A successful install should show the following `RuntimeClasses`
```
NAME        HANDLER     AGE
kata        kata        6m7s
kata-clh    kata-clh    6m7s
kata-qemu   kata-qemu   6m7s
```

## Changing Runtime bundle

You can change the runtime payload when creating the CR by changing the `payloadImage` attribute in the 
[manifest yaml](https://github.com/confidential-containers/operator/blob/main/config/samples/ccruntime.yaml#L14)


## Uninstallation

Delete the CR
```
kubectl delete  -f https://raw.githubusercontent.com/confidential-containers/operator/main/config/samples/ccruntime.yaml
```

Delete the Operator
```
kubectl delete -f https://raw.githubusercontent.com/confidential-containers/operator/main/deploy/deploy.yaml
```
