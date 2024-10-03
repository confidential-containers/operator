# Introduction
These instructions should help you to build a custom version of the operator with your
changes

## Prerequisites
- Golang compiler supported by the Go team
- Operator SDK version (1.23.x+)
- podman and podman-docker or docker
- Access to Kubernetes cluster
- Container registry to store images


## Set Environment Variables
```
export QUAY_USER=<userid>
export IMG=quay.io/${QUAY_USER}/cc-operator
```

If you do not have an account at `quay.io` and wish to deploy an operator through a local registry (for instance, one operating on port 5000), kindly export the `IMG` variable with:

```
export IMG=localhost:5000/cc-operator
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

If you have a custom payload image stored in your local registry (for example, `localhost:5000/build-kata-deploy`), the following set of commands is applicable:

```
PAYLOAD_IMG=localhost:5000/build-kata-deploy
pushd config/samples/ccruntime/default
kustomize edit set image quay.io/kata-containers/kata-deploy=${PAYLOAD_IMG}
kubectl create -k .
popd
```

Additionally, there are alternative overlay directories such as `peer-pods` or `s390x` in addition to the `default` layer. You may switch the directory accordingly based on your requirements.

## Uninstalling Resources

Ensure KUBECONFIG points to target Kubernetes cluster. Let's begin by deleting the payload image first with:

```
kubectl delete -k config/samples/ccruntime/default
```

Subsequently, proceed with uninstalling the operator:

```
make undeploy
```

Notably, the use of `make uninstall` is unnecessary, as the command above will automatically remove the resource `ccruntimes.confidentialcontainers.org` in this case.

## Using Kind Kubernetes cluster

You can use a [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/) cluster running on non-TEE hardware 
for development purposes.

Kind version `v0.16.0` have been successfully tested on the following Linux distros.
- `CentOS Stream 8`
- `RHEL9`
- `Ubuntu 20.04`
- `Ubuntu 22.04`

>**Note**: Kind clusters are not supported

## Continuous Integration (CI)

In order to be merged your opened pull request (PR) should pass the static analysis checks and end-to-end (e2e) tests.

The e2e tests jobs are executed on a variety of CcRuntime, configurations and platforms. These jobs that require confidential hardware (Intel TDX, AMD SEV, IBM SE, etc...) run on bare-metal machines and are often referred as "TEE tests". The remaining tests (a.k.a "Non-TEE") are executed on Virtual Machines (VM) deployed on-demand.

The following jobs will check for regressions on the default CcRuntime:

|Job name | TEE | OS | VMM |
|---|---|---|---|
|e2e-pr / operator tests (kata-qemu, s390x) | Non-TEE | Ubuntu 22.04 (s390x) | QEMU |
|e2e-pr / operator tests (kata-qemu, ubuntu-20.04) | Non-TEE |  Ubuntu 20.04 | QEMU |
|e2e-pr / operator tests (kata-qemu, ubuntu-22.04) | Non-TEE |  Ubuntu 22.04 | QEMU |
|e2e-pr / operator tests (kata-qemu-tdx, tdx) | TDX |  Ubuntu 24.04 | QEMU |
|e2e-pr / operator tests (kata-qemu-sev, coco-ci-amd-rome-001, ) | SEV |  Ubuntu 22.04 | QEMU |
|e2e-pr / operator tests (kata-qemu-snp, coco-ci-amd-milan-001) | SNP |  Ubuntu 22.04 | QEMU |

Additionally the following jobs will check regressions on the enclave-cc CcRuntime:

| Job name | TEE | OS |
|---|---|---|
|operator enclave-cc e2e tests| Intel SGX (Simulated Mode) | Ubuntu 22.04 |

Some of the e2e jobs are not triggered automatically. We recommend to trigger them only after some rounds of reviews to avoid wasting resources. They can be triggered only by writing `/test` in PR's comment.

>Note: only members with commit permission in the repository are allowed to trigger the e2e jobs. If you are not a committer then ask for help on our main Slack channel (#confidential-containers).

## Running CI e2e tests on your local machine

We recommend that you run the e2e Non-TEE tests on your local machine before opening a PR to check your changes will not break the CI so to avoid wasting resources. You can also use the approach described below to debug and fix failures, or test changes on the scripts themselves.

There are three main ways of running the tests locally:

1. use vagrant script to run the full suite
2. use kcli to create a VM where you run the testing
3. running them directly on your development workstation

### Using vagrant:

This is the simplest method but for each invocation it builds everything, which might take about 50m for the first time (then about 30m each). To perform this check install [Vagrant](https://www.vagrantup.com) and run the [Vagrantfile](../tests/e2e/Vagrantfile) to perform it's task (optionally setting the RUNTIMECLASS):

```shell
cd tests/e2e
export RUNTIMECLASS="kata-qemu"
vagrant up tests-e2e-ubuntu2204
```

### Using kcli

You can leverage [kcli](https://github.com/karmab/kcli/) to provide and maintain your VM that can be used for testing. Some useful commands:

```shell
# Create a machine compatible with our testing
kcli create vm -i ubuntu2204 -P memory=8G -P numcpus=4 -P disks=[50] e2e
# Sync dir from host->vm (or back)
kcli scp . e2e:~/operator -r
# Ssh to the machine
kcli ssh e2e
```

Once you get familiar with these you can keep the machine around and only start/stop it when needed, eventually sync your repos to check the latest changes. See the [Using workstation](##using-workstation) section for details how to execute things (make sure to execute ``kcli ssh first to be inside the VM``)

### Using workstation

[!WARNING]
This is only recommended on disposable machines (or in VMs) as the scripts will change your system settings heavily and despite the support to clean the environment things will be messy afterwards. **You had been warned**.

For the first time you need to get all the required deps:

```shell
# Optionally clone the operator repo (unless you already have it)
git clone --depth=1 git@github.com:confidential-containers/operator.git
# Install ansible (ubuntu)
sudo apt-get update -y
sudo apt-get install -y ansible python-is-python3
```

Now you are ready to execute the full workflow by:

```
cd operator/tests/e2e
export PATH="$PATH:/usr/local/bin"
./run-local.sh -r "kata-qemu" -u
```

where:

* ``-r "kata-qemu"`` - configures the runtime class
* ``-u`` - performs mild cleanup afterwards (but it's not thorough and might alter pre-existing configuration)

If you intend to run the tests multiple times, you can run it without the ``-u`` which leaves the configured kubernetes cluster running. Then you can configure the rootless environment by:

```shell
mkdir -p ~/.kube
sudo chown "$USER" ~/.kube
sudo cp /etc/kubernetes/admin.conf ~/.kube/config
sudo chown "$USER" ~/.kube/config
```

And then you can re-run the tests as many times by:

```shell
./tests_runner.sh -r kata-qemu
```

If you need to re-build the operator (images are stored locally using container registry on port 5000), you can delete and redeploy it by (you might need to fix the owner of cache ``sudo chown $USER:$USER "$HOME/.cache" -R`` first):

```shell
./operator.sh uninstall
# Do your changes
./operator.sh
```

Then you can simply run the testing via ``./tests_runner.sh`` using the updated operator.

If you need to clean things up you can re-run the ``./run-local.sh -u`` to clean things up (after performing the testing) or you can run following cleanup steps:

```shell
./operator.sh uninstall
./cluster/down.sh
ansible-playbook -i localhost, -c local --tags undo ansible/main.yaml
```
