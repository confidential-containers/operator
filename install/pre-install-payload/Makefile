COCO_CONTAINERD_VERSION = 1.6.8.2
OFFICIAL_CONTAINERD_VERSION = 1.7.0
VFIO_GPU_CONTAINERD_VERSION = 1.7.0.0
NYDUS_SNAPSHOTTER_VERSION = v0.12.0
NYDUS_VERSION= v2.2.3

BASH = bash

reqs-image:
	coco_containerd_version=$(COCO_CONTAINERD_VERSION) \
	official_containerd_version=$(OFFICIAL_CONTAINERD_VERSION) \
	vfio_gpu_containerd_version=$(VFIO_GPU_CONTAINERD_VERSION) \
	nydus_snapshotter_version=${NYDUS_SNAPSHOTTER_VERSION} \
	nydus_version=${NYDUS_VERSION} \
	$(BASH) -x payload.sh
