FROM alpine:3.18 as base

#### Confidential Containers forked containerd

FROM base as coco-containerd-binary-downloader

ARG ARCH
ARG COCO_CONTAINERD_VERSION

ARG DESTINATION=/opt/confidential-containers-pre-install-artifacts
ARG COCO_DESTINATION=${DESTINATION}/opt/confidential-containers

RUN \
	mkdir -p ${COCO_DESTINATION} && \
	apk --no-cache add curl && \
	curl -fOL --progress-bar https://github.com/confidential-containers/containerd/releases/download/v${COCO_CONTAINERD_VERSION}/containerd-${COCO_CONTAINERD_VERSION}-linux-${ARCH}.tar.gz && \
	tar xvzpf containerd-${COCO_CONTAINERD_VERSION}-linux-${ARCH}.tar.gz -C ${COCO_DESTINATION} && \
	rm containerd-${COCO_CONTAINERD_VERSION}-linux-${ARCH}.tar.gz

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
ARG COCO_DESTINATION=${DESTINATION}/opt/confidential-containers
ARG COCO_CONTAINERD_SYSTEMD_DESTINATION=${DESTINATION}/etc/systemd/system/containerd.service.d/

ARG CONTAINERD_SYSTEMD_ARTIFACTS=./containerd/containerd-for-cc-override.conf

COPY --from=coco-containerd-binary-downloader ${COCO_DESTINATION}/bin/ ${COCO_DESTINATION}/bin/
COPY --from=kubectl-binary-downloader /usr/bin/kubectl /usr/bin/kubectl
COPY ${CONTAINERD_SYSTEMD_ARTIFACTS} ${COCO_CONTAINERD_SYSTEMD_DESTINATION}

ARG CONTAINER_ENGINE_ARTIFACTS=./scripts
COPY ${CONTAINER_ENGINE_ARTIFACTS}/* ${DESTINATION}/scripts/
