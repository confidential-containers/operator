## Prerequisites

- Kubernetes Cluster Setup
  
  Use kcli or kubeadmin to create a Kubernetes cluster on Ubuntu 20.04

  kcli install and setup instructions are available from https://kcli.readthedocs.io/en/latest/

  Use kcli to create a two-node cluster using Ubuntu 20.04
  ```
  kcli create kube generic -P image=ubuntu2004 -P workers=1 testk8s
  ```

  If using single node cluster then label the node as shown below
  ```
  kubectl label node <node-name> node-role.kubernetes.io/worker=
  ```

- Replace containerd on the worker
  
  Replace containerd on the worker node by building a new containerd from https://github.com/confidential-containers/containerd/tree/CC-main


## Installation

Ensure KUBECONFIG points to the target Kubernetes cluster
```
kubectl apply -f https://raw.githubusercontent.com/confidential-containers/operator/ccv0-demo/deploy/deploy.yaml
```

## Create Custom Resource (CR)
```
kubectl apply  -f https://raw.githubusercontent.com/confidential-containers/operator/ccv0-demo/config/samples/ccruntime.yaml
```

## Verify the RuntimeClasses
```
kubectl get runtimeclass
```

You should see a similar output like below:

```
NAME        HANDLER     AGE
kata        kata        13s
kata-cc     kata-cc     13s
kata-qemu   kata-qemu   13s
```

## Create a sample POD

```
cat >nginx-cc.yaml <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-cc
spec:
  selector:
    matchLabels:
      app: nginx
  replicas: 1
  template:
    metadata:
      labels:
        app: nginx
    spec:
      runtimeClassName: kata-cc
      containers:
      - name: nginx
        image: bitnami/nginx:1.14
        ports:
        - containerPort: 80
        imagePullPolicy: Always  
EOF

kubectl apply -f nginx-cc.yaml
```

### Verify 

Check if the POD is in "Running" state.

```
kubectl get deploy

NAME                  READY   UP-TO-DATE   AVAILABLE   AGE
nginx-deployment-cc   1/1     1            1           24s


kubectl get pods -o wide

NAME                                   READY   STATUS    RESTARTS   AGE   IP            NODE               NOMINATED NODE   READINESS GATES
nginx-deployment-cc-5c4d54569f-4wlqg   1/1     Running   0          1m    10.244.1.18   testk8s-worker-0   <none>           <none>
```

`Exec` is blocked and will error out.

```
kubectl exec -it nginx-deployment-cc-5c4d54569f-4wlqg -- bash

error: Internal error occurred: error executing command in container: failed to exec in container: failed to start exec "8f8abe4a06a9bd3a2f70803336064dfbcbb8f7595412eed6efb2d6ee665f737f": cannot enter container 830f6da3ed785ed7f082c11482610e307f9b6aa5f18fe6c1be36c165b578f31e, with err rpc error: code = Unimplemented desc = ExecProcessRequest is blocked: unknown
```

Verify the `rootfs` of the container is not present on the worker node.

Get container ID from the POD.

```
export PODNAME=nginx-deployment-cc-5c4d54569f-4wlqg
containerID=$(kubectl get pod $PODNAME -o=jsonpath='{.status.containerStatuses[*].containerID}' | cut -d "/" -f3)

echo $containerID

830f6da3ed785ed7f082c11482610e307f9b6aa5f18fe6c1be36c165b578f31e
```

Login to the worker node (`testk8s-worker-0`) and run the following commands to extract the sandbox ID.

```
export containerID=830f6da3ed785ed7f082c11482610e307f9b6aa5f18fe6c1be36c165b578f31e

sandboxID=$(sudo crictl -r unix:///run/containerd/containerd.sock inspect $containerID | grep 'sandboxID' | cut -d ":" -f2 | sed 's/,//g;s/"//g;s/ //g')

echo $sandboxID
```

Check container `rootfs`.
```
sudo -E sandboxID=$sandboxID su
cd /run/kata-containers/shared/sandboxes/$sandboxID/shared
find . -name rootfs

./c560ddca8bca8e8f98f9879bc93bb6ec8d5a65b98cb43c1da3dca77e65e7d3da/rootfs

ls -l ./c560ddca8bca8e8f98f9879bc93bb6ec8d5a65b98cb43c1da3dca77e65e7d3da/rootfs
total 684
drwxr-xr-x 2 root root   4096 May 11 05:21 dev
drwxr-xr-x 2 root root   4096 May 11 05:21 etc
-rwxr-xr-x 1 root root 682696 Aug 25  2021 pause
drwxr-xr-x 2 root root   4096 May 11 05:21 proc
drwxr-xr-x 2 root root   4096 May 11 05:21 sys

```

As you can see, only the `rootfs` for the `pause` container is present on the host. The `rootfs for the application container `nginx`
is inside the VM.
 
For regular Kata containers, using either `kata` or `kata-qemu` runtimeclass, you'll find the `rootfs` for all the containers.

## Uninstallation

Delete the CR
```
kubectl delete  -f https://raw.githubusercontent.com/confidential-containers/operator/ccv0-demo/config/samples/ccruntime.yaml
```

Delete the Operator
```
kubectl delete -f https://raw.githubusercontent.com/confidential-containers/operator/ccv0-demo/deploy/deploy.yaml
```
