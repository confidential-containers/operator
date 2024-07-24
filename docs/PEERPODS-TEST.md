## Introduction
This document describes testing to confirm that the operators are functioning.
These tests were performed on a libvirt provider as an example.

## Check that the peerpod-config CR was created
```
kubectl get peerpodconfigs -n confidential-containers-system
NAME                 AGE
coco-config-peer-pods   15m
```

## Check that the peerpodconfig-ctrl controller created the cloud adapter daemon and it is running
```
kubectl get pods -n confidential-containers-system
NAME                                              READY   STATUS    RESTARTS   AGE
cc-operator-controller-manager-68b5979488-x8bhb   2/2     Running   0          31m
cc-operator-daemon-install-lrwc5                  1/1     Running   0          12s
cc-operator-pre-install-daemon-2cllk              1/1     Running   0          14s
peerpodconfig-ctrl-caa-daemon-bv45g               1/1     Running   0          14s
```

## Create a Peerpod
```
kubectl apply -f fedora-sleep.yaml
```
```
fedora-sleep.yaml
apiVersion: v1
kind: Pod
metadata:
  name: fedora-sleep
spec:
  runtimeClassName: kata-remote
  restartPolicy: Never
  containers:
    - name: sleep-forever
      image: registry.fedoraproject.org/fedora
      command: ["sleep"]
      args: [ "infinity"]

```

## Check that the peerpod CR was created as a result of creating the peerpod
```
kubectl get peerpods
NAME                          AGE
fedora-sleep-resource-gqjl6   7m26s
```

## Check that the peerpod pod is running and that a VM was created and is up
```
kubectl get pods
NAME           READY   STATUS    RESTARTS   AGE
fedora-sleep   1/1     Running   0          7m7s

kcli get vms
+-----------------------------+--------+-----------------+------------+----------+---------+
|             Name            | Status |        Ip       |   Source   |   Plan   | Profile |
+-----------------------------+--------+-----------------+------------+----------+---------+
|     coco-k8s-ctlplane-0     |   up   | 192.168.122.251 | ubuntu2004 | coco-k8s |  kvirt  |
|      coco-k8s-worker-0      |   up   |  192.168.122.68 | ubuntu2004 | coco-k8s |  kvirt  |
| podvm-fedora-sleep-59d239f2 |   up   |  192.168.122.23 |            |          |         |
+-----------------------------+--------+-----------------+------------+----------+---------+
```

## Check that the peerpod-ctrl controller is working
First delete the cloud adapter daemon.
```
kubectl delete pod peerpodconfig-ctrl-caa-daemon-bv45g -n confidential-containers-system
pod "peerpodconfig-ctrl-caa-daemon-bv45g" deleted
```

This will cause the peerpod to error.
```
kubectl get pods
NAME           READY   STATUS   RESTARTS   AGE
fedora-sleep   0/1     Error    0          13m
```
A new caa-daemon will start up but it will not know about the existing peerpod
VM. The peerpod-ctrl controller will clean up the orphaned peerpod resources once the pod
is deleted.
```
kubectl delete pod fedora-sleep
pod "fedora-sleep" deleted

kubectl get peerpods
No resources found in default namespace.

kcli get vms
+---------------------+--------+-----------------+------------+----------+---------+
|         Name        | Status |        Ip       |   Source   |   Plan   | Profile |
+---------------------+--------+-----------------+------------+----------+---------+
| coco-k8s-ctlplane-0 |   up   | 192.168.122.251 | ubuntu2004 | coco-k8s |  kvirt  |
|  coco-k8s-worker-0  |   up   |  192.168.122.68 | ubuntu2004 | coco-k8s |  kvirt  |
+---------------------+--------+-----------------+------------+----------+---------+
```

In the operator log look for [adaptor/cloud/libvirt] and you should see references to deleting the instance resources:
```
kubectl logs cc-operator-controller-manager-68b5979488-x8bhb -f -n confidential-containers-system --all-containers=true

INFO    deleting instance       {"controller": "peerpod", "controllerGroup": "confidentialcontainers.org", "controllerKind": "PeerPod", "PeerPod": {"name":"fedora-sleep-resource-gqjl6","namespace":"default"}, "namespace": "default", "name": "fedora-sleep-resource-gqjl6", "reconcileID": "520379c0-e37a-4628-9c85-67955502a7d0", "InstanceID": "42", "CloudProvider": "libvirt"}
[adaptor/cloud/libvirt] Deleting instance (42)
[adaptor/cloud/libvirt] Checking if instance (42) exists
[adaptor/cloud/libvirt] domainDef [{{ disk} disk     0xc00096f8c0 <nil> 0xc0006ac870 <nil> <nil> <nil> <nil> <nil> 0xc0000d7f90 <nil> <nil> <nil> <nil>     <nil> 0xc0005c41f8 <nil> <nil> 0xc0005b5200} {{ disk} cdrom     0xc00096fe60 <nil> 0xc0006ac900 <nil> <nil> <nil> <nil> <nil> 0xc00088c000 <nil> 0x308fd88 <nil> <nil>     <nil> <nil> <nil> <nil> 0xc0005b5440}]
[adaptor/cloud/libvirt] Check if podvm-fedora-sleep-59d239f2-root.qcow2 volume exists
[adaptor/cloud/libvirt] Deleting volume podvm-fedora-sleep-59d239f2-root.qcow2
[adaptor/cloud/libvirt] Check if podvm-fedora-sleep-59d239f2-cloudinit.iso volume exists
[adaptor/cloud/libvirt] Deleting volume podvm-fedora-sleep-59d239f2-cloudinit.iso
[adaptor/cloud/libvirt] deleted an instance 42
INFO    instance deleted        {"controller": "peerpod", "controllerGroup": "confidentialcontainers.org", "controllerKind": "PeerPod", "PeerPod": {"name":"fedora-sleep-resource-gqjl6","namespace":"default"}, "namespace": "default", "name": "fedora-sleep-resource-gqjl6", "reconcileID": "520379c0-e37a-4628-9c85-67955502a7d0", "InstanceID": "42", "CloudProvider": "libvirt"}
```
