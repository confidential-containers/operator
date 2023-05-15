ARG IMAGE=kindest/node
ARG VERSION=1.27
ARG MINOR=3

FROM ${IMAGE}:v${VERSION}.${MINOR}

RUN echo "Installing Packages ..." \
    && apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends dbus \
    && systemctl enable dbus
