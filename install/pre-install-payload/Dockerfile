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
	curl -fOL --progress-bar ${OFFICIAL_CONTAINERD_REPO}/releases/download/v${OFFICIAL_CONTAINERD_VERSION}/containerd-${OFFICIAL_CONTAINERD_VERSION}-linux-${ARCH}.tar.gz && \
	tar xvzpf containerd-${OFFICIAL_CONTAINERD_VERSION}-linux-${ARCH}.tar.gz -C ${NODE_DESTINATION} && \
	rm containerd-${OFFICIAL_CONTAINERD_VERSION}-linux-${ARCH}.tar.gz

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
	curl -fOL --progress-bar ${VFIO_GPU_CONTAINERD_REPO}/releases/download/v${VFIO_GPU_CONTAINERD_VERSION}/containerd-${VFIO_GPU_CONTAINERD_VERSION}-linux-${ARCH}.tar.gz && \
	tar xvzpf containerd-${VFIO_GPU_CONTAINERD_VERSION}-linux-${ARCH}.tar.gz -C ${NODE_DESTINATION} && \
	rm containerd-${VFIO_GPU_CONTAINERD_VERSION}-linux-${ARCH}.tar.gz

#### Nydus snapshotter & nydus image

FROM golang:1.19-alpine AS nydus-binary-downloader

ARG ARCH
ARG NYDUS_SNAPSHOTTER_VERSION
ARG NYDUS_SNAPSHOTTER_REPO
ARG NYDUS_REPO
ARG NYDUS_VERSION

ARG DESTINATION=/opt/confidential-containers-pre-install-artifacts
ARG NODE_DESTINATION=${DESTINATION}/opt/confidential-containers

ENV GOARCH=${ARCH}

RUN mkdir -p ${NODE_DESTINATION}/bin && \
    apk add --no-cache ca-certificates build-base git curl && \
    git clone ${NYDUS_SNAPSHOTTER_REPO} -b ${NYDUS_SNAPSHOTTER_VERSION} /nydus-snapshotter && \
    sed -i -e '/{STATIC_OPTIMIZER_SERVER_BIN}/d' /nydus-snapshotter/Makefile && \
    make -C /nydus-snapshotter static-release && \
    chmod +x /nydus-snapshotter/bin/containerd-nydus-grpc && \
	chmod +x /nydus-snapshotter/bin/nydus-overlayfs && \
    mv /nydus-snapshotter/bin/containerd-nydus-grpc ${NODE_DESTINATION}/bin && \
	mv /nydus-snapshotter/bin/nydus-overlayfs ${NODE_DESTINATION}/bin && \
    rm -rf /nydus-snapshotter 

RUN apk del build-base git curl

#### kubectl

FROM base as kubectl-binary-downloader

ARG ARCH
ARG KUBECTL_VERSION=v1.26.0

RUN \
	apk --no-cache add curl && \
	curl -fL --progress-bar -o /usr/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/${ARCH}/kubectl && \
	chmod +x /usr/bin/kubectl

#### Final image

FROM base

RUN apk --no-cache add bash

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
