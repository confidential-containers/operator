FROM quay.io/kata-containers/kata-deploy-cc:v0


ADD kata-runtime /opt/kata-artifacts/opt/kata/bin/
ADD kata-monitor /opt/kata-artifacts/opt/kata/bin/
ADD containerd-shim-kata-v2 /opt/kata-artifacts/opt/kata/bin/
ADD kata-netmon /opt/kata-artifacts/opt/kata/bin/
ADD kata-agent-ctl /opt/kata-artifacts/opt/kata/bin/

# Copy updated configuration.toml and adjust the links
ADD configuration-cc.toml /opt/kata-artifacts/opt/kata/share/defaults/kata-containers/
ADD configuration-qemu.toml /opt/kata-artifacts/opt/kata/share/defaults/kata-containers/
RUN cd  /opt/kata-artifacts/opt/kata/share/defaults/kata-containers/ && \
     rm -f configuration.toml && \
     ln -s configuration-qemu.toml configuration.toml 

# Copy new rootfs and adjust the links
ADD kata-containers-ubuntu.img /opt/kata-artifacts/opt/kata/share/kata-containers
RUN cd /opt/kata-artifacts/opt/kata/share/kata-containers/ && \
     rm -f kata-containers.img && \
     ln -s kata-containers-ubuntu.img kata-containers.img

ADD kata-deploy.sh /opt/kata-artifacts/scripts/
RUN rm -f /opt/kata-artifacts/opt/kata/bin/cloud-hypervisor \
          /opt/kata-artifacts/opt/kata/bin/jailer \
          /opt/kata-artifacts/opt/kata/bin/firecracker \
          /opt/kata-artifacts/opt/kata/share/defaults/kata-containers/configuration-clh.toml \
          /opt/kata-artifacts/opt/kata/share/defaults/kata-containers/configuration-acrn.toml \
          /opt/kata-artifacts/opt/kata/share/defaults/kata-containers/configuration-fc.toml
