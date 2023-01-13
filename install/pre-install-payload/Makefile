CONTAINERD_VERSION = 1.6.8.1

SYSTEMD_DROP_IN_CONFIG = configs/container-engine-for-cc-override.conf.in

OUTPUT_DIR = output

SED = sed
BASH = bash
MKDIR = mkdir

containerd: containerd-container-image

containerd-systemd-drop-in: $(SYSTEMD_DROP_IN_CONFIG)
	$(MKDIR) -p $(OUTPUT_DIR)
	$(SED) \
		-e "s|@CONTAINER_ENGINE@|containerd|g" \
		$< > $(OUTPUT_DIR)/containerd-for-cc-override.conf

containerd-container-image: containerd-systemd-drop-in
	containerd_version=$(CONTAINERD_VERSION) $(BASH) -x containerd/payload.sh
