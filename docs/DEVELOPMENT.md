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

## Continuous Integration (CI)

In order to be merged your opened pull request (PR) should pass the static analysis checks and end-to-end (e2e) tests.

The e2e tests jobs are executed on a variety of CcRuntime, configurations and platforms. These jobs that require confidential hardware (Intel TDX, AMD SEV, IBM SE, etc...) run on bare-metal machines and are often referred as "TEE tests". The remaining tests (a.k.a "Non-TEE") are executed on Virtual Machines (VM) deployed on-demand.

The following jobs will check for regessions on the default CcRuntime:

|Job name | TEE | OS | VMM |
|---|---|---|---|
|cc-operator-e2e-ubuntu-20.04-s390x-containerd_kata-qemu | IBM SE | Ubuntu 20.04 (s390x) | QEMU |
|ccruntime e2e tests / operator tests (kata-clh, az-ubuntu-2004) | Non-TEE |  Ubuntu 20.04 | Cloud Hypervisor |
|ccruntime e2e tests / operator tests (kata-clh, az-ubuntu-2204) | Non-TEE |  Ubuntu 22.04 | Cloud Hypervisor |
|ccruntime e2e tests / operator tests (kata-qemu, az-ubuntu-2004) | Non-TEE |  Ubuntu 20.04 | QEMU |
|ccruntime e2e tests / operator tests (kata-qemu, az-ubuntu-2204) | Non-TEE |  Ubuntu 22.04 | QEMU |

Additionally the following jobs will check regressions on the enclave-cc CcRuntime:

| Job name | TEE | OS |
|---|---|---|
|operator enclave-cc e2e tests| Intel SGX (Simulated Mode) | Ubuntu 22.04 |

Some of the e2e jobs are not triggered automatically. We recommend to trigger them only after some rounds of reviews to avoid wasting resources. They can be triggered only by writing `/test` in PR's comment.

>Note: only members with commit permission in the repository are allowed to trigger the e2e jobs. If you are not a committer then ask for help on our main Slack channel (#confidential-containers).

### Running e2e tests on your local machine

We recommend that you run the e2e Non-TEE tests on your local machine before opening a PR to check your changes will not break the CI so to avoid wasting resources. You can also use the approach described below to debug and fix failures, or test changes on the scripts themselves.

The entry point script is [tests/e2e/run-local.sh](../tests/e2e/run-local.sh). It is going to install softwares and change the system's configuration, so we recommend that you run the e2e tests on VMs with nested virtualization support and a minimum of 8GB of memory, 50 GB of disk and 4 vCPUs.

Currently the e2e tests are supported on Ubuntu 20.04 or CentOS 8 Stream, and the only requirement is to have Ansible installed.

For example, to run on a fresh Ubuntu 20.04 VM:

```shell
sudo apt-get update -y
sudo apt-get install -y ansible python-is-python3
cd tests/e2e
export PATH="$PATH:/usr/local/bin"
./run-local.sh -r "kata-qemu"
```

Notice that the `-r` parameter passed to `run-local.sh` above specifies the runtimeClass to be tested. You can switch to, for example, `kata-clh` to test Cloud Hypervisor. Another useful parameter is `-u` which is used on bare-metal CI jobs to undo the changes at the execution end. See the script's help (`run-local.sh -h`) for further details and parameters.

The `run-local.sh` (unless that executed with `-u`) will leave a running Kubernetes on your local machine, and that allows you to re-run the tests many times afterwards. Let's suppose that you are developing a new test case, first you can configure the environment to run `kubectl` rootless:

```shell
mkdir ~/.kube || true
sudo chown "$USER" ~/.kube
sudo cp /etc/kubernetes/admin.conf ~/.kube/config
sudo chown "$USER" ~/.kube/config
```

Then use the `tests_runner.sh` script to re-run the tests like shown below (similarly to `run-local.sh` the `-r` sets the runtimeclass):

```shell
./tests_runner.sh -r kata-qemu
```

Apart from Kubernetes, there is left running a containers images registry at port 5000 which is used by the install/uninstall routines to fetch the operator images so that built images are stored and served locally. For example, if you want to re-build the operator images then run the tests again:

```shell
sudo -E PATH="$PATH:/usr/local/bin" ./operator.sh build
./tests_runner.sh -r kata-qemu
```

The `operator.sh` script used on the above example provides useful commands for development, please refer to its help `./operator.sh -h` for further information.

### Running e2e test locally with Vagrant

Alternatively you can use [Vagrant](https://www.vagrantup.com) as we provide a [Vagrantfile](../tests/e2e/Vagrantfile) to automate the entire process: it will create the VM, push the local repository sources and finally execute `run-local.sh`. The same example above can be achieved by simply running:

```shell
export RUNTIMECLASS="kata-qemu"
vagrant up tests-e2e-ubuntu2004
```

Notice that with Vagrant the entire workflow can take up to 50 minutes, specially when you run for the first time and the VM image is fetched from internet and cached.