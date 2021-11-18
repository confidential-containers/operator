# Build Instructions for Payload Image

Follow these steps to build the payload image for the demo

- Clone the [kata-containers](https://github.com/kata-containers/kata-containers) repo and checkout the `CCv0` branch.
- Build kata (runtime, shim, agent) packages
- Build rootfs using `ubuntu` distro and use the following options `SKOPEO_UMOCI=yes SECCOMP=yes`. Name the image as `kata-containers-ubuntu.img`
- Copy the binaries and the rootfs to the `payload-image` dir
- Build container image using either podman or docker

