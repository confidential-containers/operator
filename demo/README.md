# K8s Cluster Setup
Use kcli or kubeadmin to create a K8s cluster on Ubuntu 20.04

kcli install and setup instructions are available here - 
https://kcli.readthedocs.io/en/latest/

Use kcli to create a two-node cluster using Ubuntu 20.04 

```
kcli create kube generic -P image=ubuntu2004 -P workers=1 testk8s
```
## Replace containerd on the worker

Replace containerd on the worker node by building a new containerd from https://github.com/confidential-containers/containerd/tree/ali-CCv0

# Install Confidential Containers Operator

```
kubectl apply -f https://raw.githubusercontent.com/confidential-containers/operator/main/deploy/deploy.yaml
```

# Install Confidential Containers Runtime

```
kubectl apply  -f https://raw.githubusercontent.com/confidential-containers/operator/main/config/samples/ccruntime.yaml
```

Check if `runtimeclass` have been successfully created
```
kubectl get runtimeclass
```

# Create sample POD

Regular Kata POD
```
kubectl apply -f  https://raw.githubusercontent.com/confidential-containers/operator/ccv0-demo/demo/nginx-deployment-kata.yaml
```

Confidential Container where container image will be pulled inside the VM
```
kubectl apply -f  https://raw.githubusercontent.com/confidential-containers/operator/ccv0-demo/demo/nginx-deployment-cc.yaml
```

# Verify 

Get container ID from POD

```
export PODNAME=<podname>
containerID=$(kubectl get pod $PODNAME -o=jsonpath='{.status.containerStatuses[*].containerID}' | cut -d "/" -f3)
echo $containerID
```

Login to the worker node and run the following commands
```
export containerID=<set-containerd-from-previous-step>
sandboxID=$(crictl inspect $containerID | jq -r '.info.sandboxID')
echo $sandboxID
```

Check container rootfs 
```
cd /run/kata-containers/shared/sandboxes/$sandboxID/shared
find . -name rootfs
```

For confidential containers you'll find rootfs of only the `pause` container.
For regular Kata containers you'll find rootfs of all the containers. 


# Interacting with the VM agent

Download the script and run it as root on a Kubernetes worker node with 
Kata CC runtime deployed using the [Operator](https://github.com/confidential-containers/confidential-containers-operator)

```
wget https://raw.githubusercontent.com/confidential-containers/operator/ccv0-demo/demo/ccv0_helper.sh
chmod +x ccv0_helper.sh
./ccv0_helper.sh
```

## Get VM shell
```
./ccv0_helper.sh open_kata_shell
```

## Get VM console
```
./ccv0_helper.sh open_kata_console
```

## Pull container image inside VM
 
```
export PULL_IMAGE=quay.io/bitnami/nginx
./ccv0_helper.sh agent_pull_image
```

## Create container inside VM
 
```
./ccv0_helper.sh agent_create_container
```
