FROM docker.io/library/ubuntu:20.04

ARG ARCH
ARG VERSION
ARG SYSTEMD_ARTIFACTS=./containerd/containerd-for-cc-override.conf
ARG CONTAINER_ENGINE_ARTIFACTS=./scripts
ARG DESTINATION=/opt/confidential-containers-pre-install-artifacts

COPY ${CONTAINERD_ARTIFACTS} ${DESTINATION}/opt/confidential-containers/
COPY ${SYSTEMD_ARTIFACTS} ${DESTINATION}/etc/systemd/system/containerd.service.d/
COPY ${CONTAINER_ENGINE_ARTIFACTS}/* ${DESTINATION}/scripts/

RUN \
apt-get update -y && \
apt-get install -y curl && \
curl -fOL --progress-bar https://github.com/confidential-containers/containerd/releases/download/v${VERSION}/containerd-${VERSION}-linux-${ARCH}.tar.gz && \
tar xvzpf containerd-${VERSION}-linux-${ARCH}.tar.gz -C ${DESTINATION}/opt/confidential-containers && \
rm -f containerd-${VERSION}-linux-${ARCH}.tar.gz && \
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/${ARCH}/kubectl" && \
install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl && \
rm -f kubectl
