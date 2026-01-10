# Confidential Containers Operator

> ## ⚠️ DEPRECATION NOTICE ⚠️
>
> **This repository is deprecated and will be archived on February 1st, 2026.**
>
> The Confidential Containers Operator has been superseded by the **[Confidential Containers Helm Chart](https://github.com/confidential-containers/charts)**, which is now the official and recommended way to deploy Confidential Containers.
>
> ### Migration
>
> Please migrate to the Helm chart:
>
> ```bash
> # Uninstall the operator first, then install via Helm:
> helm install coco oci://ghcr.io/confidential-containers/charts/confidential-containers \
>   --namespace coco-system
> ```
>
> For detailed installation instructions, see the [Helm chart documentation](https://github.com/confidential-containers/charts).
>
> ### Why the change?
>
> - **Simpler installation**: Single Helm command vs operator CRDs
> - **Better GitOps integration**: Standard Helm values files
> - **Faster updates**: Aligned directly with kata-containers releases
> - **Broader support**: Multiple Kubernetes distributions (k3s, k0s, rke2, microk8s, kubeadm)
>
> **No further updates will be made to this repository.**

---

[![Build](https://github.com/confidential-containers/operator/actions/workflows/makefile.yaml/badge.svg)](https://github.com/confidential-containers/operator/actions/workflows/makefile.yaml)
[![Container Image](https://github.com/confidential-containers/operator/actions/workflows/docker-publish.yaml/badge.svg)](https://github.com/confidential-containers/operator/actions/workflows/docker-publish.yaml)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fconfidential-containers%2Foperator.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fconfidential-containers%2Foperator?ref=badge_shield)

This Confidential Containers Operator provides a means to deploy and manage Confidential Containers Runtime on Kubernetes clusters. 
The primary resource is `CcRuntime` which describes runtime details like installation type, source, nodes to deploy etc.

Here is a short demo video showing the operator in action.

[![asciicast](https://asciinema.org/a/450899.svg)](https://asciinema.org/a/450899)

Instructions to recreate the demo setup in your own environment are available [here](https://github.com/confidential-containers/operator/blob/ccv0-demo/docs/INSTALL.md) 

## Installation

Please refer to the following [instructions](docs/INSTALL.md)

## Development

Please refer to the following [instructions](docs/DEVELOPMENT.md)


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fconfidential-containers%2Foperator.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fconfidential-containers%2Foperator?ref=badge_large)

