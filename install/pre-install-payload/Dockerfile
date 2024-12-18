FROM alpine:3.18 as base

#### Confidential Containers forked containerd

FROM base as coco-containerd-binary-downloader

ARG ARCH
ARG COCO_CONTAINERD_VERSION
ARG COCO_CONTAINERD_REPO

ARG DESTINATION=/opt/confidential-containers-pre-install-artifacts
ARG NODE_DESTINATION=${DESTINATION}/opt/confidential-containers

RUN \
	mkdir -p ${NODE_DESTINATION} && \
	apk --no-cache add curl && \
	curl -fOL --progress-bar ${COCO_CONTAINERD_REPO}/releases/download/v${COCO_CONTAINERD_VERSION}/containerd-${COCO_CONTAINERD_VERSION}-linux-${ARCH}.tar.gz && \
	tar xvzpf containerd-${COCO_CONTAINERD_VERSION}-linux-${ARCH}.tar.gz -C ${NODE_DESTINATION} && \
	rm containerd-${COCO_CONTAINERD_VERSION}-linux-${ARCH}.tar.gz

#### Official containerd

FROM base as official-containerd-binary-downloader

ARG ARCH
ARG OFFICIAL_CONTAINERD_VERSION
ARG OFFICIAL_CONTAINERD_REPO

ARG DESTINATION=/opt/confidential-containers-pre-install-artifacts
ARG NODE_DESTINATION=${DESTINATION}/opt/confidential-containers

RUN \
	mkdir -p ${NODE_DESTINATION} && \
	apk --no-cache add curl && \
	curl -fOL --progress-bar ${OFFICIAL_CONTAINERD_REPO}/releases/download/v${OFFICIAL_CONTAINERD_VERSION}/containerd-static-${OFFICIAL_CONTAINERD_VERSION}-linux-${ARCH}.tar.gz && \
	tar xvzpf containerd-static-${OFFICIAL_CONTAINERD_VERSION}-linux-${ARCH}.tar.gz -C ${NODE_DESTINATION} && \
	rm containerd-static-${OFFICIAL_CONTAINERD_VERSION}-linux-${ARCH}.tar.gz

#### Confidential Containers forked containerd for VFIO / GPU stuff

FROM base as vfio-gpu-containerd-binary-downloader

ARG ARCH
ARG VFIO_GPU_CONTAINERD_VERSION
ARG VFIO_GPU_CONTAINERD_REPO

ARG DESTINATION=/opt/confidential-containers-pre-install-artifacts
ARG NODE_DESTINATION=${DESTINATION}/opt/confidential-containers

RUN \
	mkdir -p ${NODE_DESTINATION} && \
	apk --no-cache add curl && \
	curl -fOL --progress-bar ${VFIO_GPU_CONTAINERD_REPO}/releases/download/v${VFIO_GPU_CONTAINERD_VERSION}/containerd-static-${VFIO_GPU_CONTAINERD_VERSION}-linux-${ARCH}.tar.gz && \
	tar xvzpf containerd-static-${VFIO_GPU_CONTAINERD_VERSION}-linux-${ARCH}.tar.gz -C ${NODE_DESTINATION} && \
	rm containerd-static-${VFIO_GPU_CONTAINERD_VERSION}-linux-${ARCH}.tar.gz

#### Nydus snapshotter & nydus image

FROM golang:1.22-alpine AS nydus-binary-downloader

ARG ARCH
ARG NYDUS_SNAPSHOTTER_VERSION
ARG NYDUS_SNAPSHOTTER_REPO

ARG DESTINATION=/opt/confidential-containers-pre-install-artifacts
ARG NODE_DESTINATION=${DESTINATION}/opt/confidential-containers

ENV GOARCH=${ARCH}

RUN mkdir -p ${NODE_DESTINATION}/bin && \
    apk add --no-cache curl && \
    curl -fOL --progress-bar ${NYDUS_SNAPSHOTTER_REPO}/releases/download/${NYDUS_SNAPSHOTTER_VERSION}/nydus-snapshotter-${NYDUS_SNAPSHOTTER_VERSION}-linux-${ARCH}.tar.gz && \
    tar xvzpf nydus-snapshotter-${NYDUS_SNAPSHOTTER_VERSION}-linux-${ARCH}.tar.gz -C ${NODE_DESTINATION} && \
    rm nydus-snapshotter-${NYDUS_SNAPSHOTTER_VERSION}-linux-${ARCH}.tar.gz

#### kubectl

FROM base as kubectl-binary-downloader

ARG ARCH
ARG KUBECTL_VERSION=v1.31.1

RUN \
	apk --no-cache add curl && \
	curl -fL --progress-bar https://dl.k8s.io/${KUBECTL_VERSION}/kubernetes-client-linux-${ARCH}.tar.gz | \
	tar -C /usr/bin/ -x -z --strip-components 3

#### Final image

FROM base

RUN apk --no-cache add bash gcompat

ARG DESTINATION=/opt/confidential-containers-pre-install-artifacts
ARG NODE_DESTINATION=${DESTINATION}/opt/confidential-containers
ARG NODE_CONTAINERD_SYSTEMD_DESTINATION=${DESTINATION}/etc/systemd/system/containerd.service.d/
ARG NODE_NYDUS_SNAPSHOTTER_SYSTEMD_DESTINATION=${DESTINATION}/etc/systemd/system/nydus-snapshotter.service

ARG CONTAINERD_SYSTEMD_ARTIFACTS=./containerd/containerd-for-cc-override.conf
ARG NYDUS_SNAPSHOTTER_SYSTEMD_ARTIFACTS=./remote-snapshotter/nydus-snapshotter/nydus-snapshotter.service
ARG NYDUS_SNAPSHOTTER_ARTIFACTS=./remote-snapshotter/nydus-snapshotter/config-coco-guest-pulling.toml

COPY --from=coco-containerd-binary-downloader ${NODE_DESTINATION}/bin/containerd ${NODE_DESTINATION}/bin/coco-containerd
COPY --from=official-containerd-binary-downloader ${NODE_DESTINATION}/bin/containerd ${NODE_DESTINATION}/bin/official-containerd
COPY --from=vfio-gpu-containerd-binary-downloader ${NODE_DESTINATION}/bin/containerd ${NODE_DESTINATION}/bin/vfio-gpu-containerd

COPY --from=nydus-binary-downloader ${NODE_DESTINATION}/bin/* ${NODE_DESTINATION}/bin/

COPY --from=kubectl-binary-downloader /usr/bin/kubectl /usr/bin/kubectl
COPY ${CONTAINERD_SYSTEMD_ARTIFACTS} ${NODE_CONTAINERD_SYSTEMD_DESTINATION}
COPY ${NYDUS_SNAPSHOTTER_SYSTEMD_ARTIFACTS} ${NODE_NYDUS_SNAPSHOTTER_SYSTEMD_DESTINATION}
COPY ${NYDUS_SNAPSHOTTER_ARTIFACTS} ${NODE_DESTINATION}/share/nydus-snapshotter/config-coco-guest-pulling.toml

ARG CONTAINER_ENGINE_ARTIFACTS=./scripts
COPY ${CONTAINER_ENGINE_ARTIFACTS}/* ${DESTINATION}/scripts/
