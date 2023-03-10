FROM ubuntu:20.04

ARG ARCH
ARG VERSION
ARG SYSTEMD_ARTIFACTS=./containerd/containerd-for-cc-override.conf
ARG CONTAINER_ENGINE_ARTIFACTS=./scripts
ARG DESTINATION=/opt/confidential-containers-pre-install-artifacts

COPY ${CONTAINERD_ARTIFACTS} ${DESTINATION}/opt/confidential-containers/
COPY ${SYSTEMD_ARTIFACTS} ${DESTINATION}/etc/systemd/system/containerd.service.d/
COPY ${CONTAINER_ENGINE_ARTIFACTS}/* ${DESTINATION}/scripts/

ENV DEBIAN_FRONTEND=noninteractive
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

RUN \
apt-get update && \
apt-get install -y --no-install-recommends apt-transport-https ca-certificates curl xz-utils systemd && \
mkdir -p /etc/apt/keyrings/ && \
curl -fsSLo /etc/apt/keyrings/kubernetes-archive-keyring.gpg https://packages.cloud.google.com/apt/doc/apt-key.gpg && \
echo "deb [signed-by=/etc/apt/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | tee /etc/apt/sources.list.d/kubernetes.list && \
apt-get update && \
apt-get install -y --no-install-recommends kubectl && \
apt-get clean && rm -rf /var/lib/apt/lists/ && \
curl -fOL --progress-bar https://github.com/confidential-containers/containerd/releases/download/v${VERSION}/containerd-${VERSION}-linux-${ARCH}.tar.gz && \
tar xvzpf containerd-${VERSION}-linux-${ARCH}.tar.gz -C ${DESTINATION}/opt/confidential-containers
