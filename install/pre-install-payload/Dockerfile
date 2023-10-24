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

ARG CONTAINERD_SYSTEMD_ARTIFACTS=./containerd/containerd-for-cc-override.conf

COPY --from=coco-containerd-binary-downloader ${NODE_DESTINATION}/bin/containerd ${NODE_DESTINATION}/bin/coco-containerd
COPY --from=official-containerd-binary-downloader ${NODE_DESTINATION}/bin/containerd ${NODE_DESTINATION}/bin/official-containerd
COPY --from=vfio-gpu-containerd-binary-downloader ${NODE_DESTINATION}/bin/containerd ${NODE_DESTINATION}/bin/vfio-gpu-containerd
COPY --from=kubectl-binary-downloader /usr/bin/kubectl /usr/bin/kubectl
COPY ${CONTAINERD_SYSTEMD_ARTIFACTS} ${NODE_CONTAINERD_SYSTEMD_DESTINATION}

ARG CONTAINER_ENGINE_ARTIFACTS=./scripts
COPY ${CONTAINER_ENGINE_ARTIFACTS}/* ${DESTINATION}/scripts/
