COCO_CONTAINERD_VERSION = 1.6.8.2
OFFICIAL_CONTAINERD_VERSION = 1.7.0

BASH = bash

reqs-image:
	coco_containerd_version=$(COCO_CONTAINERD_VERSION) \
	official_containerd_version=$(OFFICIAL_CONTAINERD_VERSION) \
	$(BASH) -x payload.sh
