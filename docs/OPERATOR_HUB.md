# Introduction

Our releases are published on the [OperatorHub.io](https://operatorhub.io). Find more information on [our landing page](https://operatorhub.io/operator/cc-operator) at the Operator Hub.

# Publishing a new operator version

First of all, you should have a fork of the Operator Hub [community-operators repository](https://github.com/k8s-operatorhub/community-operators) cloned.

>Note: on the examples below let's suppose that `TARGET_RELEASE="0.8.0"`

Follow the steps:

1. Bump the `VERSION` variable in [Makefile](../Makefile) to `${TARGET_RELEASE}`. For example, `VERSION ?= 0.8.0`

2. Add the `replaces` tag under `spec` in the [base CSV(ClusterServiceVersion) file](../config/manifests/bases/cc-operator.clusterserviceversion.yaml). It defines the version that this bundle should replace on an operator upgrade (see https://sdk.operatorframework.io/docs/olm-integration/generation/#upgrade-your-operator for further details). For example, at the time of this writing version 0.8.0 replaces 0.5.0:
   ```
   diff --git a/config/manifests/bases/cc-operator.clusterserviceversion.yaml b/config/manifests/bases/cc-operator.clusterserviceversion.yaml
   index f30cecf..f5f81d1 100644
   --- a/config/manifests/bases/cc-operator.clusterserviceversion.yaml
   +++ b/config/manifests/bases/cc-operator.clusterserviceversion.yaml
   @@ -54,4 +54,5 @@ spec:
      provider:
        name: Confidential Containers Community
        url: https://github.com/confidential-containers
   +  replaces: cc-operator.v0.5.0
      version: 0.0.1
   ```
3. Update the `containerImage` tag under `spec` in the [base CSV(ClusterServiceVersion) file](../config/manifests/bases/cc-operator.clusterserviceversion.yaml)

4. Re-generate the [bundle](../bundle/):
   ```shell
   make bundle IMG=quay.io/confidential-containers/operator:v${TARGET_RELEASE}
   ```

5. Copy the bundle directory to the community-operators repository directory. On the example below I got the community-operators repository cloned to `../../../github.com/k8s-operatorhub/community-operators`: 
   ```shell
   dest_dir="../../../github.com/k8s-operatorhub/community-operators/operators/cc-operator/${TARGET_RELEASE}"
   rm -rf "$dest_dir"
   mkdir "$dest_dir"
   cp -r bundle/* "$dest_dir"
   ```

6. Prepare a commit and push to your tree
   ```
   cd ../../../github.com/k8s-operatorhub/community-operators/operators/cc-operator/${TARGET_RELEASE}
   git checkout -b "new_${TARGET_RELEASE}"
   git add .
   git commit -s -m "operator cc-operator (${TARGET_RELEASE})"
   git push my-fork "new_${TARGET_RELEASE}"
   ```

7. Open a pull request to update the community-operators repository
   * In case the CI fails you will need to fix and re-do this process from step 2

8. Open a pull request on this repository with the changes
